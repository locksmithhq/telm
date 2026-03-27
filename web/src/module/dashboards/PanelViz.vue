<template>
  <!-- First-load spinner — only shown before any data arrives -->
  <div v-if="loading && !initialized" class="viz-wrap d-flex align-center justify-center">
    <v-progress-circular indeterminate size="22" width="2" color="primary" />
  </div>

  <div v-else-if="error" class="viz-wrap d-flex align-center justify-center flex-column gap-1 px-3">
    <v-icon size="18" color="error">mdi-alert-circle-outline</v-icon>
    <pre class="text-caption text-error text-center" style="white-space:pre-wrap;font-size:10px">{{ error }}</pre>
  </div>

  <div v-else-if="empty" class="viz-wrap d-flex align-center justify-center text-disabled text-caption">
    No data
  </div>

  <!-- Stat -->
  <div v-else-if="result?.viz === 'stat'" class="viz-wrap d-flex align-center justify-center flex-column">
    <div class="mono font-weight-bold" style="font-size:34px;line-height:1;color:#6366f1">{{ statValue }}</div>
    <div class="text-caption text-disabled mt-2">{{ statLabel }}</div>
  </div>

  <!-- Table -->
  <div v-else-if="result?.viz === 'table'" class="viz-table-wrap">
    <v-table density="compact" fixed-header :height="tableHeight">
      <thead>
        <tr>
          <th v-for="col in tableCols" :key="col.key" class="text-caption" :class="col.align === 'right' ? 'text-right' : ''">
            {{ col.label }}
          </th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(row, i) in tableRows" :key="i">
          <td
            v-for="col in tableCols" :key="col.key"
            class="text-caption mono"
            :class="col.align === 'right' ? 'text-right' : ''"
            :style="col.style ? col.style(row) : ''"
          >
            {{ col.fmt ? col.fmt(row) : row[col.key] }}
          </td>
        </tr>
      </tbody>
    </v-table>
  </div>

  <!-- ECharts canvas (all other viz types) — stays mounted across refreshes -->
  <div v-else ref="canvasEl" class="viz-canvas" />
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted, inject, nextTick } from 'vue'
import * as echarts from 'echarts'
import { parse, execute, fmtLabel, bucketSecs } from './tql.js'

const props = defineProps({
  query:     { type: String, default: '' },
  dashRange: { type: String, default: null },
})

const isDark      = inject('isDark')
const refreshKey  = inject('refreshKey')
const canvasEl = ref(null)
let chart      = null

const loading      = ref(false)
const initialized  = ref(false)  // true after first successful fetch
const error        = ref('')
const empty        = ref(false)
const result       = ref(null)

const tableHeight = computed(() => '100%')

// ─── Theme ────────────────────────────────────────────────────────────────
function ct() {
  const dark = isDark.value
  return {
    axis:       dark ? '#94a3b8' : '#475569',
    axisDim:    dark ? '#64748b' : '#94a3b8',
    grid:       dark ? '#1e2d45' : '#e2e8f0',
    tooltipBg:  dark ? '#1a2235' : '#ffffff',
    tooltipBdr: dark ? '#2d3f5c' : '#cbd5e1',
    tooltipTxt: dark ? '#cbd5e1' : '#0f172a',
    bg:         'transparent',
  }
}

// ─── Helpers ──────────────────────────────────────────────────────────────
function tooltip(trigger = 'axis') {
  const c = ct()
  return { trigger, backgroundColor: c.tooltipBg, borderColor: c.tooltipBdr, textStyle: { color: c.tooltipTxt, fontSize: 11 } }
}
function xCat(data) {
  const c = ct()
  return { type: 'category', data, axisLine: { lineStyle: { color: c.grid } }, axisTick: { show: false }, axisLabel: { color: c.axisDim, fontSize: 9 }, splitLine: { show: false } }
}
function yVal(name = '') {
  const c = ct()
  return { type: 'value', name, nameTextStyle: { color: c.axisDim, fontSize: 9 }, axisLine: { show: false }, axisTick: { show: false }, axisLabel: { color: c.axisDim, fontSize: 9 }, splitLine: { lineStyle: { color: c.grid, type: 'dashed' } } }
}
function baseGrid(extra = {}) {
  return { top: 10, right: 8, bottom: 22, left: 8, containLabel: true, ...extra }
}
function areaGrad(hex) {
  return { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: hex + '33' }, { offset: 1, color: hex + '05' }] }
}
function mkLine(name, data, color, area = false) {
  return {
    type: 'line', name, data, smooth: 0.4,
    lineStyle: { color, width: 1.5 }, itemStyle: { color },
    showSymbol: data.length <= 30, symbolSize: 3,
    ...(area ? { areaStyle: { color: areaGrad(color) } } : {}),
  }
}
function timeLabels(data, range) { return data.map(p => fmtLabel(p.time, range)) }

// ─── Metric value formatter ───────────────────────────────────────────────
// Detects unit from metric name and formats accordingly.
// kind='gauge'     → raw value (bytes, ratio, count)
// kind='sum|histo' → rate value already divided by bucket secs
function fmtMetricVal(val, metricName = '', kind = 'gauge') {
  if (val == null || isNaN(val)) return '–'
  const n = (metricName || '').toLowerCase()

  const isBytes = /mem(ory)?|heap|alloc|stack|rss|vms|buf|bytes?/.test(n)
  const isCpu   = /\bcpu\b|processor/.test(n)
  const isDur   = /duration|latency|elapsed|pause/.test(n)

  // Gauge memory → format as B / KB / MB / GB
  if (isBytes && kind === 'gauge') {
    if (val >= 1073741824) return (val / 1073741824).toFixed(2) + ' GB'
    if (val >= 1048576)    return (val / 1048576).toFixed(2)    + ' MB'
    if (val >= 1024)       return (val / 1024).toFixed(2)       + ' KB'
    return val.toFixed(0) + ' B'
  }

  // Sum memory rate → bytes/s
  if (isBytes && kind !== 'gauge') {
    if (val >= 1048576) return (val / 1048576).toFixed(2) + ' MB/s'
    if (val >= 1024)    return (val / 1024).toFixed(2)    + ' KB/s'
    return val.toFixed(2) + ' B/s'
  }

  // CPU gauge (ratio 0-1 or fraction of cores)
  if (isCpu && kind === 'gauge') {
    if (val <= 1.5) return (val * 100).toFixed(1) + '%'
    return val.toFixed(3) + ' cores'
  }

  // CPU rate (seconds-used/second = fraction of cores)
  if (isCpu && kind !== 'gauge') {
    return (val * 100).toFixed(1) + '%'
  }

  // Duration gauge in nanoseconds
  if (isDur && kind === 'gauge') {
    if (val >= 1e9) return (val / 1e9).toFixed(3)  + ' s'
    if (val >= 1e6) return (val / 1e6).toFixed(2)  + ' ms'
    if (val >= 1e3) return (val / 1e3).toFixed(2)  + ' μs'
    return val.toFixed(0) + ' ns'
  }

  // Generic: compact number
  if (Math.abs(val) >= 1e9) return (val / 1e9).toFixed(2) + 'B'
  if (Math.abs(val) >= 1e6) return (val / 1e6).toFixed(2) + 'M'
  if (Math.abs(val) >= 1e3) return (val / 1e3).toFixed(2) + 'K'
  return val % 1 === 0 ? val.toString() : val.toFixed(4)
}

// Detect metric kind from first data point
function metricKind(data) {
  if (!data?.length) return 'gauge'
  const p = data[0]
  // avg_value is non-null only for gauge metrics (value_double/value_int).
  // Histograms/sums store data exclusively in metric_sum/metric_count (value_double is NULL).
  if (p.avg_value != null) return 'gauge'
  if (p.total_count != null && p.total_count > 0) return 'histogram'
  if (p.total_sum != null) return 'sum'
  return 'gauge'
}

// Last value of a timeseries-like dataset
function lastVal(data, returns) {
  if (!data?.length) return null
  const last = data[data.length - 1]
  if (returns === 'timeseries') {
    const bs   = bucketSecs(result.value?.range)
    const kind = metricKind(data)
    if (kind === 'histogram') return +(last.total_count / bs).toFixed(4)
    if (kind === 'sum')       return +(last.total_sum   / bs).toFixed(4)
    return last.avg_value ?? 0
  }
  if (returns === 'throughput') {
    const rangeSecs = { '1h': 3600, '6h': 21600, '24h': 86400, '7d': 604800 }[result.value?.range] || 3600
    const total = data.reduce((s, d) => s + (d.count || 0), 0)
    return +(total / rangeSecs * 60).toFixed(1) // req/min, igual ao DashboardView
  }
  if (returns === 'errors')     return last.total > 0 ? +(last.errors / last.total * 100).toFixed(2) : 0
  return null
}

// ─── ECharts option builders ──────────────────────────────────────────────
function buildOption() {
  const { data, returns, viz, range } = result.value
  const showArea = viz === 'area'
  const lbls     = timeLabels(data, range)
  const c = ct()

  // ── Timeseries (metrics.series) ──────────────────────────────────────
  if (returns === 'timeseries') {
    const bs       = bucketSecs(range)
    const kind     = metricKind(data)
    const isRate   = kind === 'histogram' || kind === 'sum'
    const mName    = result.value?.metricName || ''
    const fmt      = v => fmtMetricVal(v, mName, isRate ? 'sum' : 'gauge')
    const vals     = data.map(p => {
      if (kind === 'histogram') return +(p.total_count / bs).toFixed(4)
      if (kind === 'sum')       return +(p.total_sum   / bs).toFixed(4)
      if (p.avg_value != null)  return +p.avg_value.toFixed(4)
      return p.total_count || 0
    })
    const yAxis    = { ...yVal(), axisLabel: { ...yVal().axisLabel, formatter: fmt } }
    const ttFmt    = { formatter: p => `${p[0].name}<br/><b>${fmt(p[0].value)}</b>` }
    if (viz === 'scatter') return {
      backgroundColor: c.bg, grid: baseGrid(),
      xAxis: xCat(lbls), yAxis,
      tooltip: { ...tooltip(), ...ttFmt },
      series: [{ type: 'scatter', data: vals, symbolSize: 5, itemStyle: { color: '#6366f1' } }],
    }
    if (viz === 'gauge') return buildGauge(lastVal(data, returns), '#6366f1', fmt(lastVal(data, returns)))
    if (viz === 'bar') return {
      backgroundColor: c.bg, grid: baseGrid(),
      xAxis: xCat(lbls), yAxis,
      tooltip: { ...tooltip(), ...ttFmt },
      series: [{ type: 'bar', data: vals, itemStyle: { color: '#6366f1', borderRadius: [2, 2, 0, 0] } }],
    }
    return { // line / area
      backgroundColor: c.bg, grid: baseGrid(),
      xAxis: xCat(lbls), yAxis,
      tooltip: { ...tooltip(), ...ttFmt },
      series: [mkLine(mName || 'value', vals, '#6366f1', showArea || viz === 'line')],
    }
  }

  // ── Throughput ───────────────────────────────────────────────────────
  if (returns === 'throughput') {
    const bs   = bucketSecs(range)
    const vals = data.map(p => +(p.count / bs).toFixed(3))  // req/s por bucket
    const fmt  = p => `${p[0].name}<br/><b>${p[0].value} req/s</b>`
    if (viz === 'scatter') return {
      backgroundColor: c.bg, grid: baseGrid(),
      xAxis: xCat(lbls), yAxis: yVal('req/s'),
      tooltip: { ...tooltip(), formatter: fmt },
      series: [{ type: 'scatter', data: vals, symbolSize: 5, itemStyle: { color: '#6366f1' } }],
    }
    if (viz === 'gauge') return buildGauge(vals[vals.length - 1] || 0, '#6366f1', 'req/s')
    if (viz === 'bar') return {
      backgroundColor: c.bg, grid: baseGrid(),
      xAxis: xCat(lbls), yAxis: yVal('req/s'),
      tooltip: { ...tooltip(), formatter: fmt },
      series: [{ type: 'bar', data: vals, itemStyle: { color: '#6366f1', borderRadius: [2, 2, 0, 0] } }],
    }
    return {
      backgroundColor: c.bg, grid: baseGrid(),
      xAxis: xCat(lbls), yAxis: yVal('req/s'),
      tooltip: { ...tooltip(), formatter: fmt },
      series: [mkLine('req/s', vals, '#6366f1', showArea || viz === 'line')],
    }
  }

  // ── Errors ───────────────────────────────────────────────────────────
  if (returns === 'errors') {
    const pct = data.map(p => p.total > 0 ? +(p.errors / p.total * 100).toFixed(2) : 0)
    if (viz === 'scatter') return {
      backgroundColor: c.bg, grid: baseGrid(),
      xAxis: xCat(lbls), yAxis: yVal('%'),
      tooltip: tooltip(),
      series: [{ type: 'scatter', data: pct, symbolSize: 5, itemStyle: { color: '#ef4444' } }],
    }
    if (viz === 'gauge') return buildGauge(pct[pct.length - 1] || 0, '#ef4444', '% error')
    if (viz === 'bar') return {
      backgroundColor: c.bg, grid: baseGrid(),
      xAxis: xCat(lbls), yAxis: yVal('%'),
      tooltip: tooltip(),
      series: [{ type: 'bar', data: pct, itemStyle: { color: '#ef4444', borderRadius: [2, 2, 0, 0] } }],
    }
    return {
      backgroundColor: c.bg, grid: baseGrid(),
      xAxis: xCat(lbls), yAxis: yVal('%'),
      tooltip: tooltip(),
      series: [mkLine('error %', pct, '#ef4444', showArea || viz === 'line')],
    }
  }

  // ── Latency ──────────────────────────────────────────────────────────
  if (returns === 'latency') {
    if (viz === 'scatter') return {
      backgroundColor: c.bg, grid: baseGrid(),
      xAxis: xCat(lbls), yAxis: yVal('ms'),
      tooltip: tooltip(),
      legend: { top: 2, right: 0, textStyle: { color: c.axisDim, fontSize: 9 }, itemWidth: 8, itemHeight: 3 },
      series: [
        { type: 'scatter', name: 'P50', data: data.map(p => +p.p50.toFixed(2)), symbolSize: 4, itemStyle: { color: '#10b981' } },
        { type: 'scatter', name: 'P95', data: data.map(p => +p.p95.toFixed(2)), symbolSize: 4, itemStyle: { color: '#f59e0b' } },
        { type: 'scatter', name: 'P99', data: data.map(p => +p.p99.toFixed(2)), symbolSize: 4, itemStyle: { color: '#ef4444' } },
      ],
    }
    if (viz === 'radar') {
      const last = data[data.length - 1]
      if (!last) return null
      const maxVal = Math.ceil(last.p99 * 1.3)
      return {
        backgroundColor: c.bg,
        tooltip: { ...tooltip('item') },
        radar: { indicator: [{ name: 'P50', max: maxVal }, { name: 'P95', max: maxVal }, { name: 'P99', max: maxVal }], axisName: { color: c.axis, fontSize: 10 }, splitLine: { lineStyle: { color: c.grid } }, axisLine: { lineStyle: { color: c.grid } } },
        series: [{
          type: 'radar', name: 'Latency',
          data: [{ value: [+last.p50.toFixed(2), +last.p95.toFixed(2), +last.p99.toFixed(2)], name: 'ms', areaStyle: { color: 'rgba(99,102,241,.15)' }, lineStyle: { color: '#6366f1' }, itemStyle: { color: '#6366f1' } }],
        }],
      }
    }
    return {
      backgroundColor: c.bg, grid: { ...baseGrid(), top: 24 },
      legend: { top: 2, right: 0, textStyle: { color: c.axisDim, fontSize: 10 }, itemWidth: 10, itemHeight: 3 },
      xAxis: xCat(lbls), yAxis: yVal('ms'),
      tooltip: tooltip(),
      series: [
        mkLine('P50', data.map(p => +p.p50.toFixed(2)), '#10b981', showArea),
        mkLine('P95', data.map(p => +p.p95.toFixed(2)), '#f59e0b', showArea),
        mkLine('P99', data.map(p => +p.p99.toFixed(2)), '#ef4444', showArea),
      ],
    }
  }

  // ── Top-ops ──────────────────────────────────────────────────────────
  if (returns === 'top-ops') {
    if (viz === 'treemap') return buildTreemap(
      data.map(o => ({ name: o.operation.length > 30 ? o.operation.slice(0, 28) + '…' : o.operation, value: o.count, errRate: o.count > 0 ? o.errors / o.count : 0 })),
      o => o.errRate > 0.05 ? '#ef4444' : o.errRate > 0.01 ? '#f59e0b' : '#10b981',
    )
    if (viz === 'pie') return buildPie(data.map(o => ({ name: o.operation.length > 22 ? o.operation.slice(0, 20) + '…' : o.operation, value: o.count })))
    if (viz === 'funnel') return buildFunnel(data.slice(0, 8).map(o => ({ name: o.operation.length > 22 ? o.operation.slice(0, 20) + '…' : o.operation, value: o.count })))
    // Default: horizontal bar
    const rev = [...data].reverse()
    return {
      backgroundColor: c.bg,
      grid: { top: 4, right: 8, bottom: 4, left: 8, containLabel: true },
      legend: { bottom: 0, textStyle: { color: c.axisDim, fontSize: 10 }, itemWidth: 10, itemHeight: 10 },
      tooltip: { ...tooltip('axis'), axisPointer: { type: 'shadow' } },
      xAxis: { type: 'value', axisLabel: { color: c.axisDim, fontSize: 9 }, splitLine: { lineStyle: { color: c.grid, type: 'dashed' } }, axisLine: { show: false }, axisTick: { show: false } },
      yAxis: { type: 'category', data: rev.map(o => o.operation.length > 28 ? o.operation.slice(0, 26) + '…' : o.operation), axisLabel: { color: c.axis, fontSize: 9 }, axisLine: { lineStyle: { color: c.grid } }, axisTick: { show: false } },
      series: [
        { name: 'OK',    type: 'bar', stack: 's', data: rev.map(o => o.count - o.errors), itemStyle: { color: 'rgba(16,185,129,.7)', borderRadius: [0, 2, 2, 0] } },
        { name: 'Error', type: 'bar', stack: 's', data: rev.map(o => o.errors),            itemStyle: { color: 'rgba(239,68,68,.7)',  borderRadius: [0, 2, 2, 0] } },
      ],
    }
  }

  // ── Severity ─────────────────────────────────────────────────────────
  if (returns === 'severity') {
    const SEV_COLORS = { ERROR: '#ef4444', FATAL: '#dc2626', WARN: '#f59e0b', INFO: '#6366f1', DEBUG: '#64748b', TRACE: '#475569' }
    const items = data.map(s => ({ name: s.severity, value: s.count, itemStyle: { color: SEV_COLORS[s.severity] || '#64748b' } }))
    if (viz === 'treemap') return buildTreemap(items, i => SEV_COLORS[i.name] || '#64748b')
    if (viz === 'funnel')  return buildFunnel(items)
    if (viz === 'bar') {
      return {
        backgroundColor: c.bg, grid: baseGrid(),
        xAxis: xCat(data.map(s => s.severity)), yAxis: yVal('count'),
        tooltip: tooltip(),
        series: [{ type: 'bar', data: data.map(s => ({ value: s.count, itemStyle: { color: SEV_COLORS[s.severity] || '#64748b', borderRadius: [2, 2, 0, 0] } })) }],
      }
    }
    // Default: pie (donut)
    return {
      backgroundColor: c.bg,
      tooltip: { ...tooltip('item'), formatter: '{b}: {c} ({d}%)' },
      legend: { orient: 'vertical', right: 0, top: 'middle', textStyle: { color: c.axis, fontSize: 10 }, itemWidth: 10, itemHeight: 10 },
      series: [{
        type: 'pie', radius: ['45%', '70%'], center: ['40%', '50%'],
        data: items, label: { show: false }, emphasis: { scale: true, scaleSize: 4 },
      }],
    }
  }

  // ── Service map ──────────────────────────────────────────────────────
  if (returns === 'service-map') {
    const { nodes = [], edges = [] } = data
    const palette  = ['#6366f1', '#10b981', '#f59e0b', '#7dd3fc', '#a78bfa', '#34d399', '#fb923c', '#f472b6']
    const nodeColor = (name) => { let h = 0; for (const ch of name) h = (h * 31 + ch.charCodeAt(0)) & 0xffff; return palette[h % palette.length] }

    if (viz === 'sankey') {
      if (!edges.length) return null
      return {
        backgroundColor: c.bg,
        tooltip: { ...tooltip('item'), formatter: p => p.dataType === 'edge' ? `${p.data.source} → ${p.data.target}<br/><b>${p.data.value} calls</b>` : p.name },
        series: [{
          type: 'sankey', layout: 'none',
          nodeGap: 20, nodeWidth: 14,
          data: nodes.map(n => ({ name: n.label, itemStyle: { color: nodeColor(n.id) } })),
          links: edges.map(e => ({ source: e.source, target: e.target, value: e.calls || 1 })),
          label: { color: c.axis, fontSize: 10 },
          lineStyle: { opacity: 0.4, curveness: 0.5 },
          emphasis: { focus: 'adjacency' },
        }],
      }
    }
    // graph (default)
    const maxEdge = Math.max(...edges.map(e => e.calls || 1), 1)
    return {
      backgroundColor: c.bg,
      tooltip: { ...tooltip('item'), formatter: p => p.dataType === 'edge' ? `${p.data.source}→${p.data.target}<br/>${p.data.calls} calls` : p.name },
      series: [{
        type: 'graph', layout: 'force', roam: true, draggable: true,
        force: { repulsion: 140, edgeLength: [60, 160] },
        label: { show: true, position: 'bottom', fontSize: 10, color: c.axis },
        lineStyle: { color: c.grid, curveness: 0.2 },
        edgeSymbol: ['none', 'arrow'], edgeSymbolSize: 6,
        nodes: nodes.map(n => ({ id: n.id, name: n.label, symbolSize: 22, itemStyle: { color: nodeColor(n.id) } })),
        edges: edges.map(e => ({ source: e.source, target: e.target, calls: e.calls, lineStyle: { width: Math.max(1, e.calls / maxEdge * 5) } })),
      }],
    }
  }

  // ── Services health ──────────────────────────────────────────────────
  if (returns === 'services-health') {
    if (viz === 'bar') {
      const c   = ct()
      const svcs = data.map(s => s.service_name)
      const errPct = data.map(s => s.total > 0 ? +(s.errors / s.total * 100).toFixed(2) : 0)
      return {
        backgroundColor: c.bg, grid: { ...baseGrid(), top: 24 },
        legend: { top: 2, right: 0, textStyle: { color: c.axisDim, fontSize: 10 }, itemWidth: 10, itemHeight: 10 },
        xAxis: xCat(svcs), yAxis: [yVal('%'), { ...yVal('ms'), position: 'right', splitLine: { show: false } }],
        tooltip: tooltip(),
        series: [
          { name: 'Err%', type: 'bar', data: errPct, itemStyle: { color: 'rgba(239,68,68,.7)', borderRadius: [2, 2, 0, 0] } },
          { name: 'P95ms', type: 'bar', yAxisIndex: 1, data: data.map(s => +s.p95_ms.toFixed(1)), itemStyle: { color: 'rgba(99,102,241,.7)', borderRadius: [2, 2, 0, 0] } },
        ],
      }
    }
    if (viz === 'radar') {
      const svcs = data.slice(0, 6)
      const maxErr = Math.max(...svcs.map(s => s.total > 0 ? s.errors / s.total * 100 : 0), 1)
      const maxRps = Math.max(...svcs.map(s => s.req_s || 0), 1)
      const maxP95 = Math.max(...svcs.map(s => s.p95_ms || 0), 1)
      return {
        backgroundColor: c.bg,
        legend: { top: 2, right: 0, textStyle: { color: c.axisDim, fontSize: 9 }, itemWidth: 8, itemHeight: 8 },
        tooltip: tooltip('item'),
        radar: {
          indicator: [{ name: 'Err%', max: maxErr }, { name: 'Req/s', max: maxRps }, { name: 'P95ms', max: maxP95 }],
          axisName: { color: c.axis, fontSize: 10 },
          splitLine: { lineStyle: { color: c.grid } },
          axisLine: { lineStyle: { color: c.grid } },
        },
        series: [{
          type: 'radar',
          data: svcs.map((s, i) => {
            const palette = ['#6366f1', '#10b981', '#f59e0b', '#ef4444', '#7dd3fc', '#a78bfa']
            const col     = palette[i % palette.length]
            return { name: s.service_name, value: [s.total > 0 ? +(s.errors / s.total * 100).toFixed(2) : 0, s.req_s || 0, +s.p95_ms.toFixed(1)], areaStyle: { color: col + '22' }, lineStyle: { color: col }, itemStyle: { color: col } }
          }),
        }],
      }
    }
  }

  // ── Resources ────────────────────────────────────────────────────────
  if (returns === 'resources') {
    if (viz === 'heatmap') {
      // Flatten: services × metrics → last value heatmap
      const svcs    = Object.keys(data)
      const metrics = [...new Set(svcs.flatMap(s => Object.keys(data[s])))]
      const vals    = []
      svcs.forEach((s, si) => {
        metrics.forEach((m, mi) => {
          const pts = data[s][m] || []
          const last = pts[pts.length - 1]
          vals.push([si, mi, last?.value != null ? +Number(last.value).toFixed(3) : '-'])
        })
      })
      return {
        backgroundColor: c.bg,
        grid: { top: 10, right: 10, bottom: 10, left: 10, containLabel: true },
        tooltip: { ...tooltip('item'), formatter: p => `${svcs[p.value[0]]}<br/>${metrics[p.value[1]]}<br/><b>${p.value[2]}</b>` },
        xAxis: { type: 'category', data: svcs, axisLabel: { color: c.axisDim, fontSize: 9 }, splitArea: { show: true } },
        yAxis: { type: 'category', data: metrics, axisLabel: { color: c.axisDim, fontSize: 9 }, splitArea: { show: true } },
        visualMap: { min: 0, max: 100, calculable: true, orient: 'horizontal', bottom: 0, left: 'center', textStyle: { color: c.axisDim, fontSize: 9 }, inRange: { color: ['#1e2d45', '#6366f1'] } },
        series: [{ type: 'heatmap', data: vals, label: { show: false } }],
      }
    }
    if (viz === 'line' || viz === 'area') {
      // First service, all metrics as separate series
      const svc  = Object.keys(data)[0]
      if (!svc) return null
      const metricsData = data[svc]
      const palette = ['#6366f1', '#10b981', '#f59e0b', '#ef4444', '#7dd3fc', '#a78bfa', '#fb923c']
      const allTimes = [...new Set(Object.values(metricsData).flatMap(pts => pts.map(p => p.time)))].sort()
      return {
        backgroundColor: c.bg,
        grid: { ...baseGrid(), top: 24 },
        legend: { top: 2, right: 0, textStyle: { color: c.axisDim, fontSize: 9 }, itemWidth: 8, itemHeight: 3 },
        xAxis: xCat(allTimes.map(t => fmtLabel(t, result.value.range))),
        yAxis: yVal(),
        tooltip: tooltip(),
        series: Object.entries(metricsData).map(([metric, pts], i) => {
          const vals = allTimes.map(t => { const p = pts.find(x => x.time === t); return p?.value != null ? +Number(p.value).toFixed(3) : null })
          return mkLine(metric, vals, palette[i % palette.length], viz === 'area')
        }),
      }
    }
  }

  // ── Traces scatter (duration vs time) ────────────────────────────────
  if (returns === 'traces' && viz === 'scatter') {
    return {
      backgroundColor: c.bg, grid: baseGrid(),
      xAxis: xCat(data.map(t => new Date(t.start_time).toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' }))),
      yAxis: yVal('ms'),
      tooltip: {
        ...tooltip('item'),
        formatter: p => {
          const t = data[p.dataIndex]
          return `${t.operation_name}<br/><b>${(t.duration_ns / 1e6).toFixed(2)} ms</b><br/>${t.service_name}`
        },
      },
      series: [{
        type: 'scatter',
        data: data.map(t => ({
          value: +(t.duration_ns / 1e6).toFixed(2),
          itemStyle: { color: t.status_code === 2 ? '#ef4444' : '#10b981' },
        })),
        symbolSize: 6,
      }],
    }
  }

  if (returns === 'traces' && viz === 'bar') {
    const bySvc = {}
    for (const t of data) {
      if (!bySvc[t.service_name]) bySvc[t.service_name] = { ok: 0, err: 0 }
      t.status_code === 2 ? bySvc[t.service_name].err++ : bySvc[t.service_name].ok++
    }
    const svcs = Object.keys(bySvc)
    return {
      backgroundColor: c.bg, grid: baseGrid(),
      xAxis: xCat(svcs), yAxis: yVal(),
      tooltip: { ...tooltip(), axisPointer: { type: 'shadow' } },
      legend: { top: 2, right: 0, textStyle: { color: c.axisDim, fontSize: 9 }, itemWidth: 8, itemHeight: 8 },
      series: [
        { name: 'OK',    type: 'bar', stack: 's', data: svcs.map(s => bySvc[s].ok),  itemStyle: { color: 'rgba(16,185,129,.7)', borderRadius: [0, 0, 0, 0] } },
        { name: 'Error', type: 'bar', stack: 's', data: svcs.map(s => bySvc[s].err), itemStyle: { color: 'rgba(239,68,68,.7)',  borderRadius: [2, 2, 0, 0] } },
      ],
    }
  }

  // ── Logs bar ─────────────────────────────────────────────────────────
  if (returns === 'logs' && (viz === 'bar' || viz === 'pie')) {
    const SEV_COLORS = { ERROR: '#ef4444', FATAL: '#dc2626', WARN: '#f59e0b', INFO: '#6366f1', DEBUG: '#64748b', TRACE: '#475569' }
    const bySev = {}
    for (const l of data) bySev[l.severity_text] = (bySev[l.severity_text] || 0) + 1
    const items = Object.entries(bySev).map(([sev, cnt]) => ({ name: sev, value: cnt, itemStyle: { color: SEV_COLORS[sev] || '#64748b' } }))
    if (viz === 'pie') return {
      backgroundColor: c.bg,
      tooltip: { ...tooltip('item'), formatter: '{b}: {c} ({d}%)' },
      legend: { orient: 'vertical', right: 0, top: 'middle', textStyle: { color: c.axis, fontSize: 10 }, itemWidth: 10, itemHeight: 10 },
      series: [{ type: 'pie', radius: ['45%', '70%'], center: ['40%', '50%'], data: items, label: { show: false }, emphasis: { scale: true, scaleSize: 4 } }],
    }
    return {
      backgroundColor: c.bg, grid: baseGrid(),
      xAxis: xCat(Object.keys(bySev)), yAxis: yVal('count'),
      tooltip: tooltip(),
      series: [{ type: 'bar', data: items.map(i => ({ value: i.value, itemStyle: i.itemStyle, borderRadius: [2, 2, 0, 0] })) }],
    }
  }

  return null
}

// ─── Shared chart builders ────────────────────────────────────────────────
function buildGauge(value, color, label) {
  const c = ct()
  return {
    backgroundColor: c.bg,
    series: [{
      type: 'gauge',
      radius: '88%',
      startAngle: 200, endAngle: -20,
      min: 0, max: Math.max(value * 1.5, 1),
      splitNumber: 4,
      axisLine: { lineStyle: { width: 12, color: [[value / Math.max(value * 1.5, 1), color], [1, c.grid]] } },
      pointer: { show: false },
      axisTick: { show: false },
      splitLine: { show: false },
      axisLabel: { show: false },
      detail: { valueAnimation: true, fontSize: 26, fontFamily: 'monospace', color, offsetCenter: [0, '15%'], formatter: v => v.toFixed(2) },
      data: [{ value, name: label }],
      title: { fontSize: 11, color: c.axisDim, offsetCenter: [0, '38%'] },
    }],
  }
}

function buildTreemap(items, colorFn) {
  const c = ct()
  return {
    backgroundColor: c.bg,
    tooltip: { ...tooltip('item'), formatter: p => `${p.name}<br/><b>${p.value}</b>` },
    series: [{
      type: 'treemap',
      data: items.map(i => ({ ...i, itemStyle: { color: typeof colorFn === 'function' ? colorFn(i) : colorFn } })),
      label: { fontSize: 10, color: '#fff' },
      breadcrumb: { show: false },
      roam: false,
    }],
  }
}

function buildPie(items) {
  const palette = ['#6366f1', '#10b981', '#f59e0b', '#ef4444', '#7dd3fc', '#a78bfa', '#fb923c', '#f472b6', '#34d399']
  const c = ct()
  return {
    backgroundColor: c.bg,
    tooltip: { ...tooltip('item'), formatter: '{b}: {c} ({d}%)' },
    legend: { orient: 'vertical', right: 0, top: 'middle', textStyle: { color: c.axis, fontSize: 10 }, itemWidth: 10, itemHeight: 10 },
    series: [{
      type: 'pie', radius: ['40%', '68%'], center: ['40%', '50%'],
      data: items.map((i, idx) => ({ ...i, itemStyle: { color: i.itemStyle?.color || palette[idx % palette.length] } })),
      label: { show: false }, emphasis: { scale: true, scaleSize: 4 },
    }],
  }
}

function buildFunnel(items) {
  const palette = ['#6366f1', '#10b981', '#f59e0b', '#ef4444', '#7dd3fc', '#a78bfa', '#fb923c', '#f472b6']
  const c = ct()
  const sorted = [...items].sort((a, b) => b.value - a.value)
  return {
    backgroundColor: c.bg,
    tooltip: { ...tooltip('item'), formatter: '{b}: {c}' },
    legend: { bottom: 0, textStyle: { color: c.axisDim, fontSize: 10 }, itemWidth: 10, itemHeight: 10 },
    series: [{
      type: 'funnel',
      left: '5%', width: '90%', top: '5%', bottom: '18%',
      sort: 'none', gap: 2,
      label: { position: 'inside', fontSize: 10, color: '#fff' },
      data: sorted.map((i, idx) => ({ name: i.name, value: i.value, itemStyle: { color: i.itemStyle?.color || palette[idx % palette.length] } })),
    }],
  }
}

// ─── Table column definitions ─────────────────────────────────────────────
const tableCols = computed(() => {
  if (!result.value) return []
  const { returns, data } = result.value
  if (!data?.length) return []
  if (returns === 'traces') return [
    { key: 'trace_id',      label: 'Trace ID',   fmt: r => r.trace_id?.slice(0, 14) + '…' },
    { key: 'service_name',  label: 'Service' },
    { key: 'operation_name',label: 'Operation',  fmt: r => r.operation_name?.length > 30 ? r.operation_name.slice(0, 28) + '…' : r.operation_name },
    { key: 'status_code',   label: 'Status',     fmt: r => r.status_code === 2 ? 'Error' : 'OK', style: r => `color:${r.status_code === 2 ? '#ef4444' : '#10b981'}` },
    { key: 'duration_ns',   label: 'Duration',   align: 'right', fmt: r => (r.duration_ns / 1e6).toFixed(2) + ' ms' },
    { key: 'start_time',    label: 'Time',       fmt: r => new Date(r.start_time).toLocaleTimeString('pt-BR') },
  ]
  if (returns === 'logs') return [
    { key: 'timestamp',     label: 'Time',    fmt: r => new Date(r.timestamp).toLocaleTimeString('pt-BR') },
    { key: 'service_name',  label: 'Service' },
    { key: 'severity_text', label: 'Level',   style: r => ({ ERROR:'color:#ef4444', FATAL:'color:#dc2626', WARN:'color:#f59e0b', INFO:'color:#6366f1', DEBUG:'color:#64748b' })[r.severity_text] || '' },
    { key: 'body',          label: 'Message', fmt: r => r.body?.length > 80 ? r.body.slice(0, 78) + '…' : r.body },
  ]
  if (returns === 'top-ops') return [
    { key: 'operation', label: 'Operation' },
    { key: 'service',   label: 'Service' },
    { key: 'count',     label: 'Requests',  align: 'right' },
    { key: 'errors',    label: 'Errors',    align: 'right', style: r => r.errors > 0 ? 'color:#ef4444' : '' },
    { key: '_ep',       label: 'Err%',      align: 'right', fmt: r => r.count > 0 ? (r.errors / r.count * 100).toFixed(1) + '%' : '0%', style: r => r.errors > 0 ? 'color:#ef4444' : '' },
  ]
  if (returns === 'severity') return [
    { key: 'severity', label: 'Level' },
    { key: 'count',    label: 'Count', align: 'right' },
  ]
  if (returns === 'services-health') return [
    { key: 'service_name', label: 'Service' },
    { key: 'req_s',        label: 'Req/s',   align: 'right' },
    { key: 'p95_ms',       label: 'P95 ms',  align: 'right', fmt: r => r.p95_ms?.toFixed(1) },
    { key: '_ep',          label: 'Err%',    align: 'right', fmt: r => r.total > 0 ? (r.errors / r.total * 100).toFixed(1) + '%' : '0%', style: r => r.errors / r.total > 0.05 ? 'color:#ef4444' : '' },
  ]
  if (returns === 'resources') return [
    { key: 'service', label: 'Service' },
    { key: 'metric',  label: 'Metric' },
    { key: 'value',   label: 'Latest', align: 'right' },
  ]
  const keys = Object.keys(data[0] || {}).slice(0, 6)
  return keys.map(k => ({ key: k, label: k }))
})

const tableRows = computed(() => {
  if (!result.value) return []
  const { returns, data } = result.value
  if (returns === 'resources') {
    const rows = []
    for (const [svc, metrics] of Object.entries(data))
      for (const [metric, pts] of Object.entries(metrics)) {
        const last = Array.isArray(pts) ? pts[pts.length - 1] : null
        rows.push({ service: svc, metric, value: last?.value != null ? Number(last.value).toFixed(3) : '–' })
      }
    return rows
  }
  return data
})

// ─── Stat ─────────────────────────────────────────────────────────────────
const statValue = computed(() => {
  const r = result.value
  if (!r || !r.data?.length) return '–'
  const v = lastVal(r.data, r.returns)
  if (v == null) return r.data.length
  if (r.returns === 'timeseries') {
    const kind  = metricKind(r.data)
    const isRate = kind === 'histogram' || kind === 'sum'
    return fmtMetricVal(v, r.metricName || '', isRate ? 'sum' : 'gauge')
  }
  if (Math.abs(v) >= 1e6) return (v / 1e6).toFixed(2) + 'M'
  if (Math.abs(v) >= 1e3) return (v / 1e3).toFixed(2) + 'K'
  return Number(v).toFixed(v % 1 === 0 ? 0 : 2)
})
const statLabel = computed(() => {
  const r = result.value
  if (!r) return ''
  if (r.returns === 'throughput') return 'req / min'
  return r.returns
})

// ─── Run query ────────────────────────────────────────────────────────────
async function run() {
  const q = (props.query || '').trim()
  if (!q) { result.value = null; initialized.value = false; error.value = ''; empty.value = false; return }
  const parsed = parse(q)
  if (parsed.error) { error.value = parsed.error; result.value = null; initialized.value = false; return }
  // Dashboard-level range overrides the panel's embedded range
  if (props.dashRange) parsed.params.range = props.dashRange
  // Keep old result visible while fetching so the chart stays mounted and
  // the ECharts instance is preserved — prevents animation replay on refresh
  loading.value = true; error.value = ''
  try {
    const res = await execute(parsed)
    const d = res.data
    empty.value = !d || (Array.isArray(d) ? d.length === 0 : Object.keys(d).length === 0)
    const wasChart = result.value && !['stat', 'table'].includes(result.value.viz)
    result.value = res
    initialized.value = true
    const nonCanvas = ['stat', 'table']
    if (!nonCanvas.includes(result.value?.viz)) {
      await nextTick()
      renderChart(!wasChart)
    }
  } catch (e) {
    error.value = e.message || 'Request failed'
  } finally {
    loading.value = false
  }
}

function renderChart(firstRender = false) {
  if (!canvasEl.value) return
  const opt = buildOption()
  if (!opt) { empty.value = true; return }
  if (!chart) chart = echarts.init(canvasEl.value, null, { renderer: 'canvas' })
  // On refresh (not first render) disable animation so data updates silently
  if (firstRender) {
    chart.setOption(opt, { notMerge: true })
  } else {
    chart.setOption({ ...opt, animation: false }, { notMerge: true })
  }
}

// Auto-resize ECharts when the canvas container changes dimensions
let resizeObserver = null
watch(canvasEl, (el) => {
  resizeObserver?.disconnect()
  if (el) {
    resizeObserver = new ResizeObserver(() => chart?.resize())
    resizeObserver.observe(el)
  }
})

watch(() => props.query,     run)
watch(() => props.dashRange, run)
watch(refreshKey, run)
watch(isDark, () => { chart?.dispose(); chart = null; if (result.value && !['stat','table'].includes(result.value.viz)) nextTick(() => renderChart(false)) })
onMounted(run)
onUnmounted(() => { resizeObserver?.disconnect(); chart?.dispose() })
</script>

<style scoped>
.viz-wrap       { width: 100%; height: 100%; background: var(--telm-bg-row); }
.viz-canvas     { width: 100%; height: 100%; background: var(--telm-bg-row); }
.viz-table-wrap { width: 100%; height: 100%; overflow: auto; background: var(--telm-bg-row); }
</style>

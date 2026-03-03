<template>
  <v-container fluid class="pa-3">

    <!-- Stat strip -->
    <v-card class="mb-3" style="overflow:hidden">
      <div class="d-flex">
        <div
          v-for="(card, i) in statCards"
          :key="card.label"
          class="flex-grow-1 pa-3"
          :style="i < statCards.length - 1 ? 'border-right:1px solid var(--telm-border)' : ''"
        >
          <div class="text-caption text-disabled mb-1" style="white-space:nowrap;font-size:10px">{{ card.label }}</div>
          <div class="mono font-weight-bold" style="font-size:18px;line-height:1" :style="`color:${card.color}`">
            {{ card.value }}
          </div>
        </div>
      </div>
    </v-card>

    <!-- Service Health Row -->
    <div class="d-flex align-center justify-space-between mb-2">
      <span class="text-caption text-disabled" style="font-size:10px;letter-spacing:.08em;text-transform:uppercase">Service Health</span>
    </div>
    <v-row dense class="mb-3">
      <v-col v-for="svc in serviceHealthData" :key="svc.service_name" cols="6" sm="3">
        <v-card class="pa-3">
          <div class="d-flex align-center mb-2">
            <div
              style="width:8px;height:8px;border-radius:50%;flex-shrink:0"
              :style="`background:${healthColor(svcErrRate(svc))}`"
              class="mr-2"
            ></div>
            <span class="mono text-caption font-weight-medium text-truncate" style="font-size:11px">{{ svc.service_name }}</span>
          </div>
          <div class="d-flex" style="gap:12px">
            <div>
              <div class="text-disabled mono" style="font-size:9px;letter-spacing:.06em">ERR%</div>
              <div class="mono font-weight-bold" style="font-size:13px" :style="`color:${healthColor(svcErrRate(svc))}`">
                {{ svcErrRate(svc).toFixed(1) }}%
              </div>
            </div>
            <div>
              <div class="text-disabled mono" style="font-size:9px;letter-spacing:.06em">REQ/S</div>
              <div class="mono font-weight-bold" style="font-size:13px;color:#7dd3fc">{{ svc.req_s }}</div>
            </div>
            <div>
              <div class="text-disabled mono" style="font-size:9px;letter-spacing:.06em">P95</div>
              <div class="mono font-weight-bold" style="font-size:13px;color:#a5b4fc">{{ svc.p95_ms.toFixed(0) }}ms</div>
            </div>
          </div>
        </v-card>
      </v-col>
      <v-col v-if="!serviceHealthData.length" cols="12">
        <v-card class="pa-3">
          <span class="text-caption text-disabled">No service data yet</span>
        </v-card>
      </v-col>
    </v-row>

    <!-- Range selector + section label -->
    <div class="d-flex align-center justify-space-between mb-2">
      <span class="text-caption text-disabled" style="font-size:10px;letter-spacing:.08em;text-transform:uppercase">Overview</span>
      <div class="d-flex gap-1">
        <v-btn
          v-for="r in timeRanges"
          :key="r.v"
          size="x-small"
          :variant="dashRange === r.v ? 'flat' : 'text'"
          :color="dashRange === r.v ? 'primary' : 'default'"
          class="mono px-2"
          style="min-width:32px;font-size:11px"
          @click="dashRange = r.v; load()"
        >{{ r.label }}</v-btn>
      </div>
    </div>

    <!-- Charts 2x2 -->
    <v-row dense class="mb-2">
      <v-col cols="12" lg="6">
        <v-card class="pa-3">
          <div class="d-flex align-center justify-space-between mb-2">
            <span class="text-caption text-medium-emphasis">Throughput</span>
            <span class="text-caption text-disabled mono" style="font-size:10px">req/s per bucket</span>
          </div>
          <div ref="throughputEl" style="height:160px"></div>
        </v-card>
      </v-col>
      <v-col cols="12" lg="6">
        <v-card class="pa-3">
          <div class="d-flex align-center justify-space-between mb-2">
            <span class="text-caption text-medium-emphasis">Error rate</span>
            <span class="text-caption text-disabled mono" style="font-size:10px">%</span>
          </div>
          <div ref="errorRateEl" style="height:160px"></div>
        </v-card>
      </v-col>
    </v-row>

    <v-row dense class="mb-2">
      <v-col cols="12" lg="6">
        <v-card class="pa-3">
          <div class="d-flex align-center justify-space-between mb-2">
            <span class="text-caption text-medium-emphasis">Latency percentiles</span>
            <span class="text-caption text-disabled mono" style="font-size:10px">ms</span>
          </div>
          <div ref="latencyEl" style="height:160px"></div>
        </v-card>
      </v-col>
      <v-col cols="12" lg="6">
        <v-card class="pa-3">
          <div class="d-flex align-center justify-space-between mb-2">
            <span class="text-caption text-medium-emphasis">Top operations</span>
            <span class="text-caption text-disabled mono" style="font-size:10px">by count</span>
          </div>
          <div ref="topOpsEl" style="height:160px"></div>
        </v-card>
      </v-col>
    </v-row>

    <!-- Bottom row: pie + table -->
    <v-row dense class="mb-2">
      <v-col cols="12" lg="4">
        <v-card class="pa-3">
          <div class="text-caption text-medium-emphasis mb-2">Log severity</div>
          <div ref="severityEl" style="height:160px"></div>
        </v-card>
      </v-col>
      <v-col cols="12" lg="8">
        <v-card>
          <div class="d-flex align-center px-3 pt-3 pb-2">
            <span class="text-caption text-medium-emphasis">Operations breakdown</span>
          </div>
          <v-table density="compact" hover>
            <thead>
              <tr class="table-header">
                <th class="text-caption">Operation</th>
                <th class="text-caption">Service</th>
                <th class="text-caption text-right">Reqs</th>
                <th class="text-caption text-right">Errors</th>
                <th class="text-caption text-right">Err%</th>
                <th class="text-caption text-right">Avg ms</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(op, i) in stats.topOps" :key="i">
                <td class="mono text-caption text-truncate" style="max-width:200px">{{ op.operation }}</td>
                <td>
                  <v-chip :color="serviceChipColor(op.service)" class="mono">{{ op.service }}</v-chip>
                </td>
                <td class="text-right mono text-caption">{{ op.count.toLocaleString() }}</td>
                <td class="text-right mono text-caption" :class="op.errors > 0 ? 'text-error' : 'text-disabled'">{{ op.errors }}</td>
                <td class="text-right mono text-caption" :class="(op.errors/op.count*100) > 5 ? 'text-error' : 'text-success'">
                  {{ op.count > 0 ? (op.errors/op.count*100).toFixed(1)+'%' : '0%' }}
                </td>
                <td class="text-right mono text-caption">{{ op.avg_ms.toFixed(1) }}</td>
              </tr>
              <tr v-if="!stats.topOps.length">
                <td colspan="6" class="text-center text-disabled py-6 text-caption">No data</td>
              </tr>
            </tbody>
          </v-table>
        </v-card>
      </v-col>
    </v-row>

    <!-- Resource Metrics (sempre visível, por serviço) -->
    <div class="d-flex align-center justify-space-between mb-2 mt-1">
      <span class="text-caption text-disabled" style="font-size:10px;letter-spacing:.08em;text-transform:uppercase">
        Resource Metrics{{ sharedFilters.service ? ' — ' + sharedFilters.service : '' }}
      </span>
    </div>
    <template v-if="Object.keys(allResourcesData).length">
      <v-card
        v-for="(svcRes, svcKey) in allResourcesData"
        :key="svcKey"
        class="pa-3 mb-2"
      >
        <div class="d-flex align-center mb-2">
          <div
            style="width:6px;height:6px;border-radius:50%;flex-shrink:0"
            :style="`background:${svcDotColor(svcKey)}`"
            class="mr-2"
          ></div>
          <span class="mono font-weight-medium" style="font-size:11px">{{ svcKey }}</span>
        </div>
        <v-row dense>
          <v-col cols="6" sm="4" lg="2" v-for="mc in metricCols" :key="mc.key">
            <div class="d-flex align-center justify-space-between" style="margin-bottom:3px">
              <span class="text-disabled mono" style="font-size:9px;letter-spacing:.04em;text-transform:uppercase">{{ mc.label }}</span>
              <span class="mono font-weight-bold" style="font-size:12px;line-height:1" :style="`color:${mc.color}`">
                {{ resLastValues[`${svcKey}:${mc.key}`] || '—' }}
              </span>
            </div>
            <div :ref="el => bindResEl(el, svcKey, mc.key)" style="height:72px"></div>
          </v-col>
        </v-row>
      </v-card>
    </template>
    <v-card v-else class="pa-3 mb-2">
      <span class="text-caption text-disabled">No resource data yet — waiting for metrics to arrive</span>
    </v-card>

    <!-- Service Map (visible when no service filter) -->
    <template v-if="!sharedFilters.service">
      <div class="d-flex align-center justify-space-between mb-2 mt-1">
        <span class="text-caption text-disabled" style="font-size:10px;letter-spacing:.08em;text-transform:uppercase">Service Map</span>
        <div class="d-flex align-center" style="gap:12px">
          <div class="d-flex align-center" style="gap:4px">
            <div style="width:8px;height:8px;border-radius:50%;background:#10b981"></div>
            <span class="text-disabled mono" style="font-size:9px">&lt;1% err</span>
          </div>
          <div class="d-flex align-center" style="gap:4px">
            <div style="width:8px;height:8px;border-radius:50%;background:#f59e0b"></div>
            <span class="text-disabled mono" style="font-size:9px">1–5% err</span>
          </div>
          <div class="d-flex align-center" style="gap:4px">
            <div style="width:8px;height:8px;border-radius:50%;background:#ef4444"></div>
            <span class="text-disabled mono" style="font-size:9px">&gt;5% err</span>
          </div>
        </div>
      </div>
      <v-card class="pa-3 mb-2">
        <div ref="serviceMapEl" style="height:320px"></div>
        <div v-if="!serviceMapData.nodes.length" class="d-flex align-center justify-center" style="height:320px;position:absolute;top:0;left:0;right:0">
          <span class="text-caption text-disabled">No service map data yet</span>
        </div>
      </v-card>
    </template>

  </v-container>
</template>

<script setup>
import { ref, inject, watch, onMounted, onUnmounted, computed, nextTick } from 'vue'
import * as echarts from 'echarts'
import api from '@/plugins/axios'

const sharedFilters = inject('sharedFilters')
const refreshKey    = inject('refreshKey')
const isDark        = inject('isDark')

// Retorna as cores do tema atual — direto do isDark, sem depender de CSS vars no DOM
function ct() {
  const dark = isDark.value
  return {
    axis:        dark ? '#94a3b8' : '#64748b',
    axisDim:     dark ? '#64748b' : '#94a3b8',
    grid:        dark ? '#1e2d45' : '#e2e8f0',
    tooltipBg:   dark ? '#1a2235' : '#ffffff',
    tooltipBdr:  dark ? '#2d3f5c' : '#cbd5e1',
    tooltipText: dark ? '#cbd5e1' : '#0f172a',
  }
}

const dashRange = ref('1h')
const timeRanges = [
  { v: '1h', label: '1h' }, { v: '6h', label: '6h' },
  { v: '24h', label: '24h' }, { v: '7d', label: '7d' },
]

const stats = ref({ throughput: [], errors: [], latency: [], topOps: [], severity: [] })
const serviceHealthData = ref([])
const serviceMapData = ref({ nodes: [], edges: [] })
const allResourcesData = ref({})

const throughputEl = ref(null)
const errorRateEl = ref(null)
const latencyEl = ref(null)
const topOpsEl = ref(null)
const severityEl = ref(null)
const serviceMapEl = ref(null)

const charts = {}
const resEls = {}   // { 'svc:key': domEl }
const resCharts = {} // { 'svc:key': echartsInstance }
const resLastValues = ref({}) // { 'svc:key': formatted string }

// Configuração das 6 colunas de métricas de resource
const metricCols = [
  { key: 'cpu',  label: 'CPU',       unit: '%',    metric: 'process.cpu.usage',       color: '#6366f1', fmt: v => v.toFixed(1) + '%' },
  { key: 'mem',  label: 'Memory',    unit: 'MB',   metric: 'process.memory.bytes',    color: '#10b981', transform: v => v / 1024 / 1024, fmt: v => v.toFixed(0) + ' MB' },
  { key: 'gor',  label: 'Goroutines',unit: 'cnt',  metric: 'runtime.goroutines',      color: '#f59e0b', fmt: v => v.toFixed(0) },
  { key: 'gc',   label: 'GC Pause',  unit: 'ms',   metric: 'runtime.gc.pause_ms',     color: '#ef4444', fmt: v => v.toFixed(2) + ' ms' },
  { key: 'ior',  label: 'IO Read',   unit: 'B',    metric: 'process.io.read_bytes',   color: '#7dd3fc', fmt: v => fmtBytes(v) },
  { key: 'iow',  label: 'IO Write',  unit: 'B',    metric: 'process.io.write_bytes',  color: '#a78bfa', fmt: v => fmtBytes(v) },
]

function bindResEl(el, svc, key) {
  const k = `${svc}:${key}`
  if (el) resEls[k] = el
  else delete resEls[k]
}

function fmtBytes(b) {
  if (b >= 1024 * 1024 * 1024) return (b / (1024 * 1024 * 1024)).toFixed(1) + ' GB'
  if (b >= 1024 * 1024) return (b / (1024 * 1024)).toFixed(1) + ' MB'
  if (b >= 1024) return (b / 1024).toFixed(1) + ' KB'
  return Math.round(b) + ' B'
}

function svcDotColor(name) {
  const palette = ['#6366f1','#10b981','#f59e0b','#ef4444','#7dd3fc','#a78bfa','#34d399','#fb923c']
  let h = 0
  for (const c of (name || '')) h = (h * 31 + c.charCodeAt(0)) & 0xffff
  return palette[h % palette.length]
}

function getOrInit(el) {
  if (!el) return null
  return echarts.getInstanceByDom(el) || echarts.init(el, null, { renderer: 'canvas' })
}

function tooltip(trigger = 'axis') {
  const c = ct()
  return {
    trigger,
    backgroundColor: c.tooltipBg, borderColor: c.tooltipBdr,
    textStyle: { color: c.tooltipText, fontSize: 12 },
  }
}
function xAxis(data) {
  const c = ct()
  return {
    type: 'category', data,
    axisLine: { lineStyle: { color: c.grid } },
    axisTick: { show: false },
    axisLabel: { color: c.axis, fontSize: 11 },
    splitLine: { show: false },
  }
}
function yAxis(name) {
  const c = ct()
  return {
    type: 'value', name,
    nameTextStyle: { color: c.axisDim, fontSize: 11 },
    axisLine: { show: false }, axisTick: { show: false },
    axisLabel: { color: c.axis, fontSize: 11 },
    splitLine: { lineStyle: { color: c.grid, type: 'dashed' } },
  }
}
function grid(opts = {}) {
  return Object.assign({ top: 16, right: 8, bottom: 24, left: 8, containLabel: true }, opts)
}
function fmtLabel(iso) {
  const d = new Date(iso)
  if (dashRange.value === '7d') return d.toLocaleDateString('pt-BR', { month: 'short', day: 'numeric' })
  return d.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' })
}
function bucketSecs() {
  return { '1h': 60, '6h': 3600, '24h': 3600, '7d': 86400 }[dashRange.value] || 60
}

function healthColor(errRate) {
  if (errRate < 1) return '#10b981'
  if (errRate < 5) return '#f59e0b'
  return '#ef4444'
}

function svcErrRate(svc) {
  return svc.total > 0 ? (svc.errors / svc.total * 100) : 0
}

function renderCharts() {
  const d = stats.value

  // Throughput
  const tc = getOrInit(throughputEl.value)
  if (tc) {
    const bs = bucketSecs()
    tc.setOption({
      backgroundColor: 'transparent', grid: grid(),
      xAxis: xAxis(d.throughput.map(p => fmtLabel(p.time))),
      yAxis: yAxis('req/s'),
      tooltip: {
        ...tooltip(),
        formatter: p => `${p[0].name}<br/><b>${p[0].value} req/s</b>`,
      },
      series: [{
        type: 'line', data: d.throughput.map(p => +(p.count / bs).toFixed(3)),
        smooth: 0.4, lineStyle: { color: '#6366f1', width: 1.5 }, itemStyle: { color: '#6366f1' },
        showSymbol: d.throughput.length <= 30, symbolSize: 3,
        areaStyle: { color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: 'rgba(99,102,241,.25)' }, { offset: 1, color: 'rgba(99,102,241,.01)' }] } },
      }],
    })
    charts.thr = tc
  }

  // Error rate
  const ec = getOrInit(errorRateEl.value)
  if (ec) {
    const pct = d.errors.map(p => p.total > 0 ? +(p.errors / p.total * 100).toFixed(2) : 0)
    ec.setOption({
      backgroundColor: 'transparent', grid: grid(),
      xAxis: xAxis(d.errors.map(p => fmtLabel(p.time))), yAxis: yAxis('%'),
      tooltip: tooltip(),
      series: [{
        type: 'line', data: pct, smooth: 0.4,
        lineStyle: { color: '#ef4444', width: 1.5 }, itemStyle: { color: '#ef4444' },
        showSymbol: d.errors.length <= 30, symbolSize: 3,
        areaStyle: { color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: 'rgba(239,68,68,.25)' }, { offset: 1, color: 'rgba(239,68,68,.01)' }] } },
      }],
    })
    charts.err = ec
  }

  // Latency
  const lc = getOrInit(latencyEl.value)
  if (lc) {
    const line = (name, color, data) => ({
      type: 'line', name, data, smooth: 0.4,
      lineStyle: { color, width: 1.5 }, itemStyle: { color },
      showSymbol: d.latency.length <= 30, symbolSize: 3,
    })
    lc.setOption({
      backgroundColor: 'transparent', grid: grid({ top: 24 }),
      legend: { top: 2, right: 0, textStyle: { color: ct().axis, fontSize: 11 }, itemWidth: 12, itemHeight: 3 },
      xAxis: xAxis(d.latency.map(p => fmtLabel(p.time))), yAxis: yAxis('ms'),
      tooltip: tooltip(),
      series: [
        line('P50', '#10b981', d.latency.map(p => +p.p50.toFixed(2))),
        line('P95', '#f59e0b', d.latency.map(p => +p.p95.toFixed(2))),
        line('P99', '#ef4444', d.latency.map(p => +p.p99.toFixed(2))),
      ],
    })
    charts.lat = lc
  }

  // Top ops
  const oc = getOrInit(topOpsEl.value)
  if (oc) {
    const rev = [...d.topOps].reverse()
    const labels = rev.map(o => o.operation.length > 28 ? o.operation.slice(0, 26) + '…' : o.operation)
    oc.setOption({
      backgroundColor: 'transparent', grid: { top: 4, right: 8, bottom: 24, left: 8, containLabel: true },
      legend: { bottom: 0, textStyle: { color: ct().axis, fontSize: 11 }, itemWidth: 10, itemHeight: 10 },
      tooltip: { ...tooltip('axis'), axisPointer: { type: 'shadow' } },
      xAxis: { type: 'value', axisLabel: { color: ct().axis, fontSize: 11 }, splitLine: { lineStyle: { color: ct().grid, type: 'dashed' } }, axisLine: { show: false }, axisTick: { show: false } },
      yAxis: { type: 'category', data: labels, axisLabel: { color: ct().axis, fontSize: 11 }, axisLine: { lineStyle: { color: ct().grid } }, axisTick: { show: false }, splitLine: { show: false } },
      series: [
        { name: 'Success', type: 'bar', stack: 'total', data: rev.map(o => o.count - o.errors), itemStyle: { color: 'rgba(16,185,129,.7)', borderRadius: [0, 2, 2, 0] } },
        { name: 'Error', type: 'bar', stack: 'total', data: rev.map(o => o.errors), itemStyle: { color: 'rgba(239,68,68,.7)', borderRadius: [0, 2, 2, 0] } },
      ],
    })
    charts.ops = oc
  }

  // Severity
  const sc = getOrInit(severityEl.value)
  if (sc) {
    const colorMap = {
      ERROR: 'rgba(239,68,68,.85)', FATAL: 'rgba(239,68,68,.95)',
      WARN: 'rgba(245,158,11,.85)', INFO: 'rgba(99,102,241,.85)',
      DEBUG: 'rgba(100,116,139,.7)', TRACE: 'rgba(100,116,139,.5)', UNKNOWN: 'rgba(51,65,85,.7)',
    }
    sc.setOption({
      backgroundColor: 'transparent',
      tooltip: { ...tooltip('item'), formatter: '{b}: {c} ({d}%)' },
      legend: { orient: 'vertical', right: 0, top: 'middle', textStyle: { color: ct().axis, fontSize: 11 }, itemWidth: 10, itemHeight: 10 },
      series: [{
        type: 'pie', radius: ['50%', '74%'], center: ['38%', '50%'],
        data: d.severity.map(s => ({ name: s.severity, value: s.count, itemStyle: { color: colorMap[s.severity] || 'rgba(100,116,139,.6)' } })),
        label: { show: false }, emphasis: { scale: true, scaleSize: 5 },
      }],
    })
    charts.sev = sc
  }
}

function sparkline(el, data, color, valueFormatter) {
  const c = getOrInit(el)
  if (!c) return null
  // Mostra no máximo 3 labels no eixo X (início, meio, fim)
  const labelInterval = data.length > 3 ? Math.floor(data.length / 2) - 1 : 0
  const c2 = ct()
  c.setOption({
    backgroundColor: 'transparent',
    grid: { top: 2, right: 2, bottom: 18, left: 2, containLabel: false },
    xAxis: {
      type: 'category',
      data: data.map(p => fmtLabel(p.time)),
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: { color: c2.axisDim, fontSize: 9, interval: labelInterval, overflow: 'truncate' },
      splitLine: { show: false },
    },
    yAxis: {
      type: 'value',
      axisLine: { show: false }, axisTick: { show: false },
      axisLabel: { show: false },
      splitLine: { show: false },
    },
    tooltip: {
      trigger: 'axis',
      backgroundColor: c2.tooltipBg, borderColor: c2.tooltipBdr,
      textStyle: { color: c2.tooltipText, fontSize: 11 },
      formatter: p => `${p[0].name}<br/><b style="font-size:13px">${valueFormatter ? valueFormatter(p[0].value) : p[0].value}</b>`,
    },
    series: [{
      type: 'line',
      data: data.map(p => p.value),
      smooth: 0.4,
      lineStyle: { color, width: 2 }, itemStyle: { color },
      showSymbol: false,
      areaStyle: { color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: color.replace(')', ',.3)').replace('rgb', 'rgba') }, { offset: 1, color: color.replace(')', ',.02)').replace('rgb', 'rgba') }] } },
    }],
  })
  return c
}

function renderAllResources() {
  const data = allResourcesData.value
  const lastVals = {}
  for (const [svcName, svcData] of Object.entries(data)) {
    for (const mc of metricCols) {
      const el = resEls[`${svcName}:${mc.key}`]
      if (!el) continue
      let pts = svcData[mc.metric] || []
      if (mc.transform) pts = pts.map(p => ({ ...p, value: mc.transform(p.value) }))
      const k = `${svcName}:${mc.key}`
      if (pts.length > 0) {
        const last = pts[pts.length - 1].value
        lastVals[k] = mc.fmt ? mc.fmt(last) : String(last)
      }
      resCharts[k] = sparkline(el, pts, mc.color, mc.fmt)
    }
  }
  resLastValues.value = lastVals
}

function renderServiceMap() {
  const mc = getOrInit(serviceMapEl.value)
  if (!mc) return
  charts.map = mc

  const { nodes, edges } = serviceMapData.value
  if (!nodes.length) {
    mc.setOption({ series: [] })
    return
  }

  const c = ct()
  const chartNodes = nodes.map(n => ({
    id: n.id,
    name: n.label,
    symbolSize: Math.max(22, Math.min(60, n.req_s * 8 + 22)),
    itemStyle: { color: healthColor(n.error_rate), borderColor: isDark.value ? 'rgba(255,255,255,.15)' : 'rgba(0,0,0,.1)', borderWidth: 1 },
    label: { show: true, color: c.tooltipText, fontSize: 10, fontFamily: 'ui-monospace,monospace' },
  }))

  const chartEdges = edges.map(e => {
    const errRate = e.calls > 0 ? e.errors / e.calls * 100 : 0
    return {
      source: e.source,
      target: e.target,
      value: e.calls,
      lineStyle: {
        width: Math.max(1, Math.min(6, e.calls / 500)),
        color: healthColor(errRate),
        opacity: 0.6,
        curveness: 0.2,
      },
      tooltip: {
        formatter: `${e.source} → ${e.target}<br/>calls: ${e.calls.toLocaleString()}<br/>errors: ${e.errors}<br/>avg: ${e.avg_ms.toFixed(1)}ms`,
      },
    }
  })

  mc.setOption({
    backgroundColor: 'transparent',
    tooltip: { ...tooltip('item') },
    series: [{
      type: 'graph',
      layout: 'force',
      data: chartNodes,
      edges: chartEdges,
      roam: true,
      force: { repulsion: 200, edgeLength: [100, 200], gravity: 0.1, layoutAnimation: true },
      emphasis: { focus: 'adjacency', scale: true, scaleSize: 8 },
      label: { show: true, position: 'bottom', distance: 4 },
      edgeLabel: { show: false },
      lineStyle: { curveness: 0.2 },
    }],
  })
}

function dashRangeParam() {
  const to = new Date()
  const mins = { '1h': 60, '6h': 360, '24h': 1440, '7d': 10080 }
  const from = new Date(to - (mins[dashRange.value] || 60) * 60_000)
  return { from: from.toISOString(), to: to.toISOString() }
}

async function load() {
  const { from, to } = dashRangeParam()
  const svc = sharedFilters.service || ''
  const params = { service: svc, from, to }

  const requests = [
    api.get('/stats/throughput', { params }).then(r => r.data),
    api.get('/stats/errors', { params }).then(r => r.data),
    api.get('/stats/latency', { params }).then(r => r.data),
    api.get('/stats/top-ops', { params }).then(r => r.data),
    api.get('/stats/severity', { params }).then(r => r.data),
    api.get('/stats/services-health', { params }).then(r => r.data),
    api.get('/stats/service-map', { params: { from, to } }).then(r => r.data),
  ]

  const [throughput, errors, latency, topOps, severity, health, svcMap, allRes] = await Promise.all([
    ...requests,
    api.get('/stats/resources/all', { params }).then(r => r.data),
  ])

  stats.value = { throughput, errors, latency, topOps, severity }
  serviceHealthData.value = health
  serviceMapData.value = svcMap
  allResourcesData.value = allRes

  await nextTick()
  renderCharts()
  renderServiceMap()
  renderAllResources()
}

const statCards = computed(() => {
  const total = stats.value.throughput.reduce((s, d) => s + d.count, 0)
  const rangeSecs = { '1h': 3600, '6h': 21600, '24h': 86400, '7d': 604800 }[dashRange.value] || 3600
  const rps = (total / rangeSecs).toFixed(2)
  const rpm = (parseFloat(rps) * 60).toFixed(1)

  const errTotal = stats.value.errors.reduce((s, d) => s + d.total, 0)
  const errCount = stats.value.errors.reduce((s, d) => s + d.errors, 0)
  const errRate = errTotal > 0 ? (errCount / errTotal * 100).toFixed(1) : '0.0'

  const pts = stats.value.latency.filter(d => d.p95 > 0)
  const p95 = pts.length ? (pts.reduce((s, d) => s + d.p95, 0) / pts.length).toFixed(1) : '0.0'
  const logTotal = stats.value.severity.reduce((s, d) => s + d.count, 0)

  return [
    { label: 'Total requests', value: total.toLocaleString(), color: '#a5b4fc' },
    { label: 'Req / s', value: rps, color: '#7dd3fc' },
    { label: 'Req / min', value: rpm, color: '#7dd3fc' },
    { label: 'Error rate', value: errRate + '%', color: parseFloat(errRate) > 5 ? '#f87171' : '#34d399' },
    { label: 'Avg P95 latency', value: p95 + ' ms', color: parseFloat(p95) > 500 ? '#fbbf24' : '#34d399' },
    { label: 'Log events', value: logTotal.toLocaleString(), color: '#a5b4fc' },
  ]
})

function serviceChipColor(name) {
  const colors = ['indigo', 'purple', 'blue', 'teal', 'green', 'cyan', 'orange', 'pink']
  let h = 0
  for (const c of (name || '')) h = (h * 31 + c.charCodeAt(0)) & 0xffff
  return colors[h % colors.length]
}

onMounted(async () => {
  await nextTick()
  await load()
  window.addEventListener('resize', () => Object.values(charts).forEach(c => c?.resize()))
})

watch(refreshKey, load)

// Re-renderiza charts quando o tema muda
watch(isDark, async () => {
  await nextTick()
  renderCharts()
  renderServiceMap()
  renderAllResources()
})

onUnmounted(() => {
  Object.values(charts).forEach(c => c?.dispose())
  Object.values(resCharts).forEach(c => c?.dispose())
})
</script>

<style scoped>
.mono { font-family: ui-monospace, 'JetBrains Mono', monospace; }
.table-header th {
  background: var(--telm-bg-header) !important;
  color: var(--telm-text-3) !important;
  font-size: 10px !important;
  letter-spacing: .06em;
  text-transform: uppercase;
}
</style>

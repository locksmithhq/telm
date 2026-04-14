<template>
  <v-container fluid class="pa-3">

    <!-- KPI strip -->
    <v-card class="mb-3" style="overflow:hidden">
      <div class="d-flex flex-wrap">
        <div
          v-for="(card, i) in kpiCards"
          :key="card.label"
          class="flex-grow-1 pa-3"
          :style="i < kpiCards.length - 1 ? 'border-right:1px solid var(--telm-border)' : ''"
          style="min-width:160px"
        >
          <div class="text-caption text-disabled mb-1 d-flex align-center" style="font-size:10px;gap:4px">
            <v-icon size="11" :color="card.color">{{ card.icon }}</v-icon>
            {{ card.label }}
          </div>
          <div class="mono font-weight-bold" style="font-size:20px;line-height:1.1" :style="`color:${card.color}`">
            {{ card.value }}
          </div>
          <div class="mono text-disabled" style="font-size:10px;margin-top:2px">{{ card.sub }}</div>
        </div>
      </div>
    </v-card>

    <!-- Range selector -->
    <div class="d-flex align-center justify-space-between mb-2">
      <span class="text-caption text-disabled" style="font-size:10px;letter-spacing:.08em;text-transform:uppercase">
        Ingestion Rate
      </span>
      <div class="d-flex gap-1">
        <v-btn
          v-for="r in timeRanges"
          :key="r.v"
          size="x-small"
          :variant="range === r.v ? 'flat' : 'text'"
          :color="range === r.v ? 'primary' : 'default'"
          class="mono px-2"
          style="min-width:32px;font-size:11px"
          @click="range = r.v; loadGrowth()"
        >{{ r.label }}</v-btn>
      </div>
    </div>

    <!-- Charts row: donut + stacked area -->
    <v-row dense class="mb-3">
      <v-col cols="12" md="4">
        <v-card class="pa-3" style="height:240px">
          <div class="d-flex align-center justify-space-between mb-1">
            <span class="text-caption text-medium-emphasis">Distribuição</span>
            <span class="mono text-disabled" style="font-size:10px">% do banco</span>
          </div>
          <div ref="donutEl" style="height:190px"></div>
        </v-card>
      </v-col>
      <v-col cols="12" md="8">
        <v-card class="pa-3" style="height:240px">
          <div class="d-flex align-center justify-space-between mb-1">
            <span class="text-caption text-medium-emphasis">Rows ingeridos por bucket</span>
            <span class="mono text-disabled" style="font-size:10px">{{ range }}</span>
          </div>
          <div ref="growthEl" style="height:190px"></div>
        </v-card>
      </v-col>
    </v-row>

    <!-- Per-table detail cards -->
    <div class="text-caption text-disabled mb-2" style="font-size:10px;letter-spacing:.08em;text-transform:uppercase">
      Detalhes por Tabela
    </div>
    <v-row dense class="mb-3">
      <v-col v-for="tbl in tableCards" :key="tbl.name" cols="12" md="4">
        <v-card class="pa-3">
          <div class="d-flex align-center mb-3">
            <div
              style="width:8px;height:8px;border-radius:50%;flex-shrink:0"
              :style="`background:${tbl.color}`"
              class="mr-2"
            ></div>
            <span class="mono font-weight-bold" style="font-size:13px">{{ tbl.name }}</span>
            <v-spacer />
            <span class="mono font-weight-bold" style="font-size:15px" :style="`color:${tbl.color}`">
              {{ tbl.size }}
            </span>
          </div>

          <div class="d-flex flex-wrap" style="gap:12px 20px">
            <div v-for="stat in tbl.stats" :key="stat.label">
              <div class="text-disabled mono" style="font-size:9px;letter-spacing:.06em;text-transform:uppercase">{{ stat.label }}</div>
              <div class="mono font-weight-medium" style="font-size:12px" :style="stat.color ? `color:${stat.color}` : ''">
                {{ stat.value }}
              </div>
            </div>
          </div>

          <!-- Retention bar -->
          <div class="mt-3">
            <div class="d-flex justify-space-between mb-1">
              <span class="mono text-disabled" style="font-size:9px">Retenção {{ tbl.retentionDays }}d</span>
              <span class="mono" style="font-size:9px" :style="`color:${tbl.retentionColor}`">
                {{ tbl.retentionLabel }}
              </span>
            </div>
            <v-progress-linear
              :model-value="tbl.retentionPct"
              :color="tbl.retentionColor"
              bg-color="surface-variant"
              height="4"
              rounded
            />
          </div>
        </v-card>
      </v-col>
    </v-row>

    <!-- Insights -->
    <div class="text-caption text-disabled mb-2" style="font-size:10px;letter-spacing:.08em;text-transform:uppercase">
      Insights
    </div>
    <v-card class="mb-3">
      <div v-if="!insights.length" class="pa-4 text-caption text-disabled">
        Sem dados suficientes para gerar insights.
      </div>
      <v-list density="compact" class="pa-0">
        <v-list-item
          v-for="(ins, i) in insights"
          :key="i"
          :style="i < insights.length - 1 ? 'border-bottom:1px solid var(--telm-border)' : ''"
          class="py-2 px-4"
        >
          <template #prepend>
            <v-icon :color="insColor(ins.level)" size="16" class="mr-3">{{ insIcon(ins.level) }}</v-icon>
          </template>
          <v-list-item-title class="text-caption" style="white-space:normal;line-height:1.4">
            {{ ins.msg }}
          </v-list-item-title>
          <template v-if="ins.detail" #subtitle>
            <span class="mono text-disabled" style="font-size:10px">{{ ins.detail }}</span>
          </template>
        </v-list-item>
      </v-list>
    </v-card>

    <!-- API Keys -->
    <div class="d-flex align-center justify-space-between mb-2">
      <span class="text-caption text-disabled" style="font-size:10px;letter-spacing:.08em;text-transform:uppercase">
        API Keys
      </span>
      <v-btn size="x-small" variant="tonal" color="primary" prepend-icon="mdi-plus" @click="newKeyDialog = true">
        Nova Key
      </v-btn>
    </div>

    <v-card class="mb-3">
      <div v-if="!apiKeys.length" class="pa-4 text-caption text-disabled">
        Nenhuma API key criada. Crie uma para enviar telemetria via HTTP.
      </div>
      <v-list density="compact" class="pa-0">
        <v-list-item
          v-for="(k, i) in apiKeys"
          :key="k.id"
          :style="i < apiKeys.length - 1 ? 'border-bottom:1px solid var(--telm-border)' : ''"
          class="py-2 px-4"
        >
          <template #prepend>
            <v-icon size="16" color="primary" class="mr-3">mdi-key-variant</v-icon>
          </template>
          <v-list-item-title class="text-caption font-weight-medium">{{ k.name }}</v-list-item-title>
          <v-list-item-subtitle>
            <span class="mono text-disabled" style="font-size:10px">{{ k.key_preview }}</span>
            <span class="text-disabled mx-2" style="font-size:10px">·</span>
            <span class="text-disabled" style="font-size:10px">criada {{ fmtDate(k.created_at) }}</span>
            <template v-if="k.last_used_at">
              <span class="text-disabled mx-2" style="font-size:10px">·</span>
              <span class="text-disabled" style="font-size:10px">último uso {{ fmtDate(k.last_used_at) }}</span>
            </template>
          </v-list-item-subtitle>
          <template #append>
            <v-btn size="x-small" variant="text" color="error" icon="mdi-delete-outline" @click="confirmRevoke(k)" />
          </template>
        </v-list-item>
      </v-list>
    </v-card>

    <!-- Endpoint hint -->
    <v-card class="mb-3 pa-3" style="border:1px solid var(--telm-border)">
      <div class="text-caption text-medium-emphasis mb-2">Endpoint OTLP HTTP</div>
      <div class="mono" style="font-size:11px;color:#6366f1">POST /otlp/v1/traces</div>
      <div class="mono" style="font-size:11px;color:#6366f1">POST /otlp/v1/metrics</div>
      <div class="mono" style="font-size:11px;color:#6366f1">POST /otlp/v1/logs</div>
      <div class="text-disabled mt-2" style="font-size:11px">
        Header obrigatório: <span class="mono" style="color:#f59e0b">X-API-Key: telm_...</span>
      </div>
    </v-card>

    <!-- Dialog: criar nova key -->
    <v-dialog v-model="newKeyDialog" max-width="440" persistent>
      <v-card>
        <v-card-title class="text-subtitle-2 pa-4 pb-2">Nova API Key</v-card-title>
        <v-card-text class="pa-4 pt-0">
          <v-text-field
            v-model="newKeyName"
            label="Nome (ex: my-service-prod)"
            density="compact"
            variant="outlined"
            autofocus
            @keyup.enter="createKey"
          />
          <v-alert v-if="createdKey" type="success" variant="tonal" density="compact" class="mt-2">
            <div class="text-caption mb-1">Guarde agora — não será exibida novamente:</div>
            <div class="mono d-flex align-center" style="font-size:11px;word-break:break-all;gap:6px">
              {{ createdKey }}
              <v-btn size="x-small" variant="text" icon="mdi-content-copy" @click="copyKey" />
            </div>
          </v-alert>
        </v-card-text>
        <v-card-actions class="pa-4 pt-0">
          <v-spacer />
          <v-btn size="small" variant="text" @click="closeKeyDialog">{{ createdKey ? 'Fechar' : 'Cancelar' }}</v-btn>
          <v-btn v-if="!createdKey" size="small" variant="flat" color="primary" :loading="creating" @click="createKey">Criar</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Dialog: confirmar revogação -->
    <v-dialog v-model="revokeDialog" max-width="380">
      <v-card>
        <v-card-title class="text-subtitle-2 pa-4 pb-2">Revogar key?</v-card-title>
        <v-card-text class="pa-4 pt-0 text-caption">
          A key <strong>{{ revokeTarget?.name }}</strong> será deletada permanentemente e aplicações que a usam pararão de funcionar.
        </v-card-text>
        <v-card-actions class="pa-4 pt-0">
          <v-spacer />
          <v-btn size="small" variant="text" @click="revokeDialog = false">Cancelar</v-btn>
          <v-btn size="small" variant="flat" color="error" :loading="revoking" @click="doRevoke">Revogar</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

  </v-container>
</template>

<script setup>
import { ref, inject, watch, onMounted, onUnmounted, computed, nextTick } from 'vue'
import * as echarts from 'echarts'
import api from '@/plugins/axios'

const refreshKey = inject('refreshKey')
const isDark     = inject('isDark')

// ── Theme helpers (same pattern as DashboardView) ────────────────────────────
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

const COLORS = {
  logs:    '#f59e0b',
  traces:  '#6366f1',
  metrics: '#10b981',
  db:      '#94a3b8',
}

// ── State ────────────────────────────────────────────────────────────────────
const stats  = ref(null)
const growth = ref([])
const range  = ref('24h')

const timeRanges = [
  { v: '6h', label: '6h' }, { v: '24h', label: '24h' },
  { v: '7d', label: '7d' }, { v: '30d', label: '30d' },
]

// ── DOM refs ─────────────────────────────────────────────────────────────────
const donutEl  = ref(null)
const growthEl = ref(null)

// ── Formatting helpers ───────────────────────────────────────────────────────
function fmtBytes(b) {
  if (!b) return '0 B'
  if (b >= 1024 ** 3) return (b / 1024 ** 3).toFixed(2) + ' GB'
  if (b >= 1024 ** 2) return (b / 1024 ** 2).toFixed(1) + ' MB'
  if (b >= 1024)      return (b / 1024).toFixed(1) + ' KB'
  return Math.round(b) + ' B'
}
function fmtNum(n) {
  if (!n) return '0'
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M'
  if (n >= 1_000)     return (n / 1_000).toFixed(1) + 'K'
  return n.toString()
}
function daysAgo(iso) {
  if (!iso) return null
  return (Date.now() - new Date(iso)) / 86_400_000
}
function nextCleanupStr() {
  const now = new Date()
  const next = new Date(now)
  next.setHours(3, 0, 0, 0)
  if (!next > now) next.setDate(next.getDate() + 1)
  const diff = next - now
  const h = Math.floor(diff / 3_600_000)
  const m = Math.floor((diff % 3_600_000) / 60_000)
  if (h === 0) return `em ${m}min`
  return `em ${h}h${m > 0 ? m + 'm' : ''}`
}

// ── Computed: KPI strip ──────────────────────────────────────────────────────
const kpiCards = computed(() => {
  if (!stats.value) return []
  const s = stats.value
  return [
    {
      label: 'DB Total',
      icon:  'mdi-database',
      color: COLORS.db,
      value: fmtBytes(s.db_size_bytes),
      sub:   'tamanho total do banco',
    },
    {
      label: 'Logs',
      icon:  'mdi-text-box-outline',
      color: COLORS.logs,
      value: fmtBytes(s.log_size_bytes),
      sub:   fmtNum(s.log_rows) + ' rows · ret. ' + s.log_retention_days + 'd',
    },
    {
      label: 'Traces',
      icon:  'mdi-transit-connection-variant',
      color: COLORS.traces,
      value: fmtBytes(s.trace_size_bytes),
      sub:   fmtNum(s.trace_rows) + ' rows · ret. ' + s.trace_retention_days + 'd',
    },
    {
      label: 'Metrics',
      icon:  'mdi-chart-line',
      color: COLORS.metrics,
      value: fmtBytes(s.metric_size_bytes),
      sub:   fmtNum(s.metric_rows) + ' rows · ret. ' + s.metric_retention_days + 'd',
    },
  ]
})

// ── Computed: per-table cards ────────────────────────────────────────────────
const tableCards = computed(() => {
  if (!stats.value) return []
  const s = stats.value

  const make = (name, sizeB, rows, oldestIso, retDays, color) => {
    const oldest   = daysAgo(oldestIso)
    const retPct   = oldest != null ? Math.min((oldest / retDays) * 100, 100) : 0
    const bytesRow = rows > 0 ? sizeB / rows : 0
    const retColor = retPct > 80 ? '#10b981' : retPct > 40 ? '#f59e0b' : '#94a3b8'

    return {
      name, color, size: fmtBytes(sizeB), retentionDays: retDays,
      retentionPct: retPct, retentionColor: retColor,
      retentionLabel: oldest != null
        ? `${oldest.toFixed(1)}d de dados (${retPct.toFixed(0)}%)`
        : 'sem dados',
      stats: [
        { label: 'Rows',         value: fmtNum(rows) },
        { label: 'Bytes/row',    value: bytesRow > 0 ? fmtBytes(bytesRow) : '—' },
        { label: 'Dado mais antigo', value: oldest != null ? oldest.toFixed(1) + 'd atrás' : '—' },
        { label: 'Próximo cleanup',  value: nextCleanupStr() },
      ],
    }
  }

  return [
    make('logs',    s.log_size_bytes,    s.log_rows,    s.oldest_log,    s.log_retention_days,    COLORS.logs),
    make('traces',  s.trace_size_bytes,  s.trace_rows,  s.oldest_trace,  s.trace_retention_days,  COLORS.traces),
    make('metrics', s.metric_size_bytes, s.metric_rows, s.oldest_metric, s.metric_retention_days, COLORS.metrics),
  ]
})

// ── Computed: ingestion rate from growth buckets ─────────────────────────────
const ingestionRate = computed(() => {
  if (growth.value.length < 2) return null
  const last = growth.value.slice(-6)
  const bucketMs = new Date(growth.value[1]?.time) - new Date(growth.value[0]?.time)
  const bucketH  = Math.max(bucketMs / 3_600_000, 1 / 60)

  const avg = (field) => last.reduce((s, b) => s + b[field], 0) / last.length
  return {
    logRowsPerHour:    avg('log_rows')    / bucketH,
    traceRowsPerHour:  avg('trace_rows')  / bucketH,
    metricRowsPerHour: avg('metric_rows') / bucketH,
  }
})

// ── Computed: insights ───────────────────────────────────────────────────────
const insights = computed(() => {
  if (!stats.value) return []
  const s   = stats.value
  const ins = []
  const signalTotal = s.log_size_bytes + s.trace_size_bytes + s.metric_size_bytes

  // 1. Log dominance
  if (signalTotal > 0) {
    const logPct = (s.log_size_bytes / signalTotal) * 100
    if (logPct > 70) {
      ins.push({
        level: 'warn',
        msg:   `Logs ocupam ${logPct.toFixed(0)}% do espaço dos sinais.`,
        detail: 'Considere aumentar MIN_LOG_SEVERITY para INFO (9) ou superior.',
      })
    } else if (logPct > 50) {
      ins.push({
        level: 'info',
        msg:   `Logs representam ${logPct.toFixed(0)}% do espaço dos sinais.`,
        detail: `Logs: ${fmtBytes(s.log_size_bytes)} · Traces: ${fmtBytes(s.trace_size_bytes)} · Metrics: ${fmtBytes(s.metric_size_bytes)}`,
      })
    }
  }

  // 2. High ingestion rate
  if (ingestionRate.value) {
    const { logRowsPerHour, traceRowsPerHour, metricRowsPerHour } = ingestionRate.value
    if (logRowsPerHour > 50_000) {
      ins.push({
        level: 'warn',
        msg:   `Taxa de ingestão de logs elevada: ~${fmtNum(Math.round(logRowsPerHour))} rows/h.`,
        detail: 'Verifique se logs TRACE/DEBUG estão sendo descartados pelo filtro de severidade.',
      })
    }
    // MB/hour growth estimate
    if (s.log_rows > 0) {
      const bytesPerLog = s.log_size_bytes / s.log_rows
      const logMBPerHour = (bytesPerLog * logRowsPerHour) / (1024 * 1024)
      const traceMBPerHour = s.trace_rows > 0
        ? ((s.trace_size_bytes / s.trace_rows) * traceRowsPerHour) / (1024 * 1024)
        : 0
      const totalMBPerHour = logMBPerHour + traceMBPerHour
      if (totalMBPerHour > 1) {
        ins.push({
          level: totalMBPerHour > 50 ? 'warn' : 'info',
          msg:   `Crescimento estimado: ~${totalMBPerHour.toFixed(1)} MB/h (${(totalMBPerHour * 24).toFixed(0)} MB/dia).`,
          detail: `Logs: ~${logMBPerHour.toFixed(1)} MB/h · Traces: ~${traceMBPerHour.toFixed(1)} MB/h`,
        })
      }
    }
  }

  // 3. Retention health per table
  const checkRetention = (name, oldestIso, retDays) => {
    const days = daysAgo(oldestIso)
    if (days == null) return
    if (days >= retDays - 0.5) {
      ins.push({
        level: 'ok',
        msg:   `Cleanup de ${name} funcionando — dado mais antigo: ${days.toFixed(1)}d (retenção: ${retDays}d).`,
      })
    } else if (days < retDays * 0.3) {
      ins.push({
        level: 'info',
        msg:   `${name} ainda acumulando — ${days.toFixed(1)}d de dados, limite de retenção: ${retDays}d.`,
        detail: 'O banco crescerá até o primeiro cleanup atingir a janela de retenção.',
      })
    }
  }
  checkRetention('logs',    s.oldest_log,    s.log_retention_days)
  checkRetention('traces',  s.oldest_trace,  s.trace_retention_days)
  checkRetention('metrics', s.oldest_metric, s.metric_retention_days)

  // 4. DB size warning
  if (s.db_size_bytes > 5 * 1024 ** 3) {
    ins.push({
      level: 'warn',
      msg:   `Banco acima de 5 GB (${fmtBytes(s.db_size_bytes)}). Considere reduzir janelas de retenção.`,
    })
  }

  // 5. No data yet
  if (s.log_rows === 0 && s.trace_rows === 0 && s.metric_rows === 0) {
    ins.push({ level: 'info', msg: 'Nenhum dado recebido ainda. Aguardando telemetria.' })
  }

  return ins
})

// ── Fetch ────────────────────────────────────────────────────────────────────
function rangeToTimes(r) {
  const to   = new Date()
  const from = new Date(to)
  const map  = { '6h': 6, '24h': 24, '7d': 168, '30d': 720 }
  from.setHours(from.getHours() - (map[r] || 24))
  return { from: from.toISOString(), to: to.toISOString() }
}

async function loadStats() {
  try {
    const { data } = await api.get('/stats/storage')
    stats.value = data
  } catch {}
}

async function loadGrowth() {
  try {
    const { from, to } = rangeToTimes(range.value)
    const { data } = await api.get('/stats/storage/growth', { params: { from, to } })
    growth.value = data || []
  } catch {}
}

async function load() {
  await Promise.all([loadStats(), loadGrowth()])
  await nextTick()
  renderDonut()
  renderGrowth()
}

// ── ECharts helpers (same pattern as DashboardView) ─────────────────────────
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
    axisLabel: { color: c.axis, fontSize: 10 },
    splitLine: { show: false },
  }
}
function yAxis(name) {
  const c = ct()
  return {
    type: 'value', name,
    nameTextStyle: { color: c.axisDim, fontSize: 10 },
    axisLine: { show: false }, axisTick: { show: false },
    axisLabel: { color: c.axis, fontSize: 10 },
    splitLine: { lineStyle: { color: c.grid, type: 'dashed' } },
  }
}
function gridOpts() {
  return { top: 16, right: 8, bottom: 28, left: 8, containLabel: true }
}
function fmtLabel(iso) {
  const d = new Date(iso)
  if (range.value === '7d' || range.value === '30d')
    return d.toLocaleDateString('pt-BR', { month: 'short', day: 'numeric' })
  return d.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' })
}

// ── Render: donut chart ──────────────────────────────────────────────────────
function renderDonut() {
  const chart = getOrInit(donutEl.value)
  if (!chart || !stats.value) return
  const s = stats.value
  const other = Math.max(0, s.db_size_bytes - s.log_size_bytes - s.trace_size_bytes - s.metric_size_bytes)
  chart.setOption({
    tooltip: {
      ...tooltip('item'),
      formatter: (p) => `${p.name}<br/>${fmtBytes(p.value)} (${p.percent}%)`,
    },
    legend: { show: false },
    series: [{
      type: 'pie',
      radius: ['42%', '68%'],
      center: ['50%', '52%'],
      itemStyle: { borderRadius: 4, borderWidth: 2, borderColor: isDark.value ? '#111827' : '#ffffff' },
      label: {
        show: true,
        formatter: '{b}\n{d}%',
        fontSize: 11,
        color: ct().axis,
      },
      data: [
        { name: 'Logs',    value: s.log_size_bytes,    itemStyle: { color: COLORS.logs } },
        { name: 'Traces',  value: s.trace_size_bytes,  itemStyle: { color: COLORS.traces } },
        { name: 'Metrics', value: s.metric_size_bytes, itemStyle: { color: COLORS.metrics } },
        { name: 'Outros',  value: other,               itemStyle: { color: ct().grid } },
      ].filter(d => d.value > 0),
    }],
  })
}

// ── Render: stacked area growth chart ────────────────────────────────────────
function renderGrowth() {
  const chart = getOrInit(growthEl.value)
  if (!chart) return
  const pts   = growth.value
  const times = pts.map(p => fmtLabel(p.time))

  const areaSeries = (name, field, color) => ({
    name, type: 'line', stack: 'total',
    data: pts.map(p => p[field]),
    smooth: true,
    symbol: 'none',
    lineStyle: { width: 1, color },
    areaStyle: { color, opacity: 0.35 },
    itemStyle: { color },
    emphasis: { focus: 'series' },
  })

  chart.setOption({
    tooltip: {
      ...tooltip('axis'),
      formatter: (params) => {
        let html = `<div style="font-size:11px;font-weight:600;margin-bottom:4px">${params[0].axisValue}</div>`
        params.forEach(p => {
          html += `<div style="display:flex;justify-content:space-between;gap:16px">
            <span>${p.marker}${p.seriesName}</span>
            <span style="font-weight:600">${fmtNum(p.value)}</span>
          </div>`
        })
        return html
      },
    },
    legend: {
      data: ['Logs', 'Traces', 'Metrics'],
      textStyle: { color: ct().axis, fontSize: 11 },
      top: 0, right: 0,
      icon: 'circle', itemWidth: 8, itemHeight: 8,
    },
    grid: gridOpts(),
    xAxis: xAxis(times),
    yAxis: { ...yAxis('rows'), axisLabel: { color: ct().axis, fontSize: 10, formatter: v => fmtNum(v) } },
    series: [
      areaSeries('Logs',    'log_rows',    COLORS.logs),
      areaSeries('Traces',  'trace_rows',  COLORS.traces),
      areaSeries('Metrics', 'metric_rows', COLORS.metrics),
    ],
  })
}

// ── Insight helpers ──────────────────────────────────────────────────────────
function insColor(level) {
  return { warn: '#f59e0b', error: '#ef4444', ok: '#10b981', info: '#7dd3fc' }[level] || '#94a3b8'
}
function insIcon(level) {
  return {
    warn:  'mdi-alert-circle-outline',
    error: 'mdi-close-circle-outline',
    ok:    'mdi-check-circle-outline',
    info:  'mdi-information-outline',
  }[level] || 'mdi-information-outline'
}

// ── Resize observer ──────────────────────────────────────────────────────────
let resizeObs = null
onMounted(() => {
  resizeObs = new ResizeObserver(() => {
    [donutEl, growthEl].forEach(r => {
      const inst = r.value && echarts.getInstanceByDom(r.value)
      if (inst) inst.resize()
    })
  })
  const container = document.querySelector('.v-main')
  if (container) resizeObs.observe(container)
})
onUnmounted(() => {
  resizeObs?.disconnect()
  ;[donutEl, growthEl].forEach(r => {
    if (r.value) echarts.getInstanceByDom(r.value)?.dispose()
  })
})

// ── API Keys ──────────────────────────────────────────────────────────────────
const apiKeys      = ref([])
const newKeyDialog = ref(false)
const newKeyName   = ref('')
const createdKey   = ref('')
const creating     = ref(false)
const revokeDialog = ref(false)
const revokeTarget = ref(null)
const revoking     = ref(false)

function fmtDate(iso) {
  return new Date(iso).toLocaleDateString('pt-BR', { day: '2-digit', month: 'short', year: 'numeric' })
}

async function loadAPIKeys() {
  try {
    const { data } = await api.get('/apikeys')
    apiKeys.value = data || []
  } catch {}
}

async function createKey() {
  if (!newKeyName.value.trim() || creating.value) return
  creating.value = true
  try {
    const { data } = await api.post('/apikeys', { name: newKeyName.value.trim() })
    createdKey.value = data.key
    await loadAPIKeys()
  } catch {
  } finally {
    creating.value = false
  }
}

function copyKey() {
  navigator.clipboard.writeText(createdKey.value)
}

function closeKeyDialog() {
  newKeyDialog.value = false
  newKeyName.value   = ''
  createdKey.value   = ''
}

function confirmRevoke(k) {
  revokeTarget.value = k
  revokeDialog.value = true
}

async function doRevoke() {
  if (!revokeTarget.value || revoking.value) return
  revoking.value = true
  try {
    await api.delete(`/apikeys/${revokeTarget.value.id}`)
    revokeDialog.value = false
    await loadAPIKeys()
  } catch {
  } finally {
    revoking.value = false
  }
}

// ── Watchers ─────────────────────────────────────────────────────────────────
watch(refreshKey, load)
watch(isDark, () => nextTick(() => { renderDonut(); renderGrowth() }))
onMounted(() => { load(); loadAPIKeys() })
</script>

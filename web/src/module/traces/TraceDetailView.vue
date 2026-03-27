<template>
  <v-container fluid class="pa-3">

    <!-- Header -->
    <div class="d-flex align-center gap-2 mb-3 flex-wrap">
      <v-btn size="small" variant="text" color="default" prepend-icon="mdi-arrow-left"
        @click="router.push('/traces')">Traces</v-btn>

      <v-divider vertical class="my-1" opacity="0.2" />

      <!-- Trace meta -->
      <div class="d-flex align-center gap-3 flex-grow-1 flex-wrap pl-2">
        <div class="d-flex align-center gap-1">
          <span class="text-disabled mono" style="font-size:10px;letter-spacing:.04em">TRACE</span>
          <code class="mono" style="font-size:11px;color:var(--telm-text-2)">{{ traceId }}</code>
          <v-tooltip text="Copy trace ID" location="top">
            <template #activator="{ props }">
              <v-btn v-bind="props" icon size="x-small" variant="text" color="grey" @click="copyText(traceId)">
                <v-icon size="12">mdi-content-copy</v-icon>
              </v-btn>
            </template>
          </v-tooltip>
        </div>
        <v-chip color="primary" variant="tonal" size="small" class="mono">
          <v-icon size="12" start>mdi-vector-polyline</v-icon>{{ spans.length }} spans
        </v-chip>
        <v-chip :color="durationColor(traceDurationNs)" variant="tonal" size="small" class="mono">
          <v-icon size="12" start>mdi-timer-outline</v-icon>{{ fmtDuration(traceDurationNs) }}
        </v-chip>
        <v-chip :color="errorCount > 0 ? 'error' : 'success'" variant="tonal" size="small" class="mono">
          <v-icon size="12" start>{{ errorCount > 0 ? 'mdi-alert-circle' : 'mdi-check-circle' }}</v-icon>
          {{ errorCount > 0 ? errorCount + ' error(s)' : 'OK' }}
        </v-chip>

        <!-- Kind legend -->
        <div class="d-flex align-center gap-2 ml-2" style="flex-wrap:wrap">
          <div v-for="k in kindLegend" :key="k.label" class="d-flex align-center gap-1">
            <span class="kind-dot" :style="`background:${k.color}`"></span>
            <span class="text-disabled mono" style="font-size:9px">{{ k.label }}</span>
          </div>
        </div>
      </div>

      <v-menu>
        <template #activator="{ props }">
          <v-btn v-bind="props" size="small" variant="tonal" color="default">
            <v-icon size="14" start>mdi-download</v-icon>Export<v-icon size="12" end>mdi-chevron-down</v-icon>
          </v-btn>
        </template>
        <v-list density="compact" min-width="140">
          <v-list-item prepend-icon="mdi-code-json" title="JSON" @click="exportData('json')" />
          <v-list-item prepend-icon="mdi-table" title="CSV" @click="exportData('csv')" />
        </v-list>
      </v-menu>
    </div>

    <!-- Log highlight banner -->
    <v-alert
      v-if="highlightSpanId"
      type="info"
      variant="tonal"
      density="compact"
      class="mb-3 mono"
      style="font-size:12px"
      closable
      @click:close="$router.replace({ query: {} })"
    >
      <v-icon size="14" start>mdi-text-box-search-outline</v-icon>
      Aberto a partir de um log —
      span <code style="font-size:11px;color:#a5b4fc">{{ highlightSpanId }}</code> destacado abaixo
    </v-alert>

    <!-- Waterfall -->
    <v-card style="overflow:hidden">
      <!-- Column headers -->
      <div class="waterfall-header px-3 py-2" style="display:grid;grid-template-columns:44% 56%">
        <div class="text-disabled" style="font-size:10px;letter-spacing:.06em;text-transform:uppercase">Span / Service</div>
        <div class="d-flex justify-space-between pl-4 pr-14 mono" style="font-size:10px;color:var(--telm-text-3)">
          <span>0</span>
          <span>{{ fmtDuration(traceDurationNs / 4) }}</span>
          <span>{{ fmtDuration(traceDurationNs / 2) }}</span>
          <span>{{ fmtDuration(traceDurationNs * 3 / 4) }}</span>
          <span>{{ fmtDuration(traceDurationNs) }}</span>
        </div>
      </div>

      <div
        v-for="span in sortedSpans"
        :key="span.span_id"
        :id="`span-${span.span_id}`"
        :class="['span-row', span._open ? 'span-open' : '', span.span_id === highlightSpanId ? 'span-highlighted' : '']"
        :style="span.status_code === 2 ? 'background:rgba(239,68,68,.04)' : ''"
      >
        <!-- Main row -->
        <div
          class="span-main"
          style="display:grid;grid-template-columns:44% 56%;cursor:pointer"
          @click="span._open = !span._open"
        >
          <!-- Left: tree + info -->
          <div
            class="d-flex align-center gap-2 py-2 pr-2"
            :style="{ paddingLeft: (12 + span._depth * 16) + 'px' }"
          >
            <!-- Tree line indicator -->
            <span class="kind-dot flex-shrink-0" :style="`background:${kindColor(span.kind)}`" />
            <div class="flex-grow-1 overflow-hidden">
              <div class="d-flex align-center gap-1">
                <v-tooltip :text="span.operation" location="top" open-delay="400">
                  <template #activator="{ props }">
                    <span v-bind="props" class="text-caption mono op-truncate" style="color:var(--telm-text-1)">{{ span.operation }}</span>
                  </template>
                </v-tooltip>
                <v-chip v-if="span.status_code === 2" color="error" variant="tonal" size="x-small" class="mono flex-shrink-0">ERR</v-chip>
                <v-chip v-if="logsBySpan[span.span_id]?.length" color="secondary" variant="tonal" size="x-small" class="mono flex-shrink-0">
                  <v-icon size="9" start>mdi-text-box-outline</v-icon>{{ logsBySpan[span.span_id].length }}
                </v-chip>
              </div>
              <div class="d-flex align-center gap-1 mt-0">
                <v-chip :color="serviceChipColor(span.service)" class="mono" size="x-small">{{ span.service }}</v-chip>
                <span class="mono" style="font-size:10px" :style="`color:${kindColor(span.kind)}`">{{ kindLabel(span.kind) }}</span>
              </div>
            </div>
            <v-icon size="12" :color="span._open ? 'primary' : 'grey'" class="flex-shrink-0">
              {{ span._open ? 'mdi-chevron-down' : 'mdi-chevron-right' }}
            </v-icon>
          </div>

          <!-- Right: waterfall bar -->
          <div class="d-flex align-center pl-4 pr-16 position-relative" style="height:44px">
            <div class="flex-grow-1 position-relative" style="height:100%">
              <div class="position-absolute bar-track" style="width:100%;top:50%;transform:translateY(-50%)"></div>
              <div
                class="position-absolute bar-fill"
                :style="{
                  left: span._left + '%',
                  width: span._width + '%',
                  background: barColor(span),
                  top: '50%',
                  transform: 'translateY(-50%)',
                }"
              ></div>
            </div>
            <span class="position-absolute end-0 mono font-weight-medium" style="min-width:60px;text-align:right;font-size:11px;padding-right:4px" :style="`color:${durationColor(span.duration_ns)}`">
              {{ fmtDuration(span.duration_ns) }}
            </span>
          </div>
        </div>

        <!-- Expanded detail -->
        <v-expand-transition>
          <div v-if="span._open" class="span-detail">

            <!-- Error alert -->
            <v-alert
              v-if="span.status_code === 2"
              type="error" variant="tonal" density="compact"
              class="mb-3 mono text-caption"
            >
              {{ span.status_message || 'span marked as error' }}
            </v-alert>

            <!-- IDs + timing row -->
            <div class="detail-grid mb-3">
              <div class="detail-item">
                <span class="detail-key">Span ID</span>
                <div class="d-flex align-center gap-1">
                  <code class="detail-val-code">{{ span.span_id }}</code>
                  <v-btn icon size="x-small" variant="text" color="grey" @click="copyText(span.span_id)">
                    <v-icon size="11">mdi-content-copy</v-icon>
                  </v-btn>
                </div>
              </div>
              <div class="detail-item" v-if="span.parent_span_id">
                <span class="detail-key">Parent Span</span>
                <div class="d-flex align-center gap-1">
                  <code class="detail-val-code">{{ span.parent_span_id }}</code>
                  <v-btn icon size="x-small" variant="text" color="grey" @click="copyText(span.parent_span_id)">
                    <v-icon size="11">mdi-content-copy</v-icon>
                  </v-btn>
                </div>
              </div>
              <div class="detail-item">
                <span class="detail-key">Kind</span>
                <span class="mono font-weight-bold" style="font-size:12px" :style="`color:${kindColor(span.kind)}`">
                  {{ kindLabel(span.kind) }}
                </span>
              </div>
              <div class="detail-item">
                <span class="detail-key">Start</span>
                <span class="detail-val">{{ fmtTime(span.start_time) }}</span>
              </div>
              <div class="detail-item">
                <span class="detail-key">End</span>
                <span class="detail-val">{{ fmtTime(span.end_time) }}</span>
              </div>
              <div class="detail-item">
                <span class="detail-key">Duration</span>
                <span class="mono font-weight-bold" style="font-size:13px" :style="`color:${durationColor(span.duration_ns)}`">
                  {{ fmtDuration(span.duration_ns) }}
                </span>
              </div>
            </div>

            <!-- Attributes -->
            <template v-if="span.attributes && Object.keys(span.attributes).length">
              <div class="section-label mb-2">
                <v-icon size="11" class="mr-1">mdi-tag-multiple-outline</v-icon>
                Attributes ({{ Object.keys(span.attributes).length }})
              </div>
              <div class="attr-table mb-3">
                <div
                  v-for="[k, v] in Object.entries(span.attributes)"
                  :key="k"
                  class="attr-row"
                >
                  <span class="attr-key">{{ k }}</span>
                  <span class="attr-eq">=</span>
                  <span class="attr-val">{{ String(v) }}</span>
                  <v-btn icon size="x-small" variant="text" color="grey" class="attr-copy" @click="copyText(String(v))">
                    <v-icon size="10">mdi-content-copy</v-icon>
                  </v-btn>
                </div>
              </div>
            </template>

            <!-- Events -->
            <template v-if="Array.isArray(span.events) && span.events.length">
              <div class="section-label mb-2">
                <v-icon size="11" class="mr-1">mdi-lightning-bolt-outline</v-icon>
                Events ({{ span.events.length }})
              </div>
              <div class="d-flex flex-column gap-1 mb-3">
                <div
                  v-for="(ev, i) in span.events"
                  :key="i"
                  class="event-row d-flex align-center gap-2 pa-2 rounded mono text-caption"
                >
                  <v-icon size="12" color="warning">mdi-rhombus</v-icon>
                  <span style="color:var(--telm-text-1)">{{ ev.name || JSON.stringify(ev) }}</span>
                  <span v-if="ev.timestamp" class="text-disabled ml-auto" style="font-size:10px">
                    {{ fmtTime(ev.timestamp) }}
                  </span>
                </div>
              </div>
            </template>

            <!-- Logs deste span -->
            <template v-if="logsBySpan[span.span_id]?.length">
              <div class="section-label mb-2">
                <v-icon size="11" class="mr-1">mdi-text-box-multiple-outline</v-icon>
                Logs ({{ logsBySpan[span.span_id].length }})
              </div>
              <div class="log-list mb-2">
                <div
                  v-for="(lg, i) in logsBySpan[span.span_id]"
                  :key="i"
                  class="log-entry"
                  :class="'sev-' + lg.severity.toLowerCase()"
                >
                  <div class="log-entry-top">
                    <span class="sev-badge" :style="`background:${sevColor(lg.severity)}22;color:${sevColor(lg.severity)}`">
                      {{ lg.severity }}
                    </span>
                    <span class="log-offset mono">+{{ fmtOffset(lg.timestamp) }}</span>
                    <span class="log-body mono">{{ lg.body }}</span>
                    <v-btn
                      icon size="x-small" variant="text"
                      :color="lg._jsonOpen ? 'primary' : 'grey'"
                      class="flex-shrink-0 ml-auto"
                      @click.stop="lg._jsonOpen = !lg._jsonOpen"
                    >
                      <v-icon size="12">mdi-code-braces</v-icon>
                    </v-btn>
                  </div>
                  <div v-if="!lg._jsonOpen && lg.attributes && Object.keys(lg.attributes).length" class="log-attrs mono">
                    <span
                      v-for="[k, v] in Object.entries(lg.attributes)"
                      :key="k"
                      class="log-attr-pair"
                    >
                      <span class="log-attr-key">{{ k }}</span>=<span class="log-attr-val">{{ String(v) }}</span>
                    </span>
                  </div>
                  <v-expand-transition>
                    <div v-if="lg._jsonOpen" class="log-json">
                      <pre>{{ JSON.stringify(logToJson(lg), null, 2) }}</pre>
                      <v-btn
                        size="x-small" variant="text" color="grey" class="log-json-copy"
                        @click="copyText(JSON.stringify(logToJson(lg), null, 2))"
                      >
                        <v-icon size="11" start>mdi-content-copy</v-icon>copy
                      </v-btn>
                    </div>
                  </v-expand-transition>
                </div>
              </div>
            </template>

          </div>
        </v-expand-transition>
      </div>

      <div v-if="!spans.length" class="text-center py-12 text-caption text-disabled">
        <v-progress-circular indeterminate size="20" class="mb-2" />
        <div>Loading spans…</div>
      </div>
    </v-card>

    <!-- All trace logs -->
    <v-card v-if="logs.length" class="mt-3" style="overflow:hidden">
      <div
        class="waterfall-header px-3 py-2 d-flex align-center gap-2"
        style="cursor:pointer"
        @click="allLogsOpen = !allLogsOpen"
      >
        <v-icon size="14" color="secondary">mdi-text-box-multiple-outline</v-icon>
        <span class="text-disabled" style="font-size:10px;letter-spacing:.06em;text-transform:uppercase">
          All Logs
        </span>
        <v-chip color="secondary" variant="tonal" size="x-small" class="mono ml-1">{{ logs.length }}</v-chip>
        <v-spacer />
        <v-icon size="16" color="grey">{{ allLogsOpen ? 'mdi-chevron-up' : 'mdi-chevron-down' }}</v-icon>
      </div>
      <v-expand-transition>
        <div v-if="allLogsOpen">
          <div class="log-list">
            <div
              v-for="(lg, i) in logs"
              :key="i"
              class="log-entry log-entry-full"
              :class="'sev-' + lg.severity.toLowerCase()"
            >
              <div class="log-entry-top">
                <span class="sev-badge" :style="`background:${sevColor(lg.severity)}22;color:${sevColor(lg.severity)}`">
                  {{ lg.severity }}
                </span>
                <span class="log-offset mono">+{{ fmtOffset(lg.timestamp) }}</span>
                <v-chip v-if="lg.service" size="x-small" :color="serviceChipColor(lg.service)" class="mono flex-shrink-0">
                  {{ lg.service }}
                </v-chip>
                <span class="log-body mono">{{ lg.body }}</span>
                <div class="d-flex align-center gap-1 flex-shrink-0 ml-auto">
                  <v-btn
                    icon size="x-small" variant="text"
                    :color="lg._jsonOpen ? 'primary' : 'grey'"
                    @click.stop="lg._jsonOpen = !lg._jsonOpen"
                  >
                    <v-icon size="12">mdi-code-braces</v-icon>
                  </v-btn>
                  <v-btn
                    v-if="lg.span_id && spanById[lg.span_id]"
                    icon size="x-small" variant="text" color="primary"
                    @click="jumpToSpan(lg.span_id)"
                  >
                    <v-icon size="12">mdi-vector-polyline</v-icon>
                  </v-btn>
                </div>
              </div>
              <div v-if="!lg._jsonOpen && lg.attributes && Object.keys(lg.attributes).length" class="log-attrs mono">
                <span
                  v-for="[k, v] in Object.entries(lg.attributes)"
                  :key="k"
                  class="log-attr-pair"
                >
                  <span class="log-attr-key">{{ k }}</span>=<span class="log-attr-val">{{ String(v) }}</span>
                </span>
              </div>
              <v-expand-transition>
                <div v-if="lg._jsonOpen" class="log-json">
                  <pre>{{ JSON.stringify(logToJson(lg), null, 2) }}</pre>
                  <v-btn
                    size="x-small" variant="text" color="grey" class="log-json-copy"
                    @click="copyText(JSON.stringify(logToJson(lg), null, 2))"
                  >
                    <v-icon size="11" start>mdi-content-copy</v-icon>copy
                  </v-btn>
                </div>
              </v-expand-transition>
            </div>
          </div>
        </div>
      </v-expand-transition>
    </v-card>

    <!-- Copy snackbar -->
    <v-snackbar v-model="copied" timeout="1500" color="success" location="bottom right" :elevation="0">
      <v-icon size="14" start>mdi-check</v-icon>Copied!
    </v-snackbar>

  </v-container>
</template>

<script setup>
import { ref, computed, reactive, watch, nextTick, shallowReactive } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import api from '@/plugins/axios'

const route = useRoute()
const router = useRouter()
const traceId = computed(() => route.params.traceId)
const highlightSpanId = computed(() => route.query.highlight || '')
const spans = ref([])
const logs = ref([])
const allLogsOpen = ref(false)
const copied = ref(false)

const traceDurationNs = computed(() => {
  if (!spans.value.length) return 0
  const st = Math.min(...spans.value.map(s => new Date(s.start_time).getTime()))
  const end = Math.max(...spans.value.map(s => new Date(s.end_time).getTime()))
  return (end - st) * 1_000_000
})

const errorCount = computed(() => spans.value.filter(s => s.status_code === 2).length)

// map span_id → span (para navegar a partir do log)
const spanById = computed(() => Object.fromEntries(spans.value.map(s => [s.span_id, s])))

// map span_id → logs[] (logs agrupados por span)
const logsBySpan = computed(() => {
  const m = {}
  for (const lg of logs.value) {
    if (lg.span_id) {
      ;(m[lg.span_id] = m[lg.span_id] || []).push(lg)
    }
  }
  return m
})

const kindLegend = [
  { label: 'INTERNAL', color: '#8b5cf6' },
  { label: 'SERVER',   color: '#3b82f6' },
  { label: 'CLIENT',   color: '#10b981' },
  { label: 'PRODUCER', color: '#f97316' },
  { label: 'CONSUMER', color: '#f59e0b' },
]

const sortedSpans = computed(() => {
  if (!spans.value.length) return []
  const minMs = Math.min(...spans.value.map(s => new Date(s.start_time).getTime()))
  const totMs = (Math.max(...spans.value.map(s => new Date(s.end_time).getTime())) - minMs) || 1
  const byPar = {}
  spans.value.forEach(s => {
    const k = s.parent_span_id || '__root__'
    ;(byPar[k] = byPar[k] || []).push(s)
  })
  const result = []
  const visit = (key, depth) => {
    ;(byPar[key] || []).sort((a, b) => new Date(a.start_time) - new Date(b.start_time)).forEach(s => {
      const stMs = new Date(s.start_time).getTime() - minMs
      const durMs = s.duration_ns / 1_000_000
      result.push(reactive({
        ...s,
        _depth: depth,
        _left: (stMs / totMs) * 100,
        _width: Math.max((durMs / totMs) * 100, 0.5),
        _open: false,
      }))
      visit(s.span_id, depth + 1)
    })
  }
  visit('__root__', 0)
  return result
})

function copyText(text) {
  navigator.clipboard?.writeText(text).then(() => { copied.value = true })
}

function fmtDuration(ns) {
  if (!ns) return '0ns'
  if (ns < 1_000) return `${ns}ns`
  if (ns < 1_000_000) return `${(ns / 1_000).toFixed(1)}µs`
  if (ns < 1e9) return `${(ns / 1_000_000).toFixed(1)}ms`
  return `${(ns / 1e9).toFixed(2)}s`
}

function fmtTime(iso) {
  if (!iso) return '–'
  const d = new Date(iso)
  return d.toLocaleDateString('pt-BR') + ' ' + d.toLocaleTimeString('pt-BR', { hour12: false })
}

function durationColor(ns) {
  if (ns > 1e9) return '#f87171'
  if (ns > 200_000_000) return '#fbbf24'
  return '#34d399'
}

function kindLabel(k) {
  return { 1: 'INTERNAL', 2: 'SERVER', 3: 'CLIENT', 4: 'PRODUCER', 5: 'CONSUMER' }[k] || 'UNSPECIFIED'
}
function kindColor(k) {
  return { 1: '#8b5cf6', 2: '#3b82f6', 3: '#10b981', 4: '#f97316', 5: '#f59e0b' }[k] || '#64748b'
}
function barColor(span) {
  if (span.status_code === 2) return '#ef4444'
  return kindColor(span.kind)
}
function logToJson(lg) {
  const obj = {
    timestamp: lg.timestamp,
    severity:  lg.severity,
    body:      lg.body,
    service:   lg.service,
  }
  if (lg.trace_id) obj.trace_id = lg.trace_id
  if (lg.span_id)  obj.span_id  = lg.span_id
  if (lg.attributes && Object.keys(lg.attributes).length) obj.attributes = lg.attributes
  return obj
}

function sevColor(sev) {
  const s = (sev || '').toUpperCase()
  if (s === 'ERROR' || s === 'FATAL') return '#f87171'
  if (s === 'WARN')  return '#fbbf24'
  if (s === 'DEBUG') return '#94a3b8'
  return '#34d399' // INFO + others
}

function fmtOffset(iso) {
  if (!spans.value.length) return '?'
  const traceStart = Math.min(...spans.value.map(s => new Date(s.start_time).getTime()))
  const ms = new Date(iso).getTime() - traceStart
  if (ms < 0) return '0ms'
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(2)}s`
}

async function jumpToSpan(spanId) {
  const target = sortedSpans.value.find(s => s.span_id === spanId)
  if (!target) return
  target._open = true
  await nextTick()
  const el = document.getElementById(`span-${spanId}`)
  if (el) el.scrollIntoView({ behavior: 'smooth', block: 'center' })
}

function serviceChipColor(name) {
  const colors = ['indigo', 'purple', 'blue', 'teal', 'green', 'cyan', 'orange', 'pink']
  let h = 0
  for (const c of (name || '')) h = (h * 31 + c.charCodeAt(0)) & 0xffff
  return colors[h % colors.length]
}

function exportData(format) {
  const ts = new Date().toISOString().slice(0, 19).replace(/:/g, '-')
  if (format === 'json') {
    const blob = new Blob([JSON.stringify(spans.value, null, 2)], { type: 'application/json' })
    dl(blob, `trace-${traceId.value.slice(0, 8)}-${ts}.json`)
  } else {
    const header = ['span_id', 'parent_span_id', 'service', 'operation', 'kind', 'start_time', 'end_time', 'duration_ms', 'status_code', 'status_message']
    const rows = spans.value.map(s => [
      s.span_id, s.parent_span_id || '', s.service, s.operation,
      kindLabel(s.kind), s.start_time, s.end_time,
      (s.duration_ns / 1e6).toFixed(3), s.status_code, s.status_message || '',
    ])
    const csv = [header, ...rows].map(r => r.map(v => `"${String(v ?? '').replace(/"/g, '""')}"`).join(',')).join('\n')
    dl(new Blob([csv], { type: 'text/csv' }), `trace-${traceId.value.slice(0, 8)}-${ts}.csv`)
  }
}

function dl(blob, name) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url; a.download = name; a.click()
  URL.revokeObjectURL(url)
}

watch(traceId, async (id) => {
  if (!id) return
  spans.value = []
  logs.value = []
  allLogsOpen.value = false

  const [spansRes, logsRes] = await Promise.all([
    api.get(`/traces/${id}`),
    api.get(`/traces/${id}/logs`),
  ])
  spans.value = spansRes.data || []
  logs.value = (logsRes.data || []).map(lg => reactive({
    ...lg,
    severity: lg.severity || 'INFO',
    attributes: typeof lg.attributes === 'string' ? JSON.parse(lg.attributes) : (lg.attributes || {}),
    _jsonOpen: false,
  }))

  if (highlightSpanId.value) {
    await nextTick()
    const target = sortedSpans.value.find(s => s.span_id === highlightSpanId.value)
    if (target) {
      target._open = true
      await nextTick()
      const el = document.getElementById(`span-${target.span_id}`)
      if (el) el.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }
  }
}, { immediate: true })
</script>

<style scoped>
.mono { font-family: ui-monospace, 'JetBrains Mono', monospace; }

.waterfall-header {
  background: var(--telm-bg-header);
  border-bottom: 1px solid var(--telm-border);
}

.span-row { border-bottom: 1px solid var(--telm-border-light); background: var(--telm-bg-row); }
.span-row:last-child { border-bottom: none; }
.span-main:hover { background: var(--telm-bg-hover); }
.span-open .span-main { background: var(--telm-bg-hover); }

.span-detail {
  padding: 12px 16px 14px 16px;
  background: var(--telm-bg-detail);
  border-top: 1px solid var(--telm-border-light);
}

.bar-track {
  height: 5px;
  background: var(--telm-bg-header);
  border-radius: 3px;
}
.bar-fill {
  height: 11px;
  border-radius: 3px;
  min-width: 3px;
  opacity: .88;
}
.kind-dot { width: 8px; height: 8px; border-radius: 50%; display: inline-block; flex-shrink: 0; }

/* Detail grid */
.detail-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 8px;
}
.detail-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 6px 10px;
  background: var(--telm-bg-code);
  border-radius: 6px;
  border: 1px solid var(--telm-border-light);
}
.detail-key {
  font-size: 9px;
  letter-spacing: .08em;
  text-transform: uppercase;
  color: var(--telm-text-3);
  font-family: ui-monospace, monospace;
}
.detail-val {
  font-size: 12px;
  color: var(--telm-text-1);
  font-family: ui-monospace, monospace;
}
.detail-val-code {
  font-size: 10px;
  color: var(--telm-text-2);
  word-break: break-all;
  line-height: 1.4;
  font-family: ui-monospace, monospace;
}

/* Section label */
.section-label {
  font-size: 10px;
  letter-spacing: .08em;
  text-transform: uppercase;
  color: var(--telm-text-3);
  font-family: ui-monospace, monospace;
  display: flex;
  align-items: center;
}

/* Attributes table */
.attr-table {
  display: flex;
  flex-direction: column;
  gap: 1px;
  border-radius: 6px;
  overflow: hidden;
  border: 1px solid var(--telm-border);
}
.attr-row {
  display: grid;
  grid-template-columns: 260px 16px 1fr auto;
  align-items: center;
  gap: 4px;
  padding: 5px 8px;
  background: var(--telm-bg-attr);
  font-family: ui-monospace, monospace;
  font-size: 11px;
  transition: background .1s;
}
.attr-row:hover { background: var(--telm-bg-hover); }
.attr-row:nth-child(even) { background: var(--telm-bg-code); }
.attr-row:nth-child(even):hover { background: var(--telm-bg-hover); }
.attr-key { color: #818cf8; font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.attr-eq  { color: var(--telm-text-4); }
.attr-val { color: var(--telm-text-2); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.attr-copy { opacity: 0; }
.attr-row:hover .attr-copy { opacity: 1; }

/* Events */
.event-row { background: var(--telm-bg-row); border: 1px solid var(--telm-border); }

/* Operation truncate */
.op-truncate {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
}

/* Log list */
.log-list {
  display: flex;
  flex-direction: column;
  background: var(--telm-bg-row);
}
.log-entry {
  padding: 5px 14px;
  border-bottom: 1px solid var(--telm-border-light);
  background: var(--telm-bg-row);
}
.log-entry:last-child { border-bottom: none; }
.log-entry-full { padding: 6px 16px; }
.log-entry-top {
  display: flex;
  align-items: baseline;
  gap: 8px;
  flex-wrap: wrap;
}
.sev-badge {
  font-size: 9px;
  font-weight: 700;
  letter-spacing: .06em;
  padding: 1px 5px;
  border-radius: 3px;
  flex-shrink: 0;
  font-family: ui-monospace, monospace;
}
.log-offset {
  font-size: 10px;
  color: var(--telm-text-3);
  flex-shrink: 0;
  min-width: 46px;
}
.log-body {
  font-size: 12px;
  color: var(--telm-text-1);
  flex-grow: 1;
  white-space: pre-wrap;
  word-break: break-word;
}
.log-attrs {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 3px;
  padding-left: 0;
}
.log-attr-pair {
  font-size: 10px;
  background: var(--telm-bg-code);
  border: 1px solid var(--telm-border-light);
  border-radius: 3px;
  padding: 1px 5px;
}
.log-attr-key { color: #818cf8; }
.log-attr-val { color: var(--telm-text-2); }

/* JSON view */
.log-json {
  position: relative;
  margin-top: 6px;
  background: var(--telm-bg-code);
  border: 1px solid var(--telm-border);
  border-radius: 5px;
  overflow: hidden;
}
.log-json pre {
  margin: 0;
  padding: 10px 12px;
  font-family: ui-monospace, 'JetBrains Mono', monospace;
  font-size: 11px;
  line-height: 1.6;
  color: var(--telm-text-1);
  white-space: pre;
  overflow-x: auto;
}
.log-json-copy {
  position: absolute;
  top: 4px;
  right: 4px;
  opacity: 0;
  transition: opacity .15s;
}
.log-json:hover .log-json-copy { opacity: 1; }

/* Highlighted span (from log navigation) */
.span-highlighted {
  animation: span-pulse 2.4s ease-out forwards;
  border-left: 3px solid #6366f1 !important;
}
.span-highlighted .span-main {
  background: rgba(99,102,241,.14) !important;
}

@keyframes span-pulse {
  0%   { background: rgba(99,102,241,.28); }
  40%  { background: rgba(99,102,241,.18); }
  100% { background: transparent; }
}
</style>

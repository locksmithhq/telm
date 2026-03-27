<template>
  <v-dialog v-model="open" fullscreen transition="dialog-bottom-transition" persistent class="editor-dialog">
    <div class="editor-root">

      <!-- Top bar -->
      <div class="editor-topbar">
        <v-icon size="15" color="primary" class="mr-2">mdi-chart-box-plus-outline</v-icon>
        <v-text-field
          v-model="title"
          placeholder="Panel title (optional)"
          hide-details variant="plain" density="compact"
          style="max-width: 300px; font-size: 13px"
        />
        <v-spacer />
        <v-btn size="small" variant="text" @click="cancel">Cancel</v-btn>
        <v-btn size="small" color="primary" variant="flat" class="ml-1" :disabled="!generatedQuery" @click="apply">Apply</v-btn>
      </div>

      <div class="editor-body">

        <!-- ── Left sidebar (scrollable) ── -->
        <div class="editor-sidebar">

          <!-- STEP 1 — Data source -->
          <div class="sidebar-section">
            <div class="sidebar-step">1 — Data source</div>
            <div class="source-list">
              <button
                v-for="(def, src) in SOURCES" :key="src"
                class="source-item" :class="{ active: source === src }"
                @click="selectSource(src)"
              >
                <v-icon size="13" :color="source === src ? 'primary' : 'grey'" class="mr-2 flex-shrink-0">{{ sourceIcon(src) }}</v-icon>
                <div class="overflow-hidden">
                  <div class="source-label text-truncate">{{ def.label }}</div>
                  <div class="source-sub mono text-truncate">{{ src }}</div>
                </div>
              </button>
            </div>
          </div>

          <!-- STEP 2 — Data selection (varies by source) -->
          <div v-if="source" class="sidebar-section">
            <div class="sidebar-step">2 — Selection &amp; filters</div>

            <!-- metrics.series: full catalog browser -->
            <template v-if="source === 'metrics.series'">
              <div class="d-flex gap-1 mb-1">
                <v-text-field
                  v-model="metricSearch"
                  placeholder="Search metrics…"
                  prepend-inner-icon="mdi-magnify"
                  hide-details density="compact" variant="outlined"
                  clearable
                  style="font-size:11px"
                />
                <v-select
                  v-model="metricTypeFilter"
                  :items="metricTypes"
                  placeholder="Type"
                  hide-details density="compact" variant="outlined"
                  style="width: 80px; font-size:11px"
                />
              </div>
              <v-select
                v-model="filters.service"
                :items="serviceOptions"
                placeholder="All services"
                hide-details density="compact" variant="outlined"
                clearable
                class="mb-2"
                style="font-size:11px"
              />
              <div class="metric-list">
                <div
                  v-for="m in filteredCatalog" :key="m.name + '::' + m.service"
                  class="metric-item" :class="{ active: filters.name === m.name && filters.service === m.service }"
                  @click="selectMetric(m)"
                >
                  <span class="type-badge" :style="`background:${typeColor(m.type)}22;color:${typeColor(m.type)}`">{{ m.type.slice(0,3).toUpperCase() }}</span>
                  <div class="d-flex flex-column overflow-hidden">
                    <span class="text-truncate" style="font-size:11px">{{ m.name }}</span>
                    <span class="text-truncate mono" style="font-size:9px;opacity:.55">{{ m.service }}</span>
                  </div>
                </div>
              </div>
            </template>

            <!-- Other sources: quick filters -->
            <template v-else-if="source === 'logs'">
              <v-tabs v-model="logsTab" density="compact" color="primary" class="mb-2" style="font-size:10px">
                <v-tab value="selection" style="font-size:10px;min-width:0;padding:0 8px;">Selection</v-tab>
                <v-tab value="logs" style="font-size:10px;min-width:0;padding:0 8px;">Logs</v-tab>
              </v-tabs>

              <div v-show="logsTab === 'selection'">
                <div class="filter-row">
                  <label class="filter-label">Service</label>
                  <v-select
                    v-model="filters.service"
                    :items="serviceOptions"
                    placeholder="All services"
                    hide-details density="compact" variant="outlined"
                    clearable style="font-size:11px"
                  />
                </div>
                <div class="filter-row">
                  <label class="filter-label">Range</label>
                  <div class="d-flex flex-wrap gap-1">
                    <button
                      v-for="r in RANGE_VALUES" :key="r"
                      class="sz-btn" :class="{ active: filters.range === r }"
                      @click="filters.range = r"
                    >{{ r }}</button>
                  </div>
                </div>
              </div>

              <div v-show="logsTab === 'logs'">
                <div class="filter-row">
                  <label class="filter-label">Operation</label>
                  <v-text-field v-model="filters.operation" placeholder="GET /api/..." hide-details density="compact" variant="outlined" clearable style="font-size:11px" />
                </div>
                <div class="filter-row">
                  <label class="filter-label">Severity</label>
                  <div class="d-flex flex-wrap gap-1 mt-1">
                    <button
                      v-for="sev in ['DEBUG','INFO','WARN','ERROR','FATAL']" :key="sev"
                      class="sz-btn mono" :class="{ active: filters.severity === sev }"
                      :style="filters.severity === sev ? `background:${sevColor(sev)}22;border-color:${sevColor(sev)}88;color:${sevColor(sev)}` : ''"
                      @click="filters.severity = filters.severity === sev ? '' : sev"
                    >{{ sev }}</button>
                  </div>
                </div>
                <div class="filter-row">
                  <label class="filter-label">Status</label>
                  <v-btn-toggle v-model="filters.status" density="compact" class="mt-1" variant="outlined">
                    <v-btn size="x-small" value="">Any</v-btn>
                    <v-btn size="x-small" value="ok">OK</v-btn>
                    <v-btn size="x-small" value="error" color="error">Error</v-btn>
                  </v-btn-toggle>
                </div>
                <div class="filter-row">
                  <label class="filter-label">Search</label>
                  <v-text-field v-model="filters.search" placeholder="keyword..." hide-details density="compact" variant="outlined" clearable style="font-size:11px" />
                </div>
                <div class="filter-row">
                  <label class="filter-label">Limit</label>
                  <v-select v-model="filters.limit" :items="['50','100','200','500']" hide-details density="compact" variant="outlined" style="font-size:11px" />
                </div>
              </div>
            </template>

            <template v-else>
              <div class="filter-row">
                <label class="filter-label">Service</label>
                <v-select
                  v-model="filters.service"
                  :items="serviceOptions"
                  placeholder="All services"
                  hide-details density="compact" variant="outlined"
                  clearable style="font-size:11px"
                />
              </div>
              <div class="filter-row">
                <label class="filter-label">Range</label>
                <div class="d-flex flex-wrap gap-1">
                  <button
                    v-for="r in RANGE_VALUES" :key="r"
                    class="sz-btn" :class="{ active: filters.range === r }"
                    @click="filters.range = r"
                  >{{ r }}</button>
                </div>
              </div>
              <template v-if="source === 'traces'">
                <div class="filter-row">
                  <label class="filter-label">Operation</label>
                  <v-text-field v-model="filters.operation" placeholder="GET /api/..." hide-details density="compact" variant="outlined" clearable style="font-size:11px" />
                </div>
                <div class="filter-row">
                  <label class="filter-label">Status</label>
                  <v-btn-toggle v-model="filters.status" density="compact" class="mt-1" variant="outlined">
                    <v-btn size="x-small" value="">Any</v-btn>
                    <v-btn size="x-small" value="ok">OK</v-btn>
                    <v-btn size="x-small" value="error" color="error">Error</v-btn>
                  </v-btn-toggle>
                </div>
                <div class="filter-row">
                  <label class="filter-label">Limit</label>
                  <v-select v-model="filters.limit" :items="['25','50','100','200']" hide-details density="compact" variant="outlined" style="font-size:11px" />
                </div>
              </template>
            </template>
          </div>

          <!-- STEP 3 — Chart type -->
          <div v-if="source" class="sidebar-section">
            <div class="sidebar-step">3 — Chart type</div>
            <div class="viz-grid">
              <button
                v-for="v in currentVizOptions" :key="v"
                class="viz-card" :class="{ active: viz === v }"
                @click="viz = v"
              >
                <v-icon size="20" :color="viz === v ? 'primary' : 'grey'">{{ vizIcon(v) }}</v-icon>
                <div class="viz-card-label">{{ vizLabel(v) }}</div>
                <div class="viz-card-sub">{{ vizDesc(v) }}</div>
              </button>
            </div>
          </div>

          <!-- Generated query -->
          <div v-if="generatedQuery" class="sidebar-section">
            <div class="sidebar-step">Generated query</div>
            <pre class="query-preview mono">{{ generatedQuery }}</pre>
          </div>
        </div>

        <!-- ── Preview (right side) ── -->
        <div class="editor-preview">
          <div v-if="!source" class="preview-empty d-flex flex-column align-center justify-center h-100">
            <v-icon size="48" color="grey-darken-2">mdi-chart-box-outline</v-icon>
            <div class="text-caption text-disabled mt-3">Select a data source to begin</div>
          </div>
          <template v-else>
            <div class="preview-topbar text-caption text-disabled mb-2">
              Live preview
              <span v-if="generatedQuery" class="ml-2 mono" style="font-size:10px;color:#6366f1">{{ generatedQuery }}</span>
            </div>
            <div class="preview-content">
              <PanelViz :key="previewKey" :query="generatedQuery" />
            </div>
          </template>
        </div>

      </div>
    </div>
  </v-dialog>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { SOURCES, RANGE_VALUES } from './tql.js'
import PanelViz from './PanelViz.vue'
import api from '@/plugins/axios'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  panel:      { type: Object,  default: null },
})
const emit = defineEmits(['update:modelValue', 'apply'])

const open = computed({ get: () => props.modelValue, set: v => emit('update:modelValue', v) })

// ─── State ────────────────────────────────────────────────────────────────
const title   = ref('')
const source  = ref('')
const viz     = ref('')
const filters = ref(emptyFilters())

const catalog         = ref([])
const metricSearch    = ref('')
const metricTypeFilter = ref('')
const services        = ref([])
const previewKey      = ref(0)
const logsTab         = ref('selection')

const metricTypes = ['', 'gauge', 'sum', 'histogram', 'summary']

function emptyFilters() {
  return { service: '', range: '1h', name: '', operation: '', status: '', severity: '', search: '', limit: '50' }
}

// ─── Catalog filtering ────────────────────────────────────────────────────
const filteredCatalog = computed(() => {
  let c = catalog.value
  if (metricSearch.value) c = c.filter(m => m.name.toLowerCase().includes(metricSearch.value.toLowerCase()))
  if (metricTypeFilter.value) c = c.filter(m => m.type === metricTypeFilter.value)
  if (filters.value.service) c = c.filter(m => m.service === filters.value.service)
  return c
})

// ─── Viz options for current source ──────────────────────────────────────
const currentVizOptions = computed(() => source.value ? SOURCES[source.value]?.vizOptions || [] : [])
const serviceOptions = computed(() => services.value.map(s => ({ title: s, value: s })))

// ─── Generated query ──────────────────────────────────────────────────────
const generatedQuery = computed(() => {
  if (!source.value) return ''
  const f = filters.value
  const parts = [source.value]
  if (f.service)                                        parts.push(`service=${f.service}`)
  if (f.range && f.range !== '1h')                      parts.push(`range=${f.range}`)
  if (source.value === 'metrics.series' && f.name)      parts.push(`name=${f.name}`)
  if (source.value === 'traces' && f.operation)         parts.push(`operation=${f.operation}`)
  if (source.value === 'traces' && f.status)            parts.push(`status=${f.status}`)
  if (source.value === 'logs'   && f.severity)          parts.push(`severity=${f.severity}`)
  if (source.value === 'logs'   && f.search)            parts.push(`search=${f.search}`)
  if (['traces','logs'].includes(source.value) && f.limit !== '50') parts.push(`limit=${f.limit}`)
  if (viz.value) parts.push(`viz=${viz.value}`)
  return parts.join(' ')
})

// ─── Debounced preview ────────────────────────────────────────────────────
let previewTimer = null
function triggerPreview() { clearTimeout(previewTimer); previewTimer = setTimeout(() => previewKey.value++, 500) }
watch(generatedQuery, triggerPreview)

// ─── Actions ─────────────────────────────────────────────────────────────
function selectSource(src) {
  source.value = src
  viz.value    = SOURCES[src]?.defaultViz || ''
  if (src === 'logs') logsTab.value = 'selection'
}

function selectMetric(m) {
  filters.value.name    = m.name
  filters.value.service = m.service
}

function apply()  { emit('apply', { title: title.value, query: generatedQuery.value }); open.value = false }
function cancel() { open.value = false }

// ─── Pre-fill from existing panel ─────────────────────────────────────────
function prefill() {
  const p = props.panel
  if (!p) { reset(); return }
  title.value = p.title || ''
  const q = (p.query || '').trim()
  if (!q) { reset(); return }
  const tokens = q.split(/\s+/)
  source.value = tokens[0] || ''
  viz.value    = SOURCES[source.value]?.defaultViz || ''
  if (source.value === 'logs') logsTab.value = 'selection'
  const f = emptyFilters()
  for (let i = 1; i < tokens.length; i++) {
    const eq = tokens[i].indexOf('=')
    if (eq < 1) continue
    const k = tokens[i].slice(0, eq), v = tokens[i].slice(eq + 1)
    if (k === 'viz') { viz.value = v; continue }
    if (k in f) f[k] = v
  }
  filters.value = f
}

function reset() {
  title.value = ''; source.value = ''; viz.value = ''
  filters.value = emptyFilters()
  logsTab.value = 'selection'
}

// ─── Data loading ─────────────────────────────────────────────────────────
async function loadData() {
  await Promise.all([
    api.get('/services').then(r => { services.value = r.data || [] }).catch(() => {}),
    api.get('/metrics/catalog').then(r => { catalog.value = r.data || [] }).catch(() => {}),
  ])
}

watch(open, v => { if (v) { prefill(); loadData() } })
onMounted(() => { if (open.value) { prefill(); loadData() } })

// ─── Icon / label helpers ─────────────────────────────────────────────────
function sourceIcon(src) {
  return {
    'metrics.series':       'mdi-chart-line',
    'stats.throughput':     'mdi-gauge',
    'stats.errors':         'mdi-alert-circle-outline',
    'stats.latency':        'mdi-timer-outline',
    'stats.top-ops':        'mdi-trophy-outline',
    'stats.severity':       'mdi-format-list-bulleted',
    'stats.service-map':    'mdi-transit-connection-variant',
    'stats.services-health':'mdi-heart-pulse',
    'stats.resources':      'mdi-memory',
    traces:                 'mdi-magnify-scan',
    logs:                   'mdi-text-box-outline',
  }[src] || 'mdi-chart-box-outline'
}

const VIZ_META = {
  line:    { icon: 'mdi-chart-line',              label: 'Line',      desc: 'Time series' },
  area:    { icon: 'mdi-chart-areaspline',        label: 'Area',      desc: 'Filled line' },
  bar:     { icon: 'mdi-chart-bar',               label: 'Bar',       desc: 'Columns' },
  scatter: { icon: 'mdi-chart-scatter-plot',      label: 'Scatter',   desc: 'Point cloud' },
  gauge:   { icon: 'mdi-gauge',                   label: 'Gauge',     desc: 'Dial meter' },
  stat:    { icon: 'mdi-numeric',                 label: 'Stat',      desc: 'Big number' },
  pie:     { icon: 'mdi-chart-pie',               label: 'Pie',       desc: 'Donut chart' },
  radar:   { icon: 'mdi-radar',                   label: 'Radar',     desc: 'Spider web' },
  treemap: { icon: 'mdi-view-quilt-outline',      label: 'Treemap',   desc: 'Nested boxes' },
  funnel:  { icon: 'mdi-filter-outline',          label: 'Funnel',    desc: 'Ranked flow' },
  heatmap: { icon: 'mdi-grid',                    label: 'Heatmap',   desc: 'Color matrix' },
  sankey:  { icon: 'mdi-call-split',              label: 'Sankey',    desc: 'Flow diagram' },
  graph:   { icon: 'mdi-graph-outline',           label: 'Graph',     desc: 'Network' },
  table:   { icon: 'mdi-table',                   label: 'Table',     desc: 'Tabular data' },
}
function vizIcon(v)  { return VIZ_META[v]?.icon  || 'mdi-chart-box-outline' }
function vizLabel(v) { return VIZ_META[v]?.label || v }
function vizDesc(v)  { return VIZ_META[v]?.desc  || '' }

function typeIcon(t)  { return { gauge: '◉', sum: '∑', histogram: '▦', summary: '◈' }[t] || '◌' }
function typeBadgeColor(t) { return { gauge: '#8b5cf6', sum: '#3b82f6', histogram: '#f97316', summary: '#14b8a6' }[t] || '#64748b' }
function typeColor(t) { return { gauge: '#8b5cf6', sum: '#3b82f6', histogram: '#f97316', summary: '#14b8a6' }[t] || '#64748b' }
function sevColor(s) { return { ERROR: '#ef4444', FATAL: '#dc2626', WARN: '#f59e0b', INFO: '#6366f1', DEBUG: '#64748b', TRACE: '#475569' }[s] || '#64748b' }
</script>

<style scoped>
.mono { font-family: ui-monospace, 'JetBrains Mono', monospace; }

.editor-root {
  display: flex; flex-direction: column; height: 100vh;
  background: rgb(var(--v-theme-surface)) !important; overflow: hidden;
}

/* Top bar */
.editor-topbar {
  display: flex; align-items: center; gap: 6px;
  padding: 7px 14px; flex-shrink: 0;
  border-bottom: 1px solid var(--telm-border);
  background: rgb(var(--v-theme-surface)) !important;
}

/* Body */
.editor-body { 
  display: flex; 
  flex: 1; 
  min-height: 0; 
  overflow: hidden; 
  height: 100%; 
}

/* Sidebar */
.editor-sidebar {
  width: 280px; 
  flex-shrink: 0;
  border-right: 1px solid var(--telm-border);
  overflow-y: auto; 
  background: rgb(var(--v-theme-surface)) !important;
}

.sidebar-section { padding: 10px 12px; border-bottom: 1px solid var(--telm-border-light); }

.sidebar-step {
  font-size: 10px; font-weight: 600; letter-spacing: 0.07em;
  text-transform: uppercase; color: var(--telm-text-3); margin-bottom: 8px;
}

/* Sources */
.source-list { display: flex; flex-direction: column; gap: 1px; }
.source-item {
  display: flex; align-items: center; padding: 5px 7px; border-radius: 6px;
  border: none; background: transparent; color: inherit;
  font-size: 11px; cursor: pointer; text-align: left; transition: background 0.1s; width: 100%;
}
.source-item:hover  { background: var(--telm-bg-hover); }
.source-item.active { background: rgba(99,102,241,0.14); color: #a5b4fc; }
.source-label { font-size: 11px; font-weight: 500; }
.source-sub   { font-size: 9px; color: var(--telm-text-3); }

/* Metric catalog */
.metric-list {
  max-height: 240px; overflow-y: auto; margin: 0 -12px;
  border-top: 1px solid var(--telm-border-light);
}
.metric-item {
  display: flex; align-items: center; padding: 5px 12px;
  cursor: pointer; border-bottom: 1px solid var(--telm-border-light);
  transition: background 0.1s;
}
.metric-item:hover  { background: var(--telm-bg-hover); }
.metric-item.active { background: rgba(99,102,241,0.12); }
.type-badge {
  font-size: 9px; font-weight: 600; padding: 1px 5px; border-radius: 4px;
  letter-spacing: 0.04em; white-space: nowrap; flex-shrink: 0; margin-right: 6px;
}

/* Filters */
.filter-row   { margin-bottom: 8px; }
.filter-label { display: block; font-size: 10px; color: var(--telm-text-3); margin-bottom: 3px; }

/* Viz grid */
.viz-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 5px; }
.viz-card {
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  padding: 8px 4px; border-radius: 7px;
  border: 1px solid var(--telm-border);
  background: transparent; cursor: pointer; transition: all 0.12s;
}
.viz-card:hover  { border-color: rgba(99,102,241,0.35); background: var(--telm-bg-hover); }
.viz-card.active { border-color: rgba(99,102,241,0.6); background: rgba(99,102,241,0.14); }
.viz-card-label  { font-size: 10px; font-weight: 600; margin-top: 4px; color: inherit; }
.viz-card-sub    { font-size: 9px; color: var(--telm-text-3); margin-top: 1px; }

/* Query preview */
.query-preview {
  font-size: 10px; color: #6366f1;
  background: var(--telm-bg-hover); border-radius: 4px;
  padding: 6px 8px; white-space: pre-wrap; word-break: break-all; margin: 0;
}

/* Preview area */
.editor-preview { 
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: rgb(var(--v-theme-surface));
}
.preview-topbar { 
  font-size: 10px; 
  letter-spacing: 0.05em; 
  text-transform: uppercase; 
  padding: 8px 14px;
  border-bottom: 1px solid var(--telm-border-light);
  flex-shrink: 0;
}
.preview-content {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}
.preview-empty  { 
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

/* Size / range buttons */
.sz-btn {
  font-size: 10px; padding: 2px 7px; border-radius: 4px;
  border: 1px solid var(--telm-border); background: transparent;
  color: var(--telm-text-3); cursor: pointer; transition: all 0.1s;
}
.sz-btn:hover  { border-color: rgba(99,102,241,0.4); color: #a5b4fc; }
.sz-btn.active { background: rgba(99,102,241,0.2); border-color: rgba(99,102,241,0.5); color: #a5b4fc; }
</style>

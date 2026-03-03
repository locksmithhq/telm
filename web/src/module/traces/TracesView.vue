<template>
  <v-container fluid class="pa-3">

    <!-- Filters -->
    <div class="d-flex align-center gap-3 mb-3 flex-wrap">
      <v-text-field
        v-model="filters.operation"
        placeholder="Filter operation…"
        prepend-inner-icon="mdi-magnify"
        clearable
        hide-details
        density="compact"
        style="min-width:200px;max-width:260px"
        @keydown.enter="load"
      />

      <v-menu v-model="fromOpen" :close-on-content-click="false" min-width="auto">
        <template #activator="{ props }">
          <v-text-field
            v-bind="props"
            :model-value="filters.from ? fmtDate(filters.from) : ''"
            readonly clearable placeholder="From"
            prepend-inner-icon="mdi-calendar-start"
            hide-details density="compact" style="min-width:140px;max-width:165px"
            @click:clear="filters.from = ''"
          />
        </template>
        <v-date-picker :model-value="filters.from ? new Date(filters.from) : null" hide-header
          @update:model-value="d => selectDate(d, 'from')" />
      </v-menu>

      <v-menu v-model="toOpen" :close-on-content-click="false" min-width="auto">
        <template #activator="{ props }">
          <v-text-field
            v-bind="props"
            :model-value="filters.to ? fmtDate(filters.to) : ''"
            readonly clearable placeholder="To"
            prepend-inner-icon="mdi-calendar-end"
            hide-details density="compact" style="min-width:140px;max-width:165px"
            @click:clear="filters.to = ''"
          />
        </template>
        <v-date-picker :model-value="filters.to ? new Date(filters.to) : null" hide-header
          @update:model-value="d => selectDate(d, 'to')" />
      </v-menu>

      <v-select v-model="filters.limit" :items="[25, 50, 100, 200]" hide-details density="compact"
        style="min-width:90px;max-width:110px" @update:model-value="load" />

      <v-btn size="small" color="primary" variant="flat" @click="load">
        <v-icon size="14" start>mdi-magnify</v-icon>Search
      </v-btn>

      <v-spacer />

      <div class="d-flex align-center gap-2">
        <span class="text-caption text-disabled mono">{{ traces.length }} trace(s)</span>
        <v-chip v-if="errorCount > 0" color="error" variant="tonal" size="small">
          <v-icon size="11" start>mdi-alert-circle</v-icon>{{ errorCount }} error(s)
        </v-chip>
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
    </div>

    <!-- Table -->
    <v-card style="overflow:hidden">
      <v-table hover>
        <thead>
          <tr class="table-header">
            <th style="width:36px"></th>
            <th>Service</th>
            <th>Operation</th>
            <th>Trace ID</th>
            <th>Spans</th>
            <th>Start Time</th>
            <th class="text-right" style="width:160px">Duration</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="t in traces"
            :key="t.span_id"
            class="trace-row"
            @click="router.push('/traces/' + t.trace_id)"
          >
            <!-- Status -->
            <td class="pl-3 pr-0">
              <v-tooltip :text="t.status_code === 2 ? 'ERROR' : 'OK'" location="right">
                <template #activator="{ props }">
                  <span
                    v-bind="props"
                    class="status-dot"
                    :style="t.status_code === 2 ? 'background:#ef4444;box-shadow:0 0 6px #ef444488' : 'background:#10b981;box-shadow:0 0 6px #10b98188'"
                  />
                </template>
              </v-tooltip>
            </td>

            <!-- Service(s) -->
            <td style="min-width:130px">
              <div class="d-flex align-center gap-1 flex-wrap">
                <v-chip
                  v-for="svc in (t.services?.length ? t.services : [t.service])"
                  :key="svc"
                  :color="serviceChipColor(svc)"
                  class="mono"
                  size="small"
                >{{ svc }}</v-chip>
              </div>
            </td>

            <!-- Operation -->
            <td style="min-width:200px;max-width:320px">
              <div class="d-flex align-center gap-2">
                <v-tooltip :text="t.operation" location="top" open-delay="400">
                  <template #activator="{ props }">
                    <span v-bind="props" class="mono text-body-2 op-truncate">{{ t.operation }}</span>
                  </template>
                </v-tooltip>
                <v-chip v-if="t.status_code === 2" color="error" variant="tonal" size="x-small" class="mono flex-shrink-0">ERROR</v-chip>
              </div>
            </td>

            <!-- Trace ID full -->
            <td style="min-width:240px">
              <div class="d-flex align-center gap-1">
                <code class="mono trace-id-text">{{ t.trace_id }}</code>
                <v-tooltip text="Copy trace ID" location="top">
                  <template #activator="{ props }">
                    <v-btn
                      v-bind="props"
                      icon
                      size="x-small"
                      variant="text"
                      color="grey"
                      class="copy-btn"
                      @click.stop="copyText(t.trace_id)"
                    >
                      <v-icon size="12">mdi-content-copy</v-icon>
                    </v-btn>
                  </template>
                </v-tooltip>
              </div>
            </td>

            <!-- Span count -->
            <td style="width:80px">
              <span class="mono text-medium-emphasis" style="font-size:13px">
                {{ t.span_count ?? '—' }}
              </span>
            </td>

            <!-- Time -->
            <td style="min-width:170px">
              <div class="mono" style="font-size:12px;color:var(--telm-text-1)">{{ fmtTime(t.start_time) }}</div>
              <div class="mono text-disabled" style="font-size:10px">{{ relTime(t.start_time) }}</div>
            </td>

            <!-- Duration -->
            <td class="text-right">
              <div class="d-flex flex-column align-end gap-1">
                <span
                  class="mono font-weight-bold"
                  style="font-size:13px"
                  :style="`color:${durationColor(t.duration_ns)}`"
                >{{ fmtDuration(t.duration_ns) }}</span>
                <v-progress-linear
                  :model-value="durationPct(t.duration_ns)"
                  :color="t.status_code === 2 ? 'error' : durationBarColor(t.duration_ns)"
                  height="3" rounded style="width:120px"
                />
              </div>
            </td>
          </tr>

          <tr v-if="!traces.length">
            <td colspan="7" class="text-center py-12">
              <v-icon size="36" color="grey-darken-2">mdi-transit-connection-variant</v-icon>
              <div class="text-caption text-disabled mt-2">No traces found</div>
            </td>
          </tr>
        </tbody>
      </v-table>
    </v-card>

    <!-- Copy snackbar -->
    <v-snackbar v-model="copied" timeout="1500" color="success" location="bottom right" :elevation="0">
      <v-icon size="14" start>mdi-check</v-icon>Copied!
    </v-snackbar>

  </v-container>
</template>

<script setup>
import { ref, inject, watch, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import api from '@/plugins/axios'

const router = useRouter()
const sharedFilters = inject('sharedFilters')
const refreshKey = inject('refreshKey')

const traces = ref([])
const filters = ref({ operation: '', from: '', to: '', limit: 100 })
const fromOpen = ref(false)
const toOpen = ref(false)
const copied = ref(false)

const errorCount = computed(() => traces.value.filter(t => t.status_code === 2).length)

async function load() {
  const params = {
    service: sharedFilters.service || '',
    operation: filters.value.operation,
    limit: filters.value.limit,
    from: filters.value.from,
    to: filters.value.to,
  }
  const { data } = await api.get('/traces', { params })
  traces.value = data || []
}

function selectDate(d, field) {
  if (!d) { filters.value[field] = ''; return }
  const dt = new Date(d)
  if (field === 'from') { dt.setHours(0, 0, 0, 0); fromOpen.value = false }
  else { dt.setHours(23, 59, 59, 999); toOpen.value = false }
  filters.value[field] = dt.toISOString()
}

function copyText(text) {
  navigator.clipboard?.writeText(text).then(() => { copied.value = true })
}

function fmtDate(iso) { return new Date(iso).toLocaleDateString('pt-BR') }

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
  return d.toLocaleDateString('pt-BR') + ' ' + d.toLocaleTimeString('pt-BR')
}

function relTime(iso) {
  if (!iso) return ''
  const s = Math.floor((Date.now() - new Date(iso).getTime()) / 1000)
  if (s < 60) return `${s}s ago`
  if (s < 3600) return `${Math.floor(s / 60)}m ago`
  if (s < 86400) return `${Math.floor(s / 3600)}h ago`
  return `${Math.floor(s / 86400)}d ago`
}

function durationColor(ns) {
  if (ns > 1e9) return '#f87171'
  if (ns > 200_000_000) return '#fbbf24'
  return '#34d399'
}
function durationBarColor(ns) {
  if (ns > 1e9) return 'error'
  if (ns > 200_000_000) return 'warning'
  return 'success'
}
function durationPct(ns) { return Math.min(100, (ns / 2e9) * 100) }

function serviceChipColor(name) {
  const colors = ['indigo', 'purple', 'blue', 'teal', 'green', 'cyan', 'orange', 'pink']
  let h = 0
  for (const c of (name || '')) h = (h * 31 + c.charCodeAt(0)) & 0xffff
  return colors[h % colors.length]
}

function exportData(format) {
  const ts = new Date().toISOString().slice(0, 19).replace(/:/g, '-')
  if (format === 'json') {
    const blob = new Blob([JSON.stringify(traces.value, null, 2)], { type: 'application/json' })
    download(blob, `traces-${ts}.json`)
  } else {
    const header = ['trace_id', 'service', 'operation', 'start_time', 'duration_ms', 'status', 'span_count']
    const rows = traces.value.map(t => [
      t.trace_id, t.service, t.operation, t.start_time,
      (t.duration_ns / 1e6).toFixed(3),
      t.status_code === 2 ? 'error' : 'ok',
      t.span_count ?? '',
    ])
    const csv = [header, ...rows].map(r => r.map(v => `"${String(v ?? '').replace(/"/g, '""')}"`).join(',')).join('\n')
    download(new Blob([csv], { type: 'text/csv' }), `traces-${ts}.csv`)
  }
}

function download(blob, name) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url; a.download = name; a.click()
  URL.revokeObjectURL(url)
}

onMounted(load)
watch(refreshKey, load)
</script>

<style scoped>
.mono { font-family: ui-monospace, 'JetBrains Mono', monospace; }

.table-header th {
  background: var(--telm-bg-header) !important;
  color: var(--telm-text-2) !important;
  font-size: 11px !important;
  font-weight: 600 !important;
  letter-spacing: .06em;
  text-transform: uppercase;
  border-bottom: 1px solid var(--telm-border) !important;
  padding-top: 10px !important;
  padding-bottom: 10px !important;
}

.trace-row { cursor: pointer; transition: background .12s; }
.trace-row:hover td { background: var(--telm-bg-hover) !important; }

.status-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.trace-id-text {
  font-size: 10px;
  color: var(--telm-text-2);
  letter-spacing: 0.03em;
  word-break: break-all;
  line-height: 1.4;
}

.op-truncate {
  display: block;
  max-width: 280px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.copy-btn {
  opacity: 0;
  transition: opacity .15s;
}
.trace-row:hover .copy-btn { opacity: 1; }
</style>

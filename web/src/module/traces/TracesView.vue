<template>
  <v-container fluid class="pa-3">
    <!-- Filter header bar (sempre visível) -->
    <div class="d-flex align-center gap-2 mb-2">
      <v-icon size="15" color="grey">mdi-filter-variant</v-icon>
      <span class="text-caption font-weight-medium">Filters</span>
      <v-badge
        v-if="activeFilterCount > 0"
        :content="activeFilterCount"
        color="primary"
        inline
        class="ml-1"
      />
      <v-spacer />
      <div class="d-flex align-center gap-2">
        <span class="text-caption text-disabled mono"
          >{{ traces.length }} trace(s)</span
        >
        <v-chip
          v-if="errorCount > 0"
          color="error"
          variant="tonal"
          size="small"
        >
          <v-icon size="11" start>mdi-alert-circle</v-icon
          >{{ errorCount }} error(s)
        </v-chip>
        <v-menu>
          <template #activator="{ props }">
            <v-btn v-bind="props" size="small" variant="tonal" color="default">
              <v-icon size="14" start>mdi-download</v-icon>Export<v-icon
                size="12"
                end
                >mdi-chevron-down</v-icon
              >
            </v-btn>
          </template>
          <v-list density="compact" min-width="140">
            <v-list-item
              prepend-icon="mdi-code-json"
              title="JSON"
              @click="exportData('json')"
            />
            <v-list-item
              prepend-icon="mdi-table"
              title="CSV"
              @click="exportData('csv')"
            />
          </v-list>
        </v-menu>
        <v-btn
          size="small"
          variant="text"
          color="grey"
          @click="panelOpen = !panelOpen"
        >
          <v-icon size="16">{{
            panelOpen ? "mdi-chevron-up" : "mdi-chevron-down"
          }}</v-icon>
        </v-btn>
      </div>
    </div>

    <!-- Filter panel (colapsável) -->
    <v-expand-transition>
      <v-card
        v-show="panelOpen"
        border
        variant="outlined"
        class="mb-3 pa-3"
        :elevation="0"
      >
        <!-- Seção 1: Busca & Identificação -->
        <div class="filter-section">
          <div class="section-header">
            <v-icon size="12" color="grey">mdi-magnify</v-icon>
            <span>Search</span>
          </div>
          <div class="d-flex align-center gap-3 flex-wrap">
            <v-text-field
              v-model="filters.operation"
              placeholder="Operation name"
              prepend-inner-icon="mdi-function-variant"
              clearable
              hide-details
              density="compact"
              style="min-width: 220px; flex: 1"
              @keydown.enter="load"
            />
            <v-text-field
              v-model="filters.traceId"
              placeholder="Trace ID"
              prepend-inner-icon="mdi-identifier"
              clearable
              hide-details
              density="compact"
              style="min-width: 280px; flex: 1"
            />
          </div>
        </div>

        <!-- Seção 2: Tempo -->
        <div class="filter-section">
          <div class="section-header">
            <v-icon size="12" color="grey">mdi-clock-outline</v-icon>
            <span>Time Range</span>
            <v-chip
              v-if="filters.timePreset"
              size="x-small"
              variant="flat"
              color="primary"
              class="ml-2"
            >{{ filters.timePreset }}</v-chip>
          </div>
          <div class="d-flex align-center gap-2 flex-wrap">
            <div class="range-picker">
              <button
                v-for="r in TIME_PRESETS"
                :key="r"
                class="range-btn"
                :class="{ active: filters.timePreset === r }"
                @click="applyPreset(r)"
              >
                {{ r }}
              </button>
              <button
                class="range-btn"
                :class="{ active: filters.timePreset === '' }"
                @click="filters.timePreset = ''"
              >
                custom
              </button>
            </div>
            <template v-if="!filters.timePreset">
              <v-menu
                v-model="fromOpen"
                :close-on-content-click="false"
                min-width="auto"
              >
                <template #activator="{ props }">
                  <v-text-field
                    v-bind="props"
                    :model-value="filters.from ? fmtDate(filters.from) : ''"
                    readonly
                    clearable
                    placeholder="From"
                    prepend-inner-icon="mdi-calendar-start"
                    hide-details
                    density="compact"
                    style="width: 140px"
                    @click:clear="filters.from = ''"
                  />
                </template>
                <v-date-picker
                  :model-value="filters.from ? new Date(filters.from) : null"
                  hide-header
                  @update:model-value="(d) => selectDate(d, 'from')"
                />
              </v-menu>
              <span class="text-caption text-disabled">→</span>
              <v-menu
                v-model="toOpen"
                :close-on-content-click="false"
                min-width="auto"
              >
                <template #activator="{ props }">
                  <v-text-field
                    v-bind="props"
                    :model-value="filters.to ? fmtDate(filters.to) : ''"
                    readonly
                    clearable
                    placeholder="To"
                    prepend-inner-icon="mdi-calendar-end"
                    hide-details
                    density="compact"
                    style="width: 140px"
                    @click:clear="filters.to = ''"
                  />
                </template>
                <v-date-picker
                  :model-value="filters.to ? new Date(filters.to) : null"
                  hide-header
                  @update:model-value="(d) => selectDate(d, 'to')"
                />
              </v-menu>
            </template>
          </div>
        </div>

        <!-- Seção 3: Status & Kind -->
        <div class="filter-section">
          <div class="section-header">
            <v-icon size="12" color="grey">mdi-flag-variant</v-icon>
            <span>Result Type</span>
          </div>
          <div class="d-flex align-center gap-3 flex-wrap">
            <div class="filter-group">
              <span class="filter-label">Status</span>
              <v-chip
                v-for="s in STATUS_OPTIONS"
                :key="s.value"
                :color="filters.statusCodes.includes(s.value) ? s.color : undefined"
                :variant="filters.statusCodes.includes(s.value) ? 'tonal' : 'outlined'"
                size="small"
                class="filter-chip"
                @click="toggleStatus(s.value)"
              >{{ s.label }}</v-chip>
            </div>
            <v-divider vertical class="mx-1" style="height: 24px; align-self: center" />
            <div class="filter-group">
              <span class="filter-label">Kind</span>
              <v-chip
                v-for="k in KIND_OPTIONS"
                :key="k.value"
                :color="filters.kinds.includes(k.value) ? 'primary' : undefined"
                :variant="filters.kinds.includes(k.value) ? 'tonal' : 'outlined'"
                size="small"
                class="filter-chip"
                @click="toggleKind(k.value)"
              >{{ k.label }}</v-chip>
            </div>
            <v-divider vertical class="mx-1" style="height: 24px; align-self: center" />
            <v-chip
              :color="filters.hasError ? 'error' : undefined"
              :variant="filters.hasError ? 'tonal' : 'outlined'"
              size="small"
              class="filter-chip"
              prepend-icon="mdi-alert-circle-outline"
              @click="toggleHasError"
            >Has Error</v-chip>
          </div>
        </div>

        <!-- Seção 4: Performance & Atributos -->
        <div class="filter-section">
          <div class="section-header">
            <v-icon size="12" color="grey">mdi-speedometer</v-icon>
            <span>Performance</span>
          </div>
          <div class="d-flex align-center gap-3 flex-wrap">
            <div class="filter-group">
              <span class="filter-label">Duration</span>
              <div class="d-flex align-center gap-1">
                <v-text-field
                  v-model.number="filters.durationMinMs"
                  type="number"
                  min="0"
                  placeholder="Min"
                  hide-details
                  density="compact"
                  style="width: 90px"
                />
                <span class="text-caption text-disabled">–</span>
                <v-text-field
                  v-model.number="filters.durationMaxMs"
                  type="number"
                  min="0"
                  placeholder="Max"
                  hide-details
                  density="compact"
                  style="width: 90px"
                />
                <span class="text-caption text-disabled ml-1">ms</span>
              </div>
            </div>
            <v-text-field
              v-model.number="filters.minSpanCount"
              type="number"
              min="1"
              placeholder="Min spans"
              prepend-inner-icon="mdi-counter"
              hide-details
              density="compact"
              style="width: 120px"
            />
          </div>
        </div>

        <!-- Seção 5: Atributos -->
        <div class="filter-section">
          <div class="section-header">
            <v-icon size="12" color="grey">mdi-tag-multiple-outline</v-icon>
            <span>Attributes</span>
            <span class="text-caption text-disabled ml-1">(any span)</span>
            <v-spacer />
            <v-btn
              size="x-small"
              variant="text"
              color="primary"
              @click="addAttrFilter"
            >
              <v-icon size="12" start>mdi-plus</v-icon>Add
            </v-btn>
          </div>
          <div class="attr-filters d-flex flex-column gap-2">
            <div
              v-for="(attr, idx) in filters.attributes"
              :key="idx"
              class="attr-filter-row d-flex align-center gap-2"
            >
              <v-text-field
                v-model="attr.key"
                placeholder="Key (e.g. http.method)"
                hide-details
                density="compact"
                style="width: 180px"
              />
              <v-select
                v-model="attr.op"
                :items="[
                  { title: '=', value: '=' },
                  { title: '≠', value: '!=' },
                  { title: '∋', value: 'exists' },
                  { title: '∤', value: 'not_exists' }
                ]"
                hide-details
                density="compact"
                style="width: 80px"
              />
              <v-text-field
                v-if="attr.op === '=' || attr.op === '!='"
                v-model="attr.value"
                placeholder="Value"
                hide-details
                density="compact"
                style="width: 160px"
              />
              <div v-else style="width: 160px"></div>
              <v-btn
                icon
                size="x-small"
                variant="text"
                color="grey"
                @click="removeAttrFilter(idx)"
              >
                <v-icon size="14">mdi-close</v-icon>
              </v-btn>
            </div>
            <div v-if="!filters.attributes.length" class="text-caption text-disabled">
              Click "Add" to filter by span attributes
            </div>
          </div>
        </div>

        <!-- Ações -->
        <div class="d-flex align-center gap-2 mt-2 pt-2 actions-bar">
          <v-select
            v-model="filters.limit"
            :items="[25, 50, 100, 200]"
            label="Results"
            hide-details
            density="compact"
            style="width: 100px"
          />
          <v-spacer />
          <v-btn size="small" variant="text" color="grey" @click="resetFilters">
            <v-icon size="14" start>mdi-refresh</v-icon>Reset
          </v-btn>
          <v-btn
            size="small"
            variant="flat"
            color="primary"
            prepend-icon="mdi-magnify"
            @click="load"
          >Search</v-btn>
        </div>
      </v-card>
    </v-expand-transition>

    <!-- Table -->
    <v-card style="overflow: hidden">
      <v-table hover>
        <thead>
          <tr class="table-header">
            <th style="width: 36px"></th>
            <th>Service</th>
            <th>Operation</th>
            <th>Trace ID</th>
            <th>Spans</th>
            <th>Start Time</th>
            <th class="text-right" style="width: 160px">Duration</th>
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
              <v-tooltip
                :text="t.status_code === 2 ? 'ERROR' : 'OK'"
                location="right"
              >
                <template #activator="{ props }">
                  <span
                    v-bind="props"
                    class="status-dot"
                    :style="
                      t.status_code === 2
                        ? 'background:#ef4444;box-shadow:0 0 6px #ef444488'
                        : 'background:#10b981;box-shadow:0 0 6px #10b98188'
                    "
                  />
                </template>
              </v-tooltip>
            </td>

            <!-- Service(s) -->
            <td style="min-width: 130px">
              <div class="d-flex align-center gap-1 flex-wrap">
                <v-chip
                  v-for="svc in t.services?.length ? t.services : [t.service]"
                  :key="svc"
                  :color="serviceChipColor(svc)"
                  class="mono"
                  size="small"
                  >{{ svc }}</v-chip
                >
              </div>
            </td>

            <!-- Operation -->
            <td style="min-width: 200px; max-width: 320px">
              <div class="d-flex align-center gap-2">
                <v-tooltip :text="t.operation" location="top" open-delay="400">
                  <template #activator="{ props }">
                    <span v-bind="props" class="mono text-body-2 op-truncate">{{
                      t.operation
                    }}</span>
                  </template>
                </v-tooltip>
                <v-chip
                  v-if="t.status_code === 2"
                  color="error"
                  variant="tonal"
                  size="x-small"
                  class="mono flex-shrink-0"
                  >ERROR</v-chip
                >
              </div>
            </td>

            <!-- Trace ID full -->
            <td style="min-width: 240px">
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
            <td style="width: 80px">
              <span class="mono text-medium-emphasis" style="font-size: 13px">
                {{ t.span_count ?? "—" }}
              </span>
            </td>

            <!-- Time -->
            <td style="min-width: 170px">
              <div
                class="mono"
                style="font-size: 12px; color: var(--telm-text-1)"
              >
                {{ fmtTime(t.start_time) }}
              </div>
              <div class="mono text-disabled" style="font-size: 10px">
                {{ relTime(t.start_time) }}
              </div>
            </td>

            <!-- Duration -->
            <td class="text-right">
              <div class="d-flex flex-column align-end gap-1">
                <span
                  class="mono font-weight-bold"
                  style="font-size: 13px"
                  :style="`color:${durationColor(t.duration_ns)}`"
                  >{{ fmtDuration(t.duration_ns) }}</span
                >
                <v-progress-linear
                  :model-value="durationPct(t.duration_ns)"
                  :color="
                    t.status_code === 2
                      ? 'error'
                      : durationBarColor(t.duration_ns)
                  "
                  height="3"
                  rounded
                  style="width: 120px"
                />
              </div>
            </td>
          </tr>

          <tr v-if="!traces.length">
            <td colspan="7" class="text-center py-12">
              <v-icon size="36" color="grey-darken-2"
                >mdi-transit-connection-variant</v-icon
              >
              <div class="text-caption text-disabled mt-2">No traces found</div>
            </td>
          </tr>
        </tbody>
      </v-table>
    </v-card>

    <!-- Copy snackbar -->
    <v-snackbar
      v-model="copied"
      timeout="1500"
      color="success"
      location="bottom right"
      :elevation="0"
    >
      <v-icon size="14" start>mdi-check</v-icon>Copied!
    </v-snackbar>
  </v-container>
</template>

<script setup>
import { ref, inject, watch, onMounted, computed } from "vue";
import { useRouter } from "vue-router";
import api from "@/plugins/axios";
import { rangeToParams } from "../dashboards/tql.js";

const TIME_PRESETS = ["5m", "15m", "1h", "6h", "24h", "7d"];

const STATUS_OPTIONS = [
  { value: 0, label: "UNSET", color: "grey" },
  { value: 1, label: "OK", color: "success" },
  { value: 2, label: "ERROR", color: "error" },
];

const KIND_OPTIONS = [
  { value: 0, label: "INTERNAL" },
  { value: 1, label: "SERVER" },
  { value: 2, label: "CLIENT" },
  { value: 3, label: "PRODUCER" },
  { value: 4, label: "CONSUMER" },
];

const router = useRouter();
const sharedFilters = inject("sharedFilters");
const refreshKey = inject("refreshKey");

const traces = ref([]);
const panelOpen = ref(false);
const fromOpen = ref(false);
const toOpen = ref(false);
const copied = ref(false);

function defaultFilters() {
  const { from, to } = rangeToParams("1h");
  return {
    operation: "",
    limit: 100,
    timePreset: "1h",
    from,
    to,
    traceId: "",
    statusCodes: [],
    kinds: [],
    durationMinMs: null,
    durationMaxMs: null,
    hasError: false,
    minSpanCount: null,
    attributes: [],
  };
}

function addAttrFilter() {
  filters.value.attributes.push({ key: "", value: "", op: "=" });
}

function removeAttrFilter(idx) {
  filters.value.attributes.splice(idx, 1);
}

const filters = ref(defaultFilters());

const errorCount = computed(
  () => traces.value.filter((t) => t.status_code === 2).length,
);

const activeFilterCount = computed(() => {
  const f = filters.value;
  let n = 0;
  if (f.operation) n++;
  if (f.traceId) n++;
  if (f.statusCodes.length) n++;
  if (f.kinds.length) n++;
  if (f.durationMinMs != null || f.durationMaxMs != null) n++;
  if (f.hasError) n++;
  if (f.minSpanCount != null) n++;
  if (f.attributes.length) {
    n += f.attributes.filter(a => a.key).length;
  }
  return n;
});

function applyPreset(r) {
  const { from, to } = rangeToParams(r);
  filters.value.timePreset = r;
  filters.value.from = from;
  filters.value.to = to;
}

function toggleStatus(value) {
  const arr = filters.value.statusCodes;
  const idx = arr.indexOf(value);
  if (idx === -1) arr.push(value);
  else arr.splice(idx, 1);
  if (arr.length > 0) filters.value.hasError = false;
}

function toggleKind(value) {
  const arr = filters.value.kinds;
  const idx = arr.indexOf(value);
  if (idx === -1) arr.push(value);
  else arr.splice(idx, 1);
}

function toggleHasError() {
  filters.value.hasError = !filters.value.hasError;
  if (filters.value.hasError) filters.value.statusCodes = [];
}

function resetFilters() {
  filters.value = defaultFilters();
  load();
}

async function load() {
  const f = filters.value;
  const params = {
    service: sharedFilters.service || "",
    operation: f.operation || "",
    limit: f.limit,
    from: f.from,
    to: f.to,
  };
  if (f.traceId) params.trace_id = f.traceId;
  if (f.statusCodes.length) params.status_codes = f.statusCodes.join(",");
  if (f.kinds.length) params.kinds = f.kinds.join(",");
  if (f.durationMinMs != null && f.durationMinMs > 0)
    params.duration_min_ms = f.durationMinMs;
  if (f.durationMaxMs != null && f.durationMaxMs > 0)
    params.duration_max_ms = f.durationMaxMs;
  if (f.hasError) params.has_error = "true";
  if (f.minSpanCount != null && f.minSpanCount > 0)
    params.min_span_count = f.minSpanCount;
  
  f.attributes.forEach((attr, idx) => {
    if (attr.key) {
      params[`attr_key_${idx + 1}`] = attr.key;
      if (attr.value) {
        params[`attr_value_${idx + 1}`] = attr.value;
      }
      if (attr.op === '!=' || attr.op === 'not_exists') {
        params[`attr_invert_${idx + 1}`] = "true";
      }
    }
  });

  const { data } = await api.get("/traces", { params });
  traces.value = data || [];
}

function selectDate(d, field) {
  if (!d) {
    filters.value[field] = "";
    return;
  }
  const dt = new Date(d);
  if (field === "from") {
    dt.setHours(0, 0, 0, 0);
    fromOpen.value = false;
  } else {
    dt.setHours(23, 59, 59, 999);
    toOpen.value = false;
  }
  filters.value[field] = dt.toISOString();
  filters.value.timePreset = "";
}

function copyText(text) {
  navigator.clipboard?.writeText(text).then(() => {
    copied.value = true;
  });
}

function fmtDate(iso) {
  return new Date(iso).toLocaleDateString("pt-BR");
}

function fmtDuration(ns) {
  if (!ns) return "0ns";
  if (ns < 1_000) return `${ns}ns`;
  if (ns < 1_000_000) return `${(ns / 1_000).toFixed(1)}µs`;
  if (ns < 1e9) return `${(ns / 1_000_000).toFixed(1)}ms`;
  return `${(ns / 1e9).toFixed(2)}s`;
}

function fmtTime(iso) {
  if (!iso) return "–";
  const d = new Date(iso);
  return d.toLocaleDateString("pt-BR") + " " + d.toLocaleTimeString("pt-BR");
}

function relTime(iso) {
  if (!iso) return "";
  const s = Math.floor((Date.now() - new Date(iso).getTime()) / 1000);
  if (s < 60) return `${s}s ago`;
  if (s < 3600) return `${Math.floor(s / 60)}m ago`;
  if (s < 86400) return `${Math.floor(s / 3600)}h ago`;
  return `${Math.floor(s / 86400)}d ago`;
}

function durationColor(ns) {
  if (ns > 1e9) return "#f87171";
  if (ns > 200_000_000) return "#fbbf24";
  return "#34d399";
}
function durationBarColor(ns) {
  if (ns > 1e9) return "error";
  if (ns > 200_000_000) return "warning";
  return "success";
}
function durationPct(ns) {
  return Math.min(100, (ns / 2e9) * 100);
}

function serviceChipColor(name) {
  const colors = [
    "indigo",
    "purple",
    "blue",
    "teal",
    "green",
    "cyan",
    "orange",
    "pink",
  ];
  let h = 0;
  for (const c of name || "") h = (h * 31 + c.charCodeAt(0)) & 0xffff;
  return colors[h % colors.length];
}

function exportData(format) {
  const ts = new Date().toISOString().slice(0, 19).replace(/:/g, "-");
  if (format === "json") {
    const blob = new Blob([JSON.stringify(traces.value, null, 2)], {
      type: "application/json",
    });
    download(blob, `traces-${ts}.json`);
  } else {
    const header = [
      "trace_id",
      "service",
      "operation",
      "start_time",
      "duration_ms",
      "status",
      "span_count",
    ];
    const rows = traces.value.map((t) => [
      t.trace_id,
      t.service,
      t.operation,
      t.start_time,
      (t.duration_ns / 1e6).toFixed(3),
      t.status_code === 2 ? "error" : "ok",
      t.span_count ?? "",
    ]);
    const csv = [header, ...rows]
      .map((r) =>
        r.map((v) => `"${String(v ?? "").replace(/"/g, '""')}"`).join(","),
      )
      .join("\n");
    download(new Blob([csv], { type: "text/csv" }), `traces-${ts}.csv`);
  }
}

function download(blob, name) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = name;
  a.click();
  URL.revokeObjectURL(url);
}

onMounted(load);
watch(refreshKey, load);
watch(() => sharedFilters.service, load);
</script>

<style scoped>
.mono {
  font-family: ui-monospace, "JetBrains Mono", monospace;
}

.filter-label {
  font-size: 10px;
  color: var(--telm-text-3);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  flex-shrink: 0;
}

.filter-chip {
  cursor: pointer;
  user-select: none;
  transition: all 0.12s;
}

.filter-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.filter-section {
  padding: 8px 0;
  border-bottom: 1px solid var(--telm-border-light);
}
.filter-section:last-of-type {
  border-bottom: none;
}

.section-header {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  color: var(--telm-text-3);
  margin-bottom: 8px;
  font-weight: 500;
}

.attr-filter-row {
  background: var(--telm-bg-row);
  border-radius: 6px;
  padding: 6px 10px;
}

/* ─── Time range picker ──────────────────────────────────────────────────── */
.range-picker {
  display: flex;
  align-items: center;
  background: var(--telm-bg-row);
  border: 1px solid var(--telm-border);
  border-radius: 6px;
  padding: 2px;
  gap: 1px;
}

.range-btn {
  font-size: 11px;
  font-weight: 500;
  line-height: 1;
  padding: 4px 8px;
  border-radius: 4px;
  border: none;
  background: transparent;
  color: var(--telm-text-3);
  cursor: pointer;
  transition:
    background 0.12s,
    color 0.12s;
  white-space: nowrap;
}
.range-btn:hover {
  background: var(--telm-bg-hover);
  color: var(--telm-text-1);
}
.range-btn.active {
  background: #6366f1;
  color: #fff;
}

/* ─── Table ──────────────────────────────────────────────────────────────── */
.table-header th {
  background: var(--telm-bg-header) !important;
  color: var(--telm-text-2) !important;
  font-size: 11px !important;
  font-weight: 600 !important;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  border-bottom: 1px solid var(--telm-border) !important;
  padding-top: 10px !important;
  padding-bottom: 10px !important;
}

.trace-row td {
  background: var(--telm-bg-row) !important;
  border-bottom: 1px solid var(--telm-border-light) !important;
}
.trace-row:hover td {
  background: var(--telm-bg-hover) !important;
}

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
  transition: opacity 0.15s;
}
.trace-row:hover .copy-btn {
  opacity: 1;
}

.actions-bar {
  border-top: 1px solid var(--telm-border);
}
</style>

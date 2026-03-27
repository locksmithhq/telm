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
        <span class="text-caption text-disabled mono">{{ logs.length }} record(s)</span>
        <v-menu>
          <template #activator="{ props }">
            <v-btn v-bind="props" size="small" variant="tonal" color="default">
              <v-icon size="14" start>mdi-download</v-icon>Export<v-icon
                size="12"
                end
              >mdi-chevron-down</v-icon>
            </v-btn>
          </template>
          <v-list density="compact" min-width="140">
            <v-list-item prepend-icon="mdi-code-json" title="JSON" @click="exportData('json')" />
            <v-list-item prepend-icon="mdi-table" title="CSV" @click="exportData('csv')" />
          </v-list>
        </v-menu>
        <v-btn size="small" variant="text" color="grey" @click="panelOpen = !panelOpen">
          <v-icon size="16">{{ panelOpen ? "mdi-chevron-up" : "mdi-chevron-down" }}</v-icon>
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
        <!-- Seção 1: Busca -->
        <div class="filter-section">
          <div class="section-header">
            <v-icon size="12" color="grey">mdi-magnify</v-icon>
            <span>Search</span>
          </div>
          <div class="d-flex align-center gap-3 flex-wrap">
            <v-text-field
              v-model="filters.search"
              placeholder="Message content…"
              prepend-inner-icon="mdi-text-box-outline"
              clearable
              hide-details
              density="compact"
              style="min-width: 220px; flex: 1"
              @keydown.enter="load"
            />
            <v-text-field
              v-model="filters.operation"
              placeholder="Operation name"
              prepend-inner-icon="mdi-function-variant"
              clearable
              hide-details
              density="compact"
              style="min-width: 200px; flex: 1"
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
              >{{ r }}</button>
              <button
                class="range-btn"
                :class="{ active: filters.timePreset === '' }"
                @click="filters.timePreset = ''"
              >custom</button>
            </div>
            <template v-if="!filters.timePreset">
              <v-menu v-model="fromOpen" :close-on-content-click="false" min-width="auto">
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
              <v-menu v-model="toOpen" :close-on-content-click="false" min-width="auto">
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

        <!-- Seção 3: Severity -->
        <div class="filter-section">
          <div class="section-header">
            <v-icon size="12" color="grey">mdi-flag-variant</v-icon>
            <span>Severity</span>
          </div>
          <div class="d-flex align-center gap-3 flex-wrap">
            <v-chip
              v-for="s in severityItems"
              :key="s.value"
              :color="filters.severity === s.value ? severityChipColor(s.value) : undefined"
              :variant="filters.severity === s.value ? 'tonal' : 'outlined'"
              size="small"
              class="filter-chip"
              @click="toggleSeverity(s.value)"
            >
              <span class="sev-dot mr-1" :style="`background:${severityColor(s.value)}`" />
              {{ s.label }}
            </v-chip>
          </div>
        </div>

        <!-- Seção 4: Status -->
        <div class="filter-section">
          <div class="section-header">
            <v-icon size="12" color="grey">mdi-check-circle-outline</v-icon>
            <span>Status</span>
          </div>
          <div class="d-flex align-center gap-2 flex-wrap">
            <v-chip
              :color="filters.hasError ? 'error' : undefined"
              :variant="filters.hasError ? 'tonal' : 'outlined'"
              size="small"
              class="filter-chip"
              prepend-icon="mdi-alert-circle-outline"
              @click="filters.hasError = !filters.hasError"
            >Has Error</v-chip>
            <v-chip
              :color="filters.hasTrace ? 'primary' : undefined"
              :variant="filters.hasTrace ? 'tonal' : 'outlined'"
              size="small"
              class="filter-chip"
              prepend-icon="mdi-transit-connection-variant"
              @click="filters.hasTrace = !filters.hasTrace"
            >Has Trace</v-chip>
          </div>
        </div>

        <!-- Seção 5: Atributos -->
        <div class="filter-section">
          <div class="section-header">
            <v-icon size="12" color="grey">mdi-tag-multiple-outline</v-icon>
            <span>Attributes</span>
            <span class="text-caption text-disabled ml-1">(any match)</span>
            <v-spacer />
            <v-btn size="x-small" variant="text" color="primary" @click="addAttrFilter">
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
              <v-btn icon size="x-small" variant="text" color="grey" @click="removeAttrFilter(idx)">
                <v-icon size="14">mdi-close</v-icon>
              </v-btn>
            </div>
            <div v-if="!filters.attributes.length" class="text-caption text-disabled">
              Click "Add" to filter by log attributes
            </div>
          </div>
        </div>

        <!-- Ações -->
        <div class="d-flex align-center gap-2 mt-2 pt-2 actions-bar">
          <v-select
            v-model="filters.limit"
            :items="[25, 50, 100, 200, 500]"
            label="Results"
            hide-details
            density="compact"
            style="width: 100px"
          />
          <v-spacer />
          <v-btn size="small" variant="text" color="grey" @click="resetFilters">
            <v-icon size="14" start>mdi-refresh</v-icon>Reset
          </v-btn>
          <v-btn size="small" variant="flat" color="primary" prepend-icon="mdi-magnify" @click="load">
            Search
          </v-btn>
        </div>
      </v-card>
    </v-expand-transition>

    <!-- Severity stats bar -->
    <div v-if="logs.length" class="d-flex align-center gap-2 mb-3 flex-wrap">
      <div
        v-for="s in severityStats"
        :key="s.label"
        class="sev-stat-chip"
        :class="filters.severity === s.label ? 'sev-stat-active' : ''"
        :style="`--sev-color:${s.color};border-color:${s.color}30`"
        @click="toggleSeverityFilter(s.label)"
      >
        <span class="sev-dot" :style="`background:${s.color}`"></span>
        <span class="mono" style="font-size: 11px; font-weight: 600" :style="`color:${s.color}`">{{ s.label }}</span>
        <span class="mono" style="font-size: 12px; color: var(--telm-text-2); margin-left: 4px">{{ s.count }}</span>
      </div>
      <span
        v-if="filters.severity"
        class="text-caption text-disabled"
        style="cursor: pointer"
        @click="filters.severity = ''; load()"
      >
        <v-icon size="12">mdi-close</v-icon> clear filter
      </span>
    </div>

    <!-- Log list -->
    <v-card style="overflow: hidden">
      <!-- Column headers -->
      <div class="log-header">
        <span class="col-sev">SEVERITY</span>
        <span class="col-time">TIMESTAMP</span>
        <span class="col-svc">SERVICE</span>
        <span class="col-body">MESSAGE</span>
        <span class="col-actions"></span>
      </div>

      <template v-if="logs.length">
        <div
          v-for="(l, i) in logs"
          :key="i"
          class="log-row"
          :style="`border-left: 3px solid ${severityColor(l.severity)}`"
          :class="l._open ? 'log-row-open' : ''"
          @click="l._open = !l._open"
        >
          <!-- Main row -->
          <div class="log-main">
            <div class="col-sev">
              <span
                class="sev-badge"
                :style="`color:${severityColor(l.severity)};background:${severityColor(l.severity)}1a`"
              >
                <v-icon size="10" :style="`color:${severityColor(l.severity)}`">{{ severityIcon(l.severity) }}</v-icon>
                {{ sevLabel(l.severity) }}
              </span>
            </div>

            <div class="col-time">
              <div class="mono" style="font-size: 11px; color: var(--telm-text-1); white-space: nowrap">
                {{ fmtTimeMs(l.timestamp) }}
              </div>
              <div class="mono text-disabled" style="font-size: 10px; white-space: nowrap">
                {{ relTime(l.timestamp) }}
              </div>
            </div>

            <div class="col-svc">
              <v-chip :color="serviceChipColor(l.service)" class="mono" size="small">{{ l.service }}</v-chip>
            </div>

            <div class="col-body overflow-hidden">
              <div
                class="mono log-body"
                :style="`color:${severityTextColor(l.severity)}`"
                :class="l._open ? 'log-body-open' : ''"
                v-html="highlightSearch(l.body)"
              ></div>
              <div v-if="!l._open && hasAttrs(l)" class="attr-preview mono">
                <span v-for="[k, v] in previewAttrs(l)" :key="k" class="attr-preview-item">
                  <span style="color: #818cf8">{{ k }}</span><span style="color: var(--telm-text-4)">=</span><span style="color: var(--telm-text-2)">{{ truncVal(String(v)) }}</span>
                </span>
                <span v-if="attrCount(l) > 3" style="color: var(--telm-text-4); font-size: 10px">
                  +{{ attrCount(l) - 3 }} more</span>
              </div>
            </div>

            <div class="col-actions d-flex align-center gap-1">
              <v-tooltip v-if="l.trace_id" text="View trace" location="left">
                <template #activator="{ props }">
                  <v-btn
                    v-bind="props"
                    icon
                    size="x-small"
                    variant="text"
                    color="primary"
                    @click.stop="router.push({ path: '/traces/' + l.trace_id, query: l.span_id ? { highlight: l.span_id } : {} })"
                  >
                    <v-icon size="14">mdi-transit-connection-variant</v-icon>
                  </v-btn>
                </template>
              </v-tooltip>
              <v-tooltip text="View JSON" location="left">
                <template #activator="{ props }">
                  <v-btn
                    v-bind="props"
                    icon
                    size="x-small"
                    variant="text"
                    :color="l._jsonOpen ? 'primary' : 'grey'"
                    @click.stop="l._open = true; l._jsonOpen = !l._jsonOpen"
                  >
                    <v-icon size="13">mdi-code-braces</v-icon>
                  </v-btn>
                </template>
              </v-tooltip>
              <v-icon size="13" :color="l._open ? 'primary' : 'grey'">
                {{ l._open ? "mdi-chevron-up" : "mdi-chevron-down" }}
              </v-icon>
            </div>
          </div>

          <!-- Expanded detail -->
          <v-expand-transition>
            <div v-if="l._open" class="log-detail" @click.stop>
              <div class="detail-grid mb-3">
                <div class="detail-item">
                  <span class="detail-key">Timestamp</span>
                  <span class="detail-val">{{ fmtTimeFull(l.timestamp) }}</span>
                </div>
                <div class="detail-item">
                  <span class="detail-key">Severity</span>
                  <span class="mono font-weight-bold" style="font-size: 12px" :style="`color:${severityColor(l.severity)}`">
                    {{ l.severity || "—" }}
                    <span v-if="l.severity_number" style="color: var(--telm-text-3); font-weight: 400">({{ l.severity_number }})</span>
                  </span>
                </div>
                <div class="detail-item">
                  <span class="detail-key">Service</span>
                  <span class="detail-val">{{ l.service }}</span>
                </div>
                <div v-if="l.trace_id" class="detail-item">
                  <span class="detail-key">Trace ID</span>
                  <div class="d-flex align-center gap-1">
                    <code class="detail-val-code">{{ l.trace_id }}</code>
                    <v-btn icon size="x-small" variant="text" color="grey" @click="copyText(l.trace_id)">
                      <v-icon size="11">mdi-content-copy</v-icon>
                    </v-btn>
                    <v-btn icon size="x-small" variant="text" color="primary" @click="router.push({ path: '/traces/' + l.trace_id, query: l.span_id ? { highlight: l.span_id } : {} })">
                      <v-icon size="11">mdi-open-in-new</v-icon>
                    </v-btn>
                  </div>
                </div>
                <div v-if="l.span_id" class="detail-item">
                  <span class="detail-key">Span ID</span>
                  <div class="d-flex align-center gap-1">
                    <code class="detail-val-code">{{ l.span_id }}</code>
                    <v-btn icon size="x-small" variant="text" color="grey" @click="copyText(l.span_id)">
                      <v-icon size="11">mdi-content-copy</v-icon>
                    </v-btn>
                  </div>
                </div>
              </div>

              <div class="section-label mb-2">
                <v-icon size="11" class="mr-1">mdi-text-long</v-icon>Message
              </div>
              <div class="log-body-full mb-3" v-html="highlightSearch(l.body)"></div>

              <template v-if="hasAttrs(l) && !l._jsonOpen">
                <div class="section-label mb-2">
                  <v-icon size="11" class="mr-1">mdi-tag-multiple-outline</v-icon>
                  Attributes ({{ attrCount(l) }})
                </div>
                <div class="attr-table mb-1">
                  <div v-for="[k, v] in Object.entries(l.attributes || {})" :key="k" class="attr-row">
                    <span class="attr-key">{{ k }}</span>
                    <span class="attr-eq">=</span>
                    <span class="attr-val">{{ String(v) }}</span>
                    <v-btn icon size="x-small" variant="text" color="grey" class="attr-copy" @click="copyText(String(v))">
                      <v-icon size="10">mdi-content-copy</v-icon>
                    </v-btn>
                  </div>
                </div>
              </template>

              <div v-if="l._jsonOpen" class="log-json mt-2">
                <div class="d-flex align-center justify-space-between mb-1">
                  <span class="section-label">
                    <v-icon size="11" class="mr-1">mdi-code-braces</v-icon>JSON
                  </span>
                  <v-btn icon size="x-small" variant="text" color="grey" @click="copyText(JSON.stringify(logToJson(l), null, 2))">
                    <v-icon size="11">mdi-content-copy</v-icon>
                  </v-btn>
                </div>
                <pre>{{ JSON.stringify(logToJson(l), null, 2) }}</pre>
              </div>
            </div>
          </v-expand-transition>
        </div>
      </template>

      <div v-else class="text-center py-12">
        <v-icon size="36" color="grey-darken-2">mdi-text-box-multiple-outline</v-icon>
        <div class="text-caption text-disabled mt-2">No logs found</div>
      </div>
    </v-card>

    <v-snackbar v-model="copied" timeout="1500" color="success" location="bottom right" :elevation="0">
      <v-icon size="14" start>mdi-check</v-icon>Copied!
    </v-snackbar>
  </v-container>
</template>

<script setup>
import { ref, inject, watch, onMounted, reactive, computed } from "vue";
import { useRouter } from "vue-router";
import api from "@/plugins/axios";
import { rangeToParams } from "../dashboards/tql.js";

const router = useRouter();
const sharedFilters = inject("sharedFilters");
const refreshKey = inject("refreshKey");

const TIME_PRESETS = ["5m", "15m", "1h", "6h", "24h", "7d"];

const severityItems = ["TRACE", "DEBUG", "INFO", "WARN", "ERROR", "FATAL"].map(v => ({ label: v, value: v }));

const logs = ref([]);
const panelOpen = ref(false);
const fromOpen = ref(false);
const toOpen = ref(false);
const copied = ref(false);

function defaultFilters() {
  const { from, to } = rangeToParams("1h");
  return {
    search: "",
    operation: "",
    severity: "",
    timePreset: "1h",
    from,
    to,
    limit: 100,
    hasError: false,
    hasTrace: false,
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

const activeFilterCount = computed(() => {
  const f = filters.value;
  let n = 0;
  if (f.search) n++;
  if (f.operation) n++;
  if (f.severity) n++;
  if (f.hasError) n++;
  if (f.hasTrace) n++;
  if (f.attributes.length) n += f.attributes.filter(a => a.key).length;
  return n;
});

function applyPreset(r) {
  const { from, to } = rangeToParams(r);
  filters.value.timePreset = r;
  filters.value.from = from;
  filters.value.to = to;
}

function toggleSeverity(value) {
  filters.value.severity = filters.value.severity === value ? "" : value;
}

function toggleSeverityFilter(label) {
  filters.value.severity = filters.value.severity === label ? "" : label;
  load();
}

function resetFilters() {
  filters.value = defaultFilters();
  load();
}

async function load() {
  const f = filters.value;
  const params = {
    service: sharedFilters.service || "",
    search: f.search || "",
    severity: f.severity || "",
    limit: f.limit,
    from: f.from,
    to: f.to,
  };
  if (f.operation) params.operation = f.operation;
  if (f.hasError) params.has_error = "true";
  if (f.hasTrace) params.has_trace = "true";
  f.attributes.forEach((attr, idx) => {
    if (attr.key) {
      params[`attr_key_${idx + 1}`] = attr.key;
      if (attr.value) params[`attr_value_${idx + 1}`] = attr.value;
      if (attr.op === '!=' || attr.op === 'not_exists') params[`attr_invert_${idx + 1}`] = "true";
    }
  });
  const { data } = await api.get("/logs", { params });
  logs.value = (data || []).map(l => reactive({ ...l, _open: false, _jsonOpen: false }));
}

function selectDate(d, field) {
  if (!d) { filters.value[field] = ""; return; }
  const dt = new Date(d);
  if (field === "from") { dt.setHours(0, 0, 0, 0); fromOpen.value = false; }
  else { dt.setHours(23, 59, 59, 999); toOpen.value = false; }
  filters.value[field] = dt.toISOString();
  filters.value.timePreset = "";
}

function copyText(text) { navigator.clipboard?.writeText(text).then(() => { copied.value = true; }); }

function fmtDate(iso) { return new Date(iso).toLocaleDateString("pt-BR"); }

function fmtTimeMs(iso) {
  if (!iso) return "–";
  const d = new Date(iso);
  return `${d.toLocaleTimeString("pt-BR", { hour12: false })}.${String(d.getMilliseconds()).padStart(3, "0")}`;
}

function fmtTimeFull(iso) {
  if (!iso) return "–";
  const d = new Date(iso);
  return `${d.toLocaleDateString("pt-BR")} ${fmtTimeMs(iso)}`;
}

function relTime(iso) {
  if (!iso) return "";
  const s = Math.floor((Date.now() - new Date(iso).getTime()) / 1000);
  if (s < 60) return `${s}s ago`;
  if (s < 3600) return `${Math.floor(s / 60)}m ago`;
  if (s < 86400) return `${Math.floor(s / 3600)}h ago`;
  return `${Math.floor(s / 86400)}d ago`;
}

function severityColor(s) {
  s = (s || "").toUpperCase();
  if (s === "FATAL") return "#dc2626";
  if (s === "ERROR") return "#ef4444";
  if (s === "WARN") return "#f59e0b";
  if (s === "INFO") return "#6366f1";
  if (s === "DEBUG") return "#22d3ee";
  if (s === "TRACE") return "#64748b";
  return "#475569";
}

function severityChipColor(s) {
  s = (s || "").toUpperCase();
  if (s === "FATAL") return "error";
  if (s === "ERROR") return "error";
  if (s === "WARN") return "warning";
  if (s === "INFO") return "primary";
  if (s === "DEBUG") return "info";
  return "grey";
}

function severityTextColor(s) {
  s = (s || "").toUpperCase();
  if (s === "FATAL" || s === "ERROR") return "#fca5a5";
  if (s === "WARN") return "#fde68a";
  if (s === "INFO") return "#c7d2fe";
  if (s === "DEBUG") return "#a5f3fc";
  return "#94a3b8";
}

function severityIcon(s) {
  s = (s || "").toUpperCase();
  if (s === "FATAL") return "mdi-skull-outline";
  if (s === "ERROR") return "mdi-alert-circle-outline";
  if (s === "WARN") return "mdi-alert-outline";
  if (s === "INFO") return "mdi-information-outline";
  if (s === "DEBUG") return "mdi-bug-outline";
  if (s === "TRACE") return "mdi-magnify";
  return "mdi-circle-small";
}

function sevLabel(s) {
  s = (s || "").toUpperCase();
  const m = { FATAL: "FATAL", ERROR: "ERROR", WARN: "WARN", INFO: "INFO", DEBUG: "DEBUG", TRACE: "TRACE" };
  return m[s] || s.slice(0, 5) || "?";
}

function serviceChipColor(name) {
  const colors = ["indigo", "purple", "blue", "teal", "green", "cyan", "orange", "pink"];
  let h = 0;
  for (const c of name || "") h = (h * 31 + c.charCodeAt(0)) & 0xffff;
  return colors[h % colors.length];
}

function hasAttrs(l) { return l.attributes && typeof l.attributes === "object" && Object.keys(l.attributes).length > 0; }
function attrCount(l) { return hasAttrs(l) ? Object.keys(l.attributes).length : 0; }
function previewAttrs(l) { return hasAttrs(l) ? Object.entries(l.attributes).slice(0, 3) : []; }
function truncVal(v) { return v.length > 32 ? v.slice(0, 30) + "…" : v; }

function highlightSearch(body) {
  if (!body) return "";
  const escaped = body.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
  if (!filters.value.search) return escaped;
  const term = filters.value.search.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  return escaped.replace(new RegExp(term, "gi"), m => `<mark class="search-hl">${m}</mark>`);
}

function logToJson(l) {
  return {
    timestamp: l.timestamp,
    severity: l.severity,
    severity_number: l.severity_number,
    service: l.service,
    body: l.body,
    trace_id: l.trace_id || null,
    span_id: l.span_id || null,
    attributes: l.attributes || {},
  };
}

function exportData(format) {
  const ts = new Date().toISOString().slice(0, 19).replace(/:/g, "-");
  if (format === "json") {
    const blob = new Blob([JSON.stringify(logs.value.map(({ _open, ...rest }) => rest), null, 2)], { type: "application/json" });
    dl(blob, `logs-${ts}.json`);
  } else {
    const header = ["timestamp", "severity", "severity_number", "service", "body", "trace_id", "span_id"];
    const rows = logs.value.map(l => [l.timestamp, l.severity || "", l.severity_number || "", l.service, l.body, l.trace_id || "", l.span_id || ""]);
    const csv = [header, ...rows].map(r => r.map(v => `"${String(v ?? "").replace(/"/g, '""')}"`).join(",")).join("\n");
    dl(new Blob([csv], { type: "text/csv" }), `logs-${ts}.csv`);
  }
}

function dl(blob, name) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url; a.download = name; a.click();
  URL.revokeObjectURL(url);
}

onMounted(load);
watch(refreshKey, load);
watch(() => sharedFilters.service, load);
</script>

<style scoped>
.mono { font-family: ui-monospace, "JetBrains Mono", monospace; }

/* ── Filter panel ─────────────────────────────────────────────────────────── */
.filter-label {
  font-size: 10px; color: var(--telm-text-3);
  text-transform: uppercase; letter-spacing: 0.05em; flex-shrink: 0;
}
.filter-chip { cursor: pointer; user-select: none; transition: all 0.12s; }
.filter-group { display: flex; align-items: center; gap: 8px; }
.filter-section { padding: 8px 0; border-bottom: 1px solid var(--telm-border-light); }
.filter-section:last-of-type { border-bottom: none; }
.section-header {
  display: flex; align-items: center; gap: 6px;
  font-size: 11px; color: var(--telm-text-3);
  margin-bottom: 8px; font-weight: 500;
}
.attr-filter-row {
  background: var(--telm-bg-row);
  border-radius: 6px;
  padding: 6px 10px;
}
.range-picker {
  display: flex; align-items: center;
  background: var(--telm-bg-row);
  border: 1px solid var(--telm-border);
  border-radius: 6px;
  padding: 2px; gap: 1px;
}
.range-btn {
  font-size: 11px; font-weight: 500; line-height: 1;
  padding: 4px 8px; border-radius: 4px;
  border: none; background: transparent;
  color: var(--telm-text-3); cursor: pointer;
  transition: background 0.12s, color 0.12s; white-space: nowrap;
}
.range-btn:hover { background: var(--telm-bg-hover); color: var(--telm-text-1); }
.range-btn.active { background: #6366f1; color: #fff; }

/* ── Severity stats ───────────────────────────────────────────────────────── */
.sev-dot { width: 7px; height: 7px; border-radius: 50%; display: inline-block; flex-shrink: 0; }
.sev-stat-chip {
  display: inline-flex; align-items: center; gap: 6px;
  padding: 4px 10px; border-radius: 20px; border: 1px solid;
  background: var(--telm-bg-attr); cursor: pointer;
  transition: background 0.15s; user-select: none;
}
.sev-stat-chip:hover { background: var(--telm-bg-code); }
.sev-stat-active { background: color-mix(in srgb, var(--sev-color) 12%, transparent) !important; }

/* ── Log list header ───────────────────────────────────────────────────────── */
.log-header {
  display: grid;
  grid-template-columns: 90px 148px 130px 1fr 60px;
  align-items: center;
  padding: 7px 12px;
  background: var(--telm-bg-code);
  border-bottom: 1px solid var(--telm-border);
  font-family: ui-monospace, monospace;
  font-size: 10px;
  letter-spacing: 0.07em;
  text-transform: uppercase;
  color: var(--telm-text-2);
}

/* ── Log row ───────────────────────────────────────────────────────────────── */
.log-row { border-bottom: 1px solid var(--telm-border-light); cursor: pointer; transition: background 0.1s; }
.log-row:last-child { border-bottom: none; }
.log-row:hover { background: var(--telm-bg-attr); }
.log-row-open { background: var(--telm-bg-row); }
.log-row-open:hover { background: var(--telm-bg-hover); }
.log-main {
  display: grid;
  grid-template-columns: 90px 148px 130px 1fr 60px;
  align-items: center;
  gap: 0;
  padding: 6px 12px;
  min-height: 40px;
}
.col-sev { padding-right: 8px; }
.col-time { padding-right: 8px; }
.col-svc { padding-right: 8px; }
.col-body { min-width: 0; }
.col-actions { display: flex; align-items: center; justify-content: flex-end; gap: 2px; padding-left: 4px; }

/* ── Severity badge ─────────────────────────────────────────────────────────── */
.sev-badge {
  font-family: ui-monospace, monospace; font-size: 10px; font-weight: 700;
  padding: 2px 6px; border-radius: 4px; letter-spacing: 0.04em;
  display: inline-flex; align-items: center; gap: 3px; white-space: nowrap;
}

/* ── Body ───────────────────────────────────────────────────────────────────── */
.log-body { font-size: 12px; line-height: 1.45; display: -webkit-box; -webkit-line-clamp: 1; -webkit-box-orient: vertical; overflow: hidden; }
.log-body-open { display: block; -webkit-line-clamp: unset; white-space: pre-wrap; word-break: break-word; }
.attr-preview { font-size: 10px; color: var(--telm-text-4); margin-top: 3px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.attr-preview-item { margin-right: 10px; }

/* ── Detail panel ───────────────────────────────────────────────────────────── */
.log-detail { padding: 12px 16px 14px 16px; background: var(--telm-bg-detail); border-top: 1px solid var(--telm-border-light); }
.detail-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: 8px; }
.detail-item { display: flex; flex-direction: column; gap: 2px; padding: 6px 10px; background: var(--telm-bg-code); border-radius: 6px; border: 1px solid var(--telm-border-light); }
.detail-key { font-size: 9px; letter-spacing: 0.08em; text-transform: uppercase; color: var(--telm-text-3); font-family: ui-monospace, monospace; }
.detail-val { font-size: 12px; color: var(--telm-text-1); font-family: ui-monospace, monospace; }
.detail-val-code { font-size: 10px; color: var(--telm-text-2); word-break: break-all; line-height: 1.4; font-family: ui-monospace, monospace; }
.section-label { font-size: 10px; letter-spacing: 0.08em; text-transform: uppercase; color: var(--telm-text-3); font-family: ui-monospace, monospace; display: flex; align-items: center; }
.log-body-full { font-family: ui-monospace, monospace; font-size: 12px; line-height: 1.6; color: var(--telm-text-1); background: var(--telm-bg-code); border: 1px solid var(--telm-border); border-radius: 6px; padding: 10px 12px; white-space: pre-wrap; word-break: break-word; }

/* ── Attributes table ───────────────────────────────────────────────────────── */
.attr-table { display: flex; flex-direction: column; gap: 1px; border-radius: 6px; overflow: hidden; border: 1px solid var(--telm-border); }
.attr-row { display: grid; grid-template-columns: 240px 16px 1fr auto; align-items: center; gap: 4px; padding: 5px 8px; background: var(--telm-bg-attr); font-family: ui-monospace, monospace; font-size: 11px; transition: background 0.1s; }
.attr-row:hover { background: var(--telm-bg-hover); }
.attr-row:nth-child(even) { background: var(--telm-bg-code); }
.attr-row:nth-child(even):hover { background: var(--telm-bg-hover); }
.attr-key { color: #818cf8; font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.attr-eq { color: var(--telm-text-4); }
.attr-val { color: var(--telm-text-2); word-break: break-all; }
.attr-copy { opacity: 0; }
.attr-row:hover .attr-copy { opacity: 1; }

/* ── JSON view ──────────────────────────────────────────────────────────────── */
.log-json { background: var(--telm-bg-code); border: 1px solid var(--telm-border); border-radius: 6px; overflow: hidden; padding: 10px 12px; }
.log-json pre { font-family: ui-monospace, monospace; font-size: 11px; line-height: 1.6; color: var(--telm-text-1); white-space: pre-wrap; word-break: break-all; margin: 0; }

/* ── Search highlight ───────────────────────────────────────────────────────── */
:deep(.search-hl) { background: rgba(250, 204, 21, 0.25); color: #fde047; border-radius: 2px; padding: 0 1px; }

/* ── Actions bar ────────────────────────────────────────────────────────────── */
.actions-bar { border-top: 1px solid var(--telm-border); }
</style>

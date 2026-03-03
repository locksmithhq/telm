<template>
  <v-container fluid class="pa-3">
    <!-- Filters -->
    <div class="d-flex align-center gap-3 mb-3 flex-wrap">
      <v-text-field
        v-model="search"
        placeholder="Filter metrics…"
        prepend-inner-icon="mdi-magnify"
        clearable
        hide-details
        density="compact"
        style="min-width: 200px; max-width: 280px"
      />
      <v-select
        v-model="typeFilter"
        :items="['gauge', 'sum', 'histogram', 'summary']"
        placeholder="All types"
        hide-details
        density="compact"
        clearable
        style="min-width: 140px; max-width: 170px"
      />
      <v-spacer />
      <span class="text-caption text-disabled"
        >{{ filtered.length }} metric(s)</span
      >
    </div>

    <!-- Grid -->
    <v-row dense>
      <v-col v-for="m in filtered" :key="metricKey(m)" cols="12" sm="6" xl="4">
        <v-card class="overflow-hidden">
          <!-- Header -->
          <div class="metric-header" @click="toggleMetric(m)">
            <span
              class="type-badge"
              :style="`color:${typeBadgeColor(m.type)};background:${typeBadgeColor(m.type)}1a`"
            >
              {{ typeIcon(m.type) }} {{ m.type }}
            </span>
            <div class="flex-grow-1 overflow-hidden mx-2">
              <div
                class="mono text-truncate"
                style="
                  font-size: 12px;
                  font-weight: 500;
                  color: var(--telm-text-1);
                "
              >
                {{ m.name }}
              </div>
              <div class="d-flex align-center gap-1 mt-0">
                <v-chip :color="serviceChipColor(m.service)" class="mono">{{
                  m.service
                }}</v-chip>
                <span
                  v-if="m.unit"
                  class="text-disabled mono"
                  style="font-size: 10px"
                  >({{ m.unit }})</span
                >
              </div>
            </div>
            <div class="d-flex align-center gap-2 flex-shrink-0">
              <div class="text-right">
                <div
                  class="mono font-weight-bold text-success"
                  style="font-size: 15px; line-height: 1"
                >
                  {{ fmtValue(m) }}
                </div>
                <div class="text-disabled mono" style="font-size: 10px">
                  {{ relTime(m.timestamp) }}
                </div>
              </div>
              <v-icon size="14" color="grey">
                {{
                  expanded[metricKey(m)] ? "mdi-chevron-up" : "mdi-chevron-down"
                }}
              </v-icon>
            </div>
          </div>

          <!-- Chart panel — @after-enter garante que o DOM tem tamanho antes de inicializar o ECharts -->
          <v-expand-transition @after-enter="onPanelEnter(m)">
            <div v-if="expanded[metricKey(m)]" class="chart-panel">
              <div class="d-flex gap-1 mb-2">
                <button
                  v-for="rv in ['1h', '6h', '24h', '7d']"
                  :key="rv"
                  class="range-btn mono"
                  :class="{ active: (ranges[metricKey(m)] || '1h') === rv }"
                  @click.stop="changeRange(m, rv)"
                >
                  {{ rv }}
                </button>
              </div>

              <!-- chart div sempre no DOM; overlay cobre enquanto carrega -->
              <div style="position: relative; height: 100px">
                <div :id="elId(m)" style="width: 100%; height: 100%"></div>
                <div
                  v-if="loadingChart[metricKey(m)]"
                  class="chart-overlay d-flex align-center justify-center"
                >
                  <v-progress-circular
                    indeterminate
                    size="20"
                    width="2"
                    color="primary"
                  />
                </div>
              </div>
            </div>
          </v-expand-transition>
        </v-card>
      </v-col>
    </v-row>

    <div v-if="!filtered.length" class="text-center py-12">
      <v-icon size="32" color="grey-darken-2">mdi-chart-line</v-icon>
      <div class="text-caption text-disabled mt-2">No metrics found</div>
    </div>
  </v-container>
</template>

<script setup>
import {
  ref,
  inject,
  watch,
  onMounted,
  onUnmounted,
  computed,
  nextTick,
} from "vue";
import * as echarts from "echarts";
import api from "@/plugins/axios";

const sharedFilters = inject("sharedFilters");
const refreshKey = inject("refreshKey");
const isDark = inject("isDark");

function ct() {
  const dark = isDark.value;
  return {
    axis: dark ? "#94a3b8" : "#64748b",
    axisDim: dark ? "#64748b" : "#94a3b8",
    grid: dark ? "#1e2d45" : "#e2e8f0",
    tooltipBg: dark ? "#1a2235" : "#ffffff",
    tooltipBdr: dark ? "#2d3f5c" : "#cbd5e1",
    tooltipText: dark ? "#cbd5e1" : "#0f172a",
  };
}

const catalog = ref([]);
const search = ref("");
const typeFilter = ref(null);
const expanded = ref({});
const ranges = ref({});
const loadingChart = ref({});
const chartInstances = {};

const filtered = computed(() => {
  let c = catalog.value;
  if (search.value)
    c = c.filter((m) =>
      m.name.toLowerCase().includes(search.value.toLowerCase()),
    );
  if (typeFilter.value) c = c.filter((m) => m.type === typeFilter.value);
  return c;
});

function metricKey(m) {
  return m.name + "||" + m.service;
}
function elId(m) {
  return "mc_" + (m.name + "_" + m.service).replace(/[^a-z0-9]/gi, "_");
}

function toggleMetric(m) {
  const key = metricKey(m);
  if (expanded.value[key]) {
    chartInstances[key]?.dispose();
    delete chartInstances[key];
    expanded.value = { ...expanded.value, [key]: false };
  } else {
    if (!ranges.value[key]) ranges.value = { ...ranges.value, [key]: "1h" };
    expanded.value = { ...expanded.value, [key]: true };
    // renderChart será chamado por @after-enter quando a animação terminar
  }
}

// Chamado após a animação de expand — DOM tem altura correta aqui
async function onPanelEnter(m) {
  await renderChart(m, ranges.value[metricKey(m)] || "1h");
}

async function changeRange(m, range) {
  const key = metricKey(m);
  ranges.value = { ...ranges.value, [key]: range };
  chartInstances[key]?.dispose();
  delete chartInstances[key];
  await renderChart(m, range);
}

function rangeParams(range) {
  const to = new Date();
  const mins = { "1h": 60, "6h": 360, "24h": 1440, "7d": 10080 };
  const from = new Date(to - (mins[range] || 60) * 60_000);
  return { from: from.toISOString(), to: to.toISOString() };
}

async function renderChart(m, range) {
  const key = metricKey(m);
  loadingChart.value = { ...loadingChart.value, [key]: true };
  try {
    const { from, to } = rangeParams(range);
    const { data: pts } = await api.get("/metrics/series", {
      params: { name: m.name, service: m.service, from, to },
    });

    // Desativa overlay ANTES do nextTick para o el estar visível
    loadingChart.value = { ...loadingChart.value, [key]: false };
    await nextTick();

    const el = document.getElementById(elId(m));
    if (!el) return;

    const chart =
      echarts.getInstanceByDom(el) ||
      echarts.init(el, null, { renderer: "canvas" });
    const labels = pts.map((p) => {
      const d = new Date(p.time);
      if (range === "7d")
        return d.toLocaleDateString("pt-BR", {
          month: "short",
          day: "numeric",
        });
      return d.toLocaleTimeString("pt-BR", {
        hour: "2-digit",
        minute: "2-digit",
      });
    });
    const vals = pts.map((p) =>
      p.avg_value != null ? +p.avg_value.toFixed(4) : p.total_count || 0,
    );
    const c = ct();
    chart.setOption({
      backgroundColor: "transparent",
      grid: { top: 4, right: 4, bottom: 20, left: 4, containLabel: true },
      xAxis: {
        type: "category",
        data: labels,
        axisLine: { lineStyle: { color: c.grid } },
        axisTick: { show: false },
        axisLabel: { color: c.axisDim, fontSize: 9 },
        splitLine: { show: false },
      },
      yAxis: {
        type: "value",
        axisLine: { show: false },
        axisTick: { show: false },
        axisLabel: { color: c.axisDim, fontSize: 9 },
        splitLine: { lineStyle: { color: c.grid, type: "dashed" } },
      },
      tooltip: {
        trigger: "axis",
        backgroundColor: c.tooltipBg,
        borderColor: c.tooltipBdr,
        textStyle: { color: c.tooltipText, fontSize: 11 },
      },
      series: [
        {
          type: "line",
          data: vals,
          smooth: 0.4,
          lineStyle: { color: "#10b981", width: 1.5 },
          itemStyle: { color: "#10b981" },
          showSymbol: pts.length <= 30,
          symbolSize: 3,
          areaStyle: {
            color: {
              type: "linear",
              x: 0,
              y: 0,
              x2: 0,
              y2: 1,
              colorStops: [
                { offset: 0, color: "rgba(16,185,129,.2)" },
                { offset: 1, color: "rgba(16,185,129,.01)" },
              ],
            },
          },
        },
      ],
    });
    chartInstances[key] = chart;
  } catch {
    loadingChart.value = { ...loadingChart.value, [key]: false };
  }
}

function fmtValue(m) {
  if (m.value_double != null) {
    const v = m.value_double;
    if (Math.abs(v) >= 1e6) return (v / 1e6).toFixed(2) + "M";
    if (Math.abs(v) >= 1e3) return (v / 1e3).toFixed(2) + "K";
    return v.toFixed(v % 1 === 0 ? 0 : 3);
  }
  if (m.value_int != null) return String(m.value_int);
  if (m.count != null) return "n=" + m.count;
  return "–";
}

function relTime(iso) {
  if (!iso) return "";
  const s = Math.floor((Date.now() - new Date(iso).getTime()) / 1000);
  if (s < 60) return `${s}s ago`;
  if (s < 3600) return `${Math.floor(s / 60)}m ago`;
  if (s < 86400) return `${Math.floor(s / 3600)}h ago`;
  return `${Math.floor(s / 86400)}d ago`;
}

function typeIcon(t) {
  return { gauge: "◉", sum: "∑", histogram: "▦", summary: "◈" }[t] || "◌";
}
function typeBadgeColor(t) {
  return (
    {
      gauge: "#8b5cf6",
      sum: "#3b82f6",
      histogram: "#f97316",
      summary: "#14b8a6",
    }[t] || "#64748b"
  );
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

async function load() {
  const { data } = await api.get("/metrics/catalog", {
    params: { service: sharedFilters.service || "" },
  });
  catalog.value = data || [];
}

onMounted(load);
watch(refreshKey, load);
watch(isDark, async () => {
  await nextTick();
  for (const m of catalog.value) {
    const key = metricKey(m);
    if (expanded.value[key]) {
      chartInstances[key]?.dispose();
      delete chartInstances[key];
      await renderChart(m, ranges.value[key] || "1h");
    }
  }
});
onUnmounted(() => {
  Object.values(chartInstances).forEach((c) => c?.dispose());
});
</script>

<style scoped>
.mono {
  font-family: ui-monospace, "JetBrains Mono", monospace;
}

.metric-header {
  display: flex;
  align-items: center;
  padding: 10px 12px;
  cursor: pointer;
  user-select: none;
}
.metric-header:hover {
  background: rgba(255, 255, 255, 0.02);
}

.type-badge {
  font-family: ui-monospace, "JetBrains Mono", monospace;
  font-size: 10px;
  font-weight: 600;
  padding: 2px 6px;
  border-radius: 4px;
  letter-spacing: 0.04em;
  white-space: nowrap;
  flex-shrink: 0;
  margin-right: 8px;
}

.chart-panel {
  padding: 8px 12px 10px;
  border-top: 1px solid rgba(255, 255, 255, 0.05);
  background: rgba(0, 0, 0, 0.12);
}

.chart-overlay {
  position: absolute;
  inset: 0;
  background: rgba(9, 14, 26, 0.7);
}

.range-btn {
  font-size: 10px;
  padding: 2px 7px;
  border-radius: 4px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: transparent;
  color: #64748b;
  cursor: pointer;
  transition: all 0.1s;
}
.range-btn:hover {
  border-color: rgba(99, 102, 241, 0.4);
  color: #a5b4fc;
}
.range-btn.active {
  background: rgba(99, 102, 241, 0.2);
  border-color: rgba(99, 102, 241, 0.5);
  color: #a5b4fc;
}
</style>

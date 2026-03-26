<template>
  <v-app>
    <v-app-bar :elevation="0" border="b" color="surface" height="44">
      <div class="d-flex align-center h-100 pl-4 gap-3">
        <span
          class="mono text-caption font-weight-bold text-primary pr-2"
          style="letter-spacing: 0.1em"
          >⟋ telm</span
        >

        <v-divider vertical class="my-2" opacity="0.2" />

        <v-tabs
          v-model="currentTab"
          density="compact"
          color="primary"
          height="44"
          @update:model-value="navigate"
        >
          <v-tab
            value="dashboard"
            min-width="0"
            class="px-3 text-caption font-weight-medium"
          >
            <v-icon size="13" start>mdi-view-dashboard-outline</v-icon>Dashboard
          </v-tab>
          <v-tab
            value="traces"
            min-width="0"
            class="px-3 text-caption font-weight-medium"
          >
            <v-icon size="13" start>mdi-transit-connection-variant</v-icon
            >Traces
          </v-tab>
          <v-tab
            value="metrics"
            min-width="0"
            class="px-3 text-caption font-weight-medium"
          >
            <v-icon size="13" start>mdi-chart-line</v-icon>Metrics
          </v-tab>
          <v-tab
            value="logs"
            min-width="0"
            class="px-3 text-caption font-weight-medium"
          >
            <v-icon size="13" start>mdi-text-box-outline</v-icon>Logs
          </v-tab>
          <v-tab
            value="dashboards"
            min-width="0"
            class="px-3 text-caption font-weight-medium"
          >
            <v-icon size="13" start>mdi-monitor-dashboard</v-icon>Dashboards
          </v-tab>
        </v-tabs>
      </div>

      <v-spacer />

      <div class="d-flex align-center gap-2 pr-3">
        <v-select
          v-model="sharedFilters.service"
          :items="services"
          hide-details
          density="compact"
          style="min-width: 150px; max-width: 200px"
          placeholder="All services"
          clearable
          @update:model-value="onServiceChange"
        />

        <v-tooltip
          :text="isDark ? 'Switch to light mode' : 'Switch to dark mode'"
          location="bottom"
        >
          <template #activator="{ props }">
            <v-btn
              v-bind="props"
              icon
              size="small"
              variant="text"
              @click="toggleTheme"
            >
              <v-icon size="16">{{
                isDark ? "mdi-weather-sunny" : "mdi-weather-night"
              }}</v-icon>
            </v-btn>
          </template>
        </v-tooltip>

        <v-btn
          icon
          size="small"
          variant="text"
          :loading="loading"
          @click="refresh"
        >
          <v-icon size="16">mdi-refresh</v-icon>
        </v-btn>
      </div>
    </v-app-bar>

    <v-main>
      <router-view v-slot="{ Component }">
        <keep-alive :exclude="['TraceDetailView']">
          <component :is="Component" />
        </keep-alive>
      </router-view>
    </v-main>
  </v-app>
</template>

<script setup>
import { ref, reactive, computed, onMounted, provide, watch } from "vue";
import { useRouter, useRoute } from "vue-router";
import { useTheme } from "vuetify";
import api from "@/plugins/axios";

const router = useRouter();
const route = useRoute();
const theme = useTheme();

const isDark = computed(() => theme.global.name.value === "dark");

function toggleTheme() {
  const next = isDark.value ? "light" : "dark";
  theme.global.name.value = next;
  localStorage.setItem("telm-theme", next);
}

const services = ref([]);
const loading = ref(false);
const sharedFilters = reactive({ service: null });
const refreshKey = ref(0);

const tabRouteMap = {
  dashboard: "/dashboard",
  traces: "/traces",
  metrics: "/metrics",
  logs: "/logs",
  dashboards: "/dashboards",
};
const routeTabMap = Object.fromEntries(
  Object.entries(tabRouteMap).map(([k, v]) => [v, k]),
);
const currentTab = ref("dashboard");

watch(
  () => route.path,
  (path) => {
    const base = "/" + path.split("/")[1];
    currentTab.value = routeTabMap[base] || "dashboard";
  },
  { immediate: true },
);

function navigate(tab) {
  router.push(tabRouteMap[tab] || "/dashboard");
}
function onServiceChange() {
  refresh();
}
function refresh() {
  refreshKey.value++;
}

provide("sharedFilters", sharedFilters);
provide("refreshKey", refreshKey);
provide("loading", loading);
provide("isDark", isDark);

onMounted(async () => {
  try {
    const { data } = await api.get("/services");
    services.value = data || [];
  } catch {}
  setInterval(() => refreshKey.value++, 30_000);
});
</script>

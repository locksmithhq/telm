import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  { path: '/', redirect: '/dashboard' },
  {
    path: '/dashboard',
    component: () => import('@/module/dashboard/DashboardView.vue'),
  },
  {
    path: '/traces',
    component: () => import('@/module/traces/TracesView.vue'),
  },
  {
    path: '/traces/:traceId',
    component: () => import('@/module/traces/TraceDetailView.vue'),
  },
  {
    path: '/metrics',
    component: () => import('@/module/metrics/MetricsView.vue'),
  },
  {
    path: '/logs',
    component: () => import('@/module/logs/LogsView.vue'),
  },
  {
    path: '/dashboards',
    component: () => import('@/module/dashboards/DashboardsView.vue'),
  },
  {
    path: '/dashboards/:id',
    component: () => import('@/module/dashboards/DashboardView.vue'),
  },
]

export default createRouter({
  history: createWebHistory(),
  routes,
})

import { createRouter, createWebHistory } from 'vue-router'
import api from '@/plugins/axios.js'

const routes = [
  { path: '/', redirect: '/dashboard' },
  {
    path: '/login',
    component: () => import('@/module/auth/LoginView.vue'),
    meta: { public: true },
  },
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
  {
    path: '/storage',
    component: () => import('@/module/storage/StorageView.vue'),
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

// Verificado uma única vez por page load — evita chamada em toda navegação
let sessionVerified = false

router.beforeEach(async (to, from, next) => {
  const hasFlag = !!localStorage.getItem('telm-auth')

  if (to.meta.public) {
    if (!hasFlag) return next()
    if (sessionVerified) return next('/')
    // Tem flag mas ainda não verificou com o servidor nesta sessão
    try { await api.get('/auth/me'); sessionVerified = true } catch { localStorage.removeItem('telm-auth') }
    return localStorage.getItem('telm-auth') ? next('/') : next()
  }

  // Rota protegida
  if (!hasFlag) return next('/login')
  if (sessionVerified) return next()

  try {
    await api.get('/auth/me')
    sessionVerified = true
    next()
  } catch {
    localStorage.removeItem('telm-auth')
    next('/login')
  }
})

export default router

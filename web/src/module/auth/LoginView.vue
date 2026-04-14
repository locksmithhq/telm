<template>
  <div class="login-root">
    <canvas ref="canvasEl" class="bg-canvas" />

    <div class="login-layout">
      <!-- ══════════════ LEFT PANEL ══════════════ -->
      <div class="left-panel">

        <!-- Row 1: Brand -->
        <div class="brand-row">
          <div class="brand-icon-wrap">
            <div class="brand-ring" />
            <div class="brand-ring ring2" />
            <span class="brand-glyph">⟋</span>
          </div>
          <div>
            <div class="brand-name">telm</div>
            <div class="brand-sub">Observability Platform</div>
          </div>
          <div class="live-pill"><span class="live-dot" />LIVE</div>
        </div>

        <!-- Row 2: Metrics -->
        <div class="metrics-grid">
          <div class="metric-card" v-for="m in metrics" :key="m.label">
            <div class="mc-top">
              <span class="mc-label">{{ m.label }}</span>
              <span class="mc-badge" :style="`color:${m.color};background:${m.color}20`">{{ m.trend }}</span>
            </div>
            <div class="mc-value" :style="`color:${m.color}`">{{ m.display }}</div>
            <svg viewBox="0 0 80 22" preserveAspectRatio="none" class="mc-spark">
              <defs>
                <linearGradient :id="`mg${m.key}`" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" :stop-color="m.color" stop-opacity="0.3"/>
                  <stop offset="100%" :stop-color="m.color" stop-opacity="0"/>
                </linearGradient>
              </defs>
              <path :d="m.areaPath" :fill="`url(#mg${m.key})`"/>
              <path :d="m.linePath" fill="none" :stroke="m.color" stroke-width="1.2"/>
            </svg>
          </div>
        </div>

        <!-- Row 3: Main chart + right mini-column -->
        <div class="charts-row">
          <div class="main-chart-card">
            <div class="card-header">
              <span class="card-title">Ingestion throughput</span>
              <div class="legend-row">
                <span v-for="s in services" :key="s.name" class="legend-item">
                  <span class="legend-dot" :style="`background:${s.color}`"/>{{ s.name }}
                </span>
              </div>
            </div>
            <div class="chart-body">
              <svg viewBox="0 0 400 100" preserveAspectRatio="none" class="fill-svg">
                <defs>
                  <linearGradient v-for="s in services" :key="s.name" :id="`cg${s.name}`" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" :stop-color="s.color" stop-opacity="0.1"/>
                    <stop offset="100%" :stop-color="s.color" stop-opacity="0"/>
                  </linearGradient>
                </defs>
                <line v-for="y in [25,50,75]" :key="y" x1="0" :y1="y" x2="400" :y2="y" stroke="rgba(255,255,255,0.04)" stroke-width="1"/>
                <g v-for="s in services" :key="s.name">
                  <path :d="s.areaPath" :fill="`url(#cg${s.name})`"/>
                  <path :d="s.linePath" fill="none" :stroke="s.color" stroke-width="0.8"/>
                  <circle :cx="s.lastX" :cy="s.lastY" r="1.5" :fill="s.color"/>
                </g>
                <line :x1="scanX" y1="0" :x2="scanX" y2="100"
                  stroke="rgba(255,255,255,0.1)" stroke-width="1" stroke-dasharray="3 3"/>
              </svg>
            </div>
          </div>

          <!-- Right mini column -->
          <div class="right-mini-col">
            <!-- Heatmap -->
            <div class="mini-card heatmap-card">
              <div class="card-header"><span class="card-title">Activity (6h)</span></div>
              <div class="heatmap-grid">
                <div v-for="(v, i) in heatmap" :key="i" class="hm-cell"
                  :style="`background:${heatColor(v)};opacity:${0.35 + v * 0.65}`"/>
              </div>
              <div class="hm-labels">
                <span>-6h</span><span>-3h</span><span>now</span>
              </div>
            </div>

            <!-- P95 latency bars -->
            <div class="mini-card latency-card">
              <div class="card-header"><span class="card-title">P95 latency</span></div>
              <div class="lat-list">
                <div v-for="l in latencyBars" :key="l.svc" class="lat-row">
                  <span class="lat-svc" :style="`color:${l.color}`">{{ l.svc }}</span>
                  <div class="lat-track">
                    <div class="lat-bar" :style="`width:${l.pct}%;background:${l.color}`"/>
                  </div>
                  <span class="lat-val" :class="l.ms > 300 ? 'slow' : l.ms > 150 ? 'mid' : ''">{{ l.ms }}ms</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Row 4: Log stream + Waterfall -->
        <div class="bottom-row">
          <!-- Log stream -->
          <div class="log-card">
            <div class="card-header">
              <span class="card-title">Log stream</span>
              <div class="log-header-right">
                <span v-for="(c, sev) in sevCount" :key="sev" class="sev-count" :class="sev.toLowerCase()">
                  {{ sev }} {{ c }}
                </span>
                <span class="card-count">{{ logCount.toLocaleString() }} total</span>
              </div>
            </div>
            <div class="log-list" ref="logListEl">
              <div v-for="l in logLines" :key="l.id" class="log-line">
                <span class="log-ts">{{ l.ts }}</span>
                <span class="log-sev" :class="l.sev.toLowerCase()">{{ l.sev }}</span>
                <span class="log-svc" :style="`color:${l.color}`">{{ l.svc }}</span>
                <span class="log-msg">{{ l.msg }}</span>
              </div>
            </div>
          </div>

          <!-- Waterfall + error budget column -->
          <div class="right-bottom-col">
            <!-- Trace waterfall -->
            <div class="mini-card waterfall-card">
              <div class="card-header">
                <span class="card-title">Trace waterfall</span>
                <span class="card-count">{{ wfTotal }}ms</span>
              </div>
              <div class="wf-list">
                <div v-for="s in waterfall" :key="s.id" class="wf-row">
                  <span class="wf-name">{{ s.name }}</span>
                  <div class="wf-track">
                    <div class="wf-bar" :style="`left:${s.left}%;width:${s.width}%;background:${s.color}`"/>
                  </div>
                  <span class="wf-ms" :class="s.ms > 200 ? 'slow' : ''">{{ s.ms }}</span>
                </div>
              </div>
            </div>

            <!-- Error budget donuts -->
            <div class="mini-card budget-card">
              <div class="card-header"><span class="card-title">Error budget</span></div>
              <div class="budget-list">
                <div v-for="b in budget" :key="b.name" class="budget-row">
                  <svg viewBox="0 0 32 32" class="donut-svg">
                    <circle cx="16" cy="16" r="12" fill="none" stroke="rgba(255,255,255,0.06)" stroke-width="4"/>
                    <circle cx="16" cy="16" r="12" fill="none" :stroke="b.color" stroke-width="4"
                      stroke-linecap="round"
                      :stroke-dasharray="`${b.pct * 0.754} 75.4`"
                      stroke-dashoffset="18.85"
                      style="transition:stroke-dasharray .5s ease"/>
                  </svg>
                  <div class="budget-info">
                    <span class="budget-name">{{ b.name }}</span>
                    <span class="budget-pct" :style="`color:${b.color}`">{{ b.pct.toFixed(1) }}%</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

      </div>

      <!-- ══════════════ RIGHT PANEL ══════════════ -->
      <div class="right-panel">
        <div class="form-wrap">
          <div class="form-brand">
            <span class="fb-glyph">⟋</span>
            <span class="fb-name">telm</span>
          </div>

          <div class="form-title">Welcome back</div>
          <div class="form-subtitle">Sign in to your observability workspace</div>

          <v-alert v-if="error" type="error" variant="tonal" density="compact"
            class="mb-4 mt-1" style="font-size:12px">
            {{ error }}
          </v-alert>

          <div class="field-group">
            <label class="field-label">Email address</label>
            <v-text-field v-model="email" type="email" placeholder="admin@telm.local"
              hide-details autocomplete="email" prepend-inner-icon="mdi-email-outline"
              @keyup.enter="login"/>
          </div>

          <div class="field-group">
            <label class="field-label">Password</label>
            <v-text-field v-model="password" :type="showPassword ? 'text' : 'password'"
              placeholder="••••••••" hide-details autocomplete="current-password"
              prepend-inner-icon="mdi-lock-outline"
              :append-inner-icon="showPassword ? 'mdi-eye-off-outline' : 'mdi-eye-outline'"
              @click:append-inner="showPassword = !showPassword"
              @keyup.enter="login"/>
          </div>

          <v-btn color="primary" block size="large" :loading="loading"
            :disabled="!email || !password" class="mt-6 login-btn" @click="login">
            <v-icon start size="16">mdi-login-variant</v-icon>
            Sign in
          </v-btn>

          <div class="form-divider"><span>secured by</span></div>

          <div class="security-pills">
            <div class="sec-pill">
              <v-icon size="12" color="success">mdi-shield-check-outline</v-icon>Argon2id
            </div>
            <div class="sec-pill">
              <v-icon size="12" color="info">mdi-cookie-outline</v-icon>HttpOnly
            </div>
            <div class="sec-pill">
              <v-icon size="12" color="warning">mdi-key-outline</v-icon>JWT HS256
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import api from '@/plugins/axios.js'

const router    = useRouter()
const canvasEl  = ref(null)
const logListEl = ref(null)

const email        = ref('')
const password     = ref('')
const showPassword = ref(false)
const loading      = ref(false)
const error        = ref('')

// ── Path builder ──────────────────────────────────────────────────────────────
function buildPaths(data, W, H, pad = 3) {
  const min = Math.min(...data), max = Math.max(...data)
  const rng = max - min || 1
  const pts = data.map((v, i) => ({
    x: (i / (data.length - 1)) * W,
    y: H - pad - ((v - min) / rng) * (H - pad * 2),
  }))
  const line = pts.map((p, i) => `${i ? 'L' : 'M'}${p.x.toFixed(1)},${p.y.toFixed(1)}`).join(' ')
  const last = pts[pts.length - 1], first = pts[0]
  return { line, area: `${line} L${last.x},${H} L${first.x},${H} Z`, last }
}

function makeHist(base, noise, len = 18) {
  return Array.from({ length: len }, () => Math.max(0, base + (Math.random() - 0.5) * noise))
}

// ── Metrics ───────────────────────────────────────────────────────────────────
const metricDefs = [
  { key: 'm0', label: 'Spans/s',    color: '#6366f1', base: 1200, noise: 400, suffix: '',   trend: '↑ 12%' },
  { key: 'm1', label: 'Logs/s',     color: '#8b5cf6', base: 8200, noise: 1200, suffix: '',   trend: '↑ 4%'  },
  { key: 'm2', label: 'Error rate', color: '#ef4444', base: 2.4,  noise: 1.2, suffix: '%',  trend: '↓ 0.3%' },
  { key: 'm3', label: 'P95',        color: '#f59e0b', base: 142,  noise: 60,  suffix: 'ms', trend: '→ 0%'   },
]

const metricHists = ref(metricDefs.map(m => makeHist(m.base, m.noise)))

const metrics = computed(() =>
  metricDefs.map((m, i) => {
    const hist = metricHists.value[i]
    const { line, area } = buildPaths(hist, 80, 22)
    const v = hist[hist.length - 1]
    const display = m.suffix === '%' || m.suffix === 'ms'
      ? v.toFixed(1) + m.suffix
      : Math.round(v).toLocaleString()
    return { ...m, linePath: line, areaPath: area, display }
  })
)

// ── Multi-series chart ────────────────────────────────────────────────────────
const SVC_DEFS = [
  { name: 'api-gw',    color: '#6366f1' },
  { name: 'auth-svc',  color: '#8b5cf6' },
  { name: 'order-svc', color: '#10b981' },
  { name: 'db-proxy',  color: '#f59e0b' },
]

const svcHists = ref(SVC_DEFS.map(s => makeHist(300 + Math.random() * 400, 250, 24)))

const services = computed(() =>
  SVC_DEFS.map((s, i) => {
    const { line, area, last } = buildPaths(svcHists.value[i], 400, 100, 4)
    return { ...s, linePath: line, areaPath: area, lastX: last.x, lastY: last.y }
  })
)

const scanX = ref(0)

// ── Heatmap (8 rows × 24 cols) ────────────────────────────────────────────────
const heatmap = ref(Array.from({ length: 8 * 24 }, () => Math.random()))

function heatColor(v) {
  if (v > 0.88) return '#ef4444'
  if (v > 0.68) return '#f59e0b'
  if (v > 0.42) return '#6366f1'
  return '#1a2d4a'
}

// ── P95 latency bars ──────────────────────────────────────────────────────────
const LAT_SVCS = [
  { svc: 'api-gw',    color: '#6366f1', base: 120 },
  { svc: 'auth-svc',  color: '#8b5cf6', base: 85  },
  { svc: 'order-svc', color: '#10b981', base: 280  },
  { svc: 'db-proxy',  color: '#f59e0b', base: 155  },
  { svc: 'worker',    color: '#7dd3fc', base: 320  },
]

const latencyRaw = ref(LAT_SVCS.map(l => l.base + (Math.random() - 0.5) * 40))

const latencyBars = computed(() => {
  const max = Math.max(...latencyRaw.value)
  return LAT_SVCS.map((l, i) => ({
    ...l,
    ms: Math.round(latencyRaw.value[i]),
    pct: (latencyRaw.value[i] / max) * 100,
  }))
})

// ── Error budget donuts ───────────────────────────────────────────────────────
const budgetRaw = ref([
  { name: 'api-gw',   color: '#6366f1', pct: 98.7 },
  { name: 'auth',     color: '#8b5cf6', pct: 99.4 },
  { name: 'order',    color: '#10b981', pct: 96.2 },
])

const budget = computed(() => budgetRaw.value)

// ── Log stream ────────────────────────────────────────────────────────────────
const LOG_MSGS = [
  'request processed successfully',
  'cache miss — fetching from db',
  'connection pool exhausted',
  'retrying after timeout',
  'payment webhook received',
  'user session expired',
  'query took 842ms — slow',
  'rate limit applied',
  'batch job completed (1.2k rows)',
  'health check passed',
  'failed to parse payload',
  'upstream returned 503',
  'span exported to collector',
  'metric flush: 240 points',
  'trace sampled at 10%',
  'circuit breaker opened',
]

const LOG_SVCS  = ['api-gw', 'auth', 'order', 'db-proxy', 'worker', 'collector']
const SVC_COLS  = { 'api-gw': '#6366f1', auth: '#8b5cf6', order: '#10b981', 'db-proxy': '#f59e0b', worker: '#7dd3fc', collector: '#34d399' }
const SEV_LIST  = ['INFO', 'INFO', 'INFO', 'INFO', 'DEBUG', 'WARN', 'ERROR']

let logSeq = 0
const logLines  = ref(Array.from({ length: 20 }, () => makeLogLine()))
const logCount  = ref(284_192)
const sevCount  = ref({ ERROR: 0, WARN: 0 })

function makeLogLine() {
  const now = new Date()
  const ts  = `${pad(now.getHours())}:${pad(now.getMinutes())}:${pad(now.getSeconds())}.${String(now.getMilliseconds()).padStart(3,'0')}`
  const svc = LOG_SVCS[Math.floor(Math.random() * LOG_SVCS.length)]
  const sev = SEV_LIST[Math.floor(Math.random() * SEV_LIST.length)]
  return { id: logSeq++, ts, sev, svc, color: SVC_COLS[svc] || '#94a3b8', msg: LOG_MSGS[Math.floor(Math.random() * LOG_MSGS.length)] }
}

function pad(n) { return String(n).padStart(2, '0') }

// ── Trace waterfall ───────────────────────────────────────────────────────────
const WF_NAMES = ['api-gw', 'api-gw→auth', 'auth→db', 'api-gw→order', 'order→pay', 'order→notify', 'notify→queue']
const WF_COLS  = ['#6366f1', '#6366f1', '#8b5cf6', '#6366f1', '#10b981', '#f59e0b', '#7dd3fc']

const waterfall = ref(makeWaterfall())
const wfTotal   = computed(() => waterfall.value.reduce((s, r) => Math.max(s, Math.round((r.left + r.width) * 8)), 0))

function makeWaterfall() {
  let cursor = 0
  return WF_NAMES.map((name, i) => {
    const left  = i === 0 ? 0 : Math.min(cursor + Math.random() * 5, 60)
    const width = i === 0 ? 95 : 8 + Math.random() * 35
    const ms    = Math.round(width * 8)
    cursor = left + width * 0.5
    return { id: i, name, left: +left.toFixed(1), width: +Math.min(width, 95 - left).toFixed(1), ms, color: WF_COLS[i] }
  })
}

// ── Canvas ────────────────────────────────────────────────────────────────────
let animId = null, canvasCleanup = null

function initCanvas() {
  const canvas = canvasEl.value
  if (!canvas) return
  const ctx = canvas.getContext('2d')
  let W, H
  const nodes = []

  function resize() { W = canvas.width = canvas.offsetWidth; H = canvas.height = canvas.offsetHeight }
  resize()

  for (let i = 0; i < 90; i++) nodes.push({
    x: Math.random() * W, y: Math.random() * H,
    vx: (Math.random() - 0.5) * 0.3, vy: (Math.random() - 0.5) * 0.3,
    r: 0.8 + Math.random() * 1.6,
    hue: [239, 262, 160, 45][Math.floor(Math.random() * 4)],
  })

  window.addEventListener('resize', resize)

  function draw() {
    ctx.clearRect(0, 0, W, H)
    for (const n of nodes) {
      n.x += n.vx; n.y += n.vy
      if (n.x < 0 || n.x > W) n.vx *= -1
      if (n.y < 0 || n.y > H) n.vy *= -1
    }
    for (let i = 0; i < nodes.length; i++) {
      for (let j = i + 1; j < nodes.length; j++) {
        const dx = nodes[i].x - nodes[j].x, dy = nodes[i].y - nodes[j].y
        const d  = Math.sqrt(dx * dx + dy * dy)
        if (d < 120) {
          ctx.beginPath(); ctx.moveTo(nodes[i].x, nodes[i].y); ctx.lineTo(nodes[j].x, nodes[j].y)
          ctx.strokeStyle = `hsla(${nodes[i].hue},70%,65%,${0.09 * (1 - d / 120)})`
          ctx.lineWidth = 0.6; ctx.stroke()
        }
      }
    }
    for (const n of nodes) {
      ctx.beginPath(); ctx.arc(n.x, n.y, n.r, 0, Math.PI * 2)
      ctx.fillStyle = `hsla(${n.hue},70%,65%,0.5)`; ctx.fill()
    }
    animId = requestAnimationFrame(draw)
  }
  draw()
  canvasCleanup = () => window.removeEventListener('resize', resize)
}

// ── Tickers ───────────────────────────────────────────────────────────────────
let t1 = null, t2 = null, t3 = null

function startTickers() {
  // Fast: logs + scan
  t1 = setInterval(() => {
    const line = makeLogLine()
    logLines.value = [...logLines.value.slice(-30), line]
    logCount.value += Math.floor(Math.random() * 15 + 3)
    if (line.sev === 'ERROR') sevCount.value = { ...sevCount.value, ERROR: sevCount.value.ERROR + 1 }
    if (line.sev === 'WARN')  sevCount.value = { ...sevCount.value, WARN:  sevCount.value.WARN  + 1 }
    nextTick(() => { if (logListEl.value) logListEl.value.scrollTop = logListEl.value.scrollHeight })
  }, 700)

  // Medium: metrics, charts, heatmap, latency, budget
  t2 = setInterval(() => {
    metricHists.value = metricHists.value.map((hist, i) => {
      const m = metricDefs[i]
      const last = hist[hist.length - 1]
      return [...hist.slice(1), Math.max(0, last + (Math.random() - 0.48) * m.noise * 0.25)]
    })
    svcHists.value = svcHists.value.map(hist => {
      const last = hist[hist.length - 1]
      return [...hist.slice(1), Math.max(50, last + (Math.random() - 0.48) * 120)]
    })
    // heatmap: update a few cells
    const newHeat = [...heatmap.value]
    for (let k = 0; k < 4; k++) newHeat[Math.floor(Math.random() * newHeat.length)] = Math.random()
    heatmap.value = newHeat
    // latency drift
    latencyRaw.value = latencyRaw.value.map((v, i) =>
      Math.max(30, v + (Math.random() - 0.5) * 25)
    )
    // budget drift
    budgetRaw.value = budgetRaw.value.map(b => ({
      ...b,
      pct: Math.min(100, Math.max(90, b.pct + (Math.random() - 0.52) * 0.3)),
    }))
    // waterfall occasionally
    if (Math.random() < 0.25) waterfall.value = makeWaterfall()
  }, 1100)

  // Scan line
  let sx = 0
  t3 = setInterval(() => { sx = (sx + 4) % 400; scanX.value = sx }, 25)
}

onMounted(() => { initCanvas(); startTickers() })
onUnmounted(() => {
  cancelAnimationFrame(animId)
  clearInterval(t1); clearInterval(t2); clearInterval(t3)
  canvasCleanup?.()
})

// ── Login ─────────────────────────────────────────────────────────────────────
async function login() {
  if (!email.value || !password.value) return
  loading.value = true; error.value = ''
  try {
    await api.post('/auth/login', { email: email.value, password: password.value })
    localStorage.setItem('telm-auth', '1')
    router.push('/')
  } catch (e) {
    error.value = e.response?.data?.error || 'Login failed'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
/* ── Root ── */
.login-root {
  position: relative;
  width: 100vw;
  height: 100vh;
  overflow: hidden;
  background: #090e1a;
  display: flex;
}

.bg-canvas {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
  z-index: 0;
}

.login-layout {
  position: relative;
  z-index: 1;
  display: flex;
  width: 100%;
  height: 100%;
}

/* ════════════════════════════════
   LEFT PANEL  —  CSS Grid rows
════════════════════════════════ */
.left-panel {
  flex: 1;
  min-width: 0;
  display: grid;
  grid-template-rows: auto auto 180px 1.4fr;
  gap: 10px;
  padding: 16px 14px 16px 20px;
  border-right: 1px solid rgba(99,102,241,0.12);
  background: linear-gradient(160deg, rgba(99,102,241,0.04) 0%, transparent 50%);
  overflow: hidden;
}

/* ── Brand row ── */
.brand-row {
  display: flex;
  align-items: center;
  gap: 11px;
}

.brand-icon-wrap {
  position: relative;
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.brand-ring {
  position: absolute;
  inset: 0;
  border-radius: 50%;
  border: 1px solid rgba(99,102,241,0.35);
  animation: ring-pulse 2.4s ease-in-out infinite;
}
.ring2 { inset: -7px; border-color: rgba(99,102,241,0.12); animation-delay: .8s; }

.brand-glyph { font-size: 18px; color: #6366f1; font-weight: 700; position: relative; z-index: 1; }
.brand-name  { font-size: 18px; font-weight: 700; color: #e2e8f0; font-family: 'Courier New', monospace; letter-spacing: .06em; line-height: 1.1; }
.brand-sub   { font-size: 9px; color: #2d3748; letter-spacing: .12em; text-transform: uppercase; }

.live-pill {
  margin-left: auto;
  display: flex; align-items: center; gap: 5px;
  font-size: 8px; font-weight: 700; letter-spacing: .12em; color: #10b981;
  background: rgba(16,185,129,.1); border: 1px solid rgba(16,185,129,.22);
  border-radius: 20px; padding: 3px 9px;
}
.live-dot { width: 5px; height: 5px; border-radius: 50%; background: #10b981; animation: live-blink 1.2s ease-in-out infinite; }

/* ── Metrics ── */
.metrics-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 8px;
}

.metric-card {
  background: rgba(15,22,35,.8);
  border: 1px solid rgba(99,102,241,.1);
  border-radius: 8px;
  padding: 9px 11px 5px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.mc-top   { display: flex; align-items: center; justify-content: space-between; margin-bottom: 3px; }
.mc-label { font-size: 9px; color: #374151; text-transform: uppercase; letter-spacing: .1em; }
.mc-badge { font-size: 9px; font-weight: 600; padding: 1px 5px; border-radius: 4px; }
.mc-value { font-size: 17px; font-weight: 700; font-family: 'Courier New', monospace; line-height: 1; margin-bottom: 5px; }
.mc-spark { width: 100%; height: 20px; display: block; }

/* ── Charts row ── */
.charts-row {
  display: flex;
  gap: 10px;
  min-height: 0;
  overflow: hidden;
}

.main-chart-card {
  flex: 1;
  background: rgba(15,22,35,.8);
  border: 1px solid rgba(99,102,241,.1);
  border-radius: 8px;
  padding: 10px 12px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.chart-body { flex: 1; min-height: 0; }

.fill-svg { width: 100%; height: 100%; display: block; }

.right-mini-col {
  width: 170px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.mini-card {
  background: rgba(15,22,35,.8);
  border: 1px solid rgba(99,102,241,.1);
  border-radius: 8px;
  padding: 10px 11px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.heatmap-card { flex: 1; }
.latency-card { flex: 1; }

/* Heatmap */
.heatmap-grid {
  flex: 1;
  display: grid;
  grid-template-columns: repeat(24, 1fr);
  grid-template-rows: repeat(8, 1fr);
  gap: 1.5px;
  margin-bottom: 4px;
}

.hm-cell { border-radius: 1px; transition: background .5s ease, opacity .5s ease; }

.hm-labels {
  display: flex;
  justify-content: space-between;
  font-size: 8px;
  color: #2d3748;
}

/* Latency bars */
.lat-list { flex: 1; display: flex; flex-direction: column; justify-content: space-around; }

.lat-row { display: flex; align-items: center; gap: 5px; }
.lat-svc { width: 52px; font-size: 9px; font-weight: 600; font-family: 'Courier New', monospace; flex-shrink: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.lat-track { flex: 1; height: 5px; background: rgba(255,255,255,.05); border-radius: 3px; overflow: hidden; }
.lat-bar { height: 100%; border-radius: 3px; opacity: .8; transition: width .6s ease; }
.lat-val { width: 32px; text-align: right; font-size: 9px; color: #4b5563; font-family: 'Courier New', monospace; flex-shrink: 0; }
.lat-val.mid  { color: #f59e0b; }
.lat-val.slow { color: #ef4444; }

/* Chart header */
.card-header {
  display: flex; align-items: center; justify-content: space-between;
  margin-bottom: 7px; flex-shrink: 0;
}
.card-title { font-size: 9px; color: #4b5563; text-transform: uppercase; letter-spacing: .1em; }
.card-count { font-size: 9px; color: #2d3748; }

.legend-row   { display: flex; gap: 10px; }
.legend-item  { display: flex; align-items: center; gap: 4px; font-size: 9px; color: #4b5563; }
.legend-dot   { width: 6px; height: 6px; border-radius: 50%; }

/* ── Bottom row ── */
.bottom-row {
  display: flex;
  gap: 10px;
  min-height: 0;
  overflow: hidden;
}

.log-card {
  flex: 1;
  background: rgba(15,22,35,.8);
  border: 1px solid rgba(99,102,241,.1);
  border-radius: 8px;
  padding: 10px 12px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.log-header-right { display: flex; align-items: center; gap: 8px; }

.sev-count {
  font-size: 9px; font-weight: 700;
  padding: 1px 6px; border-radius: 4px;
}
.sev-count.error { color: #ef4444; background: rgba(239,68,68,.12); }
.sev-count.warn  { color: #f59e0b; background: rgba(245,158,11,.12); }

.log-list {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  scrollbar-width: none;
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.log-list::-webkit-scrollbar { display: none; }

.log-line {
  display: flex; align-items: baseline; gap: 6px;
  font-size: 10px; font-family: 'Courier New', monospace;
  padding: 2px 3px; border-radius: 3px; white-space: nowrap;
  overflow: hidden; animation: log-in .15s ease;
}
.log-line:hover { background: rgba(255,255,255,.03); }

.log-ts  { color: #1f2937; flex-shrink: 0; font-size: 9px; }
.log-sev { flex-shrink: 0; font-weight: 700; font-size: 8px; padding: 1px 4px; border-radius: 3px; letter-spacing: .05em; }
.log-sev.info  { color: #6366f1; background: rgba(99,102,241,.14); }
.log-sev.warn  { color: #f59e0b; background: rgba(245,158,11,.14); }
.log-sev.error { color: #ef4444; background: rgba(239,68,68,.14); }
.log-sev.debug { color: #475569; background: rgba(71,85,105,.14); }
.log-svc { flex-shrink: 0; font-size: 9px; font-weight: 600; }
.log-msg { color: #374151; font-size: 10px; overflow: hidden; text-overflow: ellipsis; }

/* Right bottom column */
.right-bottom-col {
  width: 210px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.waterfall-card { flex: 1.4; }
.budget-card    { flex: 1; }

/* Waterfall */
.wf-list { flex: 1; display: flex; flex-direction: column; justify-content: space-around; }

.wf-row   { display: flex; align-items: center; gap: 5px; }
.wf-name  { width: 72px; font-size: 9px; color: #4b5563; font-family: 'Courier New', monospace; flex-shrink: 0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.wf-track { flex: 1; height: 7px; background: rgba(255,255,255,.04); border-radius: 2px; position: relative; overflow: hidden; }
.wf-bar   { position: absolute; top: 0; height: 100%; border-radius: 2px; opacity: .75; transition: left .5s ease, width .5s ease; }
.wf-ms    { width: 26px; text-align: right; font-size: 9px; color: #374151; font-family: 'Courier New', monospace; flex-shrink: 0; }
.wf-ms.slow { color: #f59e0b; }

/* Error budget donuts */
.budget-list { flex: 1; display: flex; flex-direction: column; justify-content: space-around; }

.budget-row  { display: flex; align-items: center; gap: 8px; }
.donut-svg   { width: 28px; height: 28px; flex-shrink: 0; transform: rotate(-90deg); }
.budget-info { display: flex; flex-direction: column; gap: 1px; }
.budget-name { font-size: 9px; color: #4b5563; font-family: 'Courier New', monospace; }
.budget-pct  { font-size: 12px; font-weight: 700; font-family: 'Courier New', monospace; line-height: 1; }

/* ════════════════════════════════
   RIGHT PANEL
════════════════════════════════ */
.right-panel {
  width: 560px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(9,14,26,.7);
  backdrop-filter: blur(18px);
  border-left: 1px solid rgba(99,102,241,.08);
  padding: 32px 28px;
}

.form-wrap { width: 100%; max-width: 380px; margin: 0 auto; }

.form-brand { display: flex; align-items: baseline; gap: 5px; margin-bottom: 22px; }
.fb-glyph   { font-size: 20px; color: #6366f1; font-weight: 700; }
.fb-name    { font-size: 18px; font-weight: 700; color: #cbd5e1; font-family: 'Courier New', monospace; letter-spacing: .06em; }

.form-title    { font-size: 21px; font-weight: 700; color: #f1f5f9; margin-bottom: 3px; }
.form-subtitle { font-size: 11px; color: #2d3748; margin-bottom: 22px; }

.field-group { margin-bottom: 13px; }
.field-label { display: block; font-size: 10px; color: #4b5563; text-transform: uppercase; letter-spacing: .08em; margin-bottom: 5px; }

.login-btn { font-weight: 600; letter-spacing: .03em; }

.form-divider {
  display: flex; align-items: center; gap: 10px;
  margin: 20px 0 12px; font-size: 9px; color: #1f2937;
  text-transform: uppercase; letter-spacing: .1em;
}
.form-divider::before, .form-divider::after {
  content: ''; flex: 1; height: 1px; background: rgba(255,255,255,.04);
}

.security-pills { display: flex; gap: 5px; justify-content: center; flex-wrap: wrap; }
.sec-pill {
  display: flex; align-items: center; gap: 4px;
  font-size: 9px; color: #2d3748;
  background: rgba(255,255,255,.03); border: 1px solid rgba(255,255,255,.05);
  border-radius: 20px; padding: 3px 8px;
}

/* ── Animations ── */
@keyframes ring-pulse {
  0%, 100% { transform: scale(1); opacity: .6; }
  50%       { transform: scale(1.35); opacity: .12; }
}
@keyframes live-blink {
  0%, 100% { opacity: 1; }
  50%       { opacity: .25; }
}
@keyframes log-in {
  from { opacity: 0; transform: translateX(-4px); }
  to   { opacity: 1; transform: translateX(0); }
}

@media (max-width: 860px) {
  .left-panel  { display: none; }
  .right-panel { width: 100%; }
}
</style>

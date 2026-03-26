import api from '@/plugins/axios'

// ─── Source definitions ────────────────────────────────────────────────────
export const SOURCES = {
  'metrics.series': {
    label: 'Metric time-series',
    hint: 'metrics.series name=<metric> [service=<svc>] [range=1h]',
    endpoint: '/metrics/series',
    vizOptions: ['line', 'area', 'bar', 'scatter', 'gauge', 'stat'],
    defaultViz: 'line',
    returns: 'timeseries',
  },
  'stats.throughput': {
    label: 'Request throughput',
    hint: 'stats.throughput [service=<svc>] [range=1h]',
    endpoint: '/stats/throughput',
    vizOptions: ['line', 'area', 'bar', 'scatter', 'gauge', 'stat'],
    defaultViz: 'line',
    returns: 'throughput',
  },
  'stats.errors': {
    label: 'Error rate',
    hint: 'stats.errors [service=<svc>] [range=1h]',
    endpoint: '/stats/errors',
    vizOptions: ['line', 'area', 'bar', 'scatter', 'gauge', 'stat'],
    defaultViz: 'line',
    returns: 'errors',
  },
  'stats.latency': {
    label: 'Latency (P50/P95/P99)',
    hint: 'stats.latency [service=<svc>] [range=1h]',
    endpoint: '/stats/latency',
    vizOptions: ['line', 'area', 'scatter', 'radar'],
    defaultViz: 'line',
    returns: 'latency',
  },
  'stats.top-ops': {
    label: 'Top operations',
    hint: 'stats.top-ops [service=<svc>] [range=1h]',
    endpoint: '/stats/top-ops',
    vizOptions: ['bar', 'treemap', 'pie', 'funnel', 'table'],
    defaultViz: 'bar',
    returns: 'top-ops',
  },
  'stats.severity': {
    label: 'Log severity distribution',
    hint: 'stats.severity [service=<svc>] [range=1h]',
    endpoint: '/stats/severity',
    vizOptions: ['pie', 'bar', 'treemap', 'funnel', 'table'],
    defaultViz: 'pie',
    returns: 'severity',
  },
  'stats.service-map': {
    label: 'Service dependency graph',
    hint: 'stats.service-map [range=1h]',
    endpoint: '/stats/service-map',
    vizOptions: ['graph', 'sankey', 'table'],
    defaultViz: 'graph',
    returns: 'service-map',
  },
  'stats.services-health': {
    label: 'Services health overview',
    hint: 'stats.services-health [range=1h]',
    endpoint: '/stats/services-health',
    vizOptions: ['table', 'bar', 'radar', 'stat'],
    defaultViz: 'table',
    returns: 'services-health',
  },
  'stats.resources': {
    label: 'Resource metrics (CPU / memory / GC)',
    hint: 'stats.resources [service=<svc>] [range=1h]',
    endpoint: '/stats/resources/all',
    vizOptions: ['table', 'line', 'area', 'heatmap'],
    defaultViz: 'table',
    returns: 'resources',
  },
  traces: {
    label: 'Distributed traces',
    hint: 'traces [service=<svc>] [operation=<op>] [status=ok|error] [range=1h] [limit=50]',
    endpoint: '/traces',
    vizOptions: ['table', 'scatter', 'bar'],
    defaultViz: 'table',
    returns: 'traces',
  },
  logs: {
    label: 'Log entries',
    hint: 'logs [service=<svc>] [severity=ERROR] [search=<text>] [range=1h] [limit=100]',
    endpoint: '/logs',
    vizOptions: ['table', 'bar', 'pie'],
    defaultViz: 'table',
    returns: 'logs',
  },
}

// ─── Parser ────────────────────────────────────────────────────────────────
function tokenize(str) {
  const tokens = []
  let i = 0
  while (i < str.length) {
    while (i < str.length && /\s/.test(str[i])) i++
    if (i >= str.length) break
    if (str[i] === '"' || str[i] === "'") {
      const q = str[i++]
      let val = ''
      while (i < str.length && str[i] !== q) val += str[i++]
      if (i < str.length) i++
      tokens.push(val)
    } else {
      let token = ''
      while (i < str.length && !/\s/.test(str[i])) token += str[i++]
      tokens.push(token)
    }
  }
  return tokens
}

export function parse(queryStr) {
  const str = (queryStr || '').trim()
  if (!str) return { error: 'Empty query' }

  const tokens = tokenize(str)
  if (!tokens.length) return { error: 'Empty query' }

  const source = tokens[0]
  const def = SOURCES[source]
  if (!def) {
    return {
      error: `Unknown source "${source}"\nAvailable: ${Object.keys(SOURCES).join(', ')}`,
    }
  }

  const params = {}
  for (let j = 1; j < tokens.length; j++) {
    const eq = tokens[j].indexOf('=')
    if (eq > 0) {
      params[tokens[j].slice(0, eq)] = tokens[j].slice(eq + 1).replace(/^["']|["']$/g, '')
    }
  }

  const viz = params.viz || def.defaultViz
  delete params.viz

  return { source, params, viz, def }
}

// ─── Executor ──────────────────────────────────────────────────────────────
export function rangeToParams(range) {
  const to = new Date()
  const mins = { '1h': 60, '6h': 360, '24h': 1440, '7d': 10080 }
  const from = new Date(to - (mins[range] || 60) * 60_000)
  return { from: from.toISOString(), to: to.toISOString() }
}

export function bucketSecs(range) {
  return { '1h': 60, '6h': 300, '24h': 1800, '7d': 3600 }[range] || 60
}

export function fmtLabel(iso, range) {
  const d = new Date(iso)
  if (range === '7d') return d.toLocaleDateString('pt-BR', { month: 'short', day: 'numeric' })
  return d.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' })
}

export async function execute(parsed) {
  if (!parsed || parsed.error) throw new Error(parsed?.error || 'Invalid query')

  const { params, viz, def } = parsed
  const { range = '1h', limit, service, name, operation, status, severity, search } = params

  const { from, to } = rangeToParams(range)
  const qp = { from, to }
  if (service) qp.service = service
  if (name) qp.name = name
  if (operation) qp.operation = operation
  if (status) qp.status = status
  if (severity) qp.severity = severity
  if (search) qp.search = search
  if (limit) qp.limit = parseInt(limit) || 50

  const { data } = await api.get(def.endpoint, { params: qp })
  return { data: data || [], returns: def.returns, viz, range }
}

// ─── Autocomplete helpers ──────────────────────────────────────────────────
export const RANGE_VALUES = ['1h', '6h', '24h', '7d']
export const ALL_PARAMS = ['service', 'name', 'range', 'operation', 'status', 'severity', 'search', 'limit', 'viz']

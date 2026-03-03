package postgres

import (
	"context"
	"fmt"
	"time"
)

type StatsFilter struct {
	Service string
	From    time.Time
	To      time.Time
}

type ThroughputPoint struct {
	Time  time.Time `db:"time"  json:"time"`
	Count int64     `db:"count" json:"count"`
}

type ErrorPoint struct {
	Time   time.Time `db:"time"   json:"time"`
	Total  int64     `db:"total"  json:"total"`
	Errors int64     `db:"errors" json:"errors"`
}

type LatencyPoint struct {
	Time time.Time `db:"time" json:"time"`
	P50  float64   `db:"p50"  json:"p50"`
	P95  float64   `db:"p95"  json:"p95"`
	P99  float64   `db:"p99"  json:"p99"`
}

type TopOp struct {
	Operation string  `db:"operation" json:"operation"`
	Service   string  `db:"service"   json:"service"`
	Count     int64   `db:"count"     json:"count"`
	AvgMs     float64 `db:"avg_ms"    json:"avg_ms"`
	Errors    int64   `db:"errors"    json:"errors"`
}

type SeverityPoint struct {
	Severity string `db:"severity" json:"severity"`
	Count    int64  `db:"count"    json:"count"`
}

// QueryThroughput retorna contagem de traces (root spans) por bucket de tempo.
func (c *Client) QueryThroughput(ctx context.Context, f StatsFilter) ([]ThroughputPoint, error) {
	setStatsDefaults(&f)
	q := fmt.Sprintf(`
		SELECT
			date_trunc('%s', start_time) AS time,
			COUNT(*) AS count
		FROM traces
		WHERE parent_span_id IS NULL
		  AND ($1 = '' OR service_name = $1)
		  AND start_time >= $2
		  AND start_time <= $3
		GROUP BY time
		ORDER BY time`, intervalFor(f.From, f.To))

	var result []ThroughputPoint
	err := c.db.SelectContext(ctx, &result, q, f.Service, f.From, f.To)
	if result == nil {
		result = []ThroughputPoint{}
	}
	return result, err
}

// QueryErrors retorna total e erros por bucket para calcular error rate.
func (c *Client) QueryErrors(ctx context.Context, f StatsFilter) ([]ErrorPoint, error) {
	setStatsDefaults(&f)
	q := fmt.Sprintf(`
		SELECT
			date_trunc('%s', start_time) AS time,
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE status_code = 2) AS errors
		FROM traces
		WHERE parent_span_id IS NULL
		  AND ($1 = '' OR service_name = $1)
		  AND start_time >= $2
		  AND start_time <= $3
		GROUP BY time
		ORDER BY time`, intervalFor(f.From, f.To))

	var result []ErrorPoint
	err := c.db.SelectContext(ctx, &result, q, f.Service, f.From, f.To)
	if result == nil {
		result = []ErrorPoint{}
	}
	return result, err
}

// QueryLatency retorna percentis P50/P95/P99 por bucket (em ms).
func (c *Client) QueryLatency(ctx context.Context, f StatsFilter) ([]LatencyPoint, error) {
	setStatsDefaults(&f)
	q := fmt.Sprintf(`
		SELECT
			date_trunc('%s', start_time) AS time,
			COALESCE(PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY duration_ns), 0) / 1e6 AS p50,
			COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_ns), 0) / 1e6 AS p95,
			COALESCE(PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY duration_ns), 0) / 1e6 AS p99
		FROM traces
		WHERE parent_span_id IS NULL
		  AND ($1 = '' OR service_name = $1)
		  AND start_time >= $2
		  AND start_time <= $3
		GROUP BY time
		ORDER BY time`, intervalFor(f.From, f.To))

	var result []LatencyPoint
	err := c.db.SelectContext(ctx, &result, q, f.Service, f.From, f.To)
	if result == nil {
		result = []LatencyPoint{}
	}
	return result, err
}

// QueryTopOps retorna as 10 operações mais frequentes com avg latency e erros.
func (c *Client) QueryTopOps(ctx context.Context, f StatsFilter) ([]TopOp, error) {
	setStatsDefaults(&f)
	const q = `
		SELECT
			operation_name AS operation,
			service_name   AS service,
			COUNT(*)       AS count,
			COALESCE(AVG(duration_ns), 0) / 1e6 AS avg_ms,
			COUNT(*) FILTER (WHERE status_code = 2) AS errors
		FROM traces
		WHERE parent_span_id IS NULL
		  AND ($1 = '' OR service_name = $1)
		  AND start_time >= $2
		  AND start_time <= $3
		GROUP BY operation_name, service_name
		ORDER BY count DESC
		LIMIT 10`

	var result []TopOp
	err := c.db.SelectContext(ctx, &result, q, f.Service, f.From, f.To)
	if result == nil {
		result = []TopOp{}
	}
	return result, err
}

// QuerySeverityDist retorna distribuição de severidade dos logs.
func (c *Client) QuerySeverityDist(ctx context.Context, f StatsFilter) ([]SeverityPoint, error) {
	setStatsDefaults(&f)
	const q = `
		SELECT
			UPPER(COALESCE(NULLIF(severity_text, ''), 'UNKNOWN')) AS severity,
			COUNT(*) AS count
		FROM logs
		WHERE ($1 = '' OR service_name = $1)
		  AND timestamp >= $2
		  AND timestamp <= $3
		GROUP BY severity
		ORDER BY count DESC`

	var result []SeverityPoint
	err := c.db.SelectContext(ctx, &result, q, f.Service, f.From, f.To)
	if result == nil {
		result = []SeverityPoint{}
	}
	return result, err
}

// ServiceHealth holds health metrics per service.
type ServiceHealth struct {
	ServiceName string  `db:"service_name" json:"service_name"`
	Total       int64   `db:"total"        json:"total"`
	Errors      int64   `db:"errors"       json:"errors"`
	ReqS        float64 `db:"req_s"        json:"req_s"`
	P95Ms       float64 `db:"p95_ms"       json:"p95_ms"`
}

// ServiceMapEdge represents a call between two services.
type ServiceMapEdge struct {
	Source string  `db:"source"  json:"source"`
	Target string  `db:"target"  json:"target"`
	Calls  int64   `db:"calls"   json:"calls"`
	Errors int64   `db:"errors"  json:"errors"`
	AvgMs  float64 `db:"avg_ms"  json:"avg_ms"`
}

// ServiceMapNode is derived from edge data.
type ServiceMapNode struct {
	ID        string  `json:"id"`
	Label     string  `json:"label"`
	ReqS      float64 `json:"req_s"`
	ErrorRate float64 `json:"error_rate"`
}

// ServiceMapResponse is the full graph.
type ServiceMapResponse struct {
	Nodes []ServiceMapNode `json:"nodes"`
	Edges []ServiceMapEdge `json:"edges"`
}

// ResourcePoint holds a single aggregated metric value at a point in time.
type ResourcePoint struct {
	Time  time.Time `db:"time"  json:"time"`
	Value float64   `db:"value" json:"value"`
}

// QueryServiceHealth returns per-service health summary for the given filter.
func (c *Client) QueryServiceHealth(ctx context.Context, f StatsFilter) ([]ServiceHealth, error) {
	setStatsDefaults(&f)
	const q = `
		SELECT
			service_name,
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE status_code = 2) AS errors,
			ROUND(COUNT(*) / EXTRACT(EPOCH FROM ($3::timestamptz - $2::timestamptz))::numeric, 2) AS req_s,
			COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_ns), 0) / 1e6 AS p95_ms
		FROM traces
		WHERE parent_span_id IS NULL
		  AND ($1 = '' OR service_name = $1)
		  AND start_time BETWEEN $2 AND $3
		GROUP BY service_name
		ORDER BY total DESC`

	var result []ServiceHealth
	err := c.db.SelectContext(ctx, &result, q, f.Service, f.From, f.To)
	if result == nil {
		result = []ServiceHealth{}
	}
	return result, err
}

// QueryServiceMap returns the service dependency graph for the given time range.
func (c *Client) QueryServiceMap(ctx context.Context, f StatsFilter) (ServiceMapResponse, error) {
	setStatsDefaults(&f)
	const q = `
		SELECT
			service_name AS source,
			(attributes::jsonb)->>'peer.service' AS target,
			COUNT(*) AS calls,
			COUNT(*) FILTER (WHERE status_code = 2) AS errors,
			COALESCE(AVG(duration_ns), 0) / 1e6 AS avg_ms
		FROM traces
		WHERE kind = 3
		  AND attributes IS NOT NULL AND attributes <> ''
		  AND (attributes::jsonb)->>'peer.service' IS NOT NULL
		  AND start_time BETWEEN $1 AND $2
		GROUP BY service_name, (attributes::jsonb)->>'peer.service'`

	empty := ServiceMapResponse{Nodes: []ServiceMapNode{}, Edges: []ServiceMapEdge{}}

	var edges []ServiceMapEdge
	if err := c.db.SelectContext(ctx, &edges, q, f.From, f.To); err != nil {
		return empty, err
	}
	if edges == nil {
		return empty, nil
	}

	totalSecs := f.To.Sub(f.From).Seconds()
	if totalSecs <= 0 {
		totalSecs = 1
	}

	type nodeAgg struct {
		calls  int64
		errors int64
	}
	nodeMap := map[string]*nodeAgg{}
	for _, e := range edges {
		if nodeMap[e.Source] == nil {
			nodeMap[e.Source] = &nodeAgg{}
		}
		if nodeMap[e.Target] == nil {
			nodeMap[e.Target] = &nodeAgg{}
		}
		nd := nodeMap[e.Source]
		nd.calls += e.Calls
		nd.errors += e.Errors
	}

	nodes := make([]ServiceMapNode, 0, len(nodeMap))
	for id, nd := range nodeMap {
		errRate := 0.0
		if nd.calls > 0 {
			errRate = float64(nd.errors) / float64(nd.calls) * 100
		}
		nodes = append(nodes, ServiceMapNode{
			ID:        id,
			Label:     id,
			ReqS:      float64(nd.calls) / totalSecs,
			ErrorRate: errRate,
		})
	}

	return ServiceMapResponse{Nodes: nodes, Edges: edges}, nil
}

// QueryResourceSeries returns time-bucketed series for the 4 resource metrics of a service.
func (c *Client) QueryResourceSeries(ctx context.Context, service string, f StatsFilter) (map[string][]ResourcePoint, error) {
	setStatsDefaults(&f)
	q := fmt.Sprintf(`
		SELECT
			metric_name,
			date_trunc('%s', timestamp) AS time,
			COALESCE(
				AVG(value_double),
				AVG(value_int::float8),
				CASE WHEN SUM(metric_count) > 0 THEN SUM(metric_sum) / SUM(metric_count) ELSE NULL END
			) AS value
		FROM metrics
		WHERE service_name = $1
		  AND metric_name IN ('process.cpu.usage', 'process.memory.bytes', 'runtime.goroutines', 'runtime.gc.pause_ms')
		  AND timestamp BETWEEN $2 AND $3
		GROUP BY metric_name, time
		ORDER BY time`, intervalFor(f.From, f.To))

	type row struct {
		MetricName string    `db:"metric_name"`
		Time       time.Time `db:"time"`
		Value      float64   `db:"value"`
	}

	var rows []row
	if err := c.db.SelectContext(ctx, &rows, q, service, f.From, f.To); err != nil {
		return nil, err
	}

	result := map[string][]ResourcePoint{
		"process.cpu.usage":    {},
		"process.memory.bytes": {},
		"runtime.goroutines":   {},
		"runtime.gc.pause_ms":  {},
	}
	for _, r := range rows {
		result[r.MetricName] = append(result[r.MetricName], ResourcePoint{Time: r.Time, Value: r.Value})
	}
	return result, nil
}

// QueryAllResourceSeries retorna séries temporais das métricas de recursos
// agrupadas por serviço. Quando f.Service != "" filtra por serviço.
func (c *Client) QueryAllResourceSeries(ctx context.Context, f StatsFilter) (map[string]map[string][]ResourcePoint, error) {
	setStatsDefaults(&f)
	q := fmt.Sprintf(`
		SELECT
			service_name,
			metric_name,
			date_trunc('%s', timestamp) AS time,
			COALESCE(
				AVG(value_double),
				AVG(value_int::float8),
				CASE WHEN SUM(metric_count) > 0 THEN SUM(metric_sum) / SUM(metric_count) ELSE NULL END
			) AS value
		FROM metrics
		WHERE ($1 = '' OR service_name = $1)
		  AND metric_name IN (
			'process.cpu.usage', 'process.memory.bytes',
			'runtime.goroutines', 'runtime.gc.pause_ms',
			'process.io.read_bytes', 'process.io.write_bytes'
		  )
		  AND timestamp BETWEEN $2 AND $3
		GROUP BY service_name, metric_name, time
		ORDER BY service_name, time`, intervalFor(f.From, f.To))

	type row struct {
		ServiceName string    `db:"service_name"`
		MetricName  string    `db:"metric_name"`
		Time        time.Time `db:"time"`
		Value       float64   `db:"value"`
	}

	var rows []row
	if err := c.db.SelectContext(ctx, &rows, q, f.Service, f.From, f.To); err != nil {
		return nil, err
	}

	allMetrics := []string{
		"process.cpu.usage", "process.memory.bytes",
		"runtime.goroutines", "runtime.gc.pause_ms",
		"process.io.read_bytes", "process.io.write_bytes",
	}
	result := map[string]map[string][]ResourcePoint{}
	for _, r := range rows {
		if result[r.ServiceName] == nil {
			m := map[string][]ResourcePoint{}
			for _, name := range allMetrics {
				m[name] = []ResourcePoint{}
			}
			result[r.ServiceName] = m
		}
		result[r.ServiceName][r.MetricName] = append(
			result[r.ServiceName][r.MetricName],
			ResourcePoint{Time: r.Time, Value: r.Value},
		)
	}
	return result, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func setStatsDefaults(f *StatsFilter) {
	if f.From.IsZero() {
		f.From = time.Now().Add(-1 * time.Hour)
	}
	if f.To.IsZero() {
		f.To = time.Now()
	}
}

// intervalFor escolhe o granularidade do date_trunc com base na janela de tempo.
// Retorna apenas strings fixas (não vem de input do usuário).
func intervalFor(from, to time.Time) string {
	d := to.Sub(from)
	switch {
	case d <= 3*time.Hour:
		return "minute"
	case d <= 7*24*time.Hour:
		return "hour"
	default:
		return "day"
	}
}

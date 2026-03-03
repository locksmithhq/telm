package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ── Filter types ──────────────────────────────────────────────────────────────

type TraceFilter struct {
	Service   string
	Operation string
	From      time.Time
	To        time.Time
	Limit     int
}

type MetricFilter struct {
	Service string
	Name    string
	Type    string
	From    time.Time
	To      time.Time
	Limit   int
}

type MetricSeriesFilter struct {
	Name    string
	Service string
	From    time.Time
	To      time.Time
}

type LogFilter struct {
	Service  string
	Severity string
	Search   string
	From     time.Time
	To       time.Time
	Limit    int
}

// ── Query result types (com json tags para a API) ─────────────────────────────

type QuerySpan struct {
	TraceID      string    `db:"trace_id"       json:"trace_id"`
	SpanID       string    `db:"span_id"        json:"span_id"`
	ParentSpanID *string   `db:"parent_span_id" json:"parent_span_id"`
	Operation    string    `db:"operation_name" json:"operation"`
	Service      string    `db:"service_name"   json:"service"`
	StartTime    time.Time `db:"start_time"     json:"start_time"`
	EndTime      time.Time `db:"end_time"       json:"end_time"`
	DurationNs   int64     `db:"duration_ns"    json:"duration_ns"`
	StatusCode   int32     `db:"status_code"    json:"status_code"`
	Kind         int32     `db:"kind"           json:"kind"`
	SpanCount    int64     `db:"span_count"     json:"span_count"`

	// stored as TEXT; re-emitidos como JSON raw para o frontend
	attrsRaw  string `db:"attributes"`
	eventsRaw string `db:"events"`

	Attributes json.RawMessage `db:"-" json:"attributes"`
	Events     json.RawMessage `db:"-" json:"events"`
	Services   []string        `db:"-" json:"services"`
}

type QueryMetric struct {
	MetricName  string    `db:"metric_name"  json:"name"`
	MetricType  string    `db:"metric_type"  json:"type"`
	Service     string    `db:"service_name" json:"service"`
	Timestamp   time.Time `db:"timestamp"    json:"timestamp"`
	ValueDouble *float64  `db:"value_double" json:"value_double"`
	ValueInt    *int64    `db:"value_int"    json:"value_int"`
	Count       *int64    `db:"metric_count" json:"count"`
	Sum         *float64  `db:"metric_sum"   json:"sum"`
	Unit        *string   `db:"unit"         json:"unit"`

	attrsRaw string          `db:"attributes"`
	Attrs    json.RawMessage `db:"-" json:"attributes"`
}

type MetricCatalogEntry struct {
	Name      string    `db:"metric_name"  json:"name"`
	Type      string    `db:"metric_type"  json:"type"`
	Service   string    `db:"service_name" json:"service"`
	Unit      *string   `db:"unit"         json:"unit"`
	ValueDouble *float64 `db:"value_double" json:"value_double"`
	ValueInt    *int64   `db:"value_int"    json:"value_int"`
	Count     *int64    `db:"metric_count" json:"count"`
	Sum       *float64  `db:"metric_sum"   json:"sum"`
	Timestamp time.Time `db:"timestamp"    json:"timestamp"`
}

type MetricSeriesPoint struct {
	Time       time.Time `db:"bucket"      json:"time"`
	AvgValue   *float64  `db:"avg_value"   json:"avg_value"`
	TotalCount *int64    `db:"total_count" json:"total_count"`
	TotalSum   *float64  `db:"total_sum"   json:"total_sum"`
}

type QueryLog struct {
	Timestamp      time.Time `db:"timestamp"      json:"timestamp"`
	SeverityNumber int32     `db:"severity_number" json:"severity_number"`
	SeverityText   string    `db:"severity_text"  json:"severity"`
	Body           string    `db:"body"           json:"body"`
	Service        string    `db:"service_name"   json:"service"`
	TraceID        *string   `db:"trace_id"       json:"trace_id"`
	SpanID         *string   `db:"span_id"        json:"span_id"`

	attrsRaw string          `db:"attributes"`
	Attrs    json.RawMessage `db:"-" json:"attributes"`
}

// ── Query methods ─────────────────────────────────────────────────────────────

// QuerySpans retorna apenas root spans (parent_span_id IS NULL) para a listagem,
// incluindo a lista de serviços distintos envolvidos em cada trace.
func (c *Client) QuerySpans(ctx context.Context, f TraceFilter) ([]QuerySpan, error) {
	setTraceDefaults(&f)

	const q = `
		WITH roots AS (
			SELECT trace_id, span_id, parent_span_id, operation_name, service_name,
			       start_time, end_time, duration_ns, status_code, kind,
			       COALESCE(attributes, '{}') AS attributes, COALESCE(events, '[]') AS events
			FROM traces
			WHERE parent_span_id IS NULL
			  AND ($1 = '' OR service_name = $1)
			  AND ($2 = '' OR operation_name ILIKE '%' || $2 || '%')
			  AND start_time >= $3
			  AND start_time <= $4
			ORDER BY start_time DESC
			LIMIT $5
		)
		SELECT r.trace_id, r.span_id, r.parent_span_id, r.operation_name, r.service_name,
		       r.start_time, r.end_time, r.duration_ns, r.status_code, r.kind,
		       COALESCE(r.attributes, '{}') AS attributes, COALESCE(r.events, '[]') AS events,
		       (SELECT COUNT(*) FROM traces WHERE trace_id = r.trace_id) AS span_count,
		       (
		           SELECT STRING_AGG(svc, ',' ORDER BY svc)
		           FROM (SELECT DISTINCT service_name AS svc FROM traces WHERE trace_id = r.trace_id) sub
		       ) AS services_list
		FROM roots r
		ORDER BY r.start_time DESC`

	type rootSpanScan struct {
		TraceID      string    `db:"trace_id"`
		SpanID       string    `db:"span_id"`
		ParentSpanID *string   `db:"parent_span_id"`
		Operation    string    `db:"operation_name"`
		Service      string    `db:"service_name"`
		StartTime    time.Time `db:"start_time"`
		EndTime      time.Time `db:"end_time"`
		DurationNs   int64     `db:"duration_ns"`
		StatusCode   int32     `db:"status_code"`
		Kind         int32     `db:"kind"`
		SpanCount    int64     `db:"span_count"`
		AttrsRaw     string    `db:"attributes"`
		EventsRaw    string    `db:"events"`
		ServicesList string    `db:"services_list"`
	}

	rows, err := c.db.QueryxContext(ctx, q, f.Service, f.Operation, f.From, f.To, f.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []QuerySpan
	for rows.Next() {
		var r rootSpanScan
		if err := rows.StructScan(&r); err != nil {
			return nil, err
		}
		services := []string{}
		if r.ServicesList != "" {
			services = strings.Split(r.ServicesList, ",")
		}
		result = append(result, QuerySpan{
			TraceID:      r.TraceID,
			SpanID:       r.SpanID,
			ParentSpanID: r.ParentSpanID,
			Operation:    r.Operation,
			Service:      r.Service,
			StartTime:    r.StartTime,
			EndTime:      r.EndTime,
			DurationNs:   r.DurationNs,
			StatusCode:   r.StatusCode,
			Kind:         r.Kind,
			SpanCount:    r.SpanCount,
			Attributes:   toRawJSON(r.AttrsRaw),
			Events:       toRawJSON(r.EventsRaw),
			Services:     services,
		})
	}
	if result == nil {
		result = []QuerySpan{}
	}
	return result, rows.Err()
}

// QuerySpansByTraceID retorna todos os spans de um trace para o waterfall.
func (c *Client) QuerySpansByTraceID(ctx context.Context, traceID string) ([]QuerySpan, error) {
	const q = `
		SELECT trace_id, span_id, parent_span_id, operation_name, service_name,
		       start_time, end_time, duration_ns, status_code, kind,
		       COALESCE(attributes, '{}') AS attributes, COALESCE(events, '[]') AS events
		FROM traces
		WHERE trace_id = $1
		ORDER BY start_time ASC`

	rows, err := c.db.QueryxContext(ctx, q, traceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSpans(rows)
}

func (c *Client) QueryMetrics(ctx context.Context, f MetricFilter) ([]QueryMetric, error) {
	setMetricDefaults(&f)

	const q = `
		SELECT metric_name, metric_type, service_name, timestamp,
		       value_double, value_int, metric_count, metric_sum, unit, COALESCE(attributes, '{}') AS attributes
		FROM metrics
		WHERE ($1 = '' OR service_name = $1)
		  AND ($2 = '' OR metric_name ILIKE '%' || $2 || '%')
		  AND ($3 = '' OR metric_type = $3)
		  AND timestamp >= $4
		  AND timestamp <= $5
		ORDER BY timestamp DESC
		LIMIT $6`

	type row struct {
		QueryMetric
		AttrsRaw string `db:"attributes"`
	}

	var rows []row
	if err := c.db.SelectContext(ctx, &rows, q, f.Service, f.Name, f.Type, f.From, f.To, f.Limit); err != nil {
		return nil, err
	}

	result := make([]QueryMetric, len(rows))
	for i, r := range rows {
		result[i] = r.QueryMetric
		result[i].Attrs = toRawJSON(r.AttrsRaw)
	}
	return result, nil
}

func (c *Client) QueryMetricCatalog(ctx context.Context, service string) ([]MetricCatalogEntry, error) {
	const q = `
		SELECT DISTINCT ON (metric_name, service_name)
		       metric_name, metric_type, service_name,
		       unit, value_double, value_int, metric_count, metric_sum, timestamp
		FROM metrics
		WHERE ($1 = '' OR service_name = $1)
		ORDER BY metric_name, service_name, timestamp DESC`

	var rows []MetricCatalogEntry
	if err := c.db.SelectContext(ctx, &rows, q, service); err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []MetricCatalogEntry{}
	}
	return rows, nil
}

func (c *Client) QueryMetricSeries(ctx context.Context, f MetricSeriesFilter) ([]MetricSeriesPoint, error) {
	if f.From.IsZero() {
		f.From = time.Now().Add(-1 * time.Hour)
	}
	if f.To.IsZero() {
		f.To = time.Now()
	}

	dur := f.To.Sub(f.From)
	bucket := "minute"
	switch {
	case dur > 7*24*time.Hour:
		bucket = "hour"
	case dur > 24*time.Hour:
		bucket = "30 minutes"
	case dur > 6*time.Hour:
		bucket = "5 minutes"
	}

	q := fmt.Sprintf(`
		SELECT date_trunc('%s', timestamp) AS bucket,
		       AVG(COALESCE(value_double, value_int::float8)) AS avg_value,
		       SUM(metric_count) AS total_count,
		       SUM(metric_sum)   AS total_sum
		FROM metrics
		WHERE metric_name = $1
		  AND ($2 = '' OR service_name = $2)
		  AND timestamp >= $3
		  AND timestamp <= $4
		GROUP BY bucket
		ORDER BY bucket ASC`, bucket)

	var rows []MetricSeriesPoint
	if err := c.db.SelectContext(ctx, &rows, q, f.Name, f.Service, f.From, f.To); err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []MetricSeriesPoint{}
	}
	return rows, nil
}

func (c *Client) QueryLogs(ctx context.Context, f LogFilter) ([]QueryLog, error) {
	setLogDefaults(&f)

	const q = `
		SELECT timestamp, severity_number, severity_text, body, service_name,
		       trace_id, span_id, COALESCE(attributes, '{}') AS attributes
		FROM logs
		WHERE ($1 = '' OR service_name = $1)
		  AND ($2 = '' OR UPPER(severity_text) = UPPER($2))
		  AND ($3 = '' OR body ILIKE '%' || $3 || '%')
		  AND timestamp >= $4
		  AND timestamp <= $5
		ORDER BY timestamp DESC
		LIMIT $6`

	type row struct {
		QueryLog
		AttrsRaw string `db:"attributes"`
	}

	var rows []row
	if err := c.db.SelectContext(ctx, &rows, q, f.Service, f.Severity, f.Search, f.From, f.To, f.Limit); err != nil {
		return nil, err
	}

	result := make([]QueryLog, len(rows))
	for i, r := range rows {
		result[i] = r.QueryLog
		result[i].Attrs = toRawJSON(r.AttrsRaw)
	}
	return result, nil
}

func (c *Client) QueryLogsByTraceID(ctx context.Context, traceID string) ([]QueryLog, error) {
	const q = `
		SELECT timestamp, severity_number, severity_text, body, service_name,
		       trace_id, span_id, COALESCE(attributes, '{}') AS attributes
		FROM logs
		WHERE trace_id = $1
		ORDER BY timestamp ASC`

	type row struct {
		QueryLog
		AttrsRaw string `db:"attributes"`
	}

	var rows []row
	if err := c.db.SelectContext(ctx, &rows, q, traceID); err != nil {
		return nil, err
	}

	result := make([]QueryLog, len(rows))
	for i, r := range rows {
		result[i] = r.QueryLog
		result[i].Attrs = toRawJSON(r.AttrsRaw)
	}
	if result == nil {
		result = []QueryLog{}
	}
	return result, nil
}

func (c *Client) ListServices(ctx context.Context) ([]string, error) {
	const q = `
		SELECT DISTINCT service_name FROM (
			SELECT service_name FROM traces
			UNION ALL SELECT service_name FROM metrics
			UNION ALL SELECT service_name FROM logs
		) s ORDER BY service_name`

	var result []string
	err := c.db.SelectContext(ctx, &result, q)
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = []string{}
	}
	return result, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func scanSpans(rows interface{ StructScan(any) error; Next() bool; Err() error }) ([]QuerySpan, error) {
	type spanScan struct {
		TraceID      string    `db:"trace_id"`
		SpanID       string    `db:"span_id"`
		ParentSpanID *string   `db:"parent_span_id"`
		Operation    string    `db:"operation_name"`
		Service      string    `db:"service_name"`
		StartTime    time.Time `db:"start_time"`
		EndTime      time.Time `db:"end_time"`
		DurationNs   int64     `db:"duration_ns"`
		StatusCode   int32     `db:"status_code"`
		Kind         int32     `db:"kind"`
		AttrsRaw     string    `db:"attributes"`
		EventsRaw    string    `db:"events"`
	}

	var result []QuerySpan
	for rows.Next() {
		var r spanScan
		if err := rows.StructScan(&r); err != nil {
			return nil, err
		}
		result = append(result, QuerySpan{
			TraceID:      r.TraceID,
			SpanID:       r.SpanID,
			ParentSpanID: r.ParentSpanID,
			Operation:    r.Operation,
			Service:      r.Service,
			StartTime:    r.StartTime,
			EndTime:      r.EndTime,
			DurationNs:   r.DurationNs,
			StatusCode:   r.StatusCode,
			Kind:         r.Kind,
			Attributes:   toRawJSON(r.AttrsRaw),
			Events:       toRawJSON(r.EventsRaw),
		})
	}
	if result == nil {
		result = []QuerySpan{}
	}
	return result, rows.Err()
}

func toRawJSON(s string) json.RawMessage {
	if s == "" || s == "null" {
		return json.RawMessage("{}")
	}
	return json.RawMessage(s)
}

func setTraceDefaults(f *TraceFilter) {
	if f.From.IsZero() {
		f.From = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	if f.To.IsZero() {
		f.To = time.Now().Add(24 * time.Hour)
	}
	if f.Limit <= 0 {
		f.Limit = 100
	}
}

func setMetricDefaults(f *MetricFilter) {
	if f.From.IsZero() {
		f.From = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	if f.To.IsZero() {
		f.To = time.Now().Add(24 * time.Hour)
	}
	if f.Limit <= 0 {
		f.Limit = 100
	}
}

func setLogDefaults(f *LogFilter) {
	if f.From.IsZero() {
		f.From = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	if f.To.IsZero() {
		f.To = time.Now().Add(24 * time.Hour)
	}
	if f.Limit <= 0 {
		f.Limit = 100
	}
}

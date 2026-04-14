package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// ── Filter types ──────────────────────────────────────────────────────────────

type AttrFilter struct {
	Key    string
	Value  string
	Invert bool
}

type TraceFilter struct {
	Service       string
	Operation     string
	From          time.Time
	To            time.Time
	Limit         int
	TraceID       string
	StatusCodes   []int
	Kinds         []int
	DurationMinMs int64
	DurationMaxMs int64
	MinSpanCount  int64
	Attributes    []AttrFilter
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
	Service     string
	Severity    string
	Search      string
	Operation   string
	HasError    bool
	HasTrace    bool
	AttrFilters []AttrFilter
	From        time.Time
	To          time.Time
	Limit       int
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
	Name        string    `db:"metric_name"  json:"name"`
	Type        string    `db:"metric_type"  json:"type"`
	Service     string    `db:"service_name" json:"service"`
	Unit        *string   `db:"unit"         json:"unit"`
	ValueDouble *float64  `db:"value_double" json:"value_double"`
	ValueInt    *int64    `db:"value_int"    json:"value_int"`
	Count       *int64    `db:"metric_count" json:"count"`
	Sum         *float64  `db:"metric_sum"   json:"sum"`
	Timestamp   time.Time `db:"timestamp"    json:"timestamp"`
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
// Usa CTEs encadeadas com JOINs — sem subqueries correlacionadas.
func (c *Client) QuerySpans(ctx context.Context, f TraceFilter) ([]QuerySpan, error) {
	setTraceDefaults(&f)

	var ctes      strings.Builder // "attr_N AS (...),\n" por filtro de atributo
	var joins      strings.Builder // JOIN/LEFT JOIN attr_N por filtro
	var rootWhere  strings.Builder // filtros escalares do CTE roots
	var args       []any

	add := func(v any) int {
		args = append(args, v)
		return len(args)
	}

	// ── filtros escalares ────────────────────────────────────────────────────
	opN   := add(f.Operation)
	fromN := add(f.From)
	toN   := add(f.To)
	fmt.Fprintf(&rootWhere, "        AND ($%d = '' OR t.operation_name ILIKE '%%' || $%d || '%%')\n", opN, opN)
	fmt.Fprintf(&rootWhere, "        AND t.start_time >= $%d\n", fromN)
	fmt.Fprintf(&rootWhere, "        AND t.start_time <= $%d\n", toN)

	// Filtro de serviço: busca trace_ids onde QUALQUER span pertence ao serviço,
	// não apenas o root span — permite encontrar traces distribuídos por serviço interno.
	if f.Service != "" {
		svcN := add(f.Service)
		fmt.Fprintf(&ctes,
			"    svc_traces AS (\n"+
				"        SELECT DISTINCT trace_id FROM traces\n"+
				"        WHERE service_name = $%d\n"+
				"          AND start_time >= $%d AND start_time <= $%d\n"+
				"    ),\n",
			svcN, fromN, toN)
		joins.WriteString("        JOIN svc_traces ON t.trace_id = svc_traces.trace_id\n")
	}

	if f.TraceID != "" {
		fmt.Fprintf(&rootWhere, "        AND t.trace_id = $%d\n", add(f.TraceID))
	}
	if len(f.StatusCodes) > 0 {
		rootWhere.WriteString("        AND t.status_code IN (")
		for i, v := range f.StatusCodes {
			if i > 0 {
				rootWhere.WriteString(", ")
			}
			fmt.Fprintf(&rootWhere, "$%d", add(v))
		}
		rootWhere.WriteString(")\n")
	}
	if len(f.Kinds) > 0 {
		rootWhere.WriteString("        AND t.kind IN (")
		for i, v := range f.Kinds {
			if i > 0 {
				rootWhere.WriteString(", ")
			}
			fmt.Fprintf(&rootWhere, "$%d", add(v))
		}
		rootWhere.WriteString(")\n")
	}
	if f.DurationMinMs > 0 {
		fmt.Fprintf(&rootWhere, "        AND t.duration_ns >= $%d\n", add(f.DurationMinMs*1_000_000))
	}
	if f.DurationMaxMs > 0 {
		fmt.Fprintf(&rootWhere, "        AND t.duration_ns <= $%d\n", add(f.DurationMaxMs*1_000_000))
	}

	// ── filtros de atributo: 1 CTE por filtro + JOIN no roots ───────────────
	attrIdx := 0
	for _, attr := range f.Attributes {
		if attr.Key == "" {
			continue
		}
		cteName := fmt.Sprintf("attr_%d", attrIdx)
		attrIdx++

		kn := add(attr.Key)
		if attr.Value != "" {
			vn := add(attr.Value)
			fmt.Fprintf(&ctes,
				"    %s AS (\n"+
					"        SELECT DISTINCT trace_id FROM traces\n"+
					"        WHERE attributes::jsonb ->> $%d = $%d\n"+
					"    ),\n",
				cteName, kn, vn)
		} else {
			fmt.Fprintf(&ctes,
				"    %s AS (\n"+
					"        SELECT DISTINCT trace_id FROM traces\n"+
					"        WHERE attributes::jsonb ? $%d\n"+
					"    ),\n",
				cteName, kn)
		}

		if attr.Invert {
			fmt.Fprintf(&joins, "        LEFT JOIN %s ON t.trace_id = %s.trace_id\n", cteName, cteName)
			fmt.Fprintf(&rootWhere, "        AND %s.trace_id IS NULL\n", cteName)
		} else {
			fmt.Fprintf(&joins, "        JOIN %s ON t.trace_id = %s.trace_id\n", cteName, cteName)
		}
	}

	limitN   := add(f.Limit)
	minSpanN := add(f.MinSpanCount)

	q := fmt.Sprintf(`
WITH
%s    roots AS (
        SELECT t.trace_id, t.span_id, t.parent_span_id, t.operation_name, t.service_name,
               t.start_time, t.end_time, t.duration_ns, t.status_code, t.kind,
               COALESCE(t.attributes, '{}') AS attributes,
               COALESCE(t.events,     '[]') AS events
        FROM traces t
%s        WHERE t.parent_span_id IS NULL
%s        ORDER BY t.start_time DESC
        LIMIT $%d
    ),
    trace_stats AS (
        SELECT t.trace_id,
               COUNT(*) AS span_count,
               STRING_AGG(DISTINCT t.service_name, ',' ORDER BY t.service_name) AS services_list
        FROM traces t
        JOIN roots r ON t.trace_id = r.trace_id
        GROUP BY t.trace_id
    )
SELECT r.trace_id, r.span_id, r.parent_span_id, r.operation_name, r.service_name,
       r.start_time, r.end_time, r.duration_ns, r.status_code, r.kind,
       r.attributes, r.events,
       ts.span_count,
       COALESCE(ts.services_list, r.service_name) AS services_list
FROM roots r
JOIN trace_stats ts ON r.trace_id = ts.trace_id
WHERE ($%d = 0 OR ts.span_count >= $%d)
ORDER BY r.start_time DESC`,
		ctes.String(), joins.String(), rootWhere.String(),
		limitN, minSpanN, minSpanN)

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

	rows, err := c.db.QueryxContext(ctx, q, args...)
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
	// Lê metrics_current — uma linha por partição (metric_name + service + label set).
	// prev_* já está pré-calculado no write path (upsertCurrentMetric).
	// Big-O: O(D) onde D = partições distintas. Sem window functions, sem subqueries.
	const q = `
		SELECT
		    metric_name, metric_type, service_name, unit,
		    MAX(timestamp) AS timestamp,
		    CASE
		        WHEN MAX(aggregation_temporality) IS NULL
		            THEN AVG(value_double)
		        WHEN MAX(aggregation_temporality) = 2
		            THEN NULLIF(SUM(CASE
		                WHEN value_double IS NOT NULL AND prev_value_double IS NOT NULL
		                THEN GREATEST(value_double - prev_value_double, 0) ELSE 0 END), 0)
		        ELSE NULLIF(SUM(COALESCE(value_double, 0)), 0)
		    END::float8 AS value_double,
		    CASE
		        WHEN MAX(aggregation_temporality) IS NULL
		            THEN (AVG(value_int::float8))::bigint
		        WHEN MAX(aggregation_temporality) = 2
		            THEN NULLIF(SUM(CASE
		                WHEN value_int IS NOT NULL AND prev_value_int IS NOT NULL
		                THEN GREATEST(value_int - prev_value_int, 0) ELSE 0 END), 0)::bigint
		        ELSE NULLIF(SUM(COALESCE(value_int, 0)), 0)::bigint
		    END AS value_int,
		    CASE
		        WHEN MAX(aggregation_temporality) = 2
		            THEN NULLIF(SUM(CASE
		                WHEN metric_count IS NOT NULL AND prev_metric_count IS NOT NULL
		                THEN GREATEST(metric_count - prev_metric_count, 0) ELSE 0 END), 0)
		        ELSE NULLIF(SUM(COALESCE(metric_count, 0)), 0)
		    END AS metric_count,
		    CASE
		        WHEN MAX(aggregation_temporality) = 2
		            THEN NULLIF(SUM(CASE
		                WHEN metric_sum IS NOT NULL AND prev_metric_sum IS NOT NULL
		                THEN GREATEST(metric_sum - prev_metric_sum, 0) ELSE 0 END), 0)
		        ELSE NULLIF(SUM(COALESCE(metric_sum, 0)), 0)
		    END AS metric_sum
		FROM metrics_current
		WHERE ($1 = '' OR service_name = $1)
		GROUP BY metric_name, metric_type, service_name, unit`

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
	var bucketExpr string
	switch {
	case dur >= 7*24*time.Hour:
		bucketExpr = "date_trunc('hour', timestamp)"
	case dur >= 24*time.Hour:
		bucketExpr = "to_timestamp(floor(extract(epoch from timestamp) / 1800) * 1800)"
	case dur >= 6*time.Hour:
		bucketExpr = "to_timestamp(floor(extract(epoch from timestamp) / 300) * 300)"
	default:
		bucketExpr = "date_trunc('minute', timestamp)"
	}

	q := fmt.Sprintf(`
		SELECT %s AS bucket,
		       CASE WHEN MAX(aggregation_temporality) IS NOT NULL THEN NULL
		            ELSE AVG(COALESCE(value_double, value_int::float8))
		       END AS avg_value,
		       CASE WHEN MAX(aggregation_temporality) = 2
		            THEN GREATEST(MAX(metric_count) - MIN(metric_count), 0)
		            ELSE SUM(metric_count)
		       END AS total_count,
		       CASE WHEN MAX(aggregation_temporality) = 2
		            THEN GREATEST(
		              MAX(COALESCE(metric_sum, value_double, value_int::float8)) -
		              MIN(COALESCE(metric_sum, value_double, value_int::float8)), 0)
		            ELSE SUM(COALESCE(metric_sum, value_double, value_int::float8))
		       END AS total_sum
		FROM metrics
		WHERE metric_name = $1
		  AND ($2 = '' OR service_name = $2)
		  AND timestamp >= $3
		  AND timestamp <= $4
		GROUP BY bucket
		ORDER BY bucket ASC`, bucketExpr)

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

	var where strings.Builder
	var args []any

	add := func(v any) int {
		args = append(args, v)
		return len(args)
	}

	if f.Service != "" {
		where.WriteString(fmt.Sprintf(" AND service_name = $%d", add(f.Service)))
	}
	if f.Severity != "" {
		where.WriteString(fmt.Sprintf(" AND UPPER(severity_text) = UPPER($%d)", add(f.Severity)))
	}
	if f.Search != "" {
		where.WriteString(fmt.Sprintf(
			" AND to_tsvector('simple', COALESCE(body, '')) @@ plainto_tsquery('simple', $%d)",
			add(f.Search),
		))
	}
	if f.Operation != "" {
		where.WriteString(fmt.Sprintf(" AND attributes->>'operation' ILIKE '%' || $%d || '%%'", add(f.Operation)))
	}
	if f.HasError {
		where.WriteString(fmt.Sprintf(" AND (severity_text ILIKE '%%ERROR%%' OR severity_text ILIKE '%%FATAL%%')"))
	}
	if f.HasTrace {
		where.WriteString(" AND trace_id IS NOT NULL AND trace_id != ''")
	}
	for _, af := range f.AttrFilters {
		if af.Key == "" {
			continue
		}
		key := add(af.Key)
		if af.Invert {
			if af.Value != "" {
				where.WriteString(fmt.Sprintf(" AND NOT (attributes->>$%d = $%d)", key, add(af.Value)))
			} else {
				where.WriteString(fmt.Sprintf(" AND (attributes->>$%d IS NULL OR attributes->>$%d = 'null')", key, key))
			}
		} else {
			if af.Value != "" {
				where.WriteString(fmt.Sprintf(" AND attributes->>$%d = $%d", key, add(af.Value)))
			} else {
				where.WriteString(fmt.Sprintf(" AND attributes->>$%d IS NOT NULL AND attributes->>$%d != 'null'", key, key))
			}
		}
	}
	where.WriteString(fmt.Sprintf(" AND timestamp >= $%d", add(f.From)))
	where.WriteString(fmt.Sprintf(" AND timestamp <= $%d", add(f.To)))

	q := fmt.Sprintf(`
		SELECT timestamp, severity_number, severity_text, body, service_name,
		       trace_id, span_id, COALESCE(attributes, '{}') AS attributes
		FROM logs
		WHERE 1=1%s
		ORDER BY timestamp DESC
		LIMIT $%d`, where.String(), add(f.Limit))

	type row struct {
		QueryLog
		AttrsRaw string `db:"attributes"`
	}

	var rows []row
	if err := c.db.SelectContext(ctx, &rows, q, args...); err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []row{}
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
	seen := make(map[string]struct{})
	for _, tbl := range []string{"traces", "metrics", "logs"} {
		var svcs []string
		q := "SELECT DISTINCT service_name FROM " + tbl + " WHERE service_name != '' ORDER BY service_name"
		if err := c.db.SelectContext(ctx, &svcs, q); err != nil {
			return nil, err
		}
		for _, s := range svcs {
			seen[s] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for s := range seen {
		result = append(result, s)
	}
	sort.Strings(result)
	return result, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func scanSpans(rows interface {
	StructScan(any) error
	Next() bool
	Err() error
}) ([]QuerySpan, error) {
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

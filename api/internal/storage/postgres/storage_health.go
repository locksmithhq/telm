package postgres

import (
	"context"
	"fmt"
	"time"
)

// StorageStats is a point-in-time snapshot of storage utilization per signal table.
type StorageStats struct {
	DBSizeBytes     int64      `db:"db_size_bytes"     json:"db_size_bytes"`
	LogSizeBytes    int64      `db:"log_size_bytes"    json:"log_size_bytes"`
	TraceSizeBytes  int64      `db:"trace_size_bytes"  json:"trace_size_bytes"`
	MetricSizeBytes int64      `db:"metric_size_bytes" json:"metric_size_bytes"`
	LogRows         int64      `db:"log_rows"          json:"log_rows"`
	TraceRows       int64      `db:"trace_rows"        json:"trace_rows"`
	MetricRows      int64      `db:"metric_rows"       json:"metric_rows"`
	OldestLog       *time.Time `db:"oldest_log"        json:"oldest_log"`
	OldestTrace     *time.Time `db:"oldest_trace"      json:"oldest_trace"`
	OldestMetric    *time.Time `db:"oldest_metric"     json:"oldest_metric"`
	// Populated from cleanup constants — not stored in DB.
	LogRetentionDays    int `json:"log_retention_days"`
	TraceRetentionDays  int `json:"trace_retention_days"`
	MetricRetentionDays int `json:"metric_retention_days"`
}

// StorageGrowthPoint represents ingestion volume for one time bucket.
type StorageGrowthPoint struct {
	Time       time.Time `db:"time"        json:"time"`
	LogRows    int64     `db:"log_rows"    json:"log_rows"`
	TraceRows  int64     `db:"trace_rows"  json:"trace_rows"`
	MetricRows int64     `db:"metric_rows" json:"metric_rows"`
}

// QueryStorageStats returns the current storage snapshot using pg catalog functions.
// Row counts come from pg_stat_user_tables.n_live_tup (autovacuum-maintained estimate).
func (c *Client) QueryStorageStats(ctx context.Context) (StorageStats, error) {
	const q = `
		SELECT
			pg_database_size(current_database())                                                AS db_size_bytes,
			pg_total_relation_size('logs')                                                      AS log_size_bytes,
			pg_total_relation_size('traces')                                                    AS trace_size_bytes,
			pg_total_relation_size('metrics')                                                   AS metric_size_bytes,
			COALESCE((SELECT n_live_tup FROM pg_stat_user_tables WHERE relname = 'logs'),    0) AS log_rows,
			COALESCE((SELECT n_live_tup FROM pg_stat_user_tables WHERE relname = 'traces'),  0) AS trace_rows,
			COALESCE((SELECT n_live_tup FROM pg_stat_user_tables WHERE relname = 'metrics'), 0) AS metric_rows,
			(SELECT MIN(timestamp)  FROM logs)    AS oldest_log,
			(SELECT MIN(start_time) FROM traces)  AS oldest_trace,
			(SELECT MIN(timestamp)  FROM metrics) AS oldest_metric`

	var s StorageStats
	if err := c.db.GetContext(ctx, &s, q); err != nil {
		return s, err
	}
	s.LogRetentionDays = LogRetentionDays
	s.TraceRetentionDays = TraceRetentionDays
	s.MetricRetentionDays = MetricRetentionDays
	return s, nil
}

// QueryStorageGrowth returns ingestion volume (row count) per time bucket for the given range.
// Uses created_at which is indexed on all three signal tables.
func (c *Client) QueryStorageGrowth(ctx context.Context, from, to time.Time) ([]StorageGrowthPoint, error) {
	interval := intervalFor(from, to)
	q := fmt.Sprintf(`
		WITH
		  lc AS (
		    SELECT date_trunc('%s', created_at) AS t, COUNT(*) AS n
		    FROM logs
		    WHERE created_at BETWEEN $1 AND $2
		    GROUP BY t
		  ),
		  tc AS (
		    SELECT date_trunc('%s', created_at) AS t, COUNT(*) AS n
		    FROM traces
		    WHERE created_at BETWEEN $1 AND $2
		    GROUP BY t
		  ),
		  mc AS (
		    SELECT date_trunc('%s', created_at) AS t, COUNT(*) AS n
		    FROM metrics
		    WHERE created_at BETWEEN $1 AND $2
		    GROUP BY t
		  ),
		  times AS (
		    SELECT t FROM lc
		    UNION SELECT t FROM tc
		    UNION SELECT t FROM mc
		  )
		SELECT
		    times.t            AS time,
		    COALESCE(lc.n, 0) AS log_rows,
		    COALESCE(tc.n, 0) AS trace_rows,
		    COALESCE(mc.n, 0) AS metric_rows
		FROM times
		LEFT JOIN lc ON lc.t = times.t
		LEFT JOIN tc ON tc.t = times.t
		LEFT JOIN mc ON mc.t = times.t
		ORDER BY time`,
		interval, interval, interval)

	var result []StorageGrowthPoint
	if err := c.db.SelectContext(ctx, &result, q, from, to); err != nil {
		return nil, err
	}
	if result == nil {
		result = []StorageGrowthPoint{}
	}
	return result, nil
}

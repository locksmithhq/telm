package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/locksmithhq/telm/internal/storage"
)

type metricRow struct {
	MetricName             string    `db:"metric_name"`
	MetricType             string    `db:"metric_type"`
	ServiceName            string    `db:"service_name"`
	Timestamp              time.Time `db:"timestamp"`
	ValueDouble            *float64  `db:"value_double"`
	ValueInt               *int64    `db:"value_int"`
	MetricCount            *int64    `db:"metric_count"`
	MetricSum              *float64  `db:"metric_sum"`
	Attributes             string    `db:"attributes"`
	Unit                   *string   `db:"unit"`
	IsMonotonic            *bool     `db:"is_monotonic"`
	AggregationTemporality *int32    `db:"aggregation_temporality"`
	BucketCounts           string    `db:"bucket_counts"`
	ExplicitBounds         string    `db:"explicit_bounds"`
}

const insertMetric = `
INSERT INTO metrics (
	metric_name, metric_type, service_name, timestamp,
	value_double, value_int, metric_count, metric_sum,
	attributes, unit,
	is_monotonic, aggregation_temporality, bucket_counts, explicit_bounds
) VALUES (
	:metric_name, :metric_type, :service_name, :timestamp,
	:value_double, :value_int, :metric_count, :metric_sum,
	:attributes, :unit,
	:is_monotonic, :aggregation_temporality, :bucket_counts, :explicit_bounds
)
ON CONFLICT (metric_name, service_name, timestamp, md5(coalesce(attributes, '{}')))
DO UPDATE SET
	value_double            = EXCLUDED.value_double,
	value_int               = EXCLUDED.value_int,
	metric_count            = EXCLUDED.metric_count,
	metric_sum              = EXCLUDED.metric_sum,
	bucket_counts           = EXCLUDED.bucket_counts,
	explicit_bounds         = EXCLUDED.explicit_bounds`

func (c *Client) SaveMetrics(ctx context.Context, points []storage.MetricPoint) error {
	if len(points) == 0 {
		return nil
	}

	tx, err := c.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	for _, p := range points {
		row := toMetricRow(p)
		if _, err := tx.NamedExecContext(ctx, insertMetric, row); err != nil {
			return fmt.Errorf("insert metric %s: %w", p.Name, err)
		}
	}

	return tx.Commit()
}

func toMetricRow(p storage.MetricPoint) metricRow {
	row := metricRow{
		MetricName:             p.Name,
		MetricType:             p.Type,
		ServiceName:            p.ServiceName,
		Timestamp:              p.Timestamp.Truncate(time.Minute),
		ValueDouble:            p.ValueDouble,
		ValueInt:               p.ValueInt,
		MetricCount:            p.Count,
		MetricSum:              p.Sum,
		Attributes:             mustJSON(p.Attributes),
		IsMonotonic:            p.IsMonotonic,
		AggregationTemporality: p.AggregationTemporality,
		BucketCounts:           mustJSON(p.BucketCounts),
		ExplicitBounds:         mustJSON(p.ExplicitBounds),
	}

	if p.Unit != "" {
		row.Unit = &p.Unit
	}

	return row
}

package postgres

import (
	"context"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

// JSON fields are stored as TEXT to avoid driver type issues.
// Cast to jsonb in queries when needed: attributes::jsonb->>'key'
const schema = `
CREATE TABLE IF NOT EXISTS traces (
	id                  BIGSERIAL PRIMARY KEY,
	trace_id            VARCHAR(32)  NOT NULL,
	span_id             VARCHAR(16)  NOT NULL,
	parent_span_id      VARCHAR(16),
	operation_name      TEXT         NOT NULL,
	service_name        TEXT         NOT NULL,
	start_time          TIMESTAMPTZ  NOT NULL,
	end_time            TIMESTAMPTZ  NOT NULL,
	duration_ns         BIGINT       NOT NULL,
	status_code         SMALLINT     NOT NULL DEFAULT 0,
	status_message      TEXT,
	kind                SMALLINT     NOT NULL DEFAULT 0,
	attributes          TEXT,
	resource_attributes TEXT,
	events              TEXT,
	links               TEXT,
	created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_traces_trace_id   ON traces(trace_id);
CREATE INDEX IF NOT EXISTS idx_traces_service    ON traces(service_name);
CREATE INDEX IF NOT EXISTS idx_traces_start_time ON traces(start_time);

CREATE TABLE IF NOT EXISTS metrics (
	id                      BIGSERIAL PRIMARY KEY,
	metric_name             TEXT        NOT NULL,
	metric_type             TEXT        NOT NULL,
	service_name            TEXT        NOT NULL,
	timestamp               TIMESTAMPTZ NOT NULL,
	value_double            DOUBLE PRECISION,
	value_int               BIGINT,
	metric_count            BIGINT,
	metric_sum              DOUBLE PRECISION,
	attributes              TEXT,
	resource_attributes     TEXT,
	unit                    TEXT,
	description             TEXT,
	is_monotonic            BOOLEAN,
	aggregation_temporality SMALLINT,
	bucket_counts           TEXT,
	explicit_bounds         TEXT,
	created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_metrics_name      ON metrics(metric_name);
CREATE INDEX IF NOT EXISTS idx_metrics_service   ON metrics(service_name);
CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metrics(timestamp);

CREATE TABLE IF NOT EXISTS logs (
	id                  BIGSERIAL PRIMARY KEY,
	timestamp           TIMESTAMPTZ NOT NULL,
	observed_timestamp  TIMESTAMPTZ,
	trace_id            VARCHAR(32),
	span_id             VARCHAR(16),
	severity_number     SMALLINT,
	severity_text       TEXT,
	body                TEXT,
	service_name        TEXT        NOT NULL,
	attributes          TEXT,
	resource_attributes TEXT,
	created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_logs_service   ON logs(service_name);
CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON logs(timestamp);
CREATE INDEX IF NOT EXISTS idx_logs_trace_id  ON logs(trace_id);
`

type Client struct {
	db *tracedDB
}

func New(dsn string) (*Client, error) {
	db, err := sqlx.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}

	c := &Client{db: &tracedDB{db}}

	if err := c.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return c, nil
}

func (c *Client) migrate() error {
	_, err := c.db.ExecContext(context.Background(), schema)
	return err
}

func (c *Client) Close() error {
	return c.db.Close()
}

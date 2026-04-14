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
	attributes TEXT,
	events      TEXT,
	links       TEXT,
	created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_traces_trace_id   ON traces(trace_id);
CREATE INDEX IF NOT EXISTS idx_traces_service    ON traces(service_name);
CREATE INDEX IF NOT EXISTS idx_traces_start_time ON traces(start_time);

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS idx_traces_root_svc_time  ON traces(service_name, start_time DESC) WHERE parent_span_id IS NULL;
CREATE INDEX IF NOT EXISTS idx_traces_root_time      ON traces(start_time DESC)               WHERE parent_span_id IS NULL;
CREATE INDEX IF NOT EXISTS idx_traces_trace_id_svc   ON traces(trace_id, service_name);
CREATE INDEX IF NOT EXISTS idx_traces_attrs_gin      ON traces USING gin((attributes::jsonb));
CREATE INDEX IF NOT EXISTS idx_traces_operation_trgm ON traces USING gin(operation_name gin_trgm_ops);

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
	unit                    TEXT,
	is_monotonic            BOOLEAN,
	aggregation_temporality SMALLINT,
	bucket_counts           TEXT,
	explicit_bounds         TEXT,
	created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_metrics_name          ON metrics(metric_name);
CREATE INDEX IF NOT EXISTS idx_metrics_service       ON metrics(service_name);
CREATE INDEX IF NOT EXISTS idx_metrics_timestamp     ON metrics(timestamp);
CREATE INDEX IF NOT EXISTS idx_metrics_name_svc_time ON metrics(metric_name, service_name, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_metrics_svc_time      ON metrics(service_name, timestamp DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_metrics_upsert_key ON metrics(metric_name, service_name, timestamp, md5(coalesce(attributes, '{}')));

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
	attributes TEXT,
	created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_logs_service       ON logs(service_name);
CREATE INDEX IF NOT EXISTS idx_logs_timestamp     ON logs(timestamp);
CREATE INDEX IF NOT EXISTS idx_logs_trace_id      ON logs(trace_id);
CREATE INDEX IF NOT EXISTS idx_logs_svc_time      ON logs(service_name, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_logs_severity_time ON logs(severity_text, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_logs_attrs_gin     ON logs USING gin((attributes::jsonb));
CREATE INDEX IF NOT EXISTS idx_logs_body_fts      ON logs USING gin(to_tsvector('simple', COALESCE(body, '')));

CREATE TABLE IF NOT EXISTS dashboards (
	id          VARCHAR(36) PRIMARY KEY,
	name        TEXT        NOT NULL,
	panels      TEXT        NOT NULL DEFAULT '[]',
	created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_dashboards_created ON dashboards(created_at);

CREATE TABLE IF NOT EXISTS users (
	id         BIGSERIAL    PRIMARY KEY,
	email      TEXT         NOT NULL UNIQUE,
	password   TEXT         NOT NULL,
	created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS api_keys (
	id           BIGSERIAL    PRIMARY KEY,
	name         TEXT         NOT NULL,
	key_hash     TEXT         NOT NULL UNIQUE,
	created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
	last_used_at TIMESTAMPTZ
);
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

// dropOrphanCols remove colunas que pararam de ser usadas.
// Usa IF EXISTS para ser idempotente em bancos já migrados ou recém-criados.
const dropOrphanCols = `
ALTER TABLE traces  DROP COLUMN IF EXISTS resource_attributes;
ALTER TABLE metrics DROP COLUMN IF EXISTS resource_attributes;
ALTER TABLE metrics DROP COLUMN IF EXISTS description;
ALTER TABLE logs    DROP COLUMN IF EXISTS resource_attributes;
`

// deduplicateMetrics removes duplicate metric rows before the unique index is created.
// Keeps the latest row (highest id) per (metric_name, service_name, minute_bucket, attributes).
// Safe to run repeatedly — no-op when no duplicates exist.
const deduplicateMetrics = `
DELETE FROM metrics a
USING (
	SELECT id,
	       ROW_NUMBER() OVER (
	           PARTITION BY metric_name, service_name, date_trunc('minute', timestamp), md5(coalesce(attributes, '{}'))
	           ORDER BY id DESC
	       ) AS rn
	FROM metrics
) b
WHERE a.id = b.id AND b.rn > 1
`

func (c *Client) migrate() error {
	if _, err := c.db.ExecContext(context.Background(), schema); err != nil {
		return err
	}
	if _, err := c.db.ExecContext(context.Background(), deduplicateMetrics); err != nil {
		return fmt.Errorf("deduplicate metrics: %w", err)
	}
	_, err := c.db.ExecContext(context.Background(), dropOrphanCols)
	return err
}

func (c *Client) Close() error {
	return c.db.Close()
}

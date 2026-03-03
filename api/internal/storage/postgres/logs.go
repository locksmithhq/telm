package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/locksmithhq/telm/internal/storage"
)

type logRow struct {
	Timestamp          time.Time  `db:"timestamp"`
	ObservedTimestamp  *time.Time `db:"observed_timestamp"`
	TraceID            *string    `db:"trace_id"`
	SpanID             *string    `db:"span_id"`
	SeverityNumber     int32      `db:"severity_number"`
	SeverityText       string     `db:"severity_text"`
	Body               string     `db:"body"`
	ServiceName        string     `db:"service_name"`
	Attributes         string     `db:"attributes"`
	ResourceAttributes string     `db:"resource_attributes"`
}

const insertLog = `
INSERT INTO logs (
	timestamp, observed_timestamp, trace_id, span_id,
	severity_number, severity_text, body, service_name,
	attributes, resource_attributes
) VALUES (
	:timestamp, :observed_timestamp, :trace_id, :span_id,
	:severity_number, :severity_text, :body, :service_name,
	:attributes, :resource_attributes
)`

func (c *Client) SaveLogs(ctx context.Context, logs []storage.Log) error {
	if len(logs) == 0 {
		return nil
	}

	tx, err := c.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	for _, l := range logs {
		row := toLogRow(l)
		if _, err := tx.NamedExecContext(ctx, insertLog, row); err != nil {
			return fmt.Errorf("insert log: %w", err)
		}
	}

	return tx.Commit()
}

func toLogRow(l storage.Log) logRow {
	row := logRow{
		Timestamp:          l.Timestamp,
		ObservedTimestamp:  l.ObservedTimestamp,
		SeverityNumber:     l.SeverityNumber,
		SeverityText:       l.SeverityText,
		Body:               l.Body,
		ServiceName:        l.ServiceName,
		Attributes:         mustJSON(l.Attributes),
		ResourceAttributes: mustJSON(l.ResourceAttributes),
	}

	if l.TraceID != "" {
		row.TraceID = &l.TraceID
	}
	if l.SpanID != "" {
		row.SpanID = &l.SpanID
	}

	return row
}

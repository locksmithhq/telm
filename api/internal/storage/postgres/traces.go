package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/locksmithhq/telm/internal/storage"
)

type spanRow struct {
	TraceID            string    `db:"trace_id"`
	SpanID             string    `db:"span_id"`
	ParentSpanID       *string   `db:"parent_span_id"`
	OperationName      string    `db:"operation_name"`
	ServiceName        string    `db:"service_name"`
	StartTime          time.Time `db:"start_time"`
	EndTime            time.Time `db:"end_time"`
	DurationNs         int64     `db:"duration_ns"`
	StatusCode         int32     `db:"status_code"`
	StatusMessage      *string   `db:"status_message"`
	Kind               int32     `db:"kind"`
	Attributes         string    `db:"attributes"`
	ResourceAttributes string    `db:"resource_attributes"`
	Events             string    `db:"events"`
	Links              string    `db:"links"`
}

const insertSpan = `
INSERT INTO traces (
	trace_id, span_id, parent_span_id, operation_name, service_name,
	start_time, end_time, duration_ns, status_code, status_message, kind,
	attributes, resource_attributes, events, links
) VALUES (
	:trace_id, :span_id, :parent_span_id, :operation_name, :service_name,
	:start_time, :end_time, :duration_ns, :status_code, :status_message, :kind,
	:attributes, :resource_attributes, :events, :links
)`

func (c *Client) SaveSpans(ctx context.Context, spans []storage.Span) error {
	if len(spans) == 0 {
		return nil
	}

	tx, err := c.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	for _, s := range spans {
		row := toSpanRow(s)
		if _, err := tx.NamedExecContext(ctx, insertSpan, row); err != nil {
			return fmt.Errorf("insert span %s: %w", s.SpanID, err)
		}
	}

	return tx.Commit()
}

func toSpanRow(s storage.Span) spanRow {
	row := spanRow{
		TraceID:            s.TraceID,
		SpanID:             s.SpanID,
		OperationName:      s.OperationName,
		ServiceName:        s.ServiceName,
		StartTime:          s.StartTime,
		EndTime:            s.EndTime,
		DurationNs:         s.DurationNs,
		StatusCode:         s.StatusCode,
		Kind:               s.Kind,
		Attributes:         mustJSON(s.Attributes),
		ResourceAttributes: mustJSON(s.ResourceAttributes),
		Events:             mustJSON(s.Events),
		Links:              mustJSON(s.Links),
	}

	if s.ParentSpanID != "" {
		row.ParentSpanID = &s.ParentSpanID
	}
	if s.StatusMessage != "" {
		row.StatusMessage = &s.StatusMessage
	}

	return row
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "null"
	}
	return string(b)
}

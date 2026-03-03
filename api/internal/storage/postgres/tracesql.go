package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"go.opentelemetry.io/otel/trace"
)

// tracedDB wraps sqlx.DB and prepends a W3C traceparent SQL comment on every
// SELECT query so that pgaudit logs can be correlated with traces in telm.
type tracedDB struct {
	*sqlx.DB
}

// QueryxContext injects the traceparent comment before executing.
func (t *tracedDB) QueryxContext(ctx context.Context, query string, args ...any) (*sqlx.Rows, error) {
	return t.DB.QueryxContext(ctx, withTraceComment(ctx, query), args...)
}

// SelectContext injects the traceparent comment before executing.
func (t *tracedDB) SelectContext(ctx context.Context, dest any, query string, args ...any) error {
	return t.DB.SelectContext(ctx, dest, withTraceComment(ctx, query), args...)
}

// withTraceComment prepends a SQL comment containing the active W3C traceparent.
// If there is no valid span in the context the query is returned unchanged.
//
// pgaudit captures the full statement including this comment, which the OTEL
// Collector filelog receiver parses to link the audit log entry back to the
// originating trace.
//
// Example output:
//
//	/*traceparent='00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01'*/ SELECT ...
func withTraceComment(ctx context.Context, query string) string {
	sc := trace.SpanFromContext(ctx).SpanContext()
	if !sc.IsValid() {
		return query
	}
	return fmt.Sprintf("/*traceparent='00-%s-%s-01'*/ %s",
		sc.TraceID(), sc.SpanID(), query)
}

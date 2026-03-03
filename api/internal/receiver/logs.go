package receiver

import (
	"context"
	"log/slog"

	logscollv1 "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	logsv1 "go.opentelemetry.io/proto/otlp/logs/v1"

	"github.com/locksmithhq/telm/internal/storage"
)

type logsService struct {
	logscollv1.UnimplementedLogsServiceServer
	store storage.Storage
}

func (s *logsService) Export(ctx context.Context, req *logscollv1.ExportLogsServiceRequest) (*logscollv1.ExportLogsServiceResponse, error) {
	logs := extractLogs(req.ResourceLogs)

	slog.Debug("received log records", "count", len(logs))

	if err := s.store.SaveLogs(ctx, logs); err != nil {
		slog.Error("failed to save logs", "error", err)
		return nil, err
	}

	return &logscollv1.ExportLogsServiceResponse{}, nil
}

// severityNumberToText deriva o texto a partir do número quando o sender não envia SeverityText.
// https://opentelemetry.io/docs/specs/otel/logs/data-model/#field-severitynumber
func severityNumberToText(n int32) string {
	switch {
	case n >= 21:
		return "FATAL"
	case n >= 17:
		return "ERROR"
	case n >= 13:
		return "WARN"
	case n >= 9:
		return "INFO"
	case n >= 5:
		return "DEBUG"
	case n >= 1:
		return "TRACE"
	default:
		return ""
	}
}

func extractLogs(resourceLogs []*logsv1.ResourceLogs) []storage.Log {
	var logs []storage.Log

	for _, rl := range resourceLogs {
		resourceAttrs := attributesToMap(rl.GetResource().GetAttributes())
		serviceName := getServiceName(resourceAttrs)

		for _, sl := range rl.ScopeLogs {
			for _, lr := range sl.LogRecords {
				sevText := lr.SeverityText
				if sevText == "" {
					sevText = severityNumberToText(int32(lr.SeverityNumber))
				}

				attrs := attributesToMap(lr.Attributes)

				// trace_id/span_id can come either from the OTLP LogRecord fields
				// (SDK-emitted logs) or from attributes (filelog/pgaudit path, where
				// the stanza regex_parser extracts them from the SQL traceparent comment).
				traceID := traceIDToHex(lr.TraceId)
				if traceID == "" {
					traceID = attrs["trace_id"]
				}
				spanID := spanIDToHex(lr.SpanId)
				if spanID == "" {
					spanID = attrs["span_id"]
				}

				l := storage.Log{
					Timestamp:          nanosToTime(lr.TimeUnixNano),
					TraceID:            traceID,
					SpanID:             spanID,
					SeverityNumber:     int32(lr.SeverityNumber),
					SeverityText:       sevText,
					Body:               anyValueToString(lr.Body),
					ServiceName:        serviceName,
					Attributes:         attrs,
					ResourceAttributes: resourceAttrs,
				}

				if lr.ObservedTimeUnixNano != 0 {
					t := nanosToTime(lr.ObservedTimeUnixNano)
					l.ObservedTimestamp = &t
				}

				logs = append(logs, l)
			}
		}
	}

	return logs
}

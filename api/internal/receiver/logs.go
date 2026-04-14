package receiver

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"unicode/utf8"

	logscollv1 "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	logsv1 "go.opentelemetry.io/proto/otlp/logs/v1"

	"github.com/locksmithhq/telm/internal/storage"
)

// minLogSeverity descarta registros abaixo deste nível.
// 0=unspecified 1-4=TRACE 5-8=DEBUG 9-12=INFO 13-16=WARN 17-20=ERROR 21-24=FATAL
// Override via MIN_LOG_SEVERITY env var (default: 9 = INFO).
var minLogSeverity = func() int32 {
	v, err := strconv.ParseInt(os.Getenv("MIN_LOG_SEVERITY"), 10, 32)
	if err != nil || v <= 0 {
		return 9
	}
	return int32(v)
}()

// maxLogBodyBytes limita o tamanho do campo body por registro.
// Override via MAX_LOG_BODY_BYTES env var (default: 4096).
var maxLogBodyBytes = func() int {
	v, err := strconv.Atoi(os.Getenv("MAX_LOG_BODY_BYTES"))
	if err != nil || v <= 0 {
		return 4096
	}
	return v
}()

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
				sev := int32(lr.SeverityNumber)

				// Descartar registros abaixo da severidade mínima.
				// SeverityNumber == 0 significa não especificado — manter.
				if sev > 0 && sev < minLogSeverity {
					continue
				}

				sevText := lr.SeverityText
				if sevText == "" {
					sevText = severityNumberToText(sev)
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
					Timestamp:      nanosToTime(lr.TimeUnixNano),
					TraceID:        traceID,
					SpanID:         spanID,
					SeverityNumber: sev,
					SeverityText:   sevText,
					Body:           truncateBody(anyValueToString(lr.Body), maxLogBodyBytes),
					ServiceName:    serviceName,
					Attributes:     attrs,
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

// truncateBody corta s para no máximo maxBytes de forma UTF-8 safe.
func truncateBody(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	b := []byte(s[:maxBytes])
	for len(b) > 0 && !utf8.Valid(b) {
		b = b[:len(b)-1]
	}
	return string(b)
}

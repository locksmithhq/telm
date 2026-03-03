package receiver

import (
	"context"
	"log/slog"

	tracecollv1 "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	tracev1 "go.opentelemetry.io/proto/otlp/trace/v1"

	"github.com/locksmithhq/telm/internal/storage"
)

type traceService struct {
	tracecollv1.UnimplementedTraceServiceServer
	store storage.Storage
}

func (s *traceService) Export(ctx context.Context, req *tracecollv1.ExportTraceServiceRequest) (*tracecollv1.ExportTraceServiceResponse, error) {
	spans := extractSpans(req.ResourceSpans)

	slog.Debug("received spans", "count", len(spans))

	if err := s.store.SaveSpans(ctx, spans); err != nil {
		slog.Error("failed to save spans", "error", err)
		return nil, err
	}

	return &tracecollv1.ExportTraceServiceResponse{}, nil
}

func extractSpans(resourceSpans []*tracev1.ResourceSpans) []storage.Span {
	var spans []storage.Span

	for _, rs := range resourceSpans {
		resourceAttrs := attributesToMap(rs.GetResource().GetAttributes())
		serviceName := getServiceName(resourceAttrs)

		for _, ss := range rs.ScopeSpans {
			for _, s := range ss.Spans {
				span := storage.Span{
					TraceID:            traceIDToHex(s.TraceId),
					SpanID:             spanIDToHex(s.SpanId),
					ParentSpanID:       spanIDToHex(s.ParentSpanId),
					OperationName:      s.Name,
					ServiceName:        serviceName,
					StartTime:          nanosToTime(s.StartTimeUnixNano),
					EndTime:            nanosToTime(s.EndTimeUnixNano),
					DurationNs:         int64(s.EndTimeUnixNano - s.StartTimeUnixNano),
					StatusCode:         int32(s.Status.GetCode()),
					StatusMessage:      s.Status.GetMessage(),
					Kind:               int32(s.Kind),
					Attributes:         attributesToMap(s.Attributes),
					ResourceAttributes: resourceAttrs,
					Events:             extractEvents(s.Events),
					Links:              extractLinks(s.Links),
				}
				spans = append(spans, span)
			}
		}
	}

	return spans
}

func extractEvents(events []*tracev1.Span_Event) []storage.SpanEvent {
	result := make([]storage.SpanEvent, 0, len(events))
	for _, e := range events {
		result = append(result, storage.SpanEvent{
			Name:       e.Name,
			Time:       nanosToTime(e.TimeUnixNano),
			Attributes: attributesToMap(e.Attributes),
		})
	}
	return result
}

func extractLinks(links []*tracev1.Span_Link) []storage.SpanLink {
	result := make([]storage.SpanLink, 0, len(links))
	for _, l := range links {
		result = append(result, storage.SpanLink{
			TraceID:    traceIDToHex(l.TraceId),
			SpanID:     spanIDToHex(l.SpanId),
			Attributes: attributesToMap(l.Attributes),
		})
	}
	return result
}

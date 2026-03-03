package storage

import (
	"context"
	"time"
)

type Span struct {
	TraceID            string
	SpanID             string
	ParentSpanID       string
	OperationName      string
	ServiceName        string
	StartTime          time.Time
	EndTime            time.Time
	DurationNs         int64
	StatusCode         int32
	StatusMessage      string
	Kind               int32
	Attributes         map[string]string
	ResourceAttributes map[string]string
	Events             []SpanEvent
	Links              []SpanLink
}

type SpanEvent struct {
	Name       string            `json:"name"`
	Time       time.Time         `json:"time"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

type SpanLink struct {
	TraceID    string            `json:"trace_id"`
	SpanID     string            `json:"span_id"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

type MetricPoint struct {
	Name                   string
	Type                   string // gauge, sum, histogram, summary
	ServiceName            string
	Timestamp              time.Time
	ValueDouble            *float64
	ValueInt               *int64
	Count                  *int64
	Sum                    *float64
	Attributes             map[string]string
	ResourceAttributes     map[string]string
	Unit                   string
	Description            string
	IsMonotonic            *bool
	AggregationTemporality *int32
	BucketCounts           []uint64
	ExplicitBounds         []float64
}

type Log struct {
	Timestamp          time.Time
	ObservedTimestamp  *time.Time
	TraceID            string
	SpanID             string
	SeverityNumber     int32
	SeverityText       string
	Body               string
	ServiceName        string
	Attributes         map[string]string
	ResourceAttributes map[string]string
}

type Storage interface {
	SaveSpans(ctx context.Context, spans []Span) error
	SaveMetrics(ctx context.Context, points []MetricPoint) error
	SaveLogs(ctx context.Context, logs []Log) error
	Close() error
}

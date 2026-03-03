package receiver

import (
	"context"
	"log/slog"

	metricscollv1 "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	metricsv1 "go.opentelemetry.io/proto/otlp/metrics/v1"

	"github.com/locksmithhq/telm/internal/storage"
)

type metricsService struct {
	metricscollv1.UnimplementedMetricsServiceServer
	store storage.Storage
}

func (s *metricsService) Export(ctx context.Context, req *metricscollv1.ExportMetricsServiceRequest) (*metricscollv1.ExportMetricsServiceResponse, error) {
	points := extractMetrics(req.ResourceMetrics)

	slog.Debug("received metric points", "count", len(points))

	if err := s.store.SaveMetrics(ctx, points); err != nil {
		slog.Error("failed to save metrics", "error", err)
		return nil, err
	}

	return &metricscollv1.ExportMetricsServiceResponse{}, nil
}

func extractMetrics(resourceMetrics []*metricsv1.ResourceMetrics) []storage.MetricPoint {
	var points []storage.MetricPoint

	for _, rm := range resourceMetrics {
		resourceAttrs := attributesToMap(rm.GetResource().GetAttributes())
		serviceName := getServiceName(resourceAttrs)

		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				points = append(points, extractDataPoints(m, serviceName, resourceAttrs)...)
			}
		}
	}

	return points
}

func extractDataPoints(m *metricsv1.Metric, serviceName string, resourceAttrs map[string]string) []storage.MetricPoint {
	makeBase := func(ts uint64) storage.MetricPoint {
		return storage.MetricPoint{
			Name:               m.Name,
			ServiceName:        serviceName,
			Unit:               m.Unit,
			Description:        m.Description,
			ResourceAttributes: resourceAttrs,
			Timestamp:          nanosToTime(ts),
		}
	}

	var points []storage.MetricPoint

	switch data := m.Data.(type) {
	case *metricsv1.Metric_Gauge:
		for _, dp := range data.Gauge.DataPoints {
			p := makeBase(dp.TimeUnixNano)
			p.Type = "gauge"
			p.Attributes = attributesToMap(dp.Attributes)
			setNumberValue(&p, dp)
			points = append(points, p)
		}

	case *metricsv1.Metric_Sum:
		at := int32(data.Sum.AggregationTemporality)
		isMonotonic := data.Sum.IsMonotonic
		for _, dp := range data.Sum.DataPoints {
			p := makeBase(dp.TimeUnixNano)
			p.Type = "sum"
			p.Attributes = attributesToMap(dp.Attributes)
			p.AggregationTemporality = &at
			p.IsMonotonic = &isMonotonic
			setNumberValue(&p, dp)
			points = append(points, p)
		}

	case *metricsv1.Metric_Histogram:
		at := int32(data.Histogram.AggregationTemporality)
		for _, dp := range data.Histogram.DataPoints {
			p := makeBase(dp.TimeUnixNano)
			p.Type = "histogram"
			p.Attributes = attributesToMap(dp.Attributes)
			p.AggregationTemporality = &at
			count := int64(dp.Count)
			p.Count = &count
			if dp.Sum != nil {
				p.Sum = dp.Sum
			}
			p.BucketCounts = dp.BucketCounts
			p.ExplicitBounds = dp.ExplicitBounds
			points = append(points, p)
		}

	case *metricsv1.Metric_Summary:
		for _, dp := range data.Summary.DataPoints {
			p := makeBase(dp.TimeUnixNano)
			p.Type = "summary"
			p.Attributes = attributesToMap(dp.Attributes)
			count := int64(dp.Count)
			p.Count = &count
			s := dp.Sum
			p.Sum = &s
			points = append(points, p)
		}
	}

	return points
}

func setNumberValue(p *storage.MetricPoint, dp *metricsv1.NumberDataPoint) {
	switch v := dp.Value.(type) {
	case *metricsv1.NumberDataPoint_AsDouble:
		p.ValueDouble = &v.AsDouble
	case *metricsv1.NumberDataPoint_AsInt:
		p.ValueInt = &v.AsInt
	}
}

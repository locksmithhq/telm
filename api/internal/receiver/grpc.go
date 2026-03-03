package receiver

import (
	"fmt"
	"log/slog"
	"net"

	logscollv1 "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	metricscollv1 "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	tracecollv1 "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip" // registra decompressor gzip (usado pelo otelcollector)
	"google.golang.org/grpc/reflection"

	"github.com/locksmithhq/telm/internal/storage"
)

type Server struct {
	grpc *grpc.Server
}

func NewServer(store storage.Storage) *Server {
	srv := grpc.NewServer(
		grpc.MaxRecvMsgSize(64 * 1024 * 1024), // 64 MB
	)

	tracecollv1.RegisterTraceServiceServer(srv, &traceService{store: store})
	metricscollv1.RegisterMetricsServiceServer(srv, &metricsService{store: store})
	logscollv1.RegisterLogsServiceServer(srv, &logsService{store: store})

	// reflection para uso com grpcurl / evans
	reflection.Register(srv)

	return &Server{grpc: srv}
}

func (s *Server) Start(port string) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("listen on :%s: %w", port, err)
	}

	go func() {
		if err := s.grpc.Serve(lis); err != nil {
			slog.Error("gRPC server stopped", "error", err)
		}
	}()

	return nil
}

func (s *Server) Stop() {
	s.grpc.GracefulStop()
}

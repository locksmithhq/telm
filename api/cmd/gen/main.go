// Telemetry generator — simulates an e-commerce system with multiple microservices.
//
// Simulated architecture:
//
//	api-gateway  →  user-service     (via HTTP + W3C trace propagation)
//	             →  payment-service  (via HTTP + W3C trace propagation)
//	             →  notification-service (via HTTP + W3C trace propagation)
//
// Each layer uses the telm SDK:
//
//	telm.Start / telm.Span / telm.SpanE  → traces
//	telm.Info / telm.Warn / telm.Error   → logs
//	telm.Count / telm.Record / telm.Gauge → metrics
//
// Usage:
//
//	make gen                               # via otelcollector (HTTP :4318)
//	OTLP_ENDPOINT=api-telm:4318 make gen   # direct to telm
package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/locksmithhq/telm-go"
)

func main() {
	endpoint := getEnv("OTLP_ENDPOINT", "otelcollector-telm:4318")
	svcName := getEnv("SERVICE_NAME", "api-gateway")
	workers := getEnvInt("WORKERS", 5)
	sleepMs := getEnvInt("SLEEP_MS", 0)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ── Initialize the telm SDK ──────────────────────────────────────────────
	shutdown, err := telm.Init(ctx,
		telm.WithServiceName(svcName),
		telm.WithEndpoint(endpoint),
	)
	if err != nil {
		slog.Error("telm init failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		if sErr := shutdown(context.Background()); sErr != nil {
			slog.Error("telm shutdown", "error", sErr)
		}
	}()

	// ── Build dependency graph ───────────────────────────────────────────────
	handler := newAPIHandler(svcName)

	// System routes
	type route struct {
		method    string
		path      string
		fullOrder bool // true = full flow with payment
	}
	routes := []route{
		{"POST", "/api/orders", true},
		{"GET", "/api/orders", false},
		{"GET", "/api/products", false},
		{"GET", "/api/users/me", false},
		{"PUT", "/api/users/me", false},
		{"GET", "/api/payments/history", false},
	}

	// ── Periodic stats ───────────────────────────────────────────────────────
	var total atomic.Int64
	start := time.Now()

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				n := total.Load()
				elapsed := time.Since(start).Seconds()
				telm.Info(ctx, "generator stats", telm.F{
					"total":   n,
					"req_s":   fmt.Sprintf("%.2f", float64(n)/elapsed),
					"req_min": fmt.Sprintf("%.1f", float64(n)/elapsed*60),
					"service": svcName,
				})
			}
		}
	}()

	telm.Info(ctx, "generator started", telm.F{
		"service":   svcName,
		"workers":   workers,
		"sleep_ms":  sleepMs,
		"endpoint":  endpoint,
		"transport": "OTLP/HTTP",
	})

	// ── Workers ──────────────────────────────────────────────────────────────
	var wg sync.WaitGroup
	for w := range workers {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				r := routes[rand.IntN(len(routes))]
				statusCode := pickStatus()

				telm.Gauge(ctx, "requests.active", 1, telm.F{"worker": workerID})

				var dispatchErr error
				if r.fullOrder && statusCode < 500 {
					dispatchErr = handler.HandlePlaceOrder(ctx, r.method, r.path, statusCode)
				} else {
					dispatchErr = handler.HandleSimple(ctx, r.method, r.path, statusCode)
				}

				telm.Gauge(ctx, "requests.active", -1, telm.F{"worker": workerID})

				_ = dispatchErr
				n := total.Add(1)
				if n%500 == 0 {
					telm.Debug(ctx, "worker checkpoint",
						telm.F{"worker": workerID, "total": n},
					)
				}

				if sleepMs > 0 {
					time.Sleep(time.Duration(sleepMs) * time.Millisecond)
				}
			}
		}(w)
	}

	<-ctx.Done()
	telm.Info(ctx, "generator stopping", telm.F{"total_sent": total.Load()})
	wg.Wait()
	telm.Info(ctx, "generator done", telm.F{"total_sent": total.Load()})
}

func pickStatus() int {
	n := rand.IntN(10)
	switch {
	case n < 7:
		return 200
	case n < 9:
		return 400 + rand.IntN(5)*10
	default:
		return 500
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

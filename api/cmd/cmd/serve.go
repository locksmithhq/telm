package cmd

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/locksmithhq/telm/internal/adapter/database"
	"github.com/locksmithhq/telm/internal/adapter/rest"
	"github.com/locksmithhq/telm/internal/api"
	"github.com/locksmithhq/telm/internal/receiver"
	"github.com/locksmithhq/telm/internal/storage/postgres"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use: "serve",
	Run: func(cmd *cobra.Command, args []string) {
		jwtSecret := getEnv("JWT_SECRET", "")
		if jwtSecret == "" {
			slog.Error("JWT_SECRET env var is required")
			os.Exit(1)
		}
		if len(jwtSecret) < 32 || jwtSecret == "change-me-use-openssl-rand-base64-32" {
			slog.Error("JWT_SECRET inseguro: deve ter pelo menos 32 caracteres e não pode ser o valor padrão")
			os.Exit(1)
		}

		store := database.Initialize()
		defer store.Close()

		seedAdmin(store)

		grpcPort := getEnv("GRPC_PORT", "9317")
		grpcSrv := receiver.NewServer(store)
		if err := grpcSrv.Start(grpcPort); err != nil {
			slog.Error("failed to start gRPC server", "error", err)
			os.Exit(1)
		}
		slog.Info("OTLP gRPC receiver listening", "port", grpcPort)

		startCleanupCron(store)

		// Inicia HTTP (blocking)
		rest.Initialize(store, []byte(jwtSecret))
	},
}

func seedAdmin(store *postgres.Client) {
	email := os.Getenv("ADMIN_EMAIL")
	password := os.Getenv("ADMIN_PASSWORD")
	if email == "" || password == "" {
		slog.Warn("ADMIN_EMAIL/ADMIN_PASSWORD not set — skipping admin seed")
		return
	}
	hash, err := api.HashPassword(password)
	if err != nil {
		slog.Error("failed to hash admin password", "error", err)
		os.Exit(1)
	}
	if err := store.UpsertAdminUser(context.Background(), email, hash); err != nil {
		slog.Error("failed to seed admin user", "error", err)
		os.Exit(1)
	}
	slog.Info("admin user ready")
}

// startCleanupCron dispara uma goroutine que roda os cleanups todo dia às 03:00.
// O VACUUM FULL reescreve cada tabela e devolve espaço físico ao OS.
func startCleanupCron(store *postgres.Client) {
	go func() {
		for {
			next := nextDailyRun(3)
			slog.Info("cleanup agendado",
				"next_run", next.Format(time.RFC3339),
				"log_retention_days", postgres.LogRetentionDays,
				"trace_retention_days", postgres.TraceRetentionDays,
				"metric_retention_days", postgres.MetricRetentionDays,
			)
			time.Sleep(time.Until(next))

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)

			if deleted, err := store.CleanupLogs(ctx); err != nil {
				slog.Error("log cleanup falhou", "error", err)
			} else {
				slog.Info("log cleanup concluído", "deleted_rows", deleted)
			}

			if deleted, err := store.CleanupTraces(ctx); err != nil {
				slog.Error("trace cleanup falhou", "error", err)
			} else {
				slog.Info("trace cleanup concluído", "deleted_rows", deleted)
			}

			if deleted, err := store.CleanupMetrics(ctx); err != nil {
				slog.Error("metric cleanup falhou", "error", err)
			} else {
				slog.Info("metric cleanup concluído", "deleted_rows", deleted)
			}

			cancel()
		}
	}()
}

// nextDailyRun retorna o próximo instante em que o relógio local bater hora:00:00.
// Se já passou hoje, retorna o mesmo horário amanhã.
func nextDailyRun(hour int) time.Time {
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

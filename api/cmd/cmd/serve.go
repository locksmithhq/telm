package cmd

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/locksmithhq/telm/internal/adapter/database"
	"github.com/locksmithhq/telm/internal/adapter/rest"
	"github.com/locksmithhq/telm/internal/receiver"
	"github.com/locksmithhq/telm/internal/storage/postgres"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use: "serve",
	Run: func(cmd *cobra.Command, args []string) {
		store := database.Initialize()
		defer store.Close()

		grpcPort := getEnv("GRPC_PORT", "9317")
		grpcSrv := receiver.NewServer(store)
		if err := grpcSrv.Start(grpcPort); err != nil {
			slog.Error("failed to start gRPC server", "error", err)
			os.Exit(1)
		}
		slog.Info("OTLP gRPC receiver listening", "port", grpcPort)

		startCleanupCron(store)

		// Inicia HTTP (blocking)
		rest.Initialize(store)
	},
}

// startCleanupCron dispara uma goroutine que roda CleanupLogs todo dia às 03:00.
// O VACUUM FULL reescreve a tabela de logs e devolve espaço físico ao OS.
func startCleanupCron(store *postgres.Client) {
	go func() {
		for {
			next := nextDailyRun(3)
			slog.Info("log cleanup agendado",
				"retention_days", postgres.LogRetentionDays,
				"next_run", next.Format(time.RFC3339),
			)
			time.Sleep(time.Until(next))

			// Timeout generoso: VACUUM FULL em tabelas grandes pode levar minutos.
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
			deleted, err := store.CleanupLogs(ctx)
			cancel()

			if err != nil {
				slog.Error("log cleanup falhou", "error", err, "deleted", deleted)
			} else {
				slog.Info("log cleanup concluído",
					"deleted_rows", deleted,
					"retention_days", postgres.LogRetentionDays,
				)
			}
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

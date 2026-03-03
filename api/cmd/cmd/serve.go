package cmd

import (
	"log/slog"
	"os"

	"github.com/locksmithhq/telm/internal/adapter/database"
	"github.com/locksmithhq/telm/internal/adapter/rest"
	"github.com/locksmithhq/telm/internal/receiver"

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

		// Inicia HTTP (blocking)
		rest.Initialize(store)
	},
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

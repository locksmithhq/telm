package database

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/locksmithhq/telm/internal/storage/postgres"
)

func Initialize() *postgres.Client {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_DB"),
		os.Getenv("SSL_MODE"),
	)

	store, err := postgres.New(dsn)
	if err != nil {
		slog.Error("failed to connect to postgres", "error", err)
		os.Exit(1)
	}

	slog.Info("connected to postgres")
	return store
}

// pgaudit-test is a small HTTP service that validates the pgaudit → OTEL
// trace-linking pipeline.
//
// Each endpoint:
//  1. Opens a SERVER span via the telm SDK (so a traceparent exists in ctx)
//  2. Runs real SQL through sqlx using withTraceComment, which prepends
//     /*traceparent='00-<trace_id>-<span_id>-01'*/ to the query
//  3. pgaudit captures the statement; the OTEL Collector filelog receiver
//     extracts the traceparent and correlates the audit log with the trace
//
// Run:
//
//	docker compose --profile pgaudit-test up pgaudit-test
//	./scripts/test-pgaudit.sh
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"

	"github.com/locksmithhq/telm-go"
)

var db *sqlx.DB

func main() {
	ctx := context.Background()

	// ── OTel init via telm SDK ────────────────────────────────────────────
	endpoint := getenv("OTLP_ENDPOINT", "otelcollector-telm:4318")
	shutdown, err := telm.Init(ctx,
		telm.WithServiceName("pgaudit-test"),
		telm.WithEndpoint(endpoint),
	)
	if err != nil {
		log.Fatalf("telm init: %v", err)
	}
	defer shutdown(ctx) //nolint:errcheck

	// ── Postgres connection ───────────────────────────────────────────────
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		getenv("POSTGRES_USER", "telm"),
		getenv("POSTGRES_PASSWORD", "telm123"),
		getenv("POSTGRES_HOST", "localhost"),
		getenv("POSTGRES_PORT", "5432"),
		getenv("POSTGRES_DB", "telm"),
		getenv("SSL_MODE", "disable"),
	)
	db, err = sqlx.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatalf("db ping: %v", err)
	}
	defer db.Close() //nolint:errcheck
	log.Println("connected to postgres")

	// ── HTTP routes ───────────────────────────────────────────────────────
	mux := http.NewServeMux()
	mux.HandleFunc("GET /select", handleSelect)
	mux.HandleFunc("GET /insert", handleInsert)
	mux.HandleFunc("GET /ddl", handleDDL)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	port := getenv("HTTP_PORT", "9001")
	log.Printf("pgaudit-test listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

// handleSelect executes SELECTs against all three tables.
// Each query carries the traceparent comment so pgaudit logs link to this span.
func handleSelect(w http.ResponseWriter, r *http.Request) {
	ctx, end := telm.Start(r.Context(), "select.counts", telm.Server())
	defer end(nil)

	telm.Attr(ctx, telm.F{"http.method": "GET", "http.route": "/select"})

	result := map[string]any{}
	for _, table := range []string{"traces", "metrics", "logs"} {
		q := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
		telm.Attr(ctx, telm.F{"query": q})
		var n int64
		if err := db.QueryRowContext(ctx, q).Scan(&n); err != nil {
			telm.Error(ctx, "select failed", err, telm.F{"table": table})
			http.Error(w, err.Error(), http.StatusInternalServerError)
			end(err)
			return
		}
		result[table] = n
	}

	telm.Info(ctx, "select ok", telm.F(result))
	writeJSON(w, result)
}

// handleInsert inserts a test log row and returns the generated id.
func handleInsert(w http.ResponseWriter, r *http.Request) {
	ctx, end := telm.Start(r.Context(), "insert.log_row", telm.Server())
	defer end(nil)

	telm.Attr(ctx, telm.F{"http.method": "GET", "http.route": "/insert"})

	query := `
		INSERT INTO logs (timestamp, severity_number, severity_text, body, service_name)
		VALUES ($1, 9, 'INFO', $2, 'pgaudit-test')
		RETURNING id`

	telm.Attr(ctx, telm.F{"query": query})

	var id int64
	err := db.QueryRowContext(ctx, query,
		time.Now().UTC(),
		fmt.Sprintf("pgaudit-test insert at %s", time.Now().Format(time.RFC3339)),
	).Scan(&id)
	if err != nil {
		telm.Error(ctx, "insert failed", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		end(err)
		return
	}

	telm.Info(ctx, "insert ok", telm.F{"log_id": id})
	writeJSON(w, map[string]any{"inserted_id": id})
}

// handleDDL runs a CREATE TABLE followed by a DROP TABLE.
// These trigger pgaudit DDL audit entries.
func handleDDL(w http.ResponseWriter, r *http.Request) {
	ctx, end := telm.Start(r.Context(), "ddl.create_drop", telm.Client())
	defer end(nil)

	telm.Attr(ctx, telm.F{"http.method": "GET", "http.route": "/ddl"})

	stmts := []string{
		"CREATE TABLE IF NOT EXISTS pgaudit_probe (id SERIAL PRIMARY KEY, ts TIMESTAMPTZ DEFAULT NOW())",
		"DROP TABLE IF EXISTS pgaudit_probe",
	}

	for _, stmt := range stmts {
		telm.Attr(ctx, telm.F{"query": stmt})
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			telm.Error(ctx, "ddl failed", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			end(err)
			return
		}
	}

	telm.Info(ctx, "ddl ok")
	writeJSON(w, map[string]any{"ddl": "create+drop ok"})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

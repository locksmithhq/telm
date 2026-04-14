package rest

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/locksmithhq/telm/internal/api"
	"github.com/locksmithhq/telm/internal/storage/postgres"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
)

func Initialize(store *postgres.Client, jwtSecret []byte) {
	r := chi.NewRouter()

	corsOrigins := []string{"http://localhost:3000", "http://localhost:4000"}
	if origin := os.Getenv("CORS_ORIGIN"); origin != "" {
		corsOrigins = strings.Split(origin, ",")
	}

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	h := api.NewServer(store, jwtSecret)

	// OTLP ingestion proxy — protected by API key, no /api prefix
	r.Route("/otlp", func(r chi.Router) {
		r.Use(h.APIKeyMiddleware)
		r.Handle("/*", h.OTLPProxy())
	})

	r.Route("/api", func(r chi.Router) {
		r.With(httprate.LimitByIP(10, time.Minute)).Post("/auth/login", h.HandleLogin)

		r.Group(func(r chi.Router) {
			r.Use(h.JWTMiddleware)
			r.Post("/auth/logout", h.HandleLogout)
			r.Get("/auth/me", h.HandleMe)
			r.Get("/services", h.HandleServices)
			r.Get("/traces", h.HandleTraces)
			r.Get("/traces/{traceId}", h.HandleTraceDetail)
			r.Get("/traces/{traceId}/logs", h.HandleTraceLogs)
			r.Get("/metrics", h.HandleMetrics)
			r.Get("/metrics/catalog", h.HandleMetricCatalog)
			r.Get("/metrics/series", h.HandleMetricSeries)
			r.Get("/logs", h.HandleLogs)
			r.Get("/dashboards", h.HandleDashboards)
			r.Post("/dashboards", h.HandleDashboards)
			r.Get("/dashboards/{id}", h.HandleDashboard)
			r.Put("/dashboards/{id}", h.HandleDashboard)
			r.Delete("/dashboards/{id}", h.HandleDashboard)
			r.Get("/stats/throughput", h.HandleThroughput)
			r.Get("/stats/errors", h.HandleErrors)
			r.Get("/stats/latency", h.HandleLatency)
			r.Get("/stats/top-ops", h.HandleTopOps)
			r.Get("/stats/severity", h.HandleSeverity)
			r.Get("/stats/services-health", h.HandleServiceHealth)
			r.Get("/stats/service-map", h.HandleServiceMap)
			r.Get("/stats/resources", h.HandleResources)
			r.Get("/stats/resources/all", h.HandleAllResources)
			r.Get("/stats/storage", h.HandleStorageStats)
			r.Get("/stats/storage/growth", h.HandleStorageGrowth)

			r.Post("/apikeys", h.HandleCreateAPIKey)
			r.Get("/apikeys", h.HandleListAPIKeys)
			r.Delete("/apikeys/{id}", h.HandleRevokeAPIKey)
		})
	})

	spaPath := "/web/dist"
	fs := http.FileServer(http.Dir(spaPath))
	r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
		fullPath := filepath.Join(spaPath, req.URL.Path)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			http.ServeFile(w, req, filepath.Join(spaPath, "index.html"))
			return
		}
		fs.ServeHTTP(w, req)
	})

	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("REST API serving on port: %s\n", port)
	http.ListenAndServe(":"+port, r) //nolint:errcheck
}

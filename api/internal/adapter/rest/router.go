package rest

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/locksmithhq/telm/internal/api"
	"github.com/locksmithhq/telm/internal/storage/postgres"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func Initialize(store *postgres.Client) {
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	h := api.NewServer(store)

	r.Route("/api", func(r chi.Router) {
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

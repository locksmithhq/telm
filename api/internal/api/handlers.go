package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/locksmithhq/telm/internal/storage/postgres"

	"github.com/go-chi/chi/v5"
)

func (s *Server) HandleServices(w http.ResponseWriter, r *http.Request) {
	services, err := s.store.ListServices(r.Context())
	if err != nil {
		jsonErr(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, services)
}

func (s *Server) HandleTraces(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := postgres.TraceFilter{
		Service:   q.Get("service"),
		Operation: q.Get("operation"),
		Limit:     intOr(q.Get("limit"), 100),
		From:      parseTime(q.Get("from")),
		To:        parseTime(q.Get("to")),
	}
	spans, err := s.store.QuerySpans(r.Context(), f)
	if err != nil {
		jsonErr(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, spans)
}

func (s *Server) HandleTraceDetail(w http.ResponseWriter, r *http.Request) {
	spans, err := s.store.QuerySpansByTraceID(r.Context(), chi.URLParam(r, "traceId"))
	if err != nil {
		jsonErr(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, spans)
}

func (s *Server) HandleTraceLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := s.store.QueryLogsByTraceID(r.Context(), chi.URLParam(r, "traceId"))
	if err != nil {
		jsonErr(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, logs)
}

func (s *Server) HandleMetrics(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := postgres.MetricFilter{
		Service: q.Get("service"),
		Name:    q.Get("name"),
		Type:    q.Get("type"),
		Limit:   intOr(q.Get("limit"), 200),
		From:    parseTime(q.Get("from")),
		To:      parseTime(q.Get("to")),
	}
	metrics, err := s.store.QueryMetrics(r.Context(), f)
	if err != nil {
		jsonErr(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, metrics)
}

func (s *Server) HandleMetricCatalog(w http.ResponseWriter, r *http.Request) {
	service := r.URL.Query().Get("service")
	data, err := s.store.QueryMetricCatalog(r.Context(), service)
	if err != nil {
		jsonErr(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, data)
}

func (s *Server) HandleMetricSeries(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := postgres.MetricSeriesFilter{
		Name:    q.Get("name"),
		Service: q.Get("service"),
		From:    parseTime(q.Get("from")),
		To:      parseTime(q.Get("to")),
	}
	if f.Name == "" {
		jsonErr(w, "name is required", http.StatusBadRequest)
		return
	}
	data, err := s.store.QueryMetricSeries(r.Context(), f)
	if err != nil {
		jsonErr(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, data)
}

func (s *Server) HandleLogs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := postgres.LogFilter{
		Service:  q.Get("service"),
		Severity: q.Get("severity"),
		Search:   q.Get("search"),
		Limit:    intOr(q.Get("limit"), 200),
		From:     parseTime(q.Get("from")),
		To:       parseTime(q.Get("to")),
	}
	logs, err := s.store.QueryLogs(r.Context(), f)
	if err != nil {
		jsonErr(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, logs)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func jsonOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data) //nolint:errcheck
}

func jsonErr(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg}) //nolint:errcheck
}

func intOr(s string, def int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

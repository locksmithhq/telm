package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
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
		Service:       q.Get("service"),
		Operation:     q.Get("operation"),
		Limit:         intOr(q.Get("limit"), 100),
		From:          parseTime(q.Get("from")),
		To:            parseTime(q.Get("to")),
		TraceID:       q.Get("trace_id"),
		StatusCodes:   parseInts(q.Get("status_codes")),
		Kinds:         parseInts(q.Get("kinds")),
		DurationMinMs: int64(intOr(q.Get("duration_min_ms"), 0)),
		DurationMaxMs: int64(intOr(q.Get("duration_max_ms"), 0)),
		MinSpanCount:  int64(intOr(q.Get("min_span_count"), 0)),
		Attributes:    parseAttrFilters(q),
	}
	if q.Get("has_error") == "true" {
		f.StatusCodes = []int{2}
	}
	spans, err := s.store.QuerySpans(r.Context(), f)
	if err != nil {
		jsonErr(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, spans)
}

func parseAttrFilters(q map[string][]string) []postgres.AttrFilter {
	var result []postgres.AttrFilter
	for i := 1; ; i++ {
		key := getFirst(q, fmt.Sprintf("attr_key_%d", i))
		if key == "" {
			key = getFirst(q, fmt.Sprintf("attr[%d].key", i))
		}
		if key == "" {
			break
		}
		value := getFirst(q, fmt.Sprintf("attr_value_%d", i))
		if value == "" {
			value = getFirst(q, fmt.Sprintf("attr[%d].value", i))
		}
		invert := getFirst(q, fmt.Sprintf("attr_invert_%d", i)) == "true" ||
			getFirst(q, fmt.Sprintf("attr[%d].invert", i)) == "true"
		result = append(result, postgres.AttrFilter{
			Key:    key,
			Value:  value,
			Invert: invert,
		})
	}
	return result
}

func getFirst(q map[string][]string, key string) string {
	if v, ok := q[key]; ok && len(v) > 0 {
		return v[0]
	}
	return ""
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
		Service:   q.Get("service"),
		Severity:  q.Get("severity"),
		Search:    q.Get("search"),
		Operation: q.Get("operation"),
		HasError:  q.Get("has_error") == "true",
		HasTrace:  q.Get("has_trace") == "true",
		Limit:     intOr(q.Get("limit"), 200),
		From:      parseTime(q.Get("from")),
		To:        parseTime(q.Get("to")),
	}
	for i := 1; ; i++ {
		k := fmt.Sprintf("attr_key_%d", i)
		if _, ok := q[k]; !ok {
			break
		}
		f.AttrFilters = append(f.AttrFilters, postgres.AttrFilter{
			Key:    q.Get(k),
			Value:  q.Get(fmt.Sprintf("attr_value_%d", i)),
			Invert: q.Get(fmt.Sprintf("attr_invert_%d", i)) == "true",
		})
	}
	logs, err := s.store.QueryLogs(r.Context(), f)
	if err != nil {
		jsonErr(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, logs)
}

func (s *Server) HandleDashboards(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	switch r.Method {
	case http.MethodGet:
		dashboards, err := s.store.ListDashboards(ctx)
		if err != nil {
			jsonErr(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonOK(w, dashboards)
	case http.MethodPost:
		var d postgres.Dashboard
		if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
			jsonErr(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if d.ID == "" || d.Name == "" {
			jsonErr(w, "id and name are required", http.StatusBadRequest)
			return
		}
		if err := s.store.CreateDashboard(ctx, &d); err != nil {
			jsonErr(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonOK(w, d)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	switch r.Method {
	case http.MethodGet:
		d, err := s.store.GetDashboard(ctx, id)
		if err != nil {
			if err == postgres.ErrNotFound {
				jsonErr(w, "not found", http.StatusNotFound)
				return
			}
			jsonErr(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonOK(w, d)
	case http.MethodPut:
		var d postgres.Dashboard
		if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
			jsonErr(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		d.ID = id
		if err := s.store.UpdateDashboard(ctx, &d); err != nil {
			if err == postgres.ErrNotFound {
				jsonErr(w, "not found", http.StatusNotFound)
				return
			}
			jsonErr(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonOK(w, d)
	case http.MethodDelete:
		if err := s.store.DeleteDashboard(ctx, id); err != nil {
			if err == postgres.ErrNotFound {
				jsonErr(w, "not found", http.StatusNotFound)
				return
			}
			jsonErr(w, err.Error(), http.StatusInternalServerError)
			return
		}
		jsonOK(w, map[string]string{"message": "deleted"})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
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

func parseInts(s string) []int {
	if s == "" {
		return nil
	}
	var result []int
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if n, err := strconv.Atoi(part); err == nil {
			result = append(result, n)
		}
	}
	return result
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

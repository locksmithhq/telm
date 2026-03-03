package api

import (
	"net/http"
	"time"

	"github.com/locksmithhq/telm/internal/storage/postgres"
)

func (s *Server) HandleThroughput(w http.ResponseWriter, r *http.Request) {
	f := statsFilter(r)
	data, err := s.store.QueryThroughput(r.Context(), f)
	if err != nil {
		jsonErr(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, data)
}

func (s *Server) HandleErrors(w http.ResponseWriter, r *http.Request) {
	f := statsFilter(r)
	data, err := s.store.QueryErrors(r.Context(), f)
	if err != nil {
		jsonErr(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, data)
}

func (s *Server) HandleLatency(w http.ResponseWriter, r *http.Request) {
	f := statsFilter(r)
	data, err := s.store.QueryLatency(r.Context(), f)
	if err != nil {
		jsonErr(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, data)
}

func (s *Server) HandleTopOps(w http.ResponseWriter, r *http.Request) {
	f := statsFilter(r)
	data, err := s.store.QueryTopOps(r.Context(), f)
	if err != nil {
		jsonErr(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, data)
}

func (s *Server) HandleSeverity(w http.ResponseWriter, r *http.Request) {
	f := statsFilter(r)
	data, err := s.store.QuerySeverityDist(r.Context(), f)
	if err != nil {
		jsonErr(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, data)
}

func (s *Server) HandleServiceHealth(w http.ResponseWriter, r *http.Request) {
	f := statsFilter(r)
	data, err := s.store.QueryServiceHealth(r.Context(), f)
	if err != nil {
		jsonErr(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, data)
}

func (s *Server) HandleServiceMap(w http.ResponseWriter, r *http.Request) {
	f := statsFilter(r)
	data, err := s.store.QueryServiceMap(r.Context(), f)
	if err != nil {
		jsonErr(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, data)
}

func (s *Server) HandleResources(w http.ResponseWriter, r *http.Request) {
	service := r.URL.Query().Get("service")
	if service == "" {
		jsonErr(w, "service is required", http.StatusBadRequest)
		return
	}
	f := statsFilter(r)
	data, err := s.store.QueryResourceSeries(r.Context(), service, f)
	if err != nil {
		jsonErr(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, data)
}

func (s *Server) HandleAllResources(w http.ResponseWriter, r *http.Request) {
	f := statsFilter(r)
	data, err := s.store.QueryAllResourceSeries(r.Context(), f)
	if err != nil {
		jsonErr(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, data)
}

func statsFilter(r *http.Request) postgres.StatsFilter {
	q := r.URL.Query()
	f := postgres.StatsFilter{
		Service: q.Get("service"),
		From:    parseTime(q.Get("from")),
		To:      parseTime(q.Get("to")),
	}
	if f.From.IsZero() {
		f.From = time.Now().Add(-1 * time.Hour)
	}
	if f.To.IsZero() {
		f.To = time.Now()
	}
	return f
}

package api

import (
	"net/http"
	"time"
)

func (s *Server) HandleStorageStats(w http.ResponseWriter, r *http.Request) {
	data, err := s.store.QueryStorageStats(r.Context())
	if err != nil {
		jsonErr(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, data)
}

func (s *Server) HandleStorageGrowth(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from := parseTime(q.Get("from"))
	to := parseTime(q.Get("to"))
	if from.IsZero() {
		from = time.Now().Add(-24 * time.Hour)
	}
	if to.IsZero() {
		to = time.Now()
	}
	data, err := s.store.QueryStorageGrowth(r.Context(), from, to)
	if err != nil {
		jsonErr(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, data)
}

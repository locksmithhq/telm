package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

// ── Key generation ────────────────────────────────────────────────────────────

func generateAPIKey() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return
	}
	raw = "telm_" + hex.EncodeToString(b)
	sum := sha256.Sum256([]byte(raw))
	hash = hex.EncodeToString(sum[:])
	return
}

func hashAPIKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// ── Middleware ────────────────────────────────────────────────────────────────

func (s *Server) APIKeyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.Header.Get("X-API-Key")
		if raw == "" {
			jsonErr(w, "missing X-API-Key header", http.StatusUnauthorized)
			return
		}
		_, err := s.store.FindAPIKeyByHash(r.Context(), hashAPIKey(raw))
		if err != nil {
			jsonErr(w, "invalid api key", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ── Handlers ─────────────────────────────────────────────────────────────────

func (s *Server) HandleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		jsonErr(w, "name is required", http.StatusBadRequest)
		return
	}
	raw, hash, err := generateAPIKey()
	if err != nil {
		jsonErr(w, "internal error", http.StatusInternalServerError)
		return
	}
	id, err := s.store.CreateAPIKey(r.Context(), body.Name, hash)
	if err != nil {
		jsonErr(w, "internal error", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]any{
		"id":  id,
		"key": raw, // returned only once
	})
}

func (s *Server) HandleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := s.store.ListAPIKeys(r.Context())
	if err != nil {
		jsonErr(w, "internal error", http.StatusInternalServerError)
		return
	}
	type item struct {
		ID          int64   `json:"id"`
		Name        string  `json:"name"`
		KeyPreview  string  `json:"key_preview"`
		CreatedAt   string  `json:"created_at"`
		LastUsedAt  *string `json:"last_used_at"`
	}
	out := make([]item, len(keys))
	for i, k := range keys {
		preview := k.KeyHash[:8] + "..."
		var lastUsed *string
		if k.LastUsedAt != nil {
			s := k.LastUsedAt.Format("2006-01-02T15:04:05Z07:00")
			lastUsed = &s
		}
		out[i] = item{
			ID:         k.ID,
			Name:       k.Name,
			KeyPreview: preview,
			CreatedAt:  k.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			LastUsedAt: lastUsed,
		}
	}
	jsonOK(w, out)
}

func (s *Server) HandleRevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonErr(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := s.store.RevokeAPIKey(r.Context(), id); err != nil {
		jsonErr(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── OTLP proxy ────────────────────────────────────────────────────────────────
// Proxies /otlp/* → http://localhost:4318/*
// stripping the /otlp prefix before forwarding.

func (s *Server) OTLPProxy() http.Handler {
	target, _ := url.Parse("http://localhost:4318")
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		jsonErr(w, fmt.Sprintf("collector unavailable: %v", err), http.StatusBadGateway)
	}

	// Drain and discard large bodies on error to allow connection reuse
	proxy.ModifyResponse = func(resp *http.Response) error {
		if resp.StatusCode >= 500 {
			io.Copy(io.Discard, resp.Body) //nolint:errcheck
		}
		return nil
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// strip /otlp prefix so /otlp/v1/traces → /v1/traces
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/otlp")
		if r.URL.Path == "" {
			r.URL.Path = "/"
		}
		r.URL.RawPath = strings.TrimPrefix(r.URL.RawPath, "/otlp")
		proxy.ServeHTTP(w, r)
	})
}

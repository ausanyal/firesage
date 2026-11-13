package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ausanyal/firesage/internal/engine"
)

// NewMux constructs the HTTP router serving both stateless /api/v1/* endpoints and embedded SPA assets.
func NewMux() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/health", handleHealth)
	mux.HandleFunc("GET /api/v1/metadata", handleMetadata)
	mux.HandleFunc("POST /api/v1/analyze", handleAnalyze)
	mux.HandleFunc("POST /api/v1/chat", handleChat)

	// Embedded React/TypeScript SPA handler with client-side route fallback
	mux.Handle("/", newSPAHandler())

	return withCORSAndSecurityHeaders(mux)
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"service":   "firesage",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func handleMetadata(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, engine.BuildMetadataPayload())
}

func handleAnalyze(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB max body
	var req engine.PlanInput
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid JSON request payload: " + err.Error(),
		})
		return
	}

	result := engine.RunFullAnalysis(req)
	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func withCORSAndSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

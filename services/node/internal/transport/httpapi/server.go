package httpapi

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/buildinfo"
	"github.com/YoLin02/yorva/services/node/internal/domain/node"
	"github.com/YoLin02/yorva/services/node/internal/events"
)

type HealthResponse struct {
	Status          string `json:"status"`
	Service         string `json:"service"`
	Version         string `json:"version"`
	ProtocolVersion string `json:"protocolVersion"`
}

func NewHandler(token string, localNode node.Node, broker *events.Broker) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", health)
	mux.Handle("GET /api/v1/node", requireBearer(token, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(localNode)
	})))
	mux.Handle("GET /api/v1/events", requireBearer(token, eventStream(broker, 15*time.Second)))
	return securityHeaders(restrictOrigins(routeContract(mux)))
}

func routeContract(next http.Handler) http.Handler {
	knownPaths := map[string]struct{}{
		"/api/v1/health": {},
		"/api/v1/node":   {},
		"/api/v1/events": {},
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := knownPaths[r.URL.Path]; !ok {
			writeError(w, http.StatusNotFound, ErrorBody{
				Code:      "NOT_FOUND",
				Message:   "The requested local API resource was not found.",
				Retryable: false,
			})
			return
		}
		if r.Method == http.MethodOptions {
			w.Header().Set("Allow", "GET, OPTIONS")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", "GET, OPTIONS")
			writeError(w, http.StatusMethodNotAllowed, ErrorBody{
				Code:      "METHOD_NOT_ALLOWED",
				Message:   "The request method is not allowed for this resource.",
				Retryable: false,
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(HealthResponse{
		Status:          "ok",
		Service:         buildinfo.Service,
		Version:         buildinfo.Version,
		ProtocolVersion: buildinfo.ProtocolVersion,
	})
}

func eventStream(broker *events.Broker, keepaliveInterval time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			writeError(w, http.StatusInternalServerError, ErrorBody{
				Code:      "INTERNAL_ERROR",
				Message:   "Event streaming is unavailable.",
				Retryable: true,
			})
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		_, _ = fmt.Fprint(w, ": connected\n\n")
		flusher.Flush()

		subscription := broker.Subscribe()
		defer subscription.Close()
		keepalive := time.NewTicker(keepaliveInterval)
		defer keepalive.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case event := <-subscription.Events:
				payload, err := json.Marshal(event)
				if err != nil {
					continue
				}
				if _, err := fmt.Fprintf(w, "id: %s\nevent: %s\ndata: %s\n\n", event.ID, event.Type, payload); err != nil {
					return
				}
				flusher.Flush()
			case <-keepalive.C:
				if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
					return
				}
				flusher.Flush()
			}
		}
	})
}

func requireBearer(expected string, next http.Handler) http.Handler {
	expectedHash := sha256.Sum256([]byte(expected))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scheme, supplied, ok := strings.Cut(r.Header.Get("Authorization"), " ")
		suppliedHash := sha256.Sum256([]byte(supplied))
		if !ok || !strings.EqualFold(scheme, "Bearer") || subtle.ConstantTimeCompare(expectedHash[:], suppliedHash[:]) != 1 {
			writeError(w, http.StatusUnauthorized, ErrorBody{
				Code:      "UNAUTHORIZED",
				Message:   "Authentication is required.",
				Retryable: false,
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

func restrictOrigins(next http.Handler) http.Handler {
	allowed := map[string]struct{}{
		"http://127.0.0.1:1420":  {},
		"http://tauri.localhost": {},
		"tauri://localhost":      {},
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if _, ok := allowed[origin]; !ok {
				writeError(w, http.StatusForbidden, ErrorBody{
					Code:      "ORIGIN_NOT_ALLOWED",
					Message:   "The request origin is not allowed.",
					Retryable: false,
				})
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Accept, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		}
		next.ServeHTTP(w, r)
	})
}

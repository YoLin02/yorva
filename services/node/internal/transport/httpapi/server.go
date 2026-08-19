package httpapi

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/app"
	"github.com/YoLin02/yorva/services/node/internal/buildinfo"
	"github.com/YoLin02/yorva/services/node/internal/domain/node"
	"github.com/YoLin02/yorva/services/node/internal/events"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type HealthResponse struct {
	Status          string `json:"status"`
	Service         string `json:"service"`
	Version         string `json:"version"`
	ProtocolVersion string `json:"protocolVersion"`
}

type RuntimeDiscoveryService interface {
	Detect(context.Context, yorvaruntime.Kind) (yorvaruntime.Discovery, error)
}

type RuntimeDiscoveryResponse struct {
	RuntimeKind    yorvaruntime.Kind           `json:"runtimeKind"`
	State          yorvaruntime.DiscoveryState `json:"state"`
	ErrorCode      *yorvaruntime.ErrorCode     `json:"errorCode"`
	Selected       *RuntimeCandidateResponse   `json:"selected"`
	Candidates     []RuntimeCandidateResponse  `json:"candidates"`
	Warnings       []RuntimeWarningResponse    `json:"warnings"`
	DetectedAt     time.Time                   `json:"detectedAt"`
	SupportedRange string                      `json:"supportedRange"`
}

type RuntimeCandidateResponse struct {
	Path      string                      `json:"path"`
	Version   string                      `json:"version"`
	State     yorvaruntime.DiscoveryState `json:"state"`
	ErrorCode *yorvaruntime.ErrorCode     `json:"errorCode"`
}

type RuntimeWarningResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewHandler(token string, localNode node.Node, broker *events.Broker, runtimes RuntimeDiscoveryService, installs RuntimeInstallService, instances InstanceInventoryService, dataDir string) http.Handler {
	mux := http.NewServeMux()
	models, _ := instances.(ModelConfigurationService)
	mux.HandleFunc("GET /api/v1/health", health)
	mux.Handle("GET /api/v1/node", requireBearer(token, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(localNode)
	})))
	mux.Handle("GET /api/v1/events", requireBearer(token, eventStream(broker, 15*time.Second)))
	mux.Handle("POST /api/v1/runtimes/{runtimeKind}/detect", requireBearer(token, detectRuntime(runtimes)))
	mux.Handle("POST /api/v1/runtimes/hermes/install", requireBearer(token, startHermesInstall(installs)))
	mux.Handle("GET /api/v1/runtimes/hermes/prerequisites", requireBearer(token, getHermesPrerequisites(installs)))
	mux.Handle("POST /api/v1/runtimes/hermes/prerequisites/install", requireBearer(token, startHermesPrerequisites(installs)))
	mux.Handle("GET /api/v1/runtimes/{runtimeId}/instances", requireBearer(token, listRuntimeInstances(instances)))
	mux.Handle("POST /api/v1/runtimes/{runtimeId}/instances", requireBearer(token, createRuntimeInstance(instances)))
	mux.Handle("GET /api/v1/runtimes/hermes/model-provider-presets", requireBearer(token, listModelProviderPresets(models)))
	mux.Handle("GET /api/v1/instances/{instanceId}", requireBearer(token, getInstance(instances)))
	mux.Handle("DELETE /api/v1/instances/{instanceId}", requireBearer(token, deleteInstance(instances)))
	mux.Handle("GET /api/v1/instances/{instanceId}/config", requireBearer(token, getModelConfiguration(models)))
	mux.Handle("PATCH /api/v1/instances/{instanceId}/config", requireBearer(token, patchModelConfiguration(models)))
	mux.Handle("GET /api/v1/instances/{instanceId}/credentials/model-provider", requireBearer(token, getModelCredential(models)))
	mux.Handle("PUT /api/v1/instances/{instanceId}/credentials/model-provider", requireBearer(token, putModelCredential(models)))
	mux.Handle("DELETE /api/v1/instances/{instanceId}/credentials/model-provider", requireBearer(token, deleteModelCredential(models)))
	mux.Handle("POST /api/v1/instances/{instanceId}/start", requireBearer(token, instanceLifecycleUnsupported()))
	mux.Handle("POST /api/v1/instances/{instanceId}/stop", requireBearer(token, instanceLifecycleUnsupported()))
	mux.Handle("POST /api/v1/instances/{instanceId}/restart", requireBearer(token, instanceLifecycleUnsupported()))
	mux.Handle("GET /api/v1/operations/{operationId}", requireBearer(token, getOperation(installs)))
	mux.Handle("GET /api/v1/operations/{operationId}/log", requireBearer(token, getOperationLog(installs, dataDir)))
	mux.Handle("GET /api/v1/operations", requireBearer(token, listOperations(installs)))
	mux.Handle("POST /api/v1/operations/{operationId}/cancel", requireBearer(token, cancelOperation(installs, instances)))
	return securityHeaders(restrictOrigins(routeContract(mux)))
}

func routeContract(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed, ok := allowedMethods(r.URL.Path)
		if !ok {
			writeError(w, http.StatusNotFound, ErrorBody{
				Code:      "NOT_FOUND",
				Message:   "The requested local API resource was not found.",
				Retryable: false,
			})
			return
		}
		if r.Method == http.MethodOptions {
			w.Header().Set("Allow", allowed)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if !methodAllowed(r.Method, allowed) {
			w.Header().Set("Allow", allowed)
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

func allowedMethods(path string) (string, bool) {
	switch path {
	case "/api/v1/health", "/api/v1/node", "/api/v1/events":
		return "GET, OPTIONS", true
	case "/api/v1/operations":
		return "GET, OPTIONS", true
	case "/api/v1/runtimes/hermes/install":
		return "POST, OPTIONS", true
	case "/api/v1/runtimes/hermes/prerequisites":
		return "GET, OPTIONS", true
	case "/api/v1/runtimes/hermes/prerequisites/install":
		return "POST, OPTIONS", true
	case "/api/v1/runtimes/hermes/model-provider-presets":
		return "GET, OPTIONS", true
	}
	if strings.HasPrefix(path, "/api/v1/operations/") {
		rest := strings.TrimPrefix(path, "/api/v1/operations/")
		if rest != "" && !strings.Contains(rest, "/") {
			return "GET, POST, OPTIONS", true
		}
		if strings.HasSuffix(rest, "/cancel") {
			id := strings.TrimSuffix(rest, "/cancel")
			if id != "" && !strings.Contains(id, "/") {
				return "POST, OPTIONS", true
			}
		}
		if strings.HasSuffix(rest, "/log") {
			id := strings.TrimSuffix(rest, "/log")
			if id != "" && !strings.Contains(id, "/") {
				return "GET, OPTIONS", true
			}
		}
	}
	const prefix = "/api/v1/runtimes/"
	const suffix = "/detect"
	if strings.HasPrefix(path, prefix) && strings.HasSuffix(path, suffix) {
		kind := strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix)
		if kind != "" && !strings.Contains(kind, "/") {
			return "POST, OPTIONS", true
		}
	}
	switch instancePathKind(path) {
	case "list":
		return "GET, POST, OPTIONS", true
	case "get":
		return "GET, DELETE, OPTIONS", true
	case "lifecycle":
		return "POST, OPTIONS", true
	case "config":
		return "GET, PATCH, OPTIONS", true
	case "model-credential":
		return "GET, PUT, DELETE, OPTIONS", true
	}
	return "", false
}

func methodAllowed(method, allowed string) bool {
	for _, candidate := range strings.Split(allowed, ", ") {
		if method == candidate {
			return true
		}
	}
	return false
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

func detectRuntime(runtimes RuntimeDiscoveryService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		discovery, err := runtimes.Detect(r.Context(), yorvaruntime.Kind(r.PathValue("runtimeKind")))
		if err != nil {
			if errors.Is(err, context.Canceled) && r.Context().Err() != nil {
				return
			}
			if errors.Is(err, app.ErrRuntimeKindNotFound) {
				writeError(w, http.StatusNotFound, ErrorBody{
					Code:      "RUNTIME_KIND_NOT_FOUND",
					Message:   "The requested Runtime kind is not registered.",
					Retryable: false,
				})
				return
			}
			writeError(w, http.StatusInternalServerError, ErrorBody{
				Code:      "RUNTIME_DISCOVERY_FAILED",
				Message:   "Runtime discovery could not be completed.",
				Retryable: true,
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(newRuntimeDiscoveryResponse(discovery))
	})
}

func newRuntimeDiscoveryResponse(discovery yorvaruntime.Discovery) RuntimeDiscoveryResponse {
	candidates := make([]RuntimeCandidateResponse, len(discovery.Candidates))
	for i, candidate := range discovery.Candidates {
		candidates[i] = newRuntimeCandidateResponse(candidate)
	}
	var selected *RuntimeCandidateResponse
	if discovery.Selected != nil {
		value := newRuntimeCandidateResponse(*discovery.Selected)
		selected = &value
	}
	warnings := make([]RuntimeWarningResponse, len(discovery.Warnings))
	for i, warning := range discovery.Warnings {
		warnings[i] = RuntimeWarningResponse{Code: warning.Code, Message: warning.Message}
	}
	return RuntimeDiscoveryResponse{
		RuntimeKind:    discovery.RuntimeKind,
		State:          discovery.State,
		ErrorCode:      nullableErrorCode(discovery.ErrorCode),
		Selected:       selected,
		Candidates:     candidates,
		Warnings:       warnings,
		DetectedAt:     discovery.DetectedAt,
		SupportedRange: discovery.SupportedRange,
	}
}

func newRuntimeCandidateResponse(candidate yorvaruntime.Candidate) RuntimeCandidateResponse {
	return RuntimeCandidateResponse{
		Path:      candidate.Path,
		Version:   candidate.Version,
		State:     candidate.State,
		ErrorCode: nullableErrorCode(candidate.ErrorCode),
	}
}

func nullableErrorCode(code yorvaruntime.ErrorCode) *yorvaruntime.ErrorCode {
	if code == "" {
		return nil
	}
	return &code
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
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Accept, Content-Type, Idempotency-Key")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		}
		next.ServeHTTP(w, r)
	})
}

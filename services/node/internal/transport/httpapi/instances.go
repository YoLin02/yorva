package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/app"
	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type InstanceInventoryService interface {
	ListInstances(context.Context, string) (app.InstanceList, error)
	GetInstance(context.Context, string) (app.InstanceView, error)
	StartCreate(context.Context, string, string, string) (app.InstallStartResult, error)
	CancelCreate(context.Context, string) (operation.Operation, error)
}

type InstanceCapabilitiesResponse struct {
	Instances bool `json:"instances"`
	Lifecycle bool `json:"lifecycle"`
}

type InstanceResponse struct {
	InstanceID            string                       `json:"instanceId"`
	RuntimeInstallationID string                       `json:"runtimeInstallationId"`
	Name                  string                       `json:"name"`
	Default               bool                         `json:"default"`
	Protected             bool                         `json:"protected"`
	Availability          string                       `json:"availability"`
	LastSyncedAt          *time.Time                   `json:"lastSyncedAt"`
	CreatedAt             time.Time                    `json:"createdAt"`
	UpdatedAt             time.Time                    `json:"updatedAt"`
	Capabilities          InstanceCapabilitiesResponse `json:"capabilities"`
}

type InstanceListResponse struct {
	RuntimeID             string                       `json:"runtimeId"`
	RuntimeInstallationID string                       `json:"runtimeInstallationId"`
	Freshness             string                       `json:"freshness"`
	LastSyncedAt          *time.Time                   `json:"lastSyncedAt"`
	Instances             []InstanceResponse           `json:"instances"`
	Capabilities          InstanceCapabilitiesResponse `json:"capabilities"`
	ErrorCode             *yorvaruntime.ErrorCode      `json:"errorCode"`
}

func listRuntimeInstances(inventory InstanceInventoryService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if inventory == nil {
			writeError(w, http.StatusInternalServerError, ErrorBody{Code: "INTERNAL_ERROR", Message: "Instance inventory is unavailable.", Retryable: true})
			return
		}
		result, err := inventory.ListInstances(r.Context(), r.PathValue("runtimeId"))
		if err != nil {
			writeInstanceError(w, err)
			return
		}
		items := make([]InstanceResponse, 0, len(result.Instances))
		for _, item := range result.Instances {
			items = append(items, newInstanceResponse(item))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(InstanceListResponse{
			RuntimeID:             result.RuntimeID,
			RuntimeInstallationID: result.RuntimeInstallationID,
			Freshness:             result.Freshness,
			LastSyncedAt:          result.LastSyncedAt,
			Instances:             items,
			Capabilities:          InstanceCapabilitiesResponse{Instances: result.Capabilities.Instances, Lifecycle: false},
			ErrorCode:             nullableErrorCode(result.ErrorCode),
		})
	})
}

func createRuntimeInstance(inventory InstanceInventoryService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if inventory == nil {
			writeError(w, http.StatusInternalServerError, ErrorBody{Code: "INTERNAL_ERROR", Message: "Instance inventory is unavailable.", Retryable: true})
			return
		}
		key := r.Header.Get("Idempotency-Key")
		if err := app.ValidateIdempotencyKey(key); err != nil {
			writeError(w, http.StatusBadRequest, ErrorBody{Code: "INVALID_IDEMPOTENCY_KEY", Message: "A valid Idempotency-Key header is required.", Retryable: false})
			return
		}
		name, err := decodeClosedInstanceName(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, ErrorBody{Code: "INVALID_REQUEST", Message: "The create request must be a closed JSON object with name.", Retryable: false})
			return
		}
		result, err := inventory.StartCreate(r.Context(), r.PathValue("runtimeId"), name, key)
		if err != nil {
			writeInstanceError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(newOperationResponse(result.Operation))
	})
}

func getInstance(inventory InstanceInventoryService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if inventory == nil {
			writeError(w, http.StatusInternalServerError, ErrorBody{Code: "INTERNAL_ERROR", Message: "Instance inventory is unavailable.", Retryable: true})
			return
		}
		result, err := inventory.GetInstance(r.Context(), r.PathValue("instanceId"))
		if err != nil {
			writeInstanceError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(newInstanceResponse(result))
	})
}

func instanceLifecycleUnsupported() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusConflict, ErrorBody{
			Code:      string(yorvaruntime.ErrorCapabilityNotSupported),
			Message:   "Instance lifecycle is not available in this phase.",
			Retryable: false,
		})
	})
}

func newInstanceResponse(item app.InstanceView) InstanceResponse {
	return InstanceResponse{
		InstanceID:            item.InstanceID,
		RuntimeInstallationID: item.RuntimeInstallationID,
		Name:                  item.Name,
		Default:               item.Default,
		Protected:             item.Protected,
		Availability:          string(item.Availability),
		LastSyncedAt:          item.LastSyncedAt,
		CreatedAt:             item.CreatedAt,
		UpdatedAt:             item.UpdatedAt,
		Capabilities:          InstanceCapabilitiesResponse{Instances: true, Lifecycle: false},
	}
}

func writeInstanceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, app.ErrInstanceRuntimeNotFound), errors.Is(err, app.ErrInstanceNotFound):
		writeError(w, http.StatusNotFound, ErrorBody{Code: string(yorvaruntime.ErrorInstanceNotFound), Message: "The requested instance was not found.", Retryable: false})
	case errors.Is(err, app.ErrRuntimeNotSupported), errors.Is(err, app.ErrRuntimeKindNotFound):
		writeError(w, http.StatusConflict, ErrorBody{Code: string(yorvaruntime.ErrorRuntimeNotSupported), Message: "A supported Hermes installation is required to manage instances.", Retryable: false})
	case errors.Is(err, app.ErrInstanceInvalidName):
		writeError(w, http.StatusBadRequest, ErrorBody{Code: string(yorvaruntime.ErrorInstanceInvalidName), Message: "The instance name is not allowed.", Retryable: false})
	case errors.Is(err, app.ErrInstanceAlreadyExists):
		writeError(w, http.StatusConflict, ErrorBody{Code: string(yorvaruntime.ErrorInstanceAlreadyExists), Message: "An instance with this name already exists.", Retryable: false})
	case errors.Is(err, app.ErrInstanceConflict):
		writeError(w, http.StatusConflict, ErrorBody{Code: string(yorvaruntime.ErrorInstanceConflict), Message: "Another instance operation is already running.", Retryable: false})
	case errors.Is(err, app.ErrInstanceNotCancellable):
		writeError(w, http.StatusConflict, ErrorBody{Code: string(yorvaruntime.ErrorOperationNotCancellable), Message: "This instance operation can no longer be cancelled.", Retryable: false})
	case errors.Is(err, context.Canceled):
		return
	default:
		writeError(w, http.StatusInternalServerError, ErrorBody{Code: "INTERNAL_ERROR", Message: "Instance inventory could not be completed.", Retryable: true})
	}
}

func decodeClosedInstanceName(r *http.Request) (string, error) {
	if r.Body == nil {
		return "", io.EOF
	}
	defer r.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(r.Body, maxInstallRequestBytes+1))
	if err != nil {
		return "", err
	}
	if len(payload) == 0 || len(payload) > maxInstallRequestBytes {
		return "", io.ErrUnexpectedEOF
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var body struct {
		Name string `json:"name"`
	}
	if err := decoder.Decode(&body); err != nil {
		return "", err
	}
	if body.Name == "" {
		return "", errors.New("name required")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return "", errors.New("trailing json")
	}
	return body.Name, nil
}

func instancePathKind(path string) string {
	if strings.HasPrefix(path, "/api/v1/runtimes/") && strings.HasSuffix(path, "/instances") {
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/api/v1/runtimes/"), "/instances")
		if id != "" && !strings.Contains(id, "/") {
			return "list"
		}
	}
	if strings.HasPrefix(path, "/api/v1/instances/") {
		rest := strings.TrimPrefix(path, "/api/v1/instances/")
		if rest != "" && !strings.Contains(rest, "/") {
			return "get"
		}
		for _, action := range []string{"/start", "/stop", "/restart"} {
			if strings.HasSuffix(rest, action) {
				id := strings.TrimSuffix(rest, action)
				if id != "" && !strings.Contains(id, "/") {
					return "lifecycle"
				}
			}
		}
	}
	return ""
}

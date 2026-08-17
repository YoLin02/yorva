package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/YoLin02/yorva/services/node/internal/app"
	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type RuntimeInstallService interface {
	Start(context.Context, yorvaruntime.Kind, string) (app.InstallStartResult, error)
	Get(context.Context, string) (operation.Operation, error)
	List(context.Context, string, string, int) ([]operation.Operation, error)
	Cancel(context.Context, string) (operation.Operation, error)
}

type OperationResponse struct {
	ID            string     `json:"id"`
	Type          string     `json:"type"`
	TargetType    string     `json:"targetType"`
	TargetID      string     `json:"targetId"`
	Status        string     `json:"status"`
	Stage         string     `json:"stage"`
	Progress      *int       `json:"progress"`
	ErrorCode     *string    `json:"errorCode"`
	Retryable     bool       `json:"retryable"`
	CorrelationID string     `json:"correlationId"`
	CreatedAt     time.Time  `json:"createdAt"`
	StartedAt     *time.Time `json:"startedAt"`
	CompletedAt   *time.Time `json:"completedAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

type OperationListResponse struct {
	Operations []OperationResponse `json:"operations"`
}

func startHermesInstall(installs RuntimeInstallService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if installs == nil {
			writeError(w, http.StatusNotFound, ErrorBody{Code: "NOT_FOUND", Message: "The requested local API resource was not found."})
			return
		}
		key := r.Header.Get("Idempotency-Key")
		if err := app.ValidateIdempotencyKey(key); err != nil {
			writeError(w, http.StatusBadRequest, ErrorBody{Code: "INVALID_IDEMPOTENCY_KEY", Message: "A valid Idempotency-Key header is required.", Retryable: false})
			return
		}
		if r.Body != nil {
			defer r.Body.Close()
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			var body map[string]any
			if err := decoder.Decode(&body); err != nil {
				if errors.Is(err, io.EOF) {
					body = map[string]any{}
				} else {
					writeError(w, http.StatusBadRequest, ErrorBody{Code: "INVALID_REQUEST", Message: "The install request must be a closed JSON object.", Retryable: false})
					return
				}
			}
			if len(body) != 0 {
				writeError(w, http.StatusBadRequest, ErrorBody{Code: "INVALID_REQUEST", Message: "The install request must not include version, path, URL or command fields.", Retryable: false})
				return
			}
		}
		result, err := installs.Start(r.Context(), "hermes", key)
		if err != nil {
			writeInstallError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(newOperationResponse(result.Operation))
	})
}

func getOperation(installs RuntimeInstallService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if installs == nil {
			writeError(w, http.StatusNotFound, ErrorBody{Code: "NOT_FOUND", Message: "The requested local API resource was not found."})
			return
		}
		value, err := installs.Get(r.Context(), r.PathValue("operationId"))
		if err != nil {
			writeError(w, http.StatusNotFound, ErrorBody{Code: "NOT_FOUND", Message: "The requested operation was not found."})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(newOperationResponse(value))
	})
}

func listOperations(installs RuntimeInstallService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if installs == nil {
			writeError(w, http.StatusNotFound, ErrorBody{Code: "NOT_FOUND", Message: "The requested local API resource was not found."})
			return
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		values, err := installs.List(r.Context(), r.URL.Query().Get("targetType"), r.URL.Query().Get("targetId"), limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, ErrorBody{Code: "INTERNAL_ERROR", Message: "Operations could not be listed.", Retryable: true})
			return
		}
		response := OperationListResponse{Operations: make([]OperationResponse, 0, len(values))}
		for _, value := range values {
			response.Operations = append(response.Operations, newOperationResponse(value))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
}

func cancelOperation(installs RuntimeInstallService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if installs == nil {
			writeError(w, http.StatusNotFound, ErrorBody{Code: "NOT_FOUND", Message: "The requested local API resource was not found."})
			return
		}
		value, err := installs.Cancel(r.Context(), r.PathValue("operationId"))
		if err != nil {
			writeInstallError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(newOperationResponse(value))
	})
}

func newOperationResponse(value operation.Operation) OperationResponse {
	var errorCode *string
	if value.ErrorCode != "" {
		code := string(value.ErrorCode)
		errorCode = &code
	}
	return OperationResponse{
		ID:            value.ID,
		Type:          string(value.Type),
		TargetType:    string(value.TargetType),
		TargetID:      value.TargetID,
		Status:        string(value.Status),
		Stage:         string(value.Stage),
		Progress:      nil,
		ErrorCode:     errorCode,
		Retryable:     value.Retryable,
		CorrelationID: value.CorrelationID,
		CreatedAt:     value.CreatedAt,
		StartedAt:     value.StartedAt,
		CompletedAt:   value.CompletedAt,
		UpdatedAt:     value.UpdatedAt,
	}
}

func writeInstallError(w http.ResponseWriter, err error) {
	var rejected app.InstallRejection
	if errors.As(err, &rejected) {
		status := http.StatusConflict
		if rejected.Code == yorvaruntime.ErrorOperationNotCancellable {
			status = http.StatusConflict
		}
		if rejected.Code == yorvaruntime.ErrorRuntimeInstallPlatformUnsupported {
			status = http.StatusBadRequest
		}
		details := map[string]any{}
		if rejected.ActiveID != "" {
			details["operationId"] = rejected.ActiveID
		}
		writeError(w, status, ErrorBody{
			Code:      string(rejected.Code),
			Message:   "The Runtime installation request was rejected.",
			Retryable: rejected.Retryable,
			Details:   details,
		})
		return
	}
	if errors.Is(err, app.ErrInvalidIdempotencyKey) {
		writeError(w, http.StatusBadRequest, ErrorBody{Code: "INVALID_IDEMPOTENCY_KEY", Message: "A valid Idempotency-Key header is required."})
		return
	}
	if errors.Is(err, app.ErrRuntimeKindNotFound) {
		writeError(w, http.StatusNotFound, ErrorBody{Code: "RUNTIME_KIND_NOT_FOUND", Message: "The requested Runtime kind is not registered."})
		return
	}
	writeError(w, http.StatusInternalServerError, ErrorBody{Code: "INTERNAL_ERROR", Message: "The Runtime installation request failed.", Retryable: true})
}

func visibleASCII(value string) bool {
	if value == "" || !utf8.ValidString(value) {
		return false
	}
	return !strings.ContainsFunc(value, func(r rune) bool { return r < 33 || r > 126 })
}

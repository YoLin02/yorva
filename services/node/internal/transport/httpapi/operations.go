package httpapi

import (
	"bytes"
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
	"github.com/YoLin02/yorva/services/node/internal/applog"
	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type RuntimeInstallService interface {
	Start(context.Context, yorvaruntime.Kind, string) (app.InstallStartResult, error)
	StartPrerequisites(context.Context, string) (app.InstallStartResult, error)
	InspectPrerequisites(context.Context) (app.PrerequisiteSnapshot, error)
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
	Message       string     `json:"message"`
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

type OperationLogResponse struct {
	OperationID   string `json:"operationId"`
	CorrelationID string `json:"correlationId"`
	Text          string `json:"text"`
}

type PrerequisiteResponse struct {
	Node              PrerequisiteComponent `json:"node"`
	NPM               PrerequisiteComponent `json:"npm"`
	NodeDependencies  PrerequisiteComponent `json:"nodeDependencies"`
	CheckedAt         time.Time             `json:"checkedAt"`
	ActiveOperationID *string               `json:"activeOperationId"`
}

type PrerequisiteComponent struct {
	State     string  `json:"state"`
	Version   string  `json:"version"`
	ErrorCode *string `json:"errorCode"`
	Retryable bool    `json:"retryable"`
}

func getHermesPrerequisites(installs RuntimeInstallService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if installs == nil {
			writeError(w, http.StatusNotFound, ErrorBody{Code: "NOT_FOUND", Message: "The requested local API resource was not found."})
			return
		}
		snap, err := installs.InspectPrerequisites(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, ErrorBody{Code: "INTERNAL_ERROR", Message: "Hermes prerequisites could not be inspected.", Retryable: true})
			return
		}
		var active *string
		if snap.ActiveOperationID != "" {
			id := snap.ActiveOperationID
			active = &id
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(PrerequisiteResponse{
			Node:              prereqComponent(snap.NodeState, snap.NodeVersion, snap.NodeCode, snap.Retryable),
			NPM:               prereqComponent(snap.NPMState, snap.NPMVersion, snap.NPMCode, snap.Retryable),
			NodeDependencies:  prereqComponent(snap.DepsState, "", snap.DepsCode, snap.Retryable),
			CheckedAt:         snap.CheckedAt,
			ActiveOperationID: active,
		})
	})
}

func startHermesPrerequisites(installs RuntimeInstallService) http.Handler {
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
		if err := decodeClosedEmptyObject(r); err != nil {
			writeError(w, http.StatusBadRequest, ErrorBody{Code: "INVALID_REQUEST", Message: "The install request must be a closed JSON object.", Retryable: false})
			return
		}
		result, err := installs.StartPrerequisites(r.Context(), key)
		if err != nil {
			writeInstallError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(newOperationResponse(result.Operation))
	})
}

func prereqComponent(state, version string, code yorvaruntime.ErrorCode, retryable bool) PrerequisiteComponent {
	var errorCode *string
	if code != "" {
		value := string(code)
		errorCode = &value
	}
	return PrerequisiteComponent{State: state, Version: version, ErrorCode: errorCode, Retryable: retryable}
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
		if err := decodeClosedEmptyObject(r); err != nil {
			writeError(w, http.StatusBadRequest, ErrorBody{Code: "INVALID_REQUEST", Message: "The install request must be a closed JSON object.", Retryable: false})
			return
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

func getOperationLog(installs RuntimeInstallService, dataDir string) http.Handler {
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
		text := applog.ReadMatching(dataDir, value.ID, 96*1024)
		if text == "" && value.CorrelationID != "" {
			text = applog.ReadMatching(dataDir, value.CorrelationID, 96*1024)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(OperationLogResponse{
			OperationID:   value.ID,
			CorrelationID: value.CorrelationID,
			Text:          text,
		})
	})
}

func cancelOperation(installs RuntimeInstallService, instances InstanceInventoryService, models ModelConfigurationService) http.Handler {
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
		if value.Type == operation.TypeInstanceCreate && instances != nil {
			value, err = instances.CancelCreate(r.Context(), value.ID)
		} else if value.Type == operation.TypeInstanceDelete && instances != nil {
			value, err = instances.CancelDelete(r.Context(), value.ID)
		} else if value.Type == operation.TypeModelValidate && models != nil {
			value, err = models.CancelModelValidation(r.Context(), value.ID)
		} else {
			value, err = installs.Cancel(r.Context(), value.ID)
		}
		if err != nil {
			if errors.Is(err, app.ErrInstanceNotCancellable) {
				writeInstanceError(w, err)
				return
			}
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
		Message:       value.Message,
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
		if rejected.Code == yorvaruntime.ErrorRuntimeInstallNotReady {
			status = http.StatusServiceUnavailable
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

const maxInstallRequestBytes = 4096

func decodeClosedEmptyObject(r *http.Request) error {
	if r.Body == nil {
		return io.EOF
	}
	defer r.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(r.Body, maxInstallRequestBytes+1))
	if err != nil {
		return err
	}
	if len(payload) == 0 || len(payload) > maxInstallRequestBytes {
		return io.ErrUnexpectedEOF
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var body map[string]any
	if err := decoder.Decode(&body); err != nil {
		return err
	}
	if body == nil || len(body) != 0 {
		return errors.New("closed empty object required")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("trailing json")
	}
	return nil
}

func visibleASCII(value string) bool {
	if value == "" || !utf8.ValidString(value) {
		return false
	}
	return !strings.ContainsFunc(value, func(r rune) bool { return r < 33 || r > 126 })
}

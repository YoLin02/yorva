package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/app"
	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type SharedModelService interface {
	ListModelProviderConnections(context.Context, string) ([]app.ModelProviderConnectionView, error)
	CreateModelProviderConnection(context.Context, string, string, string, []byte) (app.ModelProviderConnectionView, error)
	DeleteModelProviderConnection(context.Context, string, string) error
	ListModelProfiles(context.Context, string) ([]app.ModelProfileView, error)
	CreateModelProfile(context.Context, string, string, string, string, []string) (app.ModelProfileView, error)
	DeleteModelProfile(context.Context, string, string) error
	GetRuntimeModelDefault(context.Context, string) (app.RuntimeModelDefaultView, error)
	SetRuntimeModelDefault(context.Context, string, string) (app.RuntimeModelDefaultView, error)
	ClearRuntimeModelDefault(context.Context, string) error
	ListInstanceModelBindings(context.Context, string) ([]app.InstanceModelBindingView, error)
	StartModelProfileApplication(context.Context, string, string, []string, string, string) (app.InstallStartResult, error)
	CancelModelProfileApplication(context.Context, string) (operation.Operation, error)
}

type ModelProviderConnectionResponse struct {
	ID               string    `json:"id"`
	ProviderPresetID string    `json:"providerPresetId"`
	DisplayName      string    `json:"displayName"`
	CredentialSet    bool      `json:"credentialConfigured"`
	Status           string    `json:"status"`
	Revision         int       `json:"revision"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type ModelProfileResponse struct {
	ID                   string    `json:"id"`
	ProviderConnectionID string    `json:"providerConnectionId"`
	DisplayName          string    `json:"displayName"`
	SelectedModelIDs     []string  `json:"selectedModelIds"`
	DefaultModelID       string    `json:"defaultModelId"`
	Revision             int       `json:"revision"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

type RuntimeModelDefaultResponse struct {
	ModelProfileID  string     `json:"modelProfileId"`
	AppliedRevision int        `json:"appliedRevision"`
	UpdatedAt       *time.Time `json:"updatedAt"`
}

type InstanceModelBindingResponse struct {
	InstanceID      string                  `json:"instanceId"`
	InstanceName    string                  `json:"instanceName"`
	ModelProfileID  string                  `json:"modelProfileId"`
	Mode            string                  `json:"mode"`
	AppliedRevision int                     `json:"appliedRevision"`
	State           string                  `json:"state"`
	ErrorCode       *yorvaruntime.ErrorCode `json:"errorCode"`
	UpdatedAt       time.Time               `json:"updatedAt"`
}

func listModelProviderConnections(service SharedModelService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !sharedModelAvailable(w, service) {
			return
		}
		items, err := service.ListModelProviderConnections(r.Context(), r.PathValue("runtimeId"))
		if err != nil {
			writeSharedModelError(w, err)
			return
		}
		result := make([]ModelProviderConnectionResponse, 0, len(items))
		for _, item := range items {
			result = append(result, modelProviderConnectionResponse(item))
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": result})
	})
}

func createModelProviderConnection(service SharedModelService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !sharedModelAvailable(w, service) {
			return
		}
		presetID, displayName, secret, err := decodeModelProviderConnection(r)
		if err != nil {
			writeSharedModelError(w, app.ErrModelResourceInvalid)
			return
		}
		defer clearBytes(secret)
		value, err := service.CreateModelProviderConnection(r.Context(), r.PathValue("runtimeId"), presetID, displayName, secret)
		if err != nil {
			writeSharedModelError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, modelProviderConnectionResponse(value))
	})
}

func deleteModelProviderConnection(service SharedModelService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !sharedModelAvailable(w, service) {
			return
		}
		if err := service.DeleteModelProviderConnection(r.Context(), r.PathValue("runtimeId"), r.PathValue("connectionId")); err != nil {
			writeSharedModelError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func listModelProfiles(service SharedModelService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !sharedModelAvailable(w, service) {
			return
		}
		items, err := service.ListModelProfiles(r.Context(), r.PathValue("runtimeId"))
		if err != nil {
			writeSharedModelError(w, err)
			return
		}
		result := make([]ModelProfileResponse, 0, len(items))
		for _, item := range items {
			result = append(result, modelProfileResponse(item))
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": result})
	})
}

func createModelProfile(service SharedModelService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !sharedModelAvailable(w, service) {
			return
		}
		connectionID, displayName, defaultModelID, selectedModelIDs, err := decodeModelProfile(r)
		if err != nil {
			writeSharedModelError(w, app.ErrModelResourceInvalid)
			return
		}
		value, err := service.CreateModelProfile(r.Context(), r.PathValue("runtimeId"), connectionID, displayName, defaultModelID, selectedModelIDs)
		if err != nil {
			writeSharedModelError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, modelProfileResponse(value))
	})
}

func deleteModelProfile(service SharedModelService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !sharedModelAvailable(w, service) {
			return
		}
		if err := service.DeleteModelProfile(r.Context(), r.PathValue("runtimeId"), r.PathValue("profileId")); err != nil {
			writeSharedModelError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func getRuntimeModelDefault(service SharedModelService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !sharedModelAvailable(w, service) {
			return
		}
		value, err := service.GetRuntimeModelDefault(r.Context(), r.PathValue("runtimeId"))
		if err != nil {
			writeSharedModelError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, runtimeModelDefaultResponse(value))
	})
}

func putRuntimeModelDefault(service SharedModelService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !sharedModelAvailable(w, service) {
			return
		}
		profileID, err := decodeSingleString(r, "modelProfileId", 4096)
		if err != nil {
			writeSharedModelError(w, app.ErrModelResourceInvalid)
			return
		}
		value, err := service.SetRuntimeModelDefault(r.Context(), r.PathValue("runtimeId"), profileID)
		if err != nil {
			writeSharedModelError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, runtimeModelDefaultResponse(value))
	})
}

func deleteRuntimeModelDefault(service SharedModelService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !sharedModelAvailable(w, service) {
			return
		}
		if err := service.ClearRuntimeModelDefault(r.Context(), r.PathValue("runtimeId")); err != nil {
			writeSharedModelError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func listInstanceModelBindings(service SharedModelService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !sharedModelAvailable(w, service) {
			return
		}
		items, err := service.ListInstanceModelBindings(r.Context(), r.PathValue("runtimeId"))
		if err != nil {
			writeSharedModelError(w, err)
			return
		}
		result := make([]InstanceModelBindingResponse, 0, len(items))
		for _, item := range items {
			result = append(result, InstanceModelBindingResponse{
				InstanceID: item.InstanceID, InstanceName: item.InstanceName, ModelProfileID: item.ModelProfileID,
				Mode: item.Mode, AppliedRevision: item.AppliedRevision, State: item.State,
				ErrorCode: nullableErrorCode(item.ErrorCode), UpdatedAt: item.UpdatedAt,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": result})
	})
}

func startModelProfileApplication(service SharedModelService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !sharedModelAvailable(w, service) {
			return
		}
		profileID, instanceIDs, mode, err := decodeModelProfileApplication(r)
		if err != nil {
			writeSharedModelError(w, app.ErrModelResourceInvalid)
			return
		}
		result, err := service.StartModelProfileApplication(r.Context(), r.PathValue("runtimeId"), profileID, instanceIDs, mode, r.Header.Get("Idempotency-Key"))
		if err != nil {
			writeSharedModelError(w, err)
			return
		}
		writeJSON(w, http.StatusAccepted, newOperationResponse(result.Operation))
	})
}

func modelProviderConnectionResponse(value app.ModelProviderConnectionView) ModelProviderConnectionResponse {
	return ModelProviderConnectionResponse{
		ID: value.ID, ProviderPresetID: value.ProviderPresetID, DisplayName: value.DisplayName,
		CredentialSet: value.CredentialSet, Status: value.Status, Revision: value.Revision,
		CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt,
	}
}

func modelProfileResponse(value app.ModelProfileView) ModelProfileResponse {
	return ModelProfileResponse{
		ID: value.ID, ProviderConnectionID: value.ProviderConnectionID, DisplayName: value.DisplayName,
		SelectedModelIDs: append([]string(nil), value.SelectedModelIDs...), DefaultModelID: value.DefaultModelID,
		Revision: value.Revision, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt,
	}
}

func runtimeModelDefaultResponse(value app.RuntimeModelDefaultView) RuntimeModelDefaultResponse {
	result := RuntimeModelDefaultResponse{ModelProfileID: value.ModelProfileID, AppliedRevision: value.AppliedRevision}
	if !value.UpdatedAt.IsZero() {
		result.UpdatedAt = &value.UpdatedAt
	}
	return result
}

func decodeModelProviderConnection(r *http.Request) (string, string, []byte, error) {
	var body struct {
		ProviderPresetID string `json:"providerPresetId"`
		DisplayName      string `json:"displayName"`
		Credential       string `json:"credential"`
	}
	if err := decodeClosedJSON(r, 128*1024, &body); err != nil || body.ProviderPresetID == "" || body.DisplayName == "" || body.Credential == "" {
		return "", "", nil, app.ErrModelResourceInvalid
	}
	secret := []byte(body.Credential)
	body.Credential = ""
	return body.ProviderPresetID, body.DisplayName, secret, nil
}

func decodeModelProfile(r *http.Request) (string, string, string, []string, error) {
	var body struct {
		ProviderConnectionID string   `json:"providerConnectionId"`
		DisplayName          string   `json:"displayName"`
		DefaultModelID       string   `json:"defaultModelId"`
		SelectedModelIDs     []string `json:"selectedModelIds"`
	}
	if err := decodeClosedJSON(r, 16*1024, &body); err != nil || body.ProviderConnectionID == "" || body.DisplayName == "" || body.DefaultModelID == "" || len(body.SelectedModelIDs) == 0 {
		return "", "", "", nil, app.ErrModelResourceInvalid
	}
	return body.ProviderConnectionID, body.DisplayName, body.DefaultModelID, body.SelectedModelIDs, nil
}

func decodeModelProfileApplication(r *http.Request) (string, []string, string, error) {
	var body struct {
		ModelProfileID string   `json:"modelProfileId"`
		InstanceIDs    []string `json:"instanceIds"`
		Mode           string   `json:"mode"`
	}
	if err := decodeClosedJSON(r, 16*1024, &body); err != nil || body.ModelProfileID == "" || len(body.InstanceIDs) == 0 || (body.Mode != "INHERIT" && body.Mode != "OVERRIDE") {
		return "", nil, "", app.ErrModelResourceInvalid
	}
	return body.ModelProfileID, body.InstanceIDs, body.Mode, nil
}

func decodeSingleString(r *http.Request, field string, limit int64) (string, error) {
	if r.Body == nil {
		return "", app.ErrModelResourceInvalid
	}
	defer r.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(r.Body, limit+1))
	if err != nil || len(payload) == 0 || int64(len(payload)) > limit {
		return "", app.ErrModelResourceInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	value := map[string]string{}
	if err := decoder.Decode(&value); err != nil || len(value) != 1 || value[field] == "" {
		return "", app.ErrModelResourceInvalid
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return "", app.ErrModelResourceInvalid
	}
	return value[field], nil
}

func decodeClosedJSON(r *http.Request, limit int64, target any) error {
	if r.Body == nil {
		return io.EOF
	}
	defer r.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(r.Body, limit+1))
	if err != nil || len(payload) == 0 || int64(len(payload)) > limit {
		clearBytes(payload)
		return io.ErrUnexpectedEOF
	}
	defer clearBytes(payload)
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("trailing json")
	}
	return nil
}

func writeSharedModelError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, app.ErrModelResourceInvalid):
		writeError(w, http.StatusBadRequest, ErrorBody{Code: string(yorvaruntime.ErrorModelConfigInvalid), Message: "The shared model request is invalid.", Retryable: false})
	case errors.Is(err, app.ErrModelResourceNotFound):
		writeError(w, http.StatusNotFound, ErrorBody{Code: "MODEL_RESOURCE_NOT_FOUND", Message: "The requested model resource was not found.", Retryable: false})
	case errors.Is(err, app.ErrModelResourceConflict), errors.Is(err, yorvaruntime.ErrInstanceConfigConflict):
		writeError(w, http.StatusConflict, ErrorBody{Code: string(yorvaruntime.ErrorInstanceConfigConflict), Message: "The model resource is still in use or changed concurrently.", Retryable: true})
	case errors.Is(err, app.ErrModelSecretUnavailable):
		writeError(w, http.StatusServiceUnavailable, ErrorBody{Code: string(yorvaruntime.ErrorModelCredentialWriteFailed), Message: "The protected Provider credential is unavailable.", Retryable: true})
	case errors.Is(err, app.ErrRuntimeNotSupported):
		writeError(w, http.StatusConflict, ErrorBody{Code: string(yorvaruntime.ErrorRuntimeNotSupported), Message: "Shared models are unavailable for this Runtime.", Retryable: false})
	case errors.Is(err, context.Canceled):
		return
	default:
		writeError(w, http.StatusInternalServerError, ErrorBody{Code: "INTERNAL_ERROR", Message: "Shared model management could not be completed.", Retryable: true})
	}
}

func sharedModelAvailable(w http.ResponseWriter, service SharedModelService) bool {
	if service != nil {
		return true
	}
	writeSharedModelError(w, app.ErrRuntimeNotSupported)
	return false
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

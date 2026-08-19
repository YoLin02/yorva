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
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const maxModelConfigRequestBytes = 4096

type ModelConfigurationService interface {
	ListModelProviderPresets(context.Context) ([]yorvaruntime.ModelProviderPreset, error)
	GetModelConfiguration(context.Context, string) (app.ModelConfigurationView, error)
	PatchModelConfiguration(context.Context, string, string, string) (app.ModelConfigurationView, error)
}

type ModelProviderPresetResponse struct {
	ID                string                   `json:"id"`
	DisplayName       string                   `json:"displayName"`
	Region            yorvaruntime.ModelRegion `json:"region"`
	RecommendedModels []string                 `json:"recommendedModels"`
	HelpText          string                   `json:"helpText,omitempty"`
}

type ModelProviderPresetListResponse struct {
	Items []ModelProviderPresetResponse `json:"items"`
}

type ModelValidationSummaryResponse struct {
	State       string                  `json:"state"`
	ErrorCode   *yorvaruntime.ErrorCode `json:"errorCode"`
	CompletedAt *time.Time              `json:"completedAt"`
}

type ModelConfigurationResponse struct {
	ProviderPresetID     string                               `json:"providerPresetId"`
	ModelID              string                               `json:"modelId"`
	State                yorvaruntime.ModelConfigurationState `json:"state"`
	CredentialConfigured bool                                 `json:"credentialConfigured"`
	ObservedAt           time.Time                            `json:"observedAt"`
	Validation           ModelValidationSummaryResponse       `json:"validation"`
}

func listModelProviderPresets(models ModelConfigurationService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if models == nil {
			writeError(w, http.StatusInternalServerError, ErrorBody{Code: "INTERNAL_ERROR", Message: "Model configuration is unavailable.", Retryable: true})
			return
		}
		presets, err := models.ListModelProviderPresets(r.Context())
		if err != nil {
			writeModelError(w, app.ModelConfigurationView{}, err)
			return
		}
		items := make([]ModelProviderPresetResponse, 0, len(presets))
		for _, preset := range presets {
			items = append(items, ModelProviderPresetResponse{
				ID: preset.ID, DisplayName: preset.DisplayName, Region: preset.Region,
				RecommendedModels: append([]string(nil), preset.RecommendedModels...), HelpText: preset.HelpText,
			})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ModelProviderPresetListResponse{Items: items})
	})
}

func getModelConfiguration(models ModelConfigurationService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if models == nil {
			writeError(w, http.StatusInternalServerError, ErrorBody{Code: "INTERNAL_ERROR", Message: "Model configuration is unavailable.", Retryable: true})
			return
		}
		configuration, err := models.GetModelConfiguration(r.Context(), r.PathValue("instanceId"))
		if err != nil {
			writeModelError(w, configuration, err)
			return
		}
		writeModelConfiguration(w, http.StatusOK, configuration)
	})
}

func patchModelConfiguration(models ModelConfigurationService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if models == nil {
			writeError(w, http.StatusInternalServerError, ErrorBody{Code: "INTERNAL_ERROR", Message: "Model configuration is unavailable.", Retryable: true})
			return
		}
		presetID, modelID, err := decodeClosedModelConfig(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, ErrorBody{Code: string(yorvaruntime.ErrorModelConfigInvalid), Message: "The model configuration request is invalid.", Retryable: false})
			return
		}
		configuration, err := models.PatchModelConfiguration(r.Context(), r.PathValue("instanceId"), presetID, modelID)
		if err != nil {
			writeModelError(w, configuration, err)
			return
		}
		writeModelConfiguration(w, http.StatusOK, configuration)
	})
}

func writeModelConfiguration(w http.ResponseWriter, status int, configuration app.ModelConfigurationView) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(newModelConfigurationResponse(configuration))
}

func newModelConfigurationResponse(configuration app.ModelConfigurationView) ModelConfigurationResponse {
	return ModelConfigurationResponse{
		ProviderPresetID:     configuration.ProviderPresetID,
		ModelID:              configuration.ModelID,
		State:                configuration.State,
		CredentialConfigured: configuration.CredentialConfigured,
		ObservedAt:           configuration.ObservedAt,
		Validation:           ModelValidationSummaryResponse{State: "NOT_RUN"},
	}
}

func decodeClosedModelConfig(r *http.Request) (string, string, error) {
	if r.Body == nil {
		return "", "", io.EOF
	}
	defer r.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(r.Body, maxModelConfigRequestBytes+1))
	if err != nil || len(payload) == 0 || len(payload) > maxModelConfigRequestBytes {
		return "", "", io.ErrUnexpectedEOF
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var body struct {
		ProviderPresetID string `json:"providerPresetId"`
		ModelID          string `json:"modelId"`
	}
	if err := decoder.Decode(&body); err != nil || body.ProviderPresetID == "" || body.ModelID == "" {
		return "", "", errors.New("invalid model configuration")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return "", "", errors.New("trailing json")
	}
	return body.ProviderPresetID, body.ModelID, nil
}

func writeModelError(w http.ResponseWriter, observed app.ModelConfigurationView, err error) {
	switch {
	case errors.Is(err, app.ErrInstanceNotFound):
		writeError(w, http.StatusNotFound, ErrorBody{Code: string(yorvaruntime.ErrorInstanceNotFound), Message: "The requested instance was not found.", Retryable: false})
	case errors.Is(err, app.ErrInstanceNotAvailable):
		writeError(w, http.StatusConflict, ErrorBody{Code: "INSTANCE_NOT_AVAILABLE", Message: "The requested instance is not available.", Retryable: false})
	case errors.Is(err, yorvaruntime.ErrModelProviderUnsupported), errors.Is(err, app.ErrRuntimeNotSupported):
		writeError(w, http.StatusConflict, ErrorBody{Code: string(yorvaruntime.ErrorModelProviderUnsupported), Message: "Model configuration is not supported by this Hermes installation.", Retryable: false})
	case errors.Is(err, yorvaruntime.ErrModelConfigInvalid):
		writeError(w, http.StatusBadRequest, ErrorBody{Code: string(yorvaruntime.ErrorModelConfigInvalid), Message: "The model configuration request is invalid.", Retryable: false})
	case errors.Is(err, yorvaruntime.ErrModelCredentialRequired):
		writeError(w, http.StatusConflict, ErrorBody{Code: string(yorvaruntime.ErrorModelCredentialRequired), Message: "A configured credential is required for this provider.", Retryable: false})
	case errors.Is(err, yorvaruntime.ErrInstanceConfigConflict):
		writeObservedModelError(w, http.StatusConflict, yorvaruntime.ErrorInstanceConfigConflict, "The instance configuration changed concurrently.", observed, true)
	case errors.Is(err, yorvaruntime.ErrModelConfigIncomplete):
		writeObservedModelError(w, http.StatusConflict, yorvaruntime.ErrorModelConfigIncomplete, "The model configuration was only partially applied.", observed, true)
	case errors.Is(err, yorvaruntime.ErrModelConfigApplyFailed):
		writeError(w, http.StatusServiceUnavailable, ErrorBody{Code: string(yorvaruntime.ErrorModelConfigApplyFailed), Message: "The model configuration could not be applied.", Retryable: true})
	case errors.Is(err, yorvaruntime.ErrModelConfigQueryFailed):
		writeError(w, http.StatusServiceUnavailable, ErrorBody{Code: string(yorvaruntime.ErrorModelConfigQueryFailed), Message: "The model configuration could not be queried.", Retryable: true})
	case errors.Is(err, context.Canceled):
		return
	default:
		writeError(w, http.StatusInternalServerError, ErrorBody{Code: "INTERNAL_ERROR", Message: "Model configuration could not be completed.", Retryable: true})
	}
}

func writeObservedModelError(w http.ResponseWriter, status int, code yorvaruntime.ErrorCode, message string, observed app.ModelConfigurationView, retryable bool) {
	details := map[string]any{}
	if !observed.ObservedAt.IsZero() {
		details["observedConfiguration"] = newModelConfigurationResponse(observed)
	}
	writeError(w, status, ErrorBody{Code: string(code), Message: message, Retryable: retryable, Details: details})
}

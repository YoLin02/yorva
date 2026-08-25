package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/YoLin02/yorva/services/node/internal/app"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type ManagementSkillsService interface {
	ListSkills(context.Context, string) ([]yorvaruntime.Skill, error)
	InspectSkill(context.Context, string, string) (yorvaruntime.Skill, error)
}

type ManagementSkillResponse struct {
	ID                string                              `json:"id"`
	SourceID          string                              `json:"sourceId,omitempty"`
	Version           string                              `json:"version,omitempty"`
	InstallationState yorvaruntime.SkillInstallationState `json:"installationState"`
	EnabledState      yorvaruntime.SkillEnabledState      `json:"enabledState"`
	ScanState         yorvaruntime.SkillScanState         `json:"scanState"`
	UpdateAvailable   bool                                `json:"updateAvailable"`
}

type ManagementSkillListResponse struct {
	Items []ManagementSkillResponse `json:"items"`
}

func listInstanceSkills(service ManagementSkillsService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeManagementSkillsUnsupported(w)
			return
		}
		items, err := service.ListSkills(r.Context(), r.PathValue("instanceId"))
		if err != nil {
			writeManagementSkillsError(w, err)
			return
		}
		response := ManagementSkillListResponse{Items: make([]ManagementSkillResponse, 0, len(items))}
		for _, item := range items {
			response.Items = append(response.Items, newManagementSkillResponse(item))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
}

func inspectInstanceSkill(service ManagementSkillsService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeManagementSkillsUnsupported(w)
			return
		}
		item, err := service.InspectSkill(r.Context(), r.PathValue("instanceId"), r.PathValue("skillId"))
		if err != nil {
			writeManagementSkillsError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(newManagementSkillResponse(item))
	})
}

func newManagementSkillResponse(item yorvaruntime.Skill) ManagementSkillResponse {
	return ManagementSkillResponse{
		ID:                item.ID,
		SourceID:          item.SourceID,
		Version:           item.Version,
		InstallationState: item.InstallationState,
		EnabledState:      item.EnabledState,
		ScanState:         item.ScanState,
		UpdateAvailable:   item.UpdateAvailable,
	}
}

func writeManagementSkillsUnsupported(w http.ResponseWriter) {
	writeError(w, http.StatusConflict, ErrorBody{
		Code: string(yorvaruntime.ErrorCapabilityNotSupported), Message: "Skill management is not supported.", Retryable: false,
	})
}

func writeManagementSkillsError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, app.ErrInstanceNotFound):
		writeError(w, http.StatusNotFound, ErrorBody{Code: string(yorvaruntime.ErrorInstanceNotFound), Message: "The requested instance was not found.", Retryable: false})
	case errors.Is(err, app.ErrInstanceNotAvailable):
		writeError(w, http.StatusConflict, ErrorBody{Code: string(yorvaruntime.ErrorInstanceNotAvailable), Message: "The requested instance is not available.", Retryable: true})
	case errors.Is(err, app.ErrRuntimeNotSupported):
		writeError(w, http.StatusConflict, ErrorBody{Code: string(yorvaruntime.ErrorRuntimeNotSupported), Message: "A supported Runtime installation is required for Skill management.", Retryable: false})
	case errors.Is(err, app.ErrManagementCapabilityUnsupported):
		writeManagementSkillsUnsupported(w)
	case errors.Is(err, yorvaruntime.ErrInvalidManagementContract):
		writeError(w, http.StatusBadRequest, ErrorBody{Code: "INVALID_REQUEST", Message: "The Skill request is invalid.", Retryable: false})
	case errors.Is(err, app.ErrManagementQueryFailed):
		writeError(w, http.StatusServiceUnavailable, ErrorBody{Code: "MANAGEMENT_QUERY_FAILED", Message: "Skill inventory could not be queried.", Retryable: true})
	case errors.Is(err, context.Canceled):
		return
	default:
		writeError(w, http.StatusInternalServerError, ErrorBody{Code: "INTERNAL_ERROR", Message: "Skill management could not be completed.", Retryable: true})
	}
}

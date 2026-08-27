package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"

	"github.com/YoLin02/yorva/services/node/internal/app"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type ManagementSkillsService interface {
	ListSkills(context.Context, string) ([]yorvaruntime.Skill, error)
	InspectSkill(context.Context, string, string) (yorvaruntime.Skill, error)
	ListSources(context.Context, string) ([]app.SkillSourceView, error)
	StartInstall(context.Context, string, string, string, string) (app.InstallStartResult, error)
	StartImport(context.Context, string, string, string, string) (app.InstallStartResult, error)
	StartUpdate(context.Context, string, string, string) (app.InstallStartResult, error)
	StartEnable(context.Context, string, string, string) (app.InstallStartResult, error)
	StartDisable(context.Context, string, string, string) (app.InstallStartResult, error)
	StartRemove(context.Context, string, string, string) (app.InstallStartResult, error)
}

type ManagementSkillResponse struct {
	ID                string                              `json:"id"`
	SourceID          string                              `json:"sourceId,omitempty"`
	Version           string                              `json:"version,omitempty"`
	Description       string                              `json:"description,omitempty"`
	Preview           string                              `json:"preview,omitempty"`
	Ownership         yorvaruntime.SkillOwnership         `json:"ownership"`
	ProjectionState   yorvaruntime.SkillProjectionState   `json:"projectionState"`
	InstallationState yorvaruntime.SkillInstallationState `json:"installationState"`
	EnabledState      yorvaruntime.SkillEnabledState      `json:"enabledState"`
	ScanState         yorvaruntime.SkillScanState         `json:"scanState"`
	UpdateAvailable   bool                                `json:"updateAvailable"`
}

type ManagementSkillSourceResponse struct {
	SourceID    string `json:"sourceId"`
	SkillID     string `json:"skillId"`
	DisplayName string `json:"displayName"`
	Version     string `json:"version"`
}

type ManagementSkillSourceListResponse struct {
	Items []ManagementSkillSourceResponse `json:"items"`
}

type ManagementSkillListResponse struct {
	Items []ManagementSkillResponse `json:"items"`
}

func listInstanceSkillSources(service ManagementSkillsService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeManagementSkillsUnsupported(w)
			return
		}
		items, err := service.ListSources(r.Context(), r.PathValue("instanceId"))
		if err != nil {
			writeManagementSkillsError(w, err)
			return
		}
		response := ManagementSkillSourceListResponse{Items: make([]ManagementSkillSourceResponse, 0, len(items))}
		for _, item := range items {
			response.Items = append(response.Items, ManagementSkillSourceResponse(item))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
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
		Description:       item.Description,
		Preview:           item.Preview,
		Ownership:         item.Ownership,
		ProjectionState:   item.ProjectionState,
		InstallationState: item.InstallationState,
		EnabledState:      item.EnabledState,
		ScanState:         item.ScanState,
		UpdateAvailable:   item.UpdateAvailable,
	}
}

type skillMutationAction string

const (
	skillMutationInstall skillMutationAction = "install"
	skillMutationImport  skillMutationAction = "import"
	skillMutationUpdate  skillMutationAction = "update"
	skillMutationEnable  skillMutationAction = "enable"
	skillMutationDisable skillMutationAction = "disable"
	skillMutationRemove  skillMutationAction = "remove"
)

func startManagedSkillMutation(service ManagementSkillsService, action skillMutationAction) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeManagementSkillsUnsupported(w)
			return
		}
		key := r.Header.Get("Idempotency-Key")
		if app.ValidateIdempotencyKey(key) != nil {
			writeError(w, http.StatusBadRequest, ErrorBody{Code: "INVALID_IDEMPOTENCY_KEY", Message: "A valid Idempotency-Key header is required.", Retryable: false})
			return
		}
		instanceID, skillID := r.PathValue("instanceId"), r.PathValue("skillId")
		var (
			result app.InstallStartResult
			err    error
		)
		if action == skillMutationInstall {
			var sourceID string
			sourceID, err = decodeClosedSkillInstallRequest(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, ErrorBody{Code: "INVALID_REQUEST", Message: "The install request must be a closed JSON object with sourceId.", Retryable: false})
				return
			}
			result, err = service.StartInstall(r.Context(), instanceID, skillID, sourceID, key)
		} else if action == skillMutationImport {
			var sourceRef string
			sourceRef, err = decodeClosedSkillImportRequest(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, ErrorBody{Code: "INVALID_REQUEST", Message: "The import request must contain a native source reference.", Retryable: false})
				return
			}
			result, err = service.StartImport(r.Context(), instanceID, skillID, sourceRef, key)
		} else {
			if err = decodeClosedEmptyObject(r); err != nil {
				writeError(w, http.StatusBadRequest, ErrorBody{Code: "INVALID_REQUEST", Message: "The Skill mutation request must be a closed empty JSON object.", Retryable: false})
				return
			}
			switch action {
			case skillMutationUpdate:
				result, err = service.StartUpdate(r.Context(), instanceID, skillID, key)
			case skillMutationEnable:
				result, err = service.StartEnable(r.Context(), instanceID, skillID, key)
			case skillMutationDisable:
				result, err = service.StartDisable(r.Context(), instanceID, skillID, key)
			case skillMutationRemove:
				result, err = service.StartRemove(r.Context(), instanceID, skillID, key)
			}
		}
		if err != nil {
			writeManagementSkillsError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(newOperationResponse(result.Operation))
	})
}

var skillSourceRefPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

func decodeClosedSkillImportRequest(r *http.Request) (string, error) {
	if r.Body == nil {
		return "", io.EOF
	}
	defer r.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(r.Body, maxInstallRequestBytes+1))
	if err != nil || len(payload) == 0 || len(payload) > maxInstallRequestBytes {
		return "", io.ErrUnexpectedEOF
	}
	var body struct {
		SourceRef string `json:"sourceRef"`
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		return "", err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) || !skillSourceRefPattern.MatchString(body.SourceRef) {
		return "", errors.New("invalid source reference")
	}
	return body.SourceRef, nil
}

func decodeClosedSkillInstallRequest(r *http.Request) (string, error) {
	if r.Body == nil {
		return "", io.EOF
	}
	defer r.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(r.Body, maxInstallRequestBytes+1))
	if err != nil || len(payload) == 0 || len(payload) > maxInstallRequestBytes {
		return "", io.ErrUnexpectedEOF
	}
	var body struct {
		SourceID string `json:"sourceId"`
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		return "", err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return "", errors.New("trailing json")
	}
	if err := (yorvaruntime.SkillInstallRequest{SourceID: body.SourceID}).Validate(); err != nil {
		return "", err
	}
	return body.SourceID, nil
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
	case errors.Is(err, app.ErrInvalidIdempotencyKey):
		writeError(w, http.StatusBadRequest, ErrorBody{Code: "INVALID_IDEMPOTENCY_KEY", Message: "A valid Idempotency-Key header is required.", Retryable: false})
	case errors.Is(err, app.ErrSkillSourceNotApproved):
		writeError(w, http.StatusBadRequest, ErrorBody{Code: string(yorvaruntime.ErrorSkillSourceNotApproved), Message: "The requested Skill source is not approved.", Retryable: false})
	case errors.Is(err, app.ErrSkillOwnershipConflict):
		writeError(w, http.StatusConflict, ErrorBody{Code: string(yorvaruntime.ErrorSkillOwnershipConflict), Message: "Only YORVA-managed Skills can be changed.", Retryable: false})
	case errors.Is(err, app.ErrSkillDriftDetected):
		writeError(w, http.StatusConflict, ErrorBody{Code: string(yorvaruntime.ErrorSkillDriftDetected), Message: "The managed Skill projection has changed outside YORVA.", Retryable: false})
	case errors.Is(err, app.ErrSkillMutationConflict):
		writeError(w, http.StatusConflict, ErrorBody{Code: string(yorvaruntime.ErrorSkillMutationConflict), Message: "Another Skill mutation is already running for this Instance.", Retryable: false})
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

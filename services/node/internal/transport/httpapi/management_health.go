package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/app"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type InstanceManagementHealthService interface {
	GetInstanceHealth(context.Context, string) (yorvaruntime.HealthObservation, error)
	GetInstanceLogSnapshot(context.Context, string, yorvaruntime.LogCategory) (yorvaruntime.LogSnapshot, error)
	GetInstanceSecurityAudit(context.Context, string) (yorvaruntime.SecurityAuditResult, error)
}

type ManagementHealthFindingResponse struct {
	Code  string                   `json:"code"`
	State yorvaruntime.HealthState `json:"state"`
}

type ManagementHealthResponse struct {
	State      yorvaruntime.HealthState          `json:"state"`
	Findings   []ManagementHealthFindingResponse `json:"findings"`
	Partial    bool                              `json:"partial"`
	ObservedAt time.Time                         `json:"observedAt"`
}

type ManagementLogEntryResponse struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
}

type ManagementLogSnapshotResponse struct {
	Category   yorvaruntime.LogCategory     `json:"category"`
	Entries    []ManagementLogEntryResponse `json:"entries"`
	Truncated  bool                         `json:"truncated"`
	ObservedAt time.Time                    `json:"observedAt"`
}

type ManagementSecurityFindingResponse struct {
	ID        string                        `json:"id"`
	Component string                        `json:"component"`
	Severity  yorvaruntime.SecuritySeverity `json:"severity"`
}

type ManagementSecurityAuditResponse struct {
	State      yorvaruntime.SecurityAuditState     `json:"state"`
	Findings   []ManagementSecurityFindingResponse `json:"findings"`
	Partial    bool                                `json:"partial"`
	ObservedAt time.Time                           `json:"observedAt"`
}

func getInstanceHealth(service InstanceManagementHealthService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeManagementServiceUnavailable(w)
			return
		}
		result, err := service.GetInstanceHealth(r.Context(), r.PathValue("instanceId"))
		if err != nil {
			writeManagementHealthError(w, r, err)
			return
		}
		writeManagementHealthJSON(w, newManagementHealthResponse(result))
	})
}

func getInstanceLogSnapshot(service InstanceManagementHealthService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeManagementServiceUnavailable(w)
			return
		}
		category, ok := closedManagementLogCategory(r)
		if !ok {
			writeError(w, http.StatusBadRequest, ErrorBody{
				Code: "INVALID_REQUEST", Message: "A single supported log category is required.", Retryable: false,
			})
			return
		}
		result, err := service.GetInstanceLogSnapshot(r.Context(), r.PathValue("instanceId"), category)
		if err != nil {
			writeManagementHealthError(w, r, err)
			return
		}
		writeManagementHealthJSON(w, newManagementLogSnapshotResponse(result))
	})
}

func getInstanceSecurityAudit(service InstanceManagementHealthService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeManagementServiceUnavailable(w)
			return
		}
		result, err := service.GetInstanceSecurityAudit(r.Context(), r.PathValue("instanceId"))
		if err != nil {
			writeManagementHealthError(w, r, err)
			return
		}
		writeManagementHealthJSON(w, newManagementSecurityAuditResponse(result))
	})
}

func closedManagementLogCategory(r *http.Request) (yorvaruntime.LogCategory, bool) {
	query := r.URL.Query()
	values, ok := query["category"]
	if !ok || len(query) != 1 || len(values) != 1 {
		return "", false
	}
	category := yorvaruntime.LogCategory(values[0])
	return category, category.Valid()
}

func newManagementHealthResponse(result yorvaruntime.HealthObservation) ManagementHealthResponse {
	findings := make([]ManagementHealthFindingResponse, len(result.Findings))
	for i, finding := range result.Findings {
		findings[i] = ManagementHealthFindingResponse{Code: finding.Code, State: finding.State}
	}
	return ManagementHealthResponse{
		State: result.State, Findings: findings, Partial: result.Partial, ObservedAt: result.ObservedAt,
	}
}

func newManagementLogSnapshotResponse(result yorvaruntime.LogSnapshot) ManagementLogSnapshotResponse {
	entries := make([]ManagementLogEntryResponse, len(result.Entries))
	for i, entry := range result.Entries {
		entries[i] = ManagementLogEntryResponse{Timestamp: entry.Timestamp, Message: entry.Message}
	}
	return ManagementLogSnapshotResponse{
		Category: result.Category, Entries: entries, Truncated: result.Truncated, ObservedAt: result.ObservedAt,
	}
}

func newManagementSecurityAuditResponse(result yorvaruntime.SecurityAuditResult) ManagementSecurityAuditResponse {
	findings := make([]ManagementSecurityFindingResponse, len(result.Findings))
	for i, finding := range result.Findings {
		findings[i] = ManagementSecurityFindingResponse{ID: finding.ID, Component: finding.Component, Severity: finding.Severity}
	}
	return ManagementSecurityAuditResponse{
		State: result.State, Findings: findings, Partial: result.Partial, ObservedAt: result.ObservedAt,
	}
}

func writeManagementHealthJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func writeManagementServiceUnavailable(w http.ResponseWriter) {
	writeError(w, http.StatusInternalServerError, ErrorBody{
		Code: "INTERNAL_ERROR", Message: "Runtime management is unavailable.", Retryable: true,
	})
}

func writeManagementHealthError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, context.Canceled) && r.Context().Err() != nil:
		return
	case errors.Is(err, app.ErrInstanceNotFound):
		writeError(w, http.StatusNotFound, ErrorBody{
			Code: string(yorvaruntime.ErrorInstanceNotFound), Message: "The requested instance was not found.", Retryable: false,
		})
	case errors.Is(err, app.ErrInstanceNotAvailable):
		writeError(w, http.StatusConflict, ErrorBody{
			Code: string(yorvaruntime.ErrorInstanceNotAvailable), Message: "The requested instance is not available.", Retryable: false,
		})
	case errors.Is(err, app.ErrManagementCapabilityUnsupported), errors.Is(err, app.ErrRuntimeNotSupported):
		writeError(w, http.StatusConflict, ErrorBody{
			Code: string(yorvaruntime.ErrorCapabilityNotSupported), Message: "This management capability is not supported.", Retryable: false,
		})
	case errors.Is(err, app.ErrManagementQueryFailed), errors.Is(err, context.DeadlineExceeded):
		writeError(w, http.StatusServiceUnavailable, ErrorBody{
			Code: "MANAGEMENT_QUERY_FAILED", Message: "Runtime management data could not be queried.", Retryable: true,
		})
	default:
		writeError(w, http.StatusInternalServerError, ErrorBody{
			Code: "INTERNAL_ERROR", Message: "Runtime management could not be completed.", Retryable: true,
		})
	}
}

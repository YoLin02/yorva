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

type ManagementUpgradePlanService interface {
	PlanUpgrade(context.Context, string) (yorvaruntime.UpgradePlan, error)
}

type ManagementUpgradeService interface {
	ManagementUpgradePlanService
	StartUpgrade(context.Context, string, string) (app.InstallStartResult, error)
	StartRollback(context.Context, string, string) (app.InstallStartResult, error)
}

// ManagementUpgradePlanResponse is intentionally a safe projection. Exact
// archive/seal checks remain inside the adapter; filesystem paths, commands,
// internal seals and protection-point identities are never transported.
type ManagementUpgradePlanResponse struct {
	State                   yorvaruntime.UpgradeAvailabilityState  `json:"state"`
	CurrentVersion          string                                 `json:"currentVersion"`
	Candidate               ManagementUpgradeCandidateResponse     `json:"candidate"`
	ManagedStatus           string                                 `json:"managedStatus"`
	Compatibility           yorvaruntime.UpgradeCompatibilityState `json:"compatibility"`
	ProtectionPointRequired bool                                   `json:"protectionPointRequired"`
	ProtectionPointReady    bool                                   `json:"protectionPointReady"`
	BlockedReasons          []yorvaruntime.UpgradePlanReason       `json:"blockedReasons"`
	ObservedAt              time.Time                              `json:"observedAt"`
}

type ManagementUpgradeCandidateResponse struct {
	Label   string `json:"label"`
	Version string `json:"version"`
}

func getRuntimeUpgradePlan(service ManagementUpgradePlanService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeManagementUpgradeUnsupported(w)
			return
		}
		plan, err := service.PlanUpgrade(r.Context(), r.PathValue("runtimeId"))
		if err != nil {
			writeManagementUpgradeError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(newManagementUpgradePlanResponse(plan))
	})
}

func startRuntimeUpgrade(service ManagementUpgradeService, rollback bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeManagementUpgradeUnsupported(w)
			return
		}
		key := r.Header.Get("Idempotency-Key")
		if app.ValidateIdempotencyKey(key) != nil || decodeClosedEmptyObject(r) != nil {
			writeError(w, http.StatusBadRequest, ErrorBody{Code: "INVALID_REQUEST", Message: "A valid Idempotency-Key and closed empty JSON object are required.", Retryable: false})
			return
		}
		var (
			result app.InstallStartResult
			err    error
		)
		if rollback {
			result, err = service.StartRollback(r.Context(), r.PathValue("runtimeId"), key)
		} else {
			result, err = service.StartUpgrade(r.Context(), r.PathValue("runtimeId"), key)
		}
		if err != nil {
			writeManagementUpgradeError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(newOperationResponse(result.Operation))
	})
}

func newManagementUpgradePlanResponse(plan yorvaruntime.UpgradePlan) ManagementUpgradePlanResponse {
	return ManagementUpgradePlanResponse{
		State:          plan.State,
		CurrentVersion: plan.CurrentVersion,
		Candidate: ManagementUpgradeCandidateResponse{
			Label: plan.CandidateLabel, Version: plan.TargetVersion,
		},
		ManagedStatus:           publicManagedStatus(plan),
		Compatibility:           plan.Compatibility,
		ProtectionPointRequired: plan.ProtectionPointRequired,
		ProtectionPointReady:    plan.ProtectionPointReady,
		BlockedReasons:          append([]yorvaruntime.UpgradePlanReason(nil), plan.Reasons...),
		ObservedAt:              plan.ObservedAt,
	}
}

func publicManagedStatus(plan yorvaruntime.UpgradePlan) string {
	if plan.Managed {
		return "MANAGED"
	}
	return "UNKNOWN"
}

func writeManagementUpgradeUnsupported(w http.ResponseWriter) {
	writeError(w, http.StatusConflict, ErrorBody{
		Code: string(yorvaruntime.ErrorCapabilityNotSupported), Message: "Managed Runtime upgrade is not supported.", Retryable: false,
	})
}

func writeManagementUpgradeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, context.Canceled) && r.Context().Err() != nil:
		return
	case errors.Is(err, app.ErrRuntimeNotSupported), errors.Is(err, app.ErrRuntimeKindNotFound):
		writeError(w, http.StatusConflict, ErrorBody{Code: string(yorvaruntime.ErrorRuntimeNotSupported), Message: "A supported managed Runtime installation is required.", Retryable: false})
	case errors.Is(err, app.ErrManagementCapabilityUnsupported):
		writeManagementUpgradeUnsupported(w)
	case errors.Is(err, app.ErrRuntimeMutationConflict):
		writeError(w, http.StatusConflict, ErrorBody{Code: "RUNTIME_MUTATION_CONFLICT", Message: "Another Runtime mutation is already running.", Retryable: false})
	case errors.Is(err, app.ErrManagementQueryFailed), errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		writeError(w, http.StatusServiceUnavailable, ErrorBody{Code: "MANAGEMENT_QUERY_FAILED", Message: "The managed Runtime upgrade plan could not be queried.", Retryable: true})
	default:
		writeError(w, http.StatusInternalServerError, ErrorBody{Code: "INTERNAL_ERROR", Message: "The managed Runtime upgrade plan could not be completed.", Retryable: true})
	}
}

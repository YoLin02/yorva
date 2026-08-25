package app

import (
	"context"
	"errors"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

// ManagementHealth exposes the read-only Phase 7 health, bounded log and
// security projections. Runtime-specific acquisition remains in the selected
// Bundle capability.
type ManagementHealth struct {
	targets ManagementTargetResolver
}

func NewManagementHealth(targets ManagementTargetResolver) *ManagementHealth {
	return &ManagementHealth{targets: targets}
}

func (s *ManagementHealth) GetInstanceHealth(ctx context.Context, instanceID string) (yorvaruntime.HealthObservation, error) {
	target, err := s.resolve(ctx, instanceID)
	if err != nil {
		return yorvaruntime.HealthObservation{}, err
	}
	if target.Bundle.Health == nil {
		return yorvaruntime.HealthObservation{}, ErrManagementCapabilityUnsupported
	}

	result, err := target.Bundle.Health.InspectInstanceHealth(ctx, target.Installation, target.NativeID)
	if err != nil {
		return yorvaruntime.HealthObservation{}, managementQueryError(ctx, err)
	}
	if err := result.Validate(); err != nil {
		return yorvaruntime.HealthObservation{}, ErrManagementQueryFailed
	}
	return result, nil
}

func (s *ManagementHealth) GetInstanceLogSnapshot(ctx context.Context, instanceID string, category yorvaruntime.LogCategory) (yorvaruntime.LogSnapshot, error) {
	if !category.Valid() {
		return yorvaruntime.LogSnapshot{}, ErrManagementQueryFailed
	}
	target, err := s.resolve(ctx, instanceID)
	if err != nil {
		return yorvaruntime.LogSnapshot{}, err
	}
	if target.Bundle.Logs == nil {
		return yorvaruntime.LogSnapshot{}, ErrManagementCapabilityUnsupported
	}

	result, err := target.Bundle.Logs.ReadLogSnapshot(ctx, target.Installation, target.NativeID, category)
	if err != nil {
		return yorvaruntime.LogSnapshot{}, managementQueryError(ctx, err)
	}
	if err := result.Validate(); err != nil {
		return yorvaruntime.LogSnapshot{}, ErrManagementQueryFailed
	}
	return result, nil
}

func (s *ManagementHealth) GetInstanceSecurityAudit(ctx context.Context, instanceID string) (yorvaruntime.SecurityAuditResult, error) {
	target, err := s.resolve(ctx, instanceID)
	if err != nil {
		return yorvaruntime.SecurityAuditResult{}, err
	}
	if target.Bundle.Security == nil {
		return yorvaruntime.SecurityAuditResult{}, ErrManagementCapabilityUnsupported
	}

	result, err := target.Bundle.Security.AuditInstanceSecurity(ctx, target.Installation, target.NativeID)
	if err != nil {
		return yorvaruntime.SecurityAuditResult{}, managementQueryError(ctx, err)
	}
	if err := result.Validate(); err != nil {
		return yorvaruntime.SecurityAuditResult{}, ErrManagementQueryFailed
	}
	return result, nil
}

func (s *ManagementHealth) resolve(ctx context.Context, instanceID string) (ManagementTarget, error) {
	if s == nil || s.targets == nil {
		return ManagementTarget{}, ErrManagementQueryFailed
	}
	if instanceID == "" {
		return ManagementTarget{}, ErrInstanceNotFound
	}
	target, err := s.targets.ResolveManagementTarget(ctx, instanceID)
	if err != nil {
		return ManagementTarget{}, managementQueryError(ctx, err)
	}
	return target, nil
}

func managementQueryError(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	switch {
	case errors.Is(err, ErrInstanceNotFound):
		return ErrInstanceNotFound
	case errors.Is(err, ErrInstanceNotAvailable):
		return ErrInstanceNotAvailable
	case errors.Is(err, ErrRuntimeNotSupported):
		return ErrRuntimeNotSupported
	case errors.Is(err, ErrManagementCapabilityUnsupported):
		return ErrManagementCapabilityUnsupported
	case errors.Is(err, ErrManagementQueryFailed):
		return ErrManagementQueryFailed
	default:
		return ErrManagementQueryFailed
	}
}

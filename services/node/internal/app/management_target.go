package app

import (
	"context"
	"database/sql"
	"errors"

	"github.com/YoLin02/yorva/services/node/internal/domain/instance"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

var (
	ErrManagementCapabilityUnsupported = errors.New("management capability is not supported")
	ErrManagementQueryFailed           = errors.New("management query failed")
)

// ManagementTarget is the Runtime-neutral target passed to one fine-grained
// management capability. NativeID is adapter-owned and must not be exposed by
// transport responses.
type ManagementTarget struct {
	Installation yorvaruntime.Installation
	NativeID     string
	Bundle       yorvaruntime.Bundle
}

// ManagementTargetResolver resolves live-management ownership without making
// the individual feature use cases depend on SQLite or Runtime discovery.
type ManagementTargetResolver interface {
	ResolveManagementTarget(context.Context, string) (ManagementTarget, error)
}

func (s *InstanceInventory) ResolveManagementTarget(ctx context.Context, instanceID string) (ManagementTarget, error) {
	if s == nil || s.db == nil || s.discovery == nil || s.discovery.registry == nil || instanceID == "" {
		return ManagementTarget{}, ErrInstanceNotFound
	}

	row, err := s.db.GetInstance(ctx, instanceID)
	if errors.Is(err, sql.ErrNoRows) {
		return ManagementTarget{}, ErrInstanceNotFound
	}
	if err != nil {
		return ManagementTarget{}, managementTargetError(ctx, err)
	}
	if row.Availability != instance.Available || row.NativeID == "" {
		return ManagementTarget{}, ErrInstanceNotAvailable
	}

	accepted, err := s.db.GetAcceptedInstallationByID(ctx, row.RuntimeInstallationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ManagementTarget{}, ErrRuntimeNotSupported
		}
		return ManagementTarget{}, managementTargetError(ctx, err)
	}
	if accepted.ID != row.RuntimeInstallationID || accepted.NodeID != s.nodeID ||
		accepted.Status != "ACCEPTED" || accepted.SupportState != yorvaruntime.DiscoverySupported ||
		accepted.RuntimeKind == "" || accepted.InstallPath == "" || accepted.Version == "" {
		return ManagementTarget{}, ErrRuntimeNotSupported
	}

	bundle, ok := s.discovery.registry.Get(accepted.RuntimeKind)
	if !ok || bundle.Descriptor.Kind != accepted.RuntimeKind {
		return ManagementTarget{}, ErrRuntimeNotSupported
	}

	return ManagementTarget{
		Installation: yorvaruntime.Installation{
			RuntimeKind:  accepted.RuntimeKind,
			Path:         accepted.InstallPath,
			Version:      accepted.Version,
			SupportState: accepted.SupportState,
		},
		NativeID: row.NativeID,
		Bundle:   bundle,
	}, nil
}

func managementTargetError(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return ErrManagementQueryFailed
}

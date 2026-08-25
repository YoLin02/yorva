package app

import (
	"context"
	"database/sql"
	"errors"

	"github.com/YoLin02/yorva/services/node/internal/domain/instance"
	"github.com/YoLin02/yorva/services/node/internal/persistence/sqlite"
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

// RuntimeManagementTarget is the live Runtime installation selected for a
// Runtime-scoped management use case such as backup or upgrade. InstallationID
// is YORVA-owned coordination metadata and must not be passed to the adapter.
type RuntimeManagementTarget struct {
	Installation   yorvaruntime.Installation
	InstallationID string
	Bundle         yorvaruntime.Bundle
}

// ManagementTargetResolver resolves live-management ownership without making
// the individual feature use cases depend on SQLite or Runtime discovery.
type ManagementTargetResolver interface {
	ResolveManagementTarget(context.Context, string) (ManagementTarget, error)
}

type RuntimeManagementTargetResolver interface {
	ResolveRuntimeManagementTarget(context.Context, string) (RuntimeManagementTarget, error)
}

type runtimeManagementLocker interface {
	lockInstallation(string) func()
}

func lockRuntimeManagementTarget(resolver RuntimeManagementTargetResolver, installationID string) func() {
	if locker, ok := resolver.(runtimeManagementLocker); ok && installationID != "" {
		return locker.lockInstallation(installationID)
	}
	return func() {}
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

	detected, err := s.discovery.Detect(ctx, accepted.RuntimeKind)
	if err != nil || detected.RuntimeKind != accepted.RuntimeKind || detected.State != yorvaruntime.DiscoverySupported || detected.Selected == nil ||
		detected.Selected.State != yorvaruntime.DiscoverySupported || detected.Selected.Path == "" ||
		detected.Selected.Path != accepted.InstallPath || detected.Selected.Version == "" {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ManagementTarget{}, ctxErr
		}
		return ManagementTarget{}, ErrRuntimeNotSupported
	}

	bundle, ok := s.discovery.registry.Get(accepted.RuntimeKind)
	if !ok || bundle.Descriptor.Kind != accepted.RuntimeKind {
		return ManagementTarget{}, ErrRuntimeNotSupported
	}

	installation := yorvaruntime.Installation{
		RuntimeKind:  accepted.RuntimeKind,
		Path:         detected.Selected.Path,
		Version:      detected.Selected.Version,
		SupportState: detected.State,
	}
	bundle = bundle.ResolveInstanceManagement(ctx, installation, row.NativeID)

	return ManagementTarget{
		Installation: installation,
		NativeID:     row.NativeID,
		Bundle:       bundle,
	}, nil
}

func (s *InstanceInventory) ResolveRuntimeManagementTarget(ctx context.Context, runtimeID string) (RuntimeManagementTarget, error) {
	if s == nil || s.db == nil || s.discovery == nil || s.discovery.registry == nil || runtimeID == "" {
		return RuntimeManagementTarget{}, ErrRuntimeNotSupported
	}

	kind := yorvaruntime.Kind(runtimeID)
	detected, err := s.discovery.Detect(ctx, kind)
	if err != nil || detected.RuntimeKind != kind || detected.State != yorvaruntime.DiscoverySupported ||
		detected.Selected == nil || detected.Selected.State != yorvaruntime.DiscoverySupported ||
		detected.Selected.Path == "" || detected.Selected.Version == "" {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return RuntimeManagementTarget{}, ctxErr
		}
		return RuntimeManagementTarget{}, ErrRuntimeNotSupported
	}

	accepted, err := s.db.GetAcceptedInstallation(ctx, s.nodeID, kind, detected.Selected.Path)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RuntimeManagementTarget{}, ErrRuntimeNotSupported
		}
		return RuntimeManagementTarget{}, managementTargetError(ctx, err)
	}
	if accepted.NodeID != s.nodeID || accepted.RuntimeKind != kind || accepted.InstallPath != detected.Selected.Path ||
		accepted.Status != "ACCEPTED" || accepted.SupportState != yorvaruntime.DiscoverySupported {
		return RuntimeManagementTarget{}, ErrRuntimeNotSupported
	}

	bundle, ok := s.discovery.registry.Get(kind)
	if !ok || bundle.Descriptor.Kind != kind {
		return RuntimeManagementTarget{}, ErrRuntimeNotSupported
	}
	// Runtime backup inventory is YORVA-owned and becomes readable only after
	// both a live accepted installation and the Runtime-scoped index repository
	// have been resolved. This does not enable create, delete, or Restore.
	bundle.BackupRead = sqlite.NewRuntimeBackupReader(s.db, accepted.ID)

	return RuntimeManagementTarget{
		Installation: yorvaruntime.Installation{
			RuntimeKind:  kind,
			Path:         detected.Selected.Path,
			Version:      detected.Selected.Version,
			SupportState: detected.State,
		},
		InstallationID: accepted.ID,
		Bundle:         bundle,
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

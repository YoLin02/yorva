package app

import (
	"context"
	"errors"
	"strings"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const (
	managementBackupCollectionLimit              = 256
	managementBackupMetadataMaxBytes             = 64
	BackupScopeRuntime               BackupScope = "RUNTIME"
)

var ErrBackupNotFound = errors.New("backup not found")

type BackupScope string

// BackupView is safe management metadata. It deliberately excludes artifact
// paths, destination capabilities, encryption key references, credentials and
// archive member details.
type BackupView struct {
	ID             string
	Scope          BackupScope
	State          yorvaruntime.BackupState
	FormatVersion  string
	RuntimeVersion string
	SizeBytes      int64
	ChecksumSHA256 string
	CreatedAt      time.Time
	VerifiedAt     time.Time
	KeyMode        yorvaruntime.BackupKeyMode
}

type BackupManagement struct {
	targets RuntimeManagementTargetResolver
}

func NewBackupManagement(targets RuntimeManagementTargetResolver) *BackupManagement {
	return &BackupManagement{targets: targets}
}

func (s *BackupManagement) ListBackups(ctx context.Context, runtimeID string) ([]BackupView, error) {
	target, err := s.resolve(ctx, runtimeID)
	if err != nil {
		return nil, err
	}
	if target.Bundle.BackupRead == nil {
		return nil, ErrManagementCapabilityUnsupported
	}

	backups, err := target.Bundle.BackupRead.ListBackups(ctx, target.Installation)
	if err != nil {
		return nil, managementBackupError(ctx, err)
	}
	if len(backups) > managementBackupCollectionLimit {
		return nil, ErrManagementQueryFailed
	}

	seen := make(map[string]struct{}, len(backups))
	views := make([]BackupView, 0, len(backups))
	for _, backup := range backups {
		view, err := validatedBackupView(backup)
		if err != nil {
			return nil, err
		}
		if _, duplicate := seen[view.ID]; duplicate {
			return nil, ErrManagementQueryFailed
		}
		seen[view.ID] = struct{}{}
		views = append(views, view)
	}
	return views, nil
}

func (s *BackupManagement) InspectBackup(ctx context.Context, runtimeID, backupID string) (BackupView, error) {
	if !validManagementBackupID(backupID) {
		return BackupView{}, ErrBackupNotFound
	}
	target, err := s.resolve(ctx, runtimeID)
	if err != nil {
		return BackupView{}, err
	}
	if target.Bundle.BackupRead == nil {
		return BackupView{}, ErrManagementCapabilityUnsupported
	}
	backup, err := target.Bundle.BackupRead.GetBackup(ctx, target.Installation, backupID)
	if err != nil {
		return BackupView{}, managementBackupError(ctx, err)
	}
	if backup.ID != backupID {
		return BackupView{}, ErrManagementQueryFailed
	}
	return validatedBackupView(backup)
}

// CreateBackup is an Operation-worker boundary only. Qualification is
// represented by compile-time Bundle wiring; the caller must own durable
// Operation state, conflicts, timeout, cancellation and progress.
func (s *BackupManagement) CreateBackup(ctx context.Context, runtimeID string, request yorvaruntime.BackupCreateRequest, progress yorvaruntime.ProgressSink) (BackupView, error) {
	if err := request.Validate(); err != nil {
		return BackupView{}, ErrManagementQueryFailed
	}
	target, err := s.resolve(ctx, runtimeID)
	if err != nil {
		return BackupView{}, err
	}
	if target.Bundle.BackupMutate == nil {
		return BackupView{}, ErrManagementCapabilityUnsupported
	}
	backup, err := target.Bundle.BackupMutate.CreateBackup(ctx, target.Installation, request, progress)
	if err != nil {
		return BackupView{}, managementBackupError(ctx, err)
	}
	if backup.State != yorvaruntime.BackupAvailable {
		return BackupView{}, ErrManagementQueryFailed
	}
	return validatedBackupView(backup)
}

// DeleteBackup is an Operation-worker boundary only and has no HTTP mutation
// handler while the product Bundle leaves backup mutation unwired.
func (s *BackupManagement) DeleteBackup(ctx context.Context, runtimeID, backupID string, progress yorvaruntime.ProgressSink) error {
	if !validManagementBackupID(backupID) {
		return ErrBackupNotFound
	}
	target, err := s.resolve(ctx, runtimeID)
	if err != nil {
		return err
	}
	if target.Bundle.BackupMutate == nil {
		return ErrManagementCapabilityUnsupported
	}
	if err := target.Bundle.BackupMutate.DeleteBackup(ctx, target.Installation, backupID, progress); err != nil {
		return managementBackupError(ctx, err)
	}
	return nil
}

func (s *BackupManagement) resolve(ctx context.Context, runtimeID string) (RuntimeManagementTarget, error) {
	if s == nil || s.targets == nil {
		return RuntimeManagementTarget{}, ErrManagementCapabilityUnsupported
	}
	target, err := s.targets.ResolveRuntimeManagementTarget(ctx, runtimeID)
	if err != nil {
		return RuntimeManagementTarget{}, managementBackupError(ctx, err)
	}
	return target, nil
}

func validatedBackupView(backup yorvaruntime.Backup) (BackupView, error) {
	if err := backup.Validate(); err != nil || !backup.State.Indexed() {
		return BackupView{}, ErrManagementQueryFailed
	}
	if !validManagementBackupMetadata(backup.FormatVersion) || !validManagementBackupMetadata(backup.RuntimeVersion) {
		return BackupView{}, ErrManagementQueryFailed
	}
	return BackupView{
		ID: backup.ID, Scope: BackupScopeRuntime, State: backup.State,
		FormatVersion: backup.FormatVersion, RuntimeVersion: backup.RuntimeVersion,
		SizeBytes: backup.SizeBytes, ChecksumSHA256: backup.ChecksumSHA256,
		CreatedAt: backup.CreatedAt.UTC(), VerifiedAt: backup.VerifiedAt.UTC(), KeyMode: backup.KeyMode,
	}, nil
}

func validManagementBackupID(backupID string) bool {
	return (yorvaruntime.Backup{ID: backupID, State: yorvaruntime.BackupFailed}).Validate() == nil
}

func validManagementBackupMetadata(value string) bool {
	if value == "" || len(value) > managementBackupMetadataMaxBytes || strings.TrimSpace(value) != value {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' || r == '+' {
			continue
		}
		return false
	}
	return true
}

func managementBackupError(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	case errors.Is(err, yorvaruntime.ErrBackupNotFound):
		return ErrBackupNotFound
	case errors.Is(err, ErrBackupNotFound), errors.Is(err, ErrRuntimeNotSupported),
		errors.Is(err, ErrManagementCapabilityUnsupported), errors.Is(err, ErrManagementQueryFailed):
		return err
	default:
		return ErrManagementQueryFailed
	}
}

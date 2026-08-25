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
	managementBackupMaxArtifactBytes             = int64(2 << 30)
	managementBackupMetadataMaxBytes             = 64
	BackupScopeRuntime               BackupScope = "RUNTIME"
)

var ErrBackupNotFound = errors.New("backup not found")

type BackupScope string

// BackupView is safe management metadata. It deliberately excludes artifact
// paths, destination capabilities, encryption key references, credentials and
// archive member details.
type BackupView struct {
	ID               string
	Scope            BackupScope
	State            yorvaruntime.BackupState
	FormatVersion    string
	RuntimeVersion   string
	SizeBytes        int64
	ChecksumSHA256   string
	CreatedAt        *time.Time
	ArtifactVerified bool
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
		view, err := validatedBackupView(backup, false)
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
	backups, err := s.ListBackups(ctx, runtimeID)
	if err != nil {
		return BackupView{}, err
	}
	for _, backup := range backups {
		if backup.ID == backupID {
			return backup, nil
		}
	}
	return BackupView{}, ErrBackupNotFound
}

func (s *BackupManagement) VerifyBackup(ctx context.Context, runtimeID, backupID string) (BackupView, error) {
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

	backup, err := target.Bundle.BackupRead.VerifyBackup(ctx, target.Installation, backupID)
	if err != nil {
		return BackupView{}, managementBackupError(ctx, err)
	}
	if backup.ID != backupID || backup.State != yorvaruntime.BackupAvailable {
		return BackupView{}, ErrManagementQueryFailed
	}
	return validatedBackupView(backup, true)
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
	return validatedBackupView(backup, true)
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

func validatedBackupView(backup yorvaruntime.Backup, verified bool) (BackupView, error) {
	if err := backup.Validate(); err != nil || backup.SizeBytes > managementBackupMaxArtifactBytes {
		return BackupView{}, ErrManagementQueryFailed
	}
	view := BackupView{
		ID: backup.ID, Scope: BackupScopeRuntime, State: backup.State, SizeBytes: backup.SizeBytes,
	}
	if backup.State != yorvaruntime.BackupAvailable {
		return view, nil
	}
	if !validManagementBackupMetadata(backup.FormatVersion) || !validManagementBackupMetadata(backup.RuntimeVersion) {
		return BackupView{}, ErrManagementQueryFailed
	}
	createdAt := backup.CreatedAt.UTC()
	view.FormatVersion = backup.FormatVersion
	view.RuntimeVersion = backup.RuntimeVersion
	view.ChecksumSHA256 = backup.ChecksumSHA256
	view.CreatedAt = &createdAt
	view.ArtifactVerified = verified
	return view, nil
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
	case errors.Is(err, ErrBackupNotFound), errors.Is(err, ErrRuntimeNotSupported),
		errors.Is(err, ErrManagementCapabilityUnsupported), errors.Is(err, ErrManagementQueryFailed):
		return err
	default:
		return ErrManagementQueryFailed
	}
}

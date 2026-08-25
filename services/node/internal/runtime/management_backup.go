package runtime

import (
	"context"
	"encoding/hex"
	"time"
)

type BackupState string

const (
	BackupCreating  BackupState = "CREATING"
	BackupAvailable BackupState = "AVAILABLE"
	BackupRestoring BackupState = "RESTORING"
	BackupFailed    BackupState = "FAILED"
	BackupDeleting  BackupState = "DELETING"
)

func (s BackupState) Valid() bool {
	return s == BackupCreating || s == BackupAvailable || s == BackupRestoring || s == BackupFailed || s == BackupDeleting
}

type Backup struct {
	ID             string
	State          BackupState
	FormatVersion  string
	RuntimeVersion string
	SizeBytes      int64
	ChecksumSHA256 string
	CreatedAt      time.Time
}

func (b Backup) Validate() error {
	if err := validateManagementID("backup id", b.ID); err != nil {
		return err
	}
	if !b.State.Valid() || b.SizeBytes < 0 {
		return ErrInvalidManagementContract
	}
	if b.State != BackupAvailable {
		return nil
	}
	if b.FormatVersion == "" || b.RuntimeVersion == "" || b.SizeBytes == 0 || b.CreatedAt.IsZero() || !validSHA256(b.ChecksumSHA256) {
		return ErrInvalidManagementContract
	}
	return nil
}

// BackupCreateRequest carries only an opaque, locally-issued destination
// capability. It deliberately carries no caller path, URL, format, command, or
// key material.
type BackupCreateRequest struct {
	DestinationRef string
}

func (r BackupCreateRequest) Validate() error {
	return validateManagementID("backup destination reference", r.DestinationRef)
}

type BackupRestoreRequest struct {
	BackupID string
}

func (r BackupRestoreRequest) Validate() error {
	return validateManagementID("backup id", r.BackupID)
}

type RestoreOutcomeState string

const (
	RestoreSucceeded        RestoreOutcomeState = "SUCCEEDED"
	RestoreRolledBack       RestoreOutcomeState = "ROLLED_BACK"
	RestoreRecoveryRequired RestoreOutcomeState = "RECOVERY_REQUIRED"
	RestoreUnknown          RestoreOutcomeState = "UNKNOWN"
)

func (s RestoreOutcomeState) Valid() bool {
	return s == RestoreSucceeded || s == RestoreRolledBack || s == RestoreRecoveryRequired || s == RestoreUnknown
}

type RestoreResult struct {
	State      RestoreOutcomeState
	ObservedAt time.Time
}

func (r RestoreResult) Validate() error {
	if !r.State.Valid() || r.ObservedAt.IsZero() {
		return ErrInvalidManagementContract
	}
	return nil
}

type BackupReader interface {
	ListBackups(context.Context, Installation) ([]Backup, error)
	VerifyBackup(context.Context, Installation, string) (Backup, error)
}

type BackupManager interface {
	CreateBackup(context.Context, Installation, BackupCreateRequest, ProgressSink) (Backup, error)
	DeleteBackup(context.Context, Installation, string, ProgressSink) error
}

type RestoreManager interface {
	RestoreBackup(context.Context, Installation, BackupRestoreRequest, ProgressSink) (RestoreResult, error)
}

func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

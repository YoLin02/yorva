package runtime

import (
	"context"
	"encoding/hex"
	"errors"
	"regexp"
	"time"
)

var ErrBackupNotFound = errors.New("Runtime backup not found")

var backupDestinationRefPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

type BackupState string

const (
	BackupCreating      BackupState = "CREATING"
	BackupAvailable     BackupState = "AVAILABLE"
	BackupRestoring     BackupState = "RESTORING"
	BackupFailed        BackupState = "FAILED"
	BackupDeleting      BackupState = "DELETING"
	BackupMissing       BackupState = "MISSING"
	BackupChanged       BackupState = "CHANGED"
	BackupUndecryptable BackupState = "UNDECRYPTABLE"
	BackupMalformed     BackupState = "MALFORMED"
	BackupUnknown       BackupState = "UNKNOWN"
)

func (s BackupState) Valid() bool {
	return s == BackupCreating || s == BackupAvailable || s == BackupRestoring || s == BackupFailed || s == BackupDeleting || s.Indexed()
}

// Indexed reports whether state is one of the last-observed states persisted
// in the Runtime-scoped backup index. It does not imply a fresh file check.
func (s BackupState) Indexed() bool {
	return s == BackupAvailable || s == BackupMissing || s == BackupChanged ||
		s == BackupUndecryptable || s == BackupMalformed || s == BackupUnknown
}

type BackupKeyMode string

const (
	BackupKeyDevice     BackupKeyMode = "DEVICE"
	BackupKeyPassphrase BackupKeyMode = "PASSPHRASE"
)

func (m BackupKeyMode) Valid() bool {
	return m == BackupKeyDevice || m == BackupKeyPassphrase
}

type Backup struct {
	ID             string
	State          BackupState
	FormatVersion  string
	RuntimeVersion string
	SizeBytes      int64
	ChecksumSHA256 string
	CreatedAt      time.Time
	VerifiedAt     time.Time
	KeyMode        BackupKeyMode
}

func (b Backup) Validate() error {
	if err := validateManagementID("backup id", b.ID); err != nil {
		return err
	}
	if !b.State.Valid() || b.SizeBytes < 0 {
		return ErrInvalidManagementContract
	}
	if !b.State.Indexed() {
		return nil
	}
	if b.FormatVersion == "" || b.RuntimeVersion == "" || b.SizeBytes == 0 || b.CreatedAt.IsZero() ||
		b.VerifiedAt.IsZero() || !b.KeyMode.Valid() || !validSHA256(b.ChecksumSHA256) {
		return ErrInvalidManagementContract
	}
	return nil
}

// BackupCreateRequest carries only an opaque, locally-issued destination
// capability. It deliberately carries no caller path, URL, format, command, or
// key material.
type BackupCreateRequest struct {
	DestinationRef        string
	OperationID           string
	RuntimeInstallationID string
}

func (r BackupCreateRequest) Validate() error {
	if err := ValidateBackupDestinationRef(r.DestinationRef); err != nil {
		return err
	}
	if err := validateManagementID("backup operation id", r.OperationID); err != nil {
		return err
	}
	return validateManagementID("Runtime installation id", r.RuntimeInstallationID)
}

func ValidateBackupDestinationRef(value string) error {
	if !backupDestinationRefPattern.MatchString(value) {
		return ErrInvalidManagementContract
	}
	return nil
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
	GetBackup(context.Context, Installation, string) (Backup, error)
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

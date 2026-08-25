package app

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	"github.com/YoLin02/yorva/services/node/internal/events"
	"github.com/YoLin02/yorva/services/node/internal/persistence/sqlite"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const (
	managementBackupCollectionLimit              = 256
	managementBackupMetadataMaxBytes             = 64
	BackupScopeRuntime               BackupScope = "RUNTIME"
)

var (
	ErrBackupNotFound         = errors.New("backup not found")
	ErrBackupMutationConflict = errors.New("another backup or Runtime mutation is active")
)

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
	db      *sqlite.Database
	events  *events.Broker
	now     func() time.Time
	newID   func() (string, error)
	mu      sync.Mutex
	cancels map[string]context.CancelFunc
}

func NewBackupManagement(targets RuntimeManagementTargetResolver) *BackupManagement {
	return &BackupManagement{targets: targets, now: func() time.Time { return time.Now().UTC() }, newID: newOperationID, cancels: make(map[string]context.CancelFunc)}
}

func NewManagedBackupManagement(targets RuntimeManagementTargetResolver, db *sqlite.Database, broker *events.Broker) *BackupManagement {
	service := NewBackupManagement(targets)
	service.db, service.events = db, broker
	return service
}

func (s *InstanceInventory) NewBackupManagement() (*BackupManagement, error) {
	if s == nil || s.db == nil {
		return nil, ErrManagementQueryFailed
	}
	service := NewManagedBackupManagement(s, s.db, s.events)
	if _, err := service.RecoverInterrupted(context.Background()); err != nil {
		return nil, err
	}
	return service, nil
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

func (s *BackupManagement) StartCreateBackup(ctx context.Context, runtimeID, destinationRef, idempotencyKey string) (InstallStartResult, error) {
	if s == nil || s.db == nil || ValidateIdempotencyKey(idempotencyKey) != nil {
		return InstallStartResult{}, ErrManagementQueryFailed
	}
	if err := yorvaruntime.ValidateBackupDestinationRef(destinationRef); err != nil {
		return InstallStartResult{}, ErrManagementQueryFailed
	}
	target, err := s.resolve(ctx, runtimeID)
	if err != nil {
		return InstallStartResult{}, err
	}
	unlock := lockRuntimeManagementTarget(s.targets, target.InstallationID)
	defer unlock()
	if target.Bundle.BackupMutate == nil {
		return InstallStartResult{}, ErrManagementCapabilityUnsupported
	}
	if existing, ok, queryErr := s.db.GetOperationByIdempotencyKey(ctx, idempotencyKey); queryErr != nil {
		return InstallStartResult{}, managementBackupError(ctx, queryErr)
	} else if ok {
		if existing.Type != operation.TypeBackupCreate || existing.TargetID != target.InstallationID {
			return InstallStartResult{}, ErrManagementQueryFailed
		}
		return InstallStartResult{Operation: existing}, nil
	}
	id, err := s.newID()
	if err != nil {
		return InstallStartResult{}, ErrManagementQueryFailed
	}
	correlationID, err := newCorrelationID()
	if err != nil {
		return InstallStartResult{}, ErrManagementQueryFailed
	}
	now := s.now()
	op := operation.Operation{
		ID: id, Type: operation.TypeBackupCreate, TargetType: operation.TargetRuntimeInstallation,
		TargetID: target.InstallationID, Status: operation.StatusPending, Stage: operation.StageBackupPreflight,
		IdempotencyKey: idempotencyKey, CorrelationID: correlationID, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.db.CreateOperation(ctx, op); err != nil {
		if errors.Is(err, sqlite.ErrActiveInstanceMutation) || errors.Is(err, sqlite.ErrActiveInstallExists) || errors.Is(err, sqlite.ErrDuplicateIdempotency) {
			return InstallStartResult{}, ErrBackupMutationConflict
		}
		return InstallStartResult{}, managementBackupError(ctx, err)
	}
	s.emitOperation(operation.Operation{}, op, true)
	go s.runCreateBackup(op, target, destinationRef)
	return InstallStartResult{Operation: op, Created: true}, nil
}

func (s *BackupManagement) StartDeleteBackup(ctx context.Context, runtimeID, backupID, idempotencyKey string) (InstallStartResult, error) {
	if s == nil || s.db == nil || ValidateIdempotencyKey(idempotencyKey) != nil || !validManagementBackupID(backupID) {
		return InstallStartResult{}, ErrManagementQueryFailed
	}
	target, err := s.resolve(ctx, runtimeID)
	if err != nil {
		return InstallStartResult{}, err
	}
	unlock := lockRuntimeManagementTarget(s.targets, target.InstallationID)
	defer unlock()
	if target.Bundle.BackupMutate == nil {
		return InstallStartResult{}, ErrManagementCapabilityUnsupported
	}
	if _, err := s.InspectBackup(ctx, runtimeID, backupID); err != nil {
		return InstallStartResult{}, err
	}
	if existing, ok, queryErr := s.db.GetOperationByIdempotencyKey(ctx, idempotencyKey); queryErr != nil {
		return InstallStartResult{}, managementBackupError(ctx, queryErr)
	} else if ok {
		if existing.Type != operation.TypeBackupDelete || existing.TargetID != target.InstallationID || existing.Message != backupID {
			return InstallStartResult{}, ErrBackupMutationConflict
		}
		return InstallStartResult{Operation: existing}, nil
	}
	id, err := s.newID()
	if err != nil {
		return InstallStartResult{}, ErrManagementQueryFailed
	}
	correlationID, err := newCorrelationID()
	if err != nil {
		return InstallStartResult{}, ErrManagementQueryFailed
	}
	now := s.now()
	op := operation.Operation{
		ID: id, Type: operation.TypeBackupDelete, TargetType: operation.TargetRuntimeInstallation,
		TargetID: target.InstallationID, Status: operation.StatusPending, Stage: operation.StageBackupPreflight,
		Message: backupID, IdempotencyKey: idempotencyKey, CorrelationID: correlationID, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.db.CreateOperation(ctx, op); err != nil {
		if errors.Is(err, sqlite.ErrActiveInstanceMutation) || errors.Is(err, sqlite.ErrDuplicateIdempotency) {
			return InstallStartResult{}, ErrBackupMutationConflict
		}
		return InstallStartResult{}, managementBackupError(ctx, err)
	}
	s.emitOperation(operation.Operation{}, op, true)
	go s.runDeleteBackup(op, target, backupID)
	return InstallStartResult{Operation: op, Created: true}, nil
}

func (s *BackupManagement) runDeleteBackup(op operation.Operation, target RuntimeManagementTarget, backupID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	if !s.registerCancel(op.ID, cancel) {
		cancel()
		return
	}
	defer s.unregisterCancel(op.ID, cancel)
	current, err := s.db.GetOperation(ctx, op.ID)
	if err != nil || operation.IsTerminal(current.Status) {
		return
	}
	now := s.now()
	running := current
	running.Status, running.Stage, running.StartedAt, running.UpdatedAt = operation.StatusRunning, operation.StageBackupDelete, &now, now
	if err := s.persistOperation(ctx, current, running); err != nil {
		return
	}
	err = target.Bundle.BackupMutate.DeleteBackup(ctx, target.Installation, backupID, nil)
	completed := s.now()
	final := running
	final.Stage, final.CompletedAt, final.UpdatedAt = operation.StageBackupReconcile, &completed, completed
	if err != nil {
		final.Status, final.ErrorCode, final.Retryable = operation.StatusFailed, yorvaruntime.ErrorBackupDeleteFailed, true
	} else {
		final.Status = operation.StatusSucceeded
	}
	_ = s.persistOperation(context.Background(), running, final)
}

func (s *BackupManagement) StartRestoreBackup(ctx context.Context, runtimeID, backupID, idempotencyKey string) (InstallStartResult, error) {
	if s == nil || s.db == nil || ValidateIdempotencyKey(idempotencyKey) != nil || !validManagementBackupID(backupID) {
		return InstallStartResult{}, ErrManagementQueryFailed
	}
	target, err := s.resolve(ctx, runtimeID)
	if err != nil {
		return InstallStartResult{}, err
	}
	unlock := lockRuntimeManagementTarget(s.targets, target.InstallationID)
	defer unlock()
	if target.Bundle.Restore == nil {
		return InstallStartResult{}, ErrManagementCapabilityUnsupported
	}
	if _, err := s.InspectBackup(ctx, runtimeID, backupID); err != nil {
		return InstallStartResult{}, err
	}
	if existing, ok, queryErr := s.db.GetOperationByIdempotencyKey(ctx, idempotencyKey); queryErr != nil {
		return InstallStartResult{}, managementBackupError(ctx, queryErr)
	} else if ok {
		if existing.Type != operation.TypeBackupRestore || existing.TargetID != target.InstallationID || existing.Message != backupID {
			return InstallStartResult{}, ErrBackupMutationConflict
		}
		return InstallStartResult{Operation: existing}, nil
	}
	id, err := s.newID()
	if err != nil {
		return InstallStartResult{}, ErrManagementQueryFailed
	}
	correlationID, err := newCorrelationID()
	if err != nil {
		return InstallStartResult{}, ErrManagementQueryFailed
	}
	now := s.now()
	op := operation.Operation{
		ID: id, Type: operation.TypeBackupRestore, TargetType: operation.TargetRuntimeInstallation,
		TargetID: target.InstallationID, Status: operation.StatusPending, Stage: operation.StageRestorePreflight,
		Message: backupID, IdempotencyKey: idempotencyKey, CorrelationID: correlationID, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.db.CreateOperation(ctx, op); err != nil {
		if errors.Is(err, sqlite.ErrActiveInstanceMutation) || errors.Is(err, sqlite.ErrDuplicateIdempotency) {
			return InstallStartResult{}, ErrBackupMutationConflict
		}
		return InstallStartResult{}, managementBackupError(ctx, err)
	}
	s.emitOperation(operation.Operation{}, op, true)
	go s.runRestoreBackup(op, target, backupID)
	return InstallStartResult{Operation: op, Created: true}, nil
}

func (s *BackupManagement) runRestoreBackup(op operation.Operation, target RuntimeManagementTarget, backupID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	if !s.registerCancel(op.ID, cancel) {
		cancel()
		return
	}
	defer s.unregisterCancel(op.ID, cancel)
	current, err := s.db.GetOperation(ctx, op.ID)
	if err != nil || operation.IsTerminal(current.Status) {
		return
	}
	now := s.now()
	running := current
	running.Status, running.Stage, running.StartedAt, running.UpdatedAt = operation.StatusRunning, operation.StageRestoreProtection, &now, now
	if err := s.persistOperation(ctx, current, running); err != nil {
		return
	}
	result, err := target.Bundle.Restore.RestoreBackup(ctx, target.Installation, yorvaruntime.BackupRestoreRequest{BackupID: backupID}, nil)
	completed := s.now()
	final := running
	final.Stage, final.CompletedAt, final.UpdatedAt = operation.StageRestoreReconcile, &completed, completed
	if result.State.Valid() {
		final.Message = string(result.State)
	}
	if err != nil || result.State != yorvaruntime.RestoreSucceeded {
		final.Status, final.ErrorCode, final.Retryable = operation.StatusFailed, yorvaruntime.ErrorBackupRestoreFailed, result.State != yorvaruntime.RestoreRecoveryRequired
	} else {
		final.Status = operation.StatusSucceeded
	}
	_ = s.persistOperation(context.Background(), running, final)
}

func (s *BackupManagement) runCreateBackup(op operation.Operation, target RuntimeManagementTarget, destinationRef string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	if !s.registerCancel(op.ID, cancel) {
		cancel()
		return
	}
	defer s.unregisterCancel(op.ID, cancel)
	current, err := s.db.GetOperation(ctx, op.ID)
	if err != nil || operation.IsTerminal(current.Status) {
		return
	}
	now := s.now()
	running := current
	running.Status, running.Stage, running.StartedAt, running.UpdatedAt = operation.StatusRunning, operation.StageBackupSnapshot, &now, now
	if err := s.persistOperation(ctx, current, running); err != nil {
		return
	}
	_, err = target.Bundle.BackupMutate.CreateBackup(ctx, target.Installation, yorvaruntime.BackupCreateRequest{
		DestinationRef: destinationRef, OperationID: op.ID, RuntimeInstallationID: target.InstallationID,
	}, nil)
	completed := s.now()
	final := running
	final.Stage, final.CompletedAt, final.UpdatedAt = operation.StageBackupReconcile, &completed, completed
	if err != nil {
		final.Status, final.ErrorCode, final.Retryable = operation.StatusFailed, yorvaruntime.ErrorBackupCreateFailed, true
	} else {
		final.Status = operation.StatusSucceeded
	}
	_ = s.persistOperation(context.Background(), running, final)
}

func (s *BackupManagement) CancelBackupOperation(ctx context.Context, operationID string) (operation.Operation, error) {
	if s == nil || s.db == nil {
		return operation.Operation{}, ErrManagementCapabilityUnsupported
	}
	current, err := s.db.GetOperation(ctx, operationID)
	if err != nil {
		return operation.Operation{}, ErrManagementQueryFailed
	}
	if current.Type != operation.TypeBackupCreate && current.Type != operation.TypeBackupDelete && current.Type != operation.TypeBackupRestore {
		return operation.Operation{}, ErrManagementCapabilityUnsupported
	}
	if operation.IsTerminal(current.Status) {
		return operation.Operation{}, ErrInstanceNotCancellable
	}
	s.mu.Lock()
	cancel := s.cancels[operationID]
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	now := s.now()
	next := current
	next.Status, next.ErrorCode, next.Retryable = operation.StatusCancelled, "", false
	next.CompletedAt, next.UpdatedAt = &now, now
	if err := s.persistOperation(ctx, current, next); err != nil {
		return operation.Operation{}, ErrInstanceNotCancellable
	}
	return next, nil
}

func (s *BackupManagement) registerCancel(operationID string, cancel context.CancelFunc) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.cancels[operationID]; exists {
		return false
	}
	s.cancels[operationID] = cancel
	return true
}

func (s *BackupManagement) unregisterCancel(operationID string, cancel context.CancelFunc) {
	cancel()
	s.mu.Lock()
	delete(s.cancels, operationID)
	s.mu.Unlock()
}

func (s *BackupManagement) RecoverInterrupted(ctx context.Context) ([]operation.Operation, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	active, err := s.db.ListActiveBackupOperations(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]operation.Operation, 0, len(active))
	for _, current := range active {
		now := s.now()
		next := current
		next.Status, next.Stage = operation.StatusFailed, operation.StageBackupReconcile
		next.ErrorCode, next.Retryable, next.CompletedAt, next.UpdatedAt = yorvaruntime.ErrorOperationInterrupted, true, &now, now
		if err := s.persistOperation(ctx, current, next); err != nil {
			return result, err
		}
		result = append(result, next)
	}
	return result, nil
}

func (s *BackupManagement) persistOperation(ctx context.Context, current, next operation.Operation) error {
	if err := s.db.UpdateOperation(ctx, current, next); err != nil {
		return err
	}
	s.emitOperation(current, next, false)
	return nil
}

func (s *BackupManagement) emitOperation(previous, next operation.Operation, created bool) {
	if s.events == nil {
		return
	}
	eventType := events.TypeForCommittedOperation(created, string(previous.Status), string(next.Status))
	if eventType != "" {
		s.events.Publish(events.NewOperationEvent(eventType, events.OperationPayload{
			OperationID: next.ID, Type: string(next.Type), Status: string(next.Status), Stage: string(next.Stage),
			ErrorCode: string(next.ErrorCode), CorrelationID: next.CorrelationID,
		}, s.now()))
	}
}

// DeleteBackup is the synchronous adapter boundary used by the durable
// Operation worker. HTTP callers only start and observe the Operation.
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

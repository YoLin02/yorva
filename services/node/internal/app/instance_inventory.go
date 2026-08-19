package app

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/instance"
	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	"github.com/YoLin02/yorva/services/node/internal/persistence/sqlite"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes"
)

const hermesRuntimeID = "hermes"

var (
	ErrRuntimeNotSupported          = errors.New("runtime is not supported for instance inventory")
	ErrInstanceNotFound             = errors.New("instance not found")
	ErrInstanceQueryFailed          = errors.New("instance query failed")
	ErrInstanceOutputUnrecognized   = errors.New("instance output unrecognized")
	ErrInstanceRuntimeNotFound      = errors.New("runtime inventory target not found")
	ErrInstanceInvalidName          = errors.New("instance name is invalid")
	ErrInstanceAlreadyExists        = errors.New("instance already exists")
	ErrInstanceConflict             = errors.New("instance operation conflicts")
	ErrInstanceNotCancellable       = errors.New("instance operation is not cancellable")
	ErrInstanceProtected            = errors.New("instance is protected")
	ErrInstanceConfirmationMismatch = errors.New("instance confirmation does not match")
)

type ProfileSnapshot struct {
	NativeID string
	Default  bool
}

type ProfileSource interface {
	List(ctx context.Context, executable string) ([]ProfileSnapshot, error)
}

type ProfileMutator interface {
	Create(ctx context.Context, executable, name string) error
	Delete(ctx context.Context, executable, nativeID string) error
}

type InstanceCapabilities struct {
	Instances bool `json:"instances"`
	Lifecycle bool `json:"lifecycle"`
}

type InstanceView struct {
	InstanceID            string
	RuntimeInstallationID string
	Name                  string
	Default               bool
	Protected             bool
	Availability          instance.Availability
	LastSyncedAt          *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
	Capabilities          InstanceCapabilities
}

type InstanceList struct {
	RuntimeID             string
	RuntimeInstallationID string
	Freshness             string
	LastSyncedAt          *time.Time
	Instances             []InstanceView
	Capabilities          InstanceCapabilities
	ErrorCode             yorvaruntime.ErrorCode
}

type InstanceInventory struct {
	discovery *RuntimeDiscovery
	db        *sqlite.Database
	source    ProfileSource
	mutator   ProfileMutator
	nodeID    string
	now       func() time.Time
	newID     func() (string, error)
	mu        sync.Mutex
	locks     map[string]*sync.Mutex
	workers   map[string]context.CancelFunc
	started   map[string]bool
}

func NewInstanceInventory(discovery *RuntimeDiscovery, db *sqlite.Database, source ProfileSource, nodeID string) *InstanceInventory {
	return &InstanceInventory{
		discovery: discovery,
		db:        db,
		source:    source,
		nodeID:    nodeID,
		now:       func() time.Time { return time.Now().UTC() },
		newID:     newOperationID,
		locks:     make(map[string]*sync.Mutex),
		workers:   make(map[string]context.CancelFunc),
		started:   make(map[string]bool),
	}
}

func (s *InstanceInventory) WithMutator(mutator ProfileMutator) *InstanceInventory {
	s.mutator = mutator
	return s
}

func (s *InstanceInventory) ListInstances(ctx context.Context, runtimeID string) (InstanceList, error) {
	if runtimeID != hermesRuntimeID {
		return InstanceList{}, ErrInstanceRuntimeNotFound
	}
	if s.discovery == nil || s.db == nil || s.source == nil {
		return InstanceList{}, ErrRuntimeNotSupported
	}
	discovery, err := s.discovery.Detect(ctx, yorvaruntime.Kind(hermesRuntimeID))
	if err != nil {
		return InstanceList{}, err
	}
	if discovery.State != yorvaruntime.DiscoverySupported || discovery.Selected == nil || discovery.Selected.Path == "" {
		return InstanceList{}, ErrRuntimeNotSupported
	}
	installation, err := s.ensureInstallation(ctx, discovery)
	if err != nil {
		return InstanceList{}, err
	}

	unlock := s.lockInstallation(installation.ID)
	defer unlock()

	now := s.now()
	freshness := "FRESH"
	var queryErr error
	natives, listErr := s.source.List(ctx, discovery.Selected.Path)
	if listErr != nil {
		queryErr = classifyProfileListError(listErr)
		if markErr := s.db.MarkInstancesUnknown(ctx, installation.ID, now); markErr != nil {
			return InstanceList{}, markErr
		}
		freshness = "UNKNOWN"
	} else {
		entries := make([]sqlite.InstanceSnapshotEntry, 0, len(natives))
		for _, native := range natives {
			entries = append(entries, sqlite.InstanceSnapshotEntry{NativeID: native.NativeID, Default: native.Default || native.NativeID == "default"})
		}
		if err := s.db.ApplyInstanceSnapshot(ctx, installation.ID, entries, now); err != nil {
			return InstanceList{}, err
		}
	}

	rows, err := s.db.ListInstances(ctx, installation.ID)
	if err != nil {
		return InstanceList{}, err
	}
	views := make([]InstanceView, 0, len(rows))
	var lastSync *time.Time
	for _, row := range rows {
		views = append(views, instanceView(row))
		if row.LastSyncedAt != nil && (lastSync == nil || row.LastSyncedAt.After(*lastSync)) {
			lastSync = row.LastSyncedAt
		}
	}
	result := InstanceList{
		RuntimeID:             hermesRuntimeID,
		RuntimeInstallationID: installation.ID,
		Freshness:             freshness,
		LastSyncedAt:          lastSync,
		Instances:             views,
		Capabilities:          InstanceCapabilities{Instances: true, Lifecycle: false},
	}
	if queryErr != nil {
		if errors.Is(queryErr, ErrInstanceOutputUnrecognized) {
			result.ErrorCode = yorvaruntime.ErrorInstanceOutputUnrecognized
		} else {
			result.ErrorCode = yorvaruntime.ErrorInstanceQueryFailed
		}
	}
	return result, nil
}

func (s *InstanceInventory) GetInstance(ctx context.Context, instanceID string) (InstanceView, error) {
	if instanceID == "" {
		return InstanceView{}, ErrInstanceNotFound
	}
	row, err := s.db.GetInstance(ctx, instanceID)
	if errors.Is(err, sql.ErrNoRows) {
		return InstanceView{}, ErrInstanceNotFound
	}
	if err != nil {
		return InstanceView{}, err
	}
	return instanceView(row), nil
}

func (s *InstanceInventory) ensureInstallation(ctx context.Context, discovery yorvaruntime.Discovery) (sqlite.AcceptedInstallation, error) {
	now := s.now()
	existing, err := s.db.GetAcceptedInstallation(ctx, s.nodeID, yorvaruntime.Kind(hermesRuntimeID), discovery.Selected.Path)
	if err == nil {
		existing.Version = discovery.Selected.Version
		existing.SupportState = discovery.State
		existing.LastDetectedAt = now
		existing.UpdatedAt = now
		if err := s.db.UpsertAcceptedInstallation(ctx, existing); err != nil {
			return sqlite.AcceptedInstallation{}, err
		}
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return sqlite.AcceptedInstallation{}, err
	}
	id, err := sqlite.NewInstallationID()
	if err != nil {
		return sqlite.AcceptedInstallation{}, err
	}
	created := sqlite.AcceptedInstallation{
		ID:             id,
		NodeID:         s.nodeID,
		RuntimeKind:    yorvaruntime.Kind(hermesRuntimeID),
		InstallPath:    discovery.Selected.Path,
		Version:        discovery.Selected.Version,
		SupportState:   discovery.State,
		Status:         "ACCEPTED",
		LastDetectedAt: now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.db.UpsertAcceptedInstallation(ctx, created); err != nil {
		return sqlite.AcceptedInstallation{}, err
	}
	return created, nil
}

func (s *InstanceInventory) lockInstallation(id string) func() {
	s.mu.Lock()
	lock, ok := s.locks[id]
	if !ok {
		lock = &sync.Mutex{}
		s.locks[id] = lock
	}
	s.mu.Unlock()
	lock.Lock()
	return lock.Unlock
}

func instanceView(row instance.Instance) InstanceView {
	return InstanceView{
		InstanceID:            row.ID,
		RuntimeInstallationID: row.RuntimeInstallationID,
		Name:                  row.Name,
		Default:               row.Default,
		Protected:             row.Protected,
		Availability:          row.Availability,
		LastSyncedAt:          row.LastSyncedAt,
		CreatedAt:             row.CreatedAt,
		UpdatedAt:             row.UpdatedAt,
		Capabilities:          InstanceCapabilities{Instances: true, Lifecycle: false},
	}
}

func classifyProfileListError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrInstanceOutputUnrecognized) {
		return ErrInstanceOutputUnrecognized
	}
	return ErrInstanceQueryFailed
}

func (s *InstanceInventory) StartCreate(ctx context.Context, runtimeID, name, idempotencyKey string) (InstallStartResult, error) {
	if err := ValidateIdempotencyKey(idempotencyKey); err != nil {
		return InstallStartResult{}, err
	}
	if err := hermes.ValidateCreateProfileName(name); err != nil {
		return InstallStartResult{}, ErrInstanceInvalidName
	}
	if s.mutator == nil {
		return InstallStartResult{}, ErrRuntimeNotSupported
	}
	listed, err := s.ListInstances(ctx, runtimeID)
	if err != nil {
		return InstallStartResult{}, err
	}
	if existing, ok, err := s.db.GetOperationByIdempotencyKey(ctx, idempotencyKey); err != nil {
		return InstallStartResult{}, err
	} else if ok {
		if existing.Type != operation.TypeInstanceCreate || existing.Message != name {
			return InstallStartResult{}, ErrInstanceConflict
		}
		return InstallStartResult{Operation: existing}, nil
	}
	for _, item := range listed.Instances {
		if item.Name == name && item.Availability == instance.Available {
			return InstallStartResult{}, ErrInstanceAlreadyExists
		}
	}
	if active, ok, err := s.db.ActiveInstanceMutation(ctx, listed.RuntimeInstallationID); err != nil {
		return InstallStartResult{}, err
	} else if ok {
		return InstallStartResult{}, instanceConflict(active.ID)
	}

	now := s.now()
	id, err := s.newID()
	if err != nil {
		return InstallStartResult{}, err
	}
	correlation, err := newCorrelationID()
	if err != nil {
		return InstallStartResult{}, err
	}
	op := operation.Operation{
		ID:             id,
		Type:           operation.TypeInstanceCreate,
		TargetType:     operation.TargetRuntimeInstallation,
		TargetID:       listed.RuntimeInstallationID,
		Status:         operation.StatusPending,
		Stage:          operation.StagePreflight,
		Message:        name,
		IdempotencyKey: idempotencyKey,
		CorrelationID:  correlation,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.db.CreateOperation(ctx, op); err != nil {
		if errors.Is(err, sqlite.ErrDuplicateIdempotency) {
			existing, ok, getErr := s.db.GetOperationByIdempotencyKey(ctx, idempotencyKey)
			if getErr == nil && ok {
				if existing.Message != name {
					return InstallStartResult{}, ErrInstanceConflict
				}
				return InstallStartResult{Operation: existing}, nil
			}
		}
		if errors.Is(err, sqlite.ErrActiveInstanceMutation) {
			return InstallStartResult{}, ErrInstanceConflict
		}
		return InstallStartResult{}, err
	}
	s.startCreateWorker(op, listed.RuntimeInstallationID, name)
	return InstallStartResult{Operation: op, Created: true}, nil
}

func (s *InstanceInventory) CancelCreate(ctx context.Context, operationID string) (operation.Operation, error) {
	current, err := s.db.GetOperation(ctx, operationID)
	if err != nil {
		return operation.Operation{}, err
	}
	if current.Type != operation.TypeInstanceCreate {
		return operation.Operation{}, ErrInstanceNotFound
	}
	s.mu.Lock()
	started := s.started[operationID]
	cancel := s.workers[operationID]
	s.mu.Unlock()
	if started || current.Status == operation.StatusRunning {
		return current, ErrInstanceNotCancellable
	}
	if cancel != nil {
		cancel()
	}
	if operation.IsTerminal(current.Status) {
		return current, nil
	}
	now := s.now()
	next := current
	next.Status = operation.StatusCancelled
	next.CompletedAt = &now
	next.UpdatedAt = now
	if err := s.db.UpdateOperation(ctx, current, next); err != nil {
		return operation.Operation{}, err
	}
	return next, nil
}

func (s *InstanceInventory) startCreateWorker(op operation.Operation, installationID, name string) {
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.workers[op.ID] = cancel
	s.mu.Unlock()
	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.workers, op.ID)
			s.mu.Unlock()
			cancel()
		}()
		s.runCreate(ctx, op, installationID, name)
	}()
}

func (s *InstanceInventory) runCreate(ctx context.Context, op operation.Operation, installationID, name string) {
	unlock := s.lockInstallation(installationID)
	defer unlock()
	current, err := s.db.GetOperation(ctx, op.ID)
	if err != nil || operation.IsTerminal(current.Status) {
		return
	}
	now := s.now()
	running := current
	running.Status = operation.StatusRunning
	running.Stage = operation.StageInstanceCreate
	running.StartedAt = &now
	running.UpdatedAt = now
	if err := s.db.UpdateOperation(ctx, current, running); err != nil {
		return
	}
	s.mu.Lock()
	s.started[op.ID] = true
	s.mu.Unlock()

	cmdCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	discovery, err := s.discovery.Detect(cmdCtx, yorvaruntime.Kind(hermesRuntimeID))
	if err != nil || discovery.Selected == nil {
		s.failCreate(running, yorvaruntime.ErrorRuntimeNotSupported, false)
		return
	}
	createErr := s.mutator.Create(cmdCtx, discovery.Selected.Path, name)
	_, _ = s.reconcileLocked(cmdCtx, installationID, discovery.Selected.Path)
	present, presentErr := s.profilePresent(ctx, installationID, name)
	if presentErr == nil && present {
		if createErr != nil {
			s.failCreate(running, yorvaruntime.ErrorInstanceAlreadyExists, false)
			return
		}
		s.succeedCreate(running)
		return
	}
	if createErr != nil {
		s.failCreate(running, yorvaruntime.ErrorInstanceQueryFailed, true)
		return
	}
	s.failCreate(running, yorvaruntime.ErrorInstanceQueryFailed, true)
}

func (s *InstanceInventory) reconcileLocked(ctx context.Context, installationID, executable string) (InstanceList, error) {
	now := s.now()
	natives, listErr := s.source.List(ctx, executable)
	if listErr != nil {
		_ = s.db.MarkInstancesUnknown(ctx, installationID, now)
		return InstanceList{}, classifyProfileListError(listErr)
	}
	entries := make([]sqlite.InstanceSnapshotEntry, 0, len(natives))
	for _, native := range natives {
		entries = append(entries, sqlite.InstanceSnapshotEntry{NativeID: native.NativeID, Default: native.Default || native.NativeID == "default"})
	}
	if err := s.db.ApplyInstanceSnapshot(ctx, installationID, entries, now); err != nil {
		return InstanceList{}, err
	}
	rows, err := s.db.ListInstances(ctx, installationID)
	if err != nil {
		return InstanceList{}, err
	}
	views := make([]InstanceView, 0, len(rows))
	for _, row := range rows {
		views = append(views, instanceView(row))
	}
	return InstanceList{RuntimeInstallationID: installationID, Freshness: "FRESH", Instances: views}, nil
}

func (s *InstanceInventory) profilePresent(ctx context.Context, installationID, name string) (bool, error) {
	rows, err := s.db.ListInstances(ctx, installationID)
	if err != nil {
		return false, err
	}
	for _, row := range rows {
		if row.NativeID == name && row.Availability == instance.Available {
			return true, nil
		}
	}
	return false, nil
}

func (s *InstanceInventory) succeedCreate(current operation.Operation) {
	now := s.now()
	next := current
	next.Status = operation.StatusSucceeded
	next.Stage = operation.StageInstanceReconcile
	next.CompletedAt = &now
	next.UpdatedAt = now
	_ = s.db.UpdateOperation(context.Background(), current, next)
}

func (s *InstanceInventory) failCreate(current operation.Operation, code yorvaruntime.ErrorCode, retryable bool) {
	now := s.now()
	next := current
	next.Status = operation.StatusFailed
	next.ErrorCode = code
	next.Retryable = retryable
	next.CompletedAt = &now
	next.UpdatedAt = now
	_ = s.db.UpdateOperation(context.Background(), current, next)
}

func instanceConflict(activeID string) error {
	_ = activeID
	return ErrInstanceConflict
}

func (s *InstanceInventory) StartDelete(ctx context.Context, instanceID, confirmationName, idempotencyKey string) (InstallStartResult, error) {
	if err := ValidateIdempotencyKey(idempotencyKey); err != nil {
		return InstallStartResult{}, err
	}
	row, err := s.db.GetInstance(ctx, instanceID)
	if errors.Is(err, sql.ErrNoRows) {
		return InstallStartResult{}, ErrInstanceNotFound
	}
	if err != nil {
		return InstallStartResult{}, err
	}
	if row.Default || row.Protected || row.NativeID == "default" {
		return InstallStartResult{}, ErrInstanceProtected
	}
	if confirmationName != row.NativeID {
		return InstallStartResult{}, ErrInstanceConfirmationMismatch
	}
	if existing, ok, err := s.db.GetOperationByIdempotencyKey(ctx, idempotencyKey); err != nil {
		return InstallStartResult{}, err
	} else if ok {
		if existing.Type != operation.TypeInstanceDelete || existing.Message != confirmationName || existing.TargetID != row.RuntimeInstallationID {
			return InstallStartResult{}, ErrInstanceConflict
		}
		return InstallStartResult{Operation: existing}, nil
	}

	listed, err := s.ListInstances(ctx, hermesRuntimeID)
	if err != nil {
		return InstallStartResult{}, err
	}
	if listed.RuntimeInstallationID != row.RuntimeInstallationID {
		return InstallStartResult{}, ErrInstanceNotFound
	}
	present := false
	for _, item := range listed.Instances {
		if item.InstanceID == instanceID && item.Availability == instance.Available {
			present = true
		}
	}
	if active, ok, err := s.db.ActiveInstanceMutation(ctx, row.RuntimeInstallationID); err != nil {
		return InstallStartResult{}, err
	} else if ok {
		return InstallStartResult{}, instanceConflict(active.ID)
	}

	now := s.now()
	id, err := s.newID()
	if err != nil {
		return InstallStartResult{}, err
	}
	correlation, err := newCorrelationID()
	if err != nil {
		return InstallStartResult{}, err
	}
	op := operation.Operation{
		ID:             id,
		Type:           operation.TypeInstanceDelete,
		TargetType:     operation.TargetRuntimeInstallation,
		TargetID:       row.RuntimeInstallationID,
		Status:         operation.StatusPending,
		Stage:          operation.StagePreflight,
		Message:        confirmationName,
		IdempotencyKey: idempotencyKey,
		CorrelationID:  correlation,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.db.CreateOperation(ctx, op); err != nil {
		if errors.Is(err, sqlite.ErrDuplicateIdempotency) {
			existing, ok, getErr := s.db.GetOperationByIdempotencyKey(ctx, idempotencyKey)
			if getErr == nil && ok && existing.Message == confirmationName {
				return InstallStartResult{Operation: existing}, nil
			}
			return InstallStartResult{}, ErrInstanceConflict
		}
		if errors.Is(err, sqlite.ErrActiveInstanceMutation) {
			return InstallStartResult{}, ErrInstanceConflict
		}
		return InstallStartResult{}, err
	}
	s.startDeleteWorker(op, row.RuntimeInstallationID, row.NativeID, present)
	return InstallStartResult{Operation: op, Created: true}, nil
}

func (s *InstanceInventory) CancelDelete(ctx context.Context, operationID string) (operation.Operation, error) {
	current, err := s.db.GetOperation(ctx, operationID)
	if err != nil {
		return operation.Operation{}, err
	}
	if current.Type != operation.TypeInstanceDelete {
		return operation.Operation{}, ErrInstanceNotFound
	}
	s.mu.Lock()
	started := s.started[operationID]
	cancel := s.workers[operationID]
	s.mu.Unlock()
	if started || current.Status == operation.StatusRunning {
		return current, ErrInstanceNotCancellable
	}
	if cancel != nil {
		cancel()
	}
	if operation.IsTerminal(current.Status) {
		return current, nil
	}
	now := s.now()
	next := current
	next.Status = operation.StatusCancelled
	next.CompletedAt = &now
	next.UpdatedAt = now
	if err := s.db.UpdateOperation(ctx, current, next); err != nil {
		return operation.Operation{}, err
	}
	return next, nil
}

func (s *InstanceInventory) startDeleteWorker(op operation.Operation, installationID, nativeID string, present bool) {
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.workers[op.ID] = cancel
	s.mu.Unlock()
	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.workers, op.ID)
			s.mu.Unlock()
			cancel()
		}()
		s.runDelete(ctx, op, installationID, nativeID, present)
	}()
}

func (s *InstanceInventory) runDelete(ctx context.Context, op operation.Operation, installationID, nativeID string, present bool) {
	unlock := s.lockInstallation(installationID)
	defer unlock()
	current, err := s.db.GetOperation(ctx, op.ID)
	if err != nil || operation.IsTerminal(current.Status) {
		return
	}
	now := s.now()
	running := current
	running.Status = operation.StatusRunning
	running.Stage = operation.StageInstanceDelete
	running.StartedAt = &now
	running.UpdatedAt = now
	if err := s.db.UpdateOperation(ctx, current, running); err != nil {
		return
	}
	s.mu.Lock()
	s.started[op.ID] = true
	s.mu.Unlock()

	cmdCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	discovery, err := s.discovery.Detect(cmdCtx, yorvaruntime.Kind(hermesRuntimeID))
	if err != nil || discovery.Selected == nil {
		s.failCreate(running, yorvaruntime.ErrorRuntimeNotSupported, false)
		return
	}
	_, _ = s.reconcileLocked(cmdCtx, installationID, discovery.Selected.Path)
	stillPresent, presentErr := s.profilePresent(cmdCtx, installationID, nativeID)
	if presentErr == nil && !stillPresent {
		s.succeedDelete(running)
		return
	}
	if present || stillPresent {
		if err := s.mutator.Delete(cmdCtx, discovery.Selected.Path, nativeID); err != nil && !errors.Is(err, context.Canceled) {
			_, _ = s.reconcileLocked(cmdCtx, installationID, discovery.Selected.Path)
			gone, goneErr := s.profilePresent(cmdCtx, installationID, nativeID)
			if goneErr == nil && !gone {
				s.succeedDelete(running)
				return
			}
			s.failCreate(running, yorvaruntime.ErrorInstanceQueryFailed, true)
			return
		}
	}
	_, _ = s.reconcileLocked(cmdCtx, installationID, discovery.Selected.Path)
	gone, goneErr := s.profilePresent(cmdCtx, installationID, nativeID)
	if goneErr == nil && !gone {
		s.succeedDelete(running)
		return
	}
	s.failCreate(running, yorvaruntime.ErrorInstanceQueryFailed, true)
}

func (s *InstanceInventory) succeedDelete(current operation.Operation) {
	now := s.now()
	next := current
	next.Status = operation.StatusSucceeded
	next.Stage = operation.StageInstanceReconcile
	next.CompletedAt = &now
	next.UpdatedAt = now
	_ = s.db.UpdateOperation(context.Background(), current, next)
}

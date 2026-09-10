package app

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/instance"
	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	"github.com/YoLin02/yorva/services/node/internal/events"
	"github.com/YoLin02/yorva/services/node/internal/persistence/sqlite"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

var (
	ErrRuntimeNotSupported          = errors.New("runtime is not supported for instance inventory")
	ErrInstanceNotFound             = errors.New("instance not found")
	ErrInstanceQueryFailed          = yorvaruntime.ErrInstanceInventoryFailed
	ErrInstanceOutputUnrecognized   = yorvaruntime.ErrInstanceOutputUnrecognized
	ErrInstanceOperationTimedOut    = yorvaruntime.ErrInstanceOperationTimedOut
	ErrInstanceRuntimeNotFound      = errors.New("runtime inventory target not found")
	ErrInstanceInvalidName          = yorvaruntime.ErrInstanceNameInvalid
	ErrInstanceAlreadyExists        = errors.New("instance already exists")
	ErrInstanceConflict             = errors.New("instance operation conflicts")
	ErrInstanceNotCancellable       = errors.New("instance operation is not cancellable")
	ErrInstanceProtected            = errors.New("instance is protected")
	ErrInstanceConfirmationMismatch = errors.New("instance confirmation does not match")
	ErrInstanceRecordNotRemoved     = errors.New("instance record is not removed")
)

type InstanceCapabilities struct {
	Instances     bool                                 `json:"instances"`
	Models        bool                                 `json:"models"`
	Channels      bool                                 `json:"channels"`
	Lifecycle     bool                                 `json:"lifecycle"`
	HealthRead    bool                                 `json:"healthRead"`
	LogsRead      bool                                 `json:"logsRead"`
	SecurityAudit bool                                 `json:"securityAudit"`
	SkillRead     bool                                 `json:"skillRead"`
	SkillMutate   bool                                 `json:"skillMutate"`
	NativeSkills  yorvaruntime.NativeSkillCapabilities `json:"nativeSkills"`
	MCPRead       bool                                 `json:"mcpRead"`
	MCPMutate     bool                                 `json:"mcpMutate"`
	MCPTest       bool                                 `json:"mcpTest"`
	BackupRead    bool                                 `json:"backupRead"`
	BackupMutate  bool                                 `json:"backupMutate"`
	Restore       bool                                 `json:"restore"`
	UpgradePlan   bool                                 `json:"upgradePlan"`
	Upgrade       bool                                 `json:"upgrade"`
	Rollback      bool                                 `json:"rollback"`
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
	discovery    *RuntimeDiscovery
	db           *sqlite.Database
	nodeID       string
	now          func() time.Time
	newID        func() (string, error)
	mu           sync.Mutex
	operationMu  sync.Mutex
	ensureMu     sync.Mutex
	locks        map[string]*sync.Mutex
	workers      map[string]context.CancelFunc
	started      map[string]bool
	events       *events.Broker
	channelQR    *channelQRBroker
	modelSecrets ModelSecretStore
}

func (s *InstanceInventory) WithModelSecrets(store ModelSecretStore) *InstanceInventory {
	s.modelSecrets = store
	return s
}

func NewInstanceInventory(discovery *RuntimeDiscovery, db *sqlite.Database, nodeID string) *InstanceInventory {
	return &InstanceInventory{
		discovery: discovery,
		db:        db,
		nodeID:    nodeID,
		now:       func() time.Time { return time.Now().UTC() },
		newID:     newOperationID,
		locks:     make(map[string]*sync.Mutex),
		workers:   make(map[string]context.CancelFunc),
		started:   make(map[string]bool),
		channelQR: newChannelQRBroker(),
	}
}

func (s *InstanceInventory) WithEvents(broker *events.Broker) *InstanceInventory {
	s.events = broker
	return s
}

func (s *InstanceInventory) ListInstances(ctx context.Context, runtimeID string) (InstanceList, error) {
	if s.discovery == nil || s.db == nil || s.discovery.registry == nil {
		return InstanceList{}, ErrRuntimeNotSupported
	}
	kind := yorvaruntime.Kind(runtimeID)
	bundle, ok := s.discovery.registry.Get(kind)
	if !ok {
		return InstanceList{}, ErrInstanceRuntimeNotFound
	}
	if bundle.Instances == nil {
		return InstanceList{}, ErrManagementCapabilityUnsupported
	}
	discovery, err := s.discovery.Detect(ctx, kind)
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
	natives, listErr := bundle.Instances.List(ctx, discovery.Selected.Path)
	if listErr != nil {
		queryErr = classifyProfileListError(listErr)
		if markErr := s.db.MarkInstancesUnknown(ctx, installation.ID, now); markErr != nil {
			return InstanceList{}, markErr
		}
		freshness = "UNKNOWN"
	} else {
		entries := make([]sqlite.InstanceSnapshotEntry, 0, len(natives))
		for _, native := range natives {
			entries = append(entries, sqlite.InstanceSnapshotEntry{NativeID: native.NativeID, Default: native.Default, Protected: native.Protected})
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
	installationTarget := yorvaruntime.Installation{
		RuntimeKind:  kind,
		Path:         discovery.Selected.Path,
		Version:      discovery.Selected.Version,
		SupportState: discovery.State,
	}
	capabilities := bundleInstanceCapabilities(bundle)
	var lastSync *time.Time
	for _, row := range rows {
		resolved := bundle.ResolveInstanceManagement(ctx, installationTarget, row.NativeID)
		rowCapabilities := bundleInstanceCapabilities(resolved)
		views = append(views, instanceView(row, rowCapabilities))
		capabilities = unionInstanceCapabilities(capabilities, rowCapabilities)
		if row.LastSyncedAt != nil && (lastSync == nil || row.LastSyncedAt.After(*lastSync)) {
			lastSync = row.LastSyncedAt
		}
	}
	result := InstanceList{
		RuntimeID:             runtimeID,
		RuntimeInstallationID: installation.ID,
		Freshness:             freshness,
		LastSyncedAt:          lastSync,
		Instances:             views,
		Capabilities:          capabilities,
	}
	if queryErr != nil {
		result.ErrorCode = errorCodeFrom(queryErr)
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
	capabilities := InstanceCapabilities{}
	if target, targetErr := s.ResolveManagementTarget(ctx, instanceID); targetErr == nil {
		capabilities = bundleInstanceCapabilities(target.Bundle)
	} else if ctxErr := ctx.Err(); ctxErr != nil {
		return InstanceView{}, ctxErr
	}
	return instanceView(row, capabilities), nil
}

func (s *InstanceInventory) ensureInstallation(ctx context.Context, discovery yorvaruntime.Discovery) (sqlite.AcceptedInstallation, error) {
	s.ensureMu.Lock()
	defer s.ensureMu.Unlock()
	now := s.now()
	existing, err := s.db.GetAcceptedInstallation(ctx, s.nodeID, discovery.RuntimeKind, discovery.Selected.Path)
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
		RuntimeKind:    discovery.RuntimeKind,
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
	stored, err := s.db.GetAcceptedInstallation(ctx, s.nodeID, discovery.RuntimeKind, discovery.Selected.Path)
	if err != nil {
		return sqlite.AcceptedInstallation{}, err
	}
	return stored, nil
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

func (s *InstanceInventory) lockInstance(id string) func() {
	return s.lockInstallation("instance:" + id)
}

func instanceView(row instance.Instance, capabilities InstanceCapabilities) InstanceView {
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
		Capabilities:          capabilities,
	}
}

func (s *InstanceInventory) capabilities(kind yorvaruntime.Kind) InstanceCapabilities {
	capabilities := InstanceCapabilities{}
	if s == nil || s.discovery == nil || s.discovery.registry == nil {
		return capabilities
	}
	bundle, ok := s.discovery.registry.Get(kind)
	if !ok {
		return capabilities
	}
	capabilities = bundleInstanceCapabilities(bundle)
	// BackupRead is backed by the Runtime-scoped SQLite index and therefore is
	// published dynamically by the application composition, not Hermes static
	// registration. Mutating backup and Restore capabilities remain false.
	capabilities.BackupRead = capabilities.BackupRead || s.db != nil
	return capabilities
}

func bundleInstanceCapabilities(bundle yorvaruntime.Bundle) InstanceCapabilities {
	capabilities := instanceCapabilities(bundle.ManagementCapabilities(), bundle.Lifecycle != nil)
	capabilities.Instances = bundle.Instances != nil
	capabilities.Models = bundle.Models != nil
	capabilities.Channels = bundle.Channels != nil
	capabilities.NativeSkills = bundle.NativeSkillCapabilities
	return capabilities
}

func instanceCapabilities(management yorvaruntime.ManagementCapabilities, lifecycle bool) InstanceCapabilities {
	capabilities := InstanceCapabilities{Instances: true, Lifecycle: lifecycle}
	capabilities.HealthRead = management.HealthRead
	capabilities.LogsRead = management.LogsRead
	capabilities.SecurityAudit = management.SecurityAudit
	capabilities.SkillRead = management.SkillRead
	capabilities.SkillMutate = management.SkillMutate
	capabilities.MCPRead = management.MCPRead
	capabilities.MCPMutate = management.MCPMutate
	capabilities.MCPTest = management.MCPTest
	capabilities.BackupRead = management.BackupRead
	capabilities.BackupMutate = management.BackupMutate
	capabilities.Restore = management.Restore
	capabilities.UpgradePlan = management.UpgradePlan
	capabilities.Upgrade = management.Upgrade
	capabilities.Rollback = management.Rollback
	return capabilities
}

func unionInstanceCapabilities(left, right InstanceCapabilities) InstanceCapabilities {
	return InstanceCapabilities{
		Instances: left.Instances || right.Instances, Lifecycle: left.Lifecycle || right.Lifecycle,
		Models: left.Models || right.Models, Channels: left.Channels || right.Channels,
		HealthRead: left.HealthRead || right.HealthRead, LogsRead: left.LogsRead || right.LogsRead,
		SecurityAudit: left.SecurityAudit || right.SecurityAudit,
		SkillRead:     left.SkillRead || right.SkillRead, SkillMutate: left.SkillMutate || right.SkillMutate,
		NativeSkills: unionNativeSkillCapabilities(left.NativeSkills, right.NativeSkills),
		MCPRead:      left.MCPRead || right.MCPRead, MCPMutate: left.MCPMutate || right.MCPMutate,
		MCPTest:    left.MCPTest || right.MCPTest,
		BackupRead: left.BackupRead || right.BackupRead, BackupMutate: left.BackupMutate || right.BackupMutate,
		Restore: left.Restore || right.Restore, UpgradePlan: left.UpgradePlan || right.UpgradePlan,
		Upgrade: left.Upgrade || right.Upgrade, Rollback: left.Rollback || right.Rollback,
	}
}

func unionNativeSkillCapabilities(left, right yorvaruntime.NativeSkillCapabilities) yorvaruntime.NativeSkillCapabilities {
	return yorvaruntime.NativeSkillCapabilities{
		Inventory:            unionNativeSkillCapability(left.Inventory, right.Inventory),
		NativeInstall:        unionNativeSkillCapability(left.NativeInstall, right.NativeInstall),
		NativeUpdate:         unionNativeSkillCapability(left.NativeUpdate, right.NativeUpdate),
		NativeRemove:         unionNativeSkillCapability(left.NativeRemove, right.NativeRemove),
		NativeEnableDisable:  unionNativeSkillCapability(left.NativeEnableDisable, right.NativeEnableDisable),
		NativeProfileBinding: unionNativeSkillCapability(left.NativeProfileBinding, right.NativeProfileBinding),
	}
}

func unionNativeSkillCapability(left, right yorvaruntime.NativeSkillCapability) yorvaruntime.NativeSkillCapability {
	if left.Supported || left.Reason != "" {
		return left
	}
	return right
}

func classifyProfileListError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrInstanceOutputUnrecognized) {
		return ErrInstanceOutputUnrecognized
	}
	if errors.Is(err, ErrInstanceOperationTimedOut) || errors.Is(err, context.DeadlineExceeded) {
		return ErrInstanceOperationTimedOut
	}
	return ErrInstanceQueryFailed
}

func errorCodeFrom(err error) yorvaruntime.ErrorCode {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrInstanceOutputUnrecognized):
		return yorvaruntime.ErrorInstanceOutputUnrecognized
	case errors.Is(err, ErrInstanceOperationTimedOut), errors.Is(err, context.DeadlineExceeded):
		return yorvaruntime.ErrorInstanceOperationTimedOut
	default:
		return yorvaruntime.ErrorInstanceQueryFailed
	}
}

func instanceQueryError(code yorvaruntime.ErrorCode) error {
	switch code {
	case yorvaruntime.ErrorInstanceOutputUnrecognized:
		return ErrInstanceOutputUnrecognized
	case yorvaruntime.ErrorInstanceOperationTimedOut:
		return ErrInstanceOperationTimedOut
	default:
		return ErrInstanceQueryFailed
	}
}

func instanceAvailable(listed InstanceList, nativeID string) bool {
	for _, item := range listed.Instances {
		if item.Name == nativeID && item.Availability == instance.Available {
			return true
		}
	}
	return false
}

func (s *InstanceInventory) StartCreate(ctx context.Context, runtimeID, name, idempotencyKey string) (InstallStartResult, error) {
	if err := ValidateIdempotencyKey(idempotencyKey); err != nil {
		return InstallStartResult{}, err
	}
	if s.discovery == nil || s.discovery.registry == nil {
		return InstallStartResult{}, ErrRuntimeNotSupported
	}
	bundle, ok := s.discovery.registry.Get(yorvaruntime.Kind(runtimeID))
	if !ok {
		return InstallStartResult{}, ErrInstanceRuntimeNotFound
	}
	if bundle.Instances == nil {
		return InstallStartResult{}, ErrManagementCapabilityUnsupported
	}
	if err := bundle.Instances.ValidateName(name); err != nil {
		return InstallStartResult{}, ErrInstanceInvalidName
	}
	listed, err := s.ListInstances(ctx, runtimeID)
	if err != nil {
		return InstallStartResult{}, err
	}
	if existing, ok, err := s.db.GetOperationByIdempotencyKey(ctx, idempotencyKey); err != nil {
		return InstallStartResult{}, err
	} else if ok {
		if existing.Type != operation.TypeInstanceCreate || existing.Message != name || existing.TargetID != listed.RuntimeInstallationID {
			return InstallStartResult{}, ErrInstanceConflict
		}
		return InstallStartResult{Operation: existing}, nil
	}
	if listed.Freshness != "FRESH" {
		return InstallStartResult{}, instanceQueryError(listed.ErrorCode)
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
				if existing.Type != operation.TypeInstanceCreate || existing.Message != name || existing.TargetID != listed.RuntimeInstallationID {
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

	cmdCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	target, err := s.resolveAcceptedInstallation(cmdCtx, installationID)
	if err != nil || target.Bundle.Instances == nil {
		s.failCreate(running, yorvaruntime.ErrorRuntimeNotSupported, false)
		return
	}
	createErr := target.Bundle.Instances.Create(cmdCtx, target.Installation.Path, name)
	_, _ = s.reconcileLocked(cmdCtx, installationID, target.Installation.Path)
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
	target, err := s.resolveAcceptedInstallation(ctx, installationID)
	if err != nil || target.Installation.Path != executable || target.Bundle.Instances == nil {
		_ = s.db.MarkInstancesUnknown(ctx, installationID, now)
		return InstanceList{}, ErrRuntimeNotSupported
	}
	natives, listErr := target.Bundle.Instances.List(ctx, executable)
	if listErr != nil {
		_ = s.db.MarkInstancesUnknown(ctx, installationID, now)
		return InstanceList{}, classifyProfileListError(listErr)
	}
	entries := make([]sqlite.InstanceSnapshotEntry, 0, len(natives))
	for _, native := range natives {
		entries = append(entries, sqlite.InstanceSnapshotEntry{NativeID: native.NativeID, Default: native.Default, Protected: native.Protected})
	}
	if err := s.db.ApplyInstanceSnapshot(ctx, installationID, entries, now); err != nil {
		return InstanceList{}, err
	}
	rows, err := s.db.ListInstances(ctx, installationID)
	if err != nil {
		return InstanceList{}, err
	}
	views := make([]InstanceView, 0, len(rows))
	capabilities := s.capabilities(target.Installation.RuntimeKind)
	for _, row := range rows {
		views = append(views, instanceView(row, capabilities))
	}
	return InstanceList{RuntimeInstallationID: installationID, Freshness: "FRESH", Instances: views, Capabilities: capabilities}, nil
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

// ClearRemovedInstance deletes YORVA-owned metadata only after a fresh Runtime
// reconciliation still confirms that the Runtime-owned profile is absent.
func (s *InstanceInventory) ClearRemovedInstance(ctx context.Context, instanceID string) error {
	row, err := s.db.GetInstance(ctx, instanceID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInstanceNotFound
	}
	if err != nil {
		return err
	}
	if row.Default || row.Protected {
		return ErrInstanceProtected
	}

	unlock := s.lockInstallation(row.RuntimeInstallationID)
	defer unlock()

	target, err := s.resolveAcceptedInstallation(ctx, row.RuntimeInstallationID)
	if err != nil {
		return err
	}
	if _, err := s.reconcileLocked(ctx, row.RuntimeInstallationID, target.Installation.Path); err != nil {
		return err
	}

	current, err := s.db.GetInstance(ctx, instanceID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInstanceNotFound
	}
	if err != nil {
		return err
	}
	if current.Availability != instance.Missing {
		return ErrInstanceRecordNotRemoved
	}
	if _, active, err := s.db.ActiveInstanceRuntimeMutation(ctx, instanceID); err != nil {
		return err
	} else if active {
		return ErrInstanceConflict
	}

	deleted, err := s.db.DeleteMissingInstanceRecord(ctx, instanceID)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrInstanceRecordNotRemoved
	}
	return nil
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
	if row.Default || row.Protected {
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

	target, err := s.resolveAcceptedInstallation(ctx, row.RuntimeInstallationID)
	if err != nil {
		return InstallStartResult{}, err
	}
	listed, err := s.ListInstances(ctx, string(target.Installation.RuntimeKind))
	if err != nil {
		return InstallStartResult{}, err
	}
	if listed.RuntimeInstallationID != row.RuntimeInstallationID {
		return InstallStartResult{}, ErrInstanceNotFound
	}
	if listed.Freshness != "FRESH" {
		return InstallStartResult{}, instanceQueryError(listed.ErrorCode)
	}
	s.operationMu.Lock()
	defer s.operationMu.Unlock()
	if target.Bundle.Lifecycle != nil {
		lifecycle, lifecycleErr := s.GetLifecycle(ctx, instanceID)
		if lifecycleErr != nil || lifecycle.State != yorvaruntime.LifecycleStopped {
			return InstallStartResult{}, ErrInstanceConflict
		}
		if _, active, activeErr := s.db.ActiveInstanceRuntimeMutation(ctx, instanceID); activeErr != nil {
			return InstallStartResult{}, activeErr
		} else if active {
			return InstallStartResult{}, ErrInstanceConflict
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
			if getErr == nil && ok && existing.Type == operation.TypeInstanceDelete && existing.Message == confirmationName && existing.TargetID == row.RuntimeInstallationID {
				return InstallStartResult{Operation: existing}, nil
			}
			return InstallStartResult{}, ErrInstanceConflict
		}
		if errors.Is(err, sqlite.ErrActiveInstanceMutation) {
			return InstallStartResult{}, ErrInstanceConflict
		}
		return InstallStartResult{}, err
	}
	s.startDeleteWorker(op, row.RuntimeInstallationID, row.NativeID)
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

func (s *InstanceInventory) startDeleteWorker(op operation.Operation, installationID, nativeID string) {
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
		s.runDelete(ctx, op, installationID, nativeID)
	}()
}

func (s *InstanceInventory) runDelete(ctx context.Context, op operation.Operation, installationID, nativeID string) {
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

	cmdCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	target, err := s.resolveAcceptedInstallation(cmdCtx, installationID)
	if err != nil || target.Bundle.Instances == nil {
		s.failCreate(running, yorvaruntime.ErrorRuntimeNotSupported, false)
		return
	}
	listed, reconErr := s.reconcileLocked(cmdCtx, installationID, target.Installation.Path)
	if reconErr != nil {
		s.failCreate(running, errorCodeFrom(reconErr), true)
		return
	}
	if !instanceAvailable(listed, nativeID) {
		s.succeedDelete(running)
		return
	}
	deleteErr := target.Bundle.Instances.Delete(cmdCtx, target.Installation.Path, nativeID)
	post, postErr := s.reconcileLocked(cmdCtx, installationID, target.Installation.Path)
	if postErr != nil {
		s.failCreate(running, errorCodeFrom(postErr), true)
		return
	}
	if !instanceAvailable(post, nativeID) {
		s.succeedDelete(running)
		return
	}
	if deleteErr != nil && !errors.Is(deleteErr, context.Canceled) {
		s.failCreate(running, errorCodeFrom(deleteErr), true)
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

func (s *InstanceInventory) RecoverStale(ctx context.Context) ([]operation.Operation, error) {
	if s.db == nil {
		return nil, nil
	}
	ops, err := s.db.ListActiveInstanceOperations(ctx)
	if err != nil || len(ops) == 0 {
		return ops, err
	}
	recovered := make([]operation.Operation, 0, len(ops))
	for _, op := range ops {
		unlock := s.lockInstallation(op.TargetID)
		current, getErr := s.db.GetOperation(ctx, op.ID)
		if getErr != nil {
			unlock()
			return recovered, getErr
		}
		if operation.IsTerminal(current.Status) {
			unlock()
			continue
		}
		next, recErr := s.recoverOneLocked(ctx, current)
		unlock()
		if recErr != nil {
			return recovered, recErr
		}
		recovered = append(recovered, next)
	}
	return recovered, nil
}

func (s *InstanceInventory) recoverOneLocked(ctx context.Context, current operation.Operation) (operation.Operation, error) {
	listed, queryErr := s.queryAuthoritative(ctx, current.TargetID)
	if queryErr != nil {
		return s.persistRecoveredFail(ctx, current, errorCodeFrom(queryErr), true)
	}
	present := instanceAvailable(listed, current.Message)
	switch current.Type {
	case operation.TypeInstanceCreate:
		if present {
			return s.persistRecoveredSucceed(ctx, current)
		}
		return s.persistRecoveredFail(ctx, current, yorvaruntime.ErrorOperationInterrupted, true)
	case operation.TypeInstanceDelete:
		if !present {
			return s.persistRecoveredSucceed(ctx, current)
		}
		return s.persistRecoveredFail(ctx, current, yorvaruntime.ErrorOperationInterrupted, true)
	default:
		return current, nil
	}
}

func (s *InstanceInventory) queryAuthoritative(ctx context.Context, installationID string) (InstanceList, error) {
	target, err := s.resolveAcceptedInstallation(ctx, installationID)
	if err != nil || target.Bundle.Instances == nil {
		_ = s.db.MarkInstancesUnknown(ctx, installationID, s.now())
		return InstanceList{Freshness: "UNKNOWN"}, ErrRuntimeNotSupported
	}
	return s.reconcileLocked(ctx, installationID, target.Installation.Path)
}

func (s *InstanceInventory) persistRecoveredSucceed(ctx context.Context, current operation.Operation) (operation.Operation, error) {
	now := s.now()
	if current.Status == operation.StatusPending {
		running := current
		running.Status = operation.StatusRunning
		running.StartedAt = &now
		running.UpdatedAt = now
		if err := s.db.UpdateOperation(ctx, current, running); err != nil {
			return current, err
		}
		current = running
	}
	next := current
	next.Status = operation.StatusSucceeded
	next.Stage = operation.StageInstanceReconcile
	next.CompletedAt = &now
	next.UpdatedAt = now
	if err := s.db.UpdateOperation(ctx, current, next); err != nil {
		return current, err
	}
	return next, nil
}

func (s *InstanceInventory) persistRecoveredFail(ctx context.Context, current operation.Operation, code yorvaruntime.ErrorCode, retryable bool) (operation.Operation, error) {
	now := s.now()
	next := current
	next.Status = operation.StatusFailed
	next.ErrorCode = code
	next.Retryable = retryable
	next.CompletedAt = &now
	next.UpdatedAt = now
	if err := s.db.UpdateOperation(ctx, current, next); err != nil {
		return current, err
	}
	return next, nil
}

package app

import (
	"context"
	"errors"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	"github.com/YoLin02/yorva/services/node/internal/events"
	"github.com/YoLin02/yorva/services/node/internal/managedskills"
	"github.com/YoLin02/yorva/services/node/internal/persistence/sqlite"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const maxManagementSkillItems = 256

var (
	ErrSkillSourceNotApproved = errors.New("Skill source is not approved")
	ErrSkillOwnershipConflict = errors.New("Skill is not owned by YORVA")
	ErrSkillDriftDetected     = errors.New("managed Skill projection has drifted")
	ErrSkillMutationConflict  = errors.New("another Skill mutation is active")
)

type SkillSourceView struct {
	SourceID    string
	SkillID     string
	DisplayName string
	Version     string
}

type managedSkillAction string

const (
	managedSkillInstall managedSkillAction = "install"
	managedSkillUpdate  managedSkillAction = "update"
	managedSkillEnable  managedSkillAction = "enable"
	managedSkillDisable managedSkillAction = "disable"
	managedSkillRemove  managedSkillAction = "remove"
)

// ManagementSkills merges Runtime-native inventory with YORVA-owned package
// and projection state. Mutations use SkillProjection only and never call the
// Runtime-native SkillManager contract.
type ManagementSkills struct {
	targets ManagementTargetResolver
	db      *sqlite.Database
	store   *managedskills.Store
	events  *events.Broker
	now     func() time.Time
	newID   func() (string, error)
	mu      sync.Mutex
	locks   map[string]*sync.Mutex
}

func NewManagementSkills(targets ManagementTargetResolver) *ManagementSkills {
	return &ManagementSkills{
		targets: targets,
		now:     func() time.Time { return time.Now().UTC() },
		newID:   newOperationID,
		locks:   make(map[string]*sync.Mutex),
	}
}

func NewManagedManagementSkills(targets ManagementTargetResolver, db *sqlite.Database, store *managedskills.Store, broker *events.Broker) *ManagementSkills {
	service := NewManagementSkills(targets)
	service.db, service.store, service.events = db, store, broker
	return service
}

// NewManagementSkills is the production composition point used by HTTP. It
// also performs the required fail-closed restart recovery before serving.
func (s *InstanceInventory) NewManagementSkills(dataDir string) (*ManagementSkills, error) {
	if s == nil || s.db == nil {
		return nil, ErrManagementQueryFailed
	}
	store, err := managedskills.NewStore(dataDir)
	if err != nil {
		return nil, ErrManagementQueryFailed
	}
	service := NewManagedManagementSkills(s, s.db, store, s.events)
	if _, err := service.RecoverInterrupted(context.Background()); err != nil {
		return nil, ErrManagementQueryFailed
	}
	return service, nil
}

func (s *ManagementSkills) ListSources(ctx context.Context, instanceID string) ([]SkillSourceView, error) {
	if s == nil || s.store == nil {
		return nil, ErrManagementCapabilityUnsupported
	}
	target, err := s.resolve(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if target.Bundle.SkillProjection == nil {
		return nil, ErrManagementCapabilityUnsupported
	}
	entries := s.store.ListCatalog()
	if len(entries) > maxManagementSkillItems {
		return nil, ErrManagementQueryFailed
	}
	result := make([]SkillSourceView, 0, len(entries))
	for _, entry := range entries {
		result = append(result, SkillSourceView{SourceID: entry.SourceID, SkillID: entry.SkillID, DisplayName: entry.Description, Version: entry.Version})
	}
	return result, nil
}

func (s *ManagementSkills) ListSkills(ctx context.Context, instanceID string) ([]yorvaruntime.Skill, error) {
	target, err := s.resolve(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if target.Bundle.SkillRead == nil && target.Bundle.SkillProjection == nil {
		return nil, ErrManagementCapabilityUnsupported
	}
	merged := make(map[string]yorvaruntime.Skill)
	if target.Bundle.SkillProjection != nil {
		items, queryErr := target.Bundle.SkillProjection.ListSkillProjections(ctx, target.Installation, target.NativeID)
		if queryErr != nil {
			return nil, managementQueryError(ctx, queryErr)
		}
		if err := mergeSkillItems(merged, items, false); err != nil {
			return nil, err
		}
	}
	if target.Bundle.SkillRead != nil {
		items, queryErr := target.Bundle.SkillRead.ListSkills(ctx, target.Installation, target.NativeID)
		if queryErr != nil {
			return nil, managementQueryError(ctx, queryErr)
		}
		if err := mergeSkillItems(merged, items, true); err != nil {
			return nil, err
		}
	}
	if s.db != nil {
		records, queryErr := s.db.ListManagedSkills(ctx, instanceID)
		if queryErr != nil {
			return nil, managementQueryError(ctx, queryErr)
		}
		for _, record := range records {
			item, exists := merged[record.SkillID]
			item.ID, item.SourceID, item.Version = record.SkillID, record.SourceID, record.SourceVersion
			item.Ownership = yorvaruntime.SkillOwnershipYORVAManaged
			item.InstallationState, item.ScanState = yorvaruntime.SkillInstalled, yorvaruntime.SkillScanClean
			if !exists {
				if record.DesiredEnabled {
					item.ProjectionState, item.EnabledState = yorvaruntime.SkillProjectionDriftMissing, yorvaruntime.SkillEnabledUnknown
				} else {
					item.ProjectionState, item.EnabledState = yorvaruntime.SkillProjectionNotProjected, yorvaruntime.SkillDisabled
				}
			} else if item.ProjectionState == yorvaruntime.SkillProjectionProjected {
				if record.DesiredEnabled {
					item.EnabledState = yorvaruntime.SkillEnabled
				} else {
					item.ProjectionState, item.EnabledState = yorvaruntime.SkillProjectionConflict, yorvaruntime.SkillEnabledUnknown
				}
			} else if isProjectionDrift(item.ProjectionState) {
				item.EnabledState = yorvaruntime.SkillEnabledUnknown
			} else if !record.DesiredEnabled {
				item.EnabledState = yorvaruntime.SkillDisabled
			}
			item.UpdateAvailable = sourceUpdateAvailable(s.store, record)
			merged[record.SkillID] = item
		}
	}
	if len(merged) > maxManagementSkillItems {
		return nil, ErrManagementQueryFailed
	}
	result := make([]yorvaruntime.Skill, 0, len(merged))
	for _, item := range merged {
		normalizeUnclassifiedSkill(&item)
		if item.Validate() != nil {
			return nil, ErrManagementQueryFailed
		}
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func (s *ManagementSkills) InspectSkill(ctx context.Context, instanceID, skillID string) (yorvaruntime.Skill, error) {
	if err := validateSkillID(skillID); err != nil {
		return yorvaruntime.Skill{}, err
	}
	items, err := s.ListSkills(ctx, instanceID)
	if err != nil {
		return yorvaruntime.Skill{}, err
	}
	for _, item := range items {
		if item.ID == skillID {
			return item, nil
		}
	}
	return yorvaruntime.Skill{}, ErrManagementQueryFailed
}

func (s *ManagementSkills) StartInstall(ctx context.Context, instanceID, skillID, sourceID, key string) (InstallStartResult, error) {
	if err := validateSkillID(skillID); err != nil {
		return InstallStartResult{}, err
	}
	if err := (yorvaruntime.SkillInstallRequest{SourceID: sourceID}).Validate(); err != nil {
		return InstallStartResult{}, err
	}
	return s.start(ctx, instanceID, skillID, sourceID, key, managedSkillInstall)
}

func (s *ManagementSkills) StartUpdate(ctx context.Context, instanceID, skillID, key string) (InstallStartResult, error) {
	return s.startExisting(ctx, instanceID, skillID, key, managedSkillUpdate)
}

func (s *ManagementSkills) StartEnable(ctx context.Context, instanceID, skillID, key string) (InstallStartResult, error) {
	return s.startExisting(ctx, instanceID, skillID, key, managedSkillEnable)
}

func (s *ManagementSkills) StartDisable(ctx context.Context, instanceID, skillID, key string) (InstallStartResult, error) {
	return s.startExisting(ctx, instanceID, skillID, key, managedSkillDisable)
}

func (s *ManagementSkills) StartRemove(ctx context.Context, instanceID, skillID, key string) (InstallStartResult, error) {
	return s.startExisting(ctx, instanceID, skillID, key, managedSkillRemove)
}

func (s *ManagementSkills) startExisting(ctx context.Context, instanceID, skillID, key string, action managedSkillAction) (InstallStartResult, error) {
	if err := validateSkillID(skillID); err != nil {
		return InstallStartResult{}, err
	}
	if s == nil || s.db == nil {
		return InstallStartResult{}, ErrManagementCapabilityUnsupported
	}
	record, err := s.db.GetManagedSkill(ctx, instanceID, skillID)
	if errors.Is(err, sqlite.ErrManagedSkillNotFound) {
		return InstallStartResult{}, ErrSkillOwnershipConflict
	}
	if err != nil {
		return InstallStartResult{}, managementQueryError(ctx, err)
	}
	return s.start(ctx, instanceID, skillID, record.SourceID, key, action)
}

func (s *ManagementSkills) start(ctx context.Context, instanceID, skillID, sourceID, key string, action managedSkillAction) (InstallStartResult, error) {
	if s == nil || s.db == nil || s.store == nil {
		return InstallStartResult{}, ErrManagementCapabilityUnsupported
	}
	if err := ValidateIdempotencyKey(key); err != nil {
		return InstallStartResult{}, err
	}
	target, err := s.resolve(ctx, instanceID)
	if err != nil {
		return InstallStartResult{}, err
	}
	if target.Bundle.SkillProjection == nil {
		return InstallStartResult{}, ErrManagementCapabilityUnsupported
	}
	if action == managedSkillInstall && !catalogContains(s.store, skillID, sourceID) {
		return InstallStartResult{}, ErrSkillSourceNotApproved
	}
	opType, stage := managedSkillOperationShape(action)
	if existing, ok, queryErr := s.db.GetOperationByIdempotencyKey(ctx, key); queryErr != nil {
		return InstallStartResult{}, managementQueryError(ctx, queryErr)
	} else if ok {
		if existing.Type != opType || existing.TargetID != instanceID || existing.Message != skillID {
			return InstallStartResult{}, ErrSkillMutationConflict
		}
		return InstallStartResult{Operation: existing}, nil
	}
	id, err := s.newID()
	if err != nil {
		return InstallStartResult{}, ErrManagementQueryFailed
	}
	correlation, err := newCorrelationID()
	if err != nil {
		return InstallStartResult{}, ErrManagementQueryFailed
	}
	deploymentID, err := sqlite.NewManagedSkillID()
	if err != nil {
		return InstallStartResult{}, ErrManagementQueryFailed
	}
	now := s.now()
	op := operation.Operation{
		ID: id, Type: opType, TargetType: operation.TargetInstance, TargetID: instanceID,
		Status: operation.StatusPending, Stage: stage, Message: skillID,
		IdempotencyKey: key, CorrelationID: correlation, SourcePin: sourceID,
		OwnershipNonce: deploymentID, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.db.CreateOperation(ctx, op); err != nil {
		if errors.Is(err, sqlite.ErrDuplicateIdempotency) || errors.Is(err, sqlite.ErrActiveInstanceMutation) {
			return InstallStartResult{}, ErrSkillMutationConflict
		}
		return InstallStartResult{}, managementQueryError(ctx, err)
	}
	s.emitOperation(operation.Operation{}, op, true)
	s.startWorker(op, target, action)
	return InstallStartResult{Operation: op, Created: true}, nil
}

func (s *ManagementSkills) startWorker(op operation.Operation, target ManagementTarget, action managedSkillAction) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		unlock := s.lockInstance(op.TargetID)
		defer unlock()
		s.runWorker(ctx, op, target, action)
	}()
}

func (s *ManagementSkills) runWorker(ctx context.Context, op operation.Operation, target ManagementTarget, action managedSkillAction) {
	current, err := s.db.GetOperation(ctx, op.ID)
	if err != nil || operation.IsTerminal(current.Status) {
		return
	}
	now := s.now()
	running := current
	running.Status, running.Stage, running.StartedAt, running.UpdatedAt = operation.StatusRunning, operation.StageSkillPreflight, &now, now
	if err := s.persistOperation(ctx, current, running); err != nil {
		return
	}
	var mutationErr error
	switch action {
	case managedSkillInstall:
		mutationErr = s.installManaged(ctx, running, target)
	case managedSkillUpdate:
		mutationErr = s.updateManaged(ctx, running, target)
	case managedSkillEnable:
		mutationErr = s.enableManaged(ctx, running, target)
	case managedSkillDisable:
		mutationErr = s.disableManaged(ctx, running, target)
	case managedSkillRemove:
		mutationErr = s.removeManaged(ctx, running, target)
	}
	if mutationErr != nil {
		s.finishOperation(running, operation.StatusFailed, managedSkillErrorCode(mutationErr), true)
		return
	}
	s.finishOperation(running, operation.StatusSucceeded, "", false)
}

func (s *ManagementSkills) installManaged(ctx context.Context, op operation.Operation, target ManagementTarget) error {
	if _, err := s.db.GetManagedSkill(ctx, op.TargetID, op.Message); err == nil {
		return ErrSkillOwnershipConflict
	} else if !errors.Is(err, sqlite.ErrManagedSkillNotFound) {
		return err
	}
	acquired, err := s.store.AcquireToManaged(ctx, op.TargetID, op.Message, op.SourcePin)
	if err != nil {
		return err
	}
	now := s.now()
	record := sqlite.ManagedSkill{
		ID: op.OwnershipNonce, InstanceID: op.TargetID, SkillID: op.Message,
		SourceID: acquired.SourceID, SourceVersion: acquired.Version, ContentSHA256: acquired.ContentSHA256,
		ManagedRelativePath: filepath.ToSlash(acquired.RelativePath), ProjectionRelativePath: acquired.SkillID,
		DeploymentID: op.OwnershipNonce, DesiredEnabled: true, ProjectionState: yorvaruntime.SkillProjectionNotProjected,
		InstalledAt: now, UpdatedAt: now,
	}
	if err := s.db.CreateManagedSkill(ctx, record); err != nil {
		return err
	}
	projected, err := target.Bundle.SkillProjection.ProjectSkill(ctx, target.Installation, target.NativeID, projectRequest(acquired, record.DeploymentID), nil)
	if err != nil {
		_ = s.db.DeleteManagedSkill(context.Background(), record.InstanceID, record.SkillID)
		return err
	}
	if err := verifyProjectedSkill(projected, record); err != nil {
		return err
	}
	if err := s.db.UpdateManagedSkillProjection(ctx, record.InstanceID, record.SkillID, true, yorvaruntime.SkillProjectionProjected, s.now()); err != nil {
		return err
	}
	return s.restartIfRunning(ctx, target)
}

func (s *ManagementSkills) updateManaged(ctx context.Context, op operation.Operation, target ManagementTarget) error {
	current, err := s.requireOwnedProjection(ctx, op, target, false)
	if err != nil {
		return err
	}
	acquired, err := s.store.AcquireToManaged(ctx, current.InstanceID, current.SkillID, current.SourceID)
	if err != nil {
		return err
	}
	next := current
	next.SourceVersion, next.ContentSHA256 = acquired.Version, acquired.ContentSHA256
	next.ManagedRelativePath, next.UpdatedAt = filepath.ToSlash(acquired.RelativePath), s.now()
	if current.DesiredEnabled {
		projected, projectErr := target.Bundle.SkillProjection.ProjectSkill(ctx, target.Installation, target.NativeID, projectRequest(acquired, current.DeploymentID), nil)
		if projectErr != nil {
			return projectErr
		}
		next.ProjectionState = yorvaruntime.SkillProjectionProjected
		if err := verifyProjectedSkill(projected, next); err != nil {
			return err
		}
	} else {
		next.ProjectionState = yorvaruntime.SkillProjectionNotProjected
	}
	if err := s.db.ReplaceManagedSkillRelease(ctx, current, next); err != nil {
		return err
	}
	if current.DesiredEnabled {
		return s.restartIfRunning(ctx, target)
	}
	return nil
}

func (s *ManagementSkills) enableManaged(ctx context.Context, op operation.Operation, target ManagementTarget) error {
	record, err := s.requireOwnedProjection(ctx, op, target, false)
	if err != nil && !errors.Is(err, ErrSkillDriftDetected) {
		return err
	}
	if record.ProjectionState == yorvaruntime.SkillProjectionDriftModified || record.ProjectionState == yorvaruntime.SkillProjectionConflict {
		return ErrSkillDriftDetected
	}
	sourceDir := filepath.Join(s.store.Root(), filepath.FromSlash(record.ManagedRelativePath))
	bundle, err := managedskills.InspectSourceDirectory(sourceDir)
	if err != nil || bundle.ContentSHA256 != record.ContentSHA256 {
		return ErrSkillOwnershipConflict
	}
	projected, err := target.Bundle.SkillProjection.ProjectSkill(ctx, target.Installation, target.NativeID, yorvaruntime.SkillProjectRequest{
		SkillID: record.SkillID, SourceDir: sourceDir, SourceID: record.SourceID, Version: record.SourceVersion,
		ContentSHA256: record.ContentSHA256, DeploymentID: record.DeploymentID,
	}, nil)
	if err != nil {
		return err
	}
	if err := verifyProjectedSkill(projected, record); err != nil {
		return err
	}
	if err := s.db.UpdateManagedSkillProjection(ctx, record.InstanceID, record.SkillID, true, yorvaruntime.SkillProjectionProjected, s.now()); err != nil {
		return err
	}
	return s.restartIfRunning(ctx, target)
}

func (s *ManagementSkills) disableManaged(ctx context.Context, op operation.Operation, target ManagementTarget) error {
	record, err := s.requireOwnedProjection(ctx, op, target, true)
	if err != nil {
		return err
	}
	if _, err := target.Bundle.SkillProjection.UnprojectSkill(ctx, target.Installation, target.NativeID, record.SkillID, record.DeploymentID, nil); err != nil {
		return err
	}
	if err := s.db.UpdateManagedSkillProjection(ctx, record.InstanceID, record.SkillID, false, yorvaruntime.SkillProjectionNotProjected, s.now()); err != nil {
		return err
	}
	return s.restartIfRunning(ctx, target)
}

func (s *ManagementSkills) removeManaged(ctx context.Context, op operation.Operation, target ManagementTarget) error {
	record, err := s.requireOwnedProjection(ctx, op, target, false)
	if err != nil && !errors.Is(err, ErrSkillDriftDetected) {
		return err
	}
	if record.ProjectionState == yorvaruntime.SkillProjectionDriftModified || record.ProjectionState == yorvaruntime.SkillProjectionConflict {
		return ErrSkillDriftDetected
	}
	wasProjected := record.ProjectionState == yorvaruntime.SkillProjectionProjected
	if wasProjected {
		if _, err := target.Bundle.SkillProjection.UnprojectSkill(ctx, target.Installation, target.NativeID, record.SkillID, record.DeploymentID, nil); err != nil {
			return err
		}
	}
	if err := s.db.DeleteManagedSkill(ctx, record.InstanceID, record.SkillID); err != nil {
		return err
	}
	if wasProjected {
		return s.restartIfRunning(ctx, target)
	}
	return nil
}

func (s *ManagementSkills) requireOwnedProjection(ctx context.Context, op operation.Operation, target ManagementTarget, requireProjected bool) (sqlite.ManagedSkill, error) {
	record, err := s.db.GetManagedSkill(ctx, op.TargetID, op.Message)
	if errors.Is(err, sqlite.ErrManagedSkillNotFound) {
		return sqlite.ManagedSkill{}, ErrSkillOwnershipConflict
	}
	if err != nil {
		return sqlite.ManagedSkill{}, err
	}
	observed, err := target.Bundle.SkillProjection.InspectSkillProjection(ctx, target.Installation, target.NativeID, record.SkillID)
	if err != nil {
		return sqlite.ManagedSkill{}, err
	}
	record.ProjectionState = observed.ProjectionState
	if observed.Ownership == yorvaruntime.SkillOwnershipExternal || observed.Ownership == yorvaruntime.SkillOwnershipRuntimeBundled {
		return record, ErrSkillOwnershipConflict
	}
	if observed.ProjectionState == yorvaruntime.SkillProjectionDriftModified || observed.ProjectionState == yorvaruntime.SkillProjectionConflict {
		return record, ErrSkillDriftDetected
	}
	if observed.ProjectionState == yorvaruntime.SkillProjectionProjected && observed.Ownership != yorvaruntime.SkillOwnershipYORVAManaged {
		return record, ErrSkillOwnershipConflict
	}
	if requireProjected && (observed.ProjectionState != yorvaruntime.SkillProjectionProjected || observed.Ownership != yorvaruntime.SkillOwnershipYORVAManaged) {
		return record, ErrSkillDriftDetected
	}
	return record, nil
}

func (s *ManagementSkills) restartIfRunning(ctx context.Context, target ManagementTarget) error {
	if target.Bundle.Lifecycle == nil {
		return nil
	}
	installation := yorvaruntime.LifecycleInstallation{Executable: target.Installation.Path, Version: target.Installation.Version}
	status, err := target.Bundle.Lifecycle.Status(ctx, installation, target.NativeID)
	if err != nil {
		return err
	}
	if status.State != yorvaruntime.LifecycleRunning {
		return nil
	}
	return target.Bundle.Lifecycle.Restart(ctx, installation, target.NativeID)
}

func (s *ManagementSkills) RecoverInterrupted(ctx context.Context) ([]operation.Operation, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	active, err := s.db.ListActiveSkillOperations(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]operation.Operation, 0, len(active))
	for _, current := range active {
		now := s.now()
		next := current
		next.Status, next.Stage = operation.StatusFailed, operation.StageSkillReconcile
		next.ErrorCode, next.Retryable = yorvaruntime.ErrorOperationInterrupted, true
		next.CompletedAt, next.UpdatedAt = &now, now
		if err := s.persistOperation(ctx, current, next); err != nil {
			return result, err
		}
		result = append(result, next)
	}
	return result, nil
}

func (s *ManagementSkills) finishOperation(current operation.Operation, status operation.Status, code yorvaruntime.ErrorCode, retryable bool) {
	now := s.now()
	next := current
	next.Status, next.Stage, next.ErrorCode, next.Retryable = status, operation.StageSkillReconcile, code, retryable
	next.CompletedAt, next.UpdatedAt = &now, now
	_ = s.persistOperation(context.Background(), current, next)
}

func (s *ManagementSkills) persistOperation(ctx context.Context, current, next operation.Operation) error {
	if err := s.db.UpdateOperation(ctx, current, next); err != nil {
		return err
	}
	s.emitOperation(current, next, false)
	return nil
}

func (s *ManagementSkills) emitOperation(previous, next operation.Operation, created bool) {
	if s == nil || s.events == nil {
		return
	}
	eventType := events.TypeForCommittedOperation(created, string(previous.Status), string(next.Status))
	if eventType == "" {
		return
	}
	s.events.Publish(events.NewOperationEvent(eventType, events.OperationPayload{
		OperationID: next.ID, Type: string(next.Type), Status: string(next.Status), Stage: string(next.Stage),
		ErrorCode: string(next.ErrorCode), CorrelationID: next.CorrelationID,
	}, s.now()))
}

func (s *ManagementSkills) lockInstance(instanceID string) func() {
	s.mu.Lock()
	lock := s.locks[instanceID]
	if lock == nil {
		lock = &sync.Mutex{}
		s.locks[instanceID] = lock
	}
	s.mu.Unlock()
	lock.Lock()
	return lock.Unlock
}

func (s *ManagementSkills) resolve(ctx context.Context, instanceID string) (ManagementTarget, error) {
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

func mergeSkillItems(merged map[string]yorvaruntime.Skill, items []yorvaruntime.Skill, native bool) error {
	if len(items) > maxManagementSkillItems {
		return ErrManagementQueryFailed
	}
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if item.Validate() != nil {
			return ErrManagementQueryFailed
		}
		if _, duplicate := seen[item.ID]; duplicate {
			return ErrManagementQueryFailed
		}
		seen[item.ID] = struct{}{}
		current, exists := merged[item.ID]
		if !exists || !native {
			merged[item.ID] = item
			continue
		}
		current.EnabledState = item.EnabledState
		if current.Version == "" {
			current.Version = item.Version
		}
		merged[item.ID] = current
	}
	return nil
}

func normalizeUnclassifiedSkill(item *yorvaruntime.Skill) {
	if item.Ownership == "" {
		item.Ownership = yorvaruntime.SkillOwnershipUnknown
	}
	if item.ProjectionState == "" {
		item.ProjectionState = yorvaruntime.SkillProjectionUnknown
	}
}

func isProjectionDrift(state yorvaruntime.SkillProjectionState) bool {
	return state == yorvaruntime.SkillProjectionDriftMissing || state == yorvaruntime.SkillProjectionDriftModified || state == yorvaruntime.SkillProjectionConflict
}

func sourceUpdateAvailable(store *managedskills.Store, record sqlite.ManagedSkill) bool {
	if store == nil {
		return false
	}
	for _, entry := range store.ListCatalog() {
		if entry.SourceID == record.SourceID && entry.SkillID == record.SkillID {
			return entry.Version != record.SourceVersion
		}
	}
	return false
}

func catalogContains(store *managedskills.Store, skillID, sourceID string) bool {
	if store == nil {
		return false
	}
	for _, entry := range store.ListCatalog() {
		if entry.SkillID == skillID && entry.SourceID == sourceID {
			return true
		}
	}
	return false
}

func projectRequest(acquired managedskills.Acquisition, deploymentID string) yorvaruntime.SkillProjectRequest {
	return yorvaruntime.SkillProjectRequest{
		SkillID: acquired.SkillID, SourceDir: acquired.AbsolutePath, SourceID: acquired.SourceID,
		Version: acquired.Version, ContentSHA256: acquired.ContentSHA256, DeploymentID: deploymentID,
	}
}

func verifyProjectedSkill(item yorvaruntime.Skill, record sqlite.ManagedSkill) error {
	if item.Validate() != nil || item.ID != record.SkillID || item.SourceID != record.SourceID ||
		item.Version != record.SourceVersion || item.Ownership != yorvaruntime.SkillOwnershipYORVAManaged ||
		item.ProjectionState != yorvaruntime.SkillProjectionProjected {
		return ErrSkillDriftDetected
	}
	return nil
}

func managedSkillOperationShape(action managedSkillAction) (operation.Type, operation.Stage) {
	switch action {
	case managedSkillUpdate:
		return operation.TypeSkillUpdate, operation.StageSkillProject
	case managedSkillEnable:
		return operation.TypeSkillEnable, operation.StageSkillProject
	case managedSkillDisable:
		return operation.TypeSkillDisable, operation.StageSkillUnproject
	case managedSkillRemove:
		return operation.TypeSkillRemove, operation.StageSkillUnproject
	default:
		return operation.TypeSkillInstall, operation.StageSkillProject
	}
}

func managedSkillErrorCode(err error) yorvaruntime.ErrorCode {
	switch {
	case errors.Is(err, ErrSkillSourceNotApproved), errors.Is(err, managedskills.ErrCatalogNotFound), errors.Is(err, managedskills.ErrCatalogEntryInvalid):
		return yorvaruntime.ErrorSkillSourceNotApproved
	case errors.Is(err, ErrSkillOwnershipConflict), errors.Is(err, sqlite.ErrManagedSkillOwnership), errors.Is(err, sqlite.ErrManagedSkillConflict):
		return yorvaruntime.ErrorSkillOwnershipConflict
	case errors.Is(err, ErrSkillDriftDetected), errors.Is(err, sqlite.ErrManagedSkillStale):
		return yorvaruntime.ErrorSkillDriftDetected
	case errors.Is(err, ErrSkillMutationConflict), errors.Is(err, sqlite.ErrActiveInstanceMutation):
		return yorvaruntime.ErrorSkillMutationConflict
	case errors.Is(err, managedskills.ErrManagedPathConflict), errors.Is(err, managedskills.ErrBundleInvalid):
		return yorvaruntime.ErrorSkillAcquireFailed
	case errors.Is(err, yorvaruntime.ErrLifecycleMutationFailed), errors.Is(err, yorvaruntime.ErrLifecyclePostcondition):
		return yorvaruntime.ErrorLifecycleRestartFailed
	default:
		return yorvaruntime.ErrorSkillProjectFailed
	}
}

func validateSkillID(skillID string) error {
	return (yorvaruntime.SkillConfigureRequest{SkillID: skillID}).Validate()
}

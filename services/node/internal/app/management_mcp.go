package app

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	"github.com/YoLin02/yorva/services/node/internal/events"
	"github.com/YoLin02/yorva/services/node/internal/persistence/sqlite"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const managementMCPCollectionLimit = 256

var ErrMCPMutationConflict = errors.New("another MCP or Instance mutation is active")

type MCPServerView struct {
	ID             string
	PresetID       string
	Ownership      string
	EnabledToolIDs []string
	State          yorvaruntime.MCPState
	ReadyAt        *time.Time
	ObservedAt     time.Time
}

type MCPPresetView struct {
	ID                 string
	DisplayName        string
	Description        string
	HomepageURL        string
	DocumentationURL   string
	AllowedToolIDs     []string
	CredentialRequired bool
	Source             string
	Editable           bool
}

// MCPTestView is the safe result consumed by a future Operation worker. It is
// deliberately not exposed through a synchronous HTTP mutation endpoint.
type MCPTestView struct {
	ServerID string
	State    yorvaruntime.MCPState
	ToolIDs  []string
	ReadyAt  time.Time
	TestedAt time.Time
}

// MCPManagement owns Runtime-neutral validation for the closed MCP read and
// test boundaries. The Runtime adapter remains authoritative for live state.
type MCPManagement struct {
	targets        ManagementTargetResolver
	runtimeTargets RuntimeManagementTargetResolver
	db             *sqlite.Database
	events         *events.Broker
	now            func() time.Time
	newID          func() (string, error)
	mu             sync.Mutex
	cancels        map[string]context.CancelFunc
}

func NewMCPManagement(targets ManagementTargetResolver) *MCPManagement {
	service := &MCPManagement{targets: targets, now: time.Now, newID: newOperationID, cancels: make(map[string]context.CancelFunc)}
	service.runtimeTargets, _ = targets.(RuntimeManagementTargetResolver)
	return service
}

func (s *InstanceInventory) NewMCPManagement() (*MCPManagement, error) {
	if s == nil || s.db == nil {
		return nil, ErrManagementQueryFailed
	}
	service := NewMCPManagement(s)
	service.db, service.events = s.db, s.events
	if _, err := service.RecoverInterrupted(context.Background()); err != nil {
		return nil, err
	}
	return service, nil
}

func (s *MCPManagement) ListMCPServers(ctx context.Context, instanceID string) ([]MCPServerView, error) {
	target, err := s.resolve(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if target.Bundle.MCPRead == nil {
		return nil, ErrManagementCapabilityUnsupported
	}

	servers, err := target.Bundle.MCPRead.ListMCPServers(ctx, target.Installation, target.NativeID)
	if err != nil {
		return nil, managementMCPError(ctx, err)
	}
	if len(servers) > managementMCPCollectionLimit {
		return nil, ErrManagementQueryFailed
	}

	managed := make(map[string]string)
	if s.db != nil {
		bindings, bindingErr := s.db.ListManagedMCPBindings(ctx, instanceID)
		if bindingErr != nil {
			return nil, managementMCPError(ctx, bindingErr)
		}
		for _, binding := range bindings {
			managed[binding.ServerID] = binding.PresetID
		}
	}
	seen := make(map[string]struct{}, len(servers))
	views := make([]MCPServerView, 0, len(servers))
	for _, server := range servers {
		if err := server.Validate(); err != nil {
			return nil, ErrManagementQueryFailed
		}
		if _, exists := seen[server.ID]; exists {
			return nil, ErrManagementQueryFailed
		}
		seen[server.ID] = struct{}{}

		readyAt, valid := validatedMCPReadyAt(server.State, server.ReadyAt, server.ObservedAt)
		if !valid {
			return nil, ErrManagementQueryFailed
		}
		owned := managed[server.ID] == server.PresetID
		views = append(views, MCPServerView{
			ID:             server.ID,
			PresetID:       server.PresetID,
			Ownership:      mcpOwnership(owned),
			EnabledToolIDs: append([]string(nil), server.EnabledToolIDs...),
			State:          server.State,
			ReadyAt:        readyAt,
			ObservedAt:     server.ObservedAt.UTC(),
		})
	}
	return views, nil
}

func mcpOwnership(managed bool) string {
	if managed {
		return "YORVA_MANAGED"
	}
	return "EXTERNAL"
}

func (s *MCPManagement) ListMCPPresets(ctx context.Context, instanceID string) ([]MCPPresetView, error) {
	target, err := s.resolve(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if target.Bundle.MCPRead == nil {
		return nil, ErrManagementCapabilityUnsupported
	}

	presets, err := target.Bundle.MCPRead.ListMCPPresets(ctx, target.Installation, target.NativeID)
	if err != nil {
		return nil, managementMCPError(ctx, err)
	}
	return validatedMCPPresetViews(presets)
}

// ListRuntimeMCPDefinitions returns the reviewed, Runtime-owned definition
// catalog. Instance bindings are deliberately read and mutated separately.
func (s *MCPManagement) ListRuntimeMCPDefinitions(ctx context.Context, runtimeID string) ([]MCPPresetView, error) {
	if s == nil || s.runtimeTargets == nil {
		return nil, ErrManagementCapabilityUnsupported
	}
	target, err := s.runtimeTargets.ResolveRuntimeManagementTarget(ctx, runtimeID)
	if err != nil {
		return nil, err
	}
	if target.Bundle.MCPRead == nil {
		return nil, ErrManagementCapabilityUnsupported
	}
	presets, err := target.Bundle.MCPRead.ListMCPPresets(ctx, target.Installation, "")
	if err != nil {
		return nil, managementMCPError(ctx, err)
	}
	views, err := validatedMCPPresetViews(presets)
	if err != nil {
		return nil, err
	}
	for index := range views {
		views[index].Source = "BUILT_IN"
	}
	return views, nil
}

func validatedMCPPresetViews(presets []yorvaruntime.MCPPreset) ([]MCPPresetView, error) {
	if len(presets) > managementMCPCollectionLimit {
		return nil, ErrManagementQueryFailed
	}
	seen := make(map[string]struct{}, len(presets))
	views := make([]MCPPresetView, 0, len(presets))
	for _, preset := range presets {
		if err := preset.Validate(); err != nil {
			return nil, ErrManagementQueryFailed
		}
		if _, exists := seen[preset.ID]; exists {
			return nil, ErrManagementQueryFailed
		}
		seen[preset.ID] = struct{}{}
		views = append(views, MCPPresetView{
			ID: preset.ID, DisplayName: preset.DisplayName, Description: preset.Description,
			HomepageURL: preset.HomepageURL, DocumentationURL: preset.DocumentationURL,
			AllowedToolIDs: append([]string(nil), preset.AllowedToolIDs...), CredentialRequired: preset.CredentialRequired, Source: "BUILT_IN",
		})
	}
	return views, nil
}

// TestMCP is a bounded application worker boundary, not an HTTP handler. Its
// caller must own the durable Operation, timeout, cancellation, and ProgressSink.
func (s *MCPManagement) TestMCP(ctx context.Context, instanceID, serverID string, progress yorvaruntime.ProgressSink) (MCPTestView, error) {
	if err := (yorvaruntime.MCPConfigureRequest{ServerID: serverID}).Validate(); err != nil {
		return MCPTestView{}, ErrManagementQueryFailed
	}
	target, err := s.resolve(ctx, instanceID)
	if err != nil {
		return MCPTestView{}, err
	}
	if target.Bundle.MCPMutate == nil {
		return MCPTestView{}, ErrManagementCapabilityUnsupported
	}

	startedAt := s.now().UTC()
	result, err := target.Bundle.MCPMutate.TestMCP(ctx, target.Installation, target.NativeID, serverID, progress)
	completedAt := s.now().UTC()
	if err != nil {
		return MCPTestView{}, managementMCPError(ctx, err)
	}
	if err := result.Validate(); err != nil || result.ServerID != serverID || result.State != yorvaruntime.MCPReady ||
		result.ReadyAt == nil || !result.ReadyAt.Equal(result.TestedAt) || result.TestedAt.Before(startedAt) || result.TestedAt.After(completedAt) {
		return MCPTestView{}, ErrManagementQueryFailed
	}

	return MCPTestView{
		ServerID: result.ServerID,
		State:    result.State,
		ToolIDs:  append([]string(nil), result.ToolIDs...),
		ReadyAt:  result.ReadyAt.UTC(),
		TestedAt: result.TestedAt.UTC(),
	}, nil
}

type mcpMutationAction string

const (
	mcpMutationInstall      mcpMutationAction = "install"
	mcpMutationAuthenticate mcpMutationAction = "authenticate"
	mcpMutationTest         mcpMutationAction = "test"
	mcpMutationConfigure    mcpMutationAction = "configure"
	mcpMutationRemove       mcpMutationAction = "remove"
)

type mcpMutationInput struct {
	action     mcpMutationAction
	presetID   string
	serverID   string
	credential []byte
	toolIDs    []string
}

func (s *MCPManagement) StartInstall(ctx context.Context, instanceID, presetID string, credential []byte, toolIDs []string, key string) (InstallStartResult, error) {
	if err := (yorvaruntime.MCPInstallRequest{PresetID: presetID}).Validate(); err != nil {
		return InstallStartResult{}, err
	}
	if len(credential) > 0 {
		if err := (yorvaruntime.MCPAuthenticateRequest{ServerID: presetID, Credential: credential}).Validate(); err != nil {
			return InstallStartResult{}, err
		}
	}
	if err := (yorvaruntime.MCPConfigureRequest{ServerID: presetID, EnabledToolIDs: toolIDs}).Validate(); err != nil {
		return InstallStartResult{}, err
	}
	return s.start(ctx, instanceID, key, mcpMutationInput{
		action: mcpMutationInstall, presetID: presetID,
		credential: append([]byte(nil), credential...), toolIDs: append([]string(nil), toolIDs...),
	})
}

func (s *MCPManagement) StartAuthenticate(ctx context.Context, instanceID, serverID string, credential []byte, key string) (InstallStartResult, error) {
	request := yorvaruntime.MCPAuthenticateRequest{ServerID: serverID, Credential: credential}
	if err := request.Validate(); err != nil {
		return InstallStartResult{}, err
	}
	return s.start(ctx, instanceID, key, mcpMutationInput{
		action: mcpMutationAuthenticate, serverID: serverID, credential: append([]byte(nil), credential...),
	})
}

func (s *MCPManagement) StartTest(ctx context.Context, instanceID, serverID, key string) (InstallStartResult, error) {
	if err := (yorvaruntime.MCPConfigureRequest{ServerID: serverID}).Validate(); err != nil {
		return InstallStartResult{}, err
	}
	return s.start(ctx, instanceID, key, mcpMutationInput{action: mcpMutationTest, serverID: serverID})
}

func (s *MCPManagement) StartConfigure(ctx context.Context, instanceID, serverID string, toolIDs []string, key string) (InstallStartResult, error) {
	request := yorvaruntime.MCPConfigureRequest{ServerID: serverID, EnabledToolIDs: toolIDs}
	if err := request.Validate(); err != nil {
		return InstallStartResult{}, err
	}
	return s.start(ctx, instanceID, key, mcpMutationInput{action: mcpMutationConfigure, serverID: serverID, toolIDs: append([]string(nil), toolIDs...)})
}

func (s *MCPManagement) StartRemove(ctx context.Context, instanceID, serverID, key string) (InstallStartResult, error) {
	if err := (yorvaruntime.MCPConfigureRequest{ServerID: serverID}).Validate(); err != nil {
		return InstallStartResult{}, err
	}
	return s.start(ctx, instanceID, key, mcpMutationInput{action: mcpMutationRemove, serverID: serverID})
}

func (s *MCPManagement) start(ctx context.Context, instanceID, key string, input mcpMutationInput) (InstallStartResult, error) {
	if s == nil || s.db == nil {
		clear(input.credential)
		return InstallStartResult{}, ErrManagementCapabilityUnsupported
	}
	if input.action != mcpMutationInstall {
		managed, managedErr := s.managedBinding(ctx, instanceID, input.serverID)
		if managedErr != nil {
			clear(input.credential)
			return InstallStartResult{}, managementMCPError(ctx, managedErr)
		}
		if !managed {
			clear(input.credential)
			return InstallStartResult{}, ErrManagementCapabilityUnsupported
		}
	}
	if err := ValidateIdempotencyKey(key); err != nil {
		clear(input.credential)
		return InstallStartResult{}, err
	}
	target, err := s.resolve(ctx, instanceID)
	if err != nil {
		clear(input.credential)
		return InstallStartResult{}, err
	}
	if target.Bundle.MCPMutate == nil {
		clear(input.credential)
		return InstallStartResult{}, ErrManagementCapabilityUnsupported
	}
	opType, stage, subject := mcpOperationShape(input)
	if existing, ok, queryErr := s.db.GetOperationByIdempotencyKey(ctx, key); queryErr != nil {
		clear(input.credential)
		return InstallStartResult{}, managementMCPError(ctx, queryErr)
	} else if ok {
		clear(input.credential)
		if existing.Type != opType || existing.TargetID != instanceID || existing.Message != subject {
			return InstallStartResult{}, ErrMCPMutationConflict
		}
		return InstallStartResult{Operation: existing}, nil
	}
	id, err := s.newID()
	if err != nil {
		clear(input.credential)
		return InstallStartResult{}, ErrManagementQueryFailed
	}
	correlationID, err := newCorrelationID()
	if err != nil {
		clear(input.credential)
		return InstallStartResult{}, ErrManagementQueryFailed
	}
	now := s.now().UTC()
	op := operation.Operation{
		ID: id, Type: opType, TargetType: operation.TargetInstance, TargetID: instanceID,
		Status: operation.StatusPending, Stage: stage, Message: subject,
		IdempotencyKey: key, CorrelationID: correlationID, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.db.CreateOperation(ctx, op); err != nil {
		clear(input.credential)
		if errors.Is(err, sqlite.ErrDuplicateIdempotency) || errors.Is(err, sqlite.ErrActiveInstanceMutation) {
			return InstallStartResult{}, ErrMCPMutationConflict
		}
		return InstallStartResult{}, managementMCPError(ctx, err)
	}
	s.emitOperation(operation.Operation{}, op, true)
	go s.runMutation(op, target, input)
	return InstallStartResult{Operation: op, Created: true}, nil
}

func mcpOperationShape(input mcpMutationInput) (operation.Type, operation.Stage, string) {
	subject := input.serverID
	switch input.action {
	case mcpMutationInstall:
		return operation.TypeMCPInstall, operation.StageMCPConfigure, input.presetID
	case mcpMutationAuthenticate:
		return operation.TypeMCPAuthenticate, operation.StageMCPConfigure, subject
	case mcpMutationTest:
		return operation.TypeMCPTest, operation.StageMCPTest, subject
	case mcpMutationConfigure:
		return operation.TypeMCPConfigure, operation.StageMCPConfigure, subject
	default:
		return operation.TypeMCPRemove, operation.StageMCPConfigure, subject
	}
}

func (s *MCPManagement) runMutation(op operation.Operation, target ManagementTarget, input mcpMutationInput) {
	defer clear(input.credential)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	if !s.registerCancel(op.ID, cancel) {
		cancel()
		return
	}
	defer s.unregisterCancel(op.ID, cancel)
	current, err := s.db.GetOperation(ctx, op.ID)
	if err != nil || operation.IsTerminal(current.Status) {
		return
	}
	now := s.now().UTC()
	running := current
	running.Status, running.Stage, running.StartedAt, running.UpdatedAt = operation.StatusRunning, operation.StageMCPPreflight, &now, now
	if err := s.persistOperation(ctx, current, running); err != nil {
		return
	}
	var mutationErr error
	switch input.action {
	case mcpMutationInstall:
		server, err := target.Bundle.MCPMutate.InstallMCPPreset(ctx, target.Installation, target.NativeID, yorvaruntime.MCPInstallRequest{PresetID: input.presetID}, nil)
		if mutationErr == nil {
			mutationErr = validateMCPServerMutation(server, err, "", input.presetID)
		}
		if mutationErr == nil && len(input.credential) > 0 {
			server, err = target.Bundle.MCPMutate.AuthenticateMCP(ctx, target.Installation, target.NativeID, yorvaruntime.MCPAuthenticateRequest{ServerID: input.presetID, Credential: input.credential}, nil)
			mutationErr = validateMCPServerMutation(server, err, input.presetID, input.presetID)
		}
		if mutationErr == nil && len(input.toolIDs) > 0 {
			server, err = target.Bundle.MCPMutate.ConfigureMCP(ctx, target.Installation, target.NativeID, yorvaruntime.MCPConfigureRequest{ServerID: input.presetID, EnabledToolIDs: input.toolIDs}, nil)
			mutationErr = validateMCPServerMutation(server, err, input.presetID, input.presetID)
		}
		if mutationErr == nil {
			testedFrom := s.now().UTC()
			result, testErr := target.Bundle.MCPMutate.TestMCP(ctx, target.Installation, target.NativeID, input.presetID, nil)
			testedThrough := s.now().UTC()
			if testErr != nil || result.Validate() != nil || result.ServerID != input.presetID || result.State != yorvaruntime.MCPReady ||
				result.ReadyAt == nil || !result.ReadyAt.Equal(result.TestedAt) || result.TestedAt.Before(testedFrom) || result.TestedAt.After(testedThrough) {
				mutationErr = ErrManagementQueryFailed
			}
		}
		if mutationErr == nil {
			if target.Bundle.MCPRead == nil {
				mutationErr = ErrManagementQueryFailed
			}
			servers, readErr := []yorvaruntime.MCPServer(nil), error(nil)
			if mutationErr == nil {
				servers, readErr = target.Bundle.MCPRead.ListMCPServers(ctx, target.Installation, target.NativeID)
			}
			found := false
			for _, observed := range servers {
				found = found || observed.ID == input.presetID && observed.PresetID == input.presetID && observed.State == yorvaruntime.MCPReady && observed.ReadyAt != nil
			}
			if readErr != nil || !found {
				mutationErr = ErrManagementQueryFailed
			}
		}
		if mutationErr == nil {
			now := s.now().UTC()
			mutationErr = s.db.UpsertManagedMCPBinding(ctx, sqlite.ManagedMCPBinding{
				InstanceID: op.TargetID, ServerID: input.presetID, PresetID: input.presetID, CreatedAt: now, UpdatedAt: now,
			})
		}
		if mutationErr != nil {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
			_, _ = target.Bundle.MCPMutate.RemoveMCP(cleanupCtx, target.Installation, target.NativeID, input.presetID, nil)
			cleanupCancel()
		}
	case mcpMutationAuthenticate:
		server, err := target.Bundle.MCPMutate.AuthenticateMCP(ctx, target.Installation, target.NativeID, yorvaruntime.MCPAuthenticateRequest{ServerID: input.serverID, Credential: input.credential}, nil)
		mutationErr = validateMCPServerMutation(server, err, input.serverID, "")
	case mcpMutationTest:
		startedAt := s.now().UTC()
		result, err := target.Bundle.MCPMutate.TestMCP(ctx, target.Installation, target.NativeID, input.serverID, nil)
		completedAt := s.now().UTC()
		if err != nil || result.Validate() != nil || result.ServerID != input.serverID || result.State != yorvaruntime.MCPReady ||
			result.ReadyAt == nil || !result.ReadyAt.Equal(result.TestedAt) || result.TestedAt.Before(startedAt) || result.TestedAt.After(completedAt) {
			mutationErr = ErrManagementQueryFailed
		}
	case mcpMutationConfigure:
		server, err := target.Bundle.MCPMutate.ConfigureMCP(ctx, target.Installation, target.NativeID, yorvaruntime.MCPConfigureRequest{ServerID: input.serverID, EnabledToolIDs: input.toolIDs}, nil)
		mutationErr = validateMCPServerMutation(server, err, input.serverID, "")
	case mcpMutationRemove:
		server, err := target.Bundle.MCPMutate.RemoveMCP(ctx, target.Installation, target.NativeID, input.serverID, nil)
		mutationErr = validateMCPServerMutation(server, err, input.serverID, "")
		if mutationErr == nil {
			mutationErr = s.db.DeleteManagedMCPBinding(ctx, op.TargetID, input.serverID)
		}
	}
	completed := s.now().UTC()
	final := running
	final.Stage, final.CompletedAt, final.UpdatedAt = operation.StageMCPReconcile, &completed, completed
	if mutationErr != nil {
		final.Status, final.ErrorCode, final.Retryable = operation.StatusFailed, mcpMutationErrorCode(input.action), true
	} else {
		final.Status = operation.StatusSucceeded
	}
	_ = s.persistOperation(context.Background(), running, final)
}

func (s *MCPManagement) managedBinding(ctx context.Context, instanceID, serverID string) (bool, error) {
	if s == nil || s.db == nil {
		return false, nil
	}
	items, err := s.db.ListManagedMCPBindings(ctx, instanceID)
	if err != nil {
		return false, err
	}
	for _, item := range items {
		if item.ServerID == serverID && item.PresetID == serverID {
			return true, nil
		}
	}
	return false, nil
}

func (s *MCPManagement) CancelMCPOperation(ctx context.Context, operationID string) (operation.Operation, error) {
	if s == nil || s.db == nil {
		return operation.Operation{}, ErrManagementCapabilityUnsupported
	}
	current, err := s.db.GetOperation(ctx, operationID)
	if err != nil {
		return operation.Operation{}, ErrManagementQueryFailed
	}
	if current.Type != operation.TypeMCPInstall && current.Type != operation.TypeMCPAuthenticate && current.Type != operation.TypeMCPTest &&
		current.Type != operation.TypeMCPConfigure && current.Type != operation.TypeMCPRemove {
		return operation.Operation{}, ErrManagementCapabilityUnsupported
	}
	if operation.IsTerminal(current.Status) {
		return operation.Operation{}, ErrInstanceNotCancellable
	}
	s.mu.Lock()
	cancel := s.cancels[operationID]
	s.mu.Unlock()
	now := s.now().UTC()
	next := current
	next.Status, next.ErrorCode, next.Retryable = operation.StatusCancelled, "", false
	next.CompletedAt, next.UpdatedAt = &now, now
	if err := s.persistOperation(ctx, current, next); err != nil {
		return operation.Operation{}, ErrInstanceNotCancellable
	}
	// Publish the terminal state before interrupting the adapter. Otherwise the
	// worker can observe cancellation first and win the final-state update race.
	if cancel != nil {
		cancel()
	}
	return next, nil
}

func (s *MCPManagement) registerCancel(operationID string, cancel context.CancelFunc) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.cancels[operationID]; exists {
		return false
	}
	s.cancels[operationID] = cancel
	return true
}

func (s *MCPManagement) unregisterCancel(operationID string, cancel context.CancelFunc) {
	cancel()
	s.mu.Lock()
	delete(s.cancels, operationID)
	s.mu.Unlock()
}

func validateMCPServerMutation(server yorvaruntime.MCPServer, err error, serverID, presetID string) error {
	if err != nil || server.Validate() != nil || (serverID != "" && server.ID != serverID) || (presetID != "" && server.PresetID != presetID) {
		return ErrManagementQueryFailed
	}
	return nil
}

func mcpMutationErrorCode(action mcpMutationAction) yorvaruntime.ErrorCode {
	switch action {
	case mcpMutationAuthenticate:
		return yorvaruntime.ErrorMCPAuthenticationFailed
	case mcpMutationTest:
		return yorvaruntime.ErrorMCPTestFailed
	default:
		return yorvaruntime.ErrorMCPConfigurationFailed
	}
}

func (s *MCPManagement) RecoverInterrupted(ctx context.Context) ([]operation.Operation, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	active, err := s.db.ListActiveMCPOperations(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]operation.Operation, 0, len(active))
	for _, current := range active {
		now := s.now().UTC()
		next := current
		next.Status, next.Stage = operation.StatusFailed, operation.StageMCPReconcile
		next.ErrorCode, next.Retryable, next.CompletedAt, next.UpdatedAt = yorvaruntime.ErrorOperationInterrupted, true, &now, now
		if err := s.persistOperation(ctx, current, next); err != nil {
			return result, err
		}
		result = append(result, next)
	}
	return result, nil
}

func (s *MCPManagement) persistOperation(ctx context.Context, current, next operation.Operation) error {
	if err := s.db.UpdateOperation(ctx, current, next); err != nil {
		return err
	}
	s.emitOperation(current, next, false)
	return nil
}

func (s *MCPManagement) emitOperation(previous, next operation.Operation, created bool) {
	if s.events == nil {
		return
	}
	eventType := events.TypeForCommittedOperation(created, string(previous.Status), string(next.Status))
	if eventType != "" {
		s.events.Publish(events.NewOperationEvent(eventType, events.OperationPayload{
			OperationID: next.ID, Type: string(next.Type), Status: string(next.Status), Stage: string(next.Stage),
			ErrorCode: string(next.ErrorCode), CorrelationID: next.CorrelationID,
		}, s.now().UTC()))
	}
}

func (s *MCPManagement) resolve(ctx context.Context, instanceID string) (ManagementTarget, error) {
	if s == nil || s.targets == nil {
		return ManagementTarget{}, ErrManagementCapabilityUnsupported
	}
	target, err := s.targets.ResolveManagementTarget(ctx, instanceID)
	if err != nil {
		return ManagementTarget{}, managementMCPError(ctx, err)
	}
	return target, nil
}

func validatedMCPReadyAt(state yorvaruntime.MCPState, readyAt *time.Time, observedAt time.Time) (*time.Time, bool) {
	if state != yorvaruntime.MCPReady {
		return nil, readyAt == nil
	}
	if readyAt == nil || readyAt.IsZero() || readyAt.After(observedAt) {
		return nil, false
	}
	readyUTC := readyAt.UTC()
	return &readyUTC, true
}

func managementMCPError(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	case errors.Is(err, ErrInstanceNotFound), errors.Is(err, ErrInstanceNotAvailable), errors.Is(err, ErrRuntimeNotSupported),
		errors.Is(err, ErrManagementCapabilityUnsupported), errors.Is(err, ErrManagementQueryFailed), errors.Is(err, ErrMCPMutationConflict):
		return err
	default:
		return ErrManagementQueryFailed
	}
}

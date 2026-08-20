package app

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/channel"
	"github.com/YoLin02/yorva/services/node/internal/domain/instance"
	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	"github.com/YoLin02/yorva/services/node/internal/events"
	"github.com/YoLin02/yorva/services/node/internal/persistence/sqlite"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

var (
	ErrChannelNotSupported = errors.New("channel is not supported")
	ErrChannelConflict     = errors.New("channel operation conflicts")
	ErrChannelSession      = errors.New("channel initiating session is invalid")
)

type ChannelView struct {
	Type              channel.Type
	State             channel.State
	AccountLabel      string
	ExternalID        string
	LastCheckedAt     *time.Time
	ActiveOperationID *string
}

type ChannelConnectInput struct {
	Type   channel.Type
	BotID  string
	Secret []byte
}

type channelEventSink struct {
	inventory   *InstanceInventory
	operationID string
	owner       string
}

func (s channelEventSink) Publish(event yorvaruntime.ChannelEvent) error {
	if event.Stage != "qr_ready" || len(event.QRPayload) == 0 {
		return errors.New("invalid channel event")
	}
	if err := s.inventory.channelQR.Publish(s.operationID, s.owner, event.QRPayload, event.ExpiresAt); err != nil {
		return err
	}
	if err := s.inventory.updateChannelStage(s.operationID, operation.StageChannelQRReady); err != nil {
		s.inventory.channelQR.Delete(s.operationID)
		return err
	}
	if s.inventory.events != nil {
		s.inventory.events.Publish(events.NewChannelQRReadyEvent(s.operationID, event.ExpiresAt, s.inventory.now()))
	}
	return nil
}

func (s *InstanceInventory) ListChannels(ctx context.Context, instanceID string) ([]ChannelView, error) {
	row, manager, installation, err := s.resolveChannelTarget(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	statuses, err := manager.ListChannels(ctx, installation, row.NativeID)
	if err != nil {
		if errors.Is(err, yorvaruntime.ErrChannelNotSupported) {
			return nil, ErrChannelNotSupported
		}
		return nil, yorvaruntime.ErrChannelStateUnknown
	}
	active, hasActive, err := s.db.ActiveInstanceRuntimeMutation(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	now := s.now()
	views := make([]ChannelView, 0, len(statuses))
	for _, status := range statuses {
		if !channel.ValidType(status.Type) || !channel.ValidState(status.State) {
			return nil, ErrChannelNotSupported
		}
		binding, exists, getErr := s.db.GetChannelBinding(ctx, row.ID, status.Type)
		if getErr != nil {
			return nil, getErr
		}
		if !exists {
			id, idErr := sqlite.NewChannelBindingID()
			if idErr != nil {
				return nil, idErr
			}
			binding = channel.Binding{ID: id, InstanceID: row.ID, Type: status.Type, CreatedAt: now}
		}
		// Hermes-native files prove that binding material is still present, while
		// the safe projection records that YORVA previously authenticated it.
		// Never promote externally-created or changed material to CONNECTED.
		if status.State == channel.Unknown && exists && binding.State == channel.Connected && status.ExternalID != "" && binding.ExternalID == status.ExternalID {
			status.State = channel.Connected
			if status.AccountLabel == "" {
				status.AccountLabel = binding.AccountLabel
			}
		}
		binding.State = status.State
		binding.AccountLabel = status.AccountLabel
		binding.ExternalID = status.ExternalID
		binding.LastCheckedAt = &now
		binding.UpdatedAt = now
		if err := s.db.UpsertChannelBinding(ctx, binding); err != nil {
			return nil, err
		}
		view := ChannelView{Type: status.Type, State: status.State, AccountLabel: status.AccountLabel, ExternalID: status.ExternalID, LastCheckedAt: &now}
		if hasActive && active.Message == string(status.Type) && isChannelOperation(active.Type) {
			view.ActiveOperationID = &active.ID
			if active.Type == operation.TypeChannelConnect {
				view.State = channel.Connecting
			}
		}
		views = append(views, view)
	}
	return views, nil
}

func (s *InstanceInventory) StartChannelConnect(ctx context.Context, instanceID, idempotencyKey, owner string, input ChannelConnectInput) (InstallStartResult, error) {
	defer clearBytes(input.Secret)
	if err := ValidateIdempotencyKey(idempotencyKey); err != nil {
		return InstallStartResult{}, err
	}
	if !channel.ValidType(input.Type) || !validChannelSession(owner) || (input.Type == channel.Weixin && (input.BotID != "" || len(input.Secret) != 0)) || (input.Type == channel.WeCom && (input.BotID == "" || len(input.Secret) == 0)) {
		return InstallStartResult{}, ErrChannelSession
	}
	row, _, _, err := s.resolveChannelTarget(ctx, instanceID)
	if err != nil {
		return InstallStartResult{}, err
	}
	s.operationMu.Lock()
	defer s.operationMu.Unlock()
	if existing, ok, getErr := s.db.GetOperationByIdempotencyKey(ctx, idempotencyKey); getErr != nil {
		return InstallStartResult{}, getErr
	} else if ok {
		if existing.Type != operation.TypeChannelConnect || existing.TargetID != row.ID || existing.Message != string(input.Type) {
			return InstallStartResult{}, ErrChannelConflict
		}
		return InstallStartResult{Operation: existing}, nil
	}
	if _, active, activeErr := s.db.ActiveInstanceRuntimeMutation(ctx, row.ID); activeErr != nil {
		return InstallStartResult{}, activeErr
	} else if active {
		return InstallStartResult{}, ErrChannelConflict
	}
	if _, active, activeErr := s.db.ActiveInstanceMutation(ctx, row.RuntimeInstallationID); activeErr != nil {
		return InstallStartResult{}, activeErr
	} else if active {
		return InstallStartResult{}, ErrChannelConflict
	}
	op, err := s.newChannelOperation(row.ID, operation.TypeChannelConnect, operation.StageChannelPreparing, input.Type, idempotencyKey)
	if err != nil {
		return InstallStartResult{}, err
	}
	if err := s.db.CreateOperation(ctx, op); err != nil {
		if errors.Is(err, sqlite.ErrDuplicateIdempotency) || errors.Is(err, sqlite.ErrActiveInstanceMutation) {
			return InstallStartResult{}, ErrChannelConflict
		}
		return InstallStartResult{}, err
	}
	s.emitChannelOperation(operation.Operation{}, op, true)
	secret := append([]byte(nil), input.Secret...)
	s.startChannelWorker(op, row.RuntimeInstallationID, row.NativeID, owner, ChannelConnectInput{Type: input.Type, BotID: input.BotID, Secret: secret})
	return InstallStartResult{Operation: op, Created: true}, nil
}

func (s *InstanceInventory) StartChannelDisconnect(ctx context.Context, instanceID, idempotencyKey string, kind channel.Type) (InstallStartResult, error) {
	if err := ValidateIdempotencyKey(idempotencyKey); err != nil {
		return InstallStartResult{}, err
	}
	if !channel.ValidType(kind) {
		return InstallStartResult{}, ErrChannelNotSupported
	}
	row, _, _, err := s.resolveChannelTarget(ctx, instanceID)
	if err != nil {
		return InstallStartResult{}, err
	}
	s.operationMu.Lock()
	defer s.operationMu.Unlock()
	if existing, ok, getErr := s.db.GetOperationByIdempotencyKey(ctx, idempotencyKey); getErr != nil {
		return InstallStartResult{}, getErr
	} else if ok {
		if existing.Type != operation.TypeChannelDisconnect || existing.TargetID != row.ID || existing.Message != string(kind) {
			return InstallStartResult{}, ErrChannelConflict
		}
		return InstallStartResult{Operation: existing}, nil
	}
	if _, active, activeErr := s.db.ActiveInstanceRuntimeMutation(ctx, row.ID); activeErr != nil {
		return InstallStartResult{}, activeErr
	} else if active {
		return InstallStartResult{}, ErrChannelConflict
	}
	if _, active, activeErr := s.db.ActiveInstanceMutation(ctx, row.RuntimeInstallationID); activeErr != nil {
		return InstallStartResult{}, activeErr
	} else if active {
		return InstallStartResult{}, ErrChannelConflict
	}
	op, err := s.newChannelOperation(row.ID, operation.TypeChannelDisconnect, operation.StageChannelDisconnect, kind, idempotencyKey)
	if err != nil {
		return InstallStartResult{}, err
	}
	if err := s.db.CreateOperation(ctx, op); err != nil {
		if errors.Is(err, sqlite.ErrDuplicateIdempotency) || errors.Is(err, sqlite.ErrActiveInstanceMutation) {
			return InstallStartResult{}, ErrChannelConflict
		}
		return InstallStartResult{}, err
	}
	s.emitChannelOperation(operation.Operation{}, op, true)
	s.startChannelWorker(op, row.RuntimeInstallationID, row.NativeID, "", ChannelConnectInput{Type: kind})
	return InstallStartResult{Operation: op, Created: true}, nil
}

func (s *InstanceInventory) GetChannelQR(ctx context.Context, operationID, owner string) (ChannelQRPayload, error) {
	current, err := s.db.GetOperation(ctx, operationID)
	if err != nil || current.Type != operation.TypeChannelConnect {
		return ChannelQRPayload{}, errChannelQRUnavailable
	}
	return s.channelQR.Get(operationID, owner)
}

func (s *InstanceInventory) CancelChannel(ctx context.Context, operationID string) (operation.Operation, error) {
	current, err := s.db.GetOperation(ctx, operationID)
	if err != nil || !isChannelOperation(current.Type) {
		return operation.Operation{}, ErrInstanceNotFound
	}
	if operation.IsTerminal(current.Status) {
		return current, nil
	}
	s.mu.Lock()
	cancel := s.workers[operationID]
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	now := s.now()
	next := current
	next.Status = operation.StatusCancelled
	next.ErrorCode = yorvaruntime.ErrorChannelAuthCancelled
	next.CompletedAt = &now
	next.UpdatedAt = now
	if err := s.db.UpdateOperation(ctx, current, next); err != nil {
		latest, getErr := s.db.GetOperation(ctx, operationID)
		if getErr == nil && operation.IsTerminal(latest.Status) {
			return latest, nil
		}
		return operation.Operation{}, err
	}
	s.channelQR.Delete(operationID)
	s.emitChannelOperation(current, next, false)
	return next, nil
}

func (s *InstanceInventory) RecoverChannels(ctx context.Context) ([]operation.Operation, error) {
	ops, err := s.db.ListActiveChannelOperations(ctx)
	if err != nil {
		return nil, err
	}
	recovered := make([]operation.Operation, 0, len(ops))
	for _, current := range ops {
		completed := false
		if current.Type == operation.TypeChannelDisconnect {
			views, listErr := s.ListChannels(ctx, current.TargetID)
			if listErr == nil {
				for _, view := range views {
					if string(view.Type) == current.Message && view.State == channel.NotConfigured {
						s.finishChannel(current, operation.StatusSucceeded, "", false)
						completed = true
						break
					}
				}
			}
		}
		if !completed {
			s.finishChannel(current, operation.StatusFailed, yorvaruntime.ErrorOperationInterrupted, true)
		}
		latest, getErr := s.db.GetOperation(ctx, current.ID)
		if getErr != nil {
			return recovered, getErr
		}
		recovered = append(recovered, latest)
		s.channelQR.Delete(current.ID)
	}
	return recovered, nil
}

func (s *InstanceInventory) startChannelWorker(op operation.Operation, installationID, nativeID, owner string, input ChannelConnectInput) {
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.workers[op.ID] = cancel
	s.mu.Unlock()
	go func() {
		defer func() {
			clearBytes(input.Secret)
			s.channelQR.Delete(op.ID)
			s.mu.Lock()
			delete(s.workers, op.ID)
			s.mu.Unlock()
			cancel()
		}()
		s.runChannel(ctx, op, installationID, nativeID, owner, input)
	}()
}

func (s *InstanceInventory) runChannel(ctx context.Context, op operation.Operation, installationID, nativeID, owner string, input ChannelConnectInput) {
	unlock := s.lockInstance(op.TargetID)
	defer unlock()
	current, err := s.db.GetOperation(ctx, op.ID)
	if err != nil || operation.IsTerminal(current.Status) {
		return
	}
	now := s.now()
	running := current
	running.Status = operation.StatusRunning
	running.StartedAt = &now
	running.UpdatedAt = now
	if err := s.db.UpdateOperation(ctx, current, running); err != nil {
		return
	}
	s.emitChannelOperation(current, running, false)
	row, manager, installation, resolveErr := s.resolveChannelTarget(ctx, op.TargetID)
	if resolveErr != nil || row.RuntimeInstallationID != installationID || row.NativeID != nativeID {
		s.finishChannel(running, operation.StatusFailed, yorvaruntime.ErrorChannelStateUnknown, true)
		return
	}
	var status yorvaruntime.ChannelStatus
	var mutationErr error
	if op.Type == operation.TypeChannelDisconnect {
		status, mutationErr = manager.Disconnect(ctx, installation, nativeID, input.Type)
	} else {
		var sink yorvaruntime.ChannelEventSink
		if input.Type == channel.Weixin {
			sink = channelEventSink{inventory: s, operationID: op.ID, owner: owner}
		}
		status, mutationErr = manager.BeginConnect(ctx, installation, nativeID, yorvaruntime.ChannelConnectRequest{Type: input.Type, BotID: input.BotID, Secret: input.Secret}, sink)
	}
	if mutationErr != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return
		}
		s.finishChannel(running, operation.StatusFailed, channelErrorCode(mutationErr, op.Type), channelRetryable(mutationErr))
		return
	}
	if (op.Type == operation.TypeChannelConnect && status.State != channel.Connected) || (op.Type == operation.TypeChannelDisconnect && status.State != channel.NotConfigured) {
		s.finishChannel(running, operation.StatusFailed, yorvaruntime.ErrorChannelStateUnknown, true)
		return
	}
	checked := s.now()
	binding, exists, _ := s.db.GetChannelBinding(context.Background(), row.ID, input.Type)
	if !exists {
		binding.ID, _ = sqlite.NewChannelBindingID()
		binding.InstanceID = row.ID
		binding.Type = input.Type
		binding.CreatedAt = checked
	}
	binding.State = status.State
	binding.AccountLabel = status.AccountLabel
	binding.ExternalID = status.ExternalID
	binding.LastCheckedAt = &checked
	binding.UpdatedAt = checked
	if binding.ID == "" || s.db.UpsertChannelBinding(context.Background(), binding) != nil {
		s.finishChannel(running, operation.StatusFailed, yorvaruntime.ErrorChannelStateUnknown, true)
		return
	}
	s.finishChannel(running, operation.StatusSucceeded, "", false)
}

func (s *InstanceInventory) resolveChannelTarget(ctx context.Context, instanceID string) (instance.Instance, yorvaruntime.ChannelManager, yorvaruntime.ChannelInstallation, error) {
	if s == nil || s.db == nil || s.discovery == nil || instanceID == "" {
		return instance.Instance{}, nil, yorvaruntime.ChannelInstallation{}, ErrInstanceNotFound
	}
	row, err := s.db.GetInstance(ctx, instanceID)
	if errors.Is(err, sql.ErrNoRows) {
		return instance.Instance{}, nil, yorvaruntime.ChannelInstallation{}, ErrInstanceNotFound
	}
	if err != nil || row.Availability != instance.Available {
		return instance.Instance{}, nil, yorvaruntime.ChannelInstallation{}, ErrInstanceNotAvailable
	}
	detected, err := s.discovery.Detect(ctx, yorvaruntime.Kind(hermesRuntimeID))
	if err != nil || detected.State != yorvaruntime.DiscoverySupported || detected.Selected == nil {
		return instance.Instance{}, nil, yorvaruntime.ChannelInstallation{}, ErrRuntimeNotSupported
	}
	accepted, err := s.ensureInstallation(ctx, detected)
	if err != nil || accepted.ID != row.RuntimeInstallationID {
		return instance.Instance{}, nil, yorvaruntime.ChannelInstallation{}, ErrInstanceNotAvailable
	}
	bundle, ok := s.discovery.registry.Get(yorvaruntime.Kind(hermesRuntimeID))
	if !ok || bundle.Channels == nil {
		return instance.Instance{}, nil, yorvaruntime.ChannelInstallation{}, ErrChannelNotSupported
	}
	return row, bundle.Channels, yorvaruntime.ChannelInstallation{Executable: detected.Selected.Path, Version: detected.Selected.Version}, nil
}

func (s *InstanceInventory) newChannelOperation(instanceID string, kind operation.Type, stage operation.Stage, channelType channel.Type, idempotencyKey string) (operation.Operation, error) {
	id, err := s.newID()
	if err != nil {
		return operation.Operation{}, err
	}
	correlation, err := newCorrelationID()
	if err != nil {
		return operation.Operation{}, err
	}
	now := s.now()
	return operation.Operation{ID: id, Type: kind, TargetType: operation.TargetInstance, TargetID: instanceID, Status: operation.StatusPending, Stage: stage, Message: string(channelType), IdempotencyKey: idempotencyKey, CorrelationID: correlation, CreatedAt: now, UpdatedAt: now}, nil
}

func (s *InstanceInventory) updateChannelStage(operationID string, stage operation.Stage) error {
	current, err := s.db.GetOperation(context.Background(), operationID)
	if err != nil || operation.IsTerminal(current.Status) {
		return err
	}
	next := current
	next.Stage = stage
	next.UpdatedAt = s.now()
	if err := s.db.UpdateOperation(context.Background(), current, next); err != nil {
		return err
	}
	s.emitChannelOperation(current, next, false)
	return nil
}

func (s *InstanceInventory) finishChannel(current operation.Operation, status operation.Status, code yorvaruntime.ErrorCode, retryable bool) {
	latest, err := s.db.GetOperation(context.Background(), current.ID)
	if err != nil || operation.IsTerminal(latest.Status) {
		return
	}
	now := s.now()
	next := latest
	next.Status = status
	next.Stage = operation.StageChannelCommitting
	next.ErrorCode = code
	next.Retryable = retryable
	next.CompletedAt = &now
	next.UpdatedAt = now
	if s.db.UpdateOperation(context.Background(), latest, next) == nil {
		s.emitChannelOperation(latest, next, false)
	}
}

func (s *InstanceInventory) emitChannelOperation(previous, next operation.Operation, created bool) {
	if s.events == nil {
		return
	}
	payload := events.OperationPayload{OperationID: next.ID, Type: string(next.Type), Status: string(next.Status), Stage: string(next.Stage), CorrelationID: next.CorrelationID, ErrorCode: string(next.ErrorCode)}
	s.events.Publish(events.NewOperationEvent(events.TypeForCommittedOperation(created, string(previous.Status), string(next.Status)), payload, s.now()))
}

func isChannelOperation(value operation.Type) bool {
	return value == operation.TypeChannelConnect || value == operation.TypeChannelDisconnect
}

func channelErrorCode(err error, operationType operation.Type) yorvaruntime.ErrorCode {
	switch {
	case errors.Is(err, yorvaruntime.ErrChannelNotSupported):
		return yorvaruntime.ErrorChannelNotSupported
	case errors.Is(err, yorvaruntime.ErrChannelAuthTimeout), errors.Is(err, context.DeadlineExceeded):
		return yorvaruntime.ErrorChannelAuthTimeout
	case errors.Is(err, yorvaruntime.ErrChannelDependency):
		return yorvaruntime.ErrorChannelDependencyMissing
	case errors.Is(err, yorvaruntime.ErrChannelStateUnknown):
		return yorvaruntime.ErrorChannelStateUnknown
	case operationType == operation.TypeChannelDisconnect:
		return yorvaruntime.ErrorChannelDisconnectFailed
	default:
		return yorvaruntime.ErrorChannelAuthFailed
	}
}

func channelRetryable(err error) bool {
	return !errors.Is(err, yorvaruntime.ErrChannelNotSupported) && !errors.Is(err, yorvaruntime.ErrChannelAuthFailed)
}

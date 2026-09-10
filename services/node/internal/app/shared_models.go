package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	"github.com/YoLin02/yorva/services/node/internal/events"
	"github.com/YoLin02/yorva/services/node/internal/persistence/sqlite"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const modelProfileApplyTimeout = 3 * time.Minute

var (
	ErrModelResourceInvalid   = errors.New("model resource is invalid")
	ErrModelResourceNotFound  = errors.New("model resource was not found")
	ErrModelResourceConflict  = errors.New("model resource is still in use")
	ErrModelSecretUnavailable = errors.New("model provider secret store is unavailable")
)

type ModelSecretStore interface {
	Put(context.Context, string, []byte) error
	Use(context.Context, string, func([]byte)) error
	Delete(context.Context, string) error
	Configured(context.Context, string) (bool, error)
}

type ModelProviderConnectionView struct {
	ID               string
	ProviderPresetID string
	DisplayName      string
	CredentialSet    bool
	Status           string
	Revision         int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type ModelProfileView struct {
	ID                   string
	ProviderConnectionID string
	DisplayName          string
	SelectedModelIDs     []string
	DefaultModelID       string
	Revision             int
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type RuntimeModelDefaultView struct {
	ModelProfileID  string
	AppliedRevision int
	UpdatedAt       time.Time
}

type InstanceModelBindingView struct {
	InstanceID      string
	InstanceName    string
	ModelProfileID  string
	Mode            string
	AppliedRevision int
	State           string
	ErrorCode       yorvaruntime.ErrorCode
	UpdatedAt       time.Time
}

func (s *InstanceInventory) ListModelProviderConnections(ctx context.Context, runtimeID string) ([]ModelProviderConnectionView, error) {
	installationID, _, _, err := s.runtimeModelContext(ctx, runtimeID)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.ListModelProviderConnections(ctx, installationID)
	if err != nil {
		return nil, err
	}
	result := make([]ModelProviderConnectionView, 0, len(rows))
	for _, row := range rows {
		configured := false
		if s.modelSecrets != nil {
			configured, err = s.modelSecrets.Configured(ctx, row.SecretRef)
			if err != nil {
				return nil, ErrModelSecretUnavailable
			}
		}
		status := row.Status
		if !configured {
			status = "UNAVAILABLE"
		}
		result = append(result, ModelProviderConnectionView{
			ID: row.ID, ProviderPresetID: row.ProviderPresetID, DisplayName: row.DisplayName,
			CredentialSet: configured, Status: status, Revision: row.Revision,
			CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		})
	}
	return result, nil
}

func (s *InstanceInventory) CreateModelProviderConnection(ctx context.Context, runtimeID, presetID, displayName string, secret []byte) (ModelProviderConnectionView, error) {
	installationID, models, _, err := s.runtimeModelContext(ctx, runtimeID)
	if err != nil {
		return ModelProviderConnectionView{}, err
	}
	if s.modelSecrets == nil {
		return ModelProviderConnectionView{}, ErrModelSecretUnavailable
	}
	if !validModelResourceName(displayName) || len(secret) == 0 || !hasModelPreset(models, presetID) {
		return ModelProviderConnectionView{}, ErrModelResourceInvalid
	}
	id, err := newModelResourceID("mpc")
	if err != nil {
		return ModelProviderConnectionView{}, err
	}
	secretRef, err := newModelResourceID("model-provider")
	if err != nil {
		return ModelProviderConnectionView{}, err
	}
	if err := s.modelSecrets.Put(ctx, secretRef, secret); err != nil {
		return ModelProviderConnectionView{}, ErrModelSecretUnavailable
	}
	now := s.now()
	row := sqlite.ModelProviderConnection{
		ID: id, RuntimeInstallationID: installationID, ProviderPresetID: presetID,
		DisplayName: strings.TrimSpace(displayName), SecretRef: secretRef, Status: "CONFIGURED",
		Revision: 1, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.db.InsertModelProviderConnection(ctx, row); err != nil {
		_ = s.modelSecrets.Delete(context.Background(), secretRef)
		return ModelProviderConnectionView{}, err
	}
	return ModelProviderConnectionView{
		ID: row.ID, ProviderPresetID: row.ProviderPresetID, DisplayName: row.DisplayName,
		CredentialSet: true, Status: row.Status, Revision: row.Revision, CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (s *InstanceInventory) DeleteModelProviderConnection(ctx context.Context, runtimeID, connectionID string) error {
	installationID, _, _, err := s.runtimeModelContext(ctx, runtimeID)
	if err != nil {
		return err
	}
	row, err := s.db.GetModelProviderConnection(ctx, connectionID)
	if errors.Is(err, sql.ErrNoRows) || row.RuntimeInstallationID != installationID {
		return ErrModelResourceNotFound
	}
	if err != nil {
		return err
	}
	profiles, err := s.db.ListModelProfiles(ctx, installationID)
	if err != nil {
		return err
	}
	for _, profile := range profiles {
		if profile.ProviderConnectionID == connectionID {
			return ErrModelResourceConflict
		}
	}
	if s.modelSecrets == nil || s.modelSecrets.Delete(ctx, row.SecretRef) != nil {
		return ErrModelSecretUnavailable
	}
	if err := s.db.DeleteModelProviderConnection(ctx, connectionID); err != nil {
		return err
	}
	return nil
}

func (s *InstanceInventory) ListModelProfiles(ctx context.Context, runtimeID string) ([]ModelProfileView, error) {
	installationID, _, _, err := s.runtimeModelContext(ctx, runtimeID)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.ListModelProfiles(ctx, installationID)
	if err != nil {
		return nil, err
	}
	result := make([]ModelProfileView, 0, len(rows))
	for _, row := range rows {
		result = append(result, modelProfileView(row))
	}
	return result, nil
}

func (s *InstanceInventory) CreateModelProfile(ctx context.Context, runtimeID, connectionID, displayName, defaultModelID string, selectedModelIDs []string) (ModelProfileView, error) {
	installationID, models, _, err := s.runtimeModelContext(ctx, runtimeID)
	if err != nil {
		return ModelProfileView{}, err
	}
	connection, err := s.db.GetModelProviderConnection(ctx, connectionID)
	if errors.Is(err, sql.ErrNoRows) || connection.RuntimeInstallationID != installationID {
		return ModelProfileView{}, ErrModelResourceNotFound
	}
	if err != nil {
		return ModelProfileView{}, err
	}
	if !validModelResourceName(displayName) || validateSelectedModels(models, connection.ProviderPresetID, defaultModelID, selectedModelIDs) != nil {
		return ModelProfileView{}, ErrModelResourceInvalid
	}
	id, err := newModelResourceID("mpr")
	if err != nil {
		return ModelProfileView{}, err
	}
	now := s.now()
	row := sqlite.ModelProfile{
		ID: id, RuntimeInstallationID: installationID, ProviderConnectionID: connectionID,
		DisplayName: strings.TrimSpace(displayName), SelectedModelIDs: append([]string(nil), selectedModelIDs...),
		DefaultModelID: defaultModelID, Revision: 1, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.db.InsertModelProfile(ctx, row); err != nil {
		return ModelProfileView{}, err
	}
	return modelProfileView(row), nil
}

func (s *InstanceInventory) DeleteModelProfile(ctx context.Context, runtimeID, profileID string) error {
	installationID, _, _, err := s.runtimeModelContext(ctx, runtimeID)
	if err != nil {
		return err
	}
	row, err := s.db.GetModelProfile(ctx, profileID)
	if errors.Is(err, sql.ErrNoRows) || row.RuntimeInstallationID != installationID {
		return ErrModelResourceNotFound
	}
	if err != nil {
		return err
	}
	if current, ok, readErr := s.db.GetRuntimeModelDefault(ctx, installationID); readErr != nil {
		return readErr
	} else if ok && current.ModelProfileID == profileID {
		return ErrModelResourceConflict
	}
	bindings, err := s.db.ListInstanceModelBindings(ctx, installationID)
	if err != nil {
		return err
	}
	for _, binding := range bindings {
		if binding.ModelProfileID == profileID {
			return ErrModelResourceConflict
		}
	}
	if err := s.db.DeleteModelProfile(ctx, profileID); err != nil {
		return err
	}
	return nil
}

func (s *InstanceInventory) GetRuntimeModelDefault(ctx context.Context, runtimeID string) (RuntimeModelDefaultView, error) {
	installationID, _, _, err := s.runtimeModelContext(ctx, runtimeID)
	if err != nil {
		return RuntimeModelDefaultView{}, err
	}
	value, ok, err := s.db.GetRuntimeModelDefault(ctx, installationID)
	if err != nil {
		return RuntimeModelDefaultView{}, err
	}
	if !ok {
		return RuntimeModelDefaultView{}, nil
	}
	return RuntimeModelDefaultView{ModelProfileID: value.ModelProfileID, AppliedRevision: value.AppliedRevision, UpdatedAt: value.UpdatedAt}, nil
}

func (s *InstanceInventory) SetRuntimeModelDefault(ctx context.Context, runtimeID, profileID string) (RuntimeModelDefaultView, error) {
	installationID, _, _, err := s.runtimeModelContext(ctx, runtimeID)
	if err != nil {
		return RuntimeModelDefaultView{}, err
	}
	profile, err := s.db.GetModelProfile(ctx, profileID)
	if errors.Is(err, sql.ErrNoRows) || profile.RuntimeInstallationID != installationID {
		return RuntimeModelDefaultView{}, ErrModelResourceNotFound
	}
	if err != nil {
		return RuntimeModelDefaultView{}, err
	}
	now := s.now()
	value := sqlite.RuntimeModelDefault{RuntimeInstallationID: installationID, ModelProfileID: profileID, AppliedRevision: profile.Revision, UpdatedAt: now}
	if err := s.db.PutRuntimeModelDefault(ctx, value); err != nil {
		return RuntimeModelDefaultView{}, err
	}
	return RuntimeModelDefaultView{ModelProfileID: profileID, AppliedRevision: profile.Revision, UpdatedAt: now}, nil
}

func (s *InstanceInventory) ClearRuntimeModelDefault(ctx context.Context, runtimeID string) error {
	installationID, _, _, err := s.runtimeModelContext(ctx, runtimeID)
	if err != nil {
		return err
	}
	return s.db.DeleteRuntimeModelDefault(ctx, installationID)
}

func (s *InstanceInventory) ListInstanceModelBindings(ctx context.Context, runtimeID string) ([]InstanceModelBindingView, error) {
	installationID, _, instances, err := s.runtimeModelContext(ctx, runtimeID)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.ListInstanceModelBindings(ctx, installationID)
	if err != nil {
		return nil, err
	}
	byInstance := make(map[string]sqlite.InstanceModelBinding, len(rows))
	for _, row := range rows {
		byInstance[row.InstanceID] = row
	}
	result := make([]InstanceModelBindingView, 0, len(instances))
	for _, item := range instances {
		if row, ok := byInstance[item.InstanceID]; ok {
			result = append(result, InstanceModelBindingView{
				InstanceID: row.InstanceID, InstanceName: item.Name, ModelProfileID: row.ModelProfileID,
				Mode: row.Mode, AppliedRevision: row.AppliedRevision, State: row.State,
				ErrorCode: yorvaruntime.ErrorCode(row.ErrorCode), UpdatedAt: row.UpdatedAt,
			})
			continue
		}
		configuration, readErr := s.GetModelConfiguration(ctx, item.InstanceID)
		if readErr == nil && configuration.State == yorvaruntime.ModelConfigurationConfigured {
			result = append(result, InstanceModelBindingView{
				InstanceID: item.InstanceID, InstanceName: item.Name, Mode: "EXTERNAL_CONFIGURATION",
				State: "SUCCEEDED", UpdatedAt: configuration.ObservedAt,
			})
		}
	}
	return result, nil
}

func (s *InstanceInventory) StartModelProfileApplication(ctx context.Context, runtimeID, profileID string, instanceIDs []string, mode, idempotencyKey string) (InstallStartResult, error) {
	if ValidateIdempotencyKey(idempotencyKey) != nil || (mode != "INHERIT" && mode != "OVERRIDE") || len(instanceIDs) == 0 || len(instanceIDs) > 50 {
		return InstallStartResult{}, ErrModelResourceInvalid
	}
	installationID, _, instances, err := s.runtimeModelContext(ctx, runtimeID)
	if err != nil {
		return InstallStartResult{}, err
	}
	profile, err := s.db.GetModelProfile(ctx, profileID)
	if errors.Is(err, sql.ErrNoRows) || profile.RuntimeInstallationID != installationID {
		return InstallStartResult{}, ErrModelResourceNotFound
	}
	if err != nil {
		return InstallStartResult{}, err
	}
	if mode == "INHERIT" {
		defaultValue, ok, readErr := s.db.GetRuntimeModelDefault(ctx, installationID)
		if readErr != nil {
			return InstallStartResult{}, readErr
		}
		if !ok || defaultValue.ModelProfileID != profile.ID {
			return InstallStartResult{}, ErrModelResourceInvalid
		}
	}
	available := make(map[string]struct{}, len(instances))
	for _, item := range instances {
		available[item.InstanceID] = struct{}{}
	}
	targets := append([]string(nil), instanceIDs...)
	sort.Strings(targets)
	for index, id := range targets {
		if _, ok := available[id]; !ok || (index > 0 && targets[index-1] == id) {
			return InstallStartResult{}, ErrModelResourceInvalid
		}
	}
	s.operationMu.Lock()
	defer s.operationMu.Unlock()
	sourcePin := modelProfileApplicationFingerprint(profile, targets, mode)
	if existing, ok, readErr := s.db.GetOperationByIdempotencyKey(ctx, idempotencyKey); readErr != nil {
		return InstallStartResult{}, readErr
	} else if ok {
		if existing.Type != operation.TypeModelProfileApply || existing.TargetID != installationID || existing.SourcePin != sourcePin {
			return InstallStartResult{}, yorvaruntime.ErrInstanceConfigConflict
		}
		return InstallStartResult{Operation: existing}, nil
	}
	if _, active, activeErr := s.db.ActiveInstanceMutation(ctx, installationID); activeErr != nil {
		return InstallStartResult{}, activeErr
	} else if active {
		return InstallStartResult{}, ErrModelResourceConflict
	}
	now := s.now()
	id, err := s.newID()
	if err != nil {
		return InstallStartResult{}, err
	}
	correlationID, err := newCorrelationID()
	if err != nil {
		return InstallStartResult{}, err
	}
	created := operation.Operation{
		ID: id, Type: operation.TypeModelProfileApply, TargetType: operation.TargetRuntimeInstallation,
		TargetID: installationID, Status: operation.StatusPending, Stage: operation.StagePreflight,
		IdempotencyKey: idempotencyKey, CorrelationID: correlationID, SourcePin: sourcePin,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.db.CreateOperation(ctx, created); err != nil {
		if errors.Is(err, sqlite.ErrDuplicateIdempotency) || errors.Is(err, sqlite.ErrActiveInstanceMutation) {
			return InstallStartResult{}, ErrModelResourceConflict
		}
		return InstallStartResult{}, err
	}
	for _, instanceID := range targets {
		if err := s.db.UpsertInstanceModelBinding(ctx, sqlite.InstanceModelBinding{
			InstanceID: instanceID, ModelProfileID: profile.ID, Mode: mode, AppliedRevision: profile.Revision,
			State: "PENDING", UpdatedAt: now,
		}); err != nil {
			_, _ = s.finishSharedModelOperation(created.ID, operation.StatusFailed, yorvaruntime.ErrorModelConfigIncomplete, true)
			return InstallStartResult{}, err
		}
	}
	s.emitSharedModelOperation(operation.Operation{}, created, true)
	s.startModelProfileApplicationWorker(created, profile, targets, mode)
	return InstallStartResult{Operation: created, Created: true}, nil
}

func (s *InstanceInventory) CancelModelProfileApplication(ctx context.Context, operationID string) (operation.Operation, error) {
	current, err := s.db.GetOperation(ctx, operationID)
	if err != nil || current.Type != operation.TypeModelProfileApply {
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
	return s.finishSharedModelOperation(operationID, operation.StatusCancelled, yorvaruntime.ErrorModelValidationCancelled, true)
}

func (s *InstanceInventory) RecoverModelProfileApplications(ctx context.Context) ([]operation.Operation, error) {
	active, err := s.db.ListActiveOperationsByType(ctx, operation.TypeModelProfileApply)
	if err != nil {
		return nil, err
	}
	result := make([]operation.Operation, 0, len(active))
	for _, value := range active {
		finished, finishErr := s.finishSharedModelOperation(value.ID, operation.StatusFailed, yorvaruntime.ErrorOperationInterrupted, true)
		if finishErr != nil {
			return result, finishErr
		}
		result = append(result, finished)
	}
	return result, nil
}

func (s *InstanceInventory) startModelProfileApplicationWorker(created operation.Operation, profile sqlite.ModelProfile, instanceIDs []string, mode string) {
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.workers[created.ID] = cancel
	s.mu.Unlock()
	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.workers, created.ID)
			s.mu.Unlock()
			cancel()
		}()
		s.runModelProfileApplication(ctx, created, profile, instanceIDs, mode)
	}()
}

func (s *InstanceInventory) runModelProfileApplication(parent context.Context, created operation.Operation, profile sqlite.ModelProfile, instanceIDs []string, mode string) {
	current, err := s.db.GetOperation(parent, created.ID)
	if err != nil || operation.IsTerminal(current.Status) {
		return
	}
	now := s.now()
	running := current
	running.Status = operation.StatusRunning
	running.Stage = operation.StageModelProfileApply
	running.StartedAt = &now
	running.UpdatedAt = now
	if err := s.db.UpdateOperation(parent, current, running); err != nil {
		return
	}
	s.emitSharedModelOperation(current, running, false)

	connection, err := s.db.GetModelProviderConnection(parent, profile.ProviderConnectionID)
	if err != nil || s.modelSecrets == nil {
		s.failAllModelBindings(profile, instanceIDs, mode, yorvaruntime.ErrorModelCredentialQueryFailed)
		_, _ = s.finishSharedModelOperation(created.ID, operation.StatusFailed, yorvaruntime.ErrorModelCredentialQueryFailed, true)
		return
	}
	ctx, cancel := context.WithTimeout(parent, modelProfileApplyTimeout)
	defer cancel()
	failed := false
	useErr := s.modelSecrets.Use(ctx, connection.SecretRef, func(secret []byte) {
		for index, instanceID := range instanceIDs {
			if ctx.Err() != nil {
				failed = true
				s.skipRemainingModelBindings(profile, instanceIDs[index:], mode)
				return
			}
			_, applyErr := s.saveModelCredentialConfiguration(ctx, instanceID, connection.ProviderPresetID,
				profile.DefaultModelID, profile.SelectedModelIDs, secret, created.ID)
			state := "SUCCEEDED"
			errorCode := yorvaruntime.ErrorCode("")
			if applyErr != nil {
				failed = true
				state = "FAILED"
				if errors.Is(applyErr, ErrInstanceNotAvailable) {
					state = "SKIPPED"
				}
				errorCode = sharedModelErrorCode(applyErr)
			}
			if bindingErr := s.db.UpsertInstanceModelBinding(context.Background(), sqlite.InstanceModelBinding{
				InstanceID: instanceID, ModelProfileID: profile.ID, Mode: mode, AppliedRevision: profile.Revision,
				State: state, ErrorCode: string(errorCode), UpdatedAt: s.now(),
			}); bindingErr != nil {
				failed = true
			}
		}
	})
	if useErr != nil {
		failed = true
		s.failAllModelBindings(profile, instanceIDs, mode, yorvaruntime.ErrorModelCredentialQueryFailed)
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		_, _ = s.finishSharedModelOperation(created.ID, operation.StatusFailed, yorvaruntime.ErrorModelValidationTimedOut, true)
		return
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		_, _ = s.finishSharedModelOperation(created.ID, operation.StatusCancelled, yorvaruntime.ErrorModelValidationCancelled, true)
		return
	}
	if failed {
		_, _ = s.finishSharedModelOperation(created.ID, operation.StatusFailed, yorvaruntime.ErrorModelConfigIncomplete, true)
		return
	}
	_, _ = s.finishSharedModelOperation(created.ID, operation.StatusSucceeded, "", false)
}

func (s *InstanceInventory) finishSharedModelOperation(operationID string, status operation.Status, code yorvaruntime.ErrorCode, retryable bool) (operation.Operation, error) {
	ctx := context.Background()
	current, err := s.db.GetOperation(ctx, operationID)
	if err != nil || operation.IsTerminal(current.Status) {
		return current, err
	}
	now := s.now()
	next := current
	next.Status = status
	next.ErrorCode = code
	next.Retryable = retryable
	next.CompletedAt = &now
	next.UpdatedAt = now
	if err := s.db.UpdateOperation(ctx, current, next); err != nil {
		return operation.Operation{}, err
	}
	s.emitSharedModelOperation(current, next, false)
	return next, nil
}

func (s *InstanceInventory) failAllModelBindings(profile sqlite.ModelProfile, instanceIDs []string, mode string, code yorvaruntime.ErrorCode) {
	for _, instanceID := range instanceIDs {
		_ = s.db.UpsertInstanceModelBinding(context.Background(), sqlite.InstanceModelBinding{
			InstanceID: instanceID, ModelProfileID: profile.ID, Mode: mode, AppliedRevision: profile.Revision,
			State: "FAILED", ErrorCode: string(code), UpdatedAt: s.now(),
		})
	}
}

func (s *InstanceInventory) skipRemainingModelBindings(profile sqlite.ModelProfile, instanceIDs []string, mode string) {
	for _, instanceID := range instanceIDs {
		_ = s.db.UpsertInstanceModelBinding(context.Background(), sqlite.InstanceModelBinding{
			InstanceID: instanceID, ModelProfileID: profile.ID, Mode: mode, AppliedRevision: profile.Revision,
			State: "SKIPPED", ErrorCode: string(yorvaruntime.ErrorModelValidationCancelled), UpdatedAt: s.now(),
		})
	}
}

func (s *InstanceInventory) runtimeModelContext(ctx context.Context, runtimeID string) (string, yorvaruntime.ModelConfigurator, []InstanceView, error) {
	if s == nil || s.discovery == nil || s.discovery.registry == nil {
		return "", nil, nil, ErrRuntimeNotSupported
	}
	bundle, ok := s.discovery.registry.Get(yorvaruntime.Kind(runtimeID))
	if !ok {
		return "", nil, nil, ErrRuntimeNotSupported
	}
	if bundle.Models == nil {
		return "", nil, nil, ErrManagementCapabilityUnsupported
	}
	listed, err := s.ListInstances(ctx, runtimeID)
	if err != nil {
		return "", nil, nil, err
	}
	return listed.RuntimeInstallationID, bundle.Models, listed.Instances, nil
}

func (s *InstanceInventory) emitSharedModelOperation(previous, next operation.Operation, created bool) {
	if s == nil || s.events == nil {
		return
	}
	eventType := events.TypeForCommittedOperation(created, string(previous.Status), string(next.Status))
	if eventType == "" {
		return
	}
	payload := events.OperationPayload{
		OperationID: next.ID, Type: string(next.Type), Status: string(next.Status), Stage: string(next.Stage), CorrelationID: next.CorrelationID,
	}
	if next.ErrorCode != "" {
		payload.ErrorCode = string(next.ErrorCode)
	}
	s.events.Publish(events.NewOperationEvent(eventType, payload, s.now()))
}

func modelProfileView(row sqlite.ModelProfile) ModelProfileView {
	return ModelProfileView{
		ID: row.ID, ProviderConnectionID: row.ProviderConnectionID, DisplayName: row.DisplayName,
		SelectedModelIDs: append([]string(nil), row.SelectedModelIDs...), DefaultModelID: row.DefaultModelID,
		Revision: row.Revision, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func hasModelPreset(models yorvaruntime.ModelConfigurator, presetID string) bool {
	for _, preset := range models.ListProviderPresets() {
		if preset.ID == presetID {
			return true
		}
	}
	return false
}

func validModelResourceName(value string) bool {
	value = strings.TrimSpace(value)
	return len(value) > 0 && len(value) <= 80
}

func newModelResourceID(prefix string) (string, error) {
	value := make([]byte, 12)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return prefix + "-" + hex.EncodeToString(value), nil
}

func modelProfileApplicationFingerprint(profile sqlite.ModelProfile, instanceIDs []string, mode string) string {
	payload := profile.ID + "\x00" + strconv.Itoa(profile.Revision) + "\x00" + mode + "\x00" + strings.Join(instanceIDs, "\x00")
	sum := sha256.Sum256([]byte(payload))
	return "model-profile-sha256:" + hex.EncodeToString(sum[:])
}

func sharedModelErrorCode(err error) yorvaruntime.ErrorCode {
	switch {
	case errors.Is(err, ErrInstanceNotAvailable):
		return yorvaruntime.ErrorInstanceNotAvailable
	case errors.Is(err, yorvaruntime.ErrModelCredentialRequired), errors.Is(err, yorvaruntime.ErrModelCredentialWriteFailed):
		return yorvaruntime.ErrorModelCredentialWriteFailed
	case errors.Is(err, yorvaruntime.ErrInstanceConfigConflict):
		return yorvaruntime.ErrorInstanceConfigConflict
	case errors.Is(err, yorvaruntime.ErrModelConfigInvalid):
		return yorvaruntime.ErrorModelConfigInvalid
	default:
		return yorvaruntime.ErrorModelConfigIncomplete
	}
}

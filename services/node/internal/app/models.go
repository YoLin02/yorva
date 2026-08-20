package app

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/instance"
	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

var ErrInstanceNotAvailable = errors.New("instance is not available")

type ModelConfigurationView struct {
	ProviderPresetID     string
	ModelID              string
	State                yorvaruntime.ModelConfigurationState
	CredentialConfigured bool
	ObservedAt           time.Time
	Validation           ModelValidationSummary
}

type ModelValidationSummary struct {
	State       string
	ErrorCode   yorvaruntime.ErrorCode
	CompletedAt *time.Time
}

type ModelCredentialView struct {
	ProviderPresetID string
	Configured       bool
	ObservedAt       time.Time
}

func (s *InstanceInventory) ListModelProviderPresets(context.Context) ([]yorvaruntime.ModelProviderPreset, error) {
	if s == nil || s.discovery == nil || s.discovery.registry == nil {
		return nil, ErrRuntimeNotSupported
	}
	bundle, ok := s.discovery.registry.Get(yorvaruntime.Kind(hermesRuntimeID))
	if !ok || bundle.Models == nil {
		return nil, ErrRuntimeNotSupported
	}
	return bundle.Models.ListProviderPresets(), nil
}

func (s *InstanceInventory) GetModelConfiguration(ctx context.Context, instanceID string) (ModelConfigurationView, error) {
	row, models, installation, unlock, err := s.resolveModelTarget(ctx, instanceID, false)
	if err != nil {
		return ModelConfigurationView{}, err
	}
	defer unlock()
	config, err := models.ReadModelConfig(ctx, installation, row.NativeID)
	return s.modelConfigurationView(ctx, instanceID, config, err)
}

func (s *InstanceInventory) PatchModelConfiguration(ctx context.Context, instanceID, presetID, modelID string) (ModelConfigurationView, error) {
	row, models, installation, unlock, err := s.resolveModelTarget(ctx, instanceID, true)
	if err != nil {
		return ModelConfigurationView{}, err
	}
	defer unlock()
	config, err := models.ApplyModelConfig(ctx, installation, row.NativeID, presetID, modelID)
	return s.modelConfigurationView(ctx, instanceID, config, err)
}

func (s *InstanceInventory) GetModelCredential(ctx context.Context, instanceID string) (ModelCredentialView, error) {
	row, models, installation, unlock, err := s.resolveModelTarget(ctx, instanceID, false)
	if err != nil {
		return ModelCredentialView{}, err
	}
	defer unlock()
	configuration, err := models.ReadModelConfig(ctx, installation, row.NativeID)
	if err != nil {
		return ModelCredentialView{}, err
	}
	if configuration.ProviderPresetID == "" {
		return ModelCredentialView{ObservedAt: s.now()}, nil
	}
	status, err := models.ModelCredentialStatus(ctx, installation, row.NativeID, configuration.ProviderPresetID)
	return s.modelCredentialView(status), err
}

func (s *InstanceInventory) SaveModelCredentialConfiguration(ctx context.Context, instanceID, presetID, modelID string, secret []byte) (ModelConfigurationView, error) {
	row, models, installation, unlock, err := s.resolveModelTarget(ctx, instanceID, true)
	if err != nil {
		return ModelConfigurationView{}, err
	}
	defer unlock()
	if err := models.ValidateModelSelection(presetID, modelID); err != nil {
		return ModelConfigurationView{}, err
	}
	if _, err := models.SetModelCredential(ctx, installation, row.NativeID, presetID, secret); err != nil {
		return ModelConfigurationView{}, err
	}
	configuration, err := models.ApplyModelConfig(ctx, installation, row.NativeID, presetID, modelID)
	return s.modelConfigurationView(ctx, instanceID, configuration, err)
}

func (s *InstanceInventory) DeleteModelCredential(ctx context.Context, instanceID string) (ModelCredentialView, error) {
	row, models, installation, unlock, err := s.resolveModelTarget(ctx, instanceID, true)
	if err != nil {
		return ModelCredentialView{}, err
	}
	defer unlock()
	configuration, err := models.ReadModelConfig(ctx, installation, row.NativeID)
	if err != nil {
		return ModelCredentialView{}, err
	}
	if configuration.ProviderPresetID == "" {
		return ModelCredentialView{}, yorvaruntime.ErrModelConfigInvalid
	}
	status, err := models.DeleteModelCredential(ctx, installation, row.NativeID, configuration.ProviderPresetID)
	return s.modelCredentialView(status), err
}

func (s *InstanceInventory) resolveModelTarget(ctx context.Context, instanceID string, mutation bool) (instance.Instance, yorvaruntime.ModelConfigurator, yorvaruntime.ModelInstallation, func(), error) {
	if s == nil || s.db == nil || s.discovery == nil || s.source == nil || instanceID == "" {
		return instance.Instance{}, nil, yorvaruntime.ModelInstallation{}, nil, ErrInstanceNotFound
	}
	row, err := s.db.GetInstance(ctx, instanceID)
	if errors.Is(err, sql.ErrNoRows) {
		return instance.Instance{}, nil, yorvaruntime.ModelInstallation{}, nil, ErrInstanceNotFound
	}
	if err != nil {
		return instance.Instance{}, nil, yorvaruntime.ModelInstallation{}, nil, err
	}
	unlock := s.lockInstallation(row.RuntimeInstallationID)
	fail := func(err error) (instance.Instance, yorvaruntime.ModelConfigurator, yorvaruntime.ModelInstallation, func(), error) {
		unlock()
		return instance.Instance{}, nil, yorvaruntime.ModelInstallation{}, nil, err
	}
	if mutation {
		if _, active, activeErr := s.db.ActiveInstanceMutation(ctx, row.RuntimeInstallationID); activeErr != nil {
			return fail(activeErr)
		} else if active {
			return fail(yorvaruntime.ErrInstanceConfigConflict)
		}
	}
	accepted, err := s.db.GetAcceptedInstallationByID(ctx, row.RuntimeInstallationID)
	if err != nil {
		return fail(ErrRuntimeNotSupported)
	}
	detected, err := s.discovery.Detect(ctx, yorvaruntime.Kind(hermesRuntimeID))
	if err != nil || detected.State != yorvaruntime.DiscoverySupported || detected.Selected == nil ||
		detected.Selected.Path == "" || detected.Selected.Path != accepted.InstallPath {
		return fail(ErrRuntimeNotSupported)
	}
	listed, err := s.reconcileLocked(ctx, accepted.ID, detected.Selected.Path)
	if err != nil || listed.Freshness != "FRESH" {
		return fail(ErrInstanceNotAvailable)
	}
	row, err = s.db.GetInstance(ctx, instanceID)
	if err != nil || row.Availability != instance.Available {
		return fail(ErrInstanceNotAvailable)
	}
	bundle, ok := s.discovery.registry.Get(yorvaruntime.Kind(hermesRuntimeID))
	if !ok || bundle.Models == nil {
		return fail(ErrRuntimeNotSupported)
	}
	installation := yorvaruntime.ModelInstallation{Executable: detected.Selected.Path, Version: detected.Selected.Version}
	return row, bundle.Models, installation, unlock, nil
}

func (s *InstanceInventory) modelConfigurationView(ctx context.Context, instanceID string, config yorvaruntime.ModelConfiguration, configErr error) (ModelConfigurationView, error) {
	view := ModelConfigurationView{
		ProviderPresetID:     config.ProviderPresetID,
		ModelID:              config.ModelID,
		State:                config.State,
		CredentialConfigured: config.CredentialConfigured,
		ObservedAt:           s.now(),
	}
	latest, ok, err := s.db.LatestCompletedModelValidation(ctx, instanceID)
	if err != nil && configErr == nil {
		return view, yorvaruntime.ErrModelConfigQueryFailed
	}
	if ok && latest.SourcePin == modelConfigFingerprint(config.ProviderPresetID, config.ModelID) {
		view.Validation = modelValidationSummary(latest)
	} else {
		view.Validation.State = "NOT_RUN"
	}
	return view, configErr
}

func modelConfigFingerprint(presetID, modelID string) string {
	sum := sha256.Sum256([]byte(presetID + "\x00" + modelID))
	return "model-config-sha256:" + hex.EncodeToString(sum[:])
}

func (s *InstanceInventory) modelCredentialView(status yorvaruntime.ModelCredentialStatus) ModelCredentialView {
	return ModelCredentialView{ProviderPresetID: status.ProviderPresetID, Configured: status.Configured, ObservedAt: s.now()}
}

func modelValidationSummary(value operation.Operation) ModelValidationSummary {
	result := ModelValidationSummary{State: "UNKNOWN", ErrorCode: value.ErrorCode, CompletedAt: value.CompletedAt}
	if value.Status == operation.StatusSucceeded {
		result.State = "PASSED"
		result.ErrorCode = ""
	} else if value.Status == operation.StatusFailed && value.ErrorCode == yorvaruntime.ErrorModelValidationFailed {
		result.State = "FAILED"
	}
	return result
}

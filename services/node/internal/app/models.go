package app

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/instance"
	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

var ErrInstanceNotAvailable = errors.New("instance is not available")

type ModelConfigurationView struct {
	ProviderPresetID     string
	ModelID              string
	SelectedModelIDs     []string
	State                yorvaruntime.ModelConfigurationState
	CredentialConfigured bool
	ObservedAt           time.Time
	Validation           ModelValidationSummary
}

type ModelProviderCatalogView struct {
	ProviderPresetID string
	Items            []string
	FetchedAt        time.Time
}

const maxSelectedModels = 100

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

func (s *InstanceInventory) ListModelProviderPresets(_ context.Context, runtimeID string) ([]yorvaruntime.ModelProviderPreset, error) {
	if s == nil || s.discovery == nil || s.discovery.registry == nil {
		return nil, ErrRuntimeNotSupported
	}
	bundle, ok := s.discovery.registry.Get(yorvaruntime.Kind(runtimeID))
	if !ok {
		return nil, ErrRuntimeNotSupported
	}
	if bundle.Models == nil {
		return nil, ErrManagementCapabilityUnsupported
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
	view, viewErr := s.modelConfigurationView(ctx, instanceID, config, err)
	if viewErr == nil {
		view.SelectedModelIDs = s.readSelectedModelIDs(ctx, instanceID, models, config)
	}
	return view, viewErr
}

func (s *InstanceInventory) PatchModelConfiguration(ctx context.Context, instanceID, presetID, modelID string, selectedModelIDs []string) (ModelConfigurationView, error) {
	row, models, installation, unlock, err := s.resolveModelTarget(ctx, instanceID, true)
	if err != nil {
		return ModelConfigurationView{}, err
	}
	defer unlock()
	if err := validateSelectedModels(models, presetID, modelID, selectedModelIDs); err != nil {
		return ModelConfigurationView{}, err
	}
	config, err := models.ApplyModelConfig(ctx, installation, row.NativeID, presetID, modelID)
	view, viewErr := s.modelConfigurationView(ctx, instanceID, config, err)
	if viewErr == nil {
		if persistErr := s.writeSelectedModelIDs(ctx, instanceID, presetID, selectedModelIDs); persistErr != nil {
			return view, yorvaruntime.ErrModelConfigIncomplete
		}
		view.SelectedModelIDs = append([]string(nil), selectedModelIDs...)
	}
	return view, viewErr
}

func (s *InstanceInventory) FetchModelProviderCatalog(ctx context.Context, instanceID, presetID string, secret []byte) (ModelProviderCatalogView, error) {
	_, models, _, unlock, err := s.resolveModelTarget(ctx, instanceID, false)
	if err != nil {
		return ModelProviderCatalogView{}, err
	}
	unlock()
	fetcher, ok := models.(yorvaruntime.ModelCatalogFetcher)
	if !ok {
		return ModelProviderCatalogView{}, yorvaruntime.ErrModelProviderUnsupported
	}
	items, err := fetcher.FetchProviderModels(ctx, presetID, secret)
	if err != nil {
		return ModelProviderCatalogView{}, err
	}
	return ModelProviderCatalogView{ProviderPresetID: presetID, Items: items, FetchedAt: s.now()}, nil
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

func (s *InstanceInventory) SaveModelCredentialConfiguration(ctx context.Context, instanceID, presetID, modelID string, selectedModelIDs []string, secret []byte) (ModelConfigurationView, error) {
	return s.saveModelCredentialConfiguration(ctx, instanceID, presetID, modelID, selectedModelIDs, secret, "")
}

func (s *InstanceInventory) saveModelCredentialConfiguration(ctx context.Context, instanceID, presetID, modelID string, selectedModelIDs []string, secret []byte, ownOperationID string) (ModelConfigurationView, error) {
	row, models, installation, unlock, err := s.resolveModelTargetForOperation(ctx, instanceID, true, ownOperationID)
	if err != nil {
		return ModelConfigurationView{}, err
	}
	defer unlock()
	if err := validateSelectedModels(models, presetID, modelID, selectedModelIDs); err != nil {
		return ModelConfigurationView{}, err
	}
	if _, err := models.SetModelCredential(ctx, installation, row.NativeID, presetID, secret); err != nil {
		return ModelConfigurationView{}, err
	}
	configuration, err := models.ApplyModelConfig(ctx, installation, row.NativeID, presetID, modelID)
	if err != nil && configuration.State == "" {
		configuration.State = yorvaruntime.ModelConfigurationUnconfigured
	}
	if err != nil && !errors.Is(err, context.Canceled) &&
		!errors.Is(err, yorvaruntime.ErrInstanceConfigConflict) &&
		!errors.Is(err, yorvaruntime.ErrModelConfigInvalid) &&
		!errors.Is(err, yorvaruntime.ErrModelProviderUnsupported) {
		err = yorvaruntime.ErrModelConfigIncomplete
	}
	view, viewErr := s.modelConfigurationView(ctx, instanceID, configuration, err)
	if viewErr == nil {
		if persistErr := s.writeSelectedModelIDs(ctx, instanceID, presetID, selectedModelIDs); persistErr != nil {
			return view, yorvaruntime.ErrModelConfigIncomplete
		}
		view.SelectedModelIDs = append([]string(nil), selectedModelIDs...)
	}
	return view, viewErr
}

type selectedModelsSetting struct {
	ProviderPresetID string   `json:"providerPresetId"`
	ModelIDs         []string `json:"modelIds"`
}

func validateSelectedModels(models yorvaruntime.ModelConfigurator, presetID, defaultModelID string, selected []string) error {
	if len(selected) == 0 || len(selected) > maxSelectedModels {
		return yorvaruntime.ErrModelConfigInvalid
	}
	foundDefault := false
	seen := make(map[string]struct{}, len(selected))
	for _, modelID := range selected {
		if _, exists := seen[modelID]; exists || models.ValidateModelSelection(presetID, modelID) != nil {
			return yorvaruntime.ErrModelConfigInvalid
		}
		seen[modelID] = struct{}{}
		foundDefault = foundDefault || modelID == defaultModelID
	}
	if !foundDefault {
		return yorvaruntime.ErrModelConfigInvalid
	}
	return nil
}

func (s *InstanceInventory) selectedModelsSettingKey(instanceID string) string {
	return fmt.Sprintf("model-selection:%s", instanceID)
}

func (s *InstanceInventory) readSelectedModelIDs(ctx context.Context, instanceID string, models yorvaruntime.ModelConfigurator, config yorvaruntime.ModelConfiguration) []string {
	fallback := []string{}
	if config.ModelID != "" {
		fallback = append(fallback, config.ModelID)
	}
	payload, ok, err := s.db.GetAppSetting(ctx, s.selectedModelsSettingKey(instanceID))
	if err != nil || !ok {
		return fallback
	}
	var stored selectedModelsSetting
	if json.Unmarshal(payload, &stored) != nil || stored.ProviderPresetID != config.ProviderPresetID ||
		validateSelectedModels(models, stored.ProviderPresetID, config.ModelID, stored.ModelIDs) != nil {
		return fallback
	}
	return append([]string(nil), stored.ModelIDs...)
}

func (s *InstanceInventory) writeSelectedModelIDs(ctx context.Context, instanceID, presetID string, selected []string) error {
	payload, err := json.Marshal(selectedModelsSetting{ProviderPresetID: presetID, ModelIDs: selected})
	if err != nil {
		return err
	}
	return s.db.PutAppSetting(ctx, s.selectedModelsSettingKey(instanceID), payload, s.now())
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
	return s.resolveModelTargetForOperation(ctx, instanceID, mutation, "")
}

func (s *InstanceInventory) resolveModelTargetForOperation(ctx context.Context, instanceID string, mutation bool, ownOperationID string) (instance.Instance, yorvaruntime.ModelConfigurator, yorvaruntime.ModelInstallation, func(), error) {
	if s == nil || s.db == nil || s.discovery == nil || instanceID == "" {
		return instance.Instance{}, nil, yorvaruntime.ModelInstallation{}, nil, ErrInstanceNotFound
	}
	row, err := s.db.GetInstance(ctx, instanceID)
	if errors.Is(err, sql.ErrNoRows) {
		return instance.Instance{}, nil, yorvaruntime.ModelInstallation{}, nil, ErrInstanceNotFound
	}
	if err != nil {
		return instance.Instance{}, nil, yorvaruntime.ModelInstallation{}, nil, err
	}
	unlockInstallation := s.lockInstallation(row.RuntimeInstallationID)
	unlockInstance := s.lockInstance(row.ID)
	unlock := func() {
		unlockInstance()
		unlockInstallation()
	}
	fail := func(err error) (instance.Instance, yorvaruntime.ModelConfigurator, yorvaruntime.ModelInstallation, func(), error) {
		unlock()
		return instance.Instance{}, nil, yorvaruntime.ModelInstallation{}, nil, err
	}
	if mutation {
		if activeOperation, active, activeErr := s.db.ActiveInstanceMutation(ctx, row.RuntimeInstallationID); activeErr != nil {
			return fail(activeErr)
		} else if active && activeOperation.ID != ownOperationID {
			return fail(yorvaruntime.ErrInstanceConfigConflict)
		}
		if _, active, activeErr := s.db.ActiveInstanceRuntimeMutation(ctx, row.ID); activeErr != nil {
			return fail(activeErr)
		} else if active {
			return fail(yorvaruntime.ErrInstanceConfigConflict)
		}
	}
	target, err := s.resolveAcceptedInstallation(ctx, row.RuntimeInstallationID)
	if err != nil {
		return fail(ErrRuntimeNotSupported)
	}
	if target.Bundle.Models == nil {
		return fail(ErrManagementCapabilityUnsupported)
	}
	listed, err := s.reconcileLocked(ctx, row.RuntimeInstallationID, target.Installation.Path)
	if err != nil || listed.Freshness != "FRESH" {
		return fail(ErrInstanceNotAvailable)
	}
	row, err = s.db.GetInstance(ctx, instanceID)
	if err != nil || row.Availability != instance.Available {
		return fail(ErrInstanceNotAvailable)
	}
	installation := yorvaruntime.ModelInstallation{Executable: target.Installation.Path, Version: target.Installation.Version}
	return row, target.Bundle.Models, installation, unlock, nil
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

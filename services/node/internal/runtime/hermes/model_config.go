package hermes

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"regexp"
	"strings"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const (
	modelIDMaxLength          = 200
	modelConfigOutputMaxBytes = 8 * 1024
)

var modelIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,199}$`)

type modelCommandFunc func(context.Context, string, string, []string, bool) commandResult

type ModelManager struct {
	home          func() string
	run           modelCommandFunc
	runValidation modelCommandFunc
	credentials   credentialStore
}

func NewModelManager() *ModelManager {
	root := officialHermesHome()
	return &ModelManager{
		home:          officialHermesHome,
		run:           runModelConfigCommand,
		runValidation: runModelValidationCommand,
		credentials:   credentialStore{root: root},
	}
}

func (m *ModelManager) ListProviderPresets() []yorvaruntime.ModelProviderPreset {
	return ListModelProviderPresets()
}

func (m *ModelManager) ValidateModelSelection(presetID, modelID string) error {
	if _, err := lookupModelProviderPreset(presetID); err != nil {
		return err
	}
	return validateModelID(modelID)
}

func (m *ModelManager) ReadModelConfig(ctx context.Context, installation yorvaruntime.ModelInstallation, nativeID string) (yorvaruntime.ModelConfiguration, error) {
	if err := validateModelTarget(installation, nativeID); err != nil {
		return yorvaruntime.ModelConfiguration{}, err
	}
	home := m.home()
	if home == "" || !filepath.IsAbs(home) {
		return yorvaruntime.ModelConfiguration{}, yorvaruntime.ErrModelConfigQueryFailed
	}
	first, err := m.readNativeConfig(ctx, installation.Executable, home, nativeID)
	if err != nil {
		return yorvaruntime.ModelConfiguration{}, err
	}
	confirmed, err := m.readNativeConfig(ctx, installation.Executable, home, nativeID)
	if err != nil {
		return yorvaruntime.ModelConfiguration{}, err
	}
	if confirmed != first {
		return yorvaruntime.ModelConfiguration{}, yorvaruntime.ErrModelConfigQueryFailed
	}
	return m.normalizeConfig(nativeID, confirmed.provider, confirmed.modelID)
}

func (m *ModelManager) ApplyModelConfig(ctx context.Context, installation yorvaruntime.ModelInstallation, nativeID, presetID, modelID string) (yorvaruntime.ModelConfiguration, error) {
	if err := validateModelTarget(installation, nativeID); err != nil {
		return yorvaruntime.ModelConfiguration{}, err
	}
	if err := m.ValidateModelSelection(presetID, modelID); err != nil {
		return yorvaruntime.ModelConfiguration{}, err
	}
	preset, _ := lookupModelProviderPreset(presetID)
	status, err := m.credentials.Status(nativeID, presetID)
	if err != nil {
		return yorvaruntime.ModelConfiguration{}, yorvaruntime.ErrModelConfigQueryFailed
	}
	if !status.Configured {
		return yorvaruntime.ModelConfiguration{}, yorvaruntime.ErrModelCredentialRequired
	}
	home := m.home()
	if home == "" || !filepath.IsAbs(home) {
		return yorvaruntime.ModelConfiguration{}, yorvaruntime.ErrModelConfigApplyFailed
	}
	before, err := m.readNativeConfig(ctx, installation.Executable, home, nativeID)
	if err != nil {
		return yorvaruntime.ModelConfiguration{}, err
	}
	if before.provider == preset.hermesProviderID && before.modelID == modelID {
		return m.normalizeConfig(nativeID, before.provider, before.modelID)
	}
	confirmed, err := m.readNativeConfig(ctx, installation.Executable, home, nativeID)
	if err != nil {
		return yorvaruntime.ModelConfiguration{}, err
	}
	if confirmed != before {
		observed, _ := m.normalizeConfig(nativeID, confirmed.provider, confirmed.modelID)
		return observed, yorvaruntime.ErrInstanceConfigConflict
	}
	if before.provider != preset.hermesProviderID {
		if err := m.setScalar(ctx, installation.Executable, home, nativeID, modelProviderConfigKey, preset.hermesProviderID); err != nil {
			return yorvaruntime.ModelConfiguration{}, err
		}
		observed, readErr := m.readNativeConfig(ctx, installation.Executable, home, nativeID)
		if readErr != nil {
			return yorvaruntime.ModelConfiguration{}, yorvaruntime.ErrModelConfigIncomplete
		}
		if observed.provider != preset.hermesProviderID || observed.modelID != before.modelID {
			safe, _ := m.normalizeConfig(nativeID, observed.provider, observed.modelID)
			return safe, yorvaruntime.ErrInstanceConfigConflict
		}
		before = observed
	}
	if before.modelID != modelID {
		if err := m.setScalar(ctx, installation.Executable, home, nativeID, modelDefaultConfigKey, modelID); err != nil {
			observed, _ := m.ReadModelConfig(ctx, installation, nativeID)
			return observed, yorvaruntime.ErrModelConfigIncomplete
		}
	}
	final, err := m.ReadModelConfig(ctx, installation, nativeID)
	if err != nil {
		return yorvaruntime.ModelConfiguration{}, yorvaruntime.ErrModelConfigIncomplete
	}
	if final.ProviderPresetID != presetID || final.ModelID != modelID || final.State != yorvaruntime.ModelConfigurationConfigured {
		return final, yorvaruntime.ErrModelConfigIncomplete
	}
	return final, nil
}

type nativeModelConfig struct {
	provider string
	modelID  string
}

func (m *ModelManager) readNativeConfig(ctx context.Context, executable, home, nativeID string) (nativeModelConfig, error) {
	result := m.run(ctx, executable, home, modelConfigGetArgs(nativeID, modelConfigRootKey), false)
	if result.err != nil || result.exitCode != 0 || result.limited || result.timedOut {
		return nativeModelConfig{}, yorvaruntime.ErrModelConfigQueryFailed
	}
	config, err := parseNativeModelConfig(result.stdout)
	if err != nil {
		return nativeModelConfig{}, yorvaruntime.ErrModelConfigQueryFailed
	}
	return config, nil
}

func parseNativeModelConfig(output string) (nativeModelConfig, error) {
	trimmed := strings.TrimSpace(output)
	if len(trimmed) == 0 || len(trimmed) > modelConfigOutputMaxBytes {
		return nativeModelConfig{}, errors.New("invalid model config output")
	}

	if trimmed == `""` {
		return nativeModelConfig{}, nil
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(trimmed), &fields); err != nil || fields == nil {
		return nativeModelConfig{}, errors.New("invalid model config output")
	}
	config := nativeModelConfig{}
	for key, target := range map[string]*string{
		"provider": &config.provider,
		"default":  &config.modelID,
	} {
		raw, ok := fields[key]
		if !ok {
			continue
		}
		var value any
		if err := json.Unmarshal(raw, &value); err != nil {
			return nativeModelConfig{}, errors.New("invalid model config output")
		}
		stringValue, ok := value.(string)
		if !ok {
			return nativeModelConfig{}, errors.New("invalid model config output")
		}
		*target = stringValue
	}
	return config, nil
}

func (m *ModelManager) readScalar(ctx context.Context, executable, home, nativeID, key string) (string, error) {
	result := m.run(ctx, executable, home, modelConfigGetArgs(nativeID, key), false)
	if result.err != nil || result.exitCode != 0 || result.limited || result.timedOut {
		return "", yorvaruntime.ErrModelConfigQueryFailed
	}
	value, err := parseModelConfigScalar(result.stdout)
	if err != nil {
		return "", yorvaruntime.ErrModelConfigQueryFailed
	}
	return value, nil
}

func (m *ModelManager) setScalar(ctx context.Context, executable, home, nativeID, key, value string) error {
	result := m.run(ctx, executable, home, modelConfigSetArgs(nativeID, key, value), true)
	if errors.Is(result.err, context.Canceled) {
		return context.Canceled
	}
	if result.err != nil || result.exitCode != 0 || result.limited || result.timedOut {
		return yorvaruntime.ErrModelConfigApplyFailed
	}
	return nil
}

func (m *ModelManager) normalizeConfig(nativeID, provider, modelID string) (yorvaruntime.ModelConfiguration, error) {
	if len(provider) > modelIDMaxLength || (modelID != "" && validateModelID(modelID) != nil) {
		return yorvaruntime.ModelConfiguration{}, yorvaruntime.ErrModelConfigQueryFailed
	}
	result := yorvaruntime.ModelConfiguration{ModelID: modelID, State: yorvaruntime.ModelConfigurationUnconfigured}
	for _, preset := range qualifiedModelProviderPresets {
		if preset.hermesProviderID != provider {
			continue
		}
		result.ProviderPresetID = preset.safe.ID
		status, err := m.credentials.Status(nativeID, preset.safe.ID)
		if err != nil {
			return yorvaruntime.ModelConfiguration{}, yorvaruntime.ErrModelConfigQueryFailed
		}
		result.CredentialConfigured = status.Configured
		if status.Configured && modelID != "" {
			result.State = yorvaruntime.ModelConfigurationConfigured
		}
		break
	}
	return result, nil
}

func validateModelTarget(installation yorvaruntime.ModelInstallation, nativeID string) error {
	if installation.Version != modelSurfaceVersion {
		return yorvaruntime.ErrModelProviderUnsupported
	}
	if installation.Executable == "" || !filepath.IsAbs(installation.Executable) {
		return yorvaruntime.ErrModelConfigQueryFailed
	}
	normalized, err := officialNormalizeProfileName(nativeID)
	if err != nil || normalized != nativeID || officialValidateProfileName(normalized) != nil {
		return yorvaruntime.ErrModelConfigInvalid
	}
	return nil
}

func validateModelID(value string) error {
	if len(value) == 0 || len(value) > modelIDMaxLength || !modelIDPattern.MatchString(value) ||
		strings.Contains(value, ":/") || strings.Contains(value, "..") || strings.HasSuffix(value, "/") {
		return yorvaruntime.ErrModelConfigInvalid
	}
	for _, key := range []string{modelProviderConfigKey, modelDefaultConfigKey, contextEngineConfigKey} {
		if strings.EqualFold(value, key) {
			return yorvaruntime.ErrModelConfigInvalid
		}
	}
	for _, preset := range qualifiedModelProviderPresets {
		if strings.EqualFold(value, preset.credentialEnvName) {
			return yorvaruntime.ErrModelConfigInvalid
		}
	}
	return nil
}

func parseModelConfigScalar(output string) (string, error) {
	trimmed := strings.TrimSpace(output)
	if len(trimmed) == 0 || len(trimmed) > 2048 {
		return "", errors.New("invalid model config output")
	}
	var value string
	if err := json.Unmarshal([]byte(trimmed), &value); err != nil {
		return "", errors.New("invalid model config output")
	}
	if value == "" || strings.TrimSpace(value) != value {
		return "", errors.New("invalid model config output")
	}
	return value, nil
}

func runModelConfigCommand(ctx context.Context, executable, home string, args []string, mutation bool) commandResult {
	runner := newCommandRunner()
	if mutation {
		runner = newProfileMutationRunner()
	}
	runner.environment = func() []string { return profileCommandEnvironment(home) }
	return runner.run(ctx, commandInvocation{path: executable, executable: executable, args: args})
}

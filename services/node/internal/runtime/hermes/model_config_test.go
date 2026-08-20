package hermes

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestModelManagerReadsNormalizedConfiguration(t *testing.T) {
	manager, state, calls := newTestModelManager(t, "deepseek", "deepseek-v4-pro")
	got, err := manager.ReadModelConfig(context.Background(), testModelInstallation(), "coder")
	if err != nil {
		t.Fatal(err)
	}
	if got.ProviderPresetID != "deepseek" || got.ModelID != "deepseek-v4-pro" ||
		got.State != yorvaruntime.ModelConfigurationConfigured || !got.CredentialConfigured {
		t.Fatalf("configuration = %#v", got)
	}
	if *state == (nativeModelConfig{}) || len(*calls) != 4 {
		t.Fatalf("state/calls = %#v %#v", *state, *calls)
	}
}

func TestModelManagerTreatsUnknownProviderAsUnconfigured(t *testing.T) {
	manager, _, _ := newTestModelManager(t, "custom-provider", "custom-model")
	got, err := manager.ReadModelConfig(context.Background(), testModelInstallation(), "coder")
	if err != nil {
		t.Fatal(err)
	}
	if got.ProviderPresetID != "" || got.ModelID != "custom-model" || got.State != yorvaruntime.ModelConfigurationUnconfigured || got.CredentialConfigured {
		t.Fatalf("configuration = %#v", got)
	}
}

func TestModelManagerAppliesThroughOfficialConfigSurfaceAndReadsBack(t *testing.T) {
	manager, state, calls := newTestModelManager(t, "alibaba", "qwen3.7-plus")
	got, err := manager.ApplyModelConfig(context.Background(), testModelInstallation(), "coder", "deepseek", "deepseek-v4-pro")
	if err != nil {
		t.Fatal(err)
	}
	if got.ProviderPresetID != "deepseek" || got.ModelID != "deepseek-v4-pro" || got.State != yorvaruntime.ModelConfigurationConfigured {
		t.Fatalf("configuration = %#v", got)
	}
	if state.provider != "deepseek" || state.modelID != "deepseek-v4-pro" {
		t.Fatalf("native state = %#v", *state)
	}
	wantProvider := modelConfigSetArgs("coder", modelProviderConfigKey, "deepseek")
	wantModel := modelConfigSetArgs("coder", modelDefaultConfigKey, "deepseek-v4-pro")
	if !containsArgs(*calls, wantProvider) || !containsArgs(*calls, wantModel) {
		t.Fatalf("calls = %#v", *calls)
	}
	for _, args := range *calls {
		for _, arg := range args {
			if arg == "test-deepseek-key" {
				t.Fatalf("credential leaked into argv: %#v", args)
			}
		}
	}
}

func TestModelManagerRequiresCredentialAndValidatesClosedInputs(t *testing.T) {
	manager, _, calls := newTestModelManager(t, "alibaba", "qwen3.7-plus")
	if _, err := manager.ApplyModelConfig(context.Background(), testModelInstallation(), "coder", "qwen", "qwen3.7-max"); !errors.Is(err, yorvaruntime.ErrModelCredentialRequired) {
		t.Fatalf("missing credential error = %v", err)
	}
	if len(*calls) != 0 {
		t.Fatalf("native config queried before credential precondition: %#v", *calls)
	}
	if _, err := manager.ApplyModelConfig(context.Background(), testModelInstallation(), "coder", "unknown", "model"); !errors.Is(err, yorvaruntime.ErrModelProviderUnsupported) {
		t.Fatalf("provider error = %v", err)
	}
	if _, err := manager.ApplyModelConfig(context.Background(), testModelInstallation(), "coder", "deepseek", "https://example.test/model"); !errors.Is(err, yorvaruntime.ErrModelConfigInvalid) {
		t.Fatalf("model error = %v", err)
	}
	for _, modelID := range []string{"C:/secret", "model.provider", "DEEPSEEK_API_KEY"} {
		if err := manager.ValidateModelSelection("deepseek", modelID); !errors.Is(err, yorvaruntime.ErrModelConfigInvalid) {
			t.Fatalf("unsafe model %q error = %v", modelID, err)
		}
	}
}

func TestModelManagerReadFailsClosedOnChangingNativeConfig(t *testing.T) {
	manager, state, _ := newTestModelManager(t, "deepseek", "deepseek-v4-pro")
	modelReads := 0
	baseRun := manager.run
	manager.run = func(ctx context.Context, executable, home string, args []string, mutation bool) commandResult {
		result := baseRun(ctx, executable, home, args, mutation)
		if reflect.DeepEqual(args, modelConfigGetArgs("coder", modelDefaultConfigKey)) {
			modelReads++
			if modelReads == 1 {
				state.provider = "openrouter"
				state.modelID = "openai/gpt-5.4"
			}
		}
		return result
	}
	if _, err := manager.ReadModelConfig(context.Background(), testModelInstallation(), "coder"); !errors.Is(err, yorvaruntime.ErrModelConfigQueryFailed) {
		t.Fatalf("changing native config error = %v", err)
	}
}

func TestModelManagerDetectsExternalChangeAndPartialApply(t *testing.T) {
	manager, state, calls := newTestModelManager(t, "alibaba", "qwen3.7-plus")
	readCount := 0
	baseRun := manager.run
	manager.run = func(ctx context.Context, executable, home string, args []string, mutation bool) commandResult {
		if reflect.DeepEqual(args, modelConfigGetArgs("coder", modelProviderConfigKey)) {
			readCount++
			if readCount == 2 {
				state.provider = "openrouter"
			}
		}
		return baseRun(ctx, executable, home, args, mutation)
	}
	observed, err := manager.ApplyModelConfig(context.Background(), testModelInstallation(), "coder", "deepseek", "deepseek-v4-pro")
	if !errors.Is(err, yorvaruntime.ErrInstanceConfigConflict) || observed.ProviderPresetID != "openrouter" {
		t.Fatalf("external change = %#v, %v", observed, err)
	}
	if containsArgs(*calls, modelConfigSetArgs("coder", modelProviderConfigKey, "deepseek")) {
		t.Fatalf("mutation occurred after conflict: %#v", *calls)
	}

	manager, state, _ = newTestModelManager(t, "alibaba", "qwen3.7-plus")
	baseRun = manager.run
	manager.run = func(ctx context.Context, executable, home string, args []string, mutation bool) commandResult {
		if reflect.DeepEqual(args, modelConfigSetArgs("coder", modelDefaultConfigKey, "deepseek-v4-pro")) {
			return commandResult{exitCode: 1, err: errors.New("safe fake failure")}
		}
		return baseRun(ctx, executable, home, args, mutation)
	}
	observed, err = manager.ApplyModelConfig(context.Background(), testModelInstallation(), "coder", "deepseek", "deepseek-v4-pro")
	if !errors.Is(err, yorvaruntime.ErrModelConfigIncomplete) || observed.ProviderPresetID != "deepseek" || state.provider != "deepseek" || state.modelID != "qwen3.7-plus" {
		t.Fatalf("partial apply = %#v %#v, %v", observed, *state, err)
	}
}

func TestModelManagerRejectsUnsupportedVersionAndMalformedOutput(t *testing.T) {
	manager, _, _ := newTestModelManager(t, "deepseek", "deepseek-v4-pro")
	bad := testModelInstallation()
	bad.Version = "0.20.1"
	if _, err := manager.ReadModelConfig(context.Background(), bad, "coder"); !errors.Is(err, yorvaruntime.ErrModelProviderUnsupported) {
		t.Fatalf("version error = %v", err)
	}
	manager.run = func(context.Context, string, string, []string, bool) commandResult {
		return commandResult{stdout: "not-json"}
	}
	if _, err := manager.ReadModelConfig(context.Background(), testModelInstallation(), "coder"); !errors.Is(err, yorvaruntime.ErrModelConfigQueryFailed) {
		t.Fatalf("malformed output error = %v", err)
	}
}

func newTestModelManager(t *testing.T, provider, modelID string) (*ModelManager, *nativeModelConfig, *[][]string) {
	t.Helper()
	root := t.TempDir()
	profileRoot := filepath.Join(root, "profiles", "coder")
	if err := os.MkdirAll(profileRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profileRoot, modelCredentialFileName), []byte("DEEPSEEK_API_KEY=\"test-deepseek-key\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	state := &nativeModelConfig{provider: provider, modelID: modelID}
	calls := &[][]string{}
	manager := &ModelManager{home: func() string { return root }, credentials: credentialStore{root: root}}
	manager.run = func(_ context.Context, _, _ string, args []string, mutation bool) commandResult {
		*calls = append(*calls, append([]string(nil), args...))
		if len(args) < 5 || args[0] != "--profile" || args[1] != "coder" || args[2] != "config" {
			return commandResult{exitCode: 1, err: errors.New("unexpected command")}
		}
		switch args[3] {
		case "get":
			if mutation || len(args) != 6 || args[5] != "--json" {
				return commandResult{exitCode: 1, err: errors.New("unexpected get")}
			}
			value := state.provider
			if args[4] == modelDefaultConfigKey {
				value = state.modelID
			} else if args[4] == contextEngineConfigKey {
				value = defaultContextEngine
			} else if args[4] != modelProviderConfigKey {
				return commandResult{exitCode: 1, err: errors.New("unexpected key")}
			}
			return commandResult{stdout: `"` + value + `"`}
		case "set":
			if !mutation || len(args) != 6 {
				return commandResult{exitCode: 1, err: errors.New("unexpected set")}
			}
			if args[4] == modelProviderConfigKey {
				state.provider = args[5]
			} else if args[4] == modelDefaultConfigKey {
				state.modelID = args[5]
			} else {
				return commandResult{exitCode: 1, err: errors.New("unexpected key")}
			}
			return commandResult{}
		default:
			return commandResult{exitCode: 1, err: errors.New("unexpected action")}
		}
	}
	return manager, state, calls
}

func testModelInstallation() yorvaruntime.ModelInstallation {
	return yorvaruntime.ModelInstallation{Executable: filepath.Join(`C:\`, "hermes", "hermes.exe"), Version: modelSurfaceVersion}
}

func containsArgs(haystack [][]string, needle []string) bool {
	for _, item := range haystack {
		if reflect.DeepEqual(item, needle) {
			return true
		}
	}
	return false
}

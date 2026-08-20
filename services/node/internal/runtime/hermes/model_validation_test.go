package hermes

import (
	"context"
	"reflect"
	"strings"
	"testing"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestModelValidationUsesQualifiedToolsDisabledProfileSurface(t *testing.T) {
	manager, _, _ := newTestModelManager(t, "deepseek", "deepseek-v4-pro")
	var gotArgs []string
	manager.runValidation = func(_ context.Context, _, _ string, args []string, mutation bool) commandResult {
		if mutation {
			t.Fatal("validation was marked as a config mutation")
		}
		gotArgs = append([]string(nil), args...)
		return commandResult{stdout: "provider model response that must be discarded"}
	}
	result := manager.ValidateModel(context.Background(), testModelInstallation(), "coder", "deepseek", "deepseek-v4-pro")
	if result.State != yorvaruntime.ModelValidationPassed || result.ErrorCode != "" {
		t.Fatalf("result = %#v", result)
	}
	want := modelValidationArgs("coder", qualifiedModelProviderPresets[0], "deepseek-v4-pro")
	if !reflect.DeepEqual(gotArgs, want) || !containsValidationArg(gotArgs, toolsDisabledToolset) || !containsValidationArg(gotArgs, modelValidationPrompt) {
		t.Fatalf("validation args = %#v, want %#v", gotArgs, want)
	}
	if strings.Contains(strings.Join(gotArgs, " "), "test-deepseek-key") {
		t.Fatalf("credential leaked into argv: %#v", gotArgs)
	}
}

func TestModelValidationNormalizesProviderFailureTimeoutCancelAndOutputLimit(t *testing.T) {
	tests := []struct {
		name      string
		result    commandResult
		wantState yorvaruntime.ModelValidationState
		wantCode  yorvaruntime.ErrorCode
	}{
		{name: "provider rejection", result: commandResult{exitCode: 1, stderr: "raw provider response"}, wantState: yorvaruntime.ModelValidationFailed, wantCode: yorvaruntime.ErrorModelValidationFailed},
		{name: "timeout", result: commandResult{err: context.DeadlineExceeded, timedOut: true}, wantState: yorvaruntime.ModelValidationUnknown, wantCode: yorvaruntime.ErrorModelValidationTimedOut},
		{name: "cancel", result: commandResult{err: context.Canceled}, wantState: yorvaruntime.ModelValidationUnknown, wantCode: yorvaruntime.ErrorModelValidationCancelled},
		{name: "output limit", result: commandResult{err: errOutputLimit, limited: true, stdout: strings.Repeat("discard", 100)}, wantState: yorvaruntime.ModelValidationUnknown, wantCode: yorvaruntime.ErrorModelValidationOutputLimit},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager, _, _ := newTestModelManager(t, "deepseek", "deepseek-v4-pro")
			manager.runValidation = func(context.Context, string, string, []string, bool) commandResult { return test.result }
			got := manager.ValidateModel(context.Background(), testModelInstallation(), "coder", "deepseek", "deepseek-v4-pro")
			if got.State != test.wantState || got.ErrorCode != test.wantCode {
				t.Fatalf("result = %#v", got)
			}
		})
	}
}

func TestModelValidationFailsClosedForUnsafeContextAndMissingCredential(t *testing.T) {
	manager, _, _ := newTestModelManager(t, "deepseek", "deepseek-v4-pro")
	baseRun := manager.run
	manager.run = func(ctx context.Context, executable, home string, args []string, mutation bool) commandResult {
		if reflect.DeepEqual(args, modelConfigGetArgs("coder", contextEngineConfigKey)) {
			return commandResult{stdout: `"plugin-engine"`}
		}
		return baseRun(ctx, executable, home, args, mutation)
	}
	called := false
	manager.runValidation = func(context.Context, string, string, []string, bool) commandResult {
		called = true
		return commandResult{}
	}
	got := manager.ValidateModel(context.Background(), testModelInstallation(), "coder", "deepseek", "deepseek-v4-pro")
	if got.State != yorvaruntime.ModelValidationUnknown || got.ErrorCode != yorvaruntime.ErrorModelValidationUnsafe || called {
		t.Fatalf("unsafe context = %#v called=%v", got, called)
	}

	manager, _, _ = newTestModelManager(t, "deepseek", "deepseek-v4-pro")
	if _, err := manager.DeleteModelCredential(context.Background(), testModelInstallation(), "coder", "deepseek"); err != nil {
		t.Fatal(err)
	}
	got = manager.ValidateModel(context.Background(), testModelInstallation(), "coder", "deepseek", "deepseek-v4-pro")
	if got.State != yorvaruntime.ModelValidationFailed || got.ErrorCode != yorvaruntime.ErrorModelValidationFailed {
		t.Fatalf("missing credential = %#v", got)
	}
}

func TestModelValidationHonorsCancelledContextBeforeProcessStart(t *testing.T) {
	manager, _, _ := newTestModelManager(t, "deepseek", "deepseek-v4-pro")
	called := false
	manager.runValidation = func(context.Context, string, string, []string, bool) commandResult {
		called = true
		return commandResult{}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got := manager.ValidateModel(ctx, testModelInstallation(), "coder", "deepseek", "deepseek-v4-pro")
	if got.State != yorvaruntime.ModelValidationUnknown || got.ErrorCode != yorvaruntime.ErrorModelValidationCancelled || called {
		t.Fatalf("cancel result = %#v called=%v", got, called)
	}
}

func containsValidationArg(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

package hermes

import (
	"context"
	"errors"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const modelValidationTimeout = 45 * time.Second

func (m *ModelManager) ValidateModel(ctx context.Context, installation yorvaruntime.ModelInstallation, nativeID, presetID, modelID string) yorvaruntime.ModelValidationResult {
	validationCtx, cancel := context.WithTimeout(ctx, modelValidationTimeout)
	defer cancel()
	if err := validateModelTarget(installation, nativeID); err != nil {
		return unknownModelValidation(yorvaruntime.ErrorModelValidationUnsafe)
	}
	preset, err := lookupModelProviderPreset(presetID)
	if err != nil || validateModelID(modelID) != nil {
		return unknownModelValidation(yorvaruntime.ErrorModelValidationUnsafe)
	}
	credential, err := m.ModelCredentialStatus(validationCtx, installation, nativeID, presetID)
	if err != nil {
		return modelValidationContextFailure(validationCtx, yorvaruntime.ErrorModelValidationUnsafe)
	}
	if !credential.Configured {
		return yorvaruntime.ModelValidationResult{State: yorvaruntime.ModelValidationFailed, ErrorCode: yorvaruntime.ErrorModelValidationFailed}
	}
	home := m.home()
	contextEngine, err := m.readScalar(validationCtx, installation.Executable, home, nativeID, contextEngineConfigKey)
	if err != nil {
		return modelValidationContextFailure(validationCtx, yorvaruntime.ErrorModelValidationUnsafe)
	}
	if contextEngine != defaultContextEngine {
		return unknownModelValidation(yorvaruntime.ErrorModelValidationUnsafe)
	}
	run := m.runValidation
	if run == nil {
		run = runModelValidationCommand
	}
	result := run(validationCtx, installation.Executable, home, modelValidationArgs(nativeID, preset, modelID), false)
	switch {
	case errors.Is(result.err, context.Canceled), errors.Is(validationCtx.Err(), context.Canceled):
		return unknownModelValidation(yorvaruntime.ErrorModelValidationCancelled)
	case result.timedOut, errors.Is(result.err, context.DeadlineExceeded), errors.Is(validationCtx.Err(), context.DeadlineExceeded):
		return unknownModelValidation(yorvaruntime.ErrorModelValidationTimedOut)
	case result.limited:
		return unknownModelValidation(yorvaruntime.ErrorModelValidationOutputLimit)
	case result.err != nil || result.exitCode != 0:
		return yorvaruntime.ModelValidationResult{State: yorvaruntime.ModelValidationFailed, ErrorCode: yorvaruntime.ErrorModelValidationFailed}
	default:
		return yorvaruntime.ModelValidationResult{State: yorvaruntime.ModelValidationPassed}
	}
}

func modelValidationContextFailure(ctx context.Context, fallback yorvaruntime.ErrorCode) yorvaruntime.ModelValidationResult {
	switch {
	case errors.Is(ctx.Err(), context.Canceled):
		return unknownModelValidation(yorvaruntime.ErrorModelValidationCancelled)
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return unknownModelValidation(yorvaruntime.ErrorModelValidationTimedOut)
	default:
		return unknownModelValidation(fallback)
	}
}

func unknownModelValidation(code yorvaruntime.ErrorCode) yorvaruntime.ModelValidationResult {
	return yorvaruntime.ModelValidationResult{State: yorvaruntime.ModelValidationUnknown, ErrorCode: code}
}

func runModelValidationCommand(ctx context.Context, executable, home string, args []string, _ bool) commandResult {
	runner := newCommandRunner()
	runner.timeout = modelValidationTimeout
	runner.environment = func() []string { return profileCommandEnvironment(home) }
	return runner.run(ctx, commandInvocation{path: executable, executable: executable, args: args})
}

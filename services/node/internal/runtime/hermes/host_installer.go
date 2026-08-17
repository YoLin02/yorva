package hermes

import (
	"context"
	"runtime"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type processRun func(context.Context, installInvocation, time.Duration) commandResult

type HostInstaller struct {
	source     sourceClient
	run        processRun
	stateRoot  string
	home       func() string
	installDir func() string
	shell      func() (string, error)
}

func NewHostInstaller(stateRoot string) *HostInstaller {
	return &HostInstaller{
		source:     newSourceClient(pinnedOfficialSource()),
		run:        defaultInstallRun,
		stateRoot:  stateRoot,
		home:       officialHermesHome,
		installDir: officialInstallDir,
		shell:      trustedPowerShell,
	}
}

func defaultInstallRun(ctx context.Context, invocation installInvocation, timeout time.Duration) commandResult {
	return runInstallInvocation(ctx, newInstallCommandRunner(timeout, officialHermesHome()), invocation)
}

func (h *HostInstaller) PlatformSupported() bool {
	return runtime.GOOS == "windows"
}

func (h *HostInstaller) ManagedInstallDir() string {
	return h.installDir()
}

func (h *HostInstaller) ExpectedVersion() string {
	return officialPackageVersion
}

func (h *HostInstaller) ContainsManagedPath(path string) bool {
	root := h.installDir()
	return pathWithin(root, path) || pathWithin(h.home(), path)
}

func (h *HostInstaller) ValidateTarget(retry bool) error {
	return validateInstallTarget(h.home(), h.installDir(), retry)
}

func (h *HostInstaller) Apply(ctx context.Context, operationID string, report func(operation.Stage)) error {
	if !h.PlatformSupported() {
		return installError(yorvaruntime.ErrorRuntimeInstallPlatformUnsupported, errPlatform)
	}
	if report != nil {
		report(operation.StageSourceDownload)
	}
	script, err := h.source.Fetch(ctx, h.stateRoot, operationID)
	if err != nil {
		return err
	}
	defer func() { _ = cleanupFetchedScript(script) }()
	if report != nil {
		report(operation.StageSourceVerify)
	}
	if err := verifyRegularFile(script.Path, h.source.source.ExpectedSize, h.source.source.ExpectedSHA); err != nil {
		return err
	}
	powershell, err := h.shell()
	if err != nil {
		return err
	}
	home := h.home()
	installDir := h.installDir()
	if err := writeYorvaPartialMarker(installDir, officialCommit); err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	if report != nil {
		report(operation.StageProtocolVerify)
	}
	if err := h.probe(ctx, powershell, script.Path, "ProtocolVersion", home, installDir, 30*time.Second, parseProtocolOutput); err != nil {
		return err
	}
	if err := h.probe(ctx, powershell, script.Path, "Manifest", home, installDir, 30*time.Second, parseAndValidateManifest); err != nil {
		return err
	}
	if err := verifyRegularFile(script.Path, h.source.source.ExpectedSize, h.source.source.ExpectedSHA); err != nil {
		return err
	}
	for _, stage := range approvedInstallStages() {
		if report != nil {
			report(installStageName(stage))
		}
		if err := verifyRegularFile(script.Path, h.source.source.ExpectedSize, h.source.source.ExpectedSHA); err != nil {
			return err
		}
		if err := h.runStage(ctx, powershell, script.Path, stage, home, installDir); err != nil {
			return err
		}
	}
	if report != nil {
		report(operation.StageCleanup)
	}
	return nil
}

func (h *HostInstaller) probe(ctx context.Context, powershell, script, probe, home, installDir string, timeout time.Duration, parse func(string) error) error {
	invocation, err := probeInvocation(powershell, script, probe, home, installDir)
	if err != nil {
		return err
	}
	result := h.run(ctx, invocation, timeout)
	if result.limited {
		return installError(yorvaruntime.ErrorRuntimeInstallOutputLimit, errOutputLimit)
	}
	if result.timedOut || result.err != nil {
		if result.timedOut {
			return installError(yorvaruntime.ErrorRuntimeInstallTimeout, result.err)
		}
		if result.err == context.Canceled {
			return result.err
		}
		return installError(yorvaruntime.ErrorRuntimeInstallProtocolUnsupported, result.err)
	}
	return parse(result.stdout)
}

func parseProtocolOutput(output string) error {
	return parseProtocolVersion(output)
}

func (h *HostInstaller) runStage(ctx context.Context, powershell, script, stage, home, installDir string) error {
	invocation, err := stageInvocation(powershell, script, stage, home, installDir)
	if err != nil {
		return err
	}
	result := h.run(ctx, invocation, stageTimeout(stage))
	if result.limited {
		return installError(yorvaruntime.ErrorRuntimeInstallOutputLimit, errOutputLimit)
	}
	if result.timedOut {
		return installError(yorvaruntime.ErrorRuntimeInstallTimeout, result.err)
	}
	if result.err != nil {
		if result.err == context.Canceled {
			return result.err
		}
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, result.err)
	}
	parsed, err := parseStageResult(stage, result.stdout)
	if err != nil {
		return err
	}
	if !parsed.OK && !parsed.Skipped {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, errPlatform)
	}
	if stageRequiresSuccess(stage) && parsed.Skipped {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, errPlatform)
	}
	return nil
}

func stageRequiresSuccess(stage string) bool {
	return stage != "node" && stage != "node-deps" && stage != "system-packages"
}

func stageTimeout(stage string) time.Duration {
	switch stage {
	case "dependencies":
		return 30 * time.Minute
	case "node-deps":
		return 15 * time.Minute
	case "uv", "python", "git", "node", "system-packages", "repository", "venv":
		return 10 * time.Minute
	default:
		return 2 * time.Minute
	}
}

func installStageName(stage string) operation.Stage {
	switch stage {
	case "uv":
		return operation.StageInstallUV
	case "python":
		return operation.StageInstallPython
	case "git":
		return operation.StageInstallGit
	case "node":
		return operation.StageInstallNode
	case "system-packages":
		return operation.StageInstallSystemPackages
	case "repository":
		return operation.StageInstallRepository
	case "venv":
		return operation.StageInstallVenv
	case "dependencies":
		return operation.StageInstallDependencies
	case "node-deps":
		return operation.StageInstallNodeDeps
	case "path":
		return operation.StageInstallPath
	case "config-templates":
		return operation.StageInstallConfigTemplates
	case "bootstrap-marker":
		return operation.StageInstallBootstrapMarker
	default:
		return operation.Stage(stage)
	}
}

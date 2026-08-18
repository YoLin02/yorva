package hermes

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type processRun func(context.Context, installInvocation, time.Duration) commandResult

type HostInstaller struct {
	source             sourceClient
	archive            archiveClient
	run                processRun
	stateRoot          string
	embeddedSourcePath string
	home               func() string
	installDir         func() string
	shell              func() (string, error)
	logger             *slog.Logger
	operationID        string
	currentOp          operation.Operation
	previousOp         operation.Operation
	acquireArchive     func(context.Context, string) (string, string, error)
	verifyArchive      func(string) error
	userPath           func(string) bool
}

func NewHostInstaller(stateRoot string) *HostInstaller {
	return &HostInstaller{
		source:     newSourceClient(pinnedOfficialSource()),
		archive:    newArchiveClient(),
		run:        defaultInstallRun,
		stateRoot:  stateRoot,
		home:       officialHermesHome,
		installDir: officialInstallDir,
		shell:      trustedPowerShell,
	}
}

func (h *HostInstaller) WithLogger(logger *slog.Logger) *HostInstaller {
	if logger != nil {
		h.logger = logger
	}
	return h
}

func (h *HostInstaller) WithEmbeddedSource(path string) *HostInstaller {
	h.embeddedSourcePath = strings.TrimSpace(path)
	return h
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

func (h *HostInstaller) CanonicalPublicLauncher() string {
	return filepath.Join(h.installDir(), "bin", "hermes.exe")
}

func (h *HostInstaller) installIdentity(installDir string) ownershipIdentity {
	target := installDir
	if abs, err := filepath.Abs(installDir); err == nil {
		target = abs
	}
	kind := h.currentOp.TargetID
	if kind == "" {
		kind = "hermes"
	}
	return ownershipIdentity{
		OperationID: h.currentOp.ID,
		RuntimeKind: kind,
		Target:      target,
		SourcePin:   firstNonEmpty(h.currentOp.SourcePin, officialCommit),
		Nonce:       h.currentOp.OwnershipNonce,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func (h *HostInstaller) ExpectedPin() string {
	return officialCommit
}

func (h *HostInstaller) SetInstallIdentity(op operation.Operation) {
	h.currentOp = op
	h.operationID = op.ID
}

func (h *HostInstaller) ValidateTarget(retry bool, previous operation.Operation) error {
	h.previousOp = previous
	return validateInstallTarget(h.home(), h.installDir(), retry, previous, officialCommit)
}

func (h *HostInstaller) Apply(ctx context.Context, operationID string, report func(operation.Stage, string)) error {
	if h.currentOp.ID == "" {
		h.SetInstallIdentity(operation.Operation{
			ID:             operationID,
			TargetID:       "hermes",
			SourcePin:      officialCommit,
			OwnershipNonce: "own_apply_" + operationID,
		})
	}
	if !h.PlatformSupported() {
		return installError(yorvaruntime.ErrorRuntimeInstallPlatformUnsupported, errPlatform)
	}
	if report != nil {
		report(operation.StageSourceDownload, "")
	}
	workDir, err := operationPrivateDir(h.stateRoot, operationID)
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(workDir) }()
	archivePath, origin, err := h.resolveArchive(ctx, workDir)
	if err != nil {
		return err
	}
	if report != nil {
		report(operation.StageSourceVerify, "")
	}
	if h.acquireArchive == nil {
		if err := h.checkArchive(archivePath); err != nil {
			return err
		}
	}
	script, err := h.obtainScript(ctx, archivePath, workDir)
	if err != nil {
		return err
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
	if err := writeOwnershipRecord(installDir, h.installIdentity(installDir)); err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	if report != nil {
		report(operation.StageProtocolVerify, "")
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
			note := ""
			if stage == "repository" && origin == sourceOriginBundled {
				note = warningBundledUsed
			}
			report(installStageName(stage), note)
		}
		if err := verifyRegularFile(script.Path, h.source.source.ExpectedSize, h.source.source.ExpectedSHA); err != nil {
			return err
		}
		if stage == "node" || stage == "node-deps" {
			h.debug("installer.stage.skipped", "stage", stage, "reason", "yorva-managed-prerequisite")
			continue
		}
		if stage == "repository" {
			if err := h.materializeRepository(ctx, archivePath, origin, workDir, installDir); err != nil {
				return err
			}
			if report != nil && origin == sourceOriginBundled {
				report(operation.StageInstallRepository, warningSourcePrepared)
			}
			continue
		}
		if err := h.runStage(ctx, powershell, script.Path, stage, home, installDir); err != nil {
			return err
		}
	}
	if err := h.verifyPublicLauncher(ctx); err != nil {
		return err
	}
	if report != nil {
		report(operation.StageCleanup, "")
	}
	return nil
}

func (h *HostInstaller) verifyPublicLauncher(ctx context.Context) error {
	launcher := h.CanonicalPublicLauncher()
	if !isRegularFile(launcher) || !pathWithin(h.installDir(), launcher) {
		return installError(yorvaruntime.ErrorRuntimeInstallPostcheckFailed, errors.New("canonical public launcher is missing"))
	}
	if h.userPath != nil && !h.userPath(filepath.Join(h.installDir(), "bin")) {
		return installError(yorvaruntime.ErrorRuntimeInstallPostcheckFailed, errors.New("user PATH does not contain the Hermes launcher directory"))
	}
	if h.userPath == nil && !userPathContainsDir(filepath.Join(h.installDir(), "bin")) {
		return installError(yorvaruntime.ErrorRuntimeInstallPostcheckFailed, errors.New("user PATH does not contain the Hermes launcher directory"))
	}
	_ = ctx
	return nil
}

func (h *HostInstaller) resolveArchive(ctx context.Context, workDir string) (string, string, error) {
	if h.acquireArchive != nil {
		return h.acquireArchive(ctx, workDir)
	}
	downloaded := filepath.Join(workDir, "hermes-source.zip")
	err := h.archive.download(ctx, downloaded)
	if err == nil {
		h.debug("source.archive.official", "origin", sourceOriginOfficial)
		return downloaded, sourceOriginOfficial, nil
	}
	if !isTransportArchiveError(err) {
		h.debug("source.archive.integrity", archiveLogFields(err, sourceOriginOfficial)...)
		return "", "", err
	}
	h.debug("source.archive.official_unavailable", archiveLogFields(err, sourceOriginOfficial)...)
	if h.embeddedSourcePath == "" {
		return "", "", err
	}
	if verifyErr := h.checkArchive(h.embeddedSourcePath); verifyErr != nil {
		return "", "", verifyErr
	}
	h.debug("source.archive.bundled", "origin", sourceOriginBundled)
	return h.embeddedSourcePath, sourceOriginBundled, nil
}

func (h *HostInstaller) obtainScript(ctx context.Context, archivePath, workDir string) (fetchedScript, error) {
	scriptPath := filepath.Join(workDir, "install.ps1")
	if err := officialScriptFromArchive(archivePath, scriptPath); err == nil {
		return fetchedScript{Directory: workDir, Path: scriptPath, SHA256: officialScriptSHA256, Size: officialScriptSize}, nil
	}
	h.debug("source.script.archive_unusable")
	fetched, err := h.source.Fetch(ctx, h.stateRoot, h.operationID+"-script")
	if err != nil {
		return fetchedScript{}, err
	}
	if err := copyRegularFile(fetched.Path, scriptPath); err != nil {
		_ = cleanupFetchedScript(fetched)
		return fetchedScript{}, err
	}
	_ = cleanupFetchedScript(fetched)
	return fetchedScript{Directory: workDir, Path: scriptPath, SHA256: fetched.SHA256, Size: fetched.Size}, nil
}

func (h *HostInstaller) materializeRepository(ctx context.Context, archivePath, origin, workDir, installDir string) error {
	if origin == sourceOriginBundled {
		h.debug("source.materialize", "warning", warningBundledUsed)
	}
	if err := requireExtractBudget(workDir, h.archive.diskFree); err != nil {
		return err
	}
	if err := requireExtractBudget(installDir, h.archive.diskFree); err != nil {
		return err
	}
	staging := filepath.Join(workDir, "staging")
	if err := extractOfficialArchive(ctx, archivePath, staging); err != nil {
		_ = os.RemoveAll(staging)
		return err
	}
	if h.acquireArchive == nil {
		if err := verifyOfficialExtractedIdentity(staging); err != nil {
			_ = os.RemoveAll(staging)
			return err
		}
	} else if err := verifyExtractedLayout(staging); err != nil {
		_ = os.RemoveAll(staging)
		return err
	}
	if err := replaceOwnedTree(ctx, staging, installDir, h.installIdentity(installDir), h.previousOp); err != nil {
		_ = os.RemoveAll(staging)
		return err
	}
	_ = os.RemoveAll(staging)
	h.debug("source.materialize", "warning", warningSourcePrepared, "origin", origin)
	return nil
}

func (h *HostInstaller) checkArchive(path string) error {
	if h.verifyArchive != nil {
		return h.verifyArchive(path)
	}
	return verifyArchiveFile(path)
}

func (h *HostInstaller) probe(ctx context.Context, powershell, script, probe, home, installDir string, timeout time.Duration, parse func(string) error) error {
	invocation, err := probeInvocation(powershell, script, probe, home, installDir)
	if err != nil {
		return err
	}
	result := h.run(ctx, invocation, timeout)
	h.logCommand("installer.probe", probe, "", result)
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
	h.logCommand("installer.stage", stage, "", result)
	if result.limited {
		return installError(yorvaruntime.ErrorRuntimeInstallOutputLimit, errOutputLimit)
	}
	if result.timedOut {
		if !stageRequiresSuccess(stage) {
			return nil
		}
		return installError(yorvaruntime.ErrorRuntimeInstallTimeout, result.err)
	}
	if result.err != nil {
		if result.err == context.Canceled {
			return result.err
		}
		if !stageRequiresSuccess(stage) {
			return nil
		}
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, result.err)
	}
	parsed, err := parseStageResult(stage, result.stdout)
	if err != nil {
		return err
	}
	if !parsed.OK && !parsed.Skipped {
		if !stageRequiresSuccess(stage) {
			return nil
		}
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
	case "uv", "python", "git", "repository", "venv":
		return 10 * time.Minute
	case "node", "node-deps", "system-packages":
		return 45 * time.Second
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

func archiveLogFields(err error, sourceKind string) []any {
	code := installErrorCode(err)
	return []any{
		"errorCode", string(code),
		"category", archiveErrorCategory(code),
		"retryable", isTransportArchiveError(err),
		"timeout", code == yorvaruntime.ErrorRuntimeInstallTimeout,
		"cancelled", errors.Is(err, context.Canceled),
		"integrityMismatch", code == yorvaruntime.ErrorRuntimeInstallIntegrityFailed,
		"sourceKind", sourceKind,
	}
}

func archiveErrorCategory(code yorvaruntime.ErrorCode) string {
	switch code {
	case yorvaruntime.ErrorRuntimeInstallIntegrityFailed:
		return "integrity"
	case yorvaruntime.ErrorRuntimeInstallTimeout:
		return "timeout"
	case yorvaruntime.ErrorRuntimeInstallSourceUnavailable:
		return "transport"
	default:
		return "source"
	}
}

func (h *HostInstaller) debug(event string, args ...any) {
	if h == nil || h.logger == nil {
		return
	}
	attrs := append([]any{"event", event, "operationId", h.operationID}, args...)
	h.logger.Info("runtime install", attrs...)
}

func (h *HostInstaller) logCommand(event, stage, remote string, result commandResult) {
	args := []any{
		"stage", stage,
		"exitCode", result.exitCode,
		"timedOut", result.timedOut,
		"limited", result.limited,
	}
	if remote != "" {
		args = append(args, "remote", remote)
	}
	if result.timedOut {
		args = append(args, "errorCode", string(yorvaruntime.ErrorRuntimeInstallTimeout))
	} else if result.limited {
		args = append(args, "errorCode", string(yorvaruntime.ErrorRuntimeInstallOutputLimit))
	} else if result.err != nil || result.exitCode != 0 {
		args = append(args, "errorCode", string(yorvaruntime.ErrorRuntimeInstallStageFailed))
	}
	h.debug(event, args...)
}

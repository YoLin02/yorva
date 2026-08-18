package hermes

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

// BuildStaging runs official stages into a YORVA staging InstallDir.
// It does not promote hermes-agent, write PATH, or write an ownership marker.
func (h *HostInstaller) BuildStaging(ctx context.Context, operationID, stagingDir, hermesHome string) error {
	if !h.PlatformSupported() {
		return installError(yorvaruntime.ErrorRuntimeInstallPlatformUnsupported, errPlatform)
	}
	if stagingDir == "" || hermesHome == "" {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, fmt.Errorf("staging and Hermes home are required"))
	}
	if err := rejectReparsePoint(stagingDir); err != nil {
		return err
	}
	if h.operationID == "" {
		h.operationID = operationID
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
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := h.probe(ctx, powershell, script.Path, "ProtocolVersion", hermesHome, stagingDir, 30*time.Second, parseProtocolOutput); err != nil {
		return err
	}
	if err := h.probe(ctx, powershell, script.Path, "Manifest", hermesHome, stagingDir, 30*time.Second, parseAndValidateManifest); err != nil {
		return err
	}
	if err := verifyRegularFile(script.Path, h.source.source.ExpectedSize, h.source.source.ExpectedSHA); err != nil {
		return err
	}
	for _, stage := range approvedInstallStages() {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := verifyRegularFile(script.Path, h.source.source.ExpectedSize, h.source.source.ExpectedSHA); err != nil {
			return err
		}
		switch stage {
		case "node", "node-deps", "path":
			h.debug("installer.stage.skipped", "stage", stage, "reason", "yorva-owned")
			continue
		case "repository":
			if err := h.materializeStagingRepository(ctx, archivePath, origin, workDir, stagingDir); err != nil {
				return err
			}
			continue
		case "config-templates":
			if err := h.runStage(ctx, powershell, script.Path, stage, hermesHome, stagingDir); err != nil {
				h.debug("installer.stage.warning", "stage", stage, "reason", "config-templates-nonblocking")
			}
			if h.afterStage != nil {
				h.afterStage(stage, stagingDir)
			}
			continue
		}
		if err := h.runStage(ctx, powershell, script.Path, stage, hermesHome, stagingDir); err != nil {
			return err
		}
		if h.afterStage != nil {
			h.afterStage(stage, stagingDir)
		}
	}
	if err := materializeRequiredPublicLaunchers(stagingDir); err != nil {
		return err
	}
	if h.afterStage != nil {
		h.afterStage("path", stagingDir)
	}
	return nil
}

func (h *HostInstaller) materializeStagingRepository(ctx context.Context, archivePath, origin, workDir, stagingDir string) error {
	if origin == sourceOriginBundled {
		h.debug("source.materialize", "warning", warningBundledUsed)
	}
	if err := requireExtractBudget(workDir, h.archive.diskFree); err != nil {
		return err
	}
	if err := requireExtractBudget(stagingDir, h.archive.diskFree); err != nil {
		return err
	}
	extractDir := filepath.Join(workDir, "staging")
	if err := extractOfficialArchive(ctx, archivePath, extractDir); err != nil {
		_ = os.RemoveAll(extractDir)
		return err
	}
	if h.acquireArchive == nil {
		if err := verifyOfficialExtractedIdentity(extractDir); err != nil {
			_ = os.RemoveAll(extractDir)
			return err
		}
	} else if err := verifyExtractedLayout(extractDir); err != nil {
		_ = os.RemoveAll(extractDir)
		return err
	}
	if err := copyOwnedTree(ctx, extractDir, stagingDir); err != nil {
		_ = os.RemoveAll(extractDir)
		return err
	}
	_ = os.RemoveAll(extractDir)
	h.debug("source.materialize", "warning", warningSourcePrepared, "origin", origin)
	return nil
}

func materializeRequiredPublicLaunchers(root string) error {
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0o700); err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	if err := rejectReparsePoint(bin); err != nil {
		return err
	}
	for _, name := range []string{"hermes.exe", "hermes-acp.exe"} {
		src := filepath.Join(root, "venv", "Scripts", name)
		if !isRegularFile(src) {
			return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, fmt.Errorf("missing launcher source %s", name))
		}
		dest := filepath.Join(bin, name)
		if err := copyRegularFile(src, dest); err != nil {
			return err
		}
		if !isRegularFile(dest) {
			return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, fmt.Errorf("launcher %s is not a regular file", name))
		}
	}
	return nil
}

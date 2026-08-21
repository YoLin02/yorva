package hermes

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

// BuildGeneration runs official stages directly in the inactive final generation
// directory. It does not activate the generation or write PATH.
func (h *HostInstaller) BuildGeneration(ctx context.Context, operationID, generationDir, hermesHome string) error {
	if !h.PlatformSupported() {
		return installError(yorvaruntime.ErrorRuntimeInstallPlatformUnsupported, errPlatform)
	}
	if generationDir == "" || hermesHome == "" {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, fmt.Errorf("generation and Hermes home are required"))
	}
	if err := rejectReparsePoint(generationDir); err != nil {
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
	if err := h.probe(ctx, powershell, script.Path, "ProtocolVersion", hermesHome, generationDir, 30*time.Second, parseProtocolOutput); err != nil {
		return err
	}
	if err := h.probe(ctx, powershell, script.Path, "Manifest", hermesHome, generationDir, 30*time.Second, parseAndValidateManifest); err != nil {
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
			if err := h.materializeGenerationRepository(ctx, archivePath, origin, workDir, generationDir); err != nil {
				return err
			}
			continue
		case "config-templates":
			if err := h.runStage(ctx, powershell, script.Path, stage, hermesHome, generationDir); err != nil {
				h.debug("installer.stage.warning", "stage", stage, "reason", "config-templates-nonblocking")
			}
			if h.afterStage != nil {
				h.afterStage(stage, generationDir)
			}
			continue
		}
		if err := h.runStage(ctx, powershell, script.Path, stage, hermesHome, generationDir); err != nil {
			return err
		}
		if h.afterStage != nil {
			h.afterStage(stage, generationDir)
		}
	}
	if err := materializeRequiredPublicLaunchers(generationDir); err != nil {
		return err
	}
	if h.afterStage != nil {
		h.afterStage("path", generationDir)
	}
	return nil
}

func (h *HostInstaller) materializeGenerationRepository(ctx context.Context, archivePath, origin, workDir, generationDir string) error {
	if origin == sourceOriginBundled {
		h.debug("source.materialize", "warning", warningBundledUsed)
	}
	if err := requireExtractBudget(workDir, h.archive.diskFree); err != nil {
		return err
	}
	if err := requireExtractBudget(generationDir, h.archive.diskFree); err != nil {
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
	if err := copyOwnedTree(ctx, extractDir, generationDir); err != nil {
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

package hermes

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes/downloadsources"
)

func (h *NodeHost) installDependencies(ctx context.Context, sources downloadsources.Config) error {
	installDir := h.installDir()
	if !isRegularFile(filepath.Join(installDir, "package-lock.json")) {
		return installError(yorvaruntime.ErrorHermesNodeDepsFailed, errors.New("official package-lock.json is missing"))
	}
	node := filepath.Join(h.nodeDir(), "node.exe")
	cli := managedNpmCLI(h.nodeDir())
	if strings.EqualFold(filepath.Ext(cli), ".ps1") {
		return installError(yorvaruntime.ErrorHermesNPMUnsupported, errors.New("refusing npm.ps1"))
	}
	stamp := filepath.Join(installDir, nodeDepsStampName)
	_ = os.Remove(stamp)
	args := []string{cli, "ci", "--workspaces=false", "--omit=dev", "--ignore-scripts", "--no-audit", "--no-fund", "--progress=false"}
	result := h.run(ctx, installInvocation{
		Executable: node,
		Args: args,
		Dir: installDir,
		Environment: installerEnvironment(h.home(), sources),
	}, nodeDepsTimeout)
	if result.timedOut {
		return installError(yorvaruntime.ErrorHermesNodeDepsTimeout, errors.New("npm ci timed out"))
	}
	if result.limited {
		return installError(yorvaruntime.ErrorRuntimeInstallOutputLimit, errOutputLimit)
	}
	if result.err != nil || result.exitCode != 0 {
		if errors.Is(result.err, context.Canceled) {
			return result.err
		}
		return installError(yorvaruntime.ErrorHermesNodeDepsFailed, errors.New("npm ci failed"))
	}
	digest, err := fileSHA256(filepath.Join(installDir, "package-lock.json"))
	if err != nil {
		return installError(yorvaruntime.ErrorHermesNodeDepsFailed, err)
	}
	if err := os.WriteFile(stamp, []byte(digest+"\n"), 0o600); err != nil {
		return installError(yorvaruntime.ErrorHermesNodeDepsFailed, err)
	}
	return nil
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

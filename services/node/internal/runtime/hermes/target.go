package hermes

import (
	"os"
	"path/filepath"
	"strings"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const yorvaPartialMarker = ".yorva-phase3-install"

func validateInstallTarget(home, installDir string, retry bool) error {
	canonicalHome, err := filepath.Abs(home)
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, err)
	}
	canonicalTarget, err := filepath.Abs(installDir)
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, err)
	}
	if err := rejectReparsePoint(canonicalTarget); err != nil && !os.IsNotExist(err) {
		infoErr := err
		if _, statErr := os.Lstat(canonicalTarget); statErr == nil || !os.IsNotExist(statErr) {
			if !os.IsNotExist(statErr) {
				return infoErr
			}
		}
	}
	relative, err := filepath.Rel(canonicalHome, canonicalTarget)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	info, err := os.Lstat(canonicalTarget)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, err)
	}
	if !info.IsDir() {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	if !retry {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	if hasOfficialRepositoryIdentity(canonicalTarget) || hasYorvaPartialMarker(canonicalTarget) {
		return nil
	}
	return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
}

func hasOfficialRepositoryIdentity(root string) bool {
	markers := []string{
		filepath.Join(root, "hermes"),
		filepath.Join(root, "pyproject.toml"),
		filepath.Join(root, "hermes_cli", "main.py"),
	}
	for _, marker := range markers {
		if isRegularFile(marker) {
			return true
		}
	}
	return false
}

func hasYorvaPartialMarker(root string) bool {
	return isRegularFile(filepath.Join(root, yorvaPartialMarker))
}

func writeYorvaPartialMarker(root, pin string) error {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, yorvaPartialMarker), []byte(pin+"\n"), 0o600)
}

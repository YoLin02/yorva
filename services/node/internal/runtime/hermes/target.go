package hermes

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const yorvaPartialMarker = ".yorva-phase3-install"

const maxPartialMarkerBytes = 128

// owned partial Hermes tree:
//   - real directory under the official Hermes home
//   - regular .yorva-phase3-install whose exact contents equal the expected pin
//   - no reparse/symlink at the target or marker
//
// Generic files (pyproject.toml, hermes_cli/main.py, hermes launcher) never prove ownership.
func validateInstallTarget(home, installDir string, retry bool, expectedPin string) error {
	canonicalHome, err := filepath.Abs(home)
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, err)
	}
	canonicalTarget, err := filepath.Abs(installDir)
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, err)
	}
	if err := rejectReparsePoint(canonicalTarget); err != nil && !os.IsNotExist(err) {
		if _, statErr := os.Lstat(canonicalTarget); statErr == nil || !os.IsNotExist(statErr) {
			if !os.IsNotExist(statErr) {
				return err
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
	if !info.IsDir() || isReparsePoint(info) || info.Mode()&os.ModeSymlink != 0 {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	if !retry {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	if err := ownedPartialIdentity(canonicalTarget, expectedPin); err != nil {
		return err
	}
	return nil
}

func ownedPartialIdentity(root, expectedPin string) error {
	if err := rejectReparsePoint(root); err != nil {
		return err
	}
	pin, err := readYorvaPartialMarker(root)
	if err != nil {
		return err
	}
	if expectedPin == "" || pin != expectedPin {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	return nil
}

func readYorvaPartialMarker(root string) (string, error) {
	path := filepath.Join(root, yorvaPartialMarker)
	if err := rejectReparsePoint(path); err != nil {
		return "", err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, err)
	}
	if !info.Mode().IsRegular() || isReparsePoint(info) {
		return "", installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	if info.Size() == 0 || info.Size() > maxPartialMarkerBytes {
		return "", installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		return "", installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, err)
	}
	pin := string(bytes.TrimSpace(payload))
	if pin == "" || strings.ContainsAny(pin, " \t\r\n") || !isHexCommit(pin) {
		return "", installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	return pin, nil
}

func isHexCommit(value string) bool {
	if len(value) != 40 {
		return false
	}
	for _, character := range value {
		if character >= '0' && character <= '9' || character >= 'a' && character <= 'f' || character >= 'A' && character <= 'F' {
			continue
		}
		return false
	}
	return true
}

func writeYorvaPartialMarker(root, pin string) error {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	if err := rejectReparsePoint(root); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, yorvaPartialMarker), []byte(pin+"\n"), 0o600)
}

func replaceOwnedTree(ctx context.Context, staging, installDir, pin string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := rejectReparsePoint(staging); err != nil {
		return err
	}
	parent, err := filepath.Abs(filepath.Dir(installDir))
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	if err := rejectReparsePoint(parent); err != nil {
		return err
	}
	tmp, err := uniqueSibling(parent, "hermes-agent.yorva-new-")
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	if err := os.Mkdir(tmp, 0o700); err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	replaced := false
	defer func() {
		if !replaced {
			_ = os.RemoveAll(tmp)
		}
	}()
	if err := copyOwnedTree(ctx, staging, tmp); err != nil {
		return err
	}
	if err := writeYorvaPartialMarker(tmp, pin); err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	if _, err := readYorvaPartialMarker(tmp); err != nil {
		return err
	}

	info, err := os.Lstat(installDir)
	if os.IsNotExist(err) {
		if err := os.Rename(tmp, installDir); err != nil {
			return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
		}
		replaced = true
		return nil
	}
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	if !info.IsDir() || isReparsePoint(info) || info.Mode()&os.ModeSymlink != 0 {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	if err := ownedPartialIdentity(installDir, pin); err != nil {
		return err
	}
	backup, err := uniqueSibling(parent, "hermes-agent.yorva-old-")
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	if err := os.Rename(installDir, backup); err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	if err := os.Rename(tmp, installDir); err != nil {
		_ = os.Rename(backup, installDir)
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	replaced = true
	_ = os.RemoveAll(backup)
	return nil
}

func copyOwnedTree(ctx context.Context, staging, dest string) error {
	return filepath.WalkDir(staging, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		relative, relErr := filepath.Rel(staging, path)
		if relErr != nil || !pathWithin(staging, path) {
			return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errors.New("materialized tree escaped staging"))
		}
		target := filepath.Join(dest, relative)
		if !pathWithin(dest, target) {
			return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errors.New("materialized tree escaped install directory"))
		}
		if entry.IsDir() {
			if err := os.MkdirAll(target, 0o700); err != nil {
				return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
			}
			return rejectReparsePoint(target)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
		}
		return copyRegularFile(path, target)
	})
}

func uniqueSibling(parent, prefix string) (string, error) {
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return filepath.Join(parent, prefix+hex.EncodeToString(random)), nil
}

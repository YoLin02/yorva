package hermes

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

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

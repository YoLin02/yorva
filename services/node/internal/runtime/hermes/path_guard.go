package hermes

import (
	"os"
	"path/filepath"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func rejectReparsePoint(path string) error {
	current := path
	for i := 0; i < 8; i++ {
		info, err := os.Lstat(current)
		if err != nil {
			if os.IsNotExist(err) {
				parent := filepath.Dir(current)
				if parent == current {
					return nil
				}
				current = parent
				continue
			}
			return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || isReparsePoint(info) {
			return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errReparsePoint)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
		current = parent
	}
	return nil
}

//go:build !windows

package backupmanagement

import (
	"io/fs"
	"os"

	"golang.org/x/sys/unix"
)

func unsafeSnapshotInfo(info fs.FileInfo) bool {
	return info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() && !info.IsDir()
}

func openSnapshotSourceFile(path string) (*os.File, error) {
	descriptor, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(descriptor), path), nil
}

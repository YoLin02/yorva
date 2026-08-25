//go:build !windows

package backupmanagement

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

func validateLocalDestinationPath(path string) error {
	current := filepath.Dir(path)
	for {
		info, err := os.Lstat(current)
		if err != nil {
			return verificationError(ErrorDestinationUnsafe)
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return verificationError(ErrorDestinationUnsafe)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
		current = parent
	}
}

func destinationFreeBytes(path string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}
	return stat.Bavail * uint64(stat.Bsize), nil
}

func createPrivateStaging(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, err
	}
	info, statErr := file.Stat()
	if statErr != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
		_ = file.Close()
		_ = os.Remove(path)
		return nil, errors.New("private staging permissions unavailable")
	}
	return file, nil
}

func openRegularNoFollow(path string) (*os.File, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		_ = unix.Close(fd)
		return nil, errors.New("backup file handle unavailable")
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		_ = file.Close()
		return nil, errors.New("backup file is not regular")
	}
	return file, nil
}

func openDirectoryNoFollow(path string) (*os.File, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_DIRECTORY, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		_ = unix.Close(fd)
		return nil, errors.New("backup directory handle unavailable")
	}
	return file, nil
}

func publishNoReplace(staging, destination string) error {
	if err := os.Link(staging, destination); err != nil {
		return err
	}
	if err := os.Remove(staging); err != nil {
		return err
	}
	return nil
}

func syncDestinationDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

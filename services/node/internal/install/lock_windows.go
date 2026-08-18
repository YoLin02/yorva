//go:build windows

package install

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

const lockAllBytes = ^uint32(0)

func lockExclusive(file *os.File) error {
	var overlapped windows.Overlapped
	err := windows.LockFileEx(
		windows.Handle(file.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0,
		lockAllBytes,
		lockAllBytes,
		&overlapped,
	)
	if err == nil {
		return nil
	}
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) || errors.Is(err, windows.ERROR_IO_PENDING) {
		return ErrLockBusy
	}
	return err
}

func unlockExclusive(file *os.File) error {
	var overlapped windows.Overlapped
	return windows.UnlockFileEx(windows.Handle(file.Fd()), 0, lockAllBytes, lockAllBytes, &overlapped)
}

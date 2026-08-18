//go:build !windows

package install

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func lockExclusive(file *os.File) error {
	err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err == nil {
		return nil
	}
	if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
		return ErrLockBusy
	}
	return err
}

func unlockExclusive(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_UN)
}

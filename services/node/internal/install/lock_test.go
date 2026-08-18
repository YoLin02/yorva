package install

import (
	"errors"
	"os"
	"testing"
)

func TestInstallLockIsOSHeldNotExistence(t *testing.T) {
	root := t.TempDir()
	first, err := AcquireLock(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(first.Path()); err != nil {
		t.Fatal(err)
	}
	_, err = AcquireLock(root)
	if !errors.Is(err, ErrLockBusy) {
		t.Fatalf("second acquire: %v", err)
	}
	if err := first.Release(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(first.Path()); err != nil {
		t.Fatal("lock file should remain after release")
	}
	second, err := AcquireLock(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := second.Release(); err != nil {
		t.Fatal(err)
	}
}

func TestInstallLockReleaseThenReacquire(t *testing.T) {
	root := t.TempDir()
	lock, err := AcquireLock(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
	again, err := AcquireLock(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := again.Release(); err != nil {
		t.Fatal(err)
	}
}

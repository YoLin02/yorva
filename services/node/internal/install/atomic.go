package install

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// atomicStep is one inject point from architecture §13.1.
type atomicStep int

const (
	stepBeforeTempCreate atomicStep = iota
	stepAfterTempCreate
	stepAfterWrite
	stepAfterFileSync
	stepAfterClose
	stepAfterPreReplaceDirSync
	stepAfterReplace
	stepAfterFinalDirSync
	stepAfterReadback
)

type atomicOps struct {
	Hook            func(atomicStep) error
	CreateExclusive func(path string) (*os.File, error)
	Write           func(*os.File, []byte) error
	Sync            func(*os.File) error
	Close           func(*os.File) error
	Replace         func(tmp, dest string) error
	SyncDir         func(dir string) error
	ReadFile        func(path string) ([]byte, error)
}

func defaultAtomicOps() atomicOps {
	return atomicOps{
		CreateExclusive: func(path string) (*os.File, error) {
			return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		},
		Write: func(file *os.File, payload []byte) error {
			_, err := file.Write(payload)
			return err
		},
		Sync:     func(file *os.File) error { return file.Sync() },
		Close:    func(file *os.File) error { return file.Close() },
		Replace:  replaceFileAtomic,
		SyncDir:  syncDirectory,
		ReadFile: os.ReadFile,
	}
}

func (ops atomicOps) hook(step atomicStep) error {
	if ops.Hook == nil {
		return nil
	}
	return ops.Hook(step)
}

func writeAtomicRecord(ops atomicOps, dest string, payload []byte) (err error) {
	if ops.CreateExclusive == nil {
		ops = defaultAtomicOps()
	}
	dir := filepath.Dir(dest)
	if err := rejectReparse(dir); err != nil {
		return err
	}
	if err := ops.hook(stepBeforeTempCreate); err != nil {
		return err
	}
	tmp, err := uniqueSibling(dir, ".yorva-atom-")
	if err != nil {
		return err
	}
	file, err := ops.CreateExclusive(tmp)
	if err != nil {
		return err
	}
	closed := false
	replaced := false
	defer func() {
		if !closed {
			_ = file.Close()
		}
		if err != nil && !replaced {
			_ = os.Remove(tmp)
		}
	}()
	if err = ops.hook(stepAfterTempCreate); err != nil {
		return err
	}
	if err = ops.Write(file, payload); err != nil {
		return err
	}
	if err = ops.hook(stepAfterWrite); err != nil {
		return err
	}
	if err = ops.Sync(file); err != nil {
		return err
	}
	if err = ops.hook(stepAfterFileSync); err != nil {
		return err
	}
	if err = ops.Close(file); err != nil {
		return err
	}
	closed = true
	if err = ops.hook(stepAfterClose); err != nil {
		return err
	}
	info, statErr := os.Lstat(tmp)
	if statErr != nil || !info.Mode().IsRegular() || isReparsePoint(info) {
		return fmt.Errorf("install: atomic temp is not a regular file")
	}
	if err = rejectReparse(tmp); err != nil {
		return err
	}
	if ops.SyncDir != nil {
		if err = ops.SyncDir(dir); err != nil {
			return err
		}
	}
	if err = ops.hook(stepAfterPreReplaceDirSync); err != nil {
		return err
	}
	if err = ops.Replace(tmp, dest); err != nil {
		return err
	}
	replaced = true
	if err = ops.hook(stepAfterReplace); err != nil {
		return err
	}
	var syncErr error
	if ops.SyncDir != nil {
		syncErr = ops.SyncDir(dir)
	}
	if err = ops.hook(stepAfterFinalDirSync); err != nil {
		return err
	}
	got, readErr := ops.ReadFile(dest)
	if readErr != nil {
		if syncErr != nil {
			return syncErr
		}
		return readErr
	}
	if !bytes.Equal(got, payload) {
		return ErrAtomicReadback
	}
	// Architecture §15.5: a complete readable new record is the recovery
	// truth even if the final directory-sync call failed.
	_ = syncErr
	return ops.hook(stepAfterReadback)
}

func uniqueSibling(dir, prefix string) (string, error) {
	for i := 0; i < 8; i++ {
		var raw [8]byte
		if _, err := rand.Read(raw[:]); err != nil {
			return "", err
		}
		path := filepath.Join(dir, prefix+hex.EncodeToString(raw[:]))
		if _, err := os.Lstat(path); os.IsNotExist(err) {
			return path, nil
		}
	}
	return "", fmt.Errorf("install: could not allocate atomic temp name")
}

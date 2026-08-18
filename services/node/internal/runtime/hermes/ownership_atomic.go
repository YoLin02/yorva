package hermes

import (
	"errors"
	"os"
	"path/filepath"
)

type atomicFileOps struct {
	CreateExclusive func(path string) (*os.File, error)
	Write           func(*os.File, []byte) error
	Sync            func(*os.File) error
	Close           func(*os.File) error
	Replace         func(tmp, dest string) error
	SyncDir         func(dir string) error
}

func defaultAtomicFileOps() atomicFileOps {
	return atomicFileOps{
		CreateExclusive: func(path string) (*os.File, error) {
			return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		},
		Write: func(file *os.File, payload []byte) error {
			_, err := file.Write(payload)
			return err
		},
		Sync: func(file *os.File) error {
			return file.Sync()
		},
		Close: func(file *os.File) error {
			return file.Close()
		},
		Replace: replaceFileAtomic,
		SyncDir: syncDirectory,
	}
}

func writeAtomicRegularFile(ops atomicFileOps, dest string, payload []byte) (err error) {
	if ops.CreateExclusive == nil {
		ops = defaultAtomicFileOps()
	}
	if err := rejectReparsePoint(filepath.Dir(dest)); err != nil {
		return err
	}
	tmp, err := uniqueSibling(filepath.Dir(dest), ".yorva-atom-")
	if err != nil {
		return err
	}
	file, err := ops.CreateExclusive(tmp)
	if err != nil {
		return err
	}
	closed := false
	defer func() {
		if !closed {
			_ = file.Close()
		}
		if err != nil {
			_ = os.Remove(tmp)
		}
	}()
	if err = ops.Write(file, payload); err != nil {
		return err
	}
	if err = ops.Sync(file); err != nil {
		return err
	}
	if err = ops.Close(file); err != nil {
		return err
	}
	closed = true
	info, statErr := os.Lstat(tmp)
	if statErr != nil || !info.Mode().IsRegular() || isReparsePoint(info) {
		return errors.New("atomic temp is not a regular file")
	}
	if err = rejectReparsePoint(tmp); err != nil {
		return err
	}
	dir := filepath.Dir(dest)
	if ops.SyncDir != nil {
		if err = ops.SyncDir(dir); err != nil {
			return err
		}
	}
	if err = ops.Replace(tmp, dest); err != nil {
		return err
	}
	if ops.SyncDir != nil {
		if err = ops.SyncDir(dir); err != nil {
			return err
		}
	}
	return nil
}

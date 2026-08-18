package install

import "os"

// InstallLock is an OS-held exclusive lock on control/install.lock.
// File existence is not the lock.
type InstallLock struct {
	file *os.File
	path string
}

func AcquireLock(root string) (*InstallLock, error) {
	layout, err := NewLayout(root)
	if err != nil {
		return nil, err
	}
	if err := layout.EnsureControl(); err != nil {
		return nil, err
	}
	path := layout.LockPath()
	if err := rejectReparse(path); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	if err := lockExclusive(file); err != nil {
		_ = file.Close()
		return nil, err
	}
	return &InstallLock{file: file, path: path}, nil
}

func (l *InstallLock) Path() string {
	if l == nil {
		return ""
	}
	return l.path
}

func (l *InstallLock) Release() error {
	if l == nil || l.file == nil {
		return nil
	}
	errUnlock := unlockExclusive(l.file)
	errClose := l.file.Close()
	l.file = nil
	if errUnlock != nil {
		return errUnlock
	}
	return errClose
}

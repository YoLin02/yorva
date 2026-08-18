package install

import "errors"

var (
	ErrInvalidID           = errors.New("install: invalid closed identifier")
	ErrInvalidRelativePath = errors.New("install: invalid relative path")
	ErrPathNotContained    = errors.New("install: path is not contained in the managed root")
	ErrRevisionConflict    = errors.New("install: transaction revision conflict")
	ErrInvalidRecord       = errors.New("install: invalid record")
	ErrNotFound            = errors.New("install: not found")
	ErrLockBusy            = errors.New("install: install.lock is held")
	ErrAtomicReadback      = errors.New("install: atomic read-back mismatch")
	ErrReparsePoint        = errors.New("install: path contains a symlink or reparse point")
	ErrManagedRootUnset    = errors.New("install: LOCALAPPDATA is not set")
	ErrSealInvalid         = errors.New("install: seal invalid")
	ErrStagingOccupied     = errors.New("install: staging directory already exists")
	ErrPublishConflict     = errors.New("install: publish conflict")
	ErrBlockedUnsafe       = errors.New("install: blocked unsafe")
)

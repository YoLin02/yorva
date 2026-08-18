package install

import (
	"fmt"
	"os"
	"path/filepath"
)

func (m *Manager) publish(txn *InstallTransaction) error {
	if txn.State == StatePublished || txn.State == StateActivating {
		return nil
	}
	if txn.State != StateSealed {
		return ErrInvalidRecord
	}
	stagingAbs, err := m.store.layout.StagingPath(txn.ID)
	if err != nil {
		return err
	}
	destAbs, err := m.store.layout.GenerationPath(txn.GenerationID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(m.store.layout.GenerationsRoot(), 0o700); err != nil {
		return err
	}
	if err := rejectReparse(m.store.layout.GenerationsRoot()); err != nil {
		return err
	}

	stagingPresent, stagingEmpty, stagingSealed := observePublishedTree(stagingAbs, *txn)
	destPresent, _, destSealed := observePublishedTree(destAbs, *txn)

	switch {
	case destPresent && destSealed && stagingPresent && !stagingEmpty && stagingSealed:
		return fmt.Errorf("%w: duplicate sealed trees", ErrBlockedUnsafe)
	case destPresent && destSealed:
		if stagingPresent && stagingEmpty {
			if err := os.Remove(stagingAbs); err != nil && !os.IsNotExist(err) {
				return err
			}
		} else if stagingPresent && !stagingEmpty && !stagingSealed {
			return fmt.Errorf("%w: leftover staging is not empty leftover", ErrPublishConflict)
		}
	case destPresent && !destSealed:
		return fmt.Errorf("%w: generation destination exists with different bytes", ErrPublishConflict)
	default:
		if !stagingSealed {
			return ErrSealInvalid
		}
		if err := m.failpoint(FailBeforePublishRename); err != nil {
			return err
		}
		if err := renameDirectory(stagingAbs, destAbs); err != nil {
			return err
		}
		if err := m.failpoint(FailAfterPublishRename); err != nil {
			return err
		}
	}
	if err := VerifySealedTree(destAbs, txn.GenerationID, txn.ManifestSHA256, txn.SealSHA256); err != nil {
		return err
	}
	now := m.now()
	txn.State = StatePublished
	txn.Step = "published"
	txn.PublishedAt = &now
	txn.UpdatedAt = now
	if err := m.persist(txn); err != nil {
		return err
	}
	return m.failpoint(FailAfterPublished)
}

func observePublishedTree(abs string, txn InstallTransaction) (present, empty, sealed bool) {
	info, err := os.Lstat(abs)
	if err != nil {
		return false, false, false
	}
	if err := rejectReparse(abs); err != nil || !info.IsDir() {
		return true, false, false
	}
	present = true
	empty, _ = dirEmpty(abs)
	sealed = VerifySealedTree(abs, txn.GenerationID, txn.ManifestSHA256, txn.SealSHA256) == nil
	return present, empty, sealed
}

func renameDirectory(from, to string) error {
	if err := rejectReparse(from); err != nil {
		return err
	}
	if err := rejectReparse(filepath.Dir(to)); err != nil {
		return err
	}
	if _, err := os.Lstat(to); err == nil {
		return ErrPublishConflict
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := renameDirAtomic(from, to); err != nil {
		return err
	}
	if err := rejectReparse(to); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(to))
}

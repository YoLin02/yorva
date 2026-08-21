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
	stagingAbs, err := m.store.layout.StagingPath(txn.ID)
	if err != nil {
		return err
	}
	if stagingPresent, stagingEmpty, _ := observePublishedTree(stagingAbs, *txn); stagingPresent {
		if !stagingEmpty {
			return fmt.Errorf("%w: legacy staging tree cannot be published", ErrBlockedUnsafe)
		}
		if err := os.Remove(stagingAbs); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	destPresent, _, destSealed := observePublishedTree(destAbs, *txn)
	if !destPresent {
		return ErrSealInvalid
	}
	if !destSealed {
		return fmt.Errorf("%w: generation destination exists with different bytes", ErrPublishConflict)
	}
	if err := m.failpoint(FailBeforePublishVerify); err != nil {
		return err
	}
	if err := VerifySealedTree(destAbs, txn.GenerationID, txn.ManifestSHA256, txn.SealSHA256); err != nil {
		return err
	}
	if err := m.failpoint(FailAfterPublishVerify); err != nil {
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

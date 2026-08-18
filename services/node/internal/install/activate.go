package install

import (
	"fmt"
	"time"
)

func (m *Manager) activate(txn *InstallTransaction) error {
	if txn.State != StatePublished && txn.State != StateActivating {
		return ErrInvalidRecord
	}
	destAbs, err := m.store.layout.GenerationPath(txn.GenerationID)
	if err != nil {
		return err
	}
	if err := VerifyPublishedGeneration(destAbs, txn.GenerationID, txn.ManifestSHA256, txn.SealSHA256); err != nil {
		return err
	}

	pointer := m.store.ReadActive()
	current, currentErr := m.store.LoadActive()
	hasCurrent := currentErr == nil

	if pointer.Valid && pointer.GenerationID == txn.GenerationID {
		if txn.State != StateActivating {
			now := m.now()
			txn.State = StateActivating
			txn.Step = "activating"
			txn.UpdatedAt = now
			if txn.ActivatedAt == nil {
				txn.ActivatedAt = &now
			}
			if err := m.persist(txn); err != nil {
				return err
			}
		}
		return nil
	}
	if pointer.Valid && txn.ActiveBeforeGeneration != "" && pointer.GenerationID != txn.ActiveBeforeGeneration {
		return fmt.Errorf("%w: %s", ErrBlockedUnsafe, CodeBlockedUnsafe)
	}

	if txn.State == StatePublished {
		if err := m.failpoint(FailBeforeActivatingPersist); err != nil {
			return err
		}
		if hasCurrent {
			txn.ActiveBeforeGeneration = current.GenerationID
			txn.ActiveBeforeDigest = current.SealSHA256
		}
		now := m.now()
		txn.State = StateActivating
		txn.Step = "activating"
		txn.UpdatedAt = now
		if err := m.persist(txn); err != nil {
			return err
		}
		if err := m.failpoint(FailAfterActivatingPersist); err != nil {
			return err
		}
	}

	pointer = m.store.ReadActive()
	if pointer.Valid && pointer.GenerationID != txn.GenerationID && pointer.GenerationID != txn.ActiveBeforeGeneration {
		return fmt.Errorf("%w: %s", ErrBlockedUnsafe, CodeBlockedUnsafe)
	}

	if err := m.failpoint(FailDuringActiveWrite); err != nil {
		return err
	}
	now := m.now()
	rec := ActiveRecord{
		Schema:                 activeSchema,
		RuntimeKind:            txn.RuntimeKind,
		GenerationID:           txn.GenerationID,
		GenerationRelativePath: txn.GenerationRelativePath,
		ManifestSHA256:         txn.ManifestSHA256,
		SealSHA256:             txn.SealSHA256,
		SourcePin:              txn.SourcePin,
		Version:                txn.ExpectedVersion,
		TransactionID:          txn.ID,
		ActivatedAt:            now.UTC(),
	}
	if rec.ActivatedAt.IsZero() {
		rec.ActivatedAt = time.Now().UTC()
	}
	if err := m.store.WriteActive(rec); err != nil {
		return err
	}
	if err := m.failpoint(FailAfterActiveWrite); err != nil {
		return err
	}
	got, err := m.store.LoadActive()
	if err != nil || got.GenerationID != txn.GenerationID {
		return ErrInvalidRecord
	}
	if err := VerifyPublishedGeneration(destAbs, txn.GenerationID, txn.ManifestSHA256, txn.SealSHA256); err != nil {
		return err
	}
	if txn.ActivatedAt == nil {
		txn.ActivatedAt = &now
		txn.UpdatedAt = now
		if err := m.persist(txn); err != nil {
			return err
		}
	}
	return nil
}

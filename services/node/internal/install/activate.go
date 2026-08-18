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
	if err := VerifySealedTree(destAbs, txn.GenerationID, txn.ManifestSHA256, txn.SealSHA256); err != nil {
		return err
	}

	pointer := m.store.ReadActive()
	if pointer.Invalid() {
		return fmt.Errorf("%w: %s", ErrBlockedUnsafe, CodeBlockedUnsafe)
	}
	if pointerNamesTxn(pointer, *txn) {
		return m.markActivating(txn)
	}

	if txn.State == StatePublished {
		if err := m.failpoint(FailBeforeActivatingPersist); err != nil {
			return err
		}
		if err := snapshotActiveBefore(txn, pointer); err != nil {
			return err
		}
		if err := m.markActivating(txn); err != nil {
			return err
		}
		if err := m.failpoint(FailAfterActivatingPersist); err != nil {
			return err
		}
	}

	pointer = m.store.ReadActive()
	if pointer.Invalid() {
		return fmt.Errorf("%w: %s", ErrBlockedUnsafe, CodeBlockedUnsafe)
	}
	if pointerNamesTxn(pointer, *txn) {
		return nil
	}
	if err := casAllowsActivate(*txn, pointer); err != nil {
		return err
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
	got := m.store.ReadActive()
	if !pointerNamesTxn(got, *txn) {
		return ErrInvalidRecord
	}
	if err := VerifySealedTree(destAbs, txn.GenerationID, txn.ManifestSHA256, txn.SealSHA256); err != nil {
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

func (m *Manager) markActivating(txn *InstallTransaction) error {
	if txn.State == StateActivating && txn.ActivatedAt != nil {
		return nil
	}
	now := m.now()
	txn.State = StateActivating
	txn.Step = "activating"
	txn.UpdatedAt = now
	if txn.ActivatedAt == nil {
		txn.ActivatedAt = &now
	}
	return m.persist(txn)
}

func snapshotActiveBefore(txn *InstallTransaction, pointer ActivePointer) error {
	if pointer.Invalid() {
		return fmt.Errorf("%w: %s", ErrBlockedUnsafe, CodeBlockedUnsafe)
	}
	if pointer.Missing() {
		txn.ActiveBeforeKind = ActiveBeforeAbsent
		txn.ActiveBeforeGeneration = ""
		txn.ActiveBeforeDigest = ""
		return nil
	}
	if !pointer.IsValid() {
		return fmt.Errorf("%w: %s", ErrBlockedUnsafe, CodeBlockedUnsafe)
	}
	txn.ActiveBeforeKind = ActiveBeforeValid
	txn.ActiveBeforeGeneration = pointer.GenerationID
	txn.ActiveBeforeDigest = pointer.SealSHA256
	return nil
}

func pointerNamesTxn(pointer ActivePointer, txn InstallTransaction) bool {
	return pointer.IsValid() &&
		pointer.GenerationID == txn.GenerationID &&
		pointer.SealSHA256 == txn.SealSHA256
}

func casAllowsActivate(txn InstallTransaction, pointer ActivePointer) error {
	if pointer.Invalid() {
		return fmt.Errorf("%w: %s", ErrBlockedUnsafe, CodeBlockedUnsafe)
	}
	if pointerNamesTxn(pointer, txn) {
		return nil
	}
	kind := txn.ActiveBeforeKind
	if kind == "" && txn.ActiveBeforeGeneration == "" {
		kind = ActiveBeforeAbsent
	}
	if kind == "" && txn.ActiveBeforeGeneration != "" {
		kind = ActiveBeforeValid
	}
	switch kind {
	case ActiveBeforeAbsent:
		if !pointer.Missing() {
			return fmt.Errorf("%w: %s", ErrBlockedUnsafe, CodeBlockedUnsafe)
		}
		return nil
	case ActiveBeforeValid:
		if !pointer.IsValid() || pointer.GenerationID != txn.ActiveBeforeGeneration {
			return fmt.Errorf("%w: %s", ErrBlockedUnsafe, CodeBlockedUnsafe)
		}
		if txn.ActiveBeforeDigest != "" && pointer.SealSHA256 != txn.ActiveBeforeDigest {
			return fmt.Errorf("%w: %s", ErrBlockedUnsafe, CodeBlockedUnsafe)
		}
		return nil
	default:
		return fmt.Errorf("%w: %s", ErrBlockedUnsafe, CodeBlockedUnsafe)
	}
}

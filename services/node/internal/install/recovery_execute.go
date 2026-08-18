package install

import (
	"context"
	"os"
	"time"
)

func Execute(ctx context.Context, store *Store, mgr *Manager, obs Observation, d RecoveryDecision) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	switch d.Action {
	case ActionNone:
		return persistPrimaryState(store, obs, d)
	case ActionRemoveEmptyStaging:
		if err := removeEmptyStaging(store, obs); err != nil {
			return err
		}
		return persistPrimaryState(store, obs, d)
	case ActionMoveStagingToFailed:
		if err := moveStagingToFailed(store, obs); err != nil {
			return err
		}
		return persistPrimaryState(store, obs, d)
	case ActionGCStaging:
		return removeProvenStaging(store, obs)
	case ActionPublish:
		return invokeOnPrimary(store, obs, func(txn InstallTransaction) error {
			return mgr.publish(&txn)
		})
	case ActionPersistActivating:
		return persistActivating(store, obs)
	case ActionActivate:
		return invokeOnPrimary(store, obs, func(txn InstallTransaction) error {
			return mgr.activate(&txn)
		})
	case ActionReconcileEnv:
		return invokeOnPrimary(store, obs, func(txn InstallTransaction) error {
			_, err := mgr.ReconcileEnvironment(ctx, txn)
			return err
		})
	case ActionCommit:
		d.NextState = StateCommitted
		return persistPrimaryState(store, obs, d)
	case ActionDiagnoseRetain, ActionBlockUnsafe, ActionFailTransaction:
		return nil
	default:
		return nil
	}
}

func persistPrimaryState(store *Store, obs Observation, d RecoveryDecision) error {
	if d.NextState == "" {
		return nil
	}
	primary := primaryTxn(obs.Transactions, obs.Active)
	if !primary.Valid {
		return nil
	}
	return persistTxnState(store, primary.ID, d.NextState, d.ErrorCode)
}

func persistTxnState(store *Store, id string, state TransactionState, code string) error {
	txn, err := store.LoadTransaction(id)
	if err != nil {
		return err
	}
	if txn.State == state && txn.ErrorCode == code {
		return nil
	}
	now := time.Now().UTC()
	txn.State = state
	txn.ErrorCode = code
	txn.UpdatedAt = now
	switch state {
	case StatePublished:
		if txn.PublishedAt == nil {
			txn.PublishedAt = &now
		}
	case StateActivating:
		if txn.ActivatedAt == nil {
			txn.ActivatedAt = &now
		}
	case StateCommitted:
		if txn.CommittedAt == nil {
			txn.CommittedAt = &now
		}
	}
	return store.SaveTransaction(txn)
}

func persistActivating(store *Store, obs Observation) error {
	primary := primaryTxn(obs.Transactions, obs.Active)
	if !primary.Valid {
		return ErrInvalidRecord
	}
	txn, err := store.LoadTransaction(primary.ID)
	if err != nil {
		return err
	}
	if rec, err := store.LoadActive(); err == nil && rec.GenerationID != txn.GenerationID {
		txn.ActiveBeforeGeneration = rec.GenerationID
		txn.ActiveBeforeDigest = rec.SealSHA256
	}
	now := time.Now().UTC()
	txn.State = StateActivating
	txn.Step = "activating"
	txn.UpdatedAt = now
	return store.SaveTransaction(txn)
}

func invokeOnPrimary(store *Store, obs Observation, fn func(InstallTransaction) error) error {
	primary := primaryTxn(obs.Transactions, obs.Active)
	if !primary.Valid {
		return ErrInvalidRecord
	}
	txn, err := store.LoadTransaction(primary.ID)
	if err != nil {
		return err
	}
	return fn(txn)
}

func removeEmptyStaging(store *Store, obs Observation) error {
	primary := primaryTxn(obs.Transactions, obs.Active)
	if !primary.Valid {
		return nil
	}
	path, err := store.layout.StagingPath(primary.ID)
	if err != nil {
		return err
	}
	empty, err := dirEmpty(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !empty {
		return ErrStagingOccupied
	}
	return os.Remove(path)
}

func moveStagingToFailed(store *Store, obs Observation) error {
	primary := primaryTxn(obs.Transactions, obs.Active)
	if !primary.Valid {
		return nil
	}
	src, err := store.layout.StagingPath(primary.ID)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(src); os.IsNotExist(err) {
		return nil
	}
	if err := os.MkdirAll(store.layout.FailedRoot(), 0o700); err != nil {
		return err
	}
	if err := rejectReparse(store.layout.FailedRoot()); err != nil {
		return err
	}
	dest, err := store.layout.FailedPath(primary.ID)
	if err != nil {
		return err
	}
	return renameDirectory(src, dest)
}

func removeProvenStaging(store *Store, obs Observation) error {
	primary := primaryTxn(obs.Transactions, obs.Active)
	if !primary.Valid || !obs.Staging.LineageMatch || !obs.Generation.Sealed {
		return nil
	}
	path, err := store.layout.StagingPath(primary.ID)
	if err != nil {
		return err
	}
	return os.RemoveAll(path)
}

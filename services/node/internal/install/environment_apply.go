package install

import (
	"context"
	"errors"
	"fmt"
)

const (
	FailEnvRead                 = "env-read"
	FailAfterHermesHome         = "after-hermes-home"
	FailDuringPath              = "during-path"
	FailAfterPathBeforeReadback = "after-path-before-readback"
	FailBroadcast               = "env-broadcast"
)

func defaultEnvironmentStore() EnvironmentStore {
	return EnvironmentStore{
		Read:      readUserEnvironment,
		WriteHome: writeUserHermesHome,
		WritePath: writeUserPath,
		Broadcast: broadcastEnvironmentChange,
	}
}

func (m *Manager) envStore() EnvironmentStore {
	if m.env.Read != nil {
		return m.env
	}
	return defaultEnvironmentStore()
}

// ReconcileEnvironment applies HERMES_HOME then PATH from valid active.json.
// Failure leaves ACTIVATING and never rolls back the pointer. COMMITTED only after values are observed.
func (m *Manager) ReconcileEnvironment(ctx context.Context, txn InstallTransaction) (InstallTransaction, error) {
	if err := ctx.Err(); err != nil {
		return txn, err
	}
	loaded, err := m.store.LoadTransaction(txn.ID)
	if err != nil {
		return txn, err
	}
	if loaded.State != StateActivating && loaded.State != StateCommitted {
		return loaded, ErrInvalidRecord
	}
	rec, err := m.store.LoadActive()
	if err != nil {
		return loaded, err
	}
	if rec.GenerationID != loaded.GenerationID && loaded.State == StateActivating {
		return loaded, fmt.Errorf("%w: active generation does not match transaction", ErrInvalidRecord)
	}
	if err := m.reconcileOnce(rec, loaded.State == StateCommitted || m.store.HasCommitted()); err != nil {
		return loaded, err
	}
	if loaded.State == StateActivating {
		obs, err := m.readObserved()
		if err != nil {
			return loaded, err
		}
		policy, err := PolicyFromActive(m.store, rec, false)
		if err != nil {
			return loaded, err
		}
		if !ComputeEnvironmentPlan(policy, obs).Observation(obs).Complete() {
			return loaded, errors.New("install: environment not yet observed")
		}
		now := m.now()
		loaded.State = StateCommitted
		loaded.Step = "committed"
		loaded.CommittedAt = &now
		loaded.UpdatedAt = now
		if err := m.persist(&loaded); err != nil {
			return loaded, err
		}
		_, _ = Collect(m.store, GCHooks{})
	}
	_ = m.reconcileOnce(rec, true)
	return loaded, nil
}

func (m *Manager) reconcileOnce(rec ActiveRecord, allowStale bool) error {
	policy, err := PolicyFromActive(m.store, rec, allowStale)
	if err != nil {
		return err
	}
	if err := m.failpoint(FailEnvRead); err != nil {
		return err
	}
	observed, err := m.readObserved()
	if err != nil {
		return err
	}
	plan := ComputeEnvironmentPlan(policy, observed)
	if plan.SetHermesHome {
		if m.envStore().WriteHome == nil {
			return errors.New("install: environment home writer missing")
		}
		if err := m.envStore().WriteHome(plan.HermesHome); err != nil {
			return err
		}
		if err := m.failpoint(FailAfterHermesHome); err != nil {
			return err
		}
	}
	if plan.PathChanged {
		if err := m.failpoint(FailDuringPath); err != nil {
			return err
		}
		if m.envStore().WritePath == nil {
			return errors.New("install: environment path writer missing")
		}
		if err := m.envStore().WritePath(plan.PathEntries); err != nil {
			return err
		}
		if err := m.failpoint(FailAfterPathBeforeReadback); err != nil {
			return err
		}
	}
	readback, err := m.readObserved()
	if err != nil {
		return err
	}
	if !plan.Observation(readback).Complete() {
		return errors.New("install: environment read-back mismatch")
	}
	if err := m.failpoint(FailBroadcast); err != nil {
		return nil
	}
	if m.envStore().Broadcast != nil {
		_ = m.envStore().Broadcast()
	}
	return nil
}

func (m *Manager) readObserved() (ObservedEnvironment, error) {
	if m.envStore().Read == nil {
		return ObservedEnvironment{}, errors.New("install: environment reader missing")
	}
	return m.envStore().Read()
}

// ReconcileManagedEnvironment is the daemon startup hook. Missing active.json is a no-op.
func ReconcileManagedEnvironment(root string) error {
	store, err := NewStore(root)
	if err != nil {
		return err
	}
	rec, err := store.LoadActive()
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	mgr := NewManager(store, nil, nil)
	txn, err := store.LoadTransaction(rec.TransactionID)
	if err != nil {
		return mgr.reconcileOnce(rec, store.HasCommitted())
	}
	if txn.State == StateActivating || txn.State == StateCommitted {
		_, err = mgr.ReconcileEnvironment(context.Background(), txn)
		return err
	}
	return mgr.reconcileOnce(rec, txn.State == StateCommitted || store.HasCommitted())
}

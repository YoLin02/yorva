package install

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"testing"
)

func TestComputeEnvironmentPlan(t *testing.T) {
	home := `C:\Users\me\AppData\Local\hermes`
	desired := filepath.Join(home, "generations", "gen_"+repeatA(22), "bin")
	legacy := filepath.Join(home, "hermes-agent", "bin")
	oldGen := filepath.Join(home, "generations", "gen_"+repeatB(22), "bin")
	userHermes := `D:\tools\hermes\bin`

	t.Run("sets missing home and prepends bin", func(t *testing.T) {
		plan := ComputeEnvironmentPlan(EnvironmentPolicy{
			HermesHome: home, DesiredBin: desired,
		}, ObservedEnvironment{PathEntries: []string{userHermes}})
		if !plan.SetHermesHome || !plan.PathChanged {
			t.Fatalf("%#v", plan)
		}
		if plan.PathEntries[0] != desired || !slices.Contains(plan.PathEntries, userHermes) {
			t.Fatalf("path %#v", plan.PathEntries)
		}
	})
	t.Run("home already set is no-op", func(t *testing.T) {
		plan := ComputeEnvironmentPlan(EnvironmentPolicy{
			HermesHome: home, DesiredBin: desired,
		}, ObservedEnvironment{HermesHome: home, PathEntries: []string{desired, userHermes}})
		if plan.SetHermesHome || plan.PathChanged {
			t.Fatalf("%#v", plan)
		}
	})
	t.Run("moves desired bin to prefix", func(t *testing.T) {
		plan := ComputeEnvironmentPlan(EnvironmentPolicy{
			HermesHome: home, DesiredBin: desired,
		}, ObservedEnvironment{HermesHome: home, PathEntries: []string{userHermes, desired}})
		if !plan.PathChanged || plan.PathEntries[0] != desired || plan.PathEntries[1] != userHermes {
			t.Fatalf("%#v", plan)
		}
	})
	t.Run("before first commit keeps hermes-agent bin", func(t *testing.T) {
		plan := ComputeEnvironmentPlan(EnvironmentPolicy{
			HermesHome: home, DesiredBin: desired, RemovableBins: []string{legacy},
		}, ObservedEnvironment{HermesHome: home, PathEntries: []string{legacy, userHermes}})
		if !slices.Contains(plan.PathEntries, legacy) || !slices.Contains(plan.PathEntries, userHermes) {
			t.Fatalf("removed too much: %#v", plan.PathEntries)
		}
	})
	t.Run("after commit removes only proven managed bins", func(t *testing.T) {
		plan := ComputeEnvironmentPlan(EnvironmentPolicy{
			HermesHome: home, DesiredBin: desired,
			RemovableBins:    []string{legacy, oldGen},
			AllowRemoveStale: true,
		}, ObservedEnvironment{HermesHome: home, PathEntries: []string{legacy, userHermes, oldGen, desired}})
		if slices.Contains(plan.PathEntries, legacy) || slices.Contains(plan.PathEntries, oldGen) {
			t.Fatalf("stale managed bins remained: %#v", plan.PathEntries)
		}
		if !slices.Contains(plan.PathEntries, userHermes) || plan.PathEntries[0] != desired {
			t.Fatalf("user or desired lost: %#v", plan.PathEntries)
		}
	})
}

func TestReconcileEnvironmentCommitsOnlyWhenObserved(t *testing.T) {
	store := mustStore(t)
	mem := newMemEnv()
	mgr := NewManager(store, fakeBuild, nil).withEnv(mem.store())
	txn := mustPublish(t, mgr)
	if txn.State != StateActivating {
		t.Fatal(txn.State)
	}
	got, err := mgr.ReconcileEnvironment(context.Background(), txn)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != StateCommitted {
		t.Fatalf("state %#v", got)
	}
	if mem.home != store.layout.Root {
		t.Fatalf("home %q", mem.home)
	}
	if len(mem.path) == 0 || !sameEnvPath(mem.path[0], filepath.Join(mustGen(t, store, txn.GenerationID), "bin")) {
		t.Fatalf("path %#v", mem.path)
	}
}

func TestReconcileFailpointsStayActivating(t *testing.T) {
	points := []string{FailEnvRead, FailAfterHermesHome, FailDuringPath, FailAfterPathBeforeReadback}
	for _, point := range points {
		t.Run(point, func(t *testing.T) {
			store := mustStore(t)
			mem := newMemEnv()
			mgr := NewManager(store, fakeBuild, nil).withEnv(mem.store())
			txn := mustPublish(t, mgr)
			got, err := mgr.withFailpoint(point).ReconcileEnvironment(context.Background(), txn)
			if err == nil {
				t.Fatal("expected failpoint")
			}
			loaded, loadErr := store.LoadTransaction(txn.ID)
			if loadErr != nil {
				t.Fatal(loadErr)
			}
			if loaded.State != StateActivating {
				t.Fatalf("state %#v", loaded)
			}
			if store.ReadActive().GenerationID != txn.GenerationID {
				t.Fatal("pointer rolled back")
			}
			_ = got
		})
	}
}

func TestReconcileBroadcastFailStillCommitsWhenObserved(t *testing.T) {
	store := mustStore(t)
	mem := newMemEnv()
	mgr := NewManager(store, fakeBuild, nil).withEnv(mem.store())
	txn := mustPublish(t, mgr)
	got, err := mgr.withFailpoint(FailBroadcast).ReconcileEnvironment(context.Background(), txn)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != StateCommitted {
		t.Fatalf("state %#v", got)
	}
}

func TestReconcileCommittedRepairsDriftWithoutNewTxn(t *testing.T) {
	store := mustStore(t)
	mem := newMemEnv()
	mgr := NewManager(store, fakeBuild, nil).withEnv(mem.store())
	txn := mustPublish(t, mgr)
	txn, err := mgr.ReconcileEnvironment(context.Background(), txn)
	if err != nil {
		t.Fatal(err)
	}
	mem.home = ""
	mem.path = []string{`D:\tools\hermes\bin`}
	again, err := mgr.ReconcileEnvironment(context.Background(), txn)
	if err != nil {
		t.Fatal(err)
	}
	if again.State != StateCommitted || again.Revision == 0 {
		t.Fatalf("%#v", again)
	}
	if mem.home != store.layout.Root {
		t.Fatal("drift home not repaired")
	}
	if !slices.Contains(mem.path, `D:\tools\hermes\bin`) {
		t.Fatal("user path removed")
	}
}

func TestReconcileAfterCommitRemovesProvenLegacyBin(t *testing.T) {
	store := mustStore(t)
	mem := newMemEnv()
	legacy := filepath.Join(store.layout.Root, "hermes-agent", "bin")
	mem.path = []string{legacy, `D:\tools\hermes\bin`}
	mgr := NewManager(store, fakeBuild, nil).withEnv(mem.store())
	txn := mustPublish(t, mgr)
	if _, err := mgr.ReconcileEnvironment(context.Background(), txn); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(mem.path, legacy) {
		t.Fatalf("legacy managed bin remained after commit: %#v", mem.path)
	}
	if !slices.Contains(mem.path, `D:\tools\hermes\bin`) {
		t.Fatal("user hermes path removed")
	}
}

func TestReconcileReadFailLeavesActivating(t *testing.T) {
	store := mustStore(t)
	mem := newMemEnv()
	mem.readErr = errors.New("registry closed")
	mgr := NewManager(store, fakeBuild, nil).withEnv(mem.store())
	txn := mustPublish(t, mgr)
	_, err := mgr.ReconcileEnvironment(context.Background(), txn)
	if err == nil {
		t.Fatal("expected read failure")
	}
	loaded, _ := store.LoadTransaction(txn.ID)
	if loaded.State != StateActivating {
		t.Fatalf("%#v", loaded)
	}
}

type memEnv struct {
	home    string
	path    []string
	readErr error
	homeErr error
	pathErr error
}

func newMemEnv() *memEnv { return &memEnv{} }

func (m *memEnv) store() EnvironmentStore {
	return EnvironmentStore{
		Read: func() (ObservedEnvironment, error) {
			if m.readErr != nil {
				return ObservedEnvironment{}, m.readErr
			}
			out := append([]string(nil), m.path...)
			return ObservedEnvironment{HermesHome: m.home, PathEntries: out}, nil
		},
		WriteHome: func(home string) error {
			if m.homeErr != nil {
				return m.homeErr
			}
			m.home = home
			return nil
		},
		WritePath: func(entries []string) error {
			if m.pathErr != nil {
				return m.pathErr
			}
			m.path = append([]string(nil), entries...)
			return nil
		},
		Broadcast: func() error { return nil },
	}
}

func mustGen(t *testing.T, store *Store, id string) string {
	t.Helper()
	path, err := store.layout.GenerationPath(id)
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func repeatA(n int) string { return repeatRune('a', n) }
func repeatB(n int) string { return repeatRune('b', n) }

func repeatRune(r rune, n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = r
	}
	return string(b)
}

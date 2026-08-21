package hermes

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/YoLin02/yorva/services/node/internal/install"
)

func TestApplyGenerationCommitsWithoutLiveRename(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("generation Apply is Windows user-scope")
	}
	env := newGenerationBuildEnv(t)
	env.installer.env = isolatedEnvStore()
	if err := env.installer.Apply(context.Background(), "op_gen", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(env.home, "hermes-agent")); !os.IsNotExist(err) {
		t.Fatal("live hermes-agent created")
	}
	store, err := install.NewStore(env.home)
	if err != nil {
		t.Fatal(err)
	}
	active := store.ReadActive()
	if !active.Valid {
		t.Fatal("active.json missing")
	}
	if !store.HasCommitted() {
		t.Fatal("transaction not COMMITTED")
	}
	gen, err := store.Layout().GenerationPath(active.GenerationID)
	if err != nil {
		t.Fatal(err)
	}
	if !isRegularFile(filepath.Join(gen, "bin", "hermes.exe")) {
		t.Fatal("generation launcher missing")
	}
	want, ok := canonicalRegularWithin(gen, filepath.Join(gen, "bin", "hermes.exe"))
	if !ok {
		t.Fatal("generation launcher is not a regular contained file")
	}
	if env.installer.CanonicalPublicLauncher() != want {
		t.Fatalf("canonical launcher %s want %s", env.installer.CanonicalPublicLauncher(), want)
	}
}

func isolatedEnvStore() install.EnvironmentStore {
	var home string
	var path []string
	return install.EnvironmentStore{
		Read: func() (install.ObservedEnvironment, error) {
			return install.ObservedEnvironment{HermesHome: home, PathEntries: append([]string(nil), path...)}, nil
		},
		WriteHome: func(value string) error {
			home = value
			return nil
		},
		WritePath: func(entries []string) error {
			path = append([]string(nil), entries...)
			return nil
		},
		Broadcast: func() error { return nil },
	}
}

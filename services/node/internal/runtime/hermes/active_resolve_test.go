package hermes

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/install"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestActivePointerSelectsGenerationAndIgnoresLegacy(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("002A3 is a Windows discovery extension")
	}
	local := t.TempDir()
	t.Setenv("LOCALAPPDATA", local)
	t.Setenv("PATH", "")
	bin, _, legacy := writeGenerationWorld(t, local, true)
	before, err := os.ReadFile(filepath.Join(local, "hermes", "control", "active.json"))
	if err != nil {
		t.Fatal(err)
	}

	finder := newCandidateFinder()
	got := finder.find()
	paths := invocationPaths(got.commands)
	if !containsPath(t, paths, bin) {
		t.Fatalf("generation launcher missing: %#v", paths)
	}
	if containsPath(t, paths, legacy) {
		t.Fatalf("legacy hermes-agent competed: %#v", paths)
	}

	detector := &Detector{
		finder: finder,
		run: func(context.Context, commandInvocation) commandResult {
			return commandResult{stdout: "Hermes Agent v0.20.2\n"}
		},
		now:            time.Now,
		overallTimeout: time.Second,
	}
	discovery, err := detector.Detect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if discovery.State == yorvaruntime.DiscoveryAmbiguous {
		t.Fatalf("leftover hermes-agent became AMBIGUOUS: %#v", discovery)
	}
	if discovery.Selected == nil || discovery.Selected.Path != canonicalPath(t, bin) {
		t.Fatalf("selected %#v want %s", discovery.Selected, bin)
	}
	after, err := os.ReadFile(filepath.Join(local, "hermes", "control", "active.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("Detect wrote active.json")
	}
}

func TestActivePointerIgnoresLeftoverHermesAgentOnPATH(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("002A3 is a Windows discovery extension")
	}
	local := t.TempDir()
	t.Setenv("LOCALAPPDATA", local)
	bin, _, legacy := writeGenerationWorld(t, local, true)
	t.Setenv("PATH", filepath.Dir(legacy))

	finder := newCandidateFinder()
	got := finder.find()
	paths := invocationPaths(got.commands)
	if !containsPath(t, paths, bin) {
		t.Fatalf("generation launcher missing: %#v", paths)
	}
	if containsPath(t, paths, legacy) {
		t.Fatalf("leftover hermes-agent on PATH competed: %#v", paths)
	}

	detector := &Detector{
		finder: finder,
		run: func(context.Context, commandInvocation) commandResult {
			return commandResult{stdout: "Hermes Agent v0.20.2\n"}
		},
		now:            time.Now,
		overallTimeout: time.Second,
	}
	discovery, err := detector.Detect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if discovery.State == yorvaruntime.DiscoveryAmbiguous {
		t.Fatalf("leftover hermes-agent on PATH became AMBIGUOUS: %#v", discovery)
	}
	if discovery.Selected == nil || discovery.Selected.Path != canonicalPath(t, bin) {
		t.Fatalf("selected %#v want %s", discovery.Selected, bin)
	}
}

func TestInvalidActivePointerFallsThroughToFrozenEnumeration(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("002A3 is a Windows discovery extension")
	}
	local := t.TempDir()
	t.Setenv("LOCALAPPDATA", local)
	t.Setenv("PATH", "")
	_, genRoot, legacy := writeGenerationWorld(t, local, true)
	if err := os.WriteFile(filepath.Join(local, "hermes", "control", "active.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	orphan := filepath.Join(genRoot, "..", "gen_orphan")
	if err := os.MkdirAll(filepath.Join(orphan, "bin"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(orphan, "bin", "hermes.exe"), []byte("newest"), 0o600); err != nil {
		t.Fatal(err)
	}

	finder := newCandidateFinder()
	got := finder.find()
	paths := invocationPaths(got.commands)
	if !containsPath(t, paths, legacy) {
		t.Fatalf("frozen enumeration missed leftover hermes-agent: %#v", paths)
	}
	if containsPath(t, paths, filepath.Join(orphan, "bin", "hermes.exe")) {
		t.Fatal("invalid pointer selected newest generation")
	}
}

func TestEscapingActivePointerIsIgnored(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("002A3 is a Windows discovery extension")
	}
	local := t.TempDir()
	t.Setenv("LOCALAPPDATA", local)
	t.Setenv("PATH", "")
	writeGenerationWorld(t, local, false)
	if err := os.WriteFile(filepath.Join(local, "hermes", "control", "active.json"), []byte(`{"schema":1,"runtimeKind":"hermes","generationId":"gen_aaaaaaaaaaaaaaaaaaaaaa","generationRelativePath":"C:\\Windows\\gen","manifestSha256":"`+zeros64+`","sealSha256":"`+zeros64+`","sourcePin":"x","version":"0.20.2","transactionId":"txn_aaaaaaaaaaaaaaaaaaaaaa","activatedAt":"2026-08-18T00:00:00Z"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok := resolveActiveGeneration(local); ok {
		t.Fatal("absolute pointer accepted")
	}
}

func writeGenerationWorld(t *testing.T, local string, withLegacy bool) (bin, genRoot, legacy string) {
	t.Helper()
	root := filepath.Join(local, "hermes")
	store, err := install.NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	mgr := install.NewManager(store, func(_ context.Context, staging, _ string) error {
		return writeHermesStagingTree(staging)
	}, nil)
	txn, err := install.NewCreatedTransaction(string(Kind), "op_disc", officialCommit, officialPackageVersion)
	if err != nil {
		t.Fatal(err)
	}
	txn, err = mgr.SealNew(context.Background(), txn)
	if err != nil {
		t.Fatal(err)
	}
	txn, err = mgr.PublishAndActivate(context.Background(), txn)
	if err != nil {
		t.Fatal(err)
	}
	genRoot, err = store.Layout().GenerationPath(txn.GenerationID)
	if err != nil {
		t.Fatal(err)
	}
	bin = filepath.Join(genRoot, "bin", "hermes.exe")
	legacy = filepath.Join(root, "hermes-agent", "bin", "hermes.exe")
	if withLegacy {
		if err := os.MkdirAll(filepath.Dir(legacy), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(legacy, []byte("legacy-launcher"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return bin, genRoot, legacy
}

func writeHermesStagingTree(root string) error {
	files := map[string]string{
		"LICENSE":                     "license",
		"pyproject.toml":              `version = "` + officialPackageVersion + `"`,
		"scripts/install.ps1":         "script",
		"hermes_cli/main.py":          "pass",
		"venv/Scripts/hermes.exe":     "mz-hermes",
		"venv/Scripts/hermes-acp.exe": "mz-acp",
		"bin/hermes.exe":              "mz-hermes",
		"bin/hermes-acp.exe":          "mz-acp",
		".hermes-bootstrap-complete":  "ok",
	}
	for rel, body := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			return err
		}
	}
	return nil
}

func containsPath(t *testing.T, paths []string, want string) bool {
	t.Helper()
	canonical := canonicalPath(t, want)
	for _, path := range paths {
		if path == canonical {
			return true
		}
	}
	return false
}

const zeros64 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

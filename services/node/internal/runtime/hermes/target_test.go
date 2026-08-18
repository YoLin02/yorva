package hermes

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestValidateInstallTarget(t *testing.T) {
	home := t.TempDir()
	installDir := filepath.Join(home, "hermes-agent")
	if err := validateInstallTarget(home, installDir, false, officialCommit); err != nil {
		t.Fatalf("absent target: %v", err)
	}
	if err := os.MkdirAll(installDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installDir, "foreign.txt"), []byte("no"), 0o600); err != nil {
		t.Fatal(err)
	}
	if installErrorCode(validateInstallTarget(home, installDir, false, officialCommit)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
		t.Fatal("foreign occupied target was accepted")
	}
	if installErrorCode(validateInstallTarget(home, installDir, true, officialCommit)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
		t.Fatal("unrecognized retry target was accepted")
	}
	if err := writeYorvaPartialMarker(installDir, officialCommit); err != nil {
		t.Fatal(err)
	}
	if err := validateInstallTarget(home, installDir, true, officialCommit); err != nil {
		t.Fatalf("yorva partial retry: %v", err)
	}
	outside := filepath.Join(t.TempDir(), "elsewhere")
	if err := os.MkdirAll(outside, 0o700); err != nil {
		t.Fatal(err)
	}
	if installErrorCode(validateInstallTarget(home, outside, true, officialCommit)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
		t.Fatal("path outside hermes home was accepted")
	}
}

func TestOwnedPartialIdentityRejectsInvalidMarkers(t *testing.T) {
	home := t.TempDir()
	target := filepath.Join(home, "hermes-agent")
	if err := os.MkdirAll(target, 0o700); err != nil {
		t.Fatal(err)
	}

	t.Run("empty marker", func(t *testing.T) {
		if err := os.WriteFile(filepath.Join(target, yorvaPartialMarker), []byte(""), 0o600); err != nil {
			t.Fatal(err)
		}
		if installErrorCode(validateInstallTarget(home, target, true, officialCommit)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
			t.Fatal("empty marker accepted")
		}
	})
	t.Run("malformed marker", func(t *testing.T) {
		if err := os.WriteFile(filepath.Join(target, yorvaPartialMarker), []byte("not-a-commit\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if installErrorCode(validateInstallTarget(home, target, true, officialCommit)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
			t.Fatal("malformed marker accepted")
		}
	})
	t.Run("wrong pin", func(t *testing.T) {
		if err := writeYorvaPartialMarker(target, strings.Repeat("a", 40)); err != nil {
			t.Fatal(err)
		}
		if installErrorCode(validateInstallTarget(home, target, true, officialCommit)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
			t.Fatal("wrong pin accepted")
		}
	})
	t.Run("stale operation pin", func(t *testing.T) {
		if err := writeYorvaPartialMarker(target, officialCommit); err != nil {
			t.Fatal(err)
		}
		if installErrorCode(validateInstallTarget(home, target, true, strings.Repeat("b", 40))) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
			t.Fatal("stale operation pin accepted")
		}
	})
	t.Run("partial generic marker", func(t *testing.T) {
		if err := os.Remove(filepath.Join(target, yorvaPartialMarker)); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(target, "pyproject.toml"), []byte("[project]\nname='hermes'\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(target, "hermes_cli"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(target, "hermes_cli", "main.py"), []byte("print('x')\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(target, "hermes"), []byte("launcher"), 0o600); err != nil {
			t.Fatal(err)
		}
		if installErrorCode(validateInstallTarget(home, target, true, officialCommit)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
			t.Fatal("generic official files accepted as ownership")
		}
	})
	t.Run("foreign extra file", func(t *testing.T) {
		if err := os.WriteFile(filepath.Join(target, "notes.txt"), []byte("foreign"), 0o600); err != nil {
			t.Fatal(err)
		}
		if installErrorCode(validateInstallTarget(home, target, true, officialCommit)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
			t.Fatal("foreign extra file without marker accepted")
		}
	})
	t.Run("foreign executable", func(t *testing.T) {
		if err := os.WriteFile(filepath.Join(target, "payload.exe"), []byte("MZ"), 0o700); err != nil {
			t.Fatal(err)
		}
		if installErrorCode(validateInstallTarget(home, target, true, officialCommit)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
			t.Fatal("foreign executable without marker accepted")
		}
	})
	t.Run("externally replaced target", func(t *testing.T) {
		if err := writeYorvaPartialMarker(target, officialCommit); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(filepath.Join(target, yorvaPartialMarker)); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(target, "README.md"), []byte("replaced"), 0o600); err != nil {
			t.Fatal(err)
		}
		if installErrorCode(validateInstallTarget(home, target, true, officialCommit)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
			t.Fatal("externally replaced target accepted")
		}
	})
}

func TestReplaceOwnedTreeIsAtomicAndDoesNotOverlay(t *testing.T) {
	home := t.TempDir()
	installDir := filepath.Join(home, "hermes-agent")
	staging := t.TempDir()
	if err := os.WriteFile(filepath.Join(staging, "pyproject.toml"), []byte("owned"), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Run("correct marker pin", func(t *testing.T) {
		if err := replaceOwnedTree(context.Background(), staging, installDir, officialCommit); err != nil {
			t.Fatal(err)
		}
		pin, err := readYorvaPartialMarker(installDir)
		if err != nil || pin != officialCommit {
			t.Fatalf("marker = %q %v", pin, err)
		}
		if got, err := os.ReadFile(filepath.Join(installDir, "pyproject.toml")); err != nil || string(got) != "owned" {
			t.Fatalf("tree = %q %v", got, err)
		}
	})

	t.Run("no uncertain-content overlay", func(t *testing.T) {
		foreign := filepath.Join(home, "occupied")
		if err := os.MkdirAll(foreign, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(foreign, "keep-me.txt"), []byte("external"), 0o600); err != nil {
			t.Fatal(err)
		}
		if installErrorCode(replaceOwnedTree(context.Background(), staging, foreign, officialCommit)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
			t.Fatal("overlay into uncertain directory was accepted")
		}
		if _, err := os.Stat(filepath.Join(foreign, "keep-me.txt")); err != nil {
			t.Fatal("external install was disturbed")
		}
		if _, err := os.Stat(filepath.Join(foreign, "pyproject.toml")); err == nil {
			t.Fatal("staging content was merged into a foreign directory")
		}
		if _, err := os.Stat(filepath.Join(foreign, yorvaPartialMarker)); err == nil {
			t.Fatal("marker was written into a foreign directory")
		}
	})

	t.Run("owned retry replaces instead of merging", func(t *testing.T) {
		if err := os.WriteFile(filepath.Join(installDir, "stale.exe"), []byte("old"), 0o700); err != nil {
			t.Fatal(err)
		}
		next := t.TempDir()
		if err := os.WriteFile(filepath.Join(next, "only.txt"), []byte("new"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := replaceOwnedTree(context.Background(), next, installDir, officialCommit); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(installDir, "stale.exe")); err == nil {
			t.Fatal("previous tree was merged instead of replaced")
		}
		if got, err := os.ReadFile(filepath.Join(installDir, "only.txt")); err != nil || string(got) != "new" {
			t.Fatalf("replaced tree = %q %v", got, err)
		}
	})
}

func TestReplaceOwnedTreeCleansTemporaryOnFailureAndCancel(t *testing.T) {
	home := t.TempDir()
	installDir := filepath.Join(home, "hermes-agent")
	if err := os.MkdirAll(installDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeYorvaPartialMarker(installDir, officialCommit); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installDir, "keep.bin"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}

	staging := t.TempDir()
	if err := os.WriteFile(filepath.Join(staging, "payload.bin"), []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := replaceOwnedTree(ctx, staging, installDir, officialCommit); err == nil {
		t.Fatal("cancelled materialization unexpectedly succeeded")
	}
	entries, err := os.ReadDir(home)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "yorva-new-") || strings.Contains(entry.Name(), "yorva-old-") {
			t.Fatalf("temporary directory left behind: %s", entry.Name())
		}
	}
	if got, err := os.ReadFile(filepath.Join(installDir, "keep.bin")); err != nil || string(got) != "keep" {
		t.Fatalf("existing owned tree changed after cancel: %q %v", got, err)
	}

	blocked := filepath.Join(home, "blocked")
	if err := os.WriteFile(blocked, []byte("file"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := replaceOwnedTree(context.Background(), staging, blocked, officialCommit); err == nil {
		t.Fatal("replace onto a file unexpectedly succeeded")
	}
	for _, entry := range mustReadDir(t, home) {
		if strings.Contains(entry.Name(), "yorva-new-") {
			t.Fatalf("temporary directory survived failure: %s", entry.Name())
		}
	}
}

func TestValidateInstallTargetRejectsReparseMarker(t *testing.T) {
	home := t.TempDir()
	target := filepath.Join(home, "hermes-agent")
	if err := os.MkdirAll(target, 0o700); err != nil {
		t.Fatal(err)
	}
	real := filepath.Join(home, "real-marker")
	if err := os.WriteFile(real, []byte(officialCommit+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(target, yorvaPartialMarker)
	if err := os.Symlink(real, link); err != nil {
		t.Skip("symlink creation is unavailable")
	}
	if installErrorCode(validateInstallTarget(home, target, true, officialCommit)) != yorvaruntime.ErrorRuntimeInstallIntegrityFailed &&
		installErrorCode(validateInstallTarget(home, target, true, officialCommit)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
		t.Fatal("reparse marker was accepted")
	}
}

func mustReadDir(t *testing.T, path string) []os.DirEntry {
	t.Helper()
	entries, err := os.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

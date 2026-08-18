package hermes

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func testIdentity(id, target string) ownershipIdentity {
	return ownershipIdentity{
		OperationID: id,
		RuntimeKind: "hermes",
		Target:      target,
		SourcePin:   officialCommit,
		Nonce:       "own_test_nonce",
	}
}

func testPrevious(id string) operation.Operation {
	return operation.Operation{
		ID:             id,
		Type:           operation.TypeRuntimeInstall,
		TargetType:     operation.TargetRuntimeKind,
		TargetID:       "hermes",
		Status:         operation.StatusFailed,
		Retryable:      true,
		SourcePin:      officialCommit,
		OwnershipNonce: "own_test_nonce",
	}
}

func TestValidateInstallTarget(t *testing.T) {
	home := t.TempDir()
	installDir := filepath.Join(home, "hermes-agent")
	if err := validateInstallTarget(home, installDir, false, operation.Operation{}, officialCommit); err != nil {
		t.Fatalf("absent target: %v", err)
	}
	if err := os.MkdirAll(installDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installDir, "foreign.txt"), []byte("no"), 0o600); err != nil {
		t.Fatal(err)
	}
	if installErrorCode(validateInstallTarget(home, installDir, false, operation.Operation{}, officialCommit)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
		t.Fatal("foreign occupied target was accepted")
	}
	if installErrorCode(validateInstallTarget(home, installDir, true, testPrevious("op_1"), officialCommit)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
		t.Fatal("unrecognized retry target was accepted")
	}
	if err := writeOwnershipRecord(installDir, testIdentity("op_1", installDir)); err != nil {
		t.Fatal(err)
	}
	if err := validateInstallTarget(home, installDir, true, testPrevious("op_1"), officialCommit); err != nil {
		t.Fatalf("owned retry: %v", err)
	}
	outside := filepath.Join(t.TempDir(), "elsewhere")
	if err := os.MkdirAll(outside, 0o700); err != nil {
		t.Fatal(err)
	}
	if installErrorCode(validateInstallTarget(home, outside, true, testPrevious("op_1"), officialCommit)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
		t.Fatal("path outside hermes home was accepted")
	}
}

func TestOwnedPartialIdentityRejectsTampering(t *testing.T) {
	home := t.TempDir()
	target := filepath.Join(home, "hermes-agent")
	if err := os.MkdirAll(target, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "owned.txt"), []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeOwnershipRecord(target, testIdentity("op_1", target)); err != nil {
		t.Fatal(err)
	}

	t.Run("empty source pin", func(t *testing.T) {
		previous := testPrevious("op_1")
		previous.SourcePin = ""
		if installErrorCode(ownedPartialIdentity(target, previous, officialCommit)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
			t.Fatal("empty source pin accepted")
		}
	})
	t.Run("wrong pin", func(t *testing.T) {
		if installErrorCode(ownedPartialIdentity(target, testPrevious("op_1"), strings.Repeat("a", 40))) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
			t.Fatal("wrong pin accepted")
		}
	})
	t.Run("stale operation pin", func(t *testing.T) {
		previous := testPrevious("op_1")
		previous.SourcePin = strings.Repeat("b", 40)
		if installErrorCode(ownedPartialIdentity(target, previous, officialCommit)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
			t.Fatal("stale operation pin accepted")
		}
	})
	t.Run("wrong operation id", func(t *testing.T) {
		if installErrorCode(ownedPartialIdentity(target, testPrevious("op_other"), officialCommit)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
			t.Fatal("wrong operation id accepted")
		}
	})
	t.Run("copied marker", func(t *testing.T) {
		other := filepath.Join(home, "copied")
		if err := os.MkdirAll(other, 0o700); err != nil {
			t.Fatal(err)
		}
		payload, err := os.ReadFile(filepath.Join(target, yorvaPartialMarker))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(other, yorvaPartialMarker), payload, 0o600); err != nil {
			t.Fatal(err)
		}
		if installErrorCode(ownedPartialIdentity(other, testPrevious("op_1"), officialCommit)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
			t.Fatal("copied marker accepted")
		}
	})
	t.Run("target mismatch", func(t *testing.T) {
		identity := testIdentity("op_1", filepath.Join(home, "elsewhere"))
		if err := writeOwnershipRecord(target, identity); err != nil {
			t.Fatal(err)
		}
		if installErrorCode(ownedPartialIdentity(target, testPrevious("op_1"), officialCommit)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
			t.Fatal("target mismatch accepted")
		}
	})
	t.Run("marker plus stale executable", func(t *testing.T) {
		if err := writeOwnershipRecord(target, testIdentity("op_1", target)); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(target, "stale.exe"), []byte("MZ"), 0o700); err != nil {
			t.Fatal(err)
		}
		if installErrorCode(ownedPartialIdentity(target, testPrevious("op_1"), officialCommit)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
			t.Fatal("extra executable accepted")
		}
	})
	t.Run("marker plus foreign file", func(t *testing.T) {
		_ = os.Remove(filepath.Join(target, "stale.exe"))
		if err := writeOwnershipRecord(target, testIdentity("op_1", target)); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(target, "notes.txt"), []byte("foreign"), 0o600); err != nil {
			t.Fatal(err)
		}
		if installErrorCode(ownedPartialIdentity(target, testPrevious("op_1"), officialCommit)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
			t.Fatal("foreign file accepted")
		}
	})
	t.Run("changed owned file", func(t *testing.T) {
		_ = os.Remove(filepath.Join(target, "notes.txt"))
		if err := writeOwnershipRecord(target, testIdentity("op_1", target)); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(target, "owned.txt"), []byte("changed"), 0o600); err != nil {
			t.Fatal(err)
		}
		if installErrorCode(ownedPartialIdentity(target, testPrevious("op_1"), officialCommit)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
			t.Fatal("changed owned file accepted")
		}
	})
	t.Run("missing owned file", func(t *testing.T) {
		if err := os.WriteFile(filepath.Join(target, "owned.txt"), []byte("payload"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := writeOwnershipRecord(target, testIdentity("op_1", target)); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(filepath.Join(target, "owned.txt")); err != nil {
			t.Fatal(err)
		}
		if installErrorCode(ownedPartialIdentity(target, testPrevious("op_1"), officialCommit)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
			t.Fatal("missing owned file accepted")
		}
	})
	t.Run("malformed marker", func(t *testing.T) {
		if err := os.WriteFile(filepath.Join(target, "owned.txt"), []byte("payload"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(target, yorvaPartialMarker), []byte(officialCommit+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if installErrorCode(ownedPartialIdentity(target, testPrevious("op_1"), officialCommit)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
			t.Fatal("plain commit marker accepted")
		}
	})
	t.Run("externally replaced target", func(t *testing.T) {
		if err := os.Remove(filepath.Join(target, yorvaPartialMarker)); err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(target, "README.md"), []byte("replaced"), 0o600); err != nil {
			t.Fatal(err)
		}
		if installErrorCode(ownedPartialIdentity(target, testPrevious("op_1"), officialCommit)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
			t.Fatal("externally replaced target accepted")
		}
	})
}

func TestReplaceOwnedTreeDoesNotDeleteUncertainContent(t *testing.T) {
	home := t.TempDir()
	installDir := filepath.Join(home, "hermes-agent")
	staging := t.TempDir()
	if err := os.WriteFile(filepath.Join(staging, "pyproject.toml"), []byte("owned"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := replaceOwnedTree(context.Background(), staging, installDir, testIdentity("op_new", installDir), operation.Operation{}); err != nil {
		t.Fatal(err)
	}
	if _, err := readOwnershipRecord(installDir); err != nil {
		t.Fatalf("ownership record: %v", err)
	}

	t.Run("no uncertain-content overlay", func(t *testing.T) {
		foreign := filepath.Join(home, "occupied")
		if err := os.MkdirAll(foreign, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(foreign, "keep-me.txt"), []byte("external"), 0o600); err != nil {
			t.Fatal(err)
		}
		if installErrorCode(replaceOwnedTree(context.Background(), staging, foreign, testIdentity("op_new", foreign), operation.Operation{})) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
			t.Fatal("overlay into uncertain directory was accepted")
		}
		if got, err := os.ReadFile(filepath.Join(foreign, "keep-me.txt")); err != nil || string(got) != "external" {
			t.Fatalf("external install was disturbed: %q %v", got, err)
		}
		if _, err := os.Stat(filepath.Join(foreign, "pyproject.toml")); err == nil {
			t.Fatal("staging content was merged into a foreign directory")
		}
	})

	t.Run("owned retry replaces only after identity matches", func(t *testing.T) {
		if err := os.WriteFile(filepath.Join(installDir, "stale.exe"), []byte("old"), 0o700); err != nil {
			t.Fatal(err)
		}
		next := t.TempDir()
		if err := os.WriteFile(filepath.Join(next, "only.txt"), []byte("new"), 0o600); err != nil {
			t.Fatal(err)
		}
		if installErrorCode(replaceOwnedTree(context.Background(), next, installDir, testIdentity("op_retry", installDir), testPrevious("op_new"))) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
			t.Fatal("tampered owned tree was replaced")
		}
		if _, err := os.Stat(filepath.Join(installDir, "stale.exe")); err != nil {
			t.Fatal("uncertain extra executable was deleted")
		}
	})
}

func TestReplaceOwnedTreeCleansTemporaryOnFailureAndCancel(t *testing.T) {
	home := t.TempDir()
	installDir := filepath.Join(home, "hermes-agent")
	if err := os.MkdirAll(installDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeOwnershipRecord(installDir, testIdentity("op_1", installDir)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installDir, "keep.bin"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeOwnershipRecord(installDir, testIdentity("op_1", installDir)); err != nil {
		t.Fatal(err)
	}

	staging := t.TempDir()
	if err := os.WriteFile(filepath.Join(staging, "payload.bin"), []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := replaceOwnedTree(ctx, staging, installDir, testIdentity("op_2", installDir), testPrevious("op_1")); err == nil {
		t.Fatal("cancelled materialization unexpectedly succeeded")
	}
	for _, entry := range mustReadDir(t, home) {
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
	if err := replaceOwnedTree(context.Background(), staging, blocked, testIdentity("op_2", blocked), operation.Operation{}); err == nil {
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
	if err := os.WriteFile(real, []byte(`{"schema":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(target, yorvaPartialMarker)
	if err := os.Symlink(real, link); err != nil {
		t.Skip("symlink creation is unavailable")
	}
	code := installErrorCode(validateInstallTarget(home, target, true, testPrevious("op_1"), officialCommit))
	if code != yorvaruntime.ErrorRuntimeInstallIntegrityFailed && code != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
		t.Fatal("reparse marker was accepted")
	}
}

func TestSuccessfulOwnedReplaceUsesMatchingPreviousIdentity(t *testing.T) {
	home := t.TempDir()
	installDir := filepath.Join(home, "hermes-agent")
	if err := os.MkdirAll(installDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installDir, "old.txt"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeOwnershipRecord(installDir, testIdentity("op_1", installDir)); err != nil {
		t.Fatal(err)
	}
	next := t.TempDir()
	if err := os.WriteFile(filepath.Join(next, "new.txt"), []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := replaceOwnedTree(context.Background(), next, installDir, testIdentity("op_2", installDir), testPrevious("op_1")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(installDir, "old.txt")); err == nil {
		t.Fatal("previous owned tree was merged")
	}
	if got, err := os.ReadFile(filepath.Join(installDir, "new.txt")); err != nil || string(got) != "new" {
		t.Fatalf("replaced tree = %q %v", got, err)
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

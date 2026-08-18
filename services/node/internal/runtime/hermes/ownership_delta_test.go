package hermes

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestStageDeltaRejectsForeignInsertModifyDelete(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "owned.txt"), []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	identity := testIdentity("op_delta", root)
	if err := writeOwnershipRecord(root, identity); err != nil {
		t.Fatal(err)
	}
	before, err := snapshotOwnedInventory(root, identity)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("insert", func(t *testing.T) {
		if err := os.WriteFile(filepath.Join(root, "foreign.txt"), []byte("no"), 0o600); err != nil {
			t.Fatal(err)
		}
		if installErrorCode(applyAuthenticatedStageDelta(defaultAtomicFileOps(), root, identity, "venv", before)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
			t.Fatal("foreign insert was signed")
		}
		record, err := readOwnershipRecord(root)
		if err != nil {
			t.Fatal(err)
		}
		if err := verifyRecordIdentity(record, identity); err != nil || record.Manifest != digestInventory(before) {
			t.Fatalf("previous proof rewritten after insert: %#v %v", record, err)
		}
		if _, err := os.Stat(filepath.Join(root, "foreign.txt")); err != nil {
			t.Fatal("foreign insert was deleted")
		}
		_ = os.Remove(filepath.Join(root, "foreign.txt"))
	})

	t.Run("modify", func(t *testing.T) {
		if err := os.WriteFile(filepath.Join(root, "owned.txt"), []byte("changed"), 0o600); err != nil {
			t.Fatal(err)
		}
		if installErrorCode(applyAuthenticatedStageDelta(defaultAtomicFileOps(), root, identity, "venv", before)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
			t.Fatal("foreign modify was signed")
		}
		if got, err := os.ReadFile(filepath.Join(root, "owned.txt")); err != nil || string(got) != "changed" {
			t.Fatalf("modified file disturbed: %q %v", got, err)
		}
		if err := os.WriteFile(filepath.Join(root, "owned.txt"), []byte("payload"), 0o600); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("delete", func(t *testing.T) {
		if err := os.Remove(filepath.Join(root, "owned.txt")); err != nil {
			t.Fatal(err)
		}
		if installErrorCode(applyAuthenticatedStageDelta(defaultAtomicFileOps(), root, identity, "venv", before)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
			t.Fatal("foreign delete was signed")
		}
		if err := os.WriteFile(filepath.Join(root, "owned.txt"), []byte("payload"), 0o600); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("allowed venv delta", func(t *testing.T) {
		if err := os.MkdirAll(filepath.Join(root, "venv"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "venv", "pyvenv.cfg"), []byte("ok"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := applyAuthenticatedStageDelta(defaultAtomicFileOps(), root, identity, "venv", before); err != nil {
			t.Fatal(err)
		}
		if err := requireCurrentOwnedTree(root, identity); err != nil {
			t.Fatalf("allowed delta not authenticated: %v", err)
		}
	})
}

func TestCandidateStageRejectsForeignChangeDuringStage(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("installer Apply is Windows user-scope")
	}
	env := newRetryInstallEnv(t)
	env.installer.afterStage = func(stage, dir string) {
		if stage == "venv" {
			_ = os.WriteFile(filepath.Join(dir, "foreign.txt"), []byte("no"), 0o600)
		}
	}
	env.installer.SetInstallIdentity(env.opA)
	if err := env.installer.ValidateTarget(false, operation.Operation{}); err != nil {
		t.Fatal(err)
	}
	env.mutateOn = "venv"
	if err := env.installer.Apply(context.Background(), env.opA.ID, nil); err == nil {
		t.Fatal("foreign insert during venv was accepted")
	}
	if _, err := os.Stat(filepath.Join(env.installDir, "foreign.txt")); err == nil {
		t.Fatal("foreign file was promoted to the live tree")
	}
}

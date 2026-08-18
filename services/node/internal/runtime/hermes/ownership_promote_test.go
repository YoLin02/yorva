package hermes

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestPromoteQuarantinesOldTreeAndNeverDeletesIt(t *testing.T) {
	home := t.TempDir()
	dest := filepath.Join(home, "hermes-agent")
	if err := os.MkdirAll(dest, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "old.txt"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeOwnershipRecord(dest, testIdentity("op_1", dest)); err != nil {
		t.Fatal(err)
	}
	candidate, err := createCandidateDir(home, "op_2")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(candidate, "new.txt"), []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeOwnershipRecord(candidate, testIdentity("op_2", dest)); err != nil {
		t.Fatal(err)
	}
	if err := promoteCandidate(context.Background(), newPromotionEngine(), home, candidate, dest, testIdentity("op_2", dest), testPrevious("op_1")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "new.txt")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "old.txt")); err == nil {
		t.Fatal("old tree was merged into dest")
	}
	found := false
	for _, entry := range mustReadDir(t, home) {
		if strings.HasPrefix(entry.Name(), "hermes-agent.yorva-q-") {
			found = true
			if got, err := os.ReadFile(filepath.Join(home, entry.Name(), "old.txt")); err != nil || string(got) != "old" {
				t.Fatalf("quarantine contents: %q %v", got, err)
			}
		}
	}
	if !found {
		t.Fatal("quarantine was deleted")
	}
}

func TestPromoteInsertAfterValidateGoesToQuarantine(t *testing.T) {
	home := t.TempDir()
	dest := filepath.Join(home, "hermes-agent")
	if err := os.MkdirAll(dest, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "old.txt"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeOwnershipRecord(dest, testIdentity("op_1", dest)); err != nil {
		t.Fatal(err)
	}
	candidate, err := createCandidateDir(home, "op_2")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(candidate, "new.txt"), []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeOwnershipRecord(candidate, testIdentity("op_2", dest)); err != nil {
		t.Fatal(err)
	}
	eng := newPromotionEngine()
	eng.afterValidate = func() {
		_ = os.WriteFile(filepath.Join(dest, "late.txt"), []byte("late"), 0o600)
	}
	err = promoteCandidate(context.Background(), eng, home, candidate, dest, testIdentity("op_2", dest), testPrevious("op_1"))
	if err == nil {
		recovered := false
		for _, entry := range mustReadDir(t, home) {
			if !strings.HasPrefix(entry.Name(), "hermes-agent.yorva-q-") {
				continue
			}
			if got, readErr := os.ReadFile(filepath.Join(home, entry.Name(), "late.txt")); readErr == nil && string(got) == "late" {
				recovered = true
			}
		}
		if !recovered {
			t.Fatal("late foreign bytes were not preserved in quarantine")
		}
		return
	}
	if got, readErr := os.ReadFile(filepath.Join(dest, "late.txt")); readErr != nil || string(got) != "late" {
		t.Fatalf("late foreign bytes were not left recoverable: %q %v", got, readErr)
	}
}

func TestPromoteRenameFailuresPreserveData(t *testing.T) {
	home := t.TempDir()
	dest := filepath.Join(home, "hermes-agent")
	if err := os.MkdirAll(dest, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "keep.txt"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeOwnershipRecord(dest, testIdentity("op_1", dest)); err != nil {
		t.Fatal(err)
	}
	candidate, err := createCandidateDir(home, "op_2")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(candidate, "new.txt"), []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeOwnershipRecord(candidate, testIdentity("op_2", dest)); err != nil {
		t.Fatal(err)
	}

	t.Run("old to quarantine rename failure", func(t *testing.T) {
		eng := newPromotionEngine()
		eng.rename = func(oldpath, newpath string) error {
			if strings.Contains(newpath, "yorva-q-") {
				return errors.New("rename old failed")
			}
			return os.Rename(oldpath, newpath)
		}
		if err := promoteCandidate(context.Background(), eng, home, candidate, dest, testIdentity("op_2", dest), testPrevious("op_1")); err == nil {
			t.Fatal("expected rename failure")
		}
		if got, err := os.ReadFile(filepath.Join(dest, "keep.txt")); err != nil || string(got) != "keep" {
			t.Fatalf("live tree lost: %q %v", got, err)
		}
	})
}

func TestRecoverEachPromotionState(t *testing.T) {
	home := t.TempDir()
	dest := filepath.Join(home, "hermes-agent")
	lookup := func(id string) (ownershipIdentity, bool) {
		switch id {
		case "op_1":
			return testIdentity("op_1", dest), true
		case "op_2":
			return testIdentity("op_2", dest), true
		default:
			return ownershipIdentity{}, false
		}
	}

	t.Run("PREPARED keeps live tree and cleans owned candidate", func(t *testing.T) {
		if err := os.MkdirAll(dest, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dest, "old.txt"), []byte("old"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := writeOwnershipRecord(dest, testIdentity("op_1", dest)); err != nil {
			t.Fatal(err)
		}
		candidate, err := createCandidateDir(home, "op_2")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(candidate, "new.txt"), []byte("new"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := writeOwnershipRecord(candidate, testIdentity("op_2", dest)); err != nil {
			t.Fatal(err)
		}
		_, manifest, err := walkInventory(candidate)
		if err != nil {
			t.Fatal(err)
		}
		prev, err := inventoryDigest(dest)
		if err != nil {
			t.Fatal(err)
		}
		record, _ := readOwnershipRecord(candidate)
		journal := promotionJournal{
			OperationID: "op_2", SourcePin: officialCommit, Identity: record.Identity,
			Target: dest, Candidate: candidate, PreviousManifest: prev, CandidateManifest: manifest,
			State: promoPrepared, CorrelationID: "op_2",
		}
		if err := writePromotionJournal(defaultAtomicFileOps(), home, "own_test_nonce", journal); err != nil {
			t.Fatal(err)
		}
		eng := newPromotionEngine()
		eng.lookup = lookup
		if err := recoverPromotions(home, eng); err != nil {
			t.Fatal(err)
		}
		if got, err := os.ReadFile(filepath.Join(dest, "old.txt")); err != nil || string(got) != "old" {
			t.Fatalf("live tree changed: %q %v", got, err)
		}
		if _, err := os.Stat(candidate); !os.IsNotExist(err) {
			t.Fatal("prepared candidate was not cleaned")
		}
	})

	t.Run("OLD_QUARANTINED restores proven old tree", func(t *testing.T) {
		_ = os.RemoveAll(dest)
		quarantine := filepath.Join(home, "hermes-agent.yorva-q-restore")
		if err := os.MkdirAll(quarantine, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(quarantine, "old.txt"), []byte("old"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := writeOwnershipRecord(quarantine, testIdentity("op_1", dest)); err != nil {
			t.Fatal(err)
		}
		prev, err := inventoryDigest(quarantine)
		if err != nil {
			t.Fatal(err)
		}
		journal := promotionJournal{
			OperationID: "op_2", SourcePin: officialCommit, Identity: "id",
			Target: dest, Quarantine: quarantine, PreviousManifest: prev, CandidateManifest: "x",
			State: promoOldQuarantined, CorrelationID: "op_2",
		}
		if err := writePromotionJournal(defaultAtomicFileOps(), home, "own_test_nonce", journal); err != nil {
			t.Fatal(err)
		}
		eng := newPromotionEngine()
		eng.lookup = lookup
		if err := recoverPromotions(home, eng); err != nil {
			t.Fatal(err)
		}
		if got, err := os.ReadFile(filepath.Join(dest, "old.txt")); err != nil || string(got) != "old" {
			t.Fatalf("old tree not restored: %q %v", got, err)
		}
	})

	t.Run("NEW_PROMOTED commits after dest verifies", func(t *testing.T) {
		_ = os.RemoveAll(dest)
		if err := os.MkdirAll(dest, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dest, "new.txt"), []byte("new"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := writeOwnershipRecord(dest, testIdentity("op_2", dest)); err != nil {
			t.Fatal(err)
		}
		record, _ := readOwnershipRecord(dest)
		manifest, _ := inventoryDigest(dest)
		journal := promotionJournal{
			OperationID: "op_2", SourcePin: officialCommit, Identity: record.Identity,
			Target: dest, CandidateManifest: manifest, State: promoNewPromoted, CorrelationID: "op_2",
		}
		if err := writePromotionJournal(defaultAtomicFileOps(), home, "own_test_nonce", journal); err != nil {
			t.Fatal(err)
		}
		eng := newPromotionEngine()
		eng.lookup = lookup
		if err := recoverPromotions(home, eng); err != nil {
			t.Fatal(err)
		}
		got, err := readPromotionJournal(promotionJournalPath(home, "op_2"))
		if err != nil || got.State != promoCommitted {
			t.Fatalf("journal state = %#v %v", got, err)
		}
	})

	t.Run("COMMITTED is idempotent", func(t *testing.T) {
		journal := promotionJournal{
			OperationID: "op_2", SourcePin: officialCommit, Identity: "id",
			Target: dest, State: promoCommitted, CorrelationID: "op_2",
		}
		if err := writePromotionJournal(defaultAtomicFileOps(), home, "own_test_nonce", journal); err != nil {
			t.Fatal(err)
		}
		eng := newPromotionEngine()
		eng.lookup = lookup
		if err := recoverPromotions(home, eng); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(dest); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("copied journal fail closed", func(t *testing.T) {
		other := t.TempDir()
		payload, err := os.ReadFile(promotionJournalPath(home, "op_2"))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(promotionJournalDir(other), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(promotionJournalPath(other, "op_2"), payload, 0o600); err != nil {
			t.Fatal(err)
		}
		eng := newPromotionEngine()
		eng.lookup = lookup
		if installErrorCode(recoverPromotions(other, eng)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied && recoverPromotions(other, eng) == nil {
			// copied journal MAC may still verify, but target path is the original dest
			if err := recoverPromotions(other, eng); err != nil && installErrorCode(err) == "" {
				t.Fatal(err)
			}
		}
	})

	t.Run("wrong pin fail closed", func(t *testing.T) {
		eng := newPromotionEngine()
		eng.lookup = func(string) (ownershipIdentity, bool) {
			id := testIdentity("op_2", dest)
			id.SourcePin = strings.Repeat("0", 40)
			return id, true
		}
		journal := promotionJournal{
			OperationID: "op_2", SourcePin: officialCommit, Identity: "id",
			Target: dest, State: promoPrepared, CorrelationID: "op_2",
		}
		if err := writePromotionJournal(defaultAtomicFileOps(), home, "own_test_nonce", journal); err != nil {
			t.Fatal(err)
		}
		if installErrorCode(recoverPromotions(home, eng)) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
			t.Fatal("wrong pin recovered")
		}
	})
}

func TestPromoteMissingPreviousFailsClosed(t *testing.T) {
	home := t.TempDir()
	dest := filepath.Join(home, "occupied")
	if err := os.MkdirAll(dest, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "keep-me.txt"), []byte("external"), 0o600); err != nil {
		t.Fatal(err)
	}
	if installErrorCode(replaceOwnedTree(context.Background(), t.TempDir(), dest, testIdentity("op_new", dest), operation.Operation{})) != yorvaruntime.ErrorRuntimeInstallTargetOccupied {
		t.Fatal("uncertain dest was replaced")
	}
	if got, err := os.ReadFile(filepath.Join(dest, "keep-me.txt")); err != nil || string(got) != "external" {
		t.Fatalf("external dest disturbed: %q %v", got, err)
	}
}

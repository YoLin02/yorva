package hermes

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type promotionEngine struct {
	ops             atomicFileOps
	rename          func(oldpath, newpath string) error
	afterPrepared   func()
	afterValidate   func()
	afterOldRenamed func()
	afterQuarantine func()
	afterNewRenamed func()
	afterPromoted   func()
	lookup          func(string) (ownershipIdentity, bool)
}

func newPromotionEngine() promotionEngine {
	return promotionEngine{
		ops:    defaultAtomicFileOps(),
		rename: os.Rename,
	}
}

func createCandidateDir(parent, operationID string) (string, error) {
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return "", err
	}
	if err := rejectReparsePoint(parent); err != nil {
		return "", err
	}
	id, err := randomHex(8)
	if err != nil {
		return "", err
	}
	prefix := "hermes-agent.yorva-cand-"
	if operationID != "" {
		prefix += sanitizeFileToken(operationID) + "-"
	}
	path := filepath.Join(parent, prefix+id)
	if err := os.Mkdir(path, 0o700); err != nil {
		return "", err
	}
	if err := rejectReparsePoint(path); err != nil {
		_ = os.RemoveAll(path)
		return "", err
	}
	return path, nil
}

func sanitizeFileToken(value string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, value)
}

func chooseQuarantinePath(parent string) (string, error) {
	id, err := randomHex(8)
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, "hermes-agent.yorva-q-"+id), nil
}

func promoteCandidate(ctx context.Context, eng promotionEngine, home, candidate, dest string, identity ownershipIdentity, previous operation.Operation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := rejectReparsePoint(candidate); err != nil {
		return err
	}
	parent, err := filepath.Abs(filepath.Dir(dest))
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	canonicalDest, err := filepath.Abs(dest)
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	identity.Target = canonicalDest
	record, err := readOwnershipRecord(candidate)
	if err != nil {
		return err
	}
	if err := verifyRecordIdentity(record, identity); err != nil {
		return err
	}
	_, candidateManifest, err := walkInventory(candidate)
	if err != nil {
		return err
	}
	if candidateManifest != record.Manifest {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}

	info, err := os.Lstat(dest)
	previousManifest := ""
	hasLive := false
	if err == nil {
		if !info.IsDir() || isReparsePoint(info) || info.Mode()&os.ModeSymlink != 0 {
			return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
		}
		hasLive = true
		proof := previous
		if proof.ID == "" {
			return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
		}
		if err := ownedPartialIdentity(dest, proof, identity.SourcePin); err != nil {
			return err
		}
		previousManifest, err = inventoryDigest(dest)
		if err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, err)
	}

	if eng.afterValidate != nil {
		eng.afterValidate()
	}

	quarantine := ""
	if hasLive {
		quarantine, err = chooseQuarantinePath(parent)
		if err != nil {
			return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
		}
	}
	journal := promotionJournal{
		OperationID:       identity.OperationID,
		SourcePin:         identity.SourcePin,
		Identity:          record.Identity,
		Target:            canonicalDest,
		Candidate:         candidate,
		Quarantine:        quarantine,
		PreviousManifest:  previousManifest,
		CandidateManifest: candidateManifest,
		State:             promoPrepared,
		CorrelationID:     identity.OperationID,
	}
	if err := writePromotionJournal(eng.ops, home, identity.Nonce, journal); err != nil {
		return err
	}
	if eng.afterPrepared != nil {
		eng.afterPrepared()
	}

	if hasLive {
		if err := ownedPartialIdentity(dest, previous, identity.SourcePin); err != nil {
			return err
		}
		if err := eng.rename(dest, quarantine); err != nil {
			return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
		}
		if eng.afterOldRenamed != nil {
			eng.afterOldRenamed()
		}
		journal.State = promoOldQuarantined
		if err := writePromotionJournal(eng.ops, home, identity.Nonce, journal); err != nil {
			_ = eng.rename(quarantine, dest)
			return err
		}
		if eng.afterQuarantine != nil {
			eng.afterQuarantine()
		}
	}

	if err := eng.rename(candidate, dest); err != nil {
		if hasLive {
			_ = eng.rename(quarantine, dest)
		}
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	if eng.afterNewRenamed != nil {
		eng.afterNewRenamed()
	}
	journal.State = promoNewPromoted
	if err := writePromotionJournal(eng.ops, home, identity.Nonce, journal); err != nil {
		return err
	}
	if eng.afterPromoted != nil {
		eng.afterPromoted()
	}
	if err := requireCurrentOwnedTree(dest, identity); err != nil {
		return err
	}
	journal.State = promoCommitted
	return writePromotionJournal(eng.ops, home, identity.Nonce, journal)
}

func recoverPromotions(home string, eng promotionEngine) error {
	journals, err := listPromotionJournals(home)
	if err != nil {
		return err
	}
	for _, journal := range journals {
		identity, ok := ownershipIdentity{}, false
		if eng.lookup != nil {
			identity, ok = eng.lookup(journal.OperationID)
		}
		if !ok {
			continue
		}
		if err := verifyPromotionJournal(journal, identity); err != nil {
			return err
		}
		if err := recoverOnePromotion(home, journal, identity, eng); err != nil {
			return err
		}
	}
	return nil
}

func recoverOnePromotion(home string, journal promotionJournal, identity ownershipIdentity, eng promotionEngine) error {
	switch journal.State {
	case promoPrepared:
		return recoverPrepared(home, journal, identity, eng)
	case promoOldQuarantined:
		return recoverOldQuarantined(home, journal, identity, eng)
	case promoNewPromoted:
		if err := requireCurrentOwnedTree(journal.Target, identity); err != nil {
			return err
		}
		journal.State = promoCommitted
		return writePromotionJournal(eng.ops, home, identity.Nonce, journal)
	case promoCommitted:
		return nil
	default:
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
}

func recoverPrepared(home string, journal promotionJournal, identity ownershipIdentity, eng promotionEngine) error {
	_, destErr := os.Lstat(journal.Target)
	if destErr != nil && !os.IsNotExist(destErr) {
		return destErr
	}
	if os.IsNotExist(destErr) {
		if journal.Quarantine == "" {
			return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
		}
		return restoreQuarantine(journal, identity)
	}
	if liveStillPrevious(journal.Target, journal, identity) {
		if candidateOwnedBy(journal.Candidate, identity, journal.CandidateManifest) {
			_ = os.RemoveAll(journal.Candidate)
		}
		return nil
	}
	if destMatchesCandidate(journal, identity) {
		journal.State = promoCommitted
		return writePromotionJournal(eng.ops, home, identity.Nonce, journal)
	}
	return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
}

func recoverOldQuarantined(home string, journal promotionJournal, identity ownershipIdentity, eng promotionEngine) error {
	_, destErr := os.Lstat(journal.Target)
	if os.IsNotExist(destErr) {
		return restoreQuarantine(journal, identity)
	}
	if destErr != nil {
		return destErr
	}
	if destMatchesCandidate(journal, identity) {
		journal.State = promoCommitted
		return writePromotionJournal(eng.ops, home, identity.Nonce, journal)
	}
	return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
}

func destMatchesCandidate(journal promotionJournal, identity ownershipIdentity) bool {
	if err := requireCurrentOwnedTree(journal.Target, identity); err != nil {
		return false
	}
	digest, err := inventoryDigest(journal.Target)
	return err == nil && digest == journal.CandidateManifest
}

func liveStillPrevious(target string, journal promotionJournal, identity ownershipIdentity) bool {
	if journal.PreviousManifest == "" {
		_, err := os.Lstat(target)
		return os.IsNotExist(err)
	}
	digest, err := inventoryDigest(target)
	return err == nil && digest == journal.PreviousManifest
}

func candidateOwnedBy(path string, identity ownershipIdentity, manifest string) bool {
	if path == "" {
		return false
	}
	record, err := readOwnershipRecord(path)
	if err != nil {
		return false
	}
	if verifyRecordIdentity(record, identity) != nil {
		return false
	}
	digest, err := inventoryDigest(path)
	return err == nil && digest == manifest && digest == record.Manifest
}

func restoreQuarantine(journal promotionJournal, identity ownershipIdentity) error {
	if journal.Quarantine == "" {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	if err := rejectReparsePoint(journal.Quarantine); err != nil {
		return err
	}
	digest, err := inventoryDigest(journal.Quarantine)
	if err != nil || digest != journal.PreviousManifest {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	_ = identity
	return os.Rename(journal.Quarantine, journal.Target)
}

func replaceOwnedTree(ctx context.Context, staging, installDir string, identity ownershipIdentity, previous operation.Operation) error {
	parent, err := filepath.Abs(filepath.Dir(installDir))
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	candidate, err := createCandidateDir(parent, identity.OperationID)
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	promoted := false
	defer func() {
		if !promoted {
			_ = os.RemoveAll(candidate)
		}
	}()
	if err := copyOwnedTree(ctx, staging, candidate); err != nil {
		return err
	}
	identity.Target = filepath.Clean(installDir)
	if abs, absErr := filepath.Abs(installDir); absErr == nil {
		identity.Target = abs
	}
	if err := writeOwnershipRecord(candidate, identity); err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	home := parent
	if err := promoteCandidate(ctx, newPromotionEngine(), home, candidate, installDir, identity, previous); err != nil {
		return err
	}
	promoted = true
	return nil
}

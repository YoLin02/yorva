package hermes

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const yorvaPartialMarker = ".yorva-phase3-install"

const (
	ownershipSchemaVersion  = 1
	maxOwnershipRecordBytes = 8192
)

type ownershipIdentity struct {
	OperationID string
	RuntimeKind string
	Target      string
	SourcePin   string
	Nonce       string
}

type ownershipRecord struct {
	Schema      int    `json:"schema"`
	OperationID string `json:"operationId"`
	RuntimeKind string `json:"runtimeKind"`
	Target      string `json:"target"`
	SourcePin   string `json:"sourcePin"`
	Identity    string `json:"identity"`
	Manifest    string `json:"manifest"`
	MAC         string `json:"mac"`
}

func validateInstallTarget(home, installDir string, retry bool, previous operation.Operation, expectedPin string) error {
	canonicalHome, err := filepath.Abs(home)
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, err)
	}
	canonicalTarget, err := filepath.Abs(installDir)
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, err)
	}
	if err := rejectReparsePoint(canonicalTarget); err != nil && !os.IsNotExist(err) {
		if _, statErr := os.Lstat(canonicalTarget); statErr == nil || !os.IsNotExist(statErr) {
			if !os.IsNotExist(statErr) {
				return err
			}
		}
	}
	relative, err := filepath.Rel(canonicalHome, canonicalTarget)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	info, err := os.Lstat(canonicalTarget)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, err)
	}
	if !info.IsDir() || isReparsePoint(info) || info.Mode()&os.ModeSymlink != 0 {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	if !retry {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	return ownedPartialIdentity(canonicalTarget, previous, expectedPin)
}

func ownedPartialIdentity(root string, previous operation.Operation, expectedPin string) error {
	if previous.ID == "" || previous.SourcePin == "" || previous.OwnershipNonce == "" || expectedPin == "" {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	if previous.SourcePin != expectedPin {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	if err := rejectReparsePoint(root); err != nil {
		return err
	}
	record, err := readOwnershipRecord(root)
	if err != nil {
		return err
	}
	canonical, err := filepath.Abs(root)
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, err)
	}
	if record.Schema != ownershipSchemaVersion || record.OperationID != previous.ID || record.RuntimeKind != "hermes" || previous.TargetID != "hermes" {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	if record.SourcePin != expectedPin || record.SourcePin != previous.SourcePin {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	if !sameCanonicalPath(record.Target, canonical) {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	digest, err := inventoryDigest(root)
	if err != nil {
		return err
	}
	if digest != record.Manifest {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	if !hmac.Equal([]byte(strings.ToLower(record.MAC)), []byte(ownershipMAC(previous.OwnershipNonce, record))) {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	return nil
}

func readOwnershipRecord(root string) (ownershipRecord, error) {
	path := filepath.Join(root, yorvaPartialMarker)
	if err := rejectReparsePoint(path); err != nil {
		return ownershipRecord{}, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return ownershipRecord{}, installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, err)
	}
	if !info.Mode().IsRegular() || isReparsePoint(info) {
		return ownershipRecord{}, installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	if info.Size() == 0 || info.Size() > maxOwnershipRecordBytes {
		return ownershipRecord{}, installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		return ownershipRecord{}, installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, err)
	}
	var record ownershipRecord
	if err := json.Unmarshal(payload, &record); err != nil || record.Schema != ownershipSchemaVersion || record.OperationID == "" || record.MAC == "" {
		return ownershipRecord{}, installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	return record, nil
}

func writeOwnershipRecord(root string, identity ownershipIdentity) error {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	if err := rejectReparsePoint(root); err != nil {
		return err
	}
	canonical, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	if identity.Target == "" {
		identity.Target = canonical
	}
	digest, err := inventoryDigest(root)
	if err != nil {
		return err
	}
	publicID, err := randomHex(12)
	if err != nil {
		return err
	}
	record := ownershipRecord{
		Schema:      ownershipSchemaVersion,
		OperationID: identity.OperationID,
		RuntimeKind: identity.RuntimeKind,
		Target:      identity.Target,
		SourcePin:   identity.SourcePin,
		Identity:    publicID,
		Manifest:    digest,
	}
	record.MAC = ownershipMAC(identity.Nonce, record)
	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, yorvaPartialMarker), payload, 0o600)
}

func ownershipMAC(nonce string, record ownershipRecord) string {
	mac := hmac.New(sha256.New, []byte(nonce))
	_, _ = mac.Write([]byte(strings.Join([]string{
		hex.EncodeToString([]byte{byte(record.Schema)}),
		record.OperationID,
		record.RuntimeKind,
		filepath.ToSlash(record.Target),
		record.SourcePin,
		record.Identity,
		record.Manifest,
	}, "\n")))
	return hex.EncodeToString(mac.Sum(nil))
}

func inventoryDigest(root string) (string, error) {
	var lines []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := rejectReparsePoint(path); err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Name() == yorvaPartialMarker {
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		sum, err := fileSHA256(path)
		if err != nil {
			return err
		}
		lines = append(lines, filepath.ToSlash(relative)+"\x00"+sum)
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(lines)
	digest := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(digest[:]), nil
}

func sameCanonicalPath(left, right string) bool {
	a, errA := filepath.Abs(left)
	b, errB := filepath.Abs(right)
	if errA != nil || errB != nil {
		return false
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

func replaceOwnedTree(ctx context.Context, staging, installDir string, identity ownershipIdentity, previous operation.Operation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := rejectReparsePoint(staging); err != nil {
		return err
	}
	parent, err := filepath.Abs(filepath.Dir(installDir))
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	if err := rejectReparsePoint(parent); err != nil {
		return err
	}
	tmp, err := uniqueSibling(parent, "hermes-agent.yorva-new-")
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	if err := os.Mkdir(tmp, 0o700); err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	replaced := false
	defer func() {
		if !replaced {
			_ = os.RemoveAll(tmp)
		}
	}()
	if err := copyOwnedTree(ctx, staging, tmp); err != nil {
		return err
	}
	identity.Target = filepath.Clean(installDir)
	if abs, absErr := filepath.Abs(installDir); absErr == nil {
		identity.Target = abs
	}
	if err := writeOwnershipRecord(tmp, identity); err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}

	info, err := os.Lstat(installDir)
	if os.IsNotExist(err) {
		if err := os.Rename(tmp, installDir); err != nil {
			return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
		}
		replaced = true
		return nil
	}
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	if !info.IsDir() || isReparsePoint(info) || info.Mode()&os.ModeSymlink != 0 {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	proof := previous
	if proof.ID == "" {
		proof = operation.Operation{
			ID:             identity.OperationID,
			TargetID:       identity.RuntimeKind,
			SourcePin:      identity.SourcePin,
			OwnershipNonce: identity.Nonce,
		}
	}
	if err := ownedPartialIdentity(installDir, proof, identity.SourcePin); err != nil {
		return err
	}
	backup, err := uniqueSibling(parent, "hermes-agent.yorva-old-")
	if err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	if err := os.Rename(installDir, backup); err != nil {
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	if err := os.Rename(tmp, installDir); err != nil {
		_ = os.Rename(backup, installDir)
		return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
	}
	replaced = true
	_ = os.RemoveAll(backup)
	return nil
}

func copyOwnedTree(ctx context.Context, staging, dest string) error {
	return filepath.WalkDir(staging, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		relative, relErr := filepath.Rel(staging, path)
		if relErr != nil || !pathWithin(staging, path) {
			return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errors.New("materialized tree escaped staging"))
		}
		target := filepath.Join(dest, relative)
		if !pathWithin(dest, target) {
			return installError(yorvaruntime.ErrorRuntimeInstallIntegrityFailed, errors.New("materialized tree escaped install directory"))
		}
		if entry.IsDir() {
			if err := os.MkdirAll(target, 0o700); err != nil {
				return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
			}
			return rejectReparsePoint(target)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return installError(yorvaruntime.ErrorRuntimeInstallStageFailed, err)
		}
		return copyRegularFile(path, target)
	})
}

func uniqueSibling(parent, prefix string) (string, error) {
	id, err := randomHex(8)
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, prefix+id), nil
}

func randomHex(n int) (string, error) {
	payload := make([]byte, n)
	if _, err := rand.Read(payload); err != nil {
		return "", err
	}
	return hex.EncodeToString(payload), nil
}

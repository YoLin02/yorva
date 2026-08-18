package hermes

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
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
	entries, readErr := os.ReadDir(canonicalTarget)
	if readErr == nil && len(entries) == 0 {
		return nil
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

func requireCurrentOwnedTree(root string, identity ownershipIdentity) error {
	if identity.OperationID == "" || identity.Nonce == "" || identity.SourcePin == "" {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	return ownedPartialIdentity(root, operation.Operation{
		ID:             identity.OperationID,
		TargetID:       identity.RuntimeKind,
		SourcePin:      identity.SourcePin,
		OwnershipNonce: identity.Nonce,
	}, identity.SourcePin)
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
	_, digest, err := walkInventory(root)
	return digest, err
}

func sameCanonicalPath(left, right string) bool {
	a, errA := filepath.Abs(left)
	b, errB := filepath.Abs(right)
	if errA != nil || errB != nil {
		return false
	}
	return filepath.Clean(a) == filepath.Clean(b)
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

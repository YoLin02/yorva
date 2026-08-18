package hermes

import (
	"crypto/hmac"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func writeOwnershipRecord(root string, identity ownershipIdentity) error {
	return writeOwnershipRecordWith(defaultAtomicFileOps(), root, identity)
}

func writeOwnershipRecordWith(ops atomicFileOps, root string, identity ownershipIdentity) error {
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
	_, digest, err := walkInventory(root)
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
	if len(payload) == 0 || len(payload) > maxOwnershipRecordBytes {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	dest := filepath.Join(root, yorvaPartialMarker)
	if err := writeAtomicRegularFile(ops, dest, payload); err != nil {
		return err
	}
	verified, err := readOwnershipRecord(root)
	if err != nil {
		return err
	}
	if verified.OperationID != identity.OperationID || verified.SourcePin != identity.SourcePin || verified.Manifest != digest {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	if !hmac.Equal([]byte(strings.ToLower(verified.MAC)), []byte(ownershipMAC(identity.Nonce, verified))) {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	return nil
}

func verifyRecordIdentity(record ownershipRecord, identity ownershipIdentity) error {
	if identity.OperationID == "" || identity.Nonce == "" || identity.SourcePin == "" {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	if record.Schema != ownershipSchemaVersion || record.OperationID != identity.OperationID || record.RuntimeKind != identity.RuntimeKind {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	if record.SourcePin != identity.SourcePin {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	if identity.Target != "" && !sameCanonicalPath(record.Target, identity.Target) {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	if !hmac.Equal([]byte(strings.ToLower(record.MAC)), []byte(ownershipMAC(identity.Nonce, record))) {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	return nil
}

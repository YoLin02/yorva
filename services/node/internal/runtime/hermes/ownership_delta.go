package hermes

import (
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func snapshotOwnedInventory(root string, identity ownershipIdentity) (fileInventory, error) {
	record, err := readOwnershipRecord(root)
	if err != nil {
		return nil, err
	}
	if err := verifyRecordIdentity(record, identity); err != nil {
		return nil, err
	}
	inv, digest, err := walkInventory(root)
	if err != nil {
		return nil, err
	}
	if digest != record.Manifest {
		return nil, installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	return inv, nil
}

func applyAuthenticatedStageDelta(ops atomicFileOps, root string, identity ownershipIdentity, stage string, before, produced fileInventory) error {
	current, currentDigest, err := walkInventory(root)
	if err != nil {
		return err
	}
	if currentDigest != digestInventory(produced) {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	_ = current
	delta := diffInventory(before, produced)
	if !acceptStageDelta(stage, delta) {
		return installError(yorvaruntime.ErrorRuntimeInstallTargetOccupied, errReparsePoint)
	}
	if delta.empty() {
		return nil
	}
	return writeOwnershipRecordWith(ops, root, identity)
}

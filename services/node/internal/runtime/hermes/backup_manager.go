package hermes

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes/backupmanagement"
)

const backupDeviceKeyReference = "backup-device-key-v1"

type BackupIndexEntry struct {
	ID, RuntimeInstallationID, FormatVersion, RuntimeVersion string
	ArtifactPath, ChecksumSHA256, KeyRef                     string
	SizeBytes                                                int64
	CreatedAt, VerifiedAt                                    time.Time
}

type RuntimeBackupManager struct {
	destinations  *backupmanagement.DestinationRegistry
	dataDir       string
	useDeviceKey  func(context.Context, string, func([]byte) error) error
	insertIndex   func(context.Context, BackupIndexEntry) error
	getIndex      func(context.Context, string) (BackupIndexEntry, error)
	deleteIndex   func(context.Context, string) error
	ensureStopped func(context.Context, yorvaruntime.Installation) error
	postcheck     func(context.Context, yorvaruntime.Installation) error
	now           func() time.Time
}

func NewRuntimeBackupManager(
	destinations *backupmanagement.DestinationRegistry,
	dataDir string,
	useDeviceKey func(context.Context, string, func([]byte) error) error,
	insertIndex func(context.Context, BackupIndexEntry) error,
	getIndex func(context.Context, string) (BackupIndexEntry, error),
	deleteIndex func(context.Context, string) error,
) *RuntimeBackupManager {
	return &RuntimeBackupManager{
		destinations: destinations, dataDir: dataDir, useDeviceKey: useDeviceKey,
		insertIndex: insertIndex, getIndex: getIndex, deleteIndex: deleteIndex,
		ensureStopped: ensureAllHermesProfilesStopped,
		postcheck: func(ctx context.Context, installation yorvaruntime.Installation) error {
			_, err := ListProfiles(ctx, installation.Path)
			return err
		},
		now: func() time.Time { return time.Now().UTC() },
	}
}

func (m *RuntimeBackupManager) CreateBackup(ctx context.Context, installation yorvaruntime.Installation, request yorvaruntime.BackupCreateRequest, progress yorvaruntime.ProgressSink) (yorvaruntime.Backup, error) {
	if m == nil || m.destinations == nil || m.useDeviceKey == nil || m.insertIndex == nil || m.ensureStopped == nil ||
		request.Validate() != nil || installation.RuntimeKind != Kind ||
		installation.Version != backupmanagement.QualifiedSnapshotRuntimeVersion || installation.Path == "" ||
		!filepath.IsAbs(installation.Path) || installation.SupportState != yorvaruntime.DiscoverySupported {
		return yorvaruntime.Backup{}, yorvaruntime.ErrInvalidManagementContract
	}
	consumed, err := m.destinations.Consume(request.DestinationRef, string(Kind), request.OperationID)
	if err != nil {
		return yorvaruntime.Backup{}, err
	}
	if progress != nil {
		progress.Report(yorvaruntime.ProgressUpdate{Stage: "backup.preflight"})
	}
	if err := m.ensureStopped(ctx, installation); err != nil {
		return yorvaruntime.Backup{}, err
	}

	createdAt := m.now().UTC().Truncate(time.Second)
	backupID, err := newBackupID()
	if err != nil {
		return yorvaruntime.Backup{}, err
	}
	stagingRoot := filepath.Join(m.dataDir, "backup-staging")
	if err := os.MkdirAll(stagingRoot, 0o700); err != nil {
		return yorvaruntime.Backup{}, err
	}
	operationRoot, err := os.MkdirTemp(stagingRoot, "backup-")
	if err != nil {
		return yorvaruntime.Backup{}, err
	}
	defer os.RemoveAll(operationRoot)
	payload, err := os.OpenFile(filepath.Join(operationRoot, "payload.zip"), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		return yorvaruntime.Backup{}, err
	}
	defer payload.Close()
	container, err := os.OpenFile(filepath.Join(operationRoot, "container.zip"), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		return yorvaruntime.Backup{}, err
	}
	defer container.Close()

	identity := backupmanagement.InstallationIdentity{State: backupmanagement.InstallationUnmanaged}
	if active, ok := observeManagedActive(ctx, installation); ok {
		identity = backupmanagement.InstallationIdentity{
			State: backupmanagement.InstallationManaged, GenerationID: active.GenerationID, SourcePin: active.SourcePin,
		}
	}
	snapshot, err := backupmanagement.BuildCanonicalHermesRuntimeSnapshot(ctx, backupmanagement.SnapshotDescriptor{
		BackupID: backupID, CreatedAt: createdAt.Format(time.RFC3339Nano), RuntimeVersion: installation.Version,
		Installation: identity,
	}, true, backupmanagement.SnapshotStaging{Payload: payload, Container: container}, backupmanagement.DefaultLimits())
	if err != nil {
		return yorvaruntime.Backup{}, err
	}
	if snapshot.Artifact.Metadata.BackupID != backupID {
		return yorvaruntime.Backup{}, errors.New("backup snapshot identity mismatch")
	}
	containerInfo, err := container.Stat()
	if err != nil {
		return yorvaruntime.Backup{}, err
	}
	if progress != nil {
		progress.Report(yorvaruntime.ProgressUpdate{Stage: "backup.encrypt"})
	}
	var published backupmanagement.PublishedArtifactMetadata
	err = m.useDeviceKey(ctx, backupDeviceKeyReference, func(identity []byte) error {
		recipient, deriveErr := backupmanagement.RecipientForIdentity(identity)
		if deriveErr != nil {
			return deriveErr
		}
		credential, credentialErr := backupmanagement.NewDeviceCreateCredential(recipient, identity)
		if credentialErr != nil {
			return credentialErr
		}
		published, deriveErr = backupmanagement.PublishVerifiedArtifact(ctx, backupmanagement.PublicationRequest{
			Destination: consumed.Destination, Artifact: container, ArtifactSize: containerInfo.Size(),
			Credential: credential, Limits: backupmanagement.DefaultLimits(),
		})
		return deriveErr
	})
	if err != nil || published.State != backupmanagement.PublicationVerified {
		if err == nil {
			err = errors.New("backup publication did not verify")
		}
		return yorvaruntime.Backup{}, err
	}
	verifiedAt := m.now()
	indexEntry := BackupIndexEntry{
		ID: backupID, RuntimeInstallationID: request.RuntimeInstallationID, FormatVersion: backupmanagement.FormatVersion,
		RuntimeVersion: installation.Version, ArtifactPath: consumed.IndexPath,
		SizeBytes: published.EncryptedSizeBytes, ChecksumSHA256: published.ChecksumSHA256,
		KeyRef: backupDeviceKeyReference, CreatedAt: createdAt, VerifiedAt: verifiedAt,
	}
	if err := m.insertIndex(ctx, indexEntry); err != nil {
		if verifyErr := verifyIndexedArtifact(ctx, indexEntry); verifyErr != nil {
			return yorvaruntime.Backup{}, errors.Join(err, errors.New("published backup requires manual reconciliation"))
		}
		if removeErr := os.Remove(indexEntry.ArtifactPath); removeErr != nil {
			return yorvaruntime.Backup{}, errors.Join(err, removeErr)
		}
		return yorvaruntime.Backup{}, err
	}
	return yorvaruntime.Backup{
		ID: backupID, State: yorvaruntime.BackupAvailable, FormatVersion: backupmanagement.FormatVersion,
		RuntimeVersion: installation.Version, SizeBytes: published.EncryptedSizeBytes,
		ChecksumSHA256: published.ChecksumSHA256, CreatedAt: createdAt, VerifiedAt: verifiedAt,
		KeyMode: yorvaruntime.BackupKeyDevice,
	}, nil
}

func (m *RuntimeBackupManager) DeleteBackup(ctx context.Context, installation yorvaruntime.Installation, backupID string, progress yorvaruntime.ProgressSink) error {
	if m == nil || m.getIndex == nil || m.deleteIndex == nil || installation.RuntimeKind != Kind ||
		(yorvaruntime.Backup{ID: backupID, State: yorvaruntime.BackupFailed}).Validate() != nil {
		return yorvaruntime.ErrInvalidManagementContract
	}
	entry, err := m.getIndex(ctx, backupID)
	if err != nil {
		return err
	}
	if err := verifyIndexedArtifact(ctx, entry); err != nil {
		return err
	}
	if progress != nil {
		progress.Report(yorvaruntime.ProgressUpdate{Stage: "backup.delete"})
	}
	if err := os.Remove(entry.ArtifactPath); err != nil {
		return err
	}
	return m.deleteIndex(ctx, backupID)
}

func (m *RuntimeBackupManager) RestoreBackup(ctx context.Context, installation yorvaruntime.Installation, request yorvaruntime.BackupRestoreRequest, progress yorvaruntime.ProgressSink) (yorvaruntime.RestoreResult, error) {
	if m == nil || m.getIndex == nil || m.useDeviceKey == nil || m.ensureStopped == nil || m.postcheck == nil || request.Validate() != nil ||
		installation.RuntimeKind != Kind || installation.Version != backupmanagement.QualifiedSnapshotRuntimeVersion {
		return yorvaruntime.RestoreResult{}, yorvaruntime.ErrInvalidManagementContract
	}
	entry, err := m.getIndex(ctx, request.BackupID)
	if err != nil {
		return yorvaruntime.RestoreResult{}, err
	}
	if entry.RuntimeVersion != installation.Version || entry.KeyRef == "" {
		return yorvaruntime.RestoreResult{}, yorvaruntime.ErrInvalidManagementContract
	}
	if err := m.ensureStopped(ctx, installation); err != nil {
		return yorvaruntime.RestoreResult{}, err
	}
	if err := verifyIndexedArtifact(ctx, entry); err != nil {
		return yorvaruntime.RestoreResult{}, err
	}
	root, err := backupmanagement.CanonicalHermesRuntimeRoot()
	if err != nil {
		return yorvaruntime.RestoreResult{}, err
	}
	txn, err := os.MkdirTemp(filepath.Dir(root), ".yorva-restore-txn-")
	if err != nil {
		return yorvaruntime.RestoreResult{}, err
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.RemoveAll(txn)
		}
	}()
	if err := writeRestorePhase(txn, "preparing"); err != nil {
		return yorvaruntime.RestoreResult{}, err
	}
	container, err := os.OpenFile(filepath.Join(txn, "container.zip"), os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		return yorvaruntime.RestoreResult{}, err
	}
	artifact, err := os.Open(entry.ArtifactPath)
	if err != nil {
		_ = container.Close()
		return yorvaruntime.RestoreResult{}, err
	}
	var verified backupmanagement.EncryptedArtifactVerification
	err = m.useDeviceKey(ctx, entry.KeyRef, func(identity []byte) error {
		var decryptErr error
		verified, decryptErr = backupmanagement.DecryptToVerifiedFileWithX25519(ctx, artifact, entry.SizeBytes, identity, container, backupmanagement.DefaultLimits())
		return decryptErr
	})
	artifactCloseErr := artifact.Close()
	if err != nil || artifactCloseErr != nil || verified.Artifact.Metadata.BackupID != request.BackupID || verified.Artifact.Metadata.RuntimeVersion != installation.Version {
		_ = container.Close()
		if err == nil {
			err = errors.New("backup identity does not match restore request")
		}
		return yorvaruntime.RestoreResult{}, err
	}
	candidate := filepath.Join(txn, "candidate")
	if err := os.Mkdir(candidate, 0o700); err != nil {
		_ = container.Close()
		return yorvaruntime.RestoreResult{}, err
	}
	if _, err := backupmanagement.ExtractVerifiedRuntimeSnapshot(ctx, container, verified.PlaintextSizeBytes, candidate, backupmanagement.DefaultLimits()); err != nil {
		_ = container.Close()
		return yorvaruntime.RestoreResult{}, err
	}
	if err := container.Close(); err != nil {
		return yorvaruntime.RestoreResult{}, err
	}
	if progress != nil {
		progress.Report(yorvaruntime.ProgressUpdate{Stage: "restore.apply"})
	}
	previous := filepath.Join(txn, "previous")
	if err := writeRestorePhase(txn, "prepared"); err != nil {
		return yorvaruntime.RestoreResult{}, err
	}
	if err := os.Rename(root, previous); err != nil {
		return yorvaruntime.RestoreResult{}, err
	}
	cleanup = false
	if err := writeRestorePhase(txn, "current-moved"); err != nil {
		_ = os.Rename(previous, root)
		return yorvaruntime.RestoreResult{State: yorvaruntime.RestoreRecoveryRequired, ObservedAt: m.now()}, err
	}
	if err := os.Rename(candidate, root); err != nil {
		if rollbackErr := os.Rename(previous, root); rollbackErr != nil {
			return yorvaruntime.RestoreResult{State: yorvaruntime.RestoreRecoveryRequired, ObservedAt: m.now()}, err
		}
		cleanup = true
		return yorvaruntime.RestoreResult{State: yorvaruntime.RestoreRolledBack, ObservedAt: m.now()}, err
	}
	if err := writeRestorePhase(txn, "activated"); err != nil {
		return m.rollbackRestore(root, previous, txn, err)
	}
	if err := m.postcheck(ctx, installation); err != nil {
		return m.rollbackRestore(root, previous, txn, err)
	}
	if err := os.RemoveAll(previous); err != nil {
		return yorvaruntime.RestoreResult{State: yorvaruntime.RestoreRecoveryRequired, ObservedAt: m.now()}, err
	}
	cleanup = true
	return yorvaruntime.RestoreResult{State: yorvaruntime.RestoreSucceeded, ObservedAt: m.now()}, nil
}

func (m *RuntimeBackupManager) rollbackRestore(root, previous, txn string, cause error) (yorvaruntime.RestoreResult, error) {
	failed := filepath.Join(txn, "failed")
	if err := os.Rename(root, failed); err != nil {
		return yorvaruntime.RestoreResult{State: yorvaruntime.RestoreRecoveryRequired, ObservedAt: m.now()}, cause
	}
	if err := os.Rename(previous, root); err != nil {
		return yorvaruntime.RestoreResult{State: yorvaruntime.RestoreRecoveryRequired, ObservedAt: m.now()}, cause
	}
	_ = os.RemoveAll(failed)
	_ = os.RemoveAll(txn)
	return yorvaruntime.RestoreResult{State: yorvaruntime.RestoreRolledBack, ObservedAt: m.now()}, cause
}

func writeRestorePhase(txn, phase string) error {
	file, err := os.OpenFile(filepath.Join(txn, "phase"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.WriteString(phase + "\n"); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

// RecoverInterruptedRestores restores the pre-Restore tree for any transaction
// that did not reach a verified terminal state before daemon exit.
func RecoverInterruptedRestores(ctx context.Context) error {
	root, err := backupmanagement.CanonicalHermesRuntimeRootPath()
	if err != nil {
		return err
	}
	parent := filepath.Dir(root)
	entries, err := os.ReadDir(parent)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), ".yorva-restore-txn-") {
			continue
		}
		txn := filepath.Join(parent, entry.Name())
		phaseBytes, err := os.ReadFile(filepath.Join(txn, "phase"))
		if err != nil || len(phaseBytes) > 32 {
			return errors.New("interrupted Restore requires manual recovery")
		}
		phase := strings.TrimSpace(string(phaseBytes))
		previous := filepath.Join(txn, "previous")
		rootExists, err := restoreDirectoryExists(root)
		if err != nil {
			return errors.New("interrupted Restore requires manual recovery")
		}
		previousExists, err := restoreDirectoryExists(previous)
		if err != nil {
			return errors.New("interrupted Restore requires manual recovery")
		}
		switch phase {
		case "preparing":
			if !rootExists || previousExists {
				return errors.New("interrupted Restore requires manual recovery")
			}
		case "prepared":
			switch {
			case rootExists && !previousExists:
				// The active tree was never moved.
			case !rootExists && previousExists:
				if err := os.Rename(previous, root); err != nil {
					return errors.New("interrupted Restore requires manual recovery")
				}
			default:
				return errors.New("interrupted Restore requires manual recovery")
			}
		case "current-moved":
			if !previousExists {
				return errors.New("interrupted Restore requires manual recovery")
			}
			if rootExists {
				if err := rollbackInterruptedRestore(root, previous, txn); err != nil {
					return err
				}
			} else if err := os.Rename(previous, root); err != nil {
				return errors.New("interrupted Restore requires manual recovery")
			}
		case "activated":
			if !rootExists {
				return errors.New("interrupted Restore requires manual recovery")
			}
			if previousExists {
				if err := rollbackInterruptedRestore(root, previous, txn); err != nil {
					return err
				}
			}
		default:
			return errors.New("interrupted Restore requires manual recovery")
		}
		if err := os.RemoveAll(txn); err != nil {
			return err
		}
	}
	return nil
}

func restoreDirectoryExists(path string) (bool, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return false, errors.New("unsafe Restore recovery path")
	}
	return true, nil
}

func rollbackInterruptedRestore(root, previous, txn string) error {
	failed := filepath.Join(txn, "failed-recovery")
	if err := os.Rename(root, failed); err != nil {
		return errors.New("interrupted Restore requires manual recovery")
	}
	if err := os.Rename(previous, root); err != nil {
		_ = os.Rename(failed, root)
		return errors.New("interrupted Restore requires manual recovery")
	}
	return nil
}

func verifyIndexedArtifact(ctx context.Context, entry BackupIndexEntry) error {
	if entry.ArtifactPath == "" || !filepath.IsAbs(entry.ArtifactPath) ||
		!strings.HasSuffix(entry.ArtifactPath, ".yorva-backup.age") || entry.SizeBytes <= 0 {
		return yorvaruntime.ErrInvalidManagementContract
	}
	info, err := os.Lstat(entry.ArtifactPath)
	if err != nil || !info.Mode().IsRegular() || info.Size() != entry.SizeBytes {
		return errors.New("indexed backup artifact is unavailable")
	}
	file, err := os.Open(entry.ArtifactPath)
	if err != nil {
		return err
	}
	defer file.Close()
	hash := sha256.New()
	written, err := io.Copy(hash, &contextReader{ctx: ctx, reader: file})
	if err != nil || written != entry.SizeBytes || hex.EncodeToString(hash.Sum(nil)) != entry.ChecksumSHA256 {
		return errors.New("indexed backup artifact changed")
	}
	return nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(buffer []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(buffer)
}

func ensureAllHermesProfilesStopped(ctx context.Context, installation yorvaruntime.Installation) error {
	profiles, err := ListProfiles(ctx, installation.Path)
	if err != nil {
		return err
	}
	lifecycle := NewLifecycleManager()
	for _, profile := range profiles {
		status, err := lifecycle.Status(ctx, yorvaruntime.LifecycleInstallation{Executable: installation.Path, Version: installation.Version}, profile.NativeID)
		if err != nil || status.State != yorvaruntime.LifecycleStopped {
			return fmt.Errorf("Hermes Runtime must be stopped before backup")
		}
	}
	return nil
}

func newBackupID() (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyz234567"
	var random [22]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	body := make([]byte, len(random))
	for index, value := range random {
		body[index] = alphabet[value&31]
	}
	return "backup_" + string(body), nil
}

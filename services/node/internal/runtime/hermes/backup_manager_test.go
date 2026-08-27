package hermes

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes/backupmanagement"
)

func TestRuntimeBackupManagerCreateAndRestoreRoundTrip(t *testing.T) {
	localAppData := t.TempDir()
	t.Setenv("LOCALAPPDATA", localAppData)
	runtimeRoot := filepath.Join(localAppData, "hermes")
	writeRestoreTestTree(t, runtimeRoot, "before-backup")

	dataDir := t.TempDir()
	_, identity, err := backupmanagement.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	var indexed BackupIndexEntry
	manager := NewRuntimeBackupManager(
		nil,
		dataDir,
		func(_ context.Context, reference string, use func([]byte) error) error {
			if reference != backupDeviceKeyReference {
				t.Fatalf("key reference = %q", reference)
			}
			return use(identity)
		},
		func(_ context.Context, entry BackupIndexEntry) error { indexed = entry; return nil },
		func(_ context.Context, backupID string) (BackupIndexEntry, error) {
			if indexed.ID != backupID {
				return BackupIndexEntry{}, errors.New("backup not found")
			}
			return indexed, nil
		},
		func(context.Context, string) error { return nil },
	)
	manager.ensureStopped = func(context.Context, yorvaruntime.Installation) error { return nil }
	manager.postcheck = func(context.Context, yorvaruntime.Installation) error { return nil }
	installation := yorvaruntime.Installation{
		RuntimeKind: Kind, Path: filepath.Join(t.TempDir(), "hermes.exe"),
		Version: backupmanagement.QualifiedSnapshotRuntimeVersion, SupportState: yorvaruntime.DiscoverySupported,
	}
	created, err := manager.CreateBackup(context.Background(), installation, yorvaruntime.BackupCreateRequest{
		OperationID: "op_backup_roundtrip", RuntimeInstallationID: "rtinst_roundtrip",
	}, nil)
	if err != nil || created.State != yorvaruntime.BackupAvailable || indexed.ID != created.ID {
		t.Fatalf("CreateBackup() = %#v, index %#v, %v", created, indexed, err)
	}
	if filepath.Dir(indexed.ArtifactPath) != filepath.Join(dataDir, "backups") || filepath.Base(indexed.ArtifactPath) != created.ID+".yorva-backup.age" {
		t.Fatalf("system backup destination = %q", indexed.ArtifactPath)
	}
	if info, statErr := os.Stat(indexed.ArtifactPath); statErr != nil || !info.Mode().IsRegular() {
		t.Fatalf("system backup artifact = %#v, %v", info, statErr)
	}
	if err := os.WriteFile(filepath.Join(runtimeRoot, "state.txt"), []byte("after-backup"), 0o600); err != nil {
		t.Fatal(err)
	}

	restored, err := manager.RestoreBackup(context.Background(), installation, yorvaruntime.BackupRestoreRequest{BackupID: created.ID}, nil)
	if err != nil || restored.State != yorvaruntime.RestoreSucceeded {
		t.Fatalf("RestoreBackup() = %#v, %v", restored, err)
	}
	contents, err := os.ReadFile(filepath.Join(runtimeRoot, "state.txt"))
	if err != nil || string(contents) != "before-backup" {
		t.Fatalf("restored contents = %q, %v", contents, err)
	}
}

func TestRuntimeBackupManagerRemovesPublishedArtifactWhenIndexInsertFails(t *testing.T) {
	localAppData := t.TempDir()
	t.Setenv("LOCALAPPDATA", localAppData)
	writeRestoreTestTree(t, filepath.Join(localAppData, "hermes"), "backup-data")
	destination := filepath.Join(t.TempDir(), "index-failure.yorva-backup.age")
	registry, err := backupmanagement.NewDestinationRegistry("test-session")
	if err != nil {
		t.Fatal(err)
	}
	destinationRef := strings.Repeat("b", 43)
	if err := registry.Grant(destinationRef, "hermes", destination); err != nil {
		t.Fatal(err)
	}
	_, identity, err := backupmanagement.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	manager := NewRuntimeBackupManager(
		registry,
		t.TempDir(),
		func(_ context.Context, _ string, use func([]byte) error) error { return use(identity) },
		func(context.Context, BackupIndexEntry) error { return errors.New("index unavailable") },
		nil,
		nil,
	)
	manager.ensureStopped = func(context.Context, yorvaruntime.Installation) error { return nil }
	installation := yorvaruntime.Installation{
		RuntimeKind: Kind, Path: filepath.Join(t.TempDir(), "hermes.exe"),
		Version: backupmanagement.QualifiedSnapshotRuntimeVersion, SupportState: yorvaruntime.DiscoverySupported,
	}
	if _, err := manager.CreateBackup(context.Background(), installation, yorvaruntime.BackupCreateRequest{
		DestinationRef: destinationRef, OperationID: "op_backup_index_failure", RuntimeInstallationID: "rtinst_index_failure",
	}, nil); err == nil {
		t.Fatal("index failure was accepted")
	}
	if _, err := os.Lstat(destination); !os.IsNotExist(err) {
		t.Fatalf("unindexed published artifact remains: %v", err)
	}
}

func TestRecoverInterruptedRestoresClosesAtomicRenameWindows(t *testing.T) {
	tests := []struct {
		name         string
		phase        string
		rootPresent  bool
		previous     bool
		wantContents string
	}{
		{name: "prepared after old tree moved", phase: "prepared", previous: true, wantContents: "old"},
		{name: "current moved after candidate activated", phase: "current-moved", rootPresent: true, previous: true, wantContents: "old"},
		{name: "activated before old tree cleanup", phase: "activated", rootPresent: true, previous: true, wantContents: "old"},
		{name: "activated after old tree cleanup", phase: "activated", rootPresent: true, wantContents: "new"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			localAppData := t.TempDir()
			t.Setenv("LOCALAPPDATA", localAppData)
			root := filepath.Join(localAppData, "hermes")
			txn := filepath.Join(localAppData, ".yorva-restore-txn-test")
			if err := os.MkdirAll(txn, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(txn, "phase"), []byte(test.phase+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if test.rootPresent {
				writeRestoreTestTree(t, root, "new")
			}
			if test.previous {
				writeRestoreTestTree(t, filepath.Join(txn, "previous"), "old")
			}

			if err := RecoverInterruptedRestores(context.Background()); err != nil {
				t.Fatalf("RecoverInterruptedRestores() error = %v", err)
			}
			contents, err := os.ReadFile(filepath.Join(root, "state.txt"))
			if err != nil || string(contents) != test.wantContents {
				t.Fatalf("recovered state = %q, %v", contents, err)
			}
			if _, err := os.Lstat(txn); !os.IsNotExist(err) {
				t.Fatalf("transaction still exists: %v", err)
			}
		})
	}
}

func TestRecoverInterruptedRestoresRejectsAmbiguousState(t *testing.T) {
	localAppData := t.TempDir()
	t.Setenv("LOCALAPPDATA", localAppData)
	writeRestoreTestTree(t, filepath.Join(localAppData, "hermes"), "current")
	txn := filepath.Join(localAppData, ".yorva-restore-txn-test")
	writeRestoreTestTree(t, filepath.Join(txn, "previous"), "previous")
	if err := os.WriteFile(filepath.Join(txn, "phase"), []byte("prepared\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := RecoverInterruptedRestores(context.Background()); err == nil {
		t.Fatal("ambiguous Restore state was accepted")
	}
	contents, err := os.ReadFile(filepath.Join(localAppData, "hermes", "state.txt"))
	if err != nil || string(contents) != "current" {
		t.Fatalf("ambiguous recovery mutated active state = %q, %v", contents, err)
	}
}

func writeRestoreTestTree(t *testing.T, root, contents string) {
	t.Helper()
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "state.txt"), []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

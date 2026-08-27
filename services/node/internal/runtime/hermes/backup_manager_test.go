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

func TestRuntimeBackupManagerDisposableRestoreLifecycle(t *testing.T) {
	t.Run("encrypted restore succeeds and reads back before cleanup", func(t *testing.T) {
		manager, installation, index, runtimeRoot, localAppData := newDisposableRestoreFixture(t)
		stoppedChecks := 0
		manager.ensureStopped = func(context.Context, yorvaruntime.Installation) error {
			stoppedChecks++
			return nil
		}
		postcheckCalls := 0
		manager.postcheck = func(context.Context, yorvaruntime.Installation) error {
			postcheckCalls++
			assertRestoreTestState(t, runtimeRoot, "before-backup")
			return nil
		}

		created := createDisposableRestoreBackup(t, manager, installation)
		artifactBytes, err := os.ReadFile(index.entry.ArtifactPath)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(artifactBytes), "before-backup") {
			t.Fatal("encrypted artifact contains the plaintext state marker")
		}
		writeRestoreTestTree(t, runtimeRoot, "after-backup")

		restored, err := manager.RestoreBackup(context.Background(), installation, yorvaruntime.BackupRestoreRequest{BackupID: created.ID}, nil)
		if err != nil || restored.State != yorvaruntime.RestoreSucceeded {
			t.Fatalf("RestoreBackup() = %#v, %v", restored, err)
		}
		assertRestoreTestState(t, runtimeRoot, "before-backup")
		assertNoRestoreTransactions(t, localAppData)
		if stoppedChecks != 2 || postcheckCalls != 1 {
			t.Fatalf("precondition checks = %d, postchecks = %d", stoppedChecks, postcheckCalls)
		}

		artifactPath := index.entry.ArtifactPath
		if err := manager.DeleteBackup(context.Background(), installation, created.ID, nil); err != nil {
			t.Fatalf("DeleteBackup() error = %v", err)
		}
		if index.present {
			t.Fatal("deleted backup remains in authoritative index")
		}
		if _, err := os.Lstat(artifactPath); !os.IsNotExist(err) {
			t.Fatalf("deleted backup artifact remains: %v", err)
		}
	})

	t.Run("failed postcheck rolls back the previous tree", func(t *testing.T) {
		manager, installation, _, runtimeRoot, localAppData := newDisposableRestoreFixture(t)
		created := createDisposableRestoreBackup(t, manager, installation)
		writeRestoreTestTree(t, runtimeRoot, "after-backup")
		postcheckSawCandidate := false
		manager.postcheck = func(context.Context, yorvaruntime.Installation) error {
			assertRestoreTestState(t, runtimeRoot, "before-backup")
			postcheckSawCandidate = true
			return errors.New("forced disposable postcheck failure")
		}

		restored, err := manager.RestoreBackup(context.Background(), installation, yorvaruntime.BackupRestoreRequest{BackupID: created.ID}, nil)
		if err == nil || restored.State != yorvaruntime.RestoreRolledBack {
			t.Fatalf("RestoreBackup() = %#v, %v", restored, err)
		}
		if !postcheckSawCandidate {
			t.Fatal("restore candidate was not authoritatively checked")
		}
		assertRestoreTestState(t, runtimeRoot, "after-backup")
		assertNoRestoreTransactions(t, localAppData)
	})

	t.Run("tampered artifact is rejected before runtime mutation", func(t *testing.T) {
		manager, installation, index, runtimeRoot, localAppData := newDisposableRestoreFixture(t)
		created := createDisposableRestoreBackup(t, manager, installation)
		writeRestoreTestTree(t, runtimeRoot, "after-backup")
		artifact, err := os.OpenFile(index.entry.ArtifactPath, os.O_APPEND|os.O_WRONLY, 0)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := artifact.Write([]byte("tampered")); err != nil {
			_ = artifact.Close()
			t.Fatal(err)
		}
		if err := artifact.Close(); err != nil {
			t.Fatal(err)
		}
		postcheckCalled := false
		manager.postcheck = func(context.Context, yorvaruntime.Installation) error {
			postcheckCalled = true
			return nil
		}

		restored, err := manager.RestoreBackup(context.Background(), installation, yorvaruntime.BackupRestoreRequest{BackupID: created.ID}, nil)
		if err == nil || restored.State == yorvaruntime.RestoreSucceeded {
			t.Fatalf("tampered RestoreBackup() = %#v, %v", restored, err)
		}
		if postcheckCalled {
			t.Fatal("tampered artifact reached the postcheck")
		}
		assertRestoreTestState(t, runtimeRoot, "after-backup")
		assertNoRestoreTransactions(t, localAppData)
	})
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

type disposableRestoreIndex struct {
	entry   BackupIndexEntry
	present bool
}

func newDisposableRestoreFixture(t *testing.T) (*RuntimeBackupManager, yorvaruntime.Installation, *disposableRestoreIndex, string, string) {
	t.Helper()
	localAppData := t.TempDir()
	t.Setenv("LOCALAPPDATA", localAppData)
	runtimeRoot := filepath.Join(localAppData, "hermes")
	writeRestoreTestTree(t, runtimeRoot, "before-backup")
	_, identity, err := backupmanagement.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	index := &disposableRestoreIndex{}
	manager := NewRuntimeBackupManager(
		nil,
		t.TempDir(),
		func(_ context.Context, reference string, use func([]byte) error) error {
			if reference != backupDeviceKeyReference {
				t.Fatalf("key reference = %q", reference)
			}
			return use(identity)
		},
		func(_ context.Context, entry BackupIndexEntry) error {
			index.entry, index.present = entry, true
			return nil
		},
		func(_ context.Context, backupID string) (BackupIndexEntry, error) {
			if !index.present || index.entry.ID != backupID {
				return BackupIndexEntry{}, errors.New("backup not found")
			}
			return index.entry, nil
		},
		func(_ context.Context, backupID string) error {
			if !index.present || index.entry.ID != backupID {
				return errors.New("backup not found")
			}
			index.present = false
			return nil
		},
	)
	manager.ensureStopped = func(context.Context, yorvaruntime.Installation) error { return nil }
	manager.postcheck = func(context.Context, yorvaruntime.Installation) error { return nil }
	installation := yorvaruntime.Installation{
		RuntimeKind:  Kind,
		Path:         filepath.Join(t.TempDir(), "hermes.exe"),
		Version:      backupmanagement.QualifiedSnapshotRuntimeVersion,
		SupportState: yorvaruntime.DiscoverySupported,
	}
	return manager, installation, index, runtimeRoot, localAppData
}

func createDisposableRestoreBackup(t *testing.T, manager *RuntimeBackupManager, installation yorvaruntime.Installation) yorvaruntime.Backup {
	t.Helper()
	created, err := manager.CreateBackup(context.Background(), installation, yorvaruntime.BackupCreateRequest{
		OperationID: "op_disposable_restore", RuntimeInstallationID: "rtinst_disposable_restore",
	}, nil)
	if err != nil || created.State != yorvaruntime.BackupAvailable {
		t.Fatalf("CreateBackup() = %#v, %v", created, err)
	}
	return created
}

func assertRestoreTestState(t *testing.T, runtimeRoot, want string) {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(runtimeRoot, "state.txt"))
	if err != nil || string(contents) != want {
		t.Fatalf("runtime state = %q, %v; want %q", contents, err, want)
	}
}

func assertNoRestoreTransactions(t *testing.T, localAppData string) {
	t.Helper()
	entries, err := os.ReadDir(localAppData)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".yorva-restore-txn-") {
			t.Fatalf("Restore transaction was not cleaned up: %s", entry.Name())
		}
	}
}

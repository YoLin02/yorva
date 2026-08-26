package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type b6RuntimeTargetResolver struct {
	target    RuntimeManagementTarget
	err       error
	gotID     string
	callCount int
}

func (f *b6RuntimeTargetResolver) ResolveRuntimeManagementTarget(_ context.Context, runtimeID string) (RuntimeManagementTarget, error) {
	f.gotID = runtimeID
	f.callCount++
	return f.target, f.err
}

type b6BackupReader struct {
	backups     []yorvaruntime.Backup
	inspected   yorvaruntime.Backup
	listErr     error
	inspectErr  error
	gotInstall  yorvaruntime.Installation
	gotBackupID string
}

func (f *b6BackupReader) ListBackups(_ context.Context, installation yorvaruntime.Installation) ([]yorvaruntime.Backup, error) {
	f.gotInstall = installation
	return f.backups, f.listErr
}

func (f *b6BackupReader) GetBackup(_ context.Context, installation yorvaruntime.Installation, backupID string) (yorvaruntime.Backup, error) {
	f.gotInstall = installation
	f.gotBackupID = backupID
	return f.inspected, f.inspectErr
}

type b6BackupManager struct {
	createCalls int
	deleteCalls int
	created     yorvaruntime.Backup
	createErr   error
	deleteErr   error
	gotRequest  yorvaruntime.BackupCreateRequest
	gotDeleteID string
}

func (f *b6BackupManager) CreateBackup(_ context.Context, _ yorvaruntime.Installation, request yorvaruntime.BackupCreateRequest, _ yorvaruntime.ProgressSink) (yorvaruntime.Backup, error) {
	f.createCalls++
	f.gotRequest = request
	return f.created, f.createErr
}

func (f *b6BackupManager) DeleteBackup(_ context.Context, _ yorvaruntime.Installation, backupID string, _ yorvaruntime.ProgressSink) error {
	f.deleteCalls++
	f.gotDeleteID = backupID
	return f.deleteErr
}

func TestBackupManagementListAndInspectRuntimeScope(t *testing.T) {
	now := time.Date(2026, 8, 25, 13, 0, 0, 0, time.FixedZone("test", 8*60*60))
	available := b6AvailableBackup("backup-1", now)
	missing := b6AvailableBackup("backup-2", now)
	missing.State = yorvaruntime.BackupMissing
	reader := &b6BackupReader{backups: []yorvaruntime.Backup{available, missing}, inspected: available}
	installation := yorvaruntime.Installation{RuntimeKind: "hermes", Path: `C:\hermes\hermes.exe`, Version: "0.20.5", SupportState: yorvaruntime.DiscoverySupported}
	resolver := &b6RuntimeTargetResolver{target: RuntimeManagementTarget{
		Installation: installation, InstallationID: "rtinst_1", Bundle: yorvaruntime.Bundle{BackupRead: reader},
	}}
	service := NewBackupManagement(resolver)

	items, err := service.ListBackups(context.Background(), "hermes")
	if err != nil || len(items) != 2 {
		t.Fatalf("ListBackups() = %#v, %v", items, err)
	}
	if resolver.gotID != "hermes" || reader.gotInstall != installation {
		t.Fatalf("target = runtime %q, installation %#v", resolver.gotID, reader.gotInstall)
	}
	if items[0].Scope != BackupScopeRuntime || items[0].CreatedAt.Location() != time.UTC || items[0].VerifiedAt.Location() != time.UTC {
		t.Fatalf("available list projection = %#v", items[0])
	}
	if items[1].State != yorvaruntime.BackupMissing || items[1].FormatVersion == "" || items[1].VerifiedAt.IsZero() {
		t.Fatalf("last-observed missing metadata = %#v", items[1])
	}

	inspected, err := service.InspectBackup(context.Background(), "hermes", "backup-1")
	if err != nil || inspected.ID != "backup-1" {
		t.Fatalf("InspectBackup() = %#v, %v", inspected, err)
	}
	reader.inspectErr = yorvaruntime.ErrBackupNotFound
	if _, err := service.InspectBackup(context.Background(), "hermes", "backup-missing"); !errors.Is(err, ErrBackupNotFound) {
		t.Fatalf("missing inspect error = %v", err)
	}
	if reader.gotBackupID != "backup-missing" {
		t.Fatalf("inspected ID = %q", reader.gotBackupID)
	}
}

func TestBackupManagementFailsClosedForCapabilityAndInvalidResults(t *testing.T) {
	service := NewBackupManagement(&b6RuntimeTargetResolver{target: RuntimeManagementTarget{Bundle: yorvaruntime.Bundle{}}})
	if _, err := service.ListBackups(context.Background(), "hermes"); !errors.Is(err, ErrManagementCapabilityUnsupported) {
		t.Fatalf("list capability error = %v", err)
	}

	now := time.Now().UTC()
	tests := []struct {
		name    string
		backups []yorvaruntime.Backup
	}{
		{name: "duplicate", backups: []yorvaruntime.Backup{b6AvailableBackup("backup-1", now), b6AvailableBackup("backup-1", now)}},
		{name: "malformed", backups: []yorvaruntime.Backup{{ID: "backup-1", State: yorvaruntime.BackupAvailable}}},
		{name: "unsafe format", backups: []yorvaruntime.Backup{func() yorvaruntime.Backup {
			value := b6AvailableBackup("backup-1", now)
			value.FormatVersion = "secret/path"
			return value
		}()}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := &b6BackupReader{backups: tt.backups}
			service := NewBackupManagement(&b6RuntimeTargetResolver{target: RuntimeManagementTarget{Bundle: yorvaruntime.Bundle{BackupRead: reader}}})
			if _, err := service.ListBackups(context.Background(), "hermes"); !errors.Is(err, ErrManagementQueryFailed) {
				t.Fatalf("error = %v, want ErrManagementQueryFailed", err)
			}
		})
	}

	tooMany := make([]yorvaruntime.Backup, managementBackupCollectionLimit+1)
	for i := range tooMany {
		tooMany[i] = b6AvailableBackup(fmt.Sprintf("backup-%d", i), now)
	}
	service = NewBackupManagement(&b6RuntimeTargetResolver{target: RuntimeManagementTarget{Bundle: yorvaruntime.Bundle{BackupRead: &b6BackupReader{backups: tooMany}}}})
	if _, err := service.ListBackups(context.Background(), "hermes"); !errors.Is(err, ErrManagementQueryFailed) {
		t.Fatalf("oversized collection error = %v", err)
	}
}

func TestBackupManagementInspectRejectsFalseIdentity(t *testing.T) {
	now := time.Now().UTC()
	reader := &b6BackupReader{inspected: b6AvailableBackup("backup-other", now)}
	service := NewBackupManagement(&b6RuntimeTargetResolver{target: RuntimeManagementTarget{Bundle: yorvaruntime.Bundle{BackupRead: reader}}})
	if _, err := service.InspectBackup(context.Background(), "hermes", "backup-1"); !errors.Is(err, ErrManagementQueryFailed) {
		t.Fatalf("error = %v, want ErrManagementQueryFailed", err)
	}
	service = NewBackupManagement(&b6RuntimeTargetResolver{target: RuntimeManagementTarget{Bundle: yorvaruntime.Bundle{BackupRead: &b6BackupReader{}}}})
	if _, err := service.InspectBackup(context.Background(), "hermes", "https://unsafe.invalid"); !errors.Is(err, ErrBackupNotFound) {
		t.Fatalf("unsafe backup ID error = %v", err)
	}
}

func TestBackupManagementNormalizesErrorsAndCancellation(t *testing.T) {
	service := NewBackupManagement(&b6RuntimeTargetResolver{err: errors.New("database path and secret")})
	if _, err := service.ListBackups(context.Background(), "hermes"); !errors.Is(err, ErrManagementQueryFailed) || err.Error() != ErrManagementQueryFailed.Error() {
		t.Fatalf("resolver error = %v", err)
	}

	reader := &b6BackupReader{listErr: errors.New("archive path and key reference")}
	service = NewBackupManagement(&b6RuntimeTargetResolver{target: RuntimeManagementTarget{Bundle: yorvaruntime.Bundle{BackupRead: reader}}})
	if _, err := service.ListBackups(context.Background(), "hermes"); !errors.Is(err, ErrManagementQueryFailed) || err.Error() != ErrManagementQueryFailed.Error() {
		t.Fatalf("adapter error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.ListBackups(ctx, "hermes"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled error = %v", err)
	}
}

func TestBackupCreateOperationErrorPreservesStoppedPrecondition(t *testing.T) {
	if got := backupCreateOperationError(fmt.Errorf("adapter context: %w", yorvaruntime.ErrBackupRuntimeNotStopped)); got != yorvaruntime.ErrorBackupRuntimeNotStopped {
		t.Fatalf("backupCreateOperationError() = %q", got)
	}
	if got := backupCreateOperationError(errors.New("private adapter detail")); got != yorvaruntime.ErrorBackupCreateFailed {
		t.Fatalf("generic backupCreateOperationError() = %q", got)
	}
}

func TestBackupCreateDeleteWorkersUseOnlyQualifiedBundleWiring(t *testing.T) {
	validRequest := yorvaruntime.BackupCreateRequest{
		DestinationRef:        strings.Repeat("a", 43),
		OperationID:           "op_backup_1",
		RuntimeInstallationID: "rtinst_1",
	}
	resolver := &b6RuntimeTargetResolver{target: RuntimeManagementTarget{Bundle: yorvaruntime.Bundle{}}}
	service := NewBackupManagement(resolver)
	if _, err := service.CreateBackup(context.Background(), "hermes", validRequest, nil); !errors.Is(err, ErrManagementCapabilityUnsupported) {
		t.Fatalf("CreateBackup() error = %v", err)
	}
	if err := service.DeleteBackup(context.Background(), "hermes", "backup-1", nil); !errors.Is(err, ErrManagementCapabilityUnsupported) {
		t.Fatalf("DeleteBackup() error = %v", err)
	}

	now := time.Now().UTC()
	manager := &b6BackupManager{created: b6AvailableBackup("backup-1", now)}
	resolver = &b6RuntimeTargetResolver{target: RuntimeManagementTarget{Bundle: yorvaruntime.Bundle{BackupMutate: manager}}}
	service = NewBackupManagement(resolver)
	created, err := service.CreateBackup(context.Background(), "hermes", validRequest, nil)
	if err != nil || created.State != yorvaruntime.BackupAvailable || manager.gotRequest != validRequest {
		t.Fatalf("CreateBackup() = %#v, %v; request %#v", created, err, manager.gotRequest)
	}
	if err := service.DeleteBackup(context.Background(), "hermes", "backup-1", nil); err != nil || manager.gotDeleteID != "backup-1" {
		t.Fatalf("DeleteBackup() error = %v, ID %q", err, manager.gotDeleteID)
	}

	manager.created = yorvaruntime.Backup{ID: "backup-2", State: yorvaruntime.BackupFailed}
	if _, err := service.CreateBackup(context.Background(), "hermes", validRequest, nil); !errors.Is(err, ErrManagementQueryFailed) {
		t.Fatalf("false create result error = %v", err)
	}

	invalidResolver := &b6RuntimeTargetResolver{}
	if _, err := NewBackupManagement(invalidResolver).CreateBackup(context.Background(), "hermes", yorvaruntime.BackupCreateRequest{DestinationRef: "../../unsafe"}, nil); !errors.Is(err, ErrManagementQueryFailed) {
		t.Fatalf("unsafe create request error = %v", err)
	}
	if invalidResolver.callCount != 0 {
		t.Fatalf("unsafe request reached resolver %d times", invalidResolver.callCount)
	}
}

func b6AvailableBackup(id string, createdAt time.Time) yorvaruntime.Backup {
	return yorvaruntime.Backup{
		ID: id, State: yorvaruntime.BackupAvailable, FormatVersion: "v1", RuntimeVersion: "0.20.5",
		SizeBytes: 1024, ChecksumSHA256: strings.Repeat("a", 64), CreatedAt: createdAt,
		VerifiedAt: createdAt.Add(time.Minute), KeyMode: yorvaruntime.BackupKeyDevice,
	}
}

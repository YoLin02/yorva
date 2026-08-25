package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/app"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type b6BackupReadService struct {
	items       []app.BackupView
	inspected   app.BackupView
	err         error
	gotRuntime  string
	gotBackupID string
}

func (f *b6BackupReadService) ListBackups(_ context.Context, runtimeID string) ([]app.BackupView, error) {
	f.gotRuntime = runtimeID
	return f.items, f.err
}

func (f *b6BackupReadService) InspectBackup(_ context.Context, runtimeID, backupID string) (app.BackupView, error) {
	f.gotRuntime = runtimeID
	f.gotBackupID = backupID
	return f.inspected, f.err
}

func TestBackupReadHandlersUseRuntimePathAndSafeDTO(t *testing.T) {
	now := time.Date(2026, 8, 25, 14, 0, 0, 0, time.FixedZone("test", 8*60*60))
	available := app.BackupView{
		ID: "backup-1", Scope: app.BackupScopeRuntime, State: yorvaruntime.BackupAvailable,
		FormatVersion: "v1", RuntimeVersion: "0.20.5", SizeBytes: 1024,
		ChecksumSHA256: strings.Repeat("a", 64), CreatedAt: now, VerifiedAt: now.Add(time.Minute), KeyMode: yorvaruntime.BackupKeyDevice,
	}
	missing := available
	missing.ID = "backup-2"
	missing.State = yorvaruntime.BackupMissing
	service := &b6BackupReadService{items: []app.BackupView{available, missing}, inspected: available}

	tests := []struct {
		name       string
		handler    http.Handler
		path       string
		backupID   string
		wantFields []string
	}{
		{name: "list", handler: listRuntimeBackups(service), path: "/backups", wantFields: []string{`"scope":"RUNTIME"`, `"items":[`, `"state":"MISSING"`, `"verifiedAt":"`, `"keyMode":"DEVICE"`}},
		{name: "inspect", handler: getRuntimeBackup(service), path: "/backups/backup-1", backupID: "backup-1", wantFields: []string{`"backupId":"backup-1"`, `"scope":"RUNTIME"`, `"checksumSha256":"`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			req.SetPathValue("runtimeId", "hermes")
			if tt.backupID != "" {
				req.SetPathValue("backupId", tt.backupID)
			}
			res := httptest.NewRecorder()
			tt.handler.ServeHTTP(res, req)
			if res.Code != http.StatusOK || res.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("response = %d %q %s", res.Code, res.Header().Get("Content-Type"), res.Body.String())
			}
			for _, field := range tt.wantFields {
				if !strings.Contains(res.Body.String(), field) {
					t.Fatalf("body %s does not contain %s", res.Body.String(), field)
				}
			}
			for _, forbidden := range []string{"instanceId", "installationId", "path", "destinationRef", "keyRef", "passphrase", "payload", "member"} {
				if strings.Contains(res.Body.String(), forbidden) {
					t.Fatalf("body exposed forbidden backup detail %q: %s", forbidden, res.Body.String())
				}
			}
			if service.gotRuntime != "hermes" || service.gotBackupID != tt.backupID {
				t.Fatalf("path target = runtime %q backup %q", service.gotRuntime, service.gotBackupID)
			}
		})
	}
}

func TestBackupReadHandlersReturnEmptyArrayNotNull(t *testing.T) {
	service := &b6BackupReadService{}
	req := httptest.NewRequest(http.MethodGet, "/backups", nil)
	req.SetPathValue("runtimeId", "hermes")
	res := httptest.NewRecorder()
	listRuntimeBackups(service).ServeHTTP(res, req)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"items":[]`) {
		t.Fatalf("response = %d %s", res.Code, res.Body.String())
	}
}

func TestBackupReadHandlersMapStableSafeErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "not found", err: app.ErrBackupNotFound, wantStatus: http.StatusNotFound, wantCode: "BACKUP_NOT_FOUND"},
		{name: "capability", err: app.ErrManagementCapabilityUnsupported, wantStatus: http.StatusConflict, wantCode: string(yorvaruntime.ErrorCapabilityNotSupported)},
		{name: "runtime", err: app.ErrRuntimeNotSupported, wantStatus: http.StatusConflict, wantCode: string(yorvaruntime.ErrorCapabilityNotSupported)},
		{name: "query", err: app.ErrManagementQueryFailed, wantStatus: http.StatusServiceUnavailable, wantCode: "MANAGEMENT_QUERY_FAILED"},
		{name: "deadline", err: context.DeadlineExceeded, wantStatus: http.StatusServiceUnavailable, wantCode: "MANAGEMENT_QUERY_FAILED"},
		{name: "raw", err: errors.New("C:\\secret\\artifact.age keyRef=private"), wantStatus: http.StatusServiceUnavailable, wantCode: "MANAGEMENT_QUERY_FAILED"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &b6BackupReadService{err: tt.err}
			req := httptest.NewRequest(http.MethodGet, "/backups", nil)
			req.SetPathValue("runtimeId", "hermes")
			res := httptest.NewRecorder()
			listRuntimeBackups(service).ServeHTTP(res, req)
			if res.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body %s", res.Code, tt.wantStatus, res.Body.String())
			}
			var body ErrorResponse
			if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Error.Code != tt.wantCode || strings.Contains(res.Body.String(), "artifact.age") || strings.Contains(res.Body.String(), "keyRef") {
				t.Fatalf("body = %#v", body)
			}
		})
	}
}

func TestBackupReadHandlersNilServiceAndCancellation(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/backups", nil)
	res := httptest.NewRecorder()
	listRuntimeBackups(nil).ServeHTTP(res, req)
	if res.Code != http.StatusConflict || !strings.Contains(res.Body.String(), string(yorvaruntime.ErrorCapabilityNotSupported)) {
		t.Fatalf("nil service response = %d %s", res.Code, res.Body.String())
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req = httptest.NewRequest(http.MethodGet, "/backups", nil).WithContext(ctx)
	res = httptest.NewRecorder()
	listRuntimeBackups(&b6BackupReadService{err: context.Canceled}).ServeHTTP(res, req)
	if res.Body.Len() != 0 {
		t.Fatalf("cancelled response body = %s", res.Body.String())
	}
}

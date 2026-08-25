package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/app"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type ManagementBackupReadService interface {
	ListBackups(context.Context, string) ([]app.BackupView, error)
	InspectBackup(context.Context, string, string) (app.BackupView, error)
}

type ManagementBackupResponse struct {
	ID             string                     `json:"backupId"`
	Scope          app.BackupScope            `json:"scope"`
	State          yorvaruntime.BackupState   `json:"state"`
	FormatVersion  string                     `json:"formatVersion"`
	RuntimeVersion string                     `json:"runtimeVersion"`
	SizeBytes      int64                      `json:"sizeBytes"`
	ChecksumSHA256 string                     `json:"checksumSha256"`
	CreatedAt      time.Time                  `json:"createdAt"`
	VerifiedAt     time.Time                  `json:"verifiedAt"`
	KeyMode        yorvaruntime.BackupKeyMode `json:"keyMode"`
}

type ManagementBackupListResponse struct {
	Scope app.BackupScope            `json:"scope"`
	Items []ManagementBackupResponse `json:"items"`
}

func listRuntimeBackups(service ManagementBackupReadService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeBackupManagementError(w, r, app.ErrManagementCapabilityUnsupported)
			return
		}
		backups, err := service.ListBackups(r.Context(), r.PathValue("runtimeId"))
		if err != nil {
			writeBackupManagementError(w, r, err)
			return
		}
		items := make([]ManagementBackupResponse, 0, len(backups))
		for _, backup := range backups {
			items = append(items, newManagementBackupResponse(backup))
		}
		writeBackupManagementJSON(w, ManagementBackupListResponse{Scope: app.BackupScopeRuntime, Items: items})
	})
}

func getRuntimeBackup(service ManagementBackupReadService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeBackupManagementError(w, r, app.ErrManagementCapabilityUnsupported)
			return
		}
		backup, err := service.InspectBackup(r.Context(), r.PathValue("runtimeId"), r.PathValue("backupId"))
		if err != nil {
			writeBackupManagementError(w, r, err)
			return
		}
		writeBackupManagementJSON(w, newManagementBackupResponse(backup))
	})
}

func newManagementBackupResponse(backup app.BackupView) ManagementBackupResponse {
	return ManagementBackupResponse{
		ID: backup.ID, Scope: backup.Scope, State: backup.State,
		FormatVersion: backup.FormatVersion, RuntimeVersion: backup.RuntimeVersion,
		SizeBytes: backup.SizeBytes, ChecksumSHA256: backup.ChecksumSHA256,
		CreatedAt: backup.CreatedAt.UTC(), VerifiedAt: backup.VerifiedAt.UTC(), KeyMode: backup.KeyMode,
	}
}

func writeBackupManagementJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func writeBackupManagementError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, context.Canceled) && r.Context().Err() != nil:
		return
	case errors.Is(err, app.ErrBackupNotFound):
		writeError(w, http.StatusNotFound, ErrorBody{
			Code: "BACKUP_NOT_FOUND", Message: "The requested backup was not found.", Retryable: false,
		})
	case errors.Is(err, app.ErrManagementCapabilityUnsupported), errors.Is(err, app.ErrRuntimeNotSupported):
		writeError(w, http.StatusConflict, ErrorBody{
			Code: string(yorvaruntime.ErrorCapabilityNotSupported), Message: "Runtime backup management is not supported.", Retryable: false,
		})
	case errors.Is(err, app.ErrManagementQueryFailed), errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		writeError(w, http.StatusServiceUnavailable, ErrorBody{
			Code: "MANAGEMENT_QUERY_FAILED", Message: "Runtime backup data could not be queried.", Retryable: true,
		})
	default:
		writeError(w, http.StatusServiceUnavailable, ErrorBody{
			Code: "MANAGEMENT_QUERY_FAILED", Message: "Runtime backup data could not be queried.", Retryable: true,
		})
	}
}

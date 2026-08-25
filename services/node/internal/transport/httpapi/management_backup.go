package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/app"
	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type ManagementBackupReadService interface {
	ListBackups(context.Context, string) ([]app.BackupView, error)
	InspectBackup(context.Context, string, string) (app.BackupView, error)
}

type ManagementBackupService interface {
	ManagementBackupReadService
	StartCreateBackup(context.Context, string, string, string) (app.InstallStartResult, error)
	StartDeleteBackup(context.Context, string, string, string) (app.InstallStartResult, error)
	StartRestoreBackup(context.Context, string, string, string) (app.InstallStartResult, error)
	CancelBackupOperation(context.Context, string) (operation.Operation, error)
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

func startRuntimeBackup(service ManagementBackupService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeBackupManagementError(w, r, app.ErrManagementCapabilityUnsupported)
			return
		}
		key := r.Header.Get("Idempotency-Key")
		if app.ValidateIdempotencyKey(key) != nil {
			writeError(w, http.StatusBadRequest, ErrorBody{Code: "INVALID_IDEMPOTENCY_KEY", Message: "A valid Idempotency-Key header is required.", Retryable: false})
			return
		}
		destinationRef, err := decodeBackupCreateRequest(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, ErrorBody{Code: "INVALID_REQUEST", Message: "The backup request must contain only destinationRef.", Retryable: false})
			return
		}
		result, err := service.StartCreateBackup(r.Context(), r.PathValue("runtimeId"), destinationRef, key)
		if err != nil {
			writeBackupManagementError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(newOperationResponse(result.Operation))
	})
}

func startDeleteBackup(service ManagementBackupService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeBackupManagementError(w, r, app.ErrManagementCapabilityUnsupported)
			return
		}
		key := r.Header.Get("Idempotency-Key")
		if app.ValidateIdempotencyKey(key) != nil {
			writeError(w, http.StatusBadRequest, ErrorBody{Code: "INVALID_IDEMPOTENCY_KEY", Message: "A valid Idempotency-Key header is required.", Retryable: false})
			return
		}
		if err := decodeClosedEmptyObject(r); err != nil {
			writeError(w, http.StatusBadRequest, ErrorBody{Code: "INVALID_REQUEST", Message: "The delete request must be a closed empty JSON object.", Retryable: false})
			return
		}
		result, err := service.StartDeleteBackup(r.Context(), "hermes", r.PathValue("backupId"), key)
		if err != nil {
			writeBackupManagementError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(newOperationResponse(result.Operation))
	})
}

func startRestoreBackup(service ManagementBackupService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeBackupManagementError(w, r, app.ErrManagementCapabilityUnsupported)
			return
		}
		key := r.Header.Get("Idempotency-Key")
		if app.ValidateIdempotencyKey(key) != nil {
			writeError(w, http.StatusBadRequest, ErrorBody{Code: "INVALID_IDEMPOTENCY_KEY", Message: "A valid Idempotency-Key header is required.", Retryable: false})
			return
		}
		if err := decodeClosedEmptyObject(r); err != nil {
			writeError(w, http.StatusBadRequest, ErrorBody{Code: "INVALID_REQUEST", Message: "The Restore request must be a closed empty JSON object.", Retryable: false})
			return
		}
		result, err := service.StartRestoreBackup(r.Context(), "hermes", r.PathValue("backupId"), key)
		if err != nil {
			writeBackupManagementError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(newOperationResponse(result.Operation))
	})
}

func decodeBackupCreateRequest(r *http.Request) (string, error) {
	if r.Body == nil {
		return "", io.EOF
	}
	defer r.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(r.Body, 2049))
	if err != nil || len(payload) == 0 || len(payload) > 2048 {
		return "", io.ErrUnexpectedEOF
	}
	var body struct {
		DestinationRef string `json:"destinationRef"`
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		return "", err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return "", errors.New("trailing json")
	}
	if err := yorvaruntime.ValidateBackupDestinationRef(body.DestinationRef); err != nil {
		return "", err
	}
	return body.DestinationRef, nil
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
	case errors.Is(err, app.ErrBackupMutationConflict):
		writeError(w, http.StatusConflict, ErrorBody{
			Code: string(yorvaruntime.ErrorBackupMutationConflict), Message: "Another backup or Runtime mutation is already running.", Retryable: false,
		})
	case errors.Is(err, app.ErrInstanceNotCancellable):
		writeError(w, http.StatusConflict, ErrorBody{Code: string(yorvaruntime.ErrorOperationNotCancellable), Message: "This backup Operation can no longer be cancelled.", Retryable: false})
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

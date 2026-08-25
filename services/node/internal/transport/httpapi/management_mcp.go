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

type ManagementMCPReadService interface {
	ListMCPServers(context.Context, string) ([]app.MCPServerView, error)
	ListMCPPresets(context.Context, string) ([]app.MCPPresetView, error)
}

type ManagementMCPServerResponse struct {
	ID         string                `json:"id"`
	PresetID   string                `json:"presetId"`
	State      yorvaruntime.MCPState `json:"state"`
	ReadyAt    *time.Time            `json:"readyAt"`
	ObservedAt time.Time             `json:"observedAt"`
}

type ManagementMCPServerListResponse struct {
	Items []ManagementMCPServerResponse `json:"items"`
}

type ManagementMCPPresetResponse struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type ManagementMCPPresetListResponse struct {
	Items []ManagementMCPPresetResponse `json:"items"`
}

func listMCPServers(service ManagementMCPReadService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeMCPManagementError(w, r, app.ErrManagementCapabilityUnsupported)
			return
		}
		servers, err := service.ListMCPServers(r.Context(), r.PathValue("instanceId"))
		if err != nil {
			writeMCPManagementError(w, r, err)
			return
		}

		items := make([]ManagementMCPServerResponse, 0, len(servers))
		for _, server := range servers {
			items = append(items, ManagementMCPServerResponse{
				ID:         server.ID,
				PresetID:   server.PresetID,
				State:      server.State,
				ReadyAt:    copyMCPTime(server.ReadyAt),
				ObservedAt: server.ObservedAt.UTC(),
			})
		}
		writeMCPManagementJSON(w, ManagementMCPServerListResponse{Items: items})
	})
}

func listMCPPresets(service ManagementMCPReadService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeMCPManagementError(w, r, app.ErrManagementCapabilityUnsupported)
			return
		}
		presets, err := service.ListMCPPresets(r.Context(), r.PathValue("instanceId"))
		if err != nil {
			writeMCPManagementError(w, r, err)
			return
		}

		items := make([]ManagementMCPPresetResponse, 0, len(presets))
		for _, preset := range presets {
			items = append(items, ManagementMCPPresetResponse{ID: preset.ID, DisplayName: preset.DisplayName})
		}
		writeMCPManagementJSON(w, ManagementMCPPresetListResponse{Items: items})
	})
}

func writeMCPManagementJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func writeMCPManagementError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, context.Canceled) && r.Context().Err() != nil:
		return
	case errors.Is(err, app.ErrInstanceNotFound):
		writeError(w, http.StatusNotFound, ErrorBody{
			Code: string(yorvaruntime.ErrorInstanceNotFound), Message: "The requested instance was not found.", Retryable: false,
		})
	case errors.Is(err, app.ErrInstanceNotAvailable):
		writeError(w, http.StatusConflict, ErrorBody{
			Code: string(yorvaruntime.ErrorInstanceNotAvailable), Message: "The requested instance is not available.", Retryable: false,
		})
	case errors.Is(err, app.ErrManagementCapabilityUnsupported), errors.Is(err, app.ErrRuntimeNotSupported):
		writeError(w, http.StatusConflict, ErrorBody{
			Code: string(yorvaruntime.ErrorCapabilityNotSupported), Message: "MCP management is not supported.", Retryable: false,
		})
	case errors.Is(err, app.ErrManagementQueryFailed), errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		writeError(w, http.StatusServiceUnavailable, ErrorBody{
			Code: "MANAGEMENT_QUERY_FAILED", Message: "MCP management data could not be queried.", Retryable: true,
		})
	default:
		writeError(w, http.StatusServiceUnavailable, ErrorBody{
			Code: "MANAGEMENT_QUERY_FAILED", Message: "MCP management data could not be queried.", Retryable: true,
		})
	}
}

func copyMCPTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	result := value.UTC()
	return &result
}

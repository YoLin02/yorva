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

type ManagementMCPReadService interface {
	ListMCPServers(context.Context, string) ([]app.MCPServerView, error)
	ListMCPPresets(context.Context, string) ([]app.MCPPresetView, error)
}

type RuntimeMCPDefinitionService interface {
	ListRuntimeMCPDefinitions(context.Context, string) ([]app.MCPPresetView, error)
}

type ManagementMCPService interface {
	ManagementMCPReadService
	RuntimeMCPDefinitionService
	StartInstall(context.Context, string, string, []byte, []string, string) (app.InstallStartResult, error)
	StartAuthenticate(context.Context, string, string, []byte, string) (app.InstallStartResult, error)
	StartTest(context.Context, string, string, string) (app.InstallStartResult, error)
	StartConfigure(context.Context, string, string, []string, string) (app.InstallStartResult, error)
	StartRemove(context.Context, string, string, string) (app.InstallStartResult, error)
	CancelMCPOperation(context.Context, string) (operation.Operation, error)
}

type ManagementMCPServerResponse struct {
	ID             string                `json:"id"`
	PresetID       string                `json:"presetId"`
	Ownership      string                `json:"ownership"`
	EnabledToolIDs []string              `json:"enabledToolIds"`
	State          yorvaruntime.MCPState `json:"state"`
	ReadyAt        *time.Time            `json:"readyAt"`
	ObservedAt     time.Time             `json:"observedAt"`
}

type ManagementMCPServerListResponse struct {
	Items []ManagementMCPServerResponse `json:"items"`
}

type ManagementMCPPresetResponse struct {
	ID                 string   `json:"id"`
	DisplayName        string   `json:"displayName"`
	Description        string   `json:"description"`
	HomepageURL        string   `json:"homepageUrl"`
	DocumentationURL   string   `json:"documentationUrl"`
	AllowedToolIDs     []string `json:"allowedToolIds"`
	CredentialRequired bool     `json:"credentialRequired"`
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
			enabledToolIDs := server.EnabledToolIDs
			if enabledToolIDs == nil {
				enabledToolIDs = []string{}
			}
			items = append(items, ManagementMCPServerResponse{
				ID:             server.ID,
				PresetID:       server.PresetID,
				Ownership:      server.Ownership,
				EnabledToolIDs: enabledToolIDs,
				State:          server.State,
				ReadyAt:        copyMCPTime(server.ReadyAt),
				ObservedAt:     server.ObservedAt.UTC(),
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

		writeMCPPresetViews(w, presets)
	})
}

func listRuntimeMCPDefinitions(service RuntimeMCPDefinitionService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeMCPManagementError(w, r, app.ErrManagementCapabilityUnsupported)
			return
		}
		presets, err := service.ListRuntimeMCPDefinitions(r.Context(), r.PathValue("runtimeId"))
		if err != nil {
			writeMCPManagementError(w, r, err)
			return
		}
		writeMCPPresetViews(w, presets)
	})
}

func writeMCPPresetViews(w http.ResponseWriter, presets []app.MCPPresetView) {
	items := make([]ManagementMCPPresetResponse, 0, len(presets))
	for _, preset := range presets {
		allowedToolIDs := preset.AllowedToolIDs
		if allowedToolIDs == nil {
			allowedToolIDs = []string{}
		}
		items = append(items, ManagementMCPPresetResponse{
			ID: preset.ID, DisplayName: preset.DisplayName, Description: preset.Description,
			HomepageURL: preset.HomepageURL, DocumentationURL: preset.DocumentationURL,
			AllowedToolIDs: allowedToolIDs, CredentialRequired: preset.CredentialRequired,
		})
	}
	writeMCPManagementJSON(w, ManagementMCPPresetListResponse{Items: items})
}

type mcpMutationKind string

const (
	mcpInstall      mcpMutationKind = "install"
	mcpAuthenticate mcpMutationKind = "authenticate"
	mcpTest         mcpMutationKind = "test"
	mcpConfigure    mcpMutationKind = "configure"
	mcpRemove       mcpMutationKind = "remove"
)

func startMCPMutation(service ManagementMCPService, kind mcpMutationKind) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeMCPManagementError(w, r, app.ErrManagementCapabilityUnsupported)
			return
		}
		key := r.Header.Get("Idempotency-Key")
		if app.ValidateIdempotencyKey(key) != nil {
			writeError(w, http.StatusBadRequest, ErrorBody{Code: "INVALID_IDEMPOTENCY_KEY", Message: "A valid Idempotency-Key header is required.", Retryable: false})
			return
		}
		instanceID := r.PathValue("instanceId")
		var (
			result app.InstallStartResult
			err    error
		)
		switch kind {
		case mcpInstall:
			var credential []byte
			var toolIDs []string
			credential, toolIDs, err = decodeMCPInstall(r, r.PathValue("presetId"))
			if err == nil {
				result, err = service.StartInstall(r.Context(), instanceID, r.PathValue("presetId"), credential, toolIDs, key)
			}
			clear(credential)
		case mcpAuthenticate:
			var credential []byte
			credential, err = decodeMCPAuthentication(r)
			if err == nil {
				result, err = service.StartAuthenticate(r.Context(), instanceID, r.PathValue("serverId"), credential, key)
			}
			clear(credential)
		case mcpTest:
			err = decodeClosedEmptyObject(r)
			if err == nil {
				result, err = service.StartTest(r.Context(), instanceID, r.PathValue("serverId"), key)
			}
		case mcpConfigure:
			var toolIDs []string
			toolIDs, err = decodeMCPConfiguration(r, r.PathValue("serverId"))
			if err == nil {
				result, err = service.StartConfigure(r.Context(), instanceID, r.PathValue("serverId"), toolIDs, key)
			}
		case mcpRemove:
			err = decodeClosedEmptyObject(r)
			if err == nil {
				result, err = service.StartRemove(r.Context(), instanceID, r.PathValue("serverId"), key)
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, yorvaruntime.ErrInvalidManagementContract) {
				writeError(w, http.StatusBadRequest, ErrorBody{Code: "INVALID_REQUEST", Message: "The MCP request does not match the closed schema.", Retryable: false})
				return
			}
			writeMCPManagementError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(newOperationResponse(result.Operation))
	})
}

func decodeMCPInstall(r *http.Request, presetID string) ([]byte, []string, error) {
	var body struct {
		Credential     string   `json:"credential"`
		EnabledToolIDs []string `json:"enabledToolIds"`
	}
	if err := decodeClosedMCPBody(r, &body); err != nil {
		return nil, nil, err
	}
	credential := []byte(body.Credential)
	if len(credential) > 0 {
		if err := (yorvaruntime.MCPAuthenticateRequest{ServerID: presetID, Credential: credential}).Validate(); err != nil {
			clear(credential)
			return nil, nil, err
		}
	}
	if err := (yorvaruntime.MCPConfigureRequest{ServerID: presetID, EnabledToolIDs: body.EnabledToolIDs}).Validate(); err != nil {
		clear(credential)
		return nil, nil, err
	}
	if len(body.EnabledToolIDs) == 0 {
		clear(credential)
		return nil, nil, yorvaruntime.ErrInvalidManagementContract
	}
	return credential, append([]string(nil), body.EnabledToolIDs...), nil
}

func decodeMCPAuthentication(r *http.Request) ([]byte, error) {
	var body struct {
		Credential string `json:"credential"`
	}
	if err := decodeClosedMCPBody(r, &body); err != nil {
		return nil, err
	}
	credential := []byte(body.Credential)
	if err := (yorvaruntime.MCPAuthenticateRequest{ServerID: r.PathValue("serverId"), Credential: credential}).Validate(); err != nil {
		clear(credential)
		return nil, err
	}
	return credential, nil
}

func decodeMCPConfiguration(r *http.Request, serverID string) ([]string, error) {
	var body struct {
		EnabledToolIDs []string `json:"enabledToolIds"`
	}
	if err := decodeClosedMCPBody(r, &body); err != nil {
		return nil, err
	}
	if err := (yorvaruntime.MCPConfigureRequest{ServerID: serverID, EnabledToolIDs: body.EnabledToolIDs}).Validate(); err != nil {
		return nil, err
	}
	return body.EnabledToolIDs, nil
}

func decodeClosedMCPBody(r *http.Request, target any) error {
	if r.Body == nil {
		return io.EOF
	}
	defer r.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(r.Body, 8193))
	if err != nil || len(payload) == 0 || len(payload) > 8192 {
		return io.ErrUnexpectedEOF
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("trailing json")
	}
	return nil
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
	case errors.Is(err, app.ErrMCPMutationConflict):
		writeError(w, http.StatusConflict, ErrorBody{
			Code: string(yorvaruntime.ErrorMCPMutationConflict), Message: "Another MCP or Instance mutation is already running.", Retryable: false,
		})
	case errors.Is(err, app.ErrInstanceNotCancellable):
		writeError(w, http.StatusConflict, ErrorBody{
			Code: string(yorvaruntime.ErrorOperationNotCancellable), Message: "This MCP Operation can no longer be cancelled.", Retryable: false,
		})
	case errors.Is(err, yorvaruntime.ErrInvalidManagementContract):
		writeError(w, http.StatusBadRequest, ErrorBody{Code: "INVALID_REQUEST", Message: "The MCP request is invalid.", Retryable: false})
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

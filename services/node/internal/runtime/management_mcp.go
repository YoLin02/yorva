package runtime

import (
	"context"
	"time"
)

type MCPState string

const (
	MCPNotConfigured MCPState = "NOT_CONFIGURED"
	MCPConfigured    MCPState = "CONFIGURED"
	MCPAuthRequired  MCPState = "AUTH_REQUIRED"
	MCPReady         MCPState = "READY"
	MCPFailed        MCPState = "FAILED"
	MCPUnknown       MCPState = "UNKNOWN"
)

func (s MCPState) Valid() bool {
	return s == MCPNotConfigured || s == MCPConfigured || s == MCPAuthRequired || s == MCPReady || s == MCPFailed || s == MCPUnknown
}

type MCPPreset struct {
	ID                 string
	DisplayName        string
	AllowedToolIDs     []string
	CredentialRequired bool
}

func (p MCPPreset) Validate() error {
	if err := validateManagementID("MCP preset id", p.ID); err != nil {
		return err
	}
	if err := validateBoundedText("MCP preset display name", p.DisplayName); err != nil {
		return err
	}
	return validateUniqueIDs("MCP tool id", p.AllowedToolIDs)
}

type MCPServer struct {
	ID         string
	PresetID   string
	State      MCPState
	ReadyAt    *time.Time
	ObservedAt time.Time
}

func (s MCPServer) Validate() error {
	if err := validateManagementID("MCP server id", s.ID); err != nil {
		return err
	}
	if err := validateManagementID("MCP preset id", s.PresetID); err != nil {
		return err
	}
	if !s.State.Valid() || s.ObservedAt.IsZero() {
		return ErrInvalidManagementContract
	}
	if s.State == MCPReady && (s.ReadyAt == nil || s.ReadyAt.IsZero()) {
		return ErrInvalidManagementContract
	}
	if s.State != MCPReady && s.ReadyAt != nil {
		return ErrInvalidManagementContract
	}
	return nil
}

type MCPInstallRequest struct {
	PresetID string
}

func (r MCPInstallRequest) Validate() error {
	return validateManagementID("MCP preset id", r.PresetID)
}

// MCPAuthenticateRequest is limited to a request-scoped static credential for
// a reviewed preset. OAuth needs a separately qualified initiating-session
// contract and is intentionally not represented here.
type MCPAuthenticateRequest struct {
	ServerID   string
	Credential []byte
}

func (r MCPAuthenticateRequest) Validate() error {
	if err := validateManagementID("MCP server id", r.ServerID); err != nil {
		return err
	}
	return validateSecret(r.Credential)
}

type MCPConfigureRequest struct {
	ServerID       string
	EnabledToolIDs []string
}

func (r MCPConfigureRequest) Validate() error {
	if err := validateManagementID("MCP server id", r.ServerID); err != nil {
		return err
	}
	return validateUniqueIDs("MCP tool id", r.EnabledToolIDs)
}

type MCPTestResult struct {
	ServerID string
	State    MCPState
	ToolIDs  []string
	ReadyAt  *time.Time
	TestedAt time.Time
}

func (r MCPTestResult) Validate() error {
	if err := validateManagementID("MCP server id", r.ServerID); err != nil {
		return err
	}
	if !r.State.Valid() || r.TestedAt.IsZero() {
		return ErrInvalidManagementContract
	}
	if err := validateUniqueIDs("MCP tool id", r.ToolIDs); err != nil {
		return err
	}
	if r.State == MCPReady && (r.ReadyAt == nil || r.ReadyAt.IsZero()) {
		return ErrInvalidManagementContract
	}
	if r.State != MCPReady && r.ReadyAt != nil {
		return ErrInvalidManagementContract
	}
	return nil
}

type MCPReader interface {
	ListMCPServers(context.Context, Installation, string) ([]MCPServer, error)
	ListMCPPresets(context.Context, Installation, string) ([]MCPPreset, error)
}

// MCPManager has no command, argv, environment, header, URL, package, or path
// input. Preset and tool IDs must resolve through adapter-owned allowlists.
type MCPManager interface {
	InstallMCPPreset(context.Context, Installation, string, MCPInstallRequest, ProgressSink) (MCPServer, error)
	AuthenticateMCP(context.Context, Installation, string, MCPAuthenticateRequest, ProgressSink) (MCPServer, error)
	TestMCP(context.Context, Installation, string, string, ProgressSink) (MCPTestResult, error)
	ConfigureMCP(context.Context, Installation, string, MCPConfigureRequest, ProgressSink) (MCPServer, error)
	RemoveMCP(context.Context, Installation, string, string, ProgressSink) (MCPServer, error)
}

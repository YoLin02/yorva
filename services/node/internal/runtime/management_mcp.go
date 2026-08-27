package runtime

import (
	"context"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type MCPTransport string

const (
	MCPTransportHTTP  MCPTransport = "HTTP"
	MCPTransportStdio MCPTransport = "STDIO"
)

type MCPConfigValue struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Prefix string `json:"prefix"`
	Secret bool   `json:"secret"`
}

type MCPDefinition struct {
	ID               string
	DisplayName      string
	Description      string
	HomepageURL      string
	DocumentationURL string
	Transport        MCPTransport
	Endpoint         string
	Command          string
	Args             []string
	Environment      []MCPConfigValue
	Headers          []MCPConfigValue
	ToolIDs          []string
}

var mcpConfigNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.-]{0,127}$`)

func (d MCPDefinition) Validate() error {
	if err := validateManagementID("MCP definition id", d.ID); err != nil {
		return err
	}
	if err := validateBoundedText("MCP definition display name", d.DisplayName); err != nil {
		return err
	}
	if len(d.Description) > 2048 || strings.ContainsRune(d.Description, '\x00') {
		return ErrInvalidManagementContract
	}
	if !validMCPMetadataURL(d.HomepageURL) || !validMCPMetadataURL(d.DocumentationURL) {
		return ErrInvalidManagementContract
	}
	if err := validateUniqueIDs("MCP tool id", d.ToolIDs); err != nil {
		return err
	}
	if len(d.Args) > 64 || len(d.Environment) > 64 || len(d.Headers) > 64 {
		return ErrInvalidManagementContract
	}
	for _, arg := range d.Args {
		if arg == "" || len(arg) > 2048 || strings.ContainsRune(arg, '\x00') {
			return ErrInvalidManagementContract
		}
	}
	if err := validateMCPConfigValues(d.Environment, "environment"); err != nil {
		return err
	}
	if err := validateMCPConfigValues(d.Headers, "header"); err != nil {
		return err
	}
	switch d.Transport {
	case MCPTransportHTTP:
		parsed, err := url.Parse(d.Endpoint)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" || len(d.Endpoint) > 2048 || d.Command != "" || len(d.Args) != 0 || len(d.Environment) != 0 {
			return ErrInvalidManagementContract
		}
	case MCPTransportStdio:
		if d.Command == "" || len(d.Command) > 2048 || strings.ContainsRune(d.Command, '\x00') || d.Endpoint != "" || len(d.Headers) != 0 {
			return ErrInvalidManagementContract
		}
	default:
		return ErrInvalidManagementContract
	}
	return nil
}

func validateMCPConfigValues(values []MCPConfigValue, kind string) error {
	seen := make(map[string]struct{}, len(values))
	for _, item := range values {
		if !mcpConfigNamePattern.MatchString(item.Name) || len(item.Value) > 8192 || len(item.Prefix) > 128 || strings.ContainsAny(item.Value, "\x00\r\n") || strings.ContainsAny(item.Prefix, "\x00\r\n") || (item.Secret && item.Value != "") || (!item.Secret && item.Prefix != "") || (kind == "environment" && item.Prefix != "") {
			return ErrInvalidManagementContract
		}
		key := strings.ToLower(item.Name)
		upper := strings.ToUpper(item.Name)
		if !item.Secret && ((kind == "header" && (upper == "AUTHORIZATION" || upper == "COOKIE" || strings.Contains(upper, "API-KEY"))) || (kind == "environment" && (strings.Contains(upper, "TOKEN") || strings.Contains(upper, "SECRET") || strings.Contains(upper, "PASSWORD") || strings.HasSuffix(upper, "_KEY")))) {
			return ErrInvalidManagementContract
		}
		if _, ok := seen[key]; ok {
			return ErrInvalidManagementContract
		}
		seen[key] = struct{}{}
	}
	return nil
}

func (d MCPDefinition) SecretNames() []string {
	result := make([]string, 0)
	for _, item := range append(append([]MCPConfigValue(nil), d.Environment...), d.Headers...) {
		if item.Secret {
			result = append(result, item.Name)
		}
	}
	return result
}

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
	Description        string
	HomepageURL        string
	DocumentationURL   string
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
	if p.Description != "" {
		if err := validateBoundedText("MCP preset description", p.Description); err != nil {
			return err
		}
	}
	if !validMCPMetadataURL(p.HomepageURL) || !validMCPMetadataURL(p.DocumentationURL) {
		return ErrInvalidManagementContract
	}
	return validateUniqueIDs("MCP tool id", p.AllowedToolIDs)
}

func validMCPMetadataURL(value string) bool {
	if value == "" {
		return true
	}
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme == "https" && parsed.Host != "" && parsed.User == nil && parsed.Fragment == ""
}

type MCPServer struct {
	ID             string
	PresetID       string
	Managed        bool
	EnabledToolIDs []string
	State          MCPState
	ReadyAt        *time.Time
	ObservedAt     time.Time
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
	if err := validateUniqueIDs("MCP tool id", s.EnabledToolIDs); err != nil {
		return err
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

// MCPAuthenticateRequest carries the single request-scoped credential used by
// built-in presets. Custom definitions use named write-only secret values on
// MCPCustomManager instead.
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

// MCPManager is the reviewed-preset path. Fully custom definitions use the
// separate structured MCPCustomManager contract below.
type MCPManager interface {
	InstallMCPPreset(context.Context, Installation, string, MCPInstallRequest, ProgressSink) (MCPServer, error)
	AuthenticateMCP(context.Context, Installation, string, MCPAuthenticateRequest, ProgressSink) (MCPServer, error)
	TestMCP(context.Context, Installation, string, string, ProgressSink) (MCPTestResult, error)
	ConfigureMCP(context.Context, Installation, string, MCPConfigureRequest, ProgressSink) (MCPServer, error)
	RemoveMCP(context.Context, Installation, string, string, ProgressSink) (MCPServer, error)
}

// MCPCustomManager is implemented only by adapters that can round-trip a
// structured user-owned definition. Command execution is always direct argv;
// no shell string is part of this contract.
type MCPCustomManager interface {
	InstallCustomMCP(context.Context, Installation, string, MCPDefinition, map[string][]byte, ProgressSink) (MCPServer, error)
	UpdateCustomMCPCredential(context.Context, Installation, string, MCPDefinition, map[string][]byte, ProgressSink) (MCPServer, error)
	TestCustomMCP(context.Context, Installation, string, MCPDefinition, ProgressSink) (MCPTestResult, error)
	ConfigureCustomMCP(context.Context, Installation, string, MCPDefinition, MCPConfigureRequest, ProgressSink) (MCPServer, error)
	RemoveCustomMCP(context.Context, Installation, string, MCPDefinition, ProgressSink) (MCPServer, error)
}

package mcpmanagement

const (
	OAuthCapabilitySupported = false
	OAuthUnavailableCode     = "MCP_OAUTH_SESSION_ISOLATION_UNQUALIFIED"
	OAuthUnavailableReason   = "Hermes 0.20.5 OAuth is unavailable until initiating YORVA session isolation and terminal cleanup are qualified"
)

// OAuthCapability reports stable capability truth. This package contains no
// OAuth start, poll, callback, token, or browser surface.
type OAuthCapability struct{}

func CurrentOAuthCapability() OAuthCapability { return OAuthCapability{} }

func (OAuthCapability) Supported() bool { return OAuthCapabilitySupported }

func (OAuthCapability) ErrorCode() string { return OAuthUnavailableCode }

func (OAuthCapability) Reason() string { return OAuthUnavailableReason }

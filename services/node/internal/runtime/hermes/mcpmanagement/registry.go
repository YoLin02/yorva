package mcpmanagement

import (
	"errors"
	"net/url"
	"path"
	"regexp"
	"strings"
)

const (
	maxPresetIDLength    = 64
	maxDisplayNameLength = 128
	maxEndpointLength    = 2048
	maxCredentialKeyLen  = 128
)

var (
	ErrDescriptorInvalid = errors.New("Hermes MCP descriptor is invalid")
	ErrDescriptorUnknown = errors.New("Hermes MCP descriptor is not reviewed")

	presetIDPattern      = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)
	credentialKeyPattern = regexp.MustCompile(`^MCP_[A-Z0-9_]{1,123}_API_KEY$`)
)

// CredentialClass is fixed by a reviewed descriptor. It is never selected by
// a request.
type CredentialClass string

const (
	CredentialClassNone         CredentialClass = "NONE"
	CredentialClassStaticBearer CredentialClass = "STATIC_BEARER"
)

func (c CredentialClass) valid() bool {
	return c == CredentialClassNone || c == CredentialClassStaticBearer
}

// reviewedDescriptor is intentionally private: callers cannot construct or
// override an endpoint, authentication shape, credential key, or tool policy.
// There are no command, argv, environment value, header, local path, package,
// or bootstrap fields in this schema.
type reviewedDescriptor struct {
	presetID         string
	displayName      string
	description      string
	homepageURL      string
	documentationURL string
	endpoint         string
	credential       CredentialClass
	credentialKey    string
	allowedTools     []string
}

// reviewedDescriptors is a value-producing function rather than mutable
// registry state. Adding an entry requires a source review and a code change.
// No production HTTPS preset has completed descriptor/source qualification yet,
// so the product catalog remains intentionally empty. The typed mutation API is
// still available and rejects unknown preset IDs at this registry boundary.
func reviewedDescriptors() [0]reviewedDescriptor {
	return [0]reviewedDescriptor{}
}

// qualificationDescriptors is never part of the product catalog. It gives
// adapter contract tests one fixed, non-routable HTTPS identity so the full
// Profile write/probe/read-back/remove lifecycle can be exercised without
// turning a test endpoint into product authority.
func qualificationDescriptors() [1]reviewedDescriptor {
	return [1]reviewedDescriptor{{
		presetID: "yorva-mcp-test", displayName: "YORVA MCP Test",
		description: "Qualification-only MCP lifecycle preset.",
		homepageURL: "https://yorva.local/", documentationURL: "https://yorva.local/docs/mcp-test",
		endpoint: "https://mcp-test.yorva.invalid/mcp", credential: CredentialClassStaticBearer,
		credentialKey: "MCP_YORVA_TEST_API_KEY",
		allowedTools:  []string{"yorva_ping"},
	}}
}

// Preset is the safe catalog projection. Endpoint and credential-storage
// details remain adapter-owned.
type Preset struct {
	ID               string
	DisplayName      string
	Description      string
	HomepageURL      string
	DocumentationURL string
	AllowedToolIDs   []string
	CredentialClass  CredentialClass
}

// Registry provides read-only access to the compile-time reviewed set.
type Registry struct {
	descriptors []reviewedDescriptor
}

func NewRegistry() Registry {
	descriptors := reviewedDescriptors()
	return Registry{descriptors: append([]reviewedDescriptor(nil), descriptors[:]...)}
}

// NewQualificationRegistry returns only the fixed qualification preset above.
// It accepts no endpoint or execution material and must not be used for normal
// daemon composition.
func NewQualificationRegistry() Registry {
	descriptors := qualificationDescriptors()
	return Registry{descriptors: append([]reviewedDescriptor(nil), descriptors[:]...)}
}

func (r Registry) Catalog() []Preset {
	descriptors := r.descriptors
	if len(descriptors) == 0 {
		return nil
	}
	result := make([]Preset, 0, len(descriptors))
	for _, descriptor := range descriptors {
		if validateDescriptor(descriptor) != nil {
			// Invalid reviewed source must fail closed and never become catalog
			// authority.
			return nil
		}
		result = append(result, Preset{
			ID:               descriptor.presetID,
			DisplayName:      descriptor.displayName,
			Description:      descriptor.description,
			HomepageURL:      descriptor.homepageURL,
			DocumentationURL: descriptor.documentationURL,
			AllowedToolIDs:   append([]string(nil), descriptor.allowedTools...),
			CredentialClass:  descriptor.credential,
		})
	}
	return result
}

// Resolve accepts only an ID. It cannot accept or override URL, command, argv,
// environment, header, path, package, or bootstrap material.
func (r Registry) Resolve(presetID string) (Selection, error) {
	if !validPresetID(presetID) {
		return Selection{}, ErrDescriptorUnknown
	}
	for _, descriptor := range r.descriptors {
		if descriptor.presetID != presetID {
			continue
		}
		if err := validateDescriptor(descriptor); err != nil {
			return Selection{}, err
		}
		return Selection{descriptor: descriptor}, nil
	}
	return Selection{}, ErrDescriptorUnknown
}

// Selection can only be obtained from Registry.Resolve. Its fields are private
// so callers cannot substitute an endpoint or authentication shape.
type Selection struct {
	descriptor reviewedDescriptor
}

func (s Selection) PresetID() string { return s.descriptor.presetID }

func (s Selection) DisplayName() string { return s.descriptor.displayName }

func (s Selection) CredentialClass() CredentialClass { return s.descriptor.credential }

func (s Selection) AllowedToolIDs() []string {
	return append([]string(nil), s.descriptor.allowedTools...)
}

func (s Selection) CredentialKey() string { return s.descriptor.credentialKey }

func (s Selection) AuthorizationTemplate() string {
	if s.descriptor.credential != CredentialClassStaticBearer {
		return ""
	}
	return "Bearer ${" + s.descriptor.credentialKey + "}"
}

// HTTPSURL returns only a reviewed descriptor constant. It is not a caller URL
// parser or general endpoint-validation surface.
func (s Selection) HTTPSURL() (string, error) {
	if err := validateDescriptor(s.descriptor); err != nil {
		return "", err
	}
	return s.descriptor.endpoint, nil
}

func (s Selection) valid() bool { return validateDescriptor(s.descriptor) == nil }

func validateDescriptor(descriptor reviewedDescriptor) error {
	if !validPresetID(descriptor.presetID) ||
		descriptor.displayName == "" ||
		len(descriptor.displayName) > maxDisplayNameLength ||
		strings.TrimSpace(descriptor.displayName) != descriptor.displayName ||
		len(descriptor.description) > 1024 ||
		!validMetadataURL(descriptor.homepageURL) ||
		!validMetadataURL(descriptor.documentationURL) ||
		!descriptor.credential.valid() ||
		!validFixedHTTPSURL(descriptor.endpoint) {
		return ErrDescriptorInvalid
	}

	switch descriptor.credential {
	case CredentialClassNone:
		if descriptor.credentialKey != "" {
			return ErrDescriptorInvalid
		}
	case CredentialClassStaticBearer:
		if len(descriptor.credentialKey) > maxCredentialKeyLen || !credentialKeyPattern.MatchString(descriptor.credentialKey) {
			return ErrDescriptorInvalid
		}
	default:
		return ErrDescriptorInvalid
	}

	seen := make(map[string]struct{}, len(descriptor.allowedTools))
	for _, toolID := range descriptor.allowedTools {
		if !validToolID(toolID) {
			return ErrDescriptorInvalid
		}
		if _, exists := seen[toolID]; exists {
			return ErrDescriptorInvalid
		}
		seen[toolID] = struct{}{}
	}
	return nil
}

func validMetadataURL(value string) bool {
	if value == "" {
		return true
	}
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme == "https" && parsed.Host != "" && parsed.User == nil && parsed.Fragment == ""
}

func validPresetID(value string) bool {
	return len(value) <= maxPresetIDLength && presetIDPattern.MatchString(value)
}

func validFixedHTTPSURL(value string) bool {
	if value == "" || len(value) > maxEndpointLength || strings.TrimSpace(value) != value {
		return false
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Opaque != "" {
		return false
	}
	if parsed.Path == "" || !strings.HasPrefix(parsed.Path, "/") || path.Clean(parsed.Path) != parsed.Path {
		return false
	}
	return true
}

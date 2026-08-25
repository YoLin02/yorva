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
	presetID      string
	displayName   string
	endpoint      string
	credential    CredentialClass
	credentialKey string
	allowedTools  []string
}

// reviewedDescriptors is a value-producing function rather than mutable
// registry state. Adding an entry requires a source review and a code change.
// ADR-0014 is still Proposed, so the qualified set starts with zero entries.
func reviewedDescriptors() [0]reviewedDescriptor {
	return [0]reviewedDescriptor{}
}

// Preset is the safe catalog projection. Endpoint and credential-storage
// details remain adapter-owned.
type Preset struct {
	ID              string
	DisplayName     string
	CredentialClass CredentialClass
}

// Registry provides read-only access to the compile-time reviewed set.
type Registry struct{}

func NewRegistry() Registry { return Registry{} }

func (Registry) Catalog() []Preset {
	descriptors := reviewedDescriptors()
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
			ID:              descriptor.presetID,
			DisplayName:     descriptor.displayName,
			CredentialClass: descriptor.credential,
		})
	}
	return result
}

// Resolve accepts only an ID. It cannot accept or override URL, command, argv,
// environment, header, path, package, or bootstrap material.
func (Registry) Resolve(presetID string) (Selection, error) {
	if !validPresetID(presetID) {
		return Selection{}, ErrDescriptorUnknown
	}
	for _, descriptor := range reviewedDescriptors() {
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

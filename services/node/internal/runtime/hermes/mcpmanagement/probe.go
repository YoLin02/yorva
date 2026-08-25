package mcpmanagement

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"time"
)

const (
	ProbeSchemaVersion = 1
	MaxProbeBytes      = 16 * 1024
	MaxProbeTools      = 64
	MaxToolIDLength    = 128
	maxProbeJSONDepth  = 16
	maxProbeJSONValues = 256
)

var (
	ErrProbeMalformed = errors.New("Hermes MCP normalized probe result is malformed")
	ErrProbeLimit     = errors.New("Hermes MCP normalized probe result exceeds a limit")
	ErrProbeMismatch  = errors.New("Hermes MCP normalized probe result does not match its exact target")

	toolIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,127}$`)
)

type ProbeOutcome string

const (
	ProbeOutcomeReady        ProbeOutcome = "READY"
	ProbeOutcomeAuthRequired ProbeOutcome = "AUTH_REQUIRED"
	ProbeOutcomeFailed       ProbeOutcome = "FAILED"
	ProbeOutcomeUnknown      ProbeOutcome = "UNKNOWN"
	ProbeOutcomeTimedOut     ProbeOutcome = "TIMED_OUT"
)

func (o ProbeOutcome) valid() bool {
	switch o {
	case ProbeOutcomeReady, ProbeOutcomeAuthRequired, ProbeOutcomeFailed, ProbeOutcomeUnknown, ProbeOutcomeTimedOut:
		return true
	default:
		return false
	}
}

// ProbeParser parses an already-acquired normalized result. It does not own or
// execute network, CLI, Hermes, or child-process work.
type ProbeParser interface {
	Parse([]byte) (ProbeResult, error)
}

type normalizedProbeParser struct {
	selection Selection
	scope     ProfileScope
}

func (s Selection) NewProbeParser(scope ProfileScope) (ProbeParser, error) {
	if !s.valid() {
		return nil, ErrDescriptorInvalid
	}
	if !scope.valid() {
		return nil, ErrProfileScopeInvalid
	}
	return normalizedProbeParser{selection: s, scope: scope}, nil
}

type probeDocument struct {
	SchemaVersion   int              `json:"schemaVersion"`
	ProfileID       string           `json:"profileId"`
	PresetID        string           `json:"presetId"`
	Outcome         ProbeOutcome     `json:"outcome"`
	CredentialState CredentialStatus `json:"credentialStatus"`
	Connected       bool             `json:"connected"`
	Negotiated      bool             `json:"negotiated"`
	CleanupComplete bool             `json:"cleanupComplete"`
	ToolIDs         []string         `json:"toolIds"`
	ObservedAt      time.Time        `json:"observedAt"`
}

// ProbeResult contains only bounded normalized metadata. It cannot contain a
// credential, endpoint, header, raw tool description, or raw error text.
type ProbeResult struct {
	profileID        string
	presetID         string
	outcome          ProbeOutcome
	credentialStatus CredentialStatus
	toolIDs          []string
	observedAt       time.Time
}

func (r ProbeResult) ProfileID() string { return r.profileID }

func (r ProbeResult) PresetID() string { return r.presetID }

func (r ProbeResult) Outcome() ProbeOutcome { return r.outcome }

func (r ProbeResult) CredentialStatus() CredentialStatus { return r.credentialStatus }

func (r ProbeResult) ToolIDs() []string { return append([]string(nil), r.toolIDs...) }

func (r ProbeResult) ObservedAt() time.Time { return r.observedAt }

func (p normalizedProbeParser) Parse(input []byte) (ProbeResult, error) {
	if len(input) == 0 {
		return ProbeResult{}, ErrProbeMalformed
	}
	if len(input) > MaxProbeBytes {
		return ProbeResult{}, ErrProbeLimit
	}
	if err := validateUniqueJSONKeys(input); err != nil {
		return ProbeResult{}, err
	}

	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.DisallowUnknownFields()
	var document probeDocument
	if err := decoder.Decode(&document); err != nil {
		return ProbeResult{}, ErrProbeMalformed
	}
	if err := requireJSONEOF(decoder); err != nil {
		return ProbeResult{}, ErrProbeMalformed
	}
	if document.SchemaVersion != ProbeSchemaVersion || !document.Outcome.valid() || document.ObservedAt.IsZero() {
		return ProbeResult{}, ErrProbeMalformed
	}
	if document.ProfileID != p.scope.profileID || document.PresetID != p.selection.descriptor.presetID {
		return ProbeResult{}, ErrProbeMismatch
	}
	if err := p.selection.ValidateCredentialStatus(document.CredentialState); err != nil {
		return ProbeResult{}, ErrProbeMalformed
	}
	if len(document.ToolIDs) > MaxProbeTools {
		return ProbeResult{}, ErrProbeLimit
	}
	if err := validateProbeSemantics(p.selection.descriptor, document); err != nil {
		return ProbeResult{}, err
	}

	return ProbeResult{
		profileID:        document.ProfileID,
		presetID:         document.PresetID,
		outcome:          document.Outcome,
		credentialStatus: document.CredentialState,
		toolIDs:          append([]string(nil), document.ToolIDs...),
		observedAt:       document.ObservedAt.UTC(),
	}, nil
}

type jsonScanBudget struct {
	values int
}

// validateUniqueJSONKeys rejects ambiguous duplicate object members at every
// nesting level before the ordinary typed decoder runs. Its recursion and
// total value count are independently bounded even though the byte input has
// already passed MaxProbeBytes.
func validateUniqueJSONKeys(input []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(input))
	decoder.UseNumber()
	budget := jsonScanBudget{}
	if err := scanJSONValue(decoder, 0, &budget); err != nil {
		return err
	}
	if err := requireJSONEOF(decoder); err != nil {
		return ErrProbeMalformed
	}
	return nil
}

func scanJSONValue(decoder *json.Decoder, depth int, budget *jsonScanBudget) error {
	if depth > maxProbeJSONDepth {
		return ErrProbeLimit
	}
	budget.values++
	if budget.values > maxProbeJSONValues {
		return ErrProbeLimit
	}

	token, err := decoder.Token()
	if err != nil {
		return ErrProbeMalformed
	}
	delimiter, composite := token.(json.Delim)
	if !composite {
		return nil
	}

	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, keyErr := decoder.Token()
			if keyErr != nil {
				return ErrProbeMalformed
			}
			key, ok := keyToken.(string)
			if !ok {
				return ErrProbeMalformed
			}
			if _, duplicate := seen[key]; duplicate {
				return ErrProbeMalformed
			}
			seen[key] = struct{}{}
			if err := scanJSONValue(decoder, depth+1, budget); err != nil {
				return err
			}
		}
		closing, closeErr := decoder.Token()
		if closeErr != nil || closing != json.Delim('}') {
			return ErrProbeMalformed
		}
	case '[':
		for decoder.More() {
			if err := scanJSONValue(decoder, depth+1, budget); err != nil {
				return err
			}
		}
		closing, closeErr := decoder.Token()
		if closeErr != nil || closing != json.Delim(']') {
			return ErrProbeMalformed
		}
	default:
		return ErrProbeMalformed
	}
	return nil
}

func validateProbeSemantics(descriptor reviewedDescriptor, document probeDocument) error {
	seen := make(map[string]struct{}, len(document.ToolIDs))
	allowed := make(map[string]struct{}, len(descriptor.allowedTools))
	for _, toolID := range descriptor.allowedTools {
		allowed[toolID] = struct{}{}
	}
	for _, toolID := range document.ToolIDs {
		if !validToolID(toolID) {
			return ErrProbeMalformed
		}
		if _, duplicate := seen[toolID]; duplicate {
			return ErrProbeMalformed
		}
		if _, approved := allowed[toolID]; !approved {
			return ErrProbeMismatch
		}
		seen[toolID] = struct{}{}
	}

	if document.Outcome == ProbeOutcomeReady {
		if !document.Connected || !document.Negotiated || !document.CleanupComplete {
			return ErrProbeMalformed
		}
		if descriptor.credential == CredentialClassStaticBearer && document.CredentialState != CredentialStatusConfigured {
			return ErrProbeMalformed
		}
		return nil
	}

	if document.Connected || document.Negotiated || len(document.ToolIDs) != 0 {
		return ErrProbeMalformed
	}
	if document.Outcome == ProbeOutcomeAuthRequired &&
		(descriptor.credential != CredentialClassStaticBearer || document.CredentialState != CredentialStatusNotConfigured) {
		return ErrProbeMalformed
	}
	return nil
}

func validToolID(value string) bool {
	return len(value) <= MaxToolIDLength && toolIDPattern.MatchString(value)
}

func requireJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return ErrProbeMalformed
	}
	return nil
}

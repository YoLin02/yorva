package backupmanagement

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
	"slices"
	"strconv"
	"strings"
)

const (
	FormatVersion         = "yorva.hermes.runtime-backup.v1"
	ManifestSchemaVersion = 1
	ManifestPolicyVersion = 1
	RuntimeScope          = "runtime"
	RuntimeKind           = "hermes"
	PayloadArchivePath    = "payload/hermes-runtime.zip"
	PayloadArchiveRoot    = "hermes-runtime"
	MaxManifestBytes      = 64 << 10
)

type DataCategory string

const (
	CategoryAccountState  DataCategory = "account-state"
	CategoryChannelState  DataCategory = "channel-state"
	CategoryConfiguration DataCategory = "configuration"
	CategoryConversations DataCategory = "conversations"
	CategoryCredentials   DataCategory = "credentials"
	CategoryDatabases     DataCategory = "databases"
	CategoryMCPOAuth      DataCategory = "mcp-oauth"
	CategoryPairingState  DataCategory = "pairing-state"
	CategorySessions      DataCategory = "sessions"
	CategorySkills        DataCategory = "skills"
)

var knownCategories = map[DataCategory]struct{}{
	CategoryAccountState: {}, CategoryChannelState: {}, CategoryConfiguration: {},
	CategoryConversations: {}, CategoryCredentials: {}, CategoryDatabases: {},
	CategoryMCPOAuth: {}, CategoryPairingState: {}, CategorySessions: {}, CategorySkills: {},
}

// Manifest is the bounded canonical description of one decrypted proposed
// backup container. It contains metadata only and must never contain secrets.
type Manifest struct {
	FormatVersion      string         `json:"formatVersion"`
	SchemaVersion      int            `json:"schemaVersion"`
	PolicyVersion      int            `json:"policyVersion"`
	Scope              string         `json:"scope"`
	RuntimeKind        string         `json:"runtimeKind"`
	RuntimeVersion     string         `json:"runtimeVersion"`
	IncludedCategories []DataCategory `json:"includedCategories"`
	Payload            Payload        `json:"payload"`
	Complete           bool           `json:"complete"`
}

type Payload struct {
	Path               string `json:"path"`
	SizeBytes          int64  `json:"sizeBytes"`
	SHA256             string `json:"sha256"`
	MemberCount        int    `json:"memberCount"`
	TotalExpandedBytes int64  `json:"totalExpandedBytes"`
}

// Validate checks the immutable proposed-format invariants. Operational limits
// may only tighten these hard format bounds during verification.
func (m Manifest) Validate() error {
	if m.FormatVersion != FormatVersion || m.SchemaVersion != ManifestSchemaVersion || m.PolicyVersion != ManifestPolicyVersion {
		return verificationError(ErrorFormatUnsupported)
	}
	if m.Scope != RuntimeScope {
		return verificationError(ErrorScopeUnsupported)
	}
	if m.RuntimeKind != RuntimeKind || !validRuntimeVersion(m.RuntimeVersion) {
		return verificationError(ErrorRuntimeUnsupported)
	}
	if len(m.IncludedCategories) == 0 || len(m.IncludedCategories) > len(knownCategories) {
		return verificationError(ErrorManifestInvalid)
	}
	if !slices.IsSorted(m.IncludedCategories) {
		return verificationError(ErrorManifestInvalid)
	}
	for index, category := range m.IncludedCategories {
		if _, ok := knownCategories[category]; !ok {
			return verificationError(ErrorManifestInvalid)
		}
		if index > 0 && m.IncludedCategories[index-1] == category {
			return verificationError(ErrorManifestInvalid)
		}
	}
	if m.Payload.Path != PayloadArchivePath || m.Payload.SizeBytes <= 0 || m.Payload.SizeBytes > hardMaxPayloadBytes ||
		m.Payload.MemberCount <= 0 || m.Payload.MemberCount > hardMaxPayloadMembers ||
		m.Payload.TotalExpandedBytes < 0 || m.Payload.TotalExpandedBytes > hardMaxTotalExpandedBytes ||
		!validCanonicalSHA256(m.Payload.SHA256) || !m.Complete {
		return verificationError(ErrorManifestInvalid)
	}
	return nil
}

// MarshalCanonical validates and emits the only accepted byte representation:
// compact UTF-8 JSON in struct field order, with sorted category values.
func (m Manifest) MarshalCanonical() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	payload, err := json.Marshal(m)
	if err != nil || len(payload) > MaxManifestBytes {
		return nil, verificationError(ErrorManifestInvalid)
	}
	return payload, nil
}

// ParseCanonicalManifest rejects unknown fields, duplicate fields, alternate
// field order, insignificant whitespace and trailing data by byte-comparing the
// decoded value with its canonical encoding.
func ParseCanonicalManifest(payload []byte) (Manifest, error) {
	if len(payload) == 0 || len(payload) > MaxManifestBytes {
		return Manifest{}, verificationError(ErrorManifestInvalid)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, verificationError(ErrorManifestInvalid)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return Manifest{}, verificationError(ErrorManifestInvalid)
	}
	canonical, err := manifest.MarshalCanonical()
	if err != nil {
		return Manifest{}, err
	}
	if !bytes.Equal(payload, canonical) {
		return Manifest{}, verificationError(ErrorManifestNotCanonical)
	}
	return manifest, nil
}

func validCanonicalSHA256(value string) bool {
	if len(value) != 64 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validRuntimeVersion(value string) bool {
	if len(value) == 0 || len(value) > 64 {
		return false
	}
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" || (len(part) > 1 && part[0] == '0') {
			return false
		}
		if _, err := strconv.ParseUint(part, 10, 32); err != nil {
			return false
		}
	}
	return true
}

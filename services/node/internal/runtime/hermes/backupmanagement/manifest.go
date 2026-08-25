package backupmanagement

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	FormatVersion         = "yorva.hermes.runtime-backup.v1"
	ManifestSchemaVersion = 2
	ManifestPolicyVersion = 1
	RuntimeScope          = "runtime"
	RuntimeKind           = "hermes"
	InclusionPolicyID     = "hermes-runtime-user-data-v1"
	ExclusionPolicyID     = "hermes-runtime-non-user-data-v1"
	PayloadArchivePath    = "payload/hermes-runtime.zip"
	PayloadArchiveRoot    = "hermes-runtime"
	MaxManifestBytes      = 64 << 10
)

const (
	backupIDPrefix         = "backup_"
	backupIDBodyLength     = 22
	generationIDPrefix     = "gen_"
	generationIDBodyLength = 22
)

type InstallationState string

const (
	InstallationManaged   InstallationState = "MANAGED"
	InstallationUnmanaged InstallationState = "UNMANAGED"
	InstallationUnknown   InstallationState = "UNKNOWN"
)

type ExcludedDataCategory string

const (
	ExcludedBrowserProfiles         ExcludedDataCategory = "browser-profiles"
	ExcludedInstallationBinaries    ExcludedDataCategory = "installation-binaries"
	ExcludedPriorBackups            ExcludedDataCategory = "prior-backups"
	ExcludedRegenerableDependencies ExcludedDataCategory = "regenerable-dependencies"
	ExcludedRuntimeTemporaryState   ExcludedDataCategory = "runtime-temporary-state"
	ExcludedYORVAManagementState    ExcludedDataCategory = "yorva-management-state"
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

var completeRuntimeSnapshotCategories = []DataCategory{
	CategoryAccountState,
	CategoryChannelState,
	CategoryConfiguration,
	CategoryConversations,
	CategoryCredentials,
	CategoryDatabases,
	CategoryMCPOAuth,
	CategoryPairingState,
	CategorySessions,
	CategorySkills,
}

var knownExcludedCategories = map[ExcludedDataCategory]struct{}{
	ExcludedBrowserProfiles: {}, ExcludedInstallationBinaries: {}, ExcludedPriorBackups: {},
	ExcludedRegenerableDependencies: {}, ExcludedRuntimeTemporaryState: {}, ExcludedYORVAManagementState: {},
}

var completeRuntimeSnapshotExcludedCategories = []ExcludedDataCategory{
	ExcludedBrowserProfiles,
	ExcludedInstallationBinaries,
	ExcludedPriorBackups,
	ExcludedRegenerableDependencies,
	ExcludedRuntimeTemporaryState,
	ExcludedYORVAManagementState,
}

type InstallationIdentity struct {
	State        InstallationState `json:"state"`
	GenerationID string            `json:"generationId,omitempty"`
	SourcePin    string            `json:"sourcePin,omitempty"`
}

// Manifest is the bounded canonical description of one decrypted proposed
// backup container. It contains metadata only and must never contain secrets.
type Manifest struct {
	FormatVersion      string                 `json:"formatVersion"`
	SchemaVersion      int                    `json:"schemaVersion"`
	PolicyVersion      int                    `json:"policyVersion"`
	BackupID           string                 `json:"backupId"`
	CreatedAt          string                 `json:"createdAt"`
	Scope              string                 `json:"scope"`
	RuntimeKind        string                 `json:"runtimeKind"`
	RuntimeVersion     string                 `json:"runtimeVersion"`
	Installation       InstallationIdentity   `json:"installation"`
	InclusionPolicy    string                 `json:"inclusionPolicy"`
	IncludedCategories []DataCategory         `json:"includedCategories"`
	ExclusionPolicy    string                 `json:"exclusionPolicy"`
	ExcludedCategories []ExcludedDataCategory `json:"excludedCategories"`
	Payload            Payload                `json:"payload"`
	Complete           bool                   `json:"complete"`
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
	if !validClosedID(m.BackupID, backupIDPrefix, backupIDBodyLength) || !validCanonicalCreatedAt(m.CreatedAt) || !m.Installation.valid() {
		return verificationError(ErrorManifestInvalid)
	}
	if m.InclusionPolicy != InclusionPolicyID || m.ExclusionPolicy != ExclusionPolicyID {
		return verificationError(ErrorFormatUnsupported)
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
	if !slices.Equal(m.IncludedCategories, completeRuntimeSnapshotCategories) {
		return verificationError(ErrorManifestInvalid)
	}
	if len(m.ExcludedCategories) == 0 || len(m.ExcludedCategories) > len(knownExcludedCategories) || !slices.IsSorted(m.ExcludedCategories) {
		return verificationError(ErrorManifestInvalid)
	}
	for index, category := range m.ExcludedCategories {
		if _, ok := knownExcludedCategories[category]; !ok {
			return verificationError(ErrorManifestInvalid)
		}
		if index > 0 && m.ExcludedCategories[index-1] == category {
			return verificationError(ErrorManifestInvalid)
		}
	}
	if !slices.Equal(m.ExcludedCategories, completeRuntimeSnapshotExcludedCategories) {
		return verificationError(ErrorManifestInvalid)
	}
	if m.Payload.Path != PayloadArchivePath || m.Payload.SizeBytes <= 0 || m.Payload.SizeBytes > hardMaxPayloadBytes ||
		m.Payload.MemberCount <= 0 || m.Payload.MemberCount > hardMaxPayloadMembers ||
		m.Payload.TotalExpandedBytes < 0 || m.Payload.TotalExpandedBytes > hardMaxTotalExpandedBytes ||
		!validCanonicalSHA256(m.Payload.SHA256) || !m.Complete {
		return verificationError(ErrorManifestInvalid)
	}
	return nil
}

func (i InstallationIdentity) valid() bool {
	switch i.State {
	case InstallationManaged:
		return validClosedID(i.GenerationID, generationIDPrefix, generationIDBodyLength) && validSourcePin(i.SourcePin)
	case InstallationUnmanaged, InstallationUnknown:
		return i.GenerationID == "" && i.SourcePin == ""
	default:
		return false
	}
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

func validSourcePin(value string) bool {
	return len(value) == 40 && value == strings.ToLower(value) && isHex(value)
}

func validClosedID(value, prefix string, bodyLength int) bool {
	if len(value) != len(prefix)+bodyLength || !strings.HasPrefix(value, prefix) {
		return false
	}
	for _, character := range value[len(prefix):] {
		if !(character >= 'a' && character <= 'z') && !(character >= '2' && character <= '7') {
			return false
		}
	}
	return true
}

func validCanonicalCreatedAt(value string) bool {
	parsed, err := time.Parse(time.RFC3339, value)
	return err == nil && parsed.Location() == time.UTC && parsed.Nanosecond() == 0 && parsed.Format(time.RFC3339) == value
}

func isHex(value string) bool {
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

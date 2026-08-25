package backupmanagement

import (
	"bytes"
	"slices"
	"strings"
	"testing"
)

func TestManifestCanonicalRoundTrip(t *testing.T) {
	manifest := validTestManifest()
	payload, err := manifest.MarshalCanonical()
	if err != nil {
		t.Fatalf("MarshalCanonical() error = %v", err)
	}
	parsed, err := ParseCanonicalManifest(payload)
	if err != nil {
		t.Fatalf("ParseCanonicalManifest() error = %v", err)
	}
	if parsed.RuntimeVersion != manifest.RuntimeVersion || parsed.BackupID != manifest.BackupID || parsed.Installation != manifest.Installation || parsed.Payload != manifest.Payload {
		t.Fatalf("parsed manifest = %#v, want %#v", parsed, manifest)
	}
	if !bytes.Equal(payload, []byte(`{"formatVersion":"yorva.hermes.runtime-backup.v1","schemaVersion":2,"policyVersion":1,"backupId":"backup_aaaaaaaaaaaaaaaaaaaaaa","createdAt":"2026-08-25T01:02:03Z","scope":"runtime","runtimeKind":"hermes","runtimeVersion":"0.20.5","installation":{"state":"MANAGED","generationId":"gen_aaaaaaaaaaaaaaaaaaaaaa","sourcePin":"a0ca7c19204e514f9590ce3b812e029b315ab9e9"},"inclusionPolicy":"hermes-runtime-user-data-v1","includedCategories":["account-state","channel-state","configuration","conversations","credentials","databases","mcp-oauth","pairing-state","sessions","skills"],"exclusionPolicy":"hermes-runtime-non-user-data-v1","excludedCategories":["browser-profiles","installation-binaries","prior-backups","regenerable-dependencies","runtime-temporary-state","yorva-management-state"],"payload":{"path":"payload/hermes-runtime.zip","sizeBytes":128,"sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","memberCount":2,"totalExpandedBytes":64},"complete":true}`)) {
		t.Fatalf("canonical payload was not exact: %s", payload)
	}
}

func TestParseCanonicalManifestRejectsAlternateEncodingAndUnknownFields(t *testing.T) {
	canonical, err := validTestManifest().MarshalCanonical()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		payload []byte
		code    ErrorCode
	}{
		{name: "whitespace", payload: append([]byte(" "), canonical...), code: ErrorManifestNotCanonical},
		{name: "trailing", payload: append(append([]byte(nil), canonical...), '\n'), code: ErrorManifestNotCanonical},
		{name: "duplicate field", payload: []byte(strings.Replace(string(canonical), `"backupId":"backup_aaaaaaaaaaaaaaaaaaaaaa"`, `"backupId":"backup_aaaaaaaaaaaaaaaaaaaaaa","backupId":"backup_aaaaaaaaaaaaaaaaaaaaaa"`, 1)), code: ErrorManifestNotCanonical},
		{name: "unknown field", payload: []byte(strings.Replace(string(canonical), `,"complete":true}`, `,"unknown":true,"complete":true}`, 1)), code: ErrorManifestInvalid},
		{name: "pre-completeness schema", payload: []byte(strings.Replace(string(canonical), `"schemaVersion":2`, `"schemaVersion":1`, 1)), code: ErrorFormatUnsupported},
		{name: "oversized", payload: bytes.Repeat([]byte("x"), MaxManifestBytes+1), code: ErrorManifestInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ParseCanonicalManifest(test.payload)
			assertErrorCode(t, err, test.code)
		})
	}
}

func TestManifestValidationRejectsUnsupportedOrUnboundedMetadata(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Manifest)
		code   ErrorCode
	}{
		{name: "format", mutate: func(m *Manifest) { m.FormatVersion = "v2" }, code: ErrorFormatUnsupported},
		{name: "schema", mutate: func(m *Manifest) { m.SchemaVersion = 3 }, code: ErrorFormatUnsupported},
		{name: "scope", mutate: func(m *Manifest) { m.Scope = "instance" }, code: ErrorScopeUnsupported},
		{name: "backup id", mutate: func(m *Manifest) { m.BackupID = "backup_short" }, code: ErrorManifestInvalid},
		{name: "created at fractional", mutate: func(m *Manifest) { m.CreatedAt = "2026-08-25T01:02:03.1Z" }, code: ErrorManifestInvalid},
		{name: "created at non UTC", mutate: func(m *Manifest) { m.CreatedAt = "2026-08-25T09:02:03+08:00" }, code: ErrorManifestInvalid},
		{name: "runtime kind", mutate: func(m *Manifest) { m.RuntimeKind = "other" }, code: ErrorRuntimeUnsupported},
		{name: "runtime version", mutate: func(m *Manifest) { m.RuntimeVersion = "0.20" }, code: ErrorRuntimeUnsupported},
		{name: "inclusion policy", mutate: func(m *Manifest) { m.InclusionPolicy = "other" }, code: ErrorFormatUnsupported},
		{name: "exclusion policy", mutate: func(m *Manifest) { m.ExclusionPolicy = "other" }, code: ErrorFormatUnsupported},
		{name: "unknown installation state", mutate: func(m *Manifest) { m.Installation.State = "OTHER" }, code: ErrorManifestInvalid},
		{name: "managed missing generation", mutate: func(m *Manifest) { m.Installation.GenerationID = "" }, code: ErrorManifestInvalid},
		{name: "managed bad source pin", mutate: func(m *Manifest) { m.Installation.SourcePin = strings.Repeat("g", 40) }, code: ErrorManifestInvalid},
		{name: "unmanaged invented identity", mutate: func(m *Manifest) { m.Installation.State = InstallationUnmanaged }, code: ErrorManifestInvalid},
		{name: "unknown category", mutate: func(m *Manifest) { m.IncludedCategories = []DataCategory{"unknown"} }, code: ErrorManifestInvalid},
		{name: "unsorted categories", mutate: func(m *Manifest) { m.IncludedCategories = []DataCategory{CategoryCredentials, CategoryConfiguration} }, code: ErrorManifestInvalid},
		{name: "duplicate categories", mutate: func(m *Manifest) { m.IncludedCategories = []DataCategory{CategoryCredentials, CategoryCredentials} }, code: ErrorManifestInvalid},
		{name: "incomplete category policy", mutate: func(m *Manifest) { m.IncludedCategories = m.IncludedCategories[:len(m.IncludedCategories)-1] }, code: ErrorManifestInvalid},
		{name: "unknown excluded category", mutate: func(m *Manifest) { m.ExcludedCategories = []ExcludedDataCategory{"unknown"} }, code: ErrorManifestInvalid},
		{name: "unsorted excluded categories", mutate: func(m *Manifest) {
			m.ExcludedCategories = []ExcludedDataCategory{ExcludedYORVAManagementState, ExcludedBrowserProfiles}
		}, code: ErrorManifestInvalid},
		{name: "duplicate excluded categories", mutate: func(m *Manifest) {
			m.ExcludedCategories = []ExcludedDataCategory{ExcludedPriorBackups, ExcludedPriorBackups}
		}, code: ErrorManifestInvalid},
		{name: "incomplete exclusion policy", mutate: func(m *Manifest) { m.ExcludedCategories = m.ExcludedCategories[:len(m.ExcludedCategories)-1] }, code: ErrorManifestInvalid},
		{name: "uppercase checksum", mutate: func(m *Manifest) { m.Payload.SHA256 = strings.ToUpper(m.Payload.SHA256) }, code: ErrorManifestInvalid},
		{name: "payload size", mutate: func(m *Manifest) { m.Payload.SizeBytes = hardMaxPayloadBytes + 1 }, code: ErrorManifestInvalid},
		{name: "member count", mutate: func(m *Manifest) { m.Payload.MemberCount = hardMaxPayloadMembers + 1 }, code: ErrorManifestInvalid},
		{name: "incomplete", mutate: func(m *Manifest) { m.Complete = false }, code: ErrorManifestInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest := validTestManifest()
			test.mutate(&manifest)
			assertErrorCode(t, manifest.Validate(), test.code)
		})
	}
}

func TestManifestAcceptsExplicitUnmanagedAndUnknownInstallationStates(t *testing.T) {
	for _, state := range []InstallationState{InstallationUnmanaged, InstallationUnknown} {
		t.Run(string(state), func(t *testing.T) {
			manifest := validTestManifest()
			manifest.Installation = InstallationIdentity{State: state}
			if err := manifest.Validate(); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

func validTestManifest() Manifest {
	return Manifest{
		FormatVersion:  FormatVersion,
		SchemaVersion:  ManifestSchemaVersion,
		PolicyVersion:  ManifestPolicyVersion,
		BackupID:       "backup_aaaaaaaaaaaaaaaaaaaaaa",
		CreatedAt:      "2026-08-25T01:02:03Z",
		Scope:          RuntimeScope,
		RuntimeKind:    RuntimeKind,
		RuntimeVersion: "0.20.5",
		Installation: InstallationIdentity{
			State: InstallationManaged, GenerationID: "gen_aaaaaaaaaaaaaaaaaaaaaa",
			SourcePin: "a0ca7c19204e514f9590ce3b812e029b315ab9e9",
		},
		InclusionPolicy:    InclusionPolicyID,
		IncludedCategories: slices.Clone(completeRuntimeSnapshotCategories),
		ExclusionPolicy:    ExclusionPolicyID,
		ExcludedCategories: slices.Clone(completeRuntimeSnapshotExcludedCategories),
		Payload: Payload{
			Path:               PayloadArchivePath,
			SizeBytes:          128,
			SHA256:             strings.Repeat("a", 64),
			MemberCount:        2,
			TotalExpandedBytes: 64,
		},
		Complete: true,
	}
}

func assertErrorCode(t *testing.T, err error, want ErrorCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want code %s", want)
	}
	got, ok := ErrorCodeOf(err)
	if !ok || got != want {
		t.Fatalf("error = %v, code = %q, want %q", err, got, want)
	}
}

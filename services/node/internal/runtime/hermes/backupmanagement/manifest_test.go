package backupmanagement

import (
	"bytes"
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
	if parsed.RuntimeVersion != manifest.RuntimeVersion || parsed.Payload != manifest.Payload {
		t.Fatalf("parsed manifest = %#v, want %#v", parsed, manifest)
	}
	if !bytes.Equal(payload, []byte(`{"formatVersion":"yorva.hermes.runtime-backup.v1","schemaVersion":1,"policyVersion":1,"scope":"runtime","runtimeKind":"hermes","runtimeVersion":"0.20.5","includedCategories":["configuration","credentials"],"payload":{"path":"payload/hermes-runtime.zip","sizeBytes":128,"sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","memberCount":2,"totalExpandedBytes":64},"complete":true}`)) {
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
		{name: "unknown field", payload: []byte(strings.Replace(string(canonical), `,"complete":true}`, `,"unknown":true,"complete":true}`, 1)), code: ErrorManifestInvalid},
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
		{name: "schema", mutate: func(m *Manifest) { m.SchemaVersion = 2 }, code: ErrorFormatUnsupported},
		{name: "scope", mutate: func(m *Manifest) { m.Scope = "instance" }, code: ErrorScopeUnsupported},
		{name: "runtime kind", mutate: func(m *Manifest) { m.RuntimeKind = "other" }, code: ErrorRuntimeUnsupported},
		{name: "runtime version", mutate: func(m *Manifest) { m.RuntimeVersion = "0.20" }, code: ErrorRuntimeUnsupported},
		{name: "unknown category", mutate: func(m *Manifest) { m.IncludedCategories = []DataCategory{"unknown"} }, code: ErrorManifestInvalid},
		{name: "unsorted categories", mutate: func(m *Manifest) { m.IncludedCategories = []DataCategory{CategoryCredentials, CategoryConfiguration} }, code: ErrorManifestInvalid},
		{name: "duplicate categories", mutate: func(m *Manifest) { m.IncludedCategories = []DataCategory{CategoryCredentials, CategoryCredentials} }, code: ErrorManifestInvalid},
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

func validTestManifest() Manifest {
	return Manifest{
		FormatVersion:      FormatVersion,
		SchemaVersion:      ManifestSchemaVersion,
		PolicyVersion:      ManifestPolicyVersion,
		Scope:              RuntimeScope,
		RuntimeKind:        RuntimeKind,
		RuntimeVersion:     "0.20.5",
		IncludedCategories: []DataCategory{CategoryConfiguration, CategoryCredentials},
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

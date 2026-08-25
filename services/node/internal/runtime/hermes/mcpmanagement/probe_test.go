package mcpmanagement

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNormalizedProbeParserAcceptsBoundedReadyResult(t *testing.T) {
	parser, scope := newTestProbeParser(t, CredentialClassStaticBearer)
	observedAt := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)
	document := validProbeDocument(scope, observedAt)

	result, err := parser.Parse(mustProbeJSON(t, document))
	if err != nil {
		t.Fatalf("Parse(valid): %v", err)
	}
	if result.ProfileID() != scope.ProfileID() || result.PresetID() != "unit-test-only" || result.Outcome() != ProbeOutcomeReady {
		t.Fatalf("unexpected result: profile=%q preset=%q outcome=%q", result.ProfileID(), result.PresetID(), result.Outcome())
	}
	if result.CredentialStatus() != CredentialStatusConfigured || !result.ObservedAt().Equal(observedAt) {
		t.Fatalf("unexpected credential/time: %q %v", result.CredentialStatus(), result.ObservedAt())
	}
	tools := result.ToolIDs()
	if len(tools) != 2 || tools[0] != "read" || tools[1] != "search" {
		t.Fatalf("ToolIDs() = %#v", tools)
	}
	tools[0] = "caller-mutated"
	if result.ToolIDs()[0] != "read" {
		t.Fatal("ToolIDs returned mutable result storage")
	}
}

func TestNormalizedProbeParserAcceptsNoAuthReadyResult(t *testing.T) {
	parser, scope := newTestProbeParser(t, CredentialClassNone)
	document := validProbeDocument(scope, time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC))
	document.CredentialState = CredentialStatusNotRequired

	result, err := parser.Parse(mustProbeJSON(t, document))
	if err != nil {
		t.Fatalf("Parse(no-auth READY): %v", err)
	}
	if result.Outcome() != ProbeOutcomeReady || result.CredentialStatus() != CredentialStatusNotRequired {
		t.Fatalf("no-auth result = %q / %q", result.Outcome(), result.CredentialStatus())
	}
}

func TestNormalizedProbeParserRejectsInvalidUnknownAndMismatchedInput(t *testing.T) {
	parser, scope := newTestProbeParser(t, CredentialClassStaticBearer)
	now := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)

	if _, err := parser.Parse([]byte(`{"schemaVersion":`)); !errors.Is(err, ErrProbeMalformed) {
		t.Fatalf("malformed JSON error = %v", err)
	}

	unknownOutcome := validProbeDocument(scope, now)
	unknownOutcome.Outcome = ProbeOutcome("SURPRISING")
	if _, err := parser.Parse(mustProbeJSON(t, unknownOutcome)); !errors.Is(err, ErrProbeMalformed) {
		t.Fatalf("unknown outcome error = %v", err)
	}

	mismatch := validProbeDocument(scope, now)
	mismatch.ProfileID = "other"
	if _, err := parser.Parse(mustProbeJSON(t, mismatch)); !errors.Is(err, ErrProbeMismatch) {
		t.Fatalf("Profile mismatch error = %v", err)
	}

	notReady := validProbeDocument(scope, now)
	notReady.Negotiated = false
	if _, err := parser.Parse(mustProbeJSON(t, notReady)); !errors.Is(err, ErrProbeMalformed) {
		t.Fatalf("false READY error = %v", err)
	}

	callerOverrides := []byte(`{"schemaVersion":1,"profileId":"named","presetId":"unit-test-only","outcome":"READY","credentialStatus":"CONFIGURED","connected":true,"negotiated":true,"cleanupComplete":true,"toolIds":[],"observedAt":"2026-08-25T10:00:00Z","url":"https://caller.example","command":"run","argv":[],"env":{},"header":{},"path":"x","package":"x","bootstrap":"x"}`)
	if _, err := parser.Parse(callerOverrides); !errors.Is(err, ErrProbeMalformed) {
		t.Fatalf("caller override fields error = %v, want ErrProbeMalformed", err)
	}
}

func TestNormalizedProbeParserRejectsDuplicateJSONKeysAtEveryDepth(t *testing.T) {
	parser, scope := newTestProbeParser(t, CredentialClassStaticBearer)
	valid := string(mustProbeJSON(t, validProbeDocument(scope, time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC))))
	if _, err := parser.Parse([]byte(valid)); err != nil {
		t.Fatalf("valid JSON rejected: %v", err)
	}

	topLevelDuplicate := strings.Replace(valid, `"profileId":"named"`, `"profileId":"named","profileId":"other"`, 1)
	if _, err := parser.Parse([]byte(topLevelDuplicate)); !errors.Is(err, ErrProbeMalformed) {
		t.Fatalf("top-level duplicate error = %v, want ErrProbeMalformed", err)
	}

	nestedDuplicate := strings.TrimSuffix(valid, "}") + `,"future":{"key":1,"key":2}}`
	if _, err := parser.Parse([]byte(nestedDuplicate)); !errors.Is(err, ErrProbeMalformed) {
		t.Fatalf("nested duplicate error = %v, want ErrProbeMalformed", err)
	}
}

func TestNormalizedProbeParserEnforcesLimits(t *testing.T) {
	parser, scope := newTestProbeParser(t, CredentialClassStaticBearer)
	now := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)

	if _, err := parser.Parse([]byte(strings.Repeat("x", MaxProbeBytes+1))); !errors.Is(err, ErrProbeLimit) {
		t.Fatalf("oversized input error = %v", err)
	}

	document := validProbeDocument(scope, now)
	document.ToolIDs = make([]string, MaxProbeTools+1)
	for index := range document.ToolIDs {
		document.ToolIDs[index] = "read"
	}
	if _, err := parser.Parse(mustProbeJSON(t, document)); !errors.Is(err, ErrProbeLimit) {
		t.Fatalf("tool count error = %v", err)
	}

	document = validProbeDocument(scope, now)
	document.ToolIDs = []string{strings.Repeat("x", MaxToolIDLength+1)}
	if _, err := parser.Parse(mustProbeJSON(t, document)); !errors.Is(err, ErrProbeMalformed) {
		t.Fatalf("tool length error = %v", err)
	}
}

func newTestProbeParser(t *testing.T, class CredentialClass) (ProbeParser, ProfileScope) {
	t.Helper()
	scope, err := NewProfileScope("named")
	if err != nil {
		t.Fatal(err)
	}
	parser, err := (Selection{descriptor: testDescriptor(class)}).NewProbeParser(scope)
	if err != nil {
		t.Fatal(err)
	}
	return parser, scope
}

func validProbeDocument(scope ProfileScope, observedAt time.Time) probeDocument {
	return probeDocument{
		SchemaVersion:   ProbeSchemaVersion,
		ProfileID:       scope.ProfileID(),
		PresetID:        "unit-test-only",
		Outcome:         ProbeOutcomeReady,
		CredentialState: CredentialStatusConfigured,
		Connected:       true,
		Negotiated:      true,
		CleanupComplete: true,
		ToolIDs:         []string{"read", "search"},
		ObservedAt:      observedAt,
	}
}

func mustProbeJSON(t *testing.T, document probeDocument) []byte {
	t.Helper()
	encoded, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

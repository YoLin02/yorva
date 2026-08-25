package mcpmanagement

import (
	"errors"
	"testing"
	"time"
)

func TestReadinessExpiresToConfigured(t *testing.T) {
	parser, scope := newTestProbeParser(t, CredentialClassStaticBearer)
	selection := Selection{descriptor: testDescriptor(CredentialClassStaticBearer)}
	configuredAt := time.Date(2026, 8, 25, 9, 59, 0, 0, time.UTC)
	machine, err := NewConfiguredReadiness(scope, selection, configuredAt)
	if err != nil {
		t.Fatal(err)
	}
	readyAt := configuredAt.Add(time.Minute)
	result, err := parser.Parse(mustProbeJSON(t, validProbeDocument(scope, readyAt)))
	if err != nil {
		t.Fatal(err)
	}
	machine, err = machine.MarkReady(result)
	if err != nil {
		t.Fatal(err)
	}

	fresh, err := machine.Snapshot(readyAt.Add(ReadyTTL - time.Nanosecond))
	if err != nil {
		t.Fatal(err)
	}
	if fresh.State() != ReadinessReady || !fresh.ObservedAt().Equal(readyAt) {
		t.Fatalf("fresh snapshot = %q at %v", fresh.State(), fresh.ObservedAt())
	}
	if fresh.ProfileID() != scope.ProfileID() || fresh.PresetID() != selection.PresetID() {
		t.Fatalf("fresh scope = %q/%q", fresh.ProfileID(), fresh.PresetID())
	}
	expiresAt, ok := fresh.ExpiresAt()
	if !ok || !expiresAt.Equal(readyAt.Add(ReadyTTL)) {
		t.Fatalf("fresh expiry = %v, %v", expiresAt, ok)
	}

	stale, err := machine.Snapshot(readyAt.Add(ReadyTTL))
	if err != nil {
		t.Fatal(err)
	}
	if stale.State() != ReadinessConfigured {
		t.Fatalf("expired state = %q, want CONFIGURED", stale.State())
	}
	if _, ok := stale.ExpiresAt(); ok {
		t.Fatal("expired CONFIGURED snapshot retained expiry")
	}
}

func TestReadinessInvalidatedByChangeRestartAndTimeout(t *testing.T) {
	for _, reason := range []InvalidationReason{
		InvalidatedConfigurationChanged,
		InvalidatedRuntimeRestarted,
		InvalidatedProbeTimedOut,
	} {
		t.Run(string(reason), func(t *testing.T) {
			parser, scope := newTestProbeParser(t, CredentialClassStaticBearer)
			selection := Selection{descriptor: testDescriptor(CredentialClassStaticBearer)}
			configuredAt := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)
			machine, err := NewConfiguredReadiness(scope, selection, configuredAt)
			if err != nil {
				t.Fatal(err)
			}
			readyAt := configuredAt.Add(time.Second)
			result, err := parser.Parse(mustProbeJSON(t, validProbeDocument(scope, readyAt)))
			if err != nil {
				t.Fatal(err)
			}
			machine, err = machine.MarkReady(result)
			if err != nil {
				t.Fatal(err)
			}
			invalidatedAt := readyAt.Add(time.Second)
			machine, err = machine.Invalidate(reason, invalidatedAt)
			if err != nil {
				t.Fatal(err)
			}
			snapshot, err := machine.Snapshot(invalidatedAt)
			if err != nil {
				t.Fatal(err)
			}
			if snapshot.State() != ReadinessConfigured || !snapshot.ObservedAt().Equal(invalidatedAt) {
				t.Fatalf("invalidated snapshot = %q at %v", snapshot.State(), snapshot.ObservedAt())
			}
		})
	}
}

func TestReadinessRejectsUnknownOrMismatchedTransitions(t *testing.T) {
	selection := Selection{descriptor: testDescriptor(CredentialClassStaticBearer)}
	scope, err := NewProfileScope("named")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)
	machine, err := NewConfiguredReadiness(scope, selection, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := machine.Invalidate(InvalidationReason("UNKNOWN"), now); !errors.Is(err, ErrReadinessTransitionInvalid) {
		t.Fatalf("unknown invalidation error = %v", err)
	}
	if _, err := machine.MarkReady(ProbeResult{profileID: "other", presetID: selection.PresetID(), outcome: ProbeOutcomeReady, observedAt: now}); !errors.Is(err, ErrReadinessTransitionInvalid) {
		t.Fatalf("mismatched readiness error = %v", err)
	}
}

func TestOAuthCapabilityRemainsFalseWithStableReason(t *testing.T) {
	capability := CurrentOAuthCapability()
	if capability.Supported() || OAuthCapabilitySupported {
		t.Fatal("OAuth capability unexpectedly enabled")
	}
	if capability.ErrorCode() != OAuthUnavailableCode || capability.Reason() != OAuthUnavailableReason {
		t.Fatalf("unstable OAuth capability: %q %q", capability.ErrorCode(), capability.Reason())
	}
	if capability.ErrorCode() == "" || capability.Reason() == "" {
		t.Fatal("OAuth unavailability must have a stable code and reason")
	}
}

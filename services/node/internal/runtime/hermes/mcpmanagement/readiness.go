package mcpmanagement

import (
	"errors"
	"time"
)

const ReadyTTL = 2 * time.Minute

var ErrReadinessTransitionInvalid = errors.New("Hermes MCP readiness transition is invalid")

type ReadinessState string

const (
	ReadinessConfigured ReadinessState = "CONFIGURED"
	ReadinessReady      ReadinessState = "READY"
)

type InvalidationReason string

const (
	InvalidatedConfigurationChanged InvalidationReason = "CONFIGURATION_CHANGED"
	InvalidatedRuntimeRestarted     InvalidationReason = "RUNTIME_RESTARTED"
	InvalidatedProbeTimedOut        InvalidationReason = "PROBE_TIMED_OUT"
)

func (r InvalidationReason) valid() bool {
	switch r {
	case InvalidatedConfigurationChanged, InvalidatedRuntimeRestarted, InvalidatedProbeTimedOut:
		return true
	default:
		return false
	}
}

// ReadinessMachine records one exact Profile/preset readiness observation.
// READY is never persisted as timeless configuration truth.
type ReadinessMachine struct {
	profileID       string
	presetID        string
	configuredAt    time.Time
	readyObservedAt time.Time
	readyExpiresAt  time.Time
}

func NewConfiguredReadiness(scope ProfileScope, selection Selection, observedAt time.Time) (ReadinessMachine, error) {
	if !scope.valid() {
		return ReadinessMachine{}, ErrProfileScopeInvalid
	}
	if !selection.valid() {
		return ReadinessMachine{}, ErrDescriptorInvalid
	}
	if observedAt.IsZero() {
		return ReadinessMachine{}, ErrReadinessTransitionInvalid
	}
	return ReadinessMachine{
		profileID:    scope.profileID,
		presetID:     selection.descriptor.presetID,
		configuredAt: observedAt.UTC(),
	}, nil
}

func (m ReadinessMachine) MarkReady(result ProbeResult) (ReadinessMachine, error) {
	if !m.valid() || result.outcome != ProbeOutcomeReady || result.profileID != m.profileID || result.presetID != m.presetID ||
		result.observedAt.Before(m.configuredAt) {
		return ReadinessMachine{}, ErrReadinessTransitionInvalid
	}
	m.readyObservedAt = result.observedAt.UTC()
	m.readyExpiresAt = m.readyObservedAt.Add(ReadyTTL)
	return m, nil
}

// Invalidate clears READY after configuration change, Runtime restart, or a
// probe timeout. The underlying configuration remains CONFIGURED pending its
// next authoritative read-back.
func (m ReadinessMachine) Invalidate(reason InvalidationReason, observedAt time.Time) (ReadinessMachine, error) {
	if !m.valid() || !reason.valid() || observedAt.IsZero() || observedAt.Before(m.latestObservation()) {
		return ReadinessMachine{}, ErrReadinessTransitionInvalid
	}
	m.configuredAt = observedAt.UTC()
	m.readyObservedAt = time.Time{}
	m.readyExpiresAt = time.Time{}
	return m, nil
}

type ReadinessSnapshot struct {
	profileID  string
	presetID   string
	state      ReadinessState
	observedAt time.Time
	expiresAt  time.Time
}

func (s ReadinessSnapshot) ProfileID() string { return s.profileID }

func (s ReadinessSnapshot) PresetID() string { return s.presetID }

func (s ReadinessSnapshot) State() ReadinessState { return s.state }

func (s ReadinessSnapshot) ObservedAt() time.Time { return s.observedAt }

func (s ReadinessSnapshot) ExpiresAt() (time.Time, bool) {
	return s.expiresAt, !s.expiresAt.IsZero()
}

func (m ReadinessMachine) Snapshot(now time.Time) (ReadinessSnapshot, error) {
	if !m.valid() || now.IsZero() || now.Before(m.configuredAt) ||
		(!m.readyObservedAt.IsZero() && now.Before(m.readyObservedAt)) {
		return ReadinessSnapshot{}, ErrReadinessTransitionInvalid
	}
	if !m.readyExpiresAt.IsZero() && now.Before(m.readyExpiresAt) {
		return ReadinessSnapshot{
			profileID:  m.profileID,
			presetID:   m.presetID,
			state:      ReadinessReady,
			observedAt: m.readyObservedAt,
			expiresAt:  m.readyExpiresAt,
		}, nil
	}
	observedAt := m.configuredAt
	if !m.readyExpiresAt.IsZero() && m.readyExpiresAt.After(observedAt) {
		observedAt = m.readyExpiresAt
	}
	return ReadinessSnapshot{
		profileID:  m.profileID,
		presetID:   m.presetID,
		state:      ReadinessConfigured,
		observedAt: observedAt,
	}, nil
}

func (m ReadinessMachine) valid() bool {
	return m.profileID != "" && m.presetID != "" && !m.configuredAt.IsZero()
}

func (m ReadinessMachine) latestObservation() time.Time {
	if m.readyObservedAt.After(m.configuredAt) {
		return m.readyObservedAt
	}
	return m.configuredAt
}

package managementhealth

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const (
	qualifiedHermesVersion   = "0.20.5"
	MaxLivenessResponseBytes = 2 * 1024
	MaxDetailedResponseBytes = 32 * 1024
	maxDetailedPlatforms     = 32
	maxPlatformNameBytes     = 64
	maxDiscardedDetailBytes  = 256
	maxDetailedCount         = 1_000_000
)

type HealthState string

const (
	HealthHealthy   HealthState = "HEALTHY"
	HealthDegraded  HealthState = "DEGRADED"
	HealthUnhealthy HealthState = "UNHEALTHY"
	HealthUnknown   HealthState = "UNKNOWN"
)

type HealthCheckState string

const (
	CheckOK          HealthCheckState = "OK"
	CheckDegraded    HealthCheckState = "DEGRADED"
	CheckUnavailable HealthCheckState = "UNAVAILABLE"
	CheckRetrying    HealthCheckState = "RETRYING"
)

type Liveness struct {
	Alive   bool
	Version string
}

type DetailedHealth struct {
	State   HealthState
	Version string
	Checks  []HealthCheck
}

type HealthCheck struct {
	Name  string
	State HealthCheckState
}

type livenessWire struct {
	Status   string `json:"status"`
	Platform string `json:"platform"`
	Version  string `json:"version"`
}

type detailedWire struct {
	Status           string          `json:"status"`
	Readiness        readinessWire   `json:"readiness"`
	Platform         string          `json:"platform"`
	Version          string          `json:"version"`
	GatewayState     json.RawMessage `json:"gateway_state"`
	Platforms        json.RawMessage `json:"platforms"`
	ActiveAgents     json.RawMessage `json:"active_agents"`
	GatewayBusy      json.RawMessage `json:"gateway_busy"`
	GatewayDrainable json.RawMessage `json:"gateway_drainable"`
	ExitReason       json.RawMessage `json:"exit_reason"`
	UpdatedAt        json.RawMessage `json:"updated_at"`
	PID              json.RawMessage `json:"pid"`
}

type readinessWire struct {
	Status string                     `json:"status"`
	Checks map[string]json.RawMessage `json:"checks"`
}

type basicCheckWire struct {
	Status string  `json:"status"`
	Detail *string `json:"detail,omitempty"`
}

type diskCheckWire struct {
	Status      string   `json:"status"`
	Detail      *string  `json:"detail,omitempty"`
	UsedPercent *float64 `json:"used_percent,omitempty"`
	FreeBytes   *int64   `json:"free_bytes,omitempty"`
}

type gatewayCheckWire struct {
	Status             string  `json:"status"`
	Detail             *string `json:"detail,omitempty"`
	State              *string `json:"state,omitempty"`
	ConnectedPlatforms *int    `json:"connected_platforms,omitempty"`
	Platforms          *int    `json:"platforms,omitempty"`
}

type queueCheckWire struct {
	Status             string  `json:"status"`
	Detail             *string `json:"detail,omitempty"`
	ActiveAPIRuns      *int    `json:"active_api_runs,omitempty"`
	ProcessCompletions *int    `json:"process_completions,omitempty"`
	ActiveDelegations  *int    `json:"active_delegations,omitempty"`
}

var detailedCheckOrder = []string{
	"state_db",
	"session_store",
	"config",
	"model",
	"disk",
	"gateway",
	"background_queues",
}

func ParseLiveness(data []byte) (Liveness, error) {
	var wire livenessWire
	if err := decodeStrict(data, MaxLivenessResponseBytes, &wire); err != nil {
		return Liveness{}, err
	}
	if wire.Status != "ok" || wire.Platform != "hermes-agent" || wire.Version != qualifiedHermesVersion {
		return Liveness{}, fmt.Errorf("%w: invalid liveness fields", ErrMalformedResponse)
	}
	return Liveness{Alive: true, Version: wire.Version}, nil
}

// ParseDetailedHealth validates the exact Hermes 0.20.5 readiness shape and
// returns only allowlisted state. PID, platform/account details, exit reason,
// timestamps, counters, paths and free-form diagnostic detail are discarded.
func ParseDetailedHealth(data []byte) (DetailedHealth, error) {
	var wire detailedWire
	if err := decodeStrict(data, MaxDetailedResponseBytes, &wire); err != nil {
		return DetailedHealth{}, err
	}
	if wire.Platform != "hermes-agent" || wire.Version != qualifiedHermesVersion {
		return DetailedHealth{}, fmt.Errorf("%w: invalid detailed health identity", ErrMalformedResponse)
	}
	if wire.Status != wire.Readiness.Status {
		return DetailedHealth{}, fmt.Errorf("%w: inconsistent readiness state", ErrMalformedResponse)
	}
	state, err := normalizeOverallHealth(wire.Status)
	if err != nil {
		return DetailedHealth{}, err
	}
	if err := validateDiscardedDetailedFields(wire); err != nil {
		return DetailedHealth{}, err
	}
	if len(wire.Readiness.Checks) != len(detailedCheckOrder) {
		return DetailedHealth{}, fmt.Errorf("%w: unexpected readiness check set", ErrMalformedResponse)
	}

	checks := make([]HealthCheck, 0, len(detailedCheckOrder))
	hasNonOKCheck := false
	for _, name := range detailedCheckOrder {
		raw, ok := wire.Readiness.Checks[name]
		if !ok {
			return DetailedHealth{}, fmt.Errorf("%w: missing readiness check", ErrMalformedResponse)
		}
		checkState, err := parseHealthCheck(name, raw)
		if err != nil {
			return DetailedHealth{}, err
		}
		hasNonOKCheck = hasNonOKCheck || checkState != CheckOK
		checks = append(checks, HealthCheck{Name: name, State: checkState})
	}
	if (state == HealthHealthy && hasNonOKCheck) || (state == HealthDegraded && !hasNonOKCheck) {
		return DetailedHealth{}, fmt.Errorf("%w: overall state disagrees with checks", ErrMalformedResponse)
	}

	return DetailedHealth{State: state, Version: wire.Version, Checks: checks}, nil
}

func normalizeOverallHealth(status string) (HealthState, error) {
	switch status {
	case "ok":
		return HealthHealthy, nil
	case "degraded":
		return HealthDegraded, nil
	default:
		return "", fmt.Errorf("%w: unknown readiness state", ErrMalformedResponse)
	}
}

func parseHealthCheck(name string, raw json.RawMessage) (HealthCheckState, error) {
	var status string
	switch name {
	case "state_db", "session_store", "config", "model":
		var check basicCheckWire
		if err := decodeStrict(raw, maxDiscardedDetailBytes+128, &check); err != nil {
			return "", err
		}
		if err := validateDetail(check.Detail); err != nil {
			return "", err
		}
		status = check.Status
	case "disk":
		var check diskCheckWire
		if err := decodeStrict(raw, maxDiscardedDetailBytes+192, &check); err != nil {
			return "", err
		}
		if err := validateDetail(check.Detail); err != nil {
			return "", err
		}
		if check.UsedPercent != nil && (*check.UsedPercent < 0 || *check.UsedPercent > 100) {
			return "", fmt.Errorf("%w: invalid disk percentage", ErrMalformedResponse)
		}
		if check.FreeBytes != nil && *check.FreeBytes < 0 {
			return "", fmt.Errorf("%w: invalid disk byte count", ErrMalformedResponse)
		}
		status = check.Status
	case "gateway":
		var check gatewayCheckWire
		if err := decodeStrict(raw, maxDiscardedDetailBytes+192, &check); err != nil {
			return "", err
		}
		if err := validateDetail(check.Detail); err != nil {
			return "", err
		}
		if check.State != nil && len(*check.State) > 32 {
			return "", fmt.Errorf("%w: gateway state too long", ErrMalformedResponse)
		}
		if negative(check.ConnectedPlatforms) || negative(check.Platforms) {
			return "", fmt.Errorf("%w: invalid platform count", ErrMalformedResponse)
		}
		status = check.Status
	case "background_queues":
		var check queueCheckWire
		if err := decodeStrict(raw, maxDiscardedDetailBytes+192, &check); err != nil {
			return "", err
		}
		if err := validateDetail(check.Detail); err != nil {
			return "", err
		}
		if negative(check.ActiveAPIRuns) || negative(check.ProcessCompletions) || negative(check.ActiveDelegations) {
			return "", fmt.Errorf("%w: invalid queue count", ErrMalformedResponse)
		}
		status = check.Status
	default:
		return "", fmt.Errorf("%w: unknown readiness check", ErrMalformedResponse)
	}

	return normalizeCheckState(name, status)
}

func normalizeCheckState(name, status string) (HealthCheckState, error) {
	switch status {
	case "ok":
		return CheckOK, nil
	case "degraded":
		return CheckDegraded, nil
	case "unavailable":
		if name == "session_store" {
			return CheckUnavailable, nil
		}
	case "retrying":
		if name == "session_store" {
			return CheckRetrying, nil
		}
	}
	return "", fmt.Errorf("%w: unknown check state", ErrMalformedResponse)
}

func validateDetail(detail *string) error {
	if detail != nil && len(*detail) > maxDiscardedDetailBytes {
		return fmt.Errorf("%w: diagnostic detail too long", ErrMalformedResponse)
	}
	return nil
}

func negative(value *int) bool {
	return value != nil && (*value < 0 || *value > maxDetailedCount)
}

func validateDiscardedDetailedFields(wire detailedWire) error {
	required := []json.RawMessage{
		wire.GatewayState,
		wire.Platforms,
		wire.ActiveAgents,
		wire.GatewayBusy,
		wire.GatewayDrainable,
		wire.ExitReason,
		wire.UpdatedAt,
		wire.PID,
	}
	for _, raw := range required {
		if len(raw) == 0 {
			return fmt.Errorf("%w: missing detailed health field", ErrMalformedResponse)
		}
	}
	if err := validateNullableString(wire.GatewayState, 32); err != nil {
		return err
	}
	if err := validateNonnegativeInt(wire.ActiveAgents, maxDetailedCount); err != nil {
		return err
	}
	if err := validateBool(wire.GatewayBusy); err != nil {
		return err
	}
	if err := validateBool(wire.GatewayDrainable); err != nil {
		return err
	}
	if err := validateNullableString(wire.ExitReason, maxDiscardedDetailBytes); err != nil {
		return err
	}
	if err := validateNullableString(wire.UpdatedAt, 64); err != nil {
		return err
	}
	if err := validatePositiveInt(wire.PID); err != nil {
		return err
	}

	var platforms map[string]json.RawMessage
	if !bytes.Equal(bytes.TrimSpace(wire.Platforms), []byte("null")) {
		if err := decodeStrict(wire.Platforms, MaxDetailedResponseBytes, &platforms); err != nil {
			return err
		}
		if len(platforms) > maxDetailedPlatforms {
			return fmt.Errorf("%w: too many platform entries", ErrMalformedResponse)
		}
		for name := range platforms {
			if len(name) == 0 || len(name) > maxPlatformNameBytes {
				return fmt.Errorf("%w: invalid platform name", ErrMalformedResponse)
			}
		}
	}

	return nil
}

func validateNullableString(raw json.RawMessage, maxBytes int) error {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil
	}
	var value string
	if err := decodeStrict(raw, maxBytes+2, &value); err != nil {
		return err
	}
	if len(value) > maxBytes {
		return fmt.Errorf("%w: discarded string too long", ErrMalformedResponse)
	}
	return nil
}

func validateNonnegativeInt(raw json.RawMessage, maximum int) error {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return fmt.Errorf("%w: null integer", ErrMalformedResponse)
	}
	var value int
	if err := decodeStrict(raw, 32, &value); err != nil {
		return err
	}
	if value < 0 || value > maximum {
		return fmt.Errorf("%w: integer outside bounds", ErrMalformedResponse)
	}
	return nil
}

func validatePositiveInt(raw json.RawMessage) error {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return fmt.Errorf("%w: null integer", ErrMalformedResponse)
	}
	var value int64
	if err := decodeStrict(raw, 32, &value); err != nil {
		return err
	}
	if value <= 0 || value > 1<<32-1 {
		return fmt.Errorf("%w: non-positive integer", ErrMalformedResponse)
	}
	return nil
}

func validateBool(raw json.RawMessage) error {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return fmt.Errorf("%w: null boolean", ErrMalformedResponse)
	}
	var value bool
	return decodeStrict(raw, 5, &value)
}

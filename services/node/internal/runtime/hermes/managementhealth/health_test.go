package managementhealth

import (
	"errors"
	"strings"
	"testing"
)

const detailedHealthFixture = `{
  "status":"degraded",
  "readiness":{"status":"degraded","checks":{
    "state_db":{"status":"ok"},
    "session_store":{"status":"retrying"},
    "config":{"status":"ok","detail":"using defaults"},
    "model":{"status":"ok"},
    "disk":{"status":"ok","used_percent":41.2,"free_bytes":123456},
    "gateway":{"status":"ok","state":"running","connected_platforms":1,"platforms":2},
    "background_queues":{"status":"ok","active_api_runs":2,"process_completions":0,"active_delegations":1}
  }},
  "platform":"hermes-agent",
  "version":"0.20.5",
  "gateway_state":"running",
  "platforms":{"telegram":{"account":"alice@example.com","path":"C:\\Users\\Alice\\secret"}},
  "active_agents":4,
  "gateway_busy":true,
  "gateway_drainable":false,
  "exit_reason":"contains account alice@example.com and C:\\Users\\Alice",
  "updated_at":"2026-08-25T10:00:00Z",
  "pid":4242
}`

func TestParseLiveness(t *testing.T) {
	got, err := ParseLiveness([]byte(`{"status":"ok","platform":"hermes-agent","version":"0.20.5"}`))
	if err != nil {
		t.Fatalf("ParseLiveness() error = %v", err)
	}
	if !got.Alive || got.Version != "0.20.5" {
		t.Fatalf("ParseLiveness() = %#v", got)
	}
}

func TestParseLivenessRejectsMalformedAndOversized(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want error
	}{
		{name: "unknown field", body: []byte(`{"status":"ok","platform":"hermes-agent","version":"0.20.5","pid":7}`), want: ErrMalformedResponse},
		{name: "wrong platform", body: []byte(`{"status":"ok","platform":"other","version":"0.20.5"}`), want: ErrMalformedResponse},
		{name: "trailing value", body: []byte(`{"status":"ok","platform":"hermes-agent","version":"0.20.5"} {}`), want: ErrMalformedResponse},
		{name: "duplicate field", body: []byte(`{"status":"ok","status":"ok","platform":"hermes-agent","version":"0.20.5"}`), want: ErrMalformedResponse},
		{name: "oversized", body: []byte(strings.Repeat("x", MaxLivenessResponseBytes+1)), want: ErrResponseTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseLiveness(tt.body)
			if !errors.Is(err, tt.want) {
				t.Fatalf("ParseLiveness() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestHealthParsersRequireExactQualifiedVersion(t *testing.T) {
	versions := []struct {
		name    string
		version string
		accept  bool
	}{
		{name: "qualified", version: "0.20.5", accept: true},
		{name: "different patch", version: "0.20.6"},
		{name: "development suffix", version: "0.20.5-dev"},
		{name: "empty", version: ""},
		{name: "overlong", version: strings.Repeat("1", 128)},
	}

	for _, tt := range versions {
		t.Run(tt.name, func(t *testing.T) {
			liveness := `{"status":"ok","platform":"hermes-agent","version":"` + tt.version + `"}`
			_, livenessErr := ParseLiveness([]byte(liveness))
			detailed := strings.Replace(detailedHealthFixture, `"version":"0.20.5"`, `"version":"`+tt.version+`"`, 1)
			_, detailedErr := ParseDetailedHealth([]byte(detailed))

			if tt.accept {
				if livenessErr != nil || detailedErr != nil {
					t.Fatalf("qualified version errors: liveness=%v detailed=%v", livenessErr, detailedErr)
				}
				return
			}
			if !errors.Is(livenessErr, ErrMalformedResponse) {
				t.Fatalf("ParseLiveness() error = %v, want %v", livenessErr, ErrMalformedResponse)
			}
			if !errors.Is(detailedErr, ErrMalformedResponse) {
				t.Fatalf("ParseDetailedHealth() error = %v, want %v", detailedErr, ErrMalformedResponse)
			}
		})
	}
}

func TestParseDetailedHealthProjectsOnlySafeFields(t *testing.T) {
	got, err := ParseDetailedHealth([]byte(detailedHealthFixture))
	if err != nil {
		t.Fatalf("ParseDetailedHealth() error = %v", err)
	}
	if got.State != HealthDegraded || got.Version != "0.20.5" || len(got.Checks) != 7 {
		t.Fatalf("ParseDetailedHealth() = %#v", got)
	}
	if got.Checks[1] != (HealthCheck{Name: "session_store", State: CheckRetrying}) {
		t.Fatalf("session store check = %#v", got.Checks[1])
	}
	projected := strings.Join([]string{string(got.State), got.Version, got.Checks[0].Name}, " ")
	for _, prohibited := range []string{"4242", "alice@example.com", `C:\Users\Alice`, "telegram", "exit_reason"} {
		if strings.Contains(projected, prohibited) {
			t.Fatalf("safe projection contains %q: %q", prohibited, projected)
		}
	}
}

func TestParseDetailedHealthRejectsMalformedAndOversized(t *testing.T) {
	tests := []struct {
		name string
		body string
		want error
	}{
		{name: "inconsistent overall", body: strings.Replace(detailedHealthFixture, `"readiness":{"status":"degraded"`, `"readiness":{"status":"ok"`, 1), want: ErrMalformedResponse},
		{name: "missing check", body: strings.Replace(detailedHealthFixture, `"state_db":{"status":"ok"},`, "", 1), want: ErrMalformedResponse},
		{name: "unknown check", body: strings.Replace(detailedHealthFixture, `"state_db"`, `"future_check"`, 1), want: ErrMalformedResponse},
		{name: "unknown check field", body: strings.Replace(detailedHealthFixture, `"state_db":{"status":"ok"}`, `"state_db":{"status":"ok","path":"C:\\secret"}`, 1), want: ErrMalformedResponse},
		{name: "invalid state", body: strings.Replace(detailedHealthFixture, `"session_store":{"status":"retrying"}`, `"session_store":{"status":"magic"}`, 1), want: ErrMalformedResponse},
		{name: "overall disagrees with checks", body: strings.Replace(detailedHealthFixture, `"session_store":{"status":"retrying"}`, `"session_store":{"status":"ok"}`, 1), want: ErrMalformedResponse},
		{name: "invalid discarded pid", body: strings.Replace(detailedHealthFixture, `"pid":4242`, `"pid":"C:\\secret"`, 1), want: ErrMalformedResponse},
		{name: "oversized", body: strings.Repeat("x", MaxDetailedResponseBytes+1), want: ErrResponseTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseDetailedHealth([]byte(tt.body))
			if !errors.Is(err, tt.want) {
				t.Fatalf("ParseDetailedHealth() error = %v, want %v", err, tt.want)
			}
		})
	}
}

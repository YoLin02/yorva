package managementhealth

import (
	"errors"
	"strings"
	"testing"
)

const auditFixture = `{
  "total_components_scanned":2,
  "finding_count":2,
  "findings":[
    {"package":"requests","version":"2.0.0","ecosystem":"PyPI","source":"venv","vuln_id":"GHSA-1234","severity":"HIGH","summary":"token=super-secret C:\\Users\\Alice\\config <script>alert(1)</script>","fixed_versions":["2.0.1"]},
    {"package":"@scope/server","version":"1.2.3","ecosystem":"npm","source":"mcp:alice-account","vuln_id":"CVE-2026-0001","severity":"MEDIUM","summary":"contact alice@example.com","fixed_versions":[]}
  ]
}`

func TestParseSecurityAuditProjectsAndRedacts(t *testing.T) {
	got, err := ParseSecurityAudit([]byte(auditFixture), 1)
	if err != nil {
		t.Fatalf("ParseSecurityAudit() error = %v", err)
	}
	if got.Outcome != AuditThresholdReached || got.Components != 2 || got.FindingCount != 2 {
		t.Fatalf("ParseSecurityAudit() = %#v", got)
	}
	if got.Findings[0].Source != AuditSourceRuntime || got.Findings[1].Source != AuditSourceMCP {
		t.Fatalf("normalized sources = %q, %q", got.Findings[0].Source, got.Findings[1].Source)
	}
	if got.Findings[1].Severity != SeverityModerate {
		t.Fatalf("normalized severity = %q", got.Findings[1].Severity)
	}
	projected := got.Findings[0].Summary + got.Findings[1].Summary + string(got.Findings[1].Source)
	for _, prohibited := range []string{"super-secret", `C:\Users\Alice`, "<script>", "alice@example.com", "alice-account"} {
		if strings.Contains(projected, prohibited) {
			t.Fatalf("audit projection contains %q: %q", prohibited, projected)
		}
	}
}

func TestParseSecurityAuditExitSemantics(t *testing.T) {
	clean := []byte(`{"total_components_scanned":0,"finding_count":0,"findings":[]}`)
	got, err := ParseSecurityAudit(clean, 0)
	if err != nil || got.Outcome != AuditCompleted {
		t.Fatalf("exit 0 = %#v, %v", got, err)
	}

	got, err = ParseSecurityAudit(nil, 2)
	if err != nil || got.Outcome != AuditFailed {
		t.Fatalf("exit 2 = %#v, %v", got, err)
	}

	if _, err = ParseSecurityAudit(clean, 1); !errors.Is(err, ErrMalformedResponse) {
		t.Fatalf("exit 1 without findings error = %v", err)
	}
	if _, err = ParseSecurityAudit([]byte("raw error with secret"), 2); !errors.Is(err, ErrMalformedResponse) {
		t.Fatalf("exit 2 with stdout error = %v", err)
	}
	if _, err = ParseSecurityAudit(clean, 3); err == nil {
		t.Fatal("unsupported exit code unexpectedly succeeded")
	}
}

func TestParseSecurityAuditRejectsMalformedAndOversized(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want error
	}{
		{name: "unknown field", body: []byte(`{"total_components_scanned":0,"finding_count":0,"findings":[],"raw":"secret"}`), want: ErrMalformedResponse},
		{name: "count mismatch", body: []byte(`{"total_components_scanned":2,"finding_count":2,"findings":[]}`), want: ErrMalformedResponse},
		{name: "unknown source", body: []byte(strings.Replace(auditFixture, `"source":"venv"`, `"source":"profile:alice"`, 1)), want: ErrMalformedResponse},
		{name: "unknown severity", body: []byte(strings.Replace(auditFixture, `"severity":"HIGH"`, `"severity":"EXTREME"`, 1)), want: ErrMalformedResponse},
		{name: "oversized", body: []byte(strings.Repeat("x", MaxSecurityAuditResponseBytes+1)), want: ErrResponseTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSecurityAudit(tt.body, 0)
			if !errors.Is(err, tt.want) {
				t.Fatalf("ParseSecurityAudit() error = %v, want %v", err, tt.want)
			}
		})
	}
}

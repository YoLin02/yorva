package managementhealth

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

const (
	MaxSecurityAuditResponseBytes = 256 * 1024
	MaxAuditComponents            = 10_000
	MaxAuditFindings              = 512
	MaxAuditFixedVersions         = 16
	maxAuditPackageBytes          = 128
	maxAuditVersionBytes          = 64
	maxAuditIDBytes               = 64
	maxAuditSummaryBytes          = 256
	maxAuditRawSummaryBytes       = 4 * 1024
	maxAuditSourceBytes           = 128
)

type AuditOutcome string

const (
	AuditCompleted        AuditOutcome = "COMPLETED"
	AuditThresholdReached AuditOutcome = "THRESHOLD_REACHED"
	AuditFailed           AuditOutcome = "FAILED"
)

type AuditSource string

const (
	AuditSourceRuntime AuditSource = "RUNTIME"
	AuditSourcePlugin  AuditSource = "PLUGIN"
	AuditSourceMCP     AuditSource = "MCP"
)

type AuditSeverity string

const (
	SeverityUnknown  AuditSeverity = "UNKNOWN"
	SeverityLow      AuditSeverity = "LOW"
	SeverityModerate AuditSeverity = "MODERATE"
	SeverityHigh     AuditSeverity = "HIGH"
	SeverityCritical AuditSeverity = "CRITICAL"
)

type SecurityAudit struct {
	Outcome      AuditOutcome
	Components   int
	FindingCount int
	Findings     []AuditFinding
}

type AuditFinding struct {
	Package       string
	Version       string
	Ecosystem     string
	Source        AuditSource
	Vulnerability string
	Severity      AuditSeverity
	Summary       string
	FixedVersions []string
}

type auditWire struct {
	Components   *int                `json:"total_components_scanned"`
	FindingCount *int                `json:"finding_count"`
	Findings     *[]auditFindingWire `json:"findings"`
}

type auditFindingWire struct {
	Package       string    `json:"package"`
	Version       string    `json:"version"`
	Ecosystem     string    `json:"ecosystem"`
	Source        string    `json:"source"`
	Vulnerability string    `json:"vuln_id"`
	Severity      string    `json:"severity"`
	Summary       string    `json:"summary"`
	FixedVersions *[]string `json:"fixed_versions"`
}

var (
	safePackage  = regexp.MustCompile(`^[A-Za-z0-9@][A-Za-z0-9@/._+-]{0,127}$`)
	safeArtifact = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+!:-]{0,63}$`)
	safeSourceID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,95}$`)
)

// ParseSecurityAudit normalizes the documented Hermes exit contract. Exit 0
// and 1 require a valid bounded JSON result. Exit 2 is a safe failure without
// raw error detail. Other exits are unsupported and never inferred as success.
func ParseSecurityAudit(stdout []byte, exitCode int) (SecurityAudit, error) {
	switch exitCode {
	case 2:
		if len(bytes.TrimSpace(stdout)) != 0 {
			return SecurityAudit{}, fmt.Errorf("%w: unexpected output for audit failure", ErrMalformedResponse)
		}
		return SecurityAudit{Outcome: AuditFailed}, nil
	case 0, 1:
		// Continue with the structured result below.
	default:
		return SecurityAudit{}, fmt.Errorf("unsupported Hermes security audit exit code: %d", exitCode)
	}

	var wire auditWire
	if err := decodeStrict(stdout, MaxSecurityAuditResponseBytes, &wire); err != nil {
		return SecurityAudit{}, err
	}
	if wire.Components == nil || wire.FindingCount == nil || wire.Findings == nil {
		return SecurityAudit{}, fmt.Errorf("%w: missing audit fields", ErrMalformedResponse)
	}
	if *wire.Components < 0 || *wire.Components > MaxAuditComponents {
		return SecurityAudit{}, fmt.Errorf("%w: invalid component count", ErrMalformedResponse)
	}
	if *wire.FindingCount < 0 || *wire.FindingCount > MaxAuditFindings || len(*wire.Findings) != *wire.FindingCount {
		return SecurityAudit{}, fmt.Errorf("%w: inconsistent finding count", ErrMalformedResponse)
	}
	if *wire.FindingCount > 0 && *wire.Components == 0 {
		return SecurityAudit{}, fmt.Errorf("%w: findings without scanned components", ErrMalformedResponse)
	}
	if exitCode == 1 && *wire.FindingCount == 0 {
		return SecurityAudit{}, fmt.Errorf("%w: threshold exit without findings", ErrMalformedResponse)
	}

	findings := make([]AuditFinding, 0, len(*wire.Findings))
	for _, finding := range *wire.Findings {
		projected, err := projectAuditFinding(finding)
		if err != nil {
			return SecurityAudit{}, err
		}
		findings = append(findings, projected)
	}

	outcome := AuditCompleted
	if exitCode == 1 {
		outcome = AuditThresholdReached
	}
	return SecurityAudit{
		Outcome:      outcome,
		Components:   *wire.Components,
		FindingCount: *wire.FindingCount,
		Findings:     findings,
	}, nil
}

func projectAuditFinding(wire auditFindingWire) (AuditFinding, error) {
	if len(wire.Package) > maxAuditPackageBytes || !safePackage.MatchString(wire.Package) {
		return AuditFinding{}, fmt.Errorf("%w: invalid package identity", ErrMalformedResponse)
	}
	if len(wire.Version) > maxAuditVersionBytes || !safeArtifact.MatchString(wire.Version) {
		return AuditFinding{}, fmt.Errorf("%w: invalid package version", ErrMalformedResponse)
	}
	if len(wire.Vulnerability) > maxAuditIDBytes || !safeArtifact.MatchString(wire.Vulnerability) {
		return AuditFinding{}, fmt.Errorf("%w: invalid vulnerability identity", ErrMalformedResponse)
	}
	if wire.FixedVersions == nil || len(*wire.FixedVersions) > MaxAuditFixedVersions {
		return AuditFinding{}, fmt.Errorf("%w: too many fixed versions", ErrMalformedResponse)
	}
	if len(wire.Summary) > maxAuditRawSummaryBytes {
		return AuditFinding{}, fmt.Errorf("%w: audit summary too long", ErrMalformedResponse)
	}

	source, err := normalizeAuditSource(wire.Source)
	if err != nil {
		return AuditFinding{}, err
	}
	severity, err := normalizeAuditSeverity(wire.Severity)
	if err != nil {
		return AuditFinding{}, err
	}
	ecosystem := wire.Ecosystem
	if ecosystem != "PyPI" && ecosystem != "npm" {
		return AuditFinding{}, fmt.Errorf("%w: unknown audit ecosystem", ErrMalformedResponse)
	}

	fixed := make([]string, 0, len(*wire.FixedVersions))
	for _, version := range *wire.FixedVersions {
		if len(version) > maxAuditVersionBytes || !safeArtifact.MatchString(version) {
			return AuditFinding{}, fmt.Errorf("%w: invalid fixed version", ErrMalformedResponse)
		}
		fixed = append(fixed, version)
	}

	summary := redactAndSanitize(wire.Summary)
	summary, _ = truncateUTF8(summary, maxAuditSummaryBytes)
	return AuditFinding{
		Package:       wire.Package,
		Version:       wire.Version,
		Ecosystem:     ecosystem,
		Source:        source,
		Vulnerability: wire.Vulnerability,
		Severity:      severity,
		Summary:       summary,
		FixedVersions: fixed,
	}, nil
}

func normalizeAuditSource(source string) (AuditSource, error) {
	if len(source) == 0 || len(source) > maxAuditSourceBytes {
		return "", fmt.Errorf("%w: invalid audit source", ErrMalformedResponse)
	}
	if source == "venv" {
		return AuditSourceRuntime, nil
	}
	for prefix, normalized := range map[string]AuditSource{
		"plugin:": AuditSourcePlugin,
		"mcp:":    AuditSourceMCP,
	} {
		if strings.HasPrefix(source, prefix) && safeSourceID.MatchString(strings.TrimPrefix(source, prefix)) {
			return normalized, nil
		}
	}
	return "", fmt.Errorf("%w: unknown audit source", ErrMalformedResponse)
}

func normalizeAuditSeverity(severity string) (AuditSeverity, error) {
	switch strings.ToUpper(severity) {
	case "UNKNOWN":
		return SeverityUnknown, nil
	case "LOW":
		return SeverityLow, nil
	case "MODERATE", "MEDIUM":
		return SeverityModerate, nil
	case "HIGH":
		return SeverityHigh, nil
	case "CRITICAL":
		return SeverityCritical, nil
	default:
		return "", fmt.Errorf("%w: unknown audit severity", ErrMalformedResponse)
	}
}

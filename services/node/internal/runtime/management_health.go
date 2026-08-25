package runtime

import (
	"context"
	"fmt"
	"time"
)

type HealthState string

const (
	HealthHealthy   HealthState = "HEALTHY"
	HealthDegraded  HealthState = "DEGRADED"
	HealthUnhealthy HealthState = "UNHEALTHY"
	HealthUnknown   HealthState = "UNKNOWN"
)

func (s HealthState) Valid() bool {
	return s == HealthHealthy || s == HealthDegraded || s == HealthUnhealthy || s == HealthUnknown
}

type HealthFinding struct {
	Code  string
	State HealthState
}

type HealthObservation struct {
	State      HealthState
	Findings   []HealthFinding
	Partial    bool
	ObservedAt time.Time
}

func (o HealthObservation) Validate() error {
	if !o.State.Valid() || o.ObservedAt.IsZero() || len(o.Findings) > managementCollectionLimit {
		return ErrInvalidManagementContract
	}
	for _, finding := range o.Findings {
		if err := validateManagementID("health finding code", finding.Code); err != nil {
			return err
		}
		if !finding.State.Valid() {
			return fmt.Errorf("%w: invalid health finding state", ErrInvalidManagementContract)
		}
	}
	return nil
}

type HealthInspector interface {
	InspectRuntimeHealth(context.Context, Installation) (HealthObservation, error)
	InspectInstanceHealth(context.Context, Installation, string) (HealthObservation, error)
}

type LogCategory string

const (
	LogCategoryRuntime LogCategory = "RUNTIME"
	LogCategoryErrors  LogCategory = "ERRORS"
	LogCategoryGateway LogCategory = "GATEWAY"
	LogCategoryMCP     LogCategory = "MCP"
)

func (c LogCategory) Valid() bool {
	return c == LogCategoryRuntime || c == LogCategoryErrors || c == LogCategoryGateway || c == LogCategoryMCP
}

type LogEntry struct {
	Timestamp time.Time
	Message   string
}

type LogSnapshot struct {
	Category   LogCategory
	Entries    []LogEntry
	Truncated  bool
	ObservedAt time.Time
}

func (s LogSnapshot) Validate() error {
	if !s.Category.Valid() || s.ObservedAt.IsZero() || len(s.Entries) > managementCollectionLimit {
		return ErrInvalidManagementContract
	}
	totalBytes := 0
	for _, entry := range s.Entries {
		if entry.Timestamp.IsZero() {
			return fmt.Errorf("%w: missing log timestamp", ErrInvalidManagementContract)
		}
		if err := validateBoundedText("log message", entry.Message); err != nil {
			return err
		}
		totalBytes += len(entry.Message)
		if totalBytes > 64*1024 {
			return fmt.Errorf("%w: log snapshot too large", ErrInvalidManagementContract)
		}
	}
	return nil
}

type LogReader interface {
	ReadLogSnapshot(context.Context, Installation, string, LogCategory) (LogSnapshot, error)
}

type SecurityAuditState string

const (
	SecurityAuditClean   SecurityAuditState = "CLEAN"
	SecurityAuditWarning SecurityAuditState = "WARNING"
	SecurityAuditBlocked SecurityAuditState = "BLOCKED"
	SecurityAuditUnknown SecurityAuditState = "UNKNOWN"
)

func (s SecurityAuditState) Valid() bool {
	return s == SecurityAuditClean || s == SecurityAuditWarning || s == SecurityAuditBlocked || s == SecurityAuditUnknown
}

type SecuritySeverity string

const (
	SecuritySeverityLow      SecuritySeverity = "LOW"
	SecuritySeverityMedium   SecuritySeverity = "MEDIUM"
	SecuritySeverityHigh     SecuritySeverity = "HIGH"
	SecuritySeverityCritical SecuritySeverity = "CRITICAL"
	SecuritySeverityUnknown  SecuritySeverity = "UNKNOWN"
)

func (s SecuritySeverity) Valid() bool {
	return s == SecuritySeverityLow || s == SecuritySeverityMedium || s == SecuritySeverityHigh || s == SecuritySeverityCritical || s == SecuritySeverityUnknown
}

type SecurityFinding struct {
	ID        string
	Component string
	Severity  SecuritySeverity
}

type SecurityAuditResult struct {
	State      SecurityAuditState
	Findings   []SecurityFinding
	Partial    bool
	ObservedAt time.Time
}

func (r SecurityAuditResult) Validate() error {
	if !r.State.Valid() || r.ObservedAt.IsZero() || len(r.Findings) > managementCollectionLimit {
		return ErrInvalidManagementContract
	}
	for _, finding := range r.Findings {
		if err := validateManagementID("security finding id", finding.ID); err != nil {
			return err
		}
		if err := validateBoundedText("security component", finding.Component); err != nil {
			return err
		}
		if !finding.Severity.Valid() {
			return fmt.Errorf("%w: invalid security finding severity", ErrInvalidManagementContract)
		}
	}
	return nil
}

type SecurityAuditor interface {
	AuditRuntimeSecurity(context.Context, Installation) (SecurityAuditResult, error)
	AuditInstanceSecurity(context.Context, Installation, string) (SecurityAuditResult, error)
}

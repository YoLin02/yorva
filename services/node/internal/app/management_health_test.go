package app

import (
	"context"
	"errors"
	"testing"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type b3FakeTargetResolver struct {
	target ManagementTarget
	err    error
	gotID  string
}

func (f *b3FakeTargetResolver) ResolveManagementTarget(_ context.Context, instanceID string) (ManagementTarget, error) {
	f.gotID = instanceID
	return f.target, f.err
}

type b3FakeManagementAdapter struct {
	health      yorvaruntime.HealthObservation
	logs        yorvaruntime.LogSnapshot
	security    yorvaruntime.SecurityAuditResult
	healthErr   error
	logsErr     error
	securityErr error
	wantContext bool
	gotInstall  yorvaruntime.Installation
	gotNativeID string
	gotCategory yorvaruntime.LogCategory
}

func (f *b3FakeManagementAdapter) InspectRuntimeHealth(context.Context, yorvaruntime.Installation) (yorvaruntime.HealthObservation, error) {
	return yorvaruntime.HealthObservation{}, errors.New("unexpected Runtime health call")
}

func (f *b3FakeManagementAdapter) InspectInstanceHealth(ctx context.Context, installation yorvaruntime.Installation, nativeID string) (yorvaruntime.HealthObservation, error) {
	f.capture(installation, nativeID)
	if f.wantContext {
		<-ctx.Done()
		return yorvaruntime.HealthObservation{}, ctx.Err()
	}
	return f.health, f.healthErr
}

func (f *b3FakeManagementAdapter) ReadLogSnapshot(ctx context.Context, installation yorvaruntime.Installation, nativeID string, category yorvaruntime.LogCategory) (yorvaruntime.LogSnapshot, error) {
	f.capture(installation, nativeID)
	f.gotCategory = category
	if f.wantContext {
		<-ctx.Done()
		return yorvaruntime.LogSnapshot{}, ctx.Err()
	}
	return f.logs, f.logsErr
}

func (f *b3FakeManagementAdapter) AuditRuntimeSecurity(context.Context, yorvaruntime.Installation) (yorvaruntime.SecurityAuditResult, error) {
	return yorvaruntime.SecurityAuditResult{}, errors.New("unexpected Runtime security call")
}

func (f *b3FakeManagementAdapter) AuditInstanceSecurity(ctx context.Context, installation yorvaruntime.Installation, nativeID string) (yorvaruntime.SecurityAuditResult, error) {
	f.capture(installation, nativeID)
	if f.wantContext {
		<-ctx.Done()
		return yorvaruntime.SecurityAuditResult{}, ctx.Err()
	}
	return f.security, f.securityErr
}

func (f *b3FakeManagementAdapter) capture(installation yorvaruntime.Installation, nativeID string) {
	f.gotInstall = installation
	f.gotNativeID = nativeID
}

func TestManagementHealthQueriesResolveExactTargetAndValidateResults(t *testing.T) {
	now := time.Date(2026, 8, 25, 11, 0, 0, 0, time.UTC)
	adapter := &b3FakeManagementAdapter{
		health: yorvaruntime.HealthObservation{
			State: yorvaruntime.HealthHealthy, Findings: []yorvaruntime.HealthFinding{{Code: "gateway", State: yorvaruntime.HealthHealthy}}, ObservedAt: now,
		},
		logs: yorvaruntime.LogSnapshot{
			Category: yorvaruntime.LogCategoryErrors, Entries: []yorvaruntime.LogEntry{{Timestamp: now, Message: "redacted"}}, ObservedAt: now,
		},
		security: yorvaruntime.SecurityAuditResult{
			State:      yorvaruntime.SecurityAuditWarning,
			Findings:   []yorvaruntime.SecurityFinding{{ID: "dependency-1", Component: "dependency", Severity: yorvaruntime.SecuritySeverityMedium}},
			ObservedAt: now,
		},
	}
	installation := yorvaruntime.Installation{RuntimeKind: "hermes", Path: `C:\hermes\hermes.exe`, Version: "0.20.5", SupportState: yorvaruntime.DiscoverySupported}
	resolver := &b3FakeTargetResolver{target: ManagementTarget{
		Installation: installation,
		NativeID:     "coder",
		Bundle: yorvaruntime.Bundle{
			Health: adapter, Logs: adapter, Security: adapter,
		},
	}}
	queries := NewManagementHealth(resolver)

	health, err := queries.GetInstanceHealth(context.Background(), "inst_1")
	if err != nil || health.State != yorvaruntime.HealthHealthy {
		t.Fatalf("GetInstanceHealth() = %#v, %v", health, err)
	}
	logs, err := queries.GetInstanceLogSnapshot(context.Background(), "inst_1", yorvaruntime.LogCategoryErrors)
	if err != nil || len(logs.Entries) != 1 {
		t.Fatalf("GetInstanceLogSnapshot() = %#v, %v", logs, err)
	}
	security, err := queries.GetInstanceSecurityAudit(context.Background(), "inst_1")
	if err != nil || security.State != yorvaruntime.SecurityAuditWarning {
		t.Fatalf("GetInstanceSecurityAudit() = %#v, %v", security, err)
	}

	if resolver.gotID != "inst_1" || adapter.gotInstall != installation || adapter.gotNativeID != "coder" || adapter.gotCategory != yorvaruntime.LogCategoryErrors {
		t.Fatalf("target forwarding = id %q, installation %#v, native %q, category %q", resolver.gotID, adapter.gotInstall, adapter.gotNativeID, adapter.gotCategory)
	}
}

func TestManagementHealthQueriesFailClosedForMissingCapabilities(t *testing.T) {
	queries := NewManagementHealth(&b3FakeTargetResolver{target: ManagementTarget{Bundle: yorvaruntime.Bundle{}}})

	if _, err := queries.GetInstanceHealth(context.Background(), "inst_1"); !errors.Is(err, ErrManagementCapabilityUnsupported) {
		t.Fatalf("health error = %v", err)
	}
	if _, err := queries.GetInstanceLogSnapshot(context.Background(), "inst_1", yorvaruntime.LogCategoryRuntime); !errors.Is(err, ErrManagementCapabilityUnsupported) {
		t.Fatalf("logs error = %v", err)
	}
	if _, err := queries.GetInstanceSecurityAudit(context.Background(), "inst_1"); !errors.Is(err, ErrManagementCapabilityUnsupported) {
		t.Fatalf("security error = %v", err)
	}
}

func TestManagementHealthQueriesRejectMalformedAdapterResults(t *testing.T) {
	adapter := &b3FakeManagementAdapter{}
	queries := NewManagementHealth(&b3FakeTargetResolver{target: ManagementTarget{Bundle: yorvaruntime.Bundle{
		Health: adapter, Logs: adapter, Security: adapter,
	}}})

	if _, err := queries.GetInstanceHealth(context.Background(), "inst_1"); !errors.Is(err, ErrManagementQueryFailed) {
		t.Fatalf("malformed health error = %v", err)
	}
	if _, err := queries.GetInstanceLogSnapshot(context.Background(), "inst_1", yorvaruntime.LogCategoryRuntime); !errors.Is(err, ErrManagementQueryFailed) {
		t.Fatalf("malformed logs error = %v", err)
	}
	if _, err := queries.GetInstanceSecurityAudit(context.Background(), "inst_1"); !errors.Is(err, ErrManagementQueryFailed) {
		t.Fatalf("malformed security error = %v", err)
	}
	if _, err := queries.GetInstanceLogSnapshot(context.Background(), "inst_1", "arbitrary/path"); !errors.Is(err, ErrManagementQueryFailed) {
		t.Fatalf("invalid category error = %v", err)
	}
}

func TestManagementHealthQueriesNormalizeResolverAndAdapterErrors(t *testing.T) {
	queries := NewManagementHealth(&b3FakeTargetResolver{err: errors.New("database details")})
	if _, err := queries.GetInstanceHealth(context.Background(), "inst_1"); !errors.Is(err, ErrManagementQueryFailed) || err.Error() != ErrManagementQueryFailed.Error() {
		t.Fatalf("resolver error = %v", err)
	}

	adapter := &b3FakeManagementAdapter{
		healthErr: errors.New("health command details"), logsErr: errors.New("log path details"), securityErr: errors.New("audit output details"),
	}
	queries = NewManagementHealth(&b3FakeTargetResolver{target: ManagementTarget{Bundle: yorvaruntime.Bundle{
		Health: adapter, Logs: adapter, Security: adapter,
	}}})
	for name, call := range map[string]func() error{
		"health": func() error { _, err := queries.GetInstanceHealth(context.Background(), "inst_1"); return err },
		"logs": func() error {
			_, err := queries.GetInstanceLogSnapshot(context.Background(), "inst_1", yorvaruntime.LogCategoryRuntime)
			return err
		},
		"security": func() error { _, err := queries.GetInstanceSecurityAudit(context.Background(), "inst_1"); return err },
	} {
		t.Run(name, func(t *testing.T) {
			err := call()
			if !errors.Is(err, ErrManagementQueryFailed) || err.Error() != ErrManagementQueryFailed.Error() {
				t.Fatalf("error = %v, want stable ErrManagementQueryFailed", err)
			}
		})
	}
}

func TestManagementHealthQueriesHonorContextCancellation(t *testing.T) {
	adapter := &b3FakeManagementAdapter{wantContext: true}
	queries := NewManagementHealth(&b3FakeTargetResolver{target: ManagementTarget{Bundle: yorvaruntime.Bundle{
		Health: adapter, Logs: adapter, Security: adapter,
	}}})

	for name, call := range map[string]func(context.Context) error{
		"health": func(ctx context.Context) error { _, err := queries.GetInstanceHealth(ctx, "inst_1"); return err },
		"logs": func(ctx context.Context) error {
			_, err := queries.GetInstanceLogSnapshot(ctx, "inst_1", yorvaruntime.LogCategoryRuntime)
			return err
		},
		"security": func(ctx context.Context) error { _, err := queries.GetInstanceSecurityAudit(ctx, "inst_1"); return err },
	} {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if err := call(ctx); !errors.Is(err, context.Canceled) {
				t.Fatalf("error = %v, want context.Canceled", err)
			}
		})
	}
}

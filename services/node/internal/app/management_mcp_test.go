package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/node"
	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	"github.com/YoLin02/yorva/services/node/internal/persistence/sqlite"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type mcpTargetResolverFake struct {
	target ManagementTarget
	err    error
	calls  int
}

type mcpRuntimeTargetResolverFake struct {
	mcpTargetResolverFake
	runtimeTarget RuntimeManagementTarget
}

func (f *mcpRuntimeTargetResolverFake) ResolveRuntimeManagementTarget(context.Context, string) (RuntimeManagementTarget, error) {
	return f.runtimeTarget, nil
}

func (f *mcpTargetResolverFake) ResolveManagementTarget(context.Context, string) (ManagementTarget, error) {
	f.calls++
	return f.target, f.err
}

type mcpReaderFake struct {
	servers []yorvaruntime.MCPServer
	presets []yorvaruntime.MCPPreset
	err     error
}

func (f *mcpReaderFake) ListMCPServers(context.Context, yorvaruntime.Installation, string) ([]yorvaruntime.MCPServer, error) {
	return f.servers, f.err
}

func (f *mcpReaderFake) ListMCPPresets(context.Context, yorvaruntime.Installation, string) ([]yorvaruntime.MCPPreset, error) {
	return f.presets, f.err
}

type mcpManagerFake struct {
	result  yorvaruntime.MCPTestResult
	err     error
	calls   int
	started chan struct{}
}

type mcpLifecycleFake struct {
	sequence []string
	readyAt  time.Time
}

func (f *mcpLifecycleFake) ListMCPServers(context.Context, yorvaruntime.Installation, string) ([]yorvaruntime.MCPServer, error) {
	f.sequence = append(f.sequence, "readback")
	return []yorvaruntime.MCPServer{{ID: "preset-a", PresetID: "preset-a", Managed: true, State: yorvaruntime.MCPReady, ReadyAt: &f.readyAt, ObservedAt: f.readyAt}}, nil
}

func (*mcpLifecycleFake) ListMCPPresets(context.Context, yorvaruntime.Installation, string) ([]yorvaruntime.MCPPreset, error) {
	return []yorvaruntime.MCPPreset{{ID: "preset-a", DisplayName: "Preset A", AllowedToolIDs: []string{"tool-a"}, CredentialRequired: true}}, nil
}

func (f *mcpLifecycleFake) InstallMCPPreset(context.Context, yorvaruntime.Installation, string, yorvaruntime.MCPInstallRequest, yorvaruntime.ProgressSink) (yorvaruntime.MCPServer, error) {
	f.sequence = append(f.sequence, "install")
	return yorvaruntime.MCPServer{ID: "preset-a", PresetID: "preset-a", State: yorvaruntime.MCPAuthRequired, ObservedAt: time.Now().UTC()}, nil
}

func (f *mcpLifecycleFake) AuthenticateMCP(context.Context, yorvaruntime.Installation, string, yorvaruntime.MCPAuthenticateRequest, yorvaruntime.ProgressSink) (yorvaruntime.MCPServer, error) {
	f.sequence = append(f.sequence, "credential")
	return yorvaruntime.MCPServer{ID: "preset-a", PresetID: "preset-a", State: yorvaruntime.MCPConfigured, ObservedAt: time.Now().UTC()}, nil
}

func (f *mcpLifecycleFake) ConfigureMCP(context.Context, yorvaruntime.Installation, string, yorvaruntime.MCPConfigureRequest, yorvaruntime.ProgressSink) (yorvaruntime.MCPServer, error) {
	f.sequence = append(f.sequence, "tools")
	return yorvaruntime.MCPServer{ID: "preset-a", PresetID: "preset-a", State: yorvaruntime.MCPConfigured, ObservedAt: time.Now().UTC()}, nil
}

func (f *mcpLifecycleFake) TestMCP(context.Context, yorvaruntime.Installation, string, string, yorvaruntime.ProgressSink) (yorvaruntime.MCPTestResult, error) {
	f.sequence = append(f.sequence, "test")
	f.readyAt = time.Now().UTC()
	return yorvaruntime.MCPTestResult{ServerID: "preset-a", State: yorvaruntime.MCPReady, ToolIDs: []string{"tool-a"}, ReadyAt: &f.readyAt, TestedAt: f.readyAt}, nil
}

func (f *mcpLifecycleFake) RemoveMCP(context.Context, yorvaruntime.Installation, string, string, yorvaruntime.ProgressSink) (yorvaruntime.MCPServer, error) {
	f.sequence = append(f.sequence, "remove")
	return yorvaruntime.MCPServer{ID: "preset-a", PresetID: "preset-a", State: yorvaruntime.MCPNotConfigured, ObservedAt: time.Now().UTC()}, nil
}

func (*mcpManagerFake) InstallMCPPreset(context.Context, yorvaruntime.Installation, string, yorvaruntime.MCPInstallRequest, yorvaruntime.ProgressSink) (yorvaruntime.MCPServer, error) {
	return yorvaruntime.MCPServer{}, ErrManagementCapabilityUnsupported
}

func (*mcpManagerFake) AuthenticateMCP(context.Context, yorvaruntime.Installation, string, yorvaruntime.MCPAuthenticateRequest, yorvaruntime.ProgressSink) (yorvaruntime.MCPServer, error) {
	return yorvaruntime.MCPServer{}, ErrManagementCapabilityUnsupported
}

func (f *mcpManagerFake) TestMCP(ctx context.Context, _ yorvaruntime.Installation, _ string, _ string, _ yorvaruntime.ProgressSink) (yorvaruntime.MCPTestResult, error) {
	f.calls++
	if f.started != nil {
		close(f.started)
		<-ctx.Done()
		return yorvaruntime.MCPTestResult{}, ctx.Err()
	}
	return f.result, f.err
}

func (*mcpManagerFake) ConfigureMCP(context.Context, yorvaruntime.Installation, string, yorvaruntime.MCPConfigureRequest, yorvaruntime.ProgressSink) (yorvaruntime.MCPServer, error) {
	return yorvaruntime.MCPServer{}, ErrManagementCapabilityUnsupported
}

func (*mcpManagerFake) RemoveMCP(context.Context, yorvaruntime.Installation, string, string, yorvaruntime.ProgressSink) (yorvaruntime.MCPServer, error) {
	return yorvaruntime.MCPServer{}, ErrManagementCapabilityUnsupported
}

func TestMCPManagementListsValidatedConfiguredAndReadyState(t *testing.T) {
	readyAt := time.Date(2026, 8, 25, 1, 2, 3, 0, time.UTC)
	observedAt := readyAt.Add(time.Second)
	reader := &mcpReaderFake{
		servers: []yorvaruntime.MCPServer{
			{ID: "configured-server", PresetID: "preset-a", State: yorvaruntime.MCPConfigured, ObservedAt: observedAt},
			{ID: "ready-server", PresetID: "preset-b", State: yorvaruntime.MCPReady, ReadyAt: &readyAt, ObservedAt: observedAt},
		},
		presets: []yorvaruntime.MCPPreset{{ID: "preset-a", DisplayName: "Approved A"}},
	}
	resolver := &mcpTargetResolverFake{target: testMCPManagementTarget(reader, nil)}
	service := NewMCPManagement(resolver)

	servers, err := service.ListMCPServers(context.Background(), "inst-1")
	if err != nil {
		t.Fatalf("ListMCPServers() error = %v", err)
	}
	if len(servers) != 2 || servers[0].State != yorvaruntime.MCPConfigured || servers[0].ReadyAt != nil {
		t.Fatalf("configured server projection = %#v", servers)
	}
	if servers[1].State != yorvaruntime.MCPReady || servers[1].ReadyAt == nil || !servers[1].ReadyAt.Equal(readyAt) || !servers[1].ObservedAt.Equal(observedAt) {
		t.Fatalf("READY server projection = %#v", servers[1])
	}

	presets, err := service.ListMCPPresets(context.Background(), "inst-1")
	if err != nil {
		t.Fatalf("ListMCPPresets() error = %v", err)
	}
	if len(presets) != 1 || presets[0].ID != "preset-a" || presets[0].DisplayName != "Approved A" || len(presets[0].AllowedToolIDs) != 0 || presets[0].CredentialRequired {
		t.Fatalf("preset projection = %#v", presets)
	}
}

func TestMCPManagementListsDefinitionsAtRuntimeScope(t *testing.T) {
	reader := &mcpReaderFake{presets: []yorvaruntime.MCPPreset{{ID: "preset-a", DisplayName: "Approved A", AllowedToolIDs: []string{"tool-a"}}}}
	resolver := &mcpRuntimeTargetResolverFake{
		mcpTargetResolverFake: mcpTargetResolverFake{target: testMCPManagementTarget(reader, nil)},
		runtimeTarget: RuntimeManagementTarget{
			Installation: yorvaruntime.Installation{RuntimeKind: "hermes", Path: "fixed", Version: "0.20.5", SupportState: yorvaruntime.DiscoverySupported},
			Bundle:       yorvaruntime.Bundle{MCPRead: reader},
		},
	}
	service := NewMCPManagement(resolver)
	items, err := service.ListRuntimeMCPDefinitions(context.Background(), "hermes")
	if err != nil || len(items) != 1 || items[0].ID != "preset-a" {
		t.Fatalf("runtime definitions = %#v, %v", items, err)
	}
}

func TestMCPManagementRejectsInvalidAdapterResults(t *testing.T) {
	now := time.Date(2026, 8, 25, 2, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		servers []yorvaruntime.MCPServer
		presets []yorvaruntime.MCPPreset
		list    func(*MCPManagement) error
	}{
		{
			name:    "invalid server",
			servers: []yorvaruntime.MCPServer{{ID: "https://unsafe.invalid", PresetID: "preset-a", State: yorvaruntime.MCPConfigured, ObservedAt: now}},
			list: func(service *MCPManagement) error {
				_, err := service.ListMCPServers(context.Background(), "inst-1")
				return err
			},
		},
		{
			name: "duplicate server",
			servers: []yorvaruntime.MCPServer{
				{ID: "server-a", PresetID: "preset-a", State: yorvaruntime.MCPConfigured, ObservedAt: now},
				{ID: "server-a", PresetID: "preset-a", State: yorvaruntime.MCPConfigured, ObservedAt: now},
			},
			list: func(service *MCPManagement) error {
				_, err := service.ListMCPServers(context.Background(), "inst-1")
				return err
			},
		},
		{
			name: "READY after observation",
			servers: []yorvaruntime.MCPServer{{
				ID: "server-a", PresetID: "preset-a", State: yorvaruntime.MCPReady,
				ReadyAt: timePointer(now.Add(time.Second)), ObservedAt: now,
			}},
			list: func(service *MCPManagement) error {
				_, err := service.ListMCPServers(context.Background(), "inst-1")
				return err
			},
		},
		{
			name:    "duplicate preset",
			presets: []yorvaruntime.MCPPreset{{ID: "preset-a", DisplayName: "A"}, {ID: "preset-a", DisplayName: "B"}},
			list: func(service *MCPManagement) error {
				_, err := service.ListMCPPresets(context.Background(), "inst-1")
				return err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader := &mcpReaderFake{servers: test.servers, presets: test.presets}
			service := NewMCPManagement(&mcpTargetResolverFake{target: testMCPManagementTarget(reader, nil)})
			if err := test.list(service); !errors.Is(err, ErrManagementQueryFailed) {
				t.Fatalf("error = %v, want ErrManagementQueryFailed", err)
			}
		})
	}
}

func TestMCPManagementKeepsNilReaderAndManagerUnsupported(t *testing.T) {
	service := NewMCPManagement(&mcpTargetResolverFake{target: testMCPManagementTarget(nil, nil)})
	if _, err := service.ListMCPServers(context.Background(), "inst-1"); !errors.Is(err, ErrManagementCapabilityUnsupported) {
		t.Fatalf("nil reader error = %v", err)
	}
	if _, err := service.TestMCP(context.Background(), "inst-1", "server-a", nil); !errors.Is(err, ErrManagementCapabilityUnsupported) {
		t.Fatalf("nil manager error = %v", err)
	}

	readerOnly := NewMCPManagement(&mcpTargetResolverFake{target: testMCPManagementTarget(&mcpReaderFake{}, nil)})
	servers, err := readerOnly.ListMCPServers(context.Background(), "inst-1")
	if err != nil || len(servers) != 0 {
		t.Fatalf("reader-only result = %#v, %v", servers, err)
	}
	if servers == nil {
		t.Fatal("empty configured inventory must project as an empty list")
	}

	var nilService *MCPManagement
	if _, err := nilService.ListMCPPresets(context.Background(), "inst-1"); !errors.Is(err, ErrManagementCapabilityUnsupported) {
		t.Fatalf("nil service error = %v", err)
	}
}

func TestMCPManagementTestBoundaryRequiresFreshExactReadyResult(t *testing.T) {
	now := time.Date(2026, 8, 25, 3, 0, 0, 0, time.UTC)
	manager := &mcpManagerFake{result: yorvaruntime.MCPTestResult{
		ServerID: "server-a", State: yorvaruntime.MCPReady, ToolIDs: []string{"tool-a"}, ReadyAt: &now, TestedAt: now,
	}}
	resolver := &mcpTargetResolverFake{target: testMCPManagementTarget(nil, manager)}
	service := NewMCPManagement(resolver)
	service.now = func() time.Time { return now }

	view, err := service.TestMCP(context.Background(), "inst-1", "server-a", nil)
	if err != nil {
		t.Fatalf("TestMCP() error = %v", err)
	}
	if view.State != yorvaruntime.MCPReady || !view.ReadyAt.Equal(now) || !view.TestedAt.Equal(now) || len(view.ToolIDs) != 1 || view.ToolIDs[0] != "tool-a" {
		t.Fatalf("test projection = %#v", view)
	}
	manager.result.ToolIDs[0] = "changed"
	if view.ToolIDs[0] != "tool-a" {
		t.Fatal("test projection aliases adapter-owned tool IDs")
	}

	invalidResults := []yorvaruntime.MCPTestResult{
		{ServerID: "other-server", State: yorvaruntime.MCPReady, ReadyAt: &now, TestedAt: now},
		{ServerID: "server-a", State: yorvaruntime.MCPConfigured, TestedAt: now},
		{ServerID: "server-a", State: yorvaruntime.MCPReady, ReadyAt: timePointer(now.Add(-time.Second)), TestedAt: now},
		{ServerID: "server-a", State: yorvaruntime.MCPReady, ReadyAt: timePointer(now.Add(-time.Second)), TestedAt: now.Add(-time.Second)},
		{ServerID: "server-a", State: yorvaruntime.MCPReady, ReadyAt: timePointer(now.Add(time.Second)), TestedAt: now.Add(time.Second)},
	}
	for _, result := range invalidResults {
		manager.result = result
		if _, err := service.TestMCP(context.Background(), "inst-1", "server-a", nil); !errors.Is(err, ErrManagementQueryFailed) {
			t.Fatalf("result %#v error = %v, want ErrManagementQueryFailed", result, err)
		}
	}

	calls := manager.calls
	if _, err := service.TestMCP(context.Background(), "inst-1", "https://unsafe.invalid", nil); !errors.Is(err, ErrManagementQueryFailed) {
		t.Fatalf("unsafe server ID error = %v", err)
	}
	if manager.calls != calls || resolver.calls == 0 {
		t.Fatalf("unsafe request reached manager: calls = %d, want %d", manager.calls, calls)
	}
}

func TestMCPManagementNormalizesAdapterErrors(t *testing.T) {
	reader := &mcpReaderFake{err: errors.New("credential-canary https://private.invalid/path")}
	service := NewMCPManagement(&mcpTargetResolverFake{target: testMCPManagementTarget(reader, nil)})
	if _, err := service.ListMCPServers(context.Background(), "inst-1"); !errors.Is(err, ErrManagementQueryFailed) || err.Error() != ErrManagementQueryFailed.Error() {
		t.Fatalf("adapter error = %v", err)
	}
}

func TestMCPManagementCancelsRunningOperation(t *testing.T) {
	db, err := sqlite.Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	instanceID := seedMCPManagedTestInstance(t, db)
	now := time.Now().UTC()
	if err := db.UpsertManagedMCPBinding(context.Background(), sqlite.ManagedMCPBinding{InstanceID: instanceID, ServerID: "server-a", PresetID: "server-a", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	manager := &mcpManagerFake{started: make(chan struct{})}
	service := NewMCPManagement(&mcpTargetResolverFake{target: testMCPManagementTarget(nil, manager)})
	service.db = db
	started, err := service.StartTest(context.Background(), instanceID, "server-a", "mcp-cancel-key")
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-manager.started:
	case <-time.After(time.Second):
		t.Fatal("MCP worker did not start")
	}
	cancelled, err := service.CancelMCPOperation(context.Background(), started.Operation.ID)
	if err != nil || cancelled.Status != operation.StatusCancelled {
		t.Fatalf("CancelMCPOperation() = %#v, %v", cancelled, err)
	}
	time.Sleep(20 * time.Millisecond)
	stored, err := db.GetOperation(context.Background(), started.Operation.ID)
	if err != nil || stored.Status != operation.StatusCancelled {
		t.Fatalf("stored operation = %#v, %v", stored, err)
	}
}

func TestMCPInstallOperationRequiresCredentialTestAndAuthoritativeReadback(t *testing.T) {
	db, err := sqlite.Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	instanceID := seedMCPManagedTestInstance(t, db)
	adapter := &mcpLifecycleFake{}
	service := NewMCPManagement(&mcpTargetResolverFake{target: testMCPManagementTarget(adapter, adapter)})
	service.db = db
	started, err := service.StartInstall(context.Background(), instanceID, "preset-a", []byte("test-token"), nil, []string{"tool-a"}, "mcp-complete-key")
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		stored, readErr := db.GetOperation(context.Background(), started.Operation.ID)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if operation.IsTerminal(stored.Status) {
			if stored.Status != operation.StatusSucceeded {
				t.Fatalf("operation = %#v", stored)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("MCP lifecycle operation did not complete")
		}
		time.Sleep(10 * time.Millisecond)
	}
	want := []string{"install", "credential", "tools", "test", "readback"}
	if len(adapter.sequence) != len(want) {
		t.Fatalf("sequence = %#v", adapter.sequence)
	}
	for index := range want {
		if adapter.sequence[index] != want[index] {
			t.Fatalf("sequence = %#v, want %#v", adapter.sequence, want)
		}
	}
	servers, err := service.ListMCPServers(context.Background(), instanceID)
	if err != nil || len(servers) != 1 || servers[0].Ownership != "YORVA_MANAGED" {
		t.Fatalf("managed readback = %#v, %v", servers, err)
	}
}

func seedMCPManagedTestInstance(t *testing.T, db *sqlite.Database) string {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	localNode, err := db.LoadOrCreateNode(ctx, node.LocalMetadata{Name: "MCP test", Hostname: "localhost", Platform: "windows", Architecture: "amd64", NodeVersion: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertAcceptedInstallation(ctx, sqlite.AcceptedInstallation{
		ID: "rtinst-mcp", NodeID: localNode.ID, RuntimeKind: "hermes", InstallPath: t.TempDir(), Version: "0.20.5",
		SupportState: yorvaruntime.DiscoverySupported, Status: "ACCEPTED", LastDetectedAt: now, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.ApplyInstanceSnapshot(ctx, "rtinst-mcp", []sqlite.InstanceSnapshotEntry{{NativeID: "profile-a", Default: true}}, now); err != nil {
		t.Fatal(err)
	}
	items, err := db.ListInstances(ctx, "rtinst-mcp")
	if err != nil || len(items) != 1 {
		t.Fatalf("seed MCP instance = %#v, %v", items, err)
	}
	return items[0].ID
}

func testMCPManagementTarget(reader yorvaruntime.MCPReader, manager yorvaruntime.MCPManager) ManagementTarget {
	return ManagementTarget{
		Installation: yorvaruntime.Installation{RuntimeKind: "hermes", Version: "0.20.5", SupportState: yorvaruntime.DiscoverySupported},
		NativeID:     "profile-a",
		Bundle:       yorvaruntime.Bundle{MCPRead: reader, MCPMutate: manager},
	}
}

func timePointer(value time.Time) *time.Time {
	return &value
}

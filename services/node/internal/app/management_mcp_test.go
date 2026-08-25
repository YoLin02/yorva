package app

import (
	"context"
	"errors"
	"testing"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type mcpTargetResolverFake struct {
	target ManagementTarget
	err    error
	calls  int
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
	result yorvaruntime.MCPTestResult
	err    error
	calls  int
}

func (*mcpManagerFake) InstallMCPPreset(context.Context, yorvaruntime.Installation, string, yorvaruntime.MCPInstallRequest, yorvaruntime.ProgressSink) (yorvaruntime.MCPServer, error) {
	return yorvaruntime.MCPServer{}, ErrManagementCapabilityUnsupported
}

func (*mcpManagerFake) AuthenticateMCP(context.Context, yorvaruntime.Installation, string, yorvaruntime.MCPAuthenticateRequest, yorvaruntime.ProgressSink) (yorvaruntime.MCPServer, error) {
	return yorvaruntime.MCPServer{}, ErrManagementCapabilityUnsupported
}

func (f *mcpManagerFake) TestMCP(_ context.Context, _ yorvaruntime.Installation, _ string, _ string, _ yorvaruntime.ProgressSink) (yorvaruntime.MCPTestResult, error) {
	f.calls++
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
	if len(presets) != 1 || presets[0] != (MCPPresetView{ID: "preset-a", DisplayName: "Approved A"}) {
		t.Fatalf("preset projection = %#v", presets)
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

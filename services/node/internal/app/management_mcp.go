package app

import (
	"context"
	"errors"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const managementMCPCollectionLimit = 256

type MCPServerView struct {
	ID         string
	PresetID   string
	State      yorvaruntime.MCPState
	ReadyAt    *time.Time
	ObservedAt time.Time
}

type MCPPresetView struct {
	ID          string
	DisplayName string
}

// MCPTestView is the safe result consumed by a future Operation worker. It is
// deliberately not exposed through a synchronous HTTP mutation endpoint.
type MCPTestView struct {
	ServerID string
	State    yorvaruntime.MCPState
	ToolIDs  []string
	ReadyAt  time.Time
	TestedAt time.Time
}

// MCPManagement owns Runtime-neutral validation for the closed MCP read and
// test boundaries. The Runtime adapter remains authoritative for live state.
type MCPManagement struct {
	targets ManagementTargetResolver
	now     func() time.Time
}

func NewMCPManagement(targets ManagementTargetResolver) *MCPManagement {
	return &MCPManagement{targets: targets, now: time.Now}
}

func (s *MCPManagement) ListMCPServers(ctx context.Context, instanceID string) ([]MCPServerView, error) {
	target, err := s.resolve(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if target.Bundle.MCPRead == nil {
		return nil, ErrManagementCapabilityUnsupported
	}

	servers, err := target.Bundle.MCPRead.ListMCPServers(ctx, target.Installation, target.NativeID)
	if err != nil {
		return nil, managementMCPError(ctx, err)
	}
	if len(servers) > managementMCPCollectionLimit {
		return nil, ErrManagementQueryFailed
	}

	seen := make(map[string]struct{}, len(servers))
	views := make([]MCPServerView, 0, len(servers))
	for _, server := range servers {
		if err := server.Validate(); err != nil {
			return nil, ErrManagementQueryFailed
		}
		if _, exists := seen[server.ID]; exists {
			return nil, ErrManagementQueryFailed
		}
		seen[server.ID] = struct{}{}

		readyAt, valid := validatedMCPReadyAt(server.State, server.ReadyAt, server.ObservedAt)
		if !valid {
			return nil, ErrManagementQueryFailed
		}
		views = append(views, MCPServerView{
			ID:         server.ID,
			PresetID:   server.PresetID,
			State:      server.State,
			ReadyAt:    readyAt,
			ObservedAt: server.ObservedAt.UTC(),
		})
	}
	return views, nil
}

func (s *MCPManagement) ListMCPPresets(ctx context.Context, instanceID string) ([]MCPPresetView, error) {
	target, err := s.resolve(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if target.Bundle.MCPRead == nil {
		return nil, ErrManagementCapabilityUnsupported
	}

	presets, err := target.Bundle.MCPRead.ListMCPPresets(ctx, target.Installation, target.NativeID)
	if err != nil {
		return nil, managementMCPError(ctx, err)
	}
	if len(presets) > managementMCPCollectionLimit {
		return nil, ErrManagementQueryFailed
	}

	seen := make(map[string]struct{}, len(presets))
	views := make([]MCPPresetView, 0, len(presets))
	for _, preset := range presets {
		if err := preset.Validate(); err != nil {
			return nil, ErrManagementQueryFailed
		}
		if _, exists := seen[preset.ID]; exists {
			return nil, ErrManagementQueryFailed
		}
		seen[preset.ID] = struct{}{}
		views = append(views, MCPPresetView{ID: preset.ID, DisplayName: preset.DisplayName})
	}
	return views, nil
}

// TestMCP is a bounded application worker boundary, not an HTTP handler. Its
// caller must own the durable Operation, timeout, cancellation, and ProgressSink.
func (s *MCPManagement) TestMCP(ctx context.Context, instanceID, serverID string, progress yorvaruntime.ProgressSink) (MCPTestView, error) {
	if err := (yorvaruntime.MCPConfigureRequest{ServerID: serverID}).Validate(); err != nil {
		return MCPTestView{}, ErrManagementQueryFailed
	}
	target, err := s.resolve(ctx, instanceID)
	if err != nil {
		return MCPTestView{}, err
	}
	if target.Bundle.MCPMutate == nil {
		return MCPTestView{}, ErrManagementCapabilityUnsupported
	}

	startedAt := s.now().UTC()
	result, err := target.Bundle.MCPMutate.TestMCP(ctx, target.Installation, target.NativeID, serverID, progress)
	completedAt := s.now().UTC()
	if err != nil {
		return MCPTestView{}, managementMCPError(ctx, err)
	}
	if err := result.Validate(); err != nil || result.ServerID != serverID || result.State != yorvaruntime.MCPReady ||
		result.ReadyAt == nil || !result.ReadyAt.Equal(result.TestedAt) || result.TestedAt.Before(startedAt) || result.TestedAt.After(completedAt) {
		return MCPTestView{}, ErrManagementQueryFailed
	}

	return MCPTestView{
		ServerID: result.ServerID,
		State:    result.State,
		ToolIDs:  append([]string(nil), result.ToolIDs...),
		ReadyAt:  result.ReadyAt.UTC(),
		TestedAt: result.TestedAt.UTC(),
	}, nil
}

func (s *MCPManagement) resolve(ctx context.Context, instanceID string) (ManagementTarget, error) {
	if s == nil || s.targets == nil {
		return ManagementTarget{}, ErrManagementCapabilityUnsupported
	}
	target, err := s.targets.ResolveManagementTarget(ctx, instanceID)
	if err != nil {
		return ManagementTarget{}, managementMCPError(ctx, err)
	}
	return target, nil
}

func validatedMCPReadyAt(state yorvaruntime.MCPState, readyAt *time.Time, observedAt time.Time) (*time.Time, bool) {
	if state != yorvaruntime.MCPReady {
		return nil, readyAt == nil
	}
	if readyAt == nil || readyAt.IsZero() || readyAt.After(observedAt) {
		return nil, false
	}
	readyUTC := readyAt.UTC()
	return &readyUTC, true
}

func managementMCPError(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	case errors.Is(err, ErrInstanceNotFound), errors.Is(err, ErrInstanceNotAvailable), errors.Is(err, ErrRuntimeNotSupported),
		errors.Is(err, ErrManagementCapabilityUnsupported), errors.Is(err, ErrManagementQueryFailed):
		return err
	default:
		return ErrManagementQueryFailed
	}
}

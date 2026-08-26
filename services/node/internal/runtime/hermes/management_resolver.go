package hermes

import (
	"context"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

// ManagementResolver combines Profile-owned resource inventory with the
// optional authenticated loopback health reader. Resource visibility does not
// depend on a running Hermes API listener.
type ManagementResolver struct {
	resources *ProfileResourceReader
	api       *APIManagementReader
	mcp       yorvaruntime.MCPReader
}

func NewManagementResolver() *ManagementResolver {
	resources := NewProfileResourceReader()
	return &ManagementResolver{resources: resources, api: NewAPIManagementReader(), mcp: resources}
}

func newManagementResolverWithMCP(reader yorvaruntime.MCPReader) *ManagementResolver {
	resolver := NewManagementResolver()
	if reader != nil {
		resolver.mcp = reader
	}
	return resolver
}

func (r *ManagementResolver) ResolveInstanceManagement(ctx context.Context, installation yorvaruntime.Installation, nativeID string) (yorvaruntime.InstanceManagementFeatures, error) {
	if r == nil || r.resources == nil {
		return yorvaruntime.InstanceManagementFeatures{}, errProfileResourcesUnavailable
	}
	if _, err := r.resources.resolveProfile(ctx, installation, nativeID); err != nil {
		return yorvaruntime.InstanceManagementFeatures{}, err
	}
	mcpReader := r.mcp
	if mcpReader == nil {
		mcpReader = r.resources
	}
	features := yorvaruntime.InstanceManagementFeatures{Health: r.resources, Logs: r.resources, SkillRead: r.resources, MCPRead: mcpReader}
	if r.api != nil {
		if apiFeatures, err := r.api.ResolveInstanceManagement(ctx, installation, nativeID); err == nil {
			features.Health = apiFeatures.Health
		}
	}
	return features, nil
}

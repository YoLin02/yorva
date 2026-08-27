package hermes

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes/mcpmanagement"
	"gopkg.in/yaml.v3"
)

const mcpConfigFileLimit = 256 * 1024

var (
	errMCPProfileConflict = errors.New("Hermes MCP Profile changed concurrently")
	errMCPProfileUnsafe   = errors.New("Hermes MCP Profile configuration is unsafe")
	errMCPNotManaged      = errors.New("Hermes MCP definition is not YORVA managed")
)

// ProfileMCPManager owns the reviewed descriptor to exact-Profile compatibility
// boundary. Public callers provide only reviewed preset/server/tool IDs and a
// request-lifetime credential; transport details never cross this boundary.
type ProfileMCPManager struct {
	profiles *ProfileResourceReader
	registry mcpmanagement.Registry
	client   *http.Client
	local    *mcpmanagement.LocalTestServer
	now      func() time.Time
	mu       sync.Mutex
	ready    map[string]time.Time
}

func NewProfileMCPManager() *ProfileMCPManager {
	return newProfileMCPManager(NewProfileResourceReader(), mcpmanagement.NewRegistry(), mcpmanagement.NewReviewedHTTPSClient())
}

// NewProductionProfileMCPManager starts the daemon-owned loopback MCP test
// server and wires the production reviewed catalog to its private client.
func NewProductionProfileMCPManager() (*ProfileMCPManager, error) {
	server, err := mcpmanagement.StartLocalTestServer()
	if err != nil {
		return nil, err
	}
	client := server.Client(mcpmanagement.NewReviewedHTTPSClient())
	if client == nil {
		_ = server.Close()
		return nil, errMCPProfileUnsafe
	}
	manager := newProfileMCPManager(NewProfileResourceReader(), mcpmanagement.NewRegistry(), client)
	manager.local = server
	return manager, nil
}

func (m *ProfileMCPManager) Close() error {
	if m == nil || m.local == nil {
		return nil
	}
	return m.local.Close()
}

func newProfileMCPManager(profiles *ProfileResourceReader, registry mcpmanagement.Registry, client *http.Client) *ProfileMCPManager {
	return &ProfileMCPManager{profiles: profiles, registry: registry, client: client, now: func() time.Time { return time.Now().UTC() }, ready: make(map[string]time.Time)}
}

func (m *ProfileMCPManager) ListMCPPresets(ctx context.Context, installation yorvaruntime.Installation, nativeID string) ([]yorvaruntime.MCPPreset, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if m == nil || m.profiles == nil || m.client == nil || installation.RuntimeKind != Kind || installation.Path == "" || installation.Version == "" || installation.SupportState != yorvaruntime.DiscoverySupported {
		return nil, errMCPProfileUnsafe
	}
	if nativeID != "" {
		if _, err := m.profileRoot(ctx, installation, nativeID); err != nil {
			return nil, err
		}
	}
	catalog := m.registry.Catalog()
	items := make([]yorvaruntime.MCPPreset, 0, len(catalog))
	for _, preset := range catalog {
		item := yorvaruntime.MCPPreset{
			ID: preset.ID, DisplayName: preset.DisplayName, Description: preset.Description,
			HomepageURL: preset.HomepageURL, DocumentationURL: preset.DocumentationURL,
			AllowedToolIDs:     append([]string(nil), preset.AllowedToolIDs...),
			CredentialRequired: preset.CredentialClass != mcpmanagement.CredentialClassNone,
		}
		if item.Validate() != nil {
			return nil, errMCPProfileUnsafe
		}
		items = append(items, item)
	}
	return items, nil
}

func (m *ProfileMCPManager) ListMCPServers(ctx context.Context, installation yorvaruntime.Installation, nativeID string) ([]yorvaruntime.MCPServer, error) {
	root, err := m.profileRoot(ctx, installation, nativeID)
	if err != nil {
		return nil, err
	}
	config, err := readBoundedYAML(filepath.Join(root, "config.yaml"))
	if err != nil {
		return nil, errMCPProfileUnsafe
	}
	servers := mappingValue(config, "mcp_servers")
	if servers == nil {
		return []yorvaruntime.MCPServer{}, nil
	}
	if servers.Kind != yaml.MappingNode || len(servers.Content)%2 != 0 || len(servers.Content)/2 > profileResourceLimit {
		return nil, errMCPProfileUnsafe
	}
	observedAt := m.clock()
	items := make([]yorvaruntime.MCPServer, 0, len(servers.Content)/2)
	for index := 0; index < len(servers.Content); index += 2 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		id := servers.Content[index].Value
		item := yorvaruntime.MCPServer{ID: id, PresetID: id, State: yorvaruntime.MCPConfigured, ObservedAt: observedAt}
		if selection, resolveErr := m.registry.Resolve(id); resolveErr == nil {
			toolIDs, matches := managedMCPNodeToolIDs(selection, servers.Content[index+1])
			if !matches {
				items = append(items, item)
				continue
			}
			item.Managed = true
			item.EnabledToolIDs = toolIDs
			status, statusErr := m.credentialStatus(root, selection)
			if statusErr != nil {
				return nil, statusErr
			}
			if status == mcpmanagement.CredentialStatusNotConfigured {
				item.State = yorvaruntime.MCPAuthRequired
			}
			if readyAt, ok := m.readyObservation(nativeID, id, observedAt); ok && item.State == yorvaruntime.MCPConfigured {
				item.State, item.ReadyAt = yorvaruntime.MCPReady, &readyAt
			}
		} else {
			item.EnabledToolIDs = extractMCPToolIDs(servers.Content[index+1])
			if readyAt, ok := m.readyObservation(nativeID, id, observedAt); ok {
				item.State, item.ReadyAt = yorvaruntime.MCPReady, &readyAt
			}
		}
		if item.Validate() != nil {
			return nil, errMCPProfileUnsafe
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return items, nil
}

func extractMCPToolIDs(node *yaml.Node) []string {
	scope := mappingValue(node, "tools")
	include := mappingValue(scope, "include")
	if include == nil || include.Kind != yaml.SequenceNode {
		return nil
	}
	result := make([]string, 0, len(include.Content))
	for _, item := range include.Content {
		if item.Kind != yaml.ScalarNode {
			return nil
		}
		result = append(result, item.Value)
	}
	return result
}

func (m *ProfileMCPManager) InstallMCPPreset(ctx context.Context, installation yorvaruntime.Installation, nativeID string, request yorvaruntime.MCPInstallRequest, _ yorvaruntime.ProgressSink) (yorvaruntime.MCPServer, error) {
	if err := request.Validate(); err != nil {
		return yorvaruntime.MCPServer{}, err
	}
	selection, err := m.registry.Resolve(request.PresetID)
	if err != nil {
		return yorvaruntime.MCPServer{}, err
	}
	root, err := m.profileRoot(ctx, installation, nativeID)
	if err != nil {
		return yorvaruntime.MCPServer{}, err
	}
	path := filepath.Join(root, "config.yaml")
	if err := mutateMCPConfig(path, func(config *yaml.Node) error {
		servers := ensureYAMLMapping(config, "mcp_servers")
		if yamlMappingIndex(servers, request.PresetID) >= 0 {
			return errMCPProfileConflict
		}
		appendYAMLMapping(servers, request.PresetID, managedMCPNode(selection, selection.AllowedToolIDs()))
		return nil
	}); err != nil {
		return yorvaruntime.MCPServer{}, err
	}
	m.invalidate(nativeID, request.PresetID)
	return m.readExact(ctx, installation, nativeID, request.PresetID)
}

func (m *ProfileMCPManager) AuthenticateMCP(ctx context.Context, installation yorvaruntime.Installation, nativeID string, request yorvaruntime.MCPAuthenticateRequest, _ yorvaruntime.ProgressSink) (yorvaruntime.MCPServer, error) {
	if err := request.Validate(); err != nil {
		return yorvaruntime.MCPServer{}, err
	}
	selection, err := m.registry.Resolve(request.ServerID)
	if err != nil || selection.CredentialClass() != mcpmanagement.CredentialClassStaticBearer || selection.ValidateCredential(request.Credential) != nil {
		return yorvaruntime.MCPServer{}, errMCPNotManaged
	}
	root, err := m.profileRoot(ctx, installation, nativeID)
	if err != nil {
		return yorvaruntime.MCPServer{}, err
	}
	if err := m.requireManagedConfig(root, request.ServerID, selection); err != nil {
		return yorvaruntime.MCPServer{}, err
	}
	store := credentialStore{root: m.profiles.root}
	path, err := store.credentialPath(nativeID)
	if err != nil {
		return yorvaruntime.MCPServer{}, errMCPProfileUnsafe
	}
	snapshot, err := observeCredentialFile(path)
	if err != nil {
		return yorvaruntime.MCPServer{}, errMCPProfileUnsafe
	}
	updated, err := replaceCredentialAssignment(snapshot.data, selection.CredentialKey(), request.Credential)
	if err != nil {
		return yorvaruntime.MCPServer{}, errMCPProfileUnsafe
	}
	defer clearCredentialBytes(updated)
	if err := store.commit(path, snapshot, updated); err != nil {
		return yorvaruntime.MCPServer{}, err
	}
	m.invalidate(nativeID, request.ServerID)
	return m.readExact(ctx, installation, nativeID, request.ServerID)
}

func (m *ProfileMCPManager) ConfigureMCP(ctx context.Context, installation yorvaruntime.Installation, nativeID string, request yorvaruntime.MCPConfigureRequest, _ yorvaruntime.ProgressSink) (yorvaruntime.MCPServer, error) {
	if err := request.Validate(); err != nil {
		return yorvaruntime.MCPServer{}, err
	}
	selection, err := m.registry.Resolve(request.ServerID)
	if err != nil || !toolSelectionAllowed(selection, request.EnabledToolIDs) {
		return yorvaruntime.MCPServer{}, errMCPNotManaged
	}
	root, err := m.profileRoot(ctx, installation, nativeID)
	if err != nil {
		return yorvaruntime.MCPServer{}, err
	}
	path := filepath.Join(root, "config.yaml")
	err = mutateMCPConfig(path, func(config *yaml.Node) error {
		servers := mappingValue(config, "mcp_servers")
		index := yamlMappingIndex(servers, request.ServerID)
		if index < 0 || !managedMCPNodeMatches(selection, servers.Content[index+1]) {
			return errMCPNotManaged
		}
		servers.Content[index+1] = managedMCPNode(selection, request.EnabledToolIDs)
		return nil
	})
	if err != nil {
		return yorvaruntime.MCPServer{}, err
	}
	m.invalidate(nativeID, request.ServerID)
	return m.readExact(ctx, installation, nativeID, request.ServerID)
}

func (m *ProfileMCPManager) TestMCP(ctx context.Context, installation yorvaruntime.Installation, nativeID, serverID string, _ yorvaruntime.ProgressSink) (yorvaruntime.MCPTestResult, error) {
	selection, err := m.registry.Resolve(serverID)
	if err != nil {
		return yorvaruntime.MCPTestResult{}, errMCPNotManaged
	}
	root, err := m.profileRoot(ctx, installation, nativeID)
	if err != nil {
		return yorvaruntime.MCPTestResult{}, err
	}
	if err := m.requireManagedConfig(root, serverID, selection); err != nil {
		return yorvaruntime.MCPTestResult{}, err
	}
	credential, err := m.readCredential(root, selection)
	if err != nil {
		return yorvaruntime.MCPTestResult{}, err
	}
	defer clearCredentialBytes(credential)
	scope, err := mcpmanagement.NewProfileScope(nativeID)
	if err != nil {
		return yorvaruntime.MCPTestResult{}, err
	}
	result, err := mcpmanagement.ProbeReviewedHTTPS(ctx, selection, scope, credential, m.client, m.clock)
	if err != nil {
		m.invalidate(nativeID, serverID)
		return yorvaruntime.MCPTestResult{}, err
	}
	readyAt := result.ObservedAt()
	m.mu.Lock()
	m.ready[m.readyKey(nativeID, serverID)] = readyAt
	m.mu.Unlock()
	return yorvaruntime.MCPTestResult{ServerID: serverID, State: yorvaruntime.MCPReady, ToolIDs: result.ToolIDs(), ReadyAt: &readyAt, TestedAt: readyAt}, nil
}

func (m *ProfileMCPManager) RemoveMCP(ctx context.Context, installation yorvaruntime.Installation, nativeID, serverID string, _ yorvaruntime.ProgressSink) (yorvaruntime.MCPServer, error) {
	selection, err := m.registry.Resolve(serverID)
	if err != nil {
		return yorvaruntime.MCPServer{}, errMCPNotManaged
	}
	root, err := m.profileRoot(ctx, installation, nativeID)
	if err != nil {
		return yorvaruntime.MCPServer{}, err
	}
	path := filepath.Join(root, "config.yaml")
	err = mutateMCPConfig(path, func(config *yaml.Node) error {
		servers := mappingValue(config, "mcp_servers")
		index := yamlMappingIndex(servers, serverID)
		if index < 0 || !managedMCPNodeMatches(selection, servers.Content[index+1]) {
			return errMCPNotManaged
		}
		servers.Content = append(servers.Content[:index], servers.Content[index+2:]...)
		if len(servers.Content) == 0 {
			removeYAMLMapping(config, "mcp_servers")
		}
		return nil
	})
	if err != nil {
		return yorvaruntime.MCPServer{}, err
	}
	if selection.CredentialClass() == mcpmanagement.CredentialClassStaticBearer {
		store := credentialStore{root: m.profiles.root}
		envPath, pathErr := store.credentialPath(nativeID)
		if pathErr != nil {
			return yorvaruntime.MCPServer{}, errMCPProfileUnsafe
		}
		snapshot, observeErr := observeCredentialFile(envPath)
		if observeErr != nil {
			return yorvaruntime.MCPServer{}, errMCPProfileUnsafe
		}
		updated, found, deleteErr := deleteCredentialAssignment(snapshot.data, selection.CredentialKey())
		if deleteErr != nil {
			return yorvaruntime.MCPServer{}, errMCPProfileUnsafe
		}
		if found {
			defer clearCredentialBytes(updated)
			if commitErr := store.commit(envPath, snapshot, updated); commitErr != nil {
				return yorvaruntime.MCPServer{}, commitErr
			}
		}
	}
	m.invalidate(nativeID, serverID)
	items, readErr := m.ListMCPServers(ctx, installation, nativeID)
	if readErr != nil {
		return yorvaruntime.MCPServer{}, readErr
	}
	for _, item := range items {
		if item.ID == serverID {
			return yorvaruntime.MCPServer{}, errMCPProfileConflict
		}
	}
	return yorvaruntime.MCPServer{ID: serverID, PresetID: serverID, State: yorvaruntime.MCPNotConfigured, ObservedAt: m.clock()}, nil
}

func (m *ProfileMCPManager) profileRoot(ctx context.Context, installation yorvaruntime.Installation, nativeID string) (string, error) {
	if m == nil || m.profiles == nil || m.client == nil {
		return "", errMCPProfileUnsafe
	}
	return m.profiles.resolveProfile(ctx, installation, nativeID)
}

func (m *ProfileMCPManager) readExact(ctx context.Context, installation yorvaruntime.Installation, nativeID, serverID string) (yorvaruntime.MCPServer, error) {
	items, err := m.ListMCPServers(ctx, installation, nativeID)
	if err != nil {
		return yorvaruntime.MCPServer{}, err
	}
	for _, item := range items {
		if item.ID == serverID {
			return item, nil
		}
	}
	return yorvaruntime.MCPServer{}, errMCPNotManaged
}

func (m *ProfileMCPManager) requireManagedConfig(root, serverID string, selection mcpmanagement.Selection) error {
	config, err := readBoundedYAML(filepath.Join(root, "config.yaml"))
	if err != nil {
		return errMCPProfileUnsafe
	}
	servers := mappingValue(config, "mcp_servers")
	index := yamlMappingIndex(servers, serverID)
	if index < 0 || !managedMCPNodeMatches(selection, servers.Content[index+1]) {
		return errMCPNotManaged
	}
	return nil
}

func (m *ProfileMCPManager) credentialStatus(root string, selection mcpmanagement.Selection) (mcpmanagement.CredentialStatus, error) {
	if selection.CredentialClass() == mcpmanagement.CredentialClassNone {
		return mcpmanagement.CredentialStatusNotRequired, nil
	}
	snapshot, err := observeCredentialFile(filepath.Join(root, ".env"))
	if err != nil {
		return mcpmanagement.CredentialStatusUnknown, errMCPProfileUnsafe
	}
	configured, err := credentialConfigured(snapshot.data, selection.CredentialKey())
	if err != nil {
		return mcpmanagement.CredentialStatusUnknown, errMCPProfileUnsafe
	}
	if configured {
		return mcpmanagement.CredentialStatusConfigured, nil
	}
	return mcpmanagement.CredentialStatusNotConfigured, nil
}

func (m *ProfileMCPManager) readCredential(root string, selection mcpmanagement.Selection) ([]byte, error) {
	if selection.CredentialClass() == mcpmanagement.CredentialClassNone {
		return nil, nil
	}
	snapshot, err := observeCredentialFile(filepath.Join(root, ".env"))
	if err != nil {
		return nil, errMCPProfileUnsafe
	}
	for _, line := range splitCredentialLines(snapshot.data) {
		value, match := credentialAssignment(line.body, selection.CredentialKey())
		if !match {
			continue
		}
		value = bytes.TrimSpace(value)
		if len(value) < 2 || value[0] != '"' || value[len(value)-1] != '"' {
			return nil, errMCPProfileUnsafe
		}
		value = value[1 : len(value)-1]
		value = bytes.ReplaceAll(value, []byte(`\"`), []byte(`"`))
		value = bytes.ReplaceAll(value, []byte(`\\`), []byte(`\`))
		if err := selection.ValidateCredential(value); err != nil {
			return nil, errMCPProfileUnsafe
		}
		return append([]byte(nil), value...), nil
	}
	return nil, errMCPProfileUnsafe
}

func (m *ProfileMCPManager) clock() time.Time {
	if m != nil && m.now != nil {
		return m.now().UTC()
	}
	return time.Now().UTC()
}

func (m *ProfileMCPManager) readyKey(profileID, serverID string) string {
	return profileID + "\x00" + serverID
}

func (m *ProfileMCPManager) invalidate(profileID, serverID string) {
	m.mu.Lock()
	delete(m.ready, m.readyKey(profileID, serverID))
	m.mu.Unlock()
}

func (m *ProfileMCPManager) readyObservation(profileID, serverID string, now time.Time) (time.Time, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	observedAt, ok := m.ready[m.readyKey(profileID, serverID)]
	if !ok || !now.Before(observedAt.Add(mcpmanagement.ReadyTTL)) {
		delete(m.ready, m.readyKey(profileID, serverID))
		return time.Time{}, false
	}
	return observedAt, true
}

func managedMCPNode(selection mcpmanagement.Selection, tools []string) *yaml.Node {
	value := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	appendYAMLMapping(value, "url", yamlScalar(selectionURL(selection)))
	if template := selection.AuthorizationTemplate(); template != "" {
		headers := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		appendYAMLMapping(headers, "Authorization", yamlScalar(template))
		appendYAMLMapping(value, "headers", headers)
	}
	appendYAMLMapping(value, "enabled", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"})
	selected := append([]string(nil), tools...)
	sort.Strings(selected)
	toolNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	include := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for _, tool := range selected {
		include.Content = append(include.Content, yamlScalar(tool))
	}
	appendYAMLMapping(toolNode, "include", include)
	appendYAMLMapping(value, "tools", toolNode)
	return value
}

func selectionURL(selection mcpmanagement.Selection) string {
	value, _ := selection.HTTPSURL()
	return value
}

func managedMCPNodeMatches(selection mcpmanagement.Selection, node *yaml.Node) bool {
	_, matches := managedMCPNodeToolIDs(selection, node)
	return matches
}

func managedMCPNodeToolIDs(selection mcpmanagement.Selection, node *yaml.Node) ([]string, bool) {
	if node == nil || node.Kind != yaml.MappingNode || len(node.Content)%2 != 0 {
		return nil, false
	}
	allowedKeys := map[string]bool{"url": true, "headers": true, "enabled": true, "tools": true}
	seenKeys := make(map[string]struct{}, len(node.Content)/2)
	for index := 0; index < len(node.Content); index += 2 {
		key := node.Content[index].Value
		if !allowedKeys[key] {
			return nil, false
		}
		if _, duplicate := seenKeys[key]; duplicate {
			return nil, false
		}
		seenKeys[key] = struct{}{}
	}
	url := mappingValue(node, "url")
	enabled := mappingValue(node, "enabled")
	if url == nil || url.Kind != yaml.ScalarNode || url.Value != selectionURL(selection) || enabled == nil || enabled.Tag != "!!bool" || enabled.Value != "true" {
		return nil, false
	}
	headers := mappingValue(node, "headers")
	if template := selection.AuthorizationTemplate(); template != "" {
		if headers == nil || headers.Kind != yaml.MappingNode || len(headers.Content) != 2 || headers.Content[0].Value != "Authorization" || headers.Content[1].Value != template {
			return nil, false
		}
	} else if headers != nil {
		return nil, false
	}
	toolScope := mappingValue(node, "tools")
	if toolScope == nil || toolScope.Kind != yaml.MappingNode || len(toolScope.Content) != 2 || toolScope.Content[0].Value != "include" {
		return nil, false
	}
	tools := mappingValue(toolScope, "include")
	if tools == nil || tools.Kind != yaml.SequenceNode {
		return nil, false
	}
	observed := make([]string, 0, len(tools.Content))
	for _, item := range tools.Content {
		if item.Kind != yaml.ScalarNode || item.Tag != "!!str" {
			return nil, false
		}
		observed = append(observed, item.Value)
	}
	if !toolSelectionAllowed(selection, observed) {
		return nil, false
	}
	sort.Strings(observed)
	return observed, true
}

func toolSelectionAllowed(selection mcpmanagement.Selection, selected []string) bool {
	allowed := selection.AllowedToolIDs()
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, tool := range allowed {
		allowedSet[tool] = struct{}{}
	}
	seen := make(map[string]struct{}, len(selected))
	for _, tool := range selected {
		if _, ok := allowedSet[tool]; !ok {
			return false
		}
		if _, duplicate := seen[tool]; duplicate {
			return false
		}
		seen[tool] = struct{}{}
	}
	return len(selected) > 0
}

type mcpConfigSnapshot struct {
	exists bool
	data   []byte
	mode   os.FileMode
}

func mutateMCPConfig(path string, mutate func(*yaml.Node) error) error {
	snapshot, root, err := observeMCPConfig(path)
	if err != nil {
		return err
	}
	if root == nil {
		root = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	}
	if err := mutate(root); err != nil {
		return err
	}
	var output bytes.Buffer
	encoder := yaml.NewEncoder(&output)
	encoder.SetIndent(2)
	document := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{root}}
	if err := encoder.Encode(document); err != nil || encoder.Close() != nil || output.Len() > mcpConfigFileLimit {
		return errMCPProfileUnsafe
	}
	return commitMCPConfig(path, snapshot, output.Bytes())
}

func observeMCPConfig(path string) (mcpConfigSnapshot, *yaml.Node, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return mcpConfigSnapshot{}, nil, nil
	}
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || isReparsePoint(info) || info.Size() > mcpConfigFileLimit {
		return mcpConfigSnapshot{}, nil, errMCPProfileUnsafe
	}
	file, err := os.Open(path)
	if err != nil {
		return mcpConfigSnapshot{}, nil, errMCPProfileUnsafe
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, mcpConfigFileLimit+1))
	if err != nil || len(data) > mcpConfigFileLimit {
		return mcpConfigSnapshot{}, nil, errMCPProfileUnsafe
	}
	root, err := readBoundedYAML(path)
	if err != nil {
		return mcpConfigSnapshot{}, nil, errMCPProfileUnsafe
	}
	return mcpConfigSnapshot{exists: true, data: data, mode: info.Mode().Perm()}, root, nil
}

func commitMCPConfig(path string, expected mcpConfigSnapshot, payload []byte) error {
	current, _, err := observeMCPConfig(path)
	if err != nil || current.exists != expected.exists || current.mode != expected.mode || !bytes.Equal(current.data, expected.data) {
		return errMCPProfileConflict
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".yorva-mcp-config-*")
	if err != nil {
		return errMCPProfileUnsafe
	}
	tempPath := temp.Name()
	defer func() { _ = temp.Close(); _ = os.Remove(tempPath) }()
	if temp.Chmod(0o600) != nil || writeAndSync(temp, payload) != nil || temp.Close() != nil {
		return errMCPProfileUnsafe
	}
	current, _, err = observeMCPConfig(path)
	if err != nil || current.exists != expected.exists || current.mode != expected.mode || !bytes.Equal(current.data, expected.data) {
		return errMCPProfileConflict
	}
	if err := atomicReplaceCredentialFile(tempPath, path); err != nil {
		return errMCPProfileUnsafe
	}
	observed, _, err := observeMCPConfig(path)
	if err != nil || !observed.exists || !bytes.Equal(observed.data, payload) {
		return errMCPProfileUnsafe
	}
	return nil
}

func ensureYAMLMapping(root *yaml.Node, key string) *yaml.Node {
	if existing := mappingValue(root, key); existing != nil {
		return existing
	}
	value := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	appendYAMLMapping(root, key, value)
	return value
}

func yamlMappingIndex(mapping *yaml.Node, key string) int {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return -1
	}
	for index := 0; index < len(mapping.Content); index += 2 {
		if mapping.Content[index].Value == key {
			return index
		}
	}
	return -1
}

func appendYAMLMapping(mapping *yaml.Node, key string, value *yaml.Node) {
	mapping.Content = append(mapping.Content, yamlScalar(key), value)
}

func removeYAMLMapping(mapping *yaml.Node, key string) {
	if index := yamlMappingIndex(mapping, key); index >= 0 {
		mapping.Content = append(mapping.Content[:index], mapping.Content[index+2:]...)
	}
}

func yamlScalar(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
}

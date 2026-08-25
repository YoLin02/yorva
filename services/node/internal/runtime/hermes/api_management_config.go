package hermes

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

const (
	apiManagementVersion       = "0.20.5"
	apiManagementDefaultHost   = "127.0.0.1"
	apiManagementDefaultPort   = 8642
	apiManagementMaxConfigFile = 256 * 1024
	apiManagementMaxKeyBytes   = 4096
	apiManagementMaxYAMLNodes  = 4096
)

var (
	errAPIManagementUnavailable = errors.New("Hermes API management read is unavailable")
	errAPIManagementUnsafe      = errors.New("Hermes API management configuration is unsafe")
)

type apiManagementTarget struct {
	host       string
	port       int
	pathPrefix string
	key        []byte
}

func (t *apiManagementTarget) clear() {
	clearCredentialBytes(t.key)
	t.key = nil
}

type apiListenerConfig struct {
	multiplex bool
	allowlist []string
	enabled   *bool
	host      string
	port      int
	keyInYAML bool
}

type apiProfileFiles struct {
	env    credentialSnapshot
	config *yaml.Node
}

func resolveAPIManagementTarget(version, nativeID string) (apiManagementTarget, error) {
	return resolveAPIManagementTargetAt(officialHermesHome(), version, nativeID)
}

func resolveAPIManagementTargetAt(root, version, nativeID string) (apiManagementTarget, error) {
	if version != apiManagementVersion {
		return apiManagementTarget{}, errAPIManagementUnavailable
	}
	normalized, err := officialNormalizeProfileName(nativeID)
	if err != nil || normalized != nativeID || officialValidateProfileName(nativeID) != nil {
		return apiManagementTarget{}, errAPIManagementUnsafe
	}

	defaultFiles, err := readAPIProfileFiles(root, "default")
	if err != nil {
		return apiManagementTarget{}, err
	}
	defaultListener, err := projectAPIListener(defaultFiles.config)
	if err != nil {
		return apiManagementTarget{}, err
	}
	if err := applyAPIEnvEndpoint(&defaultListener, defaultFiles.env.data); err != nil {
		return apiManagementTarget{}, err
	}

	if nativeID != "default" && defaultListener.multiplex {
		if !profileAllowed(defaultListener.allowlist, nativeID) || defaultListener.enabled != nil && !*defaultListener.enabled {
			return apiManagementTarget{}, errAPIManagementUnavailable
		}
		profileFiles, err := readAPIProfileFiles(root, nativeID)
		if err != nil {
			return apiManagementTarget{}, err
		}
		profileListener, err := projectAPIListener(profileFiles.config)
		if err != nil {
			return apiManagementTarget{}, err
		}
		// A secondary Profile with API server enabled is skipped by Hermes 0.20.5
		// multiplex startup because it would bind a second port.
		if profileListener.keyInYAML || profileListener.enabled == nil || *profileListener.enabled {
			return apiManagementTarget{}, errAPIManagementUnavailable
		}
		defaultKey, err := readAPIKey(defaultFiles.env.data)
		if err != nil {
			return apiManagementTarget{}, err
		}
		clearCredentialBytes(defaultKey)
		key, err := readAPIKey(profileFiles.env.data)
		if err != nil {
			return apiManagementTarget{}, err
		}
		return newAPIManagementTarget(defaultListener, "/p/"+nativeID, key)
	}

	files := defaultFiles
	listener := defaultListener
	if nativeID != "default" {
		files, err = readAPIProfileFiles(root, nativeID)
		if err != nil {
			return apiManagementTarget{}, err
		}
		listener, err = projectAPIListener(files.config)
		if err != nil {
			return apiManagementTarget{}, err
		}
		if err := applyAPIEnvEndpoint(&listener, files.env.data); err != nil {
			return apiManagementTarget{}, err
		}
	}
	if nativeID != "default" && listener.multiplex || listener.enabled != nil && !*listener.enabled {
		return apiManagementTarget{}, errAPIManagementUnavailable
	}
	key, err := readAPIKey(files.env.data)
	if err != nil {
		return apiManagementTarget{}, err
	}
	return newAPIManagementTarget(listener, "", key)
}

func newAPIManagementTarget(listener apiListenerConfig, prefix string, key []byte) (apiManagementTarget, error) {
	if listener.keyInYAML {
		clearCredentialBytes(key)
		return apiManagementTarget{}, errAPIManagementUnsafe
	}
	host := listener.host
	if host == "" {
		host = apiManagementDefaultHost
	}
	if host != "127.0.0.1" && host != "::1" {
		clearCredentialBytes(key)
		return apiManagementTarget{}, errAPIManagementUnsafe
	}
	port := listener.port
	if port == 0 {
		port = apiManagementDefaultPort
	}
	if port < 1 || port > 65535 || prefix != "" && !strings.HasPrefix(prefix, "/p/") {
		clearCredentialBytes(key)
		return apiManagementTarget{}, errAPIManagementUnsafe
	}
	return apiManagementTarget{host: host, port: port, pathPrefix: prefix, key: key}, nil
}

func readAPIProfileFiles(root, nativeID string) (apiProfileFiles, error) {
	store := credentialStore{root: root}
	envPath, err := store.credentialPath(nativeID)
	if err != nil {
		return apiProfileFiles{}, errAPIManagementUnsafe
	}
	env, err := observeCredentialFile(envPath)
	if err != nil {
		return apiProfileFiles{}, errAPIManagementUnsafe
	}
	configPath := filepath.Join(filepath.Dir(envPath), "config.yaml")
	config, err := readBoundedYAML(configPath)
	if err != nil {
		return apiProfileFiles{}, err
	}
	return apiProfileFiles{env: env, config: config}, nil
}

func readBoundedYAML(path string) (*yaml.Node, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || isReparsePoint(info) || info.Size() > apiManagementMaxConfigFile {
		return nil, errAPIManagementUnsafe
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, errAPIManagementUnavailable
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, apiManagementMaxConfigFile+1))
	if err != nil || len(data) > apiManagementMaxConfigFile || !utf8.Valid(data) {
		return nil, errAPIManagementUnsafe
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		return nil, errAPIManagementUnsafe
	}
	var trailing yaml.Node
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, errAPIManagementUnsafe
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return nil, errAPIManagementUnsafe
	}
	count := 0
	if err := validateYAMLNode(document.Content[0], &count); err != nil {
		return nil, err
	}
	return document.Content[0], nil
}

func validateYAMLNode(node *yaml.Node, count *int) error {
	*count++
	if *count > apiManagementMaxYAMLNodes || node.Kind == yaml.AliasNode || node.Anchor != "" || node.Alias != nil {
		return errAPIManagementUnsafe
	}
	switch node.Kind {
	case yaml.MappingNode:
		if len(node.Content)%2 != 0 {
			return errAPIManagementUnsafe
		}
		seen := make(map[string]struct{}, len(node.Content)/2)
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i]
			if key.Kind != yaml.ScalarNode || key.Tag != "!!str" || key.Value == "" {
				return errAPIManagementUnsafe
			}
			if _, duplicate := seen[key.Value]; duplicate {
				return errAPIManagementUnsafe
			}
			seen[key.Value] = struct{}{}
			if err := validateYAMLNode(node.Content[i+1], count); err != nil {
				return err
			}
		}
	case yaml.SequenceNode:
		for _, child := range node.Content {
			if err := validateYAMLNode(child, count); err != nil {
				return err
			}
		}
	case yaml.ScalarNode:
		switch node.Tag {
		case "!!str", "!!int", "!!bool", "!!null", "!!float", "!!timestamp":
		default:
			return errAPIManagementUnsafe
		}
	default:
		return errAPIManagementUnsafe
	}
	return nil
}

func projectAPIListener(root *yaml.Node) (apiListenerConfig, error) {
	result := apiListenerConfig{}
	if root == nil {
		return result, nil
	}
	gateway := yamlMapValue(root, "gateway")
	rootMultiplex := yamlMapValue(root, "multiplex_profiles")
	gatewayMultiplex := yamlMapValue(gateway, "multiplex_profiles")
	if rootMultiplex != nil && gatewayMultiplex != nil {
		return result, errAPIManagementUnsafe
	}
	if raw := rootMultiplex; raw != nil {
		value, err := yamlBool(raw)
		if err != nil {
			return result, err
		}
		result.multiplex = value
	} else if raw := gatewayMultiplex; raw != nil {
		value, err := yamlBool(raw)
		if err != nil {
			return result, err
		}
		result.multiplex = value
	}
	rootAllowlist := yamlMapValue(root, "multiplex_profile_allowlist")
	gatewayAllowlist := yamlMapValue(gateway, "multiplex_profile_allowlist")
	if rootAllowlist != nil && gatewayAllowlist != nil {
		return result, errAPIManagementUnsafe
	}
	allowlist := rootAllowlist
	if allowlist == nil {
		allowlist = gatewayAllowlist
	}
	if allowlist != nil {
		values, err := yamlProfileList(allowlist)
		if err != nil {
			return result, err
		}
		result.allowlist = values
	}

	blocks := []*yaml.Node{
		yamlMapValue(yamlMapValue(gateway, "platforms"), "api_server"),
		yamlMapValue(yamlMapValue(root, "platforms"), "api_server"),
		yamlMapValue(gateway, "api_server"),
	}
	nonNilBlocks := 0
	for _, block := range blocks {
		if block != nil {
			nonNilBlocks++
		}
	}
	if nonNilBlocks > 1 {
		return result, errAPIManagementUnsafe
	}
	for _, block := range blocks {
		if block == nil {
			continue
		}
		if block.Kind != yaml.MappingNode {
			return result, errAPIManagementUnsafe
		}
		if raw := yamlMapValue(block, "enabled"); raw != nil {
			value, err := yamlBool(raw)
			if err != nil {
				return result, err
			}
			result.enabled = &value
		}
		extra := yamlMapValue(block, "extra")
		if extra != nil && extra.Kind != yaml.MappingNode {
			return result, errAPIManagementUnsafe
		}
		for _, name := range []string{"key", "host", "port"} {
			extraValue := yamlMapValue(extra, name)
			blockValue := yamlMapValue(block, name)
			if extraValue != nil && blockValue != nil {
				return result, errAPIManagementUnsafe
			}
			raw := extraValue
			if raw == nil {
				raw = blockValue
			}
			if raw == nil {
				continue
			}
			switch name {
			case "key":
				result.keyInYAML = true
			case "host":
				value, err := yamlString(raw, 64)
				if err != nil {
					return result, err
				}
				result.host = value
			case "port":
				value, err := yamlPort(raw)
				if err != nil {
					return result, err
				}
				result.port = value
			}
		}
	}
	return result, nil
}

func yamlMapValue(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}

func yamlBool(node *yaml.Node) (bool, error) {
	if node == nil || node.Kind != yaml.ScalarNode || node.Tag != "!!bool" {
		return false, errAPIManagementUnsafe
	}
	value, err := strconv.ParseBool(node.Value)
	if err != nil {
		return false, errAPIManagementUnsafe
	}
	return value, nil
}

func yamlString(node *yaml.Node, max int) (string, error) {
	if node == nil || node.Kind != yaml.ScalarNode || node.Tag != "!!str" || len(node.Value) == 0 || len(node.Value) > max || strings.Contains(node.Value, "${") {
		return "", errAPIManagementUnsafe
	}
	return node.Value, nil
}

func yamlPort(node *yaml.Node) (int, error) {
	if node == nil || node.Kind != yaml.ScalarNode || node.Tag != "!!int" && node.Tag != "!!str" || strings.Contains(node.Value, "${") {
		return 0, errAPIManagementUnsafe
	}
	value, err := strconv.Atoi(node.Value)
	if err != nil || value < 1 || value > 65535 {
		return 0, errAPIManagementUnsafe
	}
	return value, nil
}

func yamlProfileList(node *yaml.Node) ([]string, error) {
	if node.Kind == yaml.ScalarNode && node.Tag == "!!null" {
		return nil, nil
	}
	if node.Kind != yaml.SequenceNode || len(node.Content) > 256 {
		return nil, errAPIManagementUnsafe
	}
	result := make([]string, 0, len(node.Content))
	seen := make(map[string]struct{}, len(node.Content))
	for _, child := range node.Content {
		value, err := yamlString(child, 64)
		if err != nil {
			return nil, err
		}
		normalized, err := officialNormalizeProfileName(value)
		if err != nil || normalized != value || officialValidateProfileName(value) != nil {
			return nil, errAPIManagementUnsafe
		}
		if _, duplicate := seen[value]; duplicate {
			return nil, errAPIManagementUnsafe
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result, nil
}

func profileAllowed(allowlist []string, nativeID string) bool {
	if allowlist == nil {
		return true
	}
	for _, item := range allowlist {
		if item == nativeID {
			return true
		}
	}
	return false
}

func readAPIKey(data []byte) ([]byte, error) {
	value, found, err := fixedEnvValue(data, "API_SERVER_KEY", apiManagementMaxKeyBytes)
	if err != nil || !found || len(value) < 16 {
		clearCredentialBytes(value)
		return nil, errAPIManagementUnavailable
	}
	for remaining := value; len(remaining) > 0; {
		character, size := utf8.DecodeRune(remaining)
		if unicode.IsControl(character) {
			clearCredentialBytes(value)
			return nil, errAPIManagementUnsafe
		}
		remaining = remaining[size:]
	}
	return value, nil
}

func fixedEnvValue(data []byte, name string, maximum int) ([]byte, bool, error) {
	var value []byte
	found := false
	for _, line := range splitCredentialLines(data) {
		raw, match := credentialAssignment(line.body, name)
		if !match {
			continue
		}
		if found {
			clearCredentialBytes(value)
			return nil, false, errAPIManagementUnsafe
		}
		parsed, err := parseFixedEnvAssignment(raw, maximum)
		if err != nil {
			return nil, false, err
		}
		value = parsed
		found = true
	}
	return value, found, nil
}

func parseFixedEnvAssignment(raw []byte, maximum int) ([]byte, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil, nil
	}
	var value []byte
	if raw[0] == '\'' || raw[0] == '"' {
		quote := raw[0]
		closing := -1
		escaped := false
		for i := 1; i < len(raw); i++ {
			if quote == '"' && raw[i] == '\\' && !escaped {
				escaped = true
				continue
			}
			if raw[i] == quote && !escaped {
				closing = i
				break
			}
			escaped = false
		}
		if closing < 0 {
			return nil, errAPIManagementUnsafe
		}
		tail := bytes.TrimSpace(raw[closing+1:])
		if len(tail) > 0 && tail[0] != '#' {
			return nil, errAPIManagementUnsafe
		}
		value = append([]byte(nil), raw[1:closing]...)
		if quote == '"' && bytes.Contains(value, []byte{'\\'}) {
			// The approved writer only emits escaped slash and quote. Rejecting
			// quoted escape syntax avoids diverging from python-dotenv semantics.
			return nil, errAPIManagementUnsafe
		}
	} else {
		end := len(raw)
		for i := range raw {
			if raw[i] == '#' && (i == 0 || raw[i-1] == ' ' || raw[i-1] == '\t') {
				end = i
				break
			}
		}
		value = append([]byte(nil), bytes.TrimSpace(raw[:end])...)
	}
	if len(value) > maximum || !utf8.Valid(value) || bytes.Contains(value, []byte("${")) || bytes.IndexByte(value, 0) >= 0 {
		clearCredentialBytes(value)
		return nil, errAPIManagementUnsafe
	}
	return value, nil
}

func applyAPIEnvEndpoint(listener *apiListenerConfig, data []byte) error {
	if raw, found, err := fixedEnvValue(data, "API_SERVER_HOST", 64); err != nil {
		return err
	} else if found && len(raw) > 0 {
		listener.host = string(raw)
		clearCredentialBytes(raw)
	}
	if raw, found, err := fixedEnvValue(data, "API_SERVER_PORT", 8); err != nil {
		return err
	} else if found && len(raw) > 0 {
		value, parseErr := strconv.Atoi(string(raw))
		clearCredentialBytes(raw)
		if parseErr != nil || value < 1 || value > 65535 {
			return errAPIManagementUnsafe
		}
		listener.port = value
	}
	return nil
}

func (t apiManagementTarget) address() string {
	return net.JoinHostPort(t.host, strconv.Itoa(t.port))
}

func (t apiManagementTarget) path(suffix string) (string, error) {
	if suffix != "/health/detailed" && suffix != "/v1/skills" {
		return "", fmt.Errorf("%w", errAPIManagementUnsafe)
	}
	return t.pathPrefix + suffix, nil
}

package openclaw

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const markerName = ".yorva-instance.json"

var nativeNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)
var createNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)

type ownership struct {
	Schema  int    `json:"schema"`
	Profile string `json:"profile"`
	Port    int    `json:"port"`
}

func (a *Adapter) ValidateName(name string) error {
	if name == "default" || !createNamePattern.MatchString(name) {
		return yorvaruntime.ErrInstanceNameInvalid
	}
	return nil
}

func (a *Adapter) profileRoot(profile string) (string, error) {
	if !nativeNamePattern.MatchString(profile) {
		return "", errUnsafeTarget
	}
	if profile == "default" {
		return filepath.Join(a.home, ".openclaw"), nil
	}
	if strings.EqualFold(profile, "default") {
		return "", errUnsafeTarget
	}
	return filepath.Join(a.home, ".openclaw-"+profile), nil
}

func (a *Adapter) profileNames() ([]string, error) {
	entries, err := os.ReadDir(a.home)
	if err != nil {
		return nil, yorvaruntime.ErrInstanceInventoryFailed
	}
	profiles := []string{}
	for _, entry := range entries {
		name := entry.Name()
		profile := ""
		if name == ".openclaw" {
			profile = "default"
		} else if strings.HasPrefix(name, ".openclaw-") {
			profile = strings.TrimPrefix(name, ".openclaw-")
			if strings.EqualFold(profile, "default") || !nativeNamePattern.MatchString(profile) {
				continue
			}
		}
		if profile == "" {
			continue
		}
		root, _ := a.profileRoot(profile)
		if !safeDirectory(root) {
			return nil, yorvaruntime.ErrInstanceInventoryFailed
		}
		profiles = append(profiles, profile)
		if len(profiles) > 128 {
			return nil, yorvaruntime.ErrInstanceOutputUnrecognized
		}
	}
	return profiles, nil
}

func (a *Adapter) List(ctx context.Context, executable string) ([]yorvaruntime.NativeInstance, error) {
	node, err := a.resolveNode(executable)
	if err != nil {
		return nil, yorvaruntime.ErrInstanceInventoryFailed
	}
	profiles, err := a.profileNames()
	if err != nil {
		return nil, err
	}
	items := make([]yorvaruntime.NativeInstance, 0, len(profiles))
	for _, profile := range profiles {
		if err := a.validateProfile(ctx, node, executable, profile); err != nil {
			return nil, instanceError(err)
		}
		_, _, ownedErr := a.ownedProfile(profile)
		items = append(items, yorvaruntime.NativeInstance{NativeID: profile, Default: profile == "default", Protected: ownedErr != nil})
	}
	return items, nil
}

func (a *Adapter) validateProfile(ctx context.Context, node, entry, profile string) error {
	root, err := a.profileRoot(profile)
	if err != nil || !safeDirectory(root) || !regularFile(filepath.Join(root, "openclaw.json")) {
		return errUnsafeTarget
	}
	output, err := a.command(ctx, node, entry, profile, 15*time.Second, "config", "validate", "--json")
	defer clear(output)
	if err != nil {
		return err
	}
	var result struct {
		Valid *bool  `json:"valid"`
		Path  string `json:"path"`
	}
	if json.Unmarshal(output, &result) != nil || result.Valid == nil || !*result.Valid || !samePath(result.Path, filepath.Join(root, "openclaw.json")) {
		return yorvaruntime.ErrInstanceOutputUnrecognized
	}
	return nil
}

func (a *Adapter) ownedProfile(profile string) (string, ownership, error) {
	root, err := a.profileRoot(profile)
	if err != nil || profile == "default" || !safeDirectory(root) {
		return "", ownership{}, errUnsafeTarget
	}
	data, err := readBoundedFile(filepath.Join(root, markerName), 4096)
	if err != nil {
		return "", ownership{}, errUnsafeTarget
	}
	var marker ownership
	if json.Unmarshal(data, &marker) != nil || marker.Schema != 1 || marker.Profile != profile || marker.Port < 1024 || marker.Port > 65400 {
		return "", ownership{}, errUnsafeTarget
	}
	return root, marker, nil
}

func (a *Adapter) Create(ctx context.Context, executable, name string) error {
	if err := a.ValidateName(name); err != nil {
		return err
	}
	a.createMu.Lock()
	defer a.createMu.Unlock()
	node, err := a.qualifiedNode(executable)
	if err != nil {
		return instanceError(err)
	}
	port, err := a.availablePort(ctx, node, executable)
	if err != nil {
		return instanceError(err)
	}
	root, _ := a.profileRoot(name)
	// An existing directory is always protected, even if it is empty or broken.
	if err := os.Mkdir(root, 0700); err != nil {
		return yorvaruntime.ErrInstanceInventoryFailed
	}
	workspace := filepath.Join(root, "workspace")
	output, err := a.command(ctx, node, executable, name, 60*time.Second,
		"setup", "--non-interactive", "--accept-risk", "--auth-choice", "skip", "--mode", "local",
		"--gateway-port", strconv.Itoa(port), "--gateway-bind", "loopback", "--gateway-auth", "token",
		"--no-install-daemon", "--skip-channels", "--skip-skills", "--skip-search", "--skip-bootstrap", "--skip-ui", "--skip-health",
		"--suppress-gateway-token-output", "--workspace", workspace, "--json")
	defer clear(output)
	if err != nil {
		_ = os.Remove(root)
		return instanceError(err)
	}
	var result struct {
		OK            *bool  `json:"ok"`
		Mode          string `json:"mode"`
		Workspace     string `json:"workspace"`
		InstallDaemon *bool  `json:"installDaemon"`
		Gateway       struct {
			Port     int    `json:"port"`
			Bind     string `json:"bind"`
			AuthMode string `json:"authMode"`
		} `json:"gateway"`
	}
	if json.Unmarshal(output, &result) != nil || result.OK == nil || !*result.OK || result.Mode != "local" || result.InstallDaemon == nil || *result.InstallDaemon || !samePath(result.Workspace, workspace) || result.Gateway.Port != port || result.Gateway.Bind != "loopback" || result.Gateway.AuthMode != "token" {
		return yorvaruntime.ErrInstanceOutputUnrecognized
	}
	if err := a.validateProfile(ctx, node, executable, name); err != nil {
		return instanceError(err)
	}
	data, _ := json.Marshal(ownership{Schema: 1, Profile: name, Port: port})
	file, err := os.OpenFile(filepath.Join(root, markerName), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return yorvaruntime.ErrInstanceInventoryFailed
	}
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		return yorvaruntime.ErrInstanceInventoryFailed
	}
	return nil
}

func (a *Adapter) Delete(ctx context.Context, executable, profile string) error {
	root, _, err := a.ownedProfile(profile)
	if err != nil {
		return yorvaruntime.ErrInstanceInventoryFailed
	}
	node, err := a.qualifiedNode(executable)
	if err != nil {
		return instanceError(err)
	}
	if err := a.validateOwnedConfiguration(ctx, node, executable, profile); err != nil {
		return instanceError(err)
	}
	status, err := a.status(ctx, node, executable, profile)
	if err != nil || status.State != yorvaruntime.LifecycleStopped {
		return yorvaruntime.ErrInstanceInventoryFailed
	}
	// Both scopes are needed: upstream --state intentionally preserves nested
	// workspaces. Validation above restricts the sole workspace to this owned
	// profile. No service/global uninstall or arbitrary filesystem API exists.
	output, err := a.command(ctx, node, executable, profile, 45*time.Second, "uninstall", "--state", "--workspace", "--yes", "--non-interactive")
	clear(output)
	if err != nil {
		return instanceError(err)
	}
	if _, err := os.Lstat(root); !errors.Is(err, os.ErrNotExist) {
		return yorvaruntime.ErrInstanceInventoryFailed
	}
	return nil
}

func (a *Adapter) availablePort(ctx context.Context, node, entry string) (int, error) {
	profiles, err := a.profileNames()
	if err != nil {
		return 0, err
	}
	used := []int{}
	for _, profile := range profiles {
		if err := a.validateProfile(ctx, node, entry, profile); err != nil {
			return 0, err
		}
		output, err := a.command(ctx, node, entry, profile, 15*time.Second, "config", "get", "gateway.port", "--json")
		if err != nil {
			clear(output)
			return 0, err
		}
		var port int
		parseErr := json.Unmarshal(output, &port)
		clear(output)
		if parseErr != nil || port < 1 || port > 65535 {
			return 0, errUnsafeTarget
		}
		used = append(used, port)
	}
	for port := 29120; port < 62000; port += 300 {
		conflict := false
		for _, other := range used {
			if port <= other+120 && other <= port+120 {
				conflict = true
				break
			}
		}
		if !conflict && portsFree(port) {
			return port, nil
		}
	}
	return 0, errUnsafeTarget
}

func portsFree(base int) bool {
	listeners := []net.Listener{}
	defer func() {
		for _, listener := range listeners {
			_ = listener.Close()
		}
	}()
	for port := base; port <= base+120; port++ {
		listener, err := net.Listen("tcp4", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
		if err != nil {
			return false
		}
		listeners = append(listeners, listener)
	}
	return true
}

func (a *Adapter) qualifiedNode(entry string) (string, error) {
	node, err := a.resolveNode(entry)
	if err != nil {
		return "", err
	}
	data, err := readBoundedFile(filepath.Join(filepath.Dir(entry), "package.json"), packageMetadataLimit)
	if err != nil {
		return "", err
	}
	var pkg struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(data, &pkg) != nil || pkg.Version != supportedVersion {
		return "", errUnsafeTarget
	}
	return node, nil
}

func safeDirectory(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0
}

func samePath(left, right string) bool {
	return filepath.IsAbs(left) && strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
}

func instanceError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, context.Canceled):
		return context.Canceled
	case errors.Is(err, context.DeadlineExceeded):
		return yorvaruntime.ErrInstanceOperationTimedOut
	case errors.Is(err, yorvaruntime.ErrInstanceOutputUnrecognized):
		return yorvaruntime.ErrInstanceOutputUnrecognized
	default:
		return yorvaruntime.ErrInstanceInventoryFailed
	}
}

package openclaw

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type gatewayConfiguration struct {
	Mode string `json:"mode"`
	Bind string `json:"bind"`
	Port int    `json:"port"`
	Auth struct {
		Mode  string `json:"mode"`
		Token string `json:"token"`
	} `json:"auth"`
	Tailscale struct {
		Mode string `json:"mode"`
	} `json:"tailscale"`
	TLS struct {
		Enabled bool `json:"enabled"`
	} `json:"tls"`
}

func (a *Adapter) gatewayConfiguration(ctx context.Context, node, entry, profile string) (gatewayConfiguration, error) {
	root, err := a.profileRoot(profile)
	if err != nil || !safeDirectory(root) || !regularFile(filepath.Join(root, "openclaw.json")) {
		return gatewayConfiguration{}, errUnsafeTarget
	}
	output, err := a.command(ctx, node, entry, profile, 15*time.Second, "config", "get", "gateway", "--json")
	defer clear(output)
	var config gatewayConfiguration
	if err != nil {
		return config, err
	}
	if json.Unmarshal(output, &config) != nil {
		return config, yorvaruntime.ErrLifecycleOutputUnrecognized
	}
	if config.Mode != "local" || config.Bind != "loopback" || config.Port < 1024 || config.Port > 65535 || config.Auth.Mode != "token" || config.Auth.Token == "" || config.TLS.Enabled || (config.Tailscale.Mode != "" && config.Tailscale.Mode != "off") {
		return gatewayConfiguration{}, errUnsafeTarget
	}
	// The native CLI resolves the Gateway secret for probes and mutations. Only
	// its presence is relevant here; it never reaches a YORVA DTO or persistence.
	config.Auth.Token = ""
	return config, nil
}

func (a *Adapter) validateOwnedConfiguration(ctx context.Context, node, entry, profile string) error {
	root, marker, err := a.ownedProfile(profile)
	if err != nil {
		return err
	}
	config, err := a.gatewayConfiguration(ctx, node, entry, profile)
	if err != nil {
		return err
	}
	if config.Port != marker.Port {
		return errUnsafeTarget
	}
	output, err := a.command(ctx, node, entry, profile, 15*time.Second, "config", "get", "agents", "--json")
	defer clear(output)
	if err != nil {
		return err
	}
	var agents struct {
		Defaults struct {
			Workspace string `json:"workspace"`
		} `json:"defaults"`
		List []json.RawMessage `json:"list"`
	}
	// Additional agents can introduce independently located workspaces. P9
	// never claims ownership of those configurations or forwards their removal.
	if json.Unmarshal(output, &agents) != nil || len(agents.List) != 0 || !samePath(agents.Defaults.Workspace, filepath.Join(root, "workspace")) || !safeDirectory(agents.Defaults.Workspace) {
		return errUnsafeTarget
	}
	return nil
}

type gatewayStatus struct {
	Service struct {
		Loaded     *bool  `json:"loaded"`
		TargetRole string `json:"targetRole"`
		Runtime    struct {
			Status string `json:"status"`
		} `json:"runtime"`
	} `json:"service"`
	Config struct {
		CLI struct {
			Path   string `json:"path"`
			Exists *bool  `json:"exists"`
			Valid  *bool  `json:"valid"`
		} `json:"cli"`
		Mismatch bool `json:"mismatch"`
	} `json:"config"`
	Gateway struct {
		BindMode string  `json:"bindMode"`
		BindHost string  `json:"bindHost"`
		Port     int     `json:"port"`
		ProbeURL string  `json:"probeUrl"`
		Version  *string `json:"version"`
	} `json:"gateway"`
	Port struct {
		Port   int    `json:"port"`
		Status string `json:"status"`
	} `json:"port"`
	RPC struct {
		OK   *bool  `json:"ok"`
		Kind string `json:"kind"`
		URL  string `json:"url"`
	} `json:"rpc"`
}

func (a *Adapter) Status(ctx context.Context, installation yorvaruntime.LifecycleInstallation, profile string) (yorvaruntime.LifecycleStatus, error) {
	node, err := a.qualifiedNode(installation.Executable)
	if err != nil || installation.Version != supportedVersion {
		return unknownStatus(), yorvaruntime.ErrLifecycleQueryFailed
	}
	return a.status(ctx, node, installation.Executable, profile)
}

func (a *Adapter) status(ctx context.Context, node, entry, profile string) (yorvaruntime.LifecycleStatus, error) {
	config, err := a.gatewayConfiguration(ctx, node, entry, profile)
	if err != nil {
		return unknownStatus(), lifecycleError(err)
	}
	output, commandErr := a.command(ctx, node, entry, profile, 35*time.Second, "gateway", "status", "--json", "--require-rpc")
	defer clear(output)
	if commandErr != nil && !errors.Is(commandErr, errCommand) {
		return unknownStatus(), lifecycleError(commandErr)
	}
	root, _ := a.profileRoot(profile)
	return parseStatus(output, commandErr, filepath.Join(root, "openclaw.json"), config.Port)
}

func parseStatus(output []byte, commandErr error, configPath string, port int) (yorvaruntime.LifecycleStatus, error) {
	var response gatewayStatus
	if json.Unmarshal(output, &response) != nil {
		return unknownStatus(), yorvaruntime.ErrLifecycleOutputUnrecognized
	}
	cli := response.Config.CLI
	url := fmt.Sprintf("ws://127.0.0.1:%d", port)
	if cli.Exists == nil || !*cli.Exists || cli.Valid == nil || !*cli.Valid || !samePath(cli.Path, configPath) || response.Config.Mismatch || response.Gateway.Port != port || response.Port.Port != port || response.Gateway.BindMode != "loopback" || response.Gateway.BindHost != "127.0.0.1" || response.Gateway.ProbeURL != url || response.RPC.URL != url || response.RPC.Kind != "read" || response.RPC.OK == nil || response.Service.Loaded == nil {
		return unknownStatus(), yorvaruntime.ErrLifecycleOutputUnrecognized
	}
	// Manual lifecycle is qualified only without an installed native service.
	// A later external service installation must be inspected by its owner.
	if *response.Service.Loaded || response.Service.TargetRole != "target" {
		return unknownStatus(), yorvaruntime.ErrLifecycleQueryFailed
	}
	if commandErr == nil && *response.RPC.OK && response.Port.Status == "busy" && response.Gateway.Version != nil && *response.Gateway.Version == supportedVersion {
		return yorvaruntime.LifecycleStatus{State: yorvaruntime.LifecycleRunning}, nil
	}
	if !*response.RPC.OK && response.Port.Status == "free" && response.Service.Runtime.Status == "stopped" {
		return yorvaruntime.LifecycleStatus{State: yorvaruntime.LifecycleStopped}, nil
	}
	return unknownStatus(), yorvaruntime.ErrLifecycleQueryFailed
}

func unknownStatus() yorvaruntime.LifecycleStatus {
	return yorvaruntime.LifecycleStatus{State: yorvaruntime.LifecycleUnknown}
}

func (a *Adapter) ResolveInstanceManagement(_ context.Context, _ yorvaruntime.Installation, profile string) (yorvaruntime.InstanceManagementFeatures, error) {
	_, _, err := a.ownedProfile(profile)
	return yorvaruntime.InstanceManagementFeatures{Health: a, DisableLifecycle: err != nil}, nil
}

func lifecycleError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, yorvaruntime.ErrLifecycleOutputUnrecognized) {
		return err
	}
	return yorvaruntime.ErrLifecycleQueryFailed
}

func (a *Adapter) InspectInstanceHealth(ctx context.Context, installation yorvaruntime.Installation, profile string) (yorvaruntime.HealthObservation, error) {
	status, err := a.Status(ctx, yorvaruntime.LifecycleInstallation{Executable: installation.Path, Version: installation.Version}, profile)
	state, code := yorvaruntime.HealthUnknown, "gateway_auth_unknown"
	if err == nil {
		if status.State == yorvaruntime.LifecycleRunning {
			state, code = yorvaruntime.HealthHealthy, "gateway_authenticated"
		} else if status.State == yorvaruntime.LifecycleStopped {
			state, code = yorvaruntime.HealthUnhealthy, "gateway_stopped"
		}
	}
	return yorvaruntime.HealthObservation{State: state, Findings: []yorvaruntime.HealthFinding{{Code: code, State: state}}, Partial: err != nil, ObservedAt: time.Now().UTC()}, nil
}

func (a *Adapter) InspectRuntimeHealth(ctx context.Context, installation yorvaruntime.Installation) (yorvaruntime.HealthObservation, error) {
	items, err := a.List(ctx, installation.Path)
	result := yorvaruntime.HealthObservation{State: yorvaruntime.HealthUnknown, Findings: []yorvaruntime.HealthFinding{}, Partial: err != nil, ObservedAt: time.Now().UTC()}
	if err != nil || len(items) == 0 {
		return result, nil
	}
	result.State = yorvaruntime.HealthHealthy
	for _, item := range items {
		observed, _ := a.InspectInstanceHealth(ctx, installation, item.NativeID)
		if observed.State != yorvaruntime.HealthHealthy {
			result.State = yorvaruntime.HealthDegraded
		}
		result.Partial = result.Partial || observed.Partial
	}
	return result, nil
}

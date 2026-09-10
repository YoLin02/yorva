package openclaw

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func statusFixture(path string, port int, running bool) gatewayStatus {
	var value gatewayStatus
	yes, no := true, false
	value.Service.Loaded = &no
	value.Service.TargetRole = "target"
	value.Service.Runtime.Status = "stopped"
	value.Config.CLI.Path, value.Config.CLI.Exists, value.Config.CLI.Valid = path, &yes, &yes
	value.Gateway.BindMode, value.Gateway.BindHost, value.Gateway.Port = "loopback", "127.0.0.1", port
	value.Gateway.ProbeURL = fmt.Sprintf("ws://127.0.0.1:%d", port)
	value.RPC.URL, value.RPC.Kind, value.RPC.OK = value.Gateway.ProbeURL, "read", &no
	value.Port.Port, value.Port.Status = port, "free"
	if running {
		version := supportedVersion
		value.Gateway.Version, value.RPC.OK, value.Port.Status = &version, &yes, "busy"
	}
	return value
}

func TestStatusRequiresAuthenticatedExactTarget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "openclaw.json")
	for _, tc := range []struct {
		name   string
		mutate func(*gatewayStatus)
	}{
		{"auth failed", func(v *gatewayStatus) { no := false; v.RPC.OK = &no }},
		{"port only", func(v *gatewayStatus) { v.RPC.OK = nil }},
		{"wrong profile", func(v *gatewayStatus) { v.Config.CLI.Path += ".other" }},
		{"wrong probe", func(v *gatewayStatus) { v.RPC.URL = "ws://127.0.0.1:29420" }},
		{"remote", func(v *gatewayStatus) { v.Gateway.BindHost = "0.0.0.0" }},
		{"unqualified version", func(v *gatewayStatus) { version := "2026.9.4"; v.Gateway.Version = &version }},
		{"unknown version", func(v *gatewayStatus) { v.Gateway.Version = nil }},
		{"native service", func(v *gatewayStatus) { yes := true; v.Service.Loaded = &yes }},
		{"diagnostic role", func(v *gatewayStatus) { v.Service.TargetRole = "diagnostic-only" }},
		{"config mismatch", func(v *gatewayStatus) { v.Config.Mismatch = true }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := statusFixture(path, 29120, true)
			tc.mutate(&fixture)
			data, _ := json.Marshal(fixture)
			status, err := parseStatus(data, nil, path, 29120)
			if err == nil || status.State != yorvaruntime.LifecycleUnknown {
				t.Fatalf("state = %#v, error = %v", status, err)
			}
		})
	}
	for _, running := range []bool{false, true} {
		data, _ := json.Marshal(statusFixture(path, 29120, running))
		commandErr, expected := errCommand, yorvaruntime.LifecycleStopped
		if running {
			commandErr, expected = nil, yorvaruntime.LifecycleRunning
		}
		status, err := parseStatus(data, commandErr, path, 29120)
		if err != nil || status.State != expected {
			t.Fatalf("state = %#v, error = %v", status, err)
		}
	}
	if _, err := parseStatus([]byte(`{"token":"secret-value"}`), nil, path, 29120); !errors.Is(err, yorvaruntime.ErrLifecycleOutputUnrecognized) {
		t.Fatalf("malformed status: %v", err)
	}
}

package hermes

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestParseLifecycleStatusRequiresOneProcessAndOneServiceSignal(t *testing.T) {
	running, err := parseLifecycleStatus("✓ Scheduled Task registered: Hermes_Gateway\n✓ Gateway process running (PID: 12)\n")
	if err != nil || running.state != yorvaruntime.LifecycleRunning || !running.loginItemPresent {
		t.Fatalf("running = %#v, %v", running, err)
	}
	stopped, err := parseLifecycleStatus("✗ Gateway service not installed\n✗ No gateway process detected\n")
	if err != nil || stopped.state != yorvaruntime.LifecycleStopped || stopped.loginItemPresent {
		t.Fatalf("stopped = %#v, %v", stopped, err)
	}
	for _, malformed := range []string{"", "✓ Gateway process running (PID: 12)\n", "✗ Gateway service not installed\n"} {
		if _, err := parseLifecycleStatus(malformed); !errors.Is(err, yorvaruntime.ErrLifecycleOutputUnrecognized) {
			t.Fatalf("parse %q = %v", malformed, err)
		}
	}
}

func TestParseLifecycleStatusAcceptsExactManualGatewaySignals(t *testing.T) {
	running, err := parseLifecycleStatus("✓ Gateway is running (PID: 12)\n  (Running manually, not as a system service)\n")
	if err != nil || running.state != yorvaruntime.LifecycleRunning || running.loginItemPresent {
		t.Fatalf("running = %#v, %v", running, err)
	}
	stopped, err := parseLifecycleStatus("✗ Gateway is not running\n\nTo start:\n  hermes gateway run\n")
	if err != nil || stopped.state != yorvaruntime.LifecycleStopped || stopped.loginItemPresent {
		t.Fatalf("stopped = %#v, %v", stopped, err)
	}
	if _, err := parseLifecycleStatus("Gateway is maybe running\n"); !errors.Is(err, yorvaruntime.ErrLifecycleOutputUnrecognized) {
		t.Fatalf("ambiguous status = %v", err)
	}
}

func TestLifecycleStartWithoutLoginItemUsesFixedNonPersistentOfficialPath(t *testing.T) {
	executable := filepath.Join(t.TempDir(), "hermes.exe")
	installation := yorvaruntime.LifecycleInstallation{Executable: executable, Version: lifecycleOfficialVersion}
	type call struct {
		args           []string
		allowBreakaway bool
	}
	var calls []call
	manager := &LifecycleManager{run: func(_ context.Context, gotExecutable string, args []string, allowBreakaway bool) commandResult {
		if gotExecutable != executable {
			t.Fatalf("executable = %q", gotExecutable)
		}
		calls = append(calls, call{args: append([]string(nil), args...), allowBreakaway: allowBreakaway})
		switch len(calls) {
		case 1:
			return commandResult{stdout: "✗ Gateway service not installed\n✗ No gateway process detected\n", exitCode: 0}
		case 2:
			return commandResult{stdout: "ℹ Skipped Windows login auto-start install.\n", exitCode: 0}
		default:
			return commandResult{stdout: "✗ Gateway service not installed\n✓ Gateway process running (PID: 12)\n", exitCode: 0}
		}
	}}
	if err := manager.Start(context.Background(), installation, "coder"); err != nil {
		t.Fatal(err)
	}
	want := []string{"--profile", "coder", "gateway", "install", "--no-start-on-login", "--start-now"}
	if len(calls) != 3 || !reflect.DeepEqual(calls[1].args, want) || !calls[1].allowBreakaway {
		t.Fatalf("calls = %#v", calls)
	}
}

func TestLifecycleRestartStoppedFailsWithoutMutation(t *testing.T) {
	executable := filepath.Join(t.TempDir(), "hermes.exe")
	var calls int
	manager := &LifecycleManager{run: func(context.Context, string, []string, bool) commandResult {
		calls++
		return commandResult{stdout: "✓ Scheduled Task registered: Hermes_Gateway\n✗ No gateway process detected\n", exitCode: 0}
	}}
	err := manager.Restart(context.Background(), yorvaruntime.LifecycleInstallation{Executable: executable, Version: lifecycleOfficialVersion}, "default")
	if !errors.Is(err, yorvaruntime.ErrInstanceNotRunning) || calls != 1 {
		t.Fatalf("restart = %v calls=%d", err, calls)
	}
}

func TestLifecycleRejectsUnsupportedVersionBeforeExecution(t *testing.T) {
	manager := &LifecycleManager{run: func(context.Context, string, []string, bool) commandResult {
		t.Fatal("unexpected command")
		return commandResult{}
	}}
	_, err := manager.Status(context.Background(), yorvaruntime.LifecycleInstallation{Executable: filepath.Join(t.TempDir(), "hermes.exe"), Version: "0.21.0"}, "default")
	if !errors.Is(err, yorvaruntime.ErrLifecycleQueryFailed) {
		t.Fatalf("status = %v", err)
	}
}

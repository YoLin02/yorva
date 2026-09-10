package hermes

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"time"

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

func TestParseLifecycleStatusRejectsContradictionsAndChangedSignals(t *testing.T) {
	for _, output := range []string{
		"✓ Scheduled Task registered: Hermes_Gateway\n✓ Gateway process running (PID: 12)\n✗ Gateway is not running\n",
		"✓ Scheduled Task registered: Hermes_Gateway\n✓ Gateway process running (PID: 12, 13)\n",
		"✓ Scheduled Task registered: Hermes_Gateway\nGateway process running eventually\n",
		"✓ Scheduled Task registered:\n✓ Gateway process running (PID: 12)\n",
	} {
		if _, err := parseLifecycleStatus(output); !errors.Is(err, yorvaruntime.ErrLifecycleOutputUnrecognized) {
			t.Fatalf("unsafe output accepted: %q, %v", output, err)
		}
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

func TestLifecycleRestartProvesStoppedBeforeRegisteredOrManualStart(t *testing.T) {
	for _, test := range []struct {
		name       string
		registered bool
		initial    string
		stopped    string
		running    string
	}{
		{name: "registered", registered: true, initial: "✓ Scheduled Task registered: Hermes_Gateway\n✓ Gateway process running (PID: 12)\n", stopped: "✓ Scheduled Task registered: Hermes_Gateway\n✗ No gateway process detected\n", running: "✓ Scheduled Task registered: Hermes_Gateway\n✓ Gateway process running (PID: 13)\n"},
		{name: "manual", initial: "✗ Gateway service not installed\n✓ Gateway process running (PID: 12)\n", stopped: "✗ Gateway service not installed\n✗ No gateway process detected\n", running: "✗ Gateway service not installed\n✓ Gateway process running (PID: 13)\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			executable := filepath.Join(t.TempDir(), "hermes.exe")
			type call struct {
				args           []string
				allowBreakaway bool
			}
			var calls []call
			manager := &LifecycleManager{run: func(_ context.Context, _ string, args []string, allowBreakaway bool) commandResult {
				calls = append(calls, call{args: append([]string(nil), args...), allowBreakaway: allowBreakaway})
				switch len(calls) {
				case 1:
					return commandResult{stdout: test.initial, exitCode: 0}
				case 2:
					return commandResult{exitCode: 0}
				case 3:
					return commandResult{stdout: test.stopped, exitCode: 0}
				case 4:
					return commandResult{exitCode: 0}
				case 5:
					return commandResult{stdout: test.running, exitCode: 0}
				default:
					t.Fatalf("unexpected call %d", len(calls))
					return commandResult{}
				}
			}}
			if err := manager.Restart(context.Background(), yorvaruntime.LifecycleInstallation{Executable: executable, Version: lifecycleOfficialVersion}, "default"); err != nil {
				t.Fatal(err)
			}
			wantStart := lifecycleStartArgs("default", test.registered)
			if len(calls) != 5 || !reflect.DeepEqual(calls[1].args, lifecycleStopArgs("default")) || calls[1].allowBreakaway || !reflect.DeepEqual(calls[3].args, wantStart) || !calls[3].allowBreakaway {
				t.Fatalf("calls = %#v", calls)
			}
		})
	}
}

func TestLifecycleRestartDoesNotStartBeforeStoppedPostcondition(t *testing.T) {
	executable := filepath.Join(t.TempDir(), "hermes.exe")
	ctx, cancel := context.WithCancel(context.Background())
	var calls int
	manager := &LifecycleManager{run: func(_ context.Context, _ string, _ []string, _ bool) commandResult {
		calls++
		if calls == 2 {
			return commandResult{exitCode: 0}
		}
		if calls == 3 {
			cancel()
		}
		return commandResult{stdout: "✓ Scheduled Task registered: Hermes_Gateway\n✓ Gateway process running (PID: 12)\n", exitCode: 0}
	}}
	err := manager.Restart(ctx, yorvaruntime.LifecycleInstallation{Executable: executable, Version: lifecycleOfficialVersion}, "default")
	if !errors.Is(err, context.Canceled) || calls != 3 {
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

func TestLifecycleStartWaitsForColdGatewayAndHonorsCancellation(t *testing.T) {
	for _, cancelWhileStarting := range []bool{false, true} {
		t.Run(fmt.Sprint("cancel=", cancelWhileStarting), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
			defer cancel()
			calls := 0
			var launchedAt time.Time
			manager := &LifecycleManager{run: func(_ context.Context, _ string, _ []string, _ bool) commandResult {
				calls++
				if calls == 2 {
					launchedAt = time.Now()
					return commandResult{exitCode: 0}
				}
				if calls == 3 && cancelWhileStarting {
					cancel()
				}
				// A cold official Gateway can exceed the former 15-second
				// readiness window despite a successful launch command.
				if calls > 3 && time.Since(launchedAt) >= 16*time.Second {
					return commandResult{stdout: "✓ Gateway is running (PID: 12)\n", exitCode: 0}
				}
				return commandResult{stdout: "✗ Gateway is not running\n", exitCode: 0}
			}}
			err := manager.Start(ctx, yorvaruntime.LifecycleInstallation{Executable: filepath.Join(t.TempDir(), "hermes.exe"), Version: lifecycleOfficialVersion}, "coder")
			if cancelWhileStarting {
				if !errors.Is(err, context.Canceled) || calls != 3 {
					t.Fatalf("cancelled startup: err=%v calls=%d", err, calls)
				}
			} else if err != nil || calls < 4 {
				t.Fatalf("cold startup: err=%v calls=%d", err, calls)
			}
		})
	}
}

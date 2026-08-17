package hermes

import (
	"context"
	"testing"
	"time"
)

func TestOptionalStageTimeoutDoesNotFailInstall(t *testing.T) {
	installer := &HostInstaller{
		run: func(_ context.Context, invocation installInvocation, timeout time.Duration) commandResult {
			if timeout != 45*time.Second {
				t.Fatalf("optional timeout = %s", timeout)
			}
			if !containsArg(invocation.Args, "system-packages") {
				t.Fatalf("args = %#v", invocation.Args)
			}
			return commandResult{timedOut: true}
		},
	}
	if err := installer.runStage(context.Background(), "powershell", "install.ps1", "system-packages", "C:\\hermes", "C:\\hermes\\hermes-agent"); err != nil {
		t.Fatal(err)
	}
}

func TestOptionalStageHardFailureDoesNotFailInstall(t *testing.T) {
	installer := &HostInstaller{
		run: func(context.Context, installInvocation, time.Duration) commandResult {
			return commandResult{stdout: "{\"stage\":\"system-packages\",\"ok\":false,\"skipped\":false}\n", exitCode: 1, err: errPlatform}
		},
	}
	if err := installer.runStage(context.Background(), "powershell", "install.ps1", "system-packages", "C:\\hermes", "C:\\hermes\\hermes-agent"); err != nil {
		t.Fatal(err)
	}
}

func TestRequiredStageTimeoutFailsInstall(t *testing.T) {
	installer := &HostInstaller{
		run: func(context.Context, installInvocation, time.Duration) commandResult {
			return commandResult{timedOut: true}
		},
	}
	if err := installer.runStage(context.Background(), "powershell", "install.ps1", "python", "C:\\hermes", "C:\\hermes\\hermes-agent"); err == nil {
		t.Fatal("required stage timeout was ignored")
	}
}

func containsArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}

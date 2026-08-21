package hermes

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes/downloadsources"
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
	if err := installer.runStage(context.Background(), "powershell", "install.ps1", "system-packages", "C:\\hermes", "C:\\hermes\\hermes-agent", downloadsources.Default()); err != nil {
		t.Fatal(err)
	}
}

func TestOptionalStageHardFailureDoesNotFailInstall(t *testing.T) {
	installer := &HostInstaller{
		run: func(context.Context, installInvocation, time.Duration) commandResult {
			return commandResult{stdout: "{\"stage\":\"system-packages\",\"ok\":false,\"skipped\":false}\n", exitCode: 1, err: errPlatform}
		},
	}
	if err := installer.runStage(context.Background(), "powershell", "install.ps1", "system-packages", "C:\\hermes", "C:\\hermes\\hermes-agent", downloadsources.Default()); err != nil {
		t.Fatal(err)
	}
}

func TestLogCommandOmitsRawCommandOutput(t *testing.T) {
	var output bytes.Buffer
	installer := NewHostInstaller(t.TempDir())
	installer.logger = slog.New(slog.NewJSONHandler(&output, nil))
	installer.logCommand("installer.stage", "venv", "", commandResult{
		stdout:   "SECRET_TOKEN=super-secret\nC:\\Users\\private\\hermes",
		stderr:   "official reason: disk full at C:\\Users\\private",
		exitCode: 1,
		err:      errors.New("raw powershell error"),
	})
	text := output.String()
	if strings.Contains(text, "SECRET_TOKEN") || strings.Contains(text, "C:\\Users\\private") || strings.Contains(text, "raw powershell") {
		t.Fatalf("raw output persisted: %s", text)
	}
	if !strings.Contains(text, string(yorvaruntime.ErrorRuntimeInstallStageFailed)) {
		t.Fatalf("missing structured code: %s", text)
	}
}

func TestRequiredStageTimeoutFailsInstall(t *testing.T) {
	installer := &HostInstaller{
		run: func(context.Context, installInvocation, time.Duration) commandResult {
			return commandResult{timedOut: true}
		},
	}
	if err := installer.runStage(context.Background(), "powershell", "install.ps1", "python", "C:\\hermes", "C:\\hermes\\hermes-agent", downloadsources.Default()); err == nil {
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

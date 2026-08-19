package hermes

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDeleteProfileArgvRequiresYesAndRejectsDefault(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	executable := filepath.Join(t.TempDir(), "hermes")
	var got []string
	err := deleteProfileWith(context.Background(), executable, "coder", func(_ context.Context, _, _, nativeID string) commandResult {
		got = profileDeleteArgs(nativeID)
		return commandResult{}
	})
	if err != nil {
		t.Fatal(err)
	}
	if !equalStrings(got, []string{"profile", "delete", "coder", "--yes"}) {
		t.Fatalf("delete argv = %#v", got)
	}
	if strings.Contains(got[2], `\`) || strings.Contains(got[2], "/") {
		t.Fatal("delete argv used a path")
	}

	called := false
	if err := deleteProfileWith(context.Background(), executable, "default", func(context.Context, string, string, string) commandResult {
		called = true
		return commandResult{}
	}); err == nil || called {
		t.Fatalf("default delete started process: err=%v called=%v", err, called)
	}
}

func TestDeleteProfileSurvivesLongerThanDiscoveryTimeout(t *testing.T) {
	if profileMutationTimeout <= commandTimeout {
		t.Fatalf("profile mutation timeout %s must exceed discovery timeout %s", profileMutationTimeout, commandTimeout)
	}
	t.Setenv("LOCALAPPDATA", t.TempDir())
	t.Setenv("YORVA_FAKE_HERMES_MODE", "profile-delete-slow")
	t.Setenv("YORVA_FAKE_HERMES_SLEEP_MS", "3200")
	executable := buildFakeHermes(t)
	home := officialHermesHome()
	env := func() []string {
		return append(profileCommandEnvironment(home),
			"YORVA_FAKE_HERMES_MODE="+os.Getenv("YORVA_FAKE_HERMES_MODE"),
			"YORVA_FAKE_HERMES_SLEEP_MS="+os.Getenv("YORVA_FAKE_HERMES_SLEEP_MS"),
		)
	}
	invocation := commandInvocation{path: executable, executable: executable, args: profileDeleteArgs("coder")}

	discovery := newCommandRunner()
	discovery.environment = env
	if result := discovery.run(context.Background(), invocation); !result.timedOut {
		t.Fatalf("discovery-budget runner must time out a 3.2s delete: %#v", result)
	}

	mutation := newProfileMutationRunner()
	mutation.environment = env
	start := time.Now()
	result := mutation.run(context.Background(), invocation)
	elapsed := time.Since(start)
	if result.timedOut || result.err != nil || result.exitCode != 0 {
		t.Fatalf("mutation runner after %s: %#v", elapsed, result)
	}
	if elapsed < commandTimeout {
		t.Fatalf("slow delete returned in %s, want more than discovery timeout %s", elapsed, commandTimeout)
	}
}

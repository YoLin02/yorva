package hermes

import (
	"context"
	"strings"
	"testing"
)

func TestCreateProfileArgvIsNoCloneNoAliasNoSkills(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	var got []string
	err := createProfileWith(context.Background(), `C:\hermes\bin\hermes.exe`, "coder", func(_ context.Context, executable, home, name string) commandResult {
		got = profileCreateArgs(name)
		if executable != `C:\hermes\bin\hermes.exe` || home == "" {
			t.Fatalf("create invocation executable=%q home=%q", executable, home)
		}
		return commandResult{}
	})
	if err != nil {
		t.Fatal(err)
	}
	if !equalStrings(got, []string{"profile", "create", "coder", "--no-alias", "--no-skills"}) {
		t.Fatalf("create argv = %#v", got)
	}
	joined := strings.Join(got, " ")
	for _, flag := range []string{"--clone", "--clone-all", "--clone-from", "--description"} {
		if strings.Contains(joined, flag) {
			t.Fatalf("forbidden flag %s in %#v", flag, got)
		}
	}
}

func TestCreateProfileRejectsInvalidNameBeforeProcess(t *testing.T) {
	called := false
	err := createProfileWith(context.Background(), `C:\hermes\bin\hermes.exe`, "default", func(context.Context, string, string, string) commandResult {
		called = true
		return commandResult{}
	})
	if err == nil || called {
		t.Fatalf("default create started process: err=%v called=%v", err, called)
	}
}

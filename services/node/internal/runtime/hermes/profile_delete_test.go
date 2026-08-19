package hermes

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
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

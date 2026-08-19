package hermes

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestListProfilesUsesExactArgvAndAllowlistedEnvironment(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	t.Setenv("YORVA_FAKE_HERMES_MODE", "profile-list")
	t.Setenv("YORVA_TEST_PROVIDER_API_KEY", "must-not-leak")
	t.Setenv("OPENAI_API_KEY", "sk-test")
	fixture := readProfileFixture(t, "list-default-and-named.txt")
	t.Setenv("YORVA_FAKE_HERMES_PROFILE_LIST", fixture)
	executable := buildFakeHermes(t)

	got, err := listProfilesWith(context.Background(), executable, runFakeProfileListCommand)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].NativeID != "default" || !got[0].Default || got[1].NativeID != "work" {
		t.Fatalf("ListProfiles() = %#v", got)
	}

	env := profileCommandEnvironment(officialHermesHome())
	if profileEnvHasSecret(env, "must-not-leak") || profileEnvHasSecret(env, "sk-test") || profileEnvHasSecret(env, "OPENAI_API_KEY") {
		t.Fatalf("profile environment leaked secret: %#v", env)
	}
	foundHome := false
	for _, entry := range env {
		if strings.HasPrefix(entry, "HERMES_HOME=") {
			foundHome = true
		}
	}
	if !foundHome {
		t.Fatal("profile environment missing HERMES_HOME")
	}
}

func TestListProfilesRejectsUnrecognizedOutput(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	t.Setenv("YORVA_FAKE_HERMES_MODE", "profile-list")
	t.Setenv("YORVA_FAKE_HERMES_PROFILE_LIST", readProfileFixture(t, "list-docs-star-format.txt"))
	_, err := listProfilesWith(context.Background(), buildFakeHermes(t), runFakeProfileListCommand)
	if !IsProfileOutputUnrecognized(err) {
		t.Fatalf("unrecognized = %v", err)
	}
}

func TestListProfilesMapsTimeoutAndUnsafeExecutable(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	if _, err := ListProfiles(context.Background(), "hermes.exe"); !errors.Is(err, errProfileExecutableUnsafe) {
		t.Fatalf("relative executable = %v", err)
	}
	t.Setenv("YORVA_FAKE_HERMES_MODE", "wait")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := listProfilesWith(ctx, buildFakeHermes(t), func(ctx context.Context, executable, home string) commandResult {
		runner := newCommandRunner()
		runner.timeout = 50 * time.Millisecond
		runner.environment = func() []string { return profileCommandEnvironment(home) }
		return runner.run(ctx, commandInvocation{path: executable, executable: executable, args: profileListArgs()})
	})
	if !IsProfileQueryFailure(err) {
		t.Fatalf("timeout = %v", err)
	}
}

func TestListProfilesDoesNotAcceptCreateOrDeleteArgv(t *testing.T) {
	t.Setenv("LOCALAPPDATA", t.TempDir())
	t.Setenv("YORVA_FAKE_HERMES_MODE", "profile-list")
	t.Setenv("YORVA_FAKE_HERMES_PROFILE_LIST", readProfileFixture(t, "list-default-only.txt"))
	executable := buildFakeHermes(t)
	result := runFakeProfileListCommand(context.Background(), executable, officialHermesHome())
	if result.exitCode != 0 {
		t.Fatalf("list command failed: %#v", result)
	}
	create := newCommandRunner()
	create.environment = func() []string { return profileCommandEnvironment(officialHermesHome()) }
	created := create.run(context.Background(), commandInvocation{
		path: executable, executable: executable, args: profileCreateArgs("coder"),
	})
	if created.exitCode == 0 {
		t.Fatal("Batch 2 must not succeed a create subprocess against the list-only fake")
	}
}

func runFakeProfileListCommand(ctx context.Context, executable, home string) commandResult {
	runner := newCommandRunner()
	runner.environment = func() []string {
		return append(profileCommandEnvironment(home),
			"YORVA_FAKE_HERMES_MODE="+os.Getenv("YORVA_FAKE_HERMES_MODE"),
			"YORVA_FAKE_HERMES_PROFILE_LIST="+os.Getenv("YORVA_FAKE_HERMES_PROFILE_LIST"),
		)
	}
	return runner.run(ctx, commandInvocation{
		path:       executable,
		executable: executable,
		args:       profileListArgs(),
	})
}

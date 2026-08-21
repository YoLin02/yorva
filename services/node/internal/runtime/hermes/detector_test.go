package hermes

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestDetectorOutcomes(t *testing.T) {
	tests := []struct {
		name           string
		commands       []commandResult
		wantState      yorvaruntime.DiscoveryState
		wantCode       yorvaruntime.ErrorCode
		wantVersion    string
		wantSelected   bool
		wantCandidates int
	}{
		{
			name:           "not installed",
			wantState:      yorvaruntime.DiscoveryNotInstalled,
			wantCode:       yorvaruntime.ErrorRuntimeNotInstalled,
			wantCandidates: 0,
		},
		{
			name:      "supported",
			commands:  []commandResult{{stdout: "Hermes Agent v0.20.2 (2026.8.16)\n", exitCode: 0}},
			wantState: yorvaruntime.DiscoverySupported, wantVersion: "0.20.2", wantSelected: true, wantCandidates: 1,
		},
		{
			name:      "older version unsupported",
			commands:  []commandResult{{stdout: "Hermes Agent v0.19.0\n", exitCode: 0}},
			wantState: yorvaruntime.DiscoveryUnsupported, wantCode: yorvaruntime.ErrorRuntimeUnsupported,
			wantVersion: "0.19.0", wantSelected: true, wantCandidates: 1,
		},
		{
			name:      "broken executable",
			commands:  []commandResult{{exitCode: 3, err: errors.New("process failed")}},
			wantState: yorvaruntime.DiscoveryBrokenExecutable, wantCode: yorvaruntime.ErrorRuntimeExecutableBroken, wantCandidates: 1,
		},
		{
			name:      "malformed version",
			commands:  []commandResult{{stdout: "not a Hermes version\n", exitCode: 0}},
			wantState: yorvaruntime.DiscoveryMalformedVersion, wantCode: yorvaruntime.ErrorRuntimeVersionMalformed, wantCandidates: 1,
		},
		{
			name:      "timed out",
			commands:  []commandResult{{exitCode: -1, err: context.DeadlineExceeded, timedOut: true}},
			wantState: yorvaruntime.DiscoveryTimedOut, wantCode: yorvaruntime.ErrorRuntimeDiscoveryTimeout, wantCandidates: 1,
		},
		{
			name:      "output limited",
			commands:  []commandResult{{exitCode: -1, err: errOutputLimit, limited: true}},
			wantState: yorvaruntime.DiscoveryBrokenExecutable, wantCode: yorvaruntime.ErrorRuntimeCommandOutputLimit, wantCandidates: 1,
		},
		{
			name: "multiple runnable candidates are ambiguous",
			commands: []commandResult{
				{stdout: "Hermes Agent v0.19.0\n", exitCode: 0},
				{stdout: "Hermes Agent v0.19.1\n", exitCode: 0},
			},
			wantState: yorvaruntime.DiscoveryAmbiguous, wantCode: yorvaruntime.ErrorRuntimeDiscoveryAmbiguous, wantCandidates: 2,
		},
		{
			name: "one runnable candidate wins over broken candidate",
			commands: []commandResult{
				{exitCode: 2, err: errors.New("process failed")},
				{stdout: "Hermes Agent v0.20.2\n", exitCode: 0},
			},
			wantState: yorvaruntime.DiscoverySupported, wantVersion: "0.20.2", wantSelected: true, wantCandidates: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			detector := detectorWithResults(t, test.commands)
			got, err := detector.Detect(context.Background())
			if err != nil {
				t.Fatalf("Detect() error = %v", err)
			}
			if got.State != test.wantState || got.ErrorCode != test.wantCode {
				t.Fatalf("Detect() state/code = %s/%s, want %s/%s", got.State, got.ErrorCode, test.wantState, test.wantCode)
			}
			if len(got.Candidates) != test.wantCandidates {
				t.Fatalf("Detect() candidates = %d, want %d", len(got.Candidates), test.wantCandidates)
			}
			if (got.Selected != nil) != test.wantSelected {
				t.Fatalf("Detect() selected = %#v, wantSelected=%v", got.Selected, test.wantSelected)
			}
			if got.Selected != nil && got.Selected.Version != test.wantVersion {
				t.Fatalf("Detect() selected version = %q, want %q", got.Selected.Version, test.wantVersion)
			}
			if got.SupportedRange != supportedRange || got.DetectedAt.IsZero() {
				t.Fatalf("Detect() metadata = %#v", got)
			}
		})
	}
}

func TestDetectorWarnsForUntestedPrerelease(t *testing.T) {
	detector := detectorWithResults(t, []commandResult{{stdout: "Hermes Agent v0.20.2-rc.1\n", exitCode: 0}})
	got, err := detector.Detect(context.Background())
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if got.State != yorvaruntime.DiscoveryUnsupported || len(got.Warnings) != 1 || got.Warnings[0].Code != "PRERELEASE_UNTESTED" {
		t.Fatalf("Detect() = %#v, want unsupported prerelease warning", got)
	}
}

func TestDetectorClassifiesMissingLauncherAsBrokenWithoutExecutingEvidence(t *testing.T) {
	root := t.TempDir()
	venvScripts := filepath.Join(root, "venv", "Scripts")
	if err := os.MkdirAll(venvScripts, 0o755); err != nil {
		t.Fatal(err)
	}
	legacyAgent := filepath.Join(venvScripts, "hermes-agent.exe")
	if err := os.WriteFile(legacyAgent, []byte("must not execute"), 0o755); err != nil {
		t.Fatal(err)
	}

	executed := false
	detector := &Detector{
		finder: candidateFinder{
			executableName:    "hermes.exe",
			installationRoots: []string{root},
			limit:             maxCandidates,
		},
		run: func(context.Context, commandInvocation) commandResult {
			executed = true
			return commandResult{}
		},
		now:            time.Now,
		overallTimeout: time.Second,
	}

	got, err := detector.Detect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if executed {
		t.Fatal("Detect() executed installation evidence")
	}
	if got.State != yorvaruntime.DiscoveryBrokenExecutable || got.ErrorCode != yorvaruntime.ErrorRuntimeExecutableBroken {
		t.Fatalf("Detect() state/code = %s/%s, want BROKEN_EXECUTABLE/RUNTIME_EXECUTABLE_BROKEN", got.State, got.ErrorCode)
	}
	if len(got.Candidates) != 0 || len(got.Warnings) != 1 || got.Warnings[0].Code != "HERMES_CLI_LAUNCHER_MISSING" {
		t.Fatalf("Detect() = %#v, want missing-launcher evidence without a candidate", got)
	}
}

func TestDetectorUsesTrustedOfficialPythonEntrypointWhenLauncherIsMissing(t *testing.T) {
	root := writeOfficialPythonInstallation(t, hermesCLIEntryPoint)
	var executed commandInvocation
	detector := &Detector{
		finder: candidateFinder{
			executableName:    "hermes.exe",
			installationRoots: []string{root},
			caseInsensitive:   true,
			limit:             maxCandidates,
		},
		run: func(_ context.Context, command commandInvocation) commandResult {
			executed = command
			return commandResult{stdout: "Hermes Agent v0.20.2 (2026.8.16)\n", exitCode: 0}
		},
		now:            time.Now,
		overallTimeout: time.Second,
	}

	got, err := detector.Detect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.State != yorvaruntime.DiscoverySupported || got.ErrorCode != "" {
		t.Fatalf("Detect() state/code = %s/%s, want SUPPORTED with no error", got.State, got.ErrorCode)
	}
	if got.Selected == nil || got.Selected.Version != "0.20.2" || got.Selected.Path != executed.path {
		t.Fatalf("Detect() selected = %#v, invocation = %#v", got.Selected, executed)
	}
	if !slices.Equal(executed.args, []string{"-I", "-m", hermesCLIModule, "--version"}) {
		t.Fatalf("Detect() executed args %#v", executed.args)
	}
}

func TestDetectorPropagatesCallerCancellation(t *testing.T) {
	candidate := createCandidateFile(t)
	started := make(chan struct{})
	detector := &Detector{
		finder: candidateFinder{
			officialPaths:  []string{candidate},
			executableName: executableNameForTest(),
			limit:          maxCandidates,
		},
		run: func(ctx context.Context, _ commandInvocation) commandResult {
			close(started)
			<-ctx.Done()
			return commandResult{exitCode: -1, err: ctx.Err()}
		},
		now:            time.Now,
		overallTimeout: time.Second,
	}
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := detector.Detect(ctx)
		result <- err
	}()
	<-started
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Detect() error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Detect() did not stop after cancellation")
	}
}

func detectorWithResults(t *testing.T, results []commandResult) *Detector {
	t.Helper()
	paths := make([]string, 0, len(results))
	byPath := make(map[string]commandResult, len(results))
	for _, result := range results {
		path := createCandidateFile(t)
		canonical := canonicalPath(t, path)
		paths = append(paths, canonical)
		byPath[canonical] = result
	}
	return &Detector{
		finder: candidateFinder{
			officialPaths:  paths,
			executableName: executableNameForTest(),
			limit:          maxCandidates,
		},
		run: func(_ context.Context, command commandInvocation) commandResult {
			return byPath[command.path]
		},
		now:            func() time.Time { return time.Date(2026, 8, 14, 0, 0, 0, 0, time.UTC) },
		overallTimeout: time.Second,
	}
}

func createCandidateFile(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	path := filepath.Join(directory, executableNameForTest())
	if err := os.WriteFile(path, []byte("fixture"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

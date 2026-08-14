package hermes

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestReadBounded(t *testing.T) {
	data, err := readBounded(strings.NewReader("1234"), 4)
	if err != nil || string(data) != "1234" {
		t.Fatalf("readBounded exact = %q, %v", data, err)
	}
	data, err = readBounded(bytes.NewReader([]byte("12345")), 4)
	if !errors.Is(err, errOutputLimit) || string(data) != "1234" {
		t.Fatalf("readBounded overflow = %q, %v", data, err)
	}
}

func TestMinimalEnvironmentExcludesSecrets(t *testing.T) {
	t.Setenv("YORVA_TEST_PROVIDER_API_KEY", "must-not-leak")
	for _, entry := range minimalEnvironment() {
		if strings.HasPrefix(entry, "YORVA_TEST_PROVIDER_API_KEY=") || strings.Contains(entry, "must-not-leak") {
			t.Fatalf("minimalEnvironment leaked test secret: %q", entry)
		}
	}
}

func TestCommandRunnerExecutesOnlyVersionArgument(t *testing.T) {
	executable := buildFakeHermes(t)
	runner := testCommandRunner("success", "", time.Second)
	result := runner.run(context.Background(), executable)
	if result.err != nil || result.exitCode != 0 || result.stdout != "Hermes Agent v0.19.7 (2026.8.14)\n" {
		t.Fatalf("run() = %#v", result)
	}
}

func TestCommandRunnerBoundsOutputAndReapsOnTimeout(t *testing.T) {
	executable := buildFakeHermes(t)
	limited := testCommandRunner("output-limit", "", time.Second).run(context.Background(), executable)
	if !limited.limited || !errors.Is(limited.err, errOutputLimit) {
		t.Fatalf("output-limited run() = %#v", limited)
	}

	pidFile := filepath.Join(t.TempDir(), "pid")
	timedOut := testCommandRunner("wait", pidFile, 100*time.Millisecond).run(context.Background(), executable)
	if !timedOut.timedOut || !errors.Is(timedOut.err, context.DeadlineExceeded) {
		t.Fatalf("timed-out run() = %#v", timedOut)
	}
	pid := readPID(t, pidFile)
	if !waitForProcessExit(pid, 2*time.Second) {
		t.Fatalf("timed-out Hermes process %d still exists after run returned", pid)
	}
}

func TestCommandRunnerCancellationReapsProcess(t *testing.T) {
	executable := buildFakeHermes(t)
	pidFile := filepath.Join(t.TempDir(), "pid")
	runner := testCommandRunner("child-wait", pidFile, 5*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan commandResult, 1)
	go func() { result <- runner.run(ctx, executable) }()
	waitForFile(t, pidFile)
	cancel()
	select {
	case got := <-result:
		if !errors.Is(got.err, context.Canceled) {
			t.Fatalf("cancelled run() = %#v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("run() did not return after cancellation")
	}
	pid := readPID(t, pidFile)
	if !waitForProcessExit(pid, 2*time.Second) {
		t.Fatalf("cancelled Hermes descendant process %d still exists after run returned", pid)
	}
}

func TestCommandRunnerTimeoutReapsDescendantProcess(t *testing.T) {
	executable := buildFakeHermes(t)
	pidFile := filepath.Join(t.TempDir(), "child-pid")
	result := testCommandRunner("child-wait", pidFile, 150*time.Millisecond).run(context.Background(), executable)
	if !result.timedOut || !errors.Is(result.err, context.DeadlineExceeded) {
		t.Fatalf("timed-out process tree run() = %#v", result)
	}
	pid := readPID(t, pidFile)
	if !waitForProcessExit(pid, 2*time.Second) {
		t.Fatalf("Hermes descendant process %d still exists after run returned", pid)
	}
}

func TestCommandRunnerOwnsImmediateDescendantBeforeExecutableRuns(t *testing.T) {
	executable := buildFakeHermes(t)
	pidFile := filepath.Join(t.TempDir(), "immediate-child-pid")
	result := testCommandRunner("child-exit", pidFile, time.Second).run(context.Background(), executable)
	if result.exitCode != 0 || result.err != nil {
		t.Fatalf("immediate-child run() = %#v", result)
	}
	pid := readPID(t, pidFile)
	if !waitForProcessExit(pid, 2*time.Second) {
		t.Fatalf("immediate Hermes descendant process %d still exists after normal return", pid)
	}
}

func TestCommandRunnerReturnsAfterAlreadyCancelledContext(t *testing.T) {
	runner := commandRunner{
		timeout:     time.Second,
		waitDelay:   time.Second,
		outputLimit: commandOutputLimit,
		environment: minimalEnvironment,
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	started := time.Now()
	result := runner.run(ctx, createCandidateFile(t))
	if time.Since(started) > time.Second {
		t.Fatal("run() did not return promptly for a cancelled context")
	}
	if result.err == nil {
		t.Fatal("run() unexpectedly succeeded for a cancelled context")
	}
}

func testCommandRunner(mode, pidFile string, timeout time.Duration) commandRunner {
	return commandRunner{
		timeout:     timeout,
		waitDelay:   time.Second,
		outputLimit: commandOutputLimit,
		environment: func() []string {
			return append(minimalEnvironment(),
				"YORVA_FAKE_HERMES_MODE="+mode,
				"YORVA_FAKE_HERMES_PID_FILE="+pidFile,
			)
		},
	}
}

func buildFakeHermes(t *testing.T) string {
	t.Helper()
	name := "hermes"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	output := filepath.Join(t.TempDir(), name)
	command := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), "build", "-o", output, "./testdata/fakehermes")
	if combined, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build fake Hermes: %v\n%s", err, combined)
	}
	return output
}

func waitForFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", path)
}

func readPID(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(string(data))
	if err != nil {
		t.Fatal(err)
	}
	return pid
}

func processExists(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	defer process.Release()
	if runtime.GOOS == "windows" {
		return true
	}
	if status, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid)); err == nil {
		fields := strings.Fields(string(status))
		if len(fields) > 2 && fields[2] == "Z" {
			return false
		}
	}
	return process.Signal(syscall.Signal(0)) == nil
}

func waitForProcessExit(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !processExists(pid) {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return !processExists(pid)
}

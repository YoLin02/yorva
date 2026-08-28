package hermes

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes/downloadsources"
)

const (
	commandTimeout            = 3 * time.Second
	discoveryCommandTimeout   = 30 * time.Second
	profileMutationTimeout    = 30 * time.Second
	commandWaitDelay          = time.Second
	commandOutputLimit        = 64 * 1024
	installCommandOutputLimit = 1024 * 1024
)

var (
	errOutputLimit       = errors.New("command output limit exceeded")
	errCommandOutputRead = errors.New("command output could not be read")
)

type commandResult struct {
	stdout   string
	stderr   string
	exitCode int
	err      error
	timedOut bool
	limited  bool
	ready    bool
}

type commandRunner struct {
	timeout        time.Duration
	waitDelay      time.Duration
	outputLimit    int64
	allowBreakaway bool
	environment    func() []string
	stdoutReady    func([]byte) bool
}

func newCommandRunner() commandRunner {
	return commandRunner{
		timeout:     commandTimeout,
		waitDelay:   commandWaitDelay,
		outputLimit: commandOutputLimit,
		environment: minimalEnvironment,
	}
}

func newDiscoveryCommandRunner() commandRunner {
	runner := newCommandRunner()
	runner.timeout = discoveryCommandTimeout
	runner.environment = func() []string {
		return append(minimalEnvironment(), "PYTHONUNBUFFERED=1")
	}
	runner.stdoutReady = discoveryVersionReady
	return runner
}

func newProfileMutationRunner() commandRunner {
	runner := newCommandRunner()
	runner.timeout = profileMutationTimeout
	return runner
}

func newInstallCommandRunner(timeout time.Duration, home string) commandRunner {
	return commandRunner{
		timeout:     timeout,
		waitDelay:   commandWaitDelay,
		outputLimit: installCommandOutputLimit,
		environment: func() []string { return installerEnvironment(home, downloadsources.Default()) },
	}
}

func runInstallInvocation(ctx context.Context, runner commandRunner, invocation installInvocation) commandResult {
	if invocation.Environment != nil {
		runner.environment = func() []string { return append([]string(nil), invocation.Environment...) }
	}
	return runner.run(ctx, commandInvocation{
		path:       invocation.Executable,
		executable: invocation.Executable,
		args:       invocation.Args,
		workingDir: invocation.Dir,
	})
}

func (r commandRunner) run(ctx context.Context, invocation commandInvocation) commandResult {
	commandCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	command := exec.CommandContext(commandCtx, invocation.executable, invocation.args...)
	command.Env = r.environment()
	command.Dir = invocation.workingDir
	command.WaitDelay = r.waitDelay
	configureProcessTree(command)
	stdout, err := command.StdoutPipe()
	if err != nil {
		return commandResult{exitCode: -1, err: err}
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		return commandResult{exitCode: -1, err: err}
	}
	if err := command.Start(); err != nil {
		return commandResult{exitCode: -1, err: err}
	}
	cleanupProcessTree, err := ownProcessTree(command, r.allowBreakaway)
	if err != nil {
		_ = command.Process.Kill()
		_ = command.Wait()
		return commandResult{exitCode: -1, err: err}
	}
	defer cleanupProcessTree()

	type streamResult struct {
		name  string
		data  []byte
		ready bool
		err   error
	}
	streams := make(chan streamResult, 2)
	read := func(name string, source io.Reader) {
		if name == "stdout" && invocation.trusted && r.stdoutReady != nil {
			data, ready, readErr := readBoundedUntil(source, r.outputLimit, r.stdoutReady)
			streams <- streamResult{name: name, data: data, ready: ready, err: readErr}
			return
		}
		data, readErr := readBounded(source, r.outputLimit)
		streams <- streamResult{name: name, data: data, err: readErr}
	}
	go read("stdout", stdout)
	go read("stderr", stderr)

	var stdoutData []byte
	var stderrData []byte
	received := 0
	limited := false
	ready := false
	var readErr error
	contextDone := commandCtx.Done()
	terminate := func() {
		cleanupProcessTree()
		_ = command.Process.Kill()
	}
	for received < 2 {
		select {
		case stream := <-streams:
			received++
			if stream.name == "stdout" {
				stdoutData = stream.data
			} else {
				stderrData = stream.data
			}
			classified := classifyCommandReadError(stream.err)
			if errors.Is(classified, errOutputLimit) {
				limited = true
				terminate()
			} else if classified != nil {
				if readErr == nil {
					readErr = classified
				}
				terminate()
			}
			if stream.ready {
				ready = true
				terminate()
			}
		case <-contextDone:
			terminate()
			contextDone = nil
		}
	}

	// StdoutPipe and StderrPipe require all reads to complete before Wait.
	// Timeout/output failures terminate the owned process tree above so both
	// readers reach EOF; only then is it safe to reap the command.
	waitErr := command.Wait()

	exitCode := -1
	if command.ProcessState != nil {
		exitCode = command.ProcessState.ExitCode()
	}
	stdoutText := string(stdoutData)
	stderrText := string(stderrData)
	if limited {
		return commandResult{stdout: stdoutText, stderr: stderrText, exitCode: exitCode, err: errOutputLimit, limited: true}
	}
	if commandCtx.Err() != nil {
		return commandResult{
			stdout:   stdoutText,
			stderr:   stderrText,
			exitCode: exitCode,
			err:      commandCtx.Err(),
			timedOut: errors.Is(commandCtx.Err(), context.DeadlineExceeded),
		}
	}
	if readErr != nil {
		return commandResult{stdout: stdoutText, stderr: stderrText, exitCode: exitCode, err: readErr}
	}
	if ready {
		return commandResult{stdout: stdoutText, stderr: stderrText, exitCode: exitCode, ready: true}
	}
	return commandResult{stdout: stdoutText, stderr: stderrText, exitCode: exitCode, err: waitErr}
}

func classifyCommandReadError(err error) error {
	if err == nil || errors.Is(err, errOutputLimit) {
		return err
	}
	return errCommandOutputRead
}

func readBounded(source io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(source, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return data[:limit], errOutputLimit
	}
	return data, nil
}

func readBoundedUntil(source io.Reader, limit int64, ready func([]byte) bool) ([]byte, bool, error) {
	data := make([]byte, 0, min(limit, 4096))
	buffer := make([]byte, 4096)
	for {
		read, err := source.Read(buffer)
		if read > 0 {
			remaining := limit - int64(len(data))
			if int64(read) > remaining {
				data = append(data, buffer[:remaining]...)
				return data, false, errOutputLimit
			}
			data = append(data, buffer[:read]...)
			if ready(data) {
				return data, true, nil
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return data, false, nil
			}
			return data, false, err
		}
	}
}

func minimalEnvironment() []string {
	allowed := map[string]struct{}{
		"APPDATA": {}, "COMSPEC": {}, "HERMES_HOME": {}, "HOME": {},
		"LANG": {}, "LOCALAPPDATA": {}, "PATH": {}, "PATHEXT": {},
		"SYSTEMROOT": {}, "TEMP": {}, "TMP": {}, "USERPROFILE": {}, "WINDIR": {},
	}
	result := make([]string, 0, len(allowed))
	for _, entry := range os.Environ() {
		name, _, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		upperName := strings.ToUpper(name)
		if _, ok := allowed[upperName]; ok || strings.HasPrefix(upperName, "LC_") {
			result = append(result, entry)
		}
	}
	return result
}

package hermes

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	commandTimeout     = 3 * time.Second
	commandWaitDelay   = time.Second
	commandOutputLimit = 64 * 1024
)

var errOutputLimit = errors.New("command output limit exceeded")

type commandResult struct {
	stdout   string
	exitCode int
	err      error
	timedOut bool
	limited  bool
}

type commandRunner struct {
	timeout     time.Duration
	waitDelay   time.Duration
	outputLimit int64
	environment func() []string
}

func newCommandRunner() commandRunner {
	return commandRunner{
		timeout:     commandTimeout,
		waitDelay:   commandWaitDelay,
		outputLimit: commandOutputLimit,
		environment: minimalEnvironment,
	}
}

func (r commandRunner) run(ctx context.Context, executable string) commandResult {
	commandCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	command := exec.CommandContext(commandCtx, executable, "--version")
	command.Env = r.environment()
	command.WaitDelay = r.waitDelay
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

	type streamResult struct {
		name string
		data []byte
		err  error
	}
	streams := make(chan streamResult, 2)
	read := func(name string, source io.Reader) {
		data, readErr := readBounded(source, r.outputLimit)
		streams <- streamResult{name: name, data: data, err: readErr}
	}
	go read("stdout", stdout)
	go read("stderr", stderr)

	waited := make(chan error, 1)
	go func() { waited <- command.Wait() }()

	var stdoutData []byte
	var waitErr error
	received := 0
	waitComplete := false
	limited := false
	contextDone := commandCtx.Done()
	for received < 2 || !waitComplete {
		select {
		case stream := <-streams:
			received++
			if stream.name == "stdout" {
				stdoutData = stream.data
			}
			if errors.Is(stream.err, errOutputLimit) {
				limited = true
				_ = command.Process.Kill()
			}
		case waitErr = <-waited:
			waitComplete = true
		case <-contextDone:
			_ = command.Process.Kill()
			contextDone = nil
		}
	}

	exitCode := -1
	if command.ProcessState != nil {
		exitCode = command.ProcessState.ExitCode()
	}
	if limited {
		return commandResult{exitCode: exitCode, err: errOutputLimit, limited: true}
	}
	if commandCtx.Err() != nil {
		return commandResult{
			exitCode: exitCode,
			err:      commandCtx.Err(),
			timedOut: errors.Is(commandCtx.Err(), context.DeadlineExceeded),
		}
	}
	return commandResult{stdout: string(stdoutData), exitCode: exitCode, err: waitErr}
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

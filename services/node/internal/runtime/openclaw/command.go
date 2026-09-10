package openclaw

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var errCommand = errors.New("OpenClaw command failed")
var errOutputLimit = errors.New("OpenClaw command output exceeds limit")

// command never returns stderr or raw process errors across the adapter boundary.
func (a *Adapter) command(ctx context.Context, node, entry, profile string, timeout time.Duration, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	argv := []string{entry}
	if profile != "" {
		argv = append(argv, "--profile", profile)
	}
	cmd := exec.Command(node, append(argv, args...)...)
	cmd.Env = a.environment(node, profile)
	cmd.Dir = a.home
	configureProcess(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errCommand
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, errCommand
	}
	if cmd.Start() != nil {
		return nil, errCommand
	}
	job, err := ownProcess(cmd)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, errCommand
	}
	defer job.close()
	type output struct {
		bytes  []byte
		err    error
		stdout bool
	}
	results := make(chan output, 2)
	go func() { b, e := readBounded(stdout, 256*1024); results <- output{b, e, true} }()
	go func() { b, e := readBounded(stderr, 256*1024); clear(b); results <- output{nil, e, false} }()
	var data []byte
	var readErr error
	done := ctx.Done()
	for count := 0; count < 2; {
		select {
		case r := <-results:
			count++
			if r.stdout {
				data = r.bytes
			}
			if r.err != nil {
				readErr = r.err
				job.close()
				_ = cmd.Process.Kill()
			}
		case <-done:
			job.close()
			_ = cmd.Process.Kill()
			done = nil
		}
	}
	waitErr := cmd.Wait()
	if ctx.Err() != nil {
		clear(data)
		return nil, ctx.Err()
	}
	if readErr != nil {
		clear(data)
		return nil, readErr
	}
	if waitErr != nil {
		return data, errCommand
	}
	return data, nil
}

func readBounded(reader io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if int64(len(data)) > limit {
		clear(data)
		return nil, errOutputLimit
	}
	if err != nil {
		clear(data)
		return nil, errCommand
	}
	return data, nil
}

func (a *Adapter) environment(node, profile string) []string {
	// Exclude inherited provider credentials, NODE_OPTIONS, OpenClaw path/auth
	// overrides, YORVA's API token and shell hooks.
	allowed := map[string]bool{"SYSTEMROOT": true, "WINDIR": true, "COMSPEC": true, "TEMP": true, "TMP": true, "APPDATA": true, "LOCALAPPDATA": true, "PROGRAMDATA": true, "PROGRAMFILES": true, "PROGRAMFILES(X86)": true, "USERNAME": true, "USERDOMAIN": true}
	env := []string{}
	for _, entry := range os.Environ() {
		key, _, ok := strings.Cut(entry, "=")
		if ok && allowed[strings.ToUpper(key)] {
			env = append(env, entry)
		}
	}
	env = append(env, "USERPROFILE="+a.home, "HOME="+a.home, "PATH="+filepath.Dir(node)+string(os.PathListSeparator)+a.path, "OPENCLAW_DISABLE_BONJOUR=1", "NO_COLOR=1")
	if profile != "" {
		if root, err := a.profileRoot(profile); err == nil {
			env = append(env, "OPENCLAW_LOG_DIR="+filepath.Join(root, "logs"))
		}
	}
	return env
}

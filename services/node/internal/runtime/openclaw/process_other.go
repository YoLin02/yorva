//go:build !windows

package openclaw

import (
	"os/exec"
	"sync"
	"syscall"
)

type processJob struct {
	once sync.Once
	pid  int
}

func processRunning(cmd *exec.Cmd) bool             { return syscall.Kill(cmd.Process.Pid, 0) == nil }
func configureProcess(cmd *exec.Cmd)                { cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }
func ownProcess(cmd *exec.Cmd) (*processJob, error) { return &processJob{pid: cmd.Process.Pid}, nil }
func (j *processJob) close()                        { j.once.Do(func() { _ = syscall.Kill(-j.pid, syscall.SIGKILL) }) }
func (j *processJob) terminate()                    { j.close() }
func (j *processJob) finish() error                 { j.close(); return nil }
func (j *processJob) detach() error                 { return errUnsafeTarget }

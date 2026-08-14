//go:build !windows

package hermes

import (
	"os/exec"
	"sync"
	"syscall"
)

func configureProcessTree(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func ownProcessTree(command *exec.Cmd) (func(), error) {
	pid := command.Process.Pid
	var once sync.Once
	return func() {
		once.Do(func() { _ = syscall.Kill(-pid, syscall.SIGKILL) })
	}, nil
}

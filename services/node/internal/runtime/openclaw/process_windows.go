//go:build windows

package openclaw

import (
	"os/exec"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

type processJob struct {
	mu     sync.Mutex
	handle windows.Handle
	closed bool
}

func processRunning(cmd *exec.Cmd) bool {
	handle, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(cmd.Process.Pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(handle)
	state, err := windows.WaitForSingleObject(handle, 0)
	return err == nil && state == uint32(windows.WAIT_TIMEOUT)
}

func configureProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &windows.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.CREATE_SUSPENDED}
}

func ownProcess(cmd *exec.Cmd) (*processJob, error) {
	handle, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, err
	}
	job := &processJob{handle: handle}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(handle, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		job.close()
		return nil, err
	}
	process, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if err != nil {
		job.close()
		return nil, err
	}
	defer windows.CloseHandle(process)
	if err := windows.AssignProcessToJobObject(handle, process); err != nil {
		job.close()
		return nil, err
	}
	if err := resumeProcess(uint32(cmd.Process.Pid)); err != nil {
		job.close()
		return nil, err
	}
	return job, nil
}

func (j *processJob) close() {
	j.mu.Lock()
	defer j.mu.Unlock()
	if !j.closed {
		j.closed = true
		_ = windows.CloseHandle(j.handle)
	}
}

// detach transfers a verified foreground Gateway to Runtime ownership. Until
// this succeeds, cancellation or daemon exit kills the entire startup tree.
func (j *processJob) detach() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.closed {
		return errCommand
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	if _, err := windows.SetInformationJobObject(j.handle, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		return err
	}
	j.closed = true
	return windows.CloseHandle(j.handle)
}

func resumeProcess(pid uint32) error {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, 0)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(snapshot)
	entry := windows.ThreadEntry32{Size: uint32(unsafe.Sizeof(windows.ThreadEntry32{}))}
	if err := windows.Thread32First(snapshot, &entry); err != nil {
		return err
	}
	for {
		if entry.OwnerProcessID == pid {
			thread, err := windows.OpenThread(windows.THREAD_SUSPEND_RESUME, false, entry.ThreadID)
			if err != nil {
				return err
			}
			defer windows.CloseHandle(thread)
			_, err = windows.ResumeThread(thread)
			return err
		}
		if err := windows.Thread32Next(snapshot, &entry); err != nil {
			return err
		}
	}
}

//go:build windows

package openclaw

import (
	"errors"
	"os/exec"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type processJob struct {
	mu          sync.Mutex
	handle      windows.Handle
	closed      bool
	terminating bool
	processes   []windows.Handle
	cleanupErr  error
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
		for _, process := range j.processes {
			_ = windows.CloseHandle(process)
		}
		j.processes = nil
	}
}

func (j *processJob) terminate() {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.closed || j.terminating {
		return
	}
	j.terminating = true
	// Retain handles before requesting termination: a zero Job process count
	// can precede the process objects becoming signaled during final I/O teardown.
	// These IDs come only from our own Job and are never used for PID-based kills.
	var list struct {
		Assigned, Count uint32
		IDs             [1024]uintptr
	}
	if err := windows.QueryInformationJobObject(j.handle, windows.JobObjectBasicProcessIdList,
		uintptr(unsafe.Pointer(&list)), uint32(unsafe.Sizeof(list)), nil); err != nil || list.Count > uint32(len(list.IDs)) || list.Assigned > uint32(len(list.IDs)) {
		j.cleanupErr = errCommand
	} else {
		for _, pid := range list.IDs[:list.Count] {
			process, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
			if err != nil {
				if !errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
					j.cleanupErr = errCommand
				}
				continue // A process that already exited has no handle to join.
			}
			j.processes = append(j.processes, process)
		}
	}
	if windows.TerminateJobObject(j.handle, 1) != nil {
		j.cleanupErr = errCommand
	}
}

// finish runs after cmd.Wait releases the direct process handle. Job accounting
// retains terminated processes while handles remain open, so waiting before
// cmd.Wait would wait on our own reference. Closing a Job alone is asynchronous.
func (j *processJob) finish() error {
	j.terminate()
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.closed {
		return errCommand
	}
	deadline := time.Now().Add(5 * time.Second)
	for _, process := range j.processes {
		remaining := time.Until(deadline).Milliseconds()
		if remaining < 0 {
			remaining = 0
		}
		state, err := windows.WaitForSingleObject(process, uint32(remaining))
		if err != nil || state != windows.WAIT_OBJECT_0 {
			j.cleanupErr = errCommand
		}
		_ = windows.CloseHandle(process)
	}
	j.processes = nil
	if j.cleanupErr != nil {
		return j.cleanupErr
	}
	// JOBOBJECT_BASIC_ACCOUNTING_INFORMATION from the Windows SDK.
	var accounting struct {
		TotalUserTime, TotalKernelTime                     int64
		ThisPeriodTotalUserTime, ThisPeriodTotalKernelTime int64
		TotalPageFaultCount, TotalProcesses                uint32
		ActiveProcesses, TotalTerminatedProcesses          uint32
	}
	for {
		if err := windows.QueryInformationJobObject(j.handle, windows.JobObjectBasicAccountingInformation,
			uintptr(unsafe.Pointer(&accounting)), uint32(unsafe.Sizeof(accounting)), nil); err != nil {
			return errCommand
		}
		if accounting.ActiveProcesses == 0 {
			return nil
		}
		if !time.Now().Before(deadline) {
			return errCommand
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// detach transfers a verified foreground Gateway to Runtime ownership. Until
// this succeeds, cancellation or daemon exit kills the entire startup tree.
func (j *processJob) detach() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.closed || j.terminating {
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

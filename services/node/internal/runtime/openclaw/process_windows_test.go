//go:build windows

package openclaw

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestOwnedJobCompletionSignalsCapturedDescendant(t *testing.T) {
	a, node, entry := contractCLI(t, "tree", "")
	cmd := exec.Command(node, entry, "--version")
	cmd.Env, cmd.Dir = a.environment(node, ""), a.home
	configureProcess(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	job, err := ownProcess(cmd)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatal(err)
	}
	defer func() {
		job.close()
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()
	var pid int
	deadline := time.Now().Add(5 * time.Second)
	for {
		data, err := os.ReadFile(filepath.Join(a.home, "child.pid"))
		if err == nil {
			if _, err := fmt.Sscan(string(data), &pid); err == nil {
				break
			}
		}
		if !time.Now().Before(deadline) {
			t.Fatal("descendant was not launched")
		}
		time.Sleep(5 * time.Millisecond)
	}
	child, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(child)
	var pids struct {
		Assigned, Count uint32
		IDs             [128]uintptr
	}
	queryErr := windows.QueryInformationJobObject(job.handle, windows.JobObjectBasicProcessIdList, uintptr(unsafe.Pointer(&pids)), uint32(unsafe.Sizeof(pids)), nil)
	if queryErr != nil || pids.Count > uint32(len(pids.IDs)) {
		t.Fatalf("owned process inventory failed: count=%d err=%v", pids.Count, queryErr)
	}
	found := false
	for _, observed := range pids.IDs[:pids.Count] {
		found = found || observed == uintptr(pid)
	}
	if !found {
		t.Fatal("descendant did not inherit the owned Job")
	}
	job.terminate()
	_ = cmd.Wait()
	if err := job.finish(); err != nil {
		t.Fatal(err)
	}
	state, err := windows.WaitForSingleObject(child, 0)
	if err != nil || state != windows.WAIT_OBJECT_0 {
		t.Fatalf("captured descendant not terminated at completion: state=%d err=%v", state, err)
	}
}

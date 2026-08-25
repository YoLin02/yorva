//go:build windows

package managedskills

import (
	"errors"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32FindStream   = windows.NewLazySystemDLL("kernel32.dll")
	procFindFirstStreamW = kernel32FindStream.NewProc("FindFirstStreamW")
	procFindNextStreamW  = kernel32FindStream.NewProc("FindNextStreamW")
)

type findStreamData struct {
	StreamSize int64
	StreamName [296]uint16
}

func isReparsePoint(info os.FileInfo) bool {
	data, ok := info.Sys().(*syscall.Win32FileAttributeData)
	return !ok || data.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0
}

func hasAlternateDataStream(path string) (bool, error) {
	pathPointer, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return false, err
	}
	var data findStreamData
	handleValue, _, firstErr := procFindFirstStreamW.Call(
		uintptr(unsafe.Pointer(pathPointer)),
		0,
		uintptr(unsafe.Pointer(&data)),
		0,
	)
	handle := windows.Handle(handleValue)
	if handle == windows.InvalidHandle {
		if errors.Is(firstErr, windows.ERROR_HANDLE_EOF) {
			return false, nil
		}
		return false, firstErr
	}
	defer windows.FindClose(handle)
	for {
		if windows.UTF16ToString(data.StreamName[:]) != "::$DATA" {
			return true, nil
		}
		next, _, nextErr := procFindNextStreamW.Call(
			uintptr(handle),
			uintptr(unsafe.Pointer(&data)),
		)
		if next == 0 {
			if errors.Is(nextErr, windows.ERROR_HANDLE_EOF) {
				return false, nil
			}
			return false, nextErr
		}
	}
}

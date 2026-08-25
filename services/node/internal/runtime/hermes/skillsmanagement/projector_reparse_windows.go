//go:build windows

package skillsmanagement

import (
	"os"
	"syscall"

	"golang.org/x/sys/windows"
)

func isProjectionReparsePoint(info os.FileInfo) bool {
	data, ok := info.Sys().(*syscall.Win32FileAttributeData)
	return !ok || data.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0
}

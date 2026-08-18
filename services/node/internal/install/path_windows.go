//go:build windows

package install

import (
	"io/fs"
	"syscall"
)

const fileAttributeReparsePoint = 0x00000400

func isReparsePoint(info fs.FileInfo) bool {
	data, ok := info.Sys().(*syscall.Win32FileAttributeData)
	return ok && data.FileAttributes&fileAttributeReparsePoint != 0
}

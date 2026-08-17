//go:build windows

package hermes

import (
	"errors"
	"io/fs"
	"syscall"
)

const fileAttributeReparsePoint = 0x00000400

var errReparsePoint = errors.New("path contains a symlink or reparse point")

func isReparsePoint(info fs.FileInfo) bool {
	attrs, ok := fileAttributesOf(info.Sys())
	return ok && attrs&fileAttributeReparsePoint != 0
}

func fileAttributesOf(sys any) (uint32, bool) {
	data, ok := sys.(*syscall.Win32FileAttributeData)
	if !ok {
		return 0, false
	}
	return data.FileAttributes, true
}

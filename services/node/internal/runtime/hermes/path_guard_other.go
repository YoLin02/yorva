//go:build !windows

package hermes

import (
	"errors"
	"io/fs"
)

var errReparsePoint = errors.New("path contains a symlink or reparse point")

func isReparsePoint(info fs.FileInfo) bool {
	return false
}

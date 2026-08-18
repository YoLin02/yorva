//go:build !windows

package install

import "io/fs"

func isReparsePoint(fs.FileInfo) bool {
	return false
}

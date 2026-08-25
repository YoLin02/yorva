//go:build !windows

package skillsmanagement

import "os"

func isProjectionReparsePoint(info os.FileInfo) bool {
	return info.Mode()&os.ModeSymlink != 0
}

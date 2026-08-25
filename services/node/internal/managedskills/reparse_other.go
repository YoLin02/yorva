//go:build !windows

package managedskills

import "os"

func isReparsePoint(info os.FileInfo) bool {
	return info.Mode()&os.ModeSymlink != 0
}

func hasAlternateDataStream(string) (bool, error) { return false, nil }

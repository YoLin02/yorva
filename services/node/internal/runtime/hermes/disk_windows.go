//go:build windows

package hermes

import (
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

func volumeFreeBytes(path string) (uint64, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return 0, err
	}
	volume := filepath.VolumeName(abs)
	if volume == "" {
		volume = abs
	}
	if !strings.HasSuffix(volume, `\`) {
		volume += `\`
	}
	var free, total, totalFree uint64
	if err := windows.GetDiskFreeSpaceEx(windows.StringToUTF16Ptr(volume), &free, &total, &totalFree); err != nil {
		return 0, err
	}
	return free, nil
}

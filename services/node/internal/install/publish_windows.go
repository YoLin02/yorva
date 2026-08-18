//go:build windows

package install

import "golang.org/x/sys/windows"

func renameDirAtomic(from, to string) error {
	src, err := windows.UTF16PtrFromString(from)
	if err != nil {
		return err
	}
	dest, err := windows.UTF16PtrFromString(to)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(src, dest, windows.MOVEFILE_WRITE_THROUGH)
}

//go:build windows

package hermes

import (
	"golang.org/x/sys/windows"
)

// replaceFileAtomic uses MOVEFILE_REPLACE_EXISTING|MOVEFILE_WRITE_THROUGH so
// the replacement is requested to hit stable storage before the API returns.
// The caller still requires SyncDir success; a SyncDir error is not ignored.
func replaceFileAtomic(tmp, dest string) error {
	from, err := windows.UTF16PtrFromString(tmp)
	if err != nil {
		return err
	}
	to, err := windows.UTF16PtrFromString(dest)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(from, to, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}

func syncDirectory(dir string) error {
	handle, err := windows.CreateFile(
		windows.StringToUTF16Ptr(dir),
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS,
		0,
	)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(handle)
	return windows.FlushFileBuffers(handle)
}

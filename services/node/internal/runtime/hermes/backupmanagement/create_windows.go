//go:build windows

package backupmanagement

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

func validateLocalDestinationPath(path string) error {
	volume := filepath.VolumeName(path)
	if len(volume) != 2 || volume[1] != ':' || strings.HasPrefix(path, `\\`) || strings.Contains(path[len(volume):], ":") {
		return verificationError(ErrorDestinationInvalid)
	}
	current := filepath.Dir(path)
	for {
		info, err := os.Lstat(current)
		if err != nil {
			return verificationError(ErrorDestinationUnsafe)
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || windowsReparsePoint(info) {
			return verificationError(ErrorDestinationUnsafe)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
		current = parent
	}
}

func windowsReparsePoint(info fs.FileInfo) bool {
	data, ok := info.Sys().(*syscall.Win32FileAttributeData)
	return ok && data.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0
}

func destinationFreeBytes(path string) (uint64, error) {
	volume := filepath.VolumeName(path)
	if volume == "" {
		return 0, errors.New("destination volume unavailable")
	}
	root := volume + `\`
	var free, total, totalFree uint64
	if err := windows.GetDiskFreeSpaceEx(windows.StringToUTF16Ptr(root), &free, &total, &totalFree); err != nil {
		return 0, err
	}
	return free, nil
}

func createPrivateStaging(path string) (*os.File, error) {
	token, err := windows.OpenCurrentProcessToken()
	if err != nil {
		return nil, err
	}
	defer token.Close()
	user, err := token.GetTokenUser()
	if err != nil {
		return nil, err
	}
	sid := user.User.Sid.String()
	descriptor, err := windows.SecurityDescriptorFromString("D:P(A;;FA;;;" + sid + ")")
	if err != nil {
		return nil, err
	}
	security := &windows.SecurityAttributes{
		Length:             uint32(unsafe.Sizeof(windows.SecurityAttributes{})),
		SecurityDescriptor: descriptor,
	}
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	handle, err := windows.CreateFile(
		name,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		0,
		security,
		windows.CREATE_NEW,
		windows.FILE_ATTRIBUTE_HIDDEN|windows.FILE_ATTRIBUTE_TEMPORARY,
		0,
	)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(handle), path)
	if file == nil {
		_ = windows.CloseHandle(handle)
		return nil, errors.New("private staging handle unavailable")
	}
	return file, nil
}

func openRegularNoFollow(path string) (*os.File, error) {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	handle, err := windows.CreateFile(
		name,
		windows.GENERIC_READ,
		windows.FILE_SHARE_READ,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL|windows.FILE_FLAG_OPEN_REPARSE_POINT,
		0,
	)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(handle), path)
	if file == nil {
		_ = windows.CloseHandle(handle)
		return nil, errors.New("backup file handle unavailable")
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || windowsReparsePoint(info) {
		_ = file.Close()
		return nil, errors.New("backup file is not a regular non-reparse file")
	}
	return file, nil
}

func openDirectoryNoFollow(path string) (*os.File, error) {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	handle, err := windows.CreateFile(
		name,
		windows.GENERIC_READ,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT,
		0,
	)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(handle), path)
	if file == nil {
		_ = windows.CloseHandle(handle)
		return nil, errors.New("backup directory handle unavailable")
	}
	info, err := file.Stat()
	if err != nil || !info.IsDir() || windowsReparsePoint(info) {
		_ = file.Close()
		return nil, errors.New("backup directory is unsafe")
	}
	return file, nil
}

func publishNoReplace(staging, destination string) error {
	if _, err := os.Lstat(destination); err == nil {
		return fs.ErrExist
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	from, err := windows.UTF16PtrFromString(staging)
	if err != nil {
		return err
	}
	to, err := windows.UTF16PtrFromString(destination)
	if err != nil {
		return err
	}
	if err := windows.MoveFileEx(from, to, windows.MOVEFILE_WRITE_THROUGH); err != nil {
		return err
	}
	// Publication is authoritative once the no-replace move succeeds. Clearing
	// visibility attributes is cosmetic and must not turn a published encrypted
	// artifact into an ambiguous failed result.
	attributes, err := windows.GetFileAttributes(to)
	if err != nil {
		return nil
	}
	attributes &^= windows.FILE_ATTRIBUTE_HIDDEN | windows.FILE_ATTRIBUTE_TEMPORARY
	if attributes == 0 {
		attributes = windows.FILE_ATTRIBUTE_NORMAL
	}
	_ = windows.SetFileAttributes(to, attributes)
	return nil
}

func syncDestinationDirectory(path string) error {
	handle, err := windows.CreateFile(
		windows.StringToUTF16Ptr(path),
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

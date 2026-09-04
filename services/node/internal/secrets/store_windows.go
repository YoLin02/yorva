//go:build windows

package secrets

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	secretDirectoryName = "secrets"
	secretVersionName   = "v1"
	secretFileSuffix    = ".dpapi"
	maxProtectedBytes   = 128 * 1024
)

var protectedFileMagic = []byte("YORVA-SECRET-DPAPI-V1\n")

func newStore(root string) (*Store, error) {
	cleanRoot, err := validateRoot(root)
	if err != nil {
		return nil, err
	}
	if err := validateDirectoryChainNoFollow(cleanRoot); err != nil {
		return nil, err
	}

	first := filepath.Join(cleanRoot, secretDirectoryName)
	dir := filepath.Join(first, secretVersionName)
	// A product-identity migration copies the directory tree without copying its
	// Windows ACLs. In that state v1 initially inherits from secrets. Harden the
	// existing child first; otherwise protecting the parent DACL can remove the
	// child's inherited ACEs before YORVA can open and secure it.
	if _, statErr := os.Lstat(dir); statErr == nil {
		if err := validateDirectoryChainNoFollow(dir); err != nil {
			return nil, err
		}
		if err := ensurePrivateDirectory(dir); err != nil {
			return nil, err
		}
	} else if !errors.Is(statErr, fs.ErrNotExist) {
		return nil, ErrUnsafeStorage
	}
	if err := ensurePrivateDirectory(first); err != nil {
		return nil, err
	}
	if err := ensurePrivateDirectory(dir); err != nil {
		return nil, err
	}
	if err := validateDirectoryChainNoFollow(dir); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}

func (s *Store) inspect(ref Reference) (bool, error) {
	if err := validateDirectoryChainNoFollow(s.dir); err != nil {
		return false, err
	}
	file, err := openSecretFile(s.path(ref), windows.GENERIC_READ)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer file.Close()
	header := make([]byte, len(protectedFileMagic))
	if _, err := io.ReadFull(file, header); err != nil || !bytes.Equal(header, protectedFileMagic) {
		return false, ErrCorrupt
	}
	return true, nil
}

func (s *Store) get(ref Reference) ([]byte, error) {
	if err := validateDirectoryChainNoFollow(s.dir); err != nil {
		return nil, err
	}
	file, err := openSecretFile(s.path(ref), windows.GENERIC_READ)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.Size() < int64(len(protectedFileMagic)+1) || info.Size() > maxProtectedBytes {
		return nil, ErrCorrupt
	}
	protected, err := io.ReadAll(io.LimitReader(file, maxProtectedBytes+1))
	if err != nil || len(protected) > maxProtectedBytes || !bytes.HasPrefix(protected, protectedFileMagic) {
		return nil, ErrCorrupt
	}
	ciphertext := protected[len(protectedFileMagic):]
	defer wipe(ciphertext)
	return unprotect(ref, ciphertext)
}

func (s *Store) put(ref Reference, secret []byte) error {
	if err := validateDirectoryChainNoFollow(s.dir); err != nil {
		return err
	}
	protected, err := protect(ref, secret)
	if err != nil {
		return err
	}
	defer wipe(protected)
	if len(protected)+len(protectedFileMagic) > maxProtectedBytes {
		return ErrUnavailable
	}

	temporary, err := s.temporaryPath()
	if err != nil {
		return err
	}
	created := false
	defer func() {
		if created {
			_ = deleteExactFile(temporary)
		}
	}()
	file, err := createPrivateFile(temporary)
	if err != nil {
		return err
	}
	created = true
	writeErr := writeAll(file, protectedFileMagic, protected)
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		return ErrUnavailable
	}
	if err := validateDirectoryChainNoFollow(s.dir); err != nil {
		return err
	}
	if err := moveNoReplace(temporary, s.path(ref)); err != nil {
		return err
	}
	created = false
	return nil
}

func (s *Store) delete(ref Reference) error {
	if err := validateDirectoryChainNoFollow(s.dir); err != nil {
		return err
	}
	return deleteExactFile(s.path(ref))
}

func (s *Store) path(ref Reference) string {
	return filepath.Join(s.dir, string(ref)+secretFileSuffix)
}

func (s *Store) temporaryPath() (string, error) {
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return "", ErrUnavailable
	}
	defer wipe(random)
	return filepath.Join(s.dir, ".tmp-"+hex.EncodeToString(random)), nil
}

func validateRoot(root string) (string, error) {
	if strings.TrimSpace(root) == "" || !filepath.IsAbs(root) {
		return "", ErrInvalidRoot
	}
	clean := filepath.Clean(root)
	volume := filepath.VolumeName(clean)
	if len(volume) != 2 || volume[1] != ':' || strings.HasPrefix(clean, `\\`) || strings.Contains(clean[len(volume):], ":") {
		return "", ErrInvalidRoot
	}
	return clean, nil
}

func ensurePrivateDirectory(path string) error {
	security, err := currentUserSecurityAttributes()
	if err != nil {
		return ErrUnavailable
	}
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return ErrInvalidRoot
	}
	err = windows.CreateDirectory(name, security)
	if err != nil && !errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		return ErrUnavailable
	}
	if err := validateDirectoryNoFollow(path); err != nil {
		return err
	}
	dacl, _, err := security.SecurityDescriptor.DACL()
	if err != nil {
		return ErrUnavailable
	}
	if err := windows.SetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil,
		nil,
		dacl,
		nil,
	); err != nil {
		return ErrUnavailable
	}
	return validateDirectoryNoFollow(path)
}

func currentUserSecurityAttributes() (*windows.SecurityAttributes, error) {
	token, err := windows.OpenCurrentProcessToken()
	if err != nil {
		return nil, err
	}
	defer token.Close()
	user, err := token.GetTokenUser()
	if err != nil {
		return nil, err
	}
	descriptor, err := windows.SecurityDescriptorFromString("D:P(A;;FA;;;" + user.User.Sid.String() + ")")
	if err != nil {
		return nil, err
	}
	return &windows.SecurityAttributes{
		Length:             uint32(unsafe.Sizeof(windows.SecurityAttributes{})),
		SecurityDescriptor: descriptor,
	}, nil
}

func validateDirectoryNoFollow(path string) error {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return ErrUnsafeStorage
	}
	handle, err := windows.CreateFile(
		name,
		windows.FILE_READ_ATTRIBUTES,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT,
		0,
	)
	if err != nil {
		return ErrUnsafeStorage
	}
	file := os.NewFile(uintptr(handle), path)
	if file == nil {
		_ = windows.CloseHandle(handle)
		return ErrUnsafeStorage
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.IsDir() || isReparsePoint(info) {
		return ErrUnsafeStorage
	}
	return nil
}

func validateDirectoryChainNoFollow(path string) error {
	for current := filepath.Clean(path); ; current = filepath.Dir(current) {
		if err := validateDirectoryNoFollow(current); err != nil {
			return err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return nil
		}
	}
}

func createPrivateFile(path string) (*os.File, error) {
	security, err := currentUserSecurityAttributes()
	if err != nil {
		return nil, ErrUnavailable
	}
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, ErrUnavailable
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
		return nil, ErrUnavailable
	}
	file := os.NewFile(uintptr(handle), path)
	if file == nil {
		_ = windows.CloseHandle(handle)
		return nil, ErrUnavailable
	}
	return file, nil
}

func openSecretFile(path string, access uint32) (*os.File, error) {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, ErrUnsafeStorage
	}
	handle, err := windows.CreateFile(
		name,
		access|windows.FILE_READ_ATTRIBUTES,
		windows.FILE_SHARE_READ,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL|windows.FILE_FLAG_OPEN_REPARSE_POINT,
		0,
	)
	if errors.Is(err, windows.ERROR_FILE_NOT_FOUND) || errors.Is(err, windows.ERROR_PATH_NOT_FOUND) {
		return nil, fs.ErrNotExist
	}
	if err != nil {
		return nil, ErrUnavailable
	}
	file := os.NewFile(uintptr(handle), path)
	if file == nil {
		_ = windows.CloseHandle(handle)
		return nil, ErrUnavailable
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || isReparsePoint(info) {
		_ = file.Close()
		return nil, ErrUnsafeStorage
	}
	return file, nil
}

func isReparsePoint(info fs.FileInfo) bool {
	data, ok := info.Sys().(*syscall.Win32FileAttributeData)
	return ok && data.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0
}

func writeAll(file *os.File, parts ...[]byte) error {
	for _, part := range parts {
		if _, err := file.Write(part); err != nil {
			return err
		}
	}
	return file.Sync()
}

func moveNoReplace(from, to string) error {
	fromPtr, err := windows.UTF16PtrFromString(from)
	if err != nil {
		return ErrUnavailable
	}
	toPtr, err := windows.UTF16PtrFromString(to)
	if err != nil {
		return ErrUnavailable
	}
	if err := windows.MoveFileEx(fromPtr, toPtr, windows.MOVEFILE_WRITE_THROUGH); err != nil {
		if errors.Is(err, windows.ERROR_ALREADY_EXISTS) || errors.Is(err, windows.ERROR_FILE_EXISTS) {
			return ErrAlreadyExists
		}
		return ErrUnavailable
	}
	return nil
}

func deleteExactFile(path string) error {
	file, err := openSecretFile(path, windows.DELETE)
	if errors.Is(err, fs.ErrNotExist) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	handle := windows.Handle(file.Fd())
	dispose := byte(1)
	err = windows.SetFileInformationByHandle(
		handle,
		windows.FileDispositionInfo,
		&dispose,
		uint32(unsafe.Sizeof(dispose)),
	)
	closeErr := file.Close()
	if err != nil || closeErr != nil {
		return ErrUnavailable
	}
	return nil
}

func protect(ref Reference, plaintext []byte) ([]byte, error) {
	entropy := referenceEntropy(ref)
	input := dataBlob(plaintext)
	entropyBlob := dataBlob(entropy[:])
	name, err := windows.UTF16PtrFromString("YORVA protected secret")
	if err != nil {
		return nil, ErrUnavailable
	}
	var output windows.DataBlob
	if err := windows.CryptProtectData(&input, name, &entropyBlob, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &output); err != nil {
		return nil, ErrUnavailable
	}
	return copyAndFreeBlob(&output, maxProtectedBytes)
}

func unprotect(ref Reference, ciphertext []byte) ([]byte, error) {
	entropy := referenceEntropy(ref)
	input := dataBlob(ciphertext)
	entropyBlob := dataBlob(entropy[:])
	var output windows.DataBlob
	var description *uint16
	if err := windows.CryptUnprotectData(&input, &description, &entropyBlob, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &output); err != nil {
		return nil, ErrCorrupt
	}
	if description != nil {
		defer windows.LocalFree(windows.Handle(unsafe.Pointer(description)))
	}
	plaintext, err := copyAndFreeBlob(&output, maxSecretBytes)
	if err != nil || len(plaintext) == 0 {
		wipe(plaintext)
		return nil, ErrCorrupt
	}
	return plaintext, nil
}

func dataBlob(value []byte) windows.DataBlob {
	if len(value) == 0 {
		return windows.DataBlob{}
	}
	return windows.DataBlob{Size: uint32(len(value)), Data: &value[0]}
}

func copyAndFreeBlob(blob *windows.DataBlob, limit int) ([]byte, error) {
	if blob == nil || blob.Data == nil || blob.Size == 0 || uint64(blob.Size) > maxProtectedBytes {
		if blob != nil && blob.Data != nil {
			_, _ = windows.LocalFree(windows.Handle(unsafe.Pointer(blob.Data)))
		}
		return nil, ErrUnavailable
	}
	allocated := unsafe.Slice(blob.Data, int(blob.Size))
	if uint64(blob.Size) > uint64(limit) {
		wipe(allocated)
		_, _ = windows.LocalFree(windows.Handle(unsafe.Pointer(blob.Data)))
		blob.Data = nil
		blob.Size = 0
		return nil, ErrUnavailable
	}
	value := append([]byte(nil), allocated...)
	wipe(allocated)
	_, freeErr := windows.LocalFree(windows.Handle(unsafe.Pointer(blob.Data)))
	if freeErr != nil {
		wipe(value)
		return nil, ErrUnavailable
	}
	blob.Data = nil
	blob.Size = 0
	return value, nil
}

func referenceEntropy(ref Reference) [sha256.Size]byte {
	return sha256.Sum256([]byte("yorva-secretstore-dpapi-v1\x00" + string(ref)))
}

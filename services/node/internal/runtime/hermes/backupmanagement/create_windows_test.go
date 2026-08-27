//go:build windows

package backupmanagement

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestCreatePrivateStagingUsesProtectedCurrentUserOnlyDACL(t *testing.T) {
	const fileAllAccess windows.ACCESS_MASK = 0x001f01ff
	path := filepath.Join(t.TempDir(), stagingFilePrefix+"acl.tmp")
	file, err := createPrivateStaging(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)

	descriptor, err := windows.GetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.OWNER_SECURITY_INFORMATION,
	)
	if err != nil {
		t.Fatal(err)
	}
	control, _, err := descriptor.Control()
	if err != nil {
		t.Fatal(err)
	}
	if control&windows.SE_DACL_PROTECTED == 0 {
		t.Fatalf("staging DACL control = %#x, want protected", control)
	}
	dacl, _, err := descriptor.DACL()
	if err != nil {
		t.Fatal(err)
	}
	if dacl == nil || dacl.AceCount != 1 {
		t.Fatalf("staging DACL ACE count = %v", dacl)
	}
	var ace *windows.ACCESS_ALLOWED_ACE
	if err := windows.GetAce(dacl, 0, &ace); err != nil {
		t.Fatal(err)
	}
	if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE || ace.Mask&fileAllAccess != fileAllAccess {
		t.Fatalf("staging ACE type/mask = %d/%#x", ace.Header.AceType, ace.Mask)
	}
	token, err := windows.OpenCurrentProcessToken()
	if err != nil {
		t.Fatal(err)
	}
	defer token.Close()
	user, err := token.GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	aceSID := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
	if !aceSID.Equals(user.User.Sid) {
		t.Fatalf("staging DACL does not grant only the current process user")
	}
}

func TestOpenSnapshotSourceClassifiesExclusiveRuntimeHandle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	if err := os.WriteFile(path, []byte("database"), 0o600); err != nil {
		t.Fatal(err)
	}
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := windows.CreateFile(name, windows.GENERIC_WRITE, 0, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(handle)

	_, err = openSnapshotSourceFile(path)
	var classified *VerificationError
	if !errors.As(err, &classified) || classified.Code() != ErrorSourceRuntimeLive {
		t.Fatalf("exclusive Runtime handle error = %v", err)
	}
}

func TestBuildSnapshotPreservesExclusiveRuntimeClassification(t *testing.T) {
	root := filepath.Join(t.TempDir(), "hermes")
	path := writeSnapshotFixture(t, root, "state.db", "database")
	payload, container := openSnapshotStaging(t)
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := windows.CreateFile(name, windows.GENERIC_WRITE, 0, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(handle)

	_, err = buildHermesRuntimeSnapshot(context.Background(), root, testSnapshotDescriptor(), true, SnapshotStaging{Payload: payload, Container: container}, DefaultLimits())
	assertSnapshotErrorCode(t, err, ErrorSourceRuntimeLive)
}

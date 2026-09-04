//go:build windows

package secrets

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

func TestWindowsStoreLifecycleAndCiphertextBoundary(t *testing.T) {
	store, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	ref := Reference("backup-device-a1")
	secret := []byte("AGE-SECRET-KEY-1TEST-PLAINTEXT")
	if err := store.Put(ctx, ref, secret); err != nil {
		t.Fatal(err)
	}

	metadata, err := store.Inspect(ctx, ref)
	if err != nil {
		t.Fatal(err)
	}
	if !metadata.Configured || metadata.Reference != ref {
		t.Fatalf("metadata = %#v", metadata)
	}
	protected, err := os.ReadFile(store.path(ref))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(protected, secret) {
		t.Fatal("protected file contains plaintext")
	}
	if !bytes.HasPrefix(protected, protectedFileMagic) {
		t.Fatal("protected file lacks version marker")
	}

	var observed []byte
	if err := store.Get(ctx, ref, func(value []byte) error {
		observed = append([]byte(nil), value...)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(observed, secret) {
		t.Fatalf("secret = %q", observed)
	}

	if err := store.Put(ctx, ref, []byte("replacement")); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("second put error = %v", err)
	}
	if err := store.Delete(ctx, ref); err != nil {
		t.Fatal(err)
	}
	metadata, err = store.Inspect(ctx, ref)
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Configured {
		t.Fatal("deleted secret remains configured")
	}
	if err := store.Get(ctx, ref, func([]byte) error { return nil }); !errors.Is(err, ErrNotFound) {
		t.Fatalf("get deleted error = %v", err)
	}
}

func TestWindowsStoreGenerateRotateRetainsOldIdentity(t *testing.T) {
	store, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	oldRef := Reference("backup-device-old")
	newRef := Reference("backup-device-new")
	if err := store.Generate(ctx, oldRef, func() ([]byte, error) {
		return []byte("old-identity"), nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Rotate(ctx, oldRef, newRef, func() ([]byte, error) {
		return []byte("new-identity"), nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, ref := range []Reference{oldRef, newRef} {
		metadata, err := store.Inspect(ctx, ref)
		if err != nil {
			t.Fatal(err)
		}
		if !metadata.Configured {
			t.Fatalf("%s not retained", ref)
		}
	}
	if err := store.Rotate(ctx, oldRef, newRef, func() ([]byte, error) {
		return []byte("must-not-replace"), nil
	}); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("rotate overwrite error = %v", err)
	}
}

func TestWindowsStoreBindsCiphertextToReferenceAndRejectsTamper(t *testing.T) {
	store, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	first := Reference("backup-device-first")
	second := Reference("backup-device-second")
	if err := store.Put(ctx, first, []byte("identity-material")); err != nil {
		t.Fatal(err)
	}
	protected, err := os.ReadFile(store.path(first))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.path(second), protected, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := store.Get(ctx, second, func([]byte) error { return nil }); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("swapped reference error = %v", err)
	}

	protected[len(protected)-1] ^= 0xff
	if err := os.WriteFile(store.path(first), protected, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := store.Get(ctx, first, func([]byte) error { return nil }); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("tamper error = %v", err)
	}
}

func TestWindowsStoreRejectsReparseFileAndCanceledOperation(t *testing.T) {
	store, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(target, []byte("not-a-secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	ref := Reference("backup-device-link")
	if err := os.Symlink(target, store.path(ref)); err != nil {
		t.Skipf("creating a Windows symlink requires host permission: %v", err)
	}
	if _, err := store.Inspect(context.Background(), ref); !errors.Is(err, ErrUnsafeStorage) {
		t.Fatalf("reparse file error = %v", err)
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := store.Put(canceled, Reference("backup-device-canceled"), []byte("secret")); !errors.Is(err, ErrCanceled) {
		t.Fatalf("canceled put error = %v", err)
	}
}

func TestWindowsStoreRejectsReparseAncestor(t *testing.T) {
	targetParent := t.TempDir()
	if err := os.Mkdir(filepath.Join(targetParent, "data"), 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "redirect")
	if err := os.Symlink(targetParent, link); err != nil {
		t.Skipf("creating a Windows directory symlink requires host permission: %v", err)
	}
	if _, err := New(filepath.Join(link, "data")); !errors.Is(err, ErrUnsafeStorage) {
		t.Fatalf("reparse ancestor error = %v", err)
	}
}

func TestWindowsStoreRejectsAncestorReplacedAfterOpen(t *testing.T) {
	root := t.TempDir()
	store, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	original := filepath.Join(root, secretDirectoryName)
	moved := filepath.Join(root, "secrets-moved")
	if err := os.Rename(original, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(moved, original); err != nil {
		t.Skipf("creating a Windows directory symlink requires host permission: %v", err)
	}
	if _, err := store.Inspect(context.Background(), Reference("backup-device-after-open")); !errors.Is(err, ErrUnsafeStorage) {
		t.Fatalf("replaced ancestor error = %v", err)
	}
}

func TestWindowsStoreAppliesProtectedCurrentUserDACL(t *testing.T) {
	store, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ref := Reference("backup-device-acl")
	if err := store.Put(context.Background(), ref, []byte("identity")); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{store.dir, store.path(ref)} {
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
			t.Fatalf("DACL for %s is inheritable", filepath.Base(path))
		}
		dacl, _, err := descriptor.DACL()
		if err != nil {
			t.Fatal(err)
		}
		if dacl.AceCount != 1 {
			t.Fatalf("DACL for %s has %d ACEs, want 1", filepath.Base(path), dacl.AceCount)
		}
	}
}

func TestWindowsStoreHardensPreexistingInheritedMigrationTreeFromChildFirst(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, secretDirectoryName, secretVersionName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	store, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	if store.dir != dir {
		t.Fatalf("store dir = %q", store.dir)
	}
	for _, path := range []string{filepath.Dir(dir), dir} {
		descriptor, err := windows.GetNamedSecurityInfo(
			path,
			windows.SE_FILE_OBJECT,
			windows.DACL_SECURITY_INFORMATION,
		)
		if err != nil {
			t.Fatal(err)
		}
		control, _, err := descriptor.Control()
		if err != nil {
			t.Fatal(err)
		}
		if control&windows.SE_DACL_PROTECTED == 0 {
			t.Fatalf("DACL for %s is not protected", filepath.Base(path))
		}
	}
}

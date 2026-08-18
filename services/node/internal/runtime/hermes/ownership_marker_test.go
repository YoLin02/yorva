package hermes

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteOwnershipRecordIsAtomicAndPreservesOldProof(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "owned.txt"), []byte("payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	identity := testIdentity("op_atom", root)
	if err := writeOwnershipRecord(root, identity); err != nil {
		t.Fatal(err)
	}
	old, err := os.ReadFile(filepath.Join(root, yorvaPartialMarker))
	if err != nil {
		t.Fatal(err)
	}

	fail := errors.New("injected")
	cases := []struct {
		name string
		ops  atomicFileOps
	}{
		{name: "temp create failure", ops: withAtomicOverride(func(ops *atomicFileOps) {
			ops.CreateExclusive = func(string) (*os.File, error) { return nil, fail }
		})},
		{name: "partial write", ops: withAtomicOverride(func(ops *atomicFileOps) {
			ops.Write = func(*os.File, []byte) error { return fail }
		})},
		{name: "sync failure", ops: withAtomicOverride(func(ops *atomicFileOps) {
			ops.Sync = func(*os.File) error { return fail }
		})},
		{name: "close failure", ops: withAtomicOverride(func(ops *atomicFileOps) {
			ops.Close = func(*os.File) error { return fail }
		})},
		{name: "atomic replace failure", ops: withAtomicOverride(func(ops *atomicFileOps) {
			ops.Replace = func(string, string) error { return fail }
		})},
		{name: "directory sync before replace", ops: withAtomicOverride(func(ops *atomicFileOps) {
			ops.SyncDir = func(string) error { return fail }
		})},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := writeOwnershipRecordWith(tc.ops, root, identity); err == nil {
				t.Fatal("injected failure succeeded")
			}
			got, err := os.ReadFile(filepath.Join(root, yorvaPartialMarker))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(old) {
				t.Fatal("old complete marker was not preserved")
			}
			if err := requireCurrentOwnedTree(root, identity); err != nil {
				t.Fatalf("old proof no longer validates: %v", err)
			}
			for _, entry := range mustReadDir(t, root) {
				if len(entry.Name()) >= 12 && entry.Name()[:12] == ".yorva-atom-" {
					info, _ := os.Stat(filepath.Join(root, entry.Name()))
					if info != nil && info.Size() > 0 && info.Size() < 10 {
						t.Fatalf("partial marker left behind: %s", entry.Name())
					}
				}
			}
		})
	}
}

func withAtomicOverride(mutate func(*atomicFileOps)) atomicFileOps {
	ops := defaultAtomicFileOps()
	mutate(&ops)
	return ops
}

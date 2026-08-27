package managedskills

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportToManagedConsumesNativeStagingAndPublishesImmutableSource(t *testing.T) {
	dataDir := t.TempDir()
	store, err := NewStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	ref := strings.Repeat("a", 43)
	source := filepath.Join(dataDir, "skill-imports", ref, "directory")
	if err := os.MkdirAll(source, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, SkillFileName), []byte("---\nname: local-skill\ndescription: Local test Skill.\nversion: 1.0.0\nauthor: Test\nlicense: MIT\nplatforms: [windows, linux, macos]\n---\n\n# Local Skill\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := store.ImportToManaged(context.Background(), "inst_1", "local-skill", ref)
	if err != nil || got.SourceID != LocalImportSourceID || got.Version != "1.0.0" {
		t.Fatalf("ImportToManaged() = %#v, %v", got, err)
	}
	if _, err := InspectSourceDirectory(got.AbsolutePath); err != nil {
		t.Fatalf("managed source = %v", err)
	}
	if _, err := os.Lstat(filepath.Join(dataDir, "skill-imports", ref)); !os.IsNotExist(err) {
		t.Fatalf("native staging was not consumed: %v", err)
	}
}

func TestImportToManagedExtractsSingleRootZIPAndRejectsTraversal(t *testing.T) {
	dataDir := t.TempDir()
	store, err := NewStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	writeImportZIP := func(ref, name string) {
		root := filepath.Join(dataDir, "skill-imports", ref)
		if err := os.MkdirAll(root, 0o700); err != nil {
			t.Fatal(err)
		}
		file, err := os.Create(filepath.Join(root, "source.zip"))
		if err != nil {
			t.Fatal(err)
		}
		writer := zip.NewWriter(file)
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte("---\nname: zip-skill\ndescription: ZIP test Skill.\nversion: 1.0.0\nauthor: Test\nlicense: MIT\nplatforms: [windows, linux, macos]\n---\n\n# ZIP Skill\n")); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}
	validRef := strings.Repeat("b", 43)
	writeImportZIP(validRef, "bundle/SKILL.md")
	if _, err := store.ImportToManaged(context.Background(), "inst_1", "zip-skill", validRef); err != nil {
		t.Fatalf("valid ZIP import: %v", err)
	}
	unsafeRef := strings.Repeat("c", 43)
	writeImportZIP(unsafeRef, "../SKILL.md")
	if _, err := store.ImportToManaged(context.Background(), "inst_1", "unsafe-skill", unsafeRef); err == nil {
		t.Fatal("traversal ZIP import succeeded")
	}
}

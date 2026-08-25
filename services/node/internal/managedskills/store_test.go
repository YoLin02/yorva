package managedskills

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestBuiltInCatalogAcquireAndHash(t *testing.T) {
	catalog := ListCatalog()
	if len(catalog) != 1 {
		t.Fatalf("catalog length = %d, want 1", len(catalog))
	}
	entry := catalog[0]
	if entry.SourceID != "yorva-demo" || entry.SkillID != "yorva-managed-demo" || entry.Version != "1.0.0" {
		t.Fatalf("unexpected catalog entry: %#v", entry)
	}
	dataDir := t.TempDir()
	store, err := NewStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	acquired, err := store.AcquireToManaged(context.Background(), "instance-1", entry.SkillID, entry.SourceID)
	if err != nil {
		t.Fatal(err)
	}
	wantRoot := filepath.Join(dataDir, "skills")
	if store.Root() != wantRoot || acquired.AbsolutePath != filepath.Join(wantRoot, acquired.RelativePath) {
		t.Fatalf("managed paths = root %q, acquisition %#v", store.Root(), acquired)
	}
	if filepath.Base(acquired.AbsolutePath) != acquired.ContentSHA256 || len(acquired.ContentSHA256) != 64 {
		t.Fatalf("digest path = %#v", acquired)
	}
	bundle, err := InspectSourceDirectory(acquired.AbsolutePath)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.ContentSHA256 != acquired.ContentSHA256 {
		t.Fatalf("bundle digest = %q, acquisition = %q", bundle.ContentSHA256, acquired.ContentSHA256)
	}
	again, err := store.AcquireToManaged(context.Background(), "instance-1", entry.SkillID, entry.SourceID)
	if err != nil || again != acquired {
		t.Fatalf("idempotent acquisition = %#v, %v", again, err)
	}
}

func TestInjectedCatalogVersionsRemainImmutable(t *testing.T) {
	root := filepath.Join(t.TempDir(), "skills")
	v1 := testCatalogSource("1.0.0", "First reviewed body.")
	v2 := testCatalogSource("2.0.0", "Second reviewed body.")
	first, err := newStore(root, []catalogSource{v1}).AcquireToManaged(context.Background(), "instance-1", "demo", "reviewed")
	if err != nil {
		t.Fatal(err)
	}
	second, err := newStore(root, []catalogSource{v2}).AcquireToManaged(context.Background(), "instance-1", "demo", "reviewed")
	if err != nil {
		t.Fatal(err)
	}
	if first.ContentSHA256 == second.ContentSHA256 || first.AbsolutePath == second.AbsolutePath {
		t.Fatalf("versions reused immutable path: first %#v, second %#v", first, second)
	}
	if _, err := os.Stat(first.AbsolutePath); err != nil {
		t.Fatalf("v1 disappeared after v2 acquisition: %v", err)
	}
}

func TestCatalogStrictValidation(t *testing.T) {
	tests := []struct {
		name  string
		files fstest.MapFS
	}{
		{
			name: "unknown frontmatter",
			files: testCatalogFS(`---
name: demo
description: reviewed
version: 1.0.0
author: YORVA
license: MIT
platforms: [windows, linux, macos]
unexpected: rejected
---
# Demo
`),
		},
		{
			name: "executable surface",
			files: func() fstest.MapFS {
				files := testCatalogFS(validTestSkill("1.0.0", "Reviewed."))
				files["source/scripts/run.ps1"] = &fstest.MapFile{Data: []byte("Write-Host unsafe")}
				return files
			}(),
		},
		{
			name: "invalid UTF-8",
			files: func() fstest.MapFS {
				files := testCatalogFS(validTestSkill("1.0.0", "Reviewed."))
				files["source/references/bad.md"] = &fstest.MapFile{Data: []byte{0xff}}
				return files
			}(),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := catalogSource{descriptor: CatalogEntry{SourceID: "reviewed", SkillID: "demo", Version: "1.0.0"}, filesystem: test.files, directory: "source"}
			_, _, err := validateCatalogSource(source)
			if !errors.Is(err, ErrCatalogEntryInvalid) {
				t.Fatalf("error = %v, want ErrCatalogEntryInvalid", err)
			}
		})
	}
}

func TestInspectSourceDirectoryRejectsTraversalShapedNamesAndCaseFold(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, SkillFileName), validTestSkill("1.0.0", "Reviewed."))
	if err := os.Mkdir(filepath.Join(root, "references"), 0o700); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(root, "references", "readme.md"), "one")
	if err := os.WriteFile(filepath.Join(root, "references", "bad:name.md"), []byte("two"), 0o600); err == nil {
		if _, inspectErr := InspectSourceDirectory(root); !errors.Is(inspectErr, ErrBundleInvalid) {
			t.Fatalf("ADS-shaped name error = %v", inspectErr)
		}
	}
}

func TestClosedRelativePathValidationRejectsTraversalADSAndCaseFold(t *testing.T) {
	for _, candidate := range []string{"../SKILL.md", "references/../../escape.md", "references/bad:name.md"} {
		if validCatalogRelative(candidate) || safeRelativePath(filepath.FromSlash(candidate)) {
			t.Fatalf("unsafe relative path %q was accepted", candidate)
		}
	}
	files := testCatalogFS(validTestSkill("1.0.0", "Reviewed."))
	files["source/references/INFO.md"] = &fstest.MapFile{Data: []byte("duplicate case")}
	source := catalogSource{descriptor: CatalogEntry{SourceID: "reviewed", SkillID: "demo", Version: "1.0.0"}, filesystem: files, directory: "source"}
	if _, _, err := validateCatalogSource(source); !errors.Is(err, ErrCatalogEntryInvalid) {
		t.Fatalf("case-fold catalog collision error = %v", err)
	}
}

func testCatalogSource(version, body string) catalogSource {
	return catalogSource{
		descriptor: CatalogEntry{SourceID: "reviewed", SkillID: "demo", Version: version, Description: "test"},
		filesystem: testCatalogFS(validTestSkill(version, body)),
		directory:  "source",
	}
}

func testCatalogFS(skill string) fstest.MapFS {
	return fstest.MapFS{
		"source/SKILL.md":           &fstest.MapFile{Data: []byte(skill), Mode: 0o600},
		"source/references/info.md": &fstest.MapFile{Data: []byte("# Reference\n\nReviewed prose.\n"), Mode: 0o600},
	}
}

func validTestSkill(version, body string) string {
	return "---\nname: demo\ndescription: reviewed\nversion: " + version + "\nauthor: YORVA\nlicense: MIT\nplatforms: [windows, linux, macos]\n---\n# Demo\n\n" + body + "\n"
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), fs.FileMode(0o600)); err != nil {
		t.Fatal(err)
	}
}

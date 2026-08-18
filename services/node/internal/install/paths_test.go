package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultManagedRoot(t *testing.T) {
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		t.Skip("LOCALAPPDATA unset")
	}
	root, err := DefaultManagedRoot()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(local, "hermes")
	if root != want {
		t.Fatalf("got %q want %q", root, want)
	}
}

func TestPathContainmentRejectsEscape(t *testing.T) {
	layout, err := NewLayout(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	txnID := "txn_" + strings.Repeat("a", 22)
	genID := "gen_" + strings.Repeat("b", 22)
	if err := ValidateStagingRel(StagingRel(txnID), txnID); err != nil {
		t.Fatal(err)
	}
	if err := ValidateGenerationRel(GenerationRel(genID), genID); err != nil {
		t.Fatal(err)
	}
	escaping := []string{
		`C:\Windows\hermes`,
		`/etc/passwd`,
		`\\server\share\hermes`,
		`generations/../staging/` + txnID,
		`generations/` + genID + `/..`,
		`generations\` + genID,
		`staging/` + txnID + `/..`,
		`../generations/` + genID,
	}
	for _, rel := range escaping {
		if _, err := layout.ResolveContained(rel); err == nil {
			t.Fatalf("accepted escaping path %q", rel)
		}
	}
	notExact := append(append([]string{}, escaping...), `generations/`+genID+`/extra`)
	for _, rel := range notExact {
		if ValidateGenerationRel(rel, genID) == nil {
			t.Fatalf("accepted generation rel %q", rel)
		}
	}
	if _, err := layout.ResolveContained(GenerationRel(genID)); err != nil {
		t.Fatal(err)
	}
	resolved, err := layout.ResolveContained(GenerationRel(genID))
	if err != nil {
		t.Fatal(err)
	}
	if !pathContained(layout.Root, resolved) {
		t.Fatal("resolved generation not contained")
	}
	if ValidateGenerationRel(GenerationRel(genID), "gen_"+strings.Repeat("c", 22)) == nil {
		t.Fatal("mismatched generation id accepted")
	}
}

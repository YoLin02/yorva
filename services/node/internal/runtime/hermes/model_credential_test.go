package hermes

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const credentialSentinel = "sk-phase5-sentinel-DO-NOT-LEAK"

func TestModelCredentialSetReplaceDeletePreservesUnknownContent(t *testing.T) {
	store, defaultEnv := newTestCredentialStore(t)
	original := []byte("# owned by Hermes\r\nOTHER=value\r\nexport DEEPSEEK_API_KEY='old-key'\r\nTAIL=keep # exact\r\n")
	writeTestCredentialFile(t, defaultEnv, original)

	status, err := store.Status("default", "deepseek")
	if err != nil || !status.Configured {
		t.Fatalf("initial status = %#v, %v", status, err)
	}
	status, err = store.Set("default", "deepseek", []byte(credentialSentinel))
	if err != nil || !status.Configured {
		t.Fatalf("set status = %#v, %v", status, err)
	}
	got := readTestCredentialFile(t, defaultEnv)
	want := []byte("# owned by Hermes\r\nOTHER=value\r\nDEEPSEEK_API_KEY=\"" + credentialSentinel + "\"\r\nTAIL=keep # exact\r\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("set content differs\ngot:  %q\nwant: %q", got, want)
	}
	if leftovers, _ := filepath.Glob(filepath.Join(filepath.Dir(defaultEnv), ".yorva-model-credential-*")); len(leftovers) != 0 {
		t.Fatalf("temporary files remain: %v", leftovers)
	}

	status, err = store.Delete("default", "deepseek")
	if err != nil || status.Configured {
		t.Fatalf("delete status = %#v, %v", status, err)
	}
	got = readTestCredentialFile(t, defaultEnv)
	want = []byte("# owned by Hermes\r\nOTHER=value\r\nTAIL=keep # exact\r\n")
	if !bytes.Equal(got, want) {
		t.Fatalf("delete content differs\ngot:  %q\nwant: %q", got, want)
	}
}

func TestModelCredentialProfileIsolationAndRestartTruth(t *testing.T) {
	store, _ := newTestCredentialStore(t)
	profilesRoot := filepath.Join(store.root, "profiles")
	for _, name := range []string{"alpha", "bravo"} {
		if err := os.MkdirAll(filepath.Join(profilesRoot, name), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.Set("alpha", "kimi", []byte("alpha-secret")); err != nil {
		t.Fatal(err)
	}
	alpha, err := store.Status("alpha", "kimi")
	if err != nil || !alpha.Configured {
		t.Fatalf("alpha status = %#v, %v", alpha, err)
	}
	bravo, err := store.Status("bravo", "kimi")
	if err != nil || bravo.Configured {
		t.Fatalf("bravo status = %#v, %v", bravo, err)
	}

	restarted := credentialStore{root: store.root}
	alpha, err = restarted.Status("alpha", "kimi")
	if err != nil || !alpha.Configured {
		t.Fatalf("restart status = %#v, %v", alpha, err)
	}
	if data, err := os.ReadFile(filepath.Join(profilesRoot, "bravo", ".env")); err == nil || len(data) != 0 {
		t.Fatalf("bravo credential file unexpectedly exists: %q, %v", data, err)
	}
}

func TestModelCredentialOptimisticConflictPreservesExternalWrite(t *testing.T) {
	store, path := newTestCredentialStore(t)
	writeTestCredentialFile(t, path, []byte("OTHER=before\n"))
	store.beforeReplace = func() {
		writeTestCredentialFile(t, path, []byte("OTHER=external\n"))
	}
	_, err := store.Set("default", "deepseek", []byte(credentialSentinel))
	if !errors.Is(err, errModelCredentialConflict) {
		t.Fatalf("set error = %v", err)
	}
	if got := readTestCredentialFile(t, path); string(got) != "OTHER=external\n" {
		t.Fatalf("external content overwritten: %q", got)
	}
	if leftovers, _ := filepath.Glob(filepath.Join(filepath.Dir(path), ".yorva-model-credential-*")); len(leftovers) != 0 {
		t.Fatalf("temporary files remain: %v", leftovers)
	}
	if strings.Contains(err.Error(), credentialSentinel) {
		t.Fatal("secret leaked through conflict error")
	}
}

func TestModelCredentialFailsClosedOnUnsafeInputAndState(t *testing.T) {
	store, path := newTestCredentialStore(t)
	tests := [][]byte{nil, {}, []byte("abc"), []byte(" leading"), []byte("trailing "), []byte("line\nbreak"), []byte("your-api-key"), bytes.Repeat([]byte{'x'}, modelCredentialMaxValue+1)}
	for _, secret := range tests {
		if _, err := store.Set("default", "deepseek", secret); !IsModelCredentialInvalid(err) {
			t.Fatalf("secret length %d error = %v", len(secret), err)
		}
	}
	writeTestCredentialFile(t, path, []byte("DEEPSEEK_API_KEY=one\nexport DEEPSEEK_API_KEY=two\n"))
	if _, err := store.Status("default", "deepseek"); !IsModelCredentialUnsafe(err) {
		t.Fatalf("duplicate status error = %v", err)
	}
	if _, err := store.Delete("default", "deepseek"); !IsModelCredentialUnsafe(err) {
		t.Fatalf("duplicate delete error = %v", err)
	}
	if _, err := store.Set("../escape", "deepseek", []byte("valid")); !IsModelCredentialUnsafe(err) {
		t.Fatalf("traversal error = %v", err)
	}
	if _, err := store.Set("default", "UNKNOWN_ENV", []byte("valid")); !IsModelProviderUnsupported(err) {
		t.Fatalf("unknown preset error = %v", err)
	}
}

func TestModelCredentialProductionBoundaryIsPinned(t *testing.T) {
	if _, err := ModelCredentialStatusFor("0.20.1", "default", "deepseek"); !IsModelVersionUnsupported(err) {
		t.Fatalf("version error = %v", err)
	}
	if _, err := SetModelCredential("0.21.0", "default", "deepseek", []byte("valid")); !IsModelVersionUnsupported(err) {
		t.Fatalf("set version error = %v", err)
	}
	if _, err := DeleteModelCredential("0.19.0", "default", "deepseek"); !IsModelVersionUnsupported(err) {
		t.Fatalf("delete version error = %v", err)
	}
}

func TestModelCredentialRejectsNonRegularAndOversizedFiles(t *testing.T) {
	store, path := newTestCredentialStore(t)
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Status("default", "deepseek"); !IsModelCredentialUnsafe(err) {
		t.Fatalf("directory status error = %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	writeTestCredentialFile(t, path, bytes.Repeat([]byte{'x'}, modelCredentialMaxFile+1))
	if _, err := store.Status("default", "deepseek"); !IsModelCredentialUnsafe(err) {
		t.Fatalf("oversized status error = %v", err)
	}
}

func TestModelCredentialRejectsProfileSymlink(t *testing.T) {
	store, _ := newTestCredentialStore(t)
	target := t.TempDir()
	link := filepath.Join(store.root, "profiles", "linked")
	if err := os.MkdirAll(filepath.Dir(link), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("symlink creation not permitted: %v", err)
		}
		t.Fatal(err)
	}
	if _, err := store.Status("linked", "deepseek"); !IsModelCredentialUnsafe(err) {
		t.Fatalf("symlink status error = %v", err)
	}
}

func TestCredentialParserHandlesBOMQuotesAndAppend(t *testing.T) {
	data := []byte("\xef\xbb\xbfOTHER=1\nANTHROPIC_API_KEY=\"\"\n")
	configured, err := credentialConfigured(data, "ANTHROPIC_API_KEY")
	if err != nil || configured {
		t.Fatalf("empty quoted status = %v, %v", configured, err)
	}
	configured, err = credentialConfigured([]byte("ANTHROPIC_API_KEY= # comment\n"), "ANTHROPIC_API_KEY")
	if err != nil || configured {
		t.Fatalf("comment-only status = %v, %v", configured, err)
	}
	configured, err = credentialConfigured([]byte("ANTHROPIC_API_KEY=your-api-key\n"), "ANTHROPIC_API_KEY")
	if err != nil || configured {
		t.Fatalf("placeholder status = %v, %v", configured, err)
	}
	updated, err := replaceCredentialAssignment([]byte("OTHER=1"), "OPENAI_API_KEY", []byte(`a"b\c`))
	if err != nil {
		t.Fatal(err)
	}
	if string(updated) != "OTHER=1\nOPENAI_API_KEY=\"a\\\"b\\\\c\"\n" {
		t.Fatalf("appended content = %q", updated)
	}
	updated, err = replaceCredentialAssignment([]byte("\xef\xbb\xbfDEEPSEEK_API_KEY=old\nOTHER=1\n"), "DEEPSEEK_API_KEY", []byte("new"))
	if err != nil || !bytes.Equal(updated, []byte("\xef\xbb\xbfDEEPSEEK_API_KEY=\"new\"\nOTHER=1\n")) {
		t.Fatalf("BOM replace = %q, %v", updated, err)
	}
	updated, found, err := deleteCredentialAssignment(updated, "DEEPSEEK_API_KEY")
	if err != nil || !found || !bytes.Equal(updated, []byte("\xef\xbb\xbfOTHER=1\n")) {
		t.Fatalf("BOM delete = %q, %v, %v", updated, found, err)
	}
}

func newTestCredentialStore(t *testing.T) (credentialStore, string) {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "profiles"), 0o700); err != nil {
		t.Fatal(err)
	}
	return credentialStore{root: root}, filepath.Join(root, ".env")
}

func writeTestCredentialFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func readTestCredentialFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

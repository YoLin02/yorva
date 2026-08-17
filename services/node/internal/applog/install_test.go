package applog

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestNewWritesInstallLogWithoutSecrets(t *testing.T) {
	root := t.TempDir()
	var stderr bytes.Buffer
	logger, closeLog := New(&stderr, root)
	defer closeLog()

	logger.Info("runtime install",
		"event", "failed",
		"correlationId", "cor_test",
		"runtimeKind", "hermes",
		"stage", "install.repository",
		"status", "FAILED",
		"errorCode", "RUNTIME_INSTALL_STAGE_FAILED",
	)
	closeLog()

	path := InstallLogPath(root)
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(body, []byte(`"correlationId":"cor_test"`)) {
		t.Fatalf("log missing correlation: %s", body)
	}
	if !bytes.Contains(body, []byte(`"stage":"install.repository"`)) {
		t.Fatalf("log missing stage: %s", body)
	}
	if !bytes.Contains(body, []byte(`"errorCode":"RUNTIME_INSTALL_STAGE_FAILED"`)) {
		t.Fatalf("log missing error code: %s", body)
	}
	if bytes.Contains(body, []byte("Bearer ")) || bytes.Contains(body, []byte("sk-")) {
		t.Fatalf("log leaked a secret-looking field: %s", body)
	}

	var payload map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(body), &payload); err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != InstallLogName {
		t.Fatalf("log name = %s", path)
	}
	if stderr.Len() == 0 {
		t.Fatal("expected stderr mirror")
	}
}

func TestReadMatchingReturnsNeedleLines(t *testing.T) {
	root := t.TempDir()
	logger, closeLog := New(io.Discard, root)
	logger.Info("runtime install", "operationId", "op_keep", "detail", "first")
	logger.Info("runtime install", "operationId", "op_other", "detail", "skip")
	logger.Info("runtime install", "operationId", "op_keep", "detail", "second")
	closeLog()
	got := ReadMatching(root, "op_keep", 4096)
	if !bytes.Contains([]byte(got), []byte("first")) || !bytes.Contains([]byte(got), []byte("second")) {
		t.Fatalf("matching log = %s", got)
	}
	if bytes.Contains([]byte(got), []byte("op_other")) {
		t.Fatalf("unrelated line leaked: %s", got)
	}
}

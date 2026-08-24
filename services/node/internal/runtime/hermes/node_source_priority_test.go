package hermes

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes/downloadsources"
)

func TestPrerequisiteOnlineFirstFallsBackToVerifiedBundleOnTransportFailure(t *testing.T) {
	payload := []byte("verified artifact")
	sum := sha256.Sum256(payload)
	sha := hex.EncodeToString(sum[:])
	bundled := filepath.Join(t.TempDir(), "artifact.bin")
	if err := os.WriteFile(bundled, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)
	host := NewNodeHost(t.TempDir(), "", "")
	path, err := host.resolvePrerequisiteArtifact(context.Background(), downloadsources.PreferenceOnlineFirst, bundled, server.URL+"/artifact", filepath.Join(t.TempDir(), "download.bin"), int64(len(payload)), sha, yorvaruntime.ErrorHermesNodeArchiveIntegrityFailed)
	if err != nil || path != bundled {
		t.Fatalf("path=%q err=%v", path, err)
	}
}

func TestPrerequisiteIntegrityFailureDoesNotFallback(t *testing.T) {
	bundled := filepath.Join(t.TempDir(), "artifact.bin")
	if err := os.WriteFile(bundled, []byte("bundle"), 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("tampered"))
	}))
	t.Cleanup(server.Close)
	host := NewNodeHost(t.TempDir(), "", "")
	_, err := host.resolvePrerequisiteArtifact(context.Background(), downloadsources.PreferenceOnlineFirst, bundled, server.URL+"/artifact", filepath.Join(t.TempDir(), "download.bin"), int64(len("expected")), "0000000000000000000000000000000000000000000000000000000000000000", yorvaruntime.ErrorHermesNodeArchiveIntegrityFailed)
	if installErrorCode(err) != yorvaruntime.ErrorRuntimeInstallIntegrityFailed {
		t.Fatalf("error=%v", err)
	}
}

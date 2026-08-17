package hermes

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestSourceClientAcceptsExactBytes(t *testing.T) {
	payload := bytes.Repeat([]byte("official-install-script\n"), 32)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/install.ps1" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	client := testSourceClient(server.URL+"/install.ps1", payload)
	got, err := client.Fetch(context.Background(), t.TempDir(), "op_source")
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyRegularFile(got.Path, int64(len(payload)), got.SHA256); err != nil {
		t.Fatal(err)
	}
	if err := cleanupFetchedScript(got); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(got.Path); !os.IsNotExist(err) {
		t.Fatalf("cleanup left script: %v", err)
	}
}

func TestSourceClientRejectsChangedOversizedRedirectedAndTimedOutContent(t *testing.T) {
	payload := []byte("exact-bytes")
	sum := sha256.Sum256(payload)

	t.Run("digest", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("tampered-bytes"))
		}))
		defer server.Close()
		client := testSourceClient(server.URL, payload)
		_, err := client.Fetch(context.Background(), t.TempDir(), "op_digest")
		if installErrorCode(err) != yorvaruntime.ErrorRuntimeInstallIntegrityFailed {
			t.Fatalf("digest error = %v", err)
		}
	})

	t.Run("size", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(append(payload, 'x'))
		}))
		defer server.Close()
		client := testSourceClient(server.URL, payload)
		_, err := client.Fetch(context.Background(), t.TempDir(), "op_size")
		if installErrorCode(err) != yorvaruntime.ErrorRuntimeInstallIntegrityFailed {
			t.Fatalf("size error = %v", err)
		}
	})

	t.Run("oversize", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(bytes.Repeat([]byte("x"), 64))
		}))
		defer server.Close()
		client := testSourceClient(server.URL, payload)
		client.limit = 16
		_, err := client.Fetch(context.Background(), t.TempDir(), "op_oversize")
		if installErrorCode(err) != yorvaruntime.ErrorRuntimeInstallIntegrityFailed {
			t.Fatalf("oversize error = %v", err)
		}
	})

	t.Run("redirect", func(t *testing.T) {
		final := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write(payload)
		}))
		defer final.Close()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, final.URL, http.StatusFound)
		}))
		defer server.Close()
		client := testSourceClient(server.URL, payload)
		_, err := client.Fetch(context.Background(), t.TempDir(), "op_redirect")
		if installErrorCode(err) != yorvaruntime.ErrorRuntimeInstallSourceUnavailable {
			t.Fatalf("redirect error = %v", err)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(200 * time.Millisecond)
			_, _ = w.Write(payload)
		}))
		defer server.Close()
		client := testSourceClient(server.URL, payload)
		client.http.Timeout = 20 * time.Millisecond
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()
		_, err := client.Fetch(ctx, t.TempDir(), "op_timeout")
		if err == nil {
			t.Fatal("timeout unexpectedly succeeded")
		}
	})

	t.Run("rehash", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "install.ps1")
		if err := os.WriteFile(path, payload, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("changed"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := verifyRegularFile(path, int64(len(payload)), hex.EncodeToString(sum[:])); installErrorCode(err) != yorvaruntime.ErrorRuntimeInstallIntegrityFailed {
			t.Fatalf("rehash error = %v", err)
		}
	})
}

func testSourceClient(url string, payload []byte) sourceClient {
	sum := sha256.Sum256(payload)
	client := newSourceClient(officialSource{
		URL:          url,
		ExpectedSize: int64(len(payload)),
		ExpectedSHA:  hex.EncodeToString(sum[:]),
	})
	return client
}

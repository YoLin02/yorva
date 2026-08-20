package hermes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/YoLin02/yorva/services/node/internal/domain/channel"
)

func TestChannelCredentialsAreProfileScopedAndDisconnectIsTargeted(t *testing.T) {
	root := t.TempDir()
	for _, path := range []string{root, filepath.Join(root, "profiles", "alpha"), filepath.Join(root, "profiles", "bravo")} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	store := channelCredentialStore{credentials: credentialStore{root: root}}
	if err := store.SetWeCom("alpha", "bot-alpha", []byte("alpha-secret-value")); err != nil {
		t.Fatal(err)
	}
	alpha, err := store.Status("alpha", channel.WeCom)
	if err != nil || alpha.State != channel.Unknown || alpha.ExternalID != "bot-alpha" {
		t.Fatalf("alpha = %#v, %v", alpha, err)
	}
	bravo, err := store.Status("bravo", channel.WeCom)
	if err != nil || bravo.State != channel.NotConfigured {
		t.Fatalf("bravo = %#v, %v", bravo, err)
	}
	alphaEnv := filepath.Join(root, "profiles", "alpha", ".env")
	data, err := os.ReadFile(alphaEnv)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "WECOM_SECRET=\"alpha-secret-value\"") {
		t.Fatalf("Hermes credential authority not written: %q", data)
	}
	if err := store.Delete("alpha", channel.WeCom); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(alphaEnv)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "WECOM_") || strings.Contains(string(data), "alpha-secret-value") {
		t.Fatalf("target credential remained after disconnect: %q", data)
	}
}

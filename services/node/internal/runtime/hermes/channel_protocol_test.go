package hermes

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/YoLin02/yorva/services/node/internal/domain/channel"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestWeixinHostAllowlistAndResponseBound(t *testing.T) {
	for _, value := range []string{
		"http://ilinkai.weixin.qq.com",
		"https://ilinkai.weixin.qq.com.evil.example",
		"https://user@ilinkai.weixin.qq.com",
		"https://ilinkai.weixin.qq.com/?token=secret",
	} {
		if allowedWeixinBaseURL(value) {
			t.Fatalf("unsafe host accepted: %s", value)
		}
	}
	if !allowedWeixinBaseURL("https://ilinkai.weixin.qq.com") {
		t.Fatal("qualified Weixin host rejected")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(bytes.Repeat([]byte{'x'}, weixinResponseMax+1))
	}))
	defer server.Close()
	client := &weixinClient{http: server.Client()}
	var output map[string]any
	if err := client.getJSON(context.Background(), server.URL, &output); err == nil {
		t.Fatal("oversized Weixin response accepted")
	}
}

func TestWeComConnectVerifiesBeforeProfileScopedCommit(t *testing.T) {
	root := t.TempDir()
	for _, path := range []string{filepath.Join(root, "profiles", "alpha"), filepath.Join(root, "profiles", "bravo")} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	verified := false
	manager := &ChannelManager{
		credentials: channelCredentialStore{credentials: credentialStore{root: root}},
		verifyWeCom: func(_ context.Context, botID string, secret []byte) error {
			verified = botID == "bot-alpha" && string(secret) == "wecom-secret-value"
			return nil
		},
	}
	secret := []byte("wecom-secret-value")
	status, err := manager.BeginConnect(context.Background(), yorvaruntime.ChannelInstallation{Executable: `C:\hermes\bin\hermes.exe`, Version: "0.20.2"}, "alpha", yorvaruntime.ChannelConnectRequest{Type: channel.WeCom, BotID: "bot-alpha", Secret: secret}, nil)
	if err != nil || !verified || status.State != channel.Connected {
		t.Fatalf("connect = %#v, verified=%v, error=%v", status, verified, err)
	}
	if !bytes.Equal(secret, make([]byte, len(secret))) {
		t.Fatal("caller-owned secret copy was not cleared")
	}
	if _, err := os.Stat(filepath.Join(root, "profiles", "bravo", ".env")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unrelated profile was changed: %v", err)
	}
}

func TestWebSocketReaderRejectsServerMaskAndOversize(t *testing.T) {
	if _, _, err := readWebSocketFrame(bytes.NewReader([]byte{0x81, 0x80})); err == nil {
		t.Fatal("masked server frame accepted")
	}
	frame := []byte{0x81, 0x7f, 0, 0, 0, 0, 0, 1, 0, 1}
	if _, _, err := readWebSocketFrame(bytes.NewReader(frame)); err == nil {
		t.Fatal("oversized server frame accepted")
	}
}

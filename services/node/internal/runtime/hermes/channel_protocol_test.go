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
	status, err := manager.BeginConnect(context.Background(), yorvaruntime.ChannelInstallation{Executable: filepath.Join(root, "hermes.exe"), Version: "0.20.2"}, "alpha", yorvaruntime.ChannelConnectRequest{Type: channel.WeCom, BotID: "bot-alpha", Secret: secret}, nil)
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

func TestWeComSubscribeRequiresExplicitCorrelatedSuccess(t *testing.T) {
	for _, test := range []struct {
		name      string
		payload   string
		matched   bool
		wantError bool
	}{
		{name: "success", payload: `{"headers":{"req_id":"subscribe-one"},"errcode":0}`, matched: true},
		{name: "missing result", payload: `{"headers":{"req_id":"subscribe-one"}}`, matched: true, wantError: true},
		{name: "null result", payload: `{"headers":{"req_id":"subscribe-one"},"errcode":null}`, matched: true, wantError: true},
		{name: "explicit failure", payload: `{"headers":{"req_id":"subscribe-one"},"errcode":40013}`, matched: true, wantError: true},
		{name: "other request", payload: `{"headers":{"req_id":"subscribe-other"},"errcode":0}`},
		{name: "unknown schema", payload: `{"headers":"invalid","errcode":0}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			matched, err := parseWeComSubscribeResponse([]byte(test.payload), "subscribe-one")
			if matched != test.matched || (err != nil) != test.wantError {
				t.Fatalf("result = matched=%v err=%v", matched, err)
			}
			if test.wantError && !errors.Is(err, yorvaruntime.ErrChannelAuthFailed) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestWeComConnectDoesNotCommitAfterVerificationCancellation(t *testing.T) {
	root := t.TempDir()
	profileRoot := filepath.Join(root, "profiles", "alpha")
	if err := os.MkdirAll(profileRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	manager := &ChannelManager{
		credentials: channelCredentialStore{credentials: credentialStore{root: root}},
		verifyWeCom: func(context.Context, string, []byte) error {
			cancel()
			return nil
		},
	}
	secret := []byte("wecom-secret-value")
	_, err := manager.BeginConnect(ctx, yorvaruntime.ChannelInstallation{Executable: filepath.Join(root, "hermes.exe"), Version: "0.20.2"}, "alpha", yorvaruntime.ChannelConnectRequest{Type: channel.WeCom, BotID: "bot-alpha", Secret: secret}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("connect error = %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(profileRoot, ".env")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("cancelled verification committed credentials: %v", statErr)
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

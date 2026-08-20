package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/app"
	"github.com/YoLin02/yorva/services/node/internal/domain/channel"
	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
)

type fakeChannelInventory struct {
	fakeInstanceInventory
	views        []app.ChannelView
	operation    operation.Operation
	connect      app.ChannelConnectInput
	owner        string
	qrOwner      string
	qrPayload    app.ChannelQRPayload
	disconnected channel.Type
}

func (f *fakeChannelInventory) ListChannels(context.Context, string) ([]app.ChannelView, error) {
	return f.views, nil
}

func (f *fakeChannelInventory) StartChannelConnect(_ context.Context, _ string, _ string, owner string, input app.ChannelConnectInput) (app.InstallStartResult, error) {
	f.owner = owner
	f.connect = app.ChannelConnectInput{Type: input.Type, BotID: input.BotID, Secret: append([]byte(nil), input.Secret...)}
	return app.InstallStartResult{Operation: f.operation, Created: true}, nil
}

func (f *fakeChannelInventory) StartChannelDisconnect(_ context.Context, _ string, _ string, kind channel.Type) (app.InstallStartResult, error) {
	f.disconnected = kind
	return app.InstallStartResult{Operation: f.operation, Created: true}, nil
}

func (f *fakeChannelInventory) GetChannelQR(_ context.Context, _ string, owner string) (app.ChannelQRPayload, error) {
	if owner != f.qrOwner {
		return app.ChannelQRPayload{}, errors.New("not owner")
	}
	return f.qrPayload, nil
}

func (f *fakeChannelInventory) CancelChannel(context.Context, string) (operation.Operation, error) {
	return f.operation, nil
}

func TestChannelRoutesAreTypedAndDoNotEchoSecrets(t *testing.T) {
	now := time.Date(2026, 8, 20, 21, 0, 0, 0, time.UTC)
	inventory := &fakeChannelInventory{
		views:     []app.ChannelView{{Type: channel.Weixin, State: channel.NotConfigured, LastCheckedAt: &now}},
		operation: operation.Operation{ID: "op_channel", Type: operation.TypeChannelConnect, TargetType: operation.TargetInstance, TargetID: "inst_1", Status: operation.StatusPending, Stage: operation.StageChannelPreparing, CorrelationID: "cor_1", CreatedAt: now, UpdatedAt: now},
	}
	handler := NewHandler(testToken, testNode, nil, fakeRuntimeDiscovery{}, nil, inventory, "")

	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/instances/inst_1/channels", nil)
	listRequest.Header.Set("Authorization", "Bearer "+testToken)
	listResponse := httptest.NewRecorder()
	handler.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK || strings.Contains(listResponse.Body.String(), "secret") || strings.Contains(listResponse.Body.String(), "token") {
		t.Fatalf("list = %d %s", listResponse.Code, listResponse.Body.String())
	}

	const secret = "wecom-secret-sentinel"
	connectRequest := httptest.NewRequest(http.MethodPost, "/api/v1/instances/inst_1/channels/wecom/connect", strings.NewReader(`{"botId":"bot-one","secret":"`+secret+`"}`))
	connectRequest.Header.Set("Authorization", "Bearer "+testToken)
	connectRequest.Header.Set("Content-Type", "application/json")
	connectRequest.Header.Set("Idempotency-Key", "connect-wecom-1")
	connectRequest.Header.Set(channelSessionHeader, "desktop_session_owner_123456")
	connectResponse := httptest.NewRecorder()
	handler.ServeHTTP(connectResponse, connectRequest)
	if connectResponse.Code != http.StatusAccepted || strings.Contains(connectResponse.Body.String(), secret) {
		t.Fatalf("connect = %d %s", connectResponse.Code, connectResponse.Body.String())
	}
	if inventory.connect.BotID != "bot-one" || string(inventory.connect.Secret) != secret || inventory.owner != "desktop_session_owner_123456" {
		t.Fatalf("connect input = %#v owner=%q", inventory.connect, inventory.owner)
	}
}

func TestChannelQRRequiresInitiatingSessionAndIsNoStore(t *testing.T) {
	expires := time.Now().UTC().Add(time.Minute)
	inventory := &fakeChannelInventory{qrOwner: "desktop_session_owner_123456", qrPayload: app.ChannelQRPayload{Data: []byte("qr-source"), ExpiresAt: expires}}
	handler := NewHandler(testToken, testNode, nil, fakeRuntimeDiscovery{}, nil, inventory, "")

	request := httptest.NewRequest(http.MethodGet, "/api/v1/operations/op_1/channel-qr", nil)
	request.Header.Set("Authorization", "Bearer "+testToken)
	request.Header.Set(channelSessionHeader, inventory.qrOwner)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("owner response = %d headers=%v", response.Code, response.Header())
	}
	var payload ChannelQRResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil || payload.Payload != "qr-source" {
		t.Fatalf("payload = %#v, %v", payload, err)
	}

	other := httptest.NewRequest(http.MethodGet, "/api/v1/operations/op_1/channel-qr", nil)
	other.Header.Set("Authorization", "Bearer "+testToken)
	other.Header.Set(channelSessionHeader, "desktop_session_other_12345")
	denied := httptest.NewRecorder()
	handler.ServeHTTP(denied, other)
	if denied.Code != http.StatusNotFound || strings.Contains(denied.Body.String(), "qr-source") {
		t.Fatalf("non-owner response = %d %s", denied.Code, denied.Body.String())
	}
}

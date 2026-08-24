package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/channel"
	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type fakeChannelManager struct {
	statuses         []yorvaruntime.ChannelStatus
	ready            chan struct{}
	release          chan struct{}
	ignoreCancel     bool
	cancelObserved   chan struct{}
	disconnectReady  chan struct{}
	disconnectReturn chan struct{}
	disconnectCancel chan struct{}
}

func (f *fakeChannelManager) ListChannels(context.Context, yorvaruntime.ChannelInstallation, string) ([]yorvaruntime.ChannelStatus, error) {
	return append([]yorvaruntime.ChannelStatus(nil), f.statuses...), nil
}

func (f *fakeChannelManager) BeginConnect(ctx context.Context, _ yorvaruntime.ChannelInstallation, _ string, request yorvaruntime.ChannelConnectRequest, sink yorvaruntime.ChannelEventSink) (yorvaruntime.ChannelStatus, error) {
	if request.Type == channel.Weixin {
		if err := sink.Publish(yorvaruntime.ChannelEvent{Stage: "qr_ready", QRPayload: []byte("https://safe.example/ephemeral"), ExpiresAt: time.Now().Add(time.Minute)}); err != nil {
			return yorvaruntime.ChannelStatus{}, err
		}
		if f.ready != nil {
			close(f.ready)
		}
		if f.ignoreCancel {
			<-ctx.Done()
			close(f.cancelObserved)
			<-f.release
			return yorvaruntime.ChannelStatus{Type: request.Type, State: channel.Connected, AccountLabel: "safe", ExternalID: "external"}, nil
		}
		select {
		case <-ctx.Done():
			return yorvaruntime.ChannelStatus{}, ctx.Err()
		case <-f.release:
		}
	}
	return yorvaruntime.ChannelStatus{Type: request.Type, State: channel.Connected, AccountLabel: "safe", ExternalID: "external"}, nil
}

func (f *fakeChannelManager) Disconnect(ctx context.Context, _ yorvaruntime.ChannelInstallation, _ string, kind channel.Type) (yorvaruntime.ChannelStatus, error) {
	if f.disconnectReady != nil {
		close(f.disconnectReady)
		<-ctx.Done()
		close(f.disconnectCancel)
		<-f.disconnectReturn
	}
	return yorvaruntime.ChannelStatus{Type: kind, State: channel.NotConfigured}, nil
}

func (f *fakeChannelManager) PairingStatus(context.Context, yorvaruntime.ChannelInstallation, string, channel.Type) (yorvaruntime.ChannelPairingStatus, error) {
	return yorvaruntime.ChannelPairingStatus{PendingCount: 1}, nil
}

func (f *fakeChannelManager) ApprovePairing(context.Context, yorvaruntime.ChannelInstallation, string, channel.Type, []byte) error {
	return nil
}

func TestChannelQRIsAvailableOnlyToInitiatingSessionAndClearedOnCancel(t *testing.T) {
	manager := &fakeChannelManager{
		statuses: []yorvaruntime.ChannelStatus{{Type: channel.Weixin, State: channel.NotConfigured}, {Type: channel.WeCom, State: channel.NotConfigured}},
		ready:    make(chan struct{}),
		release:  make(chan struct{}),
	}
	inventory, _ := newTestInventoryWithManagers(t, []ProfileSnapshot{{NativeID: "default", Default: true}}, nil, nil, manager)
	listed, err := inventory.ListInstances(context.Background(), "hermes")
	if err != nil {
		t.Fatal(err)
	}
	const owner = "desktop_session_owner_123456"
	started, err := inventory.StartChannelConnect(context.Background(), listed.Instances[0].InstanceID, "channel-connect-qr-1", owner, ChannelConnectInput{Type: channel.Weixin})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-manager.ready:
	case <-time.After(time.Second):
		t.Fatal("QR was not published")
	}
	payload, err := inventory.GetChannelQR(context.Background(), started.Operation.ID, owner)
	if err != nil || string(payload.Data) != "https://safe.example/ephemeral" {
		t.Fatalf("owner QR = %q, %v", payload.Data, err)
	}
	if _, err := inventory.GetChannelQR(context.Background(), started.Operation.ID, "desktop_session_other_12345"); err == nil {
		t.Fatal("non-owner retrieved QR")
	}
	cancelled, err := inventory.CancelChannel(context.Background(), started.Operation.ID)
	if err != nil || cancelled.Status != operation.StatusCancelled {
		t.Fatalf("cancel = %#v, %v", cancelled, err)
	}
	if _, err := inventory.GetChannelQR(context.Background(), started.Operation.ID, owner); err == nil {
		t.Fatal("cancelled operation retained QR")
	}
}

func TestChannelCancelCannotOverwriteCommittedConnectSuccess(t *testing.T) {
	manager := &fakeChannelManager{
		statuses:       []yorvaruntime.ChannelStatus{{Type: channel.Weixin, State: channel.NotConfigured}},
		ready:          make(chan struct{}),
		release:        make(chan struct{}),
		ignoreCancel:   true,
		cancelObserved: make(chan struct{}),
	}
	inventory, _ := newTestInventoryWithManagers(t, []ProfileSnapshot{{NativeID: "default", Default: true}}, nil, nil, manager)
	listed, err := inventory.ListInstances(context.Background(), "hermes")
	if err != nil {
		t.Fatal(err)
	}
	started, err := inventory.StartChannelConnect(context.Background(), listed.Instances[0].InstanceID, "channel-connect-boundary-1", "desktop_session_owner_123456", ChannelConnectInput{Type: channel.Weixin})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-manager.ready:
	case <-time.After(time.Second):
		t.Fatal("connect did not reach commit boundary")
	}
	cancelResult := make(chan operation.Operation, 1)
	cancelError := make(chan error, 1)
	go func() {
		value, cancelErr := inventory.CancelChannel(context.Background(), started.Operation.ID)
		cancelResult <- value
		cancelError <- cancelErr
	}()
	select {
	case <-manager.cancelObserved:
	case <-time.After(time.Second):
		t.Fatal("connect did not observe cancellation")
	}
	select {
	case <-cancelResult:
		t.Fatal("cancel completed while the native mutation still owned the commit boundary")
	default:
	}
	close(manager.release)
	value := <-cancelResult
	if cancelErr := <-cancelError; cancelErr != nil || value.Status != operation.StatusSucceeded {
		t.Fatalf("cancel result = %#v, %v", value, cancelErr)
	}
	binding, exists, err := inventory.db.GetChannelBinding(context.Background(), listed.Instances[0].InstanceID, channel.Weixin)
	if err != nil || !exists || binding.State != channel.Connected {
		t.Fatalf("binding = %#v exists=%v err=%v", binding, exists, err)
	}
}

func TestChannelCancelCannotOverwriteCommittedDisconnectSuccess(t *testing.T) {
	manager := &fakeChannelManager{
		statuses:         []yorvaruntime.ChannelStatus{{Type: channel.WeCom, State: channel.Connected}},
		disconnectReady:  make(chan struct{}),
		disconnectReturn: make(chan struct{}),
		disconnectCancel: make(chan struct{}),
	}
	inventory, _ := newTestInventoryWithManagers(t, []ProfileSnapshot{{NativeID: "default", Default: true}}, nil, nil, manager)
	listed, err := inventory.ListInstances(context.Background(), "hermes")
	if err != nil {
		t.Fatal(err)
	}
	started, err := inventory.StartChannelDisconnect(context.Background(), listed.Instances[0].InstanceID, "channel-disconnect-boundary-1", channel.WeCom)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-manager.disconnectReady:
	case <-time.After(time.Second):
		t.Fatal("disconnect did not reach commit boundary")
	}
	cancelResult := make(chan operation.Operation, 1)
	cancelError := make(chan error, 1)
	go func() {
		value, cancelErr := inventory.CancelChannel(context.Background(), started.Operation.ID)
		cancelResult <- value
		cancelError <- cancelErr
	}()
	select {
	case <-manager.disconnectCancel:
	case <-time.After(time.Second):
		t.Fatal("disconnect did not observe cancellation")
	}
	select {
	case <-cancelResult:
		t.Fatal("cancel completed while disconnect still owned the commit boundary")
	default:
	}
	close(manager.disconnectReturn)
	value := <-cancelResult
	if cancelErr := <-cancelError; cancelErr != nil || value.Status != operation.StatusSucceeded {
		t.Fatalf("cancel result = %#v, %v", value, cancelErr)
	}
	binding, exists, err := inventory.db.GetChannelBinding(context.Background(), listed.Instances[0].InstanceID, channel.WeCom)
	if err != nil || !exists || binding.State != channel.NotConfigured {
		t.Fatalf("binding = %#v exists=%v err=%v", binding, exists, err)
	}
}

func TestChannelConnectConflictsWithActiveLifecycleMutation(t *testing.T) {
	manager := &fakeChannelManager{statuses: []yorvaruntime.ChannelStatus{{Type: channel.Weixin, State: channel.NotConfigured}}}
	inventory, _ := newTestInventoryWithManagers(t, []ProfileSnapshot{{NativeID: "default", Default: true}}, nil, nil, manager)
	listed, err := inventory.ListInstances(context.Background(), "hermes")
	if err != nil {
		t.Fatal(err)
	}
	now := inventory.now()
	active := operation.Operation{ID: "op_active_lifecycle", Type: operation.TypeInstanceStart, TargetType: operation.TargetInstance, TargetID: listed.Instances[0].InstanceID, Status: operation.StatusRunning, Stage: operation.StageInstanceStart, IdempotencyKey: "active-lifecycle", CreatedAt: now, UpdatedAt: now}
	if err := inventory.db.CreateOperation(context.Background(), active); err != nil {
		t.Fatal(err)
	}
	_, err = inventory.StartChannelConnect(context.Background(), active.TargetID, "channel-conflict-1", "desktop_session_owner_123456", ChannelConnectInput{Type: channel.Weixin})
	if !errors.Is(err, ErrChannelConflict) {
		t.Fatalf("conflict = %v", err)
	}
}

func TestChannelConnectConflictsWithActiveInstanceDelete(t *testing.T) {
	manager := &fakeChannelManager{statuses: []yorvaruntime.ChannelStatus{{Type: channel.Weixin, State: channel.NotConfigured}}}
	inventory, _ := newTestInventoryWithManagers(t, []ProfileSnapshot{{NativeID: "default", Default: true}}, nil, nil, manager)
	listed, err := inventory.ListInstances(context.Background(), "hermes")
	if err != nil {
		t.Fatal(err)
	}
	now := inventory.now()
	active := operation.Operation{ID: "op_active_delete", Type: operation.TypeInstanceDelete, TargetType: operation.TargetRuntimeInstallation, TargetID: listed.RuntimeInstallationID, Status: operation.StatusRunning, Stage: operation.StagePreflight, Message: "other", IdempotencyKey: "active-delete", CreatedAt: now, UpdatedAt: now}
	if err := inventory.db.CreateOperation(context.Background(), active); err != nil {
		t.Fatal(err)
	}
	_, err = inventory.StartChannelConnect(context.Background(), listed.Instances[0].InstanceID, "channel-delete-conflict-1", "desktop_session_owner_123456", ChannelConnectInput{Type: channel.Weixin})
	if !errors.Is(err, ErrChannelConflict) {
		t.Fatalf("conflict = %v", err)
	}
}

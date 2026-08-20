package runtime

import (
	"context"
	"errors"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/channel"
)

type ChannelInstallation struct {
	Executable string
	Version    string
}

type ChannelStatus struct {
	Type         channel.Type
	State        channel.State
	AccountLabel string
	ExternalID   string
}

type ChannelConnectRequest struct {
	Type   channel.Type
	BotID  string
	Secret []byte
}

type ChannelEvent struct {
	Stage     string
	QRPayload []byte
	ExpiresAt time.Time
}

type ChannelEventSink interface {
	Publish(ChannelEvent) error
}

type ChannelManager interface {
	ListChannels(context.Context, ChannelInstallation, string) ([]ChannelStatus, error)
	BeginConnect(context.Context, ChannelInstallation, string, ChannelConnectRequest, ChannelEventSink) (ChannelStatus, error)
	Disconnect(context.Context, ChannelInstallation, string, channel.Type) (ChannelStatus, error)
}

var (
	ErrChannelNotSupported     = errors.New("channel is not supported")
	ErrChannelStateUnknown     = errors.New("channel state is unknown")
	ErrChannelAuthFailed       = errors.New("channel authentication failed")
	ErrChannelAuthTimeout      = errors.New("channel authentication timed out")
	ErrChannelDependency       = errors.New("channel dependency is missing")
	ErrChannelDisconnectFailed = errors.New("channel disconnect failed")
)

package hermes

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/YoLin02/yorva/services/node/internal/domain/channel"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type ChannelManager struct {
	credentials channelCredentialStore
	weixin      *weixinClient
	verifyWeCom func(context.Context, string, []byte) error
}

func NewChannelManager() *ChannelManager {
	return &ChannelManager{
		credentials: newChannelCredentialStore(),
		weixin:      newWeixinClient(),
		verifyWeCom: verifyWeComCredentials,
	}
}

func (m *ChannelManager) ListChannels(ctx context.Context, installation yorvaruntime.ChannelInstallation, nativeID string) ([]yorvaruntime.ChannelStatus, error) {
	if err := validateChannelTarget(ctx, installation, nativeID); err != nil {
		return nil, err
	}
	out := make([]yorvaruntime.ChannelStatus, 0, 2)
	for _, kind := range []channel.Type{channel.Weixin, channel.WeCom} {
		status, err := m.credentials.Status(nativeID, kind)
		if err != nil {
			return nil, yorvaruntime.ErrChannelStateUnknown
		}
		out = append(out, status)
	}
	return out, nil
}

func (m *ChannelManager) BeginConnect(ctx context.Context, installation yorvaruntime.ChannelInstallation, nativeID string, request yorvaruntime.ChannelConnectRequest, sink yorvaruntime.ChannelEventSink) (yorvaruntime.ChannelStatus, error) {
	if err := validateChannelTarget(ctx, installation, nativeID); err != nil {
		return yorvaruntime.ChannelStatus{}, err
	}
	switch request.Type {
	case channel.Weixin:
		if len(request.Secret) != 0 || request.BotID != "" || sink == nil {
			return yorvaruntime.ChannelStatus{}, yorvaruntime.ErrChannelAuthFailed
		}
		credentials, err := m.weixin.Connect(ctx, sink)
		if err != nil {
			return yorvaruntime.ChannelStatus{}, err
		}
		defer clearCredentialBytes(credentials.Token)
		if err := m.credentials.SetWeixin(nativeID, credentials.AccountID, credentials.Token, credentials.BaseURL, credentials.UserID); err != nil {
			return yorvaruntime.ChannelStatus{}, yorvaruntime.ErrChannelAuthFailed
		}
		return yorvaruntime.ChannelStatus{Type: channel.Weixin, State: channel.Connected, AccountLabel: safeChannelLabel(credentials.UserID), ExternalID: credentials.AccountID}, nil
	case channel.WeCom:
		defer clearCredentialBytes(request.Secret)
		if request.BotID == "" || len(request.Secret) == 0 || sink != nil {
			return yorvaruntime.ChannelStatus{}, yorvaruntime.ErrChannelAuthFailed
		}
		if err := m.verifyWeCom(ctx, request.BotID, request.Secret); err != nil {
			return yorvaruntime.ChannelStatus{}, err
		}
		if err := m.credentials.SetWeCom(nativeID, request.BotID, request.Secret); err != nil {
			return yorvaruntime.ChannelStatus{}, yorvaruntime.ErrChannelAuthFailed
		}
		return yorvaruntime.ChannelStatus{Type: channel.WeCom, State: channel.Connected, AccountLabel: safeChannelLabel(request.BotID), ExternalID: request.BotID}, nil
	default:
		return yorvaruntime.ChannelStatus{}, yorvaruntime.ErrChannelNotSupported
	}
}

func (m *ChannelManager) Disconnect(ctx context.Context, installation yorvaruntime.ChannelInstallation, nativeID string, kind channel.Type) (yorvaruntime.ChannelStatus, error) {
	if err := validateChannelTarget(ctx, installation, nativeID); err != nil {
		return yorvaruntime.ChannelStatus{}, err
	}
	if !channel.ValidType(kind) {
		return yorvaruntime.ChannelStatus{}, yorvaruntime.ErrChannelNotSupported
	}
	if err := m.credentials.Delete(nativeID, kind); err != nil {
		return yorvaruntime.ChannelStatus{}, yorvaruntime.ErrChannelDisconnectFailed
	}
	return yorvaruntime.ChannelStatus{Type: kind, State: channel.NotConfigured}, nil
}

func validateChannelTarget(ctx context.Context, installation yorvaruntime.ChannelInstallation, nativeID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if installation.Version != channelCredentialVersion || installation.Executable == "" || !filepath.IsAbs(installation.Executable) {
		return yorvaruntime.ErrChannelNotSupported
	}
	normalized, err := officialNormalizeProfileName(nativeID)
	if err != nil || normalized != nativeID || officialValidateProfileName(nativeID) != nil {
		return yorvaruntime.ErrChannelNotSupported
	}
	return nil
}

func normalizeChannelError(err error) error {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return yorvaruntime.ErrChannelAuthTimeout
	case errors.Is(err, context.Canceled):
		return context.Canceled
	default:
		return yorvaruntime.ErrChannelAuthFailed
	}
}

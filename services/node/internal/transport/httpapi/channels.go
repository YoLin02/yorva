package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/app"
	"github.com/YoLin02/yorva/services/node/internal/domain/channel"
	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const channelSessionHeader = "Yorva-Session-Id"

var channelPairingCodePattern = regexp.MustCompile(`^[A-HJ-NP-Z2-9]{8}$`)

type ChannelService interface {
	ListChannels(context.Context, string) ([]app.ChannelView, error)
	StartChannelConnect(context.Context, string, string, string, app.ChannelConnectInput) (app.InstallStartResult, error)
	StartChannelDisconnect(context.Context, string, string, channel.Type) (app.InstallStartResult, error)
	GetChannelQR(context.Context, string, string) (app.ChannelQRPayload, error)
	GetChannelPairingStatus(context.Context, string, channel.Type) (app.ChannelPairingView, error)
	ApproveChannelPairing(context.Context, string, channel.Type, []byte) error
	CancelChannel(context.Context, string) (operation.Operation, error)
}

type ChannelResponse struct {
	Type              string     `json:"type"`
	State             string     `json:"state"`
	AccountLabel      string     `json:"accountLabel"`
	ExternalID        string     `json:"externalId"`
	LastCheckedAt     *time.Time `json:"lastCheckedAt"`
	ActiveOperationID *string    `json:"activeOperationId"`
}

type ChannelListResponse struct {
	Channels []ChannelResponse `json:"channels"`
}

type ChannelQRResponse struct {
	Payload   string    `json:"payload"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type ChannelPairingResponse struct {
	PendingCount int       `json:"pendingCount"`
	CheckedAt    time.Time `json:"checkedAt"`
}

type ChannelPairingApprovalResponse struct {
	Approved bool `json:"approved"`
}

func listInstanceChannels(service ChannelService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeChannelUnsupported(w)
			return
		}
		views, err := service.ListChannels(r.Context(), r.PathValue("instanceId"))
		if err != nil {
			writeChannelError(w, err)
			return
		}
		items := make([]ChannelResponse, 0, len(views))
		for _, view := range views {
			items = append(items, ChannelResponse{Type: string(view.Type), State: string(view.State), AccountLabel: view.AccountLabel, ExternalID: view.ExternalID, LastCheckedAt: view.LastCheckedAt, ActiveOperationID: view.ActiveOperationID})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ChannelListResponse{Channels: items})
	})
}

func connectInstanceChannel(service ChannelService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeChannelUnsupported(w)
			return
		}
		key := r.Header.Get("Idempotency-Key")
		if app.ValidateIdempotencyKey(key) != nil {
			writeError(w, http.StatusBadRequest, ErrorBody{Code: "INVALID_IDEMPOTENCY_KEY", Message: "A valid Idempotency-Key header is required."})
			return
		}
		kind := channel.Type(r.PathValue("channelType"))
		input, err := decodeChannelConnect(r, kind)
		if err != nil {
			writeError(w, http.StatusBadRequest, ErrorBody{Code: "INVALID_REQUEST", Message: "The channel connection request is invalid."})
			return
		}
		result, err := service.StartChannelConnect(r.Context(), r.PathValue("instanceId"), key, r.Header.Get(channelSessionHeader), input)
		clearRequestSecret(input.Secret)
		if err != nil {
			writeChannelError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(newOperationResponse(result.Operation))
	})
}

func disconnectInstanceChannel(service ChannelService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeChannelUnsupported(w)
			return
		}
		key := r.Header.Get("Idempotency-Key")
		if app.ValidateIdempotencyKey(key) != nil || decodeClosedEmptyObject(r) != nil {
			writeError(w, http.StatusBadRequest, ErrorBody{Code: "INVALID_REQUEST", Message: "A valid Idempotency-Key and closed JSON object are required."})
			return
		}
		result, err := service.StartChannelDisconnect(r.Context(), r.PathValue("instanceId"), key, channel.Type(r.PathValue("channelType")))
		if err != nil {
			writeChannelError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(newOperationResponse(result.Operation))
	})
}

func getChannelQR(service ChannelService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeChannelUnsupported(w)
			return
		}
		payload, err := service.GetChannelQR(r.Context(), r.PathValue("operationId"), r.Header.Get(channelSessionHeader))
		if err != nil {
			writeError(w, http.StatusNotFound, ErrorBody{Code: "CHANNEL_QR_NOT_AVAILABLE", Message: "The QR payload is unavailable or expired.", Retryable: true})
			return
		}
		defer clearRequestSecret(payload.Data)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(ChannelQRResponse{Payload: string(payload.Data), ExpiresAt: payload.ExpiresAt})
	})
}

func getChannelPairingStatus(service ChannelService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeChannelUnsupported(w)
			return
		}
		view, err := service.GetChannelPairingStatus(r.Context(), r.PathValue("instanceId"), channel.Type(r.PathValue("channelType")))
		if err != nil {
			writeChannelError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(ChannelPairingResponse{PendingCount: view.PendingCount, CheckedAt: view.CheckedAt})
	})
}

func approveChannelPairing(service ChannelService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeChannelUnsupported(w)
			return
		}
		code, err := decodePairingCode(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, ErrorBody{Code: string(yorvaruntime.ErrorChannelPairingCodeInvalid), Message: "The pairing code is invalid or expired."})
			return
		}
		defer clearRequestSecret(code)
		err = service.ApproveChannelPairing(r.Context(), r.PathValue("instanceId"), channel.Type(r.PathValue("channelType")), code)
		if err != nil {
			writeChannelError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(ChannelPairingApprovalResponse{Approved: true})
	})
}

func decodePairingCode(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, errors.New("missing body")
	}
	defer r.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(r.Body, 2049))
	if err != nil || len(payload) == 0 || len(payload) > 2048 {
		return nil, errors.New("invalid body")
	}
	defer clearRequestSecret(payload)
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var body struct {
		Code string `json:"code"`
	}
	if decoder.Decode(&body) != nil || !channelPairingCodePattern.MatchString(body.Code) {
		return nil, errors.New("invalid body")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, errors.New("trailing body")
	}
	return []byte(body.Code), nil
}

func decodeChannelConnect(r *http.Request, kind channel.Type) (app.ChannelConnectInput, error) {
	if kind == channel.Weixin {
		if err := decodeClosedEmptyObject(r); err != nil {
			return app.ChannelConnectInput{}, err
		}
		return app.ChannelConnectInput{Type: kind}, nil
	}
	if kind != channel.WeCom || r.Body == nil {
		return app.ChannelConnectInput{}, errors.New("unsupported channel")
	}
	defer r.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(r.Body, 32*1024+1))
	if err != nil || len(payload) == 0 || len(payload) > 32*1024 {
		return app.ChannelConnectInput{}, errors.New("invalid body")
	}
	defer clearRequestSecret(payload)
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var body struct {
		BotID  string `json:"botId"`
		Secret string `json:"secret"`
	}
	if decoder.Decode(&body) != nil || body.BotID == "" || body.Secret == "" {
		return app.ChannelConnectInput{}, errors.New("invalid body")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return app.ChannelConnectInput{}, errors.New("trailing body")
	}
	return app.ChannelConnectInput{Type: kind, BotID: body.BotID, Secret: []byte(body.Secret)}, nil
}

func writeChannelError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, app.ErrInstanceNotFound):
		writeError(w, http.StatusNotFound, ErrorBody{Code: string(yorvaruntime.ErrorInstanceNotFound), Message: "The requested instance was not found."})
	case errors.Is(err, app.ErrChannelNotSupported), errors.Is(err, yorvaruntime.ErrChannelNotSupported):
		writeChannelUnsupported(w)
	case errors.Is(err, app.ErrChannelConflict):
		writeError(w, http.StatusConflict, ErrorBody{Code: string(yorvaruntime.ErrorChannelConflict), Message: "Another lifecycle or channel operation is active."})
	case errors.Is(err, app.ErrChannelSession):
		writeError(w, http.StatusBadRequest, ErrorBody{Code: "INVALID_CHANNEL_SESSION", Message: "A valid initiating session is required."})
	case errors.Is(err, yorvaruntime.ErrChannelPairingCode):
		writeError(w, http.StatusBadRequest, ErrorBody{Code: string(yorvaruntime.ErrorChannelPairingCodeInvalid), Message: "The pairing code is invalid or expired."})
	case errors.Is(err, yorvaruntime.ErrChannelPairingLocked):
		writeError(w, http.StatusTooManyRequests, ErrorBody{Code: string(yorvaruntime.ErrorChannelPairingLocked), Message: "Pairing approval is temporarily locked after too many failed attempts."})
	case errors.Is(err, yorvaruntime.ErrChannelPairingQuery):
		writeError(w, http.StatusServiceUnavailable, ErrorBody{Code: string(yorvaruntime.ErrorChannelPairingQueryFailed), Message: "Pending pairing requests could not be checked.", Retryable: true})
	case errors.Is(err, yorvaruntime.ErrChannelPairingApproval):
		writeError(w, http.StatusServiceUnavailable, ErrorBody{Code: string(yorvaruntime.ErrorChannelPairingApprovalFailed), Message: "The pairing request could not be approved.", Retryable: true})
	case errors.Is(err, app.ErrInstanceNotAvailable):
		writeInstanceError(w, err)
	case errors.Is(err, context.Canceled):
		return
	default:
		writeError(w, http.StatusServiceUnavailable, ErrorBody{Code: string(yorvaruntime.ErrorChannelStateUnknown), Message: "Channel state could not be completed.", Retryable: true})
	}
}

func writeChannelUnsupported(w http.ResponseWriter) {
	writeError(w, http.StatusConflict, ErrorBody{Code: string(yorvaruntime.ErrorChannelNotSupported), Message: "This channel is not supported.", Retryable: false})
}

func clearRequestSecret(value []byte) {
	for index := range value {
		value[index] = 0
	}
}

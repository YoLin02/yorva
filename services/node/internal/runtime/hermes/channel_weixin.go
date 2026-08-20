package hermes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const (
	weixinBaseURL      = "https://ilinkai.weixin.qq.com"
	weixinQRPath       = "/ilink/bot/get_bot_qrcode?bot_type=3"
	weixinQRStatusPath = "/ilink/bot/get_qrcode_status?qrcode="
	weixinResponseMax  = 64 * 1024
	weixinQRPayloadMax = 8 * 1024
	weixinAuthTimeout  = 8 * time.Minute
	weixinQRExpiry     = 2 * time.Minute
)

type weixinClient struct {
	http *http.Client
	now  func() time.Time
}

type weixinCredentials struct {
	AccountID string
	Token     []byte
	BaseURL   string
	UserID    string
}

type weixinQRResponse struct {
	QRCode  string `json:"qrcode"`
	Content string `json:"qrcode_img_content"`
}

type weixinStatusResponse struct {
	Status       string `json:"status"`
	RedirectHost string `json:"redirect_host"`
	AccountID    string `json:"ilink_bot_id"`
	Token        string `json:"bot_token"`
	BaseURL      string `json:"baseurl"`
	UserID       string `json:"ilink_user_id"`
}

func newWeixinClient() *weixinClient {
	return &weixinClient{
		http: &http.Client{
			Timeout:       35 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
		now: func() time.Time { return time.Now().UTC() },
	}
}

func (c *weixinClient) Connect(ctx context.Context, sink yorvaruntime.ChannelEventSink) (weixinCredentials, error) {
	ctx, cancel := context.WithTimeout(ctx, weixinAuthTimeout)
	defer cancel()
	baseURL := weixinBaseURL
	qr, err := c.fetchQR(ctx)
	if err != nil {
		return weixinCredentials{}, normalizeChannelError(err)
	}
	if err := publishWeixinQR(sink, qr, c.now().Add(weixinQRExpiry)); err != nil {
		return weixinCredentials{}, yorvaruntime.ErrChannelAuthFailed
	}
	refreshes := 0
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return weixinCredentials{}, normalizeChannelError(ctx.Err())
		case <-ticker.C:
		}
		status, pollErr := c.fetchStatus(ctx, baseURL, qr.QRCode)
		if pollErr != nil {
			if ctx.Err() != nil {
				return weixinCredentials{}, normalizeChannelError(ctx.Err())
			}
			continue
		}
		switch status.Status {
		case "wait", "scaned":
			continue
		case "scaned_but_redirect":
			candidate := "https://" + status.RedirectHost
			if !allowedWeixinBaseURL(candidate) {
				return weixinCredentials{}, yorvaruntime.ErrChannelAuthFailed
			}
			baseURL = candidate
		case "expired":
			refreshes++
			if refreshes > 3 {
				return weixinCredentials{}, yorvaruntime.ErrChannelAuthTimeout
			}
			qr, err = c.fetchQR(ctx)
			if err != nil || publishWeixinQR(sink, qr, c.now().Add(weixinQRExpiry)) != nil {
				return weixinCredentials{}, yorvaruntime.ErrChannelAuthFailed
			}
			baseURL = weixinBaseURL
		case "confirmed":
			resolvedBaseURL := status.BaseURL
			if resolvedBaseURL == "" {
				resolvedBaseURL = weixinBaseURL
			}
			if !channelIdentityPattern.MatchString(status.AccountID) || status.Token == "" || len(status.Token) > modelCredentialMaxValue || !allowedWeixinBaseURL(resolvedBaseURL) || (status.UserID != "" && !channelIdentityPattern.MatchString(status.UserID)) {
				return weixinCredentials{}, yorvaruntime.ErrChannelAuthFailed
			}
			return weixinCredentials{AccountID: status.AccountID, Token: []byte(status.Token), BaseURL: resolvedBaseURL, UserID: status.UserID}, nil
		default:
			return weixinCredentials{}, yorvaruntime.ErrChannelAuthFailed
		}
	}
}

func (c *weixinClient) fetchQR(ctx context.Context) (weixinQRResponse, error) {
	var value weixinQRResponse
	if err := c.getJSON(ctx, weixinBaseURL+weixinQRPath, &value); err != nil {
		return value, err
	}
	if value.QRCode == "" || len(value.QRCode) > 512 || value.Content == "" || len(value.Content) > weixinQRPayloadMax {
		return value, errors.New("invalid Weixin QR response")
	}
	return value, nil
}

func (c *weixinClient) fetchStatus(ctx context.Context, baseURL, qrCode string) (weixinStatusResponse, error) {
	var value weixinStatusResponse
	if !allowedWeixinBaseURL(baseURL) {
		return value, errors.New("invalid Weixin status host")
	}
	err := c.getJSON(ctx, strings.TrimRight(baseURL, "/")+weixinQRStatusPath+url.QueryEscape(qrCode), &value)
	return value, err
}

func (c *weixinClient) getJSON(ctx context.Context, target string, out any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	request.Header.Set("iLink-App-Id", "bot")
	request.Header.Set("iLink-App-ClientVersion", "131584")
	response, err := c.http.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.ContentLength > weixinResponseMax {
		return fmt.Errorf("unexpected Weixin response status")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, weixinResponseMax+1))
	if err != nil || len(body) > weixinResponseMax {
		return errors.New("invalid Weixin response body")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(out); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("invalid Weixin response body")
	}
	return nil
}

func allowedWeixinBaseURL(raw string) bool {
	parsed, err := url.Parse(raw)
	return err == nil && parsed.Scheme == "https" && parsed.Host == "ilinkai.weixin.qq.com" && parsed.User == nil && parsed.RawQuery == "" && parsed.Fragment == "" && (parsed.Path == "" || parsed.Path == "/")
}

func publishWeixinQR(sink yorvaruntime.ChannelEventSink, qr weixinQRResponse, expiresAt time.Time) error {
	return sink.Publish(yorvaruntime.ChannelEvent{Stage: "qr_ready", QRPayload: []byte(qr.Content), ExpiresAt: expiresAt})
}

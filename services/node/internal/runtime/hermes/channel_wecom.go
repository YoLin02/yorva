package hermes

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const (
	wecomHost             = "openws.work.weixin.qq.com"
	wecomHandshakeTimeout = 20 * time.Second
	wecomFrameMax         = 64 * 1024
)

func verifyWeComCredentials(ctx context.Context, botID string, secret []byte) error {
	if !channelIdentityPattern.MatchString(botID) || len(secret) == 0 || len(secret) > modelCredentialMaxValue {
		return yorvaruntime.ErrChannelAuthFailed
	}
	ctx, cancel := context.WithTimeout(ctx, wecomHandshakeTimeout)
	defer cancel()
	dialer := &net.Dialer{Timeout: wecomHandshakeTimeout}
	connection, err := tls.DialWithDialer(dialer, "tcp", wecomHost+":443", &tls.Config{ServerName: wecomHost, MinVersion: tls.VersionTLS12})
	if err != nil {
		return normalizeChannelError(err)
	}
	defer connection.Close()
	deadline, ok := ctx.Deadline()
	if ok {
		_ = connection.SetDeadline(deadline)
	}
	reader := bufio.NewReaderSize(connection, wecomFrameMax)
	if err := upgradeWeComWebSocket(connection, reader); err != nil {
		return normalizeChannelError(err)
	}
	reqBytes := make([]byte, 16)
	if _, err := rand.Read(reqBytes); err != nil {
		return yorvaruntime.ErrChannelAuthFailed
	}
	reqID := "subscribe-" + hex.EncodeToString(reqBytes)
	payload, err := json.Marshal(map[string]any{
		"cmd":     "aibot_subscribe",
		"headers": map[string]string{"req_id": reqID},
		"body":    map[string]string{"bot_id": botID, "secret": string(secret), "device_id": hex.EncodeToString(reqBytes)},
	})
	if err != nil {
		return yorvaruntime.ErrChannelAuthFailed
	}
	defer clearCredentialBytes(payload)
	if err := writeWebSocketFrame(connection, 0x1, payload); err != nil {
		return normalizeChannelError(err)
	}
	for {
		opcode, response, err := readWebSocketFrame(reader)
		if err != nil {
			return normalizeChannelError(err)
		}
		switch opcode {
		case 0x8:
			return yorvaruntime.ErrChannelAuthFailed
		case 0x9:
			if err := writeWebSocketFrame(connection, 0xA, response); err != nil {
				return normalizeChannelError(err)
			}
			continue
		case 0x1:
		default:
			continue
		}
		var envelope struct {
			Headers map[string]string `json:"headers"`
			ErrCode *int              `json:"errcode"`
		}
		if json.Unmarshal(response, &envelope) != nil || envelope.Headers["req_id"] != reqID {
			continue
		}
		if envelope.ErrCode != nil && *envelope.ErrCode != 0 {
			return yorvaruntime.ErrChannelAuthFailed
		}
		return nil
	}
}

func upgradeWeComWebSocket(connection net.Conn, reader *bufio.Reader) error {
	keyBytes := make([]byte, 16)
	if _, err := rand.Read(keyBytes); err != nil {
		return err
	}
	key := base64.StdEncoding.EncodeToString(keyBytes)
	request := "GET / HTTP/1.1\r\nHost: " + wecomHost + "\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: " + key + "\r\nSec-WebSocket-Version: 13\r\n\r\n"
	if _, err := io.WriteString(connection, request); err != nil {
		return err
	}
	response, err := http.ReadResponse(reader, &http.Request{Method: http.MethodGet})
	if err != nil {
		return err
	}
	defer response.Body.Close()
	digest := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	wantAccept := base64.StdEncoding.EncodeToString(digest[:])
	if response.StatusCode != http.StatusSwitchingProtocols || !strings.EqualFold(response.Header.Get("Upgrade"), "websocket") || response.Header.Get("Sec-WebSocket-Accept") != wantAccept {
		return errors.New("WeCom WebSocket upgrade rejected")
	}
	return nil
}

func writeWebSocketFrame(writer io.Writer, opcode byte, payload []byte) error {
	if len(payload) > wecomFrameMax {
		return errors.New("WebSocket payload too large")
	}
	header := []byte{0x80 | opcode, 0x80}
	switch {
	case len(payload) < 126:
		header[1] |= byte(len(payload))
	case len(payload) <= 65535:
		header[1] |= 126
		header = append(header, byte(len(payload)>>8), byte(len(payload)))
	default:
		return errors.New("WebSocket payload too large")
	}
	mask := make([]byte, 4)
	if _, err := rand.Read(mask); err != nil {
		return err
	}
	frame := make([]byte, len(header)+len(mask)+len(payload))
	copy(frame, header)
	copy(frame[len(header):], mask)
	for index := range payload {
		frame[len(header)+len(mask)+index] = payload[index] ^ mask[index%4]
	}
	_, err := writer.Write(frame)
	clearCredentialBytes(frame)
	return err
}

func readWebSocketFrame(reader io.Reader) (byte, []byte, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(reader, header); err != nil {
		return 0, nil, err
	}
	if header[0]&0x80 == 0 || header[0]&0x70 != 0 || header[1]&0x80 != 0 {
		return 0, nil, errors.New("unsupported WebSocket frame")
	}
	opcode := header[0] & 0x0f
	length := uint64(header[1] & 0x7f)
	if length == 126 {
		extended := make([]byte, 2)
		if _, err := io.ReadFull(reader, extended); err != nil {
			return 0, nil, err
		}
		length = uint64(binary.BigEndian.Uint16(extended))
	} else if length == 127 {
		extended := make([]byte, 8)
		if _, err := io.ReadFull(reader, extended); err != nil {
			return 0, nil, err
		}
		length = binary.BigEndian.Uint64(extended)
	}
	if length > wecomFrameMax {
		return 0, nil, errors.New("WebSocket frame too large")
	}
	payload := make([]byte, int(length))
	if _, err := io.ReadFull(reader, payload); err != nil {
		return 0, nil, err
	}
	return opcode, payload, nil
}

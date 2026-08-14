package bootstrap

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const maxMessageBytes = 8 * 1024

var (
	ErrInvalidMessage          = errors.New("invalid bootstrap message")
	ErrProtocolVersionMismatch = errors.New("bootstrap protocol version mismatch")
)

type Message struct {
	ProtocolVersion string `json:"protocolVersion"`
	Token           string `json:"token"`
	DataDir         string `json:"dataDir"`
}

type Handshake struct {
	ProtocolVersion string `json:"protocolVersion"`
	Port            int    `json:"port"`
	PID             int    `json:"pid"`
}

func NewReader(r io.Reader) *bufio.Reader {
	return bufio.NewReaderSize(r, maxMessageBytes+1)
}

func ReadMessage(r *bufio.Reader, expectedProtocol string) (Message, error) {
	line, err := r.ReadSlice('\n')
	if err != nil && !(errors.Is(err, io.EOF) && len(line) > 0) {
		return Message{}, fmt.Errorf("%w: %v", ErrInvalidMessage, err)
	}
	if len(line) > maxMessageBytes {
		return Message{}, ErrInvalidMessage
	}

	decoder := json.NewDecoder(bytes.NewReader(line))
	decoder.DisallowUnknownFields()

	var message Message
	if err := decoder.Decode(&message); err != nil {
		return Message{}, fmt.Errorf("%w: %v", ErrInvalidMessage, err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Message{}, ErrInvalidMessage
	}

	if message.ProtocolVersion != expectedProtocol {
		return Message{}, ErrProtocolVersionMismatch
	}
	if !validToken(message.Token) || strings.TrimSpace(message.DataDir) == "" {
		return Message{}, ErrInvalidMessage
	}

	return message, nil
}

func WriteHandshake(w io.Writer, handshake Handshake) error {
	if err := json.NewEncoder(w).Encode(handshake); err != nil {
		return fmt.Errorf("write bootstrap handshake: %w", err)
	}
	return nil
}

func validToken(token string) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	return err == nil && len(decoded) >= 32
}

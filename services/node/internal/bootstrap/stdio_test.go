package bootstrap

import (
	"bytes"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func TestReadMessage(t *testing.T) {
	token := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x5a}, 32))
	input := `{"protocolVersion":"1","token":"` + token + `","dataDir":"C:\\yorva"}`

	message, err := ReadMessage(NewReader(strings.NewReader(input)), "1")
	if err != nil {
		t.Fatalf("ReadMessage() error = %v", err)
	}
	if message.Token != token || message.DataDir != `C:\yorva` {
		t.Fatalf("ReadMessage() = %#v", message)
	}
}

func TestReadMessageRejectsShortToken(t *testing.T) {
	input := `{"protocolVersion":"1","token":"c2hvcnQ","dataDir":"C:\\yorva"}`
	_, err := ReadMessage(NewReader(strings.NewReader(input)), "1")
	if !errors.Is(err, ErrInvalidMessage) {
		t.Fatalf("ReadMessage() error = %v, want ErrInvalidMessage", err)
	}
}

func TestReadMessageRejectsProtocolMismatch(t *testing.T) {
	token := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x5a}, 32))
	input := `{"protocolVersion":"2","token":"` + token + `","dataDir":"C:\\yorva"}`
	_, err := ReadMessage(NewReader(strings.NewReader(input)), "1")
	if !errors.Is(err, ErrProtocolVersionMismatch) {
		t.Fatalf("ReadMessage() error = %v, want ErrProtocolVersionMismatch", err)
	}
}

func TestWriteHandshakeDoesNotContainToken(t *testing.T) {
	var output bytes.Buffer
	if err := WriteHandshake(&output, Handshake{ProtocolVersion: "1", Port: 49152, PID: 123}); err != nil {
		t.Fatalf("WriteHandshake() error = %v", err)
	}
	if strings.Contains(output.String(), "token") {
		t.Fatalf("handshake contains token field: %s", output.String())
	}
}

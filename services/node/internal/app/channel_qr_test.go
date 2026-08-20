package app

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

func TestChannelQRBrokerIsInitiatingSessionOnlyAndCopiesPayload(t *testing.T) {
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	broker := newChannelQRBroker()
	broker.now = func() time.Time { return now }
	original := []byte("https://example.invalid/ephemeral-qr")
	owner := "session_12345678901234567890"
	if err := broker.Publish("op_test", owner, original, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	original[0] = 'X'
	if _, err := broker.Get("op_test", "session_00000000000000000000"); !errors.Is(err, errChannelQRUnavailable) {
		t.Fatalf("other session get = %v", err)
	}
	payload, err := broker.Get("op_test", owner)
	if err != nil || !bytes.Equal(payload.Data, []byte("https://example.invalid/ephemeral-qr")) {
		t.Fatalf("payload = %q, %v", payload.Data, err)
	}
	payload.Data[0] = 'X'
	again, _ := broker.Get("op_test", owner)
	if again.Data[0] != 'h' {
		t.Fatal("caller mutated broker payload")
	}
}

func TestChannelQRBrokerExpiresAndDeletesPayload(t *testing.T) {
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	broker := newChannelQRBroker()
	broker.now = func() time.Time { return now }
	owner := "session_12345678901234567890"
	if err := broker.Publish("op_test", owner, []byte("secret-equivalent"), now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Second)
	if _, err := broker.Get("op_test", owner); !errors.Is(err, errChannelQRUnavailable) {
		t.Fatalf("expired get = %v", err)
	}
	if len(broker.entries) != 0 {
		t.Fatal("expired payload retained")
	}
	if err := broker.Publish("op_delete", owner, []byte("secret-equivalent"), now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	broker.Delete("op_delete")
	if _, err := broker.Get("op_delete", owner); !errors.Is(err, errChannelQRUnavailable) {
		t.Fatalf("deleted get = %v", err)
	}
}

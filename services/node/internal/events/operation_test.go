package events

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestTypeForCommittedOperation(t *testing.T) {
	tests := []struct {
		created  bool
		previous string
		next     string
		want     string
	}{
		{created: true, next: "PENDING", want: TypeOperationStarted},
		{created: false, previous: "PENDING", next: "RUNNING", want: TypeOperationProgress},
		{created: false, previous: "RUNNING", next: "RUNNING", want: TypeOperationProgress},
		{created: false, previous: "RUNNING", next: "SUCCEEDED", want: TypeOperationCompleted},
		{created: false, previous: "RUNNING", next: "FAILED", want: TypeOperationFailed},
		{created: false, previous: "PENDING", next: "CANCELLED", want: TypeOperationCancelled},
		{created: false, previous: "RUNNING", next: "CANCELLED", want: TypeOperationCancelled},
	}
	for _, test := range tests {
		got := TypeForCommittedOperation(test.created, test.previous, test.next)
		if got != test.want {
			t.Fatalf("TypeForCommittedOperation(%v, %s, %s) = %s, want %s", test.created, test.previous, test.next, got, test.want)
		}
	}
}

func TestNewOperationEventExcludesRawFields(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	event := NewOperationEvent(TypeOperationFailed, OperationPayload{
		OperationID:   "op_safe",
		Type:          "runtime.install",
		Status:        "FAILED",
		Stage:         "source.verify",
		ErrorCode:     "RUNTIME_INSTALL_INTEGRITY_FAILED",
		CorrelationID: "cor_safe",
	}, now)
	if event.Type != TypeOperationFailed || event.OccurredAt != now {
		t.Fatalf("event = %#v", event)
	}
	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, forbidden := range []string{"stdout", "stderr", "https://", "C:\\\\Users", "token=", "errorMessage", "message"} {
		if strings.Contains(strings.ToLower(text), strings.ToLower(forbidden)) && forbidden != "message" {
			t.Fatalf("event leaked %q: %s", forbidden, text)
		}
	}
	var payload OperationPayload
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.OperationID != "op_safe" || payload.ErrorCode != "RUNTIME_INSTALL_INTEGRITY_FAILED" {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestChannelQRReadyEventContainsMetadataOnly(t *testing.T) {
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	expires := now.Add(time.Minute)
	event := NewChannelQRReadyEvent("op_channel", expires, now)
	if event.Type != TypeChannelQRReady || strings.Contains(string(event.Data), "payload") || strings.Contains(string(event.Data), "http") {
		t.Fatalf("unsafe QR event = %#v", event)
	}
	var payload ChannelQRReadyPayload
	if err := json.Unmarshal(event.Data, &payload); err != nil || payload.OperationID != "op_channel" || !payload.ExpiresAt.Equal(expires) {
		t.Fatalf("payload = %#v, %v", payload, err)
	}
}

func TestBrokerAssignsEventID(t *testing.T) {
	broker := NewBroker()
	sub := broker.Subscribe()
	defer sub.Close()
	broker.Publish(Event{Type: TypeOperationStarted, Data: []byte(`{"operationId":"op_1"}`)})
	select {
	case event := <-sub.Events:
		if event.ID != "evt_1" || event.Type != TypeOperationStarted {
			t.Fatalf("published event = %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("expected assigned event id")
	}
}

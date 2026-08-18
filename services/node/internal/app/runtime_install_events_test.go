package app

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	"github.com/YoLin02/yorva/services/node/internal/events"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestRuntimeInstallPublishesOrderedLifecycleEvents(t *testing.T) {
	store := newMemoryOperationStore()
	broker := events.NewBroker()
	sub := broker.Subscribe()
	defer sub.Close()
	applier := &fakeApplier{dir: `C:\Users\a\AppData\Local\hermes\hermes-agent\bin\hermes.exe`, version: "0.20.2"}
	service := newOrchestratedInstall(store, []yorvaruntime.Discovery{
		{State: yorvaruntime.DiscoveryNotInstalled},
		{
			State:    yorvaruntime.DiscoverySupported,
			Selected: &yorvaruntime.Candidate{Path: applier.dir, Version: "0.20.2", State: yorvaruntime.DiscoverySupported},
		},
	}, applier, &fakeCompleter{store: store}).WithEvents(broker)

	started, err := service.Start(context.Background(), "hermes", "install-events")
	if err != nil {
		t.Fatal(err)
	}
	waitForStatus(t, service, started.Operation.ID, operation.StatusSucceeded)
	got := collectEvents(t, sub, events.TypeOperationCompleted)
	assertEventOrder(t, got, events.TypeOperationStarted, events.TypeOperationProgress, events.TypeOperationCompleted)
	for _, event := range got {
		assertSafeOperationEvent(t, event)
		var payload events.OperationPayload
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			t.Fatal(err)
		}
		if payload.OperationID != started.Operation.ID || payload.Type != string(operation.TypeRuntimeInstall) {
			t.Fatalf("payload = %#v", payload)
		}
	}
}

func TestRuntimeInstallPublishesFailedAndCancelledEvents(t *testing.T) {
	t.Run("failed", func(t *testing.T) {
		store := newMemoryOperationStore()
		broker := events.NewBroker()
		sub := broker.Subscribe()
		defer sub.Close()
		applier := &fakeApplier{dir: `C:\Users\a\AppData\Local\hermes\hermes-agent`, version: "0.20.2"}
		service := newOrchestratedInstall(store, []yorvaruntime.Discovery{
			{State: yorvaruntime.DiscoveryNotInstalled},
			{State: yorvaruntime.DiscoveryBrokenExecutable},
		}, applier, &fakeCompleter{store: store}).WithEvents(broker)
		started, err := service.Start(context.Background(), "hermes", "install-failed-event")
		if err != nil {
			t.Fatal(err)
		}
		waitForStatus(t, service, started.Operation.ID, operation.StatusFailed)
		got := collectEvents(t, sub, events.TypeOperationFailed)
		if got[len(got)-1].Type != events.TypeOperationFailed {
			t.Fatalf("events = %#v", typesOf(got))
		}
		assertSafeOperationEvent(t, got[len(got)-1])
	})

	t.Run("cancelled", func(t *testing.T) {
		store := newMemoryOperationStore()
		broker := events.NewBroker()
		sub := broker.Subscribe()
		defer sub.Close()
		service := newTestRuntimeInstall(store, yorvaruntime.Discovery{State: yorvaruntime.DiscoveryNotInstalled}).WithEvents(broker)
		started, err := service.Start(context.Background(), "hermes", "install-cancel-event")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := service.Cancel(context.Background(), started.Operation.ID); err != nil {
			t.Fatal(err)
		}
		got := collectEvents(t, sub, events.TypeOperationCancelled)
		if got[0].Type != events.TypeOperationStarted || got[len(got)-1].Type != events.TypeOperationCancelled {
			t.Fatalf("events = %#v", typesOf(got))
		}
	})
}

func TestOperationEventsExcludeSecretsAndRawOutput(t *testing.T) {
	store := newMemoryOperationStore()
	broker := events.NewBroker()
	sub := broker.Subscribe()
	defer sub.Close()
	service := newTestRuntimeInstall(store, yorvaruntime.Discovery{State: yorvaruntime.DiscoveryNotInstalled}).WithEvents(broker)
	started, err := service.Start(context.Background(), "hermes", "install-secret-event")
	if err != nil {
		t.Fatal(err)
	}
	current := started.Operation
	now := service.now()
	next := current
	next.Status = operation.StatusFailed
	next.ErrorCode = yorvaruntime.ErrorRuntimeInstallSourceUnavailable
	next.Message = "GET https://files.example.invalid/hermes.zip?token=super-secret"
	next.ErrorMessage = `C:\Users\alice\.yorva\tmp failed: raw stderr boom`
	next.CompletedAt = &now
	next.UpdatedAt = now
	if err := service.persistUpdate(context.Background(), current, next); err != nil {
		t.Fatal(err)
	}
	got := collectEvents(t, sub, events.TypeOperationFailed)
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, forbidden := range []string{
		"https://files.example.invalid",
		"super-secret",
		`C:\Users\alice`,
		"raw stderr boom",
		"token=",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("event payload leaked %q: %s", forbidden, text)
		}
	}
}

func TestSSEDisconnectDoesNotReplaceGetAsSourceOfTruth(t *testing.T) {
	store := newMemoryOperationStore()
	broker := events.NewBroker()
	first := broker.Subscribe()
	service := newTestRuntimeInstall(store, yorvaruntime.Discovery{State: yorvaruntime.DiscoveryNotInstalled}).WithEvents(broker)
	started, err := service.Start(context.Background(), "hermes", "install-sse-reconnect")
	if err != nil {
		t.Fatal(err)
	}
	_ = collectEvents(t, first, events.TypeOperationStarted)
	first.Close()

	if _, err := service.Cancel(context.Background(), started.Operation.ID); err != nil {
		t.Fatal(err)
	}
	authoritative, err := service.Get(context.Background(), started.Operation.ID)
	if err != nil || authoritative.Status != operation.StatusCancelled {
		t.Fatalf("GET after disconnect = %#v, %v", authoritative, err)
	}

	second := broker.Subscribe()
	defer second.Close()
	select {
	case event := <-second.Events:
		t.Fatalf("reconnect must not replay missed events: %#v", event)
	case <-time.After(50 * time.Millisecond):
	}
	again, err := service.Get(context.Background(), started.Operation.ID)
	if err != nil || again.Status != operation.StatusCancelled || again.ID != started.Operation.ID {
		t.Fatalf("GET after reconnect = %#v, %v", again, err)
	}
}

func collectEvents(t *testing.T, sub *events.Subscription, stopType string) []events.Event {
	t.Helper()
	var got []events.Event
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case event := <-sub.Events:
			got = append(got, event)
			if event.Type == stopType {
				return got
			}
		case <-time.After(20 * time.Millisecond):
		}
	}
	t.Fatalf("did not observe %s; events=%v", stopType, typesOf(got))
	return got
}

func assertEventOrder(t *testing.T, got []events.Event, want ...string) {
	t.Helper()
	types := typesOf(got)
	index := 0
	for _, eventType := range types {
		if index < len(want) && eventType == want[index] {
			index++
		}
	}
	if index != len(want) {
		t.Fatalf("event order %v does not contain %v", types, want)
	}
}

func typesOf(got []events.Event) []string {
	types := make([]string, 0, len(got))
	for _, event := range got {
		types = append(types, event.Type)
	}
	return types
}

func assertSafeOperationEvent(t *testing.T, event events.Event) {
	t.Helper()
	var envelope map[string]any
	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	var payload events.OperationPayload
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.OperationID == "" || payload.Type == "" || payload.Status == "" || payload.Stage == "" {
		t.Fatalf("incomplete payload %#v", payload)
	}
	if strings.Contains(string(event.Data), "stdout") || strings.Contains(string(event.Data), "stderr") {
		t.Fatalf("payload included raw output: %s", event.Data)
	}
}

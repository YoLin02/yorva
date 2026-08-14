package events

import (
	"testing"
	"time"
)

func TestSlowSubscriberDoesNotBlockPublisher(t *testing.T) {
	broker := NewBroker()
	slow := broker.Subscribe()
	defer slow.Close()
	fast := broker.Subscribe()
	defer fast.Close()

	for i := 0; i < subscriberBuffer+4; i++ {
		broker.Publish(Event{ID: "event", Type: "test", OccurredAt: time.Now()})
	}

	select {
	case <-fast.Events:
	case <-time.After(time.Second):
		t.Fatal("publisher blocked a separate subscriber")
	}
}

func TestSubscriptionCloseIsIdempotent(t *testing.T) {
	broker := NewBroker()
	subscription := broker.Subscribe()
	if broker.SubscriberCount() != 1 {
		t.Fatalf("subscriber count = %d, want 1", broker.SubscriberCount())
	}
	subscription.Close()
	subscription.Close()
	if broker.SubscriberCount() != 0 {
		t.Fatalf("subscriber count = %d, want 0", broker.SubscriberCount())
	}
}

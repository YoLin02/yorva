package events

import (
	"encoding/json"
	"sync"
	"time"
)

const subscriberBuffer = 16

type Event struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	OccurredAt time.Time       `json:"occurredAt"`
	Data       json.RawMessage `json:"data"`
}

type Broker struct {
	mu          sync.Mutex
	nextID      uint64
	nextEvent   uint64
	subscribers map[uint64]chan Event
}

type Subscription struct {
	Events <-chan Event
	close  func()
	once   sync.Once
}

func NewBroker() *Broker {
	return &Broker{subscribers: make(map[uint64]chan Event)}
}

func (b *Broker) Subscribe() *Subscription {
	b.mu.Lock()
	id := b.nextID
	b.nextID++
	channel := make(chan Event, subscriberBuffer)
	b.subscribers[id] = channel
	b.mu.Unlock()

	return &Subscription{
		Events: channel,
		close: func() {
			b.mu.Lock()
			delete(b.subscribers, id)
			b.mu.Unlock()
		},
	}
}

func (s *Subscription) Close() {
	s.once.Do(s.close)
}

func (b *Broker) Publish(event Event) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if event.ID == "" {
		b.nextEvent++
		event.ID = formatEventID(b.nextEvent)
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}
	for _, subscriber := range b.subscribers {
		select {
		case subscriber <- event:
		default:
		}
	}
}

func (b *Broker) SubscriberCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.subscribers)
}

package app

import (
	"errors"
	"sync"
	"time"
)

const (
	channelQRMaxPayload = 8 * 1024
	channelSessionMin   = 20
	channelSessionMax   = 128
)

var (
	errChannelQRUnavailable = errors.New("channel QR payload is unavailable")
	errChannelSession       = errors.New("channel session is invalid")
)

type ChannelQRPayload struct {
	Data      []byte
	ExpiresAt time.Time
}

type channelQREntry struct {
	owner     string
	payload   []byte
	expiresAt time.Time
}

type channelQRBroker struct {
	mu      sync.Mutex
	entries map[string]channelQREntry
	now     func() time.Time
}

func newChannelQRBroker() *channelQRBroker {
	return &channelQRBroker{entries: make(map[string]channelQREntry), now: func() time.Time { return time.Now().UTC() }}
}

func (b *channelQRBroker) Publish(operationID, owner string, payload []byte, expiresAt time.Time) error {
	if b == nil || operationID == "" || !validChannelSession(owner) || len(payload) == 0 || len(payload) > channelQRMaxPayload || !expiresAt.After(b.now()) {
		return errChannelQRUnavailable
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sweepLocked()
	if previous, ok := b.entries[operationID]; ok {
		clearBytes(previous.payload)
	}
	b.entries[operationID] = channelQREntry{owner: owner, payload: append([]byte(nil), payload...), expiresAt: expiresAt}
	return nil
}

func (b *channelQRBroker) Get(operationID, owner string) (ChannelQRPayload, error) {
	if b == nil || operationID == "" || !validChannelSession(owner) {
		return ChannelQRPayload{}, errChannelSession
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sweepLocked()
	entry, ok := b.entries[operationID]
	if !ok || entry.owner != owner {
		return ChannelQRPayload{}, errChannelQRUnavailable
	}
	return ChannelQRPayload{Data: append([]byte(nil), entry.payload...), ExpiresAt: entry.expiresAt}, nil
}

func (b *channelQRBroker) Delete(operationID string) {
	if b == nil || operationID == "" {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if entry, ok := b.entries[operationID]; ok {
		clearBytes(entry.payload)
		delete(b.entries, operationID)
	}
}

func (b *channelQRBroker) sweepLocked() {
	now := b.now()
	for operationID, entry := range b.entries {
		if !entry.expiresAt.After(now) {
			clearBytes(entry.payload)
			delete(b.entries, operationID)
		}
	}
}

func validChannelSession(value string) bool {
	if len(value) < channelSessionMin || len(value) > channelSessionMax {
		return false
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '-' || char == '_' {
			continue
		}
		return false
	}
	return true
}

func clearBytes(value []byte) {
	for index := range value {
		value[index] = 0
	}
}

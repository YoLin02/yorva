package backupmanagement

import (
	"errors"
	"os/user"
	"regexp"
	"sync"
	"time"
)

const (
	destinationGrantTTL  = 2 * time.Minute
	maxDestinationGrants = 8
)

var (
	ErrDestinationGrantInvalid = errors.New("backup destination grant is invalid")
	ErrDestinationGrantBusy    = errors.New("backup destination grant capacity reached")
	destinationRefPattern      = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)
	operationIDPattern         = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$`)
)

type destinationGrant struct {
	runtimeID   string
	userID      string
	destination LocalDestination
	expiresAt   time.Time
}

// ConsumedDestination is visible only inside the Hermes adapter composition.
// IndexPath must never be projected through an ordinary API.
type ConsumedDestination struct {
	Destination LocalDestination
	IndexPath   string
}

// DestinationRegistry is the daemon-session-local authority for native Save
// Backup selections. It contains no generic path lookup and is never persisted.
type DestinationRegistry struct {
	mu        sync.Mutex
	sessionID string
	userID    string
	now       func() time.Time
	grants    map[string]destinationGrant
}

func NewDestinationRegistry(sessionID string) (*DestinationRegistry, error) {
	current, err := user.Current()
	if err != nil || current.Uid == "" || sessionID == "" {
		return nil, ErrDestinationGrantInvalid
	}
	return &DestinationRegistry{
		sessionID: sessionID,
		userID:    current.Uid,
		now:       func() time.Time { return time.Now().UTC() },
		grants:    make(map[string]destinationGrant),
	}, nil
}

// Grant accepts a path only from the private parent-control channel.
func (r *DestinationRegistry) Grant(destinationRef, runtimeID, path string) error {
	if r == nil || !destinationRefPattern.MatchString(destinationRef) || runtimeID != RuntimeKind {
		return ErrDestinationGrantInvalid
	}
	destination, err := InspectLocalDestination(path)
	if err != nil {
		return ErrDestinationGrantInvalid
	}
	current, err := user.Current()
	if err != nil || current.Uid != r.userID {
		return ErrDestinationGrantInvalid
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.cleanupLocked()
	if len(r.grants) >= maxDestinationGrants {
		return ErrDestinationGrantBusy
	}
	if _, exists := r.grants[destinationRef]; exists {
		return ErrDestinationGrantInvalid
	}
	r.grants[destinationRef] = destinationGrant{
		runtimeID: runtimeID, userID: r.userID, destination: destination,
		expiresAt: r.now().Add(destinationGrantTTL),
	}
	return nil
}

// Consume atomically gives one Operation the selected destination. A failed or
// repeated lookup never reveals whether a sensitive path exists.
func (r *DestinationRegistry) Consume(destinationRef, runtimeID, operationID string) (ConsumedDestination, error) {
	if r == nil || !destinationRefPattern.MatchString(destinationRef) || runtimeID != RuntimeKind ||
		!operationIDPattern.MatchString(operationID) {
		return ConsumedDestination{}, ErrDestinationGrantInvalid
	}
	current, err := user.Current()
	if err != nil || current.Uid != r.userID {
		return ConsumedDestination{}, ErrDestinationGrantInvalid
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.cleanupLocked()
	grant, exists := r.grants[destinationRef]
	if !exists || grant.runtimeID != runtimeID || grant.userID != r.userID {
		return ConsumedDestination{}, ErrDestinationGrantInvalid
	}
	delete(r.grants, destinationRef)
	return ConsumedDestination{Destination: grant.destination, IndexPath: grant.destination.path}, nil
}

func (r *DestinationRegistry) cleanupLocked() {
	now := r.now()
	for ref, grant := range r.grants {
		if !now.Before(grant.expiresAt) {
			delete(r.grants, ref)
		}
	}
}

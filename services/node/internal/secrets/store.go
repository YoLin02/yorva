// Package secrets stores YORVA-owned secrets behind an operating-system-backed
// boundary. It intentionally has no persistence, HTTP, Desktop, or Runtime
// integration of its own.
package secrets

import (
	"context"
	"errors"
	"runtime"
	"sync"
)

const (
	maxReferenceLength = 128
	maxSecretBytes     = 64 * 1024
)

var (
	ErrUnsupported      = errors.New("secretstore: unsupported operating system")
	ErrInvalidRoot      = errors.New("secretstore: invalid storage root")
	ErrInvalidReference = errors.New("secretstore: invalid reference")
	ErrNotFound         = errors.New("secretstore: secret not found")
	ErrAlreadyExists    = errors.New("secretstore: secret already exists")
	ErrUnsafeStorage    = errors.New("secretstore: unsafe storage boundary")
	ErrUnavailable      = errors.New("secretstore: secure storage unavailable")
	ErrCorrupt          = errors.New("secretstore: protected value is invalid")
	ErrCanceled         = errors.New("secretstore: operation canceled")
	ErrSecretInvalid    = errors.New("secretstore: secret is empty or too large")
	ErrGenerationFailed = errors.New("secretstore: secret generation failed")
	ErrConsumerFailed   = errors.New("secretstore: secret consumer failed")
)

// Reference is a non-secret, persistable identifier. Store methods validate
// its closed filename-safe grammar before using it.
type Reference string

// Metadata is the only ordinary read projection. It never contains plaintext,
// protected bytes, a filesystem path, or OS error details.
type Metadata struct {
	Reference  Reference
	Configured bool
}

// Generator creates one secret in memory. Generate and Rotate take ownership
// of the returned slice and wipe it before returning.
type Generator func() ([]byte, error)

// Store is safe for concurrent in-process use. root is supplied by the daemon
// from its established YORVA-owned data directory; this package does not infer
// authority from environment variables.
type Store struct {
	dir string
	mu  sync.Mutex
}

// New opens the OS-backed SecretStore below an existing authoritative YORVA
// data directory. It never falls back to plaintext storage.
func New(root string) (*Store, error) {
	return newStore(root)
}

// Inspect reports only whether protected material is configured.
func (s *Store) Inspect(ctx context.Context, ref Reference) (Metadata, error) {
	if s == nil || !validReference(ref) {
		return Metadata{}, ErrInvalidReference
	}
	if canceled(ctx) {
		return Metadata{}, ErrCanceled
	}
	s.mu.Lock()
	configured, err := s.inspect(ref)
	s.mu.Unlock()
	if err != nil {
		return Metadata{}, err
	}
	return Metadata{Reference: ref, Configured: configured}, nil
}

// Get decrypts a secret only for the duration of use. The plaintext buffer is
// wiped immediately after use returns and must not be retained by the callback.
func (s *Store) Get(ctx context.Context, ref Reference, use func([]byte) error) error {
	if s == nil || !validReference(ref) {
		return ErrInvalidReference
	}
	if use == nil {
		return ErrConsumerFailed
	}
	if canceled(ctx) {
		return ErrCanceled
	}
	s.mu.Lock()
	secret, err := s.get(ref)
	s.mu.Unlock()
	if err != nil {
		return err
	}
	defer wipe(secret)
	if canceled(ctx) {
		return ErrCanceled
	}
	if err := use(secret); err != nil {
		return ErrConsumerFailed
	}
	return nil
}

// Put creates a protected value without overwriting an existing reference.
// The caller retains ownership of secret; Store protects and wipes its copy.
func (s *Store) Put(ctx context.Context, ref Reference, secret []byte) error {
	if s == nil || !validReference(ref) {
		return ErrInvalidReference
	}
	owned, err := ownedSecret(secret)
	if err != nil {
		return err
	}
	defer wipe(owned)
	if canceled(ctx) {
		return ErrCanceled
	}
	s.mu.Lock()
	err = s.put(ref, owned)
	s.mu.Unlock()
	return err
}

// Generate creates and stores a new protected value without returning its
// plaintext. Generator errors are deliberately normalized.
func (s *Store) Generate(ctx context.Context, ref Reference, generate Generator) error {
	if s == nil || !validReference(ref) {
		return ErrInvalidReference
	}
	if generate == nil {
		return ErrGenerationFailed
	}
	if canceled(ctx) {
		return ErrCanceled
	}
	secret, err := generate()
	if err != nil {
		wipe(secret)
		if canceled(ctx) {
			return ErrCanceled
		}
		return ErrGenerationFailed
	}
	defer wipe(secret)
	if len(secret) == 0 || len(secret) > maxSecretBytes {
		return ErrSecretInvalid
	}
	if canceled(ctx) {
		return ErrCanceled
	}
	s.mu.Lock()
	err = s.put(ref, secret)
	s.mu.Unlock()
	return err
}

// Rotate creates a new protected reference while retaining current. This
// mirrors ADR-0013: existing backup identities are never silently replaced or
// deleted while an artifact may still reference them.
func (s *Store) Rotate(ctx context.Context, current, next Reference, generate Generator) error {
	if s == nil || !validReference(current) || !validReference(next) || current == next {
		return ErrInvalidReference
	}
	if generate == nil {
		return ErrGenerationFailed
	}
	if canceled(ctx) {
		return ErrCanceled
	}
	secret, err := generate()
	if err != nil {
		wipe(secret)
		if canceled(ctx) {
			return ErrCanceled
		}
		return ErrGenerationFailed
	}
	defer wipe(secret)
	if len(secret) == 0 || len(secret) > maxSecretBytes {
		return ErrSecretInvalid
	}

	s.mu.Lock()
	configured, err := s.inspect(current)
	if err == nil && !configured {
		err = ErrNotFound
	}
	if err == nil {
		err = s.put(next, secret)
	}
	s.mu.Unlock()
	return err
}

// Delete removes only the exact protected reference. It does not claim secure
// physical erasure and policy callers must first prove the reference is unused.
func (s *Store) Delete(ctx context.Context, ref Reference) error {
	if s == nil || !validReference(ref) {
		return ErrInvalidReference
	}
	if canceled(ctx) {
		return ErrCanceled
	}
	s.mu.Lock()
	err := s.delete(ref)
	s.mu.Unlock()
	return err
}

func validReference(ref Reference) bool {
	value := string(ref)
	if len(value) == 0 || len(value) > maxReferenceLength {
		return false
	}
	for index := 0; index < len(value); index++ {
		char := value[index]
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') {
			continue
		}
		if char == '-' && index > 0 && index < len(value)-1 {
			continue
		}
		return false
	}
	return true
}

func ownedSecret(secret []byte) ([]byte, error) {
	if len(secret) == 0 || len(secret) > maxSecretBytes {
		return nil, ErrSecretInvalid
	}
	return append([]byte(nil), secret...), nil
}

func canceled(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

func wipe(value []byte) {
	for index := range value {
		value[index] = 0
	}
	runtime.KeepAlive(value)
}

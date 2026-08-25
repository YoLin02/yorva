package secrets

import (
	"errors"
	"strings"
	"testing"
)

func TestReferenceGrammar(t *testing.T) {
	for _, value := range []Reference{"backup-device-a1", "a", "0", "abc-123"} {
		if !validReference(value) {
			t.Fatalf("reference %q rejected", value)
		}
	}
	for _, value := range []Reference{
		"", "-leading", "trailing-", "UPPER", "under_score", "../escape", `name:stream`,
		Reference(strings.Repeat("a", maxReferenceLength+1)),
	} {
		if validReference(value) {
			t.Fatalf("reference %q accepted", value)
		}
	}
}

func TestOwnedSecretRejectsInvalidLength(t *testing.T) {
	if _, err := ownedSecret(nil); !errors.Is(err, ErrSecretInvalid) {
		t.Fatalf("empty secret error = %v", err)
	}
	if _, err := ownedSecret(make([]byte, maxSecretBytes+1)); !errors.Is(err, ErrSecretInvalid) {
		t.Fatalf("oversized secret error = %v", err)
	}
}

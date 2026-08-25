package mcpmanagement

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestProfileScopeIsExact(t *testing.T) {
	for _, profileID := range []string{"default", "alice", "2nd_profile", "team-one"} {
		scope, err := NewProfileScope(profileID)
		if err != nil {
			t.Fatalf("NewProfileScope(%q): %v", profileID, err)
		}
		if scope.ProfileID() != profileID {
			t.Fatalf("ProfileID() = %q, want exact %q", scope.ProfileID(), profileID)
		}
	}

	for _, profileID := range []string{"", " Default ", "ALICE", "../alice", "a/b", "test", strings.Repeat("a", maxProfileIDLength+1)} {
		if _, err := NewProfileScope(profileID); !errors.Is(err, ErrProfileScopeInvalid) {
			t.Fatalf("NewProfileScope(%q) error = %v, want ErrProfileScopeInvalid", profileID, err)
		}
	}
}

func TestCredentialClassAndStatusValidation(t *testing.T) {
	noAuth := Selection{descriptor: testDescriptor(CredentialClassNone)}
	if err := noAuth.ValidateCredentialStatus(CredentialStatusNotRequired); err != nil {
		t.Fatalf("no-auth NOT_REQUIRED: %v", err)
	}
	if err := noAuth.ValidateCredential(nil); err != nil {
		t.Fatalf("no-auth empty credential: %v", err)
	}
	if err := noAuth.ValidateCredential([]byte("unexpected")); !errors.Is(err, ErrCredentialInvalid) {
		t.Fatalf("no-auth secret error = %v, want ErrCredentialInvalid", err)
	}
	if err := noAuth.ValidateCredentialStatus(CredentialStatusConfigured); !errors.Is(err, ErrCredentialStatusInvalid) {
		t.Fatalf("no-auth configured status error = %v, want ErrCredentialStatusInvalid", err)
	}

	bearer := Selection{descriptor: testDescriptor(CredentialClassStaticBearer)}
	for _, status := range []CredentialStatus{CredentialStatusNotConfigured, CredentialStatusConfigured, CredentialStatusUnknown} {
		if err := bearer.ValidateCredentialStatus(status); err != nil {
			t.Fatalf("bearer status %q: %v", status, err)
		}
	}
	for _, credential := range [][]byte{nil, {}, []byte("has space"), []byte("line\nbreak"), bytes.Repeat([]byte{'x'}, maxStaticBearerBytes+1)} {
		if err := bearer.ValidateCredential(credential); !errors.Is(err, ErrCredentialInvalid) {
			t.Fatalf("ValidateCredential(%q) error = %v, want ErrCredentialInvalid", credential, err)
		}
	}
}

func TestCredentialIsNotRetainedOrIncludedInErrors(t *testing.T) {
	selection := Selection{descriptor: testDescriptor(CredentialClassStaticBearer)}
	secret := []byte("mcp-secret-canary")
	if err := selection.ValidateCredential(secret); err != nil {
		t.Fatalf("ValidateCredential(valid): %v", err)
	}
	if got := fmt.Sprintf("%#v", selection); strings.Contains(got, string(secret)) {
		t.Fatalf("selection retained secret: %s", got)
	}

	invalid := []byte("mcp-secret-canary\n")
	err := selection.ValidateCredential(invalid)
	if !errors.Is(err, ErrCredentialInvalid) {
		t.Fatalf("ValidateCredential(invalid) error = %v, want ErrCredentialInvalid", err)
	}
	if strings.Contains(err.Error(), "mcp-secret-canary") {
		t.Fatalf("credential error exposed secret: %v", err)
	}
}

package app

import (
	"context"
	"errors"
	"testing"
)

func TestClassifyHermesProfileErrorPreservesTimeoutAndUnrecognized(t *testing.T) {
	if err := classifyHermesProfileError(context.DeadlineExceeded); !errors.Is(err, ErrInstanceOperationTimedOut) {
		t.Fatalf("timeout = %v", err)
	}
	if err := classifyHermesProfileError(ErrInstanceOutputUnrecognized); !errors.Is(err, ErrInstanceOutputUnrecognized) {
		t.Fatalf("unrecognized = %v", err)
	}
	if err := classifyHermesProfileError(errors.New("boom")); !errors.Is(err, ErrInstanceQueryFailed) {
		t.Fatalf("generic = %v", err)
	}
}

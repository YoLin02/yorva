package hermes

import (
	"context"
	"errors"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
	"testing"
)

func TestClassifyHermesProfileErrorPreservesTimeoutAndUnrecognized(t *testing.T) {
	if err := normalizeInstanceError(context.DeadlineExceeded); !errors.Is(err, yorvaruntime.ErrInstanceOperationTimedOut) {
		t.Fatalf("timeout = %v", err)
	}
	if err := normalizeInstanceError(yorvaruntime.ErrInstanceOutputUnrecognized); !errors.Is(err, yorvaruntime.ErrInstanceOutputUnrecognized) {
		t.Fatalf("unrecognized = %v", err)
	}
	if err := normalizeInstanceError(errors.New("boom")); !errors.Is(err, yorvaruntime.ErrInstanceInventoryFailed) {
		t.Fatalf("generic = %v", err)
	}
}

package instance

import "testing"

func TestInstanceIDIsOpaqueAndNotANativeName(t *testing.T) {
	id := NewID("abc")
	if id != "inst_abc" {
		t.Fatalf("NewID() = %q", id)
	}
	row := Instance{ID: id, NativeID: "coder", Name: "coder"}
	if row.ID == row.NativeID {
		t.Fatal("instanceId must not equal nativeId")
	}
}

func TestLifecycleCapabilityIsFalse(t *testing.T) {
	if (Instance{}).LifecycleSupported() {
		t.Fatal("Phase 4 lifecycle capability must be false")
	}
}

func TestAvailabilityStates(t *testing.T) {
	if Available != "AVAILABLE" || Missing != "MISSING" || Unknown != "UNKNOWN" {
		t.Fatalf("availability constants drifted: %s %s %s", Available, Missing, Unknown)
	}
}

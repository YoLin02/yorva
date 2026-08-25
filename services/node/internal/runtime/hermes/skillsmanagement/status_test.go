package skillsmanagement

import "testing"

func TestNormalizedStatesAreClosed(t *testing.T) {
	for _, state := range []InstalledState{InstalledStateInstalled, InstalledStateNotInstalled, InstalledStateUnknown} {
		if !state.Valid() {
			t.Fatalf("InstalledState %q must be valid", state)
		}
	}
	if InstalledState("installed").Valid() || InstalledState("").Valid() {
		t.Fatal("unknown InstalledState must fail closed")
	}

	for _, state := range []EnabledState{EnabledStateEnabled, EnabledStateDisabled, EnabledStateUnknown} {
		if !state.Valid() {
			t.Fatalf("EnabledState %q must be valid", state)
		}
	}
	if EnabledState("enabled").Valid() || EnabledState("").Valid() {
		t.Fatal("unknown EnabledState must fail closed")
	}

	for _, state := range []ScanState{
		ScanStateClean,
		ScanStateWarning,
		ScanStateBlocked,
		ScanStateNotScanned,
		ScanStateUnknown,
	} {
		if !state.Valid() {
			t.Fatalf("ScanState %q must be valid", state)
		}
	}
	if ScanState("clean").Valid() || ScanState("").Valid() {
		t.Fatal("unknown ScanState must fail closed")
	}
}

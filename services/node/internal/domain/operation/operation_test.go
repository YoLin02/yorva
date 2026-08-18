package operation

import "testing"

func TestValidTransition(t *testing.T) {
	tests := []struct {
		from Status
		to   Status
		want bool
	}{
		{StatusPending, StatusRunning, true},
		{StatusPending, StatusFailed, true},
		{StatusPending, StatusCancelled, true},
		{StatusPending, StatusSucceeded, false},
		{StatusRunning, StatusSucceeded, true},
		{StatusRunning, StatusFailed, true},
		{StatusRunning, StatusCancelled, true},
		{StatusRunning, StatusPending, false},
		{StatusSucceeded, StatusFailed, false},
		{StatusFailed, StatusRunning, false},
		{StatusCancelled, StatusPending, false},
		{StatusPending, StatusPending, false},
	}
	for _, test := range tests {
		if got := ValidTransition(test.from, test.to); got != test.want {
			t.Fatalf("ValidTransition(%s, %s) = %v, want %v", test.from, test.to, got, test.want)
		}
	}
}

func TestIsTerminal(t *testing.T) {
	if IsTerminal(StatusPending) || IsTerminal(StatusRunning) {
		t.Fatal("active statuses must not be terminal")
	}
	if !IsTerminal(StatusSucceeded) || !IsTerminal(StatusFailed) || !IsTerminal(StatusCancelled) {
		t.Fatal("completed statuses must be terminal")
	}
}

func TestValidProjectionRepair(t *testing.T) {
	if !ValidProjectionRepair(StatusFailed, StatusSucceeded) || !ValidProjectionRepair(StatusCancelled, StatusRunning) {
		t.Fatal("filesystem authority must be able to repair a wrongly terminal Operation")
	}
	if ValidProjectionRepair(StatusRunning, StatusSucceeded) || ValidProjectionRepair(StatusFailed, StatusPending) {
		t.Fatal("projection repair must not replace ordinary transitions or reopen as PENDING")
	}
	if !ValidStatusChange(StatusFailed, StatusSucceeded) || ValidStatusChange(StatusSucceeded, StatusFailed) {
		t.Fatal("ValidStatusChange must allow repair and still forbid rolling success back to failed")
	}
}

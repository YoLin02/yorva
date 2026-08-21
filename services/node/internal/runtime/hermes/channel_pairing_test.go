package hermes

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/YoLin02/yorva/services/node/internal/domain/channel"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestChannelPairingStatusUsesExactProfileAndCountsWeixin(t *testing.T) {
	manager := NewChannelManager()
	var gotArgs []string
	manager.runPairing = func(_ context.Context, _ string, args []string) commandResult {
		gotArgs = append([]string(nil), args...)
		return commandResult{stdout: `
Pending Pairing Requests (2):
Platform  Request ID        Code      Sender
--------  ----------------  --------  ------
weixin    0123456789abcdef  sender-one          Alice                1m ago
telegram  fedcba9876543210  sender-two          Bob                  2m ago

Approve with: hermes pairing approve <platform> <request-id>
The code the bot DM'd the user also works if they relay it.
`}
	}
	status, err := manager.PairingStatus(context.Background(), pairingInstallation(t), "work", channel.Weixin)
	if err != nil {
		t.Fatalf("PairingStatus() error = %v", err)
	}
	if status.PendingCount != 1 {
		t.Fatalf("PendingCount = %d, want 1", status.PendingCount)
	}
	want := []string{"--profile", "work", "pairing", "list"}
	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}

func TestChannelPairingStatusRejectsUnrecognizedOutput(t *testing.T) {
	manager := NewChannelManager()
	manager.runPairing = func(context.Context, string, []string) commandResult {
		return commandResult{stdout: "surprising output"}
	}
	_, err := manager.PairingStatus(context.Background(), pairingInstallation(t), "default", channel.Weixin)
	if !errors.Is(err, yorvaruntime.ErrChannelPairingQuery) {
		t.Fatalf("error = %v, want ErrChannelPairingQuery", err)
	}
}

func TestChannelApprovePairingValidatesAndNormalizesCode(t *testing.T) {
	manager := NewChannelManager()
	var gotArgs []string
	manager.runPairing = func(_ context.Context, _ string, args []string) commandResult {
		gotArgs = append([]string(nil), args...)
		return commandResult{stdout: "Approved! User sender on weixin can now use the bot~"}
	}
	code := []byte(" abcd2345 ")
	if err := manager.ApprovePairing(context.Background(), pairingInstallation(t), "default", channel.Weixin, code); err != nil {
		t.Fatalf("ApprovePairing() error = %v", err)
	}
	want := []string{"pairing", "approve", "weixin", "ABCD2345"}
	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
	for _, value := range code {
		if value != 0 {
			t.Fatal("caller pairing code was not cleared")
		}
	}
}

func TestChannelApprovePairingMapsExpectedFailures(t *testing.T) {
	tests := []struct {
		name   string
		result commandResult
		want   error
	}{
		{name: "expired", result: commandResult{stdout: "Pairing request or code 'ABCD2345' not found or expired for platform 'weixin'."}, want: yorvaruntime.ErrChannelPairingCode},
		{name: "locked", result: commandResult{stdout: "Platform 'weixin' is locked out after too many failed approval attempts."}, want: yorvaruntime.ErrChannelPairingLocked},
		{name: "unknown", result: commandResult{exitCode: 1, stderr: "private detail"}, want: yorvaruntime.ErrChannelPairingApproval},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager := NewChannelManager()
			manager.runPairing = func(context.Context, string, []string) commandResult { return test.result }
			err := manager.ApprovePairing(context.Background(), pairingInstallation(t), "default", channel.Weixin, []byte("ABCD2345"))
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func pairingInstallation(t *testing.T) yorvaruntime.ChannelInstallation {
	t.Helper()
	return yorvaruntime.ChannelInstallation{Executable: filepath.Join(t.TempDir(), "hermes.exe"), Version: channelCredentialVersion}
}

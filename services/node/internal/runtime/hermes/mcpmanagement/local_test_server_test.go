package mcpmanagement

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

func TestLocalTestServerCompletesReviewedMCPHandshake(t *testing.T) {
	server, err := StartLocalTestServer()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	if host, _, splitErr := net.SplitHostPort(server.listener.Addr().String()); splitErr != nil || host != "127.0.0.1" {
		t.Fatalf("listener = %q, %v", server.listener.Addr(), splitErr)
	}
	selection, err := NewRegistry().Resolve(YORVATestPresetID)
	if err != nil {
		t.Fatal(err)
	}
	scope, err := NewProfileScope("default")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 26, 18, 0, 0, 0, time.UTC)
	result, err := ProbeReviewedHTTPS(context.Background(), selection, scope, nil, server.Client(NewReviewedHTTPSClient()), func() time.Time { return now })
	if err != nil || result.Outcome() != ProbeOutcomeReady || result.ObservedAt() != now || len(result.ToolIDs()) != 1 || result.ToolIDs()[0] != YORVATestToolID {
		t.Fatalf("probe = %#v, %v", result, err)
	}
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := ProbeReviewedHTTPS(context.Background(), selection, scope, nil, server.Client(NewReviewedHTTPSClient()), time.Now); !errors.Is(err, ErrReviewedProbeTransport) {
		t.Fatalf("probe after close error = %v", err)
	}
}

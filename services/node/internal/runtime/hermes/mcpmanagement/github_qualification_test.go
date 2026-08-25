package mcpmanagement

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestGitHubQualificationExactHandshake(t *testing.T) {
	t.Parallel()
	const token = "qualification-test-token"
	var requestNumber atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		step := requestNumber.Add(1)
		if request.Method != http.MethodPost || request.URL.Path != "/mcp/x/repos/readonly" {
			t.Errorf("unexpected request target: %s %s", request.Method, request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer "+token {
			t.Error("missing request-lifetime bearer credential")
		}
		if request.Header.Get("MCP-Protocol-Version") != githubQualificationProtocolVersion {
			t.Error("missing fixed MCP protocol version")
		}
		writer.Header().Set("Content-Type", "application/json")
		switch step {
		case 1:
			writer.Header().Set("Mcp-Session-Id", "fixed-session")
			_, _ = writer.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-06-18","capabilities":{"tools":{}},"serverInfo":{"name":"fixture","version":"1"}}}`))
		case 2:
			if request.Header.Get("Mcp-Session-Id") != "fixed-session" {
				t.Error("initialized notification omitted session ID")
			}
			writer.WriteHeader(http.StatusAccepted)
		case 3:
			if request.Header.Get("Mcp-Session-Id") != "fixed-session" {
				t.Error("tools/list omitted session ID")
			}
			_, _ = writer.Write(exactToolsResponse(t, nil))
		default:
			t.Errorf("unexpected request number %d", step)
		}
	}))
	defer server.Close()

	result, err := qualifyGitHubReposReadOnly(context.Background(), []byte(token), server.URL+"/mcp/x/repos/readonly", server.Client())
	if err != nil {
		t.Fatalf("qualification failed: %v", err)
	}
	if !result.Ready() || result.ToolCount() != 14 || result.SourceRevision() != GitHubQualificationSourceRevision {
		t.Fatalf("unexpected redacted result: %#v", result)
	}
	if requestNumber.Load() != 3 {
		t.Fatalf("request count = %d, want 3", requestNumber.Load())
	}
	if strings.Contains(fmt.Sprint(result), token) || strings.Contains(fmt.Sprint(err), token) {
		t.Fatal("credential leaked through result or error")
	}
}

func TestGitHubQualificationRejectsInventoryDrift(t *testing.T) {
	t.Parallel()
	server := newQualificationServer(t, func(writer http.ResponseWriter) {
		_, _ = writer.Write(exactToolsResponse(t, []string{"unreviewed_tool"}))
	})
	defer server.Close()

	_, err := qualifyGitHubReposReadOnly(context.Background(), []byte("test-token"), server.URL, server.Client())
	if !errors.Is(err, ErrGitHubQualificationMismatch) {
		t.Fatalf("error = %v, want inventory mismatch", err)
	}
}

func TestGitHubQualificationRejectsRedirectWithoutForwardingCredential(t *testing.T) {
	t.Parallel()
	var redirected atomic.Bool
	target := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		redirected.Store(true)
	}))
	defer target.Close()
	source := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Redirect(writer, request, target.URL, http.StatusTemporaryRedirect)
	}))
	defer source.Close()

	_, err := qualifyGitHubReposReadOnly(context.Background(), []byte("test-token"), source.URL, source.Client())
	if !errors.Is(err, ErrGitHubQualificationRedirect) {
		t.Fatalf("error = %v, want redirect rejection", err)
	}
	if redirected.Load() {
		t.Fatal("redirect target was contacted")
	}
}

func TestGitHubQualificationBoundsResponse(t *testing.T) {
	t.Parallel()
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(strings.Repeat("x", maxQualificationResponseBytes+1)))
	}))
	defer server.Close()

	_, err := qualifyGitHubReposReadOnly(context.Background(), []byte("test-token"), server.URL, server.Client())
	if !errors.Is(err, ErrGitHubQualificationLimit) {
		t.Fatalf("error = %v, want response limit", err)
	}
}

func TestGitHubQualificationHonorsCancellation(t *testing.T) {
	t.Parallel()
	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewTLSServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		close(started)
		select {
		case <-request.Context().Done():
		case <-release:
		}
	}))
	defer func() {
		close(release)
		server.Close()
	}()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := qualifyGitHubReposReadOnly(ctx, []byte("test-token"), server.URL, server.Client())
		done <- err
	}()
	<-started
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, ErrGitHubQualificationCanceled) {
			t.Fatalf("error = %v, want cancellation", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("qualification did not stop after cancellation")
	}
}

func TestGitHubQualificationRejectsInvalidCredentialBeforeNetwork(t *testing.T) {
	t.Parallel()
	client := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("network called for invalid credential")
		return nil, nil
	})}
	_, err := qualifyGitHubReposReadOnly(context.Background(), []byte("line\nbreak"), "https://example.invalid", client)
	if !errors.Is(err, ErrGitHubQualificationCredential) {
		t.Fatalf("error = %v, want credential rejection", err)
	}
}

// This is the only live qualification entry point. It is skipped unless an
// operator supplies a request-lifetime token through the environment. It logs
// only bounded, non-account metadata and does not authorize the product registry.
func TestManualGitHubReposReadOnlyQualification(t *testing.T) {
	credential := []byte(os.Getenv("YORVA_GITHUB_MCP_QUALIFICATION_TOKEN"))
	if len(credential) == 0 {
		t.Skip("manual GitHub MCP qualification token not supplied")
	}
	defer clear(credential)
	result, err := QualifyGitHubReposReadOnly(context.Background(), credential)
	if err != nil {
		t.Fatalf("redacted qualification result: %v", err)
	}
	t.Logf("qualification READY; toolCount=%d; sourceRevision=%s", result.ToolCount(), result.SourceRevision())
}

func newQualificationServer(t *testing.T, toolsResponse func(http.ResponseWriter)) *httptest.Server {
	t.Helper()
	var requestNumber atomic.Int32
	return httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch requestNumber.Add(1) {
		case 1:
			writer.Header().Set("Mcp-Session-Id", "fixture-session")
			_, _ = writer.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-06-18","capabilities":{"tools":{}},"serverInfo":{"name":"fixture","version":"1"}}}`))
		case 2:
			writer.WriteHeader(http.StatusAccepted)
		case 3:
			toolsResponse(writer)
		default:
			t.Error("unexpected extra request")
		}
	}))
}

func exactToolsResponse(t *testing.T, extra []string) []byte {
	t.Helper()
	type annotation struct {
		ReadOnlyHint bool `json:"readOnlyHint"`
	}
	type tool struct {
		Name        string     `json:"name"`
		Description string     `json:"description"`
		InputSchema any        `json:"inputSchema"`
		Annotations annotation `json:"annotations"`
	}
	tools := make([]tool, 0, len(githubReposReadOnlyTools)+len(extra))
	for _, name := range append(githubReposReadOnlyTools[:], extra...) {
		tools = append(tools, tool{Name: name, Description: "fixture", InputSchema: map[string]any{"type": "object"}, Annotations: annotation{ReadOnlyHint: true}})
	}
	payload, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"result":  map[string]any{"tools": tools},
	})
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestGitHubQualificationProductionTransportUsesTLS12(t *testing.T) {
	client := newGitHubQualificationHTTPClient()
	transport, ok := client.Transport.(*http.Transport)
	if !ok || transport.TLSClientConfig == nil || transport.TLSClientConfig.MinVersion < tls.VersionTLS12 {
		t.Fatal("qualification transport permits TLS below 1.2")
	}
	if transport.Proxy != nil {
		t.Fatal("qualification transport may forward the bearer credential through an environment proxy")
	}
}

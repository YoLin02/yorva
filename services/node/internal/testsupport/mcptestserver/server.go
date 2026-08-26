// Package mcptestserver provides a deterministic HTTPS MCP endpoint for
// qualification tests. It is test infrastructure and is never registered in
// the product Runtime catalog.
package mcptestserver

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

const (
	FixedEndpoint = "https://mcp-test.yorva.invalid/mcp"
	ToolPing      = "yorva_ping"
	Credential    = "yorva-qualification-token"
	testSessionID = "yorva-test-session"
	requestLimit  = 16 * 1024
)

type Server struct {
	server   *httptest.Server
	requests atomic.Int32
}

func Start(t testing.TB) *Server {
	t.Helper()
	fixture := &Server{}
	fixture.server = httptest.NewTLSServer(http.HandlerFunc(fixture.serveHTTP))
	t.Cleanup(fixture.server.Close)
	return fixture
}

// Client returns a TLS client that routes only the fixed qualification host to
// this test server. The descriptor URL remains fixed and caller-independent.
func (s *Server) Client() *http.Client {
	client := s.server.Client()
	target, _ := url.Parse(s.server.URL)
	client.Transport = rewriteTransport{target: target, base: client.Transport}
	return client
}

func (s *Server) RequestCount() int { return int(s.requests.Load()) }

func (s *Server) serveHTTP(response http.ResponseWriter, request *http.Request) {
	s.requests.Add(1)
	if request.Method != http.MethodPost || request.URL.Path != "/mcp" ||
		!strings.HasPrefix(request.Header.Get("Content-Type"), "application/json") ||
		request.Header.Get("Authorization") != "Bearer "+Credential {
		http.Error(response, "rejected", http.StatusBadRequest)
		return
	}
	body, err := io.ReadAll(io.LimitReader(request.Body, requestLimit+1))
	if err != nil || len(body) == 0 || len(body) > requestLimit {
		http.Error(response, "invalid request", http.StatusBadRequest)
		return
	}
	var envelope struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&envelope) != nil || decoder.Decode(&struct{}{}) != io.EOF || envelope.JSONRPC != "2.0" {
		http.Error(response, "invalid envelope", http.StatusBadRequest)
		return
	}

	response.Header().Set("Content-Type", "application/json")
	switch envelope.Method {
	case "initialize":
		if string(envelope.ID) != "1" || request.Header.Get("Mcp-Session-Id") != "" {
			http.Error(response, "invalid initialize", http.StatusBadRequest)
			return
		}
		response.Header().Set("Mcp-Session-Id", testSessionID)
		_, _ = response.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-06-18","capabilities":{},"serverInfo":{"name":"yorva-mcp-test","version":"1"}}}`))
	case "notifications/initialized":
		if len(envelope.ID) != 0 || request.Header.Get("Mcp-Session-Id") != testSessionID {
			http.Error(response, "invalid session", http.StatusBadRequest)
			return
		}
		response.WriteHeader(http.StatusNoContent)
	case "tools/list":
		if string(envelope.ID) != "2" || request.Header.Get("Mcp-Session-Id") != testSessionID {
			http.Error(response, "invalid session", http.StatusBadRequest)
			return
		}
		_, _ = response.Write([]byte(`{"jsonrpc":"2.0","id":2,"result":{"tools":[{"name":"yorva_ping"}]}}`))
	default:
		http.Error(response, "unknown method", http.StatusBadRequest)
	}
}

type rewriteTransport struct {
	target *url.URL
	base   http.RoundTripper
}

func (transport rewriteTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	clone := request.Clone(request.Context())
	urlCopy := *request.URL
	if request.URL.Scheme == "https" && request.URL.Host == "mcp-test.yorva.invalid" {
		urlCopy.Scheme, urlCopy.Host = transport.target.Scheme, transport.target.Host
	}
	clone.URL = &urlCopy
	return transport.base.RoundTrip(clone)
}

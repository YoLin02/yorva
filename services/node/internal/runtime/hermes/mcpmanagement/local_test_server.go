package mcpmanagement

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	localTestSessionID = "yorva-local-test-session"
	localTestBodyLimit = 16 * 1024
)

// LocalTestServer is the first production-qualified MCP endpoint. It listens
// only on an ephemeral IPv4 loopback port and is owned by the daemon-scoped
// ProfileMCPManager. The reviewed descriptor retains a fixed HTTPS identity;
// only the private client below can route that identity to this listener.
type LocalTestServer struct {
	listener net.Listener
	server   *http.Server
	done     chan error
	once     sync.Once
	closeErr error
}

func StartLocalTestServer() (*LocalTestServer, error) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	fixture := &LocalTestServer{listener: listener, done: make(chan error, 1)}
	fixture.server = &http.Server{
		Handler:           http.HandlerFunc(fixture.serveHTTP),
		ReadHeaderTimeout: 2 * time.Second,
		IdleTimeout:       5 * time.Second,
	}
	go func() {
		fixture.done <- fixture.server.Serve(listener)
	}()
	return fixture, nil
}

// Client wraps the reviewed HTTPS client and redirects only the exact
// YORVA-owned test identity to this server's loopback listener.
func (s *LocalTestServer) Client(base *http.Client) *http.Client {
	if s == nil || s.listener == nil || base == nil {
		return nil
	}
	copy := *base
	copy.Transport = &localTestTransport{
		target: &url.URL{Scheme: "http", Host: s.listener.Addr().String()},
		base:   base.Transport,
	}
	return &copy
}

func (s *LocalTestServer) Close() error {
	if s == nil || s.server == nil {
		return nil
	}
	s.once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		s.closeErr = s.server.Shutdown(ctx)
		serveErr := <-s.done
		if s.closeErr == nil && serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			s.closeErr = serveErr
		}
	})
	return s.closeErr
}

func (s *LocalTestServer) serveHTTP(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost || request.URL.Path != "/mcp" ||
		!strings.HasPrefix(request.Header.Get("Content-Type"), "application/json") ||
		request.Header.Get("MCP-Protocol-Version") != "2025-06-18" ||
		request.Header.Get("Authorization") != "" {
		http.Error(response, "rejected", http.StatusBadRequest)
		return
	}
	body, err := io.ReadAll(io.LimitReader(request.Body, localTestBodyLimit+1))
	if err != nil || len(body) == 0 || len(body) > localTestBodyLimit {
		http.Error(response, "invalid request", http.StatusBadRequest)
		return
	}
	var envelope struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params"`
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&envelope) != nil || !errors.Is(decoder.Decode(&struct{}{}), io.EOF) || envelope.JSONRPC != "2.0" {
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
		response.Header().Set("Mcp-Session-Id", localTestSessionID)
		_, _ = response.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-06-18","capabilities":{},"serverInfo":{"name":"yorva-mcp-test","version":"1"}}}`))
	case "notifications/initialized":
		if len(envelope.ID) != 0 || request.Header.Get("Mcp-Session-Id") != localTestSessionID {
			http.Error(response, "invalid session", http.StatusBadRequest)
			return
		}
		response.WriteHeader(http.StatusNoContent)
	case "tools/list":
		if string(envelope.ID) != "2" || request.Header.Get("Mcp-Session-Id") != localTestSessionID {
			http.Error(response, "invalid session", http.StatusBadRequest)
			return
		}
		_, _ = response.Write([]byte(`{"jsonrpc":"2.0","id":2,"result":{"tools":[{"name":"yorva_ping"}]}}`))
	default:
		http.Error(response, "unknown method", http.StatusBadRequest)
	}
}

type localTestTransport struct {
	target *url.URL
	base   http.RoundTripper
}

func (transport *localTestTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if transport == nil || transport.target == nil || transport.base == nil || request == nil || request.URL == nil {
		return nil, ErrReviewedProbeTransport
	}
	clone := request.Clone(request.Context())
	urlCopy := *request.URL
	if request.URL.Scheme == "https" && request.URL.Host == "mcp-test.yorva.invalid" && request.URL.Path == "/mcp" {
		urlCopy.Scheme, urlCopy.Host = transport.target.Scheme, transport.target.Host
	}
	clone.URL = &urlCopy
	return transport.base.RoundTrip(clone)
}

func (transport *localTestTransport) CloseIdleConnections() {
	if closer, ok := transport.base.(interface{ CloseIdleConnections() }); ok {
		closer.CloseIdleConnections()
	}
}

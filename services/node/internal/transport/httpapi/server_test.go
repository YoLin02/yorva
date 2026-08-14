package httpapi

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/domain/node"
	"github.com/YoLin02/yorva/services/node/internal/events"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

const testToken = "test-token"

var testNode = node.Node{
	ID:           "node_test",
	Name:         "TEST-NODE",
	Hostname:     "TEST-NODE",
	Platform:     "windows",
	Architecture: "amd64",
	NodeVersion:  "0.0.0-test",
	CreatedAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	UpdatedAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
}

type fakeRuntimeDiscovery struct {
	result yorvaruntime.Discovery
	err    error
	ctx    chan context.Context
}

func (d fakeRuntimeDiscovery) Detect(ctx context.Context, _ yorvaruntime.Kind) (yorvaruntime.Discovery, error) {
	if d.ctx != nil {
		d.ctx <- ctx
	}
	return d.result, d.err
}

func newTestHandler(services ...RuntimeDiscoveryService) http.Handler {
	var service RuntimeDiscoveryService = fakeRuntimeDiscovery{result: yorvaruntime.Discovery{
		RuntimeKind:    "hermes",
		State:          yorvaruntime.DiscoverySupported,
		DetectedAt:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		SupportedRange: ">=0.19.0 <0.20.0",
	}}
	if len(services) == 1 {
		service = services[0]
	}
	return NewHandler(testToken, testNode, events.NewBroker(), service)
}

func TestHealthIsMinimalAndUnauthenticated(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	response := httptest.NewRecorder()

	newTestHandler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	for _, forbidden := range []string{"token", "dataDir", "environment", "arguments"} {
		if _, exists := body[forbidden]; exists || strings.Contains(response.Body.String(), forbidden) {
			t.Fatalf("health response exposes %q: %s", forbidden, response.Body.String())
		}
	}
	if body["status"] != "ok" || body["service"] != "yorvad" {
		t.Fatalf("unexpected health response: %#v", body)
	}
}

func TestNodeRequiresValidBearerToken(t *testing.T) {
	for _, test := range []struct {
		name   string
		header string
		status int
	}{
		{name: "missing", status: http.StatusUnauthorized},
		{name: "invalid", header: "Bearer invalid", status: http.StatusUnauthorized},
		{name: "valid", header: "Bearer " + testToken, status: http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/v1/node", nil)
			request.Header.Set("Authorization", test.header)
			response := httptest.NewRecorder()

			newTestHandler().ServeHTTP(response, request)

			if response.Code != test.status {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, test.status, response.Body.String())
			}
			if test.status == http.StatusUnauthorized && test.header != "" && strings.Contains(response.Body.String(), test.header) {
				t.Fatalf("response leaks authorization value: %s", response.Body.String())
			}
			if test.status == http.StatusUnauthorized {
				assertProtocolError(t, response, "UNAUTHORIZED")
			}
		})
	}
}

func TestEventsRequiresAuthentication(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)
	response := httptest.NewRecorder()
	newTestHandler().ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestEventStreamCancellationReleasesSubscriber(t *testing.T) {
	broker := events.NewBroker()
	server := httptest.NewServer(NewHandler(testToken, testNode, broker, fakeRuntimeDiscovery{}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/api/v1/events", nil)
	if err != nil {
		t.Fatalf("create events request: %v", err)
	}
	request.Header.Set("Accept", "text/event-stream")
	request.Header.Set("Authorization", "Bearer "+testToken)
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("connect event stream: %v", err)
	}
	if response.StatusCode != http.StatusOK {
		_ = response.Body.Close()
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	line, err := bufio.NewReader(response.Body).ReadString('\n')
	if err != nil || line != ": connected\n" {
		_ = response.Body.Close()
		t.Fatalf("initial stream line = %q, error = %v", line, err)
	}

	cancel()
	_ = response.Body.Close()
	deadline := time.Now().Add(2 * time.Second)
	for broker.SubscriberCount() != 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if broker.SubscriberCount() != 0 {
		t.Fatalf("subscriber count = %d after cancellation, want 0", broker.SubscriberCount())
	}
}

func TestOriginPolicyAllowsOnlyDesktopOrigins(t *testing.T) {
	t.Run("allowed preflight", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodOptions, "/api/v1/node", nil)
		request.Header.Set("Origin", "http://127.0.0.1:1420")
		response := httptest.NewRecorder()
		newTestHandler().ServeHTTP(response, request)
		if response.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
		}
		if response.Header().Get("Access-Control-Allow-Origin") != "http://127.0.0.1:1420" {
			t.Fatalf("unexpected allow origin header: %q", response.Header().Get("Access-Control-Allow-Origin"))
		}
		if response.Header().Get("Allow") != "GET, OPTIONS" {
			t.Fatalf("unexpected Allow header: %q", response.Header().Get("Allow"))
		}
	})

	t.Run("rejected origin", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
		request.Header.Set("Origin", "https://example.com")
		response := httptest.NewRecorder()
		newTestHandler().ServeHTTP(response, request)
		if response.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
		}
		if response.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Fatal("rejected origin received an allow-origin header")
		}
		assertProtocolError(t, response, "ORIGIN_NOT_ALLOWED")
	})

	t.Run("unknown preflight", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodOptions, "/api/v1/missing", nil)
		request.Header.Set("Origin", "http://tauri.localhost")
		response := httptest.NewRecorder()
		newTestHandler().ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
		}
		assertProtocolError(t, response, "NOT_FOUND")
	})
}

func TestRoutingErrorsUseStableProtocolEnvelope(t *testing.T) {
	for _, test := range []struct {
		name   string
		method string
		path   string
		status int
		code   string
	}{
		{name: "not found", method: http.MethodGet, path: "/api/v1/missing", status: http.StatusNotFound, code: "NOT_FOUND"},
		{name: "method not allowed", method: http.MethodPost, path: "/api/v1/node", status: http.StatusMethodNotAllowed, code: "METHOD_NOT_ALLOWED"},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, nil)
			response := httptest.NewRecorder()
			newTestHandler().ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d", response.Code, test.status)
			}
			assertProtocolError(t, response, test.code)
			if test.status == http.StatusMethodNotAllowed && response.Header().Get("Allow") != "GET, OPTIONS" {
				t.Fatalf("unexpected Allow header: %q", response.Header().Get("Allow"))
			}
		})
	}
}

func assertProtocolError(t *testing.T, response *httptest.ResponseRecorder, code string) {
	t.Helper()
	if response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", response.Header().Get("Content-Type"))
	}
	var body ErrorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body.Error.Code != code || body.Error.Message == "" || body.Error.Details == nil {
		t.Fatalf("invalid error response: %#v", body)
	}
}

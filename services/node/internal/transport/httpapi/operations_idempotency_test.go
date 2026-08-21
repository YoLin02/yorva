package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/app"
	"github.com/YoLin02/yorva/services/node/internal/events"
	"github.com/YoLin02/yorva/services/node/internal/persistence/sqlite"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type staticDiscoverer struct {
	discovery yorvaruntime.Discovery
}

func (s staticDiscoverer) Detect(context.Context) (yorvaruntime.Discovery, error) {
	return s.discovery, nil
}

func newLiveInstallHandler(t *testing.T) (http.Handler, *app.RuntimeInstall, *events.Broker) {
	t.Helper()
	db, err := sqlite.Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	registry := yorvaruntime.NewRegistry()
	if err := registry.Register("hermes", yorvaruntime.Bundle{
		Descriptor: yorvaruntime.Descriptor{Kind: "hermes", Name: "Hermes Agent"},
		Discoverer: staticDiscoverer{discovery: yorvaruntime.Discovery{State: yorvaruntime.DiscoveryNotInstalled}},
	}); err != nil {
		t.Fatal(err)
	}
	broker := events.NewBroker()
	discovery := app.NewRuntimeDiscovery(registry, slog.New(slog.NewTextHandler(io.Discard, nil)))
	service := app.NewRuntimeInstall(discovery, db).WithEvents(broker)
	return NewHandler(testToken, testNode, broker, discovery, service, nil, t.TempDir(), nil), service, broker
}

func TestSimultaneousSameKeyHTTPReturnsSameOperation(t *testing.T) {
	handler, _, _ := newLiveInstallHandler(t)
	const workers = 6
	type result struct {
		status int
		id     string
		code   string
	}
	results := make([]result, workers)
	var started sync.WaitGroup
	started.Add(workers)
	begin := make(chan struct{})
	for i := 0; i < workers; i++ {
		go func(i int) {
			defer started.Done()
			<-begin
			request := httptest.NewRequest(http.MethodPost, "/api/v1/runtimes/hermes/install", bytes.NewBufferString(`{}`))
			request.Header.Set("Authorization", "Bearer "+testToken)
			request.Header.Set("Idempotency-Key", "http-same-key")
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			var body map[string]any
			_ = json.Unmarshal(response.Body.Bytes(), &body)
			item := result{status: response.Code}
			if id, ok := body["id"].(string); ok {
				item.id = id
			}
			if errBody, ok := body["error"].(map[string]any); ok {
				if code, ok := errBody["code"].(string); ok {
					item.code = code
				}
			}
			results[i] = item
		}(i)
	}
	close(begin)
	started.Wait()

	ids := map[string]struct{}{}
	for _, item := range results {
		if item.status != http.StatusAccepted || item.id == "" {
			t.Fatalf("simultaneous HTTP start = %#v", results)
		}
		ids[item.id] = struct{}{}
		if item.code == "RUNTIME_INSTALL_IN_PROGRESS" {
			t.Fatalf("same-key request returned host-mutation conflict: %#v", item)
		}
	}
	if len(ids) != 1 {
		t.Fatalf("same-key HTTP returned multiple operations: %#v", results)
	}
}

func TestSameKeyDifferentEndpointHTTPIsRejected(t *testing.T) {
	handler, _, _ := newLiveInstallHandler(t)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/runtimes/hermes/install", bytes.NewBufferString(`{}`))
	request.Header.Set("Authorization", "Bearer "+testToken)
	request.Header.Set("Idempotency-Key", "cross-endpoint-key")
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("install status = %d body=%s", response.Code, response.Body.String())
	}

	prereq := httptest.NewRequest(http.MethodPost, "/api/v1/runtimes/hermes/prerequisites/install", bytes.NewBufferString(`{}`))
	prereq.Header.Set("Authorization", "Bearer "+testToken)
	prereq.Header.Set("Idempotency-Key", "cross-endpoint-key")
	prereq.Header.Set("Content-Type", "application/json")
	prereqResponse := httptest.NewRecorder()
	handler.ServeHTTP(prereqResponse, prereq)
	if prereqResponse.Code != http.StatusConflict || !bytes.Contains(prereqResponse.Body.Bytes(), []byte("IDEMPOTENCY_KEY_CONFLICT")) {
		t.Fatalf("cross-endpoint status = %d body=%s", prereqResponse.Code, prereqResponse.Body.String())
	}
	if bytes.Contains(prereqResponse.Body.Bytes(), []byte("RUNTIME_INSTALL_IN_PROGRESS")) {
		t.Fatalf("cross-endpoint reused host-mutation conflict: %s", prereqResponse.Body.String())
	}
}

func TestCrossKeyHTTPConflictRegression(t *testing.T) {
	handler, _, _ := newLiveInstallHandler(t)
	first := httptest.NewRequest(http.MethodPost, "/api/v1/runtimes/hermes/install", bytes.NewBufferString(`{}`))
	first.Header.Set("Authorization", "Bearer "+testToken)
	first.Header.Set("Idempotency-Key", "http-key-a")
	first.Header.Set("Content-Type", "application/json")
	firstResponse := httptest.NewRecorder()
	handler.ServeHTTP(firstResponse, first)
	if firstResponse.Code != http.StatusAccepted {
		t.Fatalf("first status = %d body=%s", firstResponse.Code, firstResponse.Body.String())
	}
	var created OperationResponse
	if err := json.Unmarshal(firstResponse.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	second := httptest.NewRequest(http.MethodPost, "/api/v1/runtimes/hermes/install", bytes.NewBufferString(`{}`))
	second.Header.Set("Authorization", "Bearer "+testToken)
	second.Header.Set("Idempotency-Key", "http-key-b")
	second.Header.Set("Content-Type", "application/json")
	secondResponse := httptest.NewRecorder()
	handler.ServeHTTP(secondResponse, second)
	if secondResponse.Code != http.StatusConflict || !bytes.Contains(secondResponse.Body.Bytes(), []byte("RUNTIME_INSTALL_IN_PROGRESS")) {
		t.Fatalf("cross-key status = %d body=%s", secondResponse.Code, secondResponse.Body.String())
	}
	if !bytes.Contains(secondResponse.Body.Bytes(), []byte(created.ID)) {
		t.Fatalf("cross-key conflict missing operationId: %s", secondResponse.Body.String())
	}
}

func TestOperationSSEDisconnectThenGETRecovery(t *testing.T) {
	handler, service, broker := newLiveInstallHandler(t)
	server := httptest.NewServer(handler)
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/api/v1/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Accept", "text/event-stream")
	request.Header.Set("Authorization", "Bearer "+testToken)
	stream, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if stream.StatusCode != http.StatusOK {
		_ = stream.Body.Close()
		t.Fatalf("sse status = %d", stream.StatusCode)
	}

	started, err := service.Start(context.Background(), "hermes", "sse-get-recovery")
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	_ = stream.Body.Close()
	deadline := time.Now().Add(2 * time.Second)
	for broker.SubscriberCount() != 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	if _, err := service.Cancel(context.Background(), started.Operation.ID); err != nil {
		t.Fatal(err)
	}
	get := httptest.NewRequest(http.MethodGet, "/api/v1/operations/"+started.Operation.ID, nil)
	get.Header.Set("Authorization", "Bearer "+testToken)
	getResponse := httptest.NewRecorder()
	handler.ServeHTTP(getResponse, get)
	if getResponse.Code != http.StatusOK || !bytes.Contains(getResponse.Body.Bytes(), []byte("CANCELLED")) {
		t.Fatalf("GET after SSE disconnect = %d %s", getResponse.Code, getResponse.Body.String())
	}
}

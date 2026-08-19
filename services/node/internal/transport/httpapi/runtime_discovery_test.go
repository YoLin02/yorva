package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/app"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

func TestRuntimeDiscoveryReturnsTypedNegativeAndPositiveStates(t *testing.T) {
	states := []struct {
		state yorvaruntime.DiscoveryState
		code  yorvaruntime.ErrorCode
	}{
		{yorvaruntime.DiscoveryNotInstalled, yorvaruntime.ErrorRuntimeNotInstalled},
		{yorvaruntime.DiscoverySupported, ""},
		{yorvaruntime.DiscoveryUnsupported, yorvaruntime.ErrorRuntimeUnsupported},
		{yorvaruntime.DiscoveryBrokenExecutable, yorvaruntime.ErrorRuntimeExecutableBroken},
		{yorvaruntime.DiscoveryMalformedVersion, yorvaruntime.ErrorRuntimeVersionMalformed},
		{yorvaruntime.DiscoveryTimedOut, yorvaruntime.ErrorRuntimeDiscoveryTimeout},
		{yorvaruntime.DiscoveryAmbiguous, yorvaruntime.ErrorRuntimeDiscoveryAmbiguous},
	}
	for _, test := range states {
		t.Run(string(test.state), func(t *testing.T) {
			result := yorvaruntime.Discovery{
				RuntimeKind:    "hermes",
				State:          test.state,
				ErrorCode:      test.code,
				Candidates:     []yorvaruntime.Candidate{},
				Warnings:       []yorvaruntime.Warning{},
				DetectedAt:     time.Date(2026, 8, 14, 1, 2, 3, 0, time.UTC),
				SupportedRange: ">=0.19.0 <0.21.0",
			}
			request := httptest.NewRequest(http.MethodPost, "/api/v1/runtimes/hermes/detect", nil)
			request.Header.Set("Authorization", "Bearer "+testToken)
			response := httptest.NewRecorder()

			newTestHandler(fakeRuntimeDiscovery{result: result}).ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
			}
			var body RuntimeDiscoveryResponse
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body.State != test.state || body.RuntimeKind != "hermes" {
				t.Fatalf("unexpected discovery response: %#v", body)
			}
			if test.code == "" && body.ErrorCode != nil {
				t.Fatalf("errorCode = %q, want null", *body.ErrorCode)
			}
			if test.code != "" && (body.ErrorCode == nil || *body.ErrorCode != test.code) {
				t.Fatalf("errorCode = %#v, want %q", body.ErrorCode, test.code)
			}
		})
	}
}

func TestRuntimeDiscoveryRequiresAuthentication(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/runtimes/hermes/detect", nil)
	response := httptest.NewRecorder()
	newTestHandler().ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	assertProtocolError(t, response, "UNAUTHORIZED")
}

func TestRuntimeDiscoveryNormalizesApplicationErrors(t *testing.T) {
	for _, test := range []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "unknown kind", err: app.ErrRuntimeKindNotFound, status: http.StatusNotFound, code: "RUNTIME_KIND_NOT_FOUND"},
		{name: "adapter failure", err: errors.New("secret stderr from process"), status: http.StatusInternalServerError, code: "RUNTIME_DISCOVERY_FAILED"},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v1/runtimes/unknown/detect", nil)
			request.Header.Set("Authorization", "Bearer "+testToken)
			response := httptest.NewRecorder()
			newTestHandler(fakeRuntimeDiscovery{err: test.err}).ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d", response.Code, test.status)
			}
			assertProtocolError(t, response, test.code)
			if strings.Contains(response.Body.String(), "secret stderr") {
				t.Fatalf("response leaked adapter error: %s", response.Body.String())
			}
		})
	}
}

func TestRuntimeDiscoveryRouteContract(t *testing.T) {
	preflight := httptest.NewRequest(http.MethodOptions, "/api/v1/runtimes/hermes/detect", nil)
	preflight.Header.Set("Origin", "tauri://localhost")
	preflightResponse := httptest.NewRecorder()
	newTestHandler().ServeHTTP(preflightResponse, preflight)
	if preflightResponse.Code != http.StatusNoContent || preflightResponse.Header().Get("Allow") != "POST, OPTIONS" {
		t.Fatalf("preflight status=%d Allow=%q", preflightResponse.Code, preflightResponse.Header().Get("Allow"))
	}
	if preflightResponse.Header().Get("Access-Control-Allow-Methods") != "GET, POST, DELETE, OPTIONS" {
		t.Fatalf("unexpected CORS methods: %q", preflightResponse.Header().Get("Access-Control-Allow-Methods"))
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/runtimes/hermes/detect", nil)
	response := httptest.NewRecorder()
	newTestHandler().ServeHTTP(response, request)
	if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != "POST, OPTIONS" {
		t.Fatalf("status=%d Allow=%q", response.Code, response.Header().Get("Allow"))
	}
}

type blockingRuntimeDiscovery struct {
	entered  chan struct{}
	finished chan error
}

func (d blockingRuntimeDiscovery) Detect(ctx context.Context, _ yorvaruntime.Kind) (yorvaruntime.Discovery, error) {
	close(d.entered)
	<-ctx.Done()
	d.finished <- ctx.Err()
	return yorvaruntime.Discovery{}, ctx.Err()
}

func TestRuntimeDiscoveryPropagatesRequestCancellation(t *testing.T) {
	entered := make(chan struct{})
	finished := make(chan error, 1)
	server := httptest.NewServer(newTestHandler(blockingRuntimeDiscovery{entered: entered, finished: finished}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL+"/api/v1/runtimes/hermes/detect", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	request.Header.Set("Authorization", "Bearer "+testToken)
	result := make(chan error, 1)
	go func() {
		response, requestErr := server.Client().Do(request)
		if response != nil {
			_ = response.Body.Close()
		}
		result <- requestErr
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("request did not reach Runtime discovery")
	}
	cancel()

	select {
	case got := <-finished:
		if !errors.Is(got, context.Canceled) {
			t.Fatalf("adapter context error = %v, want context.Canceled", got)
		}
	case <-time.After(time.Second):
		t.Fatal("adapter did not observe request cancellation")
	}
	select {
	case <-result:
	case <-time.After(time.Second):
		t.Fatal("HTTP request did not finish after cancellation")
	}
}

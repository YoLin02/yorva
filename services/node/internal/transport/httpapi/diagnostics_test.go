package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/diagnostics"
	"github.com/YoLin02/yorva/services/node/internal/events"
)

type fakeDiagnosticBundleService struct {
	bundle diagnostics.Bundle
	err    error
}

func (f fakeDiagnosticBundleService) Build(context.Context) (diagnostics.Bundle, error) {
	return f.bundle, f.err
}

func TestExportDiagnosticBundleReturnsOnlyCompleteZip(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/diagnostics/bundle", nil)
	exportDiagnosticBundle(fakeDiagnosticBundleService{bundle: diagnostics.Bundle{
		Bytes: []byte("PK-fixture"), FileName: "YORVA-diagnostics-fixture.zip", CreatedAt: time.Now(),
	}}).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	if recorder.Header().Get("Content-Type") != "application/zip" {
		t.Fatalf("content type = %q", recorder.Header().Get("Content-Type"))
	}
	if recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("cache control = %q", recorder.Header().Get("Cache-Control"))
	}
	if recorder.Body.String() != "PK-fixture" {
		t.Fatalf("body = %q", recorder.Body.String())
	}
}

func TestExportDiagnosticBundleUsesStableFailure(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/diagnostics/bundle", nil)
	exportDiagnosticBundle(fakeDiagnosticBundleService{err: errors.New("secret internal detail")}).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", recorder.Code)
	}
	if body := recorder.Body.String(); body == "" || contains(body, "secret internal detail") {
		t.Fatalf("unsafe error body = %q", body)
	}
}

func TestDiagnosticBundleRouteRequiresBearerAndPost(t *testing.T) {
	service := fakeDiagnosticBundleService{bundle: diagnostics.Bundle{Bytes: []byte("PK-fixture"), FileName: "diagnostics.zip"}}
	handler := NewHandler(testToken, testNode, events.NewBroker(), fakeRuntimeDiscovery{}, nil, nil, "", nil, service)
	for _, test := range []struct {
		method, token string
		status        int
	}{
		{method: http.MethodPost, status: http.StatusUnauthorized},
		{method: http.MethodGet, token: testToken, status: http.StatusMethodNotAllowed},
		{method: http.MethodPost, token: testToken, status: http.StatusOK},
	} {
		request := httptest.NewRequest(test.method, "/api/v1/diagnostics/bundle", nil)
		if test.token != "" {
			request.Header.Set("Authorization", "Bearer "+test.token)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != test.status {
			t.Fatalf("%s status = %d, want %d", test.method, response.Code, test.status)
		}
	}
}

func contains(value, needle string) bool {
	for i := 0; i+len(needle) <= len(value); i++ {
		if value[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

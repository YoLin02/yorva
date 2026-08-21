package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/YoLin02/yorva/services/node/internal/events"
	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes/downloadsources"
)

type fakeDownloadSourceSettings struct {
	config downloadsources.Config
	saves  int
	resets int
}

func (s *fakeDownloadSourceSettings) Get(context.Context) (downloadsources.Config, error) {
	return s.config, nil
}

func (s *fakeDownloadSourceSettings) Save(_ context.Context, config downloadsources.Config) (downloadsources.Config, error) {
	s.saves++
	s.config = config
	return config, nil
}

func (s *fakeDownloadSourceSettings) Reset(context.Context) (downloadsources.Config, error) {
	s.resets++
	s.config = downloadsources.Default()
	return s.config, nil
}

func TestHermesDownloadSourceSettingsRequireAuthentication(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/settings/hermes/download-sources", nil)
	response := httptest.NewRecorder()
	newSettingsTestHandler(&fakeDownloadSourceSettings{config: downloadsources.Default()}).ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestHermesDownloadSourceSettingsGetSaveAndReset(t *testing.T) {
	service := &fakeDownloadSourceSettings{config: downloadsources.Default()}
	handler := newSettingsTestHandler(service)

	get := authenticatedSettingsRequest(http.MethodGet, nil)
	getResponse := httptest.NewRecorder()
	handler.ServeHTTP(getResponse, get)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("GET status = %d body=%s", getResponse.Code, getResponse.Body.String())
	}

	config := downloadsources.Default()
	config.PythonIndexURL = "https://mirror.example/pypi/simple"
	payload, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	put := authenticatedSettingsRequest(http.MethodPut, payload)
	put.Header.Set("Content-Type", "application/json")
	putResponse := httptest.NewRecorder()
	handler.ServeHTTP(putResponse, put)
	if putResponse.Code != http.StatusOK || service.saves != 1 || service.config.PythonIndexURL != config.PythonIndexURL {
		t.Fatalf("PUT status=%d saves=%d config=%#v body=%s", putResponse.Code, service.saves, service.config, putResponse.Body.String())
	}

	reset := authenticatedSettingsRequest(http.MethodDelete, nil)
	resetResponse := httptest.NewRecorder()
	handler.ServeHTTP(resetResponse, reset)
	if resetResponse.Code != http.StatusOK || service.resets != 1 || service.config != downloadsources.Default() {
		t.Fatalf("DELETE status=%d resets=%d config=%#v", resetResponse.Code, service.resets, service.config)
	}
}

func TestHermesDownloadSourceSettingsRejectUnsafeAndUnknownFields(t *testing.T) {
	service := &fakeDownloadSourceSettings{config: downloadsources.Default()}
	handler := newSettingsTestHandler(service)
	for name, payload := range map[string][]byte{
		"http": []byte(`{"hermesArchiveUrl":"http://example.com/hermes.zip","nodeArchiveUrl":"https://example.com/node.zip","npmArchiveUrl":"https://example.com/npm.tgz","pythonIndexUrl":"https://example.com/simple","npmRegistryUrl":"https://example.com/npm"}`),
		"unknown": []byte(`{"hermesArchiveUrl":"https://example.com/hermes.zip","nodeArchiveUrl":"https://example.com/node.zip","npmArchiveUrl":"https://example.com/npm.tgz","pythonIndexUrl":"https://example.com/simple","npmRegistryUrl":"https://example.com/npm","token":"secret"}`),
	} {
		t.Run(name, func(t *testing.T) {
			request := authenticatedSettingsRequest(http.MethodPut, payload)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest || service.saves != 0 {
				t.Fatalf("status=%d saves=%d body=%s", response.Code, service.saves, response.Body.String())
			}
			assertProtocolError(t, response, "SETTINGS_INVALID")
		})
	}
}

func newSettingsTestHandler(settings HermesDownloadSourceSettingsService) http.Handler {
	return NewHandler(testToken, testNode, events.NewBroker(), fakeRuntimeDiscovery{}, nil, nil, "", settings)
}

func authenticatedSettingsRequest(method string, payload []byte) *http.Request {
	request := httptest.NewRequest(method, "/api/v1/settings/hermes/download-sources", bytes.NewReader(payload))
	request.Header.Set("Authorization", "Bearer "+testToken)
	return request
}

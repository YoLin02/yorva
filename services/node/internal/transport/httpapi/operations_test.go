package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/app"
	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type fakeInstallService struct {
	started app.InstallStartResult
	err     error
}

func (f fakeInstallService) Start(_ context.Context, _ yorvaruntime.Kind, key string) (app.InstallStartResult, error) {
	if f.err != nil {
		return app.InstallStartResult{}, f.err
	}
	result := f.started
	result.Operation.IdempotencyKey = key
	return result, nil
}

func (f fakeInstallService) Get(context.Context, string) (operation.Operation, error) {
	return f.started.Operation, nil
}

func (f fakeInstallService) List(context.Context, string, string, int) ([]operation.Operation, error) {
	return []operation.Operation{f.started.Operation}, nil
}

func (f fakeInstallService) Cancel(context.Context, string) (operation.Operation, error) {
	cancelled := f.started.Operation
	cancelled.Status = operation.StatusCancelled
	return cancelled, nil
}

func TestInstallRequiresIdempotencyKeyAndRejectsCommandFields(t *testing.T) {
	handler := NewHandler(testToken, testNode, nil, fakeRuntimeDiscovery{}, fakeInstallService{started: testInstallOperation()})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/runtimes/hermes/install", bytes.NewBufferString(`{"url":"https://evil"}`))
	request.Header.Set("Authorization", "Bearer "+testToken)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("missing key status = %d body=%s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/runtimes/hermes/install", bytes.NewBufferString(`{"url":"https://evil"}`))
	request.Header.Set("Authorization", "Bearer "+testToken)
	request.Header.Set("Idempotency-Key", "abc")
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest || !bytes.Contains(response.Body.Bytes(), []byte("INVALID_REQUEST")) {
		t.Fatalf("command field status = %d body=%s", response.Code, response.Body.String())
	}
}

func TestInstallStartGetAndCancel(t *testing.T) {
	handler := NewHandler(testToken, testNode, nil, fakeRuntimeDiscovery{}, fakeInstallService{started: testInstallOperation()})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/runtimes/hermes/install", bytes.NewBufferString(`{}`))
	request.Header.Set("Authorization", "Bearer "+testToken)
	request.Header.Set("Idempotency-Key", "install-key")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("start status = %d body=%s", response.Code, response.Body.String())
	}
	var created OperationResponse
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil || created.ID == "" || created.Progress != nil {
		t.Fatalf("start body = %s err=%v", response.Body.String(), err)
	}

	get := httptest.NewRequest(http.MethodGet, "/api/v1/operations/"+created.ID, nil)
	get.Header.Set("Authorization", "Bearer "+testToken)
	getResponse := httptest.NewRecorder()
	handler.ServeHTTP(getResponse, get)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("get status = %d", getResponse.Code)
	}

	cancel := httptest.NewRequest(http.MethodPost, "/api/v1/operations/"+created.ID+"/cancel", nil)
	cancel.Header.Set("Authorization", "Bearer "+testToken)
	cancelResponse := httptest.NewRecorder()
	handler.ServeHTTP(cancelResponse, cancel)
	if cancelResponse.Code != http.StatusOK || !bytes.Contains(cancelResponse.Body.Bytes(), []byte("CANCELLED")) {
		t.Fatalf("cancel status = %d body=%s", cancelResponse.Code, cancelResponse.Body.String())
	}
}

func testInstallOperation() app.InstallStartResult {
	now := time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC)
	return app.InstallStartResult{
		Created: true,
		Operation: operation.Operation{
			ID:            "op_test",
			Type:          operation.TypeRuntimeInstall,
			TargetType:    operation.TargetRuntimeKind,
			TargetID:      "hermes",
			Status:        operation.StatusPending,
			Stage:         operation.StagePreflight,
			CorrelationID: "cor_test",
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}
}

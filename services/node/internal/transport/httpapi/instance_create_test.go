package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/app"
	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
)

type createInventory struct {
	fakeInstanceInventory
	started app.InstallStartResult
	err     error
}

func (c createInventory) StartCreate(_ context.Context, _, name, _ string) (app.InstallStartResult, error) {
	if name != "coder" && c.err == nil {
		return app.InstallStartResult{}, app.ErrInstanceInvalidName
	}
	if c.err != nil {
		return app.InstallStartResult{}, c.err
	}
	return c.started, nil
}

func TestCreateInstanceClosedBodyAndIdempotency(t *testing.T) {
	now := time.Date(2026, 8, 19, 17, 0, 0, 0, time.UTC)
	inventory := createInventory{started: app.InstallStartResult{
		Created: true,
		Operation: operation.Operation{
			ID: "op_create", Type: operation.TypeInstanceCreate, Status: operation.StatusPending,
			TargetType: operation.TargetRuntimeInstallation, TargetID: "rtinst_1",
			Message: "coder", CreatedAt: now, UpdatedAt: now,
		},
	}}
	handler := NewHandler(testToken, testNode, nil, fakeRuntimeDiscovery{}, nil, inventory, "", nil)

	valid := httptest.NewRequest(http.MethodPost, "/api/v1/runtimes/hermes/instances", bytes.NewReader([]byte(`{"name":"coder"}`)))
	valid.Header.Set("Authorization", "Bearer "+testToken)
	valid.Header.Set("Content-Type", "application/json")
	valid.Header.Set("Idempotency-Key", "create-1")
	validRes := httptest.NewRecorder()
	handler.ServeHTTP(validRes, valid)
	if validRes.Code != http.StatusAccepted {
		t.Fatalf("valid create = %d %s", validRes.Code, validRes.Body.String())
	}

	cases := []struct {
		name string
		body string
		key  string
		code string
	}{
		{name: "unknown field", body: `{"name":"coder","path":"C:\\x"}`, key: "k", code: "INVALID_REQUEST"},
		{name: "clone field", body: `{"name":"coder","clone":true}`, key: "k", code: "INVALID_REQUEST"},
		{name: "missing name", body: `{}`, key: "k", code: "INVALID_REQUEST"},
		{name: "trailing", body: `{"name":"coder"}{}`, key: "k", code: "INVALID_REQUEST"},
		{name: "bad key", body: `{"name":"coder"}`, key: "has space", code: "INVALID_IDEMPOTENCY_KEY"},
	}
	for _, test := range cases {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/runtimes/hermes/instances", strings.NewReader(test.body))
		req.Header.Set("Authorization", "Bearer "+testToken)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", test.key)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != http.StatusBadRequest {
			t.Fatalf("%s status = %d %s", test.name, res.Code, res.Body.String())
		}
		var envelope map[string]any
		if err := json.Unmarshal(res.Body.Bytes(), &envelope); err != nil {
			t.Fatal(err)
		}
		errBody, _ := envelope["error"].(map[string]any)
		if errBody["code"] != test.code {
			t.Fatalf("%s error = %#v", test.name, envelope)
		}
	}
}

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/app"
	"github.com/YoLin02/yorva/services/node/internal/domain/instance"
	"github.com/YoLin02/yorva/services/node/internal/domain/operation"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type fakeInstanceInventory struct {
	list      app.InstanceList
	listErr   error
	get       app.InstanceView
	getErr    error
	deleteErr error
}

type fakeLifecycleInventory struct {
	fakeInstanceInventory
	view      app.LifecycleView
	operation operation.Operation
	action    app.LifecycleAction
}

func (f *fakeLifecycleInventory) GetLifecycle(context.Context, string) (app.LifecycleView, error) {
	return f.view, nil
}

func (f *fakeLifecycleInventory) StartLifecycle(_ context.Context, _ string, action app.LifecycleAction, _ string) (app.InstallStartResult, error) {
	f.action = action
	return app.InstallStartResult{Operation: f.operation, Created: true}, nil
}

func (f *fakeLifecycleInventory) CancelLifecycle(context.Context, string) (operation.Operation, error) {
	return f.operation, nil
}

func (f fakeInstanceInventory) ListInstances(_ context.Context, runtimeID string) (app.InstanceList, error) {
	if runtimeID != "hermes" && f.listErr == nil {
		return app.InstanceList{}, app.ErrInstanceRuntimeNotFound
	}
	return f.list, f.listErr
}

func (f fakeInstanceInventory) GetInstance(_ context.Context, _ string) (app.InstanceView, error) {
	return f.get, f.getErr
}

func (f fakeInstanceInventory) StartCreate(context.Context, string, string, string) (app.InstallStartResult, error) {
	return app.InstallStartResult{}, app.ErrRuntimeNotSupported
}

func (f fakeInstanceInventory) CancelCreate(context.Context, string) (operation.Operation, error) {
	return operation.Operation{}, app.ErrInstanceNotFound
}

func (f fakeInstanceInventory) StartDelete(context.Context, string, string, string) (app.InstallStartResult, error) {
	if f.deleteErr != nil {
		return app.InstallStartResult{}, f.deleteErr
	}
	return app.InstallStartResult{}, app.ErrRuntimeNotSupported
}

func (f fakeInstanceInventory) CancelDelete(context.Context, string) (operation.Operation, error) {
	return operation.Operation{}, app.ErrInstanceNotFound
}

func TestListAndGetInstancesContract(t *testing.T) {
	now := time.Date(2026, 8, 19, 16, 0, 0, 0, time.UTC)
	inventory := fakeInstanceInventory{
		list: app.InstanceList{
			RuntimeID:             "hermes",
			RuntimeInstallationID: "rtinst_1",
			Freshness:             "FRESH",
			LastSyncedAt:          &now,
			Capabilities:          app.InstanceCapabilities{Instances: true, Lifecycle: false},
			Instances: []app.InstanceView{{
				InstanceID:            "inst_1",
				RuntimeInstallationID: "rtinst_1",
				Name:                  "default",
				Default:               true,
				Protected:             true,
				Availability:          instance.Available,
				LastSyncedAt:          &now,
				CreatedAt:             now,
				UpdatedAt:             now,
				Capabilities:          app.InstanceCapabilities{Instances: true, Lifecycle: false},
			}},
		},
		get: app.InstanceView{
			InstanceID: "inst_1", RuntimeInstallationID: "rtinst_1", Name: "default",
			Default: true, Protected: true, Availability: instance.Available,
			CreatedAt: now, UpdatedAt: now,
		},
	}
	handler := NewHandler(testToken, testNode, nil, fakeRuntimeDiscovery{}, nil, inventory, "")

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/runtimes/hermes/instances", nil)
	listReq.Header.Set("Authorization", "Bearer "+testToken)
	listRes := httptest.NewRecorder()
	handler.ServeHTTP(listRes, listReq)
	if listRes.Code != http.StatusOK {
		t.Fatalf("list status = %d %s", listRes.Code, listRes.Body.String())
	}
	var listed InstanceListResponse
	if err := json.Unmarshal(listRes.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if listed.RuntimeID != "hermes" || listed.Capabilities.Lifecycle || listed.Instances[0].InstanceID != "inst_1" {
		t.Fatalf("list body = %#v", listed)
	}
	if listed.Instances[0].Name != "default" || strings.Contains(listRes.Body.String(), `"nativeId"`) {
		t.Fatalf("native leak or unexpected body: %s", listRes.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/instances/inst_1", nil)
	getReq.Header.Set("Authorization", "Bearer "+testToken)
	getRes := httptest.NewRecorder()
	handler.ServeHTTP(getRes, getReq)
	if getRes.Code != http.StatusOK {
		t.Fatalf("get status = %d %s", getRes.Code, getRes.Body.String())
	}

	lifeReq := httptest.NewRequest(http.MethodPost, "/api/v1/instances/inst_1/start", nil)
	lifeReq.Header.Set("Authorization", "Bearer "+testToken)
	lifeRes := httptest.NewRecorder()
	handler.ServeHTTP(lifeRes, lifeReq)
	if lifeRes.Code != http.StatusConflict || !strings.Contains(lifeRes.Body.String(), string(yorvaruntime.ErrorCapabilityNotSupported)) {
		t.Fatalf("lifecycle = %d %s", lifeRes.Code, lifeRes.Body.String())
	}
}

func TestLifecycleStatusAndMutationContracts(t *testing.T) {
	now := time.Date(2026, 8, 20, 20, 0, 0, 0, time.UTC)
	inventory := &fakeLifecycleInventory{
		view:      app.LifecycleView{State: yorvaruntime.LifecycleStopped, ObservedAt: now},
		operation: operation.Operation{ID: "op_life", Type: operation.TypeInstanceStart, TargetType: operation.TargetInstance, TargetID: "inst_1", Status: operation.StatusPending, Stage: operation.StagePreflight, IdempotencyKey: "life-key", CreatedAt: now, UpdatedAt: now},
	}
	handler := NewHandler(testToken, testNode, nil, fakeRuntimeDiscovery{}, nil, inventory, "")
	statusReq := authorizedRequest(http.MethodGet, "/api/v1/instances/inst_1/lifecycle", "")
	statusRes := httptest.NewRecorder()
	handler.ServeHTTP(statusRes, statusReq)
	if statusRes.Code != http.StatusOK || !strings.Contains(statusRes.Body.String(), `"state":"STOPPED"`) || strings.Contains(statusRes.Body.String(), "PID") {
		t.Fatalf("status = %d %s", statusRes.Code, statusRes.Body.String())
	}

	startReq := authorizedRequest(http.MethodPost, "/api/v1/instances/inst_1/start", "")
	startReq.Header.Set("Idempotency-Key", "life-key")
	startRes := httptest.NewRecorder()
	handler.ServeHTTP(startRes, startReq)
	if startRes.Code != http.StatusAccepted || inventory.action != app.LifecycleStart || !strings.Contains(startRes.Body.String(), `"type":"instance.start"`) {
		t.Fatalf("start = %d %s action=%s", startRes.Code, startRes.Body.String(), inventory.action)
	}

	missingKey := authorizedRequest(http.MethodPost, "/api/v1/instances/inst_1/stop", "")
	missingRes := httptest.NewRecorder()
	handler.ServeHTTP(missingRes, missingKey)
	if missingRes.Code != http.StatusBadRequest || !strings.Contains(missingRes.Body.String(), "INVALID_IDEMPOTENCY_KEY") {
		t.Fatalf("missing key = %d %s", missingRes.Code, missingRes.Body.String())
	}
}

func TestListInstancesRequiresAuthAndRejectsUnknownRuntime(t *testing.T) {
	handler := NewHandler(testToken, testNode, nil, fakeRuntimeDiscovery{}, nil, fakeInstanceInventory{}, "")
	unauth := httptest.NewRequest(http.MethodGet, "/api/v1/runtimes/hermes/instances", nil)
	unauthRes := httptest.NewRecorder()
	handler.ServeHTTP(unauthRes, unauth)
	if unauthRes.Code != http.StatusUnauthorized {
		t.Fatalf("unauth = %d", unauthRes.Code)
	}

	missing := httptest.NewRequest(http.MethodGet, "/api/v1/runtimes/other/instances", nil)
	missing.Header.Set("Authorization", "Bearer "+testToken)
	missingRes := httptest.NewRecorder()
	handler.ServeHTTP(missingRes, missing)
	if missingRes.Code != http.StatusNotFound {
		t.Fatalf("unknown runtime = %d %s", missingRes.Code, missingRes.Body.String())
	}
}

func TestDeleteInstanceQueryFailureIsStableError(t *testing.T) {
	handler := NewHandler(testToken, testNode, nil, fakeRuntimeDiscovery{}, nil, fakeInstanceInventory{deleteErr: app.ErrInstanceQueryFailed}, "")
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/instances/inst_1", strings.NewReader(`{"confirmationName":"coder"}`))
	req.Header.Set("Authorization", "Bearer "+testToken)
	req.Header.Set("Idempotency-Key", "del-query-fail")
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusServiceUnavailable || !strings.Contains(res.Body.String(), string(yorvaruntime.ErrorInstanceQueryFailed)) {
		t.Fatalf("query failure delete = %d %s", res.Code, res.Body.String())
	}
}

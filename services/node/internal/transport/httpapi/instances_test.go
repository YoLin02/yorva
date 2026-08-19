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
	list    app.InstanceList
	listErr error
	get     app.InstanceView
	getErr  error
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

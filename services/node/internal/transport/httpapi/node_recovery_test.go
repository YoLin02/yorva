package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/YoLin02/yorva/services/node/internal/app"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type recoveryInventory struct {
	fakeInstanceInventory
	result app.NodeRecovery
	err    error
	calls  int
}

func (f *recoveryInventory) CheckNodeRecovery(ctx context.Context) (app.NodeRecovery, error) {
	if _, ok := ctx.Deadline(); !ok {
		return app.NodeRecovery{}, errors.New("missing request deadline")
	}
	f.calls++
	return f.result, f.err
}

func TestNodeRecoveryContractIsAuthenticatedLiveAndSanitized(t *testing.T) {
	inventory := &recoveryInventory{result: app.NodeRecovery{Ready: true}}
	handler := NewHandler(testToken, testNode, nil, nil, nil, inventory, "", nil)
	request := func(token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/node/recovery", nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		return res
	}
	if res := request(""); res.Code != http.StatusUnauthorized || inventory.calls != 0 {
		t.Fatalf("unauthenticated recovery = %d / %d calls", res.Code, inventory.calls)
	}
	for _, ready := range []bool{true, false, true} {
		inventory.result = app.NodeRecovery{Ready: ready}
		if !ready {
			inventory.result.ErrorCode = yorvaruntime.ErrorInstanceOutputUnrecognized
		}
		res := request(testToken)
		var result NodeRecoveryResponse
		if err := json.Unmarshal(res.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if res.Code != http.StatusOK || (result.State == "READY") != ready || result.NodeVersion != testNode.NodeVersion || res.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("recovery = %d %s", res.Code, res.Body.String())
		}
		if !ready && (result.ErrorCode == nil || *result.ErrorCode != yorvaruntime.ErrorInstanceOutputUnrecognized) {
			t.Fatalf("missing typed recovery failure: %#v", result)
		}
	}
	if inventory.calls != 3 {
		t.Fatalf("cached recovery: %d calls", inventory.calls)
	}
	inventory.err = errors.New("private-database-path-and-secret")
	res := request(testToken)
	if res.Code != http.StatusServiceUnavailable || !strings.Contains(res.Body.String(), "NODE_RECOVERY_FAILED") || strings.Contains(res.Body.String(), "private-database") {
		t.Fatalf("unsafe failure response = %d %s", res.Code, res.Body.String())
	}
}

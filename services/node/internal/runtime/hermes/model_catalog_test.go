package hermes

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestFetchProviderModelsRejectsRedirectWithoutForwardingCredential(t *testing.T) {
	calls := 0
	manager := &ModelManager{catalogClient: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		if calls > 1 {
			t.Fatalf("credential request followed redirect to %s", request.URL)
		}
		return &http.Response{
			StatusCode: http.StatusFound,
			Header:     http.Header{"Location": []string{"https://unqualified.example/models"}},
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	})}}

	_, err := manager.FetchProviderModels(context.Background(), "anthropic", []byte("secret-value"))
	if !errors.Is(err, yorvaruntime.ErrModelCatalogFetchFailed) {
		t.Fatalf("error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("request calls = %d, want 1", calls)
	}
}

func TestFetchProviderModelsUsesQualifiedEndpointAndReturnsOnlySafeUniqueIDs(t *testing.T) {
	manager := &ModelManager{catalogClient: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() != "https://api.deepseek.com/models" {
			t.Fatalf("catalog URL = %s", request.URL)
		}
		if request.Header.Get("Authorization") != "Bearer secret-value" {
			t.Fatal("missing bearer credential")
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"data":[{"id":"deepseek-v4-pro"},{"id":"deepseek-v4-pro"},{"id":"bad id"},{"id":"deepseek-v4-flash"}]}`))}, nil
	})}}

	models, err := manager.FetchProviderModels(context.Background(), "deepseek", []byte("secret-value"))
	if err != nil || len(models) != 2 || models[0] != "deepseek-v4-pro" || models[1] != "deepseek-v4-flash" {
		t.Fatalf("models = %#v, err=%v", models, err)
	}
}

func TestParseDashScopeProviderCatalog(t *testing.T) {
	models, err := parseProviderModelCatalog([]byte(`{"output":{"models":[{"model":"qwen3.7-max"},{"model":"qwen3.7-plus"}]}}`), "dashscope")
	if err != nil || len(models) != 2 || models[1] != "qwen3.7-plus" {
		t.Fatalf("models = %#v, err=%v", models, err)
	}
}

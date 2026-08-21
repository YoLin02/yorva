package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClosedEmptyObjectContract(t *testing.T) {
	handler := NewHandler(testToken, testNode, nil, fakeRuntimeDiscovery{}, fakeInstallService{started: testInstallOperation()}, nil, "", nil)
	endpoints := []string{
		"/api/v1/runtimes/hermes/install",
		"/api/v1/runtimes/hermes/prerequisites/install",
	}
	cases := []struct {
		name string
		body string
		key  string
		want int
		code string
	}{
		{name: "valid empty object", body: "{}", key: "ok-key", want: http.StatusAccepted},
		{name: "unknown field", body: `{"url":"https://evil"}`, key: "ok-key", want: http.StatusBadRequest, code: "INVALID_REQUEST"},
		{name: "non-object array", body: `[]`, key: "ok-key", want: http.StatusBadRequest, code: "INVALID_REQUEST"},
		{name: "non-object string", body: `"x"`, key: "ok-key", want: http.StatusBadRequest, code: "INVALID_REQUEST"},
		{name: "null", body: `null`, key: "ok-key", want: http.StatusBadRequest, code: "INVALID_REQUEST"},
		{name: "double json", body: `{}{}`, key: "ok-key", want: http.StatusBadRequest, code: "INVALID_REQUEST"},
		{name: "trailing json", body: "{} {\"x\":1}", key: "ok-key", want: http.StatusBadRequest, code: "INVALID_REQUEST"},
		{name: "trailing garbage", body: "{}nope", key: "ok-key", want: http.StatusBadRequest, code: "INVALID_REQUEST"},
		{name: "oversized body", body: `{` + strings.Repeat(" ", maxInstallRequestBytes+8) + `}`, key: "ok-key", want: http.StatusBadRequest, code: "INVALID_REQUEST"},
		{name: "missing body", body: "", key: "ok-key", want: http.StatusBadRequest, code: "INVALID_REQUEST"},
		{name: "invalid idempotency key", body: "{}", key: "has space", want: http.StatusBadRequest, code: "INVALID_IDEMPOTENCY_KEY"},
	}
	for _, endpoint := range endpoints {
		for _, test := range cases {
			t.Run(endpoint+"/"+test.name, func(t *testing.T) {
				var reader *bytes.Reader
				if test.body == "" {
					reader = bytes.NewReader(nil)
				} else {
					reader = bytes.NewReader([]byte(test.body))
				}
				request := httptest.NewRequest(http.MethodPost, endpoint, reader)
				request.Header.Set("Authorization", "Bearer "+testToken)
				request.Header.Set("Content-Type", "application/json")
				if test.key != "" {
					request.Header.Set("Idempotency-Key", test.key)
				}
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				if response.Code != test.want {
					t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
				}
				if test.code != "" {
					var envelope map[string]any
					if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
						t.Fatal(err)
					}
					errBody, _ := envelope["error"].(map[string]any)
					if errBody["code"] != test.code {
						t.Fatalf("error = %#v", envelope)
					}
					if strings.Contains(response.Body.String(), "json:") || strings.Contains(response.Body.String(), "unmarshal") {
						t.Fatalf("raw decoder error leaked: %s", response.Body.String())
					}
				}
			})
		}
	}
}

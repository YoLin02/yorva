package mcptestserver

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestServerImplementsBoundedQualificationHandshake(t *testing.T) {
	server := Start(t)
	client := server.Client()
	session := qualificationPost(t, client, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`, "")
	if session != testSessionID {
		t.Fatalf("session = %q", session)
	}
	qualificationPost(t, client, `{"jsonrpc":"2.0","method":"notifications/initialized"}`, session)
	qualificationPost(t, client, `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`, session)
	if server.RequestCount() != 3 {
		t.Fatalf("requests = %d, want 3", server.RequestCount())
	}
}

func TestServerRejectsWrongCredentialAndOversizedBody(t *testing.T) {
	server := Start(t)
	for name, test := range map[string]struct {
		body       string
		credential string
	}{
		"wrong credential": {body: `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`, credential: "wrong"},
		"oversized":        {body: strings.Repeat("x", requestLimit+1), credential: Credential},
	} {
		t.Run(name, func(t *testing.T) {
			request, err := http.NewRequest(http.MethodPost, FixedEndpoint, strings.NewReader(test.body))
			if err != nil {
				t.Fatal(err)
			}
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", "Bearer "+test.credential)
			response, err := server.Client().Do(request)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			if response.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", response.StatusCode)
			}
		})
	}
}

func qualificationPost(t *testing.T, client *http.Client, body, session string) string {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, FixedEndpoint, bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+Credential)
	if session != "" {
		request.Header.Set("Mcp-Session-Id", session)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNoContent {
		payload, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
		t.Fatalf("status = %d: %s", response.StatusCode, payload)
	}
	return response.Header.Get("Mcp-Session-Id")
}

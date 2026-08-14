package daemon

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/bootstrap"
)

func TestRunBootstrapsLoopbackHealthServer(t *testing.T) {
	token := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x2a}, 32))
	input := fmt.Sprintf(
		`{"protocolVersion":"1","token":"%s","dataDir":%q}`,
		token,
		t.TempDir(),
	)
	stdoutReader, stdoutWriter := io.Pipe()
	stdinReader, stdinWriter := io.Pipe()
	defer stdinWriter.Close()
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)

	go func() {
		result <- Run(ctx, []string{"--bootstrap-stdio"}, Streams{
			Stdin:  stdinReader,
			Stdout: stdoutWriter,
			Stderr: io.Discard,
		})
	}()
	if _, err := fmt.Fprintln(stdinWriter, input); err != nil {
		t.Fatalf("write bootstrap: %v", err)
	}

	var handshake bootstrap.Handshake
	if err := json.NewDecoder(stdoutReader).Decode(&handshake); err != nil {
		t.Fatalf("decode handshake: %v", err)
	}
	if handshake.Port == 0 || handshake.ProtocolVersion != "1" {
		t.Fatalf("invalid handshake: %#v", handshake)
	}

	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/api/v1/health", handshake.Port))
	if err != nil {
		t.Fatalf("GET health: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d, want %d", response.StatusCode, http.StatusOK)
	}

	nodeRequest, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/api/v1/node", handshake.Port), nil)
	if err != nil {
		t.Fatalf("create node request: %v", err)
	}
	nodeRequest.Header.Set("Authorization", "Bearer "+token)
	nodeResponse, err := client.Do(nodeRequest)
	if err != nil {
		t.Fatalf("GET node: %v", err)
	}
	_ = nodeResponse.Body.Close()
	if nodeResponse.StatusCode != http.StatusOK {
		t.Fatalf("node status = %d, want %d", nodeResponse.StatusCode, http.StatusOK)
	}

	unauthorizedRequest, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/api/v1/node", handshake.Port), nil)
	if err != nil {
		t.Fatalf("create unauthorized request: %v", err)
	}
	unauthorizedResponse, err := client.Do(unauthorizedRequest)
	if err != nil {
		t.Fatalf("GET unauthorized node: %v", err)
	}
	_ = unauthorizedResponse.Body.Close()
	if unauthorizedResponse.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized node status = %d, want %d", unauthorizedResponse.StatusCode, http.StatusUnauthorized)
	}

	streamCtx, stopStream := context.WithCancel(context.Background())
	streamRequest, err := http.NewRequestWithContext(streamCtx, http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/api/v1/events", handshake.Port), nil)
	if err != nil {
		t.Fatalf("create event stream request: %v", err)
	}
	streamRequest.Header.Set("Accept", "text/event-stream")
	streamRequest.Header.Set("Authorization", "Bearer "+token)
	streamResponse, err := client.Do(streamRequest)
	if err != nil {
		t.Fatalf("connect event stream: %v", err)
	}
	line, err := bufio.NewReader(streamResponse.Body).ReadString('\n')
	if err != nil || line != ": connected\n" {
		t.Fatalf("event stream first line = %q, error = %v", line, err)
	}
	stopStream()
	_ = streamResponse.Body.Close()

	cancel()
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("daemon did not shut down after cancellation")
	}
}

func TestRunStopsWhenParentClosesBootstrapPipe(t *testing.T) {
	testParentStop(t, func(writer *io.PipeWriter) error { return writer.Close() })
}

func TestRunStopsOnParentShutdownControl(t *testing.T) {
	testParentStop(t, func(writer *io.PipeWriter) error {
		_, err := io.WriteString(writer, "{\"type\":\"shutdown\"}\n")
		return err
	})
}

func testParentStop(t *testing.T, stop func(*io.PipeWriter) error) {
	t.Helper()
	token := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x2a}, 32))
	input := fmt.Sprintf(
		`{"protocolVersion":"1","token":"%s","dataDir":%q}`,
		token,
		t.TempDir(),
	)
	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()
	result := make(chan error, 1)
	go func() {
		result <- Run(context.Background(), []string{"--bootstrap-stdio"}, Streams{
			Stdin: stdinReader, Stdout: stdoutWriter, Stderr: io.Discard,
		})
	}()
	if _, err := fmt.Fprintln(stdinWriter, input); err != nil {
		t.Fatalf("write bootstrap: %v", err)
	}
	var handshake bootstrap.Handshake
	if err := json.NewDecoder(stdoutReader).Decode(&handshake); err != nil {
		t.Fatalf("decode handshake: %v", err)
	}
	streamRequest, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/api/v1/events", handshake.Port), nil)
	if err != nil {
		t.Fatalf("create event stream request: %v", err)
	}
	streamRequest.Header.Set("Accept", "text/event-stream")
	streamRequest.Header.Set("Authorization", "Bearer "+token)
	streamResponse, err := (&http.Client{Timeout: 2 * time.Second}).Do(streamRequest)
	if err != nil {
		t.Fatalf("connect event stream: %v", err)
	}
	defer streamResponse.Body.Close()
	line, err := bufio.NewReader(streamResponse.Body).ReadString('\n')
	if err != nil || line != ": connected\n" {
		t.Fatalf("event stream first line = %q, error = %v", line, err)
	}
	if err := stop(stdinWriter); err != nil {
		t.Fatalf("stop parent channel: %v", err)
	}
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("daemon did not stop after parent channel ended")
	}
}

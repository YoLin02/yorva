package mcpmanagement

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	GitHubReposReadOnlyEndpoint        = "https://api.githubcopilot.com/mcp/x/repos/readonly"
	GitHubQualificationSourceRevision  = "8898db96b8043db1ddacff444600e408cf824fa2"
	githubQualificationProtocolVersion = "2025-06-18"
	githubQualificationTimeout         = 20 * time.Second
	maxQualificationResponseBytes      = 512 * 1024
	maxQualificationTools              = 64
)

var (
	ErrGitHubQualificationCredential = errors.New("GitHub MCP qualification credential is invalid")
	ErrGitHubQualificationTransport  = errors.New("GitHub MCP qualification transport failed")
	ErrGitHubQualificationRedirect   = errors.New("GitHub MCP qualification redirect rejected")
	ErrGitHubQualificationAuth       = errors.New("GitHub MCP qualification authentication failed")
	ErrGitHubQualificationProtocol   = errors.New("GitHub MCP qualification protocol failed")
	ErrGitHubQualificationMismatch   = errors.New("GitHub MCP qualification tool inventory mismatch")
	ErrGitHubQualificationLimit      = errors.New("GitHub MCP qualification response exceeds a limit")
	ErrGitHubQualificationCanceled   = errors.New("GitHub MCP qualification canceled")
)

// This inventory is pinned to GitHub's official github-mcp-server source at
// GitHubQualificationSourceRevision. It is qualification evidence only: the
// reviewed product registry remains empty until a later explicit gate.
var githubReposReadOnlyTools = [...]string{
	"get_commit",
	"get_file_blame",
	"get_file_contents",
	"get_latest_release",
	"get_release_by_tag",
	"get_tag",
	"list_branches",
	"list_commits",
	"list_releases",
	"list_repository_collaborators",
	"list_tags",
	"search_code",
	"search_commits",
	"search_repositories",
}

// GitHubQualificationResult intentionally exposes no endpoint, account,
// credential, response body, tool descriptions, or server error text.
type GitHubQualificationResult struct {
	toolCount int
}

func (r GitHubQualificationResult) Ready() bool { return r.toolCount == len(githubReposReadOnlyTools) }

func (r GitHubQualificationResult) ToolCount() int { return r.toolCount }

func (r GitHubQualificationResult) SourceRevision() string {
	return GitHubQualificationSourceRevision
}

// QualifyGitHubReposReadOnly performs a bounded, read-only MCP handshake with
// the one reviewed candidate. The credential is request-lifetime only and is
// never retained or returned. This function does not mutate Hermes config.
func QualifyGitHubReposReadOnly(ctx context.Context, credential []byte) (GitHubQualificationResult, error) {
	return qualifyGitHubReposReadOnly(ctx, credential, GitHubReposReadOnlyEndpoint, newGitHubQualificationHTTPClient())
}

func newGitHubQualificationHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// A process-environment proxy must never receive or forward this
	// request-lifetime bearer credential. The sole candidate is contacted
	// directly over TLS and redirects are rejected by the per-run client copy.
	transport.Proxy = nil
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	transport.ResponseHeaderTimeout = githubQualificationTimeout
	return &http.Client{Transport: transport, Timeout: githubQualificationTimeout}
}

func qualifyGitHubReposReadOnly(ctx context.Context, credential []byte, endpoint string, client *http.Client) (GitHubQualificationResult, error) {
	if err := validateStaticBearer(credential); err != nil {
		return GitHubQualificationResult{}, ErrGitHubQualificationCredential
	}
	if ctx == nil || client == nil {
		return GitHubQualificationResult{}, ErrGitHubQualificationTransport
	}

	ctx, cancel := context.WithTimeout(ctx, githubQualificationTimeout)
	defer cancel()
	clientCopy := *client
	clientCopy.Timeout = githubQualificationTimeout
	clientCopy.CheckRedirect = func(*http.Request, []*http.Request) error {
		return ErrGitHubQualificationRedirect
	}
	defer clientCopy.CloseIdleConnections()

	authorization := "Bearer " + string(credential)
	initBody := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"yorva-b5-qualification","version":"1"}}}`)
	initResponse, sessionID, err := qualificationPost(ctx, &clientCopy, endpoint, authorization, "", initBody, http.StatusOK)
	if err != nil {
		return GitHubQualificationResult{}, normalizeQualificationError(ctx, err)
	}
	if err := validateInitializeResponse(initResponse); err != nil {
		return GitHubQualificationResult{}, err
	}

	initializedBody := []byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	if _, _, err := qualificationPost(ctx, &clientCopy, endpoint, authorization, sessionID, initializedBody, http.StatusAccepted, http.StatusNoContent); err != nil {
		return GitHubQualificationResult{}, normalizeQualificationError(ctx, err)
	}

	toolsBody := []byte(`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`)
	toolsResponse, _, err := qualificationPost(ctx, &clientCopy, endpoint, authorization, sessionID, toolsBody, http.StatusOK)
	if err != nil {
		return GitHubQualificationResult{}, normalizeQualificationError(ctx, err)
	}
	count, err := validateToolsResponse(toolsResponse)
	if err != nil {
		return GitHubQualificationResult{}, err
	}
	return GitHubQualificationResult{toolCount: count}, nil
}

func qualificationPost(ctx context.Context, client *http.Client, endpoint, authorization, sessionID string, body []byte, acceptedStatus ...int) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, "", ErrGitHubQualificationTransport
	}
	req.Header.Set("Authorization", authorization)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", githubQualificationProtocolVersion)
	if sessionID != "" {
		req.Header.Set("Mcp-Session-Id", sessionID)
	}

	response, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return nil, "", ErrGitHubQualificationAuth
	}
	accepted := false
	for _, status := range acceptedStatus {
		if response.StatusCode == status {
			accepted = true
			break
		}
	}
	if !accepted {
		return nil, "", ErrGitHubQualificationProtocol
	}

	newSessionID := response.Header.Get("Mcp-Session-Id")
	if newSessionID != "" && !validSessionID(newSessionID) {
		return nil, "", ErrGitHubQualificationProtocol
	}
	if response.StatusCode == http.StatusNoContent || response.StatusCode == http.StatusAccepted {
		return nil, newSessionID, nil
	}
	payload, err := readQualificationPayload(response)
	if err != nil {
		return nil, "", err
	}
	return payload, newSessionID, nil
}

func validSessionID(value string) bool {
	if len(value) == 0 || len(value) > 256 {
		return false
	}
	for _, ch := range []byte(value) {
		if ch < 0x21 || ch > 0x7e {
			return false
		}
	}
	return true
}

func readQualificationPayload(response *http.Response) ([]byte, error) {
	mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil {
		return nil, ErrGitHubQualificationProtocol
	}
	limited := io.LimitReader(response.Body, maxQualificationResponseBytes+1)
	if mediaType == "application/json" {
		payload, readErr := io.ReadAll(limited)
		if readErr != nil {
			return nil, ErrGitHubQualificationTransport
		}
		if len(payload) > maxQualificationResponseBytes {
			return nil, ErrGitHubQualificationLimit
		}
		return payload, nil
	}
	if mediaType != "text/event-stream" {
		return nil, ErrGitHubQualificationProtocol
	}

	scanner := bufio.NewScanner(limited)
	scanner.Buffer(make([]byte, 4096), maxQualificationResponseBytes)
	var payload []byte
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data:") {
			if payload != nil {
				return nil, ErrGitHubQualificationProtocol
			}
			payload = []byte(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	if scanner.Err() != nil {
		return nil, ErrGitHubQualificationLimit
	}
	if len(payload) == 0 {
		return nil, ErrGitHubQualificationProtocol
	}
	return payload, nil
}

type qualificationEnvelope struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   json.RawMessage `json:"error"`
}

func decodeQualificationEnvelope(payload []byte, expectedID string) (json.RawMessage, error) {
	if len(payload) == 0 || !json.Valid(payload) {
		return nil, ErrGitHubQualificationProtocol
	}
	var envelope qualificationEnvelope
	decoder := json.NewDecoder(bytes.NewReader(payload))
	if err := decoder.Decode(&envelope); err != nil || envelope.JSONRPC != "2.0" || string(envelope.ID) != expectedID || len(envelope.Result) == 0 || len(envelope.Error) != 0 {
		return nil, ErrGitHubQualificationProtocol
	}
	if err := requireJSONEOF(decoder); err != nil {
		return nil, ErrGitHubQualificationProtocol
	}
	return envelope.Result, nil
}

func validateInitializeResponse(payload []byte) error {
	result, err := decodeQualificationEnvelope(payload, "1")
	if err != nil {
		return err
	}
	var initialize struct {
		ProtocolVersion string          `json:"protocolVersion"`
		Capabilities    json.RawMessage `json:"capabilities"`
		ServerInfo      json.RawMessage `json:"serverInfo"`
	}
	if err := json.Unmarshal(result, &initialize); err != nil || initialize.ProtocolVersion != githubQualificationProtocolVersion || len(initialize.Capabilities) == 0 || len(initialize.ServerInfo) == 0 {
		return ErrGitHubQualificationProtocol
	}
	return nil
}

func validateToolsResponse(payload []byte) (int, error) {
	result, err := decodeQualificationEnvelope(payload, "2")
	if err != nil {
		return 0, err
	}
	var listing struct {
		Tools []struct {
			Name        string `json:"name"`
			Annotations struct {
				ReadOnlyHint *bool `json:"readOnlyHint"`
			} `json:"annotations"`
		} `json:"tools"`
		NextCursor string `json:"nextCursor"`
	}
	if err := json.Unmarshal(result, &listing); err != nil || len(listing.Tools) == 0 {
		return 0, ErrGitHubQualificationProtocol
	}
	if len(listing.Tools) > maxQualificationTools {
		return 0, ErrGitHubQualificationLimit
	}
	if listing.NextCursor != "" {
		return 0, ErrGitHubQualificationMismatch
	}
	observed := make([]string, 0, len(listing.Tools))
	seen := make(map[string]struct{}, len(listing.Tools))
	for _, tool := range listing.Tools {
		if !validToolID(tool.Name) || tool.Annotations.ReadOnlyHint == nil || !*tool.Annotations.ReadOnlyHint {
			return 0, ErrGitHubQualificationMismatch
		}
		if _, duplicate := seen[tool.Name]; duplicate {
			return 0, ErrGitHubQualificationMismatch
		}
		seen[tool.Name] = struct{}{}
		observed = append(observed, tool.Name)
	}
	sort.Strings(observed)
	if len(observed) != len(githubReposReadOnlyTools) {
		return 0, ErrGitHubQualificationMismatch
	}
	for index, expected := range githubReposReadOnlyTools {
		if observed[index] != expected {
			return 0, ErrGitHubQualificationMismatch
		}
	}
	return len(observed), nil
}

func normalizeQualificationError(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ErrGitHubQualificationCanceled
	}
	if errors.Is(err, ErrGitHubQualificationRedirect) {
		return ErrGitHubQualificationRedirect
	}
	for _, stable := range []error{ErrGitHubQualificationAuth, ErrGitHubQualificationProtocol, ErrGitHubQualificationLimit} {
		if errors.Is(err, stable) {
			return stable
		}
	}
	return ErrGitHubQualificationTransport
}

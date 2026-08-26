package mcpmanagement

import (
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

const reviewedHTTPSProbeTimeout = 20 * time.Second

var (
	ErrReviewedProbeTransport = errors.New("reviewed MCP probe transport failed")
	ErrReviewedProbeProtocol  = errors.New("reviewed MCP probe protocol failed")
	ErrReviewedProbeAuth      = errors.New("reviewed MCP probe authentication failed")
	ErrReviewedProbeMismatch  = errors.New("reviewed MCP probe tool inventory mismatch")
)

// ProbeReviewedHTTPS performs the fixed initialize/tools-list handshake for a
// Selection obtained from Registry. Endpoint, header shape and allowed tools
// remain descriptor-owned; callers can supply only the request-lifetime secret
// required by that descriptor.
func ProbeReviewedHTTPS(ctx context.Context, selection Selection, scope ProfileScope, credential []byte, client *http.Client, now func() time.Time) (ProbeResult, error) {
	if ctx == nil || !selection.valid() || !scope.valid() || client == nil || now == nil {
		return ProbeResult{}, ErrReviewedProbeProtocol
	}
	if err := selection.ValidateCredential(credential); err != nil {
		return ProbeResult{}, err
	}
	endpoint, err := selection.HTTPSURL()
	if err != nil {
		return ProbeResult{}, err
	}

	probeCtx, cancel := context.WithTimeout(ctx, reviewedHTTPSProbeTimeout)
	defer cancel()
	clientCopy := *client
	clientCopy.Timeout = reviewedHTTPSProbeTimeout
	clientCopy.CheckRedirect = func(*http.Request, []*http.Request) error { return ErrReviewedProbeProtocol }
	defer clientCopy.CloseIdleConnections()

	authorization := ""
	credentialStatus := CredentialStatusNotRequired
	if selection.CredentialClass() == CredentialClassStaticBearer {
		authorization = "Bearer " + string(credential)
		credentialStatus = CredentialStatusConfigured
	}
	initBody := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"yorva","version":"1"}}}`)
	initPayload, sessionID, err := reviewedPost(probeCtx, &clientCopy, endpoint, authorization, "", initBody, http.StatusOK)
	if err != nil {
		return ProbeResult{}, err
	}
	if err := validateInitializeResponse(initPayload); err != nil {
		return ProbeResult{}, ErrReviewedProbeProtocol
	}
	if _, _, err := reviewedPost(probeCtx, &clientCopy, endpoint, authorization, sessionID, []byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`), http.StatusAccepted, http.StatusNoContent); err != nil {
		return ProbeResult{}, err
	}
	toolsPayload, _, err := reviewedPost(probeCtx, &clientCopy, endpoint, authorization, sessionID, []byte(`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`), http.StatusOK)
	if err != nil {
		return ProbeResult{}, err
	}
	toolIDs, err := reviewedToolIDs(toolsPayload)
	if err != nil {
		return ProbeResult{}, err
	}
	expected := selection.AllowedToolIDs()
	sort.Strings(expected)
	if len(toolIDs) != len(expected) {
		return ProbeResult{}, ErrReviewedProbeMismatch
	}
	for index := range expected {
		if toolIDs[index] != expected[index] {
			return ProbeResult{}, ErrReviewedProbeMismatch
		}
	}
	return ProbeResult{
		profileID: scope.profileID, presetID: selection.descriptor.presetID,
		outcome: ProbeOutcomeReady, credentialStatus: credentialStatus,
		toolIDs: toolIDs, observedAt: now().UTC(),
	}, nil
}

func NewReviewedHTTPSClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	transport.ResponseHeaderTimeout = reviewedHTTPSProbeTimeout
	return &http.Client{Transport: transport, Timeout: reviewedHTTPSProbeTimeout}
}

func reviewedPost(ctx context.Context, client *http.Client, endpoint, authorization, sessionID string, body []byte, accepted ...int) ([]byte, string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, "", ErrReviewedProbeTransport
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("MCP-Protocol-Version", githubQualificationProtocolVersion)
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	if sessionID != "" {
		request.Header.Set("Mcp-Session-Id", sessionID)
	}
	response, err := client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return nil, "", ctx.Err()
		}
		return nil, "", ErrReviewedProbeTransport
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return nil, "", ErrReviewedProbeAuth
	}
	ok := false
	for _, status := range accepted {
		ok = ok || response.StatusCode == status
	}
	if !ok {
		return nil, "", ErrReviewedProbeProtocol
	}
	newSessionID := response.Header.Get("Mcp-Session-Id")
	if newSessionID != "" && !validSessionID(newSessionID) {
		return nil, "", ErrReviewedProbeProtocol
	}
	if response.StatusCode == http.StatusAccepted || response.StatusCode == http.StatusNoContent {
		return nil, newSessionID, nil
	}
	mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return nil, "", ErrReviewedProbeProtocol
	}
	payload, err := io.ReadAll(io.LimitReader(response.Body, MaxProbeBytes+1))
	if err != nil || len(payload) == 0 || len(payload) > MaxProbeBytes {
		return nil, "", ErrReviewedProbeProtocol
	}
	return payload, newSessionID, nil
}

func reviewedToolIDs(payload []byte) ([]string, error) {
	result, err := decodeQualificationEnvelope(payload, "2")
	if err != nil {
		return nil, ErrReviewedProbeProtocol
	}
	var listing struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
		NextCursor string `json:"nextCursor"`
	}
	decoder := json.NewDecoder(bytes.NewReader(result))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&listing); err != nil || listing.NextCursor != "" || len(listing.Tools) == 0 || len(listing.Tools) > MaxProbeTools {
		return nil, ErrReviewedProbeProtocol
	}
	if err := requireJSONEOF(decoder); err != nil {
		return nil, ErrReviewedProbeProtocol
	}
	ids := make([]string, 0, len(listing.Tools))
	seen := make(map[string]struct{}, len(listing.Tools))
	for _, tool := range listing.Tools {
		if !validToolID(tool.Name) || strings.TrimSpace(tool.Name) != tool.Name {
			return nil, ErrReviewedProbeProtocol
		}
		if _, duplicate := seen[tool.Name]; duplicate {
			return nil, ErrReviewedProbeProtocol
		}
		seen[tool.Name] = struct{}{}
		ids = append(ids, tool.Name)
	}
	sort.Strings(ids)
	return ids, nil
}

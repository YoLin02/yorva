package hermes

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes/managementhealth"
	"github.com/YoLin02/yorva/services/node/internal/runtime/hermes/skillsmanagement"
)

const apiDetailedHealthFixture = `{
  "status":"degraded",
  "readiness":{"status":"degraded","checks":{
    "state_db":{"status":"ok"},
    "session_store":{"status":"retrying"},
    "config":{"status":"ok"},
    "model":{"status":"ok"},
    "disk":{"status":"ok","used_percent":41.2,"free_bytes":123456},
    "gateway":{"status":"ok","state":"running","connected_platforms":1,"platforms":2},
    "background_queues":{"status":"ok","active_api_runs":2,"process_completions":0,"active_delegations":1}
  }},
  "platform":"hermes-agent","version":"0.20.5","gateway_state":"running",
  "platforms":{"weixin":{"account":"must-not-project"}},"active_agents":4,
  "gateway_busy":true,"gateway_drainable":false,"exit_reason":null,
  "updated_at":"2026-08-25T10:00:00Z","pid":4242
}`

func TestAPIManagementReaderProjectsHealthAndSkills(t *testing.T) {
	observedAt := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	reader := &APIManagementReader{
		resolve: func(version, profile string) (apiManagementTarget, error) {
			if version != apiManagementVersion || profile != "work" {
				return apiManagementTarget{}, errAPIManagementUnavailable
			}
			return apiManagementTarget{host: "127.0.0.1", port: 9864, pathPrefix: "/p/work", key: []byte(namedAPIKey)}, nil
		},
		fetch: func(_ context.Context, target apiManagementTarget, path string, maximum int) ([]byte, error) {
			if string(target.key) != namedAPIKey {
				t.Fatal("request did not use the named Profile key")
			}
			switch path {
			case "/p/work/health/detailed":
				if maximum != managementhealth.MaxDetailedResponseBytes {
					t.Fatalf("health maximum = %d", maximum)
				}
				return []byte(apiDetailedHealthFixture), nil
			case "/p/work/v1/skills":
				if maximum != skillsmanagement.MaxInventoryBodyBytes {
					t.Fatalf("skills maximum = %d", maximum)
				}
				return []byte(`{"object":"list","data":[{"name":"github","description":"GitHub workflow","category":null}]}`), nil
			default:
				t.Fatalf("unexpected path %q", path)
				return nil, nil
			}
		},
		now: func() time.Time { return observedAt },
	}
	installation := qualifiedAPIManagementInstallation()

	features, err := reader.ResolveInstanceManagement(context.Background(), installation, "work")
	if err != nil || features.Health == nil || features.SkillRead == nil || features.Logs != nil || features.Security != nil || features.MCPRead != nil {
		t.Fatalf("resolved features = %#v, %v", features, err)
	}
	health, err := reader.InspectInstanceHealth(context.Background(), installation, "work")
	if err != nil {
		t.Fatalf("InspectInstanceHealth: %v", err)
	}
	if health.State != yorvaruntime.HealthDegraded || health.ObservedAt != observedAt || len(health.Findings) != 7 || health.Findings[1].State != yorvaruntime.HealthDegraded {
		t.Fatalf("health = %#v", health)
	}
	for _, finding := range health.Findings {
		if strings.Contains(finding.Code, "must-not-project") || strings.Contains(finding.Code, "4242") {
			t.Fatalf("unsafe health projection = %#v", health)
		}
	}
	skills, err := reader.ListSkills(context.Background(), installation, "work")
	if err != nil || len(skills) != 1 {
		t.Fatalf("ListSkills = %#v, %v", skills, err)
	}
	if skill := skills[0]; skill.ID != "github" || skill.InstallationState != yorvaruntime.SkillInstalled || skill.EnabledState != yorvaruntime.SkillEnabled || skill.ScanState != yorvaruntime.SkillScanUnknown || skill.SourceID != "" || skill.Version != "" || skill.UpdateAvailable {
		t.Fatalf("skill projection = %#v", skill)
	}
	if inspected, err := reader.InspectSkill(context.Background(), installation, "work", "github"); err != nil || inspected.ID != "github" {
		t.Fatalf("InspectSkill = %#v, %v", inspected, err)
	}
}

func TestAPIManagementReaderFailsClosedBeforeResolution(t *testing.T) {
	var resolves atomic.Int32
	reader := &APIManagementReader{
		resolve: func(string, string) (apiManagementTarget, error) {
			resolves.Add(1)
			return apiManagementTarget{}, nil
		},
		fetch: fetchAPIManagementJSON,
	}
	installation := qualifiedAPIManagementInstallation()
	installation.Version = "0.20.2"
	if features, err := reader.ResolveInstanceManagement(context.Background(), installation, "default"); err == nil || features.Health != nil || features.SkillRead != nil {
		t.Fatalf("unqualified features = %#v, %v", features, err)
	}
	if resolves.Load() != 0 {
		t.Fatal("unqualified version reached Profile configuration")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	installation.Version = apiManagementVersion
	if _, err := reader.ListSkills(ctx, installation, "default"); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled ListSkills error = %v", err)
	}
}

func TestFetchAPIManagementJSONUsesClosedAuthenticatedLoopbackRequest(t *testing.T) {
	key := "request-only-key-0123456789"
	var forbiddenHits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/forbidden" {
			forbiddenHits.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
			return
		}
		if request.Method != http.MethodGet || request.Header.Get("Authorization") != "Bearer "+key || request.Header.Get("Accept") != "application/json" {
			t.Error("request method or fixed authentication headers did not match")
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte(`{"object":"list","data":[]}`))
	}))
	defer server.Close()
	target := targetForTestServer(t, server, key)
	body, err := fetchAPIManagementJSON(context.Background(), target, "/v1/skills", 1024)
	if err != nil || string(body) != `{"object":"list","data":[]}` {
		t.Fatalf("fetch = %q, %v", body, err)
	}

	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		http.Redirect(w, request, server.URL+"/forbidden", http.StatusFound)
	}))
	defer redirect.Close()
	redirectTarget := targetForTestServer(t, redirect, key)
	if _, err := fetchAPIManagementJSON(context.Background(), redirectTarget, "/v1/skills", 1024); !errors.Is(err, errAPIManagementRequest) {
		t.Fatalf("redirect error = %v", err)
	}
	if forbiddenHits.Load() != 0 {
		t.Fatal("management client followed a redirect")
	}
}

func TestFetchAPIManagementJSONRejectsUnsafeTargetsAndResponses(t *testing.T) {
	if _, err := fetchAPIManagementJSON(context.Background(), apiManagementTarget{host: "example.com", port: 80, key: []byte(defaultAPIKey)}, "/v1/skills", 1024); !errors.Is(err, errAPIManagementRequest) {
		t.Fatalf("remote target error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(strings.Repeat("x", 33)))
	}))
	defer server.Close()
	target := targetForTestServer(t, server, defaultAPIKey)
	if _, err := fetchAPIManagementJSON(context.Background(), target, "/health/detailed", 32); !errors.Is(err, errAPIManagementResponse) {
		t.Fatalf("oversized body error = %v", err)
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := fetchAPIManagementJSON(canceled, target, "/health/detailed", 32); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled fetch error = %v", err)
	}
}

func qualifiedAPIManagementInstallation() yorvaruntime.Installation {
	return yorvaruntime.Installation{RuntimeKind: Kind, Path: filepathForAPITest(), Version: apiManagementVersion, SupportState: yorvaruntime.DiscoverySupported}
}

func filepathForAPITest() string {
	path, err := filepath.Abs("hermes-test-executable")
	if err != nil {
		panic(err)
	}
	return path
}

func targetForTestServer(t *testing.T, server *httptest.Server, key string) apiManagementTarget {
	t.Helper()
	host, portText, err := net.SplitHostPort(strings.TrimPrefix(server.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatal(err)
	}
	return apiManagementTarget{host: host, port: port, key: []byte(key)}
}

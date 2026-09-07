package httpapi

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/YoLin02/yorva/services/node/internal/app"
	"github.com/YoLin02/yorva/services/node/internal/buildinfo"
	"github.com/YoLin02/yorva/services/node/internal/domain/node"
	"github.com/YoLin02/yorva/services/node/internal/events"
	yorvaruntime "github.com/YoLin02/yorva/services/node/internal/runtime"
)

type HealthResponse struct {
	Status          string `json:"status"`
	Service         string `json:"service"`
	Version         string `json:"version"`
	ProtocolVersion string `json:"protocolVersion"`
}

type RuntimeDiscoveryService interface {
	Detect(context.Context, yorvaruntime.Kind) (yorvaruntime.Discovery, error)
}

type RuntimeDiscoveryResponse struct {
	RuntimeKind    yorvaruntime.Kind           `json:"runtimeKind"`
	State          yorvaruntime.DiscoveryState `json:"state"`
	ErrorCode      *yorvaruntime.ErrorCode     `json:"errorCode"`
	Selected       *RuntimeCandidateResponse   `json:"selected"`
	Candidates     []RuntimeCandidateResponse  `json:"candidates"`
	Warnings       []RuntimeWarningResponse    `json:"warnings"`
	DetectedAt     time.Time                   `json:"detectedAt"`
	SupportedRange string                      `json:"supportedRange"`
}

type RuntimeCandidateResponse struct {
	Path      string                      `json:"path"`
	Version   string                      `json:"version"`
	State     yorvaruntime.DiscoveryState `json:"state"`
	ErrorCode *yorvaruntime.ErrorCode     `json:"errorCode"`
}

type RuntimeWarningResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewHandler(token string, localNode node.Node, broker *events.Broker, runtimes RuntimeDiscoveryService, installs RuntimeInstallService, instances InstanceInventoryService, dataDir string, sourceSettings HermesDownloadSourceSettingsService, diagnosticServices ...DiagnosticBundleService) http.Handler {
	mux := http.NewServeMux()
	var diagnosticService DiagnosticBundleService
	if len(diagnosticServices) > 0 {
		diagnosticService = diagnosticServices[0]
	}
	recovery, _ := instances.(NodeRecoveryService)
	models, _ := instances.(ModelConfigurationService)
	sharedModels, _ := instances.(SharedModelService)
	lifecycle, _ := instances.(InstanceLifecycleService)
	channels, _ := instances.(ChannelService)
	var skills ManagementSkillsService
	var mcp ManagementMCPService
	var managementHealth InstanceManagementHealthService
	var managementUpgrade ManagementUpgradeService
	var managementBackups ManagementBackupService
	if targets, ok := instances.(app.ManagementTargetResolver); ok {
		skills = app.NewManagementSkills(targets)
		mcp = app.NewMCPManagement(targets)
		managementHealth = app.NewManagementHealth(targets)
	}
	if factory, ok := instances.(interface {
		NewManagementSkills(string) (*app.ManagementSkills, error)
	}); ok {
		if managed, err := factory.NewManagementSkills(dataDir); err == nil {
			skills = managed
		}
	}
	if factory, ok := instances.(interface {
		NewMCPManagement() (*app.MCPManagement, error)
	}); ok {
		if managed, err := factory.NewMCPManagement(); err == nil {
			mcp = managed
		}
	}
	if targets, ok := instances.(app.RuntimeManagementTargetResolver); ok {
		managementUpgrade = app.NewManagementUpgrade(targets)
		managementBackups = app.NewBackupManagement(targets)
		if factory, ok := instances.(interface {
			NewBackupManagement() (*app.BackupManagement, error)
		}); ok {
			managementBackups, _ = factory.NewBackupManagement()
		}
	}
	if factory, ok := instances.(interface {
		NewManagementUpgrade() (*app.ManagementUpgrade, error)
	}); ok {
		if managed, err := factory.NewManagementUpgrade(); err == nil {
			managementUpgrade = managed
		}
	}
	mux.HandleFunc("GET /api/v1/health", health)
	mux.Handle("GET /api/v1/node", requireBearer(token, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(localNode)
	})))
	mux.Handle("GET /api/v1/node/recovery", requireBearer(token, getNodeRecovery(recovery, localNode.NodeVersion)))
	mux.Handle("GET /api/v1/events", requireBearer(token, eventStream(broker, 15*time.Second)))
	mux.Handle("POST /api/v1/diagnostics/bundle", requireBearer(token, exportDiagnosticBundle(diagnosticService)))
	mux.Handle("POST /api/v1/runtimes/{runtimeKind}/detect", requireBearer(token, detectRuntime(runtimes)))
	mux.Handle("POST /api/v1/runtimes/hermes/install", requireBearer(token, startHermesInstall(installs)))
	mux.Handle("GET /api/v1/runtimes/hermes/prerequisites", requireBearer(token, getHermesPrerequisites(installs)))
	mux.Handle("POST /api/v1/runtimes/hermes/prerequisites/install", requireBearer(token, startHermesPrerequisites(installs)))
	mux.Handle("GET /api/v1/settings/hermes/download-sources", requireBearer(token, getHermesDownloadSources(sourceSettings)))
	mux.Handle("PUT /api/v1/settings/hermes/download-sources", requireBearer(token, putHermesDownloadSources(sourceSettings)))
	mux.Handle("DELETE /api/v1/settings/hermes/download-sources", requireBearer(token, deleteHermesDownloadSources(sourceSettings)))
	mux.Handle("GET /api/v1/runtimes/{runtimeId}/instances", requireBearer(token, listRuntimeInstances(instances)))
	mux.Handle("GET /api/v1/runtimes/{runtimeId}/upgrade-plan", requireBearer(token, getRuntimeUpgradePlan(managementUpgrade)))
	mux.Handle("POST /api/v1/runtimes/{runtimeId}/upgrade", requireBearer(token, startRuntimeUpgrade(managementUpgrade, false)))
	mux.Handle("POST /api/v1/runtimes/{runtimeId}/rollback", requireBearer(token, startRuntimeUpgrade(managementUpgrade, true)))
	mux.Handle("GET /api/v1/runtimes/{runtimeId}/backups", requireBearer(token, listRuntimeBackups(managementBackups)))
	mux.Handle("GET /api/v1/runtimes/{runtimeId}/mcp-definitions", requireBearer(token, listRuntimeMCPDefinitions(mcp)))
	mux.Handle("POST /api/v1/runtimes/{runtimeId}/backups", requireBearer(token, startRuntimeBackup(managementBackups)))
	mux.Handle("GET /api/v1/runtimes/{runtimeId}/backups/{backupId}", requireBearer(token, getRuntimeBackup(managementBackups)))
	mux.Handle("DELETE /api/v1/backups/{backupId}", requireBearer(token, startDeleteBackup(managementBackups)))
	mux.Handle("POST /api/v1/backups/{backupId}/restore", requireBearer(token, startRestoreBackup(managementBackups)))
	mux.Handle("POST /api/v1/runtimes/{runtimeId}/instances", requireBearer(token, createRuntimeInstance(instances)))
	mux.Handle("GET /api/v1/runtimes/hermes/model-provider-presets", requireBearer(token, listModelProviderPresets(models)))
	mux.Handle("GET /api/v1/runtimes/{runtimeId}/model-provider-connections", requireBearer(token, listModelProviderConnections(sharedModels)))
	mux.Handle("POST /api/v1/runtimes/{runtimeId}/model-provider-connections", requireBearer(token, createModelProviderConnection(sharedModels)))
	mux.Handle("DELETE /api/v1/runtimes/{runtimeId}/model-provider-connections/{connectionId}", requireBearer(token, deleteModelProviderConnection(sharedModels)))
	mux.Handle("GET /api/v1/runtimes/{runtimeId}/model-profiles", requireBearer(token, listModelProfiles(sharedModels)))
	mux.Handle("POST /api/v1/runtimes/{runtimeId}/model-profiles", requireBearer(token, createModelProfile(sharedModels)))
	mux.Handle("DELETE /api/v1/runtimes/{runtimeId}/model-profiles/{profileId}", requireBearer(token, deleteModelProfile(sharedModels)))
	mux.Handle("GET /api/v1/runtimes/{runtimeId}/model-default", requireBearer(token, getRuntimeModelDefault(sharedModels)))
	mux.Handle("PUT /api/v1/runtimes/{runtimeId}/model-default", requireBearer(token, putRuntimeModelDefault(sharedModels)))
	mux.Handle("DELETE /api/v1/runtimes/{runtimeId}/model-default", requireBearer(token, deleteRuntimeModelDefault(sharedModels)))
	mux.Handle("GET /api/v1/runtimes/{runtimeId}/model-bindings", requireBearer(token, listInstanceModelBindings(sharedModels)))
	mux.Handle("POST /api/v1/runtimes/{runtimeId}/model-profile-applications", requireBearer(token, startModelProfileApplication(sharedModels)))
	mux.Handle("GET /api/v1/instances/{instanceId}", requireBearer(token, getInstance(instances)))
	mux.Handle("DELETE /api/v1/instances/{instanceId}", requireBearer(token, deleteInstance(instances)))
	var removedRecords RemovedInstanceRecordService
	if service, ok := instances.(RemovedInstanceRecordService); ok {
		removedRecords = service
	}
	mux.Handle("DELETE /api/v1/instances/{instanceId}/record", requireBearer(token, clearRemovedInstanceRecord(removedRecords)))
	mux.Handle("GET /api/v1/instances/{instanceId}/config", requireBearer(token, getModelConfiguration(models)))
	mux.Handle("PATCH /api/v1/instances/{instanceId}/config", requireBearer(token, patchModelConfiguration(models)))
	mux.Handle("POST /api/v1/instances/{instanceId}/model-provider-models", requireBearer(token, fetchModelProviderCatalog(models)))
	mux.Handle("GET /api/v1/instances/{instanceId}/credentials/model-provider", requireBearer(token, getModelCredential(models)))
	mux.Handle("PUT /api/v1/instances/{instanceId}/credentials/model-provider", requireBearer(token, putModelCredential(models)))
	mux.Handle("DELETE /api/v1/instances/{instanceId}/credentials/model-provider", requireBearer(token, deleteModelCredential(models)))
	mux.Handle("POST /api/v1/instances/{instanceId}/model-validation", requireBearer(token, startModelValidation(models)))
	mux.Handle("GET /api/v1/instances/{instanceId}/lifecycle", requireBearer(token, getInstanceLifecycle(lifecycle)))
	mux.Handle("POST /api/v1/instances/{instanceId}/start", requireBearer(token, startInstanceLifecycle(lifecycle, app.LifecycleStart)))
	mux.Handle("POST /api/v1/instances/{instanceId}/stop", requireBearer(token, startInstanceLifecycle(lifecycle, app.LifecycleStop)))
	mux.Handle("POST /api/v1/instances/{instanceId}/restart", requireBearer(token, startInstanceLifecycle(lifecycle, app.LifecycleRestart)))
	mux.Handle("GET /api/v1/instances/{instanceId}/health", requireBearer(token, getInstanceHealth(managementHealth)))
	mux.Handle("GET /api/v1/instances/{instanceId}/logs", requireBearer(token, getInstanceLogSnapshot(managementHealth)))
	mux.Handle("GET /api/v1/instances/{instanceId}/channels", requireBearer(token, listInstanceChannels(channels)))
	mux.Handle("POST /api/v1/instances/{instanceId}/channels/{channelType}/connect", requireBearer(token, connectInstanceChannel(channels)))
	mux.Handle("DELETE /api/v1/instances/{instanceId}/channels/{channelType}", requireBearer(token, disconnectInstanceChannel(channels)))
	mux.Handle("GET /api/v1/instances/{instanceId}/channels/{channelType}/pairings", requireBearer(token, getChannelPairingStatus(channels)))
	mux.Handle("POST /api/v1/instances/{instanceId}/channels/{channelType}/pairings/approve", requireBearer(token, approveChannelPairing(channels)))
	mux.Handle("GET /api/v1/instances/{instanceId}/skills", requireBearer(token, listInstanceSkills(skills)))
	mux.Handle("GET /api/v1/instances/{instanceId}/skills/{skillId}", requireBearer(token, inspectInstanceSkill(skills)))
	mux.Handle("GET /api/v1/instances/{instanceId}/skill-sources", requireBearer(token, listInstanceSkillSources(skills)))
	mux.Handle("POST /api/v1/instances/{instanceId}/skills/{skillId}/install", requireBearer(token, startManagedSkillMutation(skills, skillMutationInstall)))
	mux.Handle("POST /api/v1/instances/{instanceId}/skills/{skillId}/import", requireBearer(token, startManagedSkillMutation(skills, skillMutationImport)))
	mux.Handle("POST /api/v1/instances/{instanceId}/skills/{skillId}/update", requireBearer(token, startManagedSkillMutation(skills, skillMutationUpdate)))
	mux.Handle("POST /api/v1/instances/{instanceId}/skills/{skillId}/enable", requireBearer(token, startManagedSkillMutation(skills, skillMutationEnable)))
	mux.Handle("POST /api/v1/instances/{instanceId}/skills/{skillId}/disable", requireBearer(token, startManagedSkillMutation(skills, skillMutationDisable)))
	mux.Handle("DELETE /api/v1/instances/{instanceId}/skills/{skillId}", requireBearer(token, startManagedSkillMutation(skills, skillMutationRemove)))
	mux.Handle("GET /api/v1/instances/{instanceId}/mcp-servers", requireBearer(token, listMCPServers(mcp)))
	mux.Handle("GET /api/v1/instances/{instanceId}/mcp-catalog", requireBearer(token, listMCPPresets(mcp)))
	mux.Handle("POST /api/v1/instances/{instanceId}/mcp-servers/{presetId}/install", requireBearer(token, startMCPMutation(mcp, mcpInstall)))
	mux.Handle("POST /api/v1/instances/{instanceId}/mcp-servers/{serverId}/authenticate", requireBearer(token, startMCPMutation(mcp, mcpAuthenticate)))
	mux.Handle("POST /api/v1/instances/{instanceId}/mcp-servers/{serverId}/test", requireBearer(token, startMCPMutation(mcp, mcpTest)))
	mux.Handle("PATCH /api/v1/instances/{instanceId}/mcp-servers/{serverId}", requireBearer(token, startMCPMutation(mcp, mcpConfigure)))
	mux.Handle("DELETE /api/v1/instances/{instanceId}/mcp-servers/{serverId}", requireBearer(token, startMCPMutation(mcp, mcpRemove)))
	// MCP definitions are Runtime-owned; these preferred routes expose only
	// instance binding mutations. Legacy mcp-servers routes remain compatible.
	mux.Handle("GET /api/v1/instances/{instanceId}/mcp-bindings", requireBearer(token, listMCPServers(mcp)))
	mux.Handle("PUT /api/v1/instances/{instanceId}/mcp-bindings/{presetId}", requireBearer(token, startMCPMutation(mcp, mcpInstall)))
	mux.Handle("PUT /api/v1/instances/{instanceId}/mcp-bindings/{serverId}/credential", requireBearer(token, startMCPMutation(mcp, mcpAuthenticate)))
	mux.Handle("POST /api/v1/instances/{instanceId}/mcp-bindings/{serverId}/test", requireBearer(token, startMCPMutation(mcp, mcpTest)))
	mux.Handle("PATCH /api/v1/instances/{instanceId}/mcp-bindings/{serverId}", requireBearer(token, startMCPMutation(mcp, mcpConfigure)))
	mux.Handle("DELETE /api/v1/instances/{instanceId}/mcp-bindings/{serverId}", requireBearer(token, startMCPMutation(mcp, mcpRemove)))
	mux.Handle("GET /api/v1/operations/{operationId}", requireBearer(token, getOperation(installs)))
	mux.Handle("GET /api/v1/operations/{operationId}/channel-qr", requireBearer(token, getChannelQR(channels)))
	mux.Handle("GET /api/v1/operations/{operationId}/log", requireBearer(token, getOperationLog(installs, dataDir)))
	mux.Handle("GET /api/v1/operations", requireBearer(token, listOperations(installs)))
	mux.Handle("POST /api/v1/operations/{operationId}/cancel", requireBearer(token, cancelOperation(installs, instances, models, sharedModels, channels, managementBackups, mcp)))
	return securityHeaders(restrictOrigins(routeContract(mux)))
}

func routeContract(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed, ok := allowedMethods(r.URL.Path)
		if !ok {
			writeError(w, http.StatusNotFound, ErrorBody{
				Code:      "NOT_FOUND",
				Message:   "The requested local API resource was not found.",
				Retryable: false,
			})
			return
		}
		if r.Method == http.MethodOptions {
			w.Header().Set("Allow", allowed)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if !methodAllowed(r.Method, allowed) {
			w.Header().Set("Allow", allowed)
			writeError(w, http.StatusMethodNotAllowed, ErrorBody{
				Code:      "METHOD_NOT_ALLOWED",
				Message:   "The request method is not allowed for this resource.",
				Retryable: false,
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func allowedMethods(path string) (string, bool) {
	switch path {
	case "/api/v1/health", "/api/v1/node", "/api/v1/node/recovery", "/api/v1/events":
		return "GET, OPTIONS", true
	case "/api/v1/diagnostics/bundle":
		return "POST, OPTIONS", true
	case "/api/v1/operations":
		return "GET, OPTIONS", true
	case "/api/v1/runtimes/hermes/install":
		return "POST, OPTIONS", true
	case "/api/v1/runtimes/hermes/prerequisites":
		return "GET, OPTIONS", true
	case "/api/v1/runtimes/hermes/prerequisites/install":
		return "POST, OPTIONS", true
	case "/api/v1/runtimes/hermes/model-provider-presets":
		return "GET, OPTIONS", true
	case "/api/v1/settings/hermes/download-sources":
		return "GET, PUT, DELETE, OPTIONS", true
	}
	if strings.HasPrefix(path, "/api/v1/backups/") {
		rest := strings.TrimPrefix(path, "/api/v1/backups/")
		if rest != "" && !strings.Contains(rest, "/") {
			return "DELETE, OPTIONS", true
		}
		if strings.HasSuffix(rest, "/restore") {
			id := strings.TrimSuffix(rest, "/restore")
			if id != "" && !strings.Contains(id, "/") {
				return "POST, OPTIONS", true
			}
		}
	}
	if strings.HasPrefix(path, "/api/v1/operations/") {
		rest := strings.TrimPrefix(path, "/api/v1/operations/")
		if rest != "" && !strings.Contains(rest, "/") {
			return "GET, POST, OPTIONS", true
		}
		if strings.HasSuffix(rest, "/cancel") {
			id := strings.TrimSuffix(rest, "/cancel")
			if id != "" && !strings.Contains(id, "/") {
				return "POST, OPTIONS", true
			}
		}
		if strings.HasSuffix(rest, "/log") {
			id := strings.TrimSuffix(rest, "/log")
			if id != "" && !strings.Contains(id, "/") {
				return "GET, OPTIONS", true
			}
		}
		if strings.HasSuffix(rest, "/channel-qr") {
			id := strings.TrimSuffix(rest, "/channel-qr")
			if id != "" && !strings.Contains(id, "/") {
				return "GET, OPTIONS", true
			}
		}
	}
	const prefix = "/api/v1/runtimes/"
	const suffix = "/detect"
	if strings.HasPrefix(path, prefix) && strings.HasSuffix(path, suffix) {
		kind := strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix)
		if kind != "" && !strings.Contains(kind, "/") {
			return "POST, OPTIONS", true
		}
	}
	const upgradePlanSuffix = "/upgrade-plan"
	if strings.HasPrefix(path, prefix) && strings.HasSuffix(path, upgradePlanSuffix) {
		kind := strings.TrimSuffix(strings.TrimPrefix(path, prefix), upgradePlanSuffix)
		if kind != "" && !strings.Contains(kind, "/") {
			return "GET, OPTIONS", true
		}
	}
	for _, mutationSuffix := range []string{"/upgrade", "/rollback"} {
		if strings.HasPrefix(path, prefix) && strings.HasSuffix(path, mutationSuffix) {
			kind := strings.TrimSuffix(strings.TrimPrefix(path, prefix), mutationSuffix)
			if kind != "" && !strings.Contains(kind, "/") {
				return "POST, OPTIONS", true
			}
		}
	}
	if kind := runtimeBackupPathKind(path); kind != "" {
		if kind == "list" {
			return "GET, POST, OPTIONS", true
		}
		return "GET, OPTIONS", true
	}
	if strings.HasPrefix(path, prefix) && strings.HasSuffix(path, "/mcp-definitions") {
		kind := strings.TrimSuffix(strings.TrimPrefix(path, prefix), "/mcp-definitions")
		if kind != "" && !strings.Contains(kind, "/") {
			return "GET, OPTIONS", true
		}
	}
	switch runtimeModelPathKind(path) {
	case "connections", "profiles":
		return "GET, POST, OPTIONS", true
	case "connection", "profile":
		return "DELETE, OPTIONS", true
	case "default":
		return "GET, PUT, DELETE, OPTIONS", true
	case "bindings":
		return "GET, OPTIONS", true
	case "applications":
		return "POST, OPTIONS", true
	}
	switch instancePathKind(path) {
	case "list":
		return "GET, POST, OPTIONS", true
	case "get":
		return "GET, DELETE, OPTIONS", true
	case "record":
		return "DELETE, OPTIONS", true
	case "lifecycle":
		return "POST, OPTIONS", true
	case "lifecycle-status":
		return "GET, OPTIONS", true
	case "config":
		return "GET, PATCH, OPTIONS", true
	case "model-credential":
		return "GET, PUT, DELETE, OPTIONS", true
	case "model-provider-models":
		return "POST, OPTIONS", true
	case "model-validation":
		return "POST, OPTIONS", true
	case "channels":
		return "GET, OPTIONS", true
	case "channel-connect":
		return "POST, OPTIONS", true
	case "channel-binding":
		return "DELETE, OPTIONS", true
	case "channel-pairings":
		return "GET, OPTIONS", true
	case "channel-pairing-approve":
		return "POST, OPTIONS", true
	}
	switch managementReadPathKind(path) {
	case "health", "logs", "skills", "skill-sources", "mcp-servers", "mcp-catalog", "mcp-bindings":
		return "GET, OPTIONS", true
	case "skill":
		return "GET, DELETE, OPTIONS", true
	case "skill-mutation":
		return "POST, OPTIONS", true
	case "mcp-server":
		return "PATCH, DELETE, OPTIONS", true
	case "mcp-mutation":
		return "POST, OPTIONS", true
	case "mcp-binding":
		return "PUT, PATCH, DELETE, OPTIONS", true
	case "mcp-binding-credential":
		return "PUT, OPTIONS", true
	case "mcp-binding-test":
		return "POST, OPTIONS", true
	}
	return "", false
}

func runtimeModelPathKind(path string) string {
	const prefix = "/api/v1/runtimes/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	parts := strings.Split(strings.TrimPrefix(path, prefix), "/")
	if len(parts) < 2 || parts[0] == "" {
		return ""
	}
	switch {
	case len(parts) == 2 && parts[1] == "model-provider-connections":
		return "connections"
	case len(parts) == 3 && parts[1] == "model-provider-connections" && parts[2] != "":
		return "connection"
	case len(parts) == 2 && parts[1] == "model-profiles":
		return "profiles"
	case len(parts) == 3 && parts[1] == "model-profiles" && parts[2] != "":
		return "profile"
	case len(parts) == 2 && parts[1] == "model-default":
		return "default"
	case len(parts) == 2 && parts[1] == "model-bindings":
		return "bindings"
	case len(parts) == 2 && parts[1] == "model-profile-applications":
		return "applications"
	default:
		return ""
	}
}

func runtimeBackupPathKind(path string) string {
	const prefix = "/api/v1/runtimes/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	parts := strings.Split(strings.TrimPrefix(path, prefix), "/")
	if len(parts) == 2 && parts[0] != "" && parts[1] == "backups" {
		return "list"
	}
	if len(parts) == 3 && parts[0] != "" && parts[1] == "backups" && parts[2] != "" {
		return "get"
	}
	return ""
}

func managementReadPathKind(path string) string {
	const prefix = "/api/v1/instances/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	parts := strings.Split(strings.TrimPrefix(path, prefix), "/")
	if len(parts) < 2 || parts[0] == "" {
		return ""
	}
	switch {
	case len(parts) == 2 && parts[1] == "health":
		return "health"
	case len(parts) == 2 && parts[1] == "logs":
		return "logs"
	case len(parts) == 2 && parts[1] == "skills":
		return "skills"
	case len(parts) == 2 && parts[1] == "skill-sources":
		return "skill-sources"
	case len(parts) == 3 && parts[1] == "skills" && parts[2] != "":
		return "skill"
	case len(parts) == 4 && parts[1] == "skills" && parts[2] != "" &&
		(parts[3] == "install" || parts[3] == "update" || parts[3] == "enable" || parts[3] == "disable"):
		return "skill-mutation"
	case len(parts) == 2 && parts[1] == "mcp-servers":
		return "mcp-servers"
	case len(parts) == 2 && parts[1] == "mcp-catalog":
		return "mcp-catalog"
	case len(parts) == 2 && parts[1] == "mcp-bindings":
		return "mcp-bindings"
	case len(parts) == 3 && parts[1] == "mcp-bindings" && parts[2] != "":
		return "mcp-binding"
	case len(parts) == 4 && parts[1] == "mcp-bindings" && parts[2] != "" && parts[3] == "credential":
		return "mcp-binding-credential"
	case len(parts) == 4 && parts[1] == "mcp-bindings" && parts[2] != "" && parts[3] == "test":
		return "mcp-binding-test"
	case len(parts) == 3 && parts[1] == "mcp-servers" && parts[2] != "":
		return "mcp-server"
	case len(parts) == 4 && parts[1] == "mcp-servers" && parts[2] != "" &&
		(parts[3] == "install" || parts[3] == "authenticate" || parts[3] == "test"):
		return "mcp-mutation"
	default:
		return ""
	}
}

func methodAllowed(method, allowed string) bool {
	for _, candidate := range strings.Split(allowed, ", ") {
		if method == candidate {
			return true
		}
	}
	return false
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(HealthResponse{
		Status:          "ok",
		Service:         buildinfo.Service,
		Version:         buildinfo.Version,
		ProtocolVersion: buildinfo.ProtocolVersion,
	})
}

func detectRuntime(runtimes RuntimeDiscoveryService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		discovery, err := runtimes.Detect(r.Context(), yorvaruntime.Kind(r.PathValue("runtimeKind")))
		if err != nil {
			if errors.Is(err, context.Canceled) && r.Context().Err() != nil {
				return
			}
			if errors.Is(err, app.ErrRuntimeKindNotFound) {
				writeError(w, http.StatusNotFound, ErrorBody{
					Code:      "RUNTIME_KIND_NOT_FOUND",
					Message:   "The requested Runtime kind is not registered.",
					Retryable: false,
				})
				return
			}
			writeError(w, http.StatusInternalServerError, ErrorBody{
				Code:      "RUNTIME_DISCOVERY_FAILED",
				Message:   "Runtime discovery could not be completed.",
				Retryable: true,
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(newRuntimeDiscoveryResponse(discovery))
	})
}

func newRuntimeDiscoveryResponse(discovery yorvaruntime.Discovery) RuntimeDiscoveryResponse {
	candidates := make([]RuntimeCandidateResponse, len(discovery.Candidates))
	for i, candidate := range discovery.Candidates {
		candidates[i] = newRuntimeCandidateResponse(candidate)
	}
	var selected *RuntimeCandidateResponse
	if discovery.Selected != nil {
		value := newRuntimeCandidateResponse(*discovery.Selected)
		selected = &value
	}
	warnings := make([]RuntimeWarningResponse, len(discovery.Warnings))
	for i, warning := range discovery.Warnings {
		warnings[i] = RuntimeWarningResponse{Code: warning.Code, Message: warning.Message}
	}
	return RuntimeDiscoveryResponse{
		RuntimeKind:    discovery.RuntimeKind,
		State:          discovery.State,
		ErrorCode:      nullableErrorCode(discovery.ErrorCode),
		Selected:       selected,
		Candidates:     candidates,
		Warnings:       warnings,
		DetectedAt:     discovery.DetectedAt,
		SupportedRange: discovery.SupportedRange,
	}
}

func newRuntimeCandidateResponse(candidate yorvaruntime.Candidate) RuntimeCandidateResponse {
	return RuntimeCandidateResponse{
		Path:      candidate.Path,
		Version:   candidate.Version,
		State:     candidate.State,
		ErrorCode: nullableErrorCode(candidate.ErrorCode),
	}
}

func nullableErrorCode(code yorvaruntime.ErrorCode) *yorvaruntime.ErrorCode {
	if code == "" {
		return nil
	}
	return &code
}

func eventStream(broker *events.Broker, keepaliveInterval time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			writeError(w, http.StatusInternalServerError, ErrorBody{
				Code:      "INTERNAL_ERROR",
				Message:   "Event streaming is unavailable.",
				Retryable: true,
			})
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		_, _ = fmt.Fprint(w, ": connected\n\n")
		flusher.Flush()

		subscription := broker.Subscribe()
		defer subscription.Close()
		keepalive := time.NewTicker(keepaliveInterval)
		defer keepalive.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case event := <-subscription.Events:
				payload, err := json.Marshal(event)
				if err != nil {
					continue
				}
				if _, err := fmt.Fprintf(w, "id: %s\nevent: %s\ndata: %s\n\n", event.ID, event.Type, payload); err != nil {
					return
				}
				flusher.Flush()
			case <-keepalive.C:
				if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
					return
				}
				flusher.Flush()
			}
		}
	})
}

func requireBearer(expected string, next http.Handler) http.Handler {
	expectedHash := sha256.Sum256([]byte(expected))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scheme, supplied, ok := strings.Cut(r.Header.Get("Authorization"), " ")
		suppliedHash := sha256.Sum256([]byte(supplied))
		if !ok || !strings.EqualFold(scheme, "Bearer") || subtle.ConstantTimeCompare(expectedHash[:], suppliedHash[:]) != 1 {
			writeError(w, http.StatusUnauthorized, ErrorBody{
				Code:      "UNAUTHORIZED",
				Message:   "Authentication is required.",
				Retryable: false,
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

func restrictOrigins(next http.Handler) http.Handler {
	allowed := map[string]struct{}{
		"http://127.0.0.1:1420":  {},
		"http://tauri.localhost": {},
		"tauri://localhost":      {},
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if _, ok := allowed[origin]; !ok {
				writeError(w, http.StatusForbidden, ErrorBody{
					Code:      "ORIGIN_NOT_ALLOWED",
					Message:   "The request origin is not allowed.",
					Retryable: false,
				})
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Accept, Content-Type, Idempotency-Key, Yorva-Session-Id")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		}
		next.ServeHTTP(w, r)
	})
}

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { DaemonClient } from "../../api/client";
import type { Instance } from "../../api/types";
import { messages } from "../../i18n";
import { ManagementPanel } from "./ManagementPanel";

vi.mock("../../api/session", () => ({
  selectSkillImport: vi.fn().mockResolvedValue({ sourceRef: "s".repeat(43), suggestedSkillId: "local-skill" }),
  discardSkillImport: vi.fn().mockResolvedValue(undefined),
}));

const instance: Instance = {
  instanceId: "inst-coder",
  runtimeInstallationId: "rtinst-test",
  name: "coder",
  default: false,
  protected: false,
  availability: "AVAILABLE",
  lastSyncedAt: "2026-08-25T10:00:00Z",
  createdAt: "2026-08-25T10:00:00Z",
  updatedAt: "2026-08-25T10:00:00Z",
  capabilities: {
    instances: true, lifecycle: false, healthRead: true, logsRead: true, securityAudit: false,
    skillRead: true, skillMutate: false, mcpRead: true, mcpMutate: false, mcpTest: false,
    nativeSkills: {
      inventory: { supported: true, reason: "dynamic_instance_readback" },
      nativeInstall: { supported: false, reason: "deferred_upstream" },
      nativeUpdate: { supported: false, reason: "deferred_upstream" },
      nativeRemove: { supported: false, reason: "deferred_upstream" },
      nativeEnableDisable: { supported: false, reason: "deferred_upstream" },
      nativeProfileBinding: { supported: false, reason: "deferred_upstream" },
    },
    backupRead: false, backupMutate: false, restore: false, upgradePlan: false, upgrade: false, rollback: false,
  },
};

function managementClient(overrides: Partial<DaemonClient> = {}) {
  return {
    scope: "http://127.0.0.1:49152",
    getInstanceHealth: vi.fn().mockResolvedValue({
      state: "DEGRADED", findings: [{ code: "gateway", state: "DEGRADED" }], partial: false, observedAt: "2026-08-25T10:00:00Z",
    }),
    getInstanceLogSnapshot: vi.fn().mockResolvedValue({
      category: "ERRORS", entries: [{ timestamp: "2026-08-25T10:00:00Z", message: "bounded redacted entry" }], truncated: false, observedAt: "2026-08-25T10:00:00Z",
    }),
    listInstanceSkills: vi.fn().mockResolvedValue({ items: [{
      id: "writer", sourceId: "official", version: "1.2.3", description: "Draft and refine documents.", ownership: "EXTERNAL", projectionState: "PROJECTED", installationState: "INSTALLED",
      enabledState: "ENABLED", scanState: "CLEAN", updateAvailable: false,
    }] }),
    inspectInstanceSkill: vi.fn().mockResolvedValue({
      id: "writer", sourceId: "official", version: "1.2.3", description: "Draft and refine documents.", preview: "# Writer\n\nUse this Skill for document work.", ownership: "EXTERNAL", projectionState: "PROJECTED", installationState: "INSTALLED",
      enabledState: "ENABLED", scanState: "CLEAN", updateAvailable: false,
    }),
    listInstanceMCPServers: vi.fn().mockResolvedValue({ items: [{
      id: "docs", presetId: "approved-docs", ownership: "EXTERNAL", enabledToolIds: [], state: "CONFIGURED", readyAt: null, observedAt: "2026-08-25T10:00:00Z",
    }] }),
    listRuntimeMCPDefinitions: vi.fn().mockResolvedValue({ items: [{
      id: "approved-docs", displayName: "Approved Docs", description: "Reviewed documentation service.",
      homepageUrl: "https://example.invalid/", documentationUrl: "https://example.invalid/docs",
      allowedToolIds: [], credentialRequired: false,
    }] }),
    getRuntimeUpgradePlan: vi.fn().mockResolvedValue({
      state: "UNKNOWN", currentVersion: "0.20.2",
      candidate: { label: "Hermes 0.20.5 packaged snapshot", version: "0.20.5" },
      managedStatus: "MANAGED", compatibility: "UNKNOWN",
      protectionPointRequired: true, protectionPointReady: false,
      blockedReasons: ["COMPATIBILITY_UNKNOWN", "PROTECTION_POINT_REQUIRED"],
      observedAt: "2026-08-25T10:00:00Z",
    }),
    listRuntimeBackups: vi.fn().mockResolvedValue({ scope: "RUNTIME", items: [] }),
    listModelProviderPresets: vi.fn().mockResolvedValue({ items: [{
      id: "qwen", displayName: "Qwen", region: "CHINA", recommendedModels: ["qwen-plus"], helpText: "Reviewed Qwen configuration.",
    }] }),
    getModelConfiguration: vi.fn().mockResolvedValue({
      providerPresetId: "qwen", modelId: "qwen-plus", selectedModelIds: ["qwen-plus"], state: "CONFIGURED",
      credentialConfigured: true, observedAt: "2026-08-25T10:00:00Z", validation: { state: "PASSED", errorCode: null, completedAt: "2026-08-25T10:00:00Z" },
    }),
    getModelCredential: vi.fn().mockResolvedValue({ providerPresetId: "qwen", configured: true, observedAt: "2026-08-25T10:00:00Z" }),
    listOperations: vi.fn().mockResolvedValue({ operations: [] }),
    listRuntimeModelProviderConnections: vi.fn().mockResolvedValue({ items: [] }),
    createRuntimeModelProviderConnection: vi.fn(),
    deleteRuntimeModelProviderConnection: vi.fn(),
    listRuntimeModelProfiles: vi.fn().mockResolvedValue({ items: [] }),
    createRuntimeModelProfile: vi.fn(),
    deleteRuntimeModelProfile: vi.fn(),
    getRuntimeModelDefault: vi.fn().mockResolvedValue({ modelProfileId: "", appliedRevision: 0, updatedAt: null }),
    setRuntimeModelDefault: vi.fn(),
    clearRuntimeModelDefault: vi.fn(),
    listRuntimeModelBindings: vi.fn().mockResolvedValue({ items: [] }),
    applyRuntimeModelProfile: vi.fn(),
    listInstanceSkillSources: vi.fn().mockResolvedValue({ items: [] }),
    installManagedSkill: vi.fn(),
    importManagedSkill: vi.fn(),
    updateManagedSkill: vi.fn(),
    enableManagedSkill: vi.fn(),
    disableManagedSkill: vi.fn(),
    removeManagedSkill: vi.fn(),
    getOperation: vi.fn().mockResolvedValue({ id: "op-mcp", status: "SUCCEEDED", stage: "mcp.reconcile", type: "mcp.install", targetType: "instance", targetId: "inst-coder", message: "", errorCode: "", errorMessage: "", retryable: false, idempotencyKey: "test", correlationId: "test", createdAt: "2026-08-25T10:00:00Z", startedAt: "2026-08-25T10:00:00Z", completedAt: "2026-08-25T10:00:00Z", updatedAt: "2026-08-25T10:00:00Z" }),
    getRuntimeBackup: vi.fn(),
    ...overrides,
  } as unknown as DaemonClient;
}

function renderPanel(client: DaemonClient, target: Instance = instance, scope: "instance" | "runtime" = "instance") {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <ManagementPanel client={client} instance={target} instances={[target]} scope={scope} copy={messages["en-US"]} locale="en-US" onClose={() => undefined} />
    </QueryClientProvider>,
  );
}

describe("ManagementPanel", () => {
	it("shows real Skill source actions and imports a native-selected directory", async () => {
		const importManagedSkill = vi.fn().mockResolvedValue({ id: "op-import", status: "PENDING" });
		const client = managementClient({ importManagedSkill });
		renderPanel(client, { ...instance, capabilities: { ...instance.capabilities, skillMutate: true } }, "runtime");

		fireEvent.click(screen.getByRole("button", { name: "Skills" }));
		expect(await screen.findByRole("button", { name: "Check for updates" })).toBeInTheDocument();
		expect(screen.getByRole("button", { name: "Install from ZIP" })).toBeInTheDocument();
		fireEvent.click(screen.getByRole("button", { name: "Import existing" }));
		const dialog = await screen.findByRole("dialog", { name: "Install local Skill" });
		fireEvent.click(within(dialog).getByRole("button", { name: "Install" }));
		await waitFor(() => expect(importManagedSkill).toHaveBeenCalledWith("inst-coder", "local-skill", "s".repeat(43), expect.any(String)));
		await waitFor(() => expect(client.getOperation).toHaveBeenCalledWith("op-import", expect.any(AbortSignal)));
	});

  it("shows safe Skill and MCP reads while keeping CONFIGURED distinct from READY", async () => {
    const client = managementClient();
    renderPanel(client);

    expect(await screen.findByText("writer")).toBeInTheDocument();
    expect(await screen.findByText("docs")).toBeInTheDocument();
    expect((await screen.findAllByText("Degraded")).length).toBeGreaterThanOrEqual(2);
    expect(screen.getByText("bounded redacted entry")).toBeInTheDocument();
    await waitFor(() => expect(client.getInstanceLogSnapshot).toHaveBeenCalledWith("inst-coder", "ERRORS", expect.any(AbortSignal)));
    fireEvent.change(screen.getByRole("combobox", { name: "Category" }), { target: { value: "MCP" } });
    await waitFor(() => expect(client.getInstanceLogSnapshot).toHaveBeenCalledWith("inst-coder", "MCP", expect.any(AbortSignal)));
    expect(screen.getByText("Configured")).toBeInTheDocument();
    expect(screen.getByText("No current Ready evidence")).toBeInTheDocument();
    expect(screen.queryByText("Ready")).not.toBeInTheDocument();
    expect(screen.getByText("approved-docs")).toBeInTheDocument();
    expect(screen.getAllByText("Runtime or externally owned; read-only in YORVA.").length).toBeGreaterThanOrEqual(1);
    expect(screen.queryByRole("button", { name: "Remove" })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Inspect" }));
    await waitFor(() => expect(client.inspectInstanceSkill).toHaveBeenCalledWith("inst-coder", "writer", expect.any(AbortSignal)));
    expect(await screen.findByText("official")).toBeInTheDocument();
    expect(screen.getByText("1.2.3")).toBeInTheDocument();

    const visible = document.body.textContent ?? "";
    for (const prohibited of ["Authorization", "Bearer", "command", "header", "C:\\", "https://"]) {
      expect(visible).not.toContain(prohibited);
    }
  });

  it("installs from the approved catalog and exposes actions only for YORVA-managed Skills", async () => {
    const installManagedSkill = vi.fn().mockResolvedValue({ id: "op-install", status: "PENDING" });
    const disableManagedSkill = vi.fn().mockResolvedValue({ id: "op-disable", status: "PENDING" });
    const client = managementClient({
      listInstanceSkillSources: vi.fn().mockResolvedValue({ items: [{ sourceId: "yorva-extra", skillId: "yorva-extra-demo", displayName: "YORVA extra", version: "1.0.0" }] }),
      listInstanceSkills: vi.fn().mockResolvedValue({ items: [{
        id: "yorva-managed-demo", sourceId: "yorva-demo", version: "1.0.0", ownership: "YORVA_MANAGED", projectionState: "PROJECTED",
        installationState: "INSTALLED", enabledState: "ENABLED", scanState: "CLEAN", updateAvailable: false,
      }] }),
      inspectInstanceSkill: vi.fn().mockResolvedValue({
        id: "yorva-managed-demo", sourceId: "yorva-demo", version: "1.0.0", ownership: "YORVA_MANAGED", projectionState: "PROJECTED",
        installationState: "INSTALLED", enabledState: "ENABLED", scanState: "CLEAN", updateAvailable: false,
      }),
      installManagedSkill,
      disableManagedSkill,
    });
    renderPanel(client, { ...instance, capabilities: { ...instance.capabilities, skillMutate: true } }, "runtime");

    fireEvent.click(screen.getByRole("button", { name: "Skills" }));
    expect(await screen.findByText("YORVA extra (1.0.0)")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Install" }));
    await waitFor(() => expect(installManagedSkill).toHaveBeenCalledWith("inst-coder", "yorva-extra-demo", "yorva-extra", expect.any(String)));
    fireEvent.click(screen.getByRole("switch", { name: "yorva-managed-demo: Disable" }));
    await waitFor(() => expect(disableManagedSkill).toHaveBeenCalledWith("inst-coder", "yorva-managed-demo", expect.any(String)));
  });

  it("opens a Runtime Skill in a separate preview page and returns to the catalog", async () => {
    const client = managementClient();
    renderPanel(client, instance, "runtime");

    fireEvent.click(screen.getByRole("button", { name: "Skills" }));
    expect(await screen.findByText("YORVA-managed Skills")).toBeInTheDocument();
    const skillsSection = screen.getByRole("region", { name: "Skills" });
    expect(within(skillsSection).queryByText("Available")).not.toBeInTheDocument();
    fireEvent.click(await screen.findByText("Hermes and external Skills"));
    expect(await screen.findByText("Draft and refine documents.")).toBeInTheDocument();
    expect(screen.getByRole("switch", { name: "writer: Disable" })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: /writer: Draft and refine documents/ }));

    await waitFor(() => expect(client.inspectInstanceSkill).toHaveBeenCalledWith("inst-coder", "writer", expect.any(AbortSignal)));
    expect(await screen.findByRole("button", { name: "← Back to Skills" })).toBeInTheDocument();
    expect(screen.getByText((_, element) => element?.tagName === "PRE" && element.textContent === "# Writer\n\nUse this Skill for document work.")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "← Back to Skills" }));
    expect(await screen.findByText("Draft and refine documents.")).toBeInTheDocument();
  });

  it("offers a bounded reproject action for a missing YORVA-managed Skill projection", async () => {
    const enableManagedSkill = vi.fn().mockResolvedValue({ id: "op-reproject", status: "PENDING" });
    const client = managementClient({
      enableManagedSkill,
      listInstanceSkills: vi.fn().mockResolvedValue({ items: [{
        id: "yorva-document-review", sourceId: "yorva-reviewed", version: "1.0.0", ownership: "YORVA_MANAGED", projectionState: "DRIFT_MISSING",
        installationState: "INSTALLED", enabledState: "UNKNOWN", scanState: "CLEAN", updateAvailable: false,
      }] }),
    });
    renderPanel(client, { ...instance, capabilities: { ...instance.capabilities, skillMutate: true } }, "runtime");

    fireEvent.click(screen.getByRole("button", { name: "Skills" }));
    expect(await screen.findByText("Missing drift")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Reproject" }));
    await waitFor(() => expect(enableManagedSkill).toHaveBeenCalledWith("inst-coder", "yorva-document-review", expect.any(String)));
  });

  it("separates Runtime MCP definitions from instance bindings and explains read-only capability", async () => {
    renderPanel(managementClient(), instance, "runtime");

    fireEvent.click(screen.getByRole("button", { name: "MCP" }));
    expect(await screen.findByText("Runtime MCP definitions")).toBeInTheDocument();
    expect(screen.getByText("MCP bindings")).toBeInTheDocument();
    expect(screen.getByText("This Runtime can read MCP configuration, but YORVA mutation is not supported yet.")).toBeInTheDocument();
    expect(screen.queryByText("No MCP servers were reported for this instance.")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add MCP" })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: "Import existing" }));
    expect(await screen.findByText("Synchronized 1 existing MCP bindings from the selected Hermes instance. External definitions remain read-only.")).toBeInTheDocument();
  });

  it("creates a binding from a reviewed MCP preset without custom execution fields", async () => {
    const installInstanceMCPPreset = vi.fn().mockResolvedValue({ id: "op-mcp", status: "PENDING" });
    const client = managementClient({
      installInstanceMCPPreset,
      listRuntimeMCPDefinitions: vi.fn().mockResolvedValue({ items: [{
        id: "yorva-mcp-test", displayName: "YORVA MCP Test", description: "Qualification preset.",
        homepageUrl: "https://example.invalid/", documentationUrl: "https://example.invalid/docs",
        allowedToolIds: ["yorva_ping"], credentialRequired: false,
      }] }),
      listInstanceMCPServers: vi.fn().mockResolvedValue({ items: [] }),
    });
    renderPanel(client, { ...instance, capabilities: { ...instance.capabilities, mcpMutate: true, mcpTest: true } }, "runtime");

    fireEvent.click(screen.getByRole("button", { name: "MCP" }));
    fireEvent.click(await screen.findByRole("button", { name: "Add MCP" }));
    expect(screen.getByLabelText("Unique identifier")).toHaveValue("yorva-mcp-test");
    expect(screen.queryByLabelText("Endpoint URL")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("HTTP headers")).not.toBeInTheDocument();
    expect(screen.queryByText("stdio")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Test and add" }));

    await waitFor(() => expect(installInstanceMCPPreset).toHaveBeenCalledWith("inst-coder", "yorva-mcp-test", "", ["yorva_ping"], expect.any(String)));
  });

  it("switches the configured Runtime instance before reading and installing Skills", async () => {
    const second: Instance = { ...instance, instanceId: "inst-review", name: "review", capabilities: { ...instance.capabilities, skillMutate: true } };
    const installManagedSkill = vi.fn().mockResolvedValue({ id: "op-install", status: "PENDING" });
    const client = managementClient({
      listInstanceSkills: vi.fn().mockImplementation((instanceId: string) => Promise.resolve({ items: instanceId === second.instanceId ? [{
        id: "review-skill", sourceId: "reviewed", version: "1.0.0", ownership: "YORVA_MANAGED", projectionState: "PROJECTED",
        installationState: "INSTALLED", enabledState: "ENABLED", scanState: "CLEAN", updateAvailable: false,
      }] : [] })),
      listInstanceSkillSources: vi.fn().mockResolvedValue({ items: [{ sourceId: "yorva-extra", skillId: "yorva-extra-demo", displayName: "YORVA extra", version: "1.0.0" }] }),
      installManagedSkill,
    });
    const target = { ...instance, capabilities: { ...instance.capabilities, skillMutate: true } };
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
    render(<QueryClientProvider client={queryClient}><ManagementPanel client={client} instance={target} instances={[target, second]} scope="runtime" copy={messages["en-US"]} locale="en-US" /></QueryClientProvider>);

    fireEvent.click(screen.getByRole("button", { name: "Skills" }));
    fireEvent.change(screen.getByRole("combobox", { name: "Configuring" }), { target: { value: "inst-review" } });
    expect(await screen.findByText("review-skill")).toBeInTheDocument();
    expect(screen.getByText("Applied instances: review")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Install" }));
    await waitFor(() => expect(installManagedSkill).toHaveBeenCalledTimes(1));
    expect(installManagedSkill).toHaveBeenCalledWith("inst-review", "yorva-extra-demo", "yorva-extra", expect.any(String));
  });

  it("opens exact-instance diagnostics from the instance list and keeps Diagnostics out of the Runtime navigation", async () => {
    const second: Instance = { ...instance, instanceId: "inst-review", name: "review" };
    const client = managementClient();
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
    render(<QueryClientProvider client={queryClient}><ManagementPanel client={client} instance={instance} instances={[instance, second]} scope="runtime" copy={messages["en-US"]} locale="en-US" /></QueryClientProvider>);

    const navigation = screen.getByRole("navigation", { name: "Runtime management sections" });
    expect(within(navigation).queryByRole("button", { name: "Diagnostics" })).not.toBeInTheDocument();
    fireEvent.click(within(navigation).getByRole("button", { name: "Instances" }));
    const reviewRow = screen.getByText("review").closest("article");
    expect(reviewRow).not.toBeNull();
    fireEvent.click(within(reviewRow!).getByRole("button", { name: "Diagnostics & logs" }));

    expect(await screen.findByText("Switch instance")).toBeInTheDocument();
    expect(screen.getByText("Select an instance to view its current health and runtime logs.")).toBeInTheDocument();
    await waitFor(() => expect(client.getInstanceHealth).toHaveBeenCalledWith("inst-review", expect.any(AbortSignal)));
    await waitFor(() => expect(client.getInstanceLogSnapshot).toHaveBeenCalledWith("inst-review", "ERRORS", expect.any(AbortSignal)));
    fireEvent.click(screen.getByRole("button", { name: "coder", pressed: false }));
    await waitFor(() => expect(client.getInstanceHealth).toHaveBeenCalledWith("inst-coder", expect.any(AbortSignal)));
  });

  it("exposes shared Provider, Profile, default and binding resources in the Runtime workspace", async () => {
    const second: Instance = { ...instance, instanceId: "inst-review", name: "review" };
    const client = managementClient({
      listRuntimeModelProviderConnections: vi.fn().mockResolvedValue({ items: [{ id: "mpc-test", providerPresetId: "qwen", displayName: "Shared Qwen", credentialConfigured: true, status: "CONFIGURED", revision: 1, createdAt: "2026-08-25T10:00:00Z", updatedAt: "2026-08-25T10:00:00Z" }] }),
      listRuntimeModelProfiles: vi.fn().mockResolvedValue({ items: [{ id: "mpr-test", providerConnectionId: "mpc-test", displayName: "Qwen production", selectedModelIds: ["qwen-plus"], defaultModelId: "qwen-plus", revision: 1, createdAt: "2026-08-25T10:00:00Z", updatedAt: "2026-08-25T10:00:00Z" }] }),
      getRuntimeModelDefault: vi.fn().mockResolvedValue({ modelProfileId: "mpr-test", appliedRevision: 1, updatedAt: "2026-08-25T10:00:00Z" }),
      listRuntimeModelBindings: vi.fn().mockResolvedValue({ items: [{ instanceId: "inst-review", instanceName: "review", modelProfileId: "mpr-test", mode: "INHERIT", appliedRevision: 1, state: "SUCCEEDED", errorCode: null, updatedAt: "2026-08-25T10:00:00Z" }] }),
    });
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
    render(<QueryClientProvider client={queryClient}><ManagementPanel client={client} instance={instance} instances={[instance, second]} scope="runtime" copy={messages["en-US"]} locale="en-US" /></QueryClientProvider>);

    const navigation = screen.getByRole("navigation", { name: "Runtime management sections" });
    fireEvent.click(within(navigation).getByRole("button", { name: "Models" }));
    expect(await screen.findByText("Shared models")).toBeInTheDocument();
    expect((await screen.findAllByText("Shared Qwen")).length).toBeGreaterThan(0);
    expect((await screen.findAllByText("Qwen production")).length).toBeGreaterThan(0);
    expect(screen.getAllByText("Runtime default").length).toBeGreaterThan(0);
    expect(screen.getAllByText("review").length).toBeGreaterThan(0);
    expect(screen.getByText("Succeeded")).toBeInTheDocument();
    expect(client.listRuntimeModelBindings).toHaveBeenCalledWith("hermes", expect.any(AbortSignal));
  });

  it("keeps model workflows off the resource page and opens dedicated configuration pages", async () => {
    const second: Instance = { ...instance, instanceId: "inst-review", name: "review" };
    const client = managementClient({
      listRuntimeModelProviderConnections: vi.fn().mockResolvedValue({ items: [{ id: "mpc-test", providerPresetId: "qwen", displayName: "Shared Qwen", credentialConfigured: true, status: "CONFIGURED", revision: 1, createdAt: "2026-08-25T10:00:00Z", updatedAt: "2026-08-25T10:00:00Z" }] }),
      listRuntimeModelProfiles: vi.fn().mockResolvedValue({ items: [{ id: "mpr-test", providerConnectionId: "mpc-test", displayName: "Qwen production", selectedModelIds: ["qwen-plus"], defaultModelId: "qwen-plus", revision: 1, createdAt: "2026-08-25T10:00:00Z", updatedAt: "2026-08-25T10:00:00Z" }] }),
      getRuntimeModelDefault: vi.fn().mockResolvedValue({ modelProfileId: "mpr-test", appliedRevision: 1, updatedAt: "2026-08-25T10:00:00Z" }),
    });
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
    render(<QueryClientProvider client={queryClient}><ManagementPanel client={client} instance={instance} instances={[instance, second]} scope="runtime" copy={messages["en-US"]} locale="en-US" /></QueryClientProvider>);

    fireEvent.click(within(screen.getByRole("navigation", { name: "Runtime management sections" })).getByRole("button", { name: "Models" }));
    expect(await screen.findByText("Shared models")).toBeInTheDocument();
    expect(screen.queryByLabelText("API Key")).not.toBeInTheDocument();

    const addConnection = screen.getByRole("button", { name: "Add connection" });
    await waitFor(() => expect(addConnection).toBeEnabled());
    fireEvent.click(addConnection);
    expect(screen.getByRole("heading", { name: "Add connection" })).toBeInTheDocument();
    expect(screen.getByLabelText("API Key")).toBeInTheDocument();
    expect(screen.queryByText("Model Profiles")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));

    fireEvent.click(screen.getByRole("button", { name: "Add Profile" }));
    expect(screen.getByRole("heading", { name: "Add Profile" })).toBeInTheDocument();
    expect(screen.getByRole("group", { name: "Models" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));

    fireEvent.click(screen.getByRole("button", { name: "Configure bindings" }));
    expect(screen.getByRole("heading", { name: "Configure bindings" })).toBeInTheDocument();
    expect(screen.getByRole("group", { name: "Managed instances" })).toBeInTheDocument();
    expect(screen.getByRole("checkbox", { name: "review" })).toBeChecked();
    fireEvent.click(screen.getByRole("button", { name: /Back to shared models/ }));
    expect(await screen.findByText("Shared models")).toBeInTheDocument();
  });

  it("does not issue reads or present fake actions when capabilities are false", () => {
    const client = managementClient();
    const unavailable: Instance = {
      ...instance,
      capabilities: { ...instance.capabilities, healthRead: false, logsRead: false, skillRead: false, mcpRead: false },
    };
    renderPanel(client, unavailable);

    expect(screen.getAllByText("This capability is unavailable for the selected Runtime version.")).toHaveLength(3);
    expect(client.getInstanceHealth).not.toHaveBeenCalled();
    expect(client.getInstanceLogSnapshot).not.toHaveBeenCalled();
    expect(client.listInstanceSkills).not.toHaveBeenCalled();
    expect(client.listInstanceMCPServers).not.toHaveBeenCalled();
    expect(client.getRuntimeUpgradePlan).not.toHaveBeenCalled();
    expect(client.listRuntimeBackups).not.toHaveBeenCalled();
    expect(screen.queryByRole("button", { name: /install|remove|test/i })).not.toBeInTheDocument();
  });

  it("shows only safe last-observed Runtime backup metadata", async () => {
    const client = managementClient({
      listRuntimeBackups: vi.fn().mockResolvedValue({ scope: "RUNTIME", items: [{
        backupId: "backup-safe", scope: "RUNTIME", state: "CHANGED", formatVersion: "yorva.hermes.runtime-backup.v1",
        runtimeVersion: "0.20.5", sizeBytes: 4096, checksumSha256: "a".repeat(64),
        createdAt: "2026-08-25T10:00:00Z", verifiedAt: "2026-08-25T10:01:00Z", keyMode: "DEVICE",
      }] }),
    });
    renderPanel(client, { ...instance, capabilities: { ...instance.capabilities, backupRead: true } }, "runtime");

    fireEvent.click(screen.getByRole("button", { name: "Maintenance" }));
    expect(await screen.findByText("backup-safe")).toBeInTheDocument();
    expect(screen.getAllByText("Changed").length).toBeGreaterThan(0);
    expect(screen.getByText("Device-managed key")).toBeInTheDocument();
    expect(client.listRuntimeBackups).toHaveBeenCalledWith("hermes", expect.any(AbortSignal));
    expect(screen.getByText("Stop every Hermes instance and close Hermes Dashboard or other Hermes background processes before creating or restoring a Runtime backup.")).toBeInTheDocument();
    const visible = document.body.textContent ?? "";
    for (const prohibited of ["artifactPath", "keyRef", "passphrase", "C:\\Backups"] ) expect(visible).not.toContain(prohibited);
  });

  it("explains the stopped-Runtime precondition when backup creation is rejected", async () => {
    const createRuntimeBackup = vi.fn().mockResolvedValue({ id: "op-backup", status: "PENDING" });
    const client = managementClient({
      createRuntimeBackup,
      getOperation: vi.fn().mockResolvedValue({
        id: "op-backup", status: "FAILED", stage: "backup.reconcile", type: "backup.create",
        targetType: "runtime-installation", targetId: "rtinst-test", message: "",
        errorCode: "BACKUP_SOURCE_RUNTIME_NOT_STOPPED", errorMessage: "", retryable: true,
        idempotencyKey: "test", correlationId: "test", createdAt: "2026-08-25T10:00:00Z",
        startedAt: "2026-08-25T10:00:00Z", completedAt: "2026-08-25T10:00:00Z", updatedAt: "2026-08-25T10:00:00Z",
      }),
    });
    renderPanel(client, { ...instance, capabilities: { ...instance.capabilities, backupRead: true, backupMutate: true } }, "runtime");

    fireEvent.click(screen.getByRole("button", { name: "Maintenance" }));
    fireEvent.click(await screen.findByRole("button", { name: "Create encrypted backup" }));
    await waitFor(() => expect(createRuntimeBackup).toHaveBeenCalledWith("hermes", expect.any(String)));
    expect(await screen.findByText("Backup was not created because Hermes data is still in use. Stop all instances and close Hermes Dashboard or other Hermes processes, then retry.")).toBeInTheDocument();
  });

  it("shows an evidence-incomplete read-only Upgrade plan without mutation actions or internal identity", async () => {
    const client = managementClient();
    renderPanel(client, { ...instance, capabilities: { ...instance.capabilities, upgradePlan: true } }, "runtime");

    fireEvent.click(screen.getByRole("button", { name: "Maintenance" }));
    expect(await screen.findByText("Hermes 0.20.5 packaged snapshot (0.20.5)")).toBeInTheDocument();
    expect(screen.getByText("Evidence still required")).toBeInTheDocument();
    expect(screen.getByText("Compatibility evidence: this exact current-to-candidate pair needs qualified Windows upgrade and rollback results.")).toBeInTheDocument();
    expect(screen.getByText("Required; no verified protection point")).toBeInTheDocument();
    expect(client.getRuntimeUpgradePlan).toHaveBeenCalledWith("hermes", expect.any(AbortSignal));
    expect(screen.queryByRole("button", { name: /upgrade|rollback/i })).not.toBeInTheDocument();
    const visible = document.body.textContent ?? "";
    for (const prohibited of ["a0ca7c1", "df4b651", "sha256", "C:\\", "https://", "executable"]) {
      expect(visible.toLowerCase()).not.toContain(prohibited.toLowerCase());
    }
  });

  it("shows a matching external Runtime version as up to date without mutation actions", async () => {
    const client = managementClient({
      getRuntimeUpgradePlan: vi.fn().mockResolvedValue({
        state: "UP_TO_DATE", currentVersion: "0.20.5",
        candidate: { label: "Hermes 0.20.5 packaged snapshot", version: "0.20.5" },
        managedStatus: "UNKNOWN", compatibility: "NOT_REQUIRED",
        protectionPointRequired: false, protectionPointReady: false, blockedReasons: [],
        observedAt: "2026-08-27T10:00:00Z",
      }),
    });
    renderPanel(client, { ...instance, capabilities: { ...instance.capabilities, upgradePlan: true } }, "runtime");

    fireEvent.click(screen.getByRole("button", { name: "Maintenance" }));
    expect(await screen.findByText("Up to date")).toBeInTheDocument();
    expect(screen.getAllByText("Not required")).toHaveLength(2);
    expect(screen.queryByText("Evidence still required")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /upgrade|rollback/i })).not.toBeInTheDocument();
  });

  it("keeps Runtime maintenance out of the Instance panel", () => {
    const client = managementClient();
    renderPanel(client, { ...instance, capabilities: { ...instance.capabilities, backupRead: true, upgradePlan: true } });

    expect(screen.queryByText("Runtime backups")).not.toBeInTheDocument();
    expect(screen.queryByText("Upgrade plan")).not.toBeInTheDocument();
    expect(client.listRuntimeBackups).not.toHaveBeenCalled();
    expect(client.getRuntimeUpgradePlan).not.toHaveBeenCalled();
    expect(screen.getAllByText("Skill bindings").length).toBeGreaterThan(0);
    expect(screen.getAllByText("MCP bindings").length).toBeGreaterThan(0);
  });

  it("offers a bounded retry for failed reads", async () => {
    const listSkills = vi.fn()
      .mockRejectedValueOnce(new Error("query failed"))
      .mockResolvedValueOnce({ items: [] });
    renderPanel(managementClient({ listInstanceSkills: listSkills }));

    const retry = await screen.findByRole("button", { name: "Retry" });
    fireEvent.click(retry);
    await waitFor(() => expect(listSkills).toHaveBeenCalledTimes(2));
    expect(await screen.findByText("No Skills were reported for this instance.")).toBeInTheDocument();
  });
});

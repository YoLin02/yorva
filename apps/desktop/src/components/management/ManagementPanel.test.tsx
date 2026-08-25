import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { DaemonClient } from "../../api/client";
import type { Instance } from "../../api/types";
import { messages } from "../../i18n";
import { ManagementPanel } from "./ManagementPanel";

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
    skillRead: true, skillMutate: false, mcpRead: true, mcpMutate: false,
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
      id: "writer", sourceId: "official", version: "1.2.3", ownership: "EXTERNAL", projectionState: "PROJECTED", installationState: "INSTALLED",
      enabledState: "ENABLED", scanState: "CLEAN", updateAvailable: false,
    }] }),
    inspectInstanceSkill: vi.fn().mockResolvedValue({
      id: "writer", sourceId: "official", version: "1.2.3", ownership: "EXTERNAL", projectionState: "PROJECTED", installationState: "INSTALLED",
      enabledState: "ENABLED", scanState: "CLEAN", updateAvailable: false,
    }),
    listInstanceMCPServers: vi.fn().mockResolvedValue({ items: [{
      id: "docs", presetId: "approved-docs", state: "CONFIGURED", readyAt: null, observedAt: "2026-08-25T10:00:00Z",
    }] }),
    listInstanceMCPPresets: vi.fn().mockResolvedValue({ items: [{ id: "approved-docs", displayName: "Approved Docs" }] }),
    getRuntimeUpgradePlan: vi.fn().mockResolvedValue({
      state: "UNKNOWN", currentVersion: "0.20.2",
      candidate: { label: "Hermes 0.20.5 packaged snapshot", version: "0.20.5" },
      managedStatus: "MANAGED", compatibility: "UNKNOWN",
      protectionPointRequired: true, protectionPointReady: false,
      blockedReasons: ["COMPATIBILITY_UNKNOWN", "PROTECTION_POINT_REQUIRED"],
      observedAt: "2026-08-25T10:00:00Z",
    }),
    listRuntimeBackups: vi.fn().mockResolvedValue({ scope: "RUNTIME", items: [] }),
    listInstanceSkillSources: vi.fn().mockResolvedValue({ items: [] }),
    installManagedSkill: vi.fn(),
    updateManagedSkill: vi.fn(),
    enableManagedSkill: vi.fn(),
    disableManagedSkill: vi.fn(),
    removeManagedSkill: vi.fn(),
    getRuntimeBackup: vi.fn(),
    ...overrides,
  } as unknown as DaemonClient;
}

function renderPanel(client: DaemonClient, target: Instance = instance) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <ManagementPanel client={client} instance={target} copy={messages["en-US"]} locale="en-US" onClose={() => undefined} />
    </QueryClientProvider>,
  );
}

describe("ManagementPanel", () => {
  it("shows safe Skill and MCP reads while keeping CONFIGURED distinct from READY", async () => {
    const client = managementClient();
    renderPanel(client);

    expect(await screen.findByText("writer")).toBeInTheDocument();
    expect(await screen.findByText("docs")).toBeInTheDocument();
    expect(await screen.findAllByText("Degraded")).toHaveLength(2);
    expect(screen.getByText("bounded redacted entry")).toBeInTheDocument();
    await waitFor(() => expect(client.getInstanceLogSnapshot).toHaveBeenCalledWith("inst-coder", "ERRORS", expect.any(AbortSignal)));
    fireEvent.change(screen.getByRole("combobox", { name: "Category" }), { target: { value: "MCP" } });
    await waitFor(() => expect(client.getInstanceLogSnapshot).toHaveBeenCalledWith("inst-coder", "MCP", expect.any(AbortSignal)));
    expect(screen.getByText("Configured")).toBeInTheDocument();
    expect(screen.getByText("No current Ready evidence")).toBeInTheDocument();
    expect(screen.queryByText("Ready")).not.toBeInTheDocument();
    expect(screen.getByText("Approved Docs")).toBeInTheDocument();
    expect(screen.getByText("Runtime or externally owned; read-only in YORVA.")).toBeInTheDocument();
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
      listInstanceSkillSources: vi.fn().mockResolvedValue({ items: [{ sourceId: "yorva-demo", skillId: "yorva-managed-demo", displayName: "YORVA demo", version: "1.0.0" }] }),
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
    renderPanel(client, { ...instance, capabilities: { ...instance.capabilities, skillMutate: true } });

    expect(await screen.findByText("YORVA demo (1.0.0)")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Install" }));
    await waitFor(() => expect(installManagedSkill).toHaveBeenCalledWith("inst-coder", "yorva-managed-demo", "yorva-demo", expect.any(String)));
    fireEvent.click(screen.getByRole("button", { name: "Disable" }));
    await waitFor(() => expect(disableManagedSkill).toHaveBeenCalledWith("inst-coder", "yorva-managed-demo", expect.any(String)));
  });

  it("does not issue reads or present fake actions when capabilities are false", () => {
    const client = managementClient();
    const unavailable: Instance = {
      ...instance,
      capabilities: { ...instance.capabilities, healthRead: false, logsRead: false, skillRead: false, mcpRead: false },
    };
    renderPanel(client, unavailable);

    expect(screen.getAllByText("This capability is unavailable for the selected Runtime version.")).toHaveLength(5);
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
    renderPanel(client, { ...instance, capabilities: { ...instance.capabilities, backupRead: true } });

    expect(await screen.findByText("backup-safe")).toBeInTheDocument();
    expect(screen.getAllByText("Changed").length).toBeGreaterThan(0);
    expect(screen.getByText("Device-managed key")).toBeInTheDocument();
    expect(client.listRuntimeBackups).toHaveBeenCalledWith("hermes", expect.any(AbortSignal));
    const visible = document.body.textContent ?? "";
    for (const prohibited of ["artifactPath", "keyRef", "passphrase", "C:\\Backups"] ) expect(visible).not.toContain(prohibited);
  });

  it("shows an evidence-incomplete read-only Upgrade plan without mutation actions or internal identity", async () => {
    const client = managementClient();
    renderPanel(client, { ...instance, capabilities: { ...instance.capabilities, upgradePlan: true } });

    expect(await screen.findByText("Hermes 0.20.5 packaged snapshot (0.20.5)")).toBeInTheDocument();
    expect(screen.getByText("Exact current-to-candidate compatibility is not proven.")).toBeInTheDocument();
    expect(screen.getByText("Required; no verified protection point")).toBeInTheDocument();
    expect(client.getRuntimeUpgradePlan).toHaveBeenCalledWith("hermes", expect.any(AbortSignal));
    expect(screen.queryByRole("button", { name: /upgrade|rollback/i })).not.toBeInTheDocument();
    const visible = document.body.textContent ?? "";
    for (const prohibited of ["a0ca7c1", "df4b651", "sha256", "C:\\", "https://", "executable"]) {
      expect(visible.toLowerCase()).not.toContain(prohibited.toLowerCase());
    }
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

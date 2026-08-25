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
    instances: true, lifecycle: false, healthRead: false, logsRead: false, securityAudit: false,
    skillRead: true, skillMutate: false, mcpRead: true, mcpMutate: false,
    backupRead: false, backupMutate: false, restore: false, upgradePlan: false, upgrade: false, rollback: false,
  },
};

function managementClient(overrides: Partial<DaemonClient> = {}) {
  return {
    scope: "http://127.0.0.1:49152",
    listInstanceSkills: vi.fn().mockResolvedValue({ items: [{
      id: "writer", sourceId: "official", version: "1.2.3", installationState: "INSTALLED",
      enabledState: "ENABLED", scanState: "CLEAN", updateAvailable: false,
    }] }),
    inspectInstanceSkill: vi.fn().mockResolvedValue({
      id: "writer", sourceId: "official", version: "1.2.3", installationState: "INSTALLED",
      enabledState: "ENABLED", scanState: "CLEAN", updateAvailable: false,
    }),
    listInstanceMCPServers: vi.fn().mockResolvedValue({ items: [{
      id: "docs", presetId: "approved-docs", state: "CONFIGURED", readyAt: null, observedAt: "2026-08-25T10:00:00Z",
    }] }),
    listInstanceMCPPresets: vi.fn().mockResolvedValue({ items: [{ id: "approved-docs", displayName: "Approved Docs" }] }),
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
    expect(screen.getByText("Configured")).toBeInTheDocument();
    expect(screen.getByText("No current Ready evidence")).toBeInTheDocument();
    expect(screen.queryByText("Ready")).not.toBeInTheDocument();
    expect(screen.getByText("Approved Docs")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Inspect" }));
    await waitFor(() => expect(client.inspectInstanceSkill).toHaveBeenCalledWith("inst-coder", "writer", expect.any(AbortSignal)));
    expect(await screen.findByText("official")).toBeInTheDocument();
    expect(screen.getByText("1.2.3")).toBeInTheDocument();

    const visible = document.body.textContent ?? "";
    for (const prohibited of ["Authorization", "Bearer", "command", "header", "C:\\", "https://"]) {
      expect(visible).not.toContain(prohibited);
    }
  });

  it("does not issue reads or present fake actions when capabilities are false", () => {
    const client = managementClient();
    const unavailable: Instance = {
      ...instance,
      capabilities: { ...instance.capabilities, skillRead: false, mcpRead: false },
    };
    renderPanel(client, unavailable);

    expect(screen.getAllByText("This capability is unavailable for the selected Runtime version.")).toHaveLength(2);
    expect(client.listInstanceSkills).not.toHaveBeenCalled();
    expect(client.listInstanceMCPServers).not.toHaveBeenCalled();
    expect(screen.queryByRole("button", { name: /install|remove|test/i })).not.toBeInTheDocument();
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

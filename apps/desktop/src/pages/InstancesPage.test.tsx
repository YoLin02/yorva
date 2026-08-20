import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { DaemonClient } from "../api/client";
import type { InstanceList } from "../api/types";
import { messages } from "../i18n";
import { InstancesPage } from "./InstancesPage";

const inventory: InstanceList = {
  runtimeId: "hermes",
  runtimeInstallationId: "rtinst_test",
  freshness: "FRESH",
  lastSyncedAt: "2026-08-19T12:00:00Z",
  instances: [
    {
      instanceId: "inst_default",
      runtimeInstallationId: "rtinst_test",
      name: "default",
      default: true,
      protected: true,
      availability: "AVAILABLE",
      lastSyncedAt: "2026-08-19T12:00:00Z",
      createdAt: "2026-08-19T12:00:00Z",
      updatedAt: "2026-08-19T12:00:00Z",
      capabilities: { instances: true, lifecycle: false },
    },
    {
      instanceId: "inst_coder",
      runtimeInstallationId: "rtinst_test",
      name: "coder",
      default: false,
      protected: false,
      availability: "AVAILABLE",
      lastSyncedAt: "2026-08-19T12:00:00Z",
      createdAt: "2026-08-19T12:00:00Z",
      updatedAt: "2026-08-19T12:00:00Z",
      capabilities: { instances: true, lifecycle: false },
    },
  ],
  capabilities: { instances: true, lifecycle: false },
  errorCode: null,
};

describe("InstancesPage", () => {
  it("explains unsupported discovery without fake lifecycle controls", () => {
    render(
      <InstancesPage
        supported={false}
        loading={false}
        error={false}
        inventory={null}
        createName=""
        createBusy={false}
        createOperation={null}
        copy={messages["en-US"]}
        locale="en-US"
        onRefresh={() => undefined}
        onPrepareCreate={() => undefined}
        onCreateNameChange={() => undefined}
        onCreate={() => undefined}
        onCancelCreate={() => undefined}
        deleteTarget={null}
        deleteConfirmation=""
        deleteBusy={false}
        deleteOperation={null}
        onDeleteTargetChange={() => undefined}
        onDeleteConfirmationChange={() => undefined}
        onDelete={() => undefined}
        onCancelDelete={() => undefined}
      />,
    );
    expect(screen.getByText(messages["en-US"].instances.unsupportedTitle)).toBeInTheDocument();
    expect(screen.getByText(messages["en-US"].instances.unsupportedDescription)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Start" })).not.toBeInTheDocument();
  });

  it("shows default protected availability and empty named state", () => {
    const onRefresh = vi.fn();
    render(
      <InstancesPage
        supported
        loading={false}
        error={false}
        inventory={inventory}
        createName=""
        createBusy={false}
        createOperation={null}
        copy={messages["en-US"]}
        locale="en-US"
        onRefresh={onRefresh}
        onPrepareCreate={() => undefined}
        onCreateNameChange={() => undefined}
        onCreate={() => undefined}
        onCancelCreate={() => undefined}
        deleteTarget={null}
        deleteConfirmation=""
        deleteBusy={false}
        deleteOperation={null}
        onDeleteTargetChange={() => undefined}
        onDeleteConfirmationChange={() => undefined}
        onDelete={() => undefined}
        onCancelDelete={() => undefined}
      />,
    );
    expect(screen.getByText("default")).toBeInTheDocument();
    expect(screen.getByText("Protected")).toBeInTheDocument();
    expect(screen.getAllByText("Available").length).toBeGreaterThan(0);
    expect(screen.queryByText(messages["en-US"].instances.emptyNamed)).not.toBeInTheDocument();
    expect(screen.getByText(messages["en-US"].instances.lifecycleUnavailable)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Refresh" }));
    expect(onRefresh).toHaveBeenCalled();
  });

  it("keeps Chinese copy for unknown freshness", () => {
    render(
      <InstancesPage
        supported
        loading={false}
        error={false}
        inventory={{ ...inventory, freshness: "UNKNOWN" }}
        createName=""
        createBusy={false}
        createOperation={null}
        copy={messages["zh-CN"]}
        locale="zh-CN"
        onRefresh={() => undefined}
        onPrepareCreate={() => undefined}
        onCreateNameChange={() => undefined}
        onCreate={() => undefined}
        onCancelCreate={() => undefined}
        deleteTarget={null}
        deleteConfirmation=""
        deleteBusy={false}
        deleteOperation={null}
        onDeleteTargetChange={() => undefined}
        onDeleteConfirmationChange={() => undefined}
        onDelete={() => undefined}
        onCancelDelete={() => undefined}
      />,
    );
    expect(screen.getByText(messages["zh-CN"].instances.freshnessUnknown)).toBeInTheDocument();
    expect(screen.getByText("受保护")).toBeInTheDocument();
  });

  it("filters the real inventory and opens create in a dialog", () => {
    const onPrepareCreate = vi.fn();
    render(
      <InstancesPage
        supported
        loading={false}
        error={false}
        inventory={inventory}
        createName=""
        createBusy={false}
        createOperation={null}
        copy={messages["en-US"]}
        locale="en-US"
        onRefresh={() => undefined}
        onPrepareCreate={onPrepareCreate}
        onCreateNameChange={() => undefined}
        onCreate={() => undefined}
        onCancelCreate={() => undefined}
        deleteTarget={null}
        deleteConfirmation=""
        deleteBusy={false}
        deleteOperation={null}
        onDeleteTargetChange={() => undefined}
        onDeleteConfirmationChange={() => undefined}
        onDelete={() => undefined}
        onCancelDelete={() => undefined}
      />,
    );

    fireEvent.change(screen.getByRole("searchbox", { name: "Search instances" }), { target: { value: "coder" } });
    expect(screen.getByText("coder")).toBeInTheDocument();
    expect(screen.queryByText("default")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Create instance" }));
    expect(onPrepareCreate).toHaveBeenCalledOnce();
    expect(screen.getByRole("dialog", { name: "Create instance" })).toBeInTheDocument();
    expect(screen.getByLabelText("New instance name")).toHaveFocus();
  });

  it("opens a modal confirmation when Delete is clicked", () => {
    const onDeleteTargetChange = vi.fn();
    const { rerender } = render(
      <InstancesPage
        supported
        loading={false}
        error={false}
        inventory={inventory}
        createName=""
        createBusy={false}
        createOperation={null}
        copy={messages["en-US"]}
        locale="en-US"
        onRefresh={() => undefined}
        onPrepareCreate={() => undefined}
        onCreateNameChange={() => undefined}
        onCreate={() => undefined}
        onCancelCreate={() => undefined}
        deleteTarget={null}
        deleteConfirmation=""
        deleteBusy={false}
        deleteOperation={null}
        onDeleteTargetChange={onDeleteTargetChange}
        onDeleteConfirmationChange={() => undefined}
        onDelete={() => undefined}
        onCancelDelete={() => undefined}
      />,
    );
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Delete" }));
    expect(onDeleteTargetChange).toHaveBeenCalledWith(inventory.instances[1]);

    rerender(
      <InstancesPage
        supported
        loading={false}
        error={false}
        inventory={inventory}
        createName=""
        createBusy={false}
        createOperation={null}
        copy={messages["en-US"]}
        locale="en-US"
        onRefresh={() => undefined}
        onPrepareCreate={() => undefined}
        onCreateNameChange={() => undefined}
        onCreate={() => undefined}
        onCancelCreate={() => undefined}
        deleteTarget={inventory.instances[1]}
        deleteConfirmation=""
        deleteBusy={false}
        deleteOperation={null}
        onDeleteTargetChange={onDeleteTargetChange}
        onDeleteConfirmationChange={() => undefined}
        onDelete={() => undefined}
        onCancelDelete={() => undefined}
      />,
    );
    const dialog = screen.getByRole("dialog", { name: "Delete instance" });
    expect(dialog).toHaveAttribute("aria-modal", "true");
    expect(screen.getByText(messages["en-US"].instances.deleteWarning)).toBeInTheDocument();
    expect(within(dialog).getByRole("button", { name: "Delete" })).toBeDisabled();
  });

  it("does not offer delete for missing or unknown instances", () => {
    const tombstoned: InstanceList = {
      ...inventory,
      instances: [
        inventory.instances[0],
        { ...inventory.instances[1], availability: "MISSING" },
        {
          ...inventory.instances[1],
          instanceId: "inst_notes",
          name: "notes",
          availability: "UNKNOWN",
        },
      ],
    };
    render(
      <InstancesPage
        supported
        loading={false}
        error={false}
        inventory={tombstoned}
        createName=""
        createBusy={false}
        createOperation={null}
        copy={messages["en-US"]}
        locale="en-US"
        onRefresh={() => undefined}
        onPrepareCreate={() => undefined}
        onCreateNameChange={() => undefined}
        onCreate={() => undefined}
        onCancelCreate={() => undefined}
        deleteTarget={null}
        deleteConfirmation=""
        deleteBusy={false}
        deleteOperation={null}
        onDeleteTargetChange={() => undefined}
        onDeleteConfirmationChange={() => undefined}
        onDelete={() => undefined}
        onCancelDelete={() => undefined}
      />,
    );
    expect(screen.getByText("coder")).toBeInTheDocument();
    expect(screen.getByText("notes")).toBeInTheDocument();
    expect(screen.getAllByText("Missing").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Unknown").length).toBeGreaterThan(0);
    expect(screen.queryByRole("button", { name: "Delete" })).not.toBeInTheDocument();
    const modelButtons = screen.getAllByRole("button", { name: "Models" });
    expect(modelButtons.filter((button) => button.hasAttribute("disabled"))).toHaveLength(2);
  });

  it("shows authoritative lifecycle state and starts an Operation", async () => {
    const startInstanceLifecycle = vi.fn().mockResolvedValue({
      id: "op_start", type: "instance.start", targetType: "instance", targetId: "inst_coder",
      status: "PENDING", stage: "preflight", progress: null, message: "start", errorCode: null,
      retryable: false, correlationId: "cor", createdAt: "2026-08-20T12:00:00Z", startedAt: null,
      completedAt: null, updatedAt: "2026-08-20T12:00:00Z",
    });
    const client = {
      scope: "http://127.0.0.1:1",
      getInstanceLifecycle: vi.fn().mockResolvedValue({ state: "STOPPED", activeOperationId: null, observedAt: "2026-08-20T12:00:00Z", errorCode: null }),
      startInstanceLifecycle,
      getOperation: vi.fn().mockResolvedValue({
        id: "op_start", type: "instance.start", targetType: "instance", targetId: "inst_coder",
        status: "PENDING", stage: "preflight", progress: null, message: "start", errorCode: null,
        retryable: false, correlationId: "cor", createdAt: "2026-08-20T12:00:00Z", startedAt: null,
        completedAt: null, updatedAt: "2026-08-20T12:00:00Z",
      }),
    } as unknown as DaemonClient;
    const lifecycleInventory: InstanceList = {
      ...inventory,
      instances: [{ ...inventory.instances[1], capabilities: { instances: true, lifecycle: true } }],
      capabilities: { instances: true, lifecycle: true },
    };
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={queryClient}>
        <InstancesPage
          supported loading={false} error={false} inventory={lifecycleInventory} createName="" createBusy={false}
          createOperation={null} copy={messages["en-US"]} locale="en-US" onRefresh={() => undefined}
          onPrepareCreate={() => undefined} onCreateNameChange={() => undefined} onCreate={() => undefined}
          onCancelCreate={() => undefined} deleteTarget={null} deleteConfirmation="" deleteBusy={false}
          deleteOperation={null} onDeleteTargetChange={() => undefined} onDeleteConfirmationChange={() => undefined}
          onDelete={() => undefined} onCancelDelete={() => undefined} client={client}
        />
      </QueryClientProvider>,
    );
    expect(await screen.findByText("Stopped")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Start" }));
    await waitFor(() => expect(startInstanceLifecycle).toHaveBeenCalledWith("inst_coder", "start", expect.any(String)));
    expect(await screen.findByText("Starting")).toBeInTheDocument();
  });

  it("confirms a stop before starting the lifecycle Operation", async () => {
    const startInstanceLifecycle = vi.fn().mockResolvedValue({
      id: "op_stop", type: "instance.stop", targetType: "instance", targetId: "inst_coder",
      status: "PENDING", stage: "preflight", progress: null, message: "stop", errorCode: null,
      retryable: false, correlationId: "cor", createdAt: "2026-08-20T12:00:00Z", startedAt: null,
      completedAt: null, updatedAt: "2026-08-20T12:00:00Z",
    });
    const client = {
      scope: "http://127.0.0.1:1",
      getInstanceLifecycle: vi.fn().mockResolvedValue({ state: "RUNNING", activeOperationId: null, observedAt: "2026-08-20T12:00:00Z", errorCode: null }),
      startInstanceLifecycle,
      getOperation: vi.fn(),
    } as unknown as DaemonClient;
    const lifecycleInventory: InstanceList = {
      ...inventory,
      instances: [{ ...inventory.instances[1], capabilities: { instances: true, lifecycle: true } }],
      capabilities: { instances: true, lifecycle: true },
    };
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={queryClient}>
        <InstancesPage
          supported loading={false} error={false} inventory={lifecycleInventory} createName="" createBusy={false}
          createOperation={null} copy={messages["en-US"]} locale="en-US" onRefresh={() => undefined}
          onPrepareCreate={() => undefined} onCreateNameChange={() => undefined} onCreate={() => undefined}
          onCancelCreate={() => undefined} deleteTarget={null} deleteConfirmation="" deleteBusy={false}
          deleteOperation={null} onDeleteTargetChange={() => undefined} onDeleteConfirmationChange={() => undefined}
          onDelete={() => undefined} onCancelDelete={() => undefined} client={client}
        />
      </QueryClientProvider>,
    );
    expect(await screen.findByText("Running")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Stop" }));
    expect(startInstanceLifecycle).not.toHaveBeenCalled();
    const dialog = screen.getByRole("dialog", { name: "Confirm lifecycle action" });
    fireEvent.click(within(dialog).getByRole("button", { name: "Continue" }));
    await waitFor(() => expect(startInstanceLifecycle).toHaveBeenCalledWith("inst_coder", "stop", expect.any(String)));
  });
});

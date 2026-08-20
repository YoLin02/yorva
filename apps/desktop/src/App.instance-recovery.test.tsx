import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { InstanceList, Operation } from "./api/types";
import { App } from "./App";

const sessionMocks = vi.hoisted(() => ({ getDaemonSession: vi.fn() }));
const clientMocks = vi.hoisted(() => ({
  getNode: vi.fn(),
  detectHermes: vi.fn(),
  getHermesPrerequisites: vi.fn(),
  listOperations: vi.fn(),
  listHermesInstances: vi.fn(),
  getOperation: vi.fn(),
  cancelOperation: vi.fn(),
  createHermesInstance: vi.fn(),
  deleteInstance: vi.fn(),
}));

vi.mock("./api/session", () => ({
  getDaemonSession: sessionMocks.getDaemonSession,
  isDaemonNotReady: () => false,
}));
vi.mock("./api/client", async () => {
  const actual = await vi.importActual<typeof import("./api/client")>("./api/client");
  return {
    ...actual,
    createDaemonClient: () => ({
      getNode: clientMocks.getNode,
      detectHermes: clientMocks.detectHermes,
      getHermesPrerequisites: clientMocks.getHermesPrerequisites,
      listOperations: clientMocks.listOperations,
      listHermesInstances: clientMocks.listHermesInstances,
      getOperation: clientMocks.getOperation,
      cancelOperation: clientMocks.cancelOperation,
      createHermesInstance: clientMocks.createHermesInstance,
      deleteInstance: clientMocks.deleteInstance,
    }),
  };
});
vi.mock("./hooks/useEventStreamStatus", () => ({ useEventStreamStatus: () => "connected" }));

const session = { baseUrl: "http://127.0.0.1:49152", token: "token", protocolVersion: "1" };
const node = {
  id: "node_test",
  name: "DESKTOP-TEST",
  hostname: "DESKTOP-TEST",
  platform: "windows",
  architecture: "amd64",
  nodeVersion: "0.0.0-test",
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-01T00:00:00Z",
};
const discovery = {
  runtimeKind: "hermes",
  state: "SUPPORTED",
  errorCode: null,
  selected: { path: "C:\\Hermes\\hermes.exe", version: "0.20.2", state: "SUPPORTED", errorCode: null },
  candidates: [],
  warnings: [],
  detectedAt: "2026-08-19T00:00:00Z",
  supportedRange: ">=0.19.0 <0.21.0",
};

const inventory: InstanceList = {
  runtimeId: "hermes",
  runtimeInstallationId: "rtinst_test",
  freshness: "FRESH",
  lastSyncedAt: "2026-08-19T00:00:00Z",
  instances: [
    {
      instanceId: "inst_default",
      runtimeInstallationId: "rtinst_test",
      name: "default",
      default: true,
      protected: true,
      availability: "AVAILABLE",
      lastSyncedAt: "2026-08-19T00:00:00Z",
      createdAt: "2026-08-19T00:00:00Z",
      updatedAt: "2026-08-19T00:00:00Z",
      capabilities: { instances: true, lifecycle: false },
    },
    {
      instanceId: "inst_coder",
      runtimeInstallationId: "rtinst_test",
      name: "coder",
      default: false,
      protected: false,
      availability: "AVAILABLE",
      lastSyncedAt: "2026-08-19T00:00:00Z",
      createdAt: "2026-08-19T00:00:00Z",
      updatedAt: "2026-08-19T00:00:00Z",
      capabilities: { instances: true, lifecycle: false },
    },
  ],
  capabilities: { instances: true, lifecycle: false },
  errorCode: null,
};

function operation(partial: Partial<Operation> & Pick<Operation, "id" | "type" | "status">): Operation {
  return {
    targetType: "runtime-installation",
    targetId: "rtinst_test",
    stage: "instance.reconcile",
    progress: null,
    message: "coder",
    errorCode: null,
    retryable: true,
    correlationId: "cor_instance",
    createdAt: "2026-08-19T02:00:00Z",
    startedAt: "2026-08-19T02:00:01Z",
    completedAt: null,
    updatedAt: "2026-08-19T02:00:01Z",
    ...partial,
  };
}

function renderApp() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>,
  );
}

describe("App instance operation recovery", () => {
  beforeEach(() => {
    sessionMocks.getDaemonSession.mockReset().mockResolvedValue(session);
    clientMocks.getNode.mockReset().mockResolvedValue(node);
    clientMocks.detectHermes.mockReset().mockResolvedValue(discovery);
    clientMocks.getHermesPrerequisites.mockReset().mockResolvedValue({
      node: { state: "READY", version: "22.23.1", errorCode: null, retryable: false },
      npm: { state: "READY", version: "12.0.2", errorCode: null, retryable: false },
      nodeDependencies: { state: "READY", version: "", errorCode: null, retryable: false },
      checkedAt: "2026-08-19T00:00:00Z",
      activeOperationId: null,
    });
    clientMocks.listHermesInstances.mockReset().mockResolvedValue(inventory);
    clientMocks.listOperations.mockReset().mockResolvedValue({ operations: [] });
    clientMocks.getOperation.mockReset();
    clientMocks.cancelOperation.mockReset();
    clientMocks.createHermesInstance.mockReset();
    clientMocks.deleteInstance.mockReset();
  });

  it("reloads a running instance.create after Desktop restart", async () => {
    const live = operation({ id: "op_create", type: "instance.create", status: "RUNNING", message: "notes" });
    clientMocks.listOperations.mockImplementation((targetType: string) => {
      if (targetType === "runtime-installation") {
        return Promise.resolve({ operations: [live] });
      }
      return Promise.resolve({ operations: [] });
    });
    clientMocks.getOperation.mockResolvedValue(live);

    renderApp();
    fireEvent.click(screen.getByRole("button", { name: "Instances" }));
    expect(await screen.findByText("Creating instance")).toBeInTheDocument();
    await waitFor(() => {
      expect(clientMocks.listOperations).toHaveBeenCalledWith("runtime-installation", "rtinst_test", expect.anything());
    });
    expect(clientMocks.getOperation).toHaveBeenCalledWith("op_create", expect.anything());
  });

  it("reloads a running instance.delete dialog after Desktop restart", async () => {
    const live = operation({ id: "op_delete", type: "instance.delete", status: "RUNNING", message: "coder" });
    clientMocks.listOperations.mockImplementation((targetType: string) => {
      if (targetType === "runtime-installation") {
        return Promise.resolve({ operations: [live] });
      }
      return Promise.resolve({ operations: [] });
    });
    clientMocks.getOperation.mockResolvedValue(live);

    renderApp();
    fireEvent.click(screen.getByRole("button", { name: "Instances" }));
    expect(await screen.findByRole("heading", { name: "Delete instance" })).toBeInTheDocument();
    expect(await screen.findByText("Deleting instance")).toBeInTheDocument();
    expect(screen.getByDisplayValue("coder")).toBeInTheDocument();
  });

  it("does not restore a terminal instance operation as active", async () => {
    clientMocks.listOperations.mockImplementation((targetType: string) => {
      if (targetType === "runtime-installation") {
        return Promise.resolve({
          operations: [operation({ id: "op_done", type: "instance.create", status: "SUCCEEDED", message: "notes" })],
        });
      }
      return Promise.resolve({ operations: [] });
    });

    renderApp();
    fireEvent.click(screen.getByRole("button", { name: "Instances" }));
    expect(await screen.findByText("coder")).toBeInTheDocument();
    expect(screen.queryByText("Creating instance")).not.toBeInTheDocument();
    expect(screen.queryByText("Instance created")).not.toBeInTheDocument();
    expect(screen.queryByText("Create queued")).not.toBeInTheDocument();
    expect(clientMocks.getOperation).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Create instance" }));
    const dialog = screen.getByRole("dialog", { name: "Create instance" });
    fireEvent.change(screen.getByLabelText("New instance name"), { target: { value: "notes" } });
    expect(within(dialog).getByRole("button", { name: "Create instance" })).toBeEnabled();
  });
});

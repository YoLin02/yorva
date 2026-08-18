import { fireEvent, render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { YorvaApiError } from "./api/client";
import type { Operation } from "./api/types";
import { App } from "./App";

const sessionMocks = vi.hoisted(() => ({ getDaemonSession: vi.fn() }));
const clientMocks = vi.hoisted(() => ({
  getNode: vi.fn(),
  detectHermes: vi.fn(),
  getHermesPrerequisites: vi.fn(),
  startHermesPrerequisites: vi.fn(),
  startHermesInstall: vi.fn(),
  getOperation: vi.fn(),
  getOperationLog: vi.fn(),
  cancelOperation: vi.fn(),
  listOperations: vi.fn(),
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
      startHermesPrerequisites: clientMocks.startHermesPrerequisites,
      startHermesInstall: clientMocks.startHermesInstall,
      getOperation: clientMocks.getOperation,
      getOperationLog: clientMocks.getOperationLog,
      cancelOperation: clientMocks.cancelOperation,
      listOperations: clientMocks.listOperations,
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

function operation(partial: Partial<Operation> & Pick<Operation, "id" | "type" | "status">): Operation {
  return {
    targetType: "runtime-kind",
    targetId: "hermes",
    stage: "install.repository",
    progress: null,
    message: "",
    errorCode: null,
    retryable: true,
    correlationId: "cor_install",
    createdAt: "2026-08-17T02:00:00Z",
    startedAt: "2026-08-17T02:00:01Z",
    completedAt: null,
    updatedAt: "2026-08-17T02:00:01Z",
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

describe("App Hermes install recovery", () => {
  beforeEach(() => {
    sessionMocks.getDaemonSession.mockReset().mockResolvedValue(session);
    clientMocks.getNode.mockReset().mockResolvedValue(node);
    clientMocks.detectHermes.mockReset().mockResolvedValue({
      runtimeKind: "hermes",
      state: "NOT_INSTALLED",
      errorCode: "RUNTIME_NOT_INSTALLED",
      selected: null,
      candidates: [],
      warnings: [],
      detectedAt: "2026-08-14T00:00:00Z",
      supportedRange: ">=0.19.0 <0.21.0",
    });
    clientMocks.getHermesPrerequisites.mockReset().mockResolvedValue({
      node: { state: "READY", version: "22.23.1", errorCode: null, retryable: false },
      npm: { state: "READY", version: "12.0.2", errorCode: null, retryable: false },
      nodeDependencies: { state: "NOT_INSTALLED", version: "", errorCode: null, retryable: true },
      checkedAt: "2026-08-17T00:00:00Z",
      activeOperationId: null,
    });
    clientMocks.listOperations.mockReset().mockResolvedValue({ operations: [] });
    clientMocks.getOperation.mockReset();
    clientMocks.getOperationLog.mockReset().mockResolvedValue({ operationId: "op_install", correlationId: "cor", text: "stage=install.repository" });
    clientMocks.cancelOperation.mockReset();
    clientMocks.startHermesInstall.mockReset();
    clientMocks.startHermesPrerequisites.mockReset();
  });

  it("reloads a running runtime.install and can cancel it", async () => {
    const live = operation({ id: "op_install", type: "runtime.install", status: "RUNNING" });
    clientMocks.listOperations.mockResolvedValue({ operations: [live] });
    clientMocks.getOperation.mockResolvedValue(live);
    clientMocks.cancelOperation.mockResolvedValue({ ...live, status: "CANCELLED" });

    renderApp();
    fireEvent.click(screen.getByRole("button", { name: "Runtimes" }));
    expect(await screen.findByText("Installing Hermes")).toBeInTheDocument();
    expect(screen.getAllByText(/install\.repository/).length).toBeGreaterThan(0);

    clientMocks.getOperation.mockResolvedValue({ ...live, status: "CANCELLED", errorCode: "RUNTIME_INSTALL_CANCELLED" });
    fireEvent.click(screen.getByRole("button", { name: "Cancel installation" }));
    expect(await screen.findByText("Installation cancelled")).toBeInTheDocument();
    expect(clientMocks.cancelOperation).toHaveBeenCalledWith("op_install");
  });

  it("does not restore a terminal install as active", async () => {
    clientMocks.listOperations.mockResolvedValue({
      operations: [operation({ id: "op_done", type: "runtime.install", status: "SUCCEEDED" })],
    });

    renderApp();
    fireEvent.click(screen.getByRole("button", { name: "Runtimes" }));
    expect(await screen.findByRole("button", { name: "Install Hermes" })).toBeEnabled();
    expect(screen.queryByText("Installing Hermes")).not.toBeInTheDocument();
    expect(clientMocks.getOperation).not.toHaveBeenCalled();
  });

  it("attaches an install conflict to the install panel, not the prerequisite panel", async () => {
    const live = operation({ id: "op_install", type: "runtime.install", status: "RUNNING" });
    clientMocks.startHermesInstall.mockRejectedValue(
      new YorvaApiError({
        code: "RUNTIME_INSTALL_IN_PROGRESS",
        message: "The Runtime installation request was rejected.",
        retryable: true,
        details: { operationId: "op_install" },
      }),
    );
    clientMocks.getOperation.mockResolvedValue(live);

    renderApp();
    fireEvent.click(screen.getByRole("button", { name: "Runtimes" }));
    fireEvent.click(await screen.findByRole("button", { name: "Install Hermes" }));
    fireEvent.click(await screen.findByRole("button", { name: "Install" }));
    expect(await screen.findByText("Installing Hermes")).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("routes a prerequisite conflict on the install button away from the install panel", async () => {
    const live = operation({
      id: "op_prereq",
      type: "hermes.prerequisites",
      status: "RUNNING",
      stage: "install.node",
    });
    clientMocks.startHermesInstall.mockRejectedValue(
      new YorvaApiError({
        code: "RUNTIME_INSTALL_IN_PROGRESS",
        message: "The Runtime installation request was rejected.",
        retryable: true,
        details: { operationId: "op_prereq" },
      }),
    );
    clientMocks.getOperation.mockResolvedValue(live);

    renderApp();
    fireEvent.click(screen.getByRole("button", { name: "Runtimes" }));
    fireEvent.click(await screen.findByRole("button", { name: "Install Hermes" }));
    fireEvent.click(await screen.findByRole("button", { name: "Install" }));
    expect(await screen.findByText(/Installing Node\.js/)).toBeInTheDocument();
    expect(screen.queryByText("Installing Hermes")).not.toBeInTheDocument();
  });

  it("rejects a conflict operationId with the wrong target", async () => {
    clientMocks.startHermesInstall.mockRejectedValue(
      new YorvaApiError({
        code: "RUNTIME_INSTALL_IN_PROGRESS",
        message: "The Runtime installation request was rejected.",
        retryable: true,
        details: { operationId: "op_other" },
      }),
    );
    clientMocks.getOperation.mockResolvedValue(
      operation({ id: "op_other", type: "runtime.install", status: "RUNNING", targetId: "other" }),
    );

    renderApp();
    fireEvent.click(screen.getByRole("button", { name: "Runtimes" }));
    fireEvent.click(await screen.findByRole("button", { name: "Install Hermes" }));
    fireEvent.click(await screen.findByRole("button", { name: "Install" }));
    expect(await screen.findByText("Installation failed")).toBeInTheDocument();
    expect(screen.getAllByText(/RUNTIME_INSTALL_IN_PROGRESS/).length).toBeGreaterThan(0);
    expect(screen.queryByText("Installing Hermes")).not.toBeInTheDocument();
  });
});

import { fireEvent, render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { YorvaApiError } from "./api/client";
import { App } from "./App";

const sessionMocks = vi.hoisted(() => ({ getDaemonSession: vi.fn() }));
const clientMocks = vi.hoisted(() => ({
  getNode: vi.fn(),
  detectHermes: vi.fn(),
  getHermesPrerequisites: vi.fn(),
  startHermesPrerequisites: vi.fn(),
  getOperation: vi.fn(),
  getOperationLog: vi.fn(),
  cancelOperation: vi.fn(),
  listOperations: vi.fn(),
  startHermesInstall: vi.fn(),
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
      getOperation: clientMocks.getOperation,
      getOperationLog: clientMocks.getOperationLog,
      cancelOperation: clientMocks.cancelOperation,
      listOperations: clientMocks.listOperations,
      startHermesInstall: clientMocks.startHermesInstall,
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

function renderApp() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>,
  );
}

describe("App Hermes prerequisite start", () => {
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
      node: { state: "MISSING", version: "", errorCode: "RUNTIME_HERMES_NODE_MISSING", retryable: true },
      npm: { state: "MISSING", version: "", errorCode: "RUNTIME_HERMES_NPM_MISSING", retryable: true },
      nodeDependencies: { state: "NOT_INSTALLED", version: "", errorCode: null, retryable: true },
      checkedAt: "2026-08-17T00:00:00Z",
      activeOperationId: null,
    });
    clientMocks.startHermesPrerequisites.mockReset();
    clientMocks.startHermesInstall.mockReset();
    clientMocks.getOperation.mockReset();
    clientMocks.getOperationLog.mockReset().mockResolvedValue({ operationId: "op_prereq", correlationId: "cor", text: "" });
    clientMocks.cancelOperation.mockReset();
    clientMocks.listOperations.mockReset().mockResolvedValue({ operations: [] });
  });

  it("surfaces a rejected Node install request and retries with a new key", async () => {
    clientMocks.startHermesPrerequisites
      .mockRejectedValueOnce(
        new YorvaApiError({
          code: "NOT_FOUND",
          message: "The requested resource was not found.",
          retryable: false,
          details: {},
        }),
      )
      .mockResolvedValueOnce({
        id: "op_prereq",
        type: "hermes.prerequisites",
        targetType: "runtime-kind",
        targetId: "hermes",
        status: "PENDING",
        stage: "preflight",
        progress: null,
        message: "",
        errorCode: null,
        retryable: true,
        correlationId: "cor_prereq",
        createdAt: "2026-08-17T02:00:00Z",
        startedAt: null,
        completedAt: null,
        updatedAt: "2026-08-17T02:00:00Z",
      });
    clientMocks.getOperation.mockResolvedValue({
      id: "op_prereq",
      type: "hermes.prerequisites",
      targetType: "runtime-kind",
      targetId: "hermes",
      status: "RUNNING",
      stage: "install.node",
      progress: null,
      message: "",
      errorCode: null,
      retryable: true,
      correlationId: "cor_prereq",
      createdAt: "2026-08-17T02:00:00Z",
      startedAt: "2026-08-17T02:00:01Z",
      completedAt: null,
      updatedAt: "2026-08-17T02:00:01Z",
    });

    renderApp();
    fireEvent.click(screen.getByRole("button", { name: "Runtimes" }));
    fireEvent.click(await screen.findByRole("button", { name: "Install or reinstall Node.js / npm" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Node.js / npm installation failed");
    expect(screen.getByText(/NOT_FOUND/)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Retry Node.js / npm installation" }));
    expect(await screen.findByText(/Installing Node\.js/)).toBeInTheDocument();
    expect(clientMocks.startHermesPrerequisites).toHaveBeenCalledTimes(2);
    const firstKey = clientMocks.startHermesPrerequisites.mock.calls[0][0] as string;
    const retryKey = clientMocks.startHermesPrerequisites.mock.calls[1][0] as string;
    expect(firstKey).toMatch(/^[0-9a-f-]{36}$/i);
    expect(retryKey).toMatch(/^[0-9a-f-]{36}$/i);
    expect(retryKey).not.toBe(firstKey);
  });

  it("resumes a running Node install so the user can cancel and continue", async () => {
    clientMocks.getHermesPrerequisites.mockResolvedValue({
      node: { state: "MISSING", version: "", errorCode: "RUNTIME_HERMES_NODE_MISSING", retryable: true },
      npm: { state: "MISSING", version: "", errorCode: "RUNTIME_HERMES_NPM_MISSING", retryable: true },
      nodeDependencies: { state: "NOT_INSTALLED", version: "", errorCode: null, retryable: true },
      checkedAt: "2026-08-17T00:00:00Z",
      activeOperationId: "op_live",
    });
    clientMocks.getOperation.mockResolvedValue({
      id: "op_live",
      type: "hermes.prerequisites",
      targetType: "runtime-kind",
      targetId: "hermes",
      status: "RUNNING",
      stage: "install.node",
      progress: null,
      message: "",
      errorCode: null,
      retryable: true,
      correlationId: "cor_live",
      createdAt: "2026-08-17T02:00:00Z",
      startedAt: "2026-08-17T02:00:01Z",
      completedAt: null,
      updatedAt: "2026-08-17T02:00:01Z",
    });

    renderApp();
    fireEvent.click(screen.getByRole("button", { name: "Runtimes" }));

    expect(await screen.findByText(/Installing Node\.js/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
    expect(screen.getByText(/cannot run at the same time/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Install Hermes" })).toBeDisabled();
  });

  it("attaches to an in-progress Operation instead of leaving the click dead", async () => {
    clientMocks.startHermesPrerequisites.mockRejectedValue(
      new YorvaApiError({
        code: "RUNTIME_INSTALL_IN_PROGRESS",
        message: "The Runtime installation request was rejected.",
        retryable: true,
        details: { operationId: "op_live" },
      }),
    );
    clientMocks.getOperation.mockResolvedValue({
      id: "op_live",
      type: "hermes.prerequisites",
      targetType: "runtime-kind",
      targetId: "hermes",
      status: "RUNNING",
      stage: "install.node",
      progress: null,
      message: "",
      errorCode: null,
      retryable: true,
      correlationId: "cor_live",
      createdAt: "2026-08-17T02:00:00Z",
      startedAt: "2026-08-17T02:00:01Z",
      completedAt: null,
      updatedAt: "2026-08-17T02:00:01Z",
    });

    renderApp();
    fireEvent.click(screen.getByRole("button", { name: "Runtimes" }));
    fireEvent.click(await screen.findByRole("button", { name: "Install or reinstall Node.js / npm" }));

    expect(await screen.findByText(/Installing Node\.js/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("does not treat a running Hermes install as a Node prerequisite", async () => {
    clientMocks.startHermesPrerequisites.mockRejectedValue(
      new YorvaApiError({
        code: "RUNTIME_INSTALL_IN_PROGRESS",
        message: "The Runtime installation request was rejected.",
        retryable: true,
        details: { operationId: "op_install" },
      }),
    );
    clientMocks.getOperation.mockResolvedValue({
      id: "op_install",
      type: "runtime.install",
      targetType: "runtime-kind",
      targetId: "hermes",
      status: "RUNNING",
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
    });

    renderApp();
    fireEvent.click(screen.getByRole("button", { name: "Runtimes" }));
    fireEvent.click(await screen.findByRole("button", { name: "Install or reinstall Node.js / npm" }));
    expect(await screen.findByText("Installing Hermes")).toBeInTheDocument();
    expect(screen.queryByText(/Installing Node\.js/)).not.toBeInTheDocument();
  });
});

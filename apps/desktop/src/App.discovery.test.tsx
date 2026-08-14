import { act, fireEvent, render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { RuntimeDiscovery } from "./api/types";
import { App } from "./App";

const sessionMocks = vi.hoisted(() => ({ getDaemonSession: vi.fn() }));
const clientMocks = vi.hoisted(() => ({ getNode: vi.fn(), detectHermes: vi.fn() }));

vi.mock("./api/session", () => ({
  getDaemonSession: sessionMocks.getDaemonSession,
  isDaemonNotReady: () => false,
}));
vi.mock("./api/client", () => ({
  createDaemonClient: () => ({ getNode: clientMocks.getNode, detectHermes: clientMocks.detectHermes }),
}));
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

function supported(version: string): RuntimeDiscovery {
  const candidate = {
    path: "C:\\Hermes\\hermes.exe",
    version,
    state: "SUPPORTED" as const,
    errorCode: null,
  };
  return {
    runtimeKind: "hermes",
    state: "SUPPORTED",
    errorCode: null,
    selected: candidate,
    candidates: [candidate],
    warnings: [],
    detectedAt: "2026-08-14T00:00:00Z",
    supportedRange: ">=0.19.0 <0.20.0",
  };
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => { resolve = done; });
  return { promise, resolve };
}

function renderApp() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>,
  );
}

describe("App Hermes discovery", () => {
  beforeEach(() => {
    sessionMocks.getDaemonSession.mockReset().mockResolvedValue(session);
    clientMocks.getNode.mockReset().mockResolvedValue(node);
    clientMocks.detectHermes.mockReset();
  });

  it("shows a safe request failure and retries", async () => {
    clientMocks.detectHermes
      .mockRejectedValueOnce(new Error("raw transport detail"))
      .mockResolvedValueOnce(supported("0.19.4"));
    renderApp();

    expect(await screen.findByRole("alert")).toHaveTextContent("Discovery unavailable");
    expect(screen.queryByText("raw transport detail")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(await screen.findByText("0.19.4")).toBeInTheDocument();
    expect(clientMocks.detectHermes).toHaveBeenCalledTimes(2);
  });

  it("cancels the in-flight request and ignores its stale completion after retry", async () => {
    const first = deferred<RuntimeDiscovery>();
    const second = deferred<RuntimeDiscovery>();
    let firstSignal: AbortSignal | undefined;
    clientMocks.detectHermes
      .mockImplementationOnce((signal: AbortSignal) => {
        firstSignal = signal;
        return first.promise;
      })
      .mockImplementationOnce(() => second.promise);
    renderApp();

    fireEvent.click(await screen.findByRole("button", { name: "Cancel" }));
    expect(firstSignal?.aborted).toBe(true);
    expect(await screen.findByText("Check cancelled")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));

    await act(async () => { first.resolve(supported("0.19.1")); });
    expect(screen.queryByText("0.19.1")).not.toBeInTheDocument();
    await act(async () => { second.resolve(supported("0.19.2")); });
    expect(await screen.findByText("0.19.2")).toBeInTheDocument();
    expect(clientMocks.detectHermes).toHaveBeenCalledTimes(2);
  });
});

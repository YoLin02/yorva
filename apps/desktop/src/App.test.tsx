import { act, render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { App } from "./App";

const sessionMocks = vi.hoisted(() => ({
  getDaemonSession: vi.fn(),
}));

const clientMocks = vi.hoisted(() => ({
  getNode: vi.fn(),
}));

vi.mock("./api/session", () => ({
  getDaemonSession: sessionMocks.getDaemonSession,
  isDaemonNotReady: (error: unknown) =>
    typeof error === "object" &&
    error !== null &&
    (error as { code?: string }).code === "DAEMON_NOT_READY",
}));

vi.mock("./api/client", () => ({
  createDaemonClient: () => ({
    getNode: clientMocks.getNode,
  }),
}));

vi.mock("./hooks/useEventStreamStatus", () => ({
  useEventStreamStatus: () => "connected",
}));

const session = {
  baseUrl: "http://127.0.0.1:49152",
  token: "desktop-owned-token",
  protocolVersion: "1",
};

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
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { gcTime: Infinity },
    },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>,
  );
}

describe("App daemon startup", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    sessionMocks.getDaemonSession.mockReset();
    clientMocks.getNode.mockReset();
    clientMocks.getNode.mockResolvedValue(node);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("keeps observing startup until the native lifecycle becomes ready", async () => {
    for (let attempt = 0; attempt < 45; attempt += 1) {
      sessionMocks.getDaemonSession.mockRejectedValueOnce({
        code: "DAEMON_NOT_READY",
        message: "The local daemon is still starting.",
        retryable: true,
      });
    }
    sessionMocks.getDaemonSession.mockResolvedValueOnce(session);

    renderApp();
    expect(screen.getByRole("status")).toHaveTextContent("Starting local node");

    await act(async () => {
      await vi.runAllTimersAsync();
    });

    expect(screen.getByText("DESKTOP-TEST")).toBeInTheDocument();
    expect(sessionMocks.getDaemonSession).toHaveBeenCalledTimes(46);
  });

  it("renders the safe terminal failure returned by the native lifecycle", async () => {
    sessionMocks.getDaemonSession
      .mockRejectedValueOnce({
        code: "DAEMON_NOT_READY",
        message: "The local daemon is still starting.",
        retryable: true,
      })
      .mockRejectedValueOnce({
        code: "DAEMON_STARTUP_FAILED",
        message: "The local daemon could not be started.",
        retryable: false,
      });

    renderApp();
    await act(async () => {
      await vi.runAllTimersAsync();
    });

    expect(screen.getByRole("alert")).toHaveTextContent("The local daemon could not start.");
    expect(screen.queryByText(/C:\\|spawn|token|stack/i)).not.toBeInTheDocument();
    expect(sessionMocks.getDaemonSession).toHaveBeenCalledTimes(2);
  });
});

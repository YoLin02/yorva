import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { App } from "./App";
import { getDaemonSession } from "./api/session";

vi.mock("./api/session", async () => {
  const actual = await vi.importActual<typeof import("./api/session")>("./api/session");
  return { ...actual, getDaemonSession: vi.fn() };
});

vi.mock("./api/client", () => ({
  createDaemonClient: () => ({
    getNode: vi.fn().mockResolvedValue({
      id: "node_test",
      name: "DESKTOP-TEST",
      hostname: "DESKTOP-TEST",
      platform: "windows",
      architecture: "amd64",
      nodeVersion: "0.0.0-test",
      createdAt: "2026-01-01T00:00:00Z",
      updatedAt: "2026-01-01T00:00:00Z",
    }),
    connectEvents: vi.fn(() => new Promise<void>(() => undefined)),
  }),
}));

vi.mock("./hooks/useEventStreamStatus", () => ({
  useEventStreamStatus: () => "connected",
}));

const mockedGetDaemonSession = vi.mocked(getDaemonSession);

function renderApp() {
  const client = new QueryClient({
    defaultOptions: { queries: { gcTime: 0 } },
  });
  return render(
    <QueryClientProvider client={client}>
      <App />
    </QueryClientProvider>,
  );
}

function notReadyError() {
  return { code: "DAEMON_NOT_READY", message: "The local daemon is still starting.", retryable: true };
}

beforeEach(() => {
  vi.useFakeTimers();
  mockedGetDaemonSession.mockReset();
});

afterEach(() => {
  vi.useRealTimers();
});

describe("App daemon startup observation", () => {
  it("keeps observing beyond the old four-second retry budget until the native lifecycle becomes ready", async () => {
    for (let attempt = 0; attempt < 25; attempt += 1) {
      mockedGetDaemonSession.mockRejectedValueOnce(notReadyError());
    }
    mockedGetDaemonSession.mockResolvedValueOnce({
      baseUrl: "http://127.0.0.1:12345",
      token: "test-token",
      protocolVersion: "1",
    });

    renderApp();
    expect(screen.getByRole("status")).toHaveTextContent("Starting local node");

    await act(async () => {
      await vi.advanceTimersByTimeAsync(5_200);
    });

    expect(await screen.findByText("Local node connected")).toBeInTheDocument();
    expect(mockedGetDaemonSession).toHaveBeenCalledTimes(26);
  });

  it("stops retrying when the authoritative native lifecycle reports final startup failure", async () => {
    mockedGetDaemonSession
      .mockRejectedValueOnce(notReadyError())
      .mockRejectedValueOnce({
        code: "DAEMON_STARTUP_FAILED",
        message: "The local daemon could not be started.",
        retryable: false,
      });

    renderApp();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(250);
    });

    expect(await screen.findByRole("alert")).toHaveTextContent("The local daemon could not start.");
    expect(mockedGetDaemonSession).toHaveBeenCalledTimes(2);
  });
});

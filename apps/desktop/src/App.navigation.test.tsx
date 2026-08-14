import { fireEvent, render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { App } from "./App";
import { localeStorageKey } from "./i18n";

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
const discovery = {
  runtimeKind: "hermes",
  state: "SUPPORTED",
  errorCode: null,
  selected: { path: "C:\\Hermes\\hermes.exe", version: "0.20.1", state: "SUPPORTED", errorCode: null },
  candidates: [],
  warnings: [],
  detectedAt: "2026-08-14T10:11:57Z",
  supportedRange: ">=0.19.0 <0.21.0",
};

function renderApp() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  return render(<QueryClientProvider client={queryClient}><App /></QueryClientProvider>);
}

describe("App Desktop navigation and locale", () => {
  beforeEach(() => {
    sessionMocks.getDaemonSession.mockReset().mockResolvedValue(session);
    clientMocks.getNode.mockReset().mockResolvedValue(node);
    clientMocks.detectHermes.mockReset().mockResolvedValue(discovery);
  });

  it("uses separate Dashboard and Runtimes surfaces", async () => {
    renderApp();
    expect(await screen.findByText("DESKTOP-TEST")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Local Node" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Hermes Runtime" })).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Runtimes" }));
    expect(await screen.findByRole("heading", { name: "Hermes discovery" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Local Node" })).not.toBeInTheDocument();
  });

  it("switches language immediately and persists the selection", async () => {
    const first = renderApp();
    await screen.findByText("DESKTOP-TEST");
    fireEvent.click(screen.getByRole("button", { name: "Settings" }));
    fireEvent.click(screen.getByRole("radio", { name: /简体中文/ }));

    expect(screen.getByRole("button", { name: "仪表盘" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "语言" })).toBeInTheDocument();
    expect(window.localStorage.getItem(localeStorageKey)).toBe("zh-CN");

    first.unmount();
    renderApp();
    expect(screen.getByRole("button", { name: "仪表盘" })).toBeInTheDocument();
  });
});

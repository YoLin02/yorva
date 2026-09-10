import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { App } from "./App";
import { localeStorageKey, messages } from "./i18n";

vi.mock("./api/session", () => ({
  getDaemonSession: async () => ({ baseUrl: "http://127.0.0.1:49152", token: "test-token", protocolVersion: "1" }),
  isDaemonNotReady: () => false,
}));
vi.mock("./hooks/useEventStreamStatus", () => ({ useEventStreamStatus: () => "connected" }));

const observedAt = "2026-09-10T00:00:00Z";
const copy = messages["en-US"];
const capabilities = { instances: true, lifecycle: true, models: false, channels: false, healthRead: true };
function inventory(kind: string, name: string) {
  return {
    runtimeId: kind, runtimeInstallationId: `rtinst_${kind}`, freshness: "FRESH", errorCode: null,
    lastSyncedAt: observedAt, capabilities,
    instances: [{ instanceId: `inst_${kind}`, runtimeInstallationId: `rtinst_${kind}`, name,
      default: false, protected: false, availability: "AVAILABLE", capabilities,
      lastSyncedAt: observedAt, createdAt: observedAt, updatedAt: observedAt }],
  };
}
function response(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

describe("second Runtime routing through the real HTTP client", () => {
  let pendingHermes: ((value: Response) => void) | undefined;
  let delayHermes: boolean;
  let requests: Array<{ path: string; init: RequestInit }>;
  beforeEach(() => {
    localStorage.setItem(localeStorageKey, "en-US");
    delayHermes = false;
    requests = [];
    vi.stubGlobal("fetch", vi.fn(async (url: string, init: RequestInit = {}) => {
      const path = new URL(url).pathname;
      requests.push({ path, init });
      if (path.endsWith("/node")) return response({ id: "node_test", name: "TEST-NODE", hostname: "TEST-NODE", platform: "windows", architecture: "amd64", nodeVersion: "test", createdAt: observedAt, updatedAt: observedAt });
      if (path.endsWith("/detect")) {
        const kind = path.includes("/openclaw/") ? "openclaw" : "hermes";
        return response({ runtimeKind: kind, state: "SUPPORTED", errorCode: null, selected: { path: `C:\\${kind}\\entry`, version: kind === "openclaw" ? "2026.9.3" : "0.20.2", state: "SUPPORTED", errorCode: null }, candidates: [], warnings: [], detectedAt: observedAt, supportedRange: "2026.9.3" });
      }
      if (path.endsWith("/operations")) return response({ operations: [] });
      if (path.endsWith("/instances") && init.method === "POST") return response({ error: { code: "INSTANCE_CONFLICT", message: "Conflict", retryable: false, details: {} } }, 409);
      if (path === "/api/v1/runtimes/hermes/instances") {
        if (delayHermes) return new Promise<Response>((resolve) => { pendingHermes = resolve; });
        return response(inventory("hermes", "hermes-only"));
      }
      if (path === "/api/v1/runtimes/openclaw/instances") return response(inventory("openclaw", "claw-only"));
      if (path.endsWith("/lifecycle")) return response({ state: "STOPPED", activeOperationId: null, errorCode: null, observedAt });
      return response({ error: { code: "NOT_FOUND", message: "Missing test route", retryable: false, details: {} } }, 404);
    }));
  });
  afterEach(() => vi.unstubAllGlobals());

  function mount() {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
    render(<QueryClientProvider client={client}><App /></QueryClientProvider>);
  }
  async function openClaw() {
    await screen.findByText("TEST-NODE");
    fireEvent.click(screen.getByRole("button", { name: "Instances" }));
    fireEvent.click(await screen.findByRole("button", { name: "OpenClaw" }));
    await screen.findByText("claw-only");
  }

  it("keeps late Hermes inventory out of OpenClaw and hides unsupported entry points", async () => {
    delayHermes = true;
    mount();
    await openClaw();
    await waitFor(() => expect(pendingHermes).toBeDefined());
    await act(async () => pendingHermes!(response(inventory("hermes", "late-hermes"))));
    expect(screen.queryByText("late-hermes")).not.toBeInTheDocument();
    expect(screen.getByText("claw-only")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /model configuration|channels/i })).not.toBeInTheDocument();
    expect(requests.some(({ path }) => /model-provider-presets|model-configuration|\/channels|\/mcp/.test(path))).toBe(false);
    expect(requests.some(({ path, init }) => path.endsWith("/start") && init.method === "POST")).toBe(false);
    delayHermes = false;
    fireEvent.click(screen.getByRole("button", { name: "Hermes Agent" }));
    expect(await screen.findByText("hermes-only")).toBeInTheDocument();
    expect(screen.queryByText("claw-only")).not.toBeInTheDocument();
  });

  it("posts a typed create request to OpenClaw and displays rejection inside the dialog", async () => {
    mount();
    await openClaw();
    fireEvent.click(screen.getByRole("button", { name: copy.instances.createAction }));
    const dialog = await screen.findByRole("dialog");
    expect(dialog).toHaveTextContent(copy.instances.createDescription);
    expect(dialog).not.toHaveTextContent("Hermes");
    fireEvent.change(within(dialog).getByLabelText(copy.instances.createLabel), { target: { value: "work" } });
    fireEvent.click(within(dialog).getByRole("button", { name: copy.instances.createAction }));
    expect(await within(dialog).findByRole("alert")).toHaveTextContent(copy.instances.createFailed);
    const request = requests.find(({ path, init }) => path === "/api/v1/runtimes/openclaw/instances" && init.method === "POST");
    expect(request?.init.body).toBe(JSON.stringify({ name: "work" }));
    expect(request?.init.headers).toMatchObject({ "Content-Type": "application/json", Authorization: "Bearer test-token", "Idempotency-Key": expect.any(String) });
    expect(requests.some(({ path, init }) => path === "/api/v1/runtimes/hermes/instances" && init.method === "POST")).toBe(false);
  });

  it("shows OpenClaw compatibility without borrowing Hermes status text", async () => {
    mount();
    await openClaw();
    fireEvent.click(screen.getByRole("button", { name: "Runtimes" }));
    const overview = await screen.findByRole("region", { name: "OpenClaw" });
    expect(await within(overview).findByText("Ready")).toBeInTheDocument();
    expect(overview).not.toHaveTextContent("Hermes");
  });
});

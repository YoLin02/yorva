import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { DaemonClient } from "../../api/client";
import type { Instance } from "../../api/types";
import { messages } from "../../i18n";
import { ChannelPanel } from "./ChannelPanel";

const instance: Instance = {
  instanceId: "inst_coder",
  runtimeInstallationId: "rtinst_test",
  name: "coder",
  default: false,
  protected: false,
  availability: "AVAILABLE",
  lastSyncedAt: "2026-08-20T12:00:00Z",
  createdAt: "2026-08-20T12:00:00Z",
  updatedAt: "2026-08-20T12:00:00Z",
  capabilities: { instances: true, lifecycle: true },
};

function channelClient(overrides: Partial<DaemonClient> = {}) {
  return {
    scope: "http://127.0.0.1:49152",
    listInstanceChannels: vi.fn().mockResolvedValue({
      channels: [
        { type: "weixin", state: "NOT_CONFIGURED", accountLabel: "", externalId: "", lastCheckedAt: "2026-08-20T12:00:00Z", activeOperationId: null },
        { type: "wecom", state: "NOT_CONFIGURED", accountLabel: "", externalId: "", lastCheckedAt: "2026-08-20T12:00:00Z", activeOperationId: null },
      ],
    }),
    getInstanceLifecycle: vi.fn().mockResolvedValue({ state: "STOPPED", activeOperationId: null, observedAt: "2026-08-20T12:00:00Z", errorCode: null }),
    connectWeixin: vi.fn(),
    connectWeCom: vi.fn().mockResolvedValue({ id: "op_wecom", type: "channel.connect", targetType: "instance", targetId: "inst_coder", status: "PENDING", stage: "preparing", progress: null, message: "", errorCode: null, retryable: false, correlationId: "cor_1", createdAt: "2026-08-20T12:00:00Z", startedAt: null, completedAt: null, updatedAt: "2026-08-20T12:00:00Z" }),
    disconnectChannel: vi.fn(),
    getChannelQr: vi.fn(),
    getOperation: vi.fn().mockResolvedValue({ id: "op_wecom", type: "channel.connect", targetType: "instance", targetId: "inst_coder", status: "PENDING", stage: "preparing", progress: null, message: "", errorCode: null, retryable: false, correlationId: "cor_1", createdAt: "2026-08-20T12:00:00Z", startedAt: null, completedAt: null, updatedAt: "2026-08-20T12:00:00Z" }),
    cancelOperation: vi.fn(),
    ...overrides,
  } as unknown as DaemonClient;
}

function renderPanel(client: DaemonClient) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <ChannelPanel client={client} instance={instance} copy={messages["zh-CN"]} onClose={() => undefined} />
    </QueryClientProvider>,
  );
}

describe("ChannelPanel", () => {
  it("separates channel state from gateway state and clears the submitted WeCom secret", async () => {
    const client = channelClient();
    renderPanel(client);
    expect(await screen.findByText("通道连接与网关运行是两个独立状态。通道已连接不代表网关正在运行。")).toBeInTheDocument();
		await waitFor(() => expect(screen.getByText(/网关状态/)).toHaveTextContent("已停止"));

    fireEvent.change(screen.getByLabelText("Bot ID"), { target: { value: "bot-one" } });
    fireEvent.change(screen.getByLabelText("Secret"), { target: { value: "secret-one" } });
    const enterpriseCard = screen.getByText("企业微信").closest("article")!;
    fireEvent.click(enterpriseCard.querySelector("button.button-primary")!);

    await waitFor(() => expect(client.connectWeCom).toHaveBeenCalledWith("inst_coder", "bot-one", "secret-one", expect.any(String)));
    expect(screen.getByLabelText("Secret")).toHaveValue("");
    expect(window.localStorage.length).toBe(0);
    expect(window.sessionStorage.length).toBe(0);
  });

  it("fetches the expiring Weixin QR only while the connect Operation is active", async () => {
    const operation = { id: "op_weixin", type: "channel.connect", targetType: "instance", targetId: "inst_coder", status: "RUNNING", stage: "channel.qr-ready", progress: null, message: "", errorCode: null, retryable: false, correlationId: "cor_2", createdAt: "2026-08-20T12:00:00Z", startedAt: "2026-08-20T12:00:01Z", completedAt: null, updatedAt: "2026-08-20T12:00:01Z" } as const;
    const getChannelQr = vi.fn().mockResolvedValue({ payload: "https://safe.example/ephemeral", expiresAt: new Date(Date.now() + 60_000).toISOString() });
    const client = channelClient({
      listInstanceChannels: vi.fn().mockResolvedValue({ channels: [
        { type: "weixin", state: "CONNECTING", accountLabel: "", externalId: "", lastCheckedAt: "2026-08-20T12:00:00Z", activeOperationId: "op_weixin" },
        { type: "wecom", state: "NOT_CONFIGURED", accountLabel: "", externalId: "", lastCheckedAt: "2026-08-20T12:00:00Z", activeOperationId: null },
      ] }),
      getOperation: vi.fn().mockResolvedValue(operation),
      getChannelQr,
    });
    const rendered = renderPanel(client);
    expect(await screen.findByText("使用微信扫码")).toBeInTheDocument();
    await waitFor(() => expect(getChannelQr).toHaveBeenCalledWith("op_weixin", expect.any(AbortSignal)));
    expect(rendered.container.querySelector("svg")).toBeInTheDocument();
    expect(screen.queryByText("https://safe.example/ephemeral")).not.toBeInTheDocument();
  });

  it("keeps a terminal Weixin dialog visible and explains a cancelled operation", async () => {
    const operation = { id: "op_cancelled", type: "channel.connect", targetType: "instance", targetId: "inst_coder", status: "CANCELLED", stage: "channel.qr-ready", progress: null, message: "", errorCode: "CHANNEL_AUTH_CANCELLED", retryable: false, correlationId: "cor_3", createdAt: "2026-08-20T12:00:00Z", startedAt: "2026-08-20T12:00:01Z", completedAt: "2026-08-20T12:00:03Z", updatedAt: "2026-08-20T12:00:03Z" } as const;
    const client = channelClient({
      listInstanceChannels: vi.fn().mockResolvedValue({ channels: [
        { type: "weixin", state: "NOT_CONFIGURED", accountLabel: "", externalId: "", lastCheckedAt: "2026-08-20T12:00:00Z", activeOperationId: "op_cancelled" },
        { type: "wecom", state: "NOT_CONFIGURED", accountLabel: "", externalId: "", lastCheckedAt: "2026-08-20T12:00:00Z", activeOperationId: null },
      ] }),
      getOperation: vi.fn().mockResolvedValue(operation),
    });
    renderPanel(client);

    expect(await screen.findByRole("dialog", { name: "使用微信扫码" })).toBeInTheDocument();
    expect(screen.getAllByText("连接已取消").length).toBeGreaterThan(0);
    expect(screen.getAllByText("CHANNEL_AUTH_CANCELLED").length).toBeGreaterThan(0);
    expect(screen.getByText("已取消")).toBeInTheDocument();
    expect(screen.queryByText(/^失败$/)).not.toBeInTheDocument();
  });
});

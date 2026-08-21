import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { DaemonClient } from "../../api/client";
import type { HermesDownloadSources } from "../../api/types";
import { messages } from "../../i18n";
import { HermesDownloadSourcesPanel } from "./HermesDownloadSourcesPanel";

const defaults: HermesDownloadSources = {
  hermesArchiveUrl: "https://github.com/example/hermes.zip",
  nodeArchiveUrl: "https://npmmirror.com/mirrors/node/node.zip",
  npmArchiveUrl: "https://registry.npmmirror.com/npm/-/npm.tgz",
  pythonIndexUrl: "https://pypi.tuna.tsinghua.edu.cn/simple",
  npmRegistryUrl: "https://registry.npmmirror.com",
};

function settingsClient() {
  return {
    getHermesDownloadSources: vi.fn().mockResolvedValue(defaults),
    saveHermesDownloadSources: vi.fn().mockImplementation(async (sources: HermesDownloadSources) => sources),
    resetHermesDownloadSources: vi.fn().mockResolvedValue(defaults),
  } as unknown as DaemonClient;
}

describe("HermesDownloadSourcesPanel", () => {
  it("loads grouped settings and saves an edited Python mirror", async () => {
    const client = settingsClient();
    render(<HermesDownloadSourcesPanel copy={messages["zh-CN"]} client={client} />);

    const input = await screen.findByDisplayValue(defaults.pythonIndexUrl);
    fireEvent.change(input, { target: { value: "https://mirror.example/pypi/simple" } });
    fireEvent.click(screen.getByRole("button", { name: "保存更改" }));

    await waitFor(() => expect(client.saveHermesDownloadSources).toHaveBeenCalledWith({
      ...defaults,
      pythonIndexUrl: "https://mirror.example/pypi/simple",
    }));
    expect(await screen.findByText("下载源配置已保存。")).toBeInTheDocument();
  });

  it("rejects credential-bearing URLs before calling the daemon", async () => {
    const client = settingsClient();
    render(<HermesDownloadSourcesPanel copy={messages["en-US"]} client={client} />);
    const input = await screen.findByDisplayValue(defaults.hermesArchiveUrl);
    fireEvent.change(input, { target: { value: "https://user:secret@example.com/hermes.zip" } });
    fireEvent.submit(input.closest("form")!);

    expect(await screen.findByText(/credential-free HTTPS URL/i)).toBeInTheDocument();
    expect(client.saveHermesDownloadSources).not.toHaveBeenCalled();
  });

  it("restores the China defaults through the daemon", async () => {
    const client = settingsClient();
    render(<HermesDownloadSourcesPanel copy={messages["en-US"]} client={client} />);
    await screen.findByDisplayValue(defaults.pythonIndexUrl);
    fireEvent.click(screen.getByRole("button", { name: "Restore China defaults" }));
    await waitFor(() => expect(client.resetHermesDownloadSources).toHaveBeenCalledOnce());
  });
});

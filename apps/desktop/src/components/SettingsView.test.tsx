import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { messages } from "../i18n";
import { SettingsView } from "./SettingsView";

const preferenceMocks = vi.hoisted(() => ({
  get: vi.fn(),
  set: vi.fn(),
}));
const updateMocks = vi.hoisted(() => ({
  status: vi.fn(),
  check: vi.fn(),
  download: vi.fn(),
  cancel: vi.fn(),
  install: vi.fn(),
}));

vi.mock("../api/desktopPreferences", () => ({
  getDesktopPreferences: preferenceMocks.get,
  setDesktopPreferences: preferenceMocks.set,
}));

vi.mock("../api/updates", () => ({
  getYorvaUpdateStatus: updateMocks.status,
  checkYorvaUpdate: updateMocks.check,
  downloadYorvaUpdate: updateMocks.download,
  cancelYorvaUpdate: updateMocks.cancel,
  installYorvaUpdate: updateMocks.install,
  isYorvaUpdateError: (value: unknown) => typeof value === "object" && value !== null && "code" in value,
}));

describe("SettingsView", () => {
  beforeEach(() => {
    preferenceMocks.get.mockReset().mockResolvedValue({ launchOnLogin: true, closeToTray: true });
    preferenceMocks.set.mockReset().mockImplementation(async (preferences) => preferences);
    updateMocks.status.mockReset().mockResolvedValue({
      installedVersion: "0.4.0",
      phase: "IDLE",
      verificationKeyConfigured: false,
      metadataSource: "https://github.com/YoLin02/yorva/releases/latest/download/yorva-update.json",
      candidate: null,
      errorCode: null,
    });
    updateMocks.check.mockReset();
    updateMocks.download.mockReset();
    updateMocks.cancel.mockReset();
    updateMocks.install.mockReset();
  });

  it("switches from general settings to Hermes download sources", () => {
    render(<SettingsView copy={messages["en-US"]} locale="en-US" onLocaleChange={vi.fn()} />);

    expect(screen.getByRole("heading", { name: "Interface language" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Uninstall and local data" })).toBeInTheDocument();
    expect(screen.getByText(/preserves YORVA settings, encrypted backups/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("tab", { name: "Advanced" }));
    expect(screen.getByRole("tab", { name: "Advanced" })).toHaveAttribute("aria-selected", "true");
    expect(screen.queryByRole("heading", { name: "Interface language" })).not.toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Installation source priority" })).toBeInTheDocument();
  });

  it("changes the interface language from the segmented control", () => {
    const onLocaleChange = vi.fn();
    render(<SettingsView copy={messages["zh-CN"]} locale="zh-CN" onLocaleChange={onLocaleChange} />);

    fireEvent.click(screen.getByRole("radio", { name: "English" }));
    expect(onLocaleChange).toHaveBeenCalledWith("en-US");
  });

  it("updates the persisted close-to-tray preference", async () => {
    render(<SettingsView copy={messages["zh-CN"]} locale="zh-CN" onLocaleChange={vi.fn()} />);

    const toggle = await screen.findByRole("switch", { name: "关闭时最小化到托盘" });
    expect(toggle).toHaveAttribute("aria-checked", "true");
    fireEvent.click(toggle);
    expect(preferenceMocks.set).toHaveBeenCalledWith({ launchOnLogin: true, closeToTray: false });
  });

  it("opens update management as a separate settings view", async () => {
    render(<SettingsView copy={messages["zh-CN"]} locale="zh-CN" onLocaleChange={vi.fn()} />);

    fireEvent.click(screen.getByRole("tab", { name: "关于" }));
    expect(await screen.findByText("此构建暂不能验证更新")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /查看更新/ }));
    expect(await screen.findByRole("heading", { name: "YORVA 更新" })).toBeInTheDocument();
    expect(screen.getByText(/当前未签名内部候选没有获批的发布验证公钥/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /检查更新/ })).toBeDisabled();
    expect(screen.queryByRole("tab", { name: "关于" })).not.toBeInTheDocument();
  });

  it("downloads a verified candidate and advances to native install", async () => {
    const available = {
      installedVersion: "0.3.2",
      phase: "AVAILABLE",
      verificationKeyConfigured: true,
      metadataSource: "https://github.com/YoLin02/yorva/releases/latest/download/yorva-update.json",
      candidate: {
        version: "0.4.0",
        releaseNotes: "Verified update fixture.",
        publishedAtUtc: "2026-09-04T00:00:00.000Z",
        sizeBytes: 1048576,
        authenticodeRequired: false,
      },
      errorCode: null,
    } as const;
    updateMocks.status.mockResolvedValue(available);
    updateMocks.download.mockResolvedValue({ ...available, phase: "READY_TO_INSTALL" });
    updateMocks.install.mockResolvedValue({ ...available, phase: "INSTALLING" });

    render(<SettingsView copy={messages["en-US"]} locale="en-US" onLocaleChange={vi.fn()} />);
    fireEvent.click(screen.getByRole("tab", { name: "About" }));
    fireEvent.click(await screen.findByRole("button", { name: /View updates/ }));
    fireEvent.click(await screen.findByRole("button", { name: /Download and verify/ }));

    await waitFor(() => expect(updateMocks.download).toHaveBeenCalledTimes(1));
    fireEvent.click(await screen.findByRole("button", { name: /Install and restart/ }));
    await waitFor(() => expect(updateMocks.install).toHaveBeenCalledTimes(1));
  });
});

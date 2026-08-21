import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { messages } from "../i18n";
import { SettingsView } from "./SettingsView";

const preferenceMocks = vi.hoisted(() => ({
  get: vi.fn(),
  set: vi.fn(),
}));

vi.mock("../api/desktopPreferences", () => ({
  getDesktopPreferences: preferenceMocks.get,
  setDesktopPreferences: preferenceMocks.set,
}));

describe("SettingsView", () => {
  beforeEach(() => {
    preferenceMocks.get.mockReset().mockResolvedValue({ launchOnLogin: true, closeToTray: true });
    preferenceMocks.set.mockReset().mockImplementation(async (preferences) => preferences);
  });

  it("switches between clickable tabs and keeps unfinished panels empty", () => {
    render(<SettingsView copy={messages["en-US"]} locale="en-US" onLocaleChange={vi.fn()} />);

    expect(screen.getByRole("heading", { name: "Interface language" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("tab", { name: "Advanced" }));
    expect(screen.getByRole("tab", { name: "Advanced" })).toHaveAttribute("aria-selected", "true");
    expect(screen.queryByRole("heading", { name: "Interface language" })).not.toBeInTheDocument();
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
});

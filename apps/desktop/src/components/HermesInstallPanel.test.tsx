import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { messages } from "../i18n";
import { HermesInstallPanel } from "./HermesInstallPanel";

const noop = () => undefined;

describe("HermesInstallPanel", () => {
  it("requires explicit confirmation before starting install", () => {
    const onConfirm = vi.fn();
    const onOpen = vi.fn();
    const { rerender } = render(
      <HermesInstallPanel
        copy={messages["en-US"]}
        windowsHost
        confirmOpen={false}
        busy={false}
        operation={null}
        onOpenConfirm={onOpen}
        onCloseConfirm={noop}
        onConfirm={onConfirm}
        onCancel={noop}
        onRetry={noop}
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: "Install Hermes" }));
    expect(onOpen).toHaveBeenCalled();
    expect(onConfirm).not.toHaveBeenCalled();

    rerender(
      <HermesInstallPanel
        copy={messages["zh-CN"]}
        windowsHost
        confirmOpen
        busy={false}
        operation={null}
        onOpenConfirm={onOpen}
        onCloseConfirm={noop}
        onConfirm={onConfirm}
        onCancel={noop}
        onRetry={noop}
      />,
    );
    expect(screen.getByText("安装官方 Hermes")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "安装" }));
    expect(onConfirm).toHaveBeenCalledTimes(1);
  });
});

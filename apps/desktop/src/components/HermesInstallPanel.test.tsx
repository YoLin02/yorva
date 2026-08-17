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
    expect(screen.getByText("已准备内置源码；依赖安装仍可能需要网络。")).toBeInTheDocument();
    expect(screen.queryByText(/offline installation/i)).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "安装" }));
    expect(onConfirm).toHaveBeenCalledTimes(1);
  });

  it("renders the bundled source sentence in English and never claims offline install", () => {
    render(
      <HermesInstallPanel
        copy={messages["en-US"]}
        windowsHost
        confirmOpen
        busy={false}
        operation={null}
        onOpenConfirm={noop}
        onCloseConfirm={noop}
        onConfirm={noop}
        onCancel={noop}
        onRetry={noop}
      />,
    );
    expect(screen.getByText("Bundled source prepared; dependencies may still require network.")).toBeInTheDocument();
    expect(screen.queryByText(/offline installation/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/gitclone|kkgithub|Gitee/i)).not.toBeInTheDocument();
  });

  it("shows failed stage, error code, correlation and log path", () => {
    render(
      <HermesInstallPanel
        copy={messages["zh-CN"]}
        windowsHost
        confirmOpen={false}
        busy={false}
        operation={{
          id: "op_test",
          type: "runtime.install",
          targetType: "runtime-kind",
          targetId: "hermes",
          status: "FAILED",
          stage: "install.repository",
          progress: null,
          message: "HERMES_SOURCE_PREPARED",
          errorCode: "RUNTIME_INSTALL_STAGE_FAILED",
          retryable: false,
          correlationId: "cor_Mt9dEyPUgM5b9MGD",
          createdAt: "2026-08-17T02:00:00Z",
          startedAt: "2026-08-17T02:00:01Z",
          completedAt: "2026-08-17T02:10:00Z",
          updatedAt: "2026-08-17T02:10:00Z",
        }}
        onOpenConfirm={noop}
        onCloseConfirm={noop}
        onConfirm={noop}
        onCancel={noop}
        onRetry={noop}
      />,
    );
    expect(screen.getByText("已准备内置源码；依赖安装仍可能需要网络。")).toBeInTheDocument();
    expect(screen.getAllByText(/install\.repository/).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/RUNTIME_INSTALL_STAGE_FAILED/).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/cor_Mt9dEyPUgM5b9MGD/).length).toBeGreaterThan(0);
    expect(screen.getByRole("button", { name: "复制日志" })).toBeInTheDocument();
    expect(screen.getByText(/"event":"failed"/)).toBeInTheDocument();
    expect(screen.getByText(/install\.ndjson/)).toBeInTheDocument();
  });
});

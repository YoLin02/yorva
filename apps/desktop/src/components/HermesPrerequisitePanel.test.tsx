import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { messages } from "../i18n";
import { HermesPrerequisitePanel } from "./HermesPrerequisitePanel";

describe("HermesPrerequisitePanel", () => {
  it("shows Node health when Hermes is already supported", () => {
    const onInstall = vi.fn();
    render(
      <HermesPrerequisitePanel
        copy={messages["zh-CN"]}
        status={{
          node: { state: "UNSUPPORTED", version: "22.23.1", errorCode: "RUNTIME_HERMES_NPM_UNSUPPORTED", retryable: true },
          npm: { state: "UNSUPPORTED", version: "10.9.8", errorCode: "RUNTIME_HERMES_NPM_UNSUPPORTED", retryable: true },
          nodeDependencies: { state: "NOT_INSTALLED", version: "", errorCode: null, retryable: true },
          checkedAt: "2026-08-17T00:00:00Z",
          activeOperationId: null,
        }}
        operation={null}
        busy={false}
        onInstall={onInstall}
        onRetryDeps={vi.fn()}
        onCancel={vi.fn()}
      />,
    );
    expect(screen.getByText("Node.js / npm 组件")).toBeInTheDocument();
    expect(screen.getByText("Node.js 版本不受支持")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "重新安装 Node.js / npm" }));
    expect(onInstall).toHaveBeenCalledTimes(1);
  });

  it("uses English copy for missing Node", () => {
    render(
      <HermesPrerequisitePanel
        copy={messages["en-US"]}
        status={{
          node: { state: "MISSING", version: "", errorCode: "RUNTIME_HERMES_NODE_MISSING", retryable: true },
          npm: { state: "MISSING", version: "", errorCode: null, retryable: true },
          nodeDependencies: { state: "NOT_INSTALLED", version: "", errorCode: null, retryable: true },
          checkedAt: "2026-08-17T00:00:00Z",
          activeOperationId: null,
        }}
        operation={null}
        busy={false}
        onInstall={vi.fn()}
        onRetryDeps={vi.fn()}
        onCancel={vi.fn()}
      />,
    );
    expect(screen.getByText("Node.js was not detected")).toBeInTheDocument();
    expect(screen.queryByText(/offline installation/i)).not.toBeInTheDocument();
  });
});

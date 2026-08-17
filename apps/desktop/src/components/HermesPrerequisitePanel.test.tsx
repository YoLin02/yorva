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

  it("shows a request rejection instead of leaving the install button idle", () => {
    const onRetry = vi.fn();
    render(
      <HermesPrerequisitePanel
        copy={messages["zh-CN"]}
        status={{
          node: { state: "MISSING", version: "", errorCode: "RUNTIME_HERMES_NODE_MISSING", retryable: true },
          npm: { state: "MISSING", version: "", errorCode: null, retryable: true },
          nodeDependencies: { state: "NOT_INSTALLED", version: "", errorCode: null, retryable: true },
          checkedAt: "2026-08-17T00:00:00Z",
          activeOperationId: null,
        }}
        operation={null}
        busy={false}
        requestError={{
          code: "NOT_FOUND",
          message: "The requested resource was not found.",
          retryable: false,
        }}
        onInstall={vi.fn()}
        onRetryDeps={onRetry}
        onCancel={vi.fn()}
      />,
    );
    expect(screen.getByRole("alert")).toHaveTextContent("Node.js / npm 安装失败");
    expect(screen.getByText(/NOT_FOUND/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "重新安装 Node.js / npm" })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "重试 Node.js / npm 安装" }));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it("renders a failed prerequisite Operation so a quick failure is visible", () => {
    render(
      <HermesPrerequisitePanel
        copy={messages["en-US"]}
        status={{
          node: { state: "MISSING", version: "", errorCode: "RUNTIME_HERMES_NODE_MISSING", retryable: true },
          npm: { state: "MISSING", version: "", errorCode: null, retryable: true },
          nodeDependencies: { state: "NOT_INSTALLED", version: "", errorCode: null, retryable: true },
          checkedAt: "2026-08-17T00:00:00Z",
          activeOperationId: "op_prereq",
        }}
        operation={{
          id: "op_prereq",
          type: "hermes.prerequisites",
          targetType: "runtime-kind",
          targetId: "hermes",
          status: "FAILED",
          stage: "install.node",
          progress: null,
          message: "",
          errorCode: "RUNTIME_HERMES_NODE_MISSING",
          retryable: true,
          correlationId: "cor_prereq_test",
          createdAt: "2026-08-17T02:00:00Z",
          startedAt: "2026-08-17T02:00:01Z",
          completedAt: "2026-08-17T02:00:02Z",
          updatedAt: "2026-08-17T02:00:02Z",
        }}
        liveLog="node archive was not available"
        busy={false}
        onInstall={vi.fn()}
        onRetryDeps={vi.fn()}
        onCancel={vi.fn()}
      />,
    );
    expect(screen.getByRole("alert")).toHaveTextContent("Node.js / npm installation failed");
    expect(screen.getByText(/Installing Node.js/)).toBeInTheDocument();
    expect(screen.getByText(/RUNTIME_HERMES_NODE_MISSING/)).toBeInTheDocument();
    expect(screen.getByText(/cor_prereq_test/)).toBeInTheDocument();
    expect(screen.getByText("node archive was not available")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Retry Node.js / npm installation" })).toBeInTheDocument();
  });

  it("tells the user they can cancel a wait and install Hermes next", () => {
    render(
      <HermesPrerequisitePanel
        copy={messages["zh-CN"]}
        status={{
          node: { state: "MISSING", version: "", errorCode: "RUNTIME_HERMES_NODE_MISSING", retryable: true },
          npm: { state: "MISSING", version: "", errorCode: null, retryable: true },
          nodeDependencies: { state: "NOT_INSTALLED", version: "", errorCode: null, retryable: true },
          checkedAt: "2026-08-17T00:00:00Z",
          activeOperationId: "op_prereq",
        }}
        operation={{
          id: "op_prereq",
          type: "hermes.prerequisites",
          targetType: "runtime-kind",
          targetId: "hermes",
          status: "RUNNING",
          stage: "install.node",
          progress: null,
          message: "",
          errorCode: null,
          retryable: true,
          correlationId: "cor_wait",
          createdAt: "2026-08-17T02:00:00Z",
          startedAt: "2026-08-17T02:00:01Z",
          completedAt: null,
          updatedAt: "2026-08-17T02:00:01Z",
        }}
        busy={false}
        hermesNotInstalled
        onInstall={vi.fn()}
        onRetryDeps={vi.fn()}
        onCancel={vi.fn()}
      />,
    );
    expect(screen.getByText(/可以先安装 Node\.js \/ npm/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "取消" })).toBeInTheDocument();
    expect(screen.getByText(/Ctrl\+C/)).toBeInTheDocument();
  });
});

import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { RuntimeDiscovery, RuntimeDiscoveryState } from "../api/types";
import { messages } from "../i18n";
import { HermesDiscoveryView } from "./HermesDiscoveryView";

const copy = messages["en-US"];

function discovery(state: RuntimeDiscoveryState): RuntimeDiscovery {
  const usable = state === "SUPPORTED" || state === "UNSUPPORTED";
  const candidate = {
    path: "C:\\Hermes\\hermes.exe",
    version: usable ? "0.19.3" : "",
    state: state === "AMBIGUOUS" || state === "NOT_INSTALLED" ? "SUPPORTED" as const : state,
    errorCode: state === "SUPPORTED" ? null : "RUNTIME_UNSUPPORTED" as const,
  };
  return {
    runtimeKind: "hermes",
    state,
    errorCode: state === "SUPPORTED" ? null : "RUNTIME_UNSUPPORTED",
    selected: usable ? candidate : null,
    candidates: state === "NOT_INSTALLED" ? [] : [candidate],
    warnings: [],
    detectedAt: "2026-08-14T00:00:00Z",
    supportedRange: "=0.20.2",
  };
}

describe("HermesDiscoveryView", () => {
  it("renders checking and allows cancellation", () => {
    const onCancel = vi.fn();
    render(<HermesDiscoveryView state={{ kind: "checking", onCancel }} copy={copy} locale="en-US" />);
    expect(screen.getByRole("status")).toHaveTextContent("Checking Hermes");
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(onCancel).toHaveBeenCalledOnce();
  });

  it.each([
    ["NOT_INSTALLED", "Hermes not installed"],
    ["SUPPORTED", "Hermes ready"],
    ["UNSUPPORTED", "Hermes version unsupported"],
    ["BROKEN_EXECUTABLE", "Hermes installation is incomplete"],
    ["MALFORMED_VERSION", "Hermes version is unreadable"],
    ["TIMED_OUT", "Hermes check timed out"],
    ["AMBIGUOUS", "Multiple Hermes executables found"],
  ] as const)("renders %s without offering installation", (state, label) => {
    const { container } = render(
      <HermesDiscoveryView state={{ kind: "complete", discovery: discovery(state), onRetry: vi.fn() }} copy={copy} locale="en-US" />,
    );
    const view = within(container);
    expect(view.getAllByText(label).length).toBeGreaterThan(0);
    expect(view.getByRole("button", { name: "Check again" })).toBeInTheDocument();
    expect(view.queryByRole("button", { name: /install/i })).not.toBeInTheDocument();
  });

  it.each([
    ["cancelled", "The Hermes check was cancelled"],
    ["failure", "Yorva could not complete Hermes discovery"],
  ] as const)("renders retry for the %s request state", (kind, message) => {
    const onRetry = vi.fn();
    const { container } = render(<HermesDiscoveryView state={{ kind, onRetry }} copy={copy} locale="en-US" />);
    const view = within(container);
    expect(view.getByRole("alert")).toHaveTextContent(message);
    fireEvent.click(view.getByRole("button", { name: "Retry" }));
    expect(onRetry).toHaveBeenCalledOnce();
  });

  it("renders executable metadata as text", () => {
    const unsafe = discovery("SUPPORTED");
    unsafe.selected = { ...unsafe.selected!, path: "<script>unsafe()</script>" };
    const { container } = render(
      <HermesDiscoveryView state={{ kind: "complete", discovery: unsafe, onRetry: vi.fn() }} copy={copy} locale="en-US" />,
    );
    expect(screen.getByText("<script>unsafe()</script>")).toHaveAttribute("title", "<script>unsafe()</script>");
    expect(container.querySelector("script")).toBeNull();
  });

  it("renders the real managed instance count and opens the instance inventory", () => {
    const onOpenInstances = vi.fn();
    render(
      <HermesDiscoveryView
        state={{ kind: "complete", discovery: discovery("SUPPORTED"), onRetry: vi.fn() }}
        copy={copy}
        locale="en-US"
        instanceCount={3}
        onOpenInstances={onOpenInstances}
      />,
    );
    expect(screen.getByText("Managed instances")).toBeInTheDocument();
    expect(screen.getByText("3")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /view instances/i }));
    expect(onOpenInstances).toHaveBeenCalledOnce();
  });

  it("shows one compatibility state and omits the support range from the runtime card", () => {
    render(
      <HermesDiscoveryView
        state={{ kind: "complete", discovery: discovery("SUPPORTED"), onRetry: vi.fn() }}
        copy={copy}
        locale="en-US"
      />,
    );
    expect(screen.getAllByText("Hermes ready")).toHaveLength(1);
    expect(screen.getByText(copy.hermes.summaryDescription)).toBeInTheDocument();
    expect(screen.queryByText(copy.hermes.states.SUPPORTED.description)).not.toBeInTheDocument();
    expect(screen.queryByText("Supported range")).not.toBeInTheDocument();
    expect(screen.getByText("Version")).toBeInTheDocument();
    expect(screen.getByText("Last checked")).toBeInTheDocument();
  });

  it("shows an ambiguous candidate count and list without selected details", () => {
    const result = discovery("AMBIGUOUS");
    result.candidates = [
      { ...result.candidates[0], path: "C:\\First\\hermes.exe", version: "0.19.1" },
      { ...result.candidates[0], path: "C:\\Second\\hermes.exe", version: "0.19.2" },
    ];
    const { container } = render(
      <HermesDiscoveryView state={{ kind: "complete", discovery: result, onRetry: vi.fn() }} copy={copy} locale="en-US" />,
    );
    const view = within(container);
    expect(view.getByText("Candidates found: 2")).toBeInTheDocument();
    expect(view.getByText("C:\\First\\hermes.exe")).toBeInTheDocument();
    expect(view.getByText("C:\\Second\\hermes.exe")).toBeInTheDocument();
    expect(view.queryByText("Executable")).not.toBeInTheDocument();
    expect(view.queryByText("Version")).not.toBeInTheDocument();
    expect(view.queryByText("0.19.1")).not.toBeInTheDocument();
    expect(view.queryByText("0.19.2")).not.toBeInTheDocument();
  });

  it("distinguishes not installed, incomplete installation, and unsupported in Chinese", () => {
    const chinese = messages["zh-CN"];
    const { rerender } = render(
      <HermesDiscoveryView state={{ kind: "complete", discovery: discovery("NOT_INSTALLED"), onRetry: vi.fn() }} copy={chinese} locale="zh-CN" />,
    );
    expect(screen.getAllByText("未检测到 Hermes").length).toBeGreaterThan(0);

    rerender(<HermesDiscoveryView state={{ kind: "complete", discovery: discovery("BROKEN_EXECUTABLE"), onRetry: vi.fn() }} copy={chinese} locale="zh-CN" />);
    expect(screen.getAllByText("Hermes 安装不完整").length).toBeGreaterThan(0);
    expect(screen.queryAllByText("未检测到 Hermes")).toHaveLength(0);

    rerender(<HermesDiscoveryView state={{ kind: "complete", discovery: discovery("UNSUPPORTED"), onRetry: vi.fn() }} copy={chinese} locale="zh-CN" />);
    expect(screen.getAllByText("Hermes 版本不受支持").length).toBeGreaterThan(0);
  });

  it("maps warnings by stable code without rendering daemon message text", () => {
    const result = discovery("BROKEN_EXECUTABLE");
    result.warnings = [{ code: "HERMES_CLI_LAUNCHER_MISSING", message: "raw daemon wording" }];
    render(<HermesDiscoveryView state={{ kind: "complete", discovery: result, onRetry: vi.fn() }} copy={copy} locale="en-US" />);
    expect(screen.getByText("The Hermes installation does not contain a safe CLI launcher.")).toBeInTheDocument();
    expect(screen.queryByText("raw daemon wording")).not.toBeInTheDocument();
  });
});

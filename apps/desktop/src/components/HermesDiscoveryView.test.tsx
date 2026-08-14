import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { RuntimeDiscovery, RuntimeDiscoveryState } from "../api/types";
import { HermesDiscoveryView } from "./HermesDiscoveryView";

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
    supportedRange: ">=0.19.0 <0.20.0",
  };
}

describe("HermesDiscoveryView", () => {
  it("renders checking and allows cancellation", () => {
    const onCancel = vi.fn();
    render(<HermesDiscoveryView state={{ kind: "checking", onCancel }} />);
    expect(screen.getByRole("status")).toHaveTextContent("Checking Hermes");
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(onCancel).toHaveBeenCalledOnce();
  });

  it.each([
    ["NOT_INSTALLED", "Hermes not installed"],
    ["SUPPORTED", "Hermes ready"],
    ["UNSUPPORTED", "Hermes version unsupported"],
    ["BROKEN_EXECUTABLE", "Hermes executable is broken"],
    ["MALFORMED_VERSION", "Hermes version is unreadable"],
    ["TIMED_OUT", "Hermes check timed out"],
    ["AMBIGUOUS", "Multiple Hermes executables found"],
  ] as const)("renders %s without offering installation", (state, label) => {
    const { container } = render(
      <HermesDiscoveryView state={{ kind: "complete", discovery: discovery(state), onRetry: vi.fn() }} />,
    );
    const view = within(container);
    expect(view.getByText(label)).toBeInTheDocument();
    expect(view.getByRole("button", { name: "Check again" })).toBeInTheDocument();
    expect(view.queryByRole("button", { name: /install/i })).not.toBeInTheDocument();
  });

  it.each([
    ["cancelled", "The Hermes check was cancelled"],
    ["failure", "YORVA could not complete Hermes discovery"],
  ] as const)("renders retry for the %s request state", (kind, message) => {
    const onRetry = vi.fn();
    const { container } = render(<HermesDiscoveryView state={{ kind, onRetry }} />);
    const view = within(container);
    expect(view.getByRole("alert")).toHaveTextContent(message);
    fireEvent.click(view.getByRole("button", { name: "Retry" }));
    expect(onRetry).toHaveBeenCalledOnce();
  });

  it("renders executable metadata as text", () => {
    const unsafe = discovery("SUPPORTED");
    unsafe.selected = { ...unsafe.selected!, path: "<script>unsafe()</script>" };
    const { container } = render(
      <HermesDiscoveryView state={{ kind: "complete", discovery: unsafe, onRetry: vi.fn() }} />,
    );
    expect(screen.getByText("<script>unsafe()</script>")).toBeInTheDocument();
    expect(container.querySelector("script")).toBeNull();
  });
});

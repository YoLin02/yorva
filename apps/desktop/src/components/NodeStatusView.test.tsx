import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { NodeStatusView } from "./NodeStatusView";

const node = {
  id: "node_test",
  name: "DESKTOP-TEST",
  hostname: "DESKTOP-TEST",
  platform: "windows",
  architecture: "amd64",
  nodeVersion: "0.0.0-test",
  createdAt: "2026-01-01T00:00:00Z",
  updatedAt: "2026-01-01T00:00:00Z",
};

describe("NodeStatusView", () => {
  it("renders the starting state", () => {
    render(<NodeStatusView state={{ kind: "starting" }} />);
    expect(screen.getByRole("status")).toHaveTextContent("Starting local node");
  });

  it("renders the connected Node state", () => {
    render(<NodeStatusView state={{ kind: "connected", node, eventStatus: "connected" }} />);
    expect(screen.getByText("DESKTOP-TEST")).toBeInTheDocument();
    expect(screen.getByText("node_test")).toBeInTheDocument();
    expect(screen.getByText("connected")).toBeInTheDocument();
  });

  it("renders the failure state", () => {
    render(<NodeStatusView state={{ kind: "failure", message: "The local Node could not be reached." }} />);
    expect(screen.getByRole("alert")).toHaveTextContent("The local Node could not be reached.");
  });

  it("renders Node metadata as text", () => {
    const unsafeNode = { ...node, name: "<script>unsafe()</script>" };
    const { container } = render(
      <NodeStatusView state={{ kind: "connected", node: unsafeNode, eventStatus: "connected" }} />,
    );
    expect(screen.getByText("<script>unsafe()</script>")).toBeInTheDocument();
    expect(container.querySelector("script")).toBeNull();
  });
});

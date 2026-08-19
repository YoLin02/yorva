import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { InstanceList } from "../api/types";
import { messages } from "../i18n";
import { InstancesPage } from "./InstancesPage";

const inventory: InstanceList = {
  runtimeId: "hermes",
  runtimeInstallationId: "rtinst_test",
  freshness: "FRESH",
  lastSyncedAt: "2026-08-19T12:00:00Z",
  instances: [
    {
      instanceId: "inst_default",
      runtimeInstallationId: "rtinst_test",
      name: "default",
      default: true,
      protected: true,
      availability: "AVAILABLE",
      lastSyncedAt: "2026-08-19T12:00:00Z",
      createdAt: "2026-08-19T12:00:00Z",
      updatedAt: "2026-08-19T12:00:00Z",
      capabilities: { instances: true, lifecycle: false },
    },
  ],
  capabilities: { instances: true, lifecycle: false },
  errorCode: null,
};

describe("InstancesPage", () => {
  it("explains unsupported discovery without fake lifecycle controls", () => {
    render(
      <InstancesPage
        supported={false}
        loading={false}
        error={false}
        inventory={null}
        createName=""
        createBusy={false}
        createOperation={null}
        copy={messages["en-US"]}
        locale="en-US"
        onRefresh={() => undefined}
        onCreateNameChange={() => undefined}
        onCreate={() => undefined}
        onCancelCreate={() => undefined}
      />,
    );
    expect(screen.getByText(messages["en-US"].instances.unsupportedTitle)).toBeInTheDocument();
    expect(screen.getByText(messages["en-US"].instances.unsupportedDescription)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Start" })).not.toBeInTheDocument();
  });

  it("shows default protected availability and empty named state", () => {
    const onRefresh = vi.fn();
    render(
      <InstancesPage
        supported
        loading={false}
        error={false}
        inventory={inventory}
        createName=""
        createBusy={false}
        createOperation={null}
        copy={messages["en-US"]}
        locale="en-US"
        onRefresh={onRefresh}
        onCreateNameChange={() => undefined}
        onCreate={() => undefined}
        onCancelCreate={() => undefined}
      />,
    );
    expect(screen.getByText("default")).toBeInTheDocument();
    expect(screen.getByText("Protected")).toBeInTheDocument();
    expect(screen.getByText("Available")).toBeInTheDocument();
    expect(screen.getByText(messages["en-US"].instances.emptyNamed)).toBeInTheDocument();
    expect(screen.getByText(messages["en-US"].instances.lifecycleUnavailable)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Refresh" }));
    expect(onRefresh).toHaveBeenCalled();
  });

  it("keeps Chinese copy for unknown freshness", () => {
    render(
      <InstancesPage
        supported
        loading={false}
        error={false}
        inventory={{ ...inventory, freshness: "UNKNOWN" }}
        createName=""
        createBusy={false}
        createOperation={null}
        copy={messages["zh-CN"]}
        locale="zh-CN"
        onRefresh={() => undefined}
        onCreateNameChange={() => undefined}
        onCreate={() => undefined}
        onCancelCreate={() => undefined}
      />,
    );
    expect(screen.getByText(messages["zh-CN"].instances.freshnessUnknown)).toBeInTheDocument();
    expect(screen.getByText("受保护")).toBeInTheDocument();
  });
});

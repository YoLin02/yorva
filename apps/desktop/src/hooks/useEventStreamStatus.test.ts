import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { DaemonClient, StreamEvent } from "../api/client";
import { useEventStreamStatus } from "./useEventStreamStatus";

afterEach(() => {
  vi.useRealTimers();
});

describe("useEventStreamStatus", () => {
  it("invokes onEvent and recovers through onReady after reconnect", async () => {
    const events: StreamEvent[] = [];
    const ready: number[] = [];
    let rejectConnect: ((reason?: unknown) => void) | undefined;
    const connectEvents = vi.fn(async (options: {
      signal: AbortSignal;
      onOpen: () => void;
      onEvent?: (event: StreamEvent) => void;
    }) => {
      options.onOpen();
      options.onEvent?.({ type: "operation.progress", data: { operationId: "op_1", status: "RUNNING" } });
      await new Promise<void>((_resolve, reject) => {
        rejectConnect = reject;
        options.signal.addEventListener("abort", () => reject(new DOMException("aborted", "AbortError")));
      });
    });
    const client = { connectEvents } as unknown as DaemonClient;
    const { result, unmount } = renderHook(() =>
      useEventStreamStatus(
        client,
        (event) => events.push(event),
        () => ready.push(Date.now()),
      ),
    );

    await waitFor(() => expect(result.current).toBe("connected"));
    expect(events).toEqual([{ type: "operation.progress", data: { operationId: "op_1", status: "RUNNING" } }]);
    expect(ready).toHaveLength(1);

    await act(async () => {
      rejectConnect?.(new Error("disconnected"));
    });
    await waitFor(() => expect(result.current).toBe("disconnected"));

    unmount();
  });
});

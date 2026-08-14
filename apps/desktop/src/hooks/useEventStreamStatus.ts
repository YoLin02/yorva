import { useEffect, useState } from "react";
import type { DaemonClient } from "../api/client";

export type EventStreamStatus = "idle" | "connecting" | "connected" | "disconnected";

export function useEventStreamStatus(client: DaemonClient | undefined): EventStreamStatus {
  const [status, setStatus] = useState<EventStreamStatus>("idle");

  useEffect(() => {
    if (!client) {
      return;
    }
    const controller = new AbortController();
    void client
      .connectEvents({
        signal: controller.signal,
        onOpen: () => setStatus("connected"),
      })
      .then(() => {
        if (!controller.signal.aborted) setStatus("disconnected");
      })
      .catch(() => {
        if (!controller.signal.aborted) setStatus("disconnected");
      });
    return () => controller.abort();
  }, [client]);

  return status;
}

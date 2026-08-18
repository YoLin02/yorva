import { useEffect, useRef, useState } from "react";
import type { DaemonClient, StreamEvent } from "../api/client";

export type EventStreamStatus = "idle" | "connecting" | "connected" | "disconnected";

export function useEventStreamStatus(
  client: DaemonClient | undefined,
  onEvent?: (event: StreamEvent) => void,
  onReady?: () => void,
): EventStreamStatus {
  const [status, setStatus] = useState<EventStreamStatus>("idle");
  const onEventRef = useRef(onEvent);
  const onReadyRef = useRef(onReady);

  useEffect(() => {
    onEventRef.current = onEvent;
    onReadyRef.current = onReady;
  }, [onEvent, onReady]);

  useEffect(() => {
    if (!client) {
      return;
    }
    let cancelled = false;
    let timer: ReturnType<typeof setTimeout> | undefined;
    let controller: AbortController | undefined;
    const connect = () => {
      if (cancelled) {
        return;
      }
      controller = new AbortController();
      setStatus("connecting");
      void client
        .connectEvents({
          signal: controller.signal,
          onOpen: () => {
            setStatus("connected");
            onReadyRef.current?.();
          },
          onEvent: (event) => onEventRef.current?.(event),
        })
        .then(() => {
          if (cancelled) return;
          setStatus("disconnected");
          timer = setTimeout(connect, 1000);
        })
        .catch(() => {
          if (cancelled) return;
          setStatus("disconnected");
          timer = setTimeout(connect, 1000);
        });
    };
    connect();
    return () => {
      cancelled = true;
      if (timer) clearTimeout(timer);
      controller?.abort();
    };
  }, [client]);

  return status;
}

import type { DaemonSession, ErrorResponse, Health, Node, Operation, OperationList, RuntimeDiscovery } from "./types";

export class YorvaApiError extends Error {
  readonly code: string;
  readonly retryable: boolean;
  readonly details: Record<string, unknown>;

  constructor(error: ErrorResponse["error"]) {
    super(error.message);
    this.name = "YorvaApiError";
    this.code = error.code;
    this.retryable = error.retryable;
    this.details = error.details;
  }
}

export type StreamEvent = {
  id?: string;
  type?: string;
  data: unknown;
};

export type DaemonClient = ReturnType<typeof createDaemonClient>;

const desktopDiscoveryTimeoutMs = 12_000;

export function createDaemonClient(session: DaemonSession) {
  async function request<T>(path: string, init: RequestInit = {}, authenticated = true): Promise<T> {
    const response = await fetch(`${session.baseUrl}${path}`, {
      ...init,
      headers: {
        Accept: "application/json",
        ...(authenticated ? { Authorization: `Bearer ${session.token}` } : {}),
        ...init.headers,
      },
    });
    if (!response.ok) {
      throw await decodeError(response);
    }
    return (await response.json()) as T;
  }

  async function connectEvents(options: {
    signal: AbortSignal;
    onOpen: () => void;
    onEvent?: (event: StreamEvent) => void;
  }): Promise<void> {
    const response = await fetch(`${session.baseUrl}/api/v1/events`, {
      headers: {
        Accept: "text/event-stream",
        Authorization: `Bearer ${session.token}`,
      },
      signal: options.signal,
    });
    if (!response.ok) {
      throw await decodeError(response);
    }
    if (!response.body) {
      throw new Error("The event stream response did not contain a body.");
    }

    options.onOpen();
    const reader = response.body.pipeThrough(new TextDecoderStream()).getReader();
    let buffer = "";
    while (true) {
      const { value, done } = await reader.read();
      if (done) return;
      buffer += value;
      let boundary = buffer.indexOf("\n\n");
      while (boundary >= 0) {
        const frame = buffer.slice(0, boundary);
        buffer = buffer.slice(boundary + 2);
        const event = parseEvent(frame);
        if (event) options.onEvent?.(event);
        boundary = buffer.indexOf("\n\n");
      }
    }
  }

  return {
    getHealth: (signal?: AbortSignal) => request<Health>("/api/v1/health", { signal }, false),
    getNode: (signal?: AbortSignal) => request<Node>("/api/v1/node", { signal }),
    detectHermes: (signal?: AbortSignal) =>
      request<RuntimeDiscovery>("/api/v1/runtimes/hermes/detect", {
        method: "POST",
        signal: signal
          ? AbortSignal.any([signal, AbortSignal.timeout(desktopDiscoveryTimeoutMs)])
          : AbortSignal.timeout(desktopDiscoveryTimeoutMs),
      }),
    getHermesPrerequisites: (signal?: AbortSignal) =>
      request<import("./types").HermesPrerequisites>("/api/v1/runtimes/hermes/prerequisites", { signal }),
    startHermesPrerequisites: (idempotencyKey: string, signal?: AbortSignal) =>
      request<Operation>("/api/v1/runtimes/hermes/prerequisites/install", {
        method: "POST",
        signal,
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": idempotencyKey,
        },
        body: "{}",
      }),
    startHermesInstall: (idempotencyKey: string, signal?: AbortSignal) =>
      request<Operation>("/api/v1/runtimes/hermes/install", {
        method: "POST",
        signal,
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": idempotencyKey,
        },
        body: "{}",
      }),
    getOperation: (operationId: string, signal?: AbortSignal) =>
      request<Operation>(`/api/v1/operations/${encodeURIComponent(operationId)}`, { signal }),
    getOperationLog: (operationId: string, signal?: AbortSignal) =>
      request<{ operationId: string; correlationId: string; text: string }>(
        `/api/v1/operations/${encodeURIComponent(operationId)}/log`,
        { signal },
      ),
    listOperations: (targetType: string, targetId: string, signal?: AbortSignal) =>
      request<OperationList>(
        `/api/v1/operations?targetType=${encodeURIComponent(targetType)}&targetId=${encodeURIComponent(targetId)}&limit=5`,
        { signal },
      ),
    cancelOperation: (operationId: string, signal?: AbortSignal) =>
      request<Operation>(`/api/v1/operations/${encodeURIComponent(operationId)}/cancel`, {
        method: "POST",
        signal,
      }),
    connectEvents,
  };
}

async function decodeError(response: Response): Promise<Error> {
  try {
    const body = (await response.json()) as ErrorResponse;
    if (body.error && typeof body.error.code === "string") {
      return new YorvaApiError(body.error);
    }
  } catch {
    // The transport fallback below intentionally avoids exposing raw response bodies.
  }
  return new Error(`The local daemon request failed with HTTP ${response.status}.`);
}

function parseEvent(frame: string): StreamEvent | undefined {
  if (frame === "" || frame.startsWith(":")) return undefined;
  let id: string | undefined;
  let type: string | undefined;
  const data: string[] = [];
  for (const line of frame.split("\n")) {
    if (line.startsWith("id:")) id = line.slice(3).trimStart();
    if (line.startsWith("event:")) type = line.slice(6).trimStart();
    if (line.startsWith("data:")) data.push(line.slice(5).trimStart());
  }
  if (data.length === 0) return undefined;
  const text = data.join("\n");
  try {
    return { id, type, data: JSON.parse(text) as unknown };
  } catch {
    return { id, type, data: text };
  }
}

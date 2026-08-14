import type { Node } from "../api/types";
import type { EventStreamStatus } from "../hooks/useEventStreamStatus";
import { APP_NAME } from "../appMetadata";

export type NodeScreenState =
  | { kind: "starting" }
  | { kind: "failure"; message: string }
  | { kind: "connected"; node: Node; eventStatus: EventStreamStatus };

export function NodeStatusView({ state }: { state: NodeScreenState }) {
  return (
    <main className="shell">
      <section className="intro" aria-labelledby="app-title">
        <p className="eyebrow">LOCAL-FIRST AI RUNTIME CONTROL</p>
        <h1 id="app-title">{APP_NAME}</h1>
        {state.kind === "starting" && (
          <div className="status-card" role="status">
            <StatusLabel tone="pending">Starting local node</StatusLabel>
            <p>Creating a private Desktop session and checking the local daemon.</p>
          </div>
        )}
        {state.kind === "failure" && (
          <div className="status-card" role="alert">
            <StatusLabel tone="error">Connection unavailable</StatusLabel>
            <p>{state.message}</p>
          </div>
        )}
        {state.kind === "connected" && (
          <div className="status-card">
            <StatusLabel tone="ready">Local node connected</StatusLabel>
            <dl className="node-details">
              <div><dt>Node name</dt><dd>{state.node.name}</dd></div>
              <div><dt>Node ID</dt><dd>{state.node.id}</dd></div>
              <div><dt>Platform</dt><dd>{state.node.platform} / {state.node.architecture}</dd></div>
              <div><dt>yorvad</dt><dd>{state.node.nodeVersion}</dd></div>
              <div><dt>Events</dt><dd>{state.eventStatus}</dd></div>
            </dl>
          </div>
        )}
      </section>
    </main>
  );
}

function StatusLabel({ children, tone }: { children: string; tone: "pending" | "ready" | "error" }) {
  return (
    <div className={`status status-${tone}`}>
      <span className="status-dot" aria-hidden="true" />
      {children}
    </div>
  );
}

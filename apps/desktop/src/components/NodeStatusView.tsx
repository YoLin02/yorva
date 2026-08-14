import type { Node } from "../api/types";
import type { EventStreamStatus } from "../hooks/useEventStreamStatus";
import type { AppMessages } from "../i18n";

export type NodeScreenState =
  | { kind: "starting" }
  | { kind: "failure"; message: string }
  | { kind: "connected"; node: Node; eventStatus: EventStreamStatus };

export function NodeStatusView({ state, copy }: { state: NodeScreenState; copy: AppMessages }) {
  if (state.kind === "starting") {
    return (
      <section className="panel status-panel" role="status">
        <StatusLabel tone="pending">{copy.node.starting}</StatusLabel>
        <h2>{copy.node.title}</h2>
        <p>{copy.node.startingDescription}</p>
      </section>
    );
  }

  if (state.kind === "failure") {
    return (
      <section className="panel status-panel" role="alert">
        <StatusLabel tone="error">{copy.node.connectionUnavailable}</StatusLabel>
        <h2>{copy.node.title}</h2>
        <p>{state.message}</p>
      </section>
    );
  }

  return (
    <section className="panel node-panel" aria-labelledby="node-title">
      <div className="panel-heading">
        <div>
          <StatusLabel tone="ready">{copy.node.connected}</StatusLabel>
          <h2 id="node-title">{copy.node.title}</h2>
          <p>{copy.node.description}</p>
        </div>
      </div>
      <dl className="detail-grid">
        <div><dt>{copy.node.name}</dt><dd>{state.node.name}</dd></div>
        <div><dt>{copy.node.id}</dt><dd>{state.node.id}</dd></div>
        <div><dt>{copy.node.platform}</dt><dd>{state.node.platform} / {state.node.architecture}</dd></div>
        <div><dt>{copy.node.version}</dt><dd>{state.node.nodeVersion}</dd></div>
        <div><dt>{copy.node.events}</dt><dd>{copy.node.eventStates[state.eventStatus]}</dd></div>
      </dl>
    </section>
  );
}

export function StatusLabel({ children, tone }: { children: string; tone: "pending" | "ready" | "error" }) {
  return (
    <div className={`status status-${tone}`}>
      <span className="status-dot" aria-hidden="true" />
      {children}
    </div>
  );
}

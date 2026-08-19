import type { Node } from "../api/types";
import type { EventStreamStatus } from "../hooks/useEventStreamStatus";
import type { AppMessages } from "../i18n";
import { Card } from "./ui/Card";
import { IconMonitor } from "./ui/icons";
import { StatusDot } from "./ui/StatusDot";
import type { StatusTone } from "../types/ui";

export type NodeScreenState =
  | { kind: "starting" }
  | { kind: "failure"; message: string }
  | { kind: "connected"; node: Node; eventStatus: EventStreamStatus };

export function NodeStatusView({ state, copy }: { state: NodeScreenState; copy: AppMessages }) {
  if (state.kind === "starting") {
    return (
      <Card className="status-card" role="status">
        <StatusLabel tone="pending">{copy.node.starting}</StatusLabel>
        <h2>{copy.node.title}</h2>
        <p className="panel-copy">{copy.node.startingDescription}</p>
      </Card>
    );
  }

  if (state.kind === "failure") {
    return (
      <Card className="status-card" role="alert">
        <StatusLabel tone="error">{copy.node.connectionUnavailable}</StatusLabel>
        <h2>{copy.node.title}</h2>
        <p className="panel-copy">{state.message}</p>
      </Card>
    );
  }

  return (
    <Card className="overview-card" aria-labelledby="node-title">
      <div className="overview-card-head">
        <div className="icon-tile icon-tile-ok">
          <IconMonitor />
        </div>
        <div>
          <h2 id="node-title">{copy.node.title}</h2>
          <StatusLabel tone="ready">{copy.node.connected}</StatusLabel>
        </div>
      </div>
      <p className="panel-copy">{copy.node.description}</p>
      <dl className="detail-list">
        <div>
          <dt>{copy.node.name}</dt>
          <dd>{state.node.name}</dd>
        </div>
        <div>
          <dt>{copy.node.id}</dt>
          <dd className="mono">{state.node.id}</dd>
        </div>
        <div>
          <dt>{copy.node.platform}</dt>
          <dd>{state.node.platform} / {state.node.architecture}</dd>
        </div>
        <div>
          <dt>{copy.node.version}</dt>
          <dd className="mono">{state.node.nodeVersion}</dd>
        </div>
        <div>
          <dt>{copy.node.events}</dt>
          <dd>{copy.node.eventStates[state.eventStatus]}</dd>
        </div>
      </dl>
    </Card>
  );
}

export function StatusLabel({ children, tone }: { children: string; tone: "pending" | "ready" | "error" }) {
  const statusTone: StatusTone = tone === "ready" ? "ok" : tone === "error" ? "error" : "pending";
  return (
    <div className={`status-label status-label-${tone}`}>
      <StatusDot tone={statusTone} />
      {children}
    </div>
  );
}

import type { EventStreamStatus } from "../../hooks/useEventStreamStatus";
import type { AppMessages } from "../../i18n";
import { Card } from "../ui/Card";
import { IconMonitor } from "../ui/icons";
import { StatusLabel } from "../NodeStatusView";

export function LocalNodeCard({
  eventStatus,
  copy,
}: {
  eventStatus: EventStreamStatus;
  copy: AppMessages;
}) {
  const connected = eventStatus === "connected";
  return (
    <Card className="status-strip" aria-labelledby="node-title">
      <div className="status-strip-main">
        <div className="icon-tile icon-tile-ok">
          <IconMonitor />
        </div>
        <div>
          <h2 id="node-title">{copy.node.title}</h2>
          <StatusLabel tone={connected ? "ready" : "pending"}>{copy.node.eventStates[eventStatus]}</StatusLabel>
        </div>
      </div>
      <div className="status-strip-aside">
        <span className="status-aside-label">{copy.dashboard.connectionState}</span>
        <span className={connected ? "status-pill status-pill-ok" : "status-pill"}>{copy.node.eventStates[eventStatus]}</span>
      </div>
    </Card>
  );
}

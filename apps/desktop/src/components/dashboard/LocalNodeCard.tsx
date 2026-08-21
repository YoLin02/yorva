import type { EventStreamStatus } from "../../hooks/useEventStreamStatus";
import type { AppMessages } from "../../i18n";
import { Card } from "../ui/Card";
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
        <div>
          <h2 id="node-title">{copy.node.title}</h2>
          <StatusLabel tone={connected ? "ready" : "pending"}>{copy.node.eventStates[eventStatus]}</StatusLabel>
        </div>
      </div>
    </Card>
  );
}

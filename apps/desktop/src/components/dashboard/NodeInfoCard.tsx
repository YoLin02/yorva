import type { Node } from "../../api/types";
import type { EventStreamStatus } from "../../hooks/useEventStreamStatus";
import type { AppMessages } from "../../i18n";
import { Card } from "../ui/Card";
import { IconChevronRight } from "../ui/icons";
import { StatusDot } from "../ui/StatusDot";

export function NodeInfoCard({
  node,
  eventStatus,
  copy,
  onOpen,
}: {
  node: Node;
  eventStatus: EventStreamStatus;
  copy: AppMessages;
  onOpen: () => void;
}) {
  return (
    <Card className="detail-card">
      <button type="button" className="detail-card-title" onClick={onOpen}>
        <span>{copy.dashboard.nodeInfo}</span>
        <IconChevronRight />
      </button>
      <dl className="detail-list">
        <div>
          <dt>{copy.node.name}</dt>
          <dd>{node.name}</dd>
        </div>
        <div>
          <dt>{copy.node.id}</dt>
          <dd className="mono">{node.id}</dd>
        </div>
        <div>
          <dt>{copy.node.version}</dt>
          <dd className="mono">{node.nodeVersion}</dd>
        </div>
        <div>
          <dt>{copy.node.events}</dt>
          <dd className="detail-inline">
            <StatusDot tone={eventStatus === "connected" ? "ok" : eventStatus === "connecting" ? "pending" : "neutral"} />
            {copy.node.eventStates[eventStatus]}
          </dd>
        </div>
      </dl>
    </Card>
  );
}

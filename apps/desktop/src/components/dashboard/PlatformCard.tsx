import type { Node } from "../../api/types";
import type { AppMessages, Locale } from "../../i18n";
import { formatDateTime } from "../../formatDateTime";
import type { HermesDiscoveryViewState } from "../HermesDiscoveryView";
import { Card } from "../ui/Card";
import { IconChevronRight } from "../ui/icons";
import { discoveryStatus } from "./discoveryStatus";

export function PlatformCard({
  node,
  discoveryState,
  copy,
  locale,
  onOpen,
}: {
  node: Node;
  discoveryState: HermesDiscoveryViewState;
  copy: AppMessages;
  locale: Locale;
  onOpen: () => void;
}) {
  const status = discoveryStatus(discoveryState, copy);
  const lastChecked = status.checkedAt ? formatDateTime(status.checkedAt, locale) : copy.dashboard.unavailableValue;
  const platform = `${node.platform} / ${node.architecture}`;

  return (
    <Card className="detail-card">
      <button type="button" className="detail-card-title" onClick={onOpen}>
        <span>{copy.dashboard.systemPlatform}</span>
        <IconChevronRight />
      </button>
      <dl className="detail-list">
        <div>
          <dt>{copy.node.platform}</dt>
          <dd>{platform}</dd>
        </div>
        <div>
          <dt>{copy.hermes.lastChecked}</dt>
          <dd className="mono">{lastChecked}</dd>
        </div>
      </dl>
    </Card>
  );
}

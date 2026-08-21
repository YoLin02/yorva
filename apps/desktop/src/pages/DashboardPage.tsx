import { HermesStatusCard } from "../components/dashboard/HermesStatusCard";
import { LocalNodeCard } from "../components/dashboard/LocalNodeCard";
import { NodeInfoCard } from "../components/dashboard/NodeInfoCard";
import { PlatformCard } from "../components/dashboard/PlatformCard";
import type { HermesDiscoveryViewState } from "../components/HermesDiscoveryView";
import { NodeStatusView, type NodeScreenState } from "../components/NodeStatusView";
import type { AppMessages, Locale } from "../i18n";

export function DashboardPage({
  nodeState,
  discoveryState,
  copy,
  locale,
  onOpenRuntimes,
  onOpenSettings,
}: {
  nodeState: NodeScreenState;
  discoveryState: HermesDiscoveryViewState;
  copy: AppMessages;
  locale: Locale;
  onOpenRuntimes: () => void;
  onOpenSettings: () => void;
}) {
  if (nodeState.kind !== "connected") {
    return <NodeStatusView state={nodeState} copy={copy} />;
  }

  return (
    <div className="dashboard">
      <div className="card-grid-2">
        <LocalNodeCard eventStatus={nodeState.eventStatus} copy={copy} />
        <HermesStatusCard state={discoveryState} copy={copy} onOpenRuntimes={onOpenRuntimes} />
        <NodeInfoCard
          node={nodeState.node}
          eventStatus={nodeState.eventStatus}
          copy={copy}
          onOpen={onOpenSettings}
        />
        <PlatformCard
          node={nodeState.node}
          discoveryState={discoveryState}
          copy={copy}
          locale={locale}
          onOpen={onOpenRuntimes}
        />
      </div>
    </div>
  );
}

import type { AppMessages } from "../../i18n";
import type { HermesDiscoveryViewState } from "../HermesDiscoveryView";
import { Button } from "../ui/Button";
import { Card } from "../ui/Card";
import { HermesMark, IconChevronRight } from "../ui/icons";
import { StatusLabel } from "../NodeStatusView";
import { discoveryStatus } from "./discoveryStatus";

export function HermesStatusCard({
  state,
  copy,
  onOpenRuntimes,
}: {
  state: HermesDiscoveryViewState;
  copy: AppMessages;
  onOpenRuntimes: () => void;
}) {
  const status = discoveryStatus(state, copy);
  const versionLabel = status.tone === "ready" && status.version
    ? `v${status.version} (${copy.dashboard.statusInstalled})`
    : status.label;

  return (
    <Card className="status-strip status-strip-clickable" aria-labelledby="runtime-summary-title" onClick={onOpenRuntimes}>
      <div className="status-strip-main">
        <div className="hermes-brand-mark hermes-brand-mark-lg" aria-hidden="true">
          <HermesMark size={52} />
        </div>
        <div>
          <h2 id="runtime-summary-title">{copy.hermes.summaryTitle}</h2>
          <StatusLabel tone={status.tone}>{versionLabel}</StatusLabel>
        </div>
      </div>
      <Button
        variant="secondary"
        className="button-compact button-neutral"
        onClick={(event) => {
          event.stopPropagation();
          onOpenRuntimes();
        }}
      >
        {copy.hermes.openRuntimes}
        <IconChevronRight />
      </Button>
    </Card>
  );
}

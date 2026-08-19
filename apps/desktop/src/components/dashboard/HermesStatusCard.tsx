import type { AppMessages } from "../../i18n";
import type { HermesDiscoveryViewState } from "../HermesDiscoveryView";
import { Button } from "../ui/Button";
import { Card } from "../ui/Card";
import { HermesMark, IconRefresh } from "../ui/icons";
import { StatusLabel } from "../NodeStatusView";
import { discoveryStatus, tileClass } from "./discoveryStatus";

export function HermesStatusCard({
  state,
  copy,
}: {
  state: HermesDiscoveryViewState;
  copy: AppMessages;
}) {
  const status = discoveryStatus(state, copy);
  const checking = state.kind === "checking";
  const onRetry = state.kind === "checking" ? undefined : state.onRetry;
  const versionLabel = status.tone === "ready" && status.version
    ? `v${status.version} (${copy.dashboard.statusInstalled})`
    : status.label;

  return (
    <Card className="status-strip" aria-labelledby="runtime-summary-title">
      <div className="status-strip-main">
        <div className={tileClass(status.tone)}>
          <HermesMark />
        </div>
        <div>
          <h2 id="runtime-summary-title">{copy.hermes.summaryTitle}</h2>
          <StatusLabel tone={status.tone}>{versionLabel}</StatusLabel>
        </div>
      </div>
      <Button variant="secondary" className="button-compact" onClick={onRetry} disabled={checking || !onRetry}>
        <IconRefresh className={checking ? "spin" : undefined} />
        {copy.hermes.checkAgain}
      </Button>
    </Card>
  );
}

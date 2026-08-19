import type { AppMessages } from "../../i18n";
import type { HermesDiscoveryViewState } from "../HermesDiscoveryView";
import { Button } from "../ui/Button";
import { Card } from "../ui/Card";
import { HermesMark, IconAlert, IconCheck, IconCheckCircle, IconInfo, IconLoader, IconXCircle, IconZap } from "../ui/icons";
import { discoveryStatus, tileClass } from "./discoveryStatus";

export function DiscoveryBanner({
  state,
  copy,
  onOpenRuntimes,
}: {
  state: HermesDiscoveryViewState;
  copy: AppMessages;
  onOpenRuntimes: () => void;
}) {
  const status = discoveryStatus(state, copy);
  const canInstall = state.kind === "complete" && state.discovery.state === "NOT_INSTALLED";
  const actionLabel = canInstall ? copy.hermes.install.action : copy.hermes.openRuntimes;

  return (
    <Card className="discovery-banner" aria-labelledby="discovery-banner-title">
      <h2 id="discovery-banner-title" className="discovery-banner-kicker">{copy.dashboard.discoveryTitle}</h2>
      <div className="discovery-banner-grid">
        <div className="discovery-banner-main">
          <div className={tileClass(status.tone)}>
            {status.tone === "ready" ? <IconCheckCircle /> : <HermesMark size={28} />}
          </div>
          <div>
            <p className={`discovery-title discovery-title-${status.tone}`}>{status.label}</p>
            <p className="discovery-copy">{status.description}</p>
            <Button variant={canInstall ? "primary" : "secondary"} className="button-compact" onClick={onOpenRuntimes}>
              {canInstall && <IconZap />}
              {actionLabel}
            </Button>
          </div>
        </div>
        <div className="discovery-banner-side">
          <p className="discovery-side-label">{copy.dashboard.supportedStatuses}</p>
          <div className="status-chip-row">
            <span className="legend-chip legend-chip-ok"><IconCheck />{copy.dashboard.statusInstalled}</span>
            <span className="legend-chip legend-chip-warn"><IconAlert />{copy.dashboard.statusUnsupported}</span>
            <span className="legend-chip legend-chip-info"><IconLoader />{copy.dashboard.statusChecking}</span>
            <span className="legend-chip legend-chip-error"><IconXCircle />{copy.dashboard.statusBroken}</span>
          </div>
          <p className="notice notice-info">
            <IconInfo />
            <span>{copy.dashboard.discoveryNotice}</span>
          </p>
        </div>
      </div>
    </Card>
  );
}

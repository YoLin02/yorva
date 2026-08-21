import type { RuntimeDiscovery } from "../api/types";
import { formatDateTime } from "../formatDateTime";
import type { AppMessages, Locale } from "../i18n";
import { Button } from "./ui/Button";
import { Card } from "./ui/Card";
import { HermesMark, IconRefresh } from "./ui/icons";
import { StatusLabel } from "./NodeStatusView";

export type HermesDiscoveryViewState =
  | { kind: "checking"; onCancel: () => void }
  | { kind: "cancelled"; onRetry: () => void }
  | { kind: "failure"; onRetry: () => void }
  | { kind: "complete"; discovery: RuntimeDiscovery; onRetry: () => void };

export function HermesDiscoveryView({
  state,
  copy,
  locale,
  instanceCount,
  onOpenInstances,
}: {
  state: HermesDiscoveryViewState;
  copy: AppMessages;
  locale: Locale;
  instanceCount?: number | null;
  onOpenInstances?: () => void;
}) {
  if (state.kind === "checking") {
    return (
      <Card className="runtime-overview-card runtime-state-card" aria-labelledby="hermes-title" role="status">
        <RuntimeIdentity copy={copy} />
        <div className="runtime-state-copy">
          <StatusLabel tone="pending">{copy.hermes.checking}</StatusLabel>
          <p className="panel-copy">{copy.hermes.checkingDescription}</p>
        </div>
        <Button variant="secondary" onClick={state.onCancel}>{copy.hermes.cancel}</Button>
      </Card>
    );
  }

  if (state.kind === "cancelled" || state.kind === "failure") {
    const cancelled = state.kind === "cancelled";
    return (
      <Card className="runtime-overview-card runtime-state-card" aria-labelledby="hermes-title" role="alert">
        <RuntimeIdentity copy={copy} />
        <div className="runtime-state-copy">
          <StatusLabel tone="error">{cancelled ? copy.hermes.cancelled : copy.hermes.unavailable}</StatusLabel>
          <p className="panel-copy">{cancelled ? copy.hermes.cancelledDescription : copy.hermes.unavailableDescription}</p>
        </div>
        <Button variant="primary" onClick={state.onRetry}>{copy.hermes.retry}</Button>
      </Card>
    );
  }

  const { discovery } = state;
  const stateCopy = copy.hermes.states[discovery.state];
  const candidate = discovery.selected;
  const isSupported = discovery.state === "SUPPORTED";

  return (
    <Card
      className="runtime-overview-card"
      aria-labelledby="hermes-title"
      role={isSupported || discovery.state === "NOT_INSTALLED" ? "status" : "alert"}
    >
      <div className="runtime-overview-primary">
        <RuntimeIdentity copy={copy} />
        {!isSupported ? <p className="runtime-overview-description">{stateCopy.description}</p> : null}
        <dl className="runtime-overview-meta">
          {candidate?.version ? (
            <div>
              <dt>{copy.hermes.version}</dt>
              <dd className="mono runtime-metric-value">{candidate.version}</dd>
            </div>
          ) : null}
          <div>
            <dt>{copy.hermes.lastChecked}</dt>
            <dd className="runtime-metric-value">
              <time dateTime={discovery.detectedAt}>{formatDateTime(discovery.detectedAt, locale)}</time>
            </dd>
          </div>
        </dl>
      </div>

      <div className="runtime-overview-secondary">
        <div className="runtime-secondary-head">
          <div>
            <span className="runtime-field-label">{copy.hermes.compatibility}</span>
            <strong className={isSupported ? "runtime-compatibility is-ready" : "runtime-compatibility"}>{stateCopy.title}</strong>
          </div>
          <Button variant="secondary" onClick={state.onRetry} className="button-compact button-neutral">
            <IconRefresh />
            {copy.hermes.checkAgain}
          </Button>
        </div>

        {candidate?.path ? (
          <div className="runtime-path-field">
            <span className="runtime-field-label">{copy.hermes.executable}</span>
            <code title={candidate.path}>{candidate.path}</code>
          </div>
        ) : null}

        {instanceCount !== undefined && instanceCount !== null ? (
          <div className="runtime-instance-summary">
            <div>
              <span className="runtime-field-label">{copy.hermes.managedInstances}</span>
              <strong>{instanceCount}</strong>
            </div>
            {onOpenInstances ? (
              <Button variant="ghost" onClick={onOpenInstances}>{copy.hermes.viewInstances} →</Button>
            ) : null}
          </div>
        ) : null}
      </div>

      {discovery.state === "AMBIGUOUS" ? (
        <section className="notice notice-warn runtime-wide-notice" aria-labelledby="candidate-title">
          <div>
            <h3 id="candidate-title">{copy.hermes.candidates}</h3>
            <p>{copy.hermes.candidateCount.replace("{count}", String(discovery.candidates.length))}</p>
            <ul className="plain-list">
              {discovery.candidates.map((item) => <li key={item.path}>{item.path}</li>)}
            </ul>
          </div>
        </section>
      ) : null}
      {discovery.warnings.length > 0 ? (
        <section className="notice notice-warn runtime-wide-notice" aria-labelledby="warning-title">
          <div>
            <h3 id="warning-title">{copy.hermes.warnings}</h3>
            <ul className="plain-list">
              {discovery.warnings.map((warning) => (
                <li key={`${warning.code}:${warning.message}`}>
                  {copy.hermes.warningMessages[warning.code] ?? copy.hermes.unknownWarning}
                </li>
              ))}
            </ul>
          </div>
        </section>
      ) : null}
    </Card>
  );
}

function RuntimeIdentity({ copy }: { copy: AppMessages }) {
  return (
    <div className="runtime-identity">
      <span className="runtime-brand-mark" aria-hidden="true"><HermesMark size={48} /></span>
      <div>
        <h2 id="hermes-title">{copy.hermes.summaryTitle}</h2>
        <p>{copy.hermes.summaryDescription}</p>
      </div>
    </div>
  );
}

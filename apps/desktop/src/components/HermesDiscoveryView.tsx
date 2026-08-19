import type { RuntimeDiscovery } from "../api/types";
import { formatDateTime } from "../formatDateTime";
import type { AppMessages, Locale } from "../i18n";
import { Button } from "./ui/Button";
import { Card } from "./ui/Card";
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
}: {
  state: HermesDiscoveryViewState;
  copy: AppMessages;
  locale: Locale;
}) {
  if (state.kind === "checking") {
    return (
      <Card className="runtime-card" aria-labelledby="hermes-title" role="status">
        <StatusLabel tone="pending">{copy.hermes.checking}</StatusLabel>
        <h2 id="hermes-title">{copy.hermes.title}</h2>
        <p className="panel-copy">{copy.hermes.checkingDescription}</p>
        <Button variant="primary" onClick={state.onCancel}>{copy.hermes.cancel}</Button>
      </Card>
    );
  }

  if (state.kind === "cancelled" || state.kind === "failure") {
    const cancelled = state.kind === "cancelled";
    return (
      <Card className="runtime-card" aria-labelledby="hermes-title" role="alert">
        <StatusLabel tone="error">{cancelled ? copy.hermes.cancelled : copy.hermes.unavailable}</StatusLabel>
        <h2 id="hermes-title">{copy.hermes.title}</h2>
        <p className="panel-copy">{cancelled ? copy.hermes.cancelledDescription : copy.hermes.unavailableDescription}</p>
        <Button variant="primary" onClick={state.onRetry}>{copy.hermes.retry}</Button>
      </Card>
    );
  }

  const { discovery } = state;
  const stateCopy = copy.hermes.states[discovery.state];
  const candidate = discovery.selected;
  const isSupported = discovery.state === "SUPPORTED";
  const tone = isSupported ? "ready" : discovery.state === "NOT_INSTALLED" ? "pending" : "error";

  return (
    <Card
      className="runtime-card"
      aria-labelledby="hermes-title"
      role={isSupported || discovery.state === "NOT_INSTALLED" ? "status" : "alert"}
    >
      <div className="panel-heading panel-heading-split">
        <div>
          <StatusLabel tone={tone}>{stateCopy.title}</StatusLabel>
          <h2 id="hermes-title">{copy.hermes.title}</h2>
          <p className="panel-copy">{stateCopy.description}</p>
        </div>
        <Button variant="secondary" onClick={state.onRetry}>{copy.hermes.checkAgain}</Button>
      </div>

      <dl className="detail-list">
        {candidate?.version && <div><dt>{copy.hermes.version}</dt><dd>{candidate.version}</dd></div>}
        {candidate?.path && <div><dt>{copy.hermes.executable}</dt><dd className="mono">{candidate.path}</dd></div>}
        <div><dt>{copy.hermes.compatibility}</dt><dd>{stateCopy.title}</dd></div>
        <div><dt>{copy.hermes.supportedRange}</dt><dd className="mono">{discovery.supportedRange}</dd></div>
        <div><dt>{copy.hermes.lastChecked}</dt><dd><time dateTime={discovery.detectedAt}>{formatDateTime(discovery.detectedAt, locale)}</time></dd></div>
      </dl>

      {discovery.state === "AMBIGUOUS" && (
        <section className="notice notice-warn" aria-labelledby="candidate-title">
          <div>
            <h3 id="candidate-title">{copy.hermes.candidates}</h3>
            <p>{copy.hermes.candidateCount.replace("{count}", String(discovery.candidates.length))}</p>
            <ul className="plain-list">
              {discovery.candidates.map((item) => <li key={item.path}>{item.path}</li>)}
            </ul>
          </div>
        </section>
      )}
      {discovery.warnings.length > 0 && (
        <section className="notice notice-warn" aria-labelledby="warning-title">
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
      )}
    </Card>
  );
}

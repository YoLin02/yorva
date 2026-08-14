import type { RuntimeDiscovery } from "../api/types";
import { formatDateTime } from "../formatDateTime";
import type { AppMessages, Locale } from "../i18n";
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
      <section className="panel runtime-panel" aria-labelledby="hermes-title" role="status">
        <StatusLabel tone="pending">{copy.hermes.checking}</StatusLabel>
        <h2 id="hermes-title">{copy.hermes.title}</h2>
        <p>{copy.hermes.checkingDescription}</p>
        <button type="button" className="primary-action" onClick={state.onCancel}>{copy.hermes.cancel}</button>
      </section>
    );
  }

  if (state.kind === "cancelled" || state.kind === "failure") {
    const cancelled = state.kind === "cancelled";
    return (
      <section className="panel runtime-panel" aria-labelledby="hermes-title" role="alert">
        <StatusLabel tone="error">{cancelled ? copy.hermes.cancelled : copy.hermes.unavailable}</StatusLabel>
        <h2 id="hermes-title">{copy.hermes.title}</h2>
        <p>{cancelled ? copy.hermes.cancelledDescription : copy.hermes.unavailableDescription}</p>
        <button type="button" className="primary-action" onClick={state.onRetry}>{copy.hermes.retry}</button>
      </section>
    );
  }

  const { discovery } = state;
  const stateCopy = copy.hermes.states[discovery.state];
  const candidate = discovery.selected;
  const isSupported = discovery.state === "SUPPORTED";
  const tone = isSupported ? "ready" : discovery.state === "NOT_INSTALLED" ? "pending" : "error";

  return (
    <section
      className="panel runtime-panel"
      aria-labelledby="hermes-title"
      role={isSupported || discovery.state === "NOT_INSTALLED" ? "status" : "alert"}
    >
      <div className="panel-heading panel-heading-split">
        <div>
          <StatusLabel tone={tone}>{stateCopy.title}</StatusLabel>
          <h2 id="hermes-title">{copy.hermes.title}</h2>
          <p>{stateCopy.description}</p>
        </div>
        <button type="button" className="primary-action action-top" onClick={state.onRetry}>{copy.hermes.checkAgain}</button>
      </div>

      <dl className="detail-grid runtime-details">
        {candidate?.version && <div><dt>{copy.hermes.version}</dt><dd>{candidate.version}</dd></div>}
        {candidate?.path && <div className="detail-wide"><dt>{copy.hermes.executable}</dt><dd>{candidate.path}</dd></div>}
        <div><dt>{copy.hermes.compatibility}</dt><dd>{stateCopy.title}</dd></div>
        <div><dt>{copy.hermes.supportedRange}</dt><dd>{discovery.supportedRange}</dd></div>
        <div><dt>{copy.hermes.lastChecked}</dt><dd><time dateTime={discovery.detectedAt}>{formatDateTime(discovery.detectedAt, locale)}</time></dd></div>
      </dl>

      {discovery.state === "AMBIGUOUS" && (
        <section className="diagnostic-block" aria-labelledby="candidate-title">
          <h3 id="candidate-title">{copy.hermes.candidates}</h3>
          <p>{copy.hermes.candidateCount.replace("{count}", String(discovery.candidates.length))}</p>
          <ul className="candidate-list">
            {discovery.candidates.map((item) => <li key={item.path}>{item.path}</li>)}
          </ul>
        </section>
      )}
      {discovery.warnings.length > 0 && (
        <section className="diagnostic-block" aria-labelledby="warning-title">
          <h3 id="warning-title">{copy.hermes.warnings}</h3>
          <ul className="runtime-warnings">
            {discovery.warnings.map((warning) => (
              <li key={`${warning.code}:${warning.message}`}>
                {copy.hermes.warningMessages[warning.code] ?? copy.hermes.unknownWarning}
              </li>
            ))}
          </ul>
        </section>
      )}
    </section>
  );
}

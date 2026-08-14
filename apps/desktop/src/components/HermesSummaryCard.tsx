import type { AppMessages } from "../i18n";
import type { HermesDiscoveryViewState } from "./HermesDiscoveryView";
import { StatusLabel } from "./NodeStatusView";

export function HermesSummaryCard({
  state,
  copy,
  onOpen,
}: {
  state: HermesDiscoveryViewState;
  copy: AppMessages;
  onOpen: () => void;
}) {
  let label: string;
  let tone: "pending" | "ready" | "error";
  let version: string | undefined;

  if (state.kind === "checking") {
    label = copy.hermes.checking;
    tone = "pending";
  } else if (state.kind === "cancelled") {
    label = copy.hermes.cancelled;
    tone = "error";
  } else if (state.kind === "failure") {
    label = copy.hermes.unavailable;
    tone = "error";
  } else {
    label = copy.hermes.states[state.discovery.state].title;
    tone = state.discovery.state === "SUPPORTED" ? "ready" : state.discovery.state === "NOT_INSTALLED" ? "pending" : "error";
    version = state.discovery.selected?.version;
  }

  return (
    <section className="panel runtime-summary" aria-labelledby="runtime-summary-title">
      <StatusLabel tone={tone}>{label}</StatusLabel>
      <h2 id="runtime-summary-title">{copy.hermes.summaryTitle}</h2>
      <p>{copy.hermes.summaryDescription}</p>
      {version && <strong className="runtime-version">Hermes {version}</strong>}
      <button type="button" className="secondary-action" onClick={onOpen}>{copy.hermes.openRuntimes}</button>
    </section>
  );
}

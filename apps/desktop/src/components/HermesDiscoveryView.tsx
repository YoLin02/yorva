import type { RuntimeDiscovery, RuntimeDiscoveryState } from "../api/types";

export type HermesDiscoveryViewState =
  | { kind: "checking"; onCancel: () => void }
  | { kind: "cancelled"; onRetry: () => void }
  | { kind: "failure"; onRetry: () => void }
  | { kind: "complete"; discovery: RuntimeDiscovery; onRetry: () => void };

const stateCopy: Record<RuntimeDiscoveryState, { title: string; description: string }> = {
  NOT_INSTALLED: {
    title: "Hermes not installed",
    description: "No Hermes executable was found in the supported local locations.",
  },
  SUPPORTED: {
    title: "Hermes ready",
    description: "The detected Hermes version is compatible with YORVA.",
  },
  UNSUPPORTED: {
    title: "Hermes version unsupported",
    description: "Hermes was found, but its version is outside the supported range.",
  },
  BROKEN_EXECUTABLE: {
    title: "Hermes executable is broken",
    description: "Hermes was found but could not report its version successfully.",
  },
  MALFORMED_VERSION: {
    title: "Hermes version is unreadable",
    description: "Hermes returned a version that YORVA could not safely interpret.",
  },
  TIMED_OUT: {
    title: "Hermes check timed out",
    description: "Hermes did not report its version before the discovery deadline.",
  },
  AMBIGUOUS: {
    title: "Multiple Hermes executables found",
    description: "YORVA found more than one runnable Hermes executable and did not choose between them.",
  },
};

export function HermesDiscoveryView({ state }: { state: HermesDiscoveryViewState }) {
  if (state.kind === "checking") {
    return (
      <section className="runtime-card" aria-labelledby="hermes-title" role="status">
        <RuntimeLabel tone="pending">Checking Hermes</RuntimeLabel>
        <h2 id="hermes-title">Hermes discovery</h2>
        <p>Looking for a local Hermes executable and checking its version.</p>
        <button type="button" onClick={state.onCancel}>Cancel</button>
      </section>
    );
  }

  if (state.kind === "cancelled" || state.kind === "failure") {
    const cancelled = state.kind === "cancelled";
    return (
      <section className="runtime-card" aria-labelledby="hermes-title" role="alert">
        <RuntimeLabel tone="error">{cancelled ? "Check cancelled" : "Discovery unavailable"}</RuntimeLabel>
        <h2 id="hermes-title">Hermes discovery</h2>
        <p>
          {cancelled
            ? "The Hermes check was cancelled. No Runtime state was changed."
            : "YORVA could not complete Hermes discovery."}
        </p>
        <button type="button" onClick={state.onRetry}>Retry</button>
      </section>
    );
  }

  const { discovery } = state;
  const copy = stateCopy[discovery.state];
  const candidate = discovery.selected ?? discovery.candidates[0];
  const isSupported = discovery.state === "SUPPORTED";
  return (
    <section
      className="runtime-card"
      aria-labelledby="hermes-title"
      role={isSupported || discovery.state === "NOT_INSTALLED" ? "status" : "alert"}
    >
      <RuntimeLabel tone={isSupported ? "ready" : discovery.state === "NOT_INSTALLED" ? "pending" : "error"}>
        {copy.title}
      </RuntimeLabel>
      <h2 id="hermes-title">Hermes discovery</h2>
      <p>{copy.description}</p>
      {(candidate || discovery.state === "NOT_INSTALLED") && (
        <dl className="runtime-details">
          {candidate?.version && <div><dt>Version</dt><dd>{candidate.version}</dd></div>}
          {candidate?.path && <div><dt>Executable</dt><dd>{candidate.path}</dd></div>}
          <div><dt>Supported range</dt><dd>{discovery.supportedRange}</dd></div>
          <div><dt>Last checked</dt><dd><time dateTime={discovery.detectedAt}>{discovery.detectedAt}</time></dd></div>
        </dl>
      )}
      {discovery.state === "AMBIGUOUS" && (
        <ul className="candidate-list" aria-label="Hermes candidates">
          {discovery.candidates.map((item) => <li key={item.path}>{item.path}</li>)}
        </ul>
      )}
      {discovery.warnings.length > 0 && (
        <ul className="runtime-warnings" aria-label="Discovery warnings">
          {discovery.warnings.map((warning) => <li key={`${warning.code}:${warning.message}`}>{warning.message}</li>)}
        </ul>
      )}
      <button type="button" onClick={state.onRetry}>Check again</button>
    </section>
  );
}

function RuntimeLabel({ children, tone }: { children: string; tone: "pending" | "ready" | "error" }) {
  return (
    <div className={`status status-${tone}`}>
      <span className="status-dot" aria-hidden="true" />
      {children}
    </div>
  );
}

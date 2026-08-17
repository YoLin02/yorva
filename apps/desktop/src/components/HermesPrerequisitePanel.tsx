import type { Operation } from "../api/types";
import type { HermesPrerequisites } from "../api/types";
import type { AppMessages } from "../i18n";

export function HermesPrerequisitePanel({
  copy,
  status,
  operation,
  liveLog,
  busy,
  onInstall,
  onRetryDeps,
  onCancel,
}: {
  copy: AppMessages;
  status: HermesPrerequisites | null;
  operation: Operation | null;
  liveLog?: string;
  busy: boolean;
  onInstall: () => void;
  onRetryDeps: () => void;
  onCancel: () => void;
}) {
  const text = copy.hermes.prerequisites;
  const running = operation && (operation.status === "PENDING" || operation.status === "RUNNING");
  const nodeState = status?.node.state ?? "MISSING";
  const npmState = status?.npm.state ?? "MISSING";
  const depsState = status?.nodeDependencies.state ?? "NOT_INSTALLED";

  let summary = text.nodeMissing;
  if (nodeState === "READY" && npmState === "READY" && depsState === "READY") {
    summary = text.nodeReady;
  } else if (nodeState === "UNSUPPORTED") {
    summary = text.nodeUnsupported;
  } else if (npmState === "UNSUPPORTED") {
    summary = text.npmUnsupported;
  } else if (depsState === "FAILED" || depsState === "TIMED_OUT") {
    summary = text.depsFailed;
  } else if (nodeState === "READY" && npmState === "READY") {
    summary = text.depsNotInstalled;
  }

  const showInstall = nodeState !== "READY" || npmState !== "READY";
  const showRetryDeps = nodeState === "READY" && npmState === "READY" && depsState !== "READY";

  return (
    <section className="diagnostic-block" aria-live="polite">
      <h3>{text.title}</h3>
      <p>{summary}</p>
      {status?.node.version && <p>Node {status.node.version}</p>}
      {status?.npm.version && <p>npm {status.npm.version}</p>}
      {status?.node.errorCode && <p>{copy.hermes.install.errorCode}: {status.node.errorCode}</p>}
      {running && (
        <>
          <p>{copy.hermes.install.stage}: {operation?.stage === "install.node" ? text.installingNode : text.installingDeps}</p>
          {liveLog && <pre className="install-log">{liveLog}</pre>}
          <button type="button" onClick={onCancel} disabled={busy}>{text.cancel}</button>
        </>
      )}
      {!running && showInstall && (
        <button type="button" className="primary-action" onClick={onInstall} disabled={busy}>
          {text.installAction}
        </button>
      )}
      {!running && showRetryDeps && (
        <button type="button" className="primary-action" onClick={onRetryDeps} disabled={busy}>
          {text.retryDeps}
        </button>
      )}
    </section>
  );
}

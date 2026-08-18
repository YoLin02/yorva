import type { Operation } from "../api/types";
import type { HermesPrerequisites } from "../api/types";
import type { InstallRequestError } from "../installDiagnostic";
import type { AppMessages } from "../i18n";

export function HermesPrerequisitePanel({
  copy,
  status,
  operation,
  liveLog,
  busy,
  blocked,
  requestError,
  hermesNotInstalled,
  onInstall,
  onRetryDeps,
  onCancel,
}: {
  copy: AppMessages;
  status: HermesPrerequisites | null;
  operation: Operation | null;
  liveLog?: string;
  busy: boolean;
  blocked?: boolean;
  requestError?: InstallRequestError | null;
  hermesNotInstalled?: boolean;
  onInstall: () => void;
  onRetryDeps: () => void;
  onCancel: () => void;
}) {
  const text = copy.hermes.prerequisites;
  const running = operation && (operation.status === "PENDING" || operation.status === "RUNNING");
  const failed = Boolean(
    requestError || (operation && (operation.status === "FAILED" || operation.status === "CANCELLED")),
  );
  const starting = busy && !blocked && !running && !failed;
  const disabled = busy || Boolean(blocked);
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
  const errorCode = operation?.errorCode ?? requestError?.code ?? status?.node.errorCode;

  return (
    <section className="diagnostic-block" aria-live="polite">
      <h3>{text.title}</h3>
      <p>{summary}</p>
      {status?.node.version && <p>Node {status.node.version}</p>}
      {status?.npm.version && <p>npm {status.npm.version}</p>}
      {hermesNotInstalled && (running || failed || showInstall) && <p>{text.continueWithoutNode}</p>}
      {!failed && status?.node.errorCode && (
        <p>{copy.hermes.install.errorCode}: {status.node.errorCode}</p>
      )}
      {starting && <p>{text.starting}</p>}
      {running && (
        <>
          <p>{copy.hermes.install.stage}: {operation?.stage === "install.node" ? text.installingNode : text.installingDeps}</p>
          {liveLog && <pre className="install-log">{liveLog}</pre>}
          <button type="button" onClick={onCancel} disabled={disabled}>{text.cancel}</button>
        </>
      )}
      {failed && !running && (
        <div role="alert">
          <p>{operation?.status === "CANCELLED" ? text.cancelled : text.failed}</p>
          {operation?.stage && (
            <p>{copy.hermes.install.stage}: {operation.stage === "install.node" ? text.installingNode : operation.stage}</p>
          )}
          {errorCode && <p>{copy.hermes.install.errorCode}: {errorCode}</p>}
          {operation?.correlationId && <p>{copy.hermes.install.correlation}: {operation.correlationId}</p>}
          {requestError?.message && <p>{requestError.message}</p>}
          {liveLog && <pre className="install-log">{liveLog}</pre>}
          <button type="button" className="primary-action" onClick={onRetryDeps} disabled={disabled}>
            {text.retry}
          </button>
        </div>
      )}
      {!running && !failed && !starting && showInstall && (
        <button type="button" className="primary-action" onClick={onInstall} disabled={disabled}>
          {text.installAction}
        </button>
      )}
      {!running && !failed && !starting && showRetryDeps && (
        <button type="button" className="primary-action" onClick={onRetryDeps} disabled={disabled}>
          {text.retryDeps}
        </button>
      )}
    </section>
  );
}

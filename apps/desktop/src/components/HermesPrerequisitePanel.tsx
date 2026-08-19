import type { Operation } from "../api/types";
import type { HermesPrerequisites } from "../api/types";
import type { InstallRequestError } from "../installDiagnostic";
import type { AppMessages } from "../i18n";
import { Button } from "./ui/Button";
import { Card } from "./ui/Card";

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
    <Card className="runtime-card" aria-live="polite">
      <h3>{text.title}</h3>
      <p className="panel-copy">{summary}</p>
      {status?.node.version && <p className="small">Node {status.node.version}</p>}
      {status?.npm.version && <p className="small">npm {status.npm.version}</p>}
      {hermesNotInstalled && (running || failed || showInstall) && <p className="notice notice-info">{text.continueWithoutNode}</p>}
      {!failed && status?.node.errorCode && (
        <p className="panel-copy">{copy.hermes.install.errorCode}: {status.node.errorCode}</p>
      )}
      {starting && <p className="panel-copy">{text.starting}</p>}
      {running && (
        <>
          <p className="panel-copy">{copy.hermes.install.stage}: {operation?.stage === "install.node" ? text.installingNode : text.installingDeps}</p>
          {liveLog && <pre className="install-log-body">{liveLog}</pre>}
          <Button onClick={onCancel} disabled={disabled}>{text.cancel}</Button>
        </>
      )}
      {failed && !running && (
        <div role="alert">
          <p className="panel-copy">{operation?.status === "CANCELLED" ? text.cancelled : text.failed}</p>
          {operation?.stage && (
            <p className="panel-copy">{copy.hermes.install.stage}: {operation.stage === "install.node" ? text.installingNode : operation.stage}</p>
          )}
          {errorCode && <p className="panel-copy">{copy.hermes.install.errorCode}: {errorCode}</p>}
          {operation?.correlationId && <p className="panel-copy">{copy.hermes.install.correlation}: {operation.correlationId}</p>}
          {requestError?.message && <p className="panel-copy">{requestError.message}</p>}
          {liveLog && <pre className="install-log-body">{liveLog}</pre>}
          <Button variant="primary" onClick={onRetryDeps} disabled={disabled}>
            {text.retry}
          </Button>
        </div>
      )}
      {!running && !failed && !starting && showInstall && (
        <Button variant="primary" onClick={onInstall} disabled={disabled}>
          {text.installAction}
        </Button>
      )}
      {!running && !failed && !starting && showRetryDeps && (
        <Button variant="primary" onClick={onRetryDeps} disabled={disabled}>
          {text.retryDeps}
        </Button>
      )}
    </Card>
  );
}

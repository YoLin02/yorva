import type { HermesPrerequisites, Operation } from "../api/types";
import { formatDateTime } from "../formatDateTime";
import type { InstallRequestError } from "../installDiagnostic";
import type { AppMessages, Locale } from "../i18n";
import { Button } from "./ui/Button";
import { Card } from "./ui/Card";

export function HermesPrerequisitePanel({
  copy,
  locale = "en-US",
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
  locale?: Locale;
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
  const failed = Boolean(requestError || (operation && (operation.status === "FAILED" || operation.status === "CANCELLED")));
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
  const hasMissingComponents = status !== null && (
    nodeState !== "READY" || npmState !== "READY" || depsState !== "READY"
  );

  return (
    <Card className="runtime-support-card" aria-live="polite">
      <div className="runtime-support-heading">
        <div>
          <h3>{text.title}</h3>
          <p className="panel-copy">{summary}</p>
        </div>
        {status?.checkedAt ? (
          <p className="runtime-support-checked">
            <span>{text.lastChecked}</span>
            <time dateTime={status.checkedAt}>{formatDateTime(status.checkedAt, locale)}</time>
          </p>
        ) : null}
      </div>

      {hasMissingComponents ? (
        <div className="prerequisite-grid">
          {nodeState !== "READY" ? <PrerequisiteItem label={text.nodeLabel} state={nodeState} version={status?.node.version} copy={copy} /> : null}
          {npmState !== "READY" ? <PrerequisiteItem label={text.npmLabel} state={npmState} version={status?.npm.version} copy={copy} /> : null}
          {depsState !== "READY" ? <PrerequisiteItem label={text.dependenciesLabel} state={depsState} version={status?.nodeDependencies.version} copy={copy} /> : null}
        </div>
      ) : null}

      {hermesNotInstalled && (running || failed || showInstall) ? <p className="notice notice-info">{text.continueWithoutNode}</p> : null}
      {!failed && status?.node.errorCode ? <p className="panel-copy">{copy.hermes.install.errorCode}: {status.node.errorCode}</p> : null}
      {starting ? <p className="panel-copy">{text.starting}</p> : null}

      {running ? (
        <div className="runtime-operation-block">
          <p className="panel-copy">{copy.hermes.install.stage}: {operation?.stage === "install.node" ? text.installingNode : text.installingDeps}</p>
          {liveLog ? <pre className="install-log-body">{liveLog}</pre> : null}
          <Button onClick={onCancel} disabled={disabled}>{text.cancel}</Button>
        </div>
      ) : null}

      {failed && !running ? (
        <div className="runtime-operation-block" role="alert">
          <p className="panel-copy">{operation?.status === "CANCELLED" ? text.cancelled : text.failed}</p>
          {operation?.stage ? <p className="panel-copy">{copy.hermes.install.stage}: {operation.stage === "install.node" ? text.installingNode : operation.stage}</p> : null}
          {errorCode ? <p className="panel-copy">{copy.hermes.install.errorCode}: {errorCode}</p> : null}
          {operation?.correlationId ? <p className="panel-copy">{copy.hermes.install.correlation}: {operation.correlationId}</p> : null}
          {requestError?.message ? <p className="panel-copy">{requestError.message}</p> : null}
          {liveLog ? <pre className="install-log-body">{liveLog}</pre> : null}
          <Button variant="primary" onClick={onRetryDeps} disabled={disabled}>{text.retry}</Button>
        </div>
      ) : null}

      {!running && !failed && !starting && showInstall ? (
        <div className="runtime-support-actions">
          <Button variant="primary" onClick={onInstall} disabled={disabled}>{text.installAction}</Button>
        </div>
      ) : null}
      {!running && !failed && !starting && showRetryDeps ? (
        <div className="runtime-support-actions">
          <Button variant="primary" onClick={onRetryDeps} disabled={disabled}>{text.retryDeps}</Button>
        </div>
      ) : null}
    </Card>
  );
}

function PrerequisiteItem({ label, state, version, copy }: {
  label: string;
  state: string;
  version?: string;
  copy: AppMessages;
}) {
  const text = copy.hermes.prerequisites;
  const ready = state === "READY";
  const attention = state === "UNSUPPORTED" || state === "FAILED" || state === "TIMED_OUT";
  const stateLabel = ready ? text.ready : attention ? text.needsAttention : text.unavailable;
  const tone = ready ? "is-ready" : attention ? "is-attention" : "is-unavailable";

  return (
    <div className="prerequisite-item">
      <div className="prerequisite-item-head">
        <span className="prerequisite-name">{label}</span>
        <span className={`prerequisite-state ${tone}`}><i />{stateLabel}</span>
      </div>
      <strong className="prerequisite-version mono">{version || copy.hermes.unavailableValue}</strong>
    </div>
  );
}

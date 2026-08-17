import type { Operation } from "../api/types";
import type { AppMessages } from "../i18n";

export function HermesInstallPanel({
  copy,
  windowsHost,
  confirmOpen,
  busy,
  operation,
  onOpenConfirm,
  onCloseConfirm,
  onConfirm,
  onCancel,
  onRetry,
}: {
  copy: AppMessages;
  windowsHost: boolean;
  confirmOpen: boolean;
  busy: boolean;
  operation: Operation | null;
  onOpenConfirm: () => void;
  onCloseConfirm: () => void;
  onConfirm: () => void;
  onCancel: () => void;
  onRetry: () => void;
}) {
  const install = copy.hermes.install;
  if (!windowsHost) {
    return <p>{install.unavailable}</p>;
  }
  if (operation && (operation.status === "PENDING" || operation.status === "RUNNING")) {
    return (
      <section className="diagnostic-block" aria-live="polite">
        <h3>{install.running}</h3>
        <p>{install.stage}: {operation.stage}</p>
        <button type="button" className="primary-action" onClick={onCancel} disabled={busy}>
          {install.cancelInstall}
        </button>
      </section>
    );
  }
  if (operation && (operation.status === "FAILED" || operation.status === "CANCELLED")) {
    const title = operation.errorCode === "OPERATION_INTERRUPTED" ? install.interrupted : operation.status === "CANCELLED" ? install.cancelled : install.failed;
    return (
      <section className="diagnostic-block" role="alert">
        <h3>{title}</h3>
        {operation.correlationId && <p>{install.correlation}: {operation.correlationId}</p>}
        {operation.retryable && (
          <button type="button" className="primary-action" onClick={onRetry} disabled={busy}>
            {install.retryInstall}
          </button>
        )}
      </section>
    );
  }
  if (confirmOpen) {
    return (
      <section className="diagnostic-block" aria-labelledby="install-confirm-title">
        <h3 id="install-confirm-title">{install.confirmTitle}</h3>
        <p>{install.confirmDescription}</p>
        <dl className="detail-grid">
          <div><dt>{install.source}</dt><dd>NousResearch/hermes-agent @ df4b65147d7ddd74dd449f9067aabbca5aef0ec7</dd></div>
          <div><dt>{install.version}</dt><dd>0.20.2 / v2026.8.16</dd></div>
          <div className="detail-wide"><dt>{install.destination}</dt><dd>%LOCALAPPDATA%\hermes\hermes-agent</dd></div>
        </dl>
        <p>{install.hostChanges}</p>
        <ul>
          {install.hostChangeItems.map((item) => <li key={item}>{item}</li>)}
        </ul>
        <p>{install.noProfileNote}</p>
        <div className="panel-heading-split">
          <button type="button" onClick={onCloseConfirm} disabled={busy}>{install.back}</button>
          <button type="button" className="primary-action" onClick={onConfirm} disabled={busy}>{install.confirm}</button>
        </div>
      </section>
    );
  }
  return (
    <button type="button" className="primary-action" onClick={onOpenConfirm} disabled={busy}>
      {install.action}
    </button>
  );
}

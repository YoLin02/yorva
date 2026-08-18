import { useState } from "react";
import type { Operation } from "../api/types";
import type { AppMessages } from "../i18n";
import { formatInstallDiagnostic, type InstallRequestError } from "../installDiagnostic";

export function HermesInstallPanel({
  copy,
  windowsHost,
  canStart = true,
  confirmOpen,
  busy,
  operation,
  liveLog,
  requestError,
  onOpenConfirm,
  onCloseConfirm,
  onConfirm,
  onCancel,
  onRetry,
}: {
  copy: AppMessages;
  windowsHost: boolean;
  canStart?: boolean;
  confirmOpen: boolean;
  busy: boolean;
  operation: Operation | null;
  liveLog?: string;
  requestError?: InstallRequestError | null;
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
        {sourceNote(install, operation.message) && <p>{sourceNote(install, operation.message)}</p>}
        <InstallLogBox copy={copy} text={liveLog || formatInstallDiagnostic({ operation })} />
        <button type="button" className="primary-action" onClick={onCancel} disabled={busy}>
          {install.cancelInstall}
        </button>
      </section>
    );
  }
  if (requestError || (operation && (operation.status === "FAILED" || operation.status === "CANCELLED"))) {
    const title = operation?.errorCode === "OPERATION_INTERRUPTED"
      ? install.interrupted
      : operation?.status === "CANCELLED"
        ? install.cancelled
        : install.failed;
    const logText = liveLog || formatInstallDiagnostic({ operation, requestError });
    return (
      <section className="diagnostic-block" role="alert">
        <h3>{title}</h3>
        {operation?.stage && <p>{install.stage}: {operation.stage}</p>}
        {sourceNote(install, operation?.message) && <p>{sourceNote(install, operation?.message)}</p>}
        {(operation?.errorCode || requestError?.code) && (
          <p>{install.errorCode}: {operation?.errorCode ?? requestError?.code}</p>
        )}
        {operation?.correlationId && <p>{install.correlation}: {operation.correlationId}</p>}
        <InstallLogBox copy={copy} text={logText} />
        {operation?.retryable && (
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
        <p>{install.bundledSourceNote}</p>
        <p>{install.noProfileNote}</p>
        <div className="panel-heading-split">
          <button type="button" onClick={onCloseConfirm} disabled={busy}>{install.back}</button>
          <button type="button" className="primary-action" onClick={onConfirm} disabled={busy}>{install.confirm}</button>
        </div>
      </section>
    );
  }
  if (!canStart) {
    return null;
  }
  return (
    <button type="button" className="primary-action" onClick={onOpenConfirm} disabled={busy}>
      {install.action}
    </button>
  );
}

function sourceNote(install: AppMessages["hermes"]["install"], code?: string) {
  if (!code) {
    return "";
  }
  return install.sourceNotes[code] ?? "";
}

function InstallLogBox({ copy, text }: { copy: AppMessages; text: string }) {
  const install = copy.hermes.install;
  const [copied, setCopied] = useState(false);
  if (!text) {
    return null;
  }
  const copyLog = async () => {
    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(text);
      } else {
        const area = document.createElement("textarea");
        area.value = text;
        area.setAttribute("readonly", "");
        area.style.position = "fixed";
        area.style.left = "-9999px";
        document.body.append(area);
        area.select();
        document.execCommand("copy");
        area.remove();
      }
      setCopied(true);
      window.setTimeout(() => setCopied(false), 2000);
    } catch {
      setCopied(false);
    }
  };
  return (
    <div className="install-log">
      <div className="install-log-heading">
        <p>{install.logTitle}</p>
        <button type="button" className="secondary-action action-top" onClick={() => { void copyLog(); }}>
          {copied ? install.logCopied : install.copyLog}
        </button>
      </div>
      <pre className="install-log-body">{text}</pre>
      <p>{install.logHint}</p>
    </div>
  );
}

import { useState } from "react";
import type { Operation } from "../api/types";
import type { AppMessages } from "../i18n";
import { formatInstallDiagnostic, type InstallRequestError } from "../installDiagnostic";
import { Button } from "./ui/Button";
import { Card } from "./ui/Card";

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
    return <p className="panel-copy">{install.unavailable}</p>;
  }
  if (operation && (operation.status === "PENDING" || operation.status === "RUNNING")) {
    return (
      <Card className="runtime-card" aria-live="polite">
        <h3>{install.running}</h3>
        <p className="panel-copy">{install.stage}: {operation.stage}</p>
        {sourceNote(install, operation.message) && <p className="panel-copy">{sourceNote(install, operation.message)}</p>}
        <InstallLogBox copy={copy} text={liveLog || formatInstallDiagnostic({ operation })} />
        <Button variant="secondary" onClick={onCancel} disabled={busy}>
          {install.cancelInstall}
        </Button>
      </Card>
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
      <Card className="runtime-card" role="alert">
        <h3>{title}</h3>
        {operation?.stage && <p className="panel-copy">{install.stage}: {operation.stage}</p>}
        {sourceNote(install, operation?.message) && <p className="panel-copy">{sourceNote(install, operation?.message)}</p>}
        {(operation?.errorCode || requestError?.code) && (
          <p className="panel-copy">{install.errorCode}: {operation?.errorCode ?? requestError?.code}</p>
        )}
        {operation?.correlationId && <p className="panel-copy">{install.correlation}: {operation.correlationId}</p>}
        <InstallLogBox copy={copy} text={logText} />
        {operation?.retryable && (
          <Button variant="primary" onClick={onRetry} disabled={busy}>
            {install.retryInstall}
          </Button>
        )}
      </Card>
    );
  }
  if (confirmOpen) {
    return (
      <Card className="runtime-card" aria-labelledby="install-confirm-title">
        <h3 id="install-confirm-title">{install.confirmTitle}</h3>
        <p className="panel-copy">{install.confirmDescription}</p>
        <dl className="detail-list">
          <div><dt>{install.source}</dt><dd>NousResearch/hermes-agent @ df4b65147d7ddd74dd449f9067aabbca5aef0ec7</dd></div>
          <div><dt>{install.version}</dt><dd>0.20.2 / v2026.8.16</dd></div>
          <div><dt>{install.destination}</dt><dd className="mono">%LOCALAPPDATA%\hermes\hermes-agent</dd></div>
        </dl>
        <p className="panel-copy">{install.hostChanges}</p>
        <ul className="plain-list">
          {install.hostChangeItems.map((item) => <li key={item}>{item}</li>)}
        </ul>
        <p className="panel-copy">{install.bundledSourceNote}</p>
        <p className="notice notice-info">{install.noProfileNote}</p>
        <div className="inline-actions">
          <Button onClick={onCloseConfirm} disabled={busy}>{install.back}</Button>
          <Button variant="primary" onClick={onConfirm} disabled={busy}>{install.confirm}</Button>
        </div>
      </Card>
    );
  }
  if (!canStart) {
    return null;
  }
  return (
    <Button variant="primary" onClick={onOpenConfirm} disabled={busy}>
      {install.action}
    </Button>
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
        <Button variant="ghost" onClick={() => { void copyLog(); }}>
          {copied ? install.logCopied : install.copyLog}
        </Button>
      </div>
      <pre className="install-log-body">{text}</pre>
      <p className="small muted">{install.logHint}</p>
    </div>
  );
}

import { useState } from "react";
import {
  exportDiagnosticBundle,
  type DiagnosticExportResult,
} from "../../api/diagnostics";
import type { AppMessages, Locale } from "../../i18n";
import { IconChevronRight, IconFileArchive } from "../ui/icons";

type DiagnosticsCopy = AppMessages["settings"]["diagnostics"];

export function DiagnosticsSummary({ copy, onOpen }: { copy: DiagnosticsCopy; onOpen: () => void }) {
  return (
    <section className="settings-section settings-update-summary" aria-labelledby="diagnostics-summary-title">
      <div>
        <h2 id="diagnostics-summary-title">{copy.title}</h2>
        <p>{copy.summary}</p>
      </div>
      <button type="button" className="settings-row-action" onClick={onOpen}>
        {copy.open}
        <IconChevronRight />
      </button>
    </section>
  );
}

export function DiagnosticsPanel({ copy, locale, onBack }: { copy: DiagnosticsCopy; locale: Locale; onBack: () => void }) {
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<DiagnosticExportResult | null>(null);
  const [failed, setFailed] = useState(false);

  const exportBundle = () => {
    setBusy(true);
    setFailed(false);
    setResult(null);
    void exportDiagnosticBundle()
      .then((next) => {
        if (next) setResult(next);
      })
      .catch(() => setFailed(true))
      .finally(() => setBusy(false));
  };

  return (
    <div className="settings-update-page settings-diagnostics-page">
      <button type="button" className="settings-update-back" onClick={onBack}>← {copy.back}</button>
      <header className="settings-update-header">
        <div>
          <h2>{copy.pageTitle}</h2>
          <p>{copy.description}</p>
        </div>
        <button type="button" className="button button-primary button-compact" disabled={busy} onClick={exportBundle}>
          <IconFileArchive />
          {busy ? copy.exporting : copy.export}
        </button>
      </header>

      <section className="settings-update-group" aria-label={copy.contentsTitle}>
        <div className="settings-update-group-heading">
          <div>
            <h3>{copy.contentsTitle}</h3>
            <p>{copy.contentsDescription}</p>
          </div>
        </div>
        <div className="settings-diagnostics-files">{copy.contents.join(" · ")}</div>
      </section>

      <p className="settings-update-policy">{copy.privacy}</p>
      {failed ? <div className="settings-update-error" role="alert">{copy.failed}</div> : null}
      {result ? (
        <div className="settings-diagnostics-success" role="status">
          <strong>{copy.succeeded}</strong>
          <span>{result.fileName}</span>
          <span>{(result.sizeBytes / 1024).toFixed(1)} KB · {new Date(result.exportedAtUnixMs).toLocaleString(locale)}</span>
        </div>
      ) : null}
    </div>
  );
}

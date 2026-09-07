import { useCallback, useEffect, useState } from "react";
import {
  cancelYorvaUpdate,
  checkYorvaUpdate,
  downloadYorvaUpdate,
  getYorvaUpdateStatus,
  installYorvaUpdate,
  isYorvaUpdateError,
  type YorvaUpdateStatus,
} from "../../api/updates";
import type { AppMessages } from "../../i18n";
import { IconChevronRight, IconRefresh } from "../ui/icons";

type UpdateCopy = AppMessages["settings"]["updates"];

function phaseText(status: YorvaUpdateStatus | null, copy: UpdateCopy): string {
  if (!status) return copy.loading;
  if (!status.verificationKeyConfigured) return copy.signingUnavailable;
  return copy.phases[status.phase];
}

function errorText(code: string | null | undefined, copy: UpdateCopy): string {
  if (!code) return copy.errors.UPDATE_STATE_FAILED;
  return copy.errors[code as keyof typeof copy.errors] ?? copy.errors.UPDATE_STATE_FAILED;
}

export function YorvaUpdateSummary({ copy, onOpen }: { copy: UpdateCopy; onOpen: () => void }) {
  const [status, setStatus] = useState<YorvaUpdateStatus | null>(null);

  useEffect(() => {
    let active = true;
    void getYorvaUpdateStatus()
      .then((next) => { if (active) setStatus(next); })
      .catch(() => { if (active) setStatus(null); });
    return () => { active = false; };
  }, []);

  return (
    <section className="settings-section settings-update-summary" aria-labelledby="yorva-update-summary-title">
      <div>
        <h2 id="yorva-update-summary-title">{copy.title}</h2>
        <p>{phaseText(status, copy)}</p>
      </div>
      <button type="button" className="settings-row-action" onClick={onOpen}>
        {copy.manage}
        <IconChevronRight />
      </button>
    </section>
  );
}

export function YorvaUpdatePanel({ copy, onBack }: { copy: UpdateCopy; onBack: () => void }) {
  const [status, setStatus] = useState<YorvaUpdateStatus | null>(null);
  const [busy, setBusy] = useState(false);
  const [errorCode, setErrorCode] = useState<string | null>(null);

  const refresh = useCallback(() => {
    setErrorCode(null);
    return getYorvaUpdateStatus()
      .then(setStatus)
      .catch((error: unknown) => setErrorCode(isYorvaUpdateError(error) ? error.code : "UPDATE_STATE_FAILED"));
  }, []);

  useEffect(() => {
    let active = true;
    void getYorvaUpdateStatus()
      .then((next) => { if (active) setStatus(next); })
      .catch((error: unknown) => {
        if (active) setErrorCode(isYorvaUpdateError(error) ? error.code : "UPDATE_STATE_FAILED");
      });
    return () => { active = false; };
  }, []);

  const run = (action: () => Promise<YorvaUpdateStatus>) => {
    setBusy(true);
    setErrorCode(null);
    if (action === downloadYorvaUpdate) {
      setStatus((current) => current ? { ...current, phase: "DOWNLOADING" } : current);
    }
    void action()
      .then(setStatus)
      .catch(async (error: unknown) => {
        await refresh();
        setErrorCode(isYorvaUpdateError(error) ? error.code : "UPDATE_STATE_FAILED");
      })
      .finally(() => setBusy(false));
  };

  const cancel = () => {
    void cancelYorvaUpdate().finally(() => void refresh());
  };

  const action = status?.phase === "AVAILABLE" || (status?.phase === "FAILED" && status.candidate)
    ? { label: copy.download, run: downloadYorvaUpdate }
    : status?.phase === "READY_TO_INSTALL"
      ? { label: copy.install, run: installYorvaUpdate }
      : { label: copy.check, run: checkYorvaUpdate };
  const canRun = status?.verificationKeyConfigured === true
    && status.phase !== "DOWNLOADING"
    && status.phase !== "INSTALLING"
    && status.phase !== "POSTCHECK";

  return (
    <div className="settings-update-page">
      <button type="button" className="settings-update-back" onClick={onBack}>← {copy.back}</button>
      <header className="settings-update-header">
        <div>
          <h2>{copy.pageTitle}</h2>
          <p>{copy.description}</p>
        </div>
        <button
          type="button"
          className="button button-secondary button-compact"
          disabled={busy || !canRun}
          onClick={() => run(action.run)}
        >
          <IconRefresh className={busy ? "is-spinning" : undefined} />
          {busy ? copy.working : action.label}
        </button>
      </header>

      {!status?.verificationKeyConfigured ? (
        <div className="settings-update-notice" role="status">
          <strong>{copy.internalCandidate}</strong>
          <span>{copy.signingExplanation}</span>
        </div>
      ) : null}
      {errorCode || status?.errorCode ? (
        <div className="settings-update-error" role="alert">{errorText(errorCode ?? status?.errorCode, copy)}</div>
      ) : null}

      <section className="settings-update-group" aria-label={copy.currentVersion}>
        <div className="settings-update-row">
          <span>{copy.currentVersion}</span>
          <strong>{status?.installedVersion ?? "—"}</strong>
        </div>
        <div className="settings-update-row">
          <span>{copy.status}</span>
          <strong>{phaseText(status, copy)}</strong>
        </div>
        <div className="settings-update-row">
          <span>{copy.source}</span>
          <strong>{copy.fixedSource}</strong>
        </div>
      </section>

      {status?.candidate ? (
        <section className="settings-update-group" aria-labelledby="yorva-update-candidate-title">
          <div className="settings-update-group-heading">
            <div>
              <h3 id="yorva-update-candidate-title">{copy.candidate}</h3>
              <p>{status.candidate.version} · {(status.candidate.sizeBytes / 1024 / 1024).toFixed(1)} MB</p>
            </div>
            <span className="settings-update-classification">
              {status.candidate.authenticodeRequired ? copy.publicRelease : copy.internalCandidate}
            </span>
          </div>
          <div className="settings-update-notes">
            <strong>{copy.releaseNotes}</strong>
            <p>{status.candidate.releaseNotes}</p>
          </div>
        </section>
      ) : null}

      {status?.phase === "DOWNLOADING" ? (
        <button type="button" className="button button-ghost button-compact" onClick={cancel}>{copy.cancel}</button>
      ) : null}
      <p className="settings-update-policy">{copy.securityPolicy}</p>
    </div>
  );
}

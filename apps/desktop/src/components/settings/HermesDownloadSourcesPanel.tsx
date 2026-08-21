import { useEffect, useState, type FormEvent } from "react";
import type { DaemonClient } from "../../api/client";
import type { HermesDownloadSources } from "../../api/types";
import type { AppMessages } from "../../i18n";

const emptySources: HermesDownloadSources = {
  hermesArchiveUrl: "",
  nodeArchiveUrl: "",
  npmArchiveUrl: "",
  pythonIndexUrl: "",
  npmRegistryUrl: "",
};

type Field = keyof HermesDownloadSources;
type State = "loading" | "idle" | "saving" | "saved" | "invalid" | "error";

export function HermesDownloadSourcesPanel({ copy, client }: { copy: AppMessages; client?: DaemonClient }) {
  const [sources, setSources] = useState<HermesDownloadSources>(emptySources);
  const [state, setState] = useState<State>("loading");

  useEffect(() => {
    if (!client) {
      return;
    }
    const controller = new AbortController();
    void client.getHermesDownloadSources(controller.signal)
      .then((value) => {
        setSources(value);
        setState("idle");
      })
      .catch(() => {
        if (!controller.signal.aborted) setState("error");
      });
    return () => controller.abort();
  }, [client]);

  const update = (field: Field, value: string) => {
    setSources((current) => ({ ...current, [field]: value }));
    if (state !== "loading") setState("idle");
  };

  const save = async (event: FormEvent) => {
    event.preventDefault();
    if (!client || !validSources(sources)) {
      setState("invalid");
      return;
    }
    setState("saving");
    try {
      const saved = await client.saveHermesDownloadSources(sources);
      setSources(saved);
      setState("saved");
    } catch {
      setState("error");
    }
  };

  const reset = async () => {
    if (!client) return;
    setState("saving");
    try {
      const defaults = await client.resetHermesDownloadSources();
      setSources(defaults);
      setState("saved");
    } catch {
      setState("error");
    }
  };

  const fields: Array<{ key: Field; label: string; help: string; group: "artifact" | "registry" }> = [
    { key: "hermesArchiveUrl", label: copy.settings.hermesArchiveLabel, help: copy.settings.hermesArchiveHelp, group: "artifact" },
    { key: "nodeArchiveUrl", label: copy.settings.nodeArchiveLabel, help: copy.settings.nodeArchiveHelp, group: "artifact" },
    { key: "npmArchiveUrl", label: copy.settings.npmArchiveLabel, help: copy.settings.npmArchiveHelp, group: "artifact" },
    { key: "pythonIndexUrl", label: copy.settings.pythonIndexLabel, help: copy.settings.pythonIndexHelp, group: "registry" },
    { key: "npmRegistryUrl", label: copy.settings.npmRegistryLabel, help: copy.settings.npmRegistryHelp, group: "registry" },
  ];
  const busy = state === "loading" || state === "saving";

  return (
    <form className="download-sources-form" onSubmit={(event) => { void save(event); }}>
      <section className="settings-section download-sources-heading" aria-labelledby="hermes-download-sources-title">
        <div className="settings-section-heading">
          <div>
            <h2 id="hermes-download-sources-title">{copy.settings.hermesSourcesTitle}</h2>
            <p>{copy.settings.hermesSourcesDescription}</p>
          </div>
          <span className="settings-default-badge">{copy.settings.hermesSourcesChinaDefault}</span>
        </div>
        <div className="settings-source-notice">
          <strong>{copy.settings.bundledFirstTitle}</strong>
          <span>{copy.settings.bundledFirstDescription}</span>
        </div>
      </section>

      {(["artifact", "registry"] as const).map((group) => (
        <section className="settings-source-group" key={group}>
          <div className="settings-source-group-heading">
            <h3>{group === "artifact" ? copy.settings.artifactSourcesTitle : copy.settings.dependencySourcesTitle}</h3>
            <p>{group === "artifact" ? copy.settings.artifactSourcesDescription : copy.settings.dependencySourcesDescription}</p>
          </div>
          <div className="settings-source-fields">
            {fields.filter((field) => field.group === group).map((field) => (
              <label className="settings-source-field" key={field.key} htmlFor={`source-${field.key}`}>
                <span>{field.label}</span>
                <input
                  id={`source-${field.key}`}
                  aria-describedby={`source-${field.key}-help`}
                  type="url"
                  required
                  value={sources[field.key]}
                  disabled={busy}
                  autoComplete="off"
                  spellCheck="false"
                  onChange={(event) => update(field.key, event.target.value)}
                />
                <small id={`source-${field.key}-help`}>{field.help}</small>
              </label>
            ))}
          </div>
        </section>
      ))}

      <p className="settings-source-policy">{copy.settings.sourcePolicyHint}</p>
      <div className="settings-source-actions">
        <div className="settings-source-status" aria-live="polite">
          {state === "loading" ? copy.settings.sourcesLoading : null}
          {state === "saved" ? copy.settings.sourcesSaved : null}
          {state === "invalid" ? copy.settings.sourcesInvalid : null}
          {state === "error" ? copy.settings.sourcesFailed : null}
        </div>
        <button type="button" className="button button-neutral" disabled={busy || !client} onClick={() => { void reset(); }}>
          {copy.settings.restoreChinaDefaults}
        </button>
        <button type="submit" className="button button-primary" disabled={busy || !client}>
          {state === "saving" ? copy.settings.sourcesSaving : copy.settings.saveSources}
        </button>
      </div>
    </form>
  );
}

function validSources(sources: HermesDownloadSources): boolean {
  return Object.values(sources).every((value) => {
    try {
      const parsed = new URL(value);
      return parsed.protocol === "https:" && parsed.hostname !== "" && parsed.username === "" && parsed.password === "" && parsed.search === "" && parsed.hash === "";
    } catch {
      return false;
    }
  });
}

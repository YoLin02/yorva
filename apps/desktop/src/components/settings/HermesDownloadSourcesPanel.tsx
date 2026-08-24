import { useEffect, useState, type FormEvent } from "react";
import type { DaemonClient } from "../../api/client";
import type { HermesDownloadSources } from "../../api/types";
import type { AppMessages } from "../../i18n";

const emptySources: HermesDownloadSources = {
  artifactPreference: "bundled-first",
  hermesArchiveUrl: "",
  nodeArchiveUrl: "",
  npmArchiveUrl: "",
  pythonArchiveUrl: "",
  pythonIndexUrl: "",
  npmRegistryUrl: "",
};

type Field = Exclude<keyof HermesDownloadSources, "artifactPreference">;
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
    { key: "pythonArchiveUrl", label: copy.settings.pythonArchiveLabel, help: copy.settings.pythonArchiveHelp, group: "artifact" },
    { key: "pythonIndexUrl", label: copy.settings.pythonIndexLabel, help: copy.settings.pythonIndexHelp, group: "registry" },
    { key: "npmRegistryUrl", label: copy.settings.npmRegistryLabel, help: copy.settings.npmRegistryHelp, group: "registry" },
  ];
  const busy = state === "loading" || state === "saving";

  return (
    <form className="download-sources-form" onSubmit={(event) => { void save(event); }}>
      <section className="settings-section" aria-labelledby="artifact-preference-title">
        <h2 id="artifact-preference-title">{copy.settings.artifactPreferenceTitle}</h2>
        <p>{copy.settings.artifactPreferenceDescription}</p>
        <div className="settings-segmented-control settings-source-preference" role="radiogroup" aria-label={copy.settings.artifactPreferenceTitle}>
          <button type="button" className={sources.artifactPreference === "bundled-first" ? "settings-segment is-active" : "settings-segment"} role="radio" aria-checked={sources.artifactPreference === "bundled-first"} disabled={busy} onClick={() => setSources((current) => ({ ...current, artifactPreference: "bundled-first" }))}>
            {copy.settings.preferBundled}
          </button>
          <button type="button" className={sources.artifactPreference === "online-first" ? "settings-segment is-active" : "settings-segment"} role="radio" aria-checked={sources.artifactPreference === "online-first"} disabled={busy} onClick={() => setSources((current) => ({ ...current, artifactPreference: "online-first" }))}>
            {copy.settings.preferOnline}
          </button>
        </div>
      </section>

      {(["artifact", "registry"] as const).map((group) => (
        <section className="settings-section settings-source-section" key={group}>
          <h2>{group === "artifact" ? copy.settings.artifactSourcesTitle : copy.settings.dependencySourcesTitle}</h2>
          <p>{group === "artifact" ? copy.settings.artifactSourcesDescription : copy.settings.dependencySourcesDescription}</p>
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
  if (sources.artifactPreference !== "bundled-first" && sources.artifactPreference !== "online-first") return false;
  return (["hermesArchiveUrl", "nodeArchiveUrl", "npmArchiveUrl", "pythonArchiveUrl", "pythonIndexUrl", "npmRegistryUrl"] as const).every((key) => {
    const value = sources[key];
    try {
      const parsed = new URL(value);
      return parsed.protocol === "https:" && parsed.hostname !== "" && parsed.username === "" && parsed.password === "" && parsed.search === "" && parsed.hash === "";
    } catch {
      return false;
    }
  });
}

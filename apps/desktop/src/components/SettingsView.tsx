import { useEffect, useState } from "react";
import { getDesktopPreferences, setDesktopPreferences, type DesktopPreferences } from "../api/desktopPreferences";
import type { AppMessages, Locale } from "../i18n";
import { messages, supportedLocales } from "../i18n";
import { IconMonitor, IconMoon, IconSun } from "./ui/icons";
import type { DaemonClient } from "../api/client";
import { HermesDownloadSourcesPanel } from "./settings/HermesDownloadSourcesPanel";

type SettingsTab = "general" | "advanced" | "diagnostics" | "about";

export function SettingsView({
  copy,
  locale,
  client,
  onLocaleChange,
}: {
  copy: AppMessages;
  locale: Locale;
  client?: DaemonClient;
  onLocaleChange: (locale: Locale) => void;
}) {
  const [activeTab, setActiveTab] = useState<SettingsTab>("general");
  const [desktopPreferences, setDesktopPreferenceState] = useState<DesktopPreferences>({
    launchOnLogin: true,
    closeToTray: true,
  });
  const [desktopPreferencesBusy, setDesktopPreferencesBusy] = useState(false);
  const [desktopPreferencesFailed, setDesktopPreferencesFailed] = useState(false);
  const tabs: Array<{ id: SettingsTab; label: string }> = [
    { id: "general", label: copy.settings.generalTab },
    { id: "advanced", label: copy.settings.advancedTab },
    { id: "diagnostics", label: copy.settings.diagnosticsTab },
    { id: "about", label: copy.settings.aboutTab },
  ];

  useEffect(() => {
    let active = true;
    void getDesktopPreferences()
      .then((preferences) => {
        if (active) setDesktopPreferenceState(preferences);
      })
      .catch(() => {
        if (active) setDesktopPreferencesFailed(true);
      });
    return () => {
      active = false;
    };
  }, []);

  const updateDesktopPreferences = (next: DesktopPreferences) => {
    setDesktopPreferencesBusy(true);
    setDesktopPreferencesFailed(false);
    void setDesktopPreferences(next)
      .then(setDesktopPreferenceState)
      .catch(() => setDesktopPreferencesFailed(true))
      .finally(() => setDesktopPreferencesBusy(false));
  };

  return (
    <div className="settings-page">
      <div className="settings-tabs" role="tablist" aria-label={copy.pages.settings.title}>
        {tabs.map((tab) => (
          <button
            type="button"
            id={`settings-tab-${tab.id}`}
            key={tab.id}
            className={activeTab === tab.id ? "settings-tab is-active" : "settings-tab"}
            role="tab"
            aria-selected={activeTab === tab.id}
            aria-controls={`settings-panel-${tab.id}`}
            onClick={() => setActiveTab(tab.id)}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {activeTab === "general" ? (
        <div
          id="settings-panel-general"
          className="settings-general"
          role="tabpanel"
          aria-labelledby="settings-tab-general"
        >
          <section className="settings-section" aria-labelledby="language-title">
            <h2 id="language-title">{copy.settings.languageLegend}</h2>
            <p>{copy.settings.languageDescription}</p>
            <div className="settings-segmented-control" role="radiogroup" aria-label={copy.settings.languageLegend}>
              {supportedLocales.map((option) => (
                <button
                  type="button"
                  key={option}
                  className={locale === option ? "settings-segment is-active" : "settings-segment"}
                  role="radio"
                  aria-checked={locale === option}
                  onClick={() => onLocaleChange(option)}
                >
                  {messages[option].languageName}
                </button>
              ))}
            </div>
          </section>

          <section className="settings-section" aria-labelledby="appearance-title">
            <h2 id="appearance-title">{copy.settings.appearance}</h2>
            <p>{copy.settings.appearanceDescription}</p>
            <div className="settings-segmented-control settings-theme-control" aria-label={copy.settings.appearance}>
              <button type="button" className="settings-segment is-active" aria-pressed="true">
                <IconSun />
                {copy.settings.lightTheme}
              </button>
              <button type="button" className="settings-segment" title={copy.settings.themeUnavailable} disabled>
                <IconMoon />
                {copy.settings.darkTheme}
              </button>
              <button type="button" className="settings-segment" title={copy.settings.themeUnavailable} disabled>
                <IconMonitor size={15} />
                {copy.settings.systemTheme}
              </button>
            </div>
          </section>

          <section className="settings-section settings-window-section" aria-labelledby="window-behavior-title">
            <h2 id="window-behavior-title">{copy.settings.windowBehavior}</h2>
            <p>{copy.settings.windowBehaviorDescription}</p>
            <div className="settings-behavior-list">
              <div className="settings-behavior-row">
                <div>
                  <strong>{copy.settings.launchOnLogin}</strong>
                  <span>{copy.settings.launchOnLoginDescription}</span>
                </div>
                <button
                  type="button"
                  role="switch"
                  aria-label={copy.settings.launchOnLogin}
                  aria-checked={desktopPreferences.launchOnLogin}
                  className={desktopPreferences.launchOnLogin ? "settings-toggle is-active" : "settings-toggle"}
                  disabled={desktopPreferencesBusy}
                  onClick={() => updateDesktopPreferences({
                    ...desktopPreferences,
                    launchOnLogin: !desktopPreferences.launchOnLogin,
                  })}
                >
                  <span />
                </button>
              </div>
              <div className="settings-behavior-row">
                <div>
                  <strong>{copy.settings.closeToTray}</strong>
                  <span>{copy.settings.closeToTrayDescription}</span>
                </div>
                <button
                  type="button"
                  role="switch"
                  aria-label={copy.settings.closeToTray}
                  aria-checked={desktopPreferences.closeToTray}
                  className={desktopPreferences.closeToTray ? "settings-toggle is-active" : "settings-toggle"}
                  disabled={desktopPreferencesBusy}
                  onClick={() => updateDesktopPreferences({
                    ...desktopPreferences,
                    closeToTray: !desktopPreferences.closeToTray,
                  })}
                >
                  <span />
                </button>
              </div>
            </div>
            {desktopPreferencesFailed ? (
              <p className="settings-save-error" role="alert">{copy.settings.desktopPreferencesFailed}</p>
            ) : null}
          </section>
        </div>
      ) : activeTab === "advanced" ? (
        <div
          id="settings-panel-advanced"
          className="settings-advanced"
          role="tabpanel"
          aria-labelledby="settings-tab-advanced"
        >
          <HermesDownloadSourcesPanel copy={copy} client={client} />
        </div>
      ) : (
        <div
          id={`settings-panel-${activeTab}`}
          className="settings-empty-panel"
          role="tabpanel"
          aria-labelledby={`settings-tab-${activeTab}`}
        />
      )}
    </div>
  );
}

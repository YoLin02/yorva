import type { AppMessages, Locale } from "../i18n";
import { messages, supportedLocales } from "../i18n";

export function SettingsView({
  copy,
  locale,
  onLocaleChange,
}: {
  copy: AppMessages;
  locale: Locale;
  onLocaleChange: (locale: Locale) => void;
}) {
  return (
    <section className="panel settings-panel" aria-labelledby="language-title">
      <div className="panel-heading">
        <div>
          <p className="section-kicker">{copy.settings.savedAutomatically}</p>
          <h2 id="language-title">{copy.settings.language}</h2>
          <p>{copy.settings.languageDescription}</p>
        </div>
      </div>
      <fieldset className="language-options">
        <legend>{copy.settings.languageLegend}</legend>
        {supportedLocales.map((option) => (
          <label key={option} className={locale === option ? "language-option language-option-active" : "language-option"}>
            <input
              type="radio"
              name="locale"
              value={option}
              checked={locale === option}
              onChange={() => onLocaleChange(option)}
            />
            <span>{messages[option].languageName}</span>
            <small>{option}</small>
          </label>
        ))}
      </fieldset>
    </section>
  );
}

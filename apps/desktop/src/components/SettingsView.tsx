import type { AppMessages, Locale } from "../i18n";
import { messages, supportedLocales } from "../i18n";
import { Card } from "./ui/Card";
import { IconGlobe } from "./ui/icons";

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
    <Card className="settings-card" aria-labelledby="language-title">
      <div className="panel-heading">
        <div className="icon-tile icon-tile-ok">
          <IconGlobe size={24} />
        </div>
        <div>
          <p className="page-kicker">{copy.settings.savedAutomatically}</p>
          <h2 id="language-title">{copy.settings.language}</h2>
          <p className="panel-copy">{copy.settings.languageDescription}</p>
        </div>
      </div>
      <fieldset className="language-options">
        <legend>{copy.settings.languageLegend}</legend>
        {supportedLocales.map((option) => (
          <label key={option} className={locale === option ? "language-option is-active" : "language-option"}>
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
    </Card>
  );
}

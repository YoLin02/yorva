import { SettingsView } from "../components/SettingsView";
import type { AppMessages, Locale } from "../i18n";

export function SettingsPage({
  copy,
  locale,
  onLocaleChange,
}: {
  copy: AppMessages;
  locale: Locale;
  onLocaleChange: (locale: Locale) => void;
}) {
  return <SettingsView copy={copy} locale={locale} onLocaleChange={onLocaleChange} />;
}

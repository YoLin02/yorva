import { SettingsView } from "../components/SettingsView";
import type { DaemonClient } from "../api/client";
import type { AppMessages, Locale } from "../i18n";

export function SettingsPage({
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
  return <SettingsView copy={copy} locale={locale} client={client} onLocaleChange={onLocaleChange} />;
}

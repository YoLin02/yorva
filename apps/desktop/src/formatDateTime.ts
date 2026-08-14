import type { Locale } from "./i18n";

export function formatDateTime(value: string, locale: Locale, timeZone?: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;

  return new Intl.DateTimeFormat(locale, {
    year: "numeric",
    month: locale === "zh-CN" ? "2-digit" : "short",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hourCycle: "h23",
    timeZone,
  }).format(date);
}

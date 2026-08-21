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

export function formatRelativeTime(value: string, locale: Locale, now = Date.now()): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;

  const elapsedSeconds = Math.max(0, Math.floor((now - date.getTime()) / 1000));
  if (elapsedSeconds < 60) return locale === "zh-CN" ? "刚刚" : "Just now";

  const formatter = new Intl.RelativeTimeFormat(locale, { numeric: "always" });
  const minutes = Math.floor(elapsedSeconds / 60);
  if (minutes < 60) return formatter.format(-minutes, "minute");
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return formatter.format(-hours, "hour");
  const days = Math.floor(hours / 24);
  if (days <= 30) return formatter.format(-days, "day");

  return new Intl.DateTimeFormat(locale, {
    year: "numeric",
    month: locale === "zh-CN" ? "numeric" : "short",
    day: "numeric",
  }).format(date);
}

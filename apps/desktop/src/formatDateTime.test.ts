import { describe, expect, it } from "vitest";
import { formatDateTime, formatRelativeTime } from "./formatDateTime";

describe("formatDateTime", () => {
  it("formats the same UTC value with an explicit locale and time zone", () => {
    const value = "2026-08-14T10:11:57Z";
    expect(formatDateTime(value, "zh-CN", "Asia/Shanghai")).toBe("2026/08/14 18:11:57");
    expect(formatDateTime(value, "en-US", "Asia/Shanghai")).toBe("Aug 14, 2026, 18:11:57");
  });

  it("does not depend on the runner time zone when one is supplied", () => {
    const value = "2026-08-14T10:11:57Z";
    expect(formatDateTime(value, "en-US", "UTC")).toBe("Aug 14, 2026, 10:11:57");
  });

  it("preserves an invalid contract value for diagnostics", () => {
    expect(formatDateTime("invalid", "en-US", "UTC")).toBe("invalid");
  });
});

describe("formatRelativeTime", () => {
  const now = new Date("2026-08-21T10:00:00Z").getTime();

  it("uses compact relative labels for recent syncs", () => {
    expect(formatRelativeTime("2026-08-21T09:59:40Z", "zh-CN", now)).toBe("刚刚");
    expect(formatRelativeTime("2026-08-21T09:55:00Z", "zh-CN", now)).toBe("5分钟前");
    expect(formatRelativeTime("2026-08-21T07:00:00Z", "en-US", now)).toBe("3 hours ago");
  });

  it("falls back to a date without a detailed timestamp for older syncs", () => {
    expect(formatRelativeTime("2026-06-01T07:08:09Z", "en-US", now)).toBe("Jun 1, 2026");
  });
});

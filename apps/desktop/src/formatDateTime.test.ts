import { describe, expect, it } from "vitest";
import { formatDateTime } from "./formatDateTime";

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

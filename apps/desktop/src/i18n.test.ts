import { describe, expect, it } from "vitest";
import { loadLocale, localeStorageKey, messages, resolveLocale, saveLocale } from "./i18n";

describe("i18n", () => {
  it("prefers a persisted supported locale and otherwise follows the system locale", () => {
    expect(resolveLocale("zh-CN", "en-US")).toBe("zh-CN");
    expect(resolveLocale(null, "zh-Hans-CN")).toBe("zh-CN");
    expect(resolveLocale("unsupported", "fr-FR")).toBe("en-US");
  });

  it("persists and restores the selected locale", () => {
    saveLocale("zh-CN");
    expect(window.localStorage.getItem(localeStorageKey)).toBe("zh-CN");
    expect(loadLocale()).toBe("zh-CN");
  });

  it("provides every Runtime state in both locales", () => {
    expect(Object.keys(messages["zh-CN"].hermes.states).sort()).toEqual(Object.keys(messages["en-US"].hermes.states).sort());
    expect(messages["zh-CN"].hermes.states.BROKEN_EXECUTABLE.title).not.toBe(messages["zh-CN"].hermes.states.NOT_INSTALLED.title);
  });
});

import type { Locale } from "../i18n";

export type RuntimeId = "hermes" | "openclaw";
const runtimeNames: Record<RuntimeId, string> = { hermes: "Hermes Agent", openclaw: "OpenClaw" };

export function RuntimePicker({ value, disabled, locale, onChange }: {
  value: RuntimeId;
  disabled: boolean;
  locale: Locale;
  onChange: (value: RuntimeId) => void;
}) {
  return (
    <div className="runtime-picker" role="group" aria-label={locale === "zh-CN" ? "选择 Runtime" : "Select Runtime"}>
      {(Object.keys(runtimeNames) as RuntimeId[]).map((id) => (
        <button key={id} type="button" aria-pressed={value === id} disabled={disabled} onClick={() => onChange(id)}>
          {runtimeNames[id]}
        </button>
      ))}
    </div>
  );
}

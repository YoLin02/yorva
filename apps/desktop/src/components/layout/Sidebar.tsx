import { useState } from "react";
import type { AppMessages, Locale, PageId } from "../../i18n";
import { messages } from "../../i18n";
import { IconBox, IconChevronDown, IconChevronUp, IconGlobe, IconHome, IconSettings } from "../ui/icons";
import { YorvaLogo } from "./YorvaLogo";

type SidebarProps = {
  activePage: PageId;
  copy: AppMessages;
  locale: Locale;
  onNavigate: (page: PageId) => void;
  onLocaleChange: (locale: Locale) => void;
};

const pageOrder: PageId[] = ["dashboard", "runtimes", "settings"];

const pageIcons: Record<PageId, typeof IconHome> = {
  dashboard: IconHome,
  runtimes: IconBox,
  settings: IconSettings,
};

export function Sidebar({
  activePage,
  copy,
  locale,
  onNavigate,
  onLocaleChange,
}: SidebarProps) {
  const [languageOpen, setLanguageOpen] = useState(true);

  return (
    <aside className="sidebar">
      <div className="sidebar-top">
        <div className="sidebar-brand">
          <YorvaLogo />
        </div>

        <nav className="sidebar-nav" aria-label={copy.navigationLabel}>
          {pageOrder.map((pageId) => {
            const Icon = pageIcons[pageId];
            const active = activePage === pageId;
            return (
              <button
                type="button"
                key={pageId}
                className={active ? "sidebar-link is-active" : "sidebar-link"}
                aria-current={active ? "page" : undefined}
                onClick={() => onNavigate(pageId)}
              >
                <Icon />
                <span>{copy.pages[pageId].navigation}</span>
              </button>
            );
          })}
        </nav>
      </div>

      <div className="sidebar-footer">
        <div className="language-switch">
          <button
            type="button"
            className="language-switch-label"
            aria-expanded={languageOpen}
            aria-label={copy.switchLanguage}
            onClick={() => setLanguageOpen((open) => !open)}
          >
            <span className="language-switch-title">
              <IconGlobe />
              <span>{copy.settings.language}</span>
            </span>
            {languageOpen ? <IconChevronUp /> : <IconChevronDown />}
          </button>
          {languageOpen && (
            <div className="language-switch-options">
              <button
                type="button"
                className={locale === "en-US" ? "language-chip is-active" : "language-chip"}
                onClick={() => onLocaleChange("en-US")}
              >
                {messages["en-US"].languageShort}
              </button>
              <button
                type="button"
                className={locale === "zh-CN" ? "language-chip is-active" : "language-chip"}
                onClick={() => onLocaleChange("zh-CN")}
              >
                {messages["zh-CN"].languageShort}
              </button>
            </div>
          )}
        </div>
      </div>
    </aside>
  );
}

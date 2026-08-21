import type { AppMessages, Locale, PageId } from "../../i18n";
import { messages } from "../../i18n";
import { IconBox, IconGlobe, IconHome, IconLayers, IconSettings } from "../ui/icons";
import { YorvaLogo } from "./YorvaLogo";

type SidebarProps = {
  activePage: PageId;
  copy: AppMessages;
  locale: Locale;
  onNavigate: (page: PageId) => void;
  onLocaleChange: (locale: Locale) => void;
};

const pageOrder: PageId[] = ["dashboard", "runtimes", "instances", "settings"];

const pageIcons: Record<PageId, typeof IconHome> = {
  dashboard: IconHome,
  runtimes: IconBox,
  instances: IconLayers,
  settings: IconSettings,
};

export function Sidebar({
  activePage,
  copy,
  locale,
  onNavigate,
  onLocaleChange,
}: SidebarProps) {
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
          <div className="language-switch-label">
            <span className="language-switch-title">
              <IconGlobe />
              <span>{copy.settings.language}</span>
            </span>
          </div>
          <div className="language-switch-options" role="group" aria-label={copy.switchLanguage}>
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
        </div>
      </div>
    </aside>
  );
}

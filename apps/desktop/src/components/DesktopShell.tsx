import type { ReactNode } from "react";
import { APP_NAME } from "../appMetadata";
import type { AppMessages, Locale, PageId } from "../i18n";

type DesktopShellProps = {
  activePage: PageId;
  copy: AppMessages;
  locale: Locale;
  nodeVersion?: string;
  onNavigate: (page: PageId) => void;
  onToggleLocale: () => void;
  targetLocaleLabel: string;
  children: ReactNode;
};

const pageOrder: PageId[] = ["dashboard", "runtimes", "settings"];

export function DesktopShell({
  activePage,
  copy,
  locale,
  nodeVersion,
  onNavigate,
  onToggleLocale,
  targetLocaleLabel,
  children,
}: DesktopShellProps) {
  const page = copy.pages[activePage];

  return (
    <div className="app-shell" lang={locale}>
      <aside className="sidebar">
        <div className="brand-block">
          <span className="brand-mark" aria-hidden="true">Y</span>
          <div>
            <strong>{APP_NAME}</strong>
            <span>{copy.brandTagline}</span>
          </div>
        </div>

        <nav className="sidebar-nav" aria-label={copy.navigationLabel}>
          {pageOrder.map((pageId) => (
            <button
              type="button"
              key={pageId}
              className={activePage === pageId ? "nav-item nav-item-active" : "nav-item"}
              aria-current={activePage === pageId ? "page" : undefined}
              onClick={() => onNavigate(pageId)}
            >
              <span className="nav-indicator" aria-hidden="true" />
              {copy.pages[pageId].navigation}
            </button>
          ))}
        </nav>

        <div className="sidebar-footer">
          <span className="sidebar-version">{nodeVersion ? `yorvad ${nodeVersion}` : copy.versionUnavailable}</span>
          <button type="button" className="language-shortcut" onClick={onToggleLocale} aria-label={copy.switchLanguage}>
            <span aria-hidden="true">文/A</span>
            {targetLocaleLabel}
          </button>
        </div>
      </aside>

      <main className="content-shell">
        <header className="page-header">
          <p className="eyebrow">{APP_NAME} / {page.navigation}</p>
          <h1>{page.title}</h1>
          <p>{page.description}</p>
        </header>
        <div className="page-content">{children}</div>
      </main>
    </div>
  );
}

import type { ReactNode } from "react";
import type { AppMessages, Locale, PageId } from "../../i18n";
import { Sidebar } from "./Sidebar";

type DesktopShellProps = {
  activePage: PageId;
  copy: AppMessages;
  locale: Locale;
  onNavigate: (page: PageId) => void;
  onLocaleChange: (locale: Locale) => void;
  hidePageHeader?: boolean;
  children: ReactNode;
};

export function DesktopShell({
  activePage,
  copy,
  locale,
  onNavigate,
  onLocaleChange,
  hidePageHeader = false,
  children,
}: DesktopShellProps) {
  const page = copy.pages[activePage];

  return (
    <div className="desktop-shell" lang={locale}>
      <Sidebar
        activePage={activePage}
        copy={copy}
        locale={locale}
        onNavigate={onNavigate}
        onLocaleChange={onLocaleChange}
      />
      <main className="page">
        {!hidePageHeader ? <header className="page-header">
          <h1 className="page-title">{page.title}</h1>
        </header> : null}
        {children}
      </main>
    </div>
  );
}

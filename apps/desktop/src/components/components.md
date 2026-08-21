# Desktop components

Visual language follows the provided Yorva reference: light slate canvas, white rounded cards, emerald accent `#10B981` / `#1b7a48`.

Daemon state is not stored in a React context. Pages receive query-backed props from `App.tsx`.

## Folders

- `ui/` — Lucide wrappers, `HermesMark`, `Button`, `Card`, `Badge`, `StatusDot` (`typography.md` records the type and icon spec)
- `layout/` — `YorvaLogo`, `Sidebar`, `DesktopShell`
- `dashboard/` — compact status cards, node/platform details, discovery banner
- feature views stay next to their tests: Node, Hermes discovery/install/prereq, Settings
- `src/pages/` — Dashboard / Runtime / Settings composition only

Do not port reference IBC chains, fake install modals, or mock Hermes config into these components.

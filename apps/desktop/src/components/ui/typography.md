# Typography and icons

Native system font stacks and a two-part icon set. Do not bundle web fonts.

## Type

English / Latin:

- macOS / iOS: `-apple-system`, `SF Pro Text` (body), `SF Pro Display` (titles and the YORVA wordmark)
- Windows: `Segoe UI`, `Segoe UI SC`
- fallback: `Roboto`, `Helvetica Neue`, `Arial`, `sans-serif`

Chinese:

- macOS: `PingFang SC`
- Windows: `Microsoft YaHei`, `Segoe UI SC`
- Linux: `Noto Sans CJK SC`, `Source Han Sans SC`

Monospace (Node ID, yorvad version, paths, install logs):

- `ui-monospace`, `SFMono-Regular`, `Menlo`, `Monaco`, `Consolas`, `Liberation Mono`, `monospace`

YORVA wordmark: `font-weight: 900` and `letter-spacing: 0.05em`.

Tokens live in `src/styles.css` as `--yorva-font`, `--yorva-display`, and `--yorva-mono`.

## Icons

Standard UI icons come from `lucide-react` at stroke `1.75`.

| Surface | Lucide name | Use |
| --- | --- | --- |
| Nav | `Home` / `Box` / `Settings` | Dashboard, Runtimes, Settings |
| Language | `Globe` + `ChevronUp` / `ChevronDown` | Locale fold |
| Local Node | `Monitor` | Node hardware |
| Recheck | `RotateCw` | Discovery retry |
| Card jump | `ChevronRight` | Node info / platform |
| Status | `Check` / `CheckCircle2` / `AlertTriangle` / `Loader2` / `XCircle` | Installed, unsupported, checking, broken |
| Notice | `Info` | Discovery hint |
| Install CTA | `Zap` | Navigate to the real Hermes install surface |

Custom SVGs (not Lucide):

- `YorvaLogo` — emerald folded polygons `#34D399` / `#10B981` / `#059669` / `#047857`
- `HermesMark` — winged messenger; amber `#F59E0B` when not ready, emerald `#10B981` when ready (`currentColor` from the tile)

`FileCode`, `Terminal`, `FolderOpen`, `Play`, and `Pause` stay unused until a later Runtime-management phase owns those surfaces.

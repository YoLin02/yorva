# Desktop library (`src/lib`)

Small UI helpers only. Runtime discovery, installation and Node state stay in `src/api` and TanStack Query.

## Files

- `cn(...)` — join optional class names without a CSS-in-JS runtime.

Icons use `lucide-react` plus the custom `YorvaLogo` and `HermesMark` SVGs. There is no mock Tauri IPC layer. Native window chrome stays with Tauri; type tokens live in `src/styles.css`.

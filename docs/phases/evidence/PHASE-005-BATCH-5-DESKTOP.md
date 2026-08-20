# Phase 5 Batch 5 — Desktop Integration Evidence

- Date: 2026-08-19
- Surface: existing Instances experience
- Gate: PASS

## Delivered contract

- The existing Instances page exposes a Models panel only for an `AVAILABLE`
  Instance. No top-level navigation, client state store or date formatter was
  added.
- The panel presents all eight qualified presets in China-first and global
  groups, displays adapter-owned safe help text, and supports recommended or
  bounded manual model IDs without exposing Hermes config keys, environment
  names, paths, endpoints or command arguments.
- Save uses the single write-only credential request when a new key is present
  and the secret-free config request otherwise. Save never starts validation.
- The API key remains page-local password state, is cleared after submission,
  Provider changes and unmount, and is absent from TanStack Query keys/cache,
  browser storage and URLs.
- Test connection is a separate explicit action. Active validation Operations
  can be recovered and cancelled, terminal Operation events refresh model
  metadata, and only normalized state/error codes plus safe guidance are shown.
- English and Simplified Chinese messages, text status indicators, local-time
  timestamps and unavailable-Instance disabling reuse existing application
  conventions.

## Focused verification

```text
pnpm --filter @yorva/desktop typecheck    PASS
pnpm --filter @yorva/desktop lint         PASS
pnpm --filter @yorva/desktop test         PASS (19 files, 79 tests)
pnpm --filter @yorva/desktop build        PASS
git diff --check                          PASS
```

Tests cover the China-first bilingual catalog, explicit-only validation,
local-time metadata, safe failed-validation guidance, generated client request
shape, Instance availability gating and password clearing/non-persistence.

Full repository, dependency, Windows smoke and exact-candidate checks continue
under the Phase 5 audit gate.

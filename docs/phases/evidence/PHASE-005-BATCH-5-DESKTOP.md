# Phase 5 Batch 5 — Desktop Integration Evidence

- Date: 2026-08-20
- Surface: existing Instances experience
- Gate: PASS

## Delivered contract

- The existing Instances page exposes a Models panel only for an `AVAILABLE`
  Instance and disables model mutations when the pinned model surface version
  is unsupported. No top-level navigation, client state store or date
  formatter was added.
- The panel presents all eight qualified presets in China-first and global
  groups, displays localized safe help text, and supports recommended or
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
pnpm --filter @yorva/desktop test         PASS (19 files, 80 tests)
pnpm --filter @yorva/desktop build        PASS
git diff --check                          PASS
```

Tests cover the China-first bilingual catalog, explicit-only validation,
local-time metadata, safe failed-validation guidance, generated client request
shape, Instance availability gating and password clearing/non-persistence.

Full repository, dependency and Windows smoke checks passed locally. Immutable
candidate `dd7c6c2f47a2b3c7331ebc810b1eb2b003ab59a9` passed exact-candidate CI
run `32343964969`. Main merge `c45a231060400cf21e41730f88ccdeab443b8a4f`
passed final-main CI run `32346079074` and Windows MSI run `32346079072`.
`AUDIT-005R1` is PASS.

# Phase 8 B5 — Sanitized Diagnostic Bundle Evidence

> Date: 2026-09-04  
> Branch: `phase/p8-local-product-hardening`  
> Candidate: B5 Batch commit  
> Packaging: non-MSI Desktop release build

## Implemented contract

- Authenticated `POST /api/v1/diagnostics/bundle` returns one complete
  `application/zip` response with `Cache-Control: no-store`.
- The daemon fixes all nine archive entries:
  `manifest.json`, `version.json`, `node-summary.json`, `runtime-summary.json`,
  `instance-summary.json`, `operations.json`, `schema.json`,
  `logs/node.ndjson`, and `redaction-report.json`.
- Bounds are enforced again at the bundle owner: 100 Instances, 50 Operations,
  512 log records, seven days, 64 KiB per input log line, 2 MiB uncompressed and
  2 MiB compressed.
- Node, hostname, Instance/Profile, Operation and target identifiers are represented
  by truncated SHA-256 labels. Runtime executable paths, warning text, operation
  messages/errors and unrecognized log fields are not exported.
- Only fixed YORVA `logs/install.ndjson` input is eligible. Each included line must be
  valid JSON in the time window; only fixed scalar fields survive. Secret/path-like
  values are replaced and the counts are recorded in `redaction-report.json`.
- The archive never reads raw SQLite content, environment variables, Hermes files,
  arbitrary paths or arbitrary user files.
- Tauri obtains the in-memory daemon session itself, calls only the fixed endpoint,
  rejects oversized/non-ZIP bodies, and exposes one native Save As command. It writes
  a same-directory private partial, flushes it, atomically replaces the selected file,
  and removes the partial on failure. React never receives the destination path or
  bearer token.
- Closing Save As returns `null` and leaves the page idle. Success is shown only after
  atomic publication and includes filename, byte size and completion time.

## Verification

Passed:

- `go test ./...`
- `go vet ./...`
- diagnostic schema, fixed-entry, record/time/size-bound, canary, hashed-identifier,
  stable-error, authenticated-route and method-contract tests
- `cargo test` — 26 passed, including atomic replacement and post-write failure cleanup
- `cargo clippy --all-targets --all-features -- -D warnings`
- Desktop Vitest — 24 files / 146 tests, including dedicated-page, Save As cancel and
  successful-result behavior
- Desktop TypeScript typecheck and ESLint
- OpenAPI regeneration and Redocly lint
- Vite production build
- `tauri build --no-bundle`

## Real Desktop smoke

The exact current non-MSI release executable started with its packaged `yorvad`. In the
Chinese UI, Settings → Diagnostics opened the dedicated export page without horizontal
overflow. Selecting **Export diagnostics** completed the live daemon request and opened
the native Windows Save As dialog; this proves the production Desktop/Tauri/authenticated
daemon/archive-generation handoff. Archive entry and redaction inspection runs against
the real ZIP encoder in the Go tests, while the native atomic filesystem publication is
exercised by the Rust tests.

No host restart, WSL restart, MSI creation, telemetry, upload or arbitrary filesystem
surface was used. The only build warning is the already-known Vite chunk-size advisory;
Windows linker output is informational.

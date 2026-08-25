# Phase 7 Wave 12 — B5–B9 Integrated Implementation Gate

- Date: 2026-08-25
- Scope: integrated MCP Operation boundary, Backup Create/Delete, Restore, Upgrade/Rollback
  Operation skeleton, Desktop state/recovery UX and shared conflict/restart handling
- Local result: PASS
- Freeze result: NOT YET ELIGIBLE

## Implemented boundary

- MCP mutations use closed preset/server/tool identifiers, write-only credentials,
  durable Operations, bounded execution and cancellation. The production MCP catalog
  remains empty and `mcpMutate` remains false until a preset completes live qualification.
- Backup destinations are issued only by the native Tauri Save dialog and cross the
  private parent control channel as one-use, expiring opaque references. HTTP never
  accepts or returns a path.
- Hermes `0.20.5` Runtime backup creates a canonical bounded snapshot, encrypts it with
  an OS-secret-store-backed age identity, verifies the final artifact, then indexes safe
  metadata. Index failure removes the verified unindexed publication.
- Restore re-hashes and authenticates the indexed artifact before bounded extraction,
  requires stopped Profiles, uses an atomic filesystem transaction, postchecks the
  Runtime, rolls back failures and recovers every recorded rename window at startup.
- Runtime-wide Backup/Restore/Upgrade Operations conflict atomically with affected
  Instance lifecycle, Channel, Skill and MCP Operations. Synchronous model mutations
  share the Runtime installation lock and observe the durable Runtime-wide Operation.
- Upgrade/Rollback mutations remain unwired and capability-false because exact
  `0.20.2 -> 0.20.5` data compatibility and safe rollback are still NO-GO in the preserved
  B8 qualification record. The read-only plan remains available.

## Defects found and fixed during the integrated review

- Backup IDs used URL-safe Base64 although the frozen manifest requires a 22-character
  lowercase Base32 body.
- Backup creation emitted fractional timestamps although canonical manifests require
  UTC whole seconds.
- Restore startup recovery skipped the exact window where the active root was absent
  between atomic renames, and did not distinguish activation-before/after old-tree
  cleanup.
- A published encrypted artifact could remain orphaned when the index insert failed.
- Rollback Operations compared a successful `ROLLED_BACK` result with the Upgrade
  success state, and the worker did not re-plan immediately before mutation.
- MCP long work had a timeout but no user cancellation path.

Each item has a focused regression test or is covered by the integrated package tests.

## Local Gate

All commands passed on the integrated working tree:

```text
go test ./...
go vet ./...
pnpm typecheck
pnpm lint
pnpm test                 # 24 files / 120 tests
pnpm build
pnpm api:lint
cargo fmt --check
cargo test                # 13 tests
cargo check
pnpm audit --audit-level low
cargo audit
git diff --check
```

`pnpm audit` found no known vulnerabilities. `cargo audit` found no vulnerability and
reported the same 17 allowed unmaintained/unsound warnings already recorded for the
existing dependency graph.

The new Backup/Restore regression set includes an encrypted create → mutate local data →
decrypt/restore round trip, index-insert cleanup, interrupted transaction recovery and
ambiguous-state refusal. It uses temporary local data and no real account or credential.

## Exact-candidate CI follow-up

- Candidate `95fddf6acd3c885c0f7aa758251d7a6b2eb518c7`, CI run `32839147445`:
  `Web and API contract` passed; `Go Node` failed in
  `TestMCPManagementCancelsRunningOperation` because cancellation interrupted the adapter
  before the durable `CANCELLED` state was published, allowing the worker to win the
  terminal-state race. The original failed run is retained.
- The fix publishes `CANCELLED` first and only then signals the adapter context. The
  existing cancellation regression passed 100 consecutive local executions. A new exact
  candidate CI run is required before this gate can pass.

## Remaining freeze gates

This evidence does not claim B5 or B8 product qualification and does not replace Windows
manual smoke, exact-candidate CI or independent B10 audit:

1. B5 needs one reviewed HTTPS MCP preset with live credential/session/tool-readback and
   Windows Profile evidence before production wiring.
2. B8 needs exact source/final-path, data compatibility, lifecycle/channel and safe
   rollback evidence before mutation wiring.
3. B6/B7 still need Windows native picker, encrypted artifact, destructive Restore and
   daemon-restart smoke on the immutable candidate.
4. The committed candidate must pass exact-commit CI, followed by the B10 audit and any
   required fresh re-audit, before merge, final-main CI or the Phase 7 baseline tag.

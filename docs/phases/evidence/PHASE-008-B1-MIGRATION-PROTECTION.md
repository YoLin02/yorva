# Phase 8 B1 — Migration Protection Evidence

Date: 2026-09-03
Branch: `phase/p8-local-product-hardening`
Starting baseline: `72ec369` (P8-B0)

## Implemented behavior

- activates the stable Desktop identifier `com.yorva.desktop`;
- copies a valid legacy `com.yorva.desktop.dev` data root through a same-volume staging
  directory before daemon startup, preserving the legacy root;
- refuses to merge two unconfirmed roots and validates the completion marker;
- advances SQLite from Phase 7 schema 016 to schema 017;
- creates a consistent `VACUUM INTO` protection database before pending migrations;
- records source/target schema, SHA-256, state, timestamps and stable error code in a
  bounded sidecar state file;
- verifies the final ledger, `quick_check` and `foreign_key_check`;
- restores the verified source after an injected or interrupted failure;
- enters `DATABASE_MIGRATION_RECOVERY_REQUIRED` when recovery evidence is tampered;
- retains only the three newest closed-name protection files and never deletes unknown
  files.

## Focused verification

```text
go test ./internal/persistence/sqlite
PASS

go test ./...
PASS

go vet ./...
PASS

cargo fmt --check
PASS

cargo test --locked --lib
PASS — 16 tests

cargo clippy --locked --all-targets -- -D warnings
PASS

cargo check --locked
PASS

pnpm build:sidecar
PASS

pwsh -NoProfile -File scripts/windows-lifecycle-smoke.ps1
PASS

pnpm --filter @yorva/desktop tauri build --no-bundle
PASS
```

The first no-bundle attempt found the prior worktree test application still running from
manual verification and holding `target/release`. Only the two processes whose executable
paths were under this worktree's verified release directory were stopped; the same build
then passed. No installed application or other worktree process was touched.

## Covered negative cases

- injected SQL failure restores schema 016 and returns
  `DATABASE_MIGRATION_FAILED_RECOVERED`;
- interrupted RUNNING state restores the verified protection before retry;
- a tampered protection digest blocks startup with
  `DATABASE_MIGRATION_RECOVERY_REQUIRED`;
- simultaneous unconfirmed legacy and stable product roots return
  `PRODUCT_DATA_IDENTITY_CONFLICT` without modifying either root;
- a completed product-identity migration is repeat-safe;
- retention preserves unknown files.

## Gate result

P8-B1 focused Gate: **PASS**.


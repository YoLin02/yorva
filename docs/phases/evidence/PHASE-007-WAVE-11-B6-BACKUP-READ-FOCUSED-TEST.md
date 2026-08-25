# Phase 7 Wave 11 — B6 Runtime BackupRead Focused Test

- Date: 2026-08-25
- Scope: Runtime-scoped backup index list/get only
- Result: PASS

## Product boundary

- `GET /api/v1/runtimes/{runtimeId}/backups` and
  `GET /api/v1/runtimes/{runtimeId}/backups/{backupId}` are authenticated and GET-only.
- Reads use the YORVA-owned `runtime_backups` index bound to the exact accepted Runtime
  installation. They do not open, decrypt, hash, reconcile, or mutate an artifact.
- `AVAILABLE` and `verifiedAt` are explicitly last-observed facts, not current Restore
  eligibility. A missing file is not guessed to be `MISSING` by an ordinary read.
- Responses expose only ID, Runtime scope, indexed state, format/Runtime versions,
  encrypted size and SHA-256, created/verified timestamps, and `DEVICE|PASSPHRASE`
  key mode. Artifact path, destination reference, key reference, passphrase, credentials,
  archive members and native installation identity are absent.
- `backupRead` is composed dynamically only when the Runtime-scoped repository is
  available. `backupMutate` and `restore` remain false; no create/delete/Restore route or
  Desktop action was added.

## Focused verification

All commands completed successfully:

```text
go test ./internal/runtime ./internal/persistence/sqlite ./internal/app ./internal/transport/httpapi
go vet ./internal/runtime/... ./internal/persistence/sqlite ./internal/app ./internal/transport/httpapi
pnpm api:generate
pnpm api:lint
pnpm --filter @yorva/desktop typecheck
pnpm --filter @yorva/desktop lint
pnpm --filter @yorva/desktop test -- client.management.test.ts ManagementPanel.test.tsx
git diff --check
```

Desktop focused result: 2 files, 12 tests PASS.

The persistence regression fixture intentionally indexes a nonexistent artifact and
proves that list/get preserve the prior observation without probing the destination or
changing the stored state. HTTP and Desktop tests assert that path/key/passphrase fields
are not emitted or displayed.

## Deferred by design

Live verification/reconciliation requires a separately owned bounded workflow and is not
performed by an ordinary read. Backup create, delete, Restore, native destination
selection, decryption-key access and destructive evidence remain outside this lane.

## Integrated Desktop FAIL pending R2

The first repository-level Desktop run after integrating this fifth unavailable
management section preserved one stale shared assertion failure: 23/24 files and
117/118 tests passed because `InstancesPage.test.tsx` still expected four repeated
unavailable messages. Typecheck, lint and OpenAPI lint passed in the same run.

The integration owner changed only that count to five. A fresh R2 repository-level
Desktop run passed: **24/24 files and 118/118 tests**. The initial FAIL remains
preserved above.

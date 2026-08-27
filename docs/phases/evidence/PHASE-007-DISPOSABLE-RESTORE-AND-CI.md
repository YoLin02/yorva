# Phase 7 Disposable Restore and Exact-Candidate CI Evidence

> Date: 2026-08-27
> Product candidate: `e21e8618f6fcda34eba31707500c29bde75893b5`
> Branch: `codex/phase7-hermes-runtime-management`
> Result: PASS

## Scope

This evidence closes the two active blockers retained after Owner Amendment 007A1:

1. exercise encrypted Runtime Restore against disposable state, including success and
   failure recovery;
2. obtain exact-candidate GitHub CI, race and Windows-native evidence.

No Owner Hermes profile or backup was read, stopped, replaced or deleted. Every Restore
test sets `LOCALAPPDATA` to a test-owned temporary directory and uses a separate temporary
YORVA backup directory and generated test-only X25519 identity.

## Automated Restore lifecycle

`TestRuntimeBackupManagerDisposableRestoreLifecycle` uses the production
`RuntimeBackupManager` and real backup-management implementation. It verifies:

- canonical Runtime snapshot creation and encrypted publication;
- plaintext state is absent from the encrypted artifact;
- Restore precondition checks run before snapshot and Restore mutation;
- decrypt, checksum/metadata verification, bounded extraction and atomic tree switch;
- post-check observes the restored candidate before success;
- authoritative filesystem read-back matches the backed-up state;
- backup deletion removes both the artifact and authoritative index entry;
- a forced post-check failure returns `ROLLED_BACK` and restores the exact pre-Restore
  active tree;
- an altered encrypted artifact is rejected before Runtime mutation and never reaches
  post-check;
- successful, rolled-back and rejected paths leave no Restore transaction directory.

Existing interrupted-transaction tests continue to cover daemon-start recovery from
`preparing`, `prepared`, `current-moved` and `activated` phases and rejection of ambiguous
state. The Windows lifecycle smoke remains the native lifecycle/process companion check.

## Local candidate Gate

The following checks passed before push:

- focused disposable Restore lifecycle test;
- focused Hermes Runtime, application and HTTP tests;
- full `go test ./...`, `go vet ./...`, `go build ./cmd/yorvad` and `govulncheck ./...`;
- pnpm frozen install, audit, OpenAPI lint/generation drift, typecheck, lint, 138 Desktop
  tests and Vite build;
- Rust format, 13 library tests, clippy, check and cargo audit;
- Windows lifecycle and MSI-inspector negative tests;
- production local MCP reviewed-Preset lifecycle;
- Tauri release build with `--no-bundle`.

The local machine still cannot supply a C compiler for Go race; exact-candidate race
evidence is supplied by GitHub Actions below.

## GitHub Actions evidence

Workflow run:
[CI run 33046427437](https://github.com/YoLin02/yorva/actions/runs/33046427437)

The run is attached to exact product commit
`e21e8618f6fcda34eba31707500c29bde75893b5` and completed with `Success`.

| Job | Result | Material evidence |
| --- | --- | --- |
| [Web and API contract](https://github.com/YoLin02/yorva/actions/runs/33046427437/job/98431383319) | PASS | audit, OpenAPI lint/drift, typecheck, lint, tests and build all succeeded |
| [Go Node](https://github.com/YoLin02/yorva/actions/runs/33046427437/job/98431383170) | PASS | `go test -race ./...`, vet, govulncheck and daemon build all succeeded |
| [Windows Desktop native shell](https://github.com/YoLin02/yorva/actions/runs/33046427437/job/98431383308) | PASS | disposable encrypted Restore lifecycle, lifecycle smoke, Rust tests/audit/clippy/check and Tauri `--no-bundle` build all succeeded |

The Windows Restore step ran from `2026-08-27T06:37:01Z` to
`2026-08-27T06:37:25Z`. The Go race step ran from `2026-08-27T06:35:51Z` to
`2026-08-27T06:38:40Z`. The final Windows non-MSI build completed at
`2026-08-27T06:53:26Z`.

## Qualification conclusion

The disposable Restore success, failed-postcheck rollback, tamper rejection,
authoritative read-back and cleanup requirements are established on both the local
candidate and the GitHub Windows runner. Exact-candidate race and native build evidence
are attributable to the same product commit.

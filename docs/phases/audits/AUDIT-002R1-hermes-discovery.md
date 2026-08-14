# YORVA Phase 2 Re-Audit R1 — Hermes Discovery & Compatibility

## Phase

Phase 2 — Hermes Discovery & Compatibility

## Baseline / Commit

- frozen predecessor: `phase-001-bootstrap-baseline` → `49de99bd3b9c908f5f2065b7c56fd69e97206284`;
- re-audit target: the complete tracked and untracked working tree on `phase/002-hermes-discovery`, based on `85b5ccf65cd4e537c99ce44f678f6bc806a1be07`;
- ancestry proof: `git merge-base HEAD phase-001-bootstrap-baseline` returned `49de99bd3b9c908f5f2065b7c56fd69e97206284`;
- reviewed diff: every file changed from the Phase 1 baseline plus the current focused audit fixes, Desktop shell and i18n work shown by `git status --short`.

## Auditor

Independent Phase 2 R1 re-audit agent operating from a fresh review context and repository evidence. The agent did not modify implementation, contracts or the preserved `AUDIT-002` FAIL report; this R1 report is its only repository change.

## Date

2026-08-14

## Gate Decision

`PASS`

## Executive Summary

All blocking findings from `AUDIT-002` are closed. Windows creates the version process suspended, assigns it to a configured kill-on-close Job Object, and resumes it only after ownership is established. Independently repeated tests cover immediate descendant creation followed by parent exit, timeout and cancellation, and every recorded descendant was gone when the request returned. The ambiguous Desktop state no longer promotes any candidate to selected details and shows a localized candidate count and full list. Application-level structured diagnostics distinguish all typed outcomes plus timeout, cancellation and unexpected failure without logging candidate paths, warning text, raw errors, output, environment data or credentials. The gated real-Windows smoke now tests its actual invariant: trusted official installation evidence cannot be `NOT_INSTALLED`, and `hermes-agent.exe` is never a candidate or invoked.

The full local Go, Web/OpenAPI, Rust/Tauri, dependency and Windows lifecycle matrix passed. Local Go race detection remains transparently environment-blocked because this host has no `gcc`; the final exact-commit GitHub Actions race job remains a mandatory downstream pre-merge/freeze gate. That pending procedural gate is not accepted technical debt and does not weaken this source-tree audit decision.

## Verification Evidence

| Check | Result | Evidence |
|---|---|---|
| Baseline ancestry and complete diff | PASS | `HEAD` is `85b5ccf65cd4e537c99ce44f678f6bc806a1be07`; both the merge base and annotated Phase 1 tag resolve to `49de99bd3b9c908f5f2065b7c56fd69e97206284`; all 47 Phase 2 baseline-diff files and current uncommitted files were reviewed. |
| `AUDIT-002-HIGH-001` — pre-assignment race | CLOSED | `process_windows.go:14-18` sets `CREATE_SUSPENDED`; `:47` assigns the process to the Job Object; `:51` resumes only afterward. Closing the Job kills descendants on normal return, timeout, cancellation, output overflow and ownership failure. |
| New Windows process regressions | PASS | `TestCommandRunnerOwnsImmediateDescendantBeforeExecutableRuns`, `TestCommandRunnerTimeoutReapsDescendantProcess`, `TestCommandRunnerCancellationReapsProcess`, and the output-limit/direct-child test passed together for 20 consecutive runs on Windows. PID-based assertions prove descendants exited. |
| `AUDIT-002-MEDIUM-001` — ambiguous presentation | CLOSED | `HermesDiscoveryView.tsx:46` uses only `discovery.selected`; `:76` renders the localized count. The focused test proves both candidate paths appear only in the candidate list while selected Executable/Version details and candidate versions are absent. English and Simplified Chinese provide typed count copy. |
| `AUDIT-002-MEDIUM-002` — safe structured diagnostics | CLOSED | `runtime_discovery.go:44-53,72-81` emits one normalized completion/failure record. `TestRuntimeDiscoveryLogsSafeStructuredOutcomes` covers all seven states; `TestRuntimeDiscoveryLogsNormalizedFailureWithoutRawError` covers unexpected error, deadline and cancellation. Tests inject a username-bearing path, warning secret and raw error secret and prove none is logged. |
| `AUDIT-002-LOW-001` — real-install smoke invariant | CLOSED | `real_windows_smoke_test.go:28-40` accepts any non-`NOT_INSTALLED` result with no adapter error and rejects `hermes-agent.exe` candidates. Independent gated run returned `BROKEN_EXECUTABLE`, zero candidates, PASS. This re-audit did not invoke `hermes-agent.exe`. |
| Forbidden execution/scope search | PASS | Production Hermes execution exists only in `command.go` as direct `exec.CommandContext(..., executable, "--version")`. `hermes-agent.exe` occurs only in fixed evidence checks, documentation and tests. No shell string, PATH mutation, installer/download, profile/lifecycle, Cloud or plugin implementation was found. |
| Affected Go suites | PASS | Safe-log tests passed 20 times; `go test ./internal/runtime/hermes ./internal/app ./internal/transport/httpapi -count=20` passed; the dedicated process suite passed 20 times. |
| Full Go verification | PASS | `go test ./...`, `go vet ./...`, `go build ./cmd/yorvad`, and `govulncheck@v1.7.0 ./...` passed; no Go vulnerability was reported. |
| Go race | ENVIRONMENT BLOCKED | `CGO_ENABLED=1 go test -race ./...` stopped before test compilation: `C compiler "gcc" not found`. No local race PASS is claimed. Exact-commit CI remains mandatory. |
| Web, OpenAPI and Desktop | PASS | Frozen install, pnpm audit, OpenAPI lint/generation/exact generated-schema diff, typecheck, lint, 9 files/34 tests and production build passed. pnpm reported no known vulnerability. |
| UI/i18n/time/accessibility | PASS | Tests cover separate Dashboard/Runtimes/Settings navigation, immediate persisted locale switching, every Runtime state in both locale resources, stable warning-code translation, explicit locale/time-zone formatting, cancellation/retry stale-result suppression, semantic navigation/current page, status/alert regions and keyboard-native controls. |
| Rust/Tauri | PASS | `cargo fmt --check`, `cargo test --locked` (9 passed), strict clippy and `cargo check --locked` passed. The localized linker import-library line is informational only. |
| Rust advisory scan | PASS WITH EXISTING WARNINGS | `cargo-audit 0.22.2` returned success with no vulnerability failure and 17 allowed unmaintained/unsound warnings inherited from the unchanged Phase 1 Tauri lockfile. Phase 2 added no Rust dependency. |
| Windows lifecycle/package | PASS | Sidecar build, `scripts/windows-lifecycle-smoke.ps1`, and Tauri release `--no-bundle` completed; `yorva-desktop.exe` was produced. |
| Data/persistence | N/A | No migration, discovery table, persisted result or authoritative cache was added. Discovery remains a live read-only query. |
| Diff hygiene | PASS | `git diff --check` found no whitespace error; only the working-copy LF→CRLF warning for `styles.css` was emitted. |
| Exact-commit GitHub Actions | PENDING | The Owner-required sequence creates the final Phase 2 commit only after this audit. That exact commit, including Go race in CI, must pass before merge or baseline freeze. |

## Dimension Results

### Scope

`PASS` — implementation is limited to Hermes discovery/compatibility and the authorized Desktop shell/i18n amendment. No Phase 3 or later capability appears.

### Correctness

`PASS` — supported, unsupported, absent, incomplete/broken, malformed, timeout, cancellation, ambiguity, output overflow and retry/stale-result paths are deterministic and tested.

### Architecture

`PASS` — the reviewed path remains React Desktop → authenticated HTTP → application use case → Runtime-neutral contract → Hermes adapter. React and Tauri contain no Hermes command or compatibility logic.

### Security

`PASS` — direct constant argv, bounded output/candidates/time, allowlisted child environment, suspended-before-Job Windows ownership, auth/CORS and safe response/logging boundaries are implemented and tested.

### Data and Persistence

`N/A` — Phase 2 adds no schema, migration, repository or persisted discovery cache.

### Concurrency and Lifecycle

`PASS` — request and overall cancellation propagate; direct children are waited; process trees are killed on every termination path; immediate descendants cannot run before Job ownership on Windows. Local race execution is environment-blocked and remains a mandatory exact-commit CI gate.

### Protocol and Compatibility

`PASS` — authenticated POST, closed DTO enums/error codes, RFC 3339 timestamps, safe errors, `0.19.x` support and `0.20.0`/untested prerelease rejection agree across Go, OpenAPI, generated TypeScript and Desktop.

### Testing and Verification

`PASS` — the risk-proportional local matrix, repeated process/log tests, real host smoke, Windows lifecycle and release build all passed. Exact-commit CI is explicitly pending by sequence.

### Maintainability

`PASS` — fixes are localized to OS process ownership, the application discovery boundary and one Runtime view; no speculative framework or unrelated dependency/refactor was added.

### Documentation

`PASS` — Phase Spec, Runtime, Protocol, Security, Data Model, Development and Roadmap describe the implemented discovery evidence, process ownership, diagnostics, UI and no-persistence behavior.

### Dependencies / Supply Chain

`PASS` — `x/sys` is now direct because the Windows adapter uses its Job APIs and was already locked transitively. No JS/Rust framework or unrelated upgrade was introduced; local scans passed with only inherited Rust warnings.

### Operations / Diagnostics

`PASS` — one structured application-level outcome record distinguishes every result safely and user-visible copy is actionable without raw process data.

## Findings

### Critical

None.

### High

None. `AUDIT-002-HIGH-001` is closed.

### Medium

None. `AUDIT-002-MEDIUM-001` and `AUDIT-002-MEDIUM-002` are closed.

### Low

None. `AUDIT-002-LOW-001` is closed.

### Info

- The current host's trusted Hermes installation evidence is correctly classified `BROKEN_EXECUTABLE` because no safe `hermes.exe` candidate exists.
- This re-audit did not execute `hermes-agent.exe`. A historical manual diagnostic invoked it once before the Owner's explicit prohibition; this report does not rewrite that history.
- Local Go race evidence is unavailable only because `gcc` is absent. The exact final commit's GitHub Actions race job must pass before merge and freeze.
- The 17 Rust advisory warnings are inherited from the frozen Phase 1 dependency graph and were not introduced by Phase 2.

## Accepted Technical Debt

None.

## Required Fixes Before Next Phase

None in the audited source tree. Phase 3 remains prohibited until the exact audited commit passes GitHub Actions, Phase 2 is merged to `main`, formally frozen and tagged according to governance.

## Gate Rationale

`PASS` is appropriate because the former High lifecycle/security defect and both Medium contract defects are demonstrably closed, no new blocking finding was found, and all locally executable mandatory checks passed. The unavailable local race run is recorded without misrepresentation and is covered by a mandatory downstream exact-commit CI gate before merge/freeze.

## Next Step

Commit the complete audited working tree including this R1 report, push the Phase 2 branch, and require the exact-commit GitHub Actions matrix — especially `go test -race ./...` and Windows native checks — to pass. Only then may the audited branch be merged to `main`, Phase 2 be marked `COMPLETE / FROZEN`, and the Phase 2 baseline tag be created. Do not begin Phase 3 before that freeze.

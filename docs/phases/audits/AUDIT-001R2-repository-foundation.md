# YORVA Phase 1 Independent Re-Audit R2 — Repository Foundation

## Phase

Phase 1 — Repository foundation / Bootstrap. This is a focused independent re-audit of the fix for `MEDIUM-R1-001` and its effect on the Phase 1 gate.

## Baseline / Working Tree

- Fix starting point and current `HEAD`: `e567c249d68a3ce4c1e7e45b39e79ff4031e3ace` on `phase/001-audit-fix`.
- The audited fix is an uncommitted working-tree change relative to that commit.
- Tracked fix files: `apps/desktop/src/App.tsx` and `docs/BOOTSTRAP.md`.
- Untracked fix test: `apps/desktop/src/App.test.tsx`.
- Existing untracked historical report: `docs/phases/audits/AUDIT-001R1-repository-foundation.md`.
- No Phase 1 baseline tag exists in the locally visible repository state.
- `git diff --check e567c249d68a3ce4c1e7e45b39e79ff4031e3ace` passed.

The complete working-tree diff from `e567c249` was inspected, including the untracked App test. This re-audit did not modify implementation, the prior audit reports, Phase state, tags, branches, or Phase 2.

## Previous Audit

- `AUDIT-001-repository-foundation.md`: `FAIL` with two High, two Medium, and four Low findings.
- `AUDIT-001R1-repository-foundation.md`: `FAIL`; all original findings were verified resolved, but `MEDIUM-R1-001` found that React stopped observing `DAEMON_NOT_READY` after approximately four seconds even though the native startup lifecycle remained valid for ten seconds.
- R1 also recorded `MEDIUM-R1-002`, a repository-history/governance observation concerning an earlier merge before the gate, plus one Low and three Info observations. Those observations are not implementation defects introduced by this fix. Owner reconciliation remains part of the baseline-freeze procedure.

## Auditor

Independent Codex R2 audit context. The auditor did not participate in this fix and did not rely on the implementation agent's conclusion.

## Date

2026-08-14

## Gate Decision

**PASS**

## Executive Summary

`MEDIUM-R1-001` is resolved. React no longer has an independent retry count or time budget. It retries only the native lifecycle's exact `DAEMON_NOT_READY` state, and therefore remains in the starting flow until Rust returns either a ready session or a terminal error. The Rust-owned ten-second `STARTUP_TIMEOUT` remains unchanged and is still the sole authoritative startup deadline.

The new App-level tests deterministically cover a ready transition after 45 retries at the configured 200 ms interval (approximately nine seconds) and a transition from `DAEMON_NOT_READY` to terminal `DAEMON_STARTUP_FAILED`. The delayed-ready test necessarily fails against the old 20-retry implementation, so the suite is a valid regression guard. The terminal test proves that non-authoritative errors are not retried indefinitely and that the UI renders only the safe failure message.

No new blocking finding was found in the affected scope. The Phase 1 success flow and mandatory verification are restored. Phase 1 is eligible for baseline freeze; this report does not itself freeze the baseline or claim that Phase 2 is unlocked.

## Evidence

### Governing material and prior evidence

The auditor read `AGENTS.md`, `docs/BOOTSTRAP.md`, `docs/PHASE_GOVERNANCE.md`, `docs/AUDIT_STANDARD.md`, `AUDIT-001-repository-foundation.md`, and `AUDIT-001R1-repository-foundation.md`. The unchanged Rust lifecycle and the relevant React/session code were reviewed directly.

### Implementation review

- `apps/desktop/src-tauri/src/daemon.rs:18` still defines `STARTUP_TIMEOUT` as ten seconds.
- `apps/desktop/src-tauri/src/daemon.rs:126-147` changes `Starting` to `Failed` when that native deadline elapses, and `daemon_session` exposes the resulting terminal code.
- `apps/desktop/src/App.tsx:12-13` has no failure-count condition. Its retry predicate returns true only when `isDaemonNotReady(error)` is true.
- `apps/desktop/src/api/session.ts:14-16` classifies only the exact `DAEMON_NOT_READY` code as retryable for startup observation.
- A ready result ends the session query successfully. `DAEMON_STARTUP_FAILED` and every other error return false from the retry predicate, end observation, and render the safe connection-failure state.
- `docs/BOOTSTRAP.md:387-391` now explicitly records native deadline ownership and the React observation contract without changing the ten-second deadline.
- The diff adds no Runtime/Hermes behavior, dependency, process, persistence, API, secret, or Phase 2 capability.

### Regression-test review

- `apps/desktop/src/App.test.tsx:74-93` supplies 45 consecutive `DAEMON_NOT_READY` results, then a ready session. With the configured 200 ms interval, readiness occurs after approximately nine seconds, inside the native ten-second lifecycle window.
- The same test requires 46 session calls and the connected Node UI. Under the old predicate `failures < 20 && isDaemonNotReady(error)`, TanStack Query stops after the twentieth retry (at most 21 total calls), never reaches the queued ready result, and fails both assertions. This establishes the counterfactual regression property without changing the audited tree.
- `apps/desktop/src/App.test.tsx:95-116` supplies one `DAEMON_NOT_READY` followed by `DAEMON_STARTUP_FAILED`, asserts exactly two calls, and verifies the safe alert contains no path, spawn, token, or stack diagnostic.
- Fake timers make both state transitions deterministic. The focused file passed 20 consecutive independent executions.
- The module mocks preserve possible production state shapes. Direct source inspection independently confirmed that the real classifier has the same exact-code behavior, avoiding a test-only retry policy.

### Independent affected checks

| Command | Result |
|---|---|
| `pnpm.cmd typecheck` | PASS |
| `pnpm.cmd lint` | PASS |
| `pnpm.cmd test` | PASS — 4 files, 9 tests |
| `pnpm.cmd build` | PASS — TypeScript build and Vite production bundle |
| focused `App.test.tsx`, repeated 20 times | PASS — 20/20 |

The first bare `pnpm typecheck` invocation was blocked by the host PowerShell policy loading `pnpm.ps1`. Re-running through the same installation's executable shim, `pnpm.cmd`, passed. This was an environment wrapper failure, not a repository check failure.

### Implementation-agent matrix cross-check

The implementation agent recorded the following complete local matrix for this exact working tree:

- PASS: `pnpm api:lint`, `pnpm lint`, `pnpm test` (4 files / 9 tests), `pnpm typecheck`, `pnpm build`, and `pnpm audit --audit-level low`.
- PASS with Go 1.26.6: `go test ./...`, `go vet ./...`, `go build -trimpath`, and `govulncheck@v1.7.0 ./...` with no vulnerabilities.
- PASS with Rust 1.97.1: `cargo fmt --check`, `cargo test --locked` (9 tests), `cargo clippy --locked --all-targets -- -D warnings`, `cargo check --locked`, and `cargo audit` with zero vulnerabilities and the same 17 allowed maintenance warnings already reviewed in R1.
- PASS: target-aware sidecar build, Windows lifecycle smoke, and Tauri `build --no-bundle`. The first sidecar attempt lacked the audited Rust toolchain on `PATH`; the same command passed after restoring that toolchain path. Tauri emitted only the previously observed localized linker informational warning.
- Not run locally: `go test -race ./...`, because the isolated Windows Go setup lacks CGO. Exact-commit CI run 31772889580 passed the race job for `e567c249`; the R2 fix touches only React, its App test, and Phase 1 lifecycle documentation.

There is no exact-fix CI run because the audited fix is still an uncommitted working tree. For this narrow React-only correction, the complete local matrix, independent affected replay, deterministic regression, and unchanged exact-CI-proven Rust/Go lifecycle are sufficient evidence.

## MEDIUM-R1-001 Resolution

**RESOLVED**

| Required property | Result | Evidence |
|---|---|---|
| Rust ten-second lifecycle remains authoritative | VERIFIED | Rust timeout constant and transition code are unchanged; React has no independent budget. |
| React does not terminate early on `DAEMON_NOT_READY` | VERIFIED | Exact-code predicate has no retry-count limit; nine-second delayed-ready test passes. |
| Ready stops observation | VERIFIED | Query resolves on attempt 46 and proceeds to the connected Node view with exactly 46 calls. |
| `DAEMON_STARTUP_FAILED` stops observation | VERIFIED | Terminal test ends after exactly two calls and renders the safe alert. |
| Other errors are not retried indefinitely | VERIFIED | Only the exact `DAEMON_NOT_READY` code is accepted by the production predicate. |
| Regression fails on old implementation | VERIFIED | Old 20-retry budget cannot consume 45 transient errors and therefore cannot reach the queued ready result. |
| Tests are deterministic and not false-positive lifecycle substitutes | VERIFIED | Fake timers, exact call counts, state assertions, 20/20 repeated runs, and direct production-classifier inspection. |
| No resource leak introduced | VERIFIED | The change uses React Query's existing observer-owned retry timer; it adds no interval, listener, process, goroutine, or retained global resource. Ready, terminal error, and component teardown cancel further observation through the existing query lifecycle. |

## Dimension Results

| Dimension | Result | R2 assessment |
|---|---|---|
| Scope | PASS | Three-file Phase 1 fix only; no Phase 2 leakage. |
| Correctness | PASS | The complete valid four-to-ten-second startup interval is observed. |
| Architecture | PASS | Native lifecycle remains authoritative; React only observes the narrow command. |
| Security | PASS | Terminal diagnostics remain sanitized; token handling is unchanged. |
| Data and Persistence | N/A | No persistence or schema change. R1 evidence remains valid. |
| Concurrency and Lifecycle | PASS | Native timeout ownership is unchanged; UI observation now follows its terminal transitions. |
| Protocol and Compatibility | PASS | Existing stable lifecycle error codes are consumed without contract changes. |
| Testing and Verification | PASS | Affected checks and deterministic regressions pass; full local matrix was cross-checked. |
| Maintainability | PASS | One predicate change and two focused tests; no speculative abstraction. |
| Documentation | PASS | Phase contract now states the implemented ownership rule. |
| Dependencies / Supply Chain | N/A | No dependency or lockfile change. R1 evidence remains valid. |
| Operations / Diagnostics | PASS | Valid slow startup no longer becomes a false terminal UI failure. |

## New Findings

### Critical

None.

### High

None.

### Medium

None.

### Low

None in the R2 fix. The previously recorded `LOW-R1-001` is unchanged and non-blocking.

### Info

No new R2 observation. The prior R1 informational maintenance items remain unchanged and do not affect this gate.

## Accepted Technical Debt

None newly accepted by this re-audit. This decision does not reclassify or erase historical Low/Info observations.

## Required Fixes Before Next Phase

None for `MEDIUM-R1-001` or the affected Phase 1 implementation path.

The repository owner must still perform the governance steps for a baseline freeze: include the complete audited working tree in the selected baseline, reconcile the repository-history observation from R1, and record the frozen Phase 1 baseline before any Phase 2 implementation begins.

## Gate Rationale

The sole R1 defect that failed the mandatory Phase 1 startup success flow is fixed and covered by a regression that fails the old implementation. The fix preserves the native ten-second authority, retries only the transient native state, stops on both success and terminal failure, introduces no unbounded retry of unrelated errors or new resource ownership, and stays within Phase 1 scope. Independent affected checks and the implementation agent's full matrix passed.

There are zero unresolved Critical findings, zero unresolved blocking High findings, the affected mandatory acceptance flow passes, and no architecture or security condition prevents baseline preparation. `PASS` is therefore the correct decision under `AUDIT_STANDARD.md` section 17.

## Next Step

Phase 1 is eligible for baseline freeze. The repository owner should review this R2 decision, commit the audited fix and reports as an intentional baseline candidate, reconcile the R1 repository-state observation, run any desired exact-candidate CI, and freeze the Phase 1 baseline. This report does not itself unlock or authorize Phase 2; Phase 2 remains gated until the owner completes the freeze required by `PHASE_GOVERNANCE.md`.

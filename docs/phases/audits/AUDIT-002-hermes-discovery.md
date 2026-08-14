# YORVA Phase 2 Audit — Hermes Discovery & Compatibility

## Phase

Phase 2 — Hermes Discovery & Compatibility

## Baseline / Commit

- frozen predecessor: `phase-001-bootstrap-baseline` → `49de99bd3b9c908f5f2065b7c56fd69e97206284`;
- audit target: the complete tracked and untracked working tree on `phase/002-hermes-discovery`, based on `85b5ccf65cd4e537c99ce44f678f6bc806a1be07`;
- ancestry proof: `git merge-base HEAD phase-001-bootstrap-baseline` returned `49de99bd3b9c908f5f2065b7c56fd69e97206284`;
- reviewed diff: all 47 files changed from the Phase 1 baseline, plus the current uncommitted Phase 2 repair, Desktop shell and i18n files shown by `git status`.

## Auditor

Independent Phase 2 audit agent, operating from a fresh review context and repository evidence rather than the implementation summary.

## Date

2026-08-14

## Gate Decision

`FAIL`

## Executive Summary

The implementation stays within Hermes discovery/compatibility scope and preserves the required Desktop → authenticated HTTP → application → Runtime contract → Hermes adapter direction. Candidate discovery is bounded and read-only, compatibility and negative states are typed, OpenAPI is synchronized, Desktop provides the requested sidebar and English/Simplified Chinese presentation, timestamps are localized, and the independently repeated deterministic verification matrix passed except for local Go race detection, which cannot compile because this host has no C compiler.

The gate nevertheless fails. The Windows command path starts the executable before assigning it to the kill-on-close Job Object. An executable can therefore spawn a descendant and exit during the gap, leaving that descendant outside YORVA ownership. This violates the phase's mandatory no-process-outlives-request guarantee. In addition, the `AMBIGUOUS` Desktop state displays the first candidate as if it were selected, and the implementation has no structured discovery outcome diagnostics despite the Phase Spec's logging contract. These findings must be fixed and re-audited before the Phase 2 branch is committed, pushed, merged or baselined.

## Verification Evidence

| Check | Result | Evidence |
|---|---|---|
| Baseline ancestry and complete diff | PASS | `git merge-base`, `git diff --name-status/--numstat phase-001-bootstrap-baseline`, and `git status --short` were reviewed. |
| Forbidden-scope/static execution search | PASS | Repository search found the production Hermes execution only at `command.go`, using direct `exec.CommandContext(..., executable, "--version")`; `hermes-agent.exe` appears only as fixed evidence/test text and is never passed to the runner. No PATH mutation, installer, shell string, generic execution API, profile/lifecycle, Cloud or plugin implementation was found. |
| Real Windows Hermes smoke | PASS | `YORVA_REAL_HERMES_SMOKE=1 go test ./internal/runtime/hermes -run TestRealWindowsHermesInstallationSmoke -v -count=1` → `BROKEN_EXECUTABLE`, zero executable candidates, PASS. The audit run did not execute `hermes-agent.exe`. |
| Repeated affected Go suites | PASS | `go test ./internal/runtime/hermes ./internal/app ./internal/transport/httpapi -count=20` passed. |
| Full Go suite/vet/build/vulnerability scan | PASS | `go test ./...`, `go vet ./...`, `go build ./cmd/yorvad`, and `govulncheck@v1.7.0 ./...` passed; `govulncheck` reported no vulnerabilities. |
| Go race | ENVIRONMENT BLOCKED | `CGO_ENABLED=1 go test -race ./...` failed before compiling tests because `gcc` is absent. Exact-commit CI race remains mandatory and must pass before merge/baseline. |
| JavaScript install/audit/OpenAPI/Desktop | PASS | Frozen install, `pnpm audit --audit-level low`, OpenAPI lint/generation/exact generated diff, typecheck, lint, 9 files/33 tests, and production build passed; pnpm reported no known vulnerabilities. |
| Rust/Tauri | PASS | `cargo fmt --check`, `cargo test --locked` (9 passed), strict clippy, and `cargo check --locked` passed. The linker emitted only its localized import-library informational warning. |
| Rust advisory scan | PASS WITH EXISTING WARNINGS | Direct `cargo-audit.exe audit` found no vulnerability failure and reported 17 allowed unmaintained/unsound warnings in the unchanged Phase 1 lockfile dependency graph. Phase 2 added no Rust dependency. |
| Windows lifecycle/package | PASS | Sidecar build, `scripts/windows-lifecycle-smoke.ps1`, and `pnpm --dir apps/desktop tauri build --no-bundle` passed; the release executable was produced. |
| Diff hygiene | PASS | `git diff --check` reported no whitespace errors; only the existing Git LF→CRLF checkout warning for `styles.css` appeared. |
| Exact-commit GitHub Actions | PENDING | No final Phase 2 commit exists yet by the Owner-required sequence. It must pass on the eventual exact commit before merge/freeze and cannot be represented as audit evidence now. |

## Dimension Results

| Dimension | Result | Notes |
|---|---|---|
| Scope | PASS | Discovery/compatibility, Desktop shell and i18n only; no Phase 3 installation or later roadmap capability. |
| Correctness | FAIL | `AMBIGUOUS` presentation exposes the first candidate as a de facto selected detail, and Windows process ownership is not race-free. |
| Architecture | PASS | React uses the typed local API; transport calls the application use case; the Runtime-neutral contract is implemented by the Hermes adapter; Tauri has no Hermes business logic. |
| Security | FAIL | High finding `AUDIT-002-HIGH-001` leaves a process-tree ownership gap on Windows. Auth, CORS, constant argv, bounded output and environment filtering otherwise passed review/tests. |
| Data and Persistence | N/A | No migration, repository change, discovery table or persisted discovery cache was added; `DATA_MODEL.md` explicitly reserves persistence for a later phase. |
| Concurrency and Lifecycle | FAIL | Timeout/cancel/wait tests pass for descendants captured by the Job Object, but assignment occurs after execution starts and cannot guarantee capture. |
| Protocol and Compatibility | PASS | Authenticated POST, closed states/error codes, safe errors, RFC 3339 time, `0.19.x` support policy and `0.20.0` rejection are synchronized across Go/OpenAPI/generated TypeScript. |
| Testing and Verification | FAIL | Broad local matrix passed, but the high process race has no regression and the gated host smoke is unnecessarily version-state-specific. Local race is transparently environment-blocked and exact-commit CI is pending. |
| Maintainability | PASS | Hermes-specific rules remain cohesive; no generic plugin framework, unnecessary service chain or unrelated refactor/dependency upgrade was introduced. |
| Documentation | FAIL | Contracts are generally synchronized, but the documented outcome logging requirement is not implemented. |
| Dependencies / Supply Chain | PASS | `x/sys` became direct because Windows Job Objects use it; it was already locked transitively. No JS/Rust framework or unrelated dependency update was added. |
| Operations / Diagnostics | FAIL | User-safe typed responses are good, but discovery emits no structured outcome diagnostics. |

## Findings

### Critical

None.

### High

#### AUDIT-002-HIGH-001 — Windows descendants can escape process ownership before Job Object assignment

`services/node/internal/runtime/hermes/command.go:61-70` calls `command.Start()` and only afterward calls `ownProcessTree`. `services/node/internal/runtime/hermes/process_windows.go:18-48` creates/configures the Job Object and assigns the already-running process. A fast executable can spawn a descendant and exit before assignment; Windows does not retroactively place that pre-existing descendant into the later-assigned job. Closing the Job Object therefore cannot guarantee cleanup of that descendant.

This contradicts Phase Spec sections 10 and 14 and `docs/SECURITY.md` section 9, which require the full process tree to be owned, cancelled and reaped with no process outliving the request. Existing tests prove cleanup only after the parent remains alive long enough to be assigned; they do not close the pre-assignment race.

Required fix: create the Windows child suspended, bind it to the configured kill-on-close Job Object before its code runs, then resume it. Add a regression in which the fake executable immediately spawns a long-lived descendant and exits, and prove timeout/cancel/normal-return cleanup. Preserve direct executable + constant argv, environment filtering and no-shell behavior.

### Medium

#### AUDIT-002-MEDIUM-001 — `AMBIGUOUS` UI implicitly promotes the first candidate

`apps/desktop/src/components/HermesDiscoveryView.tsx:46` uses `discovery.selected ?? discovery.candidates[0]`, and lines 66-67 render that candidate's version and path in the main details even when `selected` is deliberately null. For an `AMBIGUOUS` response, this visually promotes the first candidate despite the contract requiring no implicit selection. The view also lists candidates but does not explicitly present the candidate count promised by the Phase Spec.

Required fix: render selected-candidate details only when `discovery.selected` is non-null. For `AMBIGUOUS`, render an explicit localized count plus the complete candidate list, and add a test proving no candidate is displayed in the selected executable/version fields.

#### AUDIT-002-MEDIUM-002 — Required structured discovery diagnostics are absent

The Phase Spec section 15 requires logs to distinguish every discovery outcome safely. `services/node/internal/app/runtime_discovery.go:24-31`, `services/node/internal/runtime/hermes/detector.go:31-83`, and `services/node/internal/transport/httpapi/server.go:131-156` complete discovery without emitting any structured result diagnostic; the daemon logger created in `services/node/internal/daemon/daemon.go` is not passed into these paths.

Required fix: emit one safe structured completion/failure record at an owning boundary with Runtime kind, normalized result/error code, candidate count and duration/cancel/timeout flags as applicable. Do not log raw stdout/stderr, environment values, credentials or user-bearing executable paths. Add focused tests for outcome distinction and sensitive-field absence.

### Low

#### AUDIT-002-LOW-001 — Gated real-install smoke rejects a valid supported host

`services/node/internal/runtime/hermes/real_windows_smoke_test.go:35-37` accepts only `BROKEN_EXECUTABLE` or `UNSUPPORTED`. Its stated contract is that an existing trusted official installation must not become `NOT_INSTALLED`; a host with a safe supported `0.19.x` launcher would make this smoke fail incorrectly.

Required fix: make the host smoke assert the invariant it owns (existing trusted evidence is not `NOT_INSTALLED`, no `hermes-agent.exe` candidate is used, and no unexpected adapter error occurs) without hard-coding this machine's current broken/unsupported outcomes.

### Info

- This host's documented Hermes root currently produces `BROKEN_EXECUTABLE` with zero safe candidates, which correctly distinguishes an incomplete installation from `NOT_INSTALLED`.
- The current implementation, deterministic tests and this audit run never executed `hermes-agent.exe`. A separate historical diagnostic manually invoked it once before the Owner's explicit prohibition; that event is not an implementation behavior or evidence of a current code path, but it must not be rewritten as “never happened.”
- Local Go race evidence is unavailable solely because `gcc` is absent. The exact final commit's GitHub Actions race job must pass before merge/freeze.
- The Rust advisory warnings are inherited from the unchanged frozen Tauri lockfile and were not introduced by Phase 2.

## Accepted Technical Debt

None accepted in this failed gate.

## Required Fixes Before Next Phase

1. Close the Windows pre-assignment process-tree race and add an immediate-spawn regression.
2. Remove implicit first-candidate details from `AMBIGUOUS`; add localized count and regression coverage.
3. Add safe structured outcome diagnostics and tests.
4. Generalize the real Windows smoke to its actual installation-evidence invariant.
5. Re-run affected Go/React/Windows checks, the full local matrix, and a fresh independent re-audit while preserving this report.
6. After a re-audit passes, create/push the exact Phase 2 commit and require exact-commit CI (including Go race) to pass before merge and baseline freeze.

## Gate Rationale

`FAIL` is mandatory because a High lifecycle/process-cleanup finding violates an explicit security and acceptance guarantee. The two Medium findings also affect required UX and operational contracts. Passing deterministic tests cannot compensate for the uncovered pre-assignment race, and exact-commit CI has not yet run by design.

## Next Step

Keep Phase 2 in audit/fix mode. Address only the findings above, preserve this failed report unchanged, then issue `AUDIT-002R1-hermes-discovery.md` from a fresh independent re-audit. Do not merge, tag, freeze or begin Phase 3.

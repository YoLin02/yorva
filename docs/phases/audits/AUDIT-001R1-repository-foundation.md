# YORVA Phase 1 Independent Re-Audit — Repository Foundation

## Phase

Phase 1 — Repository foundation / Bootstrap, first independent re-audit after `AUDIT-001`.

Phase 2 remains locked.

## Baseline / Commit

- Starting baseline: `phase-000-docs-baseline` / `ad4267c`.
- Audited branch: `phase/001-audit-fix`.
- Audited commit: `e567c249d68a3ce4c1e7e45b39e79ff4031e3ace`.
- Previous audit: `docs/phases/audits/AUDIT-001-repository-foundation.md` (`FAIL`).
- CI evidence: GitHub Actions run [31772889580](https://github.com/YoLin02/yorva/actions/runs/31772889580), exact audited SHA, `success`.

The complete final tree and its diff from `ad4267c` were reviewed. No existing audit report or implementation file was changed by this re-audit.

## Auditor

Independent Codex re-audit context. The auditor did not rely on the implementation agent's repair summary and made no implementation changes.

## Date

2026-08-14

## Gate Decision

**FAIL**

## Executive Summary

The repair commit closes all eight findings from `AUDIT-001`. In particular, the Desktop now retains and bounds sidecar ownership, rolls back partial startup, provides graceful shutdown with forced termination fallback, observes parent EOF in `yorvad`, keeps Tauri alive after synchronous startup failure, and exposes a sanitized failure state to React. The dependency, HTTP contract, workflow pinning, token-debug, SQLite connection, and README findings are also resolved. Exact-commit CI is green, and the required local replay passed except for the local Go race command, for which the exact-commit Linux CI result supplies the missing evidence.

The re-audit nevertheless found a new correctness defect in the Phase 1 startup success flow. React stops retrying `DAEMON_NOT_READY` after about four seconds, while the native lifecycle intentionally permits startup for ten seconds. A daemon that becomes ready in the valid four-to-ten-second interval can therefore be healthy while the UI has already entered a terminal connection-failure state. There is no application-level delayed-ready regression test. This contradicts the bounded ten-second lifecycle contract and the required `starting → connected / connection failure` behavior.

Under `AUDIT_STANDARD.md`, this is a failed phase success flow and mandatory acceptance criterion, so `FAIL` is required. It cannot be converted to `PASS WITH CONDITIONS`: the remaining Medium finding has not been explicitly accepted with an owner and resolution trigger. Phase 1 must not be frozen and Phase 2 must not begin.

The audit also observed that remote `main` already contains merge commit `6b5c418b0cfcf0d4561e698b1f474174b44cfcd3`, which merges the audited commit before this gate. That external repository state is not a defect in `e567c249` and is not treated as a Phase 1 baseline freeze; it requires owner reconciliation while the phase remains failed.

## Verification Evidence

### Governing material read

The auditor read the repository instructions and the complete governing set required by the re-audit prompt:

- `AGENTS.md`, `README-DOCS.md`, and `README.md`;
- `docs/ARCHITECTURE.md`, `docs/DEVELOPMENT.md`, `docs/PROTOCOL.md`, `docs/RUNTIME.md`, `docs/DATA_MODEL.md`, and `docs/SECURITY.md`;
- `docs/BOOTSTRAP.md`, `docs/PHASE_GOVERNANCE.md`, `docs/AUDIT_STANDARD.md`, and `docs/ROADMAP.md`;
- ADR-0001 through ADR-0004;
- the phase and audit templates, `AUDIT-000`, and the unchanged `AUDIT-001`.

### Repository, scope, and implementation review

The re-audit verified the branch and exact SHA, inspected the complete `ad4267c..e567c249` diff and final tree, and reviewed all relevant Go, Rust, React/TypeScript, SQL, OpenAPI, generated API, workflow, script, manifest, and lock files. Searches were repeated for phase-scope leakage, Hermes integration, shell execution, public binds, token propagation/storage/logging, unrestricted `Debug`, CSP/capability expansion, and protocol drift.

No Phase 2 discovery, install, PATH/config/profile probing, Cloud, account, telemetry, updater, service installation, or dynamic plugin behavior was found. React does not access Hermes, SQLite, or arbitrary shell/filesystem capabilities. The daemon continues to bind an ephemeral IPv4 loopback port and uses an in-memory 256-bit bearer token that is not placed in argv, URL, handshake output, logs, or SQLite.

### Local replay

| Command / check | Result | Evidence |
|---|---|---|
| `pnpm api:lint` | PASS | OpenAPI validation completed. |
| `pnpm typecheck` | PASS | TypeScript project references compiled. |
| `pnpm lint` | PASS | ESLint completed without findings. |
| `pnpm test` | PASS | 3 files, 7 tests passed. |
| `pnpm build` | PASS | Vite production build completed. |
| `pnpm audit --audit-level low` | PASS | No Node advisory failure. |
| `go test ./...` | PASS | All Go packages passed with Go 1.26.6. |
| `go vet ./...` | PASS | No vet findings. |
| `go build -trimpath ./cmd/yorvad` | PASS | Binary built outside the repository. |
| daemon/bootstrap lifecycle tests, `-count=20` | PASS | Repeated control, EOF, and shutdown paths passed. |
| protocol package tests, `-count=20` | PASS | Repeated route/contract tests passed. |
| `govulncheck@v1.7.0 ./...` | PASS | `No vulnerabilities found.` |
| `go test -race ./...` | **NOT RUN locally** | Reason: the isolated Windows Go environment had CGO disabled and reported `-race requires cgo`. Risk: local race instrumentation was unavailable. Available CI evidence: the exact audited SHA passed `go test -race ./...` on Ubuntu in run 31772889580. |
| `cargo fmt --check` | PASS | Rust formatting check passed. |
| `cargo test --locked` | PASS | 9 tests passed. |
| `cargo clippy --locked --all-targets -- -D warnings` | PASS | No denied warnings. |
| `cargo check --locked` | PASS | Native shell check passed. |
| `cargo audit` | PASS | 438 crates / 1,216 advisories; zero vulnerabilities. Seventeen allowed warnings were reviewed separately. |
| `cargo tree --locked -i quick-xml` | PASS | Locked version is `quick-xml 0.41.0`. |
| `cargo tree --locked -i time` | PASS | Locked version is `time 0.3.47`. |
| `pnpm build:sidecar` | PASS | Target-aware Windows sidecar built. |
| `windows-lifecycle-smoke.ps1` | PASS | Local Windows PowerShell replay verified control-record and EOF shutdown. Exact `pwsh` invocation also passed in CI. |
| `pnpm --filter @yorva/desktop tauri build --no-bundle` | PASS | Packaged native application built. |

The local HTTP smoke replay verified structured `401`, `403`, `404`, and `405` responses; the `Allow: GET, OPTIONS` header; known-route `OPTIONS 204`; unknown-route `OPTIONS 404`; authenticated shutdown; and absence of a residual daemon.

Focused packaged-application manual replay also verified:

- normal Desktop close exits both Desktop and `yorvad` without an orphan;
- forced Desktop parent termination causes stdin EOF and daemon exit without an orphan;
- a missing sidecar leaves the Desktop alive and renders `Connection unavailable / The local daemon could not start.` without a raw path, spawn error, token, CLI text, or stack trace.

These packaged paths are supported by Rust fake-child tests but are not themselves automated by the current Windows lifecycle CI script.

### Exact-commit CI

Run 31772889580 is a `push` run for `phase/001-audit-fix` at `e567c249d68a3ce4c1e7e45b39e79ff4031e3ace`. It completed successfully in 18m14s. All three jobs succeeded:

- Web/API: install, `pnpm audit`, OpenAPI lint/generate/no-drift, typecheck, lint, tests, and build;
- Go Node: `go test -race`, vet, `govulncheck v1.7.0`, vulnerability scan, and build;
- Windows Desktop native shell: sidecar build, lifecycle smoke, Cargo format/test/audit/Clippy/check, and Tauri `--no-bundle` build.

Every inspected action reference in `.github/workflows/ci.yml` is pinned to a 40-character commit SHA. GitHub emitted three maintenance warnings because the pinned checkout/setup-node/setup-go releases still declare the deprecated Node 20 action runtime; GitHub forced them onto Node 24. The run remained successful.

## Dimension Results

| Dimension | Result | Notes |
|---|---|---|
| Scope | PASS | Phase 1 infrastructure only; no Phase 2 or speculative platform behavior found. |
| Correctness | **FAIL** | The React retry window ends before the native startup deadline, so an allowed slow startup can end in a terminal false failure. |
| Architecture | PASS | React, native shell, Go daemon, persistence, and Runtime skeleton ownership boundaries remain intact. |
| Security | PASS | Loopback, bearer auth, origin/CSP, token transport, safe failure text, and secret exposure checks passed. |
| Data and Persistence | PASS | Empty/repeated migrations, stable Node identity, and per-connection SQLite PRAGMAs passed. |
| Concurrency and Lifecycle | PASS | Original child ownership, rollback, timeout, graceful shutdown, parent EOF, and forced fallback defects are repaired and verified. |
| Protocol and Compatibility | PASS | HTTP behavior, OpenAPI, generated TypeScript, protocol text, and contract tests agree. |
| Testing and Verification | **FAIL** | Broad checks and exact CI pass, but the newly identified delayed-ready application transition has no deterministic test. |
| Maintainability | PASS | The implementation remains small and cohesive; no speculative abstraction was introduced. |
| Documentation | PASS | Phase 1 documentation and current root layout are aligned. |
| Dependencies / Supply Chain | PASS | Prior vulnerabilities are removed; Node/Go/Rust advisory gates pass; actions are SHA-pinned. |
| Operations / Diagnostics | **FAIL** | A valid slow start can be reported as a terminal failure; forced-kill errors also remain weakly diagnosable. |

## Exit Criteria Verification

| Phase 1 criterion | Result | Evidence |
|---|---|---|
| Minimal Tauri + React + Go + SQLite foundation builds | VERIFIED | Web, Go, Rust, sidecar, and native build passed locally and in CI. |
| Desktop owns daemon startup and bounded cleanup | VERIFIED | Guarded spawn/write/handshake/timeout state machine, control/EOF shutdown, tests, and packaged smoke. |
| Desktop reaches `starting → connected / connection failure` correctly | **FAILED** | React stops polling at about 4 seconds although native startup remains valid for 10 seconds (MEDIUM-R1-001). |
| Synchronous bootstrap failure keeps the UI alive and safe | VERIFIED | Queryable failed lifecycle state, Rust regression tests, and missing-sidecar packaged smoke. |
| React reads authenticated persisted Node state through the API boundary | VERIFIED | Typed client, handlers, persistence tests, and live HTTP smoke. |
| Loopback, health, bearer-authenticated Node/events, and restrictive origin policy | VERIFIED | Code, protocol tests, and live HTTP smoke. |
| Token is absent from argv, URL, handshake, logs, persistence, and unrestricted debug | VERIFIED | Code/search review and tests. |
| SSE cancellation and bounded subscriber behavior | VERIFIED | Implementation and repeated tests. |
| Empty migration, repeat startup, per-connection PRAGMAs, stable Node identity | VERIFIED | Persistence tests including replacement-connection coverage. |
| Runtime registry remains descriptor-only and Hermes-isolated | VERIFIED | Package and scope inspection. |
| OpenAPI is authoritative and generated TypeScript has no drift | VERIFIED | Local lint plus exact CI generate/diff step. |
| Required verification and dependency review | VERIFIED WITH RECORDED LOCAL LIMITATION | Local matrix passed except Windows CGO race; the exact SHA passed the required Linux race job. |
| Independent audit passes | **FAILED** | This report. |

## Previous Finding Resolution

| Previous finding | Resolution | Evidence |
|---|---|---|
| HIGH-001 — unsafe/orphanable sidecar lifecycle | **RESOLVED** | Rust retains the child before handshake, guards post-spawn write failure, enforces a 10-second startup deadline, performs bounded graceful shutdown with forced fallback, and handles `ExitRequested`/`Exit`. Go monitors stdin and converts EOF/control/context cancellation into bounded HTTP shutdown. Rust/Go tests, lifecycle smoke, and packaged normal/forced-exit replay passed. |
| HIGH-002 — synchronous startup failure aborts Tauri | **RESOLVED** | Setup no longer propagates startup failure; lifecycle state becomes sanitized `Failed`; `daemon_session` returns stable safe codes; React renders the safe state. Rust injected-failure tests and missing-sidecar packaged replay passed. |
| MEDIUM-001 — vulnerable Rust lock graph / absent advisory gates | **RESOLVED** | `quick-xml 0.41.0`, `time 0.3.47`, zero `cargo audit` vulnerabilities, and CI `cargo audit`, `govulncheck`, and `pnpm audit` gates. |
| MEDIUM-002 — HTTP/OpenAPI/error-envelope drift | **RESOLVED** | Router contract now provides structured 401/403/404/405, correct `Allow`/OPTIONS behavior, matching OpenAPI/generated types/protocol text, and contract tests. |
| LOW-001 — mutable CI action tags | **RESOLVED** | All workflow actions use exact 40-character SHAs. |
| LOW-002 — unrestricted `Debug` on token state | **RESOLVED** | No unrestricted `Debug` derive remains on session/lifecycle state. |
| LOW-003 — SQLite PRAGMAs only proven on one connection | **RESOLVED** | DSN PRAGMAs apply to every physical connection; the replacement-connection regression test passes. |
| LOW-004 — README layout mismatch | **RESOLVED** | The root README lists the real top-level layout and no longer claims a current `packages/` directory. |

## Findings

### Critical

None.

### High

None.

### Medium

#### MEDIUM-R1-001 — React's retry budget is shorter than the native startup contract

**Evidence**

- `apps/desktop/src/App.tsx:12-13` retries only while `failures < 20` with a fixed 200 ms delay, a budget of approximately four seconds.
- `apps/desktop/src-tauri/src/daemon.rs:18` defines `STARTUP_TIMEOUT` as ten seconds, and the lifecycle remains legitimately `Starting` until that deadline.
- `daemon_session` returns `DAEMON_NOT_READY` while startup is in progress. When React exhausts its retries, TanStack Query enters terminal `isError`; it does not automatically observe a later `Ready` or final `DAEMON_STARTUP_FAILED` transition without another refetch trigger.
- The React tests exercise the presentational `NodeStatusView` failure rendering but do not mount `App`, mock `getDaemonSession`, or cover delayed readiness/final startup failure.

**Impact**

A daemon that completes a valid handshake between roughly four and ten seconds can be available while the Desktop reports a persistent connection failure. The UI and native lifecycle then disagree about the same startup attempt. This fails the Phase 1 startup success flow and can also hide the final safe startup-failure reason.

**Recommendation**

Derive or otherwise align the UI polling budget with the native startup deadline, and keep observing until the lifecycle reaches `Ready` or its authoritative `Failed` state. Add deterministic application-level tests for delayed readiness near the deadline and for the final sanitized failure transition.

#### MEDIUM-R1-002 — Repository phase state was advanced on `main` before the re-audit gate

**Evidence**

- During this re-audit, remote `main` resolved to merge commit `6b5c418b0cfcf0d4561e698b1f474174b44cfcd3`, whose second parent is the audited `e567c249` and whose message merges PR #1 from `phase/001-audit-fix`.
- `docs/PHASE_GOVERNANCE.md` requires implementation, verification, audit, and freeze in order; a failed phase prohibits next-phase work. The logical `phase-001-bootstrap-baseline` is selected only after the audit.
- No Phase 1 baseline tag was present. The audited branch/commit itself remained unchanged.

**Impact**

The code tree on `main` now contains a phase whose independent gate is still failed, which makes repository state easy to mistake for an accepted Phase 1 baseline. This is an external governance/traceability issue, not an implementation defect in the audited commit, and the merge alone does not constitute a freeze.

**Recommendation**

The repository owner should explicitly reconcile the recorded Phase 1 state, keep Phase 2 locked, and create or identify a Phase 1 baseline only after a passing re-audit. No history rewrite is required by this finding.

### Low

#### LOW-R1-001 — Forced-termination failures are discarded after consuming the child handle

Several cleanup branches invoke the consuming `CommandChild::kill()` and discard its result. The normal control/EOF paths and forced-parent smoke passed, so this does not invalidate the repaired primary lifecycle contract. A rare OS termination failure, however, loses the retry handle and yields no actionable diagnostic. Preserve a safe diagnostic/ownership strategy for this fallback when practical and add a fault-injection test if the API permits it.

### Info

#### INFO-R1-001 — Native-shell integration coverage is partly manual

CI's Windows lifecycle smoke launches `yorvad` directly. Rust fake-child tests cover native state transitions, and this audit manually verified actual packaged Desktop close, parent termination, and missing-sidecar rendering, but these packaged paths are not automated. This is useful future hardening; it is not the evidence gap causing the gate failure.

#### INFO-R1-002 — Pinned GitHub actions declare a deprecated Node 20 runtime

GitHub warned that the pinned checkout/setup-node/setup-go revisions declare Node 20 and forced Node 24. The exact-SHA run succeeded and action pinning is correct. Upgrade to reviewed SHA revisions that natively support the current action runtime before GitHub removes compatibility.

#### INFO-R1-003 — Cargo audit reports allowed maintenance warnings

`cargo audit` found zero vulnerabilities and 17 allowed warnings: GTK3-related unmaintained crates, `proc-macro-error`, five `unic-*` crates, and `glib 0.18.5` / RUSTSEC-2024-0429. The GTK/glib chain is Linux/BSD-targeted and was not compiled or reachable in the Phase 1 Windows target; no current Windows exploit path was established. Monitor these transitive dependencies during routine Tauri upgrades.

## Accepted Technical Debt

None. The repository owner has not explicitly accepted MEDIUM-R1-001 or MEDIUM-R1-002 with owners and resolution triggers. This audit does not invent that acceptance.

## Required Fixes Before Next Phase

1. Resolve MEDIUM-R1-001 by aligning React observation with the authoritative ten-second native startup lifecycle and add deterministic application-level delayed-ready/final-failure tests.
2. Re-run the affected React/native verification and independently re-audit the changed startup path.
3. Reconcile MEDIUM-R1-002 in repository phase records. Treat the existing `main` merge as implementation history, not as a passed/frozen baseline.
4. Do not begin Phase 2 and do not create the Phase 1 baseline tag until a valid passing gate exists.

LOW-R1-001 and the informational maintenance items should be tracked, but they are not independently blocking.

## Gate Rationale

All Critical and High findings are closed, all eight findings from `AUDIT-001` are resolved, and the exact-commit verification evidence is strong. That is not sufficient for `PASS`, because the valid slow-start interval exposes a failed mandatory Desktop startup success flow. `AUDIT_STANDARD.md` section 17 requires `FAIL` when the phase success flow or a mandatory acceptance criterion fails.

`PASS WITH CONDITIONS` is also unavailable: sections 16 and 17 allow it only when remaining findings are explicitly accepted and each has an owner and resolution trigger. No such acceptance exists. The correct gate is therefore `FAIL`, narrowly caused by MEDIUM-R1-001 and reinforced by the unresolved phase-state reconciliation in MEDIUM-R1-002.

## Next Step

Keep Phase 1 in the fix/re-audit loop. Make the smallest Phase 1-only correction for MEDIUM-R1-001, add the deterministic regressions, re-run the affected matrix, and issue another independent re-audit without overwriting this report or `AUDIT-001`. The repository owner should separately reconcile the premature `main` merge. Phase 2 remains prohibited.

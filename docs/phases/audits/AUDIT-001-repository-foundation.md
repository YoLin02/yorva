# YORVA Phase 1 Audit — Repository Foundation

## Phase

Phase 1 — Repository foundation / Bootstrap.

Implementation status at audit entry: `COMPLETE`; local verification was reported as `PASS`; independent audit was pending.

## Baseline / Commit

- Phase 1 starting baseline: tag `phase-000-docs-baseline`.
- Baseline and current `HEAD`: `ad4267c` (`docs: establish YORVA phase 0 architecture baseline`).
- Audited implementation: the complete working tree relative to `ad4267c`, including untracked files.
- The implementation had not been committed at audit time. `git diff --stat phase-000-docs-baseline` therefore showed only the two tracked documentation changes, while `git status --porcelain=v2` showed the application, daemon, API, CI, lockfiles, scripts and repository configuration as untracked. Those untracked files were explicitly included in this audit.

## Auditor

Independent Codex audit context. The auditor did not participate in the Phase 1 implementation and made no implementation changes.

## Date

2026-08-14

## Gate Decision

**FAIL**

## Executive Summary

Phase 1 establishes most of the intended repository foundation and passes a broad local build/test replay, but it is not safe to freeze.

The implementation correctly preserves several critical boundaries:

- React accesses `yorvad` only through the generated/typed API layer and the narrow Tauri session command; it contains no Hermes, shell, filesystem or SQLite integration.
- Tauri contains bootstrap/native lifecycle code only; it contains no Runtime business behavior.
- `yorvad` binds with `net.Listen("tcp4", "127.0.0.1:0")`.
- `/api/v1/health` is minimal and unauthenticated; `/api/v1/node` and `/api/v1/events` use the same bearer-auth middleware.
- The 256-bit session token is generated in Rust, sent to the child over stdin, retained in memory, omitted from argv, handshake output, URLs, SQLite and current logs.
- SQLite migration from an empty database, repeat startup and stable Node identity pass.
- SSE cancellation removes subscribers, buffers are bounded and slow subscribers do not block publication.
- the Runtime Registry contains only a compile-time Hermes Descriptor; no Hermes probing, CLI, PATH, config, profile, Cloud or dynamic-plugin behavior exists.
- OpenAPI validation and generated TypeScript drift checks pass.

Two unresolved High findings block the gate:

1. the Tauri-owned sidecar lifecycle has no parent-death contract, no startup timeout and no graceful normal shutdown; partial handoff failures can also lose the only kill handle;
2. synchronous sidecar/bootstrap failures propagate out of Tauri setup and abort application construction before React can show the required user-safe connection-failure state.

The audit also found three locked Rust vulnerabilities, incomplete structured-error/OpenAPI coverage, and lower-severity hardening/documentation issues. Under `AUDIT_STANDARD.md`, unresolved High lifecycle/correctness findings require `FAIL`. Phase 2 must not begin.

## Evidence / Verification Evidence

### Governing material read

The auditor completely read:

- `AGENTS.md` and `README-DOCS.md`;
- `docs/ARCHITECTURE.md`, `docs/DEVELOPMENT.md`, `docs/PROTOCOL.md`, `docs/RUNTIME.md`, `docs/DATA_MODEL.md`, and `docs/SECURITY.md`;
- `docs/BOOTSTRAP.md`, `docs/PHASE_GOVERNANCE.md`, `docs/AUDIT_STANDARD.md`, and `docs/ROADMAP.md`;
- ADR-0001 through ADR-0004;
- `docs/phases/PHASE_TEMPLATE.md`, `docs/phases/audits/AUDIT_TEMPLATE.md`, and `docs/phases/audits/AUDIT-000-phase0-readiness.md`.

### Repository and diff inspection

The audit inspected:

- `git status --short --branch`, `git status --porcelain=v2`, `git log`, `git diff`, `git diff --stat phase-000-docs-baseline`, and `git diff --check`;
- all Phase 1 Go, Rust, TypeScript/React, SQL, OpenAPI, CI, build, manifest, lockfile and repository configuration files;
- source/import searches for Hermes, shell/process execution, plugin/Cloud scope, token/Authorization handling, public binding, storage, unsafe rendering, CSP and browser persistence;
- the exact locked `tauri-plugin-shell 2.3.5` process implementation to establish `CommandChild` ownership/drop behavior;
- target-specific Cargo dependency paths for vulnerable and unmaintained crates.

`git diff --check` passed. Verification commands did not alter tracked implementation files; the final status remained the same apart from this audit report.

### Toolchain used for replay

```text
Git       2.55.0.windows.3
Node.js   22.23.1
pnpm      11.15.1
Go        1.26.6 windows/amd64 (isolated official toolchain)
Rust      1.97.1 x86_64-pc-windows-msvc (isolated official toolchain)
Cargo     1.97.1
Tauri CLI 2.11.4
MSVC      Visual Studio 2026 Build Tools
```

Go and Rust were not initially available on the audit process `PATH`. Official toolchains were installed under a system temporary audit directory and used with temporary caches/target directories; no repository implementation file was changed for this purpose.

### Commands and results

| Command / check | Result | Evidence |
|---|---|---|
| `pnpm install --frozen-lockfile` | PASS | Lockfile reproduced; workspace already up to date. |
| `pnpm api:lint` | PASS | Redocly validated `api/openapi.yaml`. |
| OpenAPI generation to a temporary file + SHA-256 comparison | PASS | Generated output exactly matched `apps/desktop/src/api/generated/schema.ts`. |
| `pnpm typecheck` | PASS | TypeScript project references compiled. |
| `pnpm lint` | PASS | ESLint completed without findings. |
| `pnpm test` | PASS | 3 files, 7 tests passed. |
| `pnpm build` | PASS | Vite production build completed. |
| `go test ./...` | PASS | All Go package tests passed. |
| `go vet ./...` | PASS | No vet findings. |
| `go build -trimpath ./cmd/yorvad` | PASS | Audit binary built outside the repository. |
| SSE cancellation test `-count=50` | PASS | 50 repeated executions passed. |
| daemon bootstrap integration test `-count=20` | PASS | 20 repeated executions passed. |
| local `go test -race ./...` | NOT RUN | Windows audit toolchain reported `-race requires cgo`; CGO was not available. `.github/workflows/ci.yml:44` runs `go test -race ./...` on Ubuntu, where the hosted image supplies a C toolchain, but this uncommitted working tree had no remote successful CI run to inspect. |
| `govulncheck ./...` | PASS | No reachable Go vulnerabilities found. |
| `cargo fmt --check` | PASS | Formatting check passed. |
| `cargo test --locked` | PASS | 3 Rust tests passed. |
| `cargo clippy --locked --all-targets -- -D warnings` | PASS | No denied warnings. |
| `cargo check --locked` | PASS | Native shell check passed. |
| `pnpm build:sidecar` | PASS | Target-aware Windows sidecar built. |
| `pnpm --filter @yorva/desktop tauri build --no-bundle` | PASS | Windows release application built successfully. |
| `pnpm audit --audit-level low` | PASS | No known Node vulnerabilities found. |
| `cargo audit` | FAIL | 3 vulnerabilities and 17 allowed warnings found in the locked Rust graph. See MEDIUM-001. |

Rust test/release linking emitted a localized MSVC informational message that Rust recorded as a `linker_messages` warning. It did not fail tests, Clippy, check or the release build.

## Dimension Results

| Dimension | Result | Notes |
|---|---|---|
| Scope | PASS | Phase 1 infrastructure only; no Phase 2 Hermes discovery/install behavior, Cloud, accounts, telemetry, updater, service installation or dynamic plugin system was found. |
| Correctness | FAIL | Synchronous bootstrap failures abort Tauri construction instead of reaching the required connection-failure UI (HIGH-002). |
| Architecture | PASS | React → API/Tauri session and Tauri → `yorvad` boundaries are narrow; Hermes remains isolated static metadata; SQLite is Go-owned. |
| Security | PASS | Loopback, bearer auth, CSP, token transport/storage and log/URL/argv checks pass. LOW-002 is defense-in-depth, not a confirmed disclosure. |
| Data and Persistence | PASS | Empty and repeated migration, PRAGMAs on the active connection and stable Node ID pass. LOW-003 records a reconnect-hardening issue. |
| Concurrency and Lifecycle | FAIL | Sidecar parent death, partial startup, startup timeout and graceful shutdown ownership are incomplete (HIGH-001). |
| Protocol and Compatibility | FAIL | Core route schemas/types are synchronized, but actual 403/404/405 behavior is not consistently modeled or enveloped (MEDIUM-002). |
| Testing and Verification | FAIL | Broad checks pass, but blocking startup/owner-death paths have no tests; local race was unavailable and no run exists for the uncommitted CI workflow. |
| Maintainability | PASS | The implementation is small and cohesive; no speculative service/repository/DI/plugin framework was introduced. |
| Documentation | PASS | Core architecture/phase documents remain aligned. LOW-004 is a bounded README layout mismatch. |
| Dependencies / Supply Chain | FAIL | Rust lockfile contains three RustSec vulnerabilities and CI has no dependency advisory step (MEDIUM-001). Node/Go audits passed. |
| Operations / Diagnostics | FAIL | Async daemon failure is sanitized for React, but synchronous startup failures terminate app construction and no startup timeout/cleanup is present. |

## Exit Criteria Verification

| Phase 1 criterion | Result | Evidence |
|---|---|---|
| Repository/module identity is real and non-placeholder | VERIFIED | `github.com/YoLin02/yorva/services/node`; `HEAD` tagged `phase-000-docs-baseline`. |
| Minimal Tauri + React + Go + SQLite foundation builds | VERIFIED | Web, Go, Rust, sidecar and `tauri build --no-bundle` passed. |
| Desktop starts/discovers daemon safely | **FAILED** | Startup error propagation and lifecycle ownership are unsafe; HIGH-001/HIGH-002. |
| React reads authenticated persisted Node state through API boundary | VERIFIED | `App.tsx`, `api/client.ts`, Go handler/integration tests. |
| React opens bearer-authenticated fetch SSE | VERIFIED | `api/client.ts:41-75`; no `EventSource` or URL token. |
| Connection failure is shown safely | **FAILED** | Only post-construction session/query failures reach React; synchronous startup errors abort Tauri setup. |
| `yorvad` binds `127.0.0.1:0` | VERIFIED | `services/node/internal/daemon/daemon.go:73`. |
| `/health` minimal and unauthenticated | VERIFIED | Handler and `TestHealthIsMinimalAndUnauthenticated`. |
| `/node` and `/events` authenticated | VERIFIED | Shared bearer middleware and positive/missing/invalid tests. |
| Token absent from argv, URL, handshake, log and persistence | VERIFIED | Direct code/search review and bootstrap/client tests. |
| SSE cancellation/subscriber cleanup | VERIFIED | Implementation, unit test and 50-repeat audit replay. |
| Child/long-lived lifecycle has bounded safe cleanup | **FAILED** | HIGH-001. |
| Empty DB migration and idempotent restart | VERIFIED | `TestMigrationsAreIdempotentAndNodeIdentityPersists`. |
| Stable Node identity across restart | VERIFIED | Same test and persisted `nodes` row. |
| Runtime Registry is skeleton-only | VERIFIED | Descriptor-only bundle; no `Detect`, command, PATH or filesystem probing. |
| No Phase 2 Hermes capability | VERIFIED | Code/import/scope search and package inspection. |
| OpenAPI is schema source of truth and generated TS has no drift | PARTIAL | Generation is exact for declared schemas; MEDIUM-002 identifies undeclared/non-envelope response behavior. |
| Stable structured error model | **FAILED** | Auth/origin errors are structured, but router method/not-found responses are not and origin errors are absent from OpenAPI. |
| Restrictive CORS/CSP | VERIFIED | Exact origin allowlist, forbidden-origin test, no disabled CSP, no shell capability exposed to React. |
| CI covers race test unavailable locally | CONFIGURED, NOT OBSERVED | Ubuntu job runs `go test -race ./...`; current implementation is uncommitted, so no successful run exists for this exact tree. |
| Required local typecheck/lint/test/build/vet/clippy/check | VERIFIED | All non-race commands passed. |
| Dependency risk acceptable | **FAILED** | `cargo audit` found three vulnerabilities. |
| Phase 1 independent audit passes | **FAILED** | This report. |

## Findings

### Critical

None.

### High

#### HIGH-001 — The Tauri-owned sidecar lifecycle does not guarantee bounded cleanup or graceful shutdown

**Evidence**

- `apps/desktop/src-tauri/src/daemon.rs:85-93` removes the child handle and calls `child.kill()`; there is no graceful shutdown request or wait/kill escalation.
- `apps/desktop/src-tauri/src/lib.rs:19-22` invokes that logic only for `RunEvent::Exit`.
- `apps/desktop/src-tauri/src/daemon.rs:151-161` spawns the child and writes the bootstrap message before storing the `CommandChild`; if `write()` fails, the function returns and YORVA has no retained kill handle.
- `apps/desktop/src-tauri/src/daemon.rs:163-203` waits indefinitely for process events and has no bootstrap deadline.
- `services/node/internal/bootstrap/stdio.go:31-48` reads a single bootstrap object and does not retain/monitor stdin for parent loss.
- `services/node/internal/daemon/daemon.go:104-122` performs graceful HTTP shutdown only when its context is cancelled by an OS interrupt/SIGTERM. The Rust normal-close path uses child termination instead of that path.
- Exact-source inspection of locked `tauri-plugin-shell 2.3.5` found an explicit consuming `CommandChild::kill(self)` and no `Drop` implementation that kills the process. Its wait thread owns another `Arc<SharedChild>`, so dropping YORVA's handle is not a kill contract.
- Existing Rust tests cover token generation/handshake parsing only; no test covers parent death, stdin closure, partial handoff, startup timeout or graceful close.

**Impact**

An abnormal Desktop termination can leave an authenticated `yorvad` process running with no owning UI. A failed stdin handoff can also lose the kill handle, and a daemon that never produces a handshake remains alive without a timeout. Normal Desktop exit forcibly terminates the child rather than allowing `http.Server.Shutdown` and database cleanup to complete. This violates the Phase 1 ownership/lifecycle foundation that future Operations and Runtime work would inherit.

**Recommendation**

Define one bounded, Windows-tested ownership mechanism before Phase 1 is frozen. It should cover graceful normal shutdown, parent-death cleanup, startup deadline, partial-spawn rollback and forced-kill fallback. Examples include a dedicated authenticated shutdown/control path plus timeout, parent/pipe monitoring in `yorvad`, or a Windows Job Object with kill-on-close semantics. Add tests for every transition and prove no sidecar remains after normal close, startup failure, timeout and forced parent termination.

#### HIGH-002 — Synchronous daemon startup failures bypass the required safe React failure state

**Evidence**

- `apps/desktop/src-tauri/src/lib.rs:12-17` propagates `start_daemon` from `.setup()` with `?`, then calls `.build(...).expect(...)`.
- `apps/desktop/src-tauri/src/daemon.rs:141-161` can synchronously fail while resolving the data directory, converting its path, generating randomness, resolving/spawning the sidecar or writing stdin.
- `apps/desktop/src/App.tsx:26-33` can render a user-safe failure only after Tauri has built and the WebView can invoke `daemon_session`.
- `docs/BOOTSTRAP.md:670` requires a concise user-safe error when daemon bootstrap fails; the completion checklist also requires a visible safe error state.
- Rust tests do not inject a missing sidecar, spawn error, stdin write failure or data-path failure.

**Impact**

Common bootstrap failures such as a missing/corrupt sidecar abort application construction before the UI is available. The required `Connection failed` state is therefore not implemented for the full bootstrap failure surface, and the user receives process termination/panic behavior instead of an actionable safe error.

**Recommendation**

Keep the Tauri application alive when startup fails. Convert synchronous failures into a sanitized `StartupStatus::Failed`, clean up any partially spawned child, and let the existing narrow session command expose the stable error to React. Add deterministic Rust/application tests for each synchronous failure class and a Windows smoke test proving the failure screen is rendered.

### Medium

#### MEDIUM-001 — The locked Rust dependency graph contains three known vulnerabilities and CI does not detect them

**Evidence**

- `cargo audit` scanned 438 locked dependencies and failed with:
  - `quick-xml 0.38.4`: [RUSTSEC-2026-0194](https://rustsec.org/advisories/RUSTSEC-2026-0194.html), quadratic attribute-check CPU denial of service, RustSec High, patched in `>=0.41.0`;
  - `quick-xml 0.38.4`: [RUSTSEC-2026-0195](https://rustsec.org/advisories/RUSTSEC-2026-0195.html), unbounded namespace allocation denial of service, RustSec High, patched in `>=0.41.0`;
  - `time 0.3.45`: [RUSTSEC-2026-0009](https://rustsec.org/advisories/RUSTSEC-2026-0009.html), RFC 2822 parsing stack exhaustion, RustSec Medium, patched in `>=0.3.47`.
- `apps/desktop/src-tauri/Cargo.lock:2312-2313` locks `quick-xml 0.38.4`; lines `3333-3334` lock `time 0.3.45`.
- Target-specific `cargo tree -i` shows the crates enter through `plist`/`tauri-utils` and `cookie`/Tauri/Wry.
- Phase 1 does not itself accept untrusted XML or RFC 2822 input, so the audit did not establish a currently reachable exploit path. This is why the YORVA finding is Medium rather than inheriting RustSec's package CVSS mechanically.
- `.github/workflows/ci.yml` runs compile/test/lint checks but no `cargo audit`, `govulncheck` or package advisory gate.
- `cargo audit` also reported 17 allowed unmaintained/unsound warnings. GTK warnings are not in the Windows target graph; several `unic-*` warnings remain reachable through `urlpattern`/`tauri-utils`.

**Impact**

The repository can freeze and repeatedly build a dependency graph already known to be vulnerable. Future UI or Tauri usage could make a currently dormant parsing path reachable, and CI would not report the change.

**Recommendation**

Update the compatible Tauri/transitive dependency set so `cargo audit` has no vulnerabilities, or document a reviewed, time-bounded exception with reachability evidence if upstream blocks the update. Add a lightweight advisory check to CI and define handling for target-specific unmaintained warnings.

#### MEDIUM-002 — Actual HTTP error/CORS behavior is not fully represented by OpenAPI or the standard error contract

**Evidence**

- `services/node/internal/transport/httpapi/server.go:116-143` can return structured `403 ORIGIN_NOT_ALLOWED` and unauthenticated `OPTIONS 204` responses.
- The same `http.ServeMux` returns standard-library plain-text responses for unmatched paths and method mismatches; those responses do not use `ErrorResponse` from `errors.go:8-25`.
- `api/openapi.yaml:16-62` declares only the endpoint success responses and `401` for Node/events. It does not declare 403, OPTIONS, 404 or 405 behavior.
- `docs/BOOTSTRAP.md:860-888` requires the standard transport error envelope, and `docs/PROTOCOL.md` defines OpenAPI as the HTTP schema source of truth.
- Generated TypeScript exactly matches the OpenAPI file, so this is contract/handler drift rather than stale generation.

**Impact**

Desktop or future local clients cannot rely on one stable error model for the implemented API surface. Some real server responses are outside the protocol fact source, and unknown-route/method failures require text/status special cases.

**Recommendation**

Make routing/CORS failure responses consistently use the YORVA error envelope and document all intended endpoint responses in OpenAPI. If preflight responses are intentionally transport-only, state that boundary explicitly and test it. Add contract tests for 403, 404 and 405.

### Low

#### LOW-001 — GitHub Actions are referenced by mutable major-version tags

`.github/workflows/ci.yml` uses `actions/checkout@v4`, `actions/setup-node@v4`, `actions/setup-go@v5`, `pnpm/action-setup@v4` and `dtolnay/rust-toolchain@stable`. These are official actions, but mutable tags weaken workflow supply-chain reproducibility. Pin action revisions to reviewed commit SHAs and use an explicit Rust toolchain action revision.

#### LOW-002 — Secret-bearing Rust state derives unrestricted `Debug`

`apps/desktop/src-tauri/src/daemon.rs:16-22` derives `Debug` for `DaemonSession`, whose `token` field contains the bearer credential; containing lifecycle types also derive `Debug`. No current code formats these values, so no disclosure was found. Remove `Debug` or implement a redacted formatter to prevent a future diagnostic statement from exposing the token.

#### LOW-003 — SQLite security/integrity PRAGMAs are configured only on the current pooled connection

`services/node/internal/persistence/sqlite/database.go:33-51` executes `foreign_keys` and `busy_timeout` after opening/pinging one connection. With `SetMaxOpenConns(1)` and unlimited lifetime, the normal path retains that connection, and the current tests pass. If `database/sql` replaces a bad connection, the new physical connection is not explicitly initialized. Configure required PRAGMAs for every connection through the driver DSN/connector and add a reconnection test before foreign-key-bearing migrations arrive.

#### LOW-004 — README documents a package directory that is not present

`README.md:12` lists `packages/` as a current shared generated-contract directory, but generated types live under `apps/desktop/src/api/generated/` and no `packages/` directory exists. The implementation correctly avoided creating an unused package; update the README layout instead.

### Info

#### INFO-001 — Core Phase 1 trust boundaries are correctly implemented

Direct inspection found no React → Hermes/shell/SQLite path, no Tauri Runtime business logic, no public bind, no unauthenticated Node/events path, no token in argv/URL/persistence/current logs, and no Phase 2 or Cloud/plugin implementation.

#### INFO-002 — CI is designed to cover the locally unavailable Go race build

The Linux CI Node job uses `go test -race ./...`, which is the correct coverage point for the audit host's missing Windows CGO race support. Because all implementation files and the workflow itself were uncommitted, this audit could verify the configuration but could not inspect a successful run for the exact tree.

#### INFO-003 — License and temporary application identity remain correctly deferred

No license was generated. `README.md` explicitly states licensing is pending, and `tauri.conf.json` uses the reviewable development identifier `com.yorva.desktop.dev`.

## Technical Debt

No technical debt is accepted by this audit. The project owner may classify bounded Low/Info items after the blocking fixes, but correctness, lifecycle, protocol and known-vulnerability findings must not be relabeled as debt merely to pass the gate.

Potential future debt requiring explicit owner/trigger if deferred after re-audit:

- target-specific unmaintained Rust transitive dependencies that cannot yet be removed upstream;
- GitHub Action SHA pinning;
- per-connection SQLite PRAGMA hardening;
- README layout correction.

## Accepted Technical Debt

None.

## Required Fixes Before Next Phase

1. Resolve HIGH-001 with a bounded, graceful and parent-death-safe sidecar lifecycle on Windows, including partial-startup rollback and automated lifecycle tests.
2. Resolve HIGH-002 so every synchronous daemon startup failure becomes a sanitized observable UI state without aborting Tauri application construction.
3. Resolve or explicitly owner-review MEDIUM-001; rerun `cargo audit` and add advisory monitoring appropriate to the repository.
4. Resolve MEDIUM-002 by aligning handler/CORS error behavior, OpenAPI and protocol tests.
5. Re-run the full Web/OpenAPI, Go, Rust/Tauri, sidecar and dependency verification matrix.
6. Run `go test -race ./...` in CI and retain a successful result for the committed re-audit tree.
7. Re-audit at least Correctness, Concurrency/Lifecycle, Security, Protocol/Compatibility, Testing/Verification, Dependencies/Supply Chain and Operations/Diagnostics.

## Gate Rationale

`AUDIT_STANDARD.md` requires `FAIL` for an unresolved High finding, a failed mandatory acceptance criterion, or insufficient lifecycle evidence for a critical capability. HIGH-001 leaves the Phase 1 daemon ownership model unsafe, and HIGH-002 fails the required user-safe bootstrap failure behavior. These are foundation defects that would contaminate Phase 2 installation/Operation work if frozen now.

The many passing tests and sound architecture/security boundaries reduce repair scope but do not override those gate rules. The three known Rust vulnerabilities and protocol drift provide additional reasons not to freeze the current tree.

## Next Step

Stop at Phase 1. Do not implement or detail-design Phase 2.

Create a separate Phase 1 audit-fix task for the required fixes, rerun verification, and perform a focused independent re-audit. Only a subsequent `PASS` or owner-accepted `PASS WITH CONDITIONS`, followed by a clean baseline commit/tag and `FROZEN` state, may unlock Phase 2.

# YORVA Phase 3 Independent Re-Audit R2 — Hermes Installation

## Phase

Phase 3 — Hermes Installation

## Baseline / Commit

- Historical Phase 2 baseline: `phase-002-hermes-discovery-baseline-r1` → `5b89d22ed5e7ae3f4374a26f0fcda54bdabc6bf9`
- Historical Phase 3 audit candidate: `f93075428480133698f93e7915e409304aa55b68` (`AUDIT-003R1` — FAIL)
- R2 remediation branch: `fix/phase3-audit-r1-remediation`
- Immutable R2 candidate: `13d3739e0f6379aee1253abecdd4c44c59d1c31b`
- Remote branch tip at audit: `origin/fix/phase3-audit-r1-remediation` → the same SHA
- `main` / `origin/main` at audit: `9459eb9fea435704d6639be8457ee11dfad02093`

The candidate is six commits ahead of the Owner-merged implementation history. This audit does not merge, freeze, tag or delete a branch.

## Auditor

Fresh independent Phase 3 re-audit context (R2), reviewing repository state rather than the implementation agent's completion statement.

## Date

2026-08-18 (Asia/Shanghai)

## Gate Decision

**FAIL**

## Executive Summary

The R2 candidate materially improves the Phase 3 implementation. It closes the MSI filename-collision and embedded-content verification defect, simultaneous same-key Operation identity, structured/redacted archive logging, typed Operation SSE notifications, conflict routing, and exact managed Node/npm version policy. Exact-commit CI and MSI packaging are green.

The gate nevertheless remains blocked. Desktop recovery is still conditional on discovery remaining `NOT_INSTALLED`; during a real partial installation discovery can be `BROKEN_EXECUTABLE`, which hides the recovered running Operation and its Cancel control. Retry ownership still treats an exact, public marker as complete ownership proof, permits a prior durable Operation with an empty source pin, and deliberately deletes extra content beneath a marked target. This does not establish that the current target is the same unchanged YORVA-owned partial tree audited in R1.

The required managed-Node negative test matrix and candidate completion evidence also remain incomplete, and the HTTP implementation does not fully enforce the OpenAPI closed-empty-object request contract. Under the Phase 3 Spec, any unresolved High or Medium finding requires `FAIL`.

## Verification Evidence

### Repository and candidate

- `git branch --show-current` → `fix/phase3-audit-r1-remediation`
- `git rev-parse HEAD` → `13d3739e0f6379aee1253abecdd4c44c59d1c31b`
- `git rev-parse origin/main` → `9459eb9fea435704d6639be8457ee11dfad02093`
- `git ls-remote origin refs/heads/fix/phase3-audit-r1-remediation` → `13d3739e0f6379aee1253abecdd4c44c59d1c31b`
- `git diff --check f930754..13d3739` → PASS
- Audit-start worktree contained one pre-existing untracked MSI inspection residue: `YORVA_0.3.0_x64_en-US.wxs`. It is not part of the immutable candidate and was not modified by this audit.

### Exact-commit GitHub Actions

- CI run `32091058398` — `push`, branch `fix/phase3-audit-r1-remediation`, head `13d3739e0f6379aee1253abecdd4c44c59d1c31b`, **SUCCESS**:
  - Web and API contract — success;
  - Go Node, including `go test -race ./...`, vet, govulncheck and build — success;
  - Windows Desktop native shell, lifecycle smoke, Cargo checks/audit and Tauri no-bundle — success.
- MSI run `32091058406` — same branch and head, **SUCCESS**:
  - package and fail-closed inspection — success;
  - artifact `yorva-msi`, approximately 114 MB;
  - GitHub artifact digest `sha256:bfe700125513fa7ca732d7209cbd840576d17ec6f881266917202b97650e811e`.

### Local Web / OpenAPI

- `pnpm api:lint` — PASS.
- `pnpm api:generate` followed by generated-schema diff check — PASS, no generated diff.
- `pnpm typecheck` — PASS.
- `pnpm lint` — PASS.
- `pnpm test` — PASS, 16 files / 58 tests.
- `pnpm build` — PASS.
- `pnpm audit --audit-level low` — PASS, no known vulnerabilities.

### Local Go

- `go test ./...` — PASS, including the real Windows process-containment tests available on this host.
- affected application, Hermes adapter, SQLite and HTTP packages with `-count=20` — PASS.
- `go vet ./...` — PASS.
- `go build ./cmd/yorvad` — PASS.
- `govulncheck ./...` — PASS, no vulnerabilities found.
- local `go test -race ./...` — not run successfully because this host has CGO disabled / no usable C toolchain (`-race requires cgo`). The exact-commit Linux CI race run above is PASS; this report does not label the local race command PASS.

### Local Rust / Tauri / Windows

- `cargo fmt --all -- --check` — PASS.
- `cargo test --locked` — PASS, 10 tests.
- `cargo clippy --locked --all-targets --all-features -- -D warnings` — PASS.
- `cargo check --locked` — PASS.
- `cargo audit` — zero vulnerabilities; 17 inherited allowed unmaintained/unsound warnings from the existing Tauri dependency graph.
- `scripts/windows-lifecycle-smoke.ps1` — PASS.
- local Tauri release no-bundle — blocked by `PermissionDenied` because the Owner's `target/release/yorva-desktop.exe` process was running. The process was not terminated. Exact-commit Windows CI completed the same no-bundle check successfully.

### MSI inspection

- `scripts/inspect-yorva-msi.tests.ps1` — PASS for missing Hermes LICENSE, suffix collision, duplicate identity, wrong filename, wrong size, same-size/wrong-hash, wrong license content, substituted archive, unexpected executable and extraction failure.
- The current inspector successfully administratively extracted the available local MSI and re-hashed the actual embedded bytes:
  - Hermes archive: `2ED02F76AAF5DAB0BFD320BDBFA10AAD0F67E00CBBF87906CDE05462681708BA`;
  - Node archive: `7DF0BC9375723F4A86B3AA1B7CC73342423D9677A8DF4538ACA31A049E309C29`;
  - npm archive: `5DBB86C71D07A1957F2E90734092DD6A58BDCD9EBC2D8D41CA1C6E6A21D364E1`;
  - Hermes LICENSE: `821556E6336796450AB852D375117B48A4887E71D255794FD6318D99982A5AB6`;
  - Node LICENSE: `8CC9BB466B19FC7E7CC99D03E9DF1132021FDA8B01EEA2624C58BB372DBEF576`;
  - npm LICENSE: `7610D223851F421D315DF5E77974F1C68A04B97E02060E5BBBCF13D95E3CA257`.
- The local MSI itself is supplemental evidence and is not asserted to be the uploaded exact-commit artifact; exact-commit identity is established by run `32091058406` and its artifact digest.

## R1 Finding Closure

| R1 finding | R2 result | Evidence |
| --- | --- | --- |
| HIGH-R1-001 Desktop restart recovery | **NOT CLOSED** | List/recovery and type guards were added, but `App.tsx:331-349` renders the install panel only when discovery is `NOT_INSTALLED`. A partially materialized active install can be `BROKEN_EXECUTABLE`, hiding the recovered operation and Cancel UI. The recovery test fixes discovery to `NOT_INSTALLED` (`App.install-recovery.test.tsx:87-96`) and does not cover the real partial-install state. |
| HIGH-R1-002 retry target ownership | **NOT CLOSED** | `target.go:65-76` proves only marker syntax/pin. `target_test.go:180-198` explicitly accepts and deletes a marked tree containing an extra `stale.exe`. `RetryEligibleForPin` also accepts an empty durable `SourcePin` (`runtime_install.go:347-360`). |
| HIGH-R1-003 MSI acceptance inspection | CLOSED | Exact decoded names, exactly-one checks, administrative extraction and embedded byte hashes are enforced in `inspect-yorva-msi.ps1:21-173`; negative suite and exact MSI CI pass. |
| HIGH-R1-004 concurrent same-key identity | CLOSED | Application, SQLite and HTTP simultaneous tests return the same Operation; duplicate-key recovery validates type/target; exact CI race passes. |
| MEDIUM-R1-001 raw archive log leakage | CLOSED | `archiveLogFields` contains only stable structured fields and sentinel tests cover stderr, file and HTTP-readable logs. |
| MEDIUM-R1-002 managed Node behavioral tests | **NOT CLOSED** | Materialization, argv, stamp and exact-pin coverage improved, but the required entry-count/member-size/uncompressed-size rejection and whole-Operation deadline behavior are not exercised. `TestCompiledExtractionLimits` only asserts constants; the wrong-root helper test expects a nil error and merely checks that no file appeared. |
| MEDIUM-R1-003 governance consistency | **NOT CLOSED** | Phase Spec completion evidence still names commit `2f1e498...` and says exact CI/MSI are PENDING (`PHASE-003-hermes-installation.md:754-769`) although the immutable candidate is `13d3739...` and both workflows are complete. Amendment 003A1 §13 also retains pre-003A3 wording that `node` and `node-deps` continue as skip-with-warning stages. |
| MEDIUM-R1-004 Operation SSE | CLOSED | Typed, redacted events are emitted only after committed creates/transitions, all terminal types are covered, and disconnect plus GET recovery is tested. |
| MEDIUM-R1-005 wrong Desktop panel | CLOSED | Conflict IDs are fetched and checked for exact type/target; both conflict directions are tested and actions are mutually blocked. |
| LOW-R1-001 exact managed pins | CLOSED | Managed health now requires Node `22.23.1` and npm `12.0.2`; a higher compatible replacement is rejected by tests. |

## Dimension Results

### Scope

PASS. The diff stays within Phase 3 remediation, tests, MSI packaging and contract documentation. No Phase 4 Instance/Profile behavior, login, model, channel, lifecycle, Cloud or plugin work was added.

### Correctness

FAIL. The real active-install reload state can hide the Operation, and retry eligibility does not prove an unchanged owned target.

### Architecture

PASS. React continues through the typed daemon API; installation orchestration stays in the application/Hermes adapter; Rust remains a resource/bootstrap shell. No new framework or dynamic Runtime plugin system was introduced.

### Security

FAIL. Marker-only ownership permits destructive replacement of externally changed content under the fixed target. Raw archive diagnostic leakage and MSI embedded-content verification are otherwise corrected.

### Data and Persistence

FAIL. Migrations and constraints pass, but migrated empty `source_pin` values remain accepted as pin-matching retry evidence, so durable history does not prove the required source identity.

### Concurrency and Lifecycle

PASS. Cross-operation exclusion, simultaneous same-key recovery, cancellation, stale Operation interruption and process-tree tests pass. The local race environment is blocked, while exact-commit race CI is green.

### Protocol and Compatibility

FAIL. Operation SSE is now implemented, but the closed empty-object request contract is not fully enforced: the install decoder does not require EOF after its first JSON value, and the prerequisite start handler does not validate its declared empty request body.

### Testing and Verification

FAIL. The green matrix is substantial but does not cover the discovery state that breaks Desktop recovery or all mandatory managed-archive/deadline negative cases.

### Maintainability

PASS. New ownership, event and recovery responsibilities are in focused modules. Large existing Hermes files remain cohesive enough that line count alone is not a finding.

### Documentation

FAIL. Candidate SHA and CI/MSI evidence are stale, and one amendment retains superseded Node-stage behavior.

### Dependencies / Supply Chain

PASS. No new major framework was added. Exact source/Node/npm/license bytes are pinned, extracted from the final MSI and hashed; fail-closed negative inspection runs in CI.

### Operations / Diagnostics

FAIL. Structured logs and events are safe, but the reopened Desktop can still lose visibility and cancellation control for a live installation when discovery observes its partial tree.

## Findings

### Critical

None.

### High

#### HIGH-R2-001 — Active install recovery disappears when partial discovery is not `NOT_INSTALLED`

`App.tsx` successfully queries durable Operations, but the complete `HermesInstallPanel` remains nested under `notInstalled` at lines 331-349. During normal installation, creation of the managed target/marker and later partial tree gives Phase 2 discovery trusted installation evidence without a usable final launcher; the valid discovery outcome is commonly `BROKEN_EXECUTABLE`, not `NOT_INSTALLED`. On restart in that state, `followedInstallId` and `operationQuery` recover the row, but React does not render it, its structured log, or Cancel.

The R2 recovery test fixes the discovery mock to `NOT_INSTALLED` and never tests `BROKEN_EXECUTABLE`, `MALFORMED_VERSION` or another partial-state result with an active durable install.

Impact: the machine-mutating Operation continues while the control surface required to explain or cancel it disappears. This is the same required success/recovery flow as HIGH-R1-001, so it remains blocking.

Required closure:

- render a validated active `runtime.install` Operation independently of the current discovery card state;
- allow discovery state to govern only starting a new install, not following an existing one;
- test reload and cancellation with `BROKEN_EXECUTABLE`/partial discovery and with a later transition to `SUPPORTED`.

#### HIGH-R2-002 — Retry ownership still permits replacing changed content and an unpinned durable history row

`ownedPartialIdentity` (`target.go:65-76`) validates only that `.yorva-phase3-install` is a regular file containing the expected 40-hex commit. It does not establish the allowed tree identity or prove that the tree has remained unchanged since the recorded Operation. `replaceOwnedTree` then renames the complete marked directory aside and deletes it after replacement. The test `owned retry replaces instead of merging` deliberately adds `stale.exe` to a marked tree and expects it to be deleted (`target_test.go:180-198`). This is precisely the marked-plus-foreign-content case that R1 required to fail closed.

In addition, `RetryEligibleForPin` rejects a mismatch only when `latest.SourcePin` is non-empty (`runtime_install.go:357`). Migration `004_operation_source_pin.sql` assigns existing rows an empty pin, so an empty durable pin is accepted even though the closure contract requires the durable Operation, marker, target and expected pin to agree.

Impact: external/user content added beneath a once-marked directory can be treated as YORVA-owned and deleted, while durable history with no source identity can authorize the retry path. This violates the no-foreign-overwrite trust boundary.

Required closure:

- require exact non-empty durable `source_pin == expectedPin` for retry;
- bind the marker to the recorded Operation/target using a versioned ownership record;
- define and validate the allowed partial-tree identity or an equivalent tamper-evident manifest before replacement;
- reject marked targets containing unrecognized/changed files, including extra executables;
- add marker-plus-foreign-file/executable, empty durable pin and changed-tree tests.

### Medium

#### MEDIUM-R2-001 — Required managed archive and whole-deadline behavior remains untested

The candidate adds useful exact materialization and dependency tests, but the R1-required rejection behavior for entry-count, individual-member size and total uncompressed size is represented only by constant assertions (`node_contract_test.go:230-236`). The wrong-root test calls `extractPrefixedZip`, expects no error, and only confirms no output (`node_contract_test.go:45-64`). The 60-minute Operation ceiling is likewise checked only as a constant (`runtime_install_test.go:186-190`), not through a worker/deadline transition.

Required closure: add focused executable tests that force each archive bound and root-prefix failure, and an injected short whole-Operation deadline test proving process cancellation, cleanup and terminal `RUNTIME_INSTALL_TIMEOUT`.

#### MEDIUM-R2-002 — Phase 3 completion evidence does not describe the audited candidate

`PHASE-003-hermes-installation.md` §26 records `2f1e498...` and says exact-commit CI and MSI CI are PENDING. Three subsequent fixes changed MSI inspection/preparation and race-test synchronization; R2 actually audits `13d3739...`, CI `32091058398` and MSI `32091058406`. Amendment 003A1 §13 also retains superseded wording that official `node` and `node-deps` continue as skip-with-warning stages, while 003A3 says they are never spawned.

Required closure: update the candidate SHA and exact run evidence, and explicitly mark the older A1 Node-stage paragraph as superseded by 003A3. Do not mark the phase PASS/FROZEN before a later independent gate.

#### MEDIUM-R2-003 — Closed install/prerequisite request bodies are not fully enforced

OpenAPI declares a closed empty `RuntimeInstallRequest`. `startHermesInstall` decodes only one JSON value and never performs a second decode requiring EOF, so a body such as `{}` followed by another JSON value is accepted. `startHermesPrerequisites` validates the key but does not decode or reject request-body fields at all.

No user field currently reaches command construction, so this is not command injection. It is nevertheless a real typed-protocol mismatch that makes the daemon accept inputs the source-of-truth schema forbids.

Required closure: share one bounded empty-object decoder that rejects unknown fields, non-object values, trailing JSON and oversized bodies for both endpoints; add HTTP regression tests.

### Low

None.

### Info

#### INFO-R2-001 — Inherited Cargo audit warnings

`cargo audit` reports no known vulnerability and the same 17 allowed inherited unmaintained/unsound warnings from the Tauri dependency graph.

#### INFO-R2-002 — Local environmental limits and generated residue

Local race is CGO/toolchain-blocked and local Tauri no-bundle is locked by the running Owner Desktop process. Exact-commit CI covers both. The untracked root `.wxs` file is generated inspection residue and must be removed or otherwise excluded before any future freeze commit; it was not touched by this audit.

## Accepted Technical Debt

None. The unresolved High findings and mandatory Phase 3 recovery/security behavior are not eligible for conditional acceptance. No Medium finding has an Owner, trigger and explicit acceptance record.

## Required Fixes Before Next Phase

1. Make active Operation rendering independent of discovery's eligibility for starting a new install and add partial-discovery reload/cancel tests.
2. Require a non-empty exact durable source pin and reject marked targets whose content no longer matches the recorded YORVA-owned partial attempt.
3. Execute the missing Node/npm archive limit, wrong-root and whole-Operation deadline negative tests.
4. Enforce the closed empty-body contract consistently on both Phase 3 mutation endpoints.
5. Synchronize the Phase 3 candidate/CI/MSI completion evidence and superseded amendment wording.
6. Create a new immutable remediation commit, obtain exact-commit CI and MSI PASS, then request `AUDIT-003R3` from a fresh independent context.

## Gate Rationale

Green CI proves that the candidate passes its implemented checks. It does not prove the missing partial-discovery recovery path or safe ownership of a changed marked tree. Both are explicit Phase 3 mandatory behaviors, and the latter is a filesystem trust-boundary issue. With two unresolved High findings and three Medium findings, `PASS` and `PASS WITH CONDITIONS` are forbidden by `AUDIT_STANDARD.md` and the stricter Phase 3 audit rule.

## Next Step

```text
Phase 3 Implementation: NOT ACCEPTED
AUDIT-003R2 Gate:       FAIL
Phase 3 status:         AUDIT / BLOCKED BY R2 FINDINGS
Merge / freeze / tag:   NOT AUTHORIZED
Feature branch delete:  NOT AUTHORIZED
Phase 4 planning:       BLOCKED
Phase 4 implementation: BLOCKED
```

Preserve `AUDIT-003`, `AUDIT-003R1` and this R2 report as immutable audit history. Do not write the Phase 4 Spec until a later independent Phase 3 re-audit passes and the accepted Phase 3 baseline is frozen.

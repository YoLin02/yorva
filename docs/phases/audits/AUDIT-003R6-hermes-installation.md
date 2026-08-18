# YORVA Phase 3 Independent Re-Audit R6 — Hermes Installation

## Phase

Phase 3 — Hermes Installation

## Baseline / Commit

- Historical Phase 2 baseline: `phase-002-hermes-discovery-baseline-r1` → `5b89d22ed5e7ae3f4374a26f0fcda54bdabc6bf9`
- Historical R5 candidate: `d50bf48c8a2102e5560f299b007c420cc1b9e815` (`AUDIT-003R5` — FAIL)
- Branch inspected: `fix/phase3-audit-r5-remediation`
- Immutable R6 candidate: `b79ad3c5ba4024a12aec61d615096260d40ead4b`
- Remote branch tip at audit: `origin/fix/phase3-audit-r5-remediation` → the same SHA
- `main` / `origin/main` at audit: `9459eb9fea435704d6639be8457ee11dfad02093`

The candidate was fetched and locked before review. No Phase 3 tag exists. This audit did not merge, freeze, tag, delete a branch, change implementation or begin Phase 4.

Audit-start worktree was not clean: tracked `LICENSE` and `NPM-LICENSE` paths showed Windows metadata/EOL noise while their working-tree blob hashes exactly matched `HEAD`; root `YORVA_0.3.0_x64_en-US.wxs` was untracked. None is part of the R6 candidate and none was modified by this audit.

## Auditor

Fresh independent Phase 3 R6 review context, separate from implementation, reviewing governance, source, tests, the pinned upstream installer, local verification and exact-commit CI evidence.

## Date

2026-08-18 (Asia/Shanghai)

## Gate Decision

**FAIL**

## Executive Summary

R6 is a genuinely new remediation candidate and it materially improves the R5 design. Installation stages now execute against an Operation-private candidate; the prior live tree is retained in quarantine rather than deleted; markers use temporary-file replacement; and a durable promotion journal exists. The candidate does not weaken Phase 3 into an unreviewed Demo-only contract.

However, the ownership state machine is not yet internally consistent. The verified official `path` stage writes launchers into the live install tree *after* candidate promotion and marker commit, deterministically invalidating the ownership inventory. A failure in that stage or the subsequent post-check therefore leaves a YORVA-created tree that the promised safe retry rejects. The candidate-stage delta rule also proves only that a path is under the broad `venv/` or `.venv/` prefix; it cannot distinguish stage output from a foreign executable inserted into that prefix and will authenticate and later expose such bytes. Finally, a crash after moving the old live tree to quarantine but before persisting `OLD_QUARANTINED` leaves a `PREPARED` journal that recovery does not restore, so the canonical target remains absent despite both complete trees being available.

These are direct source-level execution chains in the current installation/retry/restart contract, not optional future hardening. Exact CI and MSI are green because the test suite does not exercise these boundaries. Phase 3 cannot be frozen and Phase 4 planning remains blocked by the governing Phase Spec.

## Verification Evidence

### Repository and candidate

- `git fetch --prune origin` — completed.
- `git branch --show-current` → `fix/phase3-audit-r5-remediation`.
- `git rev-parse HEAD` → `b79ad3c5ba4024a12aec61d615096260d40ead4b`.
- `git rev-parse origin/fix/phase3-audit-r5-remediation` → the same SHA.
- `git rev-parse origin/main` → `9459eb9fea435704d6639be8457ee11dfad02093`.
- `git diff --check d50bf48..b79ad3c` — PASS.
- `git tag --list 'phase-003*'` — no tag.

### Exact-commit GitHub Actions

- CI run [`32108290020`](https://github.com/YoLin02/yorva/actions/runs/32108290020) — push, branch `fix/phase3-audit-r5-remediation`, head `b79ad3c5ba4024a12aec61d615096260d40ead4b`, **SUCCESS**:
  - Web and API contract — success;
  - Go Node, including race, vet, govulncheck and build — success;
  - Windows Desktop native shell, lifecycle and Tauri no-bundle — success.
- MSI run [`32108290027`](https://github.com/YoLin02/yorva/actions/runs/32108290027) — same branch and head, **SUCCESS**:
  - Package and inspect MSI — success;
  - artifact `yorva-msi`, `119301821` bytes;
  - GitHub artifact digest `sha256:3402c9ae7dcbeb604b36d1734898cb704e451f9fb8f0cc04aff90bc5d0505de6`.

### Local Web / OpenAPI

- `pnpm api:lint` — PASS.
- `pnpm api:generate` and generated-schema diff — PASS, no generated diff.
- `pnpm typecheck` — PASS.
- `pnpm lint` — PASS.
- `pnpm test` — PASS, 16 files / 64 tests.
- `pnpm build` — PASS.
- `pnpm audit --audit-level low` — PASS, no known vulnerabilities.

### Local Go

- `go test ./...` — PASS.
- affected Hermes, application, SQLite and HTTP packages with `-count=20` — PASS (`hermes` completed 20 runs in about 240 seconds).
- `go vet ./...` — PASS.
- `go build ./cmd/yorvad` — PASS.
- `govulncheck ./...` — PASS, no vulnerabilities found.
- local `go test -race ./...` — environment-blocked (`-race requires cgo`); it is not reported as a local PASS. The exact-commit Go CI race suite is PASS.

### Local Rust / Windows / MSI

- `cargo fmt --all -- --check` — PASS.
- `cargo test --locked` — PASS, 10 tests.
- `cargo clippy --locked --all-targets --all-features -- -D warnings` — PASS.
- `cargo check --locked` — PASS.
- `cargo audit` — zero known vulnerabilities; 17 inherited allowed warnings.
- `scripts/windows-lifecycle-smoke.ps1` — PASS.
- `scripts/inspect-yorva-msi.tests.ps1` — PASS for all existing negative cases.
- Supplemental local MSI administrative extraction/hash inspection — PASS for Hermes source, Node, npm and all three LICENSE payloads. This older local MSI is supplemental only and is not represented as the exact CI artifact.
- local Tauri release `--no-bundle` — environment-blocked by Windows `PermissionDenied` on an Owner-held release output. The Owner process was not terminated. Exact-commit Windows CI no-bundle is PASS.

### R5 finding closure

| R5 finding | R6 result | Evidence |
| --- | --- | --- |
| HIGH-R5-001 stage-delta provenance | **NOT CLOSED** | Stages moved to a candidate, but `stageAllowsPath` accepts every file below `venv/` or `.venv/`. `applyAuthenticatedStageDelta` signs the complete resulting inventory and does not establish which process created an allowed-path file (`ownership_inventory.go:94-115`, `ownership_delta.go:25-37`). |
| HIGH-R5-002 validate-to-delete race | **CLOSED** | The old tree is moved to a random quarantine and is never automatically deleted; the late-change test proves bytes remain recoverable (`ownership_promote.go:131-191`, `ownership_promote_test.go:15-105`). |
| MEDIUM-R5-001 atomic marker write | **PARTIALLY CLOSED** | Temporary exclusive file, file sync, close and atomic Windows replacement are implemented. Directory-sync failure is still ignored and untested (`ownership_atomic.go:38-85`). |
| MEDIUM-R5-002 crash-recovery state | **NOT CLOSED** | A journal exists, but state is persisted after each rename; recovery does not repair the unavoidable rename-before-state-write windows (`ownership_promote.go:157-191`, `ownership_promote.go:217-249`). |

## Dimension Results

### Scope

PASS. No Phase 4 Instance/Profile, credentials/models, channels, Runtime process lifecycle, Skills/MCP, Cloud, generic package manager/plugin framework or Hermes fork was introduced. The candidate implemented the approved Phase 3 ownership remediation rather than silently removing retry acceptance criteria.

### Correctness

FAIL. A post-promotion `path` failure deterministically strands a tree with a stale ownership proof, and one crash boundary leaves the canonical installation absent without recovery.

### Architecture

PASS. New code remains Hermes-adapter-owned; Desktop, application, domain, persistence and Runtime dependency directions are unchanged. The focused modules are preferable to the previous monolithic target implementation.

### Security

FAIL. Broad allowed-path prefixes can authenticate foreign executable bytes as stage output. The verified official `path` stage may then copy those bytes into the public Hermes launcher directory.

### Data and Persistence

FAIL. Quarantine prevents deletion, but the journal state is not durably aligned with the two renames at all crash boundaries. The stored ownership manifest also no longer describes the live tree after `path`.

### Concurrency and Lifecycle

FAIL. Candidate isolation and quarantine improve race safety, but the rename-before-journal-update crash window and post-promotion mutation break required restart/retry behavior.

### Protocol and Compatibility

PASS. OpenAPI, typed requests, stable errors, SSE/idempotency and Phase 2 discovery/version contracts remain green and unchanged by R6.

### Testing and Verification

FAIL. The full implemented matrix is green, but existing tests inject a foreign file only outside allowed prefixes and synthesize already-persisted journal states. They do not test an allowed-prefix foreign executable, the official `path` stage followed by ownership validation/retry, or interruption after a rename and before the next state write.

### Maintainability

PASS WITH OBSERVATION. The ownership subsystem is now separated into cohesive files. `ownership_promote.go` is about 322 lines and remains focused. The issue is missing state-machine transitions and overly broad provenance rules, not file size or speculative abstraction.

### Documentation

FAIL. Phase 3 §10 and `SECURITY.md` say `path` is outside the directory manifest, but pinned `scripts/install.ps1` lines 2980-3000 create and populate `$InstallDir\bin`. `AMENDMENT-003A2-china-dependency-distribution.md` also still says `R1 REMEDIATION COMPLETE; AUDIT-003R2 PENDING`, while all other Phase 3 governance documents correctly show R5 remediation / R6 pending.

### Dependencies / Supply Chain

PASS. No new dependency or mutable source was added. Exact Hermes/Node/npm pins, archive hashes, licenses and MSI payload checks remain intact.

### Operations / Diagnostics

FAIL. Startup recovery can silently leave a canonical target absent, and `daemon.Run` logs a recovery error then continues serving rather than placing installation in an explicit fail-closed recovery state (`daemon.go:92-107`).

## Findings

### Critical

None.

### High

#### HIGH-R6-001 — The official `path` stage invalidates the committed ownership proof and blocks safe retry

`Apply` promotes the candidate and commits its ownership journal, then runs the official `path` stage against the live install directory (`host_installer.go:268-276`). The pinned, hash-verified `scripts/install.ps1` implements `Set-PathVariable` by creating `$InstallDir\bin` and copying `venv\Scripts\hermes.exe` and `hermes-acp.exe` into it (pinned script lines 2980-3000). Those files are therefore inside the directory inventory, despite Phase 3 §10 claiming `path` is outside it.

There is no marker refresh or second journal commit after `path`. The inventory recorded before promotion immediately becomes stale. If `path` copies one launcher and then fails while updating user environment, or if the following public-launcher/PATH post-check fails, the Operation ends `FAILED` with a YORVA-created tree whose marker no longer matches. The next request supplies that failed Operation as the prior proof, but `validateInstallTarget` → `ownedPartialIdentity` recomputes the inventory and returns `RUNTIME_INSTALL_TARGET_OCCUPIED` (`target.go:85-122`). The promised safe retry is unavailable.

The successful retry test misses the defect because it checks the marker MAC/identity only; it does not call `requireCurrentOwnedTree` after the fake `path` stage adds `bin/hermes.exe` (`host_installer_retry_test.go:53-95`, fake stage at lines 286-293).

Impact: a normal Phase 3 failure boundary can leave the user unable to retry or complete installation without manual filesystem work. This materially breaks the core installation lifecycle.

Required closure: create the launcher `bin` contents inside the authenticated candidate before promotion, or define and atomically authenticate the exact `path` tree delta before reporting success. Treat user PATH/HERMES_HOME registry changes as separate idempotent postconditions. Add a real regression proving marker validity after `path`, and failure-after-copy → next-Operation retry.

#### HIGH-R6-002 — Allowed-directory prefixes still authenticate foreign executable changes

`stageAllowsPath` allows every path below `venv/` during `venv` and `dependencies`, and every path below `.venv/` during `dependencies` (`ownership_inventory.go:94-115`). After a stage, `applyAuthenticatedStageDelta` compares only file paths/hashes to the pre-stage snapshot and then signs the entire current candidate inventory (`ownership_delta.go:25-37`). It has no owned-output list, child-process file identity or more specific expected delta.

Deterministic reproduction with the existing injection hook:

1. Start a candidate with a valid marker.
2. Run `dependencies`.
3. In `HostInstaller.afterStage`, create or replace `candidate/venv/Scripts/hermes.exe`.
4. The delta is accepted because the path starts with `venv/`.
5. The candidate marker authenticates the foreign bytes and promotion accepts them.
6. The official `path` stage copies that executable to the public `$InstallDir\bin\hermes.exe`.

The new test injects only root-level `foreign.txt`, which is outside the allowed prefix and therefore does not test the trust boundary (`ownership_delta_test.go:91-112`). `TestStageDeltaRejectsForeignInsertModifyDelete` likewise proves rejection only for root paths, then explicitly proves arbitrary `venv/pyvenv.cfg` is trusted based on location alone (`ownership_delta_test.go:14-88`).

Impact: R5's trust violation remains for the exact directories containing executable Python/launcher content. Candidate path unpredictability narrows the race but is not provenance, and the Phase Spec explicitly requires foreign changes to fail closed.

Required closure: derive a narrowly reviewed stage-output contract that cannot accept arbitrary executables merely by directory prefix, or move generation/copy of security-sensitive launchers to a YORVA-owned verified step. At minimum inject foreign files and replacements inside every allowed prefix and prove they are rejected or cryptographically/structurally verified before authentication.

#### HIGH-R6-003 — Crash between the first rename and journal state write is not recoverable

For retry, promotion writes `PREPARED`, validates the old live tree, renames live → quarantine, and only then writes `OLD_QUARANTINED` (`ownership_promote.go:150-168`). A process or host interruption between lines 161 and 165 leaves:

```text
journal = PREPARED
canonical target = absent
old proven tree = quarantine
new proven tree = candidate
```

On restart, the `PREPARED` recovery branch only removes an owned candidate when the old live target is still present and unchanged; when the target is absent it neither restores the journal-named quarantine nor advances promotion (`ownership_promote.go:217-228`). It returns success, so daemon startup continues, subsequently interrupting the Operation. The canonical Hermes installation remains absent even though both recoverable trees exist.

The synthetic recovery tests begin after journal states have already been persisted. Hooks also run after state writes, so they cannot inject this unavoidable rename-before-write interruption (`ownership_promote_test.go:147-292`). The symmetric new-tree rename-before-`NEW_PROMOTED` write window similarly leaves a stale journal state.

Impact: the explicitly required daemon/host restart recovery can make an existing Hermes installation disappear from its canonical path and require manual recovery. Quarantine preserves bytes, so this is not classified Critical, but it is a blocking current-phase lifecycle failure.

Required closure: make recovery infer and validate the filesystem state for each prior journal state, including `PREPARED + target missing + proven quarantine` and `OLD_QUARANTINED + proven new target`, then deterministically restore or complete without deleting uncertain data. Add crash injection immediately after each rename but before the following journal write.

### Medium

#### MEDIUM-R6-001 — Atomic write reports success when directory durability fails

`writeAtomicRegularFile` correctly writes, file-syncs, closes and replaces an exclusive temporary file, but ignores `SyncDir` errors (`ownership_atomic.go:79-84`). The marker test injects create/write/file-sync/close/replace failures only; no directory-sync failure is asserted (`ownership_marker_test.go:24-69`). The same helper persists promotion journals.

Impact: the code can report a durable marker/journal transition even though its final directory durability operation failed, weakening the crash-state evidence relied on by promotion recovery.

Required closure: handle the supported-platform directory durability result explicitly, document the Windows `MOVEFILE_WRITE_THROUGH` guarantee used, and add a failure test with an honest old-or-new complete-record outcome.

#### MEDIUM-R6-002 — Phase 3 amendment status and path-manifest documentation are inconsistent

`AMENDMENT-003A2-china-dependency-distribution.md:6-7` still records R1 remediation / R2 pending, unlike the Phase Spec, Roadmap, A1 and A3. More materially, Phase 3 §10 and `SECURITY.md` describe `path` as outside the install-tree manifest while the pinned official implementation writes `bin` inside it.

Impact: implementation, security contract and governance evidence do not describe the same recovery boundary. The path mismatch directly contributed to HIGH-R6-001.

Required closure: correct A2 audit status and describe the tree-writing launcher-copy and environment-variable postconditions separately, matching the final implementation.

### Low

None.

### Info

#### INFO-R6-001 — Local/environment observations

Cargo retains 17 inherited allowed warnings and no known vulnerability. Local Go race remains CGo-toolchain-blocked, and local Tauri no-bundle is locked by an Owner-held release output; exact-commit CI covers both. The root `.wxs` residue and EOL-noise resource files must not be included accidentally in a future audit/freeze commit.

## Accepted Technical Debt

None. The High findings affect the current install/retry/restart trust and correctness model. Phase 3 §22 also explicitly makes any Critical, High or Medium finding gate-blocking.

## Required Fixes Before Next Phase

1. Close the install tree before promotion: ensure all tree-mutating output, including `bin` launcher creation, is present and authenticated before the candidate is committed. Keep user environment changes separate.
2. Replace broad allowed-prefix trust with a verifiable contract for security-sensitive stage output; prove that injected/replaced executables within `venv/` and `.venv/` cannot be authenticated.
3. Recover the actual filesystem configuration at every rename-before-journal-write crash boundary. Preserve quarantine and never delete uncertain data.
4. Make atomic marker/journal durability results explicit and tested.
5. Add deterministic regression tests for path-copy failure/retry, allowed-prefix foreign mutation, and crashes immediately between each rename and state write.
6. Synchronize A2, Phase 3 and Security documentation.
7. Produce a new immutable candidate with exact-commit CI/MSI PASS and request a fresh independent `AUDIT-003R7`.

## Gate Rationale

The candidate closes the R5 deletion race and substantially improves isolation, but PASS requires zero blocking High findings and evidence for mandatory retry/restart behavior. The official path stage deterministically breaks the committed proof, allowed-prefix provenance remains unresolved for executable content, and a documented crash boundary is not recovered. These conditions satisfy the `AUDIT_STANDARD.md` FAIL rules for a required workflow failure, violated security trust model and insufficient recovery evidence. Owner schedule preference and green CI do not authorize lowering correctness or data-safety gates.

## Next Step

```text
Phase 3 Implementation: NOT ACCEPTED
AUDIT-003R6 Gate:       FAIL
Phase 3 status:         AUDIT / BLOCKED BY R6 FINDINGS
Merge / freeze / tag:   NOT AUTHORIZED
Feature branch delete:  NOT AUTHORIZED
Phase 4 planning:       BLOCKED
Phase 4 implementation: BLOCKED
```

Preserve `AUDIT-003` and `AUDIT-003R1` through `AUDIT-003R6` as immutable audit history.

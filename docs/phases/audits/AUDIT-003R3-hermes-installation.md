# YORVA Phase 3 Independent Re-Audit R3 — Hermes Installation

## Phase

Phase 3 — Hermes Installation

## Baseline / Commit

- Historical Phase 2 baseline: `phase-002-hermes-discovery-baseline-r1` → `5b89d22ed5e7ae3f4374a26f0fcda54bdabc6bf9`
- Historical R2 candidate: `13d3739e0f6379aee1253abecdd4c44c59d1c31b` (`AUDIT-003R2` — FAIL)
- R3 remediation branch: `fix/phase3-audit-r2-remediation`
- Immutable R3 candidate: `d214b51a839b165a62261a4adc4be7c31b486936`
- Remote branch tip at audit: `origin/fix/phase3-audit-r2-remediation` → the same SHA
- `main` / `origin/main` at audit: `9459eb9fea435704d6639be8457ee11dfad02093`

The audit did not merge, freeze, tag, delete a branch, or begin Phase 4.

## Auditor

Fresh independent Phase 3 re-audit context (R3), reviewing the immutable repository candidate and exact-commit evidence rather than the implementation completion statement.

## Date

2026-08-18 (Asia/Shanghai)

## Gate Decision

**FAIL**

## Executive Summary

The R3 candidate closes the visible R2 Desktop recovery defect, rejects empty durable pins and tampered/foreign partial trees, exercises the closed request-body contract, and adds substantial archive/deadline tests. Exact-commit CI and MSI packaging are green.

The gate cannot pass, including under the Owner's authorization to conditionally accept non-blocking defects. The new ownership-record lifecycle makes every legitimate retry fail after preflight: `Apply` replaces the previous Operation's record with the new Operation identity before repository materialization, while `replaceOwnedTree` then validates the same target against the previous Operation. The two identities cannot match. Later official stages also mutate the recorded tree without refreshing its inventory digest. Safe retry is an explicit Phase 3 lifecycle requirement and a mandatory audit criterion, so this is a blocking High correctness defect rather than deferrable technical debt.

## Verification Evidence

### Repository and candidate

- `git branch --show-current` → `fix/phase3-audit-r2-remediation`
- `git rev-parse HEAD` → `d214b51a839b165a62261a4adc4be7c31b486936`
- `git rev-parse origin/main` → `9459eb9fea435704d6639be8457ee11dfad02093`
- `git ls-remote origin refs/heads/fix/phase3-audit-r2-remediation` → the exact candidate SHA
- `git diff --check 13d3739..d214b51` → PASS
- The two resource LICENSE files show working-tree metadata/line-ending noise, but their `git hash-object` values exactly equal the candidate blobs. The pre-existing untracked root `.wxs` is not part of the candidate. This audit did not modify them.

### Exact-commit GitHub Actions

- CI run [`32097619823`](https://github.com/YoLin02/yorva/actions/runs/32097619823) — `push`, candidate branch, head `d214b51a839b165a62261a4adc4be7c31b486936`, **SUCCESS**:
  - Web and API contract — success;
  - Go Node, including race, vet, govulncheck and build — success;
  - Windows Desktop native shell, lifecycle and Tauri checks — success.
- MSI run [`32097619817`](https://github.com/YoLin02/yorva/actions/runs/32097619817) — same branch and head, **SUCCESS**:
  - Package and inspect MSI — success;
  - artifact `yorva-msi`, `119269240` bytes;
  - GitHub artifact digest `sha256:814edc6516115eb12b5b7dd2a4e0b7523ebd9884410b12b1965f52b1a0ce4ffe`.

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
- affected application, Hermes adapter, SQLite and HTTP packages with `-count=20` — PASS.
- `go vet ./...` — PASS.
- `go build ./cmd/yorvad` — PASS.
- `govulncheck ./...` — PASS, no vulnerabilities found.
- local `go test -race ./...` — blocked because CGO is disabled / no usable C toolchain (`-race requires cgo`). Exact-commit CI race is PASS; the local command is not reported as PASS.

### Local Rust / Windows / MSI

- `cargo fmt --all -- --check` — PASS.
- `cargo test --locked` — PASS, 10 tests.
- `cargo clippy --locked --all-targets --all-features -- -D warnings` — PASS.
- `cargo check --locked` — PASS.
- `cargo audit` — zero vulnerabilities; 17 inherited allowed warnings.
- `scripts/windows-lifecycle-smoke.ps1` — PASS.
- `scripts/inspect-yorva-msi.tests.ps1` — PASS for the full negative inspection matrix.

## R2 Finding Closure

| R2 finding | R3 result | Evidence |
| --- | --- | --- |
| HIGH-R2-001 active install hidden under partial discovery | **CLOSED** | `App.tsx:313-357` separates `canStart` from following an Operation. Recovery tests cover `BROKEN_EXECUTABLE`, `MALFORMED_VERSION`, cancellation and transition to `SUPPORTED`. |
| HIGH-R2-002 retry ownership and empty pin | **NOT CLOSED** | Empty pin/nonce and changed/foreign trees now fail closed, but `host_installer.go:172` writes the current identity before `materializeRepository`, which passes the prior identity at `host_installer.go:306`; `target.go:304-315` then requires that prior identity to authenticate the newly written record. Legitimate retries therefore fail. |
| MEDIUM-R2-001 archive bounds and whole deadline | **CLOSED WITH INFO** | Executable ZIP/tar bound and cleanup tests plus an injected short whole-Operation deadline test pass. The production 60-minute constant is unchanged. |
| MEDIUM-R2-002 stale candidate/docs | **CLOSED** | Historical R2 SHA/runs are accurate, the R3 candidate is intentionally locked by this report, superseded Node-stage wording is corrected, and historical FAIL reports remain intact. |
| MEDIUM-R2-003 closed request bodies | **CLOSED** | `decodeClosedEmptyObject` is bounded and shared by both endpoints; tests cover `{}`, unknown/non-object/null, multiple/trailing JSON, garbage, oversize, missing body and invalid idempotency key. |

## Dimension Results

### Scope

PASS. The candidate stays within Phase 3 remediation, tests, migrations and governing documentation. No Phase 4 feature is present.

### Correctness

FAIL. The required retry path deterministically fails after safe preflight because the ownership record changes identity before the repository replacement validates it.

### Architecture

PASS. Desktop, Node application, persistence and Hermes-adapter ownership remain in the required dependency direction. No speculative plugin or package-manager framework was added.

### Security

PASS. The R2 foreign-content deletion issue is closed: copied/malformed records, empty pins/nonces, changed inventory, extra files, executables and reparse points fail closed. No uncertain tree is deleted in the reviewed code path.

### Data and Persistence

PASS. Migration 005 is deterministic, empty values cannot authorize retry, and the nonce is not exposed through the Operation HTTP DTO.

### Concurrency and Lifecycle

FAIL. Deadline/cancellation and process containment pass, but failed/cancelled installation retry—an explicit lifecycle requirement—is nonfunctional.

### Protocol and Compatibility

PASS. Both start endpoints enforce the OpenAPI closed-empty-object contract and stable safe errors.

### Testing and Verification

FAIL. The matrix is broadly green, but no end-to-end adapter test exercises `ValidateTarget(previous) → Apply(new) → repository replacement`; focused helpers therefore missed the deterministic retry identity mismatch.

### Maintainability

PASS. New responsibilities are reasonably cohesive. No changed production file constitutes an unrelated dumping ground or speculative abstraction.

### Documentation

PASS. The historical evidence and amendment supersession are synchronized without falsely declaring the phase accepted.

### Dependencies / Supply Chain

PASS. No major dependency was added. Exact source, Node/npm and MSI verification remain pinned and fail closed.

### Operations / Diagnostics

FAIL. The user can observe/cancel active work after Desktop restart, but the advertised retry action cannot complete for a legitimate YORVA-owned partial attempt.

## Findings

### Critical

None.

### High

#### HIGH-R3-001 — Ownership-record rotation makes every legitimate installation retry fail

The application correctly retrieves the previous failed/cancelled Operation, calls `ValidateTarget` with it, and then supplies the new Operation identity (`runtime_install_run.go:117-124`). The adapter subsequently writes a record for the **new** Operation before probes/stages (`host_installer.go:172`). At repository materialization it calls `replaceOwnedTree` with the **previous** Operation (`host_installer.go:306`). Because the target now contains the new record, `ownedPartialIdentity` compares that record to the previous ID/nonce at `target.go:101-119` and rejects it at `target.go:304-315`.

This is not a theoretical attacker race: it is the normal retry sequence. A prior owned tree that passed preflight is made unverifiable by YORVA itself before replacement. In addition, after successful repository materialization the `venv`, `dependencies`, `path`, templates and marker stages can mutate the install tree (`host_installer.go:187-214`) without refreshing the recorded inventory, so a later failed attempt also leaves a manifest that no longer represents its own YORVA-created partial tree.

Impact: a failed, cancelled or daemon-interrupted Phase 3 installation cannot use the promised safe Retry workflow. The user is left with a partial fixed target that a new install rejects as occupied. This affects the required system correctness/lifecycle contract and is not eligible for `PASS WITH CONDITIONS`.

Required closure:

1. Preserve and validate the previous proof until replacement, or define one explicit authenticated ownership handoff that cannot create an identity mismatch.
2. Refresh the authenticated partial-tree inventory after every YORVA-owned stage that mutates the target, including on controlled failure/cancellation boundaries, without authenticating unknown external changes.
3. Add an adapter-level regression test covering a real sequence: first Operation creates a partial tree and fails after a mutating stage; second Operation validates that exact unchanged tree, materializes the repository, reruns approved stages and proceeds.
4. Retain fail-closed tests for copied records, extra/changed/missing files, wrong pin/nonce/target, reparse points and uncertain data.

### Medium

None.

### Low

None.

### Info

#### INFO-R3-001 — Aggregate archive-limit branch coverage could be more exact

The archive code enforces entry, member and aggregate uncompressed limits, and current tests execute rejection paths. The ZIP test named `uncompressed total` uses one member larger than both the member and total limits, so it does not independently prove the aggregate-only branch. This is a bounded test-quality improvement and does not change the gate because the implementation is direct and the blocking retry defect already requires another candidate.

#### INFO-R3-002 — Inherited/local environmental observations

Cargo retains 17 previously allowed warnings with no known vulnerability. Local race remains CGo-toolchain-blocked while exact-commit CI race passes. The root `.wxs` is generated residue and must not enter a freeze commit.

## Accepted Technical Debt

None. HIGH-R3-001 is a mandatory Phase 3 retry/lifecycle failure and cannot be accepted as deferred work. INFO-R3-001 may be improved in the same narrow remediation without changing product scope.

## Required Fixes Before Next Phase

1. Correct the ownership-record handoff/refresh lifecycle so an unchanged YORVA-owned failed attempt can actually retry while any foreign modification still fails closed.
2. Add an adapter-level two-Operation retry regression test that exercises the complete ordering rather than isolated ownership helpers.
3. Create a new immutable candidate, obtain exact-commit CI/MSI PASS, and request a fresh independent `AUDIT-003R4`.

## Gate Rationale

Green CI proves the implemented tests pass; it does not override a deterministic defect outside their scenario coverage. The Owner authorized conditional acceptance only for non-blocking defects and did not authorize downgrading a real High or mandatory acceptance failure. Safe retry is expressly required by the Phase 3 Spec and is part of the audit's target ownership/lifecycle criteria. `PASS` and `PASS WITH CONDITIONS` are therefore not allowed.

## Next Step

```text
Phase 3 Implementation: NOT ACCEPTED
AUDIT-003R3 Gate:       FAIL
Phase 3 status:         AUDIT / BLOCKED BY R3 FINDING
Merge / freeze / tag:   NOT AUTHORIZED
Phase 4 planning:       BLOCKED
Phase 4 implementation: BLOCKED
```

Preserve `AUDIT-003`, `AUDIT-003R1`, `AUDIT-003R2` and this report as immutable audit history.

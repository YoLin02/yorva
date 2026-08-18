# YORVA Phase 3 Independent Re-Audit R4 — Hermes Installation

## Phase

Phase 3 — Hermes Installation

## Baseline / Commit

- Historical Phase 2 baseline: `phase-002-hermes-discovery-baseline-r1` → `5b89d22ed5e7ae3f4374a26f0fcda54bdabc6bf9`
- Historical R3 candidate: `d214b51a839b165a62261a4adc4be7c31b486936` (`AUDIT-003R3` — FAIL)
- R4 remediation branch: `fix/phase3-audit-r3-remediation`
- Immutable R4 candidate: `d50bf48c8a2102e5560f299b007c420cc1b9e815`
- Remote branch tip at audit: `origin/fix/phase3-audit-r3-remediation` → the same SHA
- `main` / `origin/main` at audit: `9459eb9fea435704d6639be8457ee11dfad02093`

The audit did not merge, freeze, tag, delete a branch, or begin Phase 4.

## Auditor

Fresh independent Phase 3 re-audit context (R4), reviewing the immutable candidate, its exact diff and exact-commit evidence independently from the implementation statement.

## Date

2026-08-18 (Asia/Shanghai)

## Gate Decision

**FAIL**

## Executive Summary

The R4 candidate corrects the R3 ordering defect: it preserves the previous Operation proof until repository replacement, performs the new-identity handoff inside the replacement, refreshes ownership after later mutating stages, exercises a real two-Operation retry, and adds the previously missing aggregate-only ZIP limit test. Exact-commit CI and MSI packaging are green, and the full local matrix passes apart from accurately recorded environment blockers.

The gate still cannot pass. The new refresh path verifies the MAC of the **old record**, but it never verifies that the current tree still matches the old record before computing and signing a replacement manifest. Any unrelated file inserted or changed while a long official stage runs is therefore silently authenticated as YORVA-owned. A later retry accepts that manifest and may remove the unrelated content during atomic repository replacement. This is a deterministic trust-boundary gap in the mandatory safe-retry path, not a cosmetic or deferrable defect. The Owner's permission to accept non-blocking issues does not authorize downgrading security, user-data or lifecycle correctness.

## Verification Evidence

### Repository and immutable candidate

- `git branch --show-current` → `fix/phase3-audit-r3-remediation`.
- `git rev-parse HEAD` → `d50bf48c8a2102e5560f299b007c420cc1b9e815`.
- `git rev-parse origin/fix/phase3-audit-r3-remediation` and `git ls-remote` → the same SHA.
- `git rev-parse origin/main` → `9459eb9fea435704d6639be8457ee11dfad02093`.
- `git diff --check d214b51..d50bf48` — PASS.
- No `phase-003*` tag exists.
- The two resource LICENSE paths retain pre-existing Windows working-tree metadata noise, but their `git hash-object` values exactly match candidate blobs. The pre-existing untracked root `YORVA_0.3.0_x64_en-US.wxs` is outside the candidate and was not touched.

### Exact-commit GitHub Actions

- CI run [`32100135303`](https://github.com/YoLin02/yorva/actions/runs/32100135303) — `push`, branch `fix/phase3-audit-r3-remediation`, head `d50bf48c8a2102e5560f299b007c420cc1b9e815`, **SUCCESS**:
  - Web and API contract — success;
  - Go Node, including race, vet, govulncheck and build — success;
  - Windows Desktop native shell, lifecycle and Tauri no-bundle — success.
- MSI run [`32100135323`](https://github.com/YoLin02/yorva/actions/runs/32100135323) — same branch and head, **SUCCESS**:
  - Package and inspect MSI — success;
  - artifact `yorva-msi`, approximately 114 MiB;
  - GitHub artifact digest `sha256:b0034433b881042a13418bbbd217b348a42f1a609230298323d3b0d323009830`.

### Local Web / OpenAPI

- `pnpm api:lint` — PASS.
- `pnpm api:generate` plus generated-schema diff — PASS, no generated diff.
- `pnpm typecheck` — PASS.
- `pnpm lint` — PASS.
- `pnpm test` — PASS, 16 files / 64 tests.
- `pnpm build` — PASS.
- `pnpm audit --audit-level low` — PASS, no known vulnerabilities.

### Local Go

- `go test ./...` — PASS.
- affected application, Hermes adapter, SQLite and HTTP packages with `-count=20` — PASS.
- focused ownership-handoff, refresh and aggregate-limit tests with `-count=20` — PASS.
- `go vet ./...` — PASS.
- `go build ./cmd/yorvad` — PASS.
- `govulncheck ./...` — PASS, no vulnerabilities found.
- local `go test -race ./...` — environment-blocked because CGO is disabled / no usable C toolchain (`-race requires cgo`). Exact-commit CI race is PASS; the local command is not reported as PASS.

### Local Rust / Windows / MSI

- `cargo fmt --all -- --check` — PASS.
- `cargo test --locked` — PASS, 10 tests.
- `cargo clippy --locked --all-targets --all-features -- -D warnings` — PASS.
- `cargo check --locked` — PASS.
- `cargo audit` — zero vulnerabilities; 17 inherited allowed warnings.
- `scripts/windows-lifecycle-smoke.ps1` — PASS.
- `scripts/inspect-yorva-msi.tests.ps1` — PASS for all negative inspector cases.
- local Tauri release `--no-bundle` — environment-blocked by Windows `Access denied` in the Tauri build step while the Owner application/build artifact is in use. Exact-commit Windows CI no-bundle is PASS; the local build is not reported as PASS.

## R3 Finding Closure

| R3 item | R4 result | Evidence |
| --- | --- | --- |
| HIGH-R3-001 previous-proof/new-identity ordering | **CLOSED** | `host_installer.go:171-213` no longer writes the new record before repository; `target.go:298-378` validates the previous proof immediately before rename and places the new record in the replacement tree. The two-Operation adapter test completes a legitimate retry. |
| HIGH-R3-001 post-stage authenticated refresh | **NOT CLOSED** | `refreshOwnedInventory` validates only record fields and its MAC (`target.go:217-234`) before `writeOwnershipRecord` signs the complete current inventory. It never compares the pre-refresh tree with the old authenticated manifest or proves which delta the owned stage produced. See HIGH-R4-001. |
| HIGH-R3-001 two-Operation retry test | **CLOSED** | `TestOwnershipHandoffRetrySequence/operation B retries unchanged failed tree` runs Operation A through a mutating stage and failure, validates unchanged A state, then runs Operation B through handoff and later approved stages. |
| INFO-R3-001 aggregate-only archive limit | **CLOSED** | `TestExtractPrefixedZipRejectsAggregateOnlyUncompressedLimit` uses nine members, each at the per-member limit while only their sum exceeds the aggregate limit; it executes the aggregate rejection and cleanup path. |

## Dimension Results

### Scope

PASS. The diff contains only R3 remediation code/tests, immutable R3 audit history, governance evidence and the MSI branch trigger. No Phase 4 implementation or speculative framework is present.

### Correctness

FAIL. Ordinary two-Operation retry now works, but an unrelated concurrent tree change can be accepted as an owned stage delta and later removed.

### Architecture

PASS. Ownership remains inside the Hermes adapter; Desktop, application, persistence and adapter dependency direction is unchanged.

### Security

FAIL. The ownership refresh crosses a filesystem trust boundary by signing data whose provenance it has not established.

### Data and Persistence

FAIL. A later accepted retry may delete content that the current Operation did not create. The ownership record update is also not crash-safe.

### Concurrency and Lifecycle

FAIL. The check-before-stage / sign-after-stage window admits unrelated filesystem changes during long-running dependency stages, and a failed marker rewrite can make interruption retry permanently unavailable.

### Protocol and Compatibility

PASS. The R4 diff does not weaken the typed HTTP/OpenAPI, idempotency, SSE, discovery or compatibility contracts previously closed by R3.

### Testing and Verification

FAIL. The green suite verifies external mutation before a retry and isolated refresh identity mismatch, but it does not test foreign insertion/change between the pre-stage ownership check and post-stage refresh. The current refresh unit test instead demonstrates that an arbitrary content change is accepted and re-signed.

### Maintainability

PASS. The new adapter test is cohesive and the production changes remain narrowly owned. No unrelated giant module or new abstraction framework was introduced.

### Documentation

FAIL. Phase 3 §10 says a refresh never signs a tree whose current record no longer authenticates and that uncertain trees are never deleted; the implementation does not enforce that rule across a mutating stage.

### Dependencies / Supply Chain

PASS. No dependency was added; pinned source, Node/npm and MSI verification behavior remains unchanged and green.

### Operations / Diagnostics

FAIL. Diagnostics and cleanup regressions were not found, but the safe Retry operation can convert unrelated data into owned/deletable state.

## Findings

### Critical

None.

### High

#### HIGH-R4-001 — Post-stage refresh authenticates unrelated filesystem changes as YORVA-owned

For every mutating stage, `runOwnedStage` first calls `requireCurrentOwnedTree`, executes the external PowerShell stage, then calls `refreshOwnedInventory` (`host_installer.go:357-370`). The first check proves only the instant before the stage starts. During the stage, `refreshOwnedInventory` reads the old record and checks schema, Operation identity, pin, canonical target and the old MAC (`target.go:200-233`), but it never recomputes the current inventory and compares it with `record.Manifest` before deciding what is allowed to change. It calls `writeOwnershipRecord`, which hashes **all** files currently present and signs that new digest (`target.go:149-185,234`).

The behavior is directly visible in `TestRefreshOwnedInventoryRejectsForeignAndCopiedRecords`: the test changes `owned.txt` without passing any expected delta or stage-produced manifest, calls refresh, and expects the arbitrary new bytes to become a valid owned tree (`host_installer_retry_test.go:178-200`). The code has no information capable of distinguishing that test mutation from a user, another Hermes process, antivirus recovery, or another same-user program modifying/adding a file during a 30-minute dependency stage.

Reproduction sequence:

1. Start with a valid current-Operation ownership record and matching tree.
2. Allow `requireCurrentOwnedTree` to pass.
3. While a mutating stage runs, add `foreign.txt` or modify an existing file below the install root.
4. Let the stage return. Refresh signs the entire changed tree.
5. Fail/cancel later, then retry. Previous-proof validation now accepts the foreign file because it matches the newly signed digest.
6. Repository replacement renames that tree to the backup and deletes the backup after handoff, removing the unrelated content.

This violates Phase 3 §10 and the R3 remediation requirement to authenticate only YORVA-produced changes. It is a user-data/trust-boundary defect in mandatory recovery, so it cannot be accepted through `PASS WITH CONDITIONS`.

Required closure: establish a stage-owned delta rather than trusting the whole post-stage tree. Acceptable narrow designs include executing target-mutating work in an Operation-owned staging tree before atomic promotion, or defining and verifying stage-specific created/changed path inventories. In all cases, any path/change not proven to result from the owned stage must leave the old proof intact and fail closed without deletion. Add a deterministic test that injects a foreign file/change between the pre-stage check and refresh and proves it is not authenticated or later deleted.

### Medium

#### MEDIUM-R4-001 — Ownership-record refresh is not atomically durable

`writeOwnershipRecord` rewrites the live marker with `os.WriteFile` (`target.go:149-185`). Updating an existing record therefore opens/truncates the only authenticated proof before the replacement bytes are durably promoted. A write error, daemon termination or host interruption in that window can leave an empty/partial marker. The next retry correctly fails closed, but the user is permanently unable to use the promised automatic retry against an otherwise YORVA-owned tree.

Required closure: serialize the complete authenticated record to a same-directory private temporary regular file, flush/close as appropriate, re-check containment/reparse invariants, and atomically rename/replace it over the marker. Preserve the last valid proof when preparation fails. Add injected write/promotion failure and interruption-oriented tests proving either the old or new valid record remains, never a partial record.

### Low

None.

### Info

#### INFO-R4-001 — Inherited and local environment observations

Cargo retains 17 previously allowed warnings with no known vulnerability. Local Go race remains CGo-toolchain-blocked, and local Tauri no-bundle is locked by the running Owner environment; exact-commit CI covers both successfully. The root `.wxs` inspection residue remains untracked and must not enter a future freeze commit.

## Accepted Technical Debt

None. HIGH-R4-001 affects security, data ownership and the mandatory safe-retry workflow. MEDIUM-R4-001 has no explicit Owner, trigger and acceptance record and should be closed in the same narrow ownership-record remediation.

## Required Fixes Before Next Phase

1. Make post-stage inventory refresh prove a YORVA-owned delta; never sign unrelated concurrent filesystem changes.
2. Make ownership-record replacement atomically durable while retaining the previous valid proof on preparation/promotion failure.
3. Add foreign-change-during-stage and marker-update failure regression tests, while retaining the successful two-Operation retry and all fail-closed tampering tests.
4. Create a new immutable candidate, obtain exact-commit CI and MSI PASS, then request a fresh independent `AUDIT-003R5`.

## Gate Rationale

The candidate fixes the deterministic R3 handoff-order defect and has strong green verification. Green CI cannot override the remaining code-level trust violation: the implementation signs the entire post-stage tree without proving which changes belong to the owned process, enabling a later retry to delete unrelated data. `AUDIT_STANDARD.md` requires FAIL for an unresolved blocking High, an unsafe data behavior or a violated security trust model. Conditional acceptance is therefore not authorized.

## Next Step

```text
Phase 3 Implementation: NOT ACCEPTED
AUDIT-003R4 Gate:       FAIL
Phase 3 status:         AUDIT / BLOCKED BY R4 FINDINGS
Merge / freeze / tag:   NOT AUTHORIZED
Feature branch delete:  NOT AUTHORIZED
Phase 4 planning:       BLOCKED
Phase 4 implementation: BLOCKED
```

Preserve `AUDIT-003`, `AUDIT-003R1`, `AUDIT-003R2`, `AUDIT-003R3` and this report as immutable audit history.

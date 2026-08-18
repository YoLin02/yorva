# YORVA Phase 3 Independent Re-Audit R5 — Hermes Installation

## Phase

Phase 3 — Hermes Installation

## Baseline / Commit

- Historical Phase 2 baseline: `phase-002-hermes-discovery-baseline-r1` → `5b89d22ed5e7ae3f4374a26f0fcda54bdabc6bf9`
- Historical R4 candidate: `d50bf48c8a2102e5560f299b007c420cc1b9e815` (`AUDIT-003R4` — FAIL)
- Branch inspected: `fix/phase3-audit-r3-remediation`
- Immutable R5 audit object: `d50bf48c8a2102e5560f299b007c420cc1b9e815`
- Remote branch tip at audit: `origin/fix/phase3-audit-r3-remediation` → the same SHA
- `main` / `origin/main` at audit: `9459eb9fea435704d6639be8457ee11dfad02093`

No R4 remediation commit or newer remote branch existed after `git fetch --prune origin`. R5 therefore audits the same immutable implementation object as R4 and expands the ownership/retry state-machine review to the adjacent validation, promotion, backup-cleanup and crash windows. The audit did not merge, freeze, tag, delete a branch, or begin Phase 4.

## Auditor

Fresh independent Phase 3 re-audit context (R5), reviewing repository state, governing contracts, source, tests and exact-commit evidence independently from the implementation statement.

## Date

2026-08-18 (Asia/Shanghai)

## Gate Decision

**FAIL**

## Executive Summary

There is no new remediation candidate to accept. Both R4 findings remain present byte-for-byte: post-stage refresh still signs the complete current tree without proving which delta the owned stage produced, and the live ownership marker is still truncated in place rather than prepared and atomically promoted.

The requested whole-state-machine review also found a second user-data race adjacent to R4. `replaceOwnedTree` validates the old tree, then performs two renames and unconditionally removes the backup. A file inserted or changed after validation but before the first rename is moved into that backup and deleted without another proof check. The same two-rename sequence has no durable recovery record: a daemon or host interruption after the old tree is moved but before the new tree is promoted leaves the canonical target absent and an undiscoverable backup sibling. The implementation ignores both backup deletion failure and rollback failure.

These are not optional hardening items. They affect the contract that uncertain user/external data is never authenticated or deleted and that an interrupted installation remains safely recoverable. Green CI only proves the current tests do not exercise these windows. Phase 3 cannot pass or be frozen on this object.

## Verification Evidence

### Repository and immutable candidate

- `git branch --show-current` → `fix/phase3-audit-r3-remediation`.
- `git rev-parse HEAD` → `d50bf48c8a2102e5560f299b007c420cc1b9e815`.
- `git rev-parse origin/fix/phase3-audit-r3-remediation` → the same SHA.
- `git rev-parse origin/main` → `9459eb9fea435704d6639be8457ee11dfad02093`.
- `git fetch --prune origin` found no R4 remediation branch or commit.
- `git diff --check 9459eb9..d50bf48` — PASS.
- No `phase-003*` tag exists.
- Audit-start worktree was not clean: two resource LICENSE paths showed Windows working-tree metadata/EOL noise while their `git hash-object` values exactly matched the candidate blobs; `YORVA_0.3.0_x64_en-US.wxs` and `AUDIT-003R4-hermes-installation.md` were untracked. They are not part of the immutable candidate and were not modified by R5.

### Exact-commit GitHub Actions

- CI run [`32100135303`](https://github.com/YoLin02/yorva/actions/runs/32100135303) — push, branch `fix/phase3-audit-r3-remediation`, head `d50bf48c8a2102e5560f299b007c420cc1b9e815`, **SUCCESS**:
  - Web and API contract — success;
  - Go Node, including race, vet, govulncheck and build — success;
  - Windows Desktop native shell, lifecycle and Tauri no-bundle — success.
- MSI run [`32100135323`](https://github.com/YoLin02/yorva/actions/runs/32100135323) — same branch and head, **SUCCESS**:
  - Package and inspect MSI — success;
  - artifact `yorva-msi`, approximately 114 MB;
  - GitHub artifact digest `sha256:b0034433b881042a13418bbbd217b348a42f1a609230298323d3b0d323009830`.

These are the same exact-commit runs used by R4 because no new candidate exists.

### Local Web / OpenAPI

- `pnpm api:lint` — PASS.
- `pnpm api:generate` plus generated-schema diff check — PASS, no generated diff.
- `pnpm typecheck` — PASS.
- `pnpm lint` — PASS.
- `pnpm test` — PASS, 16 files / 64 tests.
- `pnpm build` — PASS.
- `pnpm audit --audit-level low` — PASS, no known vulnerabilities.

### Local Go

- `go test ./...` — PASS.
- affected Hermes adapter, application, SQLite and HTTP packages with `-count=20` — PASS.
- `go vet ./...` — PASS.
- `go build ./cmd/yorvad` — PASS.
- `govulncheck ./...` — PASS, no vulnerabilities found.
- local `go test -race ./...` — environment-blocked because CGo is disabled / no usable C toolchain (`-race requires cgo`). Exact-commit CI race is PASS; the local command is not reported as PASS.

### Local Rust / Windows / MSI

- `cargo fmt --all -- --check` — PASS.
- `cargo test --locked` — PASS, 10 tests.
- `cargo clippy --locked --all-targets --all-features -- -D warnings` — PASS.
- `cargo check --locked` — PASS.
- `cargo audit` — zero known vulnerabilities; 17 inherited allowed unmaintained/unsound warnings.
- `scripts/windows-lifecycle-smoke.ps1` — PASS.
- `scripts/inspect-yorva-msi.tests.ps1` — PASS for all existing negative cases.
- Supplemental local MSI administrative extraction/hash inspection — PASS for Hermes source, Node, npm and all three LICENSE payloads. The local MSI is not asserted to be the exact CI artifact.
- local Tauri release `--no-bundle` — environment-blocked with Windows `PermissionDenied` while an Owner process/build artifact held the release output. The process was not terminated. Exact-commit Windows CI no-bundle is PASS.

## R4 Finding Closure

| R4 finding | R5 result | Evidence |
| --- | --- | --- |
| HIGH-R4-001 post-stage delta provenance | **NOT CLOSED** | `runOwnedStage` still performs pre-check → external stage → `refreshOwnedInventory`; refresh authenticates old record fields/MAC but does not compare current inventory to the old manifest before signing the complete current tree (`host_installer.go:357-370`, `target.go:200-234`). No implementation diff exists after R4. |
| MEDIUM-R4-001 atomic ownership-record durability | **NOT CLOSED** | `writeOwnershipRecord` still calls `os.WriteFile` directly on the live marker after computing the manifest (`target.go:149-185`). There is no same-directory temporary file, flush/close, atomic replace, or injected promotion-failure test. |

## Dimension Results

### Scope

PASS. No Phase 4 Instance/Profile implementation, credentials/models, channels, lifecycle, Skills/MCP, Cloud, dynamic Runtime framework or Hermes fork was found.

### Correctness

FAIL. Legitimate retry success is covered, but concurrent foreign changes and interrupted tree promotion can still invalidate or strand the required recovery workflow.

### Architecture

PASS. The ownership implementation remains Hermes-adapter-owned and dependency direction is unchanged. The required correction can remain a small Hermes-specific filesystem transaction rather than a generic installer framework.

### Security

FAIL. The adapter still authenticates data whose provenance it has not established and has a validate-to-delete window for uncertain data.

### Data and Persistence

FAIL. Filesystem ownership proof and promotion state are not durably transacted. A later retry can delete foreign data, and a crash can leave the canonical target absent without a recorded recovery decision.

### Concurrency and Lifecycle

FAIL. The precheck→stage→refresh and validate→rename→backup-delete sequences both contain external mutation windows. Marker and tree promotion are not interruption-safe.

### Protocol and Compatibility

PASS. The R5 object does not regress the previously closed typed HTTP/OpenAPI, idempotency, SSE, Phase 2 discovery or version-compatibility contracts.

### Testing and Verification

FAIL. The implemented matrix is green but contains no deterministic injections for foreign insert/modify/delete during a stage, mutation after final validation but before rename, marker temp/promotion failure, or crashes at each directory-swap boundary.

### Maintainability

PASS WITH OBSERVATION. `host_installer.go` (approximately 514 lines) and `target.go` (approximately 424 lines) are review triggers but remain cohesive around Hermes installation orchestration and target ownership. The defect is an incomplete filesystem state machine, not file length alone. The repair should not create a generic manager/plugin framework.

### Documentation

FAIL. Phase 3 §10 promises that uncertain trees are never deleted and that a refresh never signs an unauthenticated tree; the candidate does not meet those statements. Roadmap/Spec also still show R4 pending because no R4 remediation exists.

### Dependencies / Supply Chain

PASS. No new dependency or mutable source was introduced; exact source/Node/npm/MSI verification remains green.

### Operations / Diagnostics

FAIL. Structured diagnostics remain redacted, but there is no durable recovery record for an interrupted directory promotion or marker replacement.

## Findings

### Critical

None.

### High

#### HIGH-R5-001 — R4 stage-delta provenance defect remains unremediated

`runOwnedStage` verifies the current tree only before an external stage, then `refreshOwnedInventory` checks only the old record's identity and MAC before hashing and signing every file present after the stage. It never proves that the old manifest still matched at the refresh boundary and never receives a stage-owned output manifest or allowed delta.

Reproduction chain:

1. Begin with a valid current-Operation record and matching tree.
2. Let `requireCurrentOwnedTree` pass.
3. While `venv`, `dependencies`, `path`, `config-templates` or `bootstrap-marker` runs, insert `foreign.txt`, replace an existing file, or remove one.
4. Let the stage return.
5. `refreshOwnedInventory` signs the complete mixed tree.
6. Fail or cancel later and retry.
7. Previous-proof validation accepts the foreign change; replacement can remove it as if YORVA created it.

Evidence: `services/node/internal/runtime/hermes/host_installer.go:348-370`; `services/node/internal/runtime/hermes/target.go:200-234`; the current `TestRefreshOwnedInventoryRejectsForeignAndCopiedRecords` intentionally modifies a file then expects refresh to authenticate it (`host_installer_retry_test.go:178-200`).

Impact: unrelated content can cross the ownership boundary and become deletable. This is a user-data/security defect and cannot be deferred or conditionally accepted.

Required closure: move target-mutating stages into an Operation-private, same-volume candidate tree or otherwise provide a stage-owned, verifiable delta. Never authenticate the whole live post-stage tree merely because the stage returned. A foreign insert, change or deletion during every mutating stage must leave the previous proof intact and fail closed without deleting the uncertain data.

#### HIGH-R5-002 — Validation-to-rename race can delete data added after the final proof check

`replaceOwnedTree` validates the old install tree at `target.go:362`, chooses a backup name, renames the live tree to that backup, promotes the new tree, then unconditionally calls `os.RemoveAll(backup)` at `target.go:365-377`. There is no lock or revalidation binding the validated tree to the directory actually renamed and deleted.

Reproduction chain:

1. Start a retry with a valid previous record and matching inventory.
2. Let `ownedPartialIdentity` pass at line 362.
3. Before `os.Rename(installDir, backup)`, another process creates or changes a file in the live tree.
4. The changed tree is renamed to `backup` without another inventory check.
5. The new tree is promoted.
6. `os.RemoveAll(backup)` deletes the late foreign content.

The same issue can occur after the rename if a same-user process writes into the discoverable backup path before cleanup. Cleanup ignores its error and does not verify backup identity.

Impact: the final replacement boundary can delete content never covered by the proof that authorized replacement. Fixing only post-stage refresh would leave this deletion path open.

Required closure: make final promotion a durable filesystem transaction. Immediately bind final validation to the object being moved; if that cannot be made race-free on supported Windows filesystems, quarantine the old tree and never automatically delete it in Phase 3. Any late change must abort promotion or remain recoverable in quarantine. Add a deterministic validate→mutation→rename test proving the foreign bytes survive.

### Medium

#### MEDIUM-R5-001 — Live ownership marker is still non-atomic and non-durable

`writeOwnershipRecord` computes a manifest and calls `os.WriteFile` on the only live marker. Opening/truncating the existing file precedes completion of the new authenticated record. Write failure, cancellation, process termination or host interruption can leave an empty/partial marker and permanently disable automatic retry.

Evidence: `services/node/internal/runtime/hermes/target.go:149-185`. No injected temp-write, flush, promotion or interruption test exists.

Required closure: serialize to a same-directory private `O_EXCL` regular temporary file, flush and close it, re-check containment/reparse invariants, then atomically replace the marker and durably commit the directory entry where supported. Preparation failure must preserve the old valid proof. Tests must inject write, sync, close and promotion failures and prove that either the old or new complete authenticated record survives.

#### MEDIUM-R5-002 — Two-rename tree promotion has no crash-recovery state

Promotion is:

```text
rename live target → random backup
rename prepared tree → live target
remove backup
```

There is no durable transaction/journal describing the chosen temporary and backup identities. A crash after the first rename leaves the canonical target absent; startup/preflight sees an absent target and has no authenticated rule for finding or restoring the random sibling. If the second rename fails, rollback is attempted but its error is ignored. A crash after the second rename leaves an untracked backup whose retention/cleanup ownership is unknown.

Evidence: `services/node/internal/runtime/hermes/target.go:315-377`. Existing tests cover returned rename/cancel errors, not process/host interruption at each boundary or startup recovery.

Required closure: introduce a small Phase-3-specific durable promotion record with explicit states such as `PREPARED`, `OLD_QUARANTINED`, `NEW_PROMOTED`, and `COMMITTED`. On startup or retry, recover deterministically: restore the proven old target if promotion never committed, accept the proven new target after commit, and never delete an unproven quarantine. Add crash-injection tests after every durable state transition and rename.

### Low

None.

### Info

#### INFO-R5-001 — Existing local/environment observations

Cargo retains 17 previously allowed warnings with no known vulnerability. Local Go race is CGo-toolchain-blocked and local Tauri no-bundle is locked by an Owner process/build artifact; exact-commit CI covers both. The root `.wxs` inspection residue and untracked historical R4 report must not enter a future freeze commit accidentally.

## Accepted Technical Debt

None. HIGH-R5-001 and HIGH-R5-002 affect user-data ownership and deletion safety. MEDIUM-R5-001 and MEDIUM-R5-002 are the directly adjacent interruption paths and should be closed in the same filesystem-state-machine remediation. None is eligible for Owner conditional acceptance under the current governance rules.

## Required Fixes Before Next Phase

The next remediation must close the ownership/retry subsystem as one bounded state machine rather than add another isolated check:

1. Build and mutate a same-volume Operation-private candidate tree; do not run long target-mutating stages against the live previously owned tree when an isolated candidate can be used.
2. Keep the prior authenticated tree immutable until final promotion. Revalidate it immediately at the promotion boundary.
3. If a race-free replace-and-delete cannot be proven on Windows, move the old tree to durable quarantine and do not automatically delete it in Phase 3.
4. Write ownership records through same-directory temporary files plus atomic promotion; preserve the prior proof on every failure.
5. Record directory promotion states durably and recover every crash boundary on startup/retry.
6. Treat PATH / `HERMES_HOME` and other mutations outside the install tree as separate idempotent postconditions; do not fold them into filesystem ownership proof.
7. Add deterministic injection coverage for foreign insert/modify/delete during every mutating stage, after validation/before rename, marker write/sync/promote failure, both rename failures, cancellation, timeout, daemon crash and host-restart recovery.
8. Retain the successful two-Operation retry test and every existing copied-marker, wrong pin/nonce/target, reparse, missing/changed file and uncertain-data fail-closed regression.
9. Create a genuinely new immutable remediation candidate, obtain exact-commit CI and MSI PASS, then request a fresh independent `AUDIT-003R6`.

This boundary does not require a generic transaction framework, package manager, plugin system or Phase 4 feature. A focused Hermes target-promotion module with explicit injectable filesystem operations is sufficient.

## Gate Rationale

No code changed after R4, so its blocking High necessarily remains. R5 additionally proves that the final validated tree can change before backup deletion and that the directory swap is not recoverable across interruption. `AUDIT_STANDARD.md` requires FAIL for unresolved blocking High findings, unsafe data behavior, violated security boundaries or insufficient evidence for a mandatory recovery capability. CI PASS cannot override direct source-level reproduction chains absent from the tests.

## Next Step

```text
Phase 3 Implementation: NOT ACCEPTED
AUDIT-003R5 Gate:       FAIL
Phase 3 status:         AUDIT / BLOCKED BY R5 FINDINGS
Merge / freeze / tag:   NOT AUTHORIZED
Feature branch delete:  NOT AUTHORIZED
Phase 4 planning:       BLOCKED
Phase 4 implementation: BLOCKED
```

Preserve `AUDIT-003`, `AUDIT-003R1`, `AUDIT-003R2`, `AUDIT-003R3`, `AUDIT-003R4` and this report as immutable audit history.

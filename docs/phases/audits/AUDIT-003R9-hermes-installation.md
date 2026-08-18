# YORVA Phase 3 Independent Re-Audit R9 — Hermes Installation

## Phase

Phase 3 — Hermes Installation (generation install transaction / ADR-0006)

## Baseline / Commit

- Historical Phase 2 baseline: `phase-002-hermes-discovery-baseline-r1` → `5b89d22ed5e7ae3f4374a26f0fcda54bdabc6bf9`
- Historical R6 FAIL candidate: `b79ad3c5ba4024a12aec61d615096260d40ead4b` (`AUDIT-003R6` — FAIL)
- Historical R7 FAIL candidate: `3957b61f120172fd60f3e10cef54c005a950c48c` (`AUDIT-003R7` — FAIL)
- Historical R8 PASS WITH CONDITIONS source: `7825319ddf217e229b715a4d53838ed4476dfa06` (`AUDIT-003R8`)
- Branch inspected: `fix/phase3-audit-r6-remediation`
- Immutable R9 candidate: `721325181892d0fd8534f9f7d287fe05d9603bb0`
- Tip commit subject: `fix: distinguish missing and invalid active.json pointers`

`git rev-parse 7213251` resolved to `721325181892d0fd8534f9f7d287fe05d9603bb0`. `git log -1 --oneline 7213251` is `7213251 fix: distinguish missing and invalid active.json pointers`. `git rev-parse HEAD` at audit time is the same SHA.

Working-tree noise: untracked `YORVA_0.3.0_x64_en-US.wxs` only. That file is not part of `7213251` and was not judged. Historical `AUDIT-003` through `AUDIT-003R8` were not edited.

No Phase 3 tag exists (`git tag --list 'phase-003*'` empty). This audit did not merge, freeze, tag, push, delete a branch, change implementation, or begin Phase 4 implementation.

## Auditor

Fresh independent Phase 3 R9 review context, separate from the implementation agent. Review started from governing documents and the locked tree, not from an implementation completion summary.

## Date

2026-08-18 (Asia/Shanghai)

## Gate Decision

`PASS`

## Executive Summary

R9 closes the remaining High on `721325181892d0fd8534f9f7d287fe05d9603bb0`: a present but unreadable, schema-invalid, escaping, reparse, or seal-failed `control/active.json` is no longer treated as a first-install vacancy.

Observed pointer classes are `MISSING`, `VALID`, and `INVALID`. `INVALID` is immediate `BLOCKED_UNSAFE`: Start and StartPrerequisites refuse a new transaction, `activate` does not write `active.json`, recovery does not pick newest `generations/*` or infer from PATH/Operation/directory presence, and the corrupt bytes are left in place. First activation requires observed `MISSING` plus an explicit `ActiveBefore=ABSENT` snapshot. Later activation is a compare-and-swap against `VALID(generationId + digest)`, or a roll-forward when the pointer already names this generation and seal. Recovery of an invalid pointer is idempotent and does not rewrite the file.

The four Owner-deferred items were re-read independently and remain Medium/Low. None of them is Critical or High, and none breaks the main first-install path or Phase 4 premises. Local `go test ./...` and `go vet ./...` pass. Exact-commit GitHub Actions run `32131548538` (CI) and run `32131548340` (MSI) are SUCCESS on head `7213251`.

Phase 3 §22 is not triggered: there is no unresolved Critical, High, or non-deferred Medium finding. This report does not freeze, tag, merge, or authorize Phase 4 implementation.

## Verification Evidence

### Repository and candidate

- `git branch --show-current` → `fix/phase3-audit-r6-remediation`
- `git rev-parse 7213251` → `721325181892d0fd8534f9f7d287fe05d9603bb0`
- `git rev-parse HEAD` → `721325181892d0fd8534f9f7d287fe05d9603bb0`
- `git log -1 --oneline 7213251` → `7213251 fix: distinguish missing and invalid active.json pointers`
- `git status --porcelain` → `?? YORVA_0.3.0_x64_en-US.wxs` only
- `git tag --list 'phase-003*'` — no tag
- Historical ownership files (`ownership_promote.go`, journal, delta, marker) remain absent from `services/node/internal/runtime/hermes/`
- No `ROLLED_BACK` / `ENV_APPLIED` states, second activation pointer, or plugin framework in `services/node`

### Exact-commit GitHub Actions

Confirmed against the public run pages. Head SHA on both runs is `721325181892d0fd8534f9f7d287fe05d9603bb0`. No other run IDs are reused.

- CI run [`32131548538`](https://github.com/YoLin02/yorva/actions/runs/32131548538) — push, branch `fix/phase3-audit-r6-remediation`, head `7213251`, **SUCCESS**:
  - Web and API contract;
  - Go Node (`ci.yml` runs `go test -race ./...`, `go vet ./...`, `govulncheck ./...`, `go build ./cmd/yorvad`);
  - Windows Desktop native shell.
- MSI run [`32131548340`](https://github.com/YoLin02/yorva/actions/runs/32131548340) — same branch and head, **SUCCESS**:
  - Package and inspect MSI;
  - artifact `yorva-msi`, digest `sha256:8146713032a074f385ad2d42a091b6ff6e17779cca440eee7eab8fd6062bc92e`.

Historical R6 (`32108290020` / `32108290027`) and R8-pending `7825319` runs are not reused.

### Local Go (required for this re-audit)

From `D:\workcode\myproject\services\node` against `7213251`:

- `go test ./...` — PASS (2026-08-18, this audit). Packages that ran: `internal/app` (0.437s), `internal/applog` (cached), `internal/bootstrap` (cached), `internal/daemon` (0.122s), `internal/domain/operation` (cached), `internal/events` (cached), `internal/install` (2.774s), `internal/persistence/sqlite` (cached), `internal/runtime` (cached), `internal/runtime/hermes` (4.360s), `internal/transport/httpapi` (0.155s). `cmd/yorvad` and `internal/buildinfo` have no tests; `internal/domain/node` has no tests.
- `go vet ./...` — PASS (2026-08-18, this audit; no diagnostics).

Not run in this audit (record, do not treat as a local PASS):

- `go test ./... -count=20`
- `go test -race ./...` (historically gcc/CGo-blocked on this host; not re-attempted as a fake PASS; exact-commit CI race is SUCCESS)
- `govulncheck ./...` (CI Go Node job includes it)
- frontend `pnpm` matrix (CI Web/API job SUCCESS)
- Cargo / Tauri / MSI locally (CI Windows native + MSI SUCCESS)

### Targeted R9 risk checks

| Risk | Result | Evidence |
| --- | --- | --- |
| Active pointer classes `MISSING` / `VALID` / `INVALID` | Closed | `ActiveClass` + `ReadActive` (`types.go:85-129`, `active_pointer.go:85-105`). `ErrNotFound` is `MISSING`; any other load/schema/reparse/size/JSON error or failed `VerifyPublishedGeneration` is `INVALID`. |
| `INVALID` is immediate `BLOCKED_UNSAFE` | Closed | `DecideRecovery` returns blocked before any txn action (`recovery_decide.go:6-9`). `rejectInstallGate` and `persistCreatedTransaction` refuse Start (`runtime_install.go:145-147`, `:312-319`). `StartPrerequisites` uses the same gate (`hermes_prerequisites.go:79`). |
| `INVALID` never overwrites `active.json` | Closed | `activate` and `snapshotActiveBefore` return `ErrBlockedUnsafe` before `WriteActive` (`activate.go:20-23`, `:43-46`, `:110-122`). `TestActivateInvalidPointerBlocksAndPreservesBytes` and `TestRuntimeInstallRejectsInvalidActivePointer` require the original bytes. |
| No newest-generation / PATH / Operation / directory inference | Closed | Recovery inputs are txn bytes, FS observation, `active.json`, registry (`recovery_decide.go:3-5`). `TestDecideRecoveryMatrix` row `none + generations exist + no active never auto-activates`. 002A3 Detect on `INVALID` falls through and does not select `gen_orphan` (`active_resolve.go:25-35`, `active_resolve_test.go:105-132`). |
| First activate only when observed `MISSING` | Closed | `casAllowsActivate` `ABSENT` requires `pointer.Missing()` (`activate.go:150-153`). `expectedPredecessorOrAbsent` `ABSENT` requires `obs.Active.Missing()` (`recovery_decide.go:201-202`). |
| `ActiveBefore` is `ABSENT` or `VALID(id + digest)` | Closed | `snapshotActiveBefore` writes kind + empty fields or kind + generation + seal (`activate.go:110-126`). `validateTransaction` rejects mixed/unknown kinds (`transaction.go:111-134`). |
| Strict CAS + already-this-gen roll-forward | Closed | `casAllowsActivate` (`activate.go:135-166`); tests `TestActivateMissingPointerFirstInstall`, `TestActivateExactPredecessorSucceeds`, `TestActivatePredecessorIDMismatchBlocks`, `TestActivatePredecessorDigestMismatchBlocks`, `TestActivateAlreadyNewGenerationRollsForward`. |
| Recovery idempotent; no rewrite of invalid pointer | Closed | `TestRecoverInvalidPointerBlocksWithoutRewrite`, `TestActivateCrashWindowsAndIdempotentRecover`. Second `RecoverWith` matches first; `{` bytes unchanged. |
| No `ROLLED_BACK` / `ENV_APPLIED` / journal / second pointer / plugin host | Closed | States remain `CREATED`…`COMMITTED`/`FAILED` (`types.go:7-15`). Sole writer of `active.json` in production is `Store.WriteActive` from `activate`. |

### Prior finding closure

| Finding | R9 result | Evidence |
| --- | --- | --- |
| HIGH — `active.json` MISSING vs INVALID conflated | **CLOSED** | See Gate Rationale. Old `LoadActive` success/`!Present \|\| !Valid` vacancy is gone. |
| HIGH-R7-001 persist-ahead Retry → second nonterminal | Still closed | `writeAtomicRecord` post-replace read-back; `persistCreatedTransaction` fails leftover `CREATED`/`BUILDING` then `HasNonterminal`; `ActionFailFailableExtras`. |
| MEDIUM-R7-001 `Execute` ignores `NextState` on env reconcile | Still closed | `recovery_execute.go:40-49`. |
| HIGH-R6-001 / 002 / 003 | Still closed | Official `-Stage path` skipped; seal re-walk; no live `hermes-agent` rename. |
| LOW-R8-003 `SECURITY.md` post-sync wording | **CLOSED** | `docs/SECURITY.md:157` now matches architecture §15.5 (fail closed before replace; complete readable record after replace). |
| LOW-R8-001 `List()` raw SQLite | **OPEN** (non-blocking) | `runtime_install.go:430-431`. |
| LOW-R8-002 dead ownership-nonce helpers | **OPEN** (non-blocking) | `RetryEligibleForPin` / `newOwnershipNonce` (`runtime_install.go:626-634`, `:682-687`). |

### Owner-deferred items (independent classification)

| ID | Independent class | Main install / Phase 4? | Notes |
| --- | --- | --- | --- |
| `P3-DEFERRED-001` | Medium | No | Official `uv` / `python` / `git` / templates still mutate shared `%LOCALAPPDATA%\hermes`. Failed new install cannot rewrite the active generation tree. Phase 4 Instances do not depend on relocating that home. |
| `P3-DEFERRED-002` | Low | No | `RemovableBins` lists only other `generations/*/bin` (`environment.go:120-133`). Leftover `hermes-agent\bin` is never stripped. More conservative than an unproven “YORVA wrote this PATH entry” claim. 002A3 ignores leftover PATH when the pointer is valid. |
| `P3-DEFERRED-003` | Low | No | D4 retention is implemented in `gc.go`. Older proven unreferenced dirs may be collected. Unknown trees and official user data are not. Phase 4 does not require unbounded txn history. |
| `P3-DEFERRED-004` | Low | No | Spec header still cites R8 freeze-pending (`PHASE-003-hermes-installation.md:14-20`). Numbering/status wording only. |

None of the four is raised to Critical or High.

## Dimension Results

| Dimension | Result | Notes |
|---|---|---|
| Scope | PASS | Generation transaction remains the Phase 3 machine. No Instance/Profile implementation in `services/node`. `docs/phases/PHASE-004-instance-profile.md` is a DRAFT spec in ancestry (`eb859b9`) and does not authorize code. |
| Correctness | PASS | MISSING / VALID / INVALID are distinct. INVALID cannot be first-activated or overwritten. First-install `MISSING`+`ABSENT` and predecessor CAS are tested. |
| Architecture | PASS | Filesystem transaction remains recovery authority. `active.json` remains the sole activation pointer. Operation remains a one-way projection. |
| Security | PASS | Corrupt pointer is fail-closed. Seal re-walk on publish/activate. Leftover `hermes-agent` is not adopted or deleted. No new unauthenticated surface. Same-user rewrite of `active.json` remains architecture §15.10 residual. |
| Data and Persistence | PASS | `ActiveBeforeKind` is persisted and validated. Complete replaced records remain recovery truth. Create still refuses a second nonterminal. |
| Concurrency and Lifecycle | PASS | Create and `Apply` take `install.lock`. Start/prerequisites re-read the pointer under the create lock / gate. Recovery of INVALID does not mutate. |
| Protocol and Compatibility | PASS | OpenAPI Operation/SSE shape unchanged. `INSTALL_BLOCKED_UNSAFE` is a stable application code (`discovery.go:49`) returned in the existing error envelope. Phase 2 Detect DTO unchanged; 002A3 INVALID falls through for read-only Detect only. |
| Testing and Verification | PASS | Required R9 CAS / INVALID / recovery regressions exist and are green. Local `go test`/`go vet` PASS. Exact-commit CI/MSI for this SHA SUCCESS. |
| Maintainability | PASS | `internal/install` remains cohesive. Dual `Class`/`Present`/`Valid` fallback and unused nonce helpers are Lows. |
| Documentation | PASS | `SECURITY.md`, ADR-0006, architecture §5, 003A4, and 002A3 now state INVALID ≠ MISSING. Residual header status drift is Low. |
| Dependencies / Supply Chain | PASS | No new runtime dependency. Hermes/Node/npm pins were not reopened. |
| Operations / Diagnostics | PASS | INVALID latches the install gate fail-closed until the pointer is no longer invalid and Recover runs. Desktop still surfaces the stable error code. |

## Findings

### Critical

None.

### High

None.

The R9 High is closed. The previous execution chain no longer holds:

1. `Store.ReadActive` classifies absence as `MISSING` and every other failure — including malformed JSON, non-regular/reparse/oversize, schema/id/path failure, and seal/containment failure — as `INVALID` (`active_pointer.go:85-105`).
2. `DecideRecovery` returns `BLOCKED_UNSAFE` on `obs.Active.Invalid()` before inspecting transactions (`recovery_decide.go:6-9`). The matrix row `none + absent + malformed pointer` now expects block, not `READY`/`NONE` (`recovery_decide_test.go:46-49`).
3. `persistCreatedTransaction` refuses to insert `CREATED` when the pointer is invalid (`runtime_install.go:145-147`). `rejectInstallGate` does the same for Start and StartPrerequisites and sets the in-memory gate (`runtime_install.go:312-319`; `hermes_prerequisites.go:79`). `TestRuntimeInstallRejectsInvalidActivePointer` also asserts the file bytes are unchanged.
4. `activate` will not snapshot or `WriteActive` on `INVALID` (`activate.go:20-23`, `:43-46`, `:110-112`). `casAllowsActivate` allows first write only when expected `ABSENT` and observed `MISSING` (`activate.go:150-153`). Predecessor replace requires id + digest (`activate.go:155-161`). Already-this-generation rolls forward (`activate.go:24-26`, `:47-49`, `:139-140`).
5. Recovery of `{` is `BLOCKED_UNSAFE` twice and does not rewrite the file (`activate_cas_test.go:182-213`). 002A3 Detect on the same class falls through to frozen enumeration and does not select a newest orphan generation (`active_resolve_test.go:105-132`).

That is the required contract. It is not treated as “absent” and it is not overwritten.

### Medium

None.

### Low

#### LOW-R9-001 — `List()` does not project Operation status from the transaction

Same as LOW-R8-001 / LOW-R7-001. `Get` repairs `FAILED`→`SUCCEEDED` when the txn is `COMMITTED` (`runtime_install.go:226-252`). `List` returns raw SQLite rows (`runtime_install.go:430-431`). Desktop reload that uses the list endpoint can show a stale terminal status until a later `GET`. Events remain notifications; `GET` is still the documented source of truth.

#### LOW-R9-002 — Dead ownership-nonce helpers remain

`RetryEligibleForPin` still requires `OwnershipNonce` (`runtime_install.go:626-634`). `newOwnershipNonce` is still defined (`runtime_install.go:682-687`). Production `Start` / `execute` do not generate or consult a nonce. Dead code is not retry authority.

#### LOW-R9-003 — `ActiveBefore` is snapshotted at `ACTIVATING`, not at `PUBLISHED`

Live `publish` persists `PUBLISHED` without `ActiveBefore*` (`publish.go:64-70`). Snapshot happens in `activate` / `persistActivating` (`activate.go:28-37`; `recovery_execute.go:109-116`). The Decide matrix row `PUBLISHED activate forward` uses the test helper that already sets `ActiveBefore: genB` (`recovery_decide_test.go:13-17`, `:135-137`). A replace-path crash after `PUBLISHED` persist and before the snapshot would see empty kind + empty generation, treat that as `ABSENT`, observe a `VALID` predecessor, and fail closed (`expectedPredecessorOrAbsent` + `decidePublished`). That is not an overwrite of `INVALID` and is not the Phase 3 first-install path (`MISSING` + `ABSENT` still forwards). It is a fail-closed gap against architecture §8.1’s “PUBLISHED + predecessor → persist ACTIVATING” row if a later phase performs generation replace.

#### LOW-R9-004 — Dual `Class` / `Present` / `Valid` representation

`Missing` / `Invalid` / `IsValid` prefer `Class` when set and otherwise infer from `Present`/`Valid` (`types.go:110-129`). Production `ReadActive` always sets `Class`. `Collect` still reads the `Valid` field (`gc.go:56-61`, `:91-95`). Recover does not GC while the pointer is `INVALID` (block returns before `ActionNone`). The dual form is a maintainability footgun, not a current overwrite path.

#### LOW-R9-005 — Phase 3 spec header still describes the R8 freeze condition

`docs/phases/PHASE-003-hermes-installation.md:14-20` still says `AUDIT-003R8` accepted `7825319` with freeze pending exact-commit CI, and “independent audit remains pending” at line 722. `PHASE-004-instance-profile.md:6` still says `AUDIT-003R8` PENDING. Status-line drift only.

### Info

#### INFO-R9-001 — Observations that are not defects in the judged tree

- Untracked `YORVA_0.3.0_x64_en-US.wxs` must stay out of any freeze commit.
- `docs/phases/PHASE-004-instance-profile.md` is a DRAFT planning document in the ancestry of `7213251`. Grep of `services/node/internal` found no Instance/Profile implementation. This audit does not authorize Phase 4 code.
- `ReconcileManagedEnvironment` (`environment_apply.go:143-166`) is unused. Daemon startup uses `install.Recover` (`daemon.go:100-102`).
- `HostInstaller.applyGeneration` can allocate a new transaction if `TransactionID` is empty (`apply_generation.go:51-55`). HTTP Start always links a transaction after the INVALID check when `managedRoot` is set. `activate` still refuses to overwrite `INVALID`.
- Same-user rewrite of `control/active.json` remains the accepted residual (architecture §15.10).
- Two leftover `SEALED`/`PUBLISHED`/`ACTIVATING` records still become `BLOCKED_UNSAFE` without a per-record recover attempt. The create path cannot add that pair.
- Local `-race` was not run; exact-commit CI race is SUCCESS.

## Accepted Technical Debt

Owner-deferred items, independently kept Medium/Low and not gate-blocking:

| ID | Severity | Why accepted | Owner | Trigger |
| --- | --- | --- | --- | --- |
| `P3-DEFERRED-001` | Medium | Shared `HERMES_HOME` side effects are residual, not an `active.json` overwrite or first-install break | Repository owner | Later HERMES_HOME / tool-isolation design |
| `P3-DEFERRED-002` | Low | Implementation never removes unproven `hermes-agent\bin` PATH entries | Repository owner | If a later phase must prove YORVA PATH authorship |
| `P3-DEFERRED-003` | Low | D4 retention is enough for Phase 3 / Phase 4 premises | Repository owner | If audit/history product requires longer retention |
| `P3-DEFERRED-004` | Low | Header/numbering drift does not change runtime safety | Repository owner | Freeze/docs pass |

Lows LOW-R9-001 through LOW-R9-005 may be deferred. They are not relabeled as High to force another FAIL.

## Required Fixes Before Next Phase

None that block Phase 3 source acceptance.

Before merge / freeze / tag (Owner freeze task, not a new implementation phase):

1. Freeze `721325181892d0fd8534f9f7d287fe05d9603bb0` (or a later docs-only commit that an audit explicitly includes). Keep `YORVA_0.3.0_x64_en-US.wxs` untracked.
2. Preserve `AUDIT-003` through `AUDIT-003R9` as immutable history.
3. Do not begin Phase 4 implementation until the freeze/tag exists. The Phase 4 spec draft is not authorization.

Optional non-blocking follow-ups: LOW-R9-001 through LOW-R9-005.

## Gate Rationale

Phase 3 §22 makes any Critical, High, or non-deferred Medium finding gate-blocking. The advertised R9 High was MISSING vs INVALID conflation. This pass re-read `ReadActive`, `DecideRecovery`, `casAllowsActivate`, Start/prerequisites, 002A3 Detect, and the new tests instead of trusting the commit message.

A present corrupt pointer can no longer be first-activated, overwritten, or replaced by a newest-generation guess. First install still proceeds only from observed `MISSING` plus `ActiveBefore=ABSENT`. Predecessor replace is exact CAS. Recovery of INVALID is idempotent and non-mutating. R7/R8 closures still hold. Owner-deferred items stay Medium/Low. Required local Go checks are green. Exact-commit CI `32131548538` and MSI `32131548340` are SUCCESS on this SHA.

That is enough for source acceptance. It is not a freeze, merge, tag, or Phase 4 unlock.

## Next Step

```text
Phase 3 Implementation: ACCEPTED AT SOURCE (7213251)
AUDIT-003R9 Gate:       PASS
HIGH MISSING/INVALID:   CLOSED
Phase 3 status:         AUDIT / ACCEPTED — freeze/tag is a separate Owner task
Merge / freeze / tag:   NOT PERFORMED by this audit
Feature branch delete:  NOT AUTHORIZED
Phase 4 implementation: BLOCKED until freeze/tag
```

Preserve `AUDIT-003` and `AUDIT-003R1` through `AUDIT-003R9` as immutable audit history. Do not begin Phase 4 implementation.

# YORVA Phase 3 Independent Re-Audit R8 — Hermes Installation

## Phase

Phase 3 — Hermes Installation (generation install transaction / ADR-0006)

## Baseline / Commit

- Historical Phase 2 baseline: `phase-002-hermes-discovery-baseline-r1` → `5b89d22ed5e7ae3f4374a26f0fcda54bdabc6bf9`
- Historical R6 FAIL candidate: `b79ad3c5ba4024a12aec61d615096260d40ead4b` (`AUDIT-003R6` — FAIL)
- Historical R7 FAIL candidate: `3957b61f120172fd60f3e10cef54c005a950c48c` (`AUDIT-003R7` — FAIL; immutable; not edited)
- Generation implementation landing: `6117ef1596b2edfeae504d7e82e7e0cb1865852b`
- Branch inspected: `fix/phase3-audit-r6-remediation`
- Immutable R8 candidate: `7825319ddf217e229b715a4d53838ed4476dfa06`
- Tip commit subject: `fix: close Phase 3 AUDIT-003R7 persist-ahead retry HIGH`

`git rev-parse 7825319` resolved to `7825319ddf217e229b715a4d53838ed4476dfa06`. At the start of this audit `git rev-parse HEAD` was the same SHA. Before the report was written, HEAD moved to `eb859b975b920fb4e6971d3c379f83e9ec715825` (`docs: draft Phase 4 Instance spec without implementation authorization`). `git diff 7825319 HEAD -- services/node` is empty. This audit judges `7825319` only. The later docs-only commit is not part of the candidate and is not Phase 4 implementation authorization.

Working tree noise at report time: untracked `YORVA_0.3.0_x64_en-US.wxs` and untracked `docs/phases/audits/AUDIT-003R7-hermes-installation.md`. Neither is part of `7825319`. Historical `AUDIT-003` through `AUDIT-003R7` were not edited.

No Phase 3 tag exists. This audit did not merge, freeze, tag, push, delete a branch, change implementation, or begin Phase 4 implementation.

## Auditor

Fresh independent Phase 3 R8 review context, separate from the implementation agent. Review started from governing documents and the locked tree, not from an implementation completion summary.

## Date

2026-08-18 (Asia/Shanghai)

## Gate Decision

`PASS WITH CONDITIONS`

## Executive Summary

R8 closes the two R7 gate-blockers on the locked candidate `7825319ddf217e229b715a4d53838ed4476dfa06`.

After a successful atomic replace, a complete readable transaction record is treated as persisted: post-replace directory-sync failure no longer fails `SaveTransaction`, and `Manager.persist` continues if a higher revision is already on disk. Create under `install.lock` first fails leftover `CREATED`/`BUILDING` records, then refuses a new `CREATED` while any nonterminal record remains. `DecideRecovery` recovers failable extras once (`ActionFailFailableExtras`) before `BLOCKED_UNSAFE`. `Execute` persists `NextState` before `ReconcileEnvironment`, so `FAILED` + valid `active.json` + incomplete environment rolls forward.

Seal re-walk on publish/activate, 002A3 leftover-`PATH` ignore, D5 user-authored `hermes-agent\bin` retention, Operation projection for `ACTIVATING`/`COMMITTED`, and create-under-lock remain implemented. Local `go test ./...` and `go vet ./...` are green. Exact-commit GitHub Actions for `7825319` was not observed and is PENDING. Remote `fix/phase3-audit-r6-remediation` still shows `6117ef1` as the newest published commit; `3957b61` and `7825319` were not seen on GitHub Actions.

Remaining items are non-blocking Lows and the freeze-time CI condition. Phase 3 §22 is not triggered: there is no unresolved Critical, High, or Medium finding in the inspected source.

## Verification Evidence

### Repository and candidate

- `git branch --show-current` → `fix/phase3-audit-r6-remediation`
- `git rev-parse 7825319` → `7825319ddf217e229b715a4d53838ed4476dfa06`
- Audit-start `git rev-parse HEAD` → `7825319ddf217e229b715a4d53838ed4476dfa06`
- Report-time `git rev-parse HEAD` → `eb859b975b920fb4e6971d3c379f83e9ec715825` (docs-only; `services/node` unchanged)
- `git status --porcelain` at report time → `?? YORVA_0.3.0_x64_en-US.wxs` and `?? docs/phases/audits/AUDIT-003R7-hermes-installation.md`
- `git tag --list 'phase-003*'` — no tag
- Historical ownership files (`ownership_promote.go`, journal, delta, marker) remain absent from `services/node/internal/runtime/hermes/`

### Exact-commit GitHub Actions

PENDING. This auditor opened `https://github.com/YoLin02/yorva/actions?query=7825319ddf217e229b715a4d53838ed4476dfa06` and the branch commit list. No workflow run whose head SHA is `7825319ddf217e229b715a4d53838ed4476dfa06` or `3957b61f120172fd60f3e10cef54c005a950c48c` was observed. The newest published branch runs belong to `6117ef1` (`32126118908`, `32126119025`) and must not be reused. Historical R6 runs (`32108290020`, `32108290027`) belong to `b79ad3c` and are not reused. No run ID is invented.

### Local Go (required for this re-audit)

From `D:\workcode\myproject\services\node` against the `7825319` node tree (`git diff 7825319 HEAD -- services/node` empty):

- `go test ./...` — PASS (2026-08-18, this audit). Packages that ran: `internal/app` (0.454s), `internal/applog` (cached), `internal/bootstrap` (cached), `internal/daemon` (0.148s), `internal/domain/operation` (cached), `internal/events` (cached), `internal/install` (2.420s), `internal/persistence/sqlite` (cached), `internal/runtime` (cached), `internal/runtime/hermes` (4.372s), `internal/transport/httpapi` (0.175s). `cmd/yorvad` and `internal/buildinfo` have no tests; `internal/domain/node` has no tests.
- `go vet ./...` — PASS (2026-08-18, this audit; no diagnostics).

Not run in this audit (record, do not treat as PASS):

- `go test ./... -count=20`
- `go test -race ./...` (historically gcc/CGo-blocked on this host; not re-attempted as a fake PASS)
- `govulncheck ./...`
- frontend `pnpm` matrix
- Cargo / Tauri / MSI (explicitly out of scope for this re-audit)

### Targeted R8 risk checks

| Risk | Result | Evidence |
| --- | --- | --- |
| Complete replaced record treated as persisted | Closed | `writeAtomicRecord` ignores post-replace `SyncDir` after matching read-back (`atomic.go:135-155`); `Manager.persist` continues when disk revision advanced (`manager.go:231-247`); `TestDirSyncFailure/after replace` now requires `SaveTransaction` success (`transaction_store_test.go:167-201`) |
| Refuse `CREATED` while another nonterminal exists, or fail leftover `CREATED`/`BUILDING` first | Closed | `persistCreatedTransaction` holds `install.lock`, calls `FailFailableNonterminals`, then `HasNonterminal` (`runtime_install.go:133-150`); `TestRetryFailsLeftoverBuildingTransaction` |
| Recover each failable nonterminal once before `BLOCKED_UNSAFE` | Closed for the R7 chain | `hasFailableNonterminal` + `ActionFailFailableExtras` (`recovery_decide.go:23-27`, `:229-236`; `recovery_execute.go:38-39`, `:120-129`); `TestRecoverFailsFailableExtrasThenReady`; two `SEALED`/`PUBLISHED` still block immediately (conservative; create path can no longer add a second nonterminal beside them) |
| `Execute` honors `NextState` for `FAILED`+active+incomplete env | Closed | `recovery_execute.go:40-49`; `Manager.fail` will not write `FAILED` while `active.json` names this generation (`manager.go:249-257`); `TestRecoverFailedActiveIncompleteEnvRollsForward` |
| Operation `FAILED`/`CANCELLED`/`INTERRUPTED` while txn `ACTIVATING`/`COMMITTED` | Still closed | `filesystemOwnsOperation` + `projectOperation` (`runtime_install.go:218-307`, `:544-561`); `execute.fail` (`runtime_install_run.go:96-104`, `:151-156`) |
| Publish/activate re-walk sealed tree | Still closed | `VerifySealedTree` (`verify.go:16-31`); callers `publish.go:61`, `activate.go:16`, `activate.go:97`; `TestVerifySealedTreeRejectsPostSealMutation` |
| 002A3 leftover `hermes-agent` on PATH | Still closed — pointer wins, not `AMBIGUOUS` | `active_resolve.go:41-45`; `candidates.go:62-80`, `:102-107`, `:303-321`; `TestActivePointerIgnoresLeftoverHermesAgentOnPATH` |
| D5 user-authored `hermes-agent\bin` | Still closed — not removed by official-path equality | `RemovableBins` lists only other `generations/*/bin` (`environment.go:120-133`); `TestReconcileAfterCommitKeepsUserHermesAgentBin` |
| Transaction create under `install.lock` | Closed; sibling nonterminals now excluded | `runtime_install.go:133-150`; live `Apply` also holds the lock (`apply_generation.go:23-27`) |

### R7 finding closure

| R7 finding | R8 result | Evidence |
| --- | --- | --- |
| HIGH-R7-001 persist-ahead + Retry → second nonterminal → immediate `BLOCKED_UNSAFE` | **CLOSED** | Writer treats complete replace as persisted; create fails leftover `CREATED`/`BUILDING` then refuses another nonterminal; recovery fails failable extras once; regressions exist |
| MEDIUM-R7-001 `Execute` ignores `NextState` on `ActionReconcileEnv` | **CLOSED** | `Execute` persists `ACTIVATING` before `ReconcileEnvironment`; Recover integration test requires `COMMITTED` or `ACTIVATING` and forbids `RecoverWith` error |
| LOW-R7-001 `List()` does not project Operation status | **OPEN** (non-blocking) | `runtime_install.go:419-421` still returns raw SQLite rows |
| LOW-R7-002 amendment headers say `NOT STARTED` | **CLOSED** | `AMENDMENT-002A3` and `AMENDMENT-003A4` headers now describe landed batches / R7 FAIL / R8 pending |
| LOW-R7-003 dead ownership-nonce helpers | **OPEN** (non-blocking) | `RetryEligibleForPin` / `newOwnershipNonce` still unused (`runtime_install.go:615-677`) |

### R6 finding closure (re-checked, not reopened)

| R6 finding | R8 result | Evidence |
| --- | --- | --- |
| HIGH-R6-001 official `path` invalidates ownership | **CLOSED** | Official `-Stage path` is skipped; launchers copied before Seal (`build.go:72-103`, `build.go:139-160`) |
| HIGH-R6-002 allowed-prefix foreign executables | **CLOSED** | Publish/activate re-walk the manifest |
| HIGH-R6-003 rename-before-journal crash | **CLOSED** | Live `hermes-agent` rename / promotion journal remain absent |
| MEDIUM-R6-001 ignored `SyncDir` | **CLOSED as restated by R7/architecture §15.5** | Pre-replace dir-sync still fails closed (`atomic.go:120-123`); post-replace dir-sync no longer fails a complete readable record |
| MEDIUM-R6-002 path-manifest / A2 status docs | **CLOSED** | Phase 3 §10, `SECURITY.md`, `RUNTIME.md` describe generation seal/env |

## Dimension Results

| Dimension | Result | Notes |
|---|---|---|
| Scope | PASS | Generation transaction remains the Phase 3 machine. No Instance/Profile implementation, credentials, channels, Cloud, generic installer framework, or Hermes fork in `7825319`. A later uncommitted-to-candidate Phase 4 spec draft exists at `eb859b9` and is out of this candidate. |
| Correctness | PASS | R7 persist-ahead Retry chain is closed at the writer, the create gate, and recovery. `FAILED`+active+incomplete env rolls forward. |
| Architecture | PASS | Filesystem transaction remains recovery authority. Operation remains a one-way projection. `DecideRecovery` now implements “recover each failable once” before blocking. |
| Security | PASS | Seal re-walk on publish/activate; leftover `hermes-agent` is not adopted or deleted; no new unauthenticated surface; secrets are not returned by ordinary read APIs. Same-user rewrite of `active.json` remains the documented residual (architecture §15.10). |
| Data and Persistence | PASS | Complete replaced records are persisted to the caller. Create will not insert a second nonterminal beside an existing one. |
| Concurrency and Lifecycle | PASS | Create and `Apply` take `install.lock`. Leftover `CREATED`/`BUILDING` are failed under the lock before a new `CREATED` is written. Live `Apply` holds the lock, so Retry cannot mutate the in-flight record. |
| Protocol and Compatibility | PASS | OpenAPI Operation/SSE shape is unchanged. `transaction_id` is a one-way projection and is not an activation source. Phase 2 DTO is unchanged. |
| Testing and Verification | PASS | Required R7 regressions exist and are green. Local `go test`/`go vet` PASS. Exact-commit CI for this SHA is PENDING and is a freeze condition, not a source High. |
| Maintainability | PASS | `internal/install` remains cohesive. `RetryEligibleForPin` / `newOwnershipNonce` remain unused leftovers. |
| Documentation | PASS | Phase 3 §10, `SECURITY.md`, `RUNTIME.md`, `DATA_MODEL.md` match the generation contract. Residual wording drift is Low. |
| Dependencies / Supply Chain | PASS | No new runtime dependency. Hermes/Node/npm pins were not reopened. |
| Operations / Diagnostics | PASS | Two leftover failable nonterminal files no longer permanently `BLOCKED_UNSAFE`. Two leftover sealed/published records still block, but the create path can no longer produce that pair. |

## Findings

### Critical

None.

### High

None.

HIGH-R7-001 is closed. The R7 execution chain no longer holds:

1. `writeAtomicRecord` after replace reads the destination back. If the bytes match the payload, a post-replace `SyncDir` error is discarded and the writer returns success (`atomic.go:135-155`). `TestDirSyncFailure/after replace` now fails the test if `SaveTransaction` returns an error.
2. `Manager.persist` additionally treats a readable higher revision as success (`manager.go:231-247`). Seal/publish/activate persist paths use this helper (`manager.go:97-98`, `:150-151`; `publish.go:69`; `activate.go:33`, `:55`).
3. `persistCreatedTransaction` acquires `install.lock`, fails leftover `CREATED`/`BUILDING`, and refuses insertion while any nonterminal remains (`runtime_install.go:133-150`). `TestRetryFailsLeftoverBuildingTransaction` starts from a leftover `BUILDING` record and asserts the old id is `FAILED` and the Retry id is a new nonterminal.
4. `DecideRecovery` no longer blocks immediately on two nonterminals when any is `CREATED` or `BUILDING` (`recovery_decide.go:23-27`). `Execute`/`failFailableExtras` marks those records `FAILED` (`recovery_execute.go:120-129`). `Recover` then re-observes. `TestRecoverFailsFailableExtrasThenReady` ends `READY` with both leftovers `FAILED`.
5. Live `HostInstaller.Apply` holds the same lock for Seal/Publish/Activate/Reconcile (`apply_generation.go:23-27`), so a same-session Retry cannot fail an in-flight worker record out from under it; it gets `ErrLockBusy` / `INSTALL_NOT_READY` or `RUNTIME_INSTALL_IN_PROGRESS`.

The advertised Retry path plus the architecture §13.1 / §15.5 dir-sync failpoint therefore cannot allocate a second nonterminal beside a complete replaced record, and cannot leave the install gate permanently `BLOCKED_UNSAFE` for two failable leftovers.

### Medium

None.

MEDIUM-R7-001 is closed. `DecideRecovery` still emits `NextState: StateActivating` and `Action: ActionReconcileEnv` for `FAILED` + valid pointer + incomplete env (`recovery_decide.go:39-43`). `Execute` now persists that `NextState` before `ReconcileEnvironment` (`recovery_execute.go:40-49`). `ReconcileEnvironment` accepts only `ACTIVATING` or `COMMITTED` (`environment_apply.go:43-45`). `primaryTxn` selects the failed-but-active record when no nonterminal remains (`observe.go:51-68`). `TestRecoverFailedActiveIncompleteEnvRollsForward` starts from on-disk `FAILED` + valid pointer and requires `COMMITTED` or `ACTIVATING` with `READY`/`RECONCILING`, and treats a `RecoverWith` error as failure. `Manager.fail` also refuses to persist `FAILED` while `active.json` names this generation (`manager.go:249-257`).

### Low

#### LOW-R8-001 — `List()` does not project Operation status from the transaction

Same as LOW-R7-001. `Get` repairs `FAILED`→`SUCCEEDED` when the txn is `COMMITTED` (`runtime_install.go:226-252`, `TestGetRepairsFailedOperationWhenTransactionCommitted`). `List` returns raw SQLite rows (`runtime_install.go:419-421`). Desktop reload that uses the list endpoint can show a stale terminal status until a later `GET`. Events remain notifications; `GET` is still the documented source of truth.

#### LOW-R8-002 — Dead ownership-nonce helpers remain

`RetryEligibleForPin` still requires `OwnershipNonce` (`runtime_install.go:615-623`). `newOwnershipNonce` is still defined (`runtime_install.go:671-677`). Production `Start`/`execute` no longer generate or consult a nonce. Dead code is not retry authority.

#### LOW-R8-003 — `SECURITY.md` still says post-sync durability failures fail closed

`docs/SECURITY.md:157` still states that atomic transaction/seal/pointer writes “fail closed if parent-directory durability fails after the temporary file is synced.” Pre-replace dir-sync still fails closed (`atomic.go:120-123`). After replace, architecture §15.5 and the R8 writer treat a complete readable record as persisted (`atomic.go:152-155`). The security property (do not activate a half record) is preserved by read-back. The sentence is stale relative to the implemented recovery truth.

#### LOW-R8-004 — Phase 3 spec header still says `AUDIT-003R7` is PENDING

`docs/phases/PHASE-003-hermes-installation.md:14` and `:51` still read `AUDIT-003R7` as PENDING. `AUDIT-003R7` is an immutable FAIL. Status-line drift only.

### Info

#### INFO-R8-001 — Verification gaps that are not defects in the inspected code

- Exact-commit CI/MSI for `7825319ddf217e229b715a4d53838ed4476dfa06` was not seen. Remote branch history on GitHub still ends at `6117ef1`.
- Local `-race` was not run.
- Frontend, Cargo, Tauri, and MSI were not run (out of this re-audit’s required commands).
- Untracked `YORVA_0.3.0_x64_en-US.wxs` must stay out of any future freeze commit.
- Untracked `AUDIT-003R7` must be committed as immutable history before freeze, not rewritten.
- Same-user rewrite of `control/active.json` remains an accepted residual (architecture §15.10).
- Two leftover `SEALED`/`PUBLISHED`/`ACTIVATING` records still become `BLOCKED_UNSAFE` without a per-record recover attempt. The create path can no longer add a second nonterminal beside them. That pair is no longer the R7 Retry chain.
- Report-time HEAD `eb859b9` adds `docs/phases/PHASE-004-instance-profile.md`. That is a spec draft on top of the candidate, not Phase 4 implementation, and it is not this audit’s accepted tree.

## Accepted Technical Debt

None required to pass the source gate. The Conditions below are explicit and time-bounded. Lows are not relabeled as High to force another FAIL, and they are not hidden.

## Required Fixes Before Next Phase

None that block Phase 3 source acceptance.

Before merge / freeze / tag (Owner freeze task, not a new implementation phase):

1. Observe green exact-commit GitHub Actions for `7825319ddf217e229b715a4d53838ed4476dfa06` (Go including race, Web/API, Windows native/Tauri) and the MSI job. Do not reuse `6117ef1` or R6 run IDs.
2. Freeze the accepted tree, not `eb859b9`, unless a later audit explicitly includes that docs commit. Keep `YORVA_0.3.0_x64_en-US.wxs` untracked.
3. Preserve `AUDIT-003` through `AUDIT-003R8` as immutable history.

Optional non-blocking follow-ups: LOW-R8-001 through LOW-R8-004.

## Gate Rationale

Phase 3 §22 makes any Critical, High, or Medium finding gate-blocking. R7 HIGH-R7-001 and MEDIUM-R7-001 were those findings. This pass re-read the writer, the create lock, `DecideRecovery`, `Execute`, and the new tests instead of trusting the commit message.

The persist-ahead failpoint now continues. Retry cannot insert a second in-flight transaction beside a leftover nonterminal. Recovery fails leftover `CREATED`/`BUILDING` records and returns `READY`. `FAILED` + `active.json` + incomplete env is no longer stuck on `ErrInvalidRecord`. Re-checks of seal re-walk, leftover PATH, D5, Operation projection, and create-lock still hold.

That is enough for source acceptance. It is not enough to invent a CI PASS, freeze, or unlock Phase 4 implementation. Exact-commit CI remains PENDING and is recorded as a freeze condition. Remaining Lows do not break the install transaction contract.

## Next Step

```text
Phase 3 Implementation: ACCEPTED AT SOURCE (7825319)
AUDIT-003R8 Gate:       PASS WITH CONDITIONS
Phase 3 status:         AUDIT / ACCEPTED PENDING FREEZE CONDITIONS
Merge / freeze / tag:   NOT AUTHORIZED until exact-commit CI/MSI for 7825319 is observed
Feature branch delete:  NOT AUTHORIZED
Phase 4 planning:       spec draft at eb859b9 is not implementation authorization
Phase 4 implementation: BLOCKED until freeze
```

Preserve `AUDIT-003` and `AUDIT-003R1` through `AUDIT-003R8` as immutable audit history. Do not begin Phase 4 implementation.

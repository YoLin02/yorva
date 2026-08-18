# YORVA Phase 3 Independent Re-Audit R7 — Hermes Installation

## Phase

Phase 3 — Hermes Installation (generation install transaction / ADR-0006)

## Baseline / Commit

- Historical Phase 2 baseline: `phase-002-hermes-discovery-baseline-r1` → `5b89d22ed5e7ae3f4374a26f0fcda54bdabc6bf9`
- Historical R6 FAIL candidate: `b79ad3c5ba4024a12aec61d615096260d40ead4b` (`AUDIT-003R6` — FAIL)
- Generation implementation landing: `6117ef1596b2edfeae504d7e82e7e0cb1865852b`
- Branch inspected: `fix/phase3-audit-r6-remediation`
- Immutable R7 candidate: `3957b61f120172fd60f3e10cef54c005a950c48c`
- Tip commit subject: `fix: close Phase 3 generation dual-state-machine HIGH`

`git rev-parse HEAD` and `git rev-parse 3957b61` both resolved to `3957b61f120172fd60f3e10cef54c005a950c48c`. The working tree was clean except untracked `YORVA_0.3.0_x64_en-US.wxs`, which is not part of the candidate and was not modified by this audit.

No Phase 3 tag exists. This audit did not merge, freeze, tag, push, delete a branch, change implementation, or begin Phase 4. Historical `AUDIT-003` through `AUDIT-003R6` were not edited.

## Auditor

Fresh independent Phase 3 R7 review context, separate from the implementation agent. Review started from governing documents and the locked tree, not from an implementation completion summary.

## Date

2026-08-18 (Asia/Shanghai)

## Gate Decision

`FAIL`

## Executive Summary

R7 is a genuine new candidate on top of the Owner-approved generation architecture. The old ownership / promotion-journal machine is gone. Publish and activate re-walk the sealed tree against the stored manifest. A valid `active.json` wins over leftover `hermes-agent` on disk and on `PATH`. After first `COMMITTED`, user-authored `%LOCALAPPDATA%\hermes\hermes-agent\bin` is not stripped by path equality. Transaction create takes `install.lock`. Operation `FAILED`/`CANCELLED`/`INTERRUPTED` is suppressed while the linked transaction is `ACTIVATING` or `COMMITTED`.

That is not enough. The generation machine still has two independently persisted in-flight authorities that do not share a commit: the filesystem `InstallTransaction` and the SQLite Operation. When `SaveTransaction` replaces a `BUILDING`/`SEALED`/`PUBLISHED` record and then returns a post-replace durability error, the caller treats the install as failed and the Operation becomes `FAILED`, but the on-disk transaction remains nonterminal. Desktop Retry then allocates a second `CREATED` transaction under the lock without inspecting existing nonterminal records. `DecideRecovery` immediately returns `BLOCKED_UNSAFE` for two nonterminal transactions instead of recovering each once as architecture §8.1 requires. A documented durability failpoint plus the advertised Retry path therefore leaves the install gate permanently blocked.

`Execute` also ignores `NextState` on `ActionReconcileEnv`, so the required `FAILED` + `active.json` names this generation + incomplete environment row cannot roll forward. Local `go test ./...` and `go vet ./...` are green. Exact-commit GitHub Actions for this SHA was not observed and is recorded as PENDING. Phase 3 §22 makes any High or Medium finding gate-blocking.

## Verification Evidence

### Repository and candidate

- `git branch --show-current` → `fix/phase3-audit-r6-remediation`
- `git rev-parse HEAD` → `3957b61f120172fd60f3e10cef54c005a950c48c`
- `git rev-parse 3957b61` → `3957b61f120172fd60f3e10cef54c005a950c48c`
- `git status --porcelain` → `?? YORVA_0.3.0_x64_en-US.wxs` only
- `git tag --list 'phase-003*'` — no tag
- Historical ownership files (`ownership_promote.go`, journal, delta, marker) are absent from `services/node/internal/runtime/hermes/`

### Exact-commit GitHub Actions

PENDING. This auditor did not observe a green GitHub Actions run whose head SHA is `3957b61f120172fd60f3e10cef54c005a950c48c`. Historical R6 runs (`32108290020`, `32108290027`) belong to `b79ad3c` and are not reused. No run ID is invented.

### Local Go (required for this re-audit)

From `D:\workcode\myproject\services\node`:

- `go test ./...` — PASS (2026-08-18, this audit). Packages that ran: `internal/app`, `internal/applog`, `internal/bootstrap`, `internal/daemon`, `internal/domain/operation`, `internal/events`, `internal/install` (2.315s), `internal/persistence/sqlite`, `internal/runtime`, `internal/runtime/hermes` (4.367s), `internal/transport/httpapi`. `cmd/yorvad` and `internal/buildinfo` have no tests; `internal/domain/node` has no tests.
- `go vet ./...` — PASS (2026-08-18, this audit).

Not run in this audit (record, do not treat as PASS):

- `go test ./... -count=20`
- `go test -race ./...` (historically gcc/CGo-blocked on this host; not re-attempted as a fake PASS)
- `govulncheck ./...`
- frontend `pnpm` matrix
- Cargo / Tauri / MSI (explicitly out of scope for this re-audit)

### Targeted R7 risk checks

| Risk | Result | Evidence |
| --- | --- | --- |
| Operation `FAILED`/`CANCELLED`/`INTERRUPTED` while txn `ACTIVATING`/`COMMITTED` | Closed for the live worker/startup paths | `filesystemOwnsOperation` + `projectOperation` (`runtime_install.go:212-281`, `:295-301`, `:417-424`, `:538-555`); `execute.fail` (`runtime_install_run.go:96-104`, `:151-156`); tests `TestInterruptStaleLeavesActivatingOperationRunning`, `TestGetRepairsFailedOperationWhenTransactionCommitted`, `TestApplyEnvFailureAfterActivateDoesNotFailOperation` |
| `Manager.fail()` while `active.json` names this generation | Not called after pointer write; function itself has no guard | `manager.go:243-251`; `activateError` does not `fail()` (`manager.go:193-204`) |
| Publish/activate re-walk sealed tree | Closed | `VerifySealedTree` (`verify.go:16-31`); callers `publish.go:61`, `activate.go:16`, `activate.go:97`; `TestVerifySealedTreeRejectsPostSealMutation` proves metadata-only `VerifyPublishedGeneration` still passes after a mutated launcher |
| 002A3 leftover `hermes-agent` on PATH | Closed — pointer wins, not `AMBIGUOUS` | `active_resolve.go:41-45`; `candidates.go:62-80`, `:102-107`, `:303-321`; `TestActivePointerIgnoresLeftoverHermesAgentOnPATH` |
| D5 user-authored `hermes-agent\bin` | Closed — not removed by official-path equality | `RemovableBins` lists only other `generations/*/bin` (`environment.go:120-133`); `TestReconcileAfterCommitKeepsUserHermesAgentBin` |
| Transaction create under `install.lock` | Lock is taken; sibling nonterminal txns are not excluded | `runtime_install.go:133-156` |
| Recovery matrix vs `DecideRecovery`/`Execute` | Decide table matches most §8.1 rows; two rows do not | Immediate block on 2+ nonterminal (`recovery_decide.go:23-25`); `ActionReconcileEnv` ignores `NextState` (`recovery_execute.go:38-42`) |

### R6 finding closure

| R6 finding | R7 result | Evidence |
| --- | --- | --- |
| HIGH-R6-001 official `path` invalidates ownership | **CLOSED** | Official `-Stage path` is skipped; launchers copied before Seal (`build.go:72-103`, `build.go:139-160`) |
| HIGH-R6-002 allowed-prefix foreign executables | **CLOSED** | Ownership delta/journal removed; publish/activate re-walk the manifest |
| HIGH-R6-003 rename-before-journal crash | **CLOSED** as stated | Live `hermes-agent` rename / promotion journal are gone |
| MEDIUM-R6-001 ignored `SyncDir` | **CLOSED at the writer; successor defect remains** | `atomic.go:135-138` returns the post-replace dir-sync error; `TestDirSyncFailure/after replace` proves the new record is already on disk |
| MEDIUM-R6-002 path-manifest / A2 status docs | **CLOSED** | Phase 3 §10, `SECURITY.md`, `RUNTIME.md` describe generation seal/env; A2 header is R6/R7 pending |

## Dimension Results

| Dimension | Result | Notes |
|---|---|---|
| Scope | PASS | Generation transaction replaces the journal. No Phase 4 Instance/Profile, credentials, channels, Cloud, generic installer framework, or Hermes fork. |
| Correctness | FAIL | Same-session Retry after a persist-ahead-of-caller error creates a second nonterminal transaction. Recovery then blocks the install gate. |
| Architecture | FAIL | Filesystem transaction and SQLite Operation can disagree in a way that admits a second in-flight txn. `DecideRecovery` does not implement “recover each once”. |
| Security | PASS | Seal re-walk on publish/activate; leftover `hermes-agent` is not adopted or deleted; no new unauthenticated surface; secrets are not returned by ordinary read APIs. Same-user rewrite of `active.json` remains the documented residual (architecture §15.10). |
| Data and Persistence | FAIL | Post-replace durability error leaves a complete new transaction record while the caller fails the Operation. Retry persists another `CREATED` record. |
| Concurrency and Lifecycle | FAIL | Lock is held for create, but create does not exclude an existing nonterminal txn. `markTransactionFailed` is unlocked and best-effort. |
| Protocol and Compatibility | PASS | OpenAPI Operation/SSE shape is unchanged. `transaction_id` is a one-way projection and is not an activation source. Phase 2 DTO is unchanged. |
| Testing and Verification | FAIL | Green suite encodes immediate `BLOCKED_UNSAFE` for two nonterminal txns and does not execute the `FAILED`+active+incomplete-env row. Exact-commit CI for this SHA is PENDING. |
| Maintainability | PASS | `internal/install` is cohesive. Old promotion files were removed rather than wrapped. `RetryEligibleForPin` / `newOwnershipNonce` remain unused leftovers. |
| Documentation | PASS WITH OBSERVATION | Phase 3 §10, `SECURITY.md`, `RUNTIME.md`, `DATA_MODEL.md` match the generation contract. `AMENDMENT-002A3` and `AMENDMENT-003A4` headers still say `Implementation: NOT STARTED`. |
| Dependencies / Supply Chain | PASS | No new runtime dependency. Hermes/Node/npm pins were not reopened. |
| Operations / Diagnostics | FAIL | Two leftover nonterminal txn files make the daemon gate `BLOCKED_UNSAFE` until manual filesystem surgery. Health/discovery still start. |

## Findings

### Critical

None.

### High

#### HIGH-R7-001 — Retry after a persist-ahead-of-caller error creates a second nonterminal transaction; recovery then permanently blocks

Architecture §8.1: two or more nonterminal transactions must be recovered **each once**, and only remain `BLOCKED_UNSAFE` if more than one is still nonterminal. Architecture §15.5 / §13.1: after a successful atomic replace, a complete new record is the recovery truth even if the caller later sees a directory-sync error.

The writer matches the second rule on disk and violates it at the caller:

```135:138:services/node/internal/install/atomic.go
	if ops.SyncDir != nil {
		if err = ops.SyncDir(dir); err != nil {
			return err
		}
```

`TestDirSyncFailure/after replace` (`transaction_store_test.go:167-198`) asserts both facts: `SaveTransaction` returns an error **and** `LoadTransaction` already sees the new `BUILDING` record.

`Manager.persist` / `SealNew` treat that error as a hard failure and do **not** call `fail()`:

```94:99:services/node/internal/install/manager.go
	txn.State = StateBuilding
	txn.Step = "build"
	txn.UpdatedAt = m.now()
	if err := m.persist(&txn); err != nil {
		return txn, err
	}
```

The same pattern exists for the `SEALED` persist (`manager.go:150-152`) and the `PUBLISHED` persist (`publish.go:69-70`). `filesystemOwnsOperation` is true only for `ACTIVATING`/`COMMITTED` (`runtime_install.go:295-301`), so `execute.fail` marks the SQLite Operation `FAILED` while the filesystem transaction remains `BUILDING`, `SEALED`, or `PUBLISHED`.

Advertised Retry then allocates a new transaction under `install.lock` without listing existing nonterminal records:

```133:156:services/node/internal/app/runtime_install.go
	lock, err := install.AcquireLock(s.managedRoot)
	// ...
	txn, err := install.NewCreatedTransaction(op.TargetID, op.ID, op.SourcePin, version)
	if err := store.SaveTransaction(txn); err != nil {
		return install.InstallTransaction{}, err
	}
```

`DecideRecovery` does not recover each nonterminal once. It blocks immediately:

```23:25:services/node/internal/install/recovery_decide.go
	if len(nonterminal) > 1 {
		return blocked("multiple nonterminal transactions")
	}
```

Deterministic chain:

1. Start persists `CREATED` and a `PENDING` Operation.
2. `SealNew` replaces the record with `BUILDING` or `SEALED` (or `publish` replaces with `PUBLISHED`).
3. Post-replace `SyncDir` fails. Disk has the new nonterminal record. Caller fails the Operation.
4. Desktop Retry (new idempotency key) is allowed because the Operation is terminal.
5. `persistCreatedTransaction` writes a second `CREATED` transaction.
6. Daemon restart — including a normal Desktop close during the retry — observes two nonterminal transactions and sets `BLOCKED_UNSAFE`.
7. New install/prerequisite mutations are rejected. Neither leftover transaction is failed or published. Recovery will not self-heal.

`markTransactionFailed` (`runtime_install.go:159-178`) is not a backstop: it is unlocked, ignores `SaveTransaction` errors, and is not invoked on the `persist()` error path inside `SealNew`/`publish`.

Impact: the required failed-install Retry path, combined with a failpoint the architecture itself requires, can disable the install subsystem until someone deletes `control/transactions/txn_*.json` by hand. That is a current-phase lifecycle failure, not residual hardening.

Required closure: after a successful replace, treat a complete readable record as persisted and continue or roll forward; on any caller-visible persist error, fail **that** transaction under the lock before returning; refuse to create a new transaction while another nonterminal record exists; implement “recover each once” before `BLOCKED_UNSAFE`. Add a regression that injects post-replace dir-sync failure, issues Retry, restarts `Recover`, and asserts `READY` with at most one nonterminal record.

### Medium

#### MEDIUM-R7-001 — `Execute` cannot roll forward `FAILED` + `active.json` names this generation when environment is incomplete

Architecture §3.1 / §8.1: `FAILED` is illegal while `active.json` names this generation; observation must roll forward to `ACTIVATING`/`COMMITTED` and must never delete the active generation.

`DecideRecovery` emits the required decision:

```36:41:services/node/internal/install/recovery_decide.go
	if _, ok := failedButActive(obs, valid); ok {
		if obs.Environment.Complete() {
			return RecoveryDecision{Gate: GateReady, NextState: StateCommitted, Action: ActionCommit}
		}
		return RecoveryDecision{Gate: GateReconciling, NextState: StateActivating, Action: ActionReconcileEnv}
	}
```

The table test `FAILED but active env incomplete rolls to ACTIVATING` (`recovery_decide_test.go:273-280`) expects `NextState: StateActivating` and `Action: ActionReconcileEnv`.

`Execute` does not persist `NextState` for that action. It only calls `ReconcileEnvironment` on the still-`FAILED` record:

```38:42:services/node/internal/install/recovery_execute.go
	case ActionReconcileEnv:
		return invokeOnPrimary(store, obs, func(txn InstallTransaction) error {
			_, err := mgr.ReconcileEnvironment(ctx, txn)
			return err
		})
```

`ReconcileEnvironment` rejects any state other than `ACTIVATING` or `COMMITTED` (`environment_apply.go:43-45`). `Recover` then sets `RECONCILING` and returns the error (`recover.go:89-92`). The next startup repeats the same decision. There is no `Recover` integration test for this matrix row.

The env-complete half (`ActionCommit`) works. The env-incomplete half, which is the crash window after pointer write and before environment observation, does not. `Manager.fail()` still has no `active.json` guard (`manager.go:243-251`), so the illegal `FAILED`+active combination is not prevented at the writer either.

Required closure: persist `ACTIVATING` (or call `fail()`-in-reverse under the lock) before `ReconcileEnvironment`, or have `Execute` apply `NextState` before the action. Add a `Recover` test that starts from on-disk `FAILED` + valid pointer + incomplete env and ends `COMMITTED` or still `ACTIVATING` with `READY`/`RECONCILING`, never stuck on `ErrInvalidRecord`.

### Low

#### LOW-R7-001 — `List()` does not project Operation status from the transaction

`Get` repairs `FAILED`→`SUCCEEDED` when the txn is `COMMITTED` (`runtime_install.go:405-411`, `TestGetRepairsFailedOperationWhenTransactionCommitted`). `List` returns raw SQLite rows (`runtime_install.go:413-415`). Desktop reload that uses the list endpoint can show a stale terminal status until a later `GET`. Events remain notifications; `GET` is still the documented source of truth.

#### LOW-R7-002 — Amendment headers still say implementation has not started

`AMENDMENT-002A3` line 11 and `AMENDMENT-003A4` line 9 still read `Implementation: NOT STARTED` after batches 1–8 and this R7 candidate. Status-line drift only; the Phase Spec, Roadmap, and ADR describe the implemented contract.

#### LOW-R7-003 — Dead ownership-nonce helpers remain

`RetryEligibleForPin` still requires `OwnershipNonce` (`runtime_install.go:609-616`). `newOwnershipNonce` is still defined (`runtime_install.go:665-671`). Production `Start`/`execute` no longer generate or consult a nonce. Dead code is not retry authority.

### Info

#### INFO-R7-001 — Verification gaps that are not defects in the inspected code

- Exact-commit CI/MSI for `3957b61f120172fd60f3e10cef54c005a950c48c` was not seen.
- Local `-race` was not run.
- Frontend, Cargo, Tauri, and MSI were not run (out of this re-audit’s required commands).
- Untracked `YORVA_0.3.0_x64_en-US.wxs` must stay out of any future freeze commit.
- Same-user rewrite of `control/active.json` remains an accepted residual (architecture §15.10).

## Accepted Technical Debt

None. HIGH-R7-001 and MEDIUM-R7-001 affect current install retry/recovery. Phase 3 §22 makes any Critical, High or Medium finding gate-blocking. They cannot be relabeled debt to pass the gate.

## Required Fixes Before Next Phase

1. Close the persist-ahead-of-caller window: a complete replaced transaction record must be treated as persisted; the Operation must not be failed while that record is still nonterminal unless the transaction itself is marked `FAILED` under `install.lock`.
2. Refuse `CREATED` insertion while another nonterminal `txn_*.json` exists.
3. Implement architecture §8.1 “recover each once” before `BLOCKED_UNSAFE`.
4. Make `Execute` honor `NextState` for `FAILED`+active+incomplete env (MEDIUM-R7-001).
5. Add regressions for: post-replace dir-sync error → Retry → `Recover` → `READY`; `FAILED`+active+incomplete env → roll forward; two leftover nonterminal records do not permanently disable install if each is individually failable.
6. Produce a new immutable candidate. Record exact-commit CI/MSI for that SHA. Request a fresh independent `AUDIT-003R8`.

## Gate Rationale

The generation architecture removed the R6 journal/rename defects. Seal re-walk, 002A3 leftover-PATH, D5 user-authored `hermes-agent\bin`, and Operation projection for `ACTIVATING`/`COMMITTED` are implemented and tested. PASS still requires zero blocking High findings and a recovery story that matches ADR-0006.

HIGH-R7-001 is a source-level execution chain: the store test already proves the disk can be ahead of the caller; Start will then persist a second in-flight transaction; `DecideRecovery` will not unwind that pair. That satisfies `AUDIT_STANDARD.md` FAIL rules for an unreliable required workflow, unsafe lifecycle interaction, and insufficient recovery evidence. MEDIUM-R7-001 is independently gate-blocking under Phase 3 §22. Green local `go test`/`go vet` and the absence of the old double machine do not authorize lowering those rules. Exact-commit CI for this SHA is PENDING and would not override the source findings.

## Next Step

```text
Phase 3 Implementation: NOT ACCEPTED
AUDIT-003R7 Gate:       FAIL
Phase 3 status:         AUDIT / BLOCKED BY R7 FINDINGS
Merge / freeze / tag:   NOT AUTHORIZED
Feature branch delete:  NOT AUTHORIZED
Phase 4 planning:       BLOCKED
Phase 4 implementation: BLOCKED
```

Preserve `AUDIT-003` and `AUDIT-003R1` through `AUDIT-003R7` as immutable audit history. Do not begin Phase 4.

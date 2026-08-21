# YORVA Phase 3 — Generation Implementation Batches

> Historical implementation note (2026-08-21): Batch staging-build and publish-rename
> details are superseded by `ADR-0009` / `AMENDMENT-003A6`. New installation work
> builds and validates the candidate at its final generation path.

> Status: AUTHORIZED SPLIT / BATCH 8 COMPLETE  
> Date: 2026-08-18  
> Owner: D1–D5 approved 2026-08-18  
> Governing design: `PHASE-003-generation-installation-architecture.md`  
> Amendments: `002A3`, `003A4`  
> ADR: `ADR-0006`  
> Rule: one batch at a time; do not remove the old promotion machine until Batch 8 tests pass  
> This document does not authorize merge, freeze, tag, or Phase 4

`active.json` remains the only current-generation pointer in every batch. No shim, junction, `current` directory, or SQLite active flag may appear in any batch.

## Batch 0 — Governance close (this turn)

Delivered as documents only:

- `ADR-0006`
- `AMENDMENT-002A3`
- `AMENDMENT-003A4`
- this batch list
- architecture doc locked to D1–D5

No Go/Rust/TS/OpenAPI/SQLite code.

## Batch 1 — Pure decision model

**In:** `services/node/internal/install/recovery_decide.go` + tests only.  
**Not in:** filesystem writers, daemon hook, discovery change.

- types for `Observation`, `RecoveryDecision`, `TransactionState`
- `DecideRecovery` implementing architecture §8.1
- table tests for every matrix row, including D4 “do not delete unknown”
- idempotence: `D2` no-op or forward
- no SQLite imports

Exit: `go test ./internal/install` green; no other packages required.

Delivered: `types.go`, `recovery_decide.go`, `recovery_decide_test.go`. Pure decide only; old promotion machine untouched.

## Batch 2 — Atomic records, IDs, lock

**In:** `transaction_store.go`, `active_pointer.go` (read/write helpers), `lock_windows.go`.  
**Not in:** Hermes build, discovery, GC deletes.

- closed ids `txn_`, `gen_`
- atomic write + failpoints from architecture §13.1 (injectable ops)
- `LockFileEx` install.lock
- path containment under `%LOCALAPPDATA%\hermes`
- refuse absolute / escaping `generationRelativePath`

Exit: store/lock/pointer unit tests including dir-sync failure.

Delivered: `transaction.go`, `transaction_store.go`, `active_pointer.go`, `lock.go` + windows/other, closed IDs, path containment, injectable atomic writer. No Hermes build, discovery, or GC. Old promotion machine untouched.

## Batch 3 — Staging build and seal

**In:** Hermes `build.go` / `validate_install.go` called by a thin `InstallManager` BUILDING→SEALED.  
**Not in:** live `hermes-agent` rename, `active.json` replace, PATH writes, old `promoteCandidate` deletion.

- fresh `staging/txn_*` as `-InstallDir`
- `-HermesHome` official home
- copy `bin/hermes.exe` and `bin/hermes-acp.exe` before seal
- D3: config-templates warning only
- second manifest walk; no prefix ignore
- cancel/timeout failpoints §13.2
- old promotion path still used by production `Apply` until Batch 5

Exit: seal tests; existing Phase 3 install tests still pass on the old path.

Delivered: `install/seal.go`, `install/generation.go`, `install/manager.go` (CREATED→BUILDING→SEALED); Hermes `build.go` + `validate_install.go`. Production `Apply` still uses the old promotion path. No `active.json`, PATH, or live `hermes-agent` rename.

## Batch 4 — Publish and activate

**In:** staging→`generations/gen_*` rename; persist `ACTIVATING`; write `active.json`.  
**Not in:** registry reconcile; Phase 2 reader can land in the same batch or immediately after as 4b.

- predecessor generation never moves
- failpoints §13.3–13.4
- if `active.json` already names the new gen, roll forward

Exit: publish/activate tests; no PATH mutation yet.

Delivered: `publish.go` / `activate.go` / `verify.go`. Staging rename to `generations/gen_*`; persist `ACTIVATING` with `ActiveBefore*`; write `active.json`. Predecessor generation never moves. No PATH / `hermes-agent` rename.

## Batch 4b — Phase 2 reader (`002A3`)

**In:** `candidates.go` / detector only.

- valid pointer → one generation launcher
- leftover `hermes-agent` is not `AMBIGUOUS`
- invalid pointer → frozen enumeration
- no writes

Must not ship Batch 4 without 4b in the same release candidate (discovery would miss the install).

Delivered: `hermes/active_resolve.go` + `candidates.go` read-only `active.json` resolver. Valid pointer selects the generation launcher; leftover `hermes-agent` is not `AMBIGUOUS`; invalid pointer falls through to frozen enumeration; Detect does not write.

## Batch 5 — Environment reconcile (D5)

**In:** `environment.go` + daemon startup after recovery.

- pure `ComputeEnvironmentPlan`
- `HERMES_HOME` then PATH
- remove only proven YORVA-managed `hermes-agent\bin` or previous generation `\bin` after first COMMITTED
- never touch user-authored Hermes PATH
- failpoints §13.6
- production `Apply` switches to InstallManager here (feature flag or hard cut in this batch only)

Exit: env tests; ACTIVATING stays until values observed.

Delivered: `environment.go` (`ComputeEnvironmentPlan` + apply/read-back); `ReconcileEnvironment` commits only after HERMES_HOME and generation `\bin` prefix are observed; D5 removes only exact YORVA-managed stale bins after first COMMITTED; production `Apply` uses InstallManager (old promotion remains behind `legacyPromotion` for existing tests). Daemon startup reconciles from `active.json`. RecoverPromotions still present until Batch 6.

## Batch 6 — Recovery executor and daemon gate

**In:** `Execute(Decision)` + `daemon.go` gate `READY` / `RECONCILING` / `BLOCKED_UNSAFE`.

- no warn-and-continue
- reject new install/prereq while not READY
- health/discovery remain if safe
- drop `RecoverPromotions` Operation lookup

Exit: crash-injection §13.7; health still serves.

Delivered: `Observe` / `Execute` / `Recover`; daemon gate `READY` / `RECONCILING` / `BLOCKED_UNSAFE`. Recovery failure is not warn-and-continue: gate is set and HTTP still starts for health/discovery. New install/prereq rejected unless `READY`. `RecoverPromotions` Operation lookup no longer called from daemon.

## Batch 7 — Operation projection and retry

**In:** `runtime_install.go` / `_run.go`.

- create txn then Operation
- retry = new txn + new ids; no `PreviousRuntimeInstall` owner
- project stages from txn `Step`
- accepted-installation row remains cache
- failpoints §13.5

Exit: Desktop/SSE existing tests still pass; new retry tests do not use ownership nonce.

Delivered: Start persists InstallTransaction `CREATED` then Operation `PENDING` (`transaction_id` projection). SQLite insert failure marks the txn `FAILED` with no staging. Retry allocates a new txn/gen/staging. `execute` no longer reads `PreviousRuntimeInstall` or generates `ownership_nonce`. GET/InterruptStale project Operation status from the txn. Accepted-installation remains a cache.

## Batch 8 — GC (D4) and removal of the old machine

**In:** `gc.go`; delete old promotion/ownership files listed in the architecture §11.1.

Retention:

- active generation: never
- latest previous committed generation: keep
- latest failed lineage-proven staging/generation: keep at most one
- other proven unreferenced YORVA staging/failed/generations: eligible
- unknown dirs, `hermes-agent`, `.env`, `config.yaml`, `skills`, sessions: never

Old `ownership_promote.go` / journal / delta-as-ownership removed only after generation tests pass.

Exit: GC failpoints §13.8; full Phase 3 verification matrix; then independent `AUDIT-003R7`.

Delivered: `gc.go` D4 retention + failpoints; Recover/commit call Collect best-effort; GC failure does not change `COMMITTED`. Removed candidate/quarantine/journal/delta ownership machine (`ownership_promote.go` and related). Production install is generation-only. Historical `AUDIT-003`–`R6` remain FAIL; `AUDIT-003R7` remains PENDING for an independent audit agent.

## Stop conditions (any batch)

Stop and return to Owner if a batch needs:

- a second activation pointer;
- Phase 4 Instance/Profile;
- deleting legacy `hermes-agent` contents;
- changing `HERMES_HOME`;
- a generic transaction framework.

## Suggested first implementation prompt

After this document set is committed, the next coding agent should receive **only Batch 1**.

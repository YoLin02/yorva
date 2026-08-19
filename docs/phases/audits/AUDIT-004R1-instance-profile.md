# YORVA Phase 4 Independent Re-Audit R1 — Instance / Profile Management

## Phase

Phase 4 — Instance / Profile Management

## Baseline / Commit

- Phase 3 baseline: `phase-003-hermes-installation-baseline` → `2d1925813faeead6de93f66c12083687a85957d4`
- Branch inspected: `codex/phase4-instance-profile`
- Immutable AUDIT-004 FAIL candidate (HEAD, unchanged): `10a250983cd348c56776c63965a7dbe026efb141`
- Tip commit subject: `fix(phase4): use OS-absolute Hermes paths in create/delete tests`
- `origin/codex/phase4-instance-profile` → the same SHA
- `main` / `origin/main` → `d04b1fdc298f643f84d0c84a245595baae2e8994`
- No `phase-004*` tag exists

This is a **High-remediation re-audit**, not a freeze/tag review and not an immutable-candidate audit.

HEAD is still `10a250983cd348c56776c63965a7dbe026efb141`. The High remediations exist only as **uncommitted working-tree changes** on top of that SHA. They were reviewed as `HEAD + dirty tree`, not as a new commit.

Tracked modifications at audit time:

```text
 M apps/desktop/src/App.tsx
 M apps/desktop/src/operationRecovery.test.ts
 M apps/desktop/src/operationRecovery.ts
 M services/node/internal/app/hermes_profiles.go
 M services/node/internal/app/instance_delete_test.go
 M services/node/internal/app/instance_inventory.go
 M services/node/internal/app/instance_inventory_test.go
 M services/node/internal/daemon/daemon.go
 M services/node/internal/persistence/sqlite/operations.go
 M services/node/internal/persistence/sqlite/operations_test.go
 M services/node/internal/transport/httpapi/instances.go
 M services/node/internal/transport/httpapi/instances_test.go
```

Untracked remediation/test files included in this review:

```text
?? apps/desktop/src/App.instance-recovery.test.tsx
?? services/node/internal/app/hermes_profiles_test.go
?? services/node/internal/app/instance_recovery_test.go
```

Untracked and not judged as implementation: `.tmp-yorva-ui-ref/`, `YORVA_0.3.0_x64_en-US.wxs`, and the already-written `docs/phases/audits/AUDIT-004-instance-profile.md`.

This audit did not modify source, tests, specs, or git. It did not commit, merge, tag, push, or begin Phase 5.

## Auditor

Fresh independent Phase 4 R1 review context, separate from the implementation agent. Review started from the approved Phase Specs, `AUDIT_STANDARD.md`, `AUDIT-004`, and the current execution chain. Implementation completion summaries were not treated as evidence.

## Date

2026-08-19 (Asia/Shanghai)

## Gate Decision

`PASS WITH CONDITIONS`

## Executive Summary

The two AUDIT-004 High defects are **CLOSED** in the current uncommitted worktree.

`HIGH-004-001` no longer treats a failed authoritative query as confirmed absence. Delete authorization now requires `ListInstances` freshness `FRESH`. The delete worker requires a successful `reconcileLocked` before mutation and again before `SUCCEEDED`. Timeout, malformed output, and query failure become stable terminal errors and leave the Instance row `UNKNOWN`. Genuine disappearance after a successful query still converges to `MISSING`.

`HIGH-004-002` no longer leaves orphaned `instance.create` / `instance.delete` rows without an owner. Daemon start calls `RecoverStale`, which re-queries Hermes and terminalizes `PENDING`/`RUNNING` rows from that truth. Operation status is not treated as proof of profile existence. Desktop restart can rediscover active Instance Operations by listing `targetType=runtime-installation` for the current installation.

No new Critical or High finding was found on the remediated paths.

Freeze, tag, and merge remain blocked. AUDIT-004’s required-before-freeze Medium/Low items are still open, the remediations are uncommitted, there is no new immutable candidate, and this re-audit did not obtain exact-commit CI. That combination is why the gate is `PASS WITH CONDITIONS` rather than `PASS`.

## Verification Evidence

### Repository and tree

- `git rev-parse HEAD` → `10a250983cd348c56776c63965a7dbe026efb141`
- `git branch --show-current` → `codex/phase4-instance-profile`
- `git rev-parse origin/codex/phase4-instance-profile` → the same SHA
- `git rev-parse origin/main` → `d04b1fdc298f643f84d0c84a245595baae2e8994`
- `git log -1 --format` → `10a250983cd348c56776c63965a7dbe026efb141` / `fix(phase4): use OS-absolute Hermes paths in create/delete tests` / `2026-08-19 12:54:35 +0800`
- `git tag --list phase-004*` — empty
- `git diff --stat` against HEAD → 12 files, `561 insertions(+), 73 deletions(-)`
- `git diff --check` on the remediations → clean
- `git diff --check d04b1fd..HEAD` still reports three fixture EOF blank lines from the AUDIT-004 candidate

### Exact-commit GitHub Actions

Not rerun and not reusable for this gate. CI run [`32217465890`](https://github.com/YoLin02/yorva/actions/runs/32217465890) remains SUCCESS only for immutable SHA `10a2509`, which is the AUDIT-004 FAIL candidate and does not contain these remediations.

### Focused local verification (this audit)

From `D:\workcode\myproject\services\node` against the dirty worktree:

- `go test ./internal/app ./internal/persistence/sqlite ./internal/transport/httpapi ./internal/daemon -count=1` — PASS
- named High-path rerun — PASS:
  - `go test ./internal/app -count=1 -run "Delete|RecoverStale|ClassifyHermes"`
  - `go test ./internal/persistence/sqlite -count=1 -run "ListActiveInstance"`
  - `go test ./internal/transport/httpapi -count=1 -run "DeleteInstanceQuery"`
- `go vet ./internal/app ./internal/persistence/sqlite ./internal/transport/httpapi ./internal/daemon` — PASS, no diagnostics

From `D:\workcode\myproject` Desktop package:

- `pnpm test -- src/operationRecovery.test.ts src/App.instance-recovery.test.tsx` — PASS, 2 files / 7 tests

Not run in this re-audit (record, do not treat as PASS):

- full `go test ./...`
- `go test -race ./...`
- full Desktop `pnpm typecheck` / lint / full test / build matrix
- exact-commit GitHub Actions
- real destructive Hermes delete smoke (still forbidden against Owner data)

### Execution-chain review (not test-name review)

Delete authorization:

1. `StartDelete` still loads the YORVA row for identity, protection, and confirmation (`instance_inventory.go:544-556`).
2. It then calls `ListInstances` and refuses mutation unless `listed.Freshness == "FRESH"` (`instance_inventory.go:566-575`). `UNKNOWN` returns `instanceQueryError(listed.ErrorCode)` and does not create an Operation.
3. `ListInstances` still converts a Hermes list failure into HTTP-visible `freshness=UNKNOWN` with no Go error (`instance_inventory.go:131-175`). That is acceptable for read APIs; delete no longer infers absence from that payload.
4. `runDelete` calls `reconcileLocked` before `mutator.Delete`. Any reconcile error fails the Operation and returns (`instance_inventory.go:696-699`).
5. Absence is taken only from a successful reconcile result via `instanceAvailable` (`instance_inventory.go:300-307`, `:701-703`). `instanceAvailable` is true only for `AVAILABLE`.
6. After delete, a second successful `reconcileLocked` is required before `succeedDelete` (`instance_inventory.go:706-713`). Post-delete query failure fails the Operation and leaves the row `UNKNOWN`.
7. Adapter classification now preserves `context.DeadlineExceeded` and unrecognized output (`hermes_profiles.go:38-51`; `instance_inventory.go:263-274`).

Restart recovery:

1. `daemon.Run` constructs `InstanceInventory` and calls `RecoverStale` before serving HTTP (`daemon.go:110-113`).
2. `ListActiveInstanceOperations` selects only `instance.create` / `instance.delete` in `PENDING`/`RUNNING` (`operations.go:110-128`). Install/prerequisite rows and terminal rows are ignored.
3. `recoverOneLocked` queries Hermes first. Query failure terminalizes as `FAILED` with the classified code and marks inventory `UNKNOWN`. It does not succeed (`instance_inventory.go:762-766`, `:784-794`).
4. Create succeeds only if the named profile is authoritatively present; otherwise `OPERATION_INTERRUPTED`. Delete succeeds only if the named profile is authoritatively absent; a still-present profile fails and remains `AVAILABLE` (`instance_inventory.go:767-778`).
5. Repeated `RecoverStale` is a no-op once rows are terminal.
6. Desktop lists operations for `runtime-installation` + `runtimeInstallationId`, then follows the newest active create/delete (`App.tsx:322-338`; `operationRecovery.ts:16-22`).

## Dimension Results

| Dimension | Result | Notes |
|---|---|---|
| Scope | PASS | Remediation stays inside Phase 4 delete-truth and restart recovery. No Phase 5 lifecycle, credentials, channels, Cloud, or plugin work. |
| Correctness | PASS WITH CONDITIONS | Both High delete/restart defects are closed. Create still collapses command timeout to `INSTANCE_QUERY_FAILED`. Operation target is still the installation ID. |
| Architecture | PASS | React → local HTTP → application → domain/adapter is unchanged. Hermes parsing remains in the Hermes adapter. Recovery is application-owned, not UI-owned. |
| Security | PASS | No new bind, auth, secret, or shell surface. Profile commands remain absolute argv. Public Instance DTOs still omit `nativeId`. |
| Data and Persistence | PASS WITH CONDITIONS | Tombstones remain on uncertain delete/recovery. Orphaned active Instance Operations are now listed and terminalized. Target identity still does not use `instanceId`. |
| Concurrency and Lifecycle | PASS | Per-installation mutation lock remains. Daemon-start recovery removes the cross-restart deadlock. Recovery is idempotent and does not resume a destructive delete automatically. |
| Protocol and Compatibility | PASS WITH FINDINGS | New HTTP mappings for query/timeout/unrecognized delete preflight errors are stable. Create/delete Operations still target `runtime-installation`. |
| Testing and Verification | PASS WITH CONDITIONS | Required High-path regressions exist and passed locally. No new immutable SHA and no exact-commit CI. Create timeout assertion is still missing. |
| Maintainability | PASS | `instance_inventory.go` grew to about 832 lines. Size is still an observation, not a gate defect. No speculative workflow framework was added. |
| Documentation | PASS WITH FINDINGS | Specs still describe the approved contract. Completion evidence still names `fe15203` and “CI pending”. |
| Dependencies / Supply Chain | PASS | No new dependency. Not re-audited beyond the remediation diff. |
| Operations / Diagnostics | PASS WITH CONDITIONS | Orphaned Instance Operations are now explained as terminal `FAILED`/`SUCCEEDED` after Hermes truth. RecoverStale persist errors are only logged at daemon start. |

A dimension is not `N/A`. Impacted dimensions from AUDIT-004 (Correctness, Data, Concurrency, Testing, Operations, Protocol) were re-opened. Unchanged Security/Architecture/Dependency conclusions from AUDIT-004 remain valid because the remediations did not alter those surfaces.

## Findings

### Critical

None.

### High

None open.

#### HIGH-004-001 — Failed authoritative delete queries are treated as confirmed absence

**CLOSED** in the uncommitted worktree.

The AUDIT-004 execution chain is broken at every required point:

| Required closure | Current behavior |
|---|---|
| Never infer absence from `UNKNOWN` | `StartDelete` rejects non-`FRESH` lists. `runDelete` fails on `reconcileLocked` error instead of reading `profilePresent` after ignoring that error. |
| Successful pre-delete authoritative lookup before mutation | `StartDelete` requires `listed.Freshness == "FRESH"` before `CreateOperation`. Worker requires a successful reconcile before `mutator.Delete`. |
| Successful post-delete authoritative lookup before `SUCCEEDED` | Post-delete `reconcileLocked` error calls `failCreate`/`errorCodeFrom` and returns. |
| Timeout/malformed/query failure stay terminal and keep the tombstone | `classifyHermesProfileError` / `classifyProfileListError` preserve timeout and unrecognized output. Tests leave availability `UNKNOWN`, not `MISSING`. |
| Regression tests | Before mutation, after mutation, timeout, and genuine disappearance are covered in `instance_delete_test.go`. |

Direct path that previously succeeded from `UNKNOWN` (`presentErr == nil && !stillPresent`) is gone from `runDelete`.

#### HIGH-004-002 — Active Instance Operations have no restart owner or recovery transition

**CLOSED** in the uncommitted worktree.

| Required closure | Current behavior |
|---|---|
| Daemon-start reconcile/terminalize, then Hermes truth | `daemon.go:110-113` calls `RecoverStale`. Each orphan is locked, re-read, and decided by `queryAuthoritative` + `instanceAvailable`. |
| Idempotent; Operation status is not existence proof | Repeated recover returns no rows. A `RUNNING` delete of a still-present profile fails and remains `AVAILABLE`. An absent create fails as interrupted instead of succeeding. |
| Desktop can rediscover active Instance Operations | `App.tsx` lists `runtime-installation` operations after inventory load and follows newest active create/delete. |
| Tests | Application recover tests cover absent/present create, present/absent delete, query failure, timeout, repeated recover, and post-restart new mutation. Persistence lists only active Instance rows. Desktop tests recover running create/delete and ignore terminal rows. |

Policy choice is **terminalize**, not resume. That is allowed by AUDIT-004 (“restore/re-evaluate or terminalize”). A still-present profile is not deleted during recovery.

### Medium

#### MEDIUM-004-001 — Instance Operation target identity does not follow the approved contract

**OPEN.** Unchanged by the High remediations.

Create and delete still persist `targetType=runtime-installation` and `targetId=<installationId>` (`instance_inventory.go:351-355`, `:591-595`). Phase 4 §9 still says Operation targets use stable `instanceId`, and that `nativeId` is not the Operation target. The current target is not `nativeId`, so the original leak risk is still avoided, but the approved Instance target identity is still not implemented.

Desktop recovery now works *because* it queries that installation-scoped target (`App.tsx:322-325`). Delete-dialog recovery still binds the live Operation to an Instance by `item.name === recoveredDelete.message` (`App.tsx:333-334`), which is a name/message coincidence, not `instanceId`.

This remains a bounded contract/maintainability defect. It does not reopen HIGH-004-002. It should not be copied into Phase 5 lifecycle Operations.

Required closure is unchanged: preallocate or otherwise give create a stable `instanceId` target; make delete target the existing `instanceId`; keep a separate installation-scoped concurrency key if needed.

#### MEDIUM-004-002 — Instance command timeouts collapse to generic query failure

**OPEN**, narrowed.

Closed by this remediation:

- adapter `classifyHermesProfileError` preserves `context.DeadlineExceeded` as `ErrInstanceOperationTimedOut` (`hermes_profiles.go:45-47`);
- delete preflight and delete worker persist `INSTANCE_OPERATION_TIMED_OUT` (`instance_inventory.go:270-274`, `:698`, `:708`, `:716`);
- HTTP maps that code (`instances.go:198-199`);
- delete/recovery timeout tests exist.

Still open:

- `runCreate` still writes `ErrorInstanceQueryFailed` for every non-success create outcome, including a classified timeout (`instance_inventory.go:469-473`);
- there is still no create-path assertion that `INSTANCE_OPERATION_TIMED_OUT` is persisted.

Hermes `CreateProfile` already wraps deadline expiration with `context.DeadlineExceeded` (`profile_create.go:29-30`). The adapter now surfaces that distinction; the create worker discards it.

Required closure: persist `INSTANCE_OPERATION_TIMED_OUT` from the create worker and add the create timeout assertion demanded by AUDIT-004.

### Low

#### LOW-004-001 — Completion evidence and fixture whitespace need final synchronization

**OPEN.**

`docs/phases/PHASE-004-instance-profile.md` §24 still names `fe15203e71ac1f988bdc87a9e34ed4df886a9dfb` and says exact CI is pending. `git diff --check d04b1fd..HEAD` still reports trailing blank lines at EOF in:

- `services/node/internal/runtime/hermes/testdata/profile/list-default-and-named.txt`
- `services/node/internal/runtime/hermes/testdata/profile/list-default-only.txt`
- `services/node/internal/runtime/hermes/testdata/profile/list-duplicate-names.txt`

The remediations did not touch those files.

#### LOW-004R1-001 — Daemon-start RecoverStale failure is warn-and-continue, and there is no `daemon.Run` wiring test

`RecoverStale` persist/list errors abort the remaining orphans and are logged as `Warn` (`daemon.go:111-113`). Query failures themselves are terminalized and do not return an error. A true persist failure can therefore leave a later orphan active while HTTP starts.

There is also no daemon-level test that `Run` invokes `RecoverStale`. The production call is three obvious lines and the policy is tested in `app`. This is not enough to reopen HIGH-004-002.

### Info

#### INFO-004R1-001 — This review is a dirty-tree High re-audit, not a freeze candidate

HEAD remains the AUDIT-004 FAIL SHA. Remediations are uncommitted. Exact-commit CI cannot exist until a new SHA exists. Local race and the full verification matrix were not rerun here.

#### INFO-004R1-002 — `StartCreate` still proceeds when list freshness is `UNKNOWN`

`StartCreate` uses `ListInstances` but does not require `FRESH` before creating an Operation (`instance_inventory.go:319-340`). That is not the delete-success-from-UNKNOWN defect. Create success still requires an `AVAILABLE` row after reconcile (`instance_inventory.go:460-468`). Recorded only so it is not mistaken for an incomplete HIGH-004-001 fix.

#### INFO-004-001 — Verification environment observations

Still valid: local Go race is historically CGO-blocked on this host; isolated real-Hermes delete smoke was not run against Owner data.

## Accepted Technical Debt

None newly accepted as closed-and-deferred forever.

The remaining Medium/Low items are **conditions**, not accepted Phase 4 debt. They were already listed by AUDIT-004 as required before freeze. This re-audit does not convert them into Owner-accepted residual debt.

The `instance_inventory.go` file-size observation remains non-blocking.

## Required Fixes Before Freeze

1. Keep HIGH-004-001 and HIGH-004-002 closed on the committed candidate. Do not regress the freshness/error consume path or daemon-start `RecoverStale`.
2. Close MEDIUM-004-001: Instance Operation targets use stable `instanceId`; retain a separate installation-scoped concurrency key if needed.
3. Close MEDIUM-004-002 remainder: create worker persists `INSTANCE_OPERATION_TIMED_OUT` and has a timeout assertion.
4. Close LOW-004-001: synchronize Phase Spec completion/CI evidence to the new candidate; remove only the three reported fixture EOF blank lines.
5. Commit the remediations (and any Medium/Low fixes) as a new immutable candidate. Do not freeze `10a2509`.
6. Obtain exact-commit CI PASS on that new SHA.
7. Request an independent freeze audit of the new candidate. This R1 report is not freeze authorization.

## Gate Rationale

`AUDIT_STANDARD.md` requires FAIL for an unresolved blocking High, and allows `PASS WITH CONDITIONS` only when Highs are closed and remaining findings are bounded with an explicit resolution trigger.

Independent execution-chain review plus focused tests show HIGH-004-001 and HIGH-004-002 are closed in the current dirty tree. No new High was found.

An unconditional `PASS` would be false: the remediations are not a commit, AUDIT-004’s Medium/Low freeze requirements remain, and there is no exact-commit CI for the remediated tree. Those leftovers do not re-break delete truth or restart ownership, so they do not force FAIL. They do block merge/freeze/tag and Phase 5.

## Next Step

```text
Phase 4 Implementation: High remediations present in worktree, NOT COMMITTED
Verification: focused High-path PASS; full matrix / exact-commit CI NOT RUN
AUDIT-004: FAIL (historical, SHA 10a2509)
AUDIT-004R1: PASS WITH CONDITIONS
HIGH-004-001: CLOSED
HIGH-004-002: CLOSED
MEDIUM-004-001: OPEN
MEDIUM-004-002: OPEN (create path only)
LOW-004-001: OPEN
Merge to main: BLOCKED
Phase 4 freeze/tag: BLOCKED
Phase 5: BLOCKED
```

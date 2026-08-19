# YORVA Phase 4 Independent Freeze Audit R2 — Instance / Profile Management

## Phase

Phase 4 — Instance / Profile Management

## Baseline / Commit

- Phase 3 baseline: `phase-003-hermes-installation-baseline` → `2d1925813faeead6de93f66c12083687a85957d4`
- Branch inspected: `codex/phase4-instance-profile`
- Immutable AUDIT-004 FAIL candidate: `10a250983cd348c56776c63965a7dbe026efb141`
- Immutable AUDIT-004R1 source: the same SHA plus the uncommitted High remediations that are now this commit
- **Immutable freeze candidate (this audit):** `8397dd4785a98a750f866ee191c0ca9026efe96e`
- Tip commit subject: `fix(phase4): fail closed on uncertain delete and recover instance operations`
- `origin/codex/phase4-instance-profile` → the same SHA
- `main` / `origin/main` → `d04b1fdc298f643f84d0c84a245595baae2e8994`
- No `phase-004*` tag exists

This is the **final freeze audit** of the committed High remediations. It is not an implementation pass and does not itself merge, tag, push, or begin Phase 5.

`git rev-parse HEAD` is `8397dd4785a98a750f866ee191c0ca9026efe96e`. `10a2509` is an ancestor. The tracked worktree is clean. Untracked `.tmp-yorva-ui-ref/` and `YORVA_0.3.0_x64_en-US.wxs` are outside the candidate and were not judged.

## Auditor

Fresh independent Phase 4 R2 freeze-audit context, separate from the implementation agent and from AUDIT-004 / AUDIT-004R1. Review started from the approved Phase Specs, `AUDIT_STANDARD.md`, `PHASE_GOVERNANCE.md`, the two prior audits, and the execution chain on this exact SHA. Implementation completion summaries were not treated as evidence.

## Date

2026-08-19 (Asia/Shanghai)

## Gate Decision

`PASS WITH CONDITIONS`

## Executive Summary

HIGH-004-001 and HIGH-004-002 remain **CLOSED** on immutable SHA `8397dd4785a98a750f866ee191c0ca9026efe96e`.

Delete no longer infers absence from `UNKNOWN`. Authorization and `SUCCEEDED` both require a successful authoritative Hermes query. Timeout, malformed output, and query failure are stable terminal errors and leave the Instance row `UNKNOWN`. The tombstone is not deleted. Genuine disappearance after a successful query still converges to `MISSING`.

Daemon start owns orphaned `instance.create` / `instance.delete` rows through `RecoverStale`. Recovery is idempotent, decides from a fresh Hermes query, and never treats Operation status as existence proof. A still-present profile is not deleted during recovery. Desktop rediscovers active Instance Operations by listing `targetType=runtime-installation` for the current installation.

No new Critical or High finding was found on this SHA. Exact-commit CI run `32226244512` is SUCCESS for this head, including `go test -race ./...`.

The three AUDIT-004R1 leftovers remain open and are **bounded contract / UX / hygiene**, not delete-truth or restart-deadlock:

- MEDIUM-004-001 — Operation target is still the installation ID, not `instanceId`
- MEDIUM-004-002 — create worker still collapses timeout to `INSTANCE_QUERY_FAILED`
- LOW-004-001 — Spec §24 still names `fe15203` / “CI pending”; three Profile list fixtures still have EOF blanks

Under `AUDIT_STANDARD.md`, those leftovers do not force FAIL. They are Owner-accepted conditions. Owner-authorized merge to `main` and Phase 4 freeze are **allowed** on this SHA. Phase 5 implementation remains blocked until a Phase 5 Spec is written from the frozen baseline, and must not copy the remaining Operation-target or timeout-collapse defects into lifecycle work.

## Verification Evidence

### Repository and candidate

- `git rev-parse HEAD` → `8397dd4785a98a750f866ee191c0ca9026efe96e`
- `git branch --show-current` → `codex/phase4-instance-profile`
- `git rev-parse origin/codex/phase4-instance-profile` → the same SHA
- `git rev-parse main` / `origin/main` → `d04b1fdc298f643f84d0c84a245595baae2e8994`
- `git log -1 --format` → `8397dd4785a98a750f866ee191c0ca9026efe96e` / `fix(phase4): fail closed on uncertain delete and recover instance operations` / AuthorDate `2026-08-19 15:05:29 +0800`
- `git merge-base --is-ancestor 10a250983cd348c56776c63965a7dbe026efb141 HEAD` → yes
- `git tag --list phase-004*` — empty
- `git status --porcelain=v1` → only `?? .tmp-yorva-ui-ref/` and `?? YORVA_0.3.0_x64_en-US.wxs`
- `git diff --check HEAD` — clean
- `git diff --check d04b1fd..HEAD` still reports:
  - `docs/phases/audits/AUDIT-004-instance-profile.md:232` new blank line at EOF (historical audit file)
  - `services/node/internal/runtime/hermes/testdata/profile/list-default-and-named.txt:7`
  - `services/node/internal/runtime/hermes/testdata/profile/list-default-only.txt:5`
  - `services/node/internal/runtime/hermes/testdata/profile/list-duplicate-names.txt:7`

This commit is the AUDIT-004R1 dirty tree plus the two historical audit files. It does not contain Phase 5 lifecycle, credentials, channels, Cloud, or plugin work.

### Exact-commit GitHub Actions

Re-queried GitHub API run [`32226244512`](https://github.com/YoLin02/yorva/actions/runs/32226244512) on 2026-08-19:

- `head_sha` = `8397dd4785a98a750f866ee191c0ca9026efe96e`
- `head_branch` = `codex/phase4-instance-profile`
- `event` = `push`, `run_attempt` = 1
- `conclusion` = **success**
- Jobs (all `completed` / `success`, all `head_sha` identical):
  - **Web and API contract** (`95986404000`) — `pnpm audit`, `api:lint`, `api:generate` + generated-schema drift check, typecheck, lint, test, build
  - **Go Node** (`95986403839`) — `go test -race ./...`, `go vet ./...`, `govulncheck ./...`, `go build ./cmd/yorvad`
  - **Windows Desktop native shell** (`95986404078`) — sidecar, Windows lifecycle smoke, MSI inspect tests, cargo fmt/test/audit/clippy/check, Tauri `--no-bundle`

Historical AUDIT-004 CI `32217465890` remains SUCCESS only for FAIL candidate `10a2509` and is not reused as freeze evidence.

### Focused local verification (this audit)

From `D:\workcode\myproject\services\node` against `8397dd4`:

- `go test ./internal/app ./internal/persistence/sqlite ./internal/transport/httpapi ./internal/daemon -count=1` — PASS
  - `internal/app` 16.342s
  - `internal/persistence/sqlite` 0.542s
  - `internal/transport/httpapi` 0.260s
  - `internal/daemon` 0.212s
- named High-path rerun — PASS:
  - `go test ./internal/app -count=1 -run "Delete|RecoverStale|ClassifyHermes"`
  - `go test ./internal/persistence/sqlite -count=1 -run "ListActiveInstance"`
  - `go test ./internal/transport/httpapi -count=1 -run "DeleteInstanceQuery"`
- `go vet ./internal/app ./internal/persistence/sqlite ./internal/transport/httpapi ./internal/daemon` — PASS, no diagnostics

From `D:\workcode\myproject` Desktop package:

- `pnpm test -- src/operationRecovery.test.ts src/App.instance-recovery.test.tsx` — PASS, 2 files / 7 tests

Not run in this freeze audit (record, do not treat as a local PASS; exact-commit CI covers the matrix):

- full local `go test ./...`
- local `go test -race ./...` (historically CGO-blocked on this host; CI race is SUCCESS)
- full Desktop typecheck / lint / full test / build
- real destructive Hermes delete smoke (still forbidden against Owner data)

### Execution-chain review (not test-name review)

Delete authorization and success (HIGH-004-001):

1. `StartDelete` still loads the YORVA row only for identity, protection, and typed confirmation (`instance_inventory.go:544-556`). That cache cannot authorize mutation by itself.
2. It then calls `ListInstances` and refuses unless `listed.Freshness == "FRESH"` (`instance_inventory.go:566-575`). `UNKNOWN` returns `instanceQueryError(listed.ErrorCode)` and does not create an Operation.
3. `ListInstances` still converts a Hermes list failure into HTTP-visible `freshness=UNKNOWN` with no Go error (`instance_inventory.go:131-175`). That remains correct for read APIs. Delete no longer consumes that payload as absence.
4. `runDelete` requires a successful `reconcileLocked` before `mutator.Delete` (`instance_inventory.go:696-699`). Any reconcile error fails the Operation and returns.
5. Absence is taken only from a successful reconcile via `instanceAvailable` (`instance_inventory.go:300-307`, `:701-703`). `instanceAvailable` is true only for `AVAILABLE`. `reconcileLocked` returns `Freshness: "FRESH"` or an error; it never returns a usable UNKNOWN snapshot.
6. After delete, a second successful `reconcileLocked` is required before `succeedDelete` (`instance_inventory.go:706-713`). Post-delete query failure fails the Operation and leaves the row `UNKNOWN` via `MarkInstancesUnknown`.
7. Adapter classification preserves `context.DeadlineExceeded` and unrecognized output (`hermes_profiles.go:38-51`; `instance_inventory.go:263-274`). HTTP maps those to `INSTANCE_OPERATION_TIMED_OUT` / `INSTANCE_OUTPUT_UNRECOGNIZED` / `INSTANCE_QUERY_FAILED` (`instances.go:196-201`).
8. `ApplyInstanceSnapshot` marks absent rows `MISSING` and never deletes them (`instances.go:124-136`). `succeedDelete` updates only the Operation row (`instance_inventory.go:722-729`).

Restart recovery (HIGH-004-002):

1. `daemon.Run` constructs `InstanceInventory` and calls `RecoverStale` before serving HTTP (`daemon.go:110-113`).
2. `ListActiveInstanceOperations` selects only `instance.create` / `instance.delete` in `PENDING`/`RUNNING` (`operations.go:110-128`). Install/prerequisite rows and terminal rows are ignored.
3. `recoverOneLocked` queries Hermes first. Query failure terminalizes as `FAILED` with the classified code and marks inventory `UNKNOWN`. It does not succeed (`instance_inventory.go:762-766`, `:784-794`).
4. Create succeeds only if the named profile is authoritatively present; otherwise `OPERATION_INTERRUPTED`. Delete succeeds only if the named profile is authoritatively absent; a still-present profile fails and remains `AVAILABLE` (`instance_inventory.go:767-778`).
5. Repeated `RecoverStale` is a no-op once rows are terminal. Post-recovery create/delete are unblocked (`instance_recovery_test.go`).
6. Policy is **terminalize**, not resume. A still-present profile is not deleted during recovery. Profile list commands remain bounded by the adapter `commandTimeout` (3s) even though daemon-start uses the process context (`command.go:13-44`).
7. Desktop lists operations for `runtime-installation` + `runtimeInstallationId`, then follows the newest active create/delete (`App.tsx:322-338`; `operationRecovery.ts:16-22`). Terminal rows are not restored as active.

Create path (not a High regression):

- `runCreate` still ignores the `reconcileLocked` error and then reads `profilePresent` (`instance_inventory.go:458-473`). Success still requires an `AVAILABLE` row. A failed reconcile marks rows `UNKNOWN`, so create cannot succeed from UNKNOWN. The remaining defect is only that every non-success create outcome, including a classified timeout, is persisted as `INSTANCE_QUERY_FAILED`.

## Dimension Results

| Dimension | Result | Notes |
|---|---|---|
| Scope | PASS | Candidate stays inside Phase 4 Instance/Profile management. No Start/Stop/Restart implementation, credentials, channels, Cloud, plugin system, or Phase 3 generation/`active.json` mutation. Desktop still only explains `lifecycleUnavailable`. |
| Correctness | PASS WITH CONDITIONS | HIGH-004-001 and HIGH-004-002 remain closed on this SHA. Create still collapses command timeout. Operation target is still the installation ID. |
| Architecture | PASS | React → local HTTP → application → domain/adapter is unchanged. Hermes parsing remains in the Hermes adapter. Recovery is application-owned, not UI-owned. Public Instance DTOs still omit `nativeId`. |
| Security | PASS | No new bind, auth, secret, or shell surface. Profile commands remain absolute argv with restricted environment and bounded output. Delete of `default` remains server-rejected before process start. |
| Data and Persistence | PASS WITH CONDITIONS | SQLite remains a cache. Tombstones remain on uncertain delete/recovery and on confirmed absence. Orphaned active Instance Operations are listed and terminalized. Target identity still does not use `instanceId`. |
| Concurrency and Lifecycle | PASS | Per-installation mutation lock remains. Daemon-start recovery removes the cross-restart deadlock. Recovery is idempotent and does not resume a destructive delete automatically. |
| Protocol and Compatibility | PASS WITH FINDINGS | Closed create/delete bodies, auth, capability `lifecycle: false`, and stable query/timeout/unrecognized mappings hold. Create/delete Operations still target `runtime-installation`. |
| Testing and Verification | PASS WITH CONDITIONS | Required High-path regressions exist, passed locally, and are in the immutable SHA. Exact-commit CI including race is SUCCESS. Create timeout assertion is still missing. Isolated real-Hermes delete smoke was not run. |
| Maintainability | PASS | `instance_inventory.go` is 833 lines. Size is an observation, not a gate defect. No speculative workflow framework was added. |
| Documentation | PASS WITH FINDINGS | English/Chinese Specs still agree on the approved contract. Completion evidence still names `fe15203` and “CI pending”. That is freeze-commit hygiene, not a correctness defect. |
| Dependencies / Supply Chain | PASS | This commit adds no dependency. CI `pnpm audit --audit-level low` and `govulncheck` succeeded. Cargo retains the historical allowed GTK/unic warnings. |
| Operations / Diagnostics | PASS WITH CONDITIONS | Orphaned Instance Operations become terminal `FAILED`/`SUCCEEDED` from Hermes truth. `RecoverStale` persist/list errors are still warn-and-continue at daemon start. |

A dimension is not `N/A`. Impacted dimensions from AUDIT-004 (Correctness, Data, Concurrency, Testing, Operations, Protocol) were re-opened on this SHA. Unchanged Security/Architecture/Dependency conclusions from AUDIT-004 remain valid because the remediations did not alter those surfaces.

## Findings

### Critical

None.

### High

None open.

#### HIGH-004-001 — Failed authoritative delete queries are treated as confirmed absence

**CLOSED** on `8397dd4785a98a750f866ee191c0ca9026efe96e`.

| Required closure | Current behavior on this SHA |
|---|---|
| Never infer absence from `UNKNOWN` | `StartDelete` rejects non-`FRESH` lists. `runDelete` fails on `reconcileLocked` error instead of reading presence after ignoring that error. |
| Successful pre-delete authoritative lookup before mutation | `StartDelete` requires `listed.Freshness == "FRESH"` before `CreateOperation`. Worker requires a successful reconcile before `mutator.Delete`. |
| Successful post-delete authoritative lookup before `SUCCEEDED` | Post-delete `reconcileLocked` error calls `failCreate`/`errorCodeFrom` and returns. |
| Timeout/malformed/query failure stay terminal and keep the tombstone | `classifyHermesProfileError` / `classifyProfileListError` preserve timeout and unrecognized output. Tests leave availability `UNKNOWN`, not `MISSING`. Snapshot update never deletes the row. |
| Regression tests | Before mutation, after mutation, timeout, and genuine disappearance are covered in `instance_delete_test.go`. HTTP maps query failure in `TestDeleteInstanceQueryFailureIsStableError`. |

The AUDIT-004 path `presentErr == nil && !stillPresent` after an ignored reconcile error is gone from `runDelete`.

#### HIGH-004-002 — Active Instance Operations have no restart owner or recovery transition

**CLOSED** on `8397dd4785a98a750f866ee191c0ca9026efe96e`.

| Required closure | Current behavior on this SHA |
|---|---|
| Daemon-start reconcile/terminalize, then Hermes truth | `daemon.go:110-113` calls `RecoverStale`. Each orphan is locked, re-read, and decided by `queryAuthoritative` + `instanceAvailable`. |
| Idempotent; Operation status is not existence proof | Repeated recover returns no rows. A `RUNNING` delete of a still-present profile fails and remains `AVAILABLE`. An absent create fails as interrupted instead of succeeding. |
| Desktop can rediscover active Instance Operations | `App.tsx` lists `runtime-installation` operations after inventory load and follows newest active create/delete. |
| Tests | Application recover tests cover absent/present create, present/absent delete, query failure, timeout, repeated recover, and post-restart new mutation. Persistence lists only active Instance rows. Desktop tests recover running create/delete and ignore terminal rows. |

Policy choice remains **terminalize**, not resume. That is allowed by AUDIT-004 (“restore/re-evaluate or terminalize”). A still-present profile is not deleted during recovery.

### Medium

#### MEDIUM-004-001 — Instance Operation target identity does not follow the approved contract

**OPEN.** Unchanged on this SHA. Owner-accepted condition for freeze.

Create and delete still persist `targetType=runtime-installation` and `targetId=<installationId>` (`instance_inventory.go:351-355`, `:591-595`). Phase 4 §9 still says Operation targets use stable `instanceId`, and that `nativeId` is not the Operation target. The current target is not `nativeId`, so the original leak risk is still avoided, but the approved Instance target identity is still not implemented.

Desktop recovery now works *because* it queries that installation-scoped target (`App.tsx:322-325`). Delete-dialog recovery still binds the live Operation to an Instance by `item.name === recoveredDelete.message` (`App.tsx:333-334`), which is a name/message coincidence, not `instanceId`. Phase 4 uniqueness `(runtime_installation_id, native_id)` makes that coincidence work today.

This remains a bounded contract/maintainability defect. It does not reopen HIGH-004-002. It must not be copied into Phase 5 lifecycle Operations.

Required closure: preallocate or otherwise give create a stable `instanceId` target; make delete target the existing `instanceId`; keep a separate installation-scoped concurrency key if needed.

#### MEDIUM-004-002 — Instance command timeouts collapse to generic query failure

**OPEN**, still narrowed to the create worker. Owner-accepted condition for freeze.

Still closed by the remediations:

- adapter `classifyHermesProfileError` preserves `context.DeadlineExceeded` as `ErrInstanceOperationTimedOut` (`hermes_profiles.go:45-47`);
- Hermes `CreateProfile` / `ListProfiles` / `DeleteProfile` wrap deadline expiration with `context.DeadlineExceeded` while also wrapping `errProfileQueryFailed`; the adapter unwraps the deadline;
- delete preflight and delete worker persist `INSTANCE_OPERATION_TIMED_OUT` (`instance_inventory.go:270-274`, `:698`, `:708`, `:716`);
- HTTP maps that code (`instances.go:198-199`);
- delete/recovery timeout tests exist.

Still open:

- `runCreate` still writes `ErrorInstanceQueryFailed` for every non-success create outcome, including a classified timeout (`instance_inventory.go:469-473`);
- there is still no create-path assertion that `INSTANCE_OPERATION_TIMED_OUT` is persisted.

This is UX/diagnostics, not delete-truth. Create success still cannot be inferred from UNKNOWN.

Required closure: persist `INSTANCE_OPERATION_TIMED_OUT` from the create worker and add the create timeout assertion demanded by AUDIT-004.

### Low

#### LOW-004-001 — Completion evidence and fixture whitespace need final synchronization

**OPEN.** Owner-accepted hygiene for the freeze commit.

Both language Specs §24 still name `fe15203e71ac1f988bdc87a9e34ed4df886a9dfb` and say exact CI is pending / `AUDIT-004: PENDING`. `git diff --check d04b1fd..HEAD` still reports trailing blank lines at EOF in the three Profile list fixtures listed above. This commit did not touch those files.

The freeze commit may synchronize §24 to `8397dd4` / CI `32226244512` / this gate and remove only the three reported fixture EOF blanks. That is documentation/whitespace, not a reason to reopen the Highs.

#### LOW-004R1-001 — Daemon-start RecoverStale failure is warn-and-continue, and there is no `daemon.Run` wiring test

**OPEN.** Non-blocking.

`RecoverStale` persist/list errors abort the remaining orphans and are logged as `Warn` (`daemon.go:111-113`). Query failures themselves are terminalized and do not return an error. A true persist failure can therefore leave a later orphan active while HTTP starts.

There is also no daemon-level test that `Run` invokes `RecoverStale`. The production call is three obvious lines and the policy is tested in `app`. This is not enough to reopen HIGH-004-002.

### Info

#### INFO-004R1-002 — `StartCreate` still proceeds when list freshness is `UNKNOWN`

Still valid. `StartCreate` uses `ListInstances` but does not require `FRESH` before creating an Operation (`instance_inventory.go:319-340`). Create success still requires an `AVAILABLE` row after reconcile (`instance_inventory.go:460-468`). Recorded only so it is not mistaken for an incomplete HIGH-004-001 fix.

#### INFO-004-001 — Verification environment observations

Still valid: local Go race is historically CGO-blocked on this host; isolated real-Hermes delete smoke was not run against Owner data. Exact-commit CI race and Windows native/Tauri no-bundle are SUCCESS for this SHA.

#### INFO-004R2-001 — This review is the freeze candidate that AUDIT-004R1 lacked

AUDIT-004R1 closed the Highs only in a dirty tree on top of `10a2509` and therefore blocked merge/freeze. Those remediations are now commit `8397dd4` with exact-commit CI SUCCESS. This report is the freeze verification R1 required.

## Accepted Technical Debt

The remaining Medium/Low items are **Owner-accepted conditions** for this gate. They do not weaken delete-truth, restart ownership, secret handling, or architecture direction.

| ID | Severity | Why accepted | Impact | Owner | Target / trigger |
|---|---|---|---|---|---|
| MEDIUM-004-001 | Medium | Installation-scoped target is internally consistent, does not leak `nativeId`, and is what Desktop recovery queries. Not delete-truth or restart-deadlock. | Phase 5 lifecycle Operations would inherit the wrong target identity if copied. Delete-dialog recovery remains name/message coupling. | Repository owner | Before or inside the Phase 5 Spec; do not copy this target shape into start/stop/restart Operations. |
| MEDIUM-004-002 | Medium | Delete/recovery already persist `INSTANCE_OPERATION_TIMED_OUT`. Create still fail-closes; only the error code is wrong. | Create timeout UI cannot distinguish timeout from query failure. | Repository owner | Next Instance mutation/UX pass, and before Phase 5 lifecycle timeout UX depends on the code. |
| LOW-004-001 | Low | Historical completion block and fixture EOF blanks do not change runtime behavior. | Spec §24 is stale relative to the freeze SHA. | Repository owner | Apply in the Phase 4 freeze commit. |
| LOW-004R1-001 | Low | Persist failure at daemon start is a storage emergency, not the orphan-without-owner defect. | A later orphan could remain active if SQLite write fails mid-recover. | Repository owner | If daemon-start recovery is extended, add fail-closed persist handling and a wiring test. |

The `instance_inventory.go` file-size observation remains non-blocking and is not accepted as a required split.

## Required Fixes Before Freeze

None that block the gate. Highs are closed on this SHA, exact-commit CI is SUCCESS, and the remaining findings are the Owner-accepted conditions above.

Permitted freeze-commit hygiene (does not require a new High re-audit if it touches only these):

1. Synchronize both language Specs §24 to candidate `8397dd4785a98a750f866ee191c0ca9026efe96e`, CI run `32226244512`, and this gate.
2. Remove only the three reported Profile list fixture EOF blanks.

Do not regress the freshness/error consume path or daemon-start `RecoverStale` in that freeze commit.

## Required Fixes Before Phase 5

1. Write a Phase 5 Spec from this frozen baseline. Do not begin lifecycle implementation from `ROADMAP.md` alone.
2. Do not copy MEDIUM-004-001 into start/stop/restart Operations. Phase 5 must specify `instanceId` targeting (or an explicit ADR if installation-scoped targeting is to become the contract).
3. Do not copy MEDIUM-004-002 into lifecycle workers. Preserve `INSTANCE_OPERATION_TIMED_OUT` end-to-end, including create if it remains in-scope.
4. Keep HIGH-004-001 / HIGH-004-002 closures as invariants: never infer absence from `UNKNOWN`; never treat Operation status as Hermes existence.

## Gate Rationale

`AUDIT_STANDARD.md` requires FAIL for an unresolved blocking High, and allows `PASS WITH CONDITIONS` only when Highs are closed and remaining findings are bounded, owned, and explicitly accepted.

Independent execution-chain review of this exact SHA, plus focused High-path tests and exact-commit CI `32226244512`, show HIGH-004-001 and HIGH-004-002 remain closed. No new High was found.

An unconditional `PASS` would be false: MEDIUM-004-001, MEDIUM-004-002, and LOW-004-001 are still open. Those leftovers are contract/UX/hygiene. They do not re-break delete truth or restart ownership, so they do not force FAIL.

Owner has authorized merge and Phase 4 freeze after this verification if the gate allows. This `PASS WITH CONDITIONS` is that gate. The conditions are accepted in this report and do not block freeze/merge. They do constrain Phase 5.

## Next Step

```text
Phase 4 Implementation: ACCEPTED WITH CONDITIONS
Immutable candidate: 8397dd4785a98a750f866ee191c0ca9026efe96e
Verification: focused High-path PASS; exact-commit CI 32226244512 SUCCESS
AUDIT-004: FAIL (historical, SHA 10a2509)
AUDIT-004R1: PASS WITH CONDITIONS (dirty tree; now this commit)
AUDIT-004R2: PASS WITH CONDITIONS
HIGH-004-001: CLOSED
HIGH-004-002: CLOSED
MEDIUM-004-001: OPEN (Owner-accepted)
MEDIUM-004-002: OPEN (Owner-accepted, create path only)
LOW-004-001: OPEN (Owner-accepted freeze-commit hygiene)
LOW-004R1-001: OPEN (Owner-accepted)
Merge to main: ALLOWED
Phase 4 freeze/tag: ALLOWED
Phase 5 implementation: BLOCKED until a Phase 5 Spec exists
This audit did not merge, tag, push, or begin Phase 5
```

# YORVA Phase 4 Independent Freeze Audit R3 — Instance / Profile Management

## Phase

Phase 4 — Instance / Profile Management

## Baseline / Commit

- Phase 3 baseline: `phase-003-hermes-installation-baseline` → `2d1925813faeead6de93f66c12083687a85957d4`
- Branch inspected: `fix/phase4-profile-delete-timeout`
- Immutable AUDIT-004 FAIL candidate: `10a250983cd348c56776c63965a7dbe026efb141`
- Previous freeze audit: `AUDIT-004R2` **PASS WITH CONDITIONS** on `8397dd4785a98a750f866ee191c0ca9026efe96e`
- Withdrawn freeze: merge `089a58005edc8f8f6a72b4fb44276be7c322eb1d` is now an ancestor; tag `phase-004-instance-profile-baseline` is **deleted** locally and on `origin`
- Timeout remediation: `8672af563ddb5b280fa06a617a474180d16bbdfd` — 30s profile create/delete budget
- **Immutable freeze candidate (this audit):** `35b268425a023f20c655bbfbd697f7a80c3e60a9`
- Tip commit subject: `fix(phase4): confirm delete in a modal and hide it on tombstones`
- `origin/fix/phase4-profile-delete-timeout` → the same SHA
- `main` / `origin/main` → `089a58005edc8f8f6a72b4fb44276be7c322eb1d`

This is the **final freeze audit** after the 2026-08-19 freeze was withdrawn. It is not an implementation pass and does not itself merge, tag, push, or begin Phase 5.

`git rev-parse HEAD` is `35b268425a023f20c655bbfbd697f7a80c3e60a9`. `8397dd4` and `089a580` are ancestors. The tracked worktree is clean. Untracked `.tmp-yorva-ui-ref/`, `YORVA_0.3.0_x64_en-US.wxs`, and untracked `docs/phases/PHASE-005-models-credentials.md` / `.zh-CN.md` drafts are outside the candidate and were not judged.

## Auditor

Fresh independent Phase 4 R3 freeze-audit context, separate from the implementation agent and from AUDIT-004 / AUDIT-004R1 / AUDIT-004R2. Review started from the approved Phase Specs, `AUDIT_STANDARD.md`, `PHASE_GOVERNANCE.md`, the three prior audits, and the execution chain on this exact SHA. Implementation completion summaries were not treated as evidence.

## Date

2026-08-19 (Asia/Shanghai)

## Gate Decision

`PASS WITH CONDITIONS`

## Executive Summary

HIGH-004-001 and HIGH-004-002 remain **CLOSED** on immutable SHA `35b268425a023f20c655bbfbd697f7a80c3e60a9`.

The withdrawn-freeze remediations are correct:

1. Profile create/delete commands now use a 30s mutation runner. List/discovery stay on the 3s command budget. A real-process regression proves a 3.2s delete times out under discovery and succeeds under mutation.
2. Desktop delete confirmation is a modal. The Delete button is offered only for `AVAILABLE`, non-default, non-protected instances. `MISSING` and `UNKNOWN` rows do not show Delete. Tombstone identity is still retained.

Delete still does not infer absence from `UNKNOWN`. `RecoverStale` still owns orphaned Instance Operations at daemon start. Desktop still rediscovers active create/delete by listing `targetType=runtime-installation`.

No new Critical or High finding was found on this SHA. Exact-commit CI run `32234908416` is SUCCESS for this head, including `go test -race ./...`.

The four AUDIT-004R2 leftovers remain open and are still **bounded contract / UX / hygiene**, not delete-truth, timeout-budget, or restart-deadlock:

- MEDIUM-004-001 — Operation target is still the installation ID, not `instanceId`
- MEDIUM-004-002 — create worker still collapses timeout to `INSTANCE_QUERY_FAILED`
- LOW-004-001 — Spec §24 still names `8397dd4` / CI `32226244512`; three Profile list fixtures still have EOF blanks
- LOW-004R1-001 — RecoverStale persist failure is warn-and-continue

Under `AUDIT_STANDARD.md`, those leftovers do not force FAIL. They are the same Owner-accepted R2 conditions. Owner-authorized merge to `main` and Phase 4 re-freeze are **allowed** on this SHA. Phase 5 implementation remains blocked until a Phase 5 Spec is written from the frozen baseline.

## Verification Evidence

### Repository and candidate

- `git rev-parse HEAD` → `35b268425a023f20c655bbfbd697f7a80c3e60a9`
- `git branch --show-current` → `fix/phase4-profile-delete-timeout`
- `git rev-parse origin/fix/phase4-profile-delete-timeout` → the same SHA
- `git rev-parse main` / `origin/main` → `089a58005edc8f8f6a72b4fb44276be7c322eb1d`
- `git log -1 --format` → `35b268425a023f20c655bbfbd697f7a80c3e60a9` / `fix(phase4): confirm delete in a modal and hide it on tombstones` / AuthorDate `2026-08-19 16:54:18 +0800`
- `git merge-base --is-ancestor 8397dd4785a98a750f866ee191c0ca9026efe96e HEAD` → yes
- `git merge-base --is-ancestor 8672af563ddb5b280fa06a617a474180d16bbdfd HEAD` → yes
- `git merge-base --is-ancestor 089a58005edc8f8f6a72b4fb44276be7c322eb1d HEAD` → yes
- `git tag --list phase-004*` — empty
- `git ls-remote --tags origin refs/tags/phase-004*` — empty (withdrawn tag is gone on origin)
- `git status --porcelain=v1` → only
  - `?? .tmp-yorva-ui-ref/`
  - `?? YORVA_0.3.0_x64_en-US.wxs`
  - `?? docs/phases/PHASE-005-models-credentials.md`
  - `?? docs/phases/PHASE-005-models-credentials.zh-CN.md`
- `git diff --check HEAD` — clean
- `git diff --check 089a580..HEAD` — clean
- `git diff --check d04b1fd..HEAD` still reports the historical EOF blanks:
  - `docs/phases/audits/AUDIT-004-instance-profile.md:232`
  - `services/node/internal/runtime/hermes/testdata/profile/list-default-and-named.txt:7`
  - `services/node/internal/runtime/hermes/testdata/profile/list-default-only.txt:5`
  - `services/node/internal/runtime/hermes/testdata/profile/list-duplicate-names.txt:7`

Candidate commits after the withdrawn freeze merge:

```text
8672af5 fix(phase4): give profile create/delete the 30s mutation budget
35b2684 fix(phase4): confirm delete in a modal and hide it on tombstones
```

This candidate does not contain Phase 5 lifecycle, credentials, channels, Cloud, or plugin work. The untracked PHASE-005 drafts are not part of the SHA.

### Exact-commit GitHub Actions

Re-queried GitHub API run [`32234908416`](https://github.com/YoLin02/yorva/actions/runs/32234908416) on 2026-08-19:

- `head_sha` = `35b268425a023f20c655bbfbd697f7a80c3e60a9`
- `head_branch` = `fix/phase4-profile-delete-timeout`
- `event` = `push`, `run_attempt` = 1
- `conclusion` = **success**
- Jobs (all `completed` / `success`, all `head_sha` identical):
  - **Go Node** (`96012594195`) — `go test -race ./...`, `go vet ./...`, `govulncheck ./...`, `go build ./cmd/yorvad`
  - **Web and API contract** (`96012594460`) — `pnpm audit`, `api:lint`, `api:generate` + generated-schema drift check, typecheck, lint, test, build
  - **Windows Desktop native shell** (`96012594543`) — sidecar, Windows lifecycle smoke, MSI inspect tests, cargo fmt/test/audit/clippy/check, Tauri `--no-bundle`

Historical CI is not reused as freeze evidence for this SHA:

- AUDIT-004 CI `32217465890` remains SUCCESS only for FAIL candidate `10a2509`
- AUDIT-004R2 CI `32226244512` remains SUCCESS only for withdrawn-freeze SHA `8397dd4`

### Focused local verification (this audit)

From `D:\workcode\myproject\services\node` against `35b2684`:

- `go test ./internal/app ./internal/persistence/sqlite ./internal/transport/httpapi ./internal/daemon ./internal/runtime/hermes -count=1` — PASS
  - `internal/app` 16.368s
  - `internal/persistence/sqlite` 0.541s
  - `internal/transport/httpapi` 0.247s
  - `internal/daemon` 0.190s
  - `internal/runtime/hermes` 20.833s
- named High/timeout rerun — PASS:
  - `go test ./internal/app -count=1 -run "Delete|RecoverStale|ClassifyHermes"`
  - `go test ./internal/persistence/sqlite -count=1 -run "ListActiveInstance"`
  - `go test ./internal/transport/httpapi -count=1 -run "DeleteInstanceQuery"`
  - `go test ./internal/runtime/hermes -count=1 -run "TestDeleteProfileSurvivesLongerThanDiscoveryTimeout|TestDeleteProfileArgv|TestListProfilesMapsTimeout|TestCreateProfileArgv"`
- `go vet ./internal/app ./internal/persistence/sqlite ./internal/transport/httpapi ./internal/daemon ./internal/runtime/hermes` — PASS, no diagnostics

From `D:\workcode\myproject\apps\desktop`:

- `pnpm test -- src/pages/InstancesPage.test.tsx src/App.instance-recovery.test.tsx src/operationRecovery.test.ts` — PASS, 3 files / 12 tests

Not run in this freeze audit (record, do not treat as a local PASS; exact-commit CI covers the matrix):

- full local `go test ./...`
- local `go test -race ./...` (historically CGO-blocked on this host; CI race is SUCCESS)
- full Desktop typecheck / lint / full test / build
- real destructive Hermes delete smoke (still forbidden against Owner data)

### Execution-chain review (not test-name review)

Mutation timeout (withdrawn-freeze remediation 1):

1. `commandTimeout` remains `3s`. `profileMutationTimeout` is `30s` (`command.go:13-15`).
2. `newCommandRunner()` still defaults to the 3s discovery budget (`command.go:39-46`). Detector and `ListProfiles` use that runner (`detector.go:22`; `profile_list.go:61-63`).
3. `newProfileMutationRunner()` copies the discovery runner and replaces only the timeout (`command.go:48-51`).
4. `CreateProfile` and `DeleteProfile` both construct `newProfileMutationRunner()` (`profile_create.go:41-43`; `profile_delete.go:47-49`). No other production caller uses it.
5. `TestDeleteProfileSurvivesLongerThanDiscoveryTimeout` builds a 3.2s fake `profile delete --yes`, proves the discovery runner times out, and proves the mutation runner returns success after more than 3s (`profile_delete_test.go:39-72`; `testdata/fakehermes/main.go:35-48`).
6. Application workers already wrap create/delete in `context.WithTimeout(..., 30*time.Second)` (`instance_inventory.go:451`, `:689`). The withdrawn-freeze defect was the adapter killing the Hermes command at 3s, not a missing worker deadline.

Delete modal and tombstone UX (withdrawn-freeze remediation 2):

1. `InstanceRow` offers Delete only when `!default && !protected && availability === "AVAILABLE"` (`InstancesPage.tsx:329-333`).
2. Confirmation is `DeleteConfirmDialog`: `role="dialog"`, `aria-modal="true"`, typed-name gate, destructive warning, Escape/backdrop dismiss (`InstancesPage.tsx:224-296`).
3. The dialog is mounted only for a non-protected, non-default target (`InstancesPage.tsx:156-168`). `MISSING` / `UNKNOWN` rows still render name and availability, so tombstone identity remains visible.
4. Tests: modal opens on Delete; Delete is absent for `MISSING` and `UNKNOWN` (`InstancesPage.test.tsx:140-246`).
5. Server delete contract is unchanged: `StartDelete` still rejects `default`/protected and non-`FRESH` lists; `ApplyInstanceSnapshot` still marks absent rows `MISSING` and never deletes them (`instance_inventory.go:551-575`; `instances.go:124-136`).

Delete authorization and success (HIGH-004-001, unchanged):

1. `StartDelete` still loads the YORVA row only for identity, protection, and typed confirmation (`instance_inventory.go:544-556`). That cache cannot authorize mutation by itself.
2. It then calls `ListInstances` and refuses unless `listed.Freshness == "FRESH"` (`instance_inventory.go:566-575`). `UNKNOWN` returns `instanceQueryError(listed.ErrorCode)` and does not create an Operation.
3. `runDelete` requires a successful `reconcileLocked` before `mutator.Delete` (`instance_inventory.go:696-699`). Any reconcile error fails the Operation and returns.
4. Absence is taken only from a successful reconcile via `instanceAvailable` (`instance_inventory.go:300-307`, `:701-703`). `reconcileLocked` returns `Freshness: "FRESH"` or an error; it never returns a usable UNKNOWN snapshot.
5. After delete, a second successful `reconcileLocked` is required before `succeedDelete` (`instance_inventory.go:706-713`). Post-delete query failure fails the Operation and leaves the row `UNKNOWN` via `MarkInstancesUnknown`.
6. Adapter classification still preserves `context.DeadlineExceeded` and unrecognized output (`hermes_profiles.go:38-51`; `instance_inventory.go:263-274`).

Restart recovery (HIGH-004-002, unchanged):

1. `daemon.Run` still constructs `InstanceInventory` and calls `RecoverStale` before serving HTTP (`daemon.go:110-113`).
2. `recoverOneLocked` still queries Hermes first. Query failure terminalizes as `FAILED` with the classified code and marks inventory `UNKNOWN`. It does not succeed (`instance_inventory.go:762-766`, `:784-794`).
3. Create succeeds only if the named profile is authoritatively present; delete succeeds only if authoritatively absent. A still-present profile is not deleted during recovery (`instance_inventory.go:767-778`).
4. Desktop lists operations for `runtime-installation` + `runtimeInstallationId`, then follows the newest active create/delete (`App.tsx:322-338`).

Create path (not a High regression):

- `runCreate` still writes `ErrorInstanceQueryFailed` for every non-success create outcome, including a classified timeout (`instance_inventory.go:469-473`). Success still requires an `AVAILABLE` row. A failed reconcile marks rows `UNKNOWN`, so create cannot succeed from UNKNOWN.

## Dimension Results

| Dimension | Result | Notes |
|---|---|---|
| Scope | PASS | Candidate stays inside Phase 4 timeout and delete-UX remediations. No Start/Stop/Restart implementation, credentials, channels, Cloud, plugin system, or Phase 3 generation/`active.json` mutation. Untracked PHASE-005 drafts are outside the SHA. |
| Correctness | PASS WITH CONDITIONS | HIGH-004-001 and HIGH-004-002 remain closed. 30s mutation vs 3s discovery is implemented and regression-tested. Delete is modal and hidden on tombstones. Create still collapses command timeout. Operation target is still the installation ID. |
| Architecture | PASS | React → local HTTP → application → domain/adapter is unchanged. Hermes parsing remains in the Hermes adapter. Recovery is application-owned. Public Instance DTOs still omit `nativeId`. |
| Security | PASS | No new bind, auth, secret, or shell surface. Profile commands remain absolute argv with restricted environment and bounded output. Delete of `default` remains server-rejected before process start. Modal is presentation only. |
| Data and Persistence | PASS WITH CONDITIONS | SQLite remains a cache. Tombstones remain on uncertain delete/recovery and on confirmed absence. UI no longer offers a second delete against those rows. Target identity still does not use `instanceId`. |
| Concurrency and Lifecycle | PASS | Per-installation mutation lock remains. Daemon-start recovery remains. Mutation commands now share the 30s worker budget instead of being killed at 3s. Recovery still does not resume a destructive delete automatically. |
| Protocol and Compatibility | PASS WITH FINDINGS | Closed create/delete bodies, auth, capability `lifecycle: false`, and stable query/timeout/unrecognized mappings hold. Create/delete Operations still target `runtime-installation`. |
| Testing and Verification | PASS WITH CONDITIONS | Required High-path, timeout-budget, and modal/tombstone regressions exist, passed locally, and are in the immutable SHA. Exact-commit CI including race is SUCCESS. Create timeout assertion is still missing. Isolated real-Hermes delete smoke was not run. |
| Maintainability | PASS | `instance_inventory.go` is 832 lines. Size is an observation, not a gate defect. No speculative workflow framework was added. |
| Documentation | PASS WITH FINDINGS | English/Chinese Specs still agree and correctly mark the phase UNFROZEN pending this audit. Completion evidence still names `8397dd4` and CI `32226244512`. |
| Dependencies / Supply Chain | PASS | These remediations add no dependency. CI `pnpm audit --audit-level low` and `govulncheck` succeeded. Cargo retains the historical allowed GTK/unic warnings. |
| Operations / Diagnostics | PASS WITH CONDITIONS | Orphaned Instance Operations become terminal `FAILED`/`SUCCEEDED` from Hermes truth. `RecoverStale` persist/list errors are still warn-and-continue at daemon start. Create timeout still surfaces as query failure. |

A dimension is not `N/A`. Impacted dimensions from the withdrawn freeze (Correctness, Testing, Documentation, Concurrency, Operations) were re-opened on this SHA. Unchanged Security/Architecture/Dependency conclusions from AUDIT-004R2 remain valid because the remediations did not alter those surfaces.

## Findings

### Critical

None.

### High

None open.

#### HIGH-004-001 — Failed authoritative delete queries are treated as confirmed absence

**CLOSED** on `35b268425a023f20c655bbfbd697f7a80c3e60a9`. Unchanged from AUDIT-004R2.

| Required closure | Current behavior on this SHA |
|---|---|
| Never infer absence from `UNKNOWN` | `StartDelete` rejects non-`FRESH` lists. `runDelete` fails on `reconcileLocked` error instead of reading presence after ignoring that error. |
| Successful pre-delete authoritative lookup before mutation | `StartDelete` requires `listed.Freshness == "FRESH"` before `CreateOperation`. Worker requires a successful reconcile before `mutator.Delete`. |
| Successful post-delete authoritative lookup before `SUCCEEDED` | Post-delete `reconcileLocked` error calls `failCreate`/`errorCodeFrom` and returns. |
| Timeout/malformed/query failure stay terminal and keep the tombstone | Classification still preserves timeout and unrecognized output. Snapshot update never deletes the row. Desktop no longer offers Delete on `MISSING`/`UNKNOWN`. |
| Regression tests | Before mutation, after mutation, timeout, genuine disappearance, and modal/tombstone UI are covered. |

The 30s mutation budget does not reopen this finding. A timed-out delete still fails closed; it no longer dies at the 3s discovery budget before Hermes can finish.

#### HIGH-004-002 — Active Instance Operations have no restart owner or recovery transition

**CLOSED** on `35b268425a023f20c655bbfbd697f7a80c3e60a9`. Unchanged from AUDIT-004R2.

| Required closure | Current behavior on this SHA |
|---|---|
| Daemon-start reconcile/terminalize, then Hermes truth | `daemon.go:110-113` still calls `RecoverStale`. |
| Idempotent; Operation status is not existence proof | Repeated recover returns no rows. A still-present profile is not deleted during recovery. |
| Desktop can rediscover active Instance Operations | `App.tsx` still lists `runtime-installation` operations and follows newest active create/delete. Recovered delete still opens the same modal. |
| Tests | Application recover tests and Desktop recovery tests still pass on this SHA. |

Policy choice remains **terminalize**, not resume.

### Medium

#### MEDIUM-004-001 — Instance Operation target identity does not follow the approved contract

**OPEN.** Unchanged on this SHA. Owner-accepted condition for freeze.

Create and delete still persist `targetType=runtime-installation` and `targetId=<installationId>` (`instance_inventory.go:351-355`, `:591-595`). Phase 4 §9 still says Operation targets use stable `instanceId`. The current target is not `nativeId`, so the original leak risk is still avoided, but the approved Instance target identity is still not implemented.

Desktop recovery still works *because* it queries that installation-scoped target (`App.tsx:322-325`). Delete-dialog recovery still binds the live Operation to an Instance by `item.name === recoveredDelete.message` (`App.tsx:333-334`).

This remains a bounded contract/maintainability defect. It does not reopen HIGH-004-002. It must not be copied into Phase 5 lifecycle Operations.

Required closure: preallocate or otherwise give create a stable `instanceId` target; make delete target the existing `instanceId`; keep a separate installation-scoped concurrency key if needed.

#### MEDIUM-004-002 — Instance command timeouts collapse to generic query failure

**OPEN**, still narrowed to the create worker. Owner-accepted condition for freeze.

Still closed by prior remediations and still true here:

- adapter `classifyHermesProfileError` preserves `context.DeadlineExceeded` as `ErrInstanceOperationTimedOut` (`hermes_profiles.go:45-47`);
- delete preflight and delete worker persist `INSTANCE_OPERATION_TIMED_OUT` (`instance_inventory.go:270-274`, `:698`, `:708`, `:716`);
- HTTP maps that code;
- delete/recovery timeout tests exist.

Still open:

- `runCreate` still writes `ErrorInstanceQueryFailed` for every non-success create outcome, including a classified timeout (`instance_inventory.go:469-473`);
- there is still no create-path assertion that `INSTANCE_OPERATION_TIMED_OUT` is persisted.

This is UX/diagnostics, not delete-truth and not the 3s-vs-30s budget defect. Create success still cannot be inferred from UNKNOWN. The adapter-level create command now has the same 30s budget as delete.

Required closure: persist `INSTANCE_OPERATION_TIMED_OUT` from the create worker and add the create timeout assertion demanded by AUDIT-004.

### Low

#### LOW-004-001 — Completion evidence and fixture whitespace need final synchronization

**OPEN.** Owner-accepted hygiene for the freeze commit.

Both language Specs §24 still name `8397dd4785a98a750f866ee191c0ca9026efe96e` and CI `32226244512`. They do not record `35b2684`, `8672af5`, or CI `32234908416`. `git diff --check d04b1fd..HEAD` still reports trailing blank lines at EOF in the three Profile list fixtures listed above. These remediations did not touch those fixtures.

The freeze commit may synchronize §24 to `35b268425a023f20c655bbfbd697f7a80c3e60a9`, CI `32234908416`, and this gate, and remove only the three reported fixture EOF blanks. That is documentation/whitespace, not a reason to reopen the Highs.

#### LOW-004R1-001 — Daemon-start RecoverStale failure is warn-and-continue, and there is no `daemon.Run` wiring test

**OPEN.** Non-blocking. Unchanged.

`RecoverStale` persist/list errors abort the remaining orphans and are logged as `Warn` (`daemon.go:111-113`). Query failures themselves are terminalized and do not return an error. A true persist failure can therefore leave a later orphan active while HTTP starts.

There is also no daemon-level test that `Run` invokes `RecoverStale`. The production call is three obvious lines and the policy is tested in `app`. This is not enough to reopen HIGH-004-002.

### Info

#### INFO-004R1-002 — `StartCreate` still proceeds when list freshness is `UNKNOWN`

Still valid. `StartCreate` uses `ListInstances` but does not require `FRESH` before creating an Operation (`instance_inventory.go:319-340`). Create success still requires an `AVAILABLE` row after reconcile (`instance_inventory.go:460-468`). Recorded only so it is not mistaken for an incomplete HIGH-004-001 fix.

#### INFO-004-001 — Verification environment observations

Still valid: local Go race is historically CGO-blocked on this host; isolated real-Hermes delete smoke was not run against Owner data. Exact-commit CI race and Windows native/Tauri no-bundle are SUCCESS for this SHA.

#### INFO-004R3-001 — Create has no separate 3.2s survival test

The mutation-budget regression is delete-shaped because that is the Owner-observed official command. Create uses the same `newProfileMutationRunner()`. A missing create-specific slow-process test is not a second timeout-budget defect.

#### INFO-004R3-002 — Untracked PHASE-005 drafts exist beside the candidate

Local untracked `PHASE-005-models-credentials` drafts and the ROADMAP `DRAFT — OWNER REVIEW REQUIRED` / `NOT AUTHORIZED` marker are outside this SHA. They are not Phase 4 scope expansion and do not authorize Phase 5 implementation.

#### INFO-004R3-003 — This review is the re-freeze candidate after a withdrawn tag

AUDIT-004R2 allowed freeze on `8397dd4`. That freeze was withdrawn after official `profile delete --yes` exceeded the 3s discovery budget, and tag `phase-004-instance-profile-baseline` was deleted. This report is the freeze verification of the remediations plus the still-closed Highs.

## Accepted Technical Debt

The remaining Medium/Low items are **Owner-accepted conditions** for this gate. They do not weaken delete-truth, the 30s mutation budget, restart ownership, secret handling, or architecture direction.

| ID | Severity | Why accepted | Impact | Owner | Target / trigger |
|---|---|---|---|---|---|
| MEDIUM-004-001 | Medium | Installation-scoped target is internally consistent, does not leak `nativeId`, and is what Desktop recovery queries. Not delete-truth or restart-deadlock. | Phase 5 lifecycle Operations would inherit the wrong target identity if copied. Delete-dialog recovery remains name/message coupling. | Repository owner | Before or inside the Phase 5 Spec; do not copy this target shape into start/stop/restart Operations. |
| MEDIUM-004-002 | Medium | Delete/recovery already persist `INSTANCE_OPERATION_TIMED_OUT`. Create still fail-closes; only the error code is wrong. Adapter create now has the 30s budget. | Create timeout UI cannot distinguish timeout from query failure. | Repository owner | Next Instance mutation/UX pass, and before Phase 5 lifecycle timeout UX depends on the code. |
| LOW-004-001 | Low | Historical completion block and fixture EOF blanks do not change runtime behavior. | Spec §24 is stale relative to this freeze SHA. | Repository owner | Apply in the Phase 4 re-freeze commit. |
| LOW-004R1-001 | Low | Persist failure at daemon start is a storage emergency, not the orphan-without-owner defect. | A later orphan could remain active if SQLite write fails mid-recover. | Repository owner | If daemon-start recovery is extended, add fail-closed persist handling and a wiring test. |

The `instance_inventory.go` file-size observation remains non-blocking and is not accepted as a required split.

## Required Fixes Before Next Phase

None that block this gate. Highs remain closed, the withdrawn-freeze remediations are correct, exact-commit CI is SUCCESS, and the remaining findings are the same Owner-accepted R2 conditions.

Permitted freeze-commit hygiene (does not require a new High re-audit if it touches only these):

1. Synchronize both language Specs §24 to candidate `35b268425a023f20c655bbfbd697f7a80c3e60a9`, CI run `32234908416`, and this gate. Mark the phase FROZEN only after the freeze commit/tag exists.
2. Remove only the three reported Profile list fixture EOF blanks.

Do not regress the freshness/error consume path, daemon-start `RecoverStale`, the 30s mutation / 3s discovery split, or Delete-hidden-on-tombstone UI in that freeze commit.

## Required Fixes Before Phase 5

1. Write a Phase 5 Spec from this frozen baseline. Do not begin lifecycle or models/credentials implementation from `ROADMAP.md` or the untracked PHASE-005 drafts alone.
2. Do not copy MEDIUM-004-001 into start/stop/restart Operations. Phase 5 must specify `instanceId` targeting (or an explicit ADR if installation-scoped targeting is to become the contract).
3. Do not copy MEDIUM-004-002 into lifecycle workers. Preserve `INSTANCE_OPERATION_TIMED_OUT` end-to-end, including create if it remains in-scope.
4. Keep HIGH-004-001 / HIGH-004-002 closures as invariants: never infer absence from `UNKNOWN`; never treat Operation status as Hermes existence.
5. Keep mutation commands on an explicit budget distinct from discovery list timeout.

## Gate Rationale

`AUDIT_STANDARD.md` requires FAIL for an unresolved blocking High, and allows `PASS WITH CONDITIONS` only when Highs are closed and remaining findings are bounded, owned, and explicitly accepted.

Independent execution-chain review of this exact SHA, plus focused High-path, timeout-budget, and InstancesPage tests, plus exact-commit CI `32234908416`, show:

- HIGH-004-001 and HIGH-004-002 remain closed;
- create/delete no longer share the 3s discovery timeout;
- delete confirmation is a modal and is not offered on tombstones;
- no new High was found.

An unconditional `PASS` would be false: MEDIUM-004-001, MEDIUM-004-002, and LOW-004-001 are still open. Those leftovers are the same bounded R2 conditions. They do not re-break delete truth, the mutation budget, or restart ownership, so they do not force FAIL.

Owner has authorized merge and Phase 4 re-freeze after this verification if the gate allows. This `PASS WITH CONDITIONS` is that gate. The conditions are accepted in this report and do not block freeze/merge. They do constrain Phase 5.

## Next Step

```text
Phase 4 Implementation: ACCEPTED WITH CONDITIONS
Immutable candidate: 35b268425a023f20c655bbfbd697f7a80c3e60a9
Also contains: 8672af563ddb5b280fa06a617a474180d16bbdfd
Verification: focused High-path / timeout / InstancesPage PASS; exact-commit CI 32234908416 SUCCESS
AUDIT-004: FAIL (historical, SHA 10a2509)
AUDIT-004R1: PASS WITH CONDITIONS (dirty tree; later 8397dd4)
AUDIT-004R2: PASS WITH CONDITIONS (withdrawn freeze on 8397dd4)
AUDIT-004R3: PASS WITH CONDITIONS
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

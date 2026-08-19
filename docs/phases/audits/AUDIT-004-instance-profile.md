# YORVA Phase 4 Independent Audit — Instance / Profile Management

## Phase

Phase 4 — Instance / Profile Management

## Baseline / Commit

- Phase 3 baseline: `phase-003-hermes-installation-baseline` → `2d1925813faeead6de93f66c12083687a85957d4`
- Branch inspected: `codex/phase4-instance-profile`
- Immutable audit candidate: `10a250983cd348c56776c63965a7dbe026efb141`
- Remote branch tip at audit: `origin/codex/phase4-instance-profile` → the same SHA
- `main` / `origin/main` at audit: `d04b1fdc298f643f84d0c84a245595baae2e8994`

The tracked worktree was clean at audit start. Pre-existing untracked `.tmp-yorva-ui-ref/` and `YORVA_0.3.0_x64_en-US.wxs` were outside the candidate and were not modified. This audit did not merge, freeze, tag, delete a branch, change implementation, or begin Phase 5.

## Auditor

Independent Phase 4 review against the approved Phase Spec, repository governance, actual source paths, tests, local verification, and exact-commit CI evidence.

## Date

2026-08-19 (Asia/Shanghai)

## Gate Decision

**FAIL**

## Executive Summary

The implementation is materially complete across the normal Phase 4 flows. It stays within Instance/Profile scope, uses the pinned Hermes Profile CLI through direct argv, keeps `nativeId` out of public responses, protects `default`, retains `MISSING` tombstones, presents bilingual Desktop states, and passes exact-commit CI.

No Critical defect was found. Two High correctness defects remain, both within explicit Phase 4 acceptance criteria rather than speculative hardening:

1. a failed authoritative query during delete is converted into `UNKNOWN`, but `UNKNOWN` is then treated as absence and the durable delete Operation can be marked `SUCCEEDED` without running Hermes delete;
2. active `instance.create` / `instance.delete` Operations have no daemon-start recovery path, and the Desktop does not rediscover them after restart. A daemon restart therefore leaves a durable active row that permanently conflicts with later mutations.

Green CI does not cover either path. Phase 4 is not eligible for merge/freeze until both High findings are closed and independently re-audited.

## Verification Evidence

### Repository and candidate

- `git rev-parse HEAD` → `10a250983cd348c56776c63965a7dbe026efb141`.
- `git rev-parse origin/codex/phase4-instance-profile` → the same SHA.
- `git rev-parse origin/main` → `d04b1fdc298f643f84d0c84a245595baae2e8994`.
- No `phase-004*` tag exists.
- `git diff --check d04b1fd..10a2509` reports three fixture-only trailing blank lines at EOF. This is Low cleanup, not the gate rationale.

### Exact-commit GitHub Actions

- CI run [`32217465890`](https://github.com/YoLin02/yorva/actions/runs/32217465890) — push, branch `codex/phase4-instance-profile`, head `10a250983cd348c56776c63965a7dbe026efb141`, **SUCCESS**:
  - Web and API contract — success;
  - Go Node, including `go test -race ./...`, vet, govulncheck, and build — success;
  - Windows Desktop native shell, lifecycle, Rust audit, and Tauri no-bundle — success.

### Local Web / OpenAPI

- `pnpm api:lint` — PASS.
- `pnpm api:generate` plus generated-schema drift check — PASS.
- `pnpm typecheck` — PASS.
- `pnpm lint` — PASS.
- `pnpm test` — PASS, 17 files / 67 tests.
- `pnpm build` — PASS.
- `pnpm audit --audit-level low` — PASS, no known vulnerabilities.

### Local Go

- `go test ./...` — PASS.
- `go vet ./...` — PASS.
- `go build ./cmd/yorvad` — PASS.
- `govulncheck ./...` — PASS, no vulnerabilities found.
- affected application, SQLite, HTTP and daemon packages with `-count=20` — PASS.
- the combined Hermes `-count=20` run had one fixture timing failure because a 100 ms timeout elapsed before its fake process wrote the PID file; the focused failing test then passed `-count=50`. This is recorded as test flakiness, not evidence of escaped descendants.
- local `go test -race ./...` — environment-blocked because `CGO_ENABLED=0`; it is not reported as a local PASS. Exact-commit CI race is PASS.

### Local Rust / Windows

- `cargo fmt --all -- --check` — PASS.
- `cargo test --locked` — PASS, 10 tests.
- `cargo clippy --locked --all-targets -- -D warnings` — PASS.
- `cargo check --locked` — PASS.
- `cargo audit` — zero known vulnerabilities; 17 inherited allowed GTK3/unic warnings.
- `pnpm build:sidecar` — PASS.
- `scripts/windows-lifecycle-smoke.ps1` — PASS.
- `scripts/inspect-yorva-msi.tests.ps1` — PASS.
- local Tauri release no-bundle — environment-blocked by Windows `PermissionDenied` in the existing release build directory. Exact-commit Windows CI no-bundle is PASS.
- real destructive Hermes delete smoke was not run against the Owner profile, as required by the Spec; it requires an isolated disposable account/VM.

## Dimension Results

### Scope

PASS. No Phase 5 lifecycle, credentials/models, channels, Skills/MCP, Cloud, plugin framework, Hermes installation change, or Hermes fork entered the candidate.

### Correctness

FAIL. Delete can return success from an uncertain query, and daemon restart can permanently strand active instance Operations.

### Architecture

PASS WITH OBSERVATION. Dependency direction remains React → local HTTP → application → domain/adapter. `instance_inventory.go` is large (about 701 lines), but its size is not itself a gate defect and no speculative refactor is required for Phase 4 acceptance.

### Security

PASS. Profile commands use an absolute executable, direct argv, a restricted environment, bounded output, shared process-tree containment, and no secret-bearing payload. Public Instance responses omit `nativeId` and filesystem paths. Default deletion and typed confirmation are enforced server-side before mutation.

### Data and Persistence

FAIL. Identity/tombstone reconciliation is correct on successful queries, but stale active instance Operations are never transitioned after daemon restart and remain protected by the active-mutation uniqueness constraint.

### Concurrency and Lifecycle

FAIL. Per-installation mutation serialization works during one daemon lifetime. Cross-restart lifecycle does not: active Operations outlive their workers and block all later mutations.

### Protocol and Compatibility

PASS WITH FINDINGS. Closed request bodies, authentication, stable resource DTOs, OpenAPI generation, capability flags, and `nativeId` non-disclosure pass. Operation targeting and timeout normalization have bounded contract drift recorded below.

### Testing and Verification

FAIL. The broad matrix and exact-commit CI pass, but the mandatory failed-query delete and daemon/Desktop restart paths have no regression coverage and fail by direct execution-chain review.

### Maintainability

PASS WITH OBSERVATION. The current implementation is understandable and phase-scoped. Splitting the 701-line application file can be deferred until after correctness closure; it must not replace the required fixes with a broad refactor.

### Documentation

PASS WITH FINDINGS. The approved English and Chinese Specs agree on the governing behavior. Completion evidence still names `fe15203` instead of the final candidate and says exact CI is pending; this should be synchronized in the remediation handoff.

### Dependencies / Supply Chain

PASS. No new unapproved framework or mutable Hermes source was introduced. Existing dependency audit results remain unchanged.

### Operations / Diagnostics

FAIL. A restarted daemon has no explicit reconciliation/terminalization step for Instance Operations, so UI/API projection can remain permanently active without an owner goroutine.

## Findings

### Critical

None.

### High

#### HIGH-004-001 — Failed authoritative delete queries are treated as confirmed absence

`ListInstances` intentionally converts a Hermes query failure into a successful response with `freshness=UNKNOWN`, marks cached rows `UNKNOWN`, and returns no Go error (`instance_inventory.go:131-178`). `StartDelete` then calculates `present` using only `availability == AVAILABLE` (`instance_inventory.go:533-545`), so an uncertain existing profile is treated as not present.

The worker repeats the same error. It ignores the error returned by `reconcileLocked`, which has just marked rows `UNKNOWN`, then calls `profilePresent`; that helper returns true only for `AVAILABLE` (`instance_inventory.go:443-478`, `639-670`). The resulting `presentErr == nil && !stillPresent` branch marks the Operation `SUCCEEDED` before `mutator.Delete` is called.

Deterministic impact: if `hermes profile list` times out, is malformed, or otherwise fails during delete preflight/worker execution, YORVA can report permanent deletion success while the Hermes Profile and its data still exist. This directly violates Phase 4 Sections 6, 12, and 19: uncertain state must remain `UNKNOWN`; absence is success only after a successful authoritative query.

Required closure:

1. make the delete decision consume explicit query freshness/error, never infer absence from an `UNKNOWN` cached row;
2. require a successful pre-delete authoritative lookup before authorizing mutation;
3. require a successful post-delete authoritative lookup before `SUCCEEDED`;
4. map timeout/malformed/query failure to the specified stable terminal error without deleting the tombstone;
5. add regression tests for failure before mutation, failure after mutation, timeout, and concurrent genuine disappearance.

#### HIGH-004-002 — Active Instance Operations have no restart owner or recovery transition

Daemon startup calls `RuntimeInstall.InterruptStale`, but constructs `InstanceInventory` without any corresponding Instance Operation recovery (`daemon.go:106-110`). The SQLite interrupter selects only `runtime.install` and `hermes.prerequisites` (`operations.go:231-269`). Consequently, a persisted `instance.create` or `instance.delete` row in `PENDING`/`RUNNING` survives daemon termination with no worker and no recovery action.

`ActiveInstanceMutation` and migration `008_instance_operations.sql` continue treating that row as active, so all future creates/deletes for the installation return conflict indefinitely. The Desktop restart path also cannot recover the projection: it lists only `targetType=runtime-kind,targetId=hermes` and restores install/prerequisite IDs, while create/delete IDs exist only in component-local React state (`App.tsx:38-43`, `103-110`, `310-381`).

Deterministic impact: closing/restarting the daemon or Desktop during an Instance mutation can leave the UI without progress/cancel state and the backend permanently unable to accept another Instance mutation. This directly violates Phase 4 Sections 6, 13, 17, 19, and the restart test matrix.

Required closure:

1. add one daemon-start Instance Operation reconciliation policy: restore/re-evaluate or terminalize orphaned `PENDING`/`RUNNING` rows, then perform a fresh Hermes reconciliation;
2. ensure the decision is idempotent and never treats Operation status as proof of profile existence;
3. expose/query active Instance Operations so Desktop restart can recover their projection;
4. prove daemon restart, Desktop restart, repeated restart, and post-restart new mutation with application/persistence/Desktop regression tests.

### Medium

#### MEDIUM-004-001 — Instance Operation target identity does not follow the approved contract

The Phase Spec states that Instance Operation targets use stable `instanceId` and that `nativeId` is adapter-only. Both create and delete Operations currently use `targetType=runtime-installation` and the installation ID (`instance_inventory.go:321-322`, `564-565`). This avoids leaking `nativeId`, but it does not implement the approved Instance target identity and contributes to the missing Desktop recovery query.

Required closure: define the minimal create preallocation/target behavior and make delete target the existing stable `instanceId`, while retaining a separate installation-scoped concurrency key if needed. Update persistence and OpenAPI tests without introducing a generic workflow abstraction.

#### MEDIUM-004-002 — Instance command timeouts collapse to generic query failure

The Runtime error model includes `INSTANCE_OPERATION_TIMED_OUT`, but create/delete adapters wrap command deadline expiration as profile query failure, and application workers persist `INSTANCE_QUERY_FAILED`. The UI therefore cannot distinguish the timeout guidance required by Sections 13, 17, and 19.

Required closure: preserve `context.DeadlineExceeded` through the adapter boundary, persist `INSTANCE_OPERATION_TIMED_OUT`, and add create/delete timeout assertions. Do not change the existing bounded command/process cleanup design.

### Low

#### LOW-004-001 — Completion evidence and fixture whitespace need final synchronization

The Phase Spec completion block names pre-remediation commit `fe15203` and still says exact CI is pending. `git diff --check` also reports trailing blank lines at EOF in three Profile list fixtures. Update the final candidate/CI evidence and remove only the reported fixture whitespace during remediation.

### Info

#### INFO-004-001 — Verification environment observations

Local Go race remains unavailable with `CGO_ENABLED=0`, and local Tauri no-bundle encountered a Windows permission lock; exact-commit CI passed both. Cargo retains 17 inherited allowed warnings and no known vulnerability. The isolated destructive real-Hermes smoke remains intentionally not run against Owner data.

## Accepted Technical Debt

None accepted by this audit. The file-size observation is not a defect, and the Medium/Low items do not justify a broad refactor. The two High findings must be closed because they break current Phase 4 deletion and restart behavior.

## Required Fixes Before Freeze

1. Make delete success require a successful authoritative absence result; preserve `UNKNOWN` on every uncertain query.
2. Add daemon-start reconciliation/terminalization for orphaned Instance Operations and Desktop recovery of their projection.
3. Align Instance Operation targeting with stable `instanceId` and preserve the timeout error code.
4. Add the narrowly scoped regression tests listed in HIGH-004-001 and HIGH-004-002.
5. Synchronize final completion evidence and make `git diff --check` clean.
6. Push a new immutable candidate, obtain exact-commit CI PASS, and request independent `AUDIT-004R1`.

## Gate Rationale

`AUDIT_STANDARD.md` requires FAIL for any High finding and for a mandatory phase success-flow failure. Both High findings are deterministic current-scope correctness/lifecycle failures explicitly covered by the approved Phase 4 Spec. They are not hypothetical adversarial hardening and cannot be deferred merely because the broad CI matrix is green.

## Next Step

```text
Phase 4 Implementation: NOT ACCEPTED
Verification: broad matrix PASS, required edge-path evidence FAIL
AUDIT-004: FAIL
Merge to main: BLOCKED
Phase 4 freeze/tag: BLOCKED
Phase 5: BLOCKED
```


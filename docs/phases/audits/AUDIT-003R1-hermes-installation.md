# YORVA Phase 3 Independent Re-Audit R1 — Hermes Installation

## Phase

Phase 3 — Hermes Installation, including Amendments 003A1, 003A2 and 003A3.

## Baseline / Commit

```text
Previous frozen baseline:
phase-002-hermes-discovery-baseline-r1
5b89d22ed5e7ae3f4374a26f0fcda54bdabc6bf9

Independent re-audit candidate:
fix/phase3-managed-node-prerequisites
f93075428480133698f93e7915e409304aa55b68

Candidate CI:
https://github.com/YoLin02/yorva/actions/runs/32018236206
event=push
head=f93075428480133698f93e7915e409304aa55b68
conclusion=success

Candidate MSI CI:
https://github.com/YoLin02/yorva/actions/runs/32018236189
event=push
head=f93075428480133698f93e7915e409304aa55b68
conclusion=success
artifact=yorva-msi
artifact digest=sha256:0841c65aca9733de3d04c21ccac57111e77088ecce071b26a4152f984ac16ca1
```

`f930754` is still the remote feature-branch tip. Its tree is identical to the Phase 3 side of merge commit `9459eb9fea435704d6639be8457ee11dfad02093` on `main`.

The Owner confirmed that the merge to `main` was performed manually before this independent re-audit. This is recorded as a governance-order deviation, not as an unknown or malicious repository mutation. It does not convert a failed audit into acceptance.

The audit worktree initially contained one unrelated untracked generated file, `YORVA_0.3.0_x64_en-US.wxs`. It is not part of `f930754` and was not used as candidate evidence. This report is the auditor's only repository addition.

## Auditor

Fresh independent Phase 3 re-audit agent. The implementation summary was not accepted as evidence.

## Date

2026-08-17, Asia/Shanghai.

## Gate Decision

**FAIL**

## Executive Summary

The remediation candidate materially improves Phase 3. It adds a database-enforced shared Hermes-host mutation lock, makes the worker own cancellation terminality, requires the canonical managed launcher, separates pre-Hermes Node/npm preparation from dependency installation, adds a 60-minute Operation deadline, splits the previous `node_managed.go` concentration, and provides green exact-commit CI plus a green MSI workflow.

Those improvements are not sufficient to freeze Phase 3. Original `AUDIT-003` findings HIGH-005, MEDIUM-001, MEDIUM-003 and MEDIUM-004 are not fully closed. Independent review also found required workflow and target-ownership defects that were not identified in the first audit:

- Desktop cannot recover a running `runtime.install` after restart even though the durable list API exists;
- retry target validation does not prove that the target is still the same pinned YORVA-owned partial installation before overlaying files;
- concurrent same-key starts can return an in-progress conflict instead of the same Operation;
- Phase 3 Operation SSE events are documented but never published;
- the MSI inspector can treat `NPM-LICENSE` as the Hermes `LICENSE` and does not verify embedded payload hashes.

Under `AUDIT_STANDARD.md`, required success-flow failure, unresolved High findings, and insufficient packaging evidence require `FAIL`. Phase 3 must not be marked complete/frozen, tagged, or used to authorize Phase 4.

## Original Finding Closure Matrix

| Original finding | R1 result | Evidence |
| --- | --- | --- |
| HIGH-001 — install/prerequisite mutual exclusion | CLOSED | Migration `003_hermes_host_mutation.sql` adds one partial unique index across both operation types; both start paths use `ActiveHermesHostMutation`; application and SQLite tests cover cross-type conflict. |
| HIGH-002 — cancellation terminal before cleanup | CLOSED | `installWorker` now separates `cancel` and `done`; the worker writes `CANCELLED`, then closes `done`; `Cancel` waits for the acknowledgement. Fake worker ordering and existing real Windows Job Object descendant tests pass. |
| HIGH-003 — Python fallback accepted as install success | CLOSED | `verifyPublicLauncher` requires the canonical regular launcher and user PATH; `postcheckAccepted` requires exact canonical launcher path and `0.20.2`; regression test rejects Python fallback. |
| HIGH-004 — prerequisite false-fail / partial npm false-pass | CLOSED | Pre-Hermes Apply now stops successfully after Node/npm; dependency readiness requires a stamp matching `package-lock.json`; tests cover pre-Hermes success and unstamped partial modules. |
| HIGH-005 — clean MSI not fail-closed / insufficient inspection | **NOT CLOSED** | A dedicated workflow and packaging entry point exist, but the inspector has a false-positive license match and verifies only file-table sizes, not embedded payload hashes. See HIGH-R1-003. |
| HIGH-006 — no immutable green candidate | CLOSED | Exact candidate CI run `32018236206` and MSI run `32018236189` are both successful at `f930754`. |
| MEDIUM-001 — raw command output/error persistence | **NOT CLOSED** | Raw stdout/stderr were removed from `logCommand`, but `resolveArchive` still logs `err.Error()` into the same persisted/HTTP-readable log. See MEDIUM-R1-001. |
| MEDIUM-002 — no whole-Operation deadline | CLOSED | `bindWorker` applies `context.WithTimeout(..., 60*time.Minute)` to install and prerequisite workers. |
| MEDIUM-003 — oversized managed Node module / inadequate tests | **PARTIAL** | The 467-line file is split into cohesive files, but several required artifact, argv/environment, timeout/cancellation and materialization tests remain absent. See MEDIUM-R1-002. |
| MEDIUM-004 — contradictory governance documents | **NOT CLOSED** | Header and stage table improved, but completion evidence, checklists, timeouts, amendment statuses and review record remain stale or contradictory. See MEDIUM-R1-003. |

## Verification Evidence

### Remote exact-commit evidence

CI run `32018236206`:

```text
Web and API contract:          success
Go Node, including race:       success
govulncheck:                    success
Windows Desktop native shell:  success
Tauri no-bundle:               success
```

MSI run `32018236189`:

```text
Package and inspect MSI: success
Artifact:                yorva-msi, 114 MB
Artifact digest:         0841c65aca9733de3d04c21ccac57111e77088ecce071b26a4152f984ac16ca1
```

Both runs contain inherited GitHub Actions Node.js 20 deprecation warnings for pinned action implementations. No job failed because of those warnings.

### Local PASS

```text
pnpm api:lint
pnpm api:generate
generated schema clean check
pnpm typecheck
pnpm lint
pnpm test                    13 files / 48 tests
pnpm build
pnpm audit --audit-level low no known vulnerabilities

go test ./...
go test affected packages -count=20
go vet ./...
go build ./cmd/yorvad
govulncheck ./...            no vulnerabilities

cargo fmt --all -- --check
cargo test --locked          10 tests
cargo clippy --locked --all-targets --all-features -- -D warnings
cargo check --locked
cargo audit                  0 vulnerabilities; 17 inherited allowed warnings

Windows lifecycle smoke      PASS
OpenAPI validation           PASS
candidate tracked diff       unchanged by verification
```

### Local environmental blockers / non-candidate residue

```text
go test -race ./...          BLOCKED locally: race requires CGO
                              Exact-commit CI race passed.

Tauri release --no-bundle    BLOCKED locally with Windows PermissionDenied while
                              Owner's target/release/yorvad.exe process was running.
                              Exact-commit Windows CI no-bundle passed.

Untracked WXS                YORVA_0.3.0_x64_en-US.wxs; excluded from candidate.
```

The local existing MSI was inspected only as supporting evidence. Its SHA-256 is `0CD378544635B91D096628AD93CD5BD77E13EFC6FEF072C6EC6439B39A55567E`; it is not the exact CI artifact and therefore is not substituted for the candidate artifact.

## Dimension Results

### Scope

PASS. No Phase 4 Profile/Instance implementation, credential/model configuration, channels, lifecycle, Skills/MCP, Cloud, dynamic plugin framework, Hermes fork, or generic package manager was found.

### Correctness

FAIL. Desktop restart recovery and concurrent same-key idempotency do not meet the Operation contract.

### Architecture

PASS. React still uses authenticated typed HTTP; install orchestration remains in Go; Hermes-specific behavior is adapter-owned; Tauri remains a narrow resource/bootstrap shell.

### Security

FAIL. Retry ownership evidence is too weak before a mutable installation tree is overlaid, and raw source-acquisition errors remain persistently exposed through the operation log API.

### Data and Persistence

PASS WITH FINDING CONTEXT. The shared partial unique index correctly enforces one active Hermes-host mutation across both operation types, and migrations from empty and Phase 2 pass. Application-level same-key idempotency handling above that store is still incorrect under a race.

### Concurrency and Lifecycle

FAIL. Cross-type mutation exclusion and cancellation ordering are improved, but simultaneous same-key starts are not normalized to the same Operation as required.

### Protocol and Compatibility

FAIL. Documented Phase 3 Operation SSE events are not emitted, and Desktop does not use the list/recovery contract.

### Testing and Verification

FAIL. Exact CI is green, but it does not cover the newly identified reload, same-key concurrent start, target replacement, MSI license false positive, embedded-payload hash, or Operation-event behaviors.

### Maintainability

PASS WITH TEST DEBT. The former managed Node monolith was split into `node_prereq.go`, `node_materialize.go`, `node_extract.go` and `node_deps.go`. Existing large `archive.go` and `host_installer.go` remain cohesive enough that line count alone is not a finding.

### Documentation

FAIL. Phase 3 governance and protocol documents still describe contradictory or unimplemented states.

### Dependencies / Supply Chain

FAIL. Artifact pins are strong and CI packaging is green, but the MSI acceptance inspection can false-pass license absence and does not establish embedded payload digests.

### Operations / Diagnostics

FAIL. Raw archive errors remain in the persisted/HTTP-readable operation log, and Desktop cannot resume the main install Operation after restart.

## Findings

### Critical

None.

### High

#### HIGH-R1-001 — Desktop does not recover an active Hermes installation after restart

The Phase 3 Spec requires Desktop to query the durable Operation and resume displaying it after reopening. The bounded list API exists and the generated client exposes `listOperations`, but `App.tsx` never calls it. `activeOperationId` is initialized only from local React state and is set only after the current UI successfully starts an install.

After Desktop restarts during a running install:

1. `activeOperationId` is `null`;
2. no `GET /operations?...` recovery query runs;
3. no progress, log, or Cancel control is attached to the durable Operation;
4. clicking Install receives `RUNTIME_INSTALL_IN_PROGRESS`, but unlike the prerequisite handler, `startInstall` does not use `details.operationId` to attach to the active Operation.

Evidence:

- `apps/desktop/src/App.tsx:26,71-88,89-117`;
- `apps/desktop/src/api/client.ts:127-130` defines the unused list call;
- `apps/desktop/src/App.tsx:162-165` attaches conflict IDs only for prerequisites;
- Phase 3 Spec lines 70 and 404 explicitly require reopen/recovery;
- no Desktop reload/recovery test exists.

Impact: the durable Operation continues mutating the machine while the reopened control surface cannot accurately display or cancel it. This is a required Phase 3 success/recovery flow, so it blocks the gate.

Required closure:

- on daemon/session readiness, query the bounded Hermes Operation list and attach to the newest non-terminal `runtime.install`;
- on install conflict, validate the returned Operation type/target and attach to it;
- keep prerequisite and runtime-install Operation presentation type-safe;
- add reload, active-conflict, terminal-history and cancel-after-reopen Desktop tests.

#### HIGH-R1-002 — Retry ownership validation can overlay a changed or foreign target

Retry is allowed when durable history says the previous Operation is retryable, but filesystem validation does not prove the target is still the same pinned YORVA-owned partial installation:

- `hasYorvaPartialMarker` checks only that `.yorva-phase3-install` is a regular file; it never reads the stored commit or requires `df4b651...`;
- `hasOfficialRepositoryIdentity` returns true when any one generic marker (`hermes`, `pyproject.toml`, or `hermes_cli/main.py`) exists; it verifies neither all required markers nor pinned identity;
- a marker/tree can change after the previous failed Operation;
- `placeMaterializedTree` copies and truncates matching files into the existing directory while retaining unrelated content.

Evidence:

- `services/node/internal/runtime/hermes/target.go:44-50,53-68`;
- `services/node/internal/runtime/hermes/target.go:71-75` writes the commit that validation never checks;
- `services/node/internal/runtime/hermes/archive.go:393-425` overlays the accepted target;
- the only target retry test covers a freshly written valid marker, not an empty/wrong-pin marker, changed target, partial generic marker, or foreign content.

Impact: after a failed YORVA attempt, externally replaced/changed contents can be treated as safe and overlaid. This violates the Spec's no-foreign-overwrite and same-pin retry rules and can preserve unexpected executable/source content inside the managed tree.

Required closure:

- require the exact expected marker content and the matching durable Operation pin/target identity;
- reject a missing, empty, wrong-pin, malformed, reparse, or externally replaced marker;
- define and verify the complete allowed partial-tree identity before overlay;
- materialize atomically into a verified YORVA-owned target rather than merging into uncertain contents;
- add target replacement, wrong marker, stale pin, foreign marker, extra executable and reparse regression tests.

#### HIGH-R1-003 — MSI acceptance inspection can false-pass and does not verify embedded payload content

The new release entry point verifies source files before bundling, but the post-build inspector is not strong enough to prove the MSI contract.

For every required filename it selects the first MSI key satisfying:

```powershell
$_ -eq $name -or $_ -like "*$name"
```

For required name `LICENSE`, `NODE-LICENSE` and `NPM-LICENSE` also match. Local reproduction printed `NPM-LICENSE` twice and never proved the Hermes `LICENSE` entry. Removing/renaming the Hermes license could therefore pass inspection.

The inspector checks only MSI File-table sizes. It never extracts or hashes the three embedded payloads, although Amendment 003A1 requires the archive resource itself to hash to the compiled digest. License inputs are accepted when merely present and longer than 20 bytes; their expected content digests are not checked.

Evidence:

- `scripts/inspect-yorva-msi.ps1:30-49`;
- `scripts/package-yorva-msi.ps1:21-49`;
- local inspector output contained `NPM-LICENSE` twice;
- exact MSI CI is green because it runs the same insufficient inspector.

Impact: the release gate can report a packaged, inspected MSI while required provenance/license content is absent or substituted, and there is no independent proof that embedded archive bytes match the reviewed pins. Original HIGH-005 is therefore not closed.

Required closure:

- match MSI filenames exactly after decoding WiX short/long names; require exactly one of each;
- verify exact license identities/content required by the amendments;
- extract or otherwise stream each embedded payload from the built MSI and require its exact SHA-256, not only File-table size;
- record payload and final MSI digests from the same exact-commit CI artifact;
- add negative inspector tests for missing Hermes LICENSE, name suffix collisions, duplicate entries, wrong-size and same-size/wrong-hash payloads.

#### HIGH-R1-004 — Concurrent same-key starts do not guarantee idempotent Operation identity

Both start paths first query the idempotency key and later insert the Operation. Under simultaneous requests with the same new key, both can observe no row. One insert wins; the other receives a unique-key error. `normalizeCreateConflict` then looks up the active host mutation and returns `RUNTIME_INSTALL_IN_PROGRESS` rather than fetching and returning the Operation associated with the same idempotency key.

Evidence:

- `services/node/internal/app/runtime_install.go:96-104,146-160`;
- `services/node/internal/app/hermes_prerequisites.go:78-87,99-113`;
- `services/node/internal/app/runtime_install.go:245-256`;
- existing tests repeat a key sequentially; no simultaneous same-key application/SQLite/HTTP test exists.

Impact: the exact retry/double-submit case idempotency is intended to protect can return conflict rather than the original Operation. A client can lose deterministic ownership of its request.

Required closure:

- distinguish duplicate-idempotency from cross-key active-mutation conflicts atomically;
- after a duplicate-key insert race, fetch the key's Operation, validate type/target/request identity, and return it with `Created=false`;
- reject reuse of one key for a different endpoint/type rather than returning an unrelated Operation;
- add simultaneous same-key and same-key/different-request application, persistence and HTTP tests.

### Medium

#### MEDIUM-R1-001 — Raw archive errors still reach the persisted and HTTP-readable log

`HostInstaller.logCommand` now omits stdout, stderr and raw process errors, but source acquisition still calls:

```go
h.debug("source.archive.integrity", "error", err.Error())
h.debug("source.archive.official_unavailable", "error", err.Error())
```

The logger appends to `install.ndjson`; `GET /operations/{id}/log` returns matching lines to Desktop. Network/archive errors can contain URLs, local temporary paths, OS details or raw upstream reasons. This contradicts `SECURITY.md` and the Phase 3 structured-log contract.

Evidence: `services/node/internal/runtime/hermes/host_installer.go:203-207`, `services/node/internal/applog/install.go:26-47,50-72`, and `services/node/internal/transport/httpapi/operations.go:207-228`.

Required closure: replace raw errors with stable code/boolean/category fields and add sentinel raw URL/path/reason exclusion tests across logger file, HTTP response and Desktop.

#### MEDIUM-R1-002 — Managed Node split is improved, but required behavioral tests remain incomplete

The monolithic implementation was correctly split. Current tests still do not proportionally cover:

- successful exact Node ZIP and npm tar materialization/postconditions;
- exact Node/npm argv, working directory and sanitized environment for dependency install;
- Node/npm exact-pin versus merely minimum-compatible behavior;
- extraction entry-count/member-size/uncompressed-size/root-prefix and cancellation cases;
- dependency success/failure/timeout/cancellation with stamp creation/removal;
- whole-Operation deadline behavior;
- daemon restart and component-state transitions through the real adapter/application boundary.

The current `node_managed_test.go` is about 197 lines and concentrates mainly on two version checks, one traversal case, one symlink case, wrong hash, pre-Hermes success, stamp presence and missing Node.

Required closure: add focused adapter/application tests without reintroducing a generic package manager or another large manager file.

#### MEDIUM-R1-003 — Phase 3 governance documents remain materially contradictory

Examples remaining after remediation:

- Phase Spec header still says `Status: READY — OWNER APPROVED` while implementation says `AUDIT`;
- all Exit Criteria remain unchecked despite a declared audit candidate;
- §15 still describes official `node` and `node-deps` as optional 45-second stages even though Amendment 003A3 says they are never spawned and managed dependencies use 15 minutes;
- §16 simultaneously retains text about temporary raw stdout/stderr diagnostics and then forbids raw stdout/stderr;
- §23 says implementation authorization remains pending;
- §25 review record says `Implementation auth: PENDING`;
- §26 still reports implementation commit, exact CI and full verification as pending and names the older branch;
- Amendments 003A1 and 003A3 still say `Implementation: IN_PROGRESS`;
- `PROTOCOL.md` omits `operation.cancelled` from its core event list.

Required closure: reconcile the parent Spec, amendments, roadmap, protocol, completion evidence and audit state without rewriting historical `AUDIT-003 FAIL`.

#### MEDIUM-R1-004 — Documented Phase 3 Operation SSE events are never published

The Spec and protocol require `operation.started`, `operation.progress`, `operation.completed`, `operation.failed` and `operation.cancelled` notifications. The repository contains an event broker and SSE transport, but no install/prerequisite transition publishes any Operation event; the only production reference to `Broker.Publish` is absent. Desktop relies entirely on one-second polling.

Evidence:

- `services/node/internal/events/broker.go:56-65` defines publishing;
- `services/node/internal/transport/httpapi/server.go` exposes the broker only to the SSE handler;
- no production call site publishes Phase 3 Operation events;
- `docs/PROTOCOL.md:272-305` and Phase 3 Spec §13 document them.

Required closure: publish typed, redacted events at committed Operation transitions, document `operation.cancelled`, and test payloads, ordering, disconnect/reconnect behavior and secret exclusion. Resource GET remains the source of truth.

#### MEDIUM-R1-005 — Cross-operation conflict can be attached to the wrong Desktop panel

When starting prerequisites returns `RUNTIME_INSTALL_IN_PROGRESS`, Desktop takes any returned `operationId` and treats it as a prerequisite Operation without validating its type. If the active mutation is `runtime.install`, the prerequisite panel displays the Hermes install as a Node/npm task and exposes its Cancel action there. The prerequisite button also remains available while the local Hermes install is running.

Evidence: `apps/desktop/src/App.tsx:149-180,240-259,307-313` and `HermesPrerequisitePanel.tsx:30-84`.

Required closure: use Operation type/target to route active conflicts to the correct panel, disable/replace the conflicting action while a local install is known active, and add both conflict-direction Desktop tests.

### Low

#### LOW-R1-001 — Managed Node health accepts any minimum-compatible replacement, not the exact managed pin

`inspectNode` and `inspectNPM` accept `>=22.22.0` and `>=12.0.0`, respectively, although Amendment 003A3 defines managed postconditions as exact `22.23.1` and `12.0.2`. An externally replaced higher version under the managed directory is reported `READY` and prevents exact artifact remediation.

Evidence: `services/node/internal/runtime/hermes/node_prereq.go:61-88,125-130` and `release.go:36-46`.

Resolution: either enforce exact versions for the YORVA-managed path or explicitly amend/document a compatible-adoption policy with provenance consequences and tests.

### Info

#### INFO-R1-001 — Inherited Cargo warnings

`cargo audit` reports zero vulnerabilities and 17 already-allowed unmaintained/unsound warnings inherited from the Tauri dependency graph. No new warning was attributed to the Phase 3 remediation.

#### INFO-R1-002 — GitHub Actions runtime deprecation warnings

Exact CI reports three warnings that pinned action releases target Node.js 20 and are being forced to Node.js 24. This did not affect the candidate result, but action pins should be deliberately updated through normal dependency maintenance.

## Accepted Technical Debt

None accepted by this independent auditor. Only the Repository Owner may accept eligible non-blocking findings; the unresolved High findings and required success-flow gaps are not eligible for `PASS WITH CONDITIONS`.

## Required Fixes Before Next Phase

1. Restore Desktop runtime-install reload/recovery and type-safe active conflict attachment.
2. Make retry ownership/pin validation strong enough to prevent overlaying a changed/foreign target.
3. Make concurrent same-key start truly idempotent and reject cross-request key reuse.
4. Repair MSI inspection to prove exact required names, license identities and embedded payload hashes.
5. Remove all raw error persistence/HTTP exposure.
6. Complete the managed Node, deadline, reload and packaging negative test matrix.
7. Implement the documented redacted Operation SSE events or formally amend the approved contract before implementation.
8. Reconcile the Phase 3 Spec, amendments, protocol, roadmap and completion evidence.
9. Commit a clean remediation candidate on a repair branch, run the full local matrix and obtain exact-commit CI plus MSI CI PASS.
10. Preserve `AUDIT-003` and this R1 report, then request `AUDIT-003R2` from a fresh independent review context.

## Gate Rationale

The green exact-commit CI proves that the tested candidate builds and its current tests pass. It does not prove the missing recovery flow, safe retry ownership, correct simultaneous idempotency or MSI embedded-content identity. Those are explicit Phase 3 correctness and supply-chain contracts. With unresolved High findings and material documentation/security findings, `PASS` and `PASS WITH CONDITIONS` are both forbidden.

## Next Step

```text
Phase 3 Implementation: NOT ACCEPTED
AUDIT-003R1 Gate:       FAIL
Phase 3 status:         AUDIT / BLOCKED BY R1 FINDINGS
Freeze commit:          NOT AUTHORIZED
Phase 3 baseline tag:   NOT AUTHORIZED
Feature branch delete:  NOT AUTHORIZED
Phase 4 planning:       BLOCKED
Phase 4 implementation: BLOCKED
```

Do not move or create a Phase 3 baseline tag, delete the feature branch, or change Phase 3 to `COMPLETE / FROZEN` until a later independent re-audit has Gate `PASS`.

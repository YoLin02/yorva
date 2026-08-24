# Phase 6 Runtime Lifecycle and Messaging Channels Audit

## Phase

Phase 6 — Runtime Lifecycle and Messaging Channels, independent Closeout Batch C4.

## Baseline / Candidate

- Baseline tag: `phase-005a1-post-freeze-corrections-baseline`
- Baseline peeled commit: `995777528557fa564a4c42e14f8431b8ddbd20e8`
- Audited branch: `codex/phase6-closeout`
- Audited candidate: `3d2fecfd2c1bc4934f3934914a75efc97ef6b4e5`
- Audit date: 2026-08-24 (Asia/Shanghai)

The candidate was immutable for this review. At audit start, `HEAD` matched the SHA
above and `git status --porcelain=v1` was empty. The baseline tag peeled to the stated
commit. The baseline-to-candidate range contains 15 commits and changes 169 files
(`10,603` insertions, `847` deletions).

## Auditor and Independence

The auditor is a fresh Codex C4 review context. This context did not implement or
remediate the candidate and did not use an implementation summary as proof. It read the
governing documents, both current Phase 6 Spec mirrors, relevant ADRs, historical batch
evidence, handoff and smoke records, then inspected the actual diff, implementation,
tests, migrations and git history. No candidate file was changed during review; this
report is the sole permitted output.

## Gate Decision

**FAIL**

**C5 eligibility: NO. Phase 7 remains blocked.**

The exact-candidate automated gate and package inspection are green, and the recorded
Windows lifecycle/Desktop continuity flows pass. Those facts do not override four
unresolved blocking High implementation findings, an additional blocking evidence
finding, or the explicit C1 `git diff --check` failure. The first failed audit must
remain immutable under the completion prompt.

## Scope and Non-goals

The audited Phase 6 scope is Runtime-neutral Instance lifecycle, the pinned Hermes
lifecycle adapter, Weixin and typed WeCom Channel workflows, ephemeral authentication
delivery, Weixin sender pairing, durable safe projections and Operations, and the Tauri
tray/login-start/single-instance/quit boundary. Phase 7 Runtime upgrade/repair/uninstall,
Skills, MCP, backup/restore, additional Channels, Cloud, remote command and generic
process/service/shell/file control remain excluded.

The ancestry also contains the separately governed final-path generation correction
(`ce1a328`) and general Desktop refinement (`3494033`). The handoff identifies these as
adjacent work rather than Phase 6 delivery. I found no Phase 7 capability in the new API
or Runtime contracts. Their presence broadens the candidate diff and must remain
visible, but it is not treated here as permission to audit or freeze another phase.

## Governing Inputs Read

The review read `AGENTS.md`; `DEVELOPMENT.md`, `ARCHITECTURE.md`, `PROTOCOL.md`,
`RUNTIME.md`, `DATA_MODEL.md`, `SECURITY.md`, `PHASE_GOVERNANCE.md`,
`AUDIT_STANDARD.md`, and `ROADMAP.md`; both Phase 6 Spec mirrors; Phase 5 baseline audit
and correction material; ADR-0003, ADR-0004 and ADR-0006 through ADR-0009; the Batch 1
qualification, Batches 2-8 evidence, implementation/audit handoff, Owner-authenticated
smoke and Windows continuity smoke; and the complete uncommitted completion/audit/freeze
prompt from the permitted read-only source path.

## Verification Evidence

### Independently checked in this audit

- `git rev-parse HEAD`, branch, peeled baseline, status, log, name/status and diff-stat
  checks established the target and range. The worktree was clean before this report.
- Source and test review covered Runtime contracts and registration, lifecycle and
  recovery, Operations/conflicts/cancellation, Hermes lifecycle/Channel/credential/
  pairing adapters, HTTP/OpenAPI/Desktop clients, migrations 009/010, Rust Desktop
  ownership, and the relevant security/static evidence.
- The exact local MSI exists at the recorded closeout path. Read-only measurement found
  `120,209,408` bytes and SHA-256
  `3CFFD45E75D2F41263833DCBCE98B51AABDAF3E58383663156352DFFA1E937CB`, matching the
  supplied exact-candidate package fact.
- `git diff --check
  995777528557fa564a4c42e14f8431b8ddbd20e8..3d2fecfd2c1bc4934f3934914a75efc97ef6b4e5`
  returned exit `2`; details are Finding M-02.

### Exact-candidate external Gate facts

- GitHub Actions run `32682397744` has head SHA equal to the audited candidate and
  completed successfully: Web/API job `97301207868`, Go Node job `97301207985`, and
  Windows Desktop native shell job `97301207948` all succeeded. The Go job includes
  Linux race, vet, vulnerability scan and build; the Windows job includes sidecar and
  lifecycle smoke, negative MSI inspector cases, Rust format/test/audit/clippy/check and
  Tauri no-bundle.
- The local full Gate already completed: pnpm audit/API lint/generation and drift/
  typecheck/lint/98 tests/build; Go test/vet/vulnerability scan/build; Rust format/13
  tests/audit/clippy/check; sidecar/lifecycle smoke; negative MSI tests; empty and Phase
  5 schema migration checks; and public DTO, secret-schema and browser-storage scans.
- The MSI fail-closed inspector passed and all six pinned payload sizes and hashes
  matched its locked inventory.
- Historical run `32459991187` failed on `276991b` because a Linux test used a Windows
  absolute path. Commit `379e8c3` changed the fixture to `filepath.Join` without reducing
  the assertion; the exact-candidate run is green. Commit `790076e` refreshes an already
  enabled login registration so it points to the current executable; the Rust test and
  Windows evidence cover that production correction.

The full unchanged gates were not redundantly rerun in C4. Their exact immutable inputs
were accepted as supplied Gate facts; this audit concentrated on source truth, negative
paths and evidence sufficiency.

### Manual evidence boundary

The Windows record at
`docs/phases/evidence/PHASE-006-WINDOWS-DESKTOP-CONTINUITY-SMOKE.md:37-64` records default
and named lifecycle, restart identity-change/old-gone/one-new postconditions,
close-to-tray, restore, explicit Quit, current-user hidden login registration, missing
registration recreation, single instance, and bounded Desktop-owned daemon exit. It was
run on clean production commit `790076e`; subsequent candidate changes are one
cross-platform test-only correction and evidence documents, while exact-candidate CI
and MSI identity are separately established. This supports the tested production bytes.
The host was an administrative development session, so standard-user token behavior
remains static/CI evidence rather than a manual claim
(`PHASE-006-WINDOWS-DESKTOP-CONTINUITY-SMOKE.md:71-78`).

The Owner record is intentionally weaker. It classifies the real Weixin/WeCom statement
as product-level and not exact-candidate evidence
(`PHASE-006-OWNER-AUTHENTICATED-SMOKE.md:3-6,29-46`). It does not separately establish
the complete Weixin authentication sequence, real sender approval, either local
disconnect, WeCom connected state, exact tested build/instance selection, or a
contemporaneous controlled WeCom redaction comparison
(`PHASE-006-OWNER-AUTHENTICATED-SMOKE.md:48-60,65-93`). No missing step is inferred.

## Dimension Results

### 1. Scope — PASS

The Phase 6 ownership split is recognizable, the Runtime bundle selects lifecycle and
Channel capability, and no Phase 7 or generic remote control surface was added. Adjacent
Phase 2/3 and UI commits are disclosed instead of being attributed to Phase 6. The
broadened ancestry and early-main history are residual governance concerns, not hidden
scope authorization.

### 2. Correctness — FAIL

Normal registered-task Restart can succeed from final `RUNNING` alone (H-02); the
human-output parser can accept contradictory output as truth (H-03); WeCom accepts a
correlated response without an explicit success result (H-04); and Channel cancellation
can race a completed native mutation and leave a terminal `CANCELLED` Operation beside
committed credentials/state (H-05). These affect core Phase 6 truth, not cosmetic paths.

### 3. Architecture — PASS

Runtime-neutral contracts in `services/node/internal/runtime/lifecycle.go` and
`channel.go` contain normalized types only. `runtime/registry.go` carries feature
selection; Hermes-specific command, output, credential and pairing behavior remains in
`runtime/hermes`. The React client uses typed daemon APIs, and Tauri owns native shell
and daemon-process integration rather than ordinary management logic. No generic
process/service/shell/file API was found.

### 4. Security — FAIL

Positive controls include bearer-protected loopback APIs, fixed HTTPS/WSS destinations,
response and QR bounds, session-scoped ephemeral QR retrieval, no QR replay through
SSE, Hermes-native credential authority, write-only pairing approval, closed Channel
types and static no-secret persistence scans. However, H-04 violates ADR-0008's
fail-closed unknown-schema rule before credential commit, H-05 permits cancellation
truth to diverge from credential commit, and H-01 leaves required real-flow/redaction
proof incomplete.

### 5. Data and Persistence — PASS

Migrations 009 and 010 add lifecycle/Channel Operation uniqueness and safe Channel
binding projection with checks, uniqueness and foreign-key cascade. They contain no
credential or transient authentication column. Empty-schema and Phase 5 upgrade checks
passed; the recorded inspected `metadata_json` remained `{}`. Hermes remains the
credential authority and YORVA persists only normalized safe metadata.

### 6. Concurrency and Lifecycle — FAIL

Admission uses Operation idempotency, a database uniqueness constraint and instance/
installation coordination; delete/lifecycle/Channel conflicts and daemon recovery are
represented. Orphan Restart correctly becomes `LIFECYCLE_RESULT_UNKNOWN`
(`services/node/internal/app/lifecycle.go:229-275`). Normal Restart nevertheless lacks
the same transition proof (H-02), and Channel cancellation is not synchronized with the
native/durable commit boundary (H-05).

### 7. Protocol and Compatibility — FAIL

OpenAPI-generated transport models, stable error codes, Channel closed enums and
authenticated session ownership are present. Lifecycle mutation endpoints do not
implement their specified closed `{}` request body, and the Desktop client and OpenAPI
encode the same drift (M-01). The Hermes adapter is pinned to supported version `0.20.2`,
but H-03 and H-04 make the compatibility parsers less fail-closed than the qualification
contract requires.

### 8. Testing and Verification — FAIL

Exact-candidate CI, local gates, Rust/Go/Web test suites, migrations, MSI inspection and
Windows C2 smoke are strong positive evidence. They do not cover the registered-task
Restart false-success negative, contradictory lifecycle output, the real WeCom response
schema, or cancellation at the commit boundary. Required real-flow evidence is missing
(H-01), and the mandatory diff check fails (M-02).

### 9. Maintainability — PASS

The implementation uses cohesive application, domain, persistence, transport and
Hermes-owned modules rather than speculative frameworks. Bounds, stable errors and
operation stages are named. The High findings are localized state-machine/validation
defects that require correction, not evidence that the whole design must be replaced.

### 10. Documentation — FAIL

The architecture/protocol/runtime/data/security docs, two Spec mirrors, ADR-0008 and
Windows/Owner limitation records are detailed and candid. The consolidated handoff is
stale: it still calls `790076e` the current candidate and lists now-completed exact CI
and migration work as pending
(`PHASE-006-IMPLEMENTATION-AUDIT-HANDOFF.md:3-16,159-168`). The adjacent amendment also
breaks the promised diff check (M-02). The early-main deviation is correctly recorded
(`PHASE-006-IMPLEMENTATION-AUDIT-HANDOFF.md:18-25`).

### 11. Dependencies / Supply Chain — PASS

New direct dependencies are bounded to current scope: `qrcode.react` `4.2.0`, Tauri tray
support, `tauri-plugin-autostart` `2.5.1` and
`tauri-plugin-single-instance` `2.4.3`. Lockfiles are committed. pnpm audit,
vulnerability scanning and Cargo audit completed; Cargo reported only the recorded
allowlisted warnings and no denied vulnerability. The MSI pins and independently
verifies its six execution payloads.

### 12. Operations / Diagnostics — FAIL

Operations are durable, emit stable non-secret metadata, do not keep long HTTP requests
open, and have restart recovery. Desktop quit has bounded daemon cleanup. H-02/H-03 can
still turn insufficient lifecycle evidence into operational success, while H-05 can
leave the durable Operation terminal state inconsistent with native/durable Channel
state. These are operational truth defects.

## Acceptance-Criterion Matrix

| Criterion | Result | Evidence / reason |
| --- | --- | --- |
| Immutable target, baseline and clean audit start | PASS | Git checks above. |
| Runtime-neutral Core has no Hermes branch/import | PASS | Runtime contracts and registry inspection. |
| Runtime bundle owns capability selection | PASS | `services/node/internal/runtime/registry.go`; Hermes registration. |
| No generic process/service/shell/file API | PASS | API/OpenAPI and adapter inspection. |
| Public Instance identity resolves to exact authoritative Runtime identity | PASS | Application target resolution plus default/named Windows evidence. |
| Lifecycle capability and live state are exposed | PASS | Runtime bundle/API/Desktop and exact gates. |
| Start/Stop authoritative final-state behavior | PASS | Adapter/app tests and Windows evidence. |
| Restart old exits, one new runs, and success cannot be inferred from old `RUNNING` | **FAIL** | Two Windows observations pass, but H-02 shows the production success predicate is only final `RUNNING`. |
| Unknown/contradictory lifecycle output fails closed | **FAIL** | H-03. |
| Missing login item uses bounded non-persistent path | PASS | Adapter test and Windows evidence. |
| Orphan lifecycle recovery, including no blind Restart replay | PASS | `app/lifecycle.go:229-275` and tests. |
| Lifecycle/delete/config/Channel conflict protection | PASS | Application admission, DB uniqueness and conflict tests. |
| Lifecycle mutation body is bounded closed `{}` | **FAIL** | M-01. |
| Desktop close-to-tray, restore, Quit and daemon ownership | PASS | Windows evidence lines 54-69 and Rust tests. |
| User-login hidden start, refreshed registration, no Instance autostart/elevation | PASS WITH LIMITATION | Exact Windows behavior passed; standard-user token was not manually exercised. |
| Single instance and no second daemon | PASS | Windows evidence and Rust tests. |
| Weixin QR is expiring, initiating-session-only and non-durable | PASS (automated/static) | QR broker, HTTP/session tests and storage scans. |
| Weixin real scan/confirm/connect/disconnect | **NOT ESTABLISHED — BLOCKING** | H-01; product-level attestation does not separately identify these steps. |
| Weixin sender pairing completed from Desktop in the real flow | **NOT ESTABLISHED — BLOCKING** | H-01; deterministic tests are not the required manual fact. |
| Pairing is exact-target, write-only and non-persistent | PASS (automated/static) | Adapter/API/Desktop/security tests and schema/storage scans. |
| WeCom uses typed manual flow and fixed official WSS, not undocumented QR | PASS | Adapter/OpenAPI/Desktop inspection. |
| WeCom verification requires explicit authenticated success before commit | **FAIL** | H-04. |
| WeCom real validation, connected state and local disconnect | **NOT ESTABLISHED — BLOCKING** | H-01; only a product-level result is attested. |
| Cancel clears transient data and cannot commit after terminal cancellation | **FAIL** | H-05. |
| `CONNECTED` remains distinct from lifecycle `RUNNING` | PASS | Separate DTOs/state and Windows observation. |
| No credential/transient authentication material in YORVA DB/API/events/log/storage | PASS WITH LIMITATION | Static scans and available-value inspection pass; contemporaneous proof for the unavailable WeCom value cannot be reconstructed. |
| Active sealed generation is not mutated in place | PASS | ADR-0008 ownership and credential authority path; final-path correction separately governed. |
| Empty and Phase 5 migration paths | PASS | Exact local Gate evidence. |
| Exact-candidate CI and inspected MSI | PASS | Run `32682397744`; MSI measurement/inspector facts above. |
| Mandatory Windows C2 flows | PASS WITH RECORDED LIMITATION | Production commit evidence plus exact successor CI/MSI mapping; standard-user token remains static/CI. |
| Mandatory real-flow evidence | **FAIL** | H-01. |
| `git diff --check` | **FAIL** | M-02, exit `2`. |
| Phase 7 scope exclusion | PASS | Diff/API/Runtime inspection and handoff scope statement. |
| Pre-Gate integration on `main` disclosed without rewriting history | PASS (deviation recorded) | Handoff lines 18-25; see I-01. |

## Findings

### Critical

None.

### High

#### H-01 — Mandatory real Channel-flow evidence is incomplete and not bound to the candidate

The completion prompt makes missing mandatory evidence a blocking example and requires
each unexercised Channel step to remain pending. The Owner record says exact-candidate
classification is not established, then records the complete Weixin sequence, real
sender approval, both disconnects and WeCom connected state as not separately attested,
not evidenced or not reconstructable
(`docs/phases/evidence/PHASE-006-OWNER-AUTHENTICATED-SMOKE.md:3-6,29-60,85-93`). The
available-value inspection is useful but explicitly cannot make the missing
contemporaneous WeCom claim (`:65-83`). This fails Phase 6 Spec sections 26/28 and C2/C4
without asserting that the Owner's narrower product-level statement was false.

Gate effect: blocking High; independently requires `FAIL`.

#### H-02 — Registered-task Restart can report success without proving a restart transition

For a Profile with a login item, `Restart` invokes the official restart verb and then
calls `await(... RUNNING)` (`services/node/internal/runtime/hermes/lifecycle.go:67-90`).
`await` succeeds on any observation of the requested state and retains no pre-mutation
process identity or stop/new-process evidence (`:121-138`). Therefore an ineffective or
partial restart that leaves the old gateway running satisfies the product success
predicate. This is precisely the inference rejected for orphan Restart in
`services/node/internal/app/lifecycle.go:243-249` and by the Phase 6 acceptance row that
requires old-exit/new-running/no-duplicate.

The Windows record is not missing: it observed identity change, old gone and exactly one
new gateway for both tested restarts
(`PHASE-006-WINDOWS-DESKTOP-CONTINUITY-SMOKE.md:41-47`). Those two successful observations
do not make the implementation enforce that postcondition on every operation, and the
adapter test file has no registered-running Restart transition test
(`services/node/internal/runtime/hermes/lifecycle_test.go:13-95`).

Gate effect: blocking High due to possible false lifecycle success.

#### H-03 — Lifecycle status parsing accepts contradictory/unqualified output as authoritative

`parseLifecycleStatus` searches for marker substrings anywhere and returns before
checking manual-state contradictions when the structured process/service pair matches
(`services/node/internal/runtime/hermes/lifecycle.go:152-174`; constants at
`lifecycle_contract.go:4-11`). Output containing the recognized task and running markers
plus the recognized manual-stopped marker is therefore classified `RUNNING`, not
`UNKNOWN`. Extra unqualified or changed text is not fixture-bounded. Existing tests cover
missing individual signals but not contradictory/extra signals
(`services/node/internal/runtime/hermes/lifecycle_test.go:13-41`). This violates the
Spec's unknown/localized/truncated fail-closed contract and can feed false lifecycle
success.

Gate effect: blocking High.

#### H-04 — WeCom verification treats a correlated response with no explicit result as success

The verification parser accepts a response once its request correlation matches. It
rejects only a present nonzero `errcode`; a missing `errcode` falls through to `nil`
(`services/node/internal/runtime/hermes/channel_wecom.go:83-94`). `BeginConnect` then
commits the credentials and returns `CONNECTED`
(`services/node/internal/runtime/hermes/channel.go:61-72`). ADR-0008 requires unknown
response schemas/results to fail closed
(`docs/adr/ADR-0008-hermes-native-channel-credential-authority.md:43-49`). The protocol
test replaces the verifier with a fake and verifies only ordering, not the real response
schema (`services/node/internal/runtime/hermes/channel_protocol_test.go:44-70`).

Gate effect: blocking High because unknown verification evidence can authorize a
credential commit and connected-state claim.

#### H-05 — Channel cancellation can race native/durable commit and leave contradictory terminal truth

`CancelChannel` cancels the worker and immediately writes terminal `CANCELLED`
(`services/node/internal/app/channels.go:260-289`). After the Runtime manager returns
success, `runChannel` does not re-check the context or terminal Operation before writing
the Channel binding with `context.Background()` (`:343-403`); `finishChannel` then sees a
terminal row and returns without repairing it (`:460-475`). At adapter level, both
Channel paths perform their credential write after network/authentication success without
a final cancellation check (`services/node/internal/runtime/hermes/channel.go:52-72`).
Cancellation in that boundary can therefore produce a cancelled Operation while native
credentials and a connected binding are committed. The existing cancellation test only
cancels while its fake is still blocked on the context and returns before mutation
(`services/node/internal/app/channels_test.go:24-36,51-86`).

This violates the Spec rule that a claimed external mutation is cancellable only when it
can be stopped safely and the QR-cancel acceptance criterion requiring no credential
commit.

Gate effect: blocking High.

### Medium

#### M-01 — Lifecycle mutation transport does not enforce the specified closed `{}` body

The handler validates only the idempotency header and never calls the repository's
closed-empty-object decoder (`services/node/internal/transport/httpapi/instances.go:182-201`).
OpenAPI defines no request body for start/stop/restart (`api/openapi.yaml:887-1006`), and
the Desktop client sends no body (`apps/desktop/src/api/client.ts:120-125`). Tests also
construct bodyless lifecycle requests
(`services/node/internal/transport/httpapi/instances_test.go:132,155`). Arbitrary,
trailing or oversized bodies are accepted rather than rejected as required by Phase 6
Spec line 288.

Gate effect: protocol dimension fails; fix before re-audit.

#### M-02 — The mandatory candidate-range diff check fails

The exact C1 command returned exit `2` for trailing whitespace at
`docs/phases/amendments/AMENDMENT-002A4-exact-hermes-version-compatibility.md:3-5`.
Although these Markdown line endings are in adjacent governance material and have no
runtime effect, the completion prompt explicitly requires `git diff --check` to pass.

Gate effect: mandatory closeout check failure; fix without altering the meaning of the
adjacent amendment, then rerun the check.

#### M-03 — The consolidated handoff does not identify the audited candidate or completed Gate facts

The handoff still identifies `790076e` as the current candidate and frames exact CI,
migrations and the complete Gate as pending
(`docs/phases/evidence/PHASE-006-IMPLEMENTATION-AUDIT-HANDOFF.md:3-16,159-168`). The
audited candidate is `3d2fecf`, exact CI is green, and migration/package checks are
complete. This audit records the current facts but does not silently rewrite the source
handoff.

Gate effect: closeout traceability/documentation must be corrected in the remediation
candidate.

### Low

None.

### Info

#### I-01 — Phase 6 production commits reached `main` before an independent Gate

The handoff records the historical deviation and correctly refuses to treat early
integration as acceptance or to rewrite history
(`docs/phases/evidence/PHASE-006-IMPLEMENTATION-AUDIT-HANDOFF.md:18-25`). Phase 7 remains
blocked. This report preserves the deviation; it does not create a second finding merely
from honest documentation.

## Accepted Technical Debt

None. No finding is accepted or downgraded by this audit.

## Limitations and Residual Risk

- The auditor did not possess or request real account credentials and did not perform
  missing real-flow steps. The sanitized evidence boundary is preserved.
- The real-flow attestation has no exact account-test date, candidate/MSI binding,
  selected target, or manual substep transcript. Unknown facts are not filled in.
- Standard-user token behavior was not manually demonstrated; the Windows host context
  is recorded honestly.
- Full green gates were not repeated because the input SHA is immutable and exact run/
  job facts were supplied. The auditor independently checked source, git identity,
  candidate-range diff behavior and MSI size/hash.
- No sensitive transient, credential, pairing, sender or process identifier is included
  in this report.

## Required Fixes Before Freeze / Phase 7

1. Make successful Restart prove the qualified transition: old gateway gone, one fresh
   gateway running, no duplicate; preserve the fail-closed unknown result when proof is
   unavailable. Add registered and non-persistent negative/regression coverage.
2. Replace substring-presence lifecycle parsing with the version-pinned, bounded,
   contradiction-rejecting grammar/fixture policy required by the Spec; add conflicting,
   extra, truncated and changed-output tests.
3. Require the exact qualified WeCom success schema/result before credential commit and
   add real-parser fixtures for missing, unknown, mismatched and explicit failure fields.
4. Synchronize cancellation with Channel native and projection commit. Once the safe
   cancellation boundary is passed, reject cancellation; otherwise prove no native or
   durable commit can occur after terminal cancellation. Add deterministic boundary-race
   tests for both Channel types and disconnect.
5. Enforce and document the bounded closed `{}` lifecycle bodies in HTTP, OpenAPI,
   generated client and protocol tests.
6. Obtain a sanitized mandatory real-flow record that establishes the required Weixin
   and WeCom steps for the remediation candidate, including sender approval and local
   disconnect outcomes, without retaining any sensitive value. If the necessary account
   action remains unavailable, keep it as a Gate blocker.
7. Correct the diff-check whitespace and current-candidate/handoff traceability without
   changing acceptance criteria or erasing this failed audit.
8. Run focused checks, then one complete exact-remediation-candidate Gate and package
   inspection. Use a fresh context for
   `AUDIT-006R1-runtime-lifecycle-messaging-channels.md`.

## Gate Rationale

`AUDIT_STANDARD.md` requires `FAIL` for an unresolved blocking High, failed mandatory
criterion, or insufficient evidence for a critical capability. This candidate has all
three conditions. CI and packaging establish that the candidate builds and its current
tests pass; they do not establish the missing negative properties or real-flow facts.

## Next Step

Preserve this first audit unchanged. Do not enter C5, freeze Phase 6, tag a Phase 6
baseline, or begin Phase 7. Create a bounded Phase 6 remediation plan from the findings,
produce a new immutable candidate, complete the missing mandatory evidence and full
Gate, then assign a fresh independent C4 re-auditor for `AUDIT-006R1`.

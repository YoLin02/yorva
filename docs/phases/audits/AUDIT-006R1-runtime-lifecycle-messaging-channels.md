# Phase 6 Runtime Lifecycle and Messaging Channels Re-audit R1

## Phase

Phase 6 — Runtime Lifecycle and Messaging Channels, independent Closeout Batch C4
re-audit after `AUDIT-006`.

## Baseline / Candidate

- Baseline tag: `phase-005a1-post-freeze-corrections-baseline`
- Baseline peeled commit: `995777528557fa564a4c42e14f8431b8ddbd20e8`
- Immutable first-audit candidate: `3d2fecfd2c1bc4934f3934914a75efc97ef6b4e5`
- Audited branch: `codex/phase6-closeout`
- Audited R1 candidate: `7e1123e216528ae1caf9818b6e0e32b173eeb4d1`
- Remediation code commit: `a9d903364d7e1403d649895d77367a5806be1b0c`
- Audit date: 2026-08-24 (Asia/Shanghai)

At audit start, `HEAD` matched the R1 candidate, the baseline tag peeled to the stated
commit, and `git status --porcelain=v1` was empty. The baseline-to-candidate range has
20 commits and changes 170 files (`11,552` insertions, `862` deletions). The five commits
after the first-audit candidate are the immutable failed audit, one bounded production
remediation and three documentation/evidence successors. The candidate remained
immutable while its source and evidence were reviewed; this report is the sole intended
worktree change.

## Auditor and Independence

The auditor is a fresh Codex R1 review context that did not implement or remediate the
candidate. The review did not accept an implementer summary as proof. It read the
repository rules, audit standard, both Phase 6 Spec mirrors, the completion/audit/freeze
prompt, relevant architecture/security/runtime/protocol/data/governance documents and
ADRs, the immutable first audit, qualification, handoff and smoke records, and then
independently inspected the remediation diff, implementation, tests and exact external
Gate state.

No real account credential, QR value, pairing code, account identifier, PID or other
sensitive value was requested or retained.

## Gate Decision

**FAIL**

**Code-level review: PASS.** No unresolved Critical or High code defect was found in
`9957775..7e1123e`, and the code findings H-02 through H-05 plus protocol findings M-01
through M-03 are closed as detailed below.

**Exact-candidate CI: PASS and not a blocker.** GitHub Actions run `32687028130` is
completed/success for exact head SHA `7e1123e216528ae1caf9818b6e0e32b173eeb4d1`.

**C5 merge/freeze eligibility: NO. Phase 7 remains blocked.** The remaining blocker is
not an inferred code vulnerability or missing CI. Original H-01 remains a blocking High
evidence finding: required real Weixin/WeCom steps and their tested build/Profile mapping
are not established. `AUDIT_STANDARD.md:478-489` requires `FAIL` when a blocking High or
mandatory acceptance failure remains, or evidence is insufficient for a critical
capability. `PASS WITH CONDITIONS` is therefore not available.

## Scope and Non-goals

The re-audit covers Runtime-neutral lifecycle, the pinned Hermes `0.20.2` lifecycle
adapter, Weixin and typed WeCom Channel workflows, ephemeral authentication delivery,
Weixin sender pairing, durable safe projections and Operations, and the Tauri
tray/login-start/single-instance/quit boundary. It focuses on the first audit's H-01
through H-05 and M-01 through M-03, while checking the full baseline range for new
Critical/High code issues.

Phase 7 Runtime upgrade/repair/uninstall, Skills, MCP, backup/restore, additional
Channels, Cloud, remote command and generic process/service/shell/file control remain
excluded. The adjacent final-path generation correction (`ce1a328`) and general Desktop
refinement (`3494033`) remain disclosed rather than attributed to Phase 6. No Phase 7
capability was found in the candidate surface.

## Governing Inputs Read

The review read `AGENTS.md`; `docs/DEVELOPMENT.md`, `docs/ARCHITECTURE.md`,
`docs/PROTOCOL.md`, `docs/RUNTIME.md`, `docs/DATA_MODEL.md`, `docs/SECURITY.md`,
`docs/PHASE_GOVERNANCE.md`, `docs/AUDIT_STANDARD.md` and `ROADMAP.md`; both Phase 6
Spec mirrors; relevant Phase 5 baseline/correction audits; ADR-0003, ADR-0004 and
ADR-0006 through ADR-0009; Phase 6 qualification and implementation records; the
immutable `AUDIT-006`; both Phase 6 smoke records and the consolidated handoff; and the
complete Phase 6 completion/audit/freeze prompt from its permitted read-only source.

## Verification Evidence

### Independent local checks

- `git rev-parse HEAD`, branch, peeled baseline, status, log, range count and diff-stat
  established the exact target and ancestry.
- `git diff --check
  995777528557fa564a4c42e14f8431b8ddbd20e8..7e1123e216528ae1caf9818b6e0e32b173eeb4d1`
  passed with exit `0`.
- `go test ./internal/runtime/hermes ./internal/app
  ./internal/transport/httpapi` passed during this R1 review.
- Source/test review revisited lifecycle Restart and parsing, WeCom verification,
  Channel cancellation/commit ordering, lifecycle HTTP/OpenAPI/Desktop transport, and
  every first-audit finding. The worktree was still clean after those checks.

The complete local Gate was recorded on clean evidence successor `1029f5b` and passed:
pnpm install/audit, OpenAPI lint/generation/drift, typecheck/lint/98 Desktop tests/build;
Go test/vet/build/vulnerability scan; Rust format/13 tests/clippy/check/audit; migrations,
security scans and packaging checks
(`docs/phases/evidence/PHASE-006-IMPLEMENTATION-AUDIT-HANDOFF.md:158-175`). The later
candidate commit updates only the handoff with those exact results and MSI identity.

### Exact-candidate CI

Independent read-only GitHub API inspection established:

- run `32687028130`: `completed`, `success`, exact head SHA
  `7e1123e216528ae1caf9818b6e0e32b173eeb4d1`;
- Web and API contract job `97313876484`: `success`;
- Go Node job `97313876141`: `success`;
- Windows Desktop native shell job `97313876199`: `success`.

Run URL: `https://github.com/YoLin02/yorva/actions/runs/32687028130`.

This closes the earlier pending exact-candidate CI item. Green CI does not replace the
separately mandatory real-account flow evidence.

### MSI and Windows smoke

The remediation MSI is `Yorva_0.3.2_x64_en-US.msi`, 120,213,504 bytes, SHA-256
`B942E637BE9BC59D6C5B603DA117C7AC9646CD254E88F36B1A609A0696FA8EFD`; all six pinned
preparation inputs and the fail-closed MSI inventory inspector passed
(`PHASE-006-IMPLEMENTATION-AUDIT-HANDOFF.md:177-182`). It was built from the same product
inputs as clean checkout `5009781`; its successors through the audited candidate are
evidence-only and do not alter packaged inputs.

The affected Windows re-smoke rebuilt the release executable and sidecar from product
checkout `5009781`, containing remediation `a9d9033`. The strict parser accepted real
Hermes `0.20.2` status output, and default and named Profile Restart each completed with
old identity gone, a new identity alive, a stable gateway count and final Desktop
`RUNNING` (`PHASE-006-WINDOWS-DESKTOP-CONTINUITY-SMOKE.md:87-121`). Earlier exact-build
C2 evidence covers default/named lifecycle, tray restore/Quit, hidden user-login start,
missing-login-item recreation and single-instance restoration; standard-user-token
behavior remains a recorded static/CI limitation (`:32-78`).

### Mandatory real-channel evidence boundary

The Owner record truthfully classifies the supplied real Weixin/WeCom statement as
product-level evidence and explicitly says exact-candidate classification is not
established (`PHASE-006-OWNER-AUTHENTICATED-SMOKE.md:3-13`). It does not bind the account
test to a commit, MSI/hash, date or exact Profile (`:17-33,85-93`). It also does not
separately establish:

- Weixin QR generation, scan and confirmation;
- a real message receiving an AI response;
- a real sender-pairing request and Desktop approval;
- local Weixin disconnect;
- WeCom reaching `CONNECTED` and local WeCom disconnect; or
- contemporaneous exact-value WeCom redaction inspection.

Those boundaries are stated explicitly in the evidence matrix
(`PHASE-006-OWNER-AUTHENTICATED-SMOKE.md:48-60,65-83`). The nearby safe projections and
the broad Owner statement are not promoted into unreported substeps. This is the sole
remaining Gate blocker.

## First-Audit Finding Disposition

| First-audit finding | R1 status | Independent disposition |
| --- | --- | --- |
| H-01 — mandatory real Channel-flow evidence | **OPEN — blocking High** | Product-level attestation remains non-candidate-bound and omits mandatory substeps; see Finding H-01-R1. |
| H-02 — Restart false success | **RESOLVED** | Restart is now Stop, authoritative `STOPPED`, Start, authoritative `RUNNING`; registered/manual positive and failed-stop-transition tests cover it. |
| H-03 — lifecycle parser contradiction/qualification | **RESOLVED** | Exact orthogonal signals, duplicate/contradiction/changed-signal rejection and command exit/stderr/output bounds close the defect. Pinned ancillary diagnostic lines are safely ignored by design; see I-02-R1. |
| H-04 — WeCom missing explicit result | **RESOLVED** | A matched response now requires explicit `errcode: 0`; missing/null/nonzero/malformed/unmatched cases are covered. A `cmd` requirement or unknown-field rejection would be incompatible with the pinned protocol; see I-03-R1. |
| H-05 — cancellation/commit race | **RESOLVED** | Worker ownership holds the per-Instance lock through native mutation, binding projection and terminalization; cancel obtains the same lock and re-reads terminal truth. Boundary tests cover connect and disconnect. |
| M-01 — lifecycle closed `{}` body | **RESOLVED** | HTTP rejects absent, non-object, non-empty and trailing bodies; OpenAPI, generated schemas/client and Desktop send `{}`. |
| M-02 — mandatory diff check | **RESOLVED** | Exact baseline-to-candidate `git diff --check` exits `0`. |
| M-03 — stale handoff | **RESOLVED** | Handoff identifies the immutable first failure, remediation, local Gate, Windows/MSI evidence and the still-pending real-flow state. Exact CI is independently recorded here. |
| I-01 — pre-Gate integration on `main` | **RETAINED — Info** | Historical governance deviation remains candidly recorded and is not treated as acceptance. |

## Dimension Results

| Dimension | Result | Notes |
| --- | --- | --- |
| Scope | PASS | Phase 6 boundaries are intact; adjacent ancestry is disclosed; no Phase 7/generic control surface found. |
| Correctness | PASS | H-02 through H-05 code defects are closed with regression coverage; no new Critical/High correctness issue found. |
| Architecture | PASS | Runtime-neutral Core, registry-owned capabilities, Hermes-owned integration and Tauri/React boundaries remain correct. |
| Security | **FAIL** | Code/static controls pass, but mandatory real-flow and contemporaneous WeCom redaction evidence remain insufficient under H-01-R1. |
| Data and Persistence | PASS | Safe Channel projection and Operation migrations remain deterministic and contain no credential/transient columns. |
| Concurrency and Lifecycle | PASS | Restart transition proof and Channel cancellation/commit serialization close the first-audit races. |
| Protocol and Compatibility | PASS | Closed lifecycle body and explicit WeCom success are enforced; pinned Hermes compatibility behavior is preserved. |
| Testing and Verification | **FAIL** | Focused/local/exact CI/MSI/Windows checks pass; mandatory real-account substeps are not established. |
| Maintainability | PASS | Remediation is localized and uses existing locks/contracts rather than new framework layers. |
| Documentation | PASS | Failed audit and limitations are preserved; handoff and smoke records distinguish exact from product-level evidence. |
| Dependencies / Supply Chain | PASS | No remediation dependency change; recorded audits/scans and pinned MSI inventory are green. |
| Operations / Diagnostics | PASS | Durable terminal truth, authoritative lifecycle transitions and bounded non-secret diagnostics are now coherent. |

## Acceptance-Criterion Matrix

| Criterion | Result | Evidence / reason |
| --- | --- | --- |
| Immutable target, exact baseline and clean audit start | PASS | Git identity/status checks above. |
| Runtime-neutral Core and registry-owned feature selection | PASS | Runtime contracts/registry inspection; no Hermes branch/import in Core. |
| No generic process/service/shell/file API | PASS | API/OpenAPI and adapter inspection. |
| Exact Instance resolves to exact authoritative Runtime/Profile identity | PASS | Application targeting, Profile isolation tests and Windows default/named evidence. |
| Lifecycle capability and live state exposed separately from Channels | PASS | Runtime/API/Desktop inspection and tests. |
| Start/Stop authoritative final-state behavior | PASS | Adapter/application tests and Windows evidence. |
| Restart proves stopped-before-start and final running without duplicate | PASS | `runtime/hermes/lifecycle.go:75-92`, tests `:100-165`, Windows re-smoke `:106-121`. |
| Unknown/contradictory/duplicate/changed lifecycle signal fails closed | PASS | `lifecycle.go:154-213`, tests `:13-53`; exit `0`, empty stderr and output bounds required at `:95-109`. |
| Legal pinned Hermes status diagnostics remain compatible | PASS | Pinned source emits task `Status`/`Last Run Time`/`Last Run Result`; parser ignores only lines carrying no lifecycle signal. |
| Missing login item uses bounded non-persistent path | PASS | Adapter tests, qualification and Windows evidence. |
| Orphan lifecycle recovery, including no blind Restart replay | PASS | Application recovery and regression tests unchanged. |
| Lifecycle/delete/config/Channel conflicts | PASS | Application/DB admission and race coverage. |
| Lifecycle mutation body is bounded closed `{}` | PASS | `httpapi/instances.go:182-205`, tests `:142-182`, OpenAPI and Desktop client `:120-126`. |
| Desktop close-to-tray, restore, Quit and daemon ownership | PASS | Exact-build Windows C2 record and Rust tests. |
| User-login hidden start, refreshed registration, no Instance autostart/elevation | PASS WITH RECORDED LIMITATION | Exact-build behavior passed; standard-user token was not manually exercised. |
| Single instance and no second daemon | PASS | Windows C2 record and Rust tests. |
| Weixin QR is expiring, initiating-session-only and non-durable | PASS (automated/static) | QR broker/API/session/storage coverage. |
| Real Weixin QR scan/confirm/connect and local disconnect | **NOT ESTABLISHED — BLOCKING** | H-01-R1; product-level attestation does not report the individual steps. |
| Real Weixin message receives an AI response | **NOT ESTABLISHED — BLOCKING** | Owner evidence line 54. |
| Real sender pairing request and Desktop approval | **NOT ESTABLISHED — BLOCKING** | Deterministic coverage passes, but the required account step was not evidenced. |
| Pairing is exact-target, write-only and non-persistent | PASS (automated/static) | Adapter/API/Desktop/security tests and schema scans. |
| WeCom typed manual flow uses fixed official WSS | PASS | `channel_wecom.go:23-88`; fixed TLS host and bounded dedicated connection. |
| WeCom requires explicit correlated authenticated success before commit | PASS | `channel_wecom.go:91-103`, protocol tests `:72-120`, commit ordering `channel.go:64-78`. |
| Real WeCom reaches `CONNECTED` and locally disconnects | **NOT ESTABLISHED — BLOCKING** | H-01-R1; product-level statement lacks these recorded postconditions. |
| Cancel cannot overwrite committed Channel truth or commit after cancellation | PASS | `app/channels.go:260-299,353-414`; connect/disconnect boundary tests `channels_test.go:107-204`. |
| `CONNECTED` remains distinct from lifecycle `RUNNING` | PASS | Separate contracts/resources and Windows observation. |
| No credential/transient plaintext in YORVA DB/API/events/log/storage | **NOT FULLY ESTABLISHED — BLOCKING** | Static and available Weixin-value checks pass; contemporaneous WeCom exact-value proof cannot be reconstructed. |
| Active sealed generation is not mutated in place | PASS | ADR-0008 authority path and package evidence. |
| Empty and Phase 5 migration paths | PASS | Recorded complete local Gate. |
| Exact-candidate CI | PASS | Run `32687028130`, exact SHA and all three jobs successful. |
| Inspected remediation MSI | PASS | Size/hash and six-input inventory evidence above. |
| Mandatory Windows lifecycle/Desktop flows | PASS WITH RECORDED LIMITATION | Affected remediation smoke plus retained unchanged C2 evidence; standard-user token limitation disclosed. |
| Mandatory real-channel flow evidence | **FAIL** | H-01-R1. |
| `git diff --check` | PASS | Exact baseline-to-candidate check exits `0`. |
| Phase 7 exclusion | PASS | Diff/API/Runtime inspection and handoff scope statement. |
| Pre-Gate integration on `main` remains disclosed | PASS (deviation recorded) | Handoff lines 26-33; I-01-R1. |

## Pinned Hermes Compatibility Reassessment

### Lifecycle status (original H-03)

Pinned Hermes source commit `df4b65147d7ddd74dd449f9067aabbca5aef0ec7`,
`hermes_cli/gateway_windows.py:1445-1471`, prints one service/login-item signal, may print
dynamic task diagnostics (`Status`, `Last Run Time`, `Last Run Result`), then prints one
process signal. The YORVA parser requires exactly the two orthogonal lifecycle signals
for the registered form or one exact manual signal, rejects contradictions, duplicates
and changed signal-bearing lines, and its caller additionally requires exit `0`, empty
stderr and bounded output (`services/node/internal/runtime/hermes/lifecycle.go:95-109,
154-213`).

Ignoring only ancillary lines that contain no lifecycle signal cannot create a credible
false success in this pinned, fixed-argv, trusted-executable model: success still needs
both exact authoritative signals. Rejecting every diagnostic value would instead make
the documented dynamic status output unusable. The residual behavior is compatible and
is not a defect.

Pinned source:
`https://github.com/NousResearch/hermes-agent/blob/df4b65147d7ddd74dd449f9067aabbca5aef0ec7/hermes_cli/gateway_windows.py#L1445-L1471`.

### WeCom handshake (original H-04)

Pinned `plugins/platforms/wecom/adapter.py:322-360` sends a random `req_id` and accepts
the handshake payload by matching that ID; it does not require a response `cmd`. Pinned
tests also use successful response objects shaped as
`{"headers":{"req_id":"..."},"errcode":0}` without `cmd`. YORVA uses the fixed official
TLS host, one dedicated bounded connection, a cryptographically random 128-bit request
component, exact correlation and explicit `errcode: 0` before commit
(`services/node/internal/runtime/hermes/channel_wecom.go:23-103`). Malformed or unmatched
frames cannot authorize commit and eventually reach the bounded timeout.

Requiring `cmd` or rejecting unknown additive JSON fields would be stricter than the
pinned wire behavior in a way that breaks compatible success responses without adding a
credible protection in this connection/correlation model. YORVA is already stricter
than upstream by rejecting missing/null `errcode`. The current behavior is not a High
or other defect.

Pinned source:
`https://github.com/NousResearch/hermes-agent/blob/df4b65147d7ddd74dd449f9067aabbca5aef0ec7/plugins/platforms/wecom/adapter.py#L322-L360`.

## Findings

### Critical

None.

### High

#### H-01-R1 — Mandatory real Channel-flow evidence remains incomplete and is not candidate-bound

Phase 6 Spec mandatory evidence includes real Weixin scan/confirm/connect/disconnect,
the approved real WeCom path, sender pairing completion, and no plaintext in inspected
logs/SQLite (`docs/phases/PHASE-006-runtime-lifecycle-messaging-channels.md:803-810,
839-850`). The completion prompt additionally requires each unavailable Channel substep
to remain pending and lists missing mandatory evidence as blocking.

The Owner evidence explicitly classifies itself as product-level, not exact-candidate
proof, lacks commit/MSI/Profile/test-date mapping, and marks the individual Weixin QR,
message response, sender approval, disconnect, WeCom connected/disconnect and WeCom
redaction facts as not separately attested, not evidenced or not reconstructable
(`docs/phases/evidence/PHASE-006-OWNER-AUTHENTICATED-SMOKE.md:3-13,29-60,65-93`). This
does not dispute the Owner's narrower statement that real Weixin and WeCom validation
passed; it refuses to invent the missing mandatory observations.

Gate effect: blocking High; independently requires `FAIL`. This is an evidence/acceptance
blocker, not a newly found code vulnerability and not an exact-candidate CI failure.

### Medium

None.

### Low

None.

### Info

#### I-01-R1 — Phase 6 code reached `main` before an independent Gate

The handoff preserves the historical deviation and correctly states that early
integration is not acceptance (`PHASE-006-IMPLEMENTATION-AUDIT-HANDOFF.md:26-33`). No
history rewrite is attempted. Phase 7 and freeze remain blocked by this audit.

#### I-02-R1 — Ancillary pinned lifecycle diagnostics are intentionally ignored

As assessed above, the ignored lines do not carry a recognized lifecycle signal and are
legitimate dynamic Hermes task diagnostics. Exact main-signal validation remains
fail-closed. No code change is required.

#### I-03-R1 — Pinned WeCom success does not require `cmd` or a closed JSON object

The dedicated fixed-host connection and random request correlation match pinned Hermes,
while explicit `errcode: 0` is stricter. Adding a `cmd` requirement or rejecting additive
fields would be an incompatibility, not a security correction. No code change is
required.

## Accepted Technical Debt

None. H-01-R1 is not accepted debt and cannot support `PASS WITH CONDITIONS`.

## Limitations and Residual Risk

- The auditor did not possess real Weixin/WeCom accounts and did not perform or infer
  the missing account-dependent steps.
- The product-level attestation has no exact account-test date, candidate/MSI mapping or
  selected Profile. Current durable state cannot reconstruct ephemeral steps.
- Standard-user token behavior was not manually demonstrated; this limitation remains
  explicit rather than being converted into a failure claim.
- The complete local Gate was not redundantly rerun in R1; focused impacted tests were
  rerun, and the immutable exact-candidate GitHub state was independently queried.
- No sensitive value is included in this report.

## Required Fixes Before Freeze / Phase 7

1. On identified exact candidate/product bytes, record a sanitized real-flow matrix for
   every currently missing mandatory step: Weixin QR generate/scan/confirm/connect,
   message AI response, real sender request/Desktop approval and local disconnect; WeCom
   typed verification to `CONNECTED` and local disconnect; exact tested date/build/MSI/
   Profile mapping; and contemporaneous safe redaction inspection including WeCom.
2. If account action remains unavailable, keep H-01-R1 open. Do not substitute durable
   nearby state, automated tests or the broad product-level statement for missing steps.
3. Preserve both failed audits unchanged. Commit only the sanitized evidence/report
   successor, obtain exact-candidate CI for that successor as required by C3, and assign
   a fresh independent context for `AUDIT-006R2-runtime-lifecycle-messaging-channels.md`.
4. Enter C5 final-main/freeze/tag work only after the fresh audit passes (or a genuinely
   valid Owner-accepted `PASS WITH CONDITIONS` exists), all mandatory evidence is
   recorded, and explicit authorization is present.

## Gate Rationale

The code remediation is technically acceptable and the exact-candidate automated Gate
is green. Nevertheless, the audit standard permits `PASS` only when all mandatory
criteria and required verification pass, and requires `FAIL` for a blocking High,
mandatory acceptance failure or insufficient critical-capability evidence. H-01-R1
meets all three formulations. Calling it a condition would lower the unchanged Phase 6
acceptance criteria after observing the failure.

There is no unresolved Critical/High **code** issue that independently prevents a normal
technical merge. There is an unresolved mandatory **Phase Gate evidence** issue that
prevents treating Phase 6 as accepted, merging/finalizing it through C5, freezing/tagging
the baseline or beginning Phase 7.

## Merge / Freeze Eligibility

- Code-level technical review: **PASS**.
- Exact-candidate CI: **PASS** (`32687028130`).
- Mandatory real-channel evidence: **FAIL / incomplete**.
- Overall audit: **FAIL**.
- Merge/final-main under C5: **NOT ELIGIBLE**.
- Phase 6 freeze/tag: **NOT ELIGIBLE**.
- Phase 7 start: **BLOCKED**.

## Next Step

Stop C5. The Repository Owner or an authorized operator must exercise and record the
missing real-account steps against a precisely identified build without retaining
secrets. After the evidence-only successor has exact-candidate CI, use a fresh auditor
for R2. Preserve `AUDIT-006` and this R1 report unchanged regardless of the later result.

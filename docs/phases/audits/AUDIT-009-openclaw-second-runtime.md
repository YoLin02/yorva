# YORVA Phase 9 Audit

> Current gate: **PASS** for `b958801a91234b5a602d652035e5c4f3e3dc8242`.
> See the final dated re-audit below; preceding FAIL decisions are retained history.

## Phase

Phase 9 — OpenClaw second Runtime, native Windows x64 MVP.

## Baseline / Commit

- Frozen P8: `2a7b842e668011a97804eb40642e3ff9dcab2f04`; working start
  `e95ed31d298c2548556e1295aacf7f84002b74ee` has identical product code.
- Implementation: `a731dcfe4c8de5a8fea559ee28d669c83812702f`.
- Reviewed candidate: `cc36eaa2fa692076fd35547603fd5c297d0e8d13`, including the
  development-dependency security patches. Review covers the actual 78-file diff,
  source, contracts, tests, native evidence and workflow steps.

## Auditor

Codex, fresh single-agent read-only review context. Governing inputs were AGENTS.md,
PHASE_GOVERNANCE, AUDIT_STANDARD, the Phase 9 Spec, ROADMAP, architecture, Runtime,
security, protocol, data model, ADR-0021 and the inherited P8 audit/freeze record.
No subagent was used. No product or test code was changed during the first pass.
This is the solo-project independent review procedure in PHASE_GOVERNANCE §13,
not a claim of an organizationally separate human audit.

## Date

2026-09-10. First pass follows native G1 completion at 11:34:32 UTC.

## Gate Decision

**FAIL — first-pass recommendation; remediation and final G2 evidence required.**

This first-pass decision is retained below any subsequent re-audit. No CRITICAL or
HIGH product defect was identified. The neutral test fixture needs correction,
the affected data-model documentation needs completion, and Windows native/package
jobs are still in progress. Pending checks are not reported as failed execution or PASS.

## Executive Summary

The implemented flow matches the required second-Runtime scope. Real Windows
qualification demonstrates one Hermes and two OpenClaw instances, same-name identity
separation, authenticated Gateway readiness, restart, daemon reconnect with Runtime
survival, isolated stop/delete and no login entry. Optional OpenClaw model/channel,
Skill/MCP, installer, backup mutation and upgrade capabilities remain unavailable.

Core instance dispatch resolves accepted installation identity and the current
Runtime bundle. OpenClaw-specific names, profiles, ownership, command schemas,
ports and Windows Jobs stay in its adapter. P8 authentication, CSP, update integrity,
database schema and production-signing policy were not weakened.

## Verification Evidence

- [Validation record](../evidence/PHASE-009-VALIDATION.md) retains fixture failures,
  fixes, local verification and candidate attribution.
- [Upstream qualification](../evidence/PHASE-009-OPENCLAW-UPSTREAM.md) fixes OpenClaw
  `2026.9.3`, Node `24.16.0`, official sources and measured native CLI behavior.
- `scripts/windows-second-runtime-smoke.ps1`, disposable Windows 11 x64 medium
  integrity, native attempt 12: **PASS** at `2026-09-10T11:34:32.3427700Z`.
  The sidecar SHA-256 is
  `19dee30c1dfd18ac1b741bde832aec1dbc3f353beadbf23e410c931e537cf2a4`.
  It was built from the candidate product source before the commit and records a
  dirty prior VCS revision; it is not represented as the remote MSI binary.
- [CI #102](https://github.com/YoLin02/yorva/actions/runs/34471758200): Go job
  `102853039815` passed full race tests, vet, govulncheck and build; Web/API job
  `102853039456` passed frozen install, dependency audit, OpenAPI lint/generated
  consistency, typecheck, lint, 151 tests and build. Windows job `102853039783`
  remains pending completion at first pass.
- [MSI #40](https://github.com/YoLin02/yorva/actions/runs/34471758204): final
  candidate package/inspection/upload remains pending at first pass.

## Dimension Results

### Scope

PASS. Spec §§3–4 map to the compiled adapter, generic inventory/lifecycle routing,
capability-filtered Desktop and real coexistence smoke. No P10/Control/Fleet,
plugin framework, generic command API or optional feature expansion was added.

### Correctness

PASS. `openclaw/status.go:140` validates config path, both ports, loopback probe
URL, RPC kind/success, version and absence of a native service. `start` transfers
ownership only after authenticated RUNNING; `stop` verifies STOPPED. Delete requires
ownership, the sole contained workspace and no extra agent configuration before
official uninstall and root-absence readback. Native attempt 12 covers the composed
flow. `second_runtime_test.go` covers identity, idempotency scope, capability rejection
and UNKNOWN readback without losing healthy-side management.

### Architecture

PASS for production boundaries. `runtime/instances.go` adds only the four operations
actually used by two adapters. `app/management_target.go` validates Node, accepted
installation, kind and current executable before dispatch. Hermes profile translation
moved into `runtime/hermes/instances.go`; no new concrete Runtime import remains in
the common production paths. The unchanged `app/prereq_adapter.go` is the existing
Hermes-specific installer integration explicitly retained by Spec §6. React calls
typed HTTP; Tauri receives no new business logic. Test-boundary issue M1 is below.

### Security

PASS. `openclaw/command.go` uses fixed direct argv, a constrained environment and
bounded streams; raw errors/output do not enter HTTP. `instances.go` protects default
and external roots, rejects unsafe identifiers/files and uses exclusive ownership
record creation. `process_windows.go` assigns a suspended child to its Job before
resume. Failure/cancellation kills owned descendants; success releases verified
Gateway lifetime. `TestCommandBoundsAndCancellationOwnDescendants`, environment,
ownership and strict-status negatives supplement native medium-integrity evidence.
The local same-user boundary is documented; this is not tenant sandboxing.

### Data and Persistence

PASS. No schema migration or schema deletion. The existing unique constraints on
`(node_id, runtime_kind, install_path)` and `(runtime_installation_id, native_id)`
remain. `sqlite/instances.go` applies fresh native protection/default flags in one
short transaction after CLI work; duplicate snapshot identities reject atomically.
Missing rows retain identity, query failures become UNKNOWN, and no native credential
or PID becomes SQLite authority. Existing migration/tombstone tests pass in full CI.

### Concurrency and Lifecycle

PASS within the existing scoped Operation model. Workers retain cancellation owners;
installation locks are keyed separately for Hermes/OpenClaw, and durable mutation
constraints/idempotency remain. No database transaction spans an external command.
Startup inventory shares a 30-second budget below the Desktop handshake deadline;
the dedicated daemon test proves timeout cancellation without cancelling the server.
Hermes cold-start readiness has a measured 94-second fixture observation, a 120-second
bounded RUNNING wait and a delayed-readiness/cancellation regression. STOPPED remains
15 seconds. Full Go race tests and real reconnect/lifetime/isolation pass.

### Protocol and Compatibility

PASS. OpenAPI and generated TypeScript add boolean Models/Channels capabilities;
the generalized model-provider-presets route preserves the old Hermes URL. Existing
bearer/origin/method/error-envelope handling applies to both Runtime values. Backend
rejection does not depend on the client hiding controls. No native identity, secret
or arbitrary configuration path was added to an ordinary response. Desktop/daemon
ship together; this additive response change retains `/api/v1` compatibility with
the existing Desktop parser. The exact OpenClaw release remains separately qualified.

### Testing and Verification

FAIL at first pass. Native G1, Go race and Web/API checks pass, but final Windows
native/MSI evidence is incomplete and M1 couples the neutral fixture to Hermes.
Existing migration, transport-authentication, lifecycle error, stale Operation,
capability and Desktop recovery tests were inspected as well as the new tests.
UI browser evidence uses disposable fixtures and is not counted as real Runtime
evidence; no unverified viewport size is claimed.

### Maintainability

PASS. OpenClaw is a bounded adapter split by discovery/commands, inventory, lifecycle,
status and platform process ownership. There is no new service layer, external
framework, runtime SDK or persisted PID manager. Common interfaces are justified
by two real implementations. Correct M1 without adding a test framework.

### Documentation

FAIL at first pass for L1. Architecture/Runtime/security/protocol and ADR-0021 describe
the implemented ownership and lifecycle. DATA_MODEL still describes only Hermes
native identity and does not explain independent default/protection flags. In-progress
Spec/validation status is truthful and must be closed with actual final evidence.

### Dependencies / Supply Chain

PASS. Go modules, Rust lockfile and production dependencies are unchanged. Vitest
4.1.11 and the narrow Redocly/js-yaml 4.3.2 override address the original CI security
failure; frozen installation, pnpm audit and Web regression all pass. Official
OpenClaw remains a separately installed pinned Runtime, not an MSI-bundled installer.
Node fixture archive identity and upstream package integrity are recorded without
claiming unperformed npm provenance verification or production signing.

### Operations / Diagnostics

PASS. Existing durable Operations report terminal states and stable errors; no new
raw CLI logging is introduced. UNKNOWN authenticated health remains distinct from
a free/stopped port. Startup recovery warnings identify Runtime kind and keep the
management server accessible. Failed upstream restart/delete primitives and the
Hermes cold-start failure are retained with the resulting bounded implementation.

## Findings

### Critical

None.

### High

None.

### Medium

**M1 — Common test fixture imports the Hermes implementation.**
`services/node/internal/app/instance_inventory_test.go:14,35–36` adds a production
Hermes import and delegates the common fake's `ValidateName` to
`hermes.InstanceManager`. The OpenClaw core tests reuse this fake, so they inherit
Hermes validation and cannot independently prove per-Runtime name-rule dispatch.
This conflicts with AUDIT_STANDARD §11's Runtime-neutral Core-test rule. Replace the
dependency with a configurable fake result, retain the invalid-name rejection
assertion, and verify distinct fake Runtime rules reach the selected instance contract.
Native name rules remain tested in their actual adapters. Owner: current P9 work;
resolution trigger: before final gate.

### Low

**L1 — Data-model explanation omits the new native mapping and protection semantics.**
`docs/DATA_MODEL.md:11–15,96–104` explains only Hermes ownership/profile identity
and names a nonexistent `status` inventory column instead of `availability`. Document
OpenClaw Gateway profile mapping, independent `is_default`/`is_protected` observations,
and the protected-tombstone restriction while preserving the unchanged schema.
Owner: current P9 work; resolution trigger: before final record.

### Info

The preexisting Vite chunk-size advisory is unchanged and does not affect G2. Optional
OpenClaw feature parity, broader version/platform qualification and production release
signing remain outside P9; none is claimed as verified. A failed native setup may leave
a protected partial profile for explicit owner inspection; no recursive cleanup is
inferred from an unverified directory. Remove the narrow js-yaml override when the
specific parent updates its dependency (Owner: repository maintainer; trigger: parent
dependency update).

## Accepted Technical Debt

No correctness or security defect is deferred. M1 and L1 are scheduled for immediate
correction within the authorized Phase 9 audit-fix scope. The limited supported release,
platform and feature set are explicit product scope, not hidden debt.

## Required Fixes Before Next Phase

1. Correct M1 and L1, then recheck Testing/Architecture/Documentation.
2. Collect completed current-candidate Windows native and MSI inspection evidence.
3. Record the final gate and baseline only after G1/G2/G3 pass.

## Gate Rationale

The real success flow and reviewed production boundaries pass. A phase cannot freeze
while a required verification job is incomplete or its test/contract findings are
uncorrected. The first-pass FAIL is therefore a review recommendation, not a claim
that the completed native smoke or completed Go/Web jobs failed.

## Next Step

Perform the bounded test/documentation fixes, run affected tests and append the
re-audit with final CI/package evidence. No next-phase feature work is authorized.

## Additional verification finding — 2026-09-10 11:39 UTC

**H1 — Required Windows bootstrap smoke did not complete.** CI #102 Windows job
`102853039783` passed the OpenClaw/application/daemon tests and encrypted Restore
test, then `scripts/windows-lifecycle-smoke.ps1:45` failed its 45-second handshake
deadline at `11:39:13.4542342Z`. The script did not report which of its five starts
failed or drain stderr during startup, so the failure cannot yet be attributed to
product startup, runner timing or a blocked diagnostic pipe. CI #101, with identical
Go product source, passed this step; that is comparison evidence, not a waiver.

Gate remains FAIL. Preserve the 45-second deadline and all lifecycle assertions.
Add bounded failure reporting and per-scenario startup timing, eliminate stderr
backpressure in the harness, and re-run the required check. Revisit Correctness,
Concurrency/Lifecycle, Testing and Operations/Diagnostics with the resulting evidence.
Do not infer that this is merely a transient runner issue. Owner: current P9 work;
resolution trigger: before final gate.

## Additional verification finding — 2026-09-10 11:50 UTC

**H2 — Command cancellation returns before Windows descendant termination is verified.**
CI #103 Windows job `102857125922` failed
`TestCommandBoundsAndCancellationOwnDescendants` at `command_test.go:256` with
`descendant survived command cancellation`; app and daemon packages passed.
`openclaw/process_windows.go:65–72` only closes a kill-on-close Job. The command
joins its direct child and output readers, but does not explicitly observe the whole
Job becoming empty. The existing regression must remain strict. Add bounded owned-Job
termination/readback before releasing the Job handle; do not add arbitrary PID killing
or extend this change into the unrelated Hermes adapter.

Gate remains FAIL. Revisit Security, Correctness, Concurrency/Lifecycle and Testing;
repeat the native cancellation regression and verify real Gateway lifecycle handoff.
Owner: current P9 work; resolution trigger: before final gate.

### H2 investigation and remediation

The first local correction explicitly called TerminateJobObject and waited for zero
Job accounting, but it was insufficient: cancellation failed 2 of 20 repetitions,
a new success-with-orphan test failed 20 of 20, and a test holding the exact descendant
handle before cancellation failed 3 of 20. The held-handle test confirms this is not
a PID-reuse assertion artifact. It observed a non-signaled process after accounting
reported zero. These failed attempts are retained; the tests were not relaxed.

The corrected implementation captures synchronization handles from its own bounded
Job process list before termination, terminates only that Job, reaps the direct child,
waits for each captured process object to signal within one five-second cleanup
budget, closes those references and checks the remaining Job count. Enumeration or
cleanup uncertainty prevents ordinary command success. Failed Gateway startup uses
the same cleanup; authenticated handoff remains a separate non-terminating detach.

This follows the documented distinction between asynchronous
[TerminateProcess](https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-terminateprocess)
(also used for each process by
[TerminateJobObject](https://learn.microsoft.com/en-us/windows/win32/api/jobapi2/nf-jobapi2-terminatejobobject))
and waiting on a process handle. Process IDs are obtained through the documented
[Job information query](https://learn.microsoft.com/en-us/windows/win32/api/jobapi2/nf-jobapi2-queryinformationjobobject);
no arbitrary PID kill or persisted process authority is introduced.

## Additional source re-audit finding — 2026-09-10 12:10 UTC

**H3 — Closed output streams bypass command cancellation while the process still runs.**
Fresh review of candidate `8e3e03f` found that `openclaw/command.go` leaves its
context-select loop after both readers finish and then calls `cmd.Wait` synchronously.
A CLI that closes stdout/stderr before exiting can therefore keep the worker blocked
beyond its command deadline. The output-completion event is not a process-exit event.
This affects the required timeout/cancellation contract even though ordinary CLI calls
and the existing inherited-pipe regression pass.

Gate remains FAIL. Add a finite external-process fixture that closes both streams
and keeps running, establish the regression, and keep cancellation active while
joining the direct process. Preserve the owned-Job teardown and all existing strict
tests. Revisit Correctness, Concurrency/Lifecycle, Security and Testing. Owner: current
P9 work; resolution trigger: before final gate.

`TestCommandDeadlineAfterClosedStreams` reproduced H3 against the reviewed product:
the fixture closed both streams, remained alive for three seconds, and the command's
500 ms deadline returned only after 3.070685 seconds. The corrected command keeps a
context select around the direct-process wait, joins that waiter on every path, and
then performs the existing H2 Job cleanup. The finite fixture and two-second assertion
bound remain unchanged between the failing and passing runs.

After H3 correction, all four strict Windows command/process tests passed 20 times
each (80 scenario executions, 54.930 seconds). The complete OpenClaw suite passed
in 3.854 seconds and its `go vet` passed. Final candidate CI and real G1 remain
required; preceding candidate successes do not substitute for these checks.

## Final re-audit — 2026-09-10

**PASS — G1, G2 and G3 satisfied for product candidate
`b958801a91234b5a602d652035e5c4f3e3dc8242`.**

This dated decision supersedes the first-pass recommendation and H1/H2/H3 FAIL
decisions above without erasing them. A fresh single-agent context reviewed the
actual final command/process implementation, lifecycle handoff, unchanged strict
tests, neutral fake and per-Runtime name-rule test, affected contracts and governing
documents before any final-record edits. No additional product change was needed.
This follows the solo-project fresh-review procedure; no subagent was used.

### Finding closure

| Finding | Resolution and verification | Final status |
| --- | --- | --- |
| M1 — Runtime-coupled Core fake | `8fd2bcb`: configurable independent name validation; `TestCreateUsesTheSelectedRuntimesNameRules` proves one Runtime can reject a name accepted by another and that mutations reach only the chosen adapter. Original invalid-name assertion retained; final Linux race and native Windows app tests pass. | CLOSED |
| L1 — Incomplete native identity/protection documentation | `8fd2bcb`: DATA_MODEL describes both native mappings, installation-scoped identity, `availability`, independent default/protection and protected tombstones. Actual SQLite schema remains unchanged. | CLOSED |
| H1 — Bootstrap smoke timeout | `8fd2bcb`: harness drains stderr during startup and records scenario timing with bounded failure tails. The cause of the original timeout remains unproven; it is not classified as a harmless runner flake. The original 45-second deadline, health checks and all five scenarios remain. Native attempt 14 and final CI #105 pass. | CLOSED |
| H2 — Descendant exit not joined | `8e3e03f`: retain synchronization handles from the owned Job before termination, reap the direct child, verify process-exit signals and empty Job within one five-second cleanup budget. Uncertain cleanup fails normal command success. All strict process tests pass, and final real G1 verifies successful Gateway lifetime handoff. No arbitrary PID kill or persistent process authority added. | CLOSED |
| H3 — Deadline lost after output closes | `b958801`: the process waiter remains under a context select, cancellation terminates its owned Job, and every path joins the waiter before Job cleanup. The same finite regression failed before the fix and passed afterward, with no relaxed timing assertion. Final CI and real G1 pass on this corrected candidate. | CLOSED |

### Final twelve-dimension review

| Dimension | Decision and evidence |
| --- | --- |
| Scope | PASS — required discovery, profile Instance lifecycle, two-Runtime routing and Desktop flow delivered; optional capabilities and P10 remain outside this baseline. |
| Correctness | PASS — final real G1 covers create, same-name identity, authenticated readiness, restart, reconnect, stop and isolated deletion; H2/H3 regressions and negative adapter tests pass. |
| Architecture | PASS — Instance targets resolve through accepted installation and Runtime bundle; native rules stay in adapters; M1 now uses independent Core fakes. |
| Security | PASS — fixed argv, constrained environment, bounded output, validated profile ownership and authenticated readiness remain; H2/H3 cleanup verified without weakening authentication, CSP, secrets or update controls. |
| Data and persistence | PASS — existing migration/SQLite tests pass; no schema or credential-authority change; fresh native protection and identity reconciliation retain stable scoped IDs. |
| Concurrency and lifecycle | PASS — full Linux race, native Windows process tests, real Runtime survival across daemon reconnect and all five bootstrap scenarios pass; the final command joins output readers, waiter and owned teardown. |
| Protocol and compatibility | PASS — OpenAPI lint and regeneration consistency pass; typed capabilities and stable errors prevent unsupported cross-Runtime dispatch; fixed OpenClaw release qualified. |
| Testing and verification | PASS — final CI #105 and MSI #43 succeed; 151 Desktop tests, native shell tests and final real G1 pass. Repeated local process regressions cover 80 executions after H3. |
| Maintainability | PASS — no speculative plugin system, service framework or new product dependency; fixes stay within command ownership and the existing test harness. |
| Documentation | PASS — actual contracts, ADR-0021, Spec, roadmap, development guide, validation, audit and baseline record agree on delivered capability, source identity and qualification limits. |
| Dependencies / supply chain | PASS — frozen lockfile, pnpm audit, govulncheck and cargo audit gates pass. Go reports zero reachable vulnerabilities, with 17 required-module advisories outside called code; it is not represented as an entirely advisory-free dependency graph. |
| Operations / diagnostics | PASS — durable Operation terminal states, recovery warnings, bounded safe failure diagnostics, encrypted Restore and isolated Desktop/daemon recovery checks pass. |

Final evidence is indexed in the [validation record](../evidence/PHASE-009-VALIDATION.md)
and [baseline record](../evidence/PHASE-009-BASELINE.md). CI #105 and MSI #43 both
test `b958801a91234b5a602d652035e5c4f3e3dc8242`; the final freeze commit changes
documentation/evidence only and retains identical product, test, workflow, script,
API and dependency trees. The annotated phase tag identifies that documentation
successor. No older package is substituted for the final MSI.

### Remaining scope and non-blocking observations

There are zero unresolved CRITICAL, HIGH, MEDIUM or LOW findings, and no correctness
or security defect is deferred. No PASS WITH CONDITIONS acceptance is required.
The existing Vite chunk advisory, development dependency override-removal trigger
and unreachable-module advisory observations remain INFO. Owner: repository
maintainer; revisit during the relevant measured frontend/dependency update. None
is a prerequisite for the next Phase Spec or an excuse to omit a required check.

The supported OpenClaw release/platform is `2026.9.3` on Windows x64. Its optional
model/channel/Skill/MCP/backup/installer/upgrade features are unavailable. Broader
qualification and a production-signed public release remain separate work. The MSI
is an unsigned internal test package. The final G1 used a disposable medium-integrity
Windows guest; normal host profiles were untouched and the host was not rebooted.

The Phase 9 baseline may be frozen under the already-authorized Spec §10. This
record does not start Phase 10, merge main or publish a production release.

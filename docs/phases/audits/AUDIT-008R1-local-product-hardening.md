# YORVA Phase 8 R1 Audit

## Phase

Phase 8 — Local Product Hardening MVP, internal candidate under Owner Amendment 008A1.

## Baseline / Commit

- Prior accepted starting point: `9834a8cb1df9e70502936153943f501ed37cb8fc`.
- Original failed audit: `AUDIT-008-local-product-hardening.md`, preserved unchanged.
- Remediation commits: `ef7eaba811e0139ea6f03649bbc71f8dd27ac589` and
  `43ac29152d3de1fa227937f35ccdd12908bac976`.
- Reviewed product candidate: exact `43ac29152d3de1fa227937f35ccdd12908bac976`.
- This report/evidence/status commit contains no further product-code change.

## Auditor

Codex, fresh single-agent review pass beginning from governing documents and actual
repository source/diff, then checking tests, remote workflow results and guest logs.
No subagent was used and no product code was modified during this review pass.
This is the solo-project fresh-review procedure in PHASE_GOVERNANCE section 13;
it is not represented as an organizationally separate human audit. The Owner authorized
fixes, commit/push and governed internal freeze in Amendment 008A1.

## Date

2026-09-07. Guest qualification completed at 09:24:17 UTC.

## Gate Decision

**PASS** for the internal Phase 8 candidate.

This permits Owner-authorized main integration. Final-main CI, package/update checks
and the annotated phase tag must still succeed before the baseline becomes FROZEN.
It does not establish public-release readiness.

## Executive Summary

Both original findings are closed by product changes and regression evidence. An
available daemon with UNKNOWN inventory now remains queryable but fails the updater's
live recovery postcheck. A process-interrupted download becomes a retryable failure on
restart; active downloads retain their worker ownership and responsive cancellation.

No unresolved CRITICAL, HIGH or MEDIUM finding was identified in the affected review.
Unaffected dimensions reuse the original audit's valid evidence under AUDIT_STANDARD
section 20. Production-signing availability is outside the internal freeze gate by the
Owner's explicit amendment; the existing metadata/package verification controls remain.

## Verification Evidence

The durable detail record is
[PHASE-008R1-REMEDIATION](../evidence/PHASE-008R1-REMEDIATION.md), with the sanitized
[daemon probe](../evidence/PHASE-008R1-READBACK-PROBE.json) and
[Windows serial evidence](../evidence/PHASE-008R1-WINDOWS-UPDATE.json).

| Check | Evidence / result |
| --- | --- |
| Exact-candidate CI | [CI #96](https://github.com/YoLin02/yorva/actions/runs/34103268935), `43ac291`, attempt 1 SUCCESS; Go race/vet/govulncheck, Web/API and Windows native jobs all pass |
| Exact-candidate package | [Windows MSI #35](https://github.com/YoLin02/yorva/actions/runs/34103268868), `43ac291`, attempt 1 SUCCESS, strict inspection and uploaded artifact |
| Native | 33 tests, format/Clippy/check/build pass; includes failed live readback, version mismatch, credential non-forwarding, response bounds, orphan and active-worker cases |
| Desktop | 148 tests in 24 files, typecheck/lint/build; retry after restart and cancellation covered |
| Go / contract | Full Go tests/vet pass; recovery application/transport regressions, OpenAPI lint/generation and no generated drift |
| Windows Happy | Ordinary clean-source 0.4.0 package installed through the qualified 0.3.2 updater; PASS 09:15:22.1675444 UTC |
| Windows ReconcileFailure | 0.4.0 installed, failed Profile readback rejected as UPDATE_POSTCHECK_FAILED, management processes remain available; PASS 09:19:29.1400628 UTC |
| Windows DownloadCrash | Kill with nonempty partial download, restart to FAILED with candidate retained, old version usable, retry to verified 0.4.0 success; PASS 09:24:17.1129966 UTC |
| Retained B1/B2/B3/B5/B6 | Migration/data protection, real guest reboot, MSI lifecycle, sanitized diagnostics and 28,802.669-second three-Profile soak evidence retain their original scope and attribution |

The Windows update tests run at medium integrity in fresh Windows 11 Enterprise
Evaluation 25H2 guest overlays. They use the existing test Ed25519 key and a renewed test
HTTPS certificate, not production signing. The baseline uses the same remediation source
with a disclosed 0.3.2 version overlay; the installed 0.4.0 candidate is an ordinary build
without the qualification feature. Local and remote MSI identities are recorded separately.
The original expired-test-certificate failure remains recorded; TLS was not disabled.

One initial existing 100 ms Go discovery test failed, then passed ten focused repetitions
and the full suite unchanged. The exact-candidate remote race suite also passed. The
initial failure remains disclosed rather than assigned an unproven root cause.

## Dimension Results

### Scope

PASS. Fixes address the accepted B2/B4/B6 findings and their necessary version and UI
postconditions. No P9, second Runtime, Cloud, managed Hermes Upgrade/Rollback, telemetry
or generic file/process API was added. Amendment 008A1 explicitly records the Owner's
internal-freeze boundary; it does not excuse either correctness finding.

### Correctness

PASS. `internal/app/node_recovery.go` performs fresh discovery/readback, rejects UNKNOWN
or errors and rejects loss of a previously accepted Runtime. A fresh Node with no
accepted Runtime may update without installing one. Native `reconcile_postcheck` requires
READY, no error and the running daemon version equal to the installed Desktop version.
Real-daemon and actual installed-MSI negative evidence supplement the native test server.
Interrupted download recovery, preserved data sentinels and verified retry pass in the guest.

### Architecture

PASS. Application code owns Runtime recovery. Authenticated transport exports a narrow DTO;
Rust only verifies the native update postcondition. React consumes typed native status.
No Hermes internal import, Core-to-adapter reversal, speculative abstraction or new backend
was introduced. The existing single-Runtime scope and ownership remain explicit.

### Security

PASS for the internal candidate. `/api/v1/node/recovery` is under ordinary bearer, origin
and no-store middleware and returns stable sanitized failures. The native request disables
proxies and redirects, bounds response size and duration and rejects unknown state/version.
A regression verifies that a redirect target receives no connection carrying a credential.
Download cleanup uses fixed validated regular paths and rejects links/reparse entries.
Metadata signature, fixed source, package hash/size/identity and public unsigned-package
rejection are unchanged. Test certificates are not production provenance.

### Data and Persistence

PASS. R1 introduces no migration/schema or repository change. Schema 017's original
empty/016/current/protection/failure-recovery evidence remains valid and the full Go suite
passes. Live UNKNOWN remains UNKNOWN in management metadata. Native state publication
uses its existing flush and atomic replacement; abandoned-download recovery persists the
terminal failure before cleaning its fixed staging files and retains candidate metadata.

### Concurrency and Lifecycle

PASS. The download holds the same mutex for the complete worker lifetime. Recovery can
transition an orphan only while owning that mutex. A status query uses read-only persisted
state when the mutex is busy, so it cannot cancel/recover a live worker or block its UI
status until completion. Cancellation remains atomic; native status runs off the UI thread.
Recovery calls use bounded HTTP and context cancellation. Existing process containment,
Go race and retained soak evidence continue to cover unchanged management workloads.

### Protocol and Compatibility

PASS. OpenAPI and generated TypeScript include the authenticated additive recovery route.
Daemon connectivity and verified recovery now have distinct documented meanings. Errors
are typed and sanitized; unsupported discovery fails recovery. Sidecar builds validate the
three product-version sources and stamp Go through direct argv build flags, establishing
an exact Desktop/daemon version predicate. Hermes protocol/internals remain unchanged.

### Testing and Verification

PASS. The new regressions cover the states missing in the original audit, including usable
daemon plus failed readback, active versus orphan download and cancellation/retry. Exact
source CI/MSI pass and three real installer workflows pass in disposable Windows. Existing
migration, tamper, lifecycle, diagnostics and soak results are reused only where changes do
not invalidate their workload. Server 2025 CI is not claimed as a Windows 10 client test.

### Maintainability

PASS. A small application method and transport handler use the existing discovery/inventory
and server construction patterns. Native recovery remains inside the updater's existing
state machine. No new dependency, framework, unrelated rename or refactor is in R1. The
sizable native updater remains cohesive; these fixes do not require a general job system.

### Documentation

PASS with the companion status/evidence updates in this audit commit. Protocol, security,
recovery and bilingual Specs describe live postcheck and retry behavior. Original FAIL
history is retained and superseded explicitly by R1. Main/freeze and public readiness are
separate gates, and artifact hashes identify local MSI, remote MSI and artifact ZIP separately.

### Dependencies / Supply Chain

PASS for the internal candidate. R1 changes no lockfile or dependency manifest. CI executes
pnpm audit, govulncheck and cargo audit. The original 17 explicitly allowed Rust dependency
warnings remain disclosed; successful audit exit is not a zero-advisory claim. Existing
pinned resources/actions and MSI inspection remain. Production signer/provenance evidence
is a later public-release requirement under Amendment 008A1.

### Operations / Diagnostics

PASS. Startup logs no longer call UNKNOWN inventory reconciled. An installed update with
failed live readback records UPDATE_POSTCHECK_FAILED while management remains available.
A lost download becomes a terminal retryable error instead of a permanent in-progress
state. Diagnostics retain the reviewed fixed sanitized projection and scoped Save As flow.

## Findings

### Critical

None.

### High

No open finding. **P8-AUDIT-HIGH-001 CLOSED** by `CheckNodeRecovery`, the authenticated
route and native `verify_daemon_recovery`. Evidence: the matching-version/failed-readback
native regression, the real daemon READY → RECOVERY_REQUIRED → READY probe and actual
Windows ReconcileFailure update rejecting success after installing 0.4.0.

### Medium

No open finding. **P8-AUDIT-MEDIUM-001 CLOSED** by mutex-owned abandoned-download
reconciliation, responsive read-only status and UI cancellation/retry. Evidence: orphan/
active-worker native regressions, Desktop cancellation test and actual DownloadCrash
restart/old-version/retry-to-success Windows qualification.

### Low

None newly identified.

### Info

- P8-R1-INFO-001: internal package is NotSigned. Production signing/source and the public
  Windows client qualification matrix remain required before public-release readiness.
  Owner: repository/release owner; trigger: any public release candidate. This is the
  accepted 008A1 scope boundary, not deferred correctness/security debt.
- P8-R1-INFO-002: retain the existing dependency-warning dispositions and initial Go timing
  failure in future candidate review. No test timeout or security policy was weakened.

## Accepted Technical Debt

No new debt. Previously recorded non-blocking dependency dispositions remain owned and
triggered as recorded in B6. Neither original product defect is deferred as debt.

## Required Fixes Before Next Phase

None from R1. Complete final-main CI/MSI/update verification and annotated internal phase
freeze before any separately authorized next-phase work. This report authorizes no P9 work.

## Gate Rationale

The two demonstrated defects have focused fixes, negative regression tests and actual
Windows update evidence. All mandatory internal-candidate checks are satisfied at the
reviewed product source, and no unresolved blocking finding remains. The Owner's explicit
signing-material scope amendment is applied without disabling verification controls.

## Next Step

Commit the audit and evidence, merge the accepted tree to main, verify that exact main
commit through CI and Windows package/update checks, then create the annotated
`phase-008-local-product-hardening-baseline` and record COMPLETE / FROZEN for the internal
baseline. Public Release publication remains outside this task.

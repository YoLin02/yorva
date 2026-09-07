# YORVA Phase 008 Audit

## Phase

Phase 8 — Local Product Hardening MVP, B0–B6 internal-candidate closeout.

## Baseline / Commit

- Reviewed candidate: `bd07bbdb16deaa6972f491a48b9bb76b231da032` on
  `phase/p8-local-product-hardening`.
- Required predecessor: P7 stability revision
  `9834a8cb1df9e70502936153943f501ed37cb8fc` above the unchanged Phase 7 baseline tag.
- Reviewed phase diff: `9834a8c..bd07bbd` (85 files; 11,081 additions, 126 deletions).
- B6 implementation record: `1f2df47cdb5acf1b7f24752c34ea51fb1ffdcba4`.
  Its two successors fix the packaged retention file's line endings (`3eccf63`) and
  recovery-smoke readiness counting (`bd07bbd`); neither changes product behavior.
- This report and companion evidence/status corrections do not modify product code,
  test expectations, acceptance criteria, dependencies, or release configuration.
  CI/MSI evidence below belongs to the exact reviewed candidate, not to a later
  documentation commit.

## Auditor

Codex, one primary agent in a fresh audit task, separate from the earlier implementation
context. No subagents were used. The first pass read the repository, diff, contracts,
tests, remote job results and retained local evidence; it did not rely on the earlier
implementation summary. Product code remained unchanged throughout the pass, following
`docs/AUDIT_STANDARD.md` section 21. The Repository Owner retains acceptance and freeze
authority. This is an internal-candidate audit; it is not a signed public-release audit.

## Date

2026-09-07. Remote job times below are UTC.

## Gate Decision

**FAIL**

## Executive Summary

Exact-candidate CI and Windows MSI packaging now pass, and the retained two-window soak
passes the current evidence analyzer. The previously missing remote evidence is closed.
The first substantive audit nevertheless found one HIGH and one MEDIUM product defect:

1. The updater accepts a usable daemon session as successful authoritative reconciliation,
   although that session can exist while Profile inventory readback is `UNKNOWN`.
2. A Desktop exit during download leaves durable `DOWNLOADING` state with no worker and
   no usable recovery action in the normal UI after restart.

The HIGH finding blocks the B6 internal-candidate Gate as well as public readiness.
Production Windows signing is independently unavailable. Historical scenario PASS results
remain valid for the scenarios and binaries actually exercised, but the statement that
all Critical/High issues are closed is superseded by this audit. Phase 8 remains in
progress; this report does not authorize merge, final-main qualification, tag, freeze,
Release publication or Phase 9 implementation.

## Verification Evidence

### Governing and source inputs

Read `AGENTS.md`, both Phase 8 Specs, `DEVELOPMENT.md`, `ARCHITECTURE.md`, `PROTOCOL.md`,
`RUNTIME.md`, `DATA_MODEL.md`, `SECURITY.md`, `PHASE_GOVERNANCE.md`, `AUDIT_STANDARD.md`,
the P8 roadmap, B1–B6 evidence, support/recovery contracts, and relevant ADRs
0004/0006/0007/0008/0009/0012/0013/0015/0016/0018/0020.

Reviewed the phase changes and their callers/tests for SQLite migration protection and
schema 017, native product-data migration, daemon recovery/readiness, update verification
and recovery, diagnostic generation/export, Settings integration, MSI packaging and
inspection, and the stability/recovery qualification harnesses. Unchanged Runtime
readback behavior was followed where it affects the new startup/update contract.

### Exact-candidate remote checks

| Evidence | Exact SHA / result |
| --- | --- |
| [CI #93, run 34097211094](https://github.com/YoLin02/yorva/actions/runs/34097211094) | `bd07bbd`, attempt 1, SUCCESS; 07:47:27–08:09:29 |
| Web/API contract job | SUCCESS: typecheck, lint, 146 tests, production build, OpenAPI lint/generation/no-drift, dependency audit |
| Go Node job | SUCCESS: race tests, vet, govulncheck and build |
| Windows native job | SUCCESS: disposable Restore/lifecycle smoke, MSI inspector negatives, Rust format/28 tests/audit/Clippy/check, no-bundle build and Desktop recovery smoke |
| [Windows MSI #32, run 34097210917](https://github.com/YoLin02/yorva/actions/runs/34097210917) | `bd07bbd`, attempt 1, SUCCESS; 07:47:27–07:56:39 |
| Packaged MSI | `YORVA_0.4.0_x64_en-US.msi`, 149,909,504 bytes, `NotSigned` |
| MSI SHA-256 from packaging log | `16DA187D7E27EB07C97C3B9151B904622B229AF07C892DB8C8330009EB6C448A` |
| [Uploaded artifact 10009327513](https://github.com/YoLin02/yorva/actions/runs/34097210917/artifacts/10009327513) | `yorva-msi`, 149,712,583 bytes; not expired when checked; expires 2026-12-06 |
| Artifact ZIP digest | `sha256:a46f7e16041abd02f1cb410bc10d71cc3fef05e0edb77dc8bbbbf96241679b70` |

The MSI hash is the file hash logged by the packaging job; the different artifact digest
belongs to GitHub's enclosing ZIP. This audit verified job logs and artifact metadata;
it did not download or install that remote MSI. The Windows CI runner is Windows Server
2025 and does not establish execution on a Windows 10 client.

### Retained qualification and local reruns

The auditor re-read the retained B2 reboot, B3 installer and B4 update guest serial
results. Their successful scenarios are attributable to the original commits/packages
listed in the respective B2–B4 evidence records. B4 Happy/Tamper/Interrupted/InstallerFailure
results use a disposable qualification source and signed metadata, not a production-signed
MSI. B4 Interrupted exercises a live network failure, not a process exit during download.

The B6 pressure and continuity summaries and all 226 samples per window were present.
The current `scripts/windows-stability-evidence.ps1` passed when run against
`.tools/p8-soak/qualification-20260904155949` and
`.tools/p8-soak/continuity-20260904200025`: 28,802.669 seconds combined, three Profiles,
2,260 Operations and 46 diagnostic exports per window, seven pressure reconnects and
one continuity daemon PID. Median continuity growth was 3,706,880 working-set bytes,
3,379,200 private bytes, 28 handles and one thread, within the recorded limits.
No new long soak or shared-host reboot was performed.

The retained B6 disposable Windows 11 Desktop recovery serial log records PASS at
`2026-09-04T16:09:52.1497364Z` with the Desktop and daemon hashes already recorded in B6.
The real Windows 11 reboot/login qualification remains the B2 result. Process replacement
and machine reboot are different evidence, and neither is represented as a new run here.

Additional audit-time checks, all exit zero on the unchanged candidate:

- `go test ./internal/persistence/sqlite ./internal/diagnostics ./internal/transport/httpapi -count=1`;
- `go test ./internal/app -run TestListInstancesQueryFailureIsUnknownNotMissing -count=1 -v`;
- `node scripts/create-yorva-update-metadata.tests.mjs` (three cases, including public unsigned rejection);
- `pwsh -NoProfile -File scripts/inspect-yorva-msi.tests.ps1` (positive plus 17 negative cases);
- combined stability evidence analyzer and `git diff --check`.

An additional isolated daemon probe reproduced the HIGH finding's readiness premise.
Its sanitized output is tracked as
[`PHASE-008-AUDIT-READBACK-PROBE.json`](../evidence/PHASE-008-AUDIT-READBACK-PROBE.json).
The original script/output remain under ignored `.tools/p8-audit-20260907`.
It used only a fresh disposable child environment and direct daemon bootstrap, never
Tauri Known Folders or a normal Hermes Profile. It passed the session credential in the
private bootstrap stream, disabled proxy/redirect forwarding, drained process pipes,
closed the owned daemon and checked its stderr for the session credential.

## Dimension Results

### Scope

PASS. Changes implement the approved local Windows product batches. Managed Hermes
Upgrade/Rollback remains deferred; no second Runtime, cloud/control plane, telemetry
service or generic execution/file API was added. This audit adds evidence only.

### Correctness

FAIL. P8-AUDIT-HIGH-001 permits update success without successful authoritative readback.
P8-AUDIT-MEDIUM-001 prevents retry after a predictable download/process interruption.
The existing positive lifecycle, migration and tamper tests do not close these cases.

### Architecture

PASS. Native installer handoff, identity migration and Save As stay in Tauri; management,
Runtime adapters and persistence remain in Go. React consumes typed native/local APIs.
No new general framework or speculative Runtime abstraction was found in the phase diff.

### Security

PASS for the internal-candidate implementation reviewed. Loopback authentication,
argument-safe installer invocation, fixed update source, metadata signature/hash checking,
public unsigned-package rejection, bounded sanitized export, no-proxy/no-redirect native
diagnostic requests and reparse-path rejection have source/test evidence. There is no
claim of production signing: that public supply-chain gate remains NOT PASS.

### Data and Persistence

PASS. Schema 017 has a deterministic migration and ledger. Empty/current/016 migration,
pre-migration protection, injected failure/restore, foreign keys and repeat startup are
covered by the re-run SQLite suite. Identity migration retains source data and rejects
links/reparse paths. Installer retention evidence is preserved. Inventory query failure
correctly retains metadata as UNKNOWN in Core; the updater's use of readiness is the defect.

### Concurrency and Lifecycle

FAIL. Download work has only in-process ownership/cancellation; durable state survives
its worker without recovery (MEDIUM-001). Existing race, process containment, daemon
replacement and eight-hour resource evidence remain positive for covered paths.

### Protocol and Compatibility

FAIL for the new update postcondition (HIGH-001). A typed UNKNOWN inventory result and
an available bootstrap session have distinct meanings, but the updater treats the latter
as authoritative recovery. OpenAPI generation, typed error handling and the approved
Hermes adapter boundaries otherwise remain consistent in the reviewed diff.

### Testing and Verification

FAIL for gate completeness. CI/MSI and retained soak are positive, but the existing
postcheck test substitutes an unavailable daemon for a failed Runtime readback. It does
not cover a usable daemon with UNKNOWN inventory. Download tests do not cover a lost
process followed by startup/cancel/retry. The public signing gate is also unfulfilled.

### Maintainability

PASS. Native updater, product-data migration and diagnostics have identifiable owners,
and no unused service/repository/plugin abstraction was introduced. The updater is sizable
but cohesive; fixing its explicit state transitions does not justify a broad refactor.

### Documentation

FAIL at the reviewed candidate. B6 and both Specs still say remote CI is pending and
all local gates are complete, while the exact remote checks have passed and this audit
finds a blocking defect. Companion evidence/Spec/roadmap/development status corrections
close this reporting drift, without claiming either product defect is fixed.

### Dependencies / Supply Chain

PASS for the internal dependency review; public signing remains NOT PASS. Native
`ed25519-dalek`, `sha2`, `semver`, `reqwest` and Windows APIs serve current verification
and OS requirements, with lockfiles present. Remote pnpm/govulncheck/cargo audit steps
succeeded. The existing 17 allowed cargo warnings, including upstream maintenance warnings
and the documented non-Windows glib advisory, remain recorded in B6; exit zero does not
mean the dependency graph has no advisories. No dependency change is made by this audit.

### Operations / Diagnostics

FAIL. A download can remain non-terminal after its process disappears (MEDIUM-001),
and update SUCCEEDED can contradict authoritative recovery state (HIGH-001). Diagnostic
bundle schema/bounds/redaction and native atomic export checks otherwise passed. Settings
provides the separate diagnostics/recovery entries required by the Spec.

## Findings

### Critical

None.

### High

#### P8-AUDIT-HIGH-001 — Update success does not require authoritative inventory readback

**Status: OPEN. Owner: P8 updater/daemon recovery implementation. Blocks B6 and phase exit.**

`apps/desktop/src-tauri/src/updater.rs:1023–1040` accepts the installed Desktop version
and `daemon.session() == Ok` as `SUCCEEDED`; no Runtime/Instance readback predicate is
checked. A session does not establish the required postcondition:

- `services/node/internal/app/instance_inventory.go:167–174,207–218` deliberately returns
  a typed inventory with `freshness=UNKNOWN` and an error code, with a nil Go error,
  when Profile listing fails. This is appropriate for a queryable management API.
- `services/node/internal/daemon/daemon.go:275–282` checks only the Go error, then
  advertises a bootstrap handshake at lines 306–310, even for that UNKNOWN result.
- The existing `postcheck_never_reports_success_for_installer_or_reconcile_failure`
  test (`updater.rs:1497–1531`) marks the entire daemon startup failed; it cannot catch
  failed authoritative readback behind an available daemon session.

Independent reproduction on the candidate's retained no-bundle daemon:

1. Build the existing `internal/runtime/hermes/testdata/p8hermes` fixture and place it
   in a fresh disposable `LOCALAPPDATA/hermes/bin`. Bootstrap the real daemon with a
   fresh private data directory, isolated APPDATA/HERMES_HOME and a system-only PATH.
2. Read inventory normally: authenticated handshake accepted, one Instance AVAILABLE,
   inventory FRESH. Close the daemon through its owned stdin.
3. Create an invalid Profile-name directory in that disposable fixture's `profiles`
   directory, causing the real adapter to reject fixture CLI output. Restart the same
   daemon/database and query inventory with the new private session.
4. Observed: handshake still accepted, inventory UNKNOWN,
   `INSTANCE_OUTPUT_UNRECOGNIZED`, retained Instance UNKNOWN.

The checked output records daemon SHA-256
`44CDA742CA1BCA45C74A082296B8B7EF3BE9EC438717E0B3A9F8F65B10D068F2` and fixture SHA-256
`E39170C4CBB92AF67B3EE1DE93C79465A3B10DE874EFC216EB762809E715537E`.
The local no-bundle daemon reports `0.0.0-dev`; this probe establishes the availability
of a session despite failed readback, not a newly executed full MSI update. Given an
otherwise successful matching-version installation, the unconditional success branch
above establishes the erroneous updater result by source trace.

Required correction: expose/check the actual migration and authoritative recovery
postconditions through a narrow typed contract before declaring update success. Keep
unavailable/unsupported Runtime state and diagnostics queryable; merely making every
Runtime failure terminate the daemon would conflate management availability with recovery.
Add a regression using a usable daemon and failed/UNKNOWN readback, plus a matching-version
successful readback case. Revisit Correctness, Protocol, Lifecycle and Tests and run the
affected disposable update qualification. This is a current P8 correctness defect, not
future technical debt.

### Medium

#### P8-AUDIT-MEDIUM-001 — Interrupted download has no restart recovery transition

**Status: OPEN. Owner: native updater. Fix before the renewed B6 Gate.**

`updater.rs:553–555` durably persists DOWNLOADING before network/package work. If Desktop
exits or is killed after that write, the download worker cannot execute its error cleanup
at lines 312–317. On next launch:

- `resume_postcheck` (`updater.rs:404–408`) returns immediately for DOWNLOADING;
- status loading/reconciliation (`updater.rs:267–269,1004–1005,1121–1126`) preserves it;
- cancel (`updater.rs:325–326`) only sets a fresh in-memory atomic flag with no worker;
- `YorvaUpdatePanel.tsx:87–95,105–109` disables check/download while DOWNLOADING; cancel
  at lines 83–84 and 159–160 refreshes the same unchanged record.

The result is a permanently busy updater in the ordinary UI, with no retry after reopen.
The old installed product remains usable, so severity is MEDIUM. The retained B4 network
interruption scenario returns through a live worker and therefore does not cover this
process-interruption state. This finding is established by durable-state and UI source
trace; this audit did not run an additional full Desktop kill-during-download scenario.

Required correction: reconcile an abandoned download into a typed retryable terminal
state, safely clean only updater-owned partial files, and permit a new verified download.
Do not reset an actually running download during a concurrent status query. Add a
persisted-DOWNLOADING/recreated-manager regression and a UI retry check, then validate
process interruption in the disposable update environment.

### Low

None.

### Info

- **P8-AUDIT-INFO-001:** Exact `bd07bbd` CI/MSI evidence is now present. The artifact
  is unsigned; public provenance/signing and signed-package qualification remain open.
- **P8-AUDIT-INFO-002:** Windows 11 disposable evidence and a Server 2025 CI runner
  must not be relabeled Windows 10 execution evidence. Public qualification must resolve
  the Windows 10/11 support matrix explicitly under P8-D1.
- **P8-AUDIT-INFO-003:** Historical eight-hour fixture-backed soak and guest lifecycle
  evidence have been rechecked, not rerun. They do not prove unrestricted real-world
  Hermes workloads or a production-signed candidate.

## Accepted Technical Debt

No newly accepted debt. HIGH-001 and MEDIUM-001 remain current-phase fixes. The existing
upstream dependency warnings retain the bounded disposition documented in B6; this audit
does not waive production signing, platform evidence or owner-controlled release gates.

## Required Fixes Before Next Phase

1. Close HIGH-001 with actual authoritative recovery evidence and regression coverage.
2. Close MEDIUM-001 with crash/restart download recovery and retry evidence.
3. Re-audit affected dimensions on the resulting exact product commit, preserve this
   first report, and obtain that candidate's CI/MSI/update results.
4. Complete production signing/provenance and the supported-Windows qualification
   matrix before public readiness; obtain Owner decisions and final-main evidence
   before any merge/tag/freeze workflow.

## Gate Rationale

`AUDIT_STANDARD.md` section 17 requires FAIL for an unresolved blocking HIGH or an
unmet mandatory correctness criterion. The Phase 8 Spec requires successful authoritative
readback and explicitly forbids reporting failure as success. Green CI and historical
positive smoke results cannot override the demonstrated missing postcondition. Missing
production signing remains an additional public-release blocker rather than the only
remaining condition.

## Next Step

Present this first-pass report for review and handle accepted findings in a separate
fix task, as specified by `AUDIT_STANDARD.md` section 21. Preserve the existing passed
evidence, add regression/failure evidence for the fixes, and issue an R1 audit on the
new product candidate. Until then, retain P8 IN PROGRESS / AUDIT FAIL and do not proceed
to Phase 9 or public release.

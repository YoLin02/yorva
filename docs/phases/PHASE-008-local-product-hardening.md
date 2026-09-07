# YORVA Phase 8 — Local Product Hardening MVP

> Status: **COMPLETE / FROZEN — INTERNAL BASELINE (008A1)**
> Phase: P8
> Required baseline: phase-007-hermes-runtime-management-completeness-baseline plus P7 stability revision `9834a8cb1df9e70502936153943f501ed37cb8fc`
> Planned branch: phase/p8-local-product-hardening
> Product input: YORVA MVP-First P7R–P13 plan, P8
> Execution: one primary agent; automatic commit after each completed Batch Gate

This English document is the execution mirror of
PHASE-008-local-product-hardening.zh-CN.md. The Chinese Spec is the Owner-review source
when wording differs.

## 1. Authorization

This Spec translates the Owner-provided MVP-first plan into the current repository
baseline. The Owner approved this Spec, B0–B6 ordering and per-Batch automatic commits
on 2026-09-03.

P8 uses the P7 stability revision `9834a8c`, merged to `main` on 2026-08-31, as its
code starting point without moving or rewriting the existing Phase 7 baseline tag. The
revision only closes bounded Hermes detect/autostart behavior and removed-Instance record
classification/cleanup; it does not expand the frozen Phase 7 product scope.

Phase 8 does not reopen the managed Hermes Upgrade/Rollback deferred by Phase 7
Amendment 007A1. “YORVA update” means updating the YORVA Desktop, yorvad, supported
database schema and packaged resources.

## 2. Goal

Turn the frozen Phase 7 single-machine feature set into a dependable Windows local
product that a normal user can install, restart, update, recover, diagnose and uninstall
without development tools, a terminal or YORVA Control.

Required user flow:

~~~text
Fresh Install
→ First Launch
→ Configure
→ Run
→ Windows Reboot
→ Recover and Reconcile
→ Upgrade YORVA
→ Migrate
→ Export Sanitized Diagnostics
→ Uninstall with an explicit data policy
~~~

The flow must run on real packages and persistent state. UI presence, API count or mock
success is not completion evidence.

## 3. Baseline facts

P7 handoff chain:

~~~text
phase-007-hermes-runtime-management-completeness-baseline (12b16bc)
→ 64761ac  bounded Hermes launch during detection and timeout stabilization
→ be7b812  separate current Instances from removed records
→ 9834a8c  complete the removed-record cleanup route
→ main
~~~

Candidate `9834a8c` passed GitHub CI run `33365096577`, including Web/API, Go race,
Windows native, Rust and no-bundle build Gates. After merge, final-main CI run
`33366271630` preserved an initial Windows runner handshake-timeout failure and passed in
full on attempt 2; Windows MSI run `33366271629` also passed. The P8 branch contains this
commit and the Owner has authorized implementation.

Reusable Phase 7 foundations:

- Tauri 2, React/TypeScript, Go yorvad and SQLite form the local product path;
- deterministic Windows MSI packaging and inspection already exist;
- embedded ordered SQLite migrations currently end at schema 016;
- daemon handshake, tray, hidden startup and single-instance behavior exist;
- Runtime/Instance Operations and recovery/reconcile foundations exist;
- detection can make one bounded launch attempt when Hermes is installed but stopped;
- externally removed Hermes Profiles are separated from current Instances and their
  YORVA records can be cleared only after authoritative absence is reconfirmed;
- CI covers Web/API, Go race, Windows native, Rust and no-bundle builds.

P8 gaps:

- no frozen stable product identifier/data migration policy;
- Demo MSI evidence does not establish the full installer lifecycle;
- no pre-migration protection and supported cross-version fixture chain;
- no YORVA self-update flow;
- no one-click sanitized diagnostic bundle;
- no 3-Instance 4–8 hour soak evidence;
- no final support matrix, release signing contract or recovery guide.

### 3.1 P8 entry conditions

- `main` contains `9834a8c` and final-main CI succeeds;
- both P8 Specs and `ROADMAP.md` name the same code baseline;
- the immutable Phase 7 tag remains unchanged and the patch is not represented as a new
  P7 capability;
- the Owner approved P8-D1–D9, B0–B6 ordering and automatic Batch commits on 2026-09-03;
- the entry conditions are satisfied and P8-B0 may proceed.

## 4. Approved Owner decisions

| ID | Decision |
| --- | --- |
| P8-D1 | Windows 10/11 x64 is the blocking release target. macOS/Linux validation is informational only in P8. |
| P8-D2 | Freeze one stable product name, identifier, version rule and data directory. A development-identifier transition must migrate discoverable YORVA data explicitly. |
| P8-D3 | Uninstall preserves YORVA user data, YORVA backups and Hermes Runtime/Profile data by default and explains that behavior. |
| P8-D4 | Supported migration inputs include an empty database and the Phase 7 schema 016 baseline. Arbitrary development databases are not promised. |
| P8-D5 | YORVA update uses a complete verifiable installer package, not an incremental patch service. |
| P8-D6 | MVP telemetry is none. A later opt-in system requires separate Owner approval and security-contract work. |
| P8-D7 | Diagnostic export uses a fixed sanitized projection plus a Tauri capability-scoped Save As flow, never a generic file API. |
| P8-D8 | A public-ready candidate requires real signing/provenance evidence. Missing signing material permits completion only as an internal candidate and blocks public-release readiness. |
| P8-D9 | Each Batch commits automatically after focused verification; milestone push/merge/tag/freeze still requires Owner authority. |

## 5. Required scope

- Windows support matrix and stable application identity;
- explicit application-data and uninstall retention policy;
- recoverable migration from supported schema versions;
- Desktop, daemon and reboot recovery with authoritative reconciliation;
- Fresh/Upgrade/Repair/Uninstall/Reinstall MSI lifecycle;
- complete-package YORVA update MVP;
- package provenance, signature, checksum and installed-version verification;
- one-click sanitized diagnostic export;
- security/threat-model refresh;
- three-Instance, 4–8 hour stability verification;
- support/recovery documentation;
- exact-candidate CI, Windows package/update smoke, audit and freeze.

Non-goals:

- managed Hermes Upgrade/Rollback;
- second Runtime, Node, Control, Fleet or remote management;
- formal Linux/macOS support;
- HA, enterprise RBAC, approvals or multitenancy;
- incremental/background update services;
- complex telemetry;
- arbitrary shell/process/file/registry/environment APIs;
- arbitrary MCP execution/configuration;
- 72-hour soak.

## 6. Architecture and ownership

~~~text
React Desktop
    ↓ typed authenticated local contract
Go Application / Operations
    ↓
Domain
    ↑
SQLite / Runtime adapters

Tauri owns only:
    native Desktop/daemon lifecycle
    installer/updater handoff
    code-signing and narrow OS integration
    capability-scoped Save As
~~~

- Hermes remains authoritative for Hermes state.
- SQLite remains authoritative only for YORVA management state.
- Migration success comes from validated new schema/data.
- Recovery success comes from Runtime/Instance authoritative readback.
- Update success requires installed-version, migration and reconcile postconditions.
- Diagnostics never exports the raw database or secret plaintext.

## 7. Batch plan

### P8-B0 — Product support and release contract

Freeze:

- Windows 10/11 x64 prerequisites and support window;
- YORVA/yorvad/schema/Hermes compatibility table;
- stable product name, identifier and synchronized version source;
- development-to-stable identifier data transition;
- application data/log/download/update/diagnostic/backup directories;
- installer lifecycle retention behavior;
- update metadata, package hash/signature and release-source contract;
- no-telemetry decision;
- support and recovery documentation structure.

The frozen B0 result is `docs/PRODUCT_SUPPORT.md`, mirrored for Owner review in
`docs/PRODUCT_SUPPORT.zh-CN.md`. B1 has completed the protected one-time legacy-data
transition and activated the stable identifier.

Gate: configuration/version/identifier consistency, complete ownership table, explicit
schema-016 migration target, and signing availability recorded truthfully.

### P8-B1 — Database migration protection

Implement:

- consistent pre-migration protection when pending migrations exist;
- source/target schema and sanitized state/error reporting;
- empty, schema-016 and current fixed fixtures;
- repeat-safe startup;
- migration failure blocks READY and Runtime mutation;
- proven rollback to the protected database or explicit RECOVERY_REQUIRED;
- post-migration ledger, foreign-key, uniqueness and repository verification;
- bounded protection retention without deleting unknown files.

Gate: empty/latest, 016/latest, repeat, injected failure and protection recovery tests;
focused Go/vet/security checks.

### P8-B2 — Crash, restart and Windows reboot recovery

Startup order:

~~~text
single daemon ownership
→ update/migration state
→ filesystem journal recovery
→ stale Operation recovery
→ Runtime discovery
→ Runtime resource reconcile
→ Instance/lifecycle/channel reconcile
→ READY or typed recovery state
~~~

Cover Desktop crash, daemon crash, Windows reboot, stale Operations, Runtime/Instance
inventory reconciliation, orphan prevention and usable recovery UI. If an installed
Hermes is stopped, YORVA performs one bounded launch and redetection attempt; launch
failure becomes a stable retryable recovery state, never an infinite respawn or a generic
timeout loop. Externally removed, restored or recreated Profiles are reclassified from
authoritative readback, and only a still-absent non-protected record may be cleared.
SQLite cache never substitutes for live Runtime state.

Gate: Desktop kill/reopen, daemon kill/restart, stale-operation fixtures, disposable
Windows reboot/login smoke, stopped-Hermes launch/redetect, launch-failure no-respawn,
external Profile removal/reappearance/cleanup, authoritative readback, no false success
or duplicate daemon.

### P8-B3 — Windows installer lifecycle

Upgrade the existing deterministic MSI foundation to:

- Fresh Install;
- Upgrade Install from a supported YORVA release;
- Repair;
- Uninstall;
- Reinstall with preserved data;
- explicit running-process handling;
- visible data-retention behavior;
- no silent Hermes/Profile/backup deletion;
- stable install identity, shortcuts and uninstall entry;
- exact version/commit/SHA-256/signing evidence.

Gate: all five lifecycle paths on disposable Windows state, negative MSI inspection,
no silent elevation/data deletion and an exact artifact summary.

### P8-B4 — YORVA update MVP

~~~text
fixed metadata
→ version/release-note display
→ full package download
→ provenance/signature/checksum verification
→ Desktop/daemon drain
→ native installer handoff
→ relaunch
→ migration
→ authoritative reconcile
→ final installed-version result
~~~

Use a fixed YORVA-owned metadata source, bounded download/staging, narrow update state and
stable errors. Never accept arbitrary URL/path/command/args/environment/header input.
Do not build incremental or background update infrastructure.

Gate: real older-supported-to-candidate update; tamper rejection before execution;
download interruption leaves the old product usable; installer/migration/reconcile failure
never reports success; final version and Runtime/Instance readback pass.

### P8-B5 — Sanitized diagnostic bundle

Fixed bundle:

~~~text
manifest.json
version.json
node-summary.json
runtime-summary.json
instance-summary.json
operations.json
schema.json
logs/*.ndjson
redaction-report.json
~~~

Add an independent Settings subpage and one-click export. Bound record count, time range
and total size. Exclude keys, tokens, channel/MCP credentials, QR/pairing values, cookies,
authorization data, ambient environment, raw database and arbitrary user files.

Gate: schema/bounds tests, secret canaries, atomic Save As, cancel/failure semantics,
partial cleanup, OpenAPI/Go/Tauri/Desktop checks and a real exported-bundle inspection.

### P8-B6 — Stability, release and phase Gate

Exercise three Hermes Instances for at least four hours and target eight hours for the
final release candidate:

- parallel start/stop/restart;
- model write/readback;
- managed Skill lifecycle;
- reviewed-Preset MCP bind/test/unbind;
- Channel status;
- Runtime backup create/verify/delete;
- Desktop close/reopen, forced daemon termination, bounded daemon replacement and
  authenticated reconnect;
- periodic diagnostic export;
- authoritative state reconciliation after each process-level recovery.

The shared development host is not rebooted during B6. Machine-reboot coverage reuses
the completed disposable Windows 11 reboot/login evidence from B2; B6 adds repeatable
process-failure recovery and two isolated four-hour windows without replacing that
machine-level evidence.

Observe crashes, deadlocks, orphan processes, goroutines/handles/memory/CPU, log/Operation/
staging growth, stale state and secret leakage.

The internal freeze Gate includes dependency/security review, exact-candidate CI/race, MSI with its actual signature state recorded,
installer/update/migration/diagnostic smoke, the existing disposable reboot smoke, soak
summary, focused independent
audit and Owner decision.

The B6 internal-candidate Gate may commit after all P8 behavior/readback/failure/smoke
checks, the four-hour minimum soak, the complete local Gate, Go race, dependency audit,
support/recovery documentation, and closure of all Critical/High findings pass. Missing
remote or signing evidence remains explicit and cannot be recorded as PASS. Exact-commit
CI, the Windows MSI workflow, independent audit, Owner authorization, merge, final-main
CI, tag and freeze remain phase-exit Gates. Owner Amendment 008A1 removes production
signing availability from the internal P8 freeze prerequisites; it remains a separate
public-release requirement.

## 8. Dependency order

~~~text
B0 Support Contract
→ B1 Migration
→ B2 Recovery
→ B3 Installer
→ B4 YORVA Update
→ B5 Diagnostics
→ B6 Stability / Release Gate
~~~

One primary agent follows this order by default because Tauri, daemon, OpenAPI and docs
are shared surfaces.

## 9. Desktop information architecture

Use the existing flat Settings language and separate detailed workflows:

~~~text
Settings
├─ About YORVA
│  ├─ version
│  └─ updates
├─ Data and Recovery
│  ├─ data policy
│  └─ schema/migration state
└─ Diagnostics and Support
   ├─ runtime summary
   ├─ recovery state
   └─ export diagnostics
~~~

Do not append all P8 controls to one long page.

## 10. Contract and failure rules

P8 APIs remain typed, authenticated and loopback-only. Long work uses Operations or
narrow native updater state. Public reads return no secret, raw SQL/log content, arbitrary
path or installer transcript. UI behavior depends on stable codes.

Required error classes include migration backup/failure/recovery, reconcile failure,
update metadata/download/integrity/install/postcheck failures and diagnostics
export/redaction failures. Exact names freeze with the relevant OpenAPI Batch.

## 11. Mandatory stop conditions

Stop the affected Batch if:

- supported data may be lost or irrecoverably migrated;
- stable identity strands old data without migration;
- repair/uninstall may silently delete YORVA/Hermes data;
- update provenance/signature/checksum cannot be verified;
- updater requires an arbitrary execution or filesystem surface;
- cached SQLite state is presented as live after reboot;
- diagnostics leaks any secret or the raw database;
- failure is displayed as success;
- real MSI/update/reboot smoke cannot run;
- the four-hour minimum soak fails or exposes unexplained crash/deadlock/unbounded growth.

## 12. Internal candidate, public exit and freeze

The B6 internal candidate may be committed when its local Gate above passes. That commit
does not change the phase to COMPLETE/FROZEN and does not claim public-release readiness.

Under [Owner Amendment 008A1](amendments/AMENDMENT-008A1-internal-freeze-signing-boundary.md),
P8 enters its internal-candidate audit and phase-exit Gate after support policy, real
installation, schema-016 migration, crash/reboot recovery, installer lifecycle, one real
YORVA update, sanitized diagnostics, three-Instance soak and exact-candidate CI/Windows
evidence pass. Obtaining/generating production signing material is outside this P8
freeze Gate. Existing update integrity and signature-policy checks remain mandatory.
Production signing/provenance qualification is required separately before public release.

After audit PASS and explicit Owner authorization:

1. merge to main;
2. run final-main CI and Windows package/update Gate;
3. create annotated tag phase-008-local-product-hardening-baseline;
4. mark Spec/ROADMAP COMPLETE / FROZEN;
5. stop; do not begin P9 without a separately approved Spec.

## 13. Execution record

| Batch | Status | Commit | Gate |
| --- | --- | --- | --- |
| P8-B0 | COMPLETE | `72ec369` | Bilingual support contract, Owner decisions, directory/retention/signing consistency review passed |
| P8-B1 | COMPLETE | `fc224a8` | Schema 017; protected/verified/recoverable 016→017; identifier data migration; Go/Rust focused Gate passed |
| P8-B2 | COMPLETE — R1 PASS | `3ee4580` + `ef7eaba` + `43ac291` | Historical reboot/recovery plus live READY/UNKNOWN/READY and installed-candidate failed-readback rejection pass |
| P8-B3 | COMPLETE | `56df01f` + `412d3c0` | Exact clean-source 0.4.0 MSI; static/negative inspection and disposable Windows Fresh/Upgrade/Repair/Uninstall/Reinstall Gate passed |
| P8-B4 | COMPLETE — R1 PASS | `6a75cde` + `ef7eaba` + `43ac291` | Retained tamper/interruption/installer-failure evidence; exact-source Happy, ReconcileFailure and DownloadCrash/retry pass |
| P8-B5 | COMPLETE | `5ca5f0a` | Fixed sanitized ZIP schema and bounds; authenticated daemon endpoint; capability-scoped atomic Save As; canary/cleanup/UI/non-MSI Gate passed |
| P8-B6 | COMPLETE — INTERNAL GATE PASS | `43ac291` + R1 audit/evidence | Retained eight-hour soak and R1 PASS; final-main `2a7b842` CI #98 / MSI #37, real update/retry and annotated tag all PASS |

## 14. Owner approval record

On 2026-09-03, the Owner confirmed:

1. Windows 10/11 x64 is the sole blocking release target;
2. MVP telemetry is none;
3. uninstall preserves all YORVA/Hermes user data by default;
4. public Windows code-signing material is not currently available, so P8 is capped at
   internal-candidate status and cannot claim public-release readiness;
5. the soak minimum is four hours and the final-candidate target is eight hours;
6. B0–B6 sequential implementation and automatic commits after each passed Batch Gate
   are authorized.

On 2026-09-04, the Owner additionally required B6 not to restart the shared Windows
host. Process-level crash/reconnect tests and the already completed disposable-Windows
reboot evidence cover the two recovery layers separately.

On 2026-09-07, the Owner authorized commit/push on the existing P8 branch to obtain
CI/MSI evidence and close out the internal-candidate review. This does not authorize
merge/main, tag, freeze, Release publication, a shared-host reboot or use of normal
Hermes Profiles. Execution remains single-agent.

## 15. Audit closeout — 2026-09-07

Exact candidate `bd07bbdb16deaa6972f491a48b9bb76b231da032` passed
[CI #93](https://github.com/YoLin02/yorva/actions/runs/34097211094) and
[Windows MSI #32](https://github.com/YoLin02/yorva/actions/runs/34097210917), attempt 1.
The MSI is `NotSigned`; its SHA-256 and artifact identity are recorded in
[evidence/PHASE-008-B6-STABILITY-RELEASE.md](evidence/PHASE-008-B6-STABILITY-RELEASE.md).
The retained two-window eight-hour soak passed the current evidence analyzer without a
new long run or host reboot.

The fresh single-agent internal-candidate
[AUDIT-008](audits/AUDIT-008-local-product-hardening.md) returns **FAIL**.
HIGH-001 permits update success without successful authoritative Profile readback;
MEDIUM-001 leaves a process-interrupted download stuck after Desktop restart. B2/B4
affected cases and the B6 Gate are reopened. The earlier completed scenario results
remain evidence, but internal-candidate Gate completion is no longer asserted.

This first audit pass changes no product code or acceptance requirement. Accepted fixes
and affected-dimension re-audit must precede renewed acceptance. Production signing and
the public-release qualification/Owner/final-main/tag/freeze Gates remain separate and
unfulfilled. The phase remains IN PROGRESS, with no Phase 9 authority.

## 16. Owner Amendment 008A1 — internal freeze authorization

On 2026-09-07, after reviewing AUDIT-008, the Owner authorized fixing the findings,
committing and freezing P8, and explicitly removed obtaining/generating production
signing material as a P8 prerequisite. See
[AMENDMENT-008A1](amendments/AMENDMENT-008A1-internal-freeze-signing-boundary.md).
The earlier same-day commit/push-only limitation is superseded for the governed phase
freeze actions. Public Release publication, Phase 9, host reboot and normal-profile
mutation remain outside scope. Both findings still require fixes, verification and R1.

## 17. Remediation and R1 acceptance — 2026-09-07

[AUDIT-008R1](audits/AUDIT-008R1-local-product-hardening.md) returns PASS for exact product
candidate `43ac29152d3de1fa227937f35ccdd12908bac976`, closing HIGH-001 and MEDIUM-001.
The original FAIL audit and earlier status records above remain historical. B2/B4/B6
are accepted again as the internal candidate. [R1 evidence](evidence/PHASE-008R1-REMEDIATION.md)
records exact-source CI/MSI, real failed-readback rejection and download-crash/retry success.
Owner Amendment 008A1 authorizes main integration and freeze after final-main Gates.
Current state is PASSED, pending final-main CI/MSI/update and annotated baseline tag.
No production signing material is required for this internal phase freeze; no public
Release or Phase 9 work is authorized.

## 18. Formal internal baseline freeze — 2026-09-07

**COMPLETE / FROZEN.** The annotated tag
`phase-008-local-product-hardening-baseline` points to accepted main commit
`2a7b842e668011a97804eb40642e3ff9dcab2f04` and has been pushed and verified remotely.
[Final-main/freeze evidence](evidence/PHASE-008-FINAL-MAIN-FREEZE.md) records exact CI #98,
Windows MSI #37, Desktop recovery, real update and original-failure retry PASS before tag
creation. This documentation-only closeout follows the tag and changes no product code.
The earlier PASSED/pending statements are historical steps in that acceptance chain.
Production signing material is outside this internal P8 freeze prerequisite under 008A1.
Public Release and Phase 9 remain separately gated; no next-phase work is started.

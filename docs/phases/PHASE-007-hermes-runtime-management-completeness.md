# YORVA Phase 7 — Hermes Runtime Management Completeness

> Status: **COMPLETE / FROZEN — 2026-08-27**
> Phase: 7
> Owner: Repository Owner
> Plan date: 2026-08-24
> Required baseline: `phase-0065-developer-led-demo-baseline` →
> `5f68e48f17e7e342e1781b37613b19d4bd1f060b`
> Branch: `codex/phase7-hermes-runtime-management`
> Chinese Owner-review Spec:
> `docs/phases/PHASE-007-hermes-runtime-management-completeness.zh-CN.md`
> Qualified development target: stable `>=0.20.2 <0.21.0`; B1 reference snapshot
> Hermes `0.20.5` / `a0ca7c19204e514f9590ce3b812e029b315ab9e9`
> Authorization: P7-D1–D8 and B0–B10 approved 2026-08-24. ADR-0013 and
> ADR-0015 accepted by the Owner on 2026-08-25. Exact-version
> qualification remains required before each product capability becomes true.

This file is the execution mirror of the Chinese Owner-review Spec. The Chinese Spec
remains authoritative if wording diverges. Any material scope, security, persistence,
or batch-order change must update both files before implementation continues.

## 1. Objective and product boundary

Phase 7 makes Hermes practical for daily local management without requiring a terminal
or direct Hermes file editing. It adds:

- normalized Runtime and Instance health, bounded diagnostics and safe logs;
- trusted Skill inventory, inspection, audit and lifecycle management;
- approved MCP catalog/configuration/authentication/testing management;
- backup/create/restore with an approved encryption and recovery design;
- managed Hermes generation upgrade and qualified rollback;
- feature-specific recovery after failure, cancellation or daemon restart.

It does not mirror every Hermes command and does not create a generic command, file,
process, service, environment or plugin execution platform. Sessions, Cron, Memory,
Plugins, Portal, Computer Use, Hooks, Approvals, Kanban, Projects and other Hermes
product surfaces require a later Owner-approved amendment.

Hermes owns Hermes state. YORVA owns management Operations, policy, safe projections,
backup indexes, audit metadata and Desktop experience. YORVA remains a Runtime control
layer, not a Hermes fork.

## 2. Entry gate

B0 is complete:

- P6.5 is `COMPLETE / FROZEN` at the required annotated tag;
- this branch starts exactly at that tag's peeled commit;
- the original developer worktree and temporary artifacts were not imported;
- P7-D1–D8 and B0–B10 are Owner-approved;
- the Chinese Spec and this execution mirror are synchronized.

B1 evidence controls each feature lane rather than acting as a global serial lock. A
rejected direct Hermes surface remains unavailable, while non-overlapping parser,
descriptor, verifier, transaction and focused-test work may proceed in parallel. Shared
B2 contracts remain an integration prerequisite. A public or mutating MCP, Backup,
Restore or Upgrade capability still requires its applicable accepted ADR and qualified
closed path.

## 3. Owner decisions

| Decision | Accepted direction |
| --- | --- |
| P7-D1 | Scope is Health/Logs, Skills, MCP, Backup/Restore, Upgrade and recovery UX. Other Hermes surfaces are later work. |
| P7-D2 | Skills use inspected/audited official or Owner-approved catalog identities, plus an explicitly native-selected local ZIP/directory copied through the bounded prose-only importer. No unchecked direct URL, caller path or force bypass. |
| P7-D3 | P7R MCP supports built-in and YORVA-reviewed Presets/Definitions only. Caller-provided stdio command, args, environment, HTTP headers, paths and arbitrary JSON are deferred. |
| P7-D4 | Prefer an encrypted Runtime-scope complete backup. Do not ship plaintext full Hermes backup if a standard encrypted format and reliable Restore cannot be qualified. ADR required. |
| P7-D5 | Managed upgrade creates a new generation under ADR-0006/0009. Never mutate the active tree with `hermes update` or `--force-venv`. ADR required. |
| P7-D6 | Health/logs expose allowlisted categories, fixed bounds and mandatory redaction only. No arbitrary path tail or raw log export. |
| P7-D7 | P7 keeps one authenticated `LOCAL_DESKTOP` actor with typed actions. Principal/Grant/RBAC remains P9/P10 scope. |
| P7-D8 | Stable `0.20.x` may be detected, but each P7 feature reports capability only after exact-version surface qualification. Unknown contracts fail closed. |

ADR-0013 (encrypted Runtime backup/Restore) and ADR-0015 (managed generation
Upgrade/Rollback) were accepted by the Owner on 2026-08-25. The Owner deferred the
ADR-0019 fully custom MCP surface on 2026-08-27; it is not part of P7R MVP authority.
Acceptance does not replace lane-specific qualification and destructive-flow evidence.

ADR-0016 was accepted by the Owner on 2026-08-25. It authorizes the exact-Profile
Hermes-native `API_SERVER_KEY` boundary for authenticated, loopback-only,
no-redirect `/health/detailed` and `/v1/skills` management reads. Product capabilities
still require focused qualification and wiring before they may become true.

ADR-0018 was accepted by the Owner on 2026-08-25. B4 now separates Hermes-native
capability truth from a YORVA-managed lifecycle. Unreliable Hermes-native mutations stay
`deferred_upstream`; YORVA owns a small managed Skill store and copy projection into the
exact Profile.

## 4. Qualified-surface facts and B1 questions

Read-only inspection of official Hermes `0.20.5` confirms command entries for status,
doctor, logs, monitoring status, JSON security audit, Skills, MCP, backup/import and
update check/plan. Entry-point existence is not product qualification.

B1 must prove for each proposed feature:

- non-interactive behavior and exact Runtime/Profile scope;
- structured or narrowly parsable output and strict bounds;
- stable success, failure, timeout and cancellation semantics;
- safe credential transport and unique credential authority;
- concurrency, child/process/network cleanup and postconditions;
- compatibility behavior for external Hermes changes;
- whether the surface is documented and safe enough to expose.

Official full/quick backup contains secret-bearing `.env` and authentication state.
Plaintext ZIP output is therefore not an acceptable YORVA backup product design.

## 5. In scope

### 5.1 Health, logs and security

- normalized `HEALTHY`, `DEGRADED`, `UNHEALTHY`, `UNKNOWN` state;
- static Doctor checks and a separate explicit live/deep Operation;
- supported Hermes dependency/MCP/Plugin security audit projection;
- fixed-category, fixed-line, fixed-byte, fixed-time-window redacted log snapshots;
- explicit unsupported, malformed, timeout, partial and unknown behavior;
- no durable SQLite, Operation or audit-log copy of log contents.

`UNKNOWN` never authorizes automatic repair.

### 5.2 Skills

- exact Runtime/Profile inventory with bounded inspect;
- separate Runtime-native inventory/mutation capability truth;
- a YORVA-owned managed-copy SSOT under the daemon data directory;
- install/update/remove only from compile-time approved inspected sources;
- enable/disable through copy projection, not Hermes native Profile configuration;
- one package digest and ownership marker bound to the SQLite deployment record;
- external/bundled/unknown Skills remain visible and read-only;
- drift detection, authoritative projection read-back and bounded recovery.

Hermes remains authoritative for native Skill observation. YORVA is authoritative only
for its managed copy, ownership record and expected Runtime projection.

### 5.3 MCP

- live configured inventory plus built-in and YORVA-managed Runtime Definitions;
- one production-visible YORVA-owned loopback lifecycle Preset (`yorva-mcp-test`)
  with fixed identity and the single `yorva_ping` tool;
- reusable Definition creation from YORVA-reviewed Presets, Profile binding, deletion and bounded connection/tool test;
- no caller-defined stdio command, args, environment, HTTP headers or arbitrary MCP JSON in the P7R MVP;
- safe browser/device/OAuth flow only if initiating-session isolation is proven;
- closed tool-selection mutation, remove, reauthentication and reconciliation;
- timeout/cancel cleanup for process and network work.

Fully custom MCP is deferred to a separately qualified post-P7R scope. P7R Desktop/API
requests select reviewed Presets and only the credential and Tool Scope declared by that
Preset; they cannot submit executable, argv, environment, headers, paths or raw config.
Credentials remain write-only and Profile-scoped.

### 5.4 Backup and Restore

- explicit Runtime or qualified Instance scope, format version, Runtime version,
  timestamp, size, checksum and state;
- fixed per-user YORVA system application-data destination; no caller path and no automatic Cloud upload;
- create, verify, list and delete;
- Restore format/scope/version/checksum/space/conflict preflight;
- pre-Restore protection point, stop plan, authoritative post-check and rollback;
- corrupt, truncated, tampered, unsupported and insufficient-space handling;
- traversal, absolute path, reparse/symlink, ADS, member-count and expansion bounds;
- explicit secret/session/account inclusion and plaintext-staging lifecycle.

Backup/Restore product code is blocked until P7-D4's ADR and recovery qualification are
accepted. `hermes backup` and `hermes import --force` are not automatically safe.

### 5.5 Managed upgrade and rollback

- compare current managed generation with the exact snapshot carried by this YORVA build;
- present version, source, impact, backup and gateway restart plan;
- mutate only a YORVA-managed installation;
- exact source/size/SHA-256/license/installer inputs;
- new Install Transaction and final-path generation construction;
- seal, functional validation, publish and `active.json` compare-and-swap activation;
- complete Instance/lifecycle/model/channel/Skill/MCP reconciliation;
- retain the previous generation and roll back only when user-data compatibility is proven;
- explain unmanaged official checkout without modifying the user's Git worktree.

### 5.6 Recovery

- reconcile orphaned Operations from Runtime truth after daemon restart;
- never blindly replay a destructive mutation;
- expose bounded retry, resume only when proven safe, rollback or manual recovery;
- retain a stable error code, failed stage, correlation ID and safe diagnostics;
- never treat command exit zero as the sole success postcondition.

## 6. Authentication and typed actions

P7 retains the authenticated local Desktop trust context. Each use case has one stable
action and actor `LOCAL_DESKTOP`:

```text
runtime.health.read
runtime.logs.read
runtime.security.audit
skill.read / skill.install / skill.update / skill.remove / skill.configure
mcp.read / mcp.install / mcp.authenticate / mcp.test / mcp.remove / mcp.configure
backup.read / backup.create / backup.restore / backup.delete
runtime.upgrade.plan / runtime.upgrade / runtime.rollback
```

The action enters the application use case and safe audit metadata, never Hermes argv.
P7 does not introduce Principal, Grant, role tables, enterprise RBAC or Channel-sender
authorization. A future Control Plane must authorize before delivery and the Node must
re-apply local policy to the same typed action/use case.

## 7. Architecture

Required direction:

```text
React Desktop
  → authenticated typed local API
  → Runtime-neutral application use case
  → compile-time Runtime feature lookup
  → Hermes-owned adapter
  → qualified official surface or separately approved narrow fallback
```

Core/application owns typed intent, stable identities, Operations, idempotency,
conflicts, cancellation, recovery decisions, normalized state/errors and safe audit
metadata. The Hermes adapter owns exact command/API selection, Profile/global targeting,
parsers, Runtime-native configuration/credentials/paths and authoritative postconditions.

Add small feature contracts only after B1 proves real callers. Do not create a giant
`HermesManager`, dynamic plugin framework or generic command abstraction.

Desktop information architecture is Runtime-centric. The Runtime page owns the Instance
inventory, shared Skills/MCP resources and multi-Instance assignment, Upgrade,
Backup/Restore, diagnostics and Operations. The Instance surface is limited to lifecycle,
model, Channels, Skill/MCP bindings and exact-Instance health/logs. This reuses the
existing Runtime/Instance APIs and does not change capability, credential authority or
qualification boundaries, or relabel Instance health as Runtime-wide health.

The Runtime management UI separates Skills and MCP into distinct pages. The Skills page
uses one exact "Configuring" Instance selector instead of duplicate target/assignment
selectors. Skill inventory and MCP
definition inventory read the exact existing Hermes Profile state. The compatibility
read remains inside the Hermes adapter, and the MCP projection returns only names and
normalized configuration state—never URL, command, args, headers, environment values,
or credentials to Desktop/API.
Runtime resource pages use a flat settings-style layout: no nested decorative card
frames around resource rows, compact separators, and explicit ownership grouping.
YORVA-managed Skills remain visible and manageable; Hermes/runtime/external Skills are
grouped separately, collapsed by default, and remain read-only unless a native mutation
surface is separately qualified.
Diagnostics does not occupy a Runtime top-level navigation tab. It opens from each
Instance row and allows direct Instance switching within the diagnostics view. The view
projects the exact Profile's partial health snapshot and fixed-category, bounded,
redacted local runtime logs; an authenticated loopback health result replaces the
partial Profile health observation when available.

## 8. Protocol rules

B1 evidence locks final paths before B2 changes OpenAPI. Candidate resources cover
health/deep checks/security audit/logs; Instance Skills and MCP; Runtime backups and
Restore; and Runtime upgrade-plan/upgrade/rollback.

The current OpenAPI locks the first static health projection to the exact Instance:

```text
GET /api/v1/instances/{instanceId}/health
GET /api/v1/instances/{instanceId}/logs?category=RUNTIME|ERRORS|GATEWAY|MCP
```

This is not relabeled as Runtime-wide health. Runtime-wide health stays unregistered
until a separately qualified Runtime target exists. Deep/live checks and security audit
remain explicit Operations; no synchronous GET shortcut is registered.

All mutations use closed typed bodies and `Idempotency-Key`. Secrets are write-only.
Source identifiers, presets, tool selection, log category/filter and every mutation
field are allowlisted. Long work returns `202 Operation`. Desktop uses TanStack Query for
daemon state and SSE only for invalidation/progress.

P7R MCP Definition requests select a reviewed Preset and its declared credential and Tool
Scope only. They do not accept executable, argv, environment, headers, caller paths, raw
configuration or plaintext secret readback.
A system-managed backup destination remains a narrowly scoped local capability and cannot
silently become a future remote file command. The daemon derives it from trusted app data
and its own backup ID; React and HTTP never submit a path.

The Runtime-scoped backup index may be exposed independently as an authenticated read-only
capability once its repository is available. List/get return last-observed safe metadata
only; they do not open, decrypt, hash, reconcile, or mutate an artifact. Backup create,
delete and Restore capabilities remain independently false until their complete qualified
Operation paths pass their gates.

## 9. Operations and conflicts

Candidate Operations include deep health check, security audit, Skill mutation/audit,
MCP install/auth/test/remove, backup create/restore/delete, Runtime upgrade and rollback.

Minimum conflicts:

| Operation | Conflicts |
| --- | --- |
| Skill mutation | Same-scope Skill mutation, Restore, Upgrade |
| MCP mutation/auth/test | Same-server mutation, Restore, Upgrade, and lifecycle transition when required |
| Backup create | Restore, Upgrade; live-gateway concurrency is decided by qualification |
| Restore | Every affected Instance mutation/lifecycle/Channel/Skill/MCP/Backup/Upgrade |
| Upgrade/Rollback | Install, prerequisite, every affected Instance mutation, Restore and Skill/MCP mutation |
| Log/health read | Normally concurrent; deep/live checks use a separate bounded budget |

Application coordination owns the narrowest real lock. No database transaction remains
open during network, Hermes, archive, MCP or process waits. Every goroutine, child,
stream and network request has an owner, timeout, cancellation and cleanup path.

## 10. Persistence and Secret boundary

Hermes inventory/config/log/health remains authoritative; do not create a shadow DB.
Potential P7 persistence is limited to closed Operation types, safe typed action audit
metadata, a scope-correct backup index and OS-backed secret references after ADR approval.

The current backup schema is Instance-scoped while the official Hermes backup is
Runtime-home-scoped. P7-D4 must resolve this before migration design. Migrations must be
deterministic from empty and formal P6/P6.5 predecessor schemas and must not reinterpret
or delete existing rows silently.

Backup keys/passwords, MCP OAuth/header credentials, registry credentials and update
tokens never enter argv, URLs, SQLite plaintext, Operations, events, logs, diagnostics,
audit metadata, Desktop storage or ambient child environments. Logs require a second
YORVA redaction layer and strict content/line/byte/time/concurrency bounds.

Skill installation cannot bypass a blocked scan. Restore extraction rejects traversal,
absolute paths, reparse/symlink, ADS and expansion bombs. Upgrade uses immutable reviewed
input and never runs `--force`, `--force-venv` or an active-tree mutation.

## 11. Batch plan and gates

| Batch | Delivery | Gate |
| --- | --- | --- |
| B0 | Accepted P6.5 baseline, clean successor, decisions, bilingual Specs | **PASS — 2026-08-24** |
| B1 | Exact Hermes `0.20.5` surface qualification and risk map | Preserve evidence; a NO-GO closes that direct route rather than all P7 work |
| B2 | Minimal capability contracts, registry flags, typed actions, Operation/error state, protocol skeleton and only decided migrations | Go contract/API/migration tests + OpenAPI drift |
| B3 | Normalized Health/Logs/Security and Desktop | Parser/bounds/redaction/timeout/API/Desktop gate |
| B4 | YORVA-managed Skills lifecycle + Hermes native capability truth | Source/ownership/conflict/Profile isolation/drift/restart/manual smoke |
| B5 | MCP | Reviewed Definition + Instance Binding lifecycle, closed request schemas, credential isolation, timeout/cancel and authoritative read-back smoke |
| B6 | Backup Create | Secret/temp/crash/archive integrity/space/manual smoke |
| B7 | Restore | Corrupt/tamper/version/cross-scope/partial-failure/destructive smoke |
| B8 | Managed Upgrade/Rollback | Exact source/final path/CAS/data compatibility/lifecycle/channel smoke |
| B9 | Complete UX and recovery | End-to-end Desktop and daemon-restart recovery |
| B10 | Candidate, full Gate, Windows smoke, exact CI, independent audit/fix/re-audit and Owner Gate | PASS before merge/final-main/tag |

Execution is dependency-driven rather than purely serial. B2 owns shared contracts.
Non-overlapping B3–B8 lanes may proceed concurrently when they do not depend on an
undecided contract or ADR. B6 precedes B7, and B8 depends on a verified protection point
and exact data-compatibility evidence. Each lane runs focused tests; B10 alone performs
the full immutable-candidate audit. A non-critical lane defect does not freeze unrelated
lanes, while security/data/cross-scope/arbitrary-execution violations close the affected
surface immediately.

## 12. Verification matrix

B1 and later tests must cover at least:

- unsupported/malformed Runtime output with capability false or `UNKNOWN` and no mutation;
- static/deep/live health semantic separation;
- oversized/malicious/secret-bearing logs with truncation, redaction and no script injection;
- default/named Profile Skill isolation, blocked scan, malicious source and partial update;
- MCP malformed Definition/JSON rejection, secret redaction, direct-argv and HTTP
  timeout cleanup, plus `CONFIGURED` versus `READY` distinction;
- encrypted/approved backup handling, crash cleanup, checksum/tamper/traversal/bomb rejection;
- Restore running-state conflict, protection point, partial failure and rollback truth;
- upgrade never mutating the active generation, final-path validation, activation CAS,
  post-check and data-compatible rollback;
- daemon restart during every Operation without blind destructive replay;
- deterministic concurrency conflict with race/deadlock coverage;
- safe typed actor/action audit metadata;
- empty and formal predecessor migrations.

Use focused gates per batch. Run the full project Gate once for the frozen Phase 7
candidate. Packaging/MSI runs only when product or packaging inputs change.

Focused gates are development verification, not independent phase audits. The dedicated
fresh-context auditor reviews the integrated B10 candidate after all selected lanes are
complete.

## 13. Audit and acceptance

B10 applies all twelve `AUDIT_STANDARD.md` dimensions and focuses on qualified surfaces,
architecture boundaries, arbitrary execution prevention, Skill supply chain, MCP Secret
authority/session cleanup, backup encryption/archive safety, Restore conflict/rollback,
new-generation Upgrade, user-data compatibility, log redaction/XSS, typed-action truth,
migrations and exact-candidate evidence.

Phase 7 acceptance requires every scoped Desktop flow, exact capability qualification,
Profile isolation, no force/command escape, an approved non-plaintext backup design,
pre-mutation tamper rejection, authoritative Restore/Upgrade postconditions, immutable
active generation, restart recovery, bounded safe diagnostics, migrations, full Gate,
Windows smoke, exact CI and an acceptable independent audit.

## 14. Mandatory stop conditions

Stop the affected dangerous surface and return it to the coordinator/Owner if any item
below applies. Unrelated lanes continue unless the shared trust boundary or candidate is
compromised:

- the branch is not a successor of the formal P6.5 baseline;
- an official surface cannot be non-interactive, scope-exact, bounded and verifiable;
- implementation requires Hermes internal Python imports or an undocumented state DB;
- MCP requires untyped JSON, shell-string construction, plaintext secret persistence or an unowned child process;
- MCP/backup/upgrade Secret authority cannot be uniquely defined;
- full backup can only produce an unencrypted credential/session/account archive;
- Restore cannot prove archive safety before mutation or define failure/rollback truth;
- Upgrade requires active-generation mutation or `--force-venv`;
- user-data migration makes previous-generation rollback unsafe;
- logs cannot be bounded/redacted without leaking conversation/account/token/path data;
- a material architecture/security change lacks an accepted ADR;
- required exact-candidate, Windows or destructive-flow evidence is unavailable.

Never weaken source, Secret, archive, postcondition, migration or audit standards to pass.

## 15. Freeze boundary

After B10 Gate and Owner authorization: freeze an immutable candidate, preserve failed
audits, re-audit fixes in fresh context, merge only after PASS, require final-main CI and
create/verify annotated tag
`phase-007-hermes-runtime-management-completeness-baseline`.

Then stop. Phase 8 requires its own approved Spec.

## 16. P7R MVP closure decision and B0 audit

On 2026-08-27 the Owner approved P7R MVP closure on the current Phase 7 branch:

- each completed P7R batch is committed automatically after its relevant Gate; push,
  merge, tag and freeze still require separate authorization;
- implementation prioritizes real mutation, authoritative Runtime read-back, real
  execution and truthful failure feedback, without per-UI-change ADRs or evidence files;
- Secret confidentiality, explicit destructive targets/results, the prohibition on
  arbitrary remote Shell/Process/File APIs, architecture boundaries and Runtime state
  ownership remain hard constraints;
- the MCP MVP accepts only YORVA-reviewed Presets/Definitions. Caller-provided stdio
  command, args, environment, HTTP headers, paths and arbitrary MCP JSON are deferred.

P7R-B0 observed the following implementation state:

| Surface | Current implementation | P7R gap / next batch |
|---|---|---|
| Runtime Workspace | Overview, Instances, Skills, MCP, Maintenance and Operations exist; Diagnostics is reached from Instance entry points | Add Models and complete Runtime-resource versus Instance-binding separation |
| Skills | install/update/enable/disable/remove, ZIP/directory import and Runtime multi-Instance selection are wired | Add real qualified catalog items, Drift/reprojection and real read-back smoke |
| MCP | reviewed test Preset, install/auth/test/configure/remove and Profile read-back exist | remove/close the uncommitted arbitrary STDIO/env/header surface; converge on reviewed Definition + Binding and add a real Preset smoke |
| Backup/Restore | authenticated Operations, encrypted container, system app-data destination, Restore and stable errors exist | run real create/read-back/restore/delete smoke after all Hermes processes are stopped |
| Upgrade | Plan, HTTP/Application Operation contracts and Desktop entry exist | daemon has no real Upgrader/Rollbacker binding; implement fixed candidate Generation, protection point, activation, post-check and rollback |
| Models | still centered on native per-Instance configuration | add Provider Connection, Model Profile, Runtime Default, Instance Override and batch Copy-on-Apply |

The P7R-B0 Gate passes only when these facts match the public contract, the prohibited
custom MCP surface is absent from the commit and focused checks pass.

P7R-B1 Runtime/Instance scope closure is implemented as follows:

- Runtime Workspace navigation is Overview, Instances, Models, Skills, MCP,
  Maintenance and Operations. Diagnostics remains intentionally hidden from the tab
  strip and is opened from an exact Instance row.
- Models now has a Runtime Workspace entry with an explicit Instance selector and reuses
  the existing authoritative Hermes Profile model configuration flow. Shared Provider
  Connections, Model Profiles and Runtime defaults remain B2/B3 work and are not faked.
- Instance management retains only exact-Instance lifecycle, model and channel entry
  points, Skill/MCP bindings, health and logs. Runtime upgrade, backup and restore remain
  Runtime-only.

The P7R-B1 Gate requires focused Desktop tests, TypeScript typecheck, lint and a non-MSI
Desktop build before its automatic commit.

P7R-B2/B3 shared model resources are implemented as real Runtime resources:

- Runtime Provider Connections store one write-only credential in the OS-backed SecretStore;
- reusable Model Profiles reference reviewed Provider presets and allowlisted model IDs;
- Runtime Default and exact-Instance `INHERIT`/`OVERRIDE` bindings are separate resources;
- multi-Instance application is a durable `model.profile.apply` Operation with one
  result per Instance and Hermes authoritative readback before success;
- existing Hermes-only model configuration remains external and is not silently adopted.

The B2/B3 Gate requires migration, application, HTTP/OpenAPI, Desktop client/component,
secret non-disclosure, typecheck/lint and non-MSI build checks before automatic commit.

P7R-B4 closes the MVP Skills loop without changing Hermes-native ownership:

- the reviewed catalog now includes a useful digest-verified prose-only document-review Skill;
- ZIP and directory imports continue through bounded validation and immutable managed storage;
- a missing YORVA-owned projection is visible as Drift and has an explicit reproject action;
- modified/conflicting or external destinations remain fail-closed and read-only;
- reproject uses the existing durable `skill.enable` Operation and authoritative readback.

The B4 Gate requires managed store/application tests, Desktop interaction tests,
typecheck/lint and non-MSI build before automatic commit.

P7R-B5 closes the restricted MCP MVP without retaining a future custom execution surface:

- Runtime Definitions are the safe projection of the compile-time reviewed Preset registry;
- Instance Binding Operations install, authenticate when declared by the Preset, configure
  Tool Scope, test, remove and reconcile against the exact Hermes Profile;
- the application and Runtime contracts no longer contain dormant arbitrary Definition,
  stdio command/argv, environment, header, endpoint, path or named-secret mutation fields;
- the YORVA-owned loopback test Preset proves create, Profile write, handshake, authoritative
  read-back, Tool Scope update, retest, second-Profile binding and deletion/absence read-back;
- `MCPRead`, `MCPMutate` and `MCPTest` are derived from the actually registered reader/manager.

The B5 Gate requires focused application/Runtime/HTTP tests, the tagged production MCP
lifecycle qualification, Go vet, API drift checks, Desktop tests, typecheck/lint and a
non-MSI build before automatic commit.

P7R-B6 closes the currently applicable Runtime maintenance MVP:

- encrypted Runtime Backup create/read/restore/delete remains wired on Windows and defaults
  to `<YORVA app data>/backups`; the Desktop and HTTP request contain no destination path;
- the device key remains OS-backed and a backup is indexed only after encrypted publication,
  checksum and authoritative verification succeed;
- the local product log recorded a successful real backup creation on 2026-08-27 and the
  corresponding encrypted artifact exists under the default application-data backup folder;
- when a supported external/development Hermes reports the same version as the packaged
  candidate, the read-only plan now truthfully reports `UP_TO_DATE` instead of unrelated
  managed-upgrade evidence gaps;
- matching-version status never grants Upgrade/Rollback mutation authority. With no newer
  packaged candidate there is no version transition to execute; older managed versions
  continue to require complete protection, compatibility and post-check evidence.

The B6 Gate requires Backup/Restore and Upgrade planner/application/HTTP tests, full Go
tests/vet, Desktop maintenance tests, typecheck/lint and a non-MSI build before automatic
commit. Destructive Restore against the Owner's live Hermes data remains an explicit B7
manual smoke action rather than an automatic test.

P7R-B7 automated integration and handoff checks are complete:

- the full Go test and vet suites, API lint/generation drift check, Desktop test suite,
  TypeScript typecheck, lint and non-MSI Vite build passed on the current branch;
- the tagged MCP qualification covered the reviewed-Preset lifecycle through Hermes
  Profile write, connection test, authoritative read-back, binding update and removal;
- the current non-MSI Desktop and its bundled `yorvad` were launched from the linked P7
  worktree for Owner inspection;
- the focused cumulative diff review found no caller-controlled MCP command, args,
  environment, headers, executable/path or arbitrary JSON mutation surface;
- no destructive Restore was run against the Owner's live Hermes data, and no push, merge,
  tag or freeze was performed.

This completes the automated B7 handoff only. The later 2026-08-27 freeze-audit decision
is recorded in section 17 and supersedes the earlier `IN_PROGRESS` handoff state here.

## 17. MVP-first plan integration and freeze audit result

The Phase 7-relevant requirements from the Owner-provided MVP-first P7R–P13 plan are now
incorporated into this repository Spec rather than depending on an external downloaded
copy. P7R requires Runtime-owned shared Models, managed Skills, reviewed MCP Definitions,
Runtime maintenance, authoritative read-back, truthful failure results, and a terminal-free
product flow. The final integration flow includes encrypted Backup/Restore and a real
fixed-candidate Upgrade/Rollback; a visible page or readable plan does not replace an
executable mutation.

The 2026-08-27 freeze audit is recorded at
`docs/phases/audits/AUDIT-007-hermes-runtime-management-completeness.md` with Gate Decision
**FAIL**. Production Upgrade/Rollback bindings are absent, disposable Windows Restore
success/failure-recovery evidence is missing, and the exact candidate has no CI/race
evidence. This candidate must not be pushed for merge, merged, tagged, or marked FROZEN.

The P8 inputs retained from the MVP-first plan are the product support matrix, database
migration, crash/restart recovery, installer lifecycle, YORVA update, sanitized diagnostics
bundle, and basic stability verification. They must not become a READY Phase 8 execution
Spec, and P8 implementation must not begin, until Phase 7 remediation and re-audit PASS.

## 18. Owner scope amendment: managed Upgrade/Rollback deferred

On 2026-08-27 the Owner approved
`docs/phases/amendments/AMENDMENT-007A1-defer-managed-hermes-upgrade.md`. Executable
managed Hermes Upgrade/Rollback is deferred to a later separately approved scope. Phase 7
keeps only the truthful read-only Upgrade Plan and keeps production Upgrade/Rollback
capabilities false. ADR-0015 remains the safety contract for future implementation.

The revised freeze blockers are the disposable Restore lifecycle and exact-candidate
CI/race evidence. The original failed audit remains preserved; a re-audit must determine
the final Gate after both remaining items are complete.

The disposable Restore and exact-candidate CI evidence is now preserved at
`docs/phases/evidence/PHASE-007-DISPOSABLE-RESTORE-AND-CI.md`. Re-audit
`docs/phases/audits/AUDIT-007R1-hermes-runtime-management-completeness.md` records
**PASS**. Main integration `248937819e5b063c7973b76fd60a70667876ce7b` passed
final-main CI run 33050692156. The formal baseline is
`phase-007-hermes-runtime-management-completeness-baseline`; Phase 7 is COMPLETE / FROZEN.

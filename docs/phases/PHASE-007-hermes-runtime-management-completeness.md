# YORVA Phase 7 — Hermes Runtime Management Completeness

> Status: **IN_PROGRESS — B0 PASS; DEPENDENCY-DRIVEN PARALLEL IMPLEMENTATION**
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
> Authorization: P7-D1–D8 and B0–B10 approved 2026-08-24. B1 qualification and
> required ADRs must pass before product-capability code begins.

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
| P7-D2 | Skills use only inspected/audited official or Owner-approved catalog/registry identities. No unchecked direct URL or force bypass. |
| P7-D3 | MCP uses approved presets/catalog plus qualified HTTPS MCP. No caller-controlled stdio command, argv, environment or header surface. |
| P7-D4 | Prefer an encrypted Runtime-scope complete backup. Do not ship plaintext full Hermes backup if a standard encrypted format and reliable Restore cannot be qualified. ADR required. |
| P7-D5 | Managed upgrade creates a new generation under ADR-0006/0009. Never mutate the active tree with `hermes update` or `--force-venv`. ADR required. |
| P7-D6 | Health/logs expose allowlisted categories, fixed bounds and mandatory redaction only. No arbitrary path tail or raw log export. |
| P7-D7 | P7 keeps one authenticated `LOCAL_DESKTOP` actor with typed actions. Principal/Grant/RBAC remains P9/P10 scope. |
| P7-D8 | Stable `0.20.x` may be detected, but each P7 feature reports capability only after exact-version surface qualification. Unknown contracts fail closed. |

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

- exact Runtime/Profile live inventory and bounded inspect;
- identity, source, version, enabled, update and audit/scan state;
- install/update/remove only from approved inspected sources;
- configure/enable/disable only through a qualified closed schema;
- mandatory scan result with no force bypass;
- authoritative read-back, external-change reconciliation and recovery.

Hermes remains authoritative for Skill files and state.

### 5.3 MCP

- live configured inventory and approved catalog/preset inventory;
- preset install, qualified HTTPS configuration and bounded connection/tool test;
- safe browser/device/OAuth flow only if initiating-session isolation is proven;
- closed tool-selection mutation, remove, reauthentication and reconciliation;
- timeout/cancel cleanup for process and network work.

No Desktop/API field may accept an arbitrary stdio command, executable, package name,
argv, environment, header, local path or secret-bearing URL. Any approved local-process
MCP requires a fixed adapter-owned descriptor.

### 5.4 Backup and Restore

- explicit Runtime or qualified Instance scope, format version, Runtime version,
  timestamp, size, checksum and state;
- explicit user-selected local destination; no automatic Cloud upload;
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

No request accepts arbitrary command, environment, local path or secret-bearing URL.
A user-selected backup destination remains a narrowly scoped local capability and cannot
silently become a future remote file command.

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
| B4 | Skills | Source/scan/traversal/Profile isolation/recovery/manual smoke |
| B5 | MCP | No generic command surface, credential/session isolation, timeout/cancel/manual smoke |
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
- MCP arbitrary command/env/header rejection, session-bound auth, timeout cleanup and
  `CONFIGURED` versus `READY` distinction;
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
- Skills/MCP requires arbitrary command/env/path/header or a force bypass;
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

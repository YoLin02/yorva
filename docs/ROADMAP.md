# YORVA Roadmap

## Product direction

YORVA starts as a local Hermes deployment/control experience and evolves into a general AI Runtime deployment and control platform.

Roadmap rules:

> Do not build the next phase to compensate for an unfinished current phase.

> Every implementation phase must pass the audit/gate process in `PHASE_GOVERNANCE.md` and `AUDIT_STANDARD.md` before the next phase begins.

`ROADMAP.md` defines direction and candidate deliverables. Detailed implementation is authorized by the current Phase Spec, not by roadmap text alone.

## Phase 0 — Architecture freeze

Goal: make the repository safe for agent-assisted initialization.

Deliverables:

- `DEVELOPMENT.md`;
- `ARCHITECTURE.md`;
- `AGENTS.md`;
- `PROTOCOL.md`;
- `RUNTIME.md`;
- `DATA_MODEL.md`;
- `SECURITY.md`;
- ADR-0001 through ADR-0004;
- repository bootstrap plan.

Exit criteria:

- no unresolved primary technology choice;
- clear module boundaries;
- clear local protocol;
- Hermes integration boundary documented;
- secret/storage model documented;
- Phase 0 document readiness review completed.

## Phase 1 — Repository foundation

Status: **COMPLETE / FROZEN**
Gate: **PASS** (`AUDIT-001R2-repository-foundation.md`)
Baseline: `phase-001-bootstrap-baseline`
Audit-accepted implementation commit: `1b759f443dbbebba4ae61a82c91e92180d7527b0`

Goal: create a minimal runnable YORVA shell with no fake business features.

Deliverables:

```text
Tauri 2 + React Desktop
Go yorvad
SQLite migrations
OpenAPI v1 skeleton
minimal unauthenticated health + authenticated Node/bootstrap
SSE event channel
Hermes adapter registry skeleton
CI: build/typecheck/test
```

Exit criteria:

- Desktop starts daemon or discovers it safely;
- Desktop can read authenticated Node information;
- schema migrates from empty DB;
- no Hermes command is called from React/Tauri UI code;
- Phase 1 re-audit passes and the final baseline commit on `main` is frozen by `phase-001-bootstrap-baseline`.

## Phase 2 — Hermes Discovery & Compatibility

Status: **COMPLETE / FROZEN** (baseline `phase-002-hermes-discovery-baseline-r1` unchanged)
Gate: **PASS** (`AUDIT-002R1-hermes-discovery.md`); amendment 002A1 **ACCEPTED** (`AUDIT-002A1-hermes-discovery.md` — `PASS`)
Spec: `docs/phases/PHASE-002-hermes-discovery.md`
Amendments:
- `AMENDMENT-002A1-hermes-windows-command-resolution.md` — ACCEPTED / FROZEN into r1
- `AMENDMENT-002A2-hermes-launcher-alias-normalization.md` — ACCEPTED
- `AMENDMENT-002A3-active-generation-discovery.md` — ACCEPTED FOR IMPLEMENTATION (Owner 2026-08-18; not a Phase 2 re-freeze)
Historical baseline: `phase-002-hermes-discovery-baseline` → `a67de04e900bc3ddce99cd76501eec13586082ed` (immutable)
Current baseline: `phase-002-hermes-discovery-baseline-r1`
Amendment implementation commit: `dbcb54da4bc4bffcff51888426848246a1900ea6`
Compatibility: `>=0.19.0 <0.21.0`

Goal: detect Hermes and report executable/version compatibility without mutating the machine.

Deliverables:

- detect Hermes;
- locate official executable candidates;
- version and compatibility reporting;
- not-installed, unsupported, broken, malformed, timeout and cancellation handling;
- deterministic multiple-candidate selection;
- authenticated discovery API/OpenAPI contract;
- Desktop discovery status view.
- Dashboard / Runtimes / Settings Desktop shell with persistent English and Simplified Chinese locales.

Exit criteria:

```text
Open YORVA
→ detect absent or installed Hermes
→ show executable, version and compatibility
→ present safe negative states and retry/cancel behavior
```

Phase 2 must not install/download Hermes, modify PATH/Python, or begin Profile/lifecycle work.

## Phase 3 — Hermes Installation

Status: **AUDIT / GENERATION INTEGRITY REMEDIATION**
Spec: `docs/phases/PHASE-003-hermes-installation.md`
Amendments:
- `AMENDMENT-003A1-embedded-hermes-source.md` — ACCEPTED
- `AMENDMENT-003A2-china-dependency-distribution.md` — ACCEPTED
- `AMENDMENT-003A3-managed-node-prerequisites.md` — ACCEPTED
- `AMENDMENT-003A4-generation-install-transaction.md` — ACCEPTED FOR IMPLEMENTATION (batch-gated; not Phase 3 PASS)
Architecture: `docs/phases/PHASE-003-generation-installation-architecture.md` — Owner-approved 2026-08-18
ADR: `ADR-0006-generation-install-transaction.md` — Accepted
Implementation branch: `fix/phase3-audit-r6-remediation` (generation batches 0–8; dual-state-machine and persist-ahead remediations landed)
Audit: `AUDIT-003` — **FAIL**; `AUDIT-003R1`–`R7` — **FAIL**; `AUDIT-003R8` — **PENDING**

Goal: install a supported official Hermes Runtime without requiring terminal use.

Candidate deliverables:

- explicit installation Operation;
- official-source provenance and integrity verification;
- narrow privilege/elevation handling where required;
- structured, redacted install progress;
- post-install discovery and compatibility verification;
- failure, cancellation and recovery UX.

Exit criteria:

```text
Hermes not installed
→ user explicitly starts installation
→ official installation completes
→ Phase 2 discovery verifies a supported executable
```

The detailed Phase 3 Spec was prepared after Phase 2 amendment 002A1 passed independent audit and `phase-002-hermes-discovery-baseline-r1` was frozen. The Repository Owner approved the Spec on 2026-08-17 and authorized implementation in automatic batch-gate mode on 2026-08-17.

## Phase 4 — Instance/Profile management

Status: **DRAFT / NOT AUTHORIZED** (blocked on Phase 3 freeze)
Spec: `docs/phases/PHASE-004-instance-profile.md`

Goal: manage multiple Hermes-backed YORVA instances.

Deliverables:

- list profiles as Instances;
- create Instance;
- delete Instance with explicit confirmation;
- reconcile external profile changes;
- instance capability/status view;
- lifecycle actions where Hermes supports the requested scope.

Exit criteria:

- multiple independent Hermes profiles are discoverable and manageable;
- YORVA recovers if profiles are changed outside YORVA.

## Phase 5 — Models and credentials

Goal: make model configuration safe and simple.

Deliverables:

- provider/model configuration UI;
- write-only credential mutation;
- secure-store integration;
- credential status metadata;
- configuration validation;
- secret-redaction tests.

Exit criteria:

- user can configure a working Hermes model without editing `.env` or YAML;
- no API key plaintext appears in SQLite or ordinary logs.

## Phase 6 — Messaging channels

Goal: deliver the key YORVA promise: one-click channel connection.

Priority:

1. Weixin;
2. WeCom;
3. additional Hermes channels based on actual user demand.

Deliverables:

- channel capability list;
- connect/disconnect workflow;
- QR/login Operation;
- live QR state events;
- success/failure/timeout states;
- no durable QR credential storage.

Exit criteria:

```text
Instance
→ Connect Weixin/WeCom
→ QR/auth flow
→ connected
→ status visible in YORVA
```

## Phase 7 — Runtime management completeness

Goal: make YORVA practical for daily local management.

Candidate deliverables:

- Skills;
- MCP;
- backups/restores;
- Hermes upgrades;
- richer health/log views;
- startup/service management;
- crash/recovery UX.

Only implement features with stable Hermes integration paths.

## Phase 8 — Local product hardening

Goal: prepare a public local release.

Deliverables:

- Windows installer/update path;
- macOS/Linux validation where feasible;
- migration/upgrade tests;
- crash recovery;
- telemetry decision (opt-in or none; separate ADR if introduced);
- security review;
- signed release pipeline;
- user-facing diagnostics/export bundle without secrets.

## Phase 9 — Optional Control Plane prototype

Start only after local product is stable.

Goal: manage multiple Nodes remotely without exposing inbound management ports.

Technology:

```text
Go modular monolith
PostgreSQL
HTTPS
outbound Node WSS
```

Deliverables:

- user/org minimum model;
- device pairing;
- Node inventory;
- heartbeat;
- typed remote command;
- remote Operation progress;
- audit trail.

Explicitly not required for first prototype:

- microservices;
- billing;
- complex RBAC;
- Kubernetes;
- Redis/Kafka unless proven necessary.

## Phase 10 — Enterprise management

Driven by real customer requirements.

Possible scope:

- organization/teams;
- RBAC/policies;
- SSO;
- fleet groups;
- bulk upgrade policy;
- audit retention;
- private deployment;
- enterprise secret provider integration;
- templates/distributions.

Do not commit to all features before discovery.

## Phase 11 — Second Runtime

This is the trigger to validate the Runtime abstraction.

Process:

1. choose a real second Runtime based on product demand;
2. implement with current contract where possible;
3. record friction/gaps;
4. generalize only demonstrated common concepts;
5. decide whether a separate Runtime SDK/plugin process is justified.

A dynamic Runtime marketplace is not built before this phase proves the need.

## Release naming suggestion

```text
0.1  foundation + Hermes discovery/compatibility
0.2  Hermes installation + multi-instance
0.3  model configuration + Weixin/WeCom channels
0.4  Skills/MCP/backup/upgrade
0.5  local hardening/public beta
0.8  optional remote Control Plane beta
1.0  stable local + remote management contract
```

Version numbers are planning labels, not promises.

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
- `AMENDMENT-002A4-exact-hermes-version-compatibility.md` — IMPLEMENTED / TARGETED AUDIT PENDING (Owner 2026-08-21)
- `AMENDMENT-0065A1-hermes-development-version-policy.md` — P6.5 CANDIDATE (Owner 2026-08-24; patch-compatible development window)
Historical baseline: `phase-002-hermes-discovery-baseline` → `a67de04e900bc3ddce99cd76501eec13586082ed` (immutable)
Current baseline: `phase-002-hermes-discovery-baseline-r1`
Amendment implementation commit: `dbcb54da4bc4bffcff51888426848246a1900ea6`
Compatibility: stable development window `>=0.20.2 <0.21.0`; packaged source remains pinned by exact archive SHA-256 (0065A1 supersedes the exact-patch development lock without changing frozen historical baselines)

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

Status: **COMPLETE / FROZEN**
Gate: **PASS** (`AUDIT-003R9-hermes-installation.md`)
Baseline: `phase-003-hermes-installation-baseline`
Audit-accepted implementation commit: `721325181892d0fd8534f9f7d287fe05d9603bb0`
Exact-commit CI: GitHub Actions run `32131548538` — SUCCESS
Exact-commit MSI: GitHub Actions run `32131548340` — SUCCESS
Spec: `docs/phases/PHASE-003-hermes-installation.md`
Amendments:
- `AMENDMENT-003A1-embedded-hermes-source.md` — ACCEPTED
- `AMENDMENT-003A2-china-dependency-distribution.md` — ACCEPTED
- `AMENDMENT-003A3-managed-node-prerequisites.md` — ACCEPTED
- `AMENDMENT-003A4-generation-install-transaction.md` — ACCEPTED
- `AMENDMENT-003A6-final-path-hermes-generation-build.md` — ACCEPTED / FROZEN (Owner 2026-08-21; frozen-baseline correctness correction)
- `AMENDMENT-003A7-configurable-download-sources.md` — ACCEPTED FOR IMPLEMENTATION (Owner 2026-08-21; post-freeze product correction)
- `AMENDMENT-003A8-embedded-python-source-priority.md` — P6.5 CANDIDATE (Owner 2026-08-24)
Architecture: `docs/phases/PHASE-003-generation-installation-architecture.md` — Owner-approved 2026-08-18
ADR: `ADR-0006-generation-install-transaction.md` — Accepted
Correction ADR: `ADR-0009-final-path-generation-build.md` — Accepted 2026-08-21
Download-source ADR: `ADR-0010-configurable-hermes-download-sources.md` — Accepted 2026-08-21
Embedded-Python ADR: `ADR-0011-embedded-python-source-priority.md` — P6.5 candidate 2026-08-24
Audit: `AUDIT-003`–`R7` — **FAIL** (immutable); `AUDIT-003R8` — **PASS WITH CONDITIONS**; `AUDIT-003R9` — **PASS**

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

Status: **COMPLETE / FROZEN**
Gate: **PASS WITH CONDITIONS** (`AUDIT-004R3-instance-profile.md`)
Baseline: `phase-004-instance-profile-baseline`
Audit-accepted implementation commit: `35b268425a023f20c655bbfbd697f7a80c3e60a9`
Exact-commit CI: GitHub Actions run `32234908416` — SUCCESS
Specs: `docs/phases/PHASE-004-instance-profile.zh-CN.md` (Owner review) and `docs/phases/PHASE-004-instance-profile.md` (Agent execution mirror) — **FROZEN**
Start commit: `d04b1fdc298f643f84d0c84a245595baae2e8994`
Branch: `fix/phase4-profile-delete-timeout`
Audit: `AUDIT-004` — **FAIL** (immutable); `AUDIT-004R1`/`R2` — **PASS WITH CONDITIONS** (historical); `AUDIT-004R3` — **PASS WITH CONDITIONS**

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

Status: **COMPLETE / FROZEN**
Specs: `docs/phases/PHASE-005-models-credentials.zh-CN.md` (Owner review) and `docs/phases/PHASE-005-models-credentials.md` (Agent execution mirror)
Target baseline: `phase-004-instance-profile-baseline`
Owner decisions: D1-D6 **APPROVED** 2026-08-19
Credential authority: `ADR-0007-hermes-native-model-credential-authority.md` — **ACCEPTED**
Accepted audit: `docs/phases/audits/AUDIT-005R1-models-credentials.md` — **PASS**
Frozen baseline: `phase-005-models-credentials-baseline` → `d82a802b59d5f2715e431f73f6e4fe44e623d7a4`

Goal: make model configuration safe and simple.

Deliverables:

- provider/model configuration UI;
- write-only credential mutation;
- qualified Hermes-native Profile credential persistence without a YORVA duplicate, using the ADR-0007 narrow compatibility writer only where the pinned official surface is unsafe;
- credential status metadata;
- configuration validation;
- secret-redaction tests.

Exit criteria:

- user can configure a working Hermes model without editing `.env` or YAML;
- no API key plaintext appears in SQLite or ordinary logs.

## Phase 6 — Runtime lifecycle and messaging channels

Status: **COMPLETE / FROZEN**
Specs: `docs/phases/PHASE-006-runtime-lifecycle-messaging-channels.zh-CN.md` (Owner review) and `docs/phases/PHASE-006-runtime-lifecycle-messaging-channels.md` (Agent execution mirror)
Target baseline: `phase-005a1-post-freeze-corrections-baseline` -> `9957775`
Execution authorization: **Owner authorized 2026-08-20**
Implementation handoff: [`PHASE-006-IMPLEMENTATION-AUDIT-HANDOFF.md`](phases/evidence/PHASE-006-IMPLEMENTATION-AUDIT-HANDOFF.md)
Owner smoke: [`PHASE-006-OWNER-AUTHENTICATED-SMOKE.md`](phases/evidence/PHASE-006-OWNER-AUTHENTICATED-SMOKE.md) — exact remediation MSI supplement recorded
First independent audit: [`AUDIT-006-runtime-lifecycle-messaging-channels.md`](phases/audits/AUDIT-006-runtime-lifecycle-messaging-channels.md) — **FAIL** (immutable)
R1 audit: [`AUDIT-006R1-runtime-lifecycle-messaging-channels.md`](phases/audits/AUDIT-006R1-runtime-lifecycle-messaging-channels.md) — **FAIL** (immutable; code PASS, evidence blocker)
R2 audit: [`AUDIT-006R2-runtime-lifecycle-messaging-channels.md`](phases/audits/AUDIT-006R2-runtime-lifecycle-messaging-channels.md) — **FAIL** (immutable; no Critical/High, one bounded OpenAPI Medium)
Accepted audit: [`AUDIT-006R3-runtime-lifecycle-messaging-channels.md`](phases/audits/AUDIT-006R3-runtime-lifecycle-messaging-channels.md) — **PASS**
Baseline: `phase-006-runtime-lifecycle-messaging-channels-baseline`
Audit-accepted candidate: `859561712583c67031d8110812df3a4236be6908`
Final-main integration: `0e0ef06ed707c20ea686cee098b9f7650c0a6f74`; CI run [`32694400178`](https://github.com/YoLin02/yorva/actions/runs/32694400178) — **PASS**
Inspected MSI SHA-256: `B942E637BE9BC59D6C5B603DA117C7AC9646CD254E88F36B1A609A0696FA8EFD`
Accepted debt/conditions: **none**

The lifecycle, Weixin, WeCom, sender-pairing and Desktop continuity batches are on
`main`, which is recorded as a governance deviation rather than acceptance. The Owner
has now tied real Weixin connection/pairing/disconnect and WeCom connection/disconnect
to the inspected remediation MSI without supplying sensitive values. First-audit
candidate `3d2fecf`, remediation candidate `7e1123e` and sanitized-evidence candidate
`13612db` had exact green CI. R2 accepted the mandatory real-channel evidence and found
no Critical/High, but failed on the bounded lifecycle OpenAPI type defect. The narrow
fix is `4cff18b`; exact candidate `8595617` passed CI run `32692682968`, independent R3
returned PASS, and final-main CI passed on merge `0e0ef06`. The accepted baseline is
frozen by the annotated Phase 6 tag above.

Goal: make a configured Instance operational, then deliver the key YORVA promise of one-click channel connection.

Priority:

1. Runtime-neutral Instance lifecycle foundation;
2. Hermes Profile gateway status/start/stop/restart;
3. startup/service management and lifecycle crash/recovery UX;
4. Weixin;
5. WeCom;
6. additional Hermes channels based on actual user demand in a later phase.

Deliverables:

- normalized lifecycle capability and live status;
- start/stop/restart Operations;
- Runtime-neutral lifecycle orchestration, conflict control and recovery;
- Hermes-specific lifecycle execution isolated in the Hermes adapter;
- explicit startup/service policy without hidden elevation;
- lifecycle-related crash/recovery UX;
- channel capability list;
- connect/disconnect workflow;
- QR/login Operation;
- live QR state events;
- success/failure/timeout states;
- no durable QR credential storage;
- Weixin sender-pairing pending count and write-only approval;
- user-session tray, close-to-tray, packaged login start and single-instance restore
  without Hermes `ON_LOGIN`, elevation or Instance startup.

Exit criteria:

```text
Instance
→ Start/Stop/Restart without a terminal
→ authoritative lifecycle status visible in YORVA
→ Connect Weixin/WeCom
→ QR/auth flow
→ connected
→ Channel status visible separately from lifecycle status
```

## Phase 6.5 — Developer-led demo optimization

Status: **COMPLETE / FROZEN**
Spec: `docs/phases/PHASE-006.5-developer-led-demo-optimization.zh-CN.md`
Target baseline: `phase-006-runtime-lifecycle-messaging-channels-baseline` → `7ca9103e7af210296a5e24916df01856539b550e`
Baseline: `phase-0065-developer-led-demo-baseline`
Owner authorization: accepted P6.5 changes may be committed, pushed, merged to `main`, and frozen after the candidate gate and audit pass (2026-08-24).
Implementation candidate: `559245cf42c6ea07dbe4706676334a717b2173fd`
Implementation handoff: [`PHASE-0065-IMPLEMENTATION-AUDIT-HANDOFF.md`](phases/evidence/PHASE-0065-IMPLEMENTATION-AUDIT-HANDOFF.md)
Accepted audit: [`AUDIT-0065-developer-led-demo-baseline.md`](phases/audits/AUDIT-0065-developer-led-demo-baseline.md) — **PASS**
Exact-candidate CI: [`32715890955`](https://github.com/YoLin02/yorva/actions/runs/32715890955) — **PASS**
Exact-candidate Windows MSI: [`32715958209`](https://github.com/YoLin02/yorva/actions/runs/32715958209) — **PASS**
Final-main integration: `b9af6a3fd057b90ef636ff3b581cc9680774dac8`; CI run [`32717542173`](https://github.com/YoLin02/yorva/actions/runs/32717542173) — **PASS**
Accepted debt/conditions: **none**

Candidate scope:

- configurable Hermes download sources;
- embedded CPython prerequisite and deterministic source priority;
- Provider model-catalog retrieval, multi-model selection, and one persisted default model;
- Hermes patch-compatible development detection (`>=0.20.2 <0.21.0`) and an exact-hash packaged official Hermes `0.20.5` source snapshot;
- directly related Desktop, packaging, contract, documentation, and regression-test changes.

P7 capabilities, personal artifacts, generated installers, temporary UI references, and unrelated
developer files are excluded from this candidate. Phase 6 remains immutable; P6.5 is its clean
successor. Audit, exact-candidate CI, Windows smoke, package inspection, security checks and
final-main CI passed before the annotated baseline was frozen.

The automatic final-main MSI run [`32717542138`](https://github.com/YoLin02/yorva/actions/runs/32717542138)
failed on both attempts before compilation because GitHub `codeload` returned HTTP 429.
This infrastructure result is retained rather than relabeled: the exact-candidate MSI run above
already passed from the unchanged product commit and its downloaded artifact was independently
hashed. No source, executable, resource or packaging input changed after that accepted MSI.

## Phase 7 — Runtime management completeness

Status: **PASS — 2026-08-27 RE-AUDIT; AWAITING BASELINE FREEZE**
Specs: `docs/phases/PHASE-007-hermes-runtime-management-completeness.zh-CN.md` (Owner review) and `docs/phases/PHASE-007-hermes-runtime-management-completeness.md` (execution mirror)
Baseline: `phase-0065-developer-led-demo-baseline` → `5f68e48f17e7e342e1781b37613b19d4bd1f060b`
Branch: `codex/phase7-hermes-runtime-management`
Owner decisions: P7-D1–D8 and B0–B10 **APPROVED** 2026-08-24
ADR-0013–ADR-0016 and ADR-0018 **ACCEPTED** 2026-08-25. The Owner selected GitHub repos
read-only as B5's sole first HTTPS MCP qualification candidate on 2026-08-25;
selection does not enable the registry entry before authenticated Windows evidence.
Owner amendment 007A1 **APPROVED** 2026-08-27: executable managed Hermes
Upgrade/Rollback is deferred from the Phase 7 freeze scope. The read-only plan remains
truthful and production mutation capability remains false. Restore and exact-candidate CI
evidence remained the active freeze blockers until both subsequently passed for product SHA
`e21e8618f6fcda34eba31707500c29bde75893b5`; `AUDIT-007R1` records PASS. Merge,
final-main CI and the formal baseline/tag remain pending, so Phase 7 is not yet FROZEN.
Current execution: B1 evidence is preserved as a per-surface risk map; B2 shared contracts
and independent B3–B8 lanes proceed in parallel where no real prerequisite exists.
Focused lane tests precede one integrated B10 independent audit.

Goal: make YORVA practical for daily local management.

Candidate deliverables:

- Skills: qualified Hermes-native inventory plus the separate YORVA-managed
  install/update/enable/disable/remove lifecycle from approved sources, with exact-Profile
  projection, ownership and drift reporting;
- MCP;
- backups/restores;
- Hermes upgrades;
- richer health/log views;
- feature-specific recovery UX for Skills, MCP, backup/restore and upgrade workflows.

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

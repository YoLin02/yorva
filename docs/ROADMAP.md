# YORVA Roadmap

## Product direction

YORVA starts as a local Hermes deployment/control experience and evolves into a **local-first, Runtime-neutral Agent Runtime management infrastructure**.

The long-term product path is deliberately ordered:

```text
manage one Runtime well
→ manage many Instances on one machine
→ validate the abstraction with a second Runtime
→ turn yorvad into a durable YORVA Node
→ manage many Nodes from YORVA Control
→ manage configuration and desired state across a Fleet
→ add enterprise governance only when real customer requirements justify it
```

YORVA is not intended to become an Agent application builder. Prompt authoring, workflow design, RAG orchestration, multi-Agent business logic and general-purpose Agent evaluation are outside the Runtime-management core unless a later ADR establishes a separate product layer.

The product boundary is:

```text
YORVA Control        optional central multi-Node control plane
        ↓
YORVA Node           yorvad as the management authority for one machine
        ↓
Runtime Contract     normalized management intent and capabilities
        ↓
Runtime Adapter      concrete Runtime integration
        ↓
Runtime / Instance   Hermes and future native AI Runtimes
```

Local operation remains first-class. A future Control Plane must not become mandatory for local functionality, and it must never bypass the same application use cases used by the local Desktop.

Roadmap rules:

> Do not build the next phase to compensate for an unfinished current phase.

> Every implementation phase must pass the audit/gate process in `PHASE_GOVERNANCE.md` and `AUDIT_STANDARD.md` before the next phase begins.

> Validate Runtime-neutral abstractions with a real second Runtime before building distributed abstractions that depend on them.

> Central management means typed Runtime/Instance management, not a generic remote shell, arbitrary process executor or unrestricted file API.

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

## Phase 7 — Single-Node Runtime operations completeness

Status: **COMPLETE / FROZEN**
Specs: `docs/phases/PHASE-007-hermes-runtime-management-completeness.zh-CN.md` (Owner review) and `docs/phases/PHASE-007-hermes-runtime-management-completeness.md` (execution mirror)
Predecessor baseline: `phase-0065-developer-led-demo-baseline` → `5f68e48f17e7e342e1781b37613b19d4bd1f060b`
Formal baseline: `phase-007-hermes-runtime-management-completeness-baseline`
Branch: `phase/p7-hermes-runtime-management`
Owner decisions: P7-D1–D8 and B0–B10 **APPROVED** 2026-08-24
ADR-0013–ADR-0016 and ADR-0018 **ACCEPTED** 2026-08-25. The Owner selected GitHub repos
read-only as B5's sole first HTTPS MCP qualification candidate on 2026-08-25;
selection does not enable the registry entry before authenticated Windows evidence.
Owner amendment 007A1 **APPROVED** 2026-08-27: executable managed Hermes
Upgrade/Rollback is deferred from the Phase 7 freeze scope. The read-only plan remains
truthful and production mutation capability remains false. Restore and exact-candidate CI
evidence remained the active freeze blockers until both subsequently passed for product SHA
`e21e8618f6fcda34eba31707500c29bde75893b5`; `AUDIT-007R1` records PASS. Main integration
`248937819e5b063c7973b76fd60a70667876ce7b` passed final-main CI run
[`33050692156`](https://github.com/YoLin02/yorva/actions/runs/33050692156) before the
formal baseline was frozen.

Post-freeze stability revision `9834a8cb1df9e70502936153943f501ed37cb8fc`
was fast-forwarded to `main` on 2026-08-31. It adds bounded Hermes launch during
detection, separates removed Instance records from the active inventory, and closes the
authenticated cleanup route after authoritative absence readback. Exact revision CI run
[`33365096577`](https://github.com/YoLin02/yorva/actions/runs/33365096577) passed. Final-main
CI run [`33366271630`](https://github.com/YoLin02/yorva/actions/runs/33366271630) retained
an initial Windows handshake-timeout failure and passed on attempt 2; Windows MSI run
[`33366271629`](https://github.com/YoLin02/yorva/actions/runs/33366271629) passed. The
original Phase 7 baseline tag remains immutable and is not moved by this corrective patch.

Goal: complete the local, terminal-free operating loop for multiple Runtime Instances on one machine before adding distributed management.

Candidate deliverables:

- qualified Hermes-native Skills inventory plus YORVA-managed install/update/enable/disable/remove through stable Runtime integration paths;
- reviewed-Preset MCP configuration, test, binding and authoritative status through stable Runtime integration paths;
- encrypted Runtime backup and restore workflows with explicit scope, rollback and recovery semantics;
- truthful read-only Hermes upgrade planning; executable managed Upgrade/Rollback is deferred by Amendment 007A1;
- richer Runtime/Instance health views;
- richer structured log and diagnostics views;
- multi-Instance operational overview on one Node;
- feature-specific recovery UX for Skills, MCP, backup/restore and lifecycle failures;
- reconciliation after supported external Runtime state changes.

Only implement capabilities that have a stable Hermes integration path. Do not introduce distributed abstractions, Fleet concepts or a second Runtime inside Phase 7.

Exit criteria:

```text
fresh supported machine
→ install/open YORVA
→ detect or install Hermes
→ create multiple Instances
→ configure models/credentials
→ configure Skills and MCP where supported
→ connect supported channels
→ run multiple Instances
→ inspect authoritative health/log state
→ create and restore backup
→ inspect truthful Runtime upgrade readiness without an unqualified mutation
→ recover from bounded failures
```

The normal path must not require the user to open a terminal or directly edit Runtime files.

## Phase 8 — Local product hardening

Status: **APPROVED / IN PROGRESS — AUDIT FAIL; B2/B4/B6 GATES REOPENED**
Specs: docs/phases/PHASE-008-local-product-hardening.zh-CN.md (Owner review) and
docs/phases/PHASE-008-local-product-hardening.md (execution mirror)
Required baseline: phase-007-hermes-runtime-management-completeness-baseline plus accepted
P7 stability revision `9834a8cb1df9e70502936153943f501ed37cb8fc` on `main`
Planned branch: phase/p8-local-product-hardening

The P7 stability handoff adds bounded Hermes launch during detection and authoritative
removed-Instance record classification/cleanup without moving the frozen P7 tag or
reopening deferred P7 capabilities. Exact revision CI run `33365096577`, final-main CI
run `33366271630` attempt 2 and Windows MSI run `33366271629` passed. The Owner approved
the P8 Spec and B0–B6 execution on 2026-09-03. Production Windows signing material is
not yet available, so public release remains blocked. Exact candidate `bd07bbd` passed
CI #93 and Windows MSI #32 on 2026-09-07. The fresh single-agent
[AUDIT-008](phases/audits/AUDIT-008-local-product-hardening.md) nevertheless returns FAIL:
update success lacks authoritative readback (HIGH-001), and interrupted download state
cannot recover after Desktop restart (MEDIUM-001). B2/B4 affected cases and the B6
internal-candidate Gate are reopened; the historical eight-hour soak remains valid.

Goal: turn the completed single-Node experience into a dependable public local product before validating additional Runtime or remote-management scope.

Deliverables:

- Windows installer/update path;
- migration and upgrade tests across supported YORVA versions;
- application and daemon crash recovery;
- machine reboot/startup continuity for supported local scenarios;
- macOS/Linux validation where feasible, without weakening the Windows baseline;
- telemetry decision (`opt-in` or none; separate ADR if introduced);
- security review and threat-model refresh;
- signing-aware release pipeline; without production signing material, only an internal candidate may pass;
- user-facing diagnostics/export bundle without secrets;
- documented support matrix and recovery guidance.

The P8 update deliverable updates YORVA Desktop/yorvad and supported local schema through
a complete verified installer. It does not reopen the managed Hermes Upgrade/Rollback
deferred by Phase 7 Amendment 007A1.

Exit criteria:

- a supported user can install, update, restart and recover YORVA without development tooling;
- supported local data survives version migration according to documented ownership rules;
- failures produce actionable diagnostics without exposing credentials;
- the local product remains fully usable without YORVA Control.

## Phase 9 — Second Runtime validation

This phase is intentionally before distributed Control Plane work.

Goal: prove that the Runtime Contract is genuinely Runtime-neutral before remote and Fleet abstractions depend on it.

Runtime selection criteria:

- real product or customer demand;
- stable enough integration surfaces to support production-quality management;
- meaningful overlap with existing YORVA capabilities;
- enough differences from Hermes to expose false assumptions in Core;
- no requirement to fake unsupported features merely to satisfy symmetry.

Process:

1. choose one real second Runtime;
2. implement it through the current Runtime registry and focused capability interfaces where possible;
3. record every contract friction point and Hermes-specific assumption exposed;
4. generalize only concepts demonstrated by both Runtimes;
5. keep truly Runtime-specific concepts inside adapter metadata or adapter-owned behavior;
6. decide only after evidence whether a separate Runtime SDK/plugin process is justified.

Candidate deliverables:

- second Runtime descriptor and adapter;
- discovery/compatibility integration;
- installation integration only if the selected Runtime has a stable and safe installation surface;
- Instance mapping where the Runtime exposes an independently manageable unit;
- lifecycle/configuration/credential capability mapping where actually supported;
- capability and unsupported-state normalization;
- cross-Runtime contract tests;
- Desktop and Node views that do not branch on concrete Runtime kind for generic behavior;
- documented abstraction gaps and accepted non-uniform behavior.

Explicitly not required:

- a Runtime marketplace;
- arbitrary third-party plugins;
- a stable external Runtime SDK;
- identical feature sets across Runtimes;
- generic file/process access to compensate for weak Runtime integrations.

Exit criteria:

- Hermes and the second Runtime can coexist on one YORVA Node;
- the same application layer can manage both through Runtime-neutral use cases;
- unsupported capabilities are explicit rather than simulated;
- Desktop generic flows do not require scattered `runtime == ...` logic;
- any Core generalization is backed by evidence from at least two real Runtime implementations.

## Phase 10 — YORVA Node

Goal: turn `yorvad` from a Desktop companion daemon into a durable, headless Node management authority that can operate a machine independently of the Desktop UI.

A YORVA Node owns management of the Runtimes and Instances on exactly one machine. Future YORVA Control manages Nodes; it does not directly manage OS processes or Runtime files.

Candidate deliverables:

- headless `yorvad` operating mode;
- supported OS service integration, beginning with the primary Windows deployment target;
- durable Node identity;
- boot/restart recovery and inventory reconciliation;
- authoritative Runtime and Instance inventory snapshot;
- concurrent management of multiple supported Runtimes and multiple Instances;
- Node health and management-readiness state;
- bounded local resource/conflict reporting where required for safe Runtime operations;
- local administrative bootstrap suitable for a server without Desktop UI;
- remote-transport boundary designed for a later outbound Control connection without enabling it prematurely.

Architectural rules:

- Node management remains typed through application use cases;
- no public unauthenticated management listener;
- no universal `shell.exec`, `process.exec` or arbitrary file mutation API;
- Runtime executable paths, commands and native process topology remain adapter-owned;
- Desktop, when present, is a client of the same Node authority rather than a second management authority.

Exit criteria:

```text
machine boots
→ YORVA Node service starts
→ Runtime inventory reconciles
→ Instance inventory reconciles
→ supported Runtime/Instance operations work without Desktop
→ service restarts without losing management authority
```

## Phase 11 — YORVA Control Plane

Start only after Phase 10 proves a stable Node boundary.

Goal: manage multiple YORVA Nodes remotely without exposing inbound Node management ports and without bypassing Node application semantics.

Initial technology direction:

```text
Go modular monolith
PostgreSQL
HTTPS
outbound Node WSS
```

Candidate deliverables:

- minimum user/organization model;
- Node/device pairing;
- Node inventory;
- Node heartbeat and connectivity state;
- Runtime/Instance inventory projection from each Node;
- typed remote commands that map to existing Node application use cases;
- remote Operation progress and terminal state;
- bounded retry/idempotency rules for remote Operations;
- audit trail for remote management actions;
- safe disconnect/reconnect behavior;
- Console views for Nodes, Runtimes, Instances and Operations.

First prototype explicitly does not require:

- microservices;
- billing;
- complex enterprise RBAC;
- Kubernetes;
- Redis/Kafka unless measured need appears;
- desired-state reconciliation across a Fleet;
- generic remote administration capabilities.

Exit criteria:

- one Control Plane can pair and observe multiple Nodes;
- a user can issue a supported typed Runtime/Instance operation to a selected Node;
- the Node executes that operation through the same application layer as local requests;
- progress/result survives bounded reconnect scenarios;
- the Node never requires an inbound public management port.

## Phase 12 — Fleet configuration and Desired State

Goal: move from one-command-at-a-time remote management to safe, centralized configuration and drift management across many Nodes.

Core concepts introduced here must remain Runtime-management concepts, not Agent application-definition concepts.

Candidate concepts:

- `FleetGroup` or equivalent Node grouping/tagging;
- `InstanceTemplate` for normalized Runtime/Instance configuration intent;
- versioned Desired State records;
- Actual State projection from Nodes;
- drift detection;
- bounded reconciliation;
- staged/batch rollout;
- pause/resume and failure containment;
- configuration diff/preview before rollout;
- bulk Runtime/Instance operations;
- policy-scoped Runtime upgrade rollout;
- audit linkage from Desired State revision to resulting Operations.

An Instance template may describe infrastructure/runtime concerns such as:

```text
Runtime kind/version constraint
Instance lifecycle/startup policy
Provider/model selection metadata
Skills/MCP configuration where normalized
Channel configuration where normalized
backup/upgrade policy references
```

It must not silently expand into application-authoring concerns such as arbitrary system prompts, RAG pipelines, business workflows or multi-Agent orchestration.

Desired State model:

```text
Control Desired State
        ↓
Node compares Desired State with Actual State
        ↓
Node executes typed YORVA use cases
        ↓
Actual State converges or reports bounded drift/failure
```

Exit criteria:

- a user can target a group of Nodes/Instances with a versioned configuration intent;
- the system can show Desired vs Actual State;
- drift is visible and explainable;
- reconciliation uses typed Node operations rather than arbitrary remote execution;
- one failed Node does not cause uncontrolled Fleet-wide retry or mutation.

## Phase 13 — Enterprise management

Driven by real customer and deployment requirements after the Runtime, Node, Control and Fleet boundaries are proven.

Possible scope:

- organizations/teams at production scale;
- role-based access control and scoped policies;
- SSO and enterprise identity integration;
- approval boundaries for sensitive or production operations;
- audit retention/export requirements;
- private/on-premises Control deployment;
- enterprise secret-provider integration;
- tenant or organizational separation where required;
- enterprise distribution/template governance;
- compliance and policy reporting;
- HA/backup requirements for YORVA Control when justified by deployments.

Do not commit to every enterprise feature before discovery. In particular, billing, SaaS multi-tenancy, microservices, Kubernetes deployment and external policy engines are not default requirements.

## Product boundaries / non-goals

The following are not part of the core roadmap unless later evidence and an ADR create a distinct product layer:

- general Agent workflow builder;
- RAG application platform;
- prompt-management studio;
- arbitrary multi-Agent orchestration framework;
- general LLM gateway built only for feature parity with other platforms;
- Kubernetes replacement or container orchestrator;
- generic RMM/remote-shell product;
- unrestricted remote file/process administration;
- dynamic Runtime marketplace before a real plugin ecosystem is justified.

YORVA may integrate with upper-layer Agent platforms. Its core responsibility is to make Runtime and Instance deployment, configuration, lifecycle, recovery and Fleet operation reliable enough that those upper layers do not need to understand each Runtime's native operational details.

## Release naming suggestion

```text
0.1  foundation + Hermes discovery/compatibility
0.2  Hermes installation + multi-instance
0.3  model configuration + lifecycle + Weixin/WeCom channels
0.4  single-Node Runtime operations completeness
0.5  local hardening/public beta
0.6  second Runtime validation
0.7  headless YORVA Node
0.8  multi-Node YORVA Control beta
0.9  Fleet configuration + Desired State
1.0  stable local + multi-Runtime + Node + Control management contract
1.x  enterprise management driven by deployment requirements
```

Version numbers are planning labels, not promises.

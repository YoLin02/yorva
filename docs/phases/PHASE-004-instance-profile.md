# YORVA Phase 4 — Instance / Profile Management

> Status: COMPLETE / FROZEN
> Language: English execution mirror for the implementation Agent
> Owner: Repository owner
> Target baseline: `phase-003-hermes-installation-baseline`
> Implementation start commit: `d04b1fdc298f643f84d0c84a245595baae2e8994`
> Previous gate: `AUDIT-003R9-hermes-installation.md` — **PASS**
> Chinese Owner-review source: `PHASE-004-instance-profile.zh-CN.md`
> Implementation authorization: **GRANTED** 2026-08-19
> Owner decisions D1–D4: **APPROVED** 2026-08-19
> Feature branch: `fix/phase4-profile-delete-timeout`
> Implementation: COMPLETE / FROZEN
> Phase 4 audit: `AUDIT-004` — FAIL (immutable); `AUDIT-004R1`/`R2` — PASS WITH CONDITIONS (historical); `AUDIT-004R3` — PASS WITH CONDITIONS
> Audit-accepted implementation commit: `35b268425a023f20c655bbfbd697f7a80c3e60a9`
> Exact-commit CI: GitHub Actions run `32234908416` — SUCCESS

This document and its Chinese mirror define the same contract. The 2026-08-19 freeze was withdrawn to fix delete timeout and tombstone delete UX, then independently re-audited. `AUDIT-004R3` returned `PASS WITH CONDITIONS` on `35b268425a023f20c655bbfbd697f7a80c3e60a9`. Historical `AUDIT-004` remains FAIL. Owner accepted the remaining Medium/Low conditions and authorized re-freeze. Phase 5 implementation is not authorized by this freeze.

## 1. Objective

Allow a user with one supported Hermes installation to manage Hermes Profiles as normalized YORVA Instances:

```text
SUPPORTED Hermes installation
→ query official Hermes profiles
→ normalize and reconcile YORVA Instances
→ create a minimal profile
→ inspect identity and availability
→ permanently delete a non-default profile after explicit confirmation
```

One YORVA Instance maps to one Hermes Profile. This does not mean one OS process per Instance. Hermes remains authoritative for profile existence and profile-owned data; SQLite stores only YORVA inventory and last-known metadata.

## 2. Entry Criteria

- [x] Phase 3 independent Gate is `PASS` (`AUDIT-003R9`).
- [x] Phase 3 is `COMPLETE / FROZEN` at `phase-003-hermes-installation-baseline`.
- [x] `control/active.json` remains the only active generation pointer.
- [x] Phase 2 discovery verifies the executable selected by the active generation.
- [x] Owner approves the decisions in Section 3.
- [x] Chinese and English specifications are synchronized and marked `READY`.
- [x] Owner separately authorizes Phase 4 implementation.

Owner message 2026-08-19 approved D1–D4, the synchronized `READY` marking, implementation authorization, start commit `d04b1fdc298f643f84d0c84a245595baae2e8994` (current `main` including the Desktop restyle already on `origin/main`), and feature branch `codex/phase4-instance-profile`.

## 3. Owner Decisions Required

The recommended smallest Phase 4 contract is:

- [x] **D1 — Official surface.** Use the pinned Hermes `0.20.2` documented Profile CLI from the active generation. The reviewed REST API requires an already-running Hermes web service, while the reviewed TUI gateway has no delete method. Phase 4 will not start a Hermes service merely to manage profiles. `list` output may use a version-pinned, fixture-tested compatibility parser only after confirming no suitable offline structured output exists; unknown output fails closed.
- [x] **D2 — Minimal create.** Create with official no-clone/no-alias/no-skills behavior. YORVA does not copy credentials, `.env`, `auth.json`, models, config, sessions or skills from another profile.
- [x] **D3 — Destructive delete.** Phase 4 includes permanent deletion of a named, non-default Hermes Profile. The UI requires typed name confirmation and explains that Hermes-owned profile data is removed. The `default` profile is never deletable. Deleting native data marks the retained YORVA Instance `MISSING`; it does not erase its stable `instanceId`.
- [x] **D4 — No lifecycle.** Phase 4 reports `lifecycle: false`; it does not expose Start/Stop/Restart. Profile management is not proof of safe Instance-scoped process lifecycle. Lifecycle remains a later phase unless an amendment is approved.

If any decision is rejected, revise both language versions before setting the phase to `READY`.

## 4. User-Visible Success Flow

1. Desktop refreshes Hermes discovery and enables Instance actions only for a `SUPPORTED` installation.
2. The Instance page lists the built-in `default` profile and named profiles, including availability, protected/default state, and last successful synchronization time.
3. Create accepts only a profile name. It accepts no path, URL, source, command, environment, clone source, credential or model input.
4. YORVA starts an `instance.create` Operation and refreshes authoritative Hermes state after the command completes.
5. Delete shows a destructive warning, requires the exact normalized profile name, and starts an `instance.delete` Operation.
6. Refresh detects profiles added or removed outside YORVA. A failed Hermes query produces `UNKNOWN`, not a false `MISSING` result.
7. After Desktop restart, existing Operation APIs recover task projection and a fresh reconciliation recovers Instance truth.

## 5. In Scope

- list the built-in and named Hermes Profiles as YORVA Instances;
- create one minimal named Profile through the Hermes adapter;
- inspect normalized identity, availability and protection state;
- permanently delete one non-default Profile with explicit confirmation;
- reconcile profiles created or removed outside YORVA;
- authenticated local HTTP and OpenAPI contracts for list/create/get/delete;
- durable `instance.create` and `instance.delete` Operations;
- SQLite Instance inventory as a cache;
- bilingual Desktop list, empty, create, pending, delete, conflict, missing and error states;
- adapter, application, persistence, protocol and Desktop tests.

## 6. Out of Scope

Hard boundaries:

- Hermes installation, repair, upgrade or uninstall;
- modification of Phase 3 generations, seals, `active.json`, install transactions, `PATH` or `HERMES_HOME`;
- profile rename, clone, import, export, selection/activation or distribution installation;
- login, authentication, credentials, API keys, provider or model configuration;
- Skills, MCP, channels, Weixin/WeCom, sessions, memory, backup/restore or Cloud;
- starting, stopping, restarting or supervising Hermes processes;
- adopting, migrating or deleting leftover `hermes-agent` trees;
- direct Hermes profile-directory scanning or importing Hermes Python internals;
- a generic dynamic Runtime plugin system;
- arbitrary shell, path, environment, URL or file APIs.

## 7. Architecture Boundary and Package Ownership

```text
React Desktop
    ↓ generated typed client
Authenticated local HTTP / OpenAPI
    ↓
Application Instance use cases + Operations
    ↓
Domain Instance model
    ↑
Hermes Profile adapter + SQLite repository
```

Target ownership:

```text
services/node/internal/domain/instance       stable Instance types and states
services/node/internal/app                   list/create/delete/reconcile use cases
services/node/internal/runtime/hermes        official Profile command adapter/parser
services/node/internal/persistence/sqlite    Instance cache and migrations
services/node/internal/transport/httpapi     closed DTOs and handlers
apps/desktop/src                             presentation and localized interaction only
```

Hermes-specific parsing stays in the Hermes adapter. Reuse existing process containment, Operation, event, logging and SQLite components. Do not reorganize unrelated Phase 2/3 files during Phase 4.

Forbidden dependencies:

```text
React → Hermes CLI or profile files
Domain → Hermes types
Hermes adapter → HTTP or UI
SQLite cache → filesystem recovery authority
Rust/Tauri → profile ownership or reconciliation decisions
```

## 8. Official Hermes Surface Qualification

The repository-pinned official source is Hermes `0.20.2` at commit `df4b65147d7ddd74dd449f9067aabbca5aef0ec7`. Current review evidence shows:

- documented CLI: `hermes profile list`, `create`, `show`, and `delete`;
- structured REST: list/create/delete, but only through an already-running Hermes web backend;
- TUI gateway JSON-RPC: list/create/describe, but no delete method;
- no confirmed offline structured output for CLI Profile list/show.

Batch 1 re-confirmed the pinned archive on 2026-08-19. D1 remains the documented Profile CLI:

- argv: `profile list`; `profile create <name> --no-alias --no-skills`; `profile delete <nativeId> --yes`;
- create `--no-alias` skips the wrapper; `--no-skills` writes `.no-bundled-skills` and skips bundled skill seeding;
- official name regex `[a-z0-9][a-z0-9_-]{0,63}` (1–64); reserved names `hermes`, `default`, `test`, `tmp`, `root`, `sudo`; `default` is a special alias, never created or deleted;
- YORVA create ingress is the closed subset that also requires a leading lowercase letter and rejects reserved names including `default`;
- `hermes profile list` has no `--json`; the 0.20.2 printer is a table (`Profile`/`Model`/`Gateway`/`Alias`/`Distribution`, active marker `◆`). The docs example that marks the active profile with `*` is stale; unknown output fails closed;
- REST `/api/profiles` is absent from `PUBLIC_API_PATHS` and requires a running dashboard plus auth when the gate is on;
- TUI methods are `profiles.list`, `profiles.create`, `profiles.describe`, `profiles.configure`, `profiles.set_asset`, `profiles.get_asset`. There is no delete method.

Selection order:

1. documented structured surface usable without adding process lifecycle or authentication scope;
2. documented official CLI;
3. narrow, exact-version compatibility parsing for documented CLI output.

Stop for an amendment if the chosen surface requires starting a service, parsing undocumented files, importing Python modules, exposing credentials, or expanding lifecycle scope. A compatibility parser must be isolated, accept only known complete output, reject unknown/partial/oversized output, and have pinned fixtures.

## 9. Identity and State Model

- `instanceId`: opaque, stable YORVA primary identity for one Instance record. API resource paths, Operation targets and future YORVA relations use this value.
- `runtimeInstallationId`: YORVA identity of the supported active Hermes installation.
- `nativeId`: exact normalized Runtime-native Hermes Profile name returned by the official surface. Only the Hermes adapter uses it to address Hermes.
- `name`: display name; initially equal to `nativeId`.
- `default`: built-in Hermes root profile; visible and protected.
- uniqueness: `(runtime_installation_id, native_id)`.

`instanceId` and `nativeId` are not interchangeable. `instanceId` must never be passed to Hermes, inferred from a profile name, or exposed as a Runtime-native identity. `nativeId` must never be used as the YORVA database primary key, API `{instanceId}`, Operation target ID, or foreign-key identity.

| State | Meaning |
|---|---|
| `AVAILABLE` | Profile exists in the most recent successful authoritative Hermes query. It does **not** mean login, model configuration, Agent readiness, gateway health or process lifecycle readiness. |
| `MISSING` | Previously known, absent from the most recent successful complete query. The YORVA row and stable `instanceId` remain retained. |
| `UNKNOWN` | Query failed, timed out, was cancelled, or output was unsafe to parse. |

`CREATING` and `DELETING` are Operation states, not Instance availability states. External rename is old `MISSING` plus new `AVAILABLE`; Phase 4 does not infer identity continuity. If the same `nativeId` later reappears under the same Runtime installation, reconciliation restores the existing row to `AVAILABLE` and preserves its `instanceId`.

## 10. Reconciliation Contract

```text
query official Hermes surface completely
→ validate and normalize every native identity
→ begin short SQLite transaction
→ upsert present profiles as AVAILABLE
→ mark previously known absent profiles MISSING
→ commit cache
```

Rules:

- never hold a SQLite transaction while running Hermes;
- on query/parser/timeout failure, preserve rows and set freshness `UNKNOWN`; never infer absence;
- duplicate or invalid native identities fail the complete snapshot closed;
- reconciliation never creates, edits or deletes Hermes data;
- Phase 4 never automatically deletes a `MISSING` row by age, startup count or refresh count. It is an indefinite tombstone preserving YORVA identity;
- future tombstone cleanup requires an explicit Owner-approved contract/migration and is not part of Phase 4;
- unknown directories are never deleted;
- startup, explicit refresh and successful mutations invoke the same use case;
- serialize profile mutation and reconciliation per Runtime installation; do not reuse the Phase 3 filesystem `install.lock`.

## 11. Create Contract

Request fields: only normalized `name`.

The ingress validator uses a closed subset of the pinned official grammar: lowercase ASCII letter first; then lowercase letters, digits, `_` or `-`; reserved official names and `default` are rejected. Exact length and reserved names must be copied into tests from the reviewed official contract, not guessed from directory behavior.

Recommended adapter invocation is equivalent to:

```text
<active-generation-hermes> profile create <name> --no-alias --no-skills
```

Requirements:

- resolve the executable only through verified Phase 2/3 discovery, never ambient `PATH`;
- set fixed `%LOCALAPPDATA%\hermes` `HERMES_HOME` and an allowlisted child environment;
- use direct argv, existing Windows Job Object containment and bounded output;
- never pass clone, config, model, skill, alias, credential or arbitrary path options;
- re-query Hermes after exit; success means the requested profile is authoritatively present;
- normalize duplicate name to `INSTANCE_ALREADY_EXISTS`;
- do not automatically clean up an uncertain partial profile; return a safe error and reconcile.

## 12. Delete Contract

Delete requires exact normalized `confirmationName`; the server verifies it matches current `nativeId`.

Recommended adapter invocation is equivalent to:

```text
<active-generation-hermes> profile delete <nativeId> --yes
```

Requirements:

- reject `default` and protected profiles before process start;
- resolve identity through a fresh successful authoritative query;
- never accept a path or derive deletion authority from SQLite alone;
- re-query after exit; confirmed absence is converged success, including a disappearance race, and the Instance row becomes `MISSING` rather than being deleted;
- if final state is uncertain, preserve the row as `UNKNOWN` and return a stable error;
- never uninstall Hermes, delete a generation, or delete unknown directories.

## 13. Operations, Timeout, Cancellation and Concurrency

- POST create and DELETE return `202` with durable `instance.create` / `instance.delete` Operations.
- Both require `Idempotency-Key`; same key/request returns the same Operation, conflicting payload is rejected.
- Commands have explicit whole-operation deadlines, bounded output and process-tree cleanup.
- Mutations are non-cancellable after the Hermes command starts. Pre-start cancellation may succeed; Desktop hides Cancel after the boundary.
- Only one profile mutation or reconciliation runs per Runtime installation.
- No database transaction spans process execution.
- Daemon restart restores Operation projection, then reconciliation determines actual Hermes state. Operation status never proves profile existence.

Stable errors include at least:

```text
RUNTIME_NOT_SUPPORTED
INSTANCE_INVALID_NAME
INSTANCE_ALREADY_EXISTS
INSTANCE_NOT_FOUND
INSTANCE_PROTECTED
INSTANCE_CONFIRMATION_MISMATCH
INSTANCE_CONFLICT
INSTANCE_QUERY_FAILED
INSTANCE_OUTPUT_UNRECOGNIZED
INSTANCE_OPERATION_TIMED_OUT
CAPABILITY_NOT_SUPPORTED
```

Raw subprocess errors, output, paths and environment values are not API error messages.

## 14. API / OpenAPI Contract

Use the routes reserved in `PROTOCOL.md`:

```text
GET    /api/v1/runtimes/{runtimeId}/instances
POST   /api/v1/runtimes/{runtimeId}/instances
GET    /api/v1/instances/{instanceId}
DELETE /api/v1/instances/{instanceId}
```

POST body is closed `{ "name": "..." }`. DELETE body is closed `{ "confirmationName": "..." }`. Reject missing bodies, unknown fields, multiple JSON values, trailing garbage and oversized bodies with stable errors.

Responses expose only normalized identity/state/capabilities/timestamps. They omit paths, config, credentials, environment, raw Hermes output and subprocess details. Lifecycle routes return `CAPABILITY_NOT_SUPPORTED`; Desktop renders no lifecycle controls.

## 15. Persistence

Use the existing `instances` model and add only a deterministic migration actually required by implementation. Minimum invariant:

```text
UNIQUE(runtime_installation_id, native_id)
```

SQLite is a cache. A cached row cannot authorize Hermes mutation, deletion or filesystem access. `MISSING` tombstones are retained indefinitely in Phase 4 so a reappearing `(runtime_installation_id, native_id)` reuses the stable `instanceId`. Tests cover empty DB, Phase 3 upgrade, uniqueness, identity separation, tombstone retention and restart reconciliation.

## 16. Desktop UX and i18n

Add an Instances page within the existing sidebar. Required Chinese and English states:

- loading/refresh, default/protected, empty named-profile state;
- create form with inline validation;
- create/delete progress and restart recovery;
- destructive delete dialog with typed confirmation;
- `AVAILABLE`, `MISSING`, `UNKNOWN`, conflict and timeout guidance;
- lifecycle unavailable explanation without fake controls.

Use generated API types and server-state queries. Local React state is only for dialog/form interaction. Render dates in local timezone. Status cannot rely on color alone; keyboard focus, labels and dialogs must be accessible.

## 17. Security and Diagnostics

- provider credentials and arbitrary inherited `HERMES_*`, Python or Node injection variables do not reach child processes;
- no secret plaintext in API, SQLite, logs, events or UI;
- structured logs contain correlation/operation ID, action, safe native identity, outcome, duration and stable error code;
- never log raw Hermes output or full filesystem paths;
- create/delete remain current-user scope and never require Administrator;
- reuse existing process containment and secure command construction;
- deletion warnings are product safety controls, not filesystem ownership proof.

## 18. Implementation Batches — After Authorization Only

Implementation is delivered as small vertical capabilities, not as one large cross-layer change. After Owner authorization, batches use automatic gates: a passing batch may enter the next batch without separate Owner confirmation, but may not implement later-batch behavior early.

### Batch 1 — Lock the official contract

- re-confirm pinned Hermes Profile commands, flags, output and name rules;
- record the D1 surface choice and add exact-version parser/command fixtures;
- make no production Profile mutation and add no generalized adapter framework.

Gate: fixture/contract tests pass, the selected surface stays inside Phase 4, and the diff contains no functional Batch 2+ code.

### Batch 2 — Read-only Instance inventory

- implement the minimum `instanceId` / `nativeId` domain model;
- implement strict Profile list parsing, SQLite cache and reconciliation;
- expose GET list/get through OpenAPI and a bilingual read-only Desktop page;
- cover `AVAILABLE`/`MISSING`/`UNKNOWN`, tombstone retention and external add/remove.

Gate: adapter, persistence, application, protocol and Desktop read-only tests pass. No create/delete subprocess exists yet.

### Batch 3 — Create one minimal Profile

- add `instance.create`, closed POST DTO, idempotency and the exact safe CLI argv;
- add create form/progress/restart projection;
- prove no clone, alias, Skills, credentials, model or arbitrary input enters the command.

Gate: focused create, duplicate, timeout, process cleanup, API and Desktop tests pass; authoritative re-query is the success criterion.

### Batch 4 — Delete one protected Profile

- add `instance.delete`, closed DELETE DTO and typed-name confirmation;
- protect `default`, re-query before/after delete, and retain the row as `MISSING`;
- add the bilingual destructive warning and Operation state.

Gate: focused delete, confirmation mismatch, protected default, disappearance race, tombstone and Desktop tests pass. No unknown path can be deleted.

### Batch 5 — Contract-bounded resilience

- complete only the restart, concurrency, timeout, cancellation-boundary, output-limit and redaction cases already required by Sections 13, 17 and 19;
- verify Phase 3 active generation/environment invariants remain unchanged;
- fix defects found by these tests without introducing a second state machine.

Gate: all affected Go/OpenAPI/Desktop/Windows process tests pass and the batch diff contains no Phase 5+ or lifecycle behavior.

### Batch 6 — Full verification and audit handoff

- run Section 20 in full, update contract/completion evidence, and obtain exact-commit CI;
- perform no new feature work except fixes required by failed verification;
- stop at `AUDIT-004 = PENDING` and hand the exact commit to an independent Agent.

For every batch:

1. inspect only that batch's intended diff;
2. run its focused tests and `git diff --check`;
3. produce a reviewable batch commit or an equally isolated commit series;
4. do not weaken a contract or test to pass the gate;
5. automatically continue only when the gate passes.

Stop and report to the Owner only for architecture conflict, lifecycle/Phase 5 expansion, unsupported official surface, unsafe deletion, uncontained process, secret exposure, required major dependency/framework, or a contract/product decision.

This batch plan is intentionally bounded. Do not add a plugin system, generic workflow engine, dependency-injection framework, second reconciliation state machine, filesystem ownership system, ACL/sandbox project, per-system-call failpoint matrix, or speculative Runtime abstraction. Defend the concrete Profile command, identity, deletion, process and data boundaries in this Spec; record unrelated hardening as future work.

## 19. Test Matrix

| Scenario | Required result |
|---|---|
| Unsupported/absent Hermes | Mutations rejected; no subprocess. |
| Active pointer missing/invalid | Fail closed; Phase 3 state unchanged. |
| Official default only | Default visible and protected. |
| Valid create | One `AVAILABLE` Instance after authoritative re-query. |
| Invalid/reserved/path-like name | Rejected before subprocess. |
| Duplicate create/idempotency replay | One native profile and one Operation. |
| Create timeout/process descendants | Tree terminated; final state reconciled, never guessed. |
| No-clone create | No credentials/config/model/skills copied by YORVA. |
| Confirmed delete | Profile absent after authoritative re-query; retained row is `MISSING`. |
| Default or confirmation mismatch delete | Rejected before subprocess. |
| Delete disappearance race | Converges idempotently to absent. |
| Query timeout/malformed/oversized output | Rows become `UNKNOWN`, never false `MISSING`. |
| Missing across repeated restarts/refreshes | Row remains `MISSING`; no TTL or automatic purge. |
| Same native profile reappears | Existing `instanceId` is preserved and state returns to `AVAILABLE`. |
| `instanceId` / `nativeId` boundary | YORVA routes/relations use `instanceId`; only Hermes adapter calls use `nativeId`. |
| Profile exists without model/login | `AVAILABLE` only; no Agent/model/login readiness claim. |
| External add/remove/rename | Cache converges without mutating Hermes. |
| Duplicate native identity in output | Whole snapshot rejected. |
| Unknown directories under `HERMES_HOME` | Never authoritative and never deleted. |
| Leftover `hermes-agent` | Not adopted as a second installation. |
| Concurrent mutation/reconcile | Serialized without torn cache or duplicates. |
| Daemon/Desktop restart | Operation recovers; Hermes query determines truth. |
| Secret sentinel | Absent from child, API, logs, events and UI. |
| Unknown/trailing API fields | Stable `INVALID_REQUEST`. |
| Lifecycle action | Capability false and `CAPABILITY_NOT_SUPPORTED`. |
| Phase 3 invariants | `active.json`, generations, seals, PATH and HERMES_HOME unchanged. |
| Chinese/English UI | Equivalent localized, accessible behavior. |

## 20. Verification Matrix

Before audit, run and record:

- API lint, OpenAPI validation and generated client drift;
- Desktop typecheck, lint, tests, build and dependency audit;
- Go format, full and repeated targeted tests, CI race, vet, build and vulnerability scan;
- Rust format, tests, clippy, check and dependency audit;
- Windows real-process containment/timeout smoke with safe fixtures;
- Tauri release no-bundle and MSI/package checks if packaging inputs change;
- exact-commit GitHub Actions success.

Real Hermes Profile smoke uses an isolated account/VM with disposable data. Never run delete smoke against an Owner profile.

## 21. Exit Criteria

- [x] Section 3 decisions are Owner-approved.
- [x] Official surface qualification and pinned fixtures are recorded.
- [x] Multiple Profiles reconcile as unique YORVA Instances.
- [x] Create and protected destructive delete work without a terminal.
- [x] Failed queries never become false absence.
- [x] SQLite remains a cache and Hermes remains authoritative.
- [x] No Phase 3 generation/environment invariant changes.
- [x] No Phase 5+ feature or lifecycle implementation enters the diff.
- [x] Full verification and exact-commit CI pass.
- [x] Independent `AUDIT-004R2-instance-profile.md` reaches Owner-accepted `PASS WITH CONDITIONS`.

Phase 4 is `COMPLETE / FROZEN` on audit-accepted commit `35b268425a023f20c655bbfbd697f7a80c3e60a9`. Do not begin Phase 5 from this freeze. Remaining Owner-accepted conditions are recorded in `AUDIT-004R3`.

## 22. Audit Requirements

The independent auditor verifies scope, official-surface provenance, strict parsing, command safety, process cleanup, deletion protection, reconciliation truth, cache semantics, concurrency, restart, redaction, API closure, bilingual Desktop behavior and every Phase 3 invariant.

Critical or High findings fail the gate. Medium follows `AUDIT_STANDARD.md` and requires explicit Owner acceptance for `PASS WITH CONDITIONS`; acceptance criteria cannot be weakened after implementation.

## 23. Agent Execution Directive

After and only after Owner approval/authorization, the implementation Agent must:

1. read `AGENTS.md`, governance/architecture/protocol/runtime/data/security docs, relevant ADRs, this Spec and the approved Chinese mirror;
2. lock the starting commit and create a non-`main` Phase 4 feature branch;
3. execute Section 18 one vertical batch at a time; pass its automatic gate and isolate its diff before starting the next;
4. preserve user work and historical audits;
5. stop at implementation complete + verification pass + audit pending;
6. hand the exact commit to a fresh independent audit Agent.

This directive does not itself authorize implementation.

## 24. Completion Evidence

```text
Implementation commit: 35b268425a023f20c655bbfbd697f7a80c3e60a9
Branch: fix/phase4-profile-delete-timeout
Batch results:
  READY docs 60b23f4 PASS
  Batch 1 618d5ca PASS
  Batch 2 b11eaa3 PASS
  Batch 3 ff62906 PASS
  Batch 4 a2fa88e PASS
  Batch 5 65d4ddf PASS
  Batch 6 verification+handoff fe15203 PASS
  High remediations 8397dd4 PASS
  Delete timeout 8672af5 PASS
  Delete modal / tombstone UX 35b2684 PASS
Verification matrix:
  pnpm install --frozen-lockfile PASS
  pnpm audit --audit-level low PASS
  pnpm api:lint PASS
  pnpm api:generate PASS; generated schema no drift
  pnpm typecheck PASS
  pnpm lint PASS after removing setState-in-effect
  pnpm test PASS
  pnpm build PASS
  gofmt on changed Go files PASS
  go test ./... PASS
  go test affected -count=20 PASS
  go test -race ./... NOT RUN locally: -race requires cgo
  go vet ./... PASS
  govulncheck ./... PASS
  go build ./cmd/yorvad PASS
  cargo fmt --check PASS
  cargo test --locked --offline --lib PASS
  cargo clippy --locked --all-targets -- -D warnings PASS
  cargo check --locked PASS
  cargo audit PASS with 17 allowed unmaintained/unsound warnings (pre-existing GTK/unic stack)
  pnpm build:sidecar PASS
  windows-lifecycle-smoke.ps1 PASS
  inspect-yorva-msi.tests.ps1 PASS
  tauri build --no-bundle PASS
  MSI packaging NOT RUN (no packaging input change)
  Real Hermes delete smoke NOT RUN (requires isolated disposable account/VM)
Exact-commit CI: GitHub Actions run 32234908416 SUCCESS on 35b2684
  Web and API contract SUCCESS
  Go Node including go test -race SUCCESS
  Windows Desktop native shell SUCCESS
Known non-blocking risks / Owner-accepted conditions:
  local go test -race blocked (CGO_ENABLED)
  cargo audit allowed historical GTK3/unic warnings
  official profile list docs example with * is stale vs 0.20.2 table printer
  MEDIUM-004-001 Operation target remains runtime-installation / installationId
  MEDIUM-004-002 create worker still persists timeout as INSTANCE_QUERY_FAILED
  LOW-004R1-001 RecoverStale persist failure is warn-and-continue
AUDIT-004: FAIL (historical, 10a2509)
AUDIT-004R1: PASS WITH CONDITIONS (historical)
AUDIT-004R2: PASS WITH CONDITIONS (historical withdrawn freeze)
AUDIT-004R3: PASS WITH CONDITIONS
```

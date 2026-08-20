# YORVA Phase 5 — Models and Credentials

> Status: COMPLETE / FROZEN
> Language: English execution mirror for the implementation Agent
> Owner: Repository owner
> Owner-designated repository snapshot: `089a58005edc8f8f6a72b4fb44276be7c322eb1d`
> Required implementation baseline: `phase-004-instance-profile-baseline` → `0dd5d432affd44e23bf577acfe9cf2fdfbfa3f45`
> Chinese Owner-review source: `PHASE-005-models-credentials.zh-CN.md`
> Owner decisions D1-D6 and ADR-0007: **APPROVED** 2026-08-19
> Implementation branch: `codex/phase5-models-credentials`
> Execution authorization: Batches 1-5, audit, CI, merge/freeze/tag and Windows release build
> Batch 1: complete
> Batch 2: complete
> Batch 3: complete
> Batch 4: complete
> Batch 5: complete
> Phase 5 audit: `AUDIT-005` — FAIL (immutable); `AUDIT-005R1` — **PASS**
> Audit-accepted candidate: `dd7c6c2f47a2b3c7331ebc810b1eb2b003ab59a9`
> Exact-candidate CI: GitHub Actions run `32343964969` — SUCCESS
> Main merge: `c45a231060400cf21e41730f88ccdeab443b8a4f`
> Final-main CI: GitHub Actions run `32346079074` — SUCCESS
> Windows MSI: GitHub Actions run `32346079072` — SUCCESS; artifact `yorva-msi` (`9398271244`)
> Frozen baseline tag: `phase-005-models-credentials-baseline`

This document and its Chinese mirror define one contract. The Owner reviews the Chinese version. On 2026-08-19 the Owner approved D1-D6, ADR-0007, the synchronized governance update, the actual Phase 4 baseline and both specifications as `READY`. The initial Batch 1 qualification correctly stopped after proving that the pinned official CLI leaks secrets through argv and that the Web/TUI setters require a long-running service. The Owner then approved the narrow compatibility fallback in D3 and one continuous authorization through Batches 1-5, audit, CI, merge/freeze/tag and the Windows release build. That historical STOP evidence remains part of the qualification record.

## 1. Objective

Deliver a China-market-first, preset-based model configuration experience for an existing Hermes-backed YORVA Instance:

```text
AVAILABLE Instance
→ open Models inside the existing Instance experience
→ choose a Provider preset
→ choose a recommended model or enter a model ID
→ enter an API key
→ save through the qualified Hermes Profile credential surface
→ explicitly test the connection
→ show a safe result
```

The user does not need to understand Hermes config keys, environment-variable names, YAML, `.env`, base URLs or command argv. The saved Profile must remain directly usable by Hermes after YORVA closes.

Phase 5 configures model/provider access only. It does not start or supervise Hermes, redefine Instance availability, add chat/inference UI, or begin Phase 6 channels.

## 2. Repository Baseline and Existing Capabilities

The Owner-designated design snapshot was `089a58005edc8f8f6a72b4fb44276be7c322eb1d`. The effective Phase 4 re-freeze is the annotated tag `phase-004-instance-profile-baseline`, peeled to `0dd5d432affd44e23bf577acfe9cf2fdfbfa3f45` on `main` and `origin/main`. Therefore:

- this Spec uses the effective re-freeze commit as its implementation baseline;
- it does not alter Phase 4 documents, code, history or prior audit conclusions;
- unrelated uncommitted user work is preserved and is not part of Phase 5.

Phase 5 must reuse, not redesign or duplicate:

- stable `instanceId` / `nativeId` identity separation;
- `AVAILABLE` / `MISSING` / `UNKNOWN` semantics and permanent `MISSING` tombstones;
- active Hermes executable resolution through existing discovery;
- the existing Hermes `commandRunner`, Windows Job Object / Unix process-group containment, bounded output and cleanup;
- the authenticated loopback API, route contract and error envelope;
- `api/openapi.yaml`, its generated Desktop client and existing DTO mapping conventions;
- the existing Runtime registry/bundle wiring;
- the existing Instance mutation/reconciliation coordination source;
- the existing Operation framework for network-waiting validation;
- the existing Desktop sidebar, `App.tsx` query composition and Instances page;
- existing `i18n.ts`, English/Simplified Chinese messages and locale persistence;
- existing `formatDateTime` local-time formatter;
- existing Go, OpenAPI, React/Vitest, Rust/Tauri and Windows process test conventions.

Phase 5 must not create a second navigation system, i18n system, process runner, Runtime registry, Operation framework, date formatter or server-state store.

## 3. Architecture Conflict and Required Governance Decision

The China-market ProviderPreset and consumer UX direction is an ordinary Phase 5 Spec refinement. The proposed credential authority is not.

| Existing frozen rule | Proposed D2 | Classification |
|---|---|---|
| `SECURITY.md §7`: instance/model provider credentials are stored behind an OS-backed `SecretStore`, with no silent plaintext fallback. | Hermes Profile's official credential storage, normally Profile `.env`, becomes the sole credential authority for the MVP. | Material security architecture conflict. |
| `DATA_MODEL.md §10`: `secret_refs` references OS-secure material. | Phase 5 would create no `secret_refs` row for Hermes model API keys. | Material data ownership/documentation change; no schema deletion is required. |
| `ARCHITECTURE.md`: persistence stores secret references and includes a secrets adapter boundary. | Hermes owns these Runtime-native credentials; YORVA delegates persistence to Hermes. | Material secret-boundary refinement. |
| `ROADMAP.md`: Phase 5 includes secure-store integration. | Phase 5 MVP omits SecretStore when the official Hermes Profile store satisfies the approved contract. | Roadmap deliverable change. |
| `PROTOCOL.md`: secrets are write-only and GET returns metadata only. | The same API rule remains. | No conflict. |
| Source-of-truth rule: Hermes owns Hermes state. | Hermes owns Profile model credentials/config. | Aligned. |

Governance treatment completed 2026-08-19:

1. D1-D6 were approved while Phase 5 was `DRAFT`; no Phase Amendment was required.
2. ADR-0007 was Owner-approved and defines Runtime-native credential authority, at-rest tradeoffs, Profile isolation and the relationship to future `SecretStore` use.
3. `SECURITY.md`, `DATA_MODEL.md`, `ARCHITECTURE.md` and `ROADMAP.md` are synchronized with ADR-0007.
4. The first Batch 1 qualification STOP is preserved. On 2026-08-19 the Owner amended D3/ADR-0007 to authorize the narrow Hermes-native credential compatibility writer described below.
5. A later material change to this authority requires a Phase 5 Amendment and any necessary superseding ADR.

## 4. Owner Decisions Required

- [x] **D1 — China-market-first ProviderPreset catalog.** The product candidate list is DeepSeek, Qwen/Alibaba DashScope, Kimi/Moonshot, MiniMax, GLM/Zhipu, OpenRouter, OpenAI and Anthropic. This is product direction, not a claim that pinned Hermes supports every candidate. Batch 1 must verify the exact Hermes provider ID, credential mechanism/name, config key, China/region endpoint behavior and recommended model IDs. Unsupported candidates are removed from the selectable MVP catalog or shown non-selectable as unsupported; YORVA does not implement their protocols.
- [x] **D2 — Hermes-native credential persistence for the MVP.** Subject to the ADR in Section 3, the official Hermes Profile credential store is the single credential truth. YORVA does not implement SecretStore or `secret_refs` for these model keys and keeps no duplicate copy. SQLite, logs, events, Operations, HTTP responses, diagnostics, argv and Desktop storage never contain the secret.
- [x] **D3 — Official surface first, narrow compatibility fallback approved.** Prefer the pinned Hermes `0.20.2` documented, non-interactive Profile/config/credential surfaces. Qualification proved that the offline official setter requires secret argv and the safe JSON setters require a long-running service. The Owner therefore authorizes only the Hermes adapter to update the exact canonical Profile `.env` with a version-fixed, Profile-scoped, Provider-allowlisted writer. Callers cannot supply paths or env keys. The writer is bounded, preserves unknown entries, changes one allowlisted key, uses same-directory atomic replacement/read-back and fails closed on observed external modification. Hermes Python imports, arbitrary `.env`/YAML editing and any generic file API remain forbidden.
- [x] **D4 — Save and validation are separate.** Saving writes credential plus non-secret provider/model configuration and performs safe read-back/status confirmation. It never sends an inference request or spends tokens. Only **Test connection** starts an explicit bounded validation. `CONFIGURED` is not `VALIDATED`, and neither changes `AVAILABLE` to `MODEL_READY`.
- [x] **D5 — Missing Instance retention.** A `MISSING` Instance and its stable `instanceId` remain indefinitely. Reconciliation does not automatically delete its Hermes credential. Config mutation and validation are rejected while `MISSING` or `UNKNOWN`; credential deletion is an explicit action when a safe official Profile surface can address it. A future cleanup policy requires a later contract.
- [x] **D6 — Five sequential batches.** Execute Section 20 in order. The next batch starts only after the focused gate passes. Once the Spec is `READY`, the Owner may separately authorize automatic continuation through all five batches. No batch may borrow Phase 6 or later scope.

Rejected or changed decisions must be updated in both language versions before `READY`.

## 5. User-Visible Success Flow

1. User opens an `AVAILABLE` Instance and expands/opens its Models panel inside the existing Instances experience.
2. YORVA shows a small catalog grouped as **Recommended in China** and **Other compatible providers**.
3. User selects a supported preset. YORVA shows a short description and recommended model list.
4. User chooses a recommended model or enters a bounded model ID. There is no custom endpoint field.
5. User enters an API key in a password field and selects **Save configuration**.
6. Go resolves `instanceId` to the authoritative current `nativeId` and active Hermes executable.
7. The Hermes adapter uses the qualified Profile credential surface, including the approved narrow fallback where required, to persist the credential and non-secret provider/model configuration, then confirms only safe status/config metadata.
8. The input is cleared. UI shows `CONFIGURED`, but no network validation has run.
9. User selects **Test connection**. YORVA starts a bounded `model.validate` Operation through the existing Operation/process infrastructure.
10. UI shows `PASSED`, `FAILED` or `UNKNOWN` with a local timestamp and safe guidance.
11. After YORVA exits, Hermes itself can use the same Profile configuration through its native credential/config mechanism.

## 6. In Scope

- a small static ProviderPreset catalog led by China-market candidates;
- pinned Hermes qualification for every selectable preset;
- recommended model IDs plus bounded manual model ID input;
- safe read/apply/read-back of Profile provider/model configuration;
- Hermes-native Profile credential set/replace/delete/status through an official surface or the ADR-approved narrow fallback;
- metadata-only credential reads and explicit connection validation;
- authenticated loopback API, OpenAPI/generated client and stable error changes;
- existing Operation integration for validation only;
- conflict protection with Profile deletion/reconciliation;
- bilingual preset-based Desktop UX inside the existing Instance page;
- secret redaction, Profile isolation, process cleanup and restart tests;
- exact-commit verification and independent `AUDIT-005` handoff.

## 7. Out of Scope

- any Phase 4 implementation or freeze remediation;
- Hermes install/repair/upgrade/uninstall or generation/environment changes;
- Profile create/delete/rename/clone/import/export changes except coordination with config mutation;
- Hermes lifecycle, gateway/service startup or supervision;
- OAuth, browser/device-code login, Nous Portal login or token import;
- online Provider marketplace or catalog download;
- dynamic Provider/Runtime plugins or a generic harness framework;
- custom Provider, custom endpoint/base URL, proxy or arbitrary auth scheme;
- fallback chains, routing policies, quotas, pricing or full online model discovery;
- free-form YAML, `.env`, shell, command, environment-variable, path or config-key editors;
- direct or general `.env` read/write/append outside the approved Hermes adapter credential writer;
- Windows user/system environment-variable mutation;
- chat/inference UI, Agent readiness, sessions, memory or persona editing;
- channels, Weixin/WeCom, Skills, MCP, backup/restore, Cloud or telemetry;
- a second SecretStore copy of Hermes model credentials.

## 8. Architecture Boundary and Reuse

```text
Existing React Instance experience
    ↓ existing generated typed client
Existing authenticated loopback API / OpenAPI
    ↓
Application model-config/credential use cases
    ↓                         ↓ existing Operation engine (validation)
Runtime-neutral intent        typed progress/result
    ↓
Hermes adapter
    ↓
Existing executable resolution + commandRunner/process containment
    ↓
Pinned Hermes Profile config/credential/validation surfaces
    └── ADR-0007 narrow canonical `.env` credential fallback where required
```

Go is authoritative for use-case coordination, identity resolution, ProviderPreset selection, command construction and result normalization. Hermes remains authoritative for Hermes Profile config and credential state. React only renders normalized state and holds the password input during the active form interaction. Rust/Tauri gains no model business logic.

ProviderPreset mapping is Hermes-specific and belongs under `services/node/internal/runtime/hermes` or a clearly Hermes-owned file/package. Stable selection/status DTOs may live in a small domain/application-owned model-config area. Do not reorganize unrelated Hermes installation/profile files.

Forbidden flows:

```text
React → Hermes CLI/.env/YAML
React → credential env name or config key
Domain → Hermes provider identifiers
Hermes adapter → HTTP/UI
SQLite/Operation → credential or config authority
Rust/Tauri → provider/credential decisions
```

## 9. ProviderPreset Model

`ProviderPreset` is a minimal compile-time allowlist, not a plugin API.

Conceptual internal fields:

```text
id
displayNameKey
region                   CHINA | GLOBAL
hermesProviderId         adapter-private
credentialEnvName        adapter-private
recommendedModels[]
compatibility            SUPPORTED | UNSUPPORTED
optionalHelpTextKey
```

Use fewer fields if the implementation can remain clear. Requirements:

- statically compiled and version-reviewed;
- actual Hermes mapping owned by the Hermes adapter;
- no arbitrary env/config/shell values;
- no dynamic discovery, marketplace or plugin loading;
- safe API/Desktop DTO omits `hermesProviderId`, `credentialEnvName` and internal config keys;
- only `SUPPORTED` presets are selectable;
- manual model ID is allowed only within a supported preset;
- recommended models are reviewed constants, not an implied complete model catalog.

Product candidate catalog before Batch 1 qualification:

| Group | Candidate | Initial compatibility |
|---|---|---|
| Recommended in China | DeepSeek | `TO_BE_QUALIFIED` |
| Recommended in China | Qwen / Alibaba DashScope | `TO_BE_QUALIFIED` |
| Recommended in China | Kimi / Moonshot | `TO_BE_QUALIFIED` |
| Recommended in China | MiniMax | `TO_BE_QUALIFIED` |
| Recommended in China | GLM / Zhipu | `TO_BE_QUALIFIED` |
| Other compatibility | OpenRouter | `TO_BE_QUALIFIED` |
| Other compatibility | OpenAI | `TO_BE_QUALIFIED` |
| Other compatibility | Anthropic | `TO_BE_QUALIFIED` |

At least one China-market preset must qualify for the MVP. If none qualifies on pinned Hermes, stop for an Owner decision about Hermes version/scope; do not create a provider protocol in YORVA.

## 10. Pinned Hermes Surface Qualification

Integration target: Hermes `0.20.2`, commit `df4b65147d7ddd74dd449f9067aabbca5aef0ec7`.

Batch 1 must establish for every candidate:

1. exact Hermes provider ID and aliases;
2. exact Profile selection mechanism using the existing `nativeId` boundary;
3. exact non-secret model/provider keys and scalar formats;
4. exact credential logical/env name and Profile isolation behavior;
5. whether China/global endpoint selection is built into Hermes without a custom URL;
6. exact official non-interactive set/replace/delete/status surface and the documented reason for any fallback;
7. the secret transport channel, proving it is not argv/output/logs;
8. exact read-back/status output and strict bounded parser requirements;
9. safe non-interactive validation with tools disabled;
10. timeout, cancellation, output-limit and exit/error mapping.

Selection order remains documented official API, documented programmatic protocol, documented CLI, then the ADR-0007-approved narrow compatibility fallback. Current upstream `main` is not evidence for the pinned version.

The completed qualification found no safe offline official credential setter. The fallback is therefore allowed only for candidates whose exact credential key and canonical Profile location are proven from pinned Hermes. A candidate is unsupported if it requires OAuth/login, a custom endpoint/provider protocol, ambiguous storage, Hermes Python imports or any mutation outside that bounded writer.

## 11. Identity, Availability and Model State

- Public routes, Operation target and YORVA relations use `instanceId`.
- Only the Hermes adapter receives `nativeId` to select the Profile.
- `instanceId` and `nativeId` are never interchangeable.
- `AVAILABLE` means only that the Profile exists in the latest successful query.
- `MISSING` is a retained tombstone; `UNKNOWN` means authoritative state is unavailable.
- `MISSING`/`UNKNOWN` reject save, credential mutation and validation.

Configuration state:

| State | Meaning |
|---|---|
| `UNCONFIGURED` | Required provider/model/credential status is incomplete or cannot be confirmed. |
| `CONFIGURED` | Supported preset, model and Hermes-native credential status are confirmed; no connection success is claimed. |

Validation state:

| State | Meaning |
|---|---|
| `NOT_RUN` | No completed explicit validation is available. |
| `PASSED` | Latest explicit validation succeeded. |
| `FAILED` | Provider/model/credential was rejected with a normalized safe error. |
| `UNKNOWN` | Timeout, cancellation, unsafe output or transport ambiguity prevented a verdict. |

No `MODEL_READY` availability exists. `PASSED` does not mean Agent, gateway, channel or lifecycle readiness.

## 12. Runtime and Credential Contracts

Keep contracts focused; exact names may follow current repository conventions:

```text
ListProviderPresets() → safe static presets
ReadModelConfig(installation, nativeId) → normalized safe config/status
ApplyModelConfig(installation, nativeId, presetId, modelId) → safe observed config
SetCredential(installation, nativeId, presetId, secret input) → metadata/status only
DeleteCredential(installation, nativeId, presetId) → metadata/status only
ValidateModel(installation, nativeId, presetId, modelId) → typed validation result
```

Do not build a generic provider plugin registry. Runtime-neutral application intent resolves through the existing Runtime bundle/registry; Hermes-specific mapping remains in the Hermes adapter.

Every process call reuses the existing active executable resolver and `commandRunner`. If a safe credential surface needs stdin, minimally extend the existing runner rather than adding another runner. Reuse current process-tree containment, bounded separate output, allowlisted environment, timeout/cancellation and cleanup.

## 13. Hermes-Native Credential Authority

Subject to D2/ADR approval:

- the selected Hermes Profile's official credential store is the sole source of truth for the model API key;
- YORVA never stores a duplicate API key in SecretStore, SQLite or its own file;
- only the Hermes adapter's approved compatibility writer may open the canonical Profile `.env`; all other production layers are prohibited from doing so;
- an official Hermes surface is preferred; the fallback accepts only `nativeId`, an allowlisted preset and a secret value, never a caller path/env key;
- status is derived only as safe presence metadata through the qualified surface/writer and never returns the value or secret-derived fragments;
- credential mutation is scoped to the exact Profile selected from `nativeId`;
- Profile A's credential must not appear in Profile B's process, status or validation;
- replace/delete failure is normalized and retryable where safe; YORVA does not guess native state;
- a partial Save is reported as `UNCONFIGURED` with a stable incomplete/apply error and can be retried idempotently;
- no cross-file transaction/journal or rollback framework is added for MVP.

Formal role of `.env`:

> Hermes Profile `.env` is Hermes-owned official credential storage, not a general YORVA file API and not an OS-secure YORVA SecretStore. Allowing it is an explicit MVP at-rest security tradeoff that requires the Section 3 ADR.

## 14. SecretStore, SQLite and Windows Environment

Recommended MVP decision:

- do not implement `SecretStore` for Hermes model API keys;
- do not create or populate `secret_refs` for these keys;
- retain the general architecture concept for future YORVA-owned device/cloud/channel secrets after the required ADR clarifies scope;
- do not delete or repurpose the documented `secret_refs` model in this Phase;
- do not store provider/model truth in SQLite; Hermes remains authoritative;
- use existing Operations only for validation projection, never as credential/config truth.

Windows user/system environment mutation is out of scope and disabled by default. Multiple Profiles may hold different keys, so a global environment variable cannot represent correct Profile isolation. A future **Sync to user environment** option requires explicit opt-in, separate security/product design and a later Phase/Amendment.

## 15. Save, Mutation and Concurrency Contract

The product **Save configuration** action uses one credential-resource PUT request, so the client does not coordinate secret and non-secret state across two calls:

```json
{
  "providerPresetId": "deepseek",
  "modelId": "reviewed-or-manual-model-id",
  "value": "write-only-api-key"
}
```

This application use case coordinates `SetCredential` and `ApplyModelConfig` on the server and returns only safe `ModelConfiguration`/credential metadata. `PATCH /config` is only for changing non-secret provider/model fields when a credential can already be confirmed; it never accepts an API key. Rules:

- reject unknown/trailing fields, empty/oversized values and control characters;
- `providerPresetId` must be a selectable static preset;
- `modelId` is bounded text, never a URL, path, config key, env name or shell/argv fragment;
- the API key is accepted once and never echoed;
- resolve fresh supported installation and `AVAILABLE` Instance before mutation;
- use the existing Instance/Profile coordination source so save conflicts with Profile delete/reconcile; do not add an independent lock registry;
- perform no network model call during Save;
- use only exact qualified surfaces, including the approved narrow fallback;
- confirm safe provider/model and credential-status metadata before returning `CONFIGURED`;
- if native steps partially succeed, return a stable partial/incomplete error and safe observed state; do not automatically delete a previously valid credential or claim crash-safe rollback;
- same desired request is safely retryable, but no generic config transaction framework is added.

Credential delete is a separate explicit action. It removes the selected preset credential through the official Profile surface and returns metadata only.

## 16. Authenticated API and OpenAPI

Extend the existing authenticated loopback API and `api/openapi.yaml`; do not create a new transport/client.

Proposed routes:

```text
GET    /api/v1/runtimes/hermes/model-provider-presets
GET    /api/v1/instances/{instanceId}/config
PATCH  /api/v1/instances/{instanceId}/config
GET    /api/v1/instances/{instanceId}/credentials/model-provider
PUT    /api/v1/instances/{instanceId}/credentials/model-provider
DELETE /api/v1/instances/{instanceId}/credentials/model-provider
POST   /api/v1/instances/{instanceId}/model-validation
```

The preset response contains only safe product fields. It does not expose env names, Hermes config keys or internal CLI details. Config GET contains safe provider preset/model/configuration state and latest validation summary. Credential GET returns only `configured`, preset ID and safe timestamp/status metadata.

Credential PUT is the complete Save request from Section 15. Config PATCH is a secret-free closed schema and may change provider/model only when credential state is confirmable. PATCH/PUT use closed schemas and no-store responses. The implementation must extend the existing route method/CORS allowlists for `PATCH` and `PUT`; it must not bypass `routeContract`, bearer authentication, origin checks or generated-client workflow.

`POST .../model-validation` accepts a closed empty body and returns `202` with the existing `Operation` schema. Operation type is `model.validate`, target type is `instance`, and target ID is the stable `instanceId`.

Minimum stable errors:

```text
INSTANCE_NOT_FOUND
INSTANCE_NOT_AVAILABLE
MODEL_PROVIDER_UNSUPPORTED
MODEL_CONFIG_INVALID
MODEL_CONFIG_QUERY_FAILED
MODEL_CONFIG_APPLY_FAILED
MODEL_CONFIG_INCOMPLETE
MODEL_CREDENTIAL_REQUIRED
MODEL_CREDENTIAL_WRITE_FAILED
MODEL_CREDENTIAL_DELETE_FAILED
MODEL_VALIDATION_FAILED
MODEL_VALIDATION_TIMED_OUT
MODEL_VALIDATION_CANCELLED
INSTANCE_CONFIG_CONFLICT
```

Raw Hermes/provider output, paths, env names, config keys and secrets never cross HTTP.

## 17. Explicit Validation Operation

Validation waits on a network provider and therefore reuses the existing durable Operation framework instead of holding the HTTP request open or creating a second task system.

- only the explicit **Test connection** action creates `model.validate`;
- validation resolves current authoritative Profile config/status at start;
- use one fixed harmless prompt and a qualified Hermes surface with tools disabled;
- do not start a gateway or leave a process running;
- use one whole-operation deadline established by Batch 1;
- client cancellation uses the existing Operation cancellation path and process cleanup;
- timeout, cancellation, provider rejection and unsafe output remain distinct stable outcomes;
- model text/raw provider response is not persisted, logged, emitted or returned;
- Operation events remain closed/redacted and Desktop recovers the Operation after restart;
- a failed validation does not modify/delete credential or config.

## 18. Desktop UX and i18n

Reuse the existing sidebar and Instances navigation. Do **not** add a new top-level Models sidebar item or a second navigation tree. Add a Models panel/detail flow for the selected Instance within the current Instances experience.

UI layout:

```text
Add model provider

Recommended in China
  DeepSeek / Qwen / Kimi / MiniMax / GLM (only qualified entries selectable)

Other compatible providers
  OpenRouter / OpenAI / Anthropic (only qualified entries selectable)

Provider description
Model: [recommended-model select] [manual model ID option]
API key: [password input]
[Save configuration] [Test connection]
```

Requirements:

- presets first; no engineering config form;
- never show Hermes config keys, env names, `.env` path or CLI argv;
- no base URL/custom endpoint editor;
- do not imitate a large CC Switch-style catalog;
- reuse existing `i18n.ts` types/messages, `en-US`/`zh-CN` locale persistence and accessibility conventions;
- reuse `formatDateTime` for validation/config timestamps;
- use TanStack Query for server state and existing generated client;
- use page-local React state only for selection/form/password interaction;
- never place API key in query keys/cache, localStorage, sessionStorage, URL, analytics or error objects;
- clear password input after submit/unmount;
- disable Save/Test for `MISSING`, `UNKNOWN` or unsupported installation;
- status must not depend on color alone.

## 19. Security and Logging Invariants

Allowing official Hermes Profile credential persistence does not weaken these rules:

1. SQLite contains no API key plaintext.
2. HTTP GET never returns a secret; PUT/PATCH never echoes it.
3. Logs, events, errors, Operations and diagnostics contain no secret.
4. Command argv/descriptors contain no secret.
5. Secret is absent from URL, browser storage, TanStack Query cache and analytics.
6. Parent provider credentials are not inherited wholesale by children.
7. Credential operations target one authoritative Profile only.
8. No Windows global/user environment mutation occurs.
9. `.env` is not exposed as a file API; only the Hermes adapter's approved bounded credential writer may inspect/update the exact allowlisted entry.
10. Unknown output/native state fails closed without deleting or inventing credential state.
11. Existing loopback authentication, Origin/CSP and no-store response protections remain.
12. Existing process-tree containment cleans up success, failure, timeout and cancellation.

Structured logs may contain `instanceId`, preset ID, action, stable outcome code, duration and timeout/cancel flags. They exclude raw/native output, API key, env name, filesystem path and model response.

Redaction remains defense in depth and covers representative Chinese/global provider key shapes, Authorization headers, structured fields, wrapped errors and child stderr.

## 20. Implementation Batches

Five deliberately small, sequential batches; no extra framework project.

### Batch 1 — Hermes Provider/config/credential qualification and fallback

- inspect pinned Hermes `0.20.2` for every product candidate;
- lock supported ProviderPreset mappings and recommended models;
- prove Profile selection, non-secret config keys, official credential set/delete/status behavior and `.env` ownership;
- preserve the original STOP evidence proving the official offline credential blocker;
- implement and test the narrow Profile-scoped Provider-allowlisted credential writer, including status/set/replace/delete, atomic read-back, cleanup and optimistic conflict detection;
- prove secret never enters argv/output and lock a tools-disabled validation surface;
- write contract evidence/fixtures and finalize errors/DTOs.

Gate: qualification/fallback tests and evidence pass. Stop only if no China preset can use the approved bounded fallback or validation cannot meet Section 17.

### Batch 2 — ProviderPreset and non-secret model config

- implement the static supported catalog in the Hermes adapter;
- implement read/apply/read-back for preset/provider/model config;
- add safe provider/config GET/PATCH OpenAPI and generated-client surface;
- integrate with existing Instance identity, availability and coordination.

Gate: adapter/application/HTTP/OpenAPI tests pass; credential HTTP lifecycle remains for Batch 3.

### Batch 3 — Hermes-native credential lifecycle

- expose the qualified adapter Profile credential status/set/replace/delete lifecycle through application and HTTP boundaries;
- implement metadata GET and write-only PUT/DELETE; credential PUT coordinates the complete Save use case;
- verify restart recovery from Hermes truth and Profile-to-Profile isolation;
- add redaction/no-argv/no-SQLite/no-browser-persistence tests.

Gate: credential tests pass without SecretStore duplication or `.env` access outside the approved Hermes adapter writer.

### Batch 4 — Explicit validation

- add `model.validate` to the existing Operation framework;
- implement tools-disabled contained validation, bounded output/deadline and cancellation;
- add process-tree cleanup, safe-result and restart projection tests.

Gate: repeated success/failure/timeout/cancel/output-limit tests pass with no surviving child and no secret/model-output leak.

### Batch 5 — Desktop integration and audit evidence

- add China-first preset UX inside the existing Instances experience;
- complete English/Simplified Chinese, accessibility, local-time and password-lifetime tests;
- run full verification, Windows/real-Hermes isolated smoke where safe and exact-commit CI;
- update completion evidence and stop at `AUDIT-005 = PENDING`.

Gate: all Phase 5 acceptance tests and full exact-candidate verification pass. Do not merge, freeze, tag or start Phase 6 before independent audit and Owner acceptance.

For every batch: inspect the isolated diff, run focused tests and `git diff --check`, preserve user work, and do not weaken contracts to pass.

## 21. Testing Strategy

| Area | Required proof |
|---|---|
| Provider qualification | Exact pinned IDs/keys/credential names/region behavior; unsupported candidates are not selectable. |
| ProviderPreset | Static allowlist; safe DTO omits env/config internals; no dynamic loading. |
| Identity/availability | API/Operation use `instanceId`; Hermes calls use `nativeId`; MISSING/UNKNOWN block mutation. |
| Config | Exact argv/surface, read-back, manual model bounds, partial/incomplete retry, unrelated config preserved. |
| Credentials | Metadata only; set/replace/delete/restart; Profile A/B isolation; no argv/direct `.env`/SQLite copy. |
| Protocol | Auth/origin/CORS, PATCH/PUT method contract, closed bodies, no echo, method/not-found and generated-client parity. |
| Validation | Explicit only, existing Operation, tools disabled, timeout/cancel/output limit/descendant cleanup. |
| Redaction | Domestic/global key sentinels absent from log/event/error/HTTP/Operation/diagnostics. |
| Desktop | Existing sidebar/i18n/date/query reuse, preset-first EN/zh-CN UX, password cleared/not cached. |
| Environment | Windows user/system env unchanged; ambient provider secrets absent from child. |
| Integration | Save through official Hermes surface, close/restart YORVA, Hermes Profile status remains configured, validate. |

Use an isolated disposable Hermes home/Profile for real credential smoke and never use a real Owner key in fixtures/CI. Do not add a speculative failpoint framework.

## 22. Full Verification

After Batch 5, run the exact repository scripts/CI equivalents for:

- OpenAPI lint/generation and generated-client drift;
- Desktop typecheck, lint, tests, build and dependency audit;
- Go format, full tests, repeated affected tests, race in exact CI, vet, build and vulnerability scan;
- Rust format, tests, clippy, check and dependency audit;
- Windows process containment/timeout/cancel and environment-nonmutation smoke;
- isolated Hermes Profile credential/config restart smoke with fake keys or a controlled test provider;
- Tauri no-bundle/MSI checks only if packaging inputs changed;
- exact-commit GitHub Actions success.

Environment-blocked checks are reported accurately and covered by exact-commit CI where available. Green CI does not override a secret leak or failed required flow.

## 23. Audit and Exit Criteria

Independent `AUDIT-005` must review baseline/scope, ADR compliance, pinned provider evidence, identity separation, official credential surface, Profile isolation, secret scans, API/OpenAPI parity, validation Operation/process cleanup, Desktop password handling, bilingual UX, source-file cohesion and exact-commit CI.

Critical/High findings, secret leakage, unsafe credential targeting, required-flow failure, unsafe process cleanup or missing mandatory evidence make the Gate `FAIL`. Medium/Low findings follow `AUDIT_STANDARD.md`; hypothetical hardening is not automatically blocking.

Phase 5 completion requires:

- [x] actual Phase 4 re-freeze baseline confirmed;
- [x] D1-D6 and the credential-authority ADR approved;
- [x] at least one China-market ProviderPreset qualifies on pinned Hermes;
- [x] user saves provider/model/key without seeing config/env/.env/argv details;
- [x] Hermes can use the same Profile configuration after YORVA exits;
- [x] SQLite, HTTP reads, logs, events, errors, argv and Desktop persistence contain no key plaintext;
- [x] no Windows global/user env mutation;
- [x] `CONFIGURED` and validation result remain separate from `AVAILABLE`;
- [x] full local/CI verification passes on the exact candidate;
- [x] formal audit reaches an acceptable Gate;
- [x] Owner authorizes merge/freeze/tag after audit and CI pass.

The current state is `COMPLETE / FROZEN`. Batches 1-5 are complete, `AUDIT-005R1` is PASS, exact-candidate CI run `32343964969` is SUCCESS, and `main` merge `c45a231` passed final-main CI run `32346079074` plus Windows MSI run `32346079072`. The annotated `phase-005-models-credentials-baseline` tag identifies this frozen closeout. Phase 6 has not started and requires its own approved Phase Spec and authorization.

## 24. Mandatory Stop Conditions

Stop and report if:

1. Phase 4 is not actually re-frozen at the intended start baseline;
2. pinned Hermes safely supports none of the China-market candidates;
3. the approved narrow credential writer cannot safely target the canonical Profile store without secret leakage;
4. safe metadata status/delete or tools-disabled validation is unavailable;
5. custom endpoint/provider protocol, OAuth/login or lifecycle is required;
6. a second navigation/i18n/runner/registry/Operation/date/state system appears necessary;
7. implementation would modify frozen Phase 3/4 behavior rather than extend it;
8. a major dependency/framework or unapproved ADR is required;
9. process cleanup, Profile isolation or secret non-leakage cannot be guaranteed;
10. Chinese and English contracts diverge or a product decision remains unresolved.

The entry gates are approved for Batches 1-5 and the stated closeout sequence. Do not modify Phase 4 or enter Phase 6; do not merge/tag until the Phase 5 audit and exact-commit CI Gate pass.

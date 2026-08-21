# YORVA Phase 6 — Runtime Lifecycle and Messaging Channels

> Status: **READY — IMPLEMENTATION AUTHORIZED**
> Language: English execution mirror
> Owner: Repository owner
> Required baseline: `phase-005a1-post-freeze-corrections-baseline` → `9957775`
> Chinese Owner-review source: `PHASE-006-runtime-lifecycle-messaging-channels.zh-CN.md`
> Roadmap entry: Phase 6, expanded by Owner direction on 2026-08-20
> Implementation branch: current owner-authorized branch
> Execution authorization: **Owner authorized Phase 6 execution on 2026-08-20**

## 1. Objective

Phase 6 makes a configured YORVA Instance operational and reachable through a supported messaging channel without requiring a terminal.

The phase has two ordered product outcomes:

1. provide normalized Instance lifecycle management and implement Hermes start, stop, restart, status and startup/service policy first;
2. provide Weixin and WeCom connection workflows on top of that lifecycle foundation.

The phase must not turn YORVA Core into a Hermes process manager. Core owns normalized lifecycle intent, Operations, concurrency and recovery. The Hermes adapter owns Hermes Profile/gateway commands, Windows service behavior and compatibility details.

## 2. Baseline and Current Reality

Phase 5 is complete and frozen at `phase-005-models-credentials-baseline`.

At that baseline:

- the authenticated start, stop and restart routes exist only as deliberate `CAPABILITY_NOT_SUPPORTED` stubs;
- Instance capability metadata always reports `lifecycle: false`;
- the Runtime registry has no lifecycle feature contract wired into a bundle;
- the Desktop explicitly says Start, Stop and Restart are unavailable;
- YORVA does not install, own, start, stop or reconcile a Hermes Profile gateway;
- Phase 5 model validation does not start a gateway or leave a Runtime process running;
- no `channel_bindings` table, Channel application use case, channel secret authority or Channel Desktop flow is implemented;
- ADR-0007 authorizes Hermes-native **model** credentials only and explicitly does not authorize channel credentials.

The pinned Hermes source exposes Profile-scoped gateway lifecycle through official CLI commands. On Windows, Hermes may use a per-Profile Scheduled Task, a Startup-folder fallback or a detached process. Those mechanisms are Hermes-specific and must not leak into the generic API or domain.

## 3. Approved Scope Direction and Owner Decisions

The following direction is already approved by the Owner on 2026-08-20:

- [x] **D1 — Phase 6 order and scope.** Instance/Runtime lifecycle is moved forward from Phase 7 and implemented before messaging channels in Phase 6.
- [x] **D2 — Unified lifecycle ownership.** Lifecycle intent, normalized state, Operations, conflict control and recovery belong in Runtime-neutral Core/application code. Hermes-specific execution remains inside the Hermes adapter.

Qualification evidence is recorded in `evidence/PHASE-006-BATCH-1-QUALIFICATION.md`.

- [x] **D3 — Hermes lifecycle target.** Start/Stop/Restart manages the selected Hermes Profile messaging gateway, never the immutable installation tree.
- [x] **D4 — Background-service policy.** `ON_LOGIN` is rejected for Phase 6 because the qualified Windows path may prompt for elevation or use an implicit persistence fallback. Phase 6 is manual lifecycle only.
- [x] **D5 — Channel credential authority.** ADR-0008 is accepted. Exact Profile-scoped Hermes-native storage is the sole authority; YORVA stores safe projections only.
- [x] **D6 — WeCom QR compatibility.** The non-public QR compatibility path is rejected. A typed manual Bot ID/Secret flow with authenticated verification satisfies the WeCom exit criterion.
- [x] **D7 — Ephemeral QR delivery.** Initiating-session-only delivery is approved. Shared SSE contains readiness metadata only.
- [x] **D8 — Messaging dependency materialization.** The qualified installed bytes may be used. Missing bytes require a new ADR-0006 sealed generation and never an in-place repair.
- [x] **D9 — Meaning of `CONNECTED`.** Channel `CONNECTED` means an authenticated binding was verified; lifecycle `RUNNING` independently means the gateway is online.
- [x] **D10 — Sender pairing completion.** Owner direction on 2026-08-21 requires the normal Weixin path to list pending sender-pairing requests and approve the eight-character code in Desktop. Hermes-native pairing state remains authoritative; YORVA persists no code, request or grant copy.

Any rejected decision must update the in-scope list, test matrix and exit criteria before implementation authorization.

## 4. Entry Criteria

Implementation may begin only when all of the following are true:

- Phase 5 remains frozen at the required tag and commit;
- the English and Chinese Phase 6 Specs are synchronized;
- D3-D9 have explicit Owner decisions;
- required ADRs are accepted;
- pinned Hermes lifecycle, Weixin and WeCom surfaces have qualification evidence;
- the Spec status is changed to `READY`;
- the Owner explicitly authorizes the implementation batches.

Creating this draft does not authorize a branch, code, migration, dependency or Runtime mutation.

## 5. User-Visible Success Flows

### 5.1 Lifecycle flow

```text
Configured Instance
  -> user selects Start
  -> YORVA creates an Instance lifecycle Operation
  -> Hermes adapter starts the exact Profile gateway
  -> authoritative status becomes RUNNING
  -> Desktop shows Running and the available Stop/Restart actions
  -> closing Desktop does not stop the gateway
```

```text
Running Instance
  -> user selects Stop or Restart
  -> YORVA serializes the mutation for that Instance
  -> Hermes adapter performs bounded graceful lifecycle work
  -> authoritative status becomes STOPPED or RUNNING
  -> Desktop shows the final state
```

### 5.2 Messaging flow

```text
Instance
  -> user selects Connect Weixin or Connect WeCom
  -> YORVA creates a Channel authentication Operation
  -> an expiring QR/auth flow is shown only to the initiating session
  -> authentication is confirmed
  -> safe Channel metadata becomes CONNECTED
  -> lifecycle status independently shows whether the gateway is RUNNING
```

## 6. In Scope

### 6.1 Runtime-neutral lifecycle foundation

- a small `LifecycleManager` Runtime feature contract;
- registry-based lifecycle capability lookup;
- application use cases for lifecycle status, Start, Stop, Restart and startup policy;
- normalized lifecycle state and startup policy;
- durable Instance-targeted lifecycle Operations;
- per-Instance mutation coordination;
- idempotency, authoritative postcondition checks and daemon-restart recovery;
- normalized lifecycle events and safe diagnostics;
- shared Desktop lifecycle presentation driven only by capability/state DTOs.

### 6.2 Hermes lifecycle implementation

- pinned Hermes `0.20.2` Profile gateway status qualification;
- Profile-scoped Start, Stop and Restart for default and named Profiles;
- Windows normal-user lifecycle management;
- manual startup only; `ON_LOGIN` policy management is deferred;
- bounded graceful stop followed by bounded escalation only through the qualified official surface;
- status reconciliation after external Hermes CLI/Studio changes or process crashes;
- no dependency on Hermes internal Python imports or database schemas.

### 6.3 Messaging channels

- channel capability list for `weixin` and `wecom` only;
- list/get Channel binding status;
- connect, disconnect, retry and cancel behavior;
- QR/authentication Operation state;
- ephemeral QR readiness notification and initiating-session retrieval;
- success, failure, cancellation and timeout states;
- safe Channel metadata persistence;
- approved credential authority and redaction;
- localized Desktop UX.
- Weixin sender-pairing pending count and one-time code approval without a terminal.

### 6.4 Lifecycle-related crash/recovery UX moved from Phase 7

- terminalization or authoritative reconciliation of orphaned lifecycle Operations;
- stale/unknown status presentation after daemon or Hermes failure;
- explicit retry actions;
- no implicit restart unless the approved startup policy requires it;
- enough safe diagnostic context to distinguish configuration, service and Runtime failures.

## 7. Non-Goals

Phase 6 does not implement:

- a generic OS process, PID, Scheduled Task, service or shell management API;
- Runtime install repair, uninstall or upgrade;
- Hermes code/generation upgrade or in-place mutation of a sealed generation;
- lifecycle control for a hypothetical second Runtime;
- multiplexed Hermes gateway configuration or cross-Profile routing;
- Skills, MCP, backup/restore, sessions, memory or chat UI;
- additional messaging channels beyond Weixin and WeCom;
- message browsing, sending or conversation management in YORVA;
- Cloud, remote command delivery, accounts, organizations or RBAC;
- arbitrary log tailing or full observability infrastructure;
- automatic elevation or hidden UAC prompts;
- an enterprise service manager or dynamic Runtime plugin system;
- a claim that authenticated Channel state alone means the gateway is online;
- a claim that a running gateway alone means a Channel credential is configured.

## 8. Architecture Boundary and Unified Lifecycle Ownership

Required direction:

```text
React Desktop
  -> authenticated typed Node API
  -> Runtime-neutral lifecycle/channel application use cases
  -> Runtime registry feature contract
  -> Hermes lifecycle/channel adapter
  -> pinned official Hermes surface or separately approved narrow fallback
```

The unified lifecycle portion owns only stable YORVA concepts:

- Instance identity;
- desired action: Start, Stop, Restart or startup policy change;
- normalized lifecycle state;
- Operation creation and state transition;
- idempotency and conflict policy;
- timeout/cancellation policy;
- recovery decision from authoritative adapter status;
- capability and event projection.

The Hermes adapter exclusively owns:

- default versus named Profile argv;
- Hermes gateway command selection;
- Windows Scheduled Task/Startup/detached-process semantics;
- Hermes PID/lock/status files where an approved official surface requires them;
- human-output compatibility parsing;
- Hermes-specific service installation and graceful-drain behavior;
- mapping raw failures to stable YORVA errors.

Forbidden:

```text
React -> Hermes CLI
Tauri -> ordinary Hermes lifecycle business logic
application/domain -> services/node/internal/runtime/hermes
generic lifecycle contract -> PID, task name, profile path or Hermes service type
Hermes adapter -> Desktop DTO/component
```

The phase should add one real feature boundary to the compile-time Runtime bundle. It must not create a broad manager/service/provider framework. New lifecycle application code must resolve the feature through the registry; it must not introduce an `app.HermesLifecycleSource` bridge.

## 9. Normalized Lifecycle Contract

Conceptual Runtime contract:

```go
type LifecycleManager interface {
    Status(ctx context.Context, installation Installation, nativeID string) (LifecycleStatus, error)
    Start(ctx context.Context, installation Installation, nativeID string) error
    Stop(ctx context.Context, installation Installation, nativeID string) error
    Restart(ctx context.Context, installation Installation, nativeID string) error
}
```

Stable observed lifecycle states:

```text
RUNNING
STOPPED
UNKNOWN
```

Transient user-visible states come from the active Operation rather than a second mutable resource-state machine:

```text
STARTING
STOPPING
RESTARTING
CONFIGURING_STARTUP
```

Rules:

- `RUNNING` requires authoritative live evidence from the adapter;
- `STOPPED` requires authoritative absence/not-running evidence;
- query failure, malformed output, ambiguous Profile targeting or stale evidence returns `UNKNOWN` with a stable error code;
- SQLite last-known state never authorizes a mutation or a success result;
- public lifecycle views never expose PID, executable path, Profile home, Scheduled Task name or raw output;
- capability is true only for a qualified supported Runtime/version and safe executable resolution.

## 10. Lifecycle Operation Semantics

Lifecycle mutations return `202 Accepted` with an Operation:

```text
instance.start
instance.stop
instance.restart
instance.lifecycle.configure
```

Every lifecycle Operation targets:

```text
target.type = instance
target.id   = <stable YORVA instanceId>
```

Rules:

- mutation requests use the repository's bounded closed `{}` body unless a typed startup-policy body is required;
- every mutation requires a valid `Idempotency-Key`;
- Start on an authoritatively `RUNNING` Instance succeeds idempotently without spawning a duplicate;
- Stop on an authoritatively `STOPPED` Instance succeeds idempotently;
- Restart requires an authoritatively `RUNNING` Instance; `STOPPED` returns `INSTANCE_NOT_RUNNING` rather than silently changing Restart into Start;
- success requires a postcondition query that observes the requested final state;
- timeout or unknown postcondition must never be reported as success;
- an external command exit code alone is not authoritative success;
- once the external mutation is claimed, cancellation is allowed only if the qualified surface can stop safely; otherwise return `OPERATION_NOT_CANCELLABLE`;
- HTTP requests do not remain open for the external lifecycle workflow;
- database transactions are never held while waiting for Hermes or Windows service control.

Timeouts, output limits and graceful-drain budgets are locked during Batch 1 qualification and then recorded in this Spec before Batch 2 begins.

## 11. Lifecycle Concurrency and Mutation Policy

Use the narrowest Instance scope.

For one Instance, the following conflict:

- Start vs Stop vs Restart vs startup-policy mutation;
- lifecycle mutation vs Instance delete;
- lifecycle mutation vs Channel connect/disconnect when the operation changes credentials or gateway activation;
- lifecycle mutation vs another lifecycle Operation recovered after daemon restart.

Additional rules:

- deleting a `RUNNING` or `UNKNOWN` Instance fails closed; the user must obtain `STOPPED` first;
- a lifecycle transition does not hold the installation-wide create/delete lock longer than required to resolve and validate identity;
- operations on different Instances may proceed concurrently when Hermes proves the Profile-scoped service model is independent;
- model/config/credential mutations may not race a lifecycle transition;
- Phase 6 does not automatically restart after a model or Channel mutation unless that behavior is explicitly added to the approved Spec;
- locks live in application coordination, not Desktop state;
- every goroutine and external process wait has an owner and cancellation path.

## 12. Pinned Hermes Lifecycle Qualification

Batch 1 must qualify the exact packaged Hermes source commit and installed `0.20.2` behavior.

Candidate official surfaces:

```text
default Profile: hermes gateway <status|start|stop|restart>
named Profile:   hermes -p <nativeID> gateway <status|start|stop|restart>
startup policy:  hermes [-p <nativeID>] gateway install <closed flags>
```

Qualification must prove:

- exact default/named Profile targeting;
- no interactive prompt on a background worker path;
- no secret in argv or environment;
- normal-user behavior on Windows;
- whether service absence is distinguishable before Start;
- whether the official CLI can configure `MANUAL` and `ON_LOGIN` without hidden elevation;
- authoritative structured or fixture-bounded status evidence;
- already-running/already-stopped behavior;
- graceful stop, escalation, timeout and descendant cleanup;
- duplicate-start prevention;
- behavior after Desktop and `yorvad` exit;
- behavior after gateway crash and Windows login;
- fixed stdout/stderr bounds and safe redaction;
- fail-closed behavior for unknown versions/output/service state.

The adapter invokes only the trusted absolute active-generation launcher with fixed command verbs and a validated native Profile ID. It does not use `cmd.exe`, PowerShell command strings, PATH-selected Hermes, `shell=true` or imported Hermes Python internals.

If official output is human-readable only, a version-pinned fixture-tested parser may be approved. Unknown, localized-unqualified, truncated or oversized output fails closed.

## 13. Startup/Service Management

Lifecycle management is moved from Phase 7 into Phase 6. Persistent login-start policy remains deferred after qualification.

The qualified boundary for Phase 6 is:

- Start is an explicit manual action and never enables login persistence;
- Stop and Restart affect only the selected Profile gateway;
- existing Hermes login items may be observed for diagnostics but are not created, changed or removed by YORVA;
- an absent login item uses the fixed official `gateway install --no-start-on-login --start-now` invocation, whose qualified Windows implementation creates no persistence;
- any unexpected prompt, elevation request or persistence change fails the Operation.

Any service installation/removal that may require elevation must:

- be a separate explicit user action;
- describe the action before the OS prompt;
- use only the qualified Hermes-owned service surface;
- never fall back silently to a weaker persistence mechanism;
- return a stable state if approval is declined;
- receive a security/architecture ADR if it crosses the automatic-elevation review trigger.

Tauri may mediate a narrowly scoped native approval handoff only if required by the accepted ADR. It must not construct Hermes commands or decide lifecycle policy.

## 14. Lifecycle Recovery and Reconciliation

On daemon startup and explicit refresh:

1. resolve the stable Instance to its authoritative native Profile;
2. resolve a supported active Hermes installation;
3. query the Hermes lifecycle adapter;
4. project `RUNNING`, `STOPPED` or `UNKNOWN`;
5. reconcile orphaned lifecycle Operations without repeating a mutation blindly.

Recovery rules:

- orphaned Start may succeed only if authoritative state is `RUNNING`;
- orphaned Stop may succeed only if authoritative state is `STOPPED`;
- orphaned Restart cannot prove that a fresh restart occurred from final `RUNNING` alone and therefore becomes terminal `FAILED`/`LIFECYCLE_RESULT_UNKNOWN` unless durable evidence from the qualified surface proves the transition;
- an `UNKNOWN` query never becomes success;
- recovery does not automatically issue Start, Stop or Restart;
- Desktop close must not stop a successfully managed Hermes gateway;
- unexpected Hermes exit becomes observed `STOPPED`/`UNKNOWN` plus safe diagnostics; automatic recovery occurs only through the approved Hermes startup policy;
- SSE is notification only; GET remains the source of truth after reconnect.

## 15. Lifecycle API and OpenAPI

Required local API surface:

```text
GET  /api/v1/instances/{instanceId}/lifecycle
POST /api/v1/instances/{instanceId}/start
POST /api/v1/instances/{instanceId}/stop
POST /api/v1/instances/{instanceId}/restart
```

OpenAPI must replace the Phase 4 `lifecycle: false` literal and unsupported-only responses with typed capabilities, lifecycle views and `202 Operation` responses. CORS, authentication, error envelopes and request-size bounds remain unchanged.

Lifecycle SSE notifications contain only safe identifiers and normalized state/error fields. They never contain raw process output, PID, path, task name, command, environment or secret.

## 16. Channel State and Contracts

Supported channel identifiers are a closed list for this phase:

```text
weixin
wecom
```

Normalized Channel binding states:

```text
NOT_CONFIGURED
CONNECTING
CONNECTED
DISCONNECTED
FAILED
UNKNOWN
```

Subject to D9:

- `CONNECTED` means the credential/binding was authenticated and verified;
- it does not mean the Hermes gateway process is currently `RUNNING`;
- `RUNNING` does not mean either Channel is configured;
- `UNKNOWN` is used when Runtime-native truth cannot be queried safely;
- ordinary GET responses expose safe configured/status metadata only.

Conceptual Runtime contract remains small:

```go
type ChannelManager interface {
    ListChannels(ctx context.Context, installation Installation, nativeID string) ([]ChannelState, error)
    BeginConnect(ctx context.Context, installation Installation, nativeID string, req ChannelConnectRequest, events ChannelEventSink) error
    Disconnect(ctx context.Context, installation Installation, nativeID string, channel string) error
    PairingStatus(ctx context.Context, installation Installation, nativeID string, channel string) (PairingStatus, error)
    ApprovePairing(ctx context.Context, installation Installation, nativeID string, channel string, code SecretValue) error
}
```

Hermes-specific QR polling, account files, environment names, remote endpoint details and gateway activation behavior do not enter the Core contract.

## 17. Channel Operations, QR Delivery and Disconnect

Channel mutations use durable Operations:

```text
channel.connect
channel.disconnect
```

Connect stages may include:

```text
preparing
qr_ready
waiting_for_scan
waiting_for_confirmation
verifying
committing
```

QR rules:

- QR payloads and credential-equivalent URLs exist in bounded memory only;
- the shared SSE stream emits only `channel.qr.ready` with the Operation ID and expiry metadata;
- the QR payload is fetched through an authenticated, Operation-scoped, initiating-session-only mechanism approved by D7;
- no QR payload is placed in URL query parameters, Operation rows, `operation_events`, logs, diagnostics, audit rows, SQLite, browser storage or backups;
- QR data has an explicit expiry and is cleared after success, failure, cancellation, timeout or daemon shutdown;
- SSE reconnect does not replay QR payloads;
- a lost/expired QR requires a new bounded authentication attempt.

Disconnect rules:

- remove the authorized local credential/binding material where the approved authority permits;
- update safe Channel metadata only after authoritative confirmation;
- never claim remote account/bot revocation unless the official platform surface actually performed it;
- if remote revocation is unavailable, Desktop must explain that local disconnect does not delete the remote Weixin/WeCom bot identity;
- disconnect must not delete unrelated Profile configuration or another Channel's material.

## 18. Channel Credential Authority and Data

Phase 6 requires a new credential-authority ADR before implementation.

The ADR must choose exactly one authority per Channel and define:

- whether credentials are Hermes-native or YORVA-owned;
- Profile/Instance isolation;
- Windows at-rest behavior;
- how a background gateway reads credentials after Desktop/daemon exit;
- set, replace, status and delete semantics;
- Weixin account/token and WeCom Bot ID/Secret handling;
- whether any version-pinned compatibility writer is authorized;
- prohibition on duplicate authorities and global environment mutation.

Mandatory invariants regardless of the choice:

- no channel credential plaintext in SQLite, API reads, Operations, events, logs, diagnostics, audit metadata, Desktop storage, argv or URLs;
- no silent plaintext fallback;
- no caller-supplied path or environment key;
- exact Instance/Profile and Channel allowlists;
- secret-bearing buffers are short-lived and cleared where practical;
- `channel_bindings` stores only safe metadata;
- `secret_refs` is used only if YORVA owns the sole OS-secure secret authority;
- ADR-0007 must not be broadened implicitly.

## 19. Channel Persistence

Phase 6 adds the `channel_bindings` migration described by `DATA_MODEL.md`.

Required constraints:

- foreign key to `instances(id)` with intentional delete behavior;
- uniqueness for the supported one-binding-per-Instance/Channel model;
- closed normalized Channel type/state validation in application/domain code;
- no QR, token, secret, raw response or sensitive URL in `metadata_json`;
- Runtime remains authoritative for Runtime-native state;
- stale metadata is clearly a projection and never authorizes secret/lifecycle mutation.

Lifecycle status should remain a live adapter query with optional safe last-known projection. Phase 6 must not persist PID or make SQLite a process authority.

Lifecycle and Channel actions write secret-free local audit metadata for Start, Stop, Restart, startup-policy change, Connect and Disconnect.

## 20. Messaging Dependencies and Sealed Generations

The current installed Hermes `0.20.2` environment was read-only qualified with the required `aiohttp`, `cryptography`, `httpx` and `qrcode` modules present.

Phase 6 must not mutate the active sealed generation in place or permit Hermes lazy installation to do so.

The approved path is:

- use the already qualified bytes for lifecycle and channel authentication;
- if any required byte is absent, stop and build/activate a new exact-lock generation under ADR-0006 before continuing.

Any generation-building path must preserve ADR-0006:

- new Install Transaction and generation ID;
- immutable reviewed locks and source pins;
- no user-selected package index;
- complete manifest/seal verification;
- compare-and-swap activation;
- rollback/recovery from filesystem transaction truth;
- no reuse of SQLite Operation as activation authority.

If this requires an upgrade/repair primitive, it must be explicitly authorized as a narrow Phase 6 prerequisite rather than silently implementing the general Phase 7/Runtime upgrade roadmap.

## 21. Weixin and WeCom Qualification

### 21.1 Weixin

Qualification must verify:

- official/documented iLink QR endpoints and payload bounds;
- QR refresh, scan, confirmation and timeout states;
- redirect-host validation;
- credential/account persistence authority;
- disconnect/revocation behavior;
- dependency availability;
- secret redaction and Profile isolation;
- a testable non-interactive adapter path without importing Hermes internals or parsing terminal QR art.

### 21.2 WeCom

Pinned Hermes explicitly documents that its QR creation endpoints are not part of the public WeCom developer API. Phase 6 may not silently treat them as stable.

If D6 approves a fallback, it must be:

- version-pinned and WeCom-only;
- fixed HTTPS endpoint/host with no caller-provided URL;
- bounded in response size, polling interval and total duration;
- strict about response schemas and redirects;
- fail-closed on unknown output/status;
- covered by fixtures and a manual Windows smoke;
- documented as compatibility behavior, not a guaranteed public API.

If D6 rejects the fallback, the Spec must explicitly authorize a secure typed manual Bot ID/Secret flow or remove WeCom from mandatory Phase 6 exit criteria.

## 22. Desktop UX and i18n

Lifecycle UX appears before Channel UX in the implementation order.

Required lifecycle presentation:

- Running, Stopped and Unknown status;
- active Starting/Stopping/Restarting Operation state;
- Start/Stop/Restart actions enabled from capability and authoritative state;
- explicit startup policy control if D4 is approved;
- confirmation for Stop/Restart when active work may be interrupted;
- recovery messaging after daemon/Runtime crash;
- no PID, service name, path or raw Hermes output;
- English and Simplified Chinese strings.

Required Channel presentation:

- Weixin and WeCom capability cards on an exact Instance;
- Connect, Disconnect, Retry and Cancel actions;
- QR modal with expiry countdown and scan/confirmation state;
- safe account label/external ID only;
- explicit distinction between Channel Connected and Gateway Running;
- clear remote-revocation limitation;
- unsupported Runtime/version and missing dependency explanations;
- keyboard-accessible modal and status announcements;
- no QR/token persistence in localStorage, sessionStorage or Zustand.
- pending Weixin sender-pairing count, an eight-character pairing-code input and an explicit Approve action;
- no pairing code in browser storage, query keys, error text, Operation rows, logs or durable YORVA state.

TanStack Query owns daemon resources. Local React state may hold the currently displayed expiring QR only for the modal lifetime.

## 23. Stable Error Codes

The final list is locked during qualification. Expected normalized errors include:

```text
CAPABILITY_NOT_SUPPORTED
RUNTIME_NOT_INSTALLED
RUNTIME_UNSUPPORTED
INSTANCE_NOT_FOUND
INSTANCE_NOT_RUNNING
INSTANCE_LIFECYCLE_CONFLICT
LIFECYCLE_STATUS_UNKNOWN
LIFECYCLE_START_FAILED
LIFECYCLE_STOP_FAILED
LIFECYCLE_RESTART_FAILED
LIFECYCLE_TIMED_OUT
LIFECYCLE_RESULT_UNKNOWN
LIFECYCLE_SETUP_REQUIRED
LIFECYCLE_APPROVAL_DECLINED
CHANNEL_NOT_SUPPORTED
CHANNEL_CONFLICT
CHANNEL_AUTH_FAILED
CHANNEL_AUTH_TIMEOUT
CHANNEL_AUTH_CANCELLED
CHANNEL_STATE_UNKNOWN
CHANNEL_DISCONNECT_FAILED
CHANNEL_DEPENDENCY_MISSING
CHANNEL_PAIRING_QUERY_FAILED
CHANNEL_PAIRING_CODE_INVALID
CHANNEL_PAIRING_LOCKED
CHANNEL_PAIRING_APPROVAL_FAILED
```

Public errors contain user-safe text only. Desktop behavior depends on codes and typed state, never message matching.

## 24. Implementation Batches — After Authorization Only

### Batch 1 — Qualification and governance lock

- qualify pinned Hermes lifecycle and Windows service behavior;
- qualify Weixin/WeCom authentication surfaces;
- decide D3-D9;
- accept required ADRs;
- lock timeouts, output bounds, states, errors and dependency path;
- update this Spec before code-bearing batches.

### Batch 2 — Runtime-neutral lifecycle foundation

- add the small Runtime lifecycle contract and registry capability;
- implement generic application status/Operation/concurrency/recovery logic;
- update lifecycle OpenAPI and generated Desktop types;
- remove lifecycle=false hard-coding without adding Hermes branches to Core/UI.

### Batch 3 — Hermes lifecycle adapter

- implement exact Profile-scoped status/Start/Stop/Restart;
- enforce manual-only startup with no hidden persistence or elevation;
- add bounded command/process behavior and postcondition checks;
- add adapter fixtures, cancellation/timeout and restart recovery tests.

### Batch 4 — Lifecycle Desktop and resilience

- expose lifecycle controls and Operations;
- implement crash/recovery and external-state reconciliation UX;
- verify Desktop close does not stop Hermes;
- complete lifecycle manual Windows smoke before Channel implementation.

### Batch 5 — Channel contracts, data and secret authority

- implement ChannelManager registry boundary and application use cases;
- add `channel_bindings` migration;
- implement the accepted credential authority;
- add typed APIs, Operation stages and ephemeral QR broker.

### Batch 6 — Weixin

- implement Weixin connect/status/disconnect;
- implement QR expiry, cancellation and redaction;
- verify exact Instance/Profile isolation.

### Batch 7 — WeCom

- implement the approved secure manual Bot ID/Secret flow;
- implement status/disconnect and revocation disclosure;
- verify exact Instance/Profile isolation.

### Batch 8 — Channel Desktop, full verification and audit handoff

- implement localized Channel UX;
- run full checks and Windows lifecycle/channel smoke;
- collect immutable candidate evidence;
- enter Phase 6 audit and stop feature work.

### Batch 8A — Owner-required sender-pairing completion

- add typed pending-count and approve-code APIs for Weixin only;
- keep Hermes-native pairing data authoritative and Profile-exact;
- add localized Desktop pending/request approval UX;
- verify code redaction, invalid/expired code, lockout and cross-Profile isolation.

No batch automatically authorizes the next. A blocking qualification, security or architecture finding stops execution.

## 25. Test Matrix

| Scenario | Expected result | Level |
|---|---|---|
| Unsupported Runtime/version lifecycle query | capability false or stable unsupported result; no process | application/adapter/API |
| Start stopped Instance | Operation reaches `SUCCEEDED` only after `RUNNING` truth | application/adapter/integration |
| Start running Instance | idempotent success; no duplicate process | application/adapter |
| Stop running Instance | graceful bounded stop and authoritative `STOPPED` | adapter/integration |
| Stop stopped Instance | idempotent success | application/adapter |
| Restart running Instance | old process exits; new running truth; no duplicate | adapter/Windows smoke |
| Restart stopped Instance | `INSTANCE_NOT_RUNNING`; no process | application/API |
| Unknown lifecycle output | `UNKNOWN`; no success inference | adapter fixtures |
| Lifecycle timeout/cancel | terminal stable error; no orphan child owned by command | adapter/application |
| Concurrent Start/Stop/Restart | one mutation wins; others conflict deterministically | application/race |
| Delete running/unknown Instance | fails closed; Profile remains | application/API |
| Different Instance lifecycle mutations | proceed concurrently when qualified safe | application/race |
| Daemon exits after successful Start | gateway remains running | Windows smoke |
| Daemon restart with orphan Start/Stop | terminal result derived from authoritative state | application/integration |
| Daemon restart with orphan Restart | no false success from final running state alone | application/integration |
| Missing login item on Start | fixed non-persistent start path; no prompt or elevation | adapter/Windows smoke |
| Channel capability list | only qualified Weixin/WeCom appear | adapter/API/Desktop |
| Weixin QR success | expiring QR -> confirmed -> safe `CONNECTED` metadata | adapter/application/Desktop/manual |
| QR expires | operation times out; payload cleared | application/security |
| QR cancel | polling stops; payload cleared; no credential commit | application/security |
| SSE reconnect | GET restores state; QR payload is not replayed | API/Desktop |
| Second authenticated non-initiator | cannot retrieve another Operation's QR payload | API/security |
| WeCom unknown QR schema/status | fail closed with stable error | adapter fixtures |
| Channel credential redaction | no plaintext in DB/API/log/event/audit/diagnostics | security/integration |
| Profile A vs Profile B | lifecycle and Channel actions never cross identity/secret scope | adapter/application |
| Disconnect | only target Channel material removed; remote revocation truth is accurate | adapter/application/Desktop |
| Weixin pending sender pairing | exact Profile pending count is shown without exposing code material | adapter/API/Desktop |
| Valid Weixin pairing code | exact Profile grant is approved and the code is never persisted or returned | adapter/API/Desktop/security |
| Invalid/expired pairing code | stable invalid-code error; no grant and no code disclosure | adapter/API/Desktop/security |
| Pairing approval lockout | stable locked error; no bypass or retry loop | adapter/API/Desktop/security |
| Sealed generation dependency attempt | in-place mutation rejected; approved new-generation path used | install/integrity |
| Migration from Phase 5 DB | succeeds with uniqueness/FK and no secret columns | persistence |

## 26. Full Verification

Before audit, run the applicable repository gates from `DEVELOPMENT.md`, including:

```text
pnpm install --frozen-lockfile
pnpm audit --audit-level low
pnpm api:lint
pnpm api:generate
generated OpenAPI drift check
pnpm typecheck
pnpm lint
pnpm test
pnpm build
go test ./...
go test -race ./... on supported CI
go vet ./...
go build ./cmd/yorvad
govulncheck
cargo fmt --check
cargo test --locked
cargo clippy --locked --all-targets -- -D warnings
cargo check --locked
cargo audit
Windows sidecar/lifecycle smoke
Tauri no-bundle release build
MSI packaging/inspection where Phase 6 changes packaged Hermes dependencies
```

Required manual evidence:

- default and named Profile Start/Stop/Restart on Windows;
- Desktop close while gateway remains running;
- real Weixin scan/confirm/connect/disconnect;
- real WeCom approved auth path;
- no plaintext secret/QR in inspected logs and SQLite;
- exact packaged Hermes generation dependency and integrity evidence.

## 27. Phase-Specific Audit Requirements

The independent audit must apply every dimension in `AUDIT_STANDARD.md` and specifically verify:

- Runtime-neutral lifecycle code has no Hermes import/branch;
- the Runtime bundle owns feature selection and capabilities;
- no generic process/service/shell API was created;
- exact Instance ID resolves to exact authoritative Profile for every mutation;
- Start/Stop/Restart postconditions are authoritative and fail closed;
- concurrency covers lifecycle vs delete/config/channel conflicts;
- Desktop close and daemon restart behavior match process ownership docs;
- Windows service/elevation behavior is explicit and approved;
- no sealed generation was mutated in place;
- Channel credential authority matches the accepted ADR;
- WeCom fallback, if any, is exactly bounded and documented;
- QR is expiring, initiating-session-only and absent from durable/event/log surfaces;
- Channel `CONNECTED` and lifecycle `RUNNING` are not conflated;
- migrations work from empty and Phase 5 schemas;
- exact-candidate CI and Windows manual evidence are attached.

Any confirmed credential/QR disclosure, arbitrary process surface, cross-Profile mutation, duplicate gateway start, false lifecycle success, hidden elevation or sealed-generation mutation is blocking.

## 28. Exit Criteria

Phase 6 can pass only when:

- a supported Hermes Instance exposes `lifecycle: true` from qualified capability data;
- Start, Stop, Restart and live status work for default and named Profiles without a terminal;
- manual-only lifecycle and lifecycle recovery behave exactly as approved;
- Desktop close does not stop managed Hermes gateways;
- Weixin and the D6-approved WeCom path complete their mandatory auth flows;
- a Weixin sender can complete the required Hermes pairing approval from Desktop without a terminal;
- Channel and lifecycle status are both visible and semantically distinct;
- no QR or channel credential plaintext appears in SQLite, logs, Operations, events, diagnostics or ordinary API responses;
- no sealed generation was changed in place;
- all mandatory tests and Windows smoke flows pass;
- independent audit returns `PASS` or Owner-accepted `PASS WITH CONDITIONS`;
- the accepted baseline is merged, frozen and tagged.

Phase 7 implementation remains prohibited until that freeze.

## 29. Mandatory Stop Conditions

Stop and return to Owner review if:

- D3-D10 remain unresolved;
- the official Hermes lifecycle surface cannot be made non-interactive, Profile-exact and fail-closed;
- Start/Stop/Restart requires a generic shell or imports Hermes internals;
- Windows service management requires hidden or automatic elevation;
- authoritative postconditions cannot distinguish `RUNNING`, `STOPPED` and `UNKNOWN`;
- Channel credentials cannot have one approved authority;
- WeCom QR requires unapproved undocumented behavior;
- QR cannot be limited to the initiating authenticated session;
- required messaging dependencies would mutate an active sealed generation;
- a material architecture/security conflict lacks an accepted ADR;
- required verification or real Windows evidence cannot be produced.

## 30. Completion Evidence

Implementation candidate recorded in
`docs/phases/evidence/PHASE-006-BATCHES-2-8-IMPLEMENTATION.md`:

- lifecycle implementation commits: `2e03a78`, `acf4139`;
- Channel implementation commits: `28b6f0f`, `b415f79`;
- local Go, OpenAPI, Desktop and Tauri no-bundle verification: passed;
- built Windows Desktop read-only Channel UX check: passed;
- real owner-authenticated Weixin/WeCom smoke: pending;
- independent audit, merge/freeze and annotated Phase 6 tag: pending.

The remaining completion evidence is:

- exact-commit CI run;
- Windows lifecycle and Channel smoke record;
- migration evidence;
- secret/QR inspection evidence;
- audit report and Gate Decision;
- merge commit;
- final-main CI/MSI evidence;
- annotated Phase 6 baseline tag.

# YORVA Phase 3 — Hermes Installation

> Status: AUDIT / R3 REMEDIATION
> Owner: Repository owner
> Previous phase: Phase 2 — Hermes Discovery & Compatibility
> Previous baseline: `phase-002-hermes-discovery-baseline-r1`
> Previous baseline commit: `5b89d22ed5e7ae3f4374a26f0fcda54bdabc6bf9`
> Previous gate: `AUDIT-002A1-hermes-discovery.md` — PASS
> Implementation: AUDIT / R3 REMEDIATION
> Amendment: `docs/phases/amendments/AMENDMENT-003A1-embedded-hermes-source.md` — ACCEPTED FOR IMPLEMENTATION
> Amendment: `docs/phases/amendments/AMENDMENT-003A2-china-dependency-distribution.md` — ACCEPTED FOR IMPLEMENTATION
> Amendment: `docs/phases/amendments/AMENDMENT-003A3-managed-node-prerequisites.md` — ACCEPTED FOR IMPLEMENTATION
> Phase 3 audit: `AUDIT-003` — FAIL; `AUDIT-003R1` — FAIL; `AUDIT-003R2` — FAIL; `AUDIT-003R3` — FAIL; `AUDIT-003R4` — PENDING
> Target platform: Windows user-scope installation
> Target Hermes release: `v2026.8.16` / package `0.20.2`
> Spec review date: 2026-08-17
> Owner approval date: 2026-08-17

The Repository Owner approved this Specification on 2026-08-17 and later authorized implementation. Feature work is frozen except for `AUDIT-003R3` remediations. This document does not declare Phase 3 COMPLETE, FROZEN, PASS or ACCEPTED.

## 1. Objective

Allow a user with no working Hermes installation to explicitly install one supported, official Hermes Runtime from YORVA Desktop without using a terminal.

The installation is a durable, cancellable YORVA Operation. YORVA verifies the provenance and exact bytes of the reviewed official installer, drives its documented structured stage protocol, owns every spawned process, and uses the frozen Phase 2 discovery contract as the authoritative post-install check.

```text
Hermes NOT_INSTALLED
→ user reviews the source, version, destination and host changes
→ user explicitly confirms Install
→ YORVA creates runtime.install Operation
→ verified official installer runs through approved stages
→ Phase 2 discovery returns SUPPORTED for the managed installation
→ Operation becomes SUCCEEDED
```

Phase 3 installs Hermes. It does not configure a Hermes profile, credentials, models, channels or Runtime lifecycle.

## 2. Entry Criteria

- [x] Phase 2 implementation and amendment 002A1 are accepted.
- [x] `AUDIT-002A1-hermes-discovery.md` has Gate `PASS`.
- [x] Phase 2 is `ACCEPTED / COMPLETE / FROZEN`.
- [x] `phase-002-hermes-discovery-baseline-r1` exists on `main`, resolves to `5b89d22ed5e7ae3f4374a26f0fcda54bdabc6bf9`, and is pushed.
- [x] `main` and `origin/main` were synchronized and the working tree was clean when this Specification was drafted.
- [x] The official Windows installation surface and its stage protocol were reviewed at an immutable upstream commit.
- [x] Repository Owner approved this Phase 3 Specification on 2026-08-17.
- [x] Repository Owner authorizes Phase 3 implementation.

Implementation authorization was granted. The remaining gate is independent re-audit `AUDIT-003R4`. Historical `AUDIT-003`, `AUDIT-003R1`, `AUDIT-003R2` and `AUDIT-003R3` reports remain FAIL.

## 3. User-Visible Success Flow

1. Desktop refreshes Phase 2 discovery immediately before offering installation.
2. Starting a new installation is enabled only for `NOT_INSTALLED` on supported Windows hosts. Discovery does not hide an already-running, type/target-validated `runtime.install` Operation.
3. Desktop presents a bilingual confirmation surface containing:
   - official source and exact Hermes version;
   - fixed user-scope destination;
   - expected downloads and host changes;
   - the fact that user `PATH` and `HERMES_HOME` are updated by the official installer;
   - the fact that no model, API key, profile or channel is configured;
   - Cancel and Install actions.
4. Explicit confirmation sends one typed install request with an idempotency key.
5. Desktop follows the returned Operation through typed stages and SSE notifications.
6. The user may cancel while a stage is running. Cancellation terminates the entire owned process tree before the Operation becomes `CANCELLED`.
7. After the approved stages finish, the application runs the unchanged Phase 2 discovery use case.
8. Success requires `SUPPORTED`, version `0.20.2`, and a selected command contained by the official managed installation root.
9. Desktop refreshes discovery and shows the installed version and command path.

Desktop closing does not implicitly cancel an installation. On reopening, Desktop lists durable Operations and follows a type/target-validated active `runtime.install` independently of discovery. A partial host (including `BROKEN_EXECUTABLE`, `MALFORMED_VERSION` or `UNSUPPORTED`) still shows Operation status, stage, safe diagnostics and Cancel. Terminal history is never restored as active. A `hermes.prerequisites` Operation is never rendered as Hermes install. When the followed Operation reaches a terminal status, Desktop refreshes discovery and shows the Phase 2 success card if the result is `SUPPORTED`.

## 4. In Scope

- Windows user-scope installation of the reviewed official Hermes release `v2026.8.16` / `0.20.2`;
- a focused Runtime installation capability and Hermes-owned installer adapter;
- a minimal durable Operation engine sufficient for `runtime.install`;
- authenticated local HTTP endpoints and OpenAPI schemas for starting, reading and cancelling installation Operations;
- idempotent start requests and one active install per Node and Runtime kind;
- exact official installer source, commit, size and content-digest verification;
- the official PowerShell installer protocol version and manifest verification;
- an exact allowlist of reviewed, non-interactive installer stages;
- direct executable-plus-argv invocation with bounded output, cancellation and full Windows process-tree ownership;
- fixed official user-scope install paths; no user-selected path;
- structured, redacted progress, error normalization and diagnostics;
- explicit user confirmation before mutation;
- safe retry UX for a failed, cancelled or interrupted YORVA-owned attempt;
- post-install Phase 2 discovery and compatibility verification;
- persistence of Operations and the successfully accepted Runtime installation metadata;
- English and Simplified Chinese Desktop copy;
- deterministic unit, contract, migration, protocol, process-lifecycle and Desktop tests;
- documentation and generated client synchronization.

## 5. Out of Scope

Hard boundaries:

- macOS or Linux Hermes installation in Phase 3;
- selecting an arbitrary version, branch, commit, mirror, URL or install directory;
- installing a moving `latest` Hermes release at runtime;
- Hermes upgrade, downgrade, repair, uninstall or rollback;
- overwriting or adopting an unknown pre-existing installation;
- moving the accepted compatibility ceiling beyond Phase 2's `>=0.19.0 <0.21.0` policy;
- Hermes Profile / YORVA Instance management;
- API keys, model/provider configuration or persona editing;
- Hermes process, agent, gateway or service lifecycle;
- Weixin, WeCom or other messaging/channel setup;
- YORVA Skills or MCP management;
- backup, restore, Cloud or remote installation;
- building or launching Hermes's separate Desktop application;
- automatic Administrator service installation or a generic elevation broker;
- a generic package manager, script runner, shell endpoint or arbitrary command API;
- a dynamic Runtime plugin system or speculative multi-Runtime installer framework;
- forking, patching or importing Hermes internal Python modules;
- starting Phase 4 work.

The official distribution may lay down its own bundled templates and skills as part of the reviewed bootstrap stage. That is installer-owned distribution content, not YORVA Skills/Profile management. YORVA must not inspect, edit, expose or manage that content in Phase 3.

## 6. Architecture Boundary

```text
React Desktop
    ↓ authenticated typed local HTTP
yorvad HTTP transport
    ↓
Install application use case
    ├── durable Operation repository
    ├── Phase 2 discovery use case
    └── Runtime install contract
            ↓
        Hermes installer adapter
            ├── verified official script source
            └── owned Windows process runner
                    ↓ fixed executable + fixed argv
              Official Hermes installer protocol
```

Ownership:

- React owns confirmation, localized presentation, Operation polling/events and explicit cancel.
- HTTP owns authentication, idempotency-header validation and DTO mapping.
- The application layer owns preflight, Operation transitions, concurrency, retry eligibility and post-install discovery.
- The domain owns Runtime-neutral Operation and installation concepts and stable error codes.
- `services/node/internal/runtime/hermes` owns all Hermes release pins, installer URLs, paths, stage names, argv, manifest/result parsing and Hermes-specific errors.
- a narrow OS process package owns direct spawn, Job Object containment, cancellation, timeout, bounded streams and child reaping.
- SQLite owns YORVA Operations and normalized accepted installation metadata; Hermes remains authoritative for its actual installed state.
- Tauri remains the native shell and daemon lifecycle boundary. It does not run the Hermes installer or contain install business logic.

Forbidden dependencies:

```text
React → PowerShell / Hermes installer / filesystem
HTTP handler → shell or generic process execution
Core/domain → Hermes URLs, paths, stages or PowerShell
Tauri → Hermes install orchestration
Hermes adapter → React or transport DTOs
Official installer → user-provided URL/path/argv
```

No new ADR is expected because installation Operations, the Runtime adapter and privilege rules are already defined by the frozen architecture. Stop for an ADR and Owner decision if implementation requires a generic installer framework, automatic elevation, a new trust boundary, a different process model or a change to frozen Phase 1/2 contracts.

## 7. Official Source and Integrity Contract

Upstream evidence reviewed on 2026-08-17:

| Field | Reviewed value |
| --- | --- |
| Repository | `NousResearch/hermes-agent` |
| Release | `v2026.8.16` |
| Package version | `0.20.2` |
| Immutable commit | `df4b65147d7ddd74dd449f9067aabbca5aef0ec7` |
| Script path | `scripts/install.ps1` |
| Exact raw URL | `https://raw.githubusercontent.com/NousResearch/hermes-agent/df4b65147d7ddd74dd449f9067aabbca5aef0ec7/scripts/install.ps1` |
| Expected size | `233712` bytes |
| YORVA-reviewed SHA-256 | `2e1de1867299ce34d5fc73ce63022934acb8966f69f3f53306a37afc3dac29a3` |
| Installer protocol | `1` |

Evidence links:

- release: <https://github.com/NousResearch/hermes-agent/releases/tag/v2026.8.16>
- immutable commit: <https://github.com/NousResearch/hermes-agent/commit/df4b65147d7ddd74dd449f9067aabbca5aef0ec7>
- reviewed installer: <https://github.com/NousResearch/hermes-agent/blob/df4b65147d7ddd74dd449f9067aabbca5aef0ec7/scripts/install.ps1>

The release API published no downloadable release assets, checksum or signature for this installer. The SHA-256 above is a YORVA-reviewed content digest calculated twice from the immutable raw file; it must never be described as an upstream-published checksum or signature.

Rules:

1. Release metadata is compiled into the Hermes adapter as a closed manifest. It is not fetched from `latest` at install time.
2. Download uses a Go HTTPS client directly against the exact URL above. No `irm | iex`, `curl | shell`, `install.cmd`, branch URL, tag URL, redirect to an unapproved host or user-configurable mirror is allowed.
3. Redirects are rejected unless an implementation review explicitly proves an exact HTTPS host/path rule is necessary. No authentication cookies or provider credentials are sent.
4. Response size is limited to 512 KiB. Both size and SHA-256 must match before the file can be executed.
5. The script is written exclusively into an Operation-private directory below YORVA's state root, created without a reparse-point escape. Do not execute from a shared temp directory.
6. The script is re-hashed immediately before every process invocation. Any change is `RUNTIME_INSTALL_INTEGRITY_FAILED`.
7. The private script and transient output are deleted after terminal success, failure or cancellation. Cleanup failure is a redacted warning, not permission to report false success.
8. Updating any release, commit, script byte, digest, stage contract or compatibility target requires a reviewed Spec amendment, new fixtures and audit evidence. It is never an automatic “latest” update.
9. The verified upstream script contains fixed official dependency sources. Phase 3 may use only those reviewed sources and fixed parameters. The adapter cannot inject user-supplied URLs or package sources.
10. Amendment 003A1: acquire the official GitHub commit archive at `df4b65147d7ddd74dd449f9067aabbca5aef0ec7` with a 180-second bound. On transport/unavailability only, use the MSI-bundled archive with the compiled size `71869305` and SHA-256 `2ed02f76aaf5dab0bfd320bdbfa10aad0f67e00cbbf87906cde05462681708ba`. YORVA materializes that tree and does not spawn the official PowerShell `repository` stage. Integrity mismatch does not fall back. No Gitee, `gitclone.com`, `kkgithub.com` or user-supplied source is permitted.
11. Amendment 003A2: the existing dependency stages receive adapter-owned `https://pypi.tuna.tsinghua.edu.cn/simple` and `https://registry.npmmirror.com` endpoints. Inherited Python, uv, pip and npm registry settings are removed first. These fixed mirrors are artifact transport only; the official lockfiles remain unchanged and the feature is not described as fully offline.

## 8. Official Installer Protocol and Stage Policy

Use the official documented programmatic mode. Resolve the trusted Windows PowerShell executable from the OS Windows directory, not a same-name binary found first on `PATH`, and invoke it directly with argv. No command string or interpolation is allowed.

Each probe or stage invocation includes fixed process options equivalent to:

```text
<trusted-powershell.exe>
  -NoLogo
  -NoProfile
  -NonInteractive
  -ExecutionPolicy Bypass
  -File <verified-operation-private-install.ps1>
  <fixed script arguments>
```

Script arguments always include the reviewed commit, `main` branch, official user paths and non-interactive/JSON mode as appropriate. All values are adapter-owned. The HTTP request supplies none of them.

Before mutation:

1. `-ProtocolVersion` must return exactly `1`.
2. `-Manifest` must parse as bounded JSON.
3. The manifest must match the reviewed ordered stage names, categories and `needs_user_input` flags. Unknown, missing, duplicated, reordered or changed stages fail closed with `RUNTIME_INSTALL_MANIFEST_MISMATCH`.

Approved execution order:

| Order | Official stage | Purpose | Policy |
| ---: | --- | --- | --- |
| 1 | `uv` | Hermes-managed Python package tool | required |
| 2 | `python` | supported Python runtime | required |
| 3 | `git` | official repository transport/tooling | required |
| 4 | `node` | official Node stage | verified in the official manifest; never spawned. Replaced by YORVA-managed Node (`AMENDMENT-003A3`) |
| 5 | `system-packages` | official ripgrep/ffmpeg prerequisite handling | required stage; bounded warning behavior may be retained |
| 6 | `repository` | official Hermes checkout pinned to the reviewed commit | required identity; YORVA materializes the verified archive and does not spawn the official stage (`AMENDMENT-003A1`) |
| 7 | `venv` | isolated Hermes Python environment | required |
| 8 | `dependencies` | Hermes Python dependencies | required |
| 9 | `node-deps` | official Node dependency stage | verified in the official manifest; never spawned. Replaced by YORVA-managed `npm ci` after Hermes exists (`AMENDMENT-003A3`) |
| 10 | `path` | official launcher directory and `HERMES_HOME` user variables | required |
| 11 | `config-templates` | official initial distribution templates | required, but never exposed as YORVA profile/skills management |
| 12 | `bootstrap-marker` | official completion marker | required |

Known manifest stages that must not execute:

- `desktop`: builds Hermes's separate Desktop application;
- `platform-sdks`: reads messaging tokens and installs channel SDKs;
- `configure`: configures API keys/models and requires user input;
- `gateway`: starts messaging lifecycle.

The full reviewed manifest still includes `platform-sdks`, `configure` and `gateway`; their presence is verified, then they are deliberately skipped by YORVA policy. `desktop` must be absent because YORVA never passes `-IncludeDesktop`.

For each approved stage, YORVA starts a new owned process and passes `-Stage <exact-name> -NonInteractive -Json`. The final non-empty stdout line must be one bounded JSON object with:

```text
stage, ok, skipped, reason, duration_ms
```

The returned stage name must equal the requested name. Missing, duplicate or malformed result frames fail closed. Human installer output before the final frame is bounded diagnostic input only; it is never returned to Desktop or used for branching. `reason` is never displayed raw.

## 9. Host Mutation and Privilege Boundary

Phase 3 is Windows user-scope only.

Fixed official destinations:

```text
Hermes home:    %LOCALAPPDATA%\hermes
Installation:   %LOCALAPPDATA%\hermes\hermes-agent
Launcher PATH:  %LOCALAPPDATA%\hermes\hermes-agent\bin
```

Paths are derived from the current user's OS-known local application-data location and canonicalized. They are not accepted from HTTP, Desktop, environment configuration or CLI flags. The install directory may be absent, or may be a retry-eligible YORVA-owned partial attempt; an unrelated occupied path fails with `RUNTIME_INSTALL_TARGET_OCCUPIED`.

The confirmation UI must disclose that the reviewed official installer may:

- download Hermes and required official dependencies, or materialize the verified bundled official source archive when GitHub is unavailable; remaining dependency stages may still need network;
- create an isolated Python environment and install Node dependencies;
- install Hermes-managed `uv` and PortableGit when needed;
- use approved Windows package sources for reviewed prerequisites;
- create official Hermes bootstrap/config-template directories;
- add only the official Hermes launcher directory to the current user's `PATH`;
- set the current user's `HERMES_HOME`;
- preserve an existing Hermes `.env` or `config.yaml` rather than overwrite it.

YORVA and `yorvad` remain normal-user processes. YORVA does not relaunch itself as Administrator, install a privileged service or expose an elevation API. If an approved official prerequisite cannot finish without privilege, normalize the result to `RUNTIME_INSTALL_PRIVILEGE_REQUIRED` with a safe explanation. Introducing automatic/narrow elevation requires an ADR, security review and Owner approval before code.

## 10. Preflight, Existing Install and Recovery Rules

Immediately before creating a new Operation, and again after acquiring the install coordination lock, run the unchanged Phase 2 discovery use case.

| Discovery result | Install policy |
| --- | --- |
| `NOT_INSTALLED` | installation may proceed after target-path validation |
| `SUPPORTED` | reject with `RUNTIME_INSTALL_ALREADY_PRESENT` |
| `UNSUPPORTED` | reject; Phase 3 is not upgrade/downgrade |
| `AMBIGUOUS` | reject; user must resolve candidates outside this phase |
| `BROKEN_EXECUTABLE` | reject unless it is a validated YORVA-owned partial-attempt retry |
| `MALFORMED_VERSION` | reject; do not overwrite uncertain state |
| `TIMED_OUT` | reject as retryable preflight failure |
| cancelled/error | do not mutate the host |

Recovery is narrow:

- a retry is eligible only when the durable local Operation history proves YORVA previously started the same pinned Phase 3 install at the same fixed target and ended `FAILED`, `CANCELLED` or `OPERATION_INTERRUPTED`;
- retry requires all of: non-empty durable `source_pin` equal to the expected pinned commit; non-empty durable `ownership_nonce`; a versioned filesystem ownership record whose HMAC, operation ID, runtime kind, canonical target and pin match that durable row; and a current partial-tree inventory digest that still matches the recorded manifest;
- an empty, migrated or mismatched `source_pin` or nonce fails closed and never authorizes retry;
- the marker is not public commit text. It is a versioned JSON ownership record containing schema version, operation ID, runtime kind, canonical target, source pin, identity and inventory digest, authenticated with HMAC-SHA256 over the durable nonce;
- extra executables, foreign files, changed owned files, missing owned files, malformed or copied markers, wrong operation IDs, target mismatches, reparse points/symlinks and manifest mismatches all reject automatic replacement;
- the expected target must remain canonically contained under the fixed Hermes home and contain no reparse-point escape;
- any successful install after that attempt, foreign installation evidence, changed origin, user-selected path or unrecognized content disables automatic retry;
- retry uses the same pinned script and repeats approved stages from the beginning because the official stages are designed to be idempotent;
- ownership handoff is explicit: keep and re-verify the previous Operation proof until a same-volume atomic replace writes the new Operation record; later `venv`, `dependencies`, `path`, template and marker stages refresh the authenticated inventory only after that handoff, and only for the current Operation identity;
- a refresh never signs a copied marker, a different Operation, or a tree whose current record no longer authenticates;
- YORVA never deletes an unknown directory, never infers whole-tree ownership from marker presence alone, and never performs an uninstall or rollback;
- if official recovery moves an invalid YORVA-owned checkout aside, the Operation records only a safe warning and destination category, never raw user paths.

After cancellation or failure, Desktop explains whether Retry is safe. When it is not safe, Desktop provides a correlation ID and non-destructive guidance; it does not offer “force install.”

## 11. Runtime Contract

Add one focused capability. Exact Go spelling follows repository conventions; semantics are fixed:

```go
type Installer interface {
    Install(ctx context.Context, req InstallRequest, progress ProgressSink) (Installation, error)
}

type InstallRequest struct {
    RuntimeKind RuntimeKind
}
```

Phase 3 does not add `Upgrade`, `Repair` or `Uninstall` behavior. If the existing conceptual `Installer` interface currently mentions future methods, implementation must split the capability or expose only the method with a real caller; do not create no-op methods.

The generic application/domain contract contains no Hermes URL, script, stage, path, PowerShell or version-pin field. `ProgressSink` receives stable Runtime-neutral stage identifiers and warnings, never raw command output.

An `Installation` is returned only after Phase 2 post-check succeeds. It contains normalized Runtime kind, canonical selected path, detected version and support state. Hermes remains the source of truth; persisted metadata is a last-known accepted record, not proof that the Runtime still exists.

## 12. Operation Contract

Phase 3 implements the smallest durable Operation capability required by installation.

Operation type:

```text
runtime.install
```

Statuses:

```text
PENDING → RUNNING → SUCCEEDED | FAILED | CANCELLED
```

Stable stages:

```text
preflight
source.download
source.verify
protocol.verify
install.uv
install.python
install.git
install.node
install.system-packages
install.repository
install.venv
install.dependencies
install.node-deps
install.path
install.config-templates
install.bootstrap-marker
postcheck.discovery
cleanup
```

Rules:

- create and commit `PENDING` before external work starts;
- transition to `RUNNING` before the first download;
- do not hold an HTTP request or SQLite transaction during external work;
- stage is authoritative; percentage progress remains `null` rather than inventing time precision;
- operation timestamps are UTC and API timestamps use RFC 3339;
- one idempotency key returns the same Operation for the same install request;
- only one non-terminal `runtime.install` Operation exists per Node and Runtime kind;
- another key during active work returns `RUNTIME_INSTALL_IN_PROGRESS` and the active Operation ID;
- cancellation is idempotent and terminal only after all owned processes are stopped and reaped;
- terminal Operations cannot be cancelled;
- daemon startup converts stale `PENDING` or `RUNNING` install Operations to `FAILED` with `OPERATION_INTERRUPTED` after process ownership guarantees cleanup;
- Desktop/SSE disconnect does not cancel;
- events are notifications; `GET /operations/{id}` remains source of truth.

## 13. API / OpenAPI Changes

Authenticated local endpoints:

```text
POST /api/v1/runtimes/hermes/install
GET  /api/v1/operations/{operationId}
GET  /api/v1/operations/{operationId}/log
GET  /api/v1/operations?targetType=runtime-kind&targetId=hermes&limit=...
POST /api/v1/operations/{operationId}/cancel
```

Start request:

- requires `Idempotency-Key`, 1–128 visible ASCII characters;
- accepts a closed JSON object with no version, URL, path, mirror, command, arguments or environment fields;
- returns HTTP `202` plus the Operation resource;
- unsupported Runtime kinds are rejected before mutation.

The operation list query is bounded and intended for Desktop reload/recovery, not an unbounded history export. Cancellation returns the current Operation representation. Stable application errors use the standard YORVA error envelope.

SSE adds/uses:

```text
operation.started
operation.progress
operation.completed
operation.failed
operation.cancelled
```

Every event contains only operation ID, type, status, stable stage, safe error code, timestamp and correlation ID as applicable. It excludes installer output, environment values, raw errors and secrets.

`api/openapi.yaml` remains the transport source of truth. Generated Desktop types must be regenerated and verified clean. Domain entities are not generated from transport schemas.

## 14. Persistence and Migration

Add deterministic SQLite migrations for:

1. `operations`, following `DATA_MODEL.md`, including status/stage/error/idempotency/correlation/timestamps and indexes;
2. `runtime_installations`, following `DATA_MODEL.md`, because Phase 3 explicitly accepts one managed installation after authoritative post-check.

Do not add `instances`, profiles, credentials, channels or generic package tables.

Required invariants:

- globally unique non-null idempotency key;
- database/application enforcement of one active install per Node and Runtime kind;
- valid status and terminal timestamp constraints;
- a successful installation row is unique by Node, Runtime kind and canonical install path;
- no installation row is created for `PENDING`, `RUNNING`, `FAILED` or `CANCELLED` work;
- post-check success upserts normalized path/version/support metadata in the same short transaction that marks the Operation `SUCCEEDED`;
- no transaction spans downloads, external commands or discovery;
- raw output, stack traces, environment, installer script, secrets and credential-bearing URLs are never stored;
- migrations pass from both an empty database and the Phase 2 schema;
- migration files are immutable after release.

Operation rows are durable management/audit evidence for Phase 3. A full generic audit-log subsystem is not added merely for this phase.

## 15. Timeout, Cancellation and Process Cleanup

The installation is long-running, but every unit is bounded.

- source download: 30-second connect/header timeout, 120-second overall deadline, 512-KiB body limit;
- protocol/manifest probes: 30 seconds each;
- required prerequisite stages (`uv`, `python`, `git`): 10 minutes each;
- official `node` and `node-deps` stages are never spawned (`AMENDMENT-003A3`); YORVA-managed Node/npm use a 15-minute dependency timeout;
- official `system-packages` remains a 45-second skippable official stage;
- repository: 10 minutes;
- virtual environment: 10 minutes;
- Python dependencies: 30 minutes;
- path/config/bootstrap stages: 2 minutes each;
- whole Operation safety ceiling: 60 minutes.

Timeout values are constants owned by the Hermes adapter and tested with injected clocks/process fakes where practical. They are not API parameters.

Windows child rules:

1. create each PowerShell process suspended;
2. assign it to a kill-on-close Job Object before resume;
3. resume only after successful assignment;
4. capture stdout and stderr separately, each limited to 1 MiB per invocation;
5. on cancellation, timeout, output overflow or daemon shutdown, terminate the complete Job Object;
6. wait for the direct child and all descendants to exit;
7. treat unexpected surviving descendants after a short fixed grace period as failure, terminate them and never report success;
8. close all handles and stop all goroutines on every path.

There is no Start-then-bind window. The Phase 2 Windows process-containment invariant is reused, not weakened.

## 16. Environment and Logging Boundary

Each installer process receives one centrally built allowlisted environment sufficient for Windows execution and the reviewed official installer: OS directories, current-user profile/application-data/temp locations, architecture, locale, `PATH`, and fixed `HERMES_HOME`/install inputs. Do not inherit provider credentials, API keys, Python injection settings, arbitrary `HERMES_*` values or user-supplied environment variables.

At minimum exclude or overwrite:

```text
PYTHONPATH
PYTHONHOME
PYTHONSTARTUP
PYTHONINSPECT
PIP_*
UV_*
OPENAI_*
ANTHROPIC_*
GOOGLE_*
GEMINI_*
HERMES_* except the fixed HERMES_HOME
```

Tests must use sentinel secrets to prove they do not reach the child or logs. Environment handling may not break standard Windows certificate/proxy behavior; any credential-bearing proxy value must be treated as secret and never logged.

Emit one structured application log per Operation transition and stage outcome. Desktop/daemon persist those records as JSON lines in `{dataDir}/logs/install.ndjson` (Windows: `%APPDATA%\com.yorva.desktop.dev\logs\install.ndjson`) and still mirror them to stderr. Desktop polls `GET /api/v1/operations/{id}/log` to show the same structured lines. Each record includes only:

- operation/correlation ID;
- Runtime kind;
- stable stage and status;
- elapsed duration;
- stable error code;
- timeout/cancellation/output-limit flags;
- approved warning identifiers.

Do not log raw stdout/stderr, script contents, full paths, environment values, repository URLs with credentials, tokens or installer `reason`. Diagnostic tails may be held transiently in bounded memory for internal classification, then discarded.

## 17. Error Model

| Code | Meaning | Retryable |
| --- | --- | --- |
| `RUNTIME_INSTALL_PLATFORM_UNSUPPORTED` | Phase 3 installation is unavailable on this OS/architecture. | false |
| `RUNTIME_INSTALL_ALREADY_PRESENT` | Phase 2 already found a supported Hermes installation. | false |
| `RUNTIME_INSTALL_STATE_CONFLICT` | Existing unsupported, malformed, ambiguous or foreign-broken state blocks install. | false |
| `RUNTIME_INSTALL_IN_PROGRESS` | Another Hermes install Operation is active. | true |
| `RUNTIME_INSTALL_TARGET_OCCUPIED` | Fixed target contains non-retry-eligible data. | false |
| `RUNTIME_INSTALL_SOURCE_UNAVAILABLE` | Exact official source could not be downloaded. | true |
| `RUNTIME_INSTALL_INSUFFICIENT_DISK` | The destination volume lacks space to materialize the official source. | true |
| `RUNTIME_INSTALL_INTEGRITY_FAILED` | Size or SHA-256 did not match. | false |
| `RUNTIME_INSTALL_PROTOCOL_UNSUPPORTED` | Official protocol is not exactly version 1. | false |
| `RUNTIME_INSTALL_MANIFEST_MISMATCH` | Reviewed stage manifest changed or is malformed. | false |
| `RUNTIME_INSTALL_STAGE_FAILED` | An approved official stage failed. | stage-dependent |
| `RUNTIME_INSTALL_PRIVILEGE_REQUIRED` | A required prerequisite cannot finish as the normal user. | false |
| `RUNTIME_INSTALL_TIMEOUT` | A stage or whole-Operation deadline expired. | true |
| `RUNTIME_INSTALL_OUTPUT_LIMIT` | Installer output exceeded a bound. | false |
| `RUNTIME_INSTALL_POSTCHECK_FAILED` | Installation ran but Phase 2 did not verify the exact supported managed result. | true |
| `RUNTIME_INSTALL_CANCELLED` | User cancellation completed after cleanup. | true |
| `OPERATION_INTERRUPTED` | Daemon exited before a terminal result. | true |
| `OPERATION_NOT_CANCELLABLE` | Operation is terminal or cannot safely transition. | false |

Public messages are safe and localized by stable code/stage. Raw upstream reasons are internal diagnostic input only. HTTP/SQLite failures retain existing YORVA codes where already defined; do not duplicate them to match this table cosmetically.

## 18. Desktop UX

Extend the existing left-sidebar Runtime page; do not redesign unrelated screens.

Required states:

- not installed / Install available;
- confirmation summary;
- queued/running with current stable stage;
- optional prerequisite warning;
- cancelling;
- cancelled;
- failed with safe code-specific guidance and correlation ID;
- interrupted with safe Retry eligibility;
- succeeded, followed by the Phase 2 supported discovery card;
- install unavailable on non-Windows;
- blocked by supported/unsupported/broken/malformed/ambiguous state when no type/target-validated active `runtime.install` exists.

Requirements:

- all copy exists in English and Simplified Chinese typed translation resources;
- locale switching remains immediate and persistent;
- times render in the user's local timezone; transport values remain UTC RFC 3339;
- installation starts only from an explicit confirmation action;
- disabling/double-click protection is backed by server idempotency, not UI state alone;
- progress uses stage names, not fake percentage precision;
- raw console output, stack traces and secret-bearing data are never rendered;
- Cancel remains available only while cancellation is safe, including after Desktop reload while a validated `runtime.install` is still active;
- Retry appears only when the application marks the prior attempt retry-eligible;
- discovery gates only the new-install confirmation; an active validated Operation remains visible under partial discovery;
- success must refresh query state instead of duplicating discovery into a client store.

## 19. Testing Strategy

Implementation must add regression tests at the owning layers. Network-dependent live installation is never part of the default unit suite.

| Scenario | Expected result | Level |
| --- | --- | --- |
| install requested from each Phase 2 discovery state | only `NOT_INSTALLED` or validated YORVA retry proceeds | application table test |
| same idempotency key repeated | same Operation returned, one worker | application + HTTP |
| second key while active | typed in-progress conflict with active ID | application + persistence |
| empty/invalid idempotency key | request rejected before mutation | HTTP/OpenAPI |
| closed empty-object body (both start endpoints) | `{}` accepted; unknown/non-object/null/double/trailing/missing/oversized rejected with stable protocol error | HTTP |
| Desktop reload + active install + BROKEN/MALFORMED discovery | status, stage, diagnostics and Cancel remain visible | React tests |
| terminal or wrong type/target Operation | not restored as active Hermes install | React tests |
| non-Windows host | stable unsupported result, no process | application/adapter |
| foreign occupied target | fail without deleting/moving data | adapter integration |
| exact script bytes | accepted | source verifier |
| byte/size change | integrity failure, never executed | source verifier |
| redirect/unapproved host/oversize/timeout | fail closed | HTTP source adapter |
| protocol not `1` | fail before any stage | adapter contract |
| exact reviewed manifest | accepted | adapter contract fixture |
| unknown/missing/duplicate/reordered/interactive stage change | manifest mismatch | adapter contract |
| excluded stage | `desktop`, `platform-sdks`, `configure`, `gateway` never spawned | argv audit test |
| approved stage | exact executable and argv, no shell string | command descriptor test |
| stage result success/skipped/failure/malformed | normalized transition/error | adapter tests |
| raw reason contains sentinel secret/path | never reaches API, DB, event or logs | security test |
| output over 1 MiB | entire process tree killed and reaped | Windows process test |
| stage timeout | entire process tree killed and reaped | Windows process test |
| explicit cancellation | child/grandchild killed, Operation then CANCELLED | Windows lifecycle + app |
| immediate descendant spawn | no pre-assignment escape | real Windows process test |
| normal parent exit with descendant | descendant cannot survive reported completion | real Windows process test |
| daemon restart with active row | `OPERATION_INTERRUPTED`, safe retry policy applied | persistence/startup |
| cancellation/failure retry | non-empty exact pin+nonce+ownership record, safe target, idempotent stages | application/adapter |
| empty/wrong pin, copied marker, foreign/changed/missing file | retry rejected; uncertain trees never deleted | adapter |
| ZIP/tarball entry-count, member-size, total-size, prefix, traversal, symlink | extract rejected; dest cleaned | adapter |
| whole-Operation deadline | context cancelled, process tree killed, temp cleaned, `RUNTIME_INSTALL_TIMEOUT` | application |
| post-check supported 0.20.2 at managed root | installation upsert + SUCCEEDED atomically | integration |
| post-check wrong version/path/ambiguous/broken | no accepted installation, FAILED | integration |
| empty DB and Phase 2 DB migrations | deterministic schema and constraints | migration |
| Desktop confirmation/progress/cancel/retry/success | localized accessible behavior | React tests |
| Chinese/English switch and local time | correct copy and timezone rendering | React tests |
| generated client | clean regeneration from valid OpenAPI | contract/CI |

Use a repository-owned fake installer fixture implementing protocol 1 for deterministic tests. The fixture may simulate stage results and process trees but must not become production fallback behavior.

A real official install smoke is manual/audit evidence only and must run in a disposable Windows VM or other explicitly isolated test account with no valued Hermes state. It must verify source pin, default user-scope install, cancellation cleanup where feasible, final `0.20.2` discovery and no excluded stage execution. Never run a destructive real install against an Owner machine as an automatic test.

## 20. Verification Matrix

Before reporting implementation complete, run the repository's exact available equivalents of:

```text
pnpm api:lint
pnpm typecheck
pnpm lint
pnpm test
pnpm build
pnpm audit

go test ./...
go test ./... -count=20                 # affected packages at minimum
go test -race ./...
go vet ./...
go build ./...
govulncheck ./...

cargo fmt --all -- --check
cargo test
cargo clippy --all-targets --all-features -- -D warnings
cargo check
cargo audit

Windows daemon/Desktop lifecycle smoke
Windows installer process-tree smoke
Tauri release build --no-bundle
OpenAPI validation and generated-client clean check
```

If local `go test -race` is blocked by a missing C toolchain, record it accurately and require exact-commit CI to run and pass the race suite. Do not report an unrun check as PASS. Existing inherited Cargo audit allowlisted warnings must be reported separately; no new vulnerability or warning may be hidden.

Exact-commit CI must pass all Web/API, Go/Node including race, and Windows native/Tauri jobs before independent audit. Implementation completion does not authorize merge, freeze or tag.

## 21. Exit Criteria

Implementation-candidate checks for `AUDIT-003R4`. These do not constitute an audit PASS:

- [x] Owner-approved Spec is unchanged or formally amended.
- [x] Installation is Windows user-scope and available only from valid preflight state.
- [x] Source is the exact immutable reviewed commit and digest.
- [x] Protocol and full manifest verify before mutation.
- [x] Official `node` and `node-deps` are verified then skipped; YORVA-owned managed Node/npm replace them. Four excluded capabilities never execute.
- [x] No HTTP or Desktop input controls command, URL, version, path or environment.
- [x] Installation is a durable, idempotent, cancellable Operation.
- [x] Process tree is contained before resume and fully cleaned on every terminal path.
- [x] Host mutations and privilege boundary are shown before explicit confirmation.
- [x] No profile, credential, model, channel, gateway, lifecycle, upgrade or Phase 4 behavior exists.
- [x] Post-check reuses Phase 2 and verifies supported `0.20.2` under the managed root.
- [x] Accepted installation metadata is persisted only after authoritative success.
- [x] English and Simplified Chinese Desktop states are complete.
- [x] Migrations, protocol, OpenAPI, generated types and governing docs agree.
- [ ] Focused and full verification matrices pass or exact environmental blockers are recorded.
- [ ] Exact-commit CI passes.
- [ ] Independent `AUDIT-003R4` is PENDING. Historical `AUDIT-003`, `AUDIT-003R1`, `AUDIT-003R2` and `AUDIT-003R3` remain FAIL.

Phase 3 is not accepted, merged, frozen or tagged by satisfying implementation criteria. Those actions require an independent audit Gate `PASS` and a separate governance task.

## 22. Audit Requirements

Create a new historical report:

```text
docs/phases/audits/AUDIT-003-hermes-installation.md
```

Prefer a fresh independent agent/context. The implementation agent's completion statement is not gate evidence.

The audit must independently verify:

- exact baseline, implementation commit and exact-commit CI run;
- source pin, size/digest and truthful provenance wording;
- no mutable Hermes `latest`, arbitrary mirror or unverified script path;
- full manifest comparison and exact executed/excluded stage set;
- no Desktop/profile/platform-SDK/configure/gateway execution;
- closed argv and environment construction;
- pre-spawn Windows Job Object assignment and descendant cleanup;
- timeout, cancellation, output-limit and daemon-interruption behavior;
- idempotency, concurrency and durable Operation transitions;
- target ownership/retry rules and absence of destructive overwrite;
- privilege boundary and disclosed host mutations;
- error/log/event/DB redaction with sentinel secrets;
- migrations from empty and Phase 2 databases;
- Phase 2 post-check exact path/version/support behavior;
- Desktop bilingual confirmation, progress, cancellation, recovery and local time;
- all Out-of-Scope exclusions;
- full verification and exact-commit CI.

Any Critical, High, Medium or blocking Low finding produces Gate `FAIL` under `AUDIT_STANDARD.md`. Preserve failed reports and use `AUDIT-003R1`, `R2`, and so on for remediation audits. Do not weaken this Specification or tests to clear a finding.

## 23. Known Risks and Decisions

1. **Upstream release has no installer asset signature/checksum.** Mitigation: immutable official commit, exact URL, YORVA-reviewed size/SHA-256, private storage and re-hash before every invocation. Residual risk is documented; do not mislabel the digest as upstream attestation.
2. **The reviewed official script downloads transitive prerequisites.** Mitigation: execute only the verified script, exact reviewed stages and fixed official sources; disclose mutations; no input can replace sources. A future source change requires amendment/review.
3. **Installation changes user environment and downloads substantial dependencies.** Mitigation: explicit bilingual confirmation, user scope, fixed destination and no hidden YORVA elevation.
4. **Cancellation cannot roll back arbitrary completed installer stages.** Mitigation: full process cleanup, durable state, no destructive rollback and narrow same-pin retry.
5. **Official templates include Hermes-owned skills/persona scaffolding.** Mitigation: treat them as opaque official distribution artifacts; no YORVA management or editing in Phase 3.
6. **Windows-only delivery leaves other platforms unsupported.** This is the deliberate smallest complete Phase 3 scope accepted with the Owner-approved Specification.
7. **GitHub may be unreachable on some networks.** Mitigation (Amendment 003A1): bounded official commit-archive download, then one verified MSI-bundled official archive. Residual risk: remaining official stages still need network for uv/Python/Node/PyPI/npm; this is not a full offline installer.

No unresolved architecture or security blocker was found during Specification review. The Owner approved the Windows-only, fixed-version and no-automatic-elevation boundaries. Implementation authorization was granted on 2026-08-17. Independent audit remains pending.

## 24. Contract and Documentation Changes During Implementation

- [x] `docs/PROTOCOL.md`
- [x] `docs/RUNTIME.md`
- [x] `docs/DATA_MODEL.md`
- [x] `docs/SECURITY.md`
- [ ] `docs/ARCHITECTURE.md` only if wording must describe the concrete Phase 3 instantiation
- [x] `docs/ROADMAP.md` only for official phase status/evidence
- [x] `api/openapi.yaml`
- [x] generated Desktop API types/client
- [x] SQLite migration documentation/tests
- [x] Amendment 003A1 embedded official Hermes source for Demo MSI

## 25. Specification Review Record

Review performed on 2026-08-17 against:

- `AGENTS.md`;
- `docs/DEVELOPMENT.md`;
- `docs/ARCHITECTURE.md`;
- `docs/PROTOCOL.md`;
- `docs/RUNTIME.md`;
- `docs/DATA_MODEL.md`;
- `docs/SECURITY.md`;
- `docs/PHASE_GOVERNANCE.md`;
- `docs/AUDIT_STANDARD.md`;
- `docs/ROADMAP.md`;
- `docs/phases/PHASE_TEMPLATE.md`;
- ADR-0001 through ADR-0004;
- the accepted Phase 2 Spec, amendment and audit history;
- official Hermes release, immutable installer source and protocol implementation listed above.

Review result:

```text
Scope:                 PASS
Architecture boundary: PASS
Security/provenance:   PASS WITH DOCUMENTED RESIDUAL RISK
Operation lifecycle:   PASS
Process cleanup:       PASS AS SPECIFIED; IMPLEMENTATION MUST PROVE
API/data contracts:    PASS
Testing/auditability:  PASS
Phase 4 leakage:       NONE FOUND
Owner approval:        PASS — 2026-08-17
Implementation auth:   GRANTED — 2026-08-17
```

## 26. Completion Evidence

R3 remediation evidence. This is not an audit PASS.

```text
Owner approval: 2026-08-17 explicit implementation authorization
R3 audited branch: fix/phase3-audit-r2-remediation
R3 audited commit: d214b51a839b165a62261a4adc4be7c31b486936
R3 exact-commit CI: https://github.com/YoLin02/yorva/actions/runs/32097619823 PASS
R3 MSI CI: https://github.com/YoLin02/yorva/actions/runs/32097619817 PASS
R3 MSI artifact digest: sha256:814edc6516115eb12b5b7dd2a4e0b7523ebd9884410b12b1965f52b1a0ce4ffe
R3 gate: FAIL (see AUDIT-003R3)
R3 remediation branch: fix/phase3-audit-r3-remediation
R3 remediation payload: the implementation commit on that branch after this evidence update
R4 audit candidate HEAD: locked and recorded by independent AUDIT-003R4
R4 exact-commit CI / MSI: locked and recorded by AUDIT-003R4; do not invent run IDs here
Focused tests: previous-proof preserved until atomic replace, two-Operation retry handoff, inventory refresh after mutating stages, fail-closed foreign mutation, aggregate-only archive limit
Windows lifecycle evidence: reused Phase 2 suspended Job Object runner with 1 MiB install output bound
Known environmental blockers: local go test -race remains gcc-blocked; exact-commit CI race is mandatory
Residual risks: official installer has no upstream checksum; YORVA-reviewed digest only
AUDIT-003 status: FAIL
AUDIT-003R1 status: FAIL
AUDIT-003R2 status: FAIL
AUDIT-003R3 status: FAIL
AUDIT-003R4 status: PENDING
```

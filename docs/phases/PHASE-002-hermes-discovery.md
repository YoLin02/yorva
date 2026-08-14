# YORVA Phase 2 — Hermes Discovery & Compatibility

> Status: AUDIT
> Owner: Repository owner
> Previous baseline: `phase-001-bootstrap-baseline`
> Previous baseline commit: `49de99bd3b9c908f5f2065b7c56fd69e97206284`
> Previous gate: `AUDIT-001R2-repository-foundation.md` — PASS
> Planning transition: DRAFT → READY on 2026-08-14
> Implementation status: COMPLETE — automatic Batches 1–7 authorized on 2026-08-14
> Verification: PASS — complete local matrix and exact-commit CI
> Audit: `AUDIT-002` PENDING

## 1. Objective

Give YORVA a read-only, deterministic way to discover an official Hermes Agent executable, obtain and normalize its version, decide whether that version is compatible, and present the result in Desktop.

```text
Open YORVA
→ yorvad discovers Hermes through the Hermes adapter
→ YORVA shows executable path, version and compatibility
→ user can refresh or cancel discovery
```

Negative outcomes are first-class results: not installed, unsupported, broken executable, malformed version and timeout. Phase 2 ends at discovery and compatibility; it never mutates Hermes or the host.

## 2. Entry Criteria

- [x] Phase 1 implementation is complete and independent re-audit R2 returned `PASS`.
- [x] Phase 1 is `COMPLETE / FROZEN`.
- [x] `phase-001-bootstrap-baseline` exists on `main`, resolves to `49de99bd3b9c908f5f2065b7c56fd69e97206284`, and is pushed.
- [x] Phase 2 scope excludes Hermes installation and later roadmap work.
- [x] The repository owner approved the Specification and later authorized continuous implementation through Batches 1–7.

The phase transitioned `READY` → `IN_PROGRESS` only after the Owner's implementation authorization. Automatic batch progression did not waive any test gate or scope boundary.

## 3. In Scope

- detect whether Hermes Agent is installed;
- locate candidate official Hermes executables without modifying `PATH`;
- invoke the official read-only version command;
- parse and normalize the Hermes version;
- classify the selected candidate as supported or unsupported;
- distinguish absent, broken, malformed-version, timed-out and cancelled discovery;
- handle multiple candidates deterministically and report alternatives;
- expose typed discovery data through authenticated local HTTP and OpenAPI;
- render discovery and compatibility states in Desktop;
- add focused Go, protocol and React tests;
- synchronize `RUNTIME.md`, `PROTOCOL.md`, OpenAPI and generated Desktop types during implementation.

## 4. Out of Scope

Hard boundaries:

- installing, downloading, updating, repairing or uninstalling Hermes;
- modifying `PATH`, shell profiles, environment variables or Python environments;
- invoking any installer, package manager, PowerShell script or shell script;
- Hermes Profile or YORVA Instance management;
- Hermes process, gateway or service lifecycle;
- models, API keys, Weixin, WeCom, Skills, MCP, backup/restore or Cloud;
- persistence of a managed Runtime installation;
- arbitrary executable browsing or a generic process execution API;
- a dynamic Runtime plugin system or speculative multi-Runtime framework;
- forking, patching, importing or modifying Hermes internals.

Hermes Installation is Phase 3.

## 5. Architecture Boundary

```text
React Desktop
    ↓ authenticated local HTTP
yorvad HTTP transport
    ↓
Discovery application use case
    ↓
Runtime discovery contract
    ↓
Hermes Adapter
    ↓ direct executable + argv
Official Hermes Runtime
```

React owns presentation, refresh and client cancellation. Transport owns authentication and DTO mapping. The application layer owns the use case and overall deadline. The Runtime contract owns normalized concepts. `services/node/internal/runtime/hermes` owns candidate locations, argv, parsing, compatibility and error normalization. A narrow OS command adapter owns direct execution, bounded output and cancellation.

Forbidden dependencies:

```text
React → Hermes CLI
Tauri → Hermes discovery logic
Core/domain → Hermes command syntax or install paths
Hermes adapter → React or HTTP DTOs
HTTP endpoint → generic shell/process execution
```

No new ADR is expected because this phase instantiates ADR-0003. A new trust boundary, public integration surface or generic plugin mechanism requires an ADR before implementation.

## 6. Hermes Discovery Strategy

Discovery is read-only and Windows-first. Candidate sources, in order:

1. every executable named `hermes` resolved from the `yorvad` process `PATH`, preserving PATH order;
2. Windows: `%LOCALAPPDATA%\hermes\hermes-agent\venv\Scripts\hermes.exe`;
3. Unix per-user: `~/.local/bin/hermes`;
4. Unix system: `/usr/local/bin/hermes`.

Only host-appropriate candidates are used. Enumeration returns absolute regular executable files, canonicalizes where possible, deduplicates Windows paths case-insensitively and Unix paths case-sensitively, preserves source order, and evaluates at most eight candidates. It never scans disks recursively, inspects Python packages, reads Hermes config or probes undocumented locations.

For each candidate the adapter directly invokes, without a shell:

```text
<absolute-hermes-executable> --version
```

No fallback command may initialize configuration or inspect unrelated runtime state. A candidate that cannot satisfy the documented global `--version` contract is broken or malformed, not repaired. Stdout and stderr are separate and limited to 64 KiB each; overflow terminates the child with `RUNTIME_COMMAND_OUTPUT_LIMIT`.

Phase 2 adds no SQLite migration. Discovery is live and non-authoritative after the response; a later phase may persist accepted installation metadata.

Upstream evidence reviewed on 2026-08-14:

- official CLI reference: <https://github.com/NousResearch/hermes-agent/blob/main/website/docs/reference/cli-commands.md>
- official Windows paths: <https://github.com/NousResearch/hermes-agent/blob/main/website/docs/user-guide/windows-native.md>
- official Unix paths: <https://github.com/NousResearch/hermes-agent/blob/main/website/docs/getting-started/installation.md>
- official package release history: <https://pypi.org/project/hermes-agent/>

## 7. Runtime Contract

Phase 2 replaces the Phase 1 discovery stub with one focused capability. Exact Go spelling may follow repository conventions; semantics are fixed:

```go
type Discoverer interface {
    Detect(ctx context.Context) (Discovery, error)
}

type Discovery struct {
    RuntimeKind RuntimeKind
    State       DiscoveryState
    Selected    *Candidate
    Candidates  []Candidate
    Warnings    []Warning
    DetectedAt  time.Time
}

type Candidate struct {
    Path      string
    Version   string
    State     CandidateState
    ErrorCode string
}
```

Public discovery states:

```text
NOT_INSTALLED
SUPPORTED
UNSUPPORTED
BROKEN_EXECUTABLE
MALFORMED_VERSION
TIMED_OUT
AMBIGUOUS
```

Rules:

- `NOT_INSTALLED` means a completed scan found no executable.
- `SUPPORTED` and `UNSUPPORTED` require a normalized version and selected path.
- `BROKEN_EXECUTABLE` means candidates exist but none completes the version command.
- `MALFORMED_VERSION` means a candidate exits successfully without one accepted version.
- `TIMED_OUT` means no decisive result was available before the deadline.
- `AMBIGUOUS` means two or more distinct runnable candidates conflict and YORVA cannot safely select one without a selector UI.
- cancellation returns `context.Canceled`; it is not persisted as discovery state.
- descriptors remain static metadata; later-phase capabilities remain absent.
- the generic contract contains no Hermes paths, output syntax or support range.

## 8. Compatibility Model

Initial tested support window:

```text
>= 0.19.0 and < 0.20.0
```

`0.19.0` is the current official packaged release reviewed for this plan. Hermes remains on a `0.x` line, so YORVA supports only the tested minor line instead of guessing forward compatibility.

Parsing accepts a documented Hermes banner containing exactly one `major.minor.patch` package version, optionally prefixed by `v`, and normalizes it to SemVer. A documented date token may be ignored only after the package version is identified. Missing, partial, ambiguous, overflowed or non-SemVer values are malformed; version must not be inferred from file metadata, paths, Git state or Python internals.

- in-range stable release: `SUPPORTED`;
- valid out-of-range release: `UNSUPPORTED`;
- prerelease without an explicit tested allowlist: `UNSUPPORTED` with warning;
- unparsable output: `MALFORMED_VERSION`, internally `UNKNOWN_VERSION`.

Changing the tested range requires upstream review, fixtures and contract tests. It needs an ADR only if the integration surface or architecture changes.

## 9. Error Model

| Code | Meaning | Retryable |
| --- | --- | --- |
| `RUNTIME_NOT_INSTALLED` | Complete scan found no candidate. | false |
| `RUNTIME_UNSUPPORTED` | Version is outside the tested window. | false |
| `RUNTIME_EXECUTABLE_BROKEN` | Candidate cannot start or exits non-zero. | true |
| `RUNTIME_VERSION_MALFORMED` | Successful output has no accepted unambiguous version. | false |
| `RUNTIME_DISCOVERY_TIMEOUT` | Candidate or overall deadline expired. | true |
| `RUNTIME_DISCOVERY_CANCELLED` | Caller cancelled; application diagnostic only. | true |
| `RUNTIME_COMMAND_OUTPUT_LIMIT` | Version output exceeded its bound. | false |
| `RUNTIME_DISCOVERY_AMBIGUOUS` | Two or more distinct runnable candidates conflict. | false |

Completed negative discovery is typed data, normally HTTP `200`, because absence or incompatibility is a valid query answer. Authentication, origin, routing and unexpected failures retain the standard error envelope. Desktop never branches on message text. User-safe messages exclude raw stderr, stack traces and environment values; internal logs retain a redacted cause and correlation ID.

## 10. Timeout / Cancellation

- per-candidate timeout: 3 seconds;
- overall daemon deadline: 10 seconds;
- Desktop timeout: 12 seconds, leaving the daemon authoritative;
- evaluate sequentially for deterministic ordering and simple ownership;
- overall cancellation stops remaining candidates;
- request cancellation propagates to the child;
- cancelled/timed-out children are waited and reaped;
- no goroutine or process outlives the request;
- retry always uses a new context.

Discovery is bounded synchronous work, not an Operation.

## 11. Multiple Candidate Handling

Evaluate all bounded candidates unless cancelled or timed out. Canonical-path duplicates execute once. If exactly one candidate is runnable, select it even when other candidates are broken, and retain structured warnings for those failures. If two or more distinct candidates are runnable, return `AMBIGUOUS` with no selection and `RUNTIME_DISCOVERY_AMBIGUOUS`; do not silently choose by PATH order. If none is runnable, report the highest-fidelity malformed, broken or timeout result deterministically.

Desktop shows only a safe ambiguity/failure state and candidate count. An executable selector is not added in Phase 2.

## 12. Desktop UX

Use TanStack Query for one discovery status surface:

- loading: “Checking for Hermes…” plus Cancel;
- supported: version, selected path and last checked time;
- not installed: Hermes not found; installation is planned for Phase 3;
- unsupported: detected version and supported range;
- broken/malformed/timeout: safe explanation plus Retry;
- multiple: candidate paths/count plus an ambiguity warning, with no implicit selection.

There is no Install, Download, Repair, PATH, Profile or lifecycle action. Raw output is hidden. Refresh only refetches discovery. Unmount and Cancel abort the request. States use accessible text, keyboard controls and announced loading status.

## 13. API / OpenAPI Changes

Add one authenticated endpoint:

```text
POST /api/v1/runtimes/{runtimeKind}/detect
```

Phase 2 accepts only registered `hermes`. The request exposes no executable path, shell text or arbitrary arguments. Its typed response contains runtime kind, state, selected candidate or null, candidates, warnings, detection time and supported range.

OpenAPI uses closed enums and `additionalProperties: false`. Generated TypeScript remains derived; Go domain types remain hand-written. Existing bearer authentication, origin policy and error envelope apply. No unauthenticated endpoint, SSE event, installation or upgrade endpoint is added.

Implementation synchronizes `RUNTIME.md`, `PROTOCOL.md`, `api/openapi.yaml` and generated types. Update `DATA_MODEL.md` only to clarify non-persistence, and `SECURITY.md` only if the documented command environment/trust boundary changes.

## 14. Security Boundary

- invoke only an enumerated absolute Hermes executable with constant argv `--version`;
- never use a shell, command string, `cmd /c`, PowerShell or user-controlled argv;
- reject directories and non-regular targets;
- use a minimal child environment containing OS execution essentials and explicitly required Hermes location variables; do not inherit provider/API credentials by default;
- bound candidates, output and execution time;
- run as the normal user without elevation;
- do not read Hermes configuration, secrets, sessions or Python internals;
- preserve loopback, bearer authentication, CORS and CSP controls.

Tests must prove argument safety and environment filtering. Any need for shell invocation blocks the phase for security review.

## 15. Logging / Diagnostics

Structured logs may contain correlation ID, runtime kind, candidate source/index/count, result, normalized version, exit code, duration and timeout/cancel flags. Normalize control characters in debug-only local paths. Never log environment values.

Raw stdout/stderr is absent from info logs and Desktop. A bounded redacted preview may exist at debug level only for broken/malformed diagnostics. Logs distinguish every discovery outcome. Usernames in paths must not be promoted to telemetry or remote logs.

## 16. Testing Strategy

| Scenario | Expected result | Level |
| --- | --- | --- |
| no candidate | `NOT_INSTALLED` | adapter integration |
| official Windows location | discovered without PATH mutation | Windows adapter |
| `0.19.0` / `0.19.x` banner | normalized `SUPPORTED` | parser/adapter |
| below range or `0.20.0` | `UNSUPPORTED` | compatibility unit |
| unlisted prerelease | unsupported warning | compatibility unit |
| missing/partial/ambiguous/overflow version | malformed | parser table |
| start failure/non-zero exit | broken | command adapter |
| output over 64 KiB | terminated/output-limit | command adapter |
| per-command/overall timeout | timed out and reaped | adapter/application |
| caller cancellation | cancelled, no process/leak | adapter/application |
| duplicate candidates | canonical path executes once | adapter |
| multiple distinct runnable candidates | `AMBIGUOUS`, no selection | adapter |
| malicious PATH text | no shell interpretation | security |
| inherited fake provider secret | absent from child | security |
| rejected auth/origin | stable `401`/`403` | protocol |
| each response state | OpenAPI-conformant DTO | protocol |
| each UI state and cancel→retry | correct state, no stale overwrite | React |

Required implementation verification:

```text
pnpm api:lint
pnpm api:generate
git diff --exit-code -- apps/desktop/src/api/generated/schema.ts
pnpm typecheck
pnpm lint
pnpm test
pnpm build

cd services/node
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/yorvad
govulncheck ./...

cd apps/desktop/src-tauri
cargo fmt --check
cargo test --locked
cargo clippy --locked --all-targets -- -D warnings
cargo check --locked
cargo audit
```

Also run sidecar build, Windows lifecycle smoke and Tauri no-bundle build. Tests must not download/install Hermes; fake executables stay test-only.

## 17. Exit Criteria

- [x] discovery exists only behind application/Runtime/Hermes adapter boundaries;
- [x] PATH and documented candidates are found without mutation;
- [x] `--version` is direct, bounded, timed and cancellable;
- [x] tested `0.19.x` compatibility is normalized;
- [x] all required negative and multiple-candidate cases, including ambiguity, are deterministic and tested;
- [x] Desktop presents every state without direct Hermes access or message parsing;
- [x] OpenAPI and generated types are synchronized;
- [x] no migration, persistent discovery cache, installation or later-phase code exists;
- [x] relevant local checks and exact-commit CI pass;
- [x] implementation is frozen for independent Phase 2 audit;
- [ ] independent gate permits a Phase 2 baseline before Phase 3 begins.

## 18. Audit Requirements

Use a fresh independent review context and `docs/AUDIT_STANDARD.md`. The audit must:

- prove the implementation descends from `phase-001-bootstrap-baseline`;
- prove scope is discovery/compatibility only;
- trace Desktop → HTTP → application → Runtime contract → Hermes adapter;
- search for forbidden direct calls, shell strings, generic execution and PATH mutation;
- validate Windows precedence, canonicalization, deduplication and candidate bounds;
- validate process timeout, cancellation, reaping, output bounds and environment filtering;
- validate compatibility at `0.19.0`, `0.19.x`, prerelease, below-range and `0.20.0`;
- validate every negative state and stable code across adapter, API and Desktop;
- validate authentication, CORS, safe messages, logs, OpenAPI and generated types;
- verify no persistence makes stale data authoritative;
- independently rerun verification and exact-commit CI;
- issue a new audit report without altering audit history.

Any correctness/security defect in selection, compatibility, process execution, timeout/cancellation, typed API or required UX blocks the gate. Phase 3 implementation cannot begin until Phase 2 is audited and frozen.

# YORVA Repository Bootstrap Specification

> Phase: Phase 1 — Repository foundation / Bootstrap  
> Status: COMPLETE / FROZEN
> Gate: PASS — `AUDIT-001R2-repository-foundation.md`
> Owner: Repository owner  
> Baseline: `phase-001-bootstrap-baseline`
> Audit-accepted implementation commit: `1b759f443dbbebba4ae61a82c91e92180d7527b0`
> Repository: `https://github.com/YoLin02/yorva`
> Go module: `github.com/YoLin02/yorva/services/node`
> Relevant roadmap entry: `docs/ROADMAP.md` — Phase 1  
> Applies to: first repository bootstrap only  
> Product: YORVA  
> Goal: establish a small, runnable, testable architecture baseline before any Hermes business feature is implemented.

## 1. Purpose

This document defines exactly what Codex or another coding agent is allowed to create during the first YORVA repository initialization.

The bootstrap is **not** the first product feature. It is the minimum executable skeleton that proves the architecture described by:

- `AGENTS.md`;
- `docs/DEVELOPMENT.md`;
- `docs/ARCHITECTURE.md`;
- `docs/PROTOCOL.md`;
- `docs/RUNTIME.md`;
- `docs/DATA_MODEL.md`;
- `docs/SECURITY.md`;
- `docs/PHASE_GOVERNANCE.md`;
- `docs/AUDIT_STANDARD.md`;
- `docs/adr/ADR-0001-*.md` through `ADR-0004-*.md`.

If this document conflicts with a higher-level architecture document, stop and resolve the conflict rather than silently inventing a new architecture.

The target result is:

```text
YORVA Desktop
    ↓ authenticated loopback HTTP
    ↓ authenticated SSE
yorvad
    ↓
SQLite
    ↓
Runtime Registry
    ↓
Hermes Adapter skeleton
```

Everything above must build and run before Phase 2 Hermes discovery begins.

---

## 1.1 Implementation entry conditions

A successful Phase 0 Readiness Gate may move this specification to `READY`. Before Phase 1 moves to `IN_PROGRESS` or any repository/code initialization begins, record and verify:

```text
actual Git repository location
matching non-placeholder Go module path
Node.js version
pnpm version
Go 1.26+ toolchain
Rust and Cargo toolchain
Tauri 2 CLI/native build prerequisites
```

An unresolved Git repository/Go module path or an unavailable required toolchain blocks Phase 1 execution. These are implementation entry conditions, not reasons to weaken or reopen the accepted Phase 0 architecture baseline.

---

## 2. Bootstrap success definition

The repository is successfully bootstrapped when a developer can clone it and complete the following flow without any Hermes installation:

```text
install toolchain dependencies
    ↓
install JavaScript dependencies
    ↓
run Desktop in development mode
    ↓
Tauri starts yorvad
    ↓
yorvad opens/migrates SQLite
    ↓
yorvad binds an ephemeral loopback port
    ↓
Tauri completes authenticated bootstrap handshake
    ↓
React loads authenticated Node information
    ↓
React opens authenticated SSE stream
    ↓
Desktop shows "Node connected"
```

The repository must also pass its defined build, type-check, lint and test commands.

No Hermes command needs to execute for bootstrap completion.

---

## 3. Bootstrap scope

The first initialization **must create only** the following platform capabilities:

1. monorepo foundation;
2. Tauri 2 + React + TypeScript Desktop shell;
3. Go `yorvad` executable;
4. authenticated Desktop ↔ Node bootstrap;
5. loopback HTTP API;
6. SSE event endpoint;
7. SQLite connection and migration runner;
8. local Node identity persistence;
9. OpenAPI source-of-truth skeleton;
10. generated or typed Desktop API boundary;
11. Runtime contracts/registry foundation;
12. empty Hermes Adapter package wired into the registry structure;
13. structured error response support;
14. structured logging baseline;
15. unit/integration smoke tests;
16. CI baseline.

This stage establishes infrastructure only.

---

## 4. Explicit non-goals

Codex must **not** implement any of the following during bootstrap:

```text
Hermes detection
Hermes installation
Hermes upgrade
Hermes CLI invocation
Hermes profile discovery
Instance creation/deletion
Model configuration
API key configuration
Weixin QR login
WeCom QR login
Telegram / Feishu / DingTalk
Skills
MCP
Backup / restore
Cloud Control Plane
Node remote pairing
User accounts
Organizations
RBAC
SSO
Billing
Plugin marketplace
Dynamic Runtime plugins
Telemetry
Auto updater
System service / permanent yorvad daemon
Native mobile applications
```

Do not create fake implementations of these features merely to make the UI look complete.

Do not add disabled dashboard sections for future features unless they are necessary to validate navigation structure.

Bootstrap UI should be intentionally minimal.

---

## 5. Technology baseline

### 5.1 Desktop

```text
Tauri 2
React
TypeScript
Vite
pnpm
```

Use the current compatible stable versions at repository initialization and pin them through the lockfile.

Do not install a state library, router, form library or schema library until bootstrap actually uses it.

For the bootstrap screen:

- React local state is sufficient for local UI state;
- TanStack Query may be introduced because Node state is daemon/server state and will remain part of the architecture;
- Zustand is **not** needed during bootstrap;
- React Router is **not** needed until more than one real route exists;
- React Hook Form and Zod are **not** needed until real forms exist.

### 5.2 Native shell

Tauri Rust code remains narrow.

It owns only:

- locating the bundled/development `yorvad` binary;
- starting the daemon;
- performing the bootstrap handoff;
- retaining the child-process lifecycle handle;
- exposing the daemon session descriptor to React;
- minimum native lifecycle integration.

Do not move Runtime management logic into Rust.

### 5.3 Node daemon

```text
Go 1.26+
net/http
log/slog
database/sql
SQLite
```

Prefer standard-library functionality where practical.

Do not add a Go web framework during bootstrap. Go's standard HTTP router is sufficient for the required API surface.

Do not add a dependency-injection framework.

Do not add a heavy ORM.

### 5.4 SQLite driver

Prefer a SQLite driver that does not require CGO for the normal YORVA build, so Desktop/sidecar packaging remains simpler across supported platforms.

Use the selected driver only through `database/sql` or a narrow persistence boundary.

Record the exact driver choice in the bootstrap commit or a small dependency note. A separate ADR is unnecessary unless the choice materially changes architecture.

### 5.5 Migrations

Do not add a migration framework during bootstrap unless it provides a concrete advantage over a small embedded runner.

Preferred initial approach:

```text
Go embed.FS
+
numbered SQL migrations
+
schema_migrations table
+
transaction per migration
```

Migration files are immutable after they have been merged.

---

## 6. Repository layout

Create this initial repository shape:

```text
yorva/
├── AGENTS.md
├── README.md
├── package.json
├── pnpm-workspace.yaml
├── pnpm-lock.yaml
│
├── api/
│   └── openapi.yaml
│
├── apps/
│   └── desktop/
│       ├── package.json
│       ├── index.html
│       ├── src/
│       │   ├── app/
│       │   ├── api/
│       │   ├── components/
│       │   ├── features/
│       │   ├── hooks/
│       │   ├── pages/
│       │   ├── types/
│       │   ├── App.tsx
│       │   └── main.tsx
│       └── src-tauri/
│           ├── Cargo.toml
│           ├── capabilities/
│           ├── src/
│           └── tauri.conf.json
│
├── services/
│   └── node/
│       ├── cmd/
│       │   └── yorvad/
│       ├── internal/
│       │   ├── app/
│       │   ├── domain/
│       │   ├── httpapi/
│       │   ├── logging/
│       │   ├── node/
│       │   ├── operation/
│       │   ├── persistence/
│       │   ├── runtime/
│       │   └── security/
│       ├── migrations/
│       └── go.mod
│
├── runtimes/
│   └── hermes/
│       └── README.md
│
├── docs/
│   ├── DEVELOPMENT.md
│   ├── ARCHITECTURE.md
│   ├── BOOTSTRAP.md
│   ├── PROTOCOL.md
│   ├── RUNTIME.md
│   ├── DATA_MODEL.md
│   ├── SECURITY.md
│   ├── ROADMAP.md
│   └── adr/
│
└── .github/
    └── workflows/
        └── ci.yml
```

This layout is a baseline, not permission to create empty placeholder packages for every hypothetical future module.

Only create a Go package directory when it contains code or tests required by bootstrap.

During Phase 1, the root `runtimes/hermes` directory contains documentation only. The minimal compile-time registration scaffold lives in the clearly Hermes-owned `services/node/internal/runtime/hermes` package defined in section 7. Do not fill either location with speculative APIs.

---

## 7. Go module boundaries

Use one Go module for the Node/runtime-control code during bootstrap unless the real repository topology proves that a second module is necessary.

The Go module path must match the actual Git repository location. Do **not** commit placeholder module paths such as:

```text
example.com/yorva
TODO/yorva
```

If the final Git host/organization is not known when bootstrap begins, resolve it before committing the initial Go module.

Do not split Hermes into a separately versioned Go module in Phase 1.

The initial codebase is too small to justify independent module versioning.

Phase 1 does not add a second Go module or a `go.work` file. The compile-time Hermes Go Skeleton belongs under:

```text
services/node/internal/runtime/hermes
```

The root-level `runtimes/hermes` directory contains only documentation for the future independent Runtime Adapter ownership boundary during Phase 1. This arrangement keeps all Phase 1 Go code inside the single Node module while preserving clear Hermes ownership.

---

## 8. Desktop bootstrap lifecycle

### 8.1 Phase 1 daemon ownership

For Phase 1 repository foundation, `yorvad` is a Tauri-managed sidecar/process for the Desktop session.

This is intentionally simpler than installing a permanent operating-system service.

```text
Tauri starts
   ↓
locate yorvad
   ↓
generate bootstrap secret
   ↓
spawn yorvad
   ↓
authenticated bootstrap
   ↓
React connects
```

A permanent background service is deferred until there is a demonstrated product requirement and must receive its own ADR because it changes lifecycle, upgrade and authentication behavior.

Closing Desktop may stop the Phase 1 `yorvad` process. This does **not** imply that Hermes itself must be terminated.

The Desktop owns the child for the whole session. Startup has a bounded handshake deadline; a write failure, malformed handshake, early exit, or timeout moves the lifecycle to a safe failed state and cleans up any child that was already spawned. A synchronous bootstrap failure must not abort the Tauri application because React must remain able to render the safe failure state.

The native lifecycle is authoritative for that deadline. While `daemon_session` returns `DAEMON_NOT_READY`, React remains in the starting state and continues observing it; React must not impose a shorter independent retry budget. Observation stops only when the native lifecycle returns a ready session or a stable terminal failure such as `DAEMON_STARTUP_FAILED`.

For normal Desktop shutdown, Tauri writes one `{"type":"shutdown"}` control record to the existing child stdin and waits for a bounded graceful exit before using forced termination as a fallback. `yorvad` also treats stdin EOF as parent death, cancels its root work, and shuts down its HTTP server. The bootstrap/control pipe never carries the token after the initial bootstrap record and is not a general command channel.

### 8.2 Listen address

`yorvad` must bind to:

```text
127.0.0.1:0
```

`0` requests an ephemeral OS-selected port.

Do not use a globally fixed management port during bootstrap.

Do not bind IPv4 all-interfaces or public interfaces.

IPv6 loopback support may be added later after testing; it is not required for the first bootstrap.

### 8.3 Bootstrap token

Tauri generates a cryptographically random session token of at least 256 bits.

Requirements:

- generated per daemon session;
- never passed as a command-line argument;
- never written to application config;
- never written to logs;
- never included in URLs;
- held only in process memory for the bootstrap session where practical.

Preferred Phase 1 transport:

```text
Tauri
  └─ spawn yorvad --bootstrap-stdio
        ↓
      write bootstrap JSON containing token through child stdin
        ↓
      yorvad validates bootstrap message
        ↓
      yorvad binds 127.0.0.1:0
        ↓
      yorvad writes one bootstrap handshake JSON record to stdout
```

The handshake may contain:

```json
{
  "protocolVersion": "1",
  "port": 49152,
  "pid": 12345
}
```

The handshake must **not** echo the token.

After bootstrap completes, ordinary daemon logs go to stderr rather than mixing arbitrary log lines into the bootstrap stdout channel.

If the implementation cannot keep stdout clean enough for this contract, use a dedicated inherited pipe instead. Do not fall back to putting the token in argv or a world-readable file.

### 8.4 React session descriptor

Tauri exposes one narrow command to React that returns the current local daemon session descriptor:

```ts
type DaemonSession = {
  baseUrl: string;
  token: string;
  protocolVersion: string;
};
```

React keeps this descriptor in memory.

Do not store the token in:

```text
localStorage
sessionStorage
IndexedDB
URL parameters
React Query persistence
```

A future architecture may keep the token entirely in Rust and proxy requests, but Phase 1 must not add that complexity without a concrete need.

---

## 9. Local HTTP authentication

Use:

```text
Authorization: Bearer <session-token>
```

for every management endpoint except the minimal unauthenticated health endpoint.

Required behavior:

- constant-time comparison where appropriate;
- missing token → `401`;
- invalid token → `401`;
- no distinction that leaks credential information;
- auth failures do not log the supplied token;
- CORS/origin policy remains restrictive;
- all management routes are loopback-only.

The bootstrap token authenticates the current Desktop session to the local daemon. It is not a user account, Cloud token or long-term device identity.

---

## 10. Minimal HTTP API

Bootstrap implements only the minimum real endpoints needed to prove the transport.

Base path:

```text
/api/v1
```

### 10.1 Health

```text
GET /api/v1/health
```

Authentication:

```text
not required
```

The response must disclose only non-sensitive liveness information, for example:

```json
{
  "status": "ok",
  "service": "yorvad",
  "version": "0.0.0-dev",
  "protocolVersion": "1"
}
```

Do not return data directories, environment variables, tokens, full process arguments or machine secrets.

### 10.2 Node

```text
GET /api/v1/node
```

Authentication: required.

Example response:

```json
{
  "id": "node_xxx",
  "name": "DESKTOP-ABC",
  "hostname": "DESKTOP-ABC",
  "platform": "windows",
  "architecture": "amd64",
  "nodeVersion": "0.0.0-dev",
  "createdAt": "...",
  "updatedAt": "..."
}
```

The endpoint must read persisted Node identity rather than generating a new Node ID on every request.

### 10.3 Event stream

```text
GET /api/v1/events
Accept: text/event-stream
Authorization: Bearer <token>
```

Authentication: required.

Bootstrap does not need fake business events.

It must prove that:

- an authenticated stream can connect;
- disconnect cancels server work;
- keepalive comments do not leak data;
- multiple subscribers do not block each other indefinitely;
- no goroutine leaks occur during repeated connect/disconnect tests.

A lightweight keepalive comment may be emitted periodically.

The native browser `EventSource` API cannot attach the required bearer header, so Desktop must use authenticated `fetch()` streaming (or a small reviewed helper) rather than placing the token in the SSE URL.

Do not weaken authentication to make native `EventSource` convenient.

---

## 11. API schema and generation

`api/openapi.yaml` is the source of truth for Desktop ↔ Node HTTP request/response schemas.

During bootstrap it must define at least:

```text
GET /api/v1/health
GET /api/v1/node
GET /api/v1/events
standard error schema
Node schema
Health schema
```

The SSE endpoint may be represented as a `text/event-stream` response even if the generated client does not manage streaming.

Generate or derive TypeScript transport types/client code from OpenAPI where practical.

Rules:

- generated files are clearly marked;
- CI validates OpenAPI;
- generated transport types are not manually edited;
- Go domain entities are not generated from OpenAPI;
- transport DTOs map to application/domain models explicitly where needed.

Do not add a second API schema system such as GraphQL, tRPC or protobuf during bootstrap.

---

## 12. Desktop API layer

React components must not call raw daemon URLs throughout the component tree.

Create one small API boundary under:

```text
apps/desktop/src/api/
```

It owns:

- base URL;
- bearer authentication header;
- JSON request/response handling;
- standard YORVA error decoding;
- SSE authenticated stream creation;
- request cancellation.

The bootstrap screen queries Node state through this API boundary.

Do not implement retry policies that can accidentally repeat mutations. Bootstrap only contains safe GET requests.

---

## 13. Minimal Desktop UI

Bootstrap UI is a system-health screen, not a mock product dashboard.

It should show only useful foundation information such as:

```text
YORVA

Local Node       Connected
Node ID          node_...
Platform         Windows / macOS / Linux
Architecture     ...
yorvad version   ...
Protocol         v1
Event stream     Connected
```

Required states:

```text
Starting daemon
Connecting
Connected
Connection failed
```

If daemon bootstrap fails, show a concise user-safe error and make diagnostic details available through development logs.

Do not add fake Agent cards, fake Hermes installations or non-working navigation.

---

## 14. Tauri permissions and CSP

Use the minimum Tauri permissions required for bootstrap.

Do not enable broad shell capabilities for React.

React must not be allowed to execute arbitrary local commands.

Production CSP should remain restrictive. Permit only the local origins/protocols actually needed by Tauri and authenticated communication with the ephemeral loopback daemon.

A typical requirement is narrowly allowing loopback connections such as:

```text
http://127.0.0.1:*
```

Do not solve development problems by disabling CSP globally.

Development-only Vite/HMR allowances must not silently become production policy.

---

## 15. SQLite location and startup

Tauri provides or resolves the application data directory and passes it to `yorvad` as a non-secret startup parameter.

Example conceptual argument:

```text
--data-dir <YORVA application data directory>
```

The directory is not secret and may be passed through argv.

The database lives under that directory, for example:

```text
<app-data>/yorva.db
```

`yorvad` owns SQLite access.

React and Tauri Rust must never open the SQLite database directly.

At startup:

```text
resolve/create data directory
    ↓
open SQLite
    ↓
enable foreign_keys
    ↓
configure safe busy timeout
    ↓
run pending migrations
    ↓
load/create local Node identity
    ↓
start HTTP server
```

If migration fails, daemon startup fails safely and reports a diagnostic error. Do not continue with a partially migrated schema.

Connection-local SQLite settings (`foreign_keys` and `busy_timeout`) are encoded in the driver DSN so the driver applies them to every physical connection, including a replacement connection created after the pool reconnects.

---

## 16. Initial migrations

The migration system must be able to migrate an empty database deterministically.

The initial schema should align with `DATA_MODEL.md`.

At minimum Phase 1 must create the tables needed by its running code:

```text
schema_migrations
nodes
app_settings
```

Because the remaining V0.1 schema has already been designed, it is acceptable to create the other stable management tables in the same initial migration **only if** their columns and constraints match `DATA_MODEL.md` and no speculative fields are introduced:

```text
runtime_installations
instances
operations
operation_events
channel_bindings
secret_refs
backups
audit_log
```

Do not create Hermes session, memory, message or profile-shadow tables.

Every table must have only the indexes/constraints justified by `DATA_MODEL.md` or an actual query requirement.

---

## 17. Local Node identity

On first successful database initialization, create one local Node identity.

Requirements:

- opaque random YORVA ID;
- persisted across Desktop restarts;
- hostname/platform/architecture may be refreshed;
- Node ID must not be derived from MAC address, motherboard serial, username or another privacy-sensitive hardware fingerprint;
- no Cloud identity exists in Phase 1.

Use cryptographically secure randomness for generated identifiers.

Readable prefixes such as `node_` are allowed but not semantically significant.

---

## 18. Runtime Registry bootstrap

Create the smallest compile-time registry required by `RUNTIME.md`.

The registry must support the future idea that multiple Runtime kinds may be registered, but Phase 1 contains only Hermes scaffolding.

Do not create:

```text
dynamic plugin loading
shared-library discovery
plugin manifests
plugin marketplace
RPC plugin processes
sandboxing
version negotiation between plugin binaries
```

A conceptual registry may expose operations such as:

```go
Register(kind RuntimeKind, bundle RuntimeBundle) error
Get(kind RuntimeKind) (RuntimeBundle, bool)
Kinds() []RuntimeKind
```

Use the exact contract shape that best fits `RUNTIME.md`; do not copy this example mechanically if the document already defines a better boundary.

---

## 19. Hermes Adapter skeleton

Create only enough Hermes-owned code to establish package ownership and registry wiring.

Allowed:

```text
Hermes Runtime kind constant/descriptor
compile-time registration
empty capability wiring where absence is represented honestly
Detect compile-time stub returning RUNTIME_DISCOVERY_NOT_AVAILABLE, if required by the interface
package tests for registration
README describing future ownership
```

`RUNTIME_DISCOVERY_NOT_AVAILABLE` means no discovery was attempted. It is not equivalent to "Hermes is not installed".

Forbidden during bootstrap:

```text
exec.Command("hermes", ...)
PATH discovery
Python discovery
Hermes config parsing
Hermes version parsing
profile scanning
installation logic
channel logic
returning a fabricated not-installed Detection result
```

Phase 2 begins with Hermes detection as the first real adapter behavior.

Do not return fabricated "Hermes installed" or fake capability results merely to populate Desktop.

---

## 20. Error model bootstrap

Implement the standard transport error envelope from `PROTOCOL.md`:

```json
{
  "error": {
    "code": "...",
    "message": "...",
    "retryable": false,
    "details": {}
  }
}
```

Bootstrap must define only errors it actually needs, for example:

```text
UNAUTHORIZED
INTERNAL_ERROR
DAEMON_NOT_READY
DATABASE_ERROR
PROTOCOL_VERSION_MISMATCH
RUNTIME_DISCOVERY_NOT_AVAILABLE
```

Do not predefine dozens of unused Runtime errors solely for completeness.

Internal errors may be wrapped with diagnostic context in Go logs but the API returns a user-safe message.

---

## 21. Logging baseline

Use Go `log/slog` unless a measured requirement demonstrates that another logger is needed.

Requirements:

- structured fields;
- service/version fields where useful;
- no bootstrap token;
- no environment dumps;
- no secret values;
- avoid logging full HTTP Authorization headers;
- errors retain enough internal context for debugging.

During bootstrap, stderr logging is sufficient.

Persistent log files and diagnostic export bundles belong to later phases.

---

## 22. Event broker baseline

The SSE event infrastructure may use an in-process broker.

Requirements:

- explicit subscriber lifecycle;
- bounded buffers;
- slow subscribers cannot block the entire daemon indefinitely;
- subscriber cleanup on context cancellation;
- no global goroutine leak;
- event payloads are typed at the application/transport boundary;
- events are notifications, never the sole source of truth.

Bootstrap does not need durable event replay.

`Last-Event-ID` persistence/replay may be added when a real Operation workflow requires it.

---

## 23. Operation engine bootstrap

`ROADMAP.md` requires the Operation model for future long-running work, but bootstrap does not need to implement fake long-running actions.

Create only the minimum durable domain/persistence foundation if required by the initial migration and architecture.

Do not add a worker pool, scheduler or generalized job framework until an authorized installation phase demonstrates the need. Phase 2 discovery does not need one.

When the first real Operation is implemented, it must follow `PROTOCOL.md` and `ARCHITECTURE.md`.

---

## 24. SecretStore bootstrap

Do not implement model/API/channel secret storage before Phase 4 unless a secret is actually required.

The Desktop ↔ daemon session token is ephemeral bootstrap state and is not stored through the future Runtime `SecretStore`.

It is acceptable for bootstrap to define a small `SecretStore` interface only if existing architecture code genuinely needs the compile-time boundary. Do not add OS keyring dependencies until the first persistent secret use case is implemented and tested.

This avoids installing security-sensitive dependencies that Phase 1 does not use.

---

## 25. Testing requirements

Bootstrap must include tests for the architecture that now exists.

### 25.1 Go

At minimum:

```text
migration from empty database
migration idempotent second startup
local Node identity persists across restarts
authenticated /node succeeds
missing/invalid bearer token returns 401
/health does not expose sensitive fields
SSE authenticated connection succeeds
SSE cancellation releases subscriber
Runtime registry registers Hermes scaffold without real Hermes calls
```

Use temporary directories/databases in tests.

Do not require Hermes to be installed for Phase 1 tests.

### 25.2 React

At minimum test:

```text
starting state
connected Node state
connection failure state
safe rendering of Node metadata
```

Mock the YORVA API boundary, not Hermes.

Use a lightweight test runner compatible with the Vite/React stack.

### 25.3 Tauri/Rust

Test pure bootstrap parsing/session logic where practical.

Do not force brittle GUI automation into Phase 1 solely to increase test count.

### 25.4 Integration smoke test

Provide at least one test or scripted CI path that starts `yorvad` on an ephemeral port with a test bootstrap token and verifies:

```text
health
bearer auth
node response
SSE connect/disconnect
```

---

## 26. Code quality commands

The repository must document the exact commands used by CI.

Expected categories:

### Desktop

```text
pnpm install --frozen-lockfile
pnpm typecheck
pnpm lint
pnpm test
pnpm build
```

### Go

```text
go test ./...
go vet ./...
```

Run race-enabled tests in CI where the selected SQLite/test configuration supports them reliably, especially for the SSE broker and daemon concurrency code.

### Rust/Tauri

```text
cargo fmt --check
cargo clippy --all-targets -- -D warnings
cargo check
```

The exact root scripts may wrap these commands, but they must remain transparent and easy to reproduce locally.

---

## 27. CI baseline

Create one small CI workflow.

The initial workflow should validate:

```text
OpenAPI syntax/contract validation
TypeScript typecheck/lint/test/build
Go test/vet
Rust fmt/clippy/check
```

Do not build release installers or publish artifacts during bootstrap CI.

Do not add deployment workflows.

Do not add a complex multi-stage release pipeline.

Release signing and packaging belong to local product hardening.

Cache dependencies only where the CI implementation stays understandable.

---

## 28. README bootstrap content

Root `README.md` should remain short and factual.

It should contain:

- what YORVA is;
- current status: early development / architecture foundation;
- Hermes is the first Runtime, not the only future Runtime;
- current supported developer flow;
- links to architecture documents;
- basic development prerequisites;
- build/test commands.

Do not advertise unfinished features as available.

Do not claim mobile, Cloud, multi-node or channel support until implemented.

---

## 29. License rule

Do not choose or generate a project license automatically during bootstrap unless the repository owner has already made that decision.

YORVA may later use different openness boundaries for:

```text
Node
Runtime SDK/adapters
Desktop
Control Plane
```

License selection is a product/legal decision, not a code-generator default.

If no license decision exists, keep the repository private during bootstrap or explicitly document that licensing is pending.

---

## 30. Naming conventions

Brand:

```text
YORVA
```

Desktop product name:

```text
YORVA
or
YORVA Desktop
```

Local daemon binary:

```text
yorvad
```

Generic API/domain terminology:

```text
Node
Runtime
Runtime Installation
Instance
Channel
Operation
```

Do not expose Hermes-specific terms such as `profile` as generic YORVA resource names.

For Hermes-specific explanatory UI later, it is acceptable to say:

```text
YORVA Instance (Hermes Profile)
```

---

## 31. Application identifier

Do not permanently lock a public signing/package identifier until the final YORVA domain/organization identity is owned.

A temporary development identifier may be used during repository initialization, but it must be clearly marked for review before the first signed public release.

Changing a production app identifier later can affect update/signing identity, so this is a release prerequisite, not an arbitrary placeholder to forget.

---

## 32. No hidden framework generation

When using project generators such as Vite or Tauri templates:

- inspect generated files;
- remove unused demo assets/components;
- do not retain template counters or example pages;
- do not accept broad permissions without review;
- do not keep generated dependencies that YORVA does not use;
- keep configuration explicit.

Generated starter code is not architecture.

---

## 33. Bootstrap implementation order

Codex must implement the bootstrap in this order unless a concrete dependency requires a small adjustment.

### Step 1 — Repository skeleton

Create:

```text
root workspace
Desktop package
Go module
Tauri shell
API directory
CI skeleton
```

Do not add feature code.

### Step 2 — Go daemon core

Implement:

```text
yorvad entrypoint
bootstrap stdin reader
session auth token
127.0.0.1:0 listener
graceful context ownership
/health
structured logging
```

Verify with Go tests before integrating Desktop.

### Step 3 — SQLite

Implement:

```text
data directory handling
SQLite open/config
migration runner
initial migration
local Node identity
/node use case
/node HTTP endpoint
```

Verify persistence across daemon restart.

### Step 4 — SSE

Implement:

```text
in-process event broker
/events endpoint
bearer authentication
subscriber cleanup
keepalive
```

Do not add fake feature events.

### Step 5 — OpenAPI

Define the real Phase 1 HTTP contract.

Validate it and generate/derive Desktop transport types/client.

Do not generate Go domain models from the transport contract.

### Step 6 — Tauri daemon bootstrap

Implement:

```text
resolve yorvad binary
create session token
spawn daemon
send bootstrap token through stdin
read handshake
retain process/session state
expose daemon session to React
```

No Hermes process execution is permitted.

### Step 7 — React API/client

Implement:

```text
daemon session acquisition
HTTP client
bearer header
Node query
authenticated SSE fetch stream
connection state
```

### Step 8 — Minimal UI

Render only real bootstrap state.

### Step 9 — Runtime registry

Add the Runtime registry and Hermes-owned compile-time scaffold.

No Hermes detection yet.

### Step 10 — CI and repository cleanup

Run all checks, remove unused template artifacts, update README and confirm no unrelated dependencies or placeholders remain.

---

## 34. Bootstrap completion checklist

Before Codex reports completion, every item below must be verified.

### Repository

- [ ] Repository matches the documented ownership boundaries.
- [ ] `AGENTS.md` is at repository root.
- [ ] Architecture documents are present under `docs/`.
- [ ] No placeholder Git module path remains.
- [ ] No accidental license was added.

### Desktop

- [ ] Tauri + React application starts.
- [ ] No template/demo UI remains.
- [ ] React does not call Hermes or the shell.
- [ ] Session token is held only in memory.
- [ ] Node information loads through the API layer.
- [ ] SSE connects using an authenticated fetch path.
- [ ] Error state is visible and safe.
- [ ] CSP is not disabled.

### yorvad

- [ ] Binds only to `127.0.0.1`.
- [ ] Uses an ephemeral port.
- [ ] Bootstrap token is not passed through argv.
- [ ] Bootstrap token is not logged.
- [ ] Management endpoints require bearer authentication.
- [ ] `/health` exposes no sensitive environment information.
- [ ] Graceful cancellation/cleanup exists for long-lived goroutines.

### Persistence

- [ ] SQLite is opened only by `yorvad`.
- [ ] Foreign keys are enabled.
- [ ] Empty DB migrates successfully.
- [ ] Reopening an already migrated DB succeeds.
- [ ] Local Node ID persists across restarts.
- [ ] No Hermes-owned data is shadowed.
- [ ] No plaintext secrets exist in SQLite.

### Runtime architecture

- [ ] Runtime registry compiles.
- [ ] Hermes has an owned adapter/scaffold location.
- [ ] No Hermes CLI or config integration exists yet.
- [ ] No dynamic plugin framework exists.
- [ ] Core domain does not depend on Hermes packages.

### Protocol

- [ ] `/api/v1/health` works.
- [ ] `/api/v1/node` requires auth and works.
- [ ] `/api/v1/events` requires auth and streams.
- [ ] Standard error envelope is implemented.
- [ ] OpenAPI describes the implemented HTTP contract.
- [ ] Desktop types/client are synchronized with OpenAPI.

### Quality

- [ ] TypeScript typecheck passes.
- [ ] Frontend lint passes.
- [ ] Frontend tests pass.
- [ ] Frontend production build passes.
- [ ] Go tests pass.
- [ ] `go vet` passes.
- [ ] Rust format check passes.
- [ ] Rust Clippy passes with warnings denied.
- [ ] Rust/Tauri check passes.
- [ ] CI runs the same core checks.
- [ ] No unrelated refactors or speculative frameworks were introduced.

---

## 35. Required final report from Codex

When bootstrap work is finished, Codex must report:

1. files/modules created;
2. architecture boundaries implemented;
3. daemon bootstrap mechanism used;
4. local API endpoints implemented;
5. SQLite migrations created;
6. test coverage added;
7. exact validation commands run and their results;
8. any check not run and the reason;
9. any deviation from `BOOTSTRAP.md`;
10. remaining work for Phase 2.

Do not report Phase 2 features as complete.

The expected next feature after a successful bootstrap is:

```text
Hermes Detection
```

---

## 36. Phase boundary

After this bootstrap passes its completion checklist, enter the Phase 1 audit defined by `AUDIT_STANDARD.md`. Create the Phase 1 baseline commit/tag only after the gate passes.

Suggested milestone meaning:

```text
YORVA repository foundation is runnable.
No real Hermes management feature exists yet.
```

Only after the Phase 1 Gate Decision is `PASS` or an explicitly accepted `PASS WITH CONDITIONS`, and the baseline is frozen, may detailed Roadmap Phase 2 implementation proceed.

The first Phase 2 implementation must start from the Runtime adapter boundary rather than adding Hermes logic directly to Desktop, Tauri or generic Core packages.

### 36.1 Phase 1 freeze record

`AUDIT-001R2-repository-foundation.md` returned `PASS` for the final Phase 1 implementation. The earlier merge of the implementation tree into `main` is retained as implementation history only; it did not grant acceptance or freeze the phase. The formal Phase 1 acceptance point is the freeze commit on `main` referenced by the annotated tag `phase-001-bootstrap-baseline`.

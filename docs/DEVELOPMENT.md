# YORVA Development Guide

> Status: Phase 1, Phase 2 and Phase 3 frozen; Phase 4 unfrozen for delete-timeout remediations; current Phase 3 baseline `phase-003-hermes-installation-baseline`
> Product: YORVA  
> Primary Runtime: Hermes Agent  
> Primary principle: **local-first, lightweight-first, single-binary-first, reversible decisions**

## 1. Product definition

YORVA is a local-first AI Runtime deployment and control platform.

The first supported Runtime is Hermes Agent. YORVA must make the common Hermes workflow available without requiring users to understand CLI commands, configuration file locations, gateway startup details, or profile directory structure.

The long-term product boundary is broader than Hermes:

```text
YORVA
├── Desktop control surface
├── Local Node daemon (yorvad)
├── Runtime adapters
│   ├── Hermes
│   └── future runtimes
└── optional future Control Plane
```

YORVA is **not** a fork of Hermes and is **not** designed as a Hermes-only GUI.

## 2. Phase 0 goals

Before feature development begins, the project must have stable answers for:

- module ownership;
- Desktop ↔ Node protocol;
- Runtime adapter boundary;
- local data ownership;
- secrets and privilege boundaries;
- long-running operation model;
- future Cloud connection direction;
- coding and agent-development rules.

The purpose of Phase 0 is to reduce architectural churn after Codex starts modifying the repository.

## 3. Engineering principles

### 3.1 Local first

Core local functionality must work without:

- a YORVA account;
- YORVA Cloud;
- a public IP;
- an inbound management port;
- a remote database.

A user should be able to install YORVA, install or detect Hermes, create a Hermes-backed instance, configure a model, connect a supported channel, and run it entirely on the local machine.

### 3.2 Lightweight first

When two approaches satisfy the same current requirement with comparable reliability and security, choose the smaller operational footprint.

Default preferences:

- one daemon before multiple services;
- one repository before multiple repositories;
- SQLite before a local database server;
- HTTP/SSE before introducing a message broker;
- Go standard library or small focused dependencies before large frameworks;
- a compiled-in Runtime adapter before a dynamic plugin platform;
- explicit SQL before a heavy ORM.

Do not introduce Redis, Kafka, Kubernetes, service mesh, plugin sandboxes, microservices, or distributed coordination without a measured requirement.

### 3.3 Runtime isolation

YORVA owns the management layer. Each Runtime owns its own runtime state.

```text
YORVA Core
   ↓
Runtime Adapter
   ↓
Official Runtime surface
```

For Hermes, integration priority is:

1. official public API / documented management API;
2. official programmatic protocols such as the TUI gateway JSON-RPC/WebSocket surface;
3. official CLI;
4. a narrowly scoped compatibility adapter for configuration only when no structured public surface exists.

Do not import Hermes internal Python modules, patch Hermes internals, or rely on undocumented database schemas.

### 3.4 Small reversible decisions

Do not generalize a concept only because a future Runtime might need it.

Generalize when:

- the concept is already stable in YORVA; or
- a second real Runtime demonstrates the common boundary.

Until then, keep Runtime-specific behavior inside the adapter.

## 4. Technology baseline

Architecture documents use major-version baselines; exact patch versions are pinned when the repository is initialized and updated through normal dependency management.

### Desktop

```text
Tauri 2
React 19
TypeScript
Vite
React Router
TanStack Query
Zustand only for truly shared client state
React Hook Form + Zod where form complexity justifies them
```

Rules:

- React owns presentation and interaction.
- TanStack Query owns server/daemon state caching and invalidation.
- Zustand is not a replacement for all local state.
- Tauri Rust code remains a narrow native shell and security/native integration layer.
- Desktop must never call Hermes CLI or write Hermes configuration directly.

Suggested structure:

```text
apps/desktop/src/
├── app/
├── api/
├── components/
├── features/
├── hooks/
├── pages/
├── router/
├── stores/
└── types/
```

### Local Node

Executable/service name:

```text
yorvad
```

Technology:

```text
Go 1.26+
SQLite
HTTP/JSON
SSE for local event streaming
structured logging
```

Responsibilities:

- node identity;
- Runtime detection;
- Runtime installation orchestration;
- instance discovery and management;
- lifecycle operations;
- configuration orchestration;
- channel setup orchestration;
- operation state;
- logging and health;
- backup metadata;
- future outbound Cloud connection.

### Hermes adapter

Implementation language: Go.

The adapter translates YORVA use cases into supported Hermes operations. Hermes profiles are represented as YORVA instances, but Hermes-specific concepts must not leak into the YORVA core API unless they are exposed as Runtime-specific metadata.

### Local persistence

```text
SQLite
```

SQLite stores YORVA management state and metadata, not a duplicate copy of Hermes authoritative state.

### Future Control Plane

The future Control Plane remains optional and is not part of V0.1.

Technology baseline:

```text
Go
PostgreSQL
HTTPS/JSON
WebSocket for Node connections
OpenAPI for public HTTP contracts
single deployable service first
```

Rules:

- keep the Control Plane as a modular monolith first;
- do not require Redis or a message broker initially;
- Node initiates the outbound secure connection;
- no public Node management port is required;
- add infrastructure only when measured scale or availability requirements justify it.

## 5. Core domain vocabulary

### Node

A machine running `yorvad`.

### Runtime installation

A detected or managed installation of an AI Runtime on a Node, for example Hermes.

### Instance

A YORVA-managed logical AI Runtime instance.

For Hermes V0.1:

```text
YORVA Instance ≈ Hermes Profile
```

This mapping is adapter-specific. YORVA Core must not assume all future Runtimes use profiles.

### Channel binding

Connection metadata between an Instance and a messaging platform such as Weixin or WeCom.

Secrets are not stored in ordinary channel metadata.

### Operation

A durable representation of a potentially long-running management action such as installation, upgrade, backup, channel authentication, or restart.

## 6. Process ownership

Desktop is not the Runtime process owner.

```text
React Desktop
     ↓
local Node API
     ↓
yorvad
     ↓
Hermes Adapter
     ↓
Hermes
```

Closing Desktop must not automatically stop managed Runtime processes.

The adapter determines the correct Runtime-specific process model. Core code must not assume every Instance maps to one OS PID.

## 7. Long-running operations

The following actions must not depend on a long-held HTTP request:

- Runtime installation;
- Runtime upgrade;
- backup/restore;
- channel QR/login flow;
- long restart/recovery actions.

Use an Operation model:

```text
PENDING
RUNNING
SUCCEEDED
FAILED
CANCELLED
```

HTTP creates the operation and returns an `operationId`. Progress is observed through resource reads and SSE events.

## 8. Secrets

Secrets include:

- model API keys;
- OAuth tokens;
- channel credentials;
- YORVA device credentials;
- future Cloud session credentials.

Rules:

- never log secrets;
- ordinary GET APIs never return secret plaintext;
- UI displays only presence/status unless explicit replacement is requested;
- use OS secure storage behind a `SecretStore` abstraction;
- never silently fall back to plaintext files;
- YORVA Cloud does not receive local Runtime secrets by default.

## 9. Repository layout

```text
yorva/
├── AGENTS.md
├── README.md
├── apps/
│   └── desktop/
│       ├── src/
│       └── src-tauri/
├── services/
│   └── node/
│       ├── cmd/yorvad/
│       └── internal/
├── runtimes/
│   └── hermes/
├── api/
│   └── openapi.yaml
├── docs/
│   ├── DEVELOPMENT.md
│   ├── ARCHITECTURE.md
│   ├── PROTOCOL.md
│   ├── RUNTIME.md
│   ├── DATA_MODEL.md
│   ├── SECURITY.md
│   ├── ROADMAP.md
│   └── adr/
├── packages/
│   └── protocol-types/     # generated/client-facing TS types if required
└── scripts/
```

Do not create packages named `common`, `utils`, or `helpers` as dumping grounds.

## 10. Dependency direction

Allowed:

```text
Desktop → Node HTTP contract
Node transport → Application → Domain
Runtime Adapter → Domain contracts
Persistence Adapter → Application/Domain contracts
```

Forbidden:

```text
Desktop → Hermes
Domain → Hermes implementation
Domain → React/Tauri
Hermes adapter → Desktop
```

## 11. V0.1 scope

### Required

- Windows-first Desktop packaging;
- `yorvad` lifecycle;
- local authenticated Node API;
- Hermes detection;
- Hermes installation workflow;
- Hermes version/capability inspection;
- instance/profile list/create/delete;
- Runtime/instance start-stop-restart where supported;
- model/provider configuration workflow;
- Weixin QR connection workflow;
- WeCom connection workflow;
- operation status and structured logs;
- safe secret handling.

### Follow immediately after the core flow

- Skills management;
- MCP management;
- backup/restore;
- Runtime update workflow;
- additional Hermes channels.

### Explicitly out of V0.1

- native Android/iOS apps;
- public SaaS Control Plane;
- enterprise RBAC;
- SSO;
- billing;
- template marketplace;
- dynamic Runtime plugin loading;
- a second Runtime;
- microservices;
- distributed scheduling infrastructure.

## 12. V0.1 product success criterion

A non-technical user should be able to complete:

```text
Install YORVA
→ detect or install Hermes
→ create an AI instance
→ configure model credentials
→ connect Weixin/WeCom
→ start the instance
→ verify healthy status
```

without opening a terminal for the normal path.

## 13. Quality gates

Before a feature is complete:

- relevant Go tests pass;
- TypeScript typecheck passes;
- lint passes;
- security-sensitive logs are reviewed for redaction;
- API contract changes are reflected in `PROTOCOL.md` / OpenAPI;
- schema changes include migrations;
- Runtime contract changes update `RUNTIME.md`;
- architectural changes require an ADR.

## 14. Local verification workflow

The repository and Go module are fixed as:

```text
Repository: https://github.com/YoLin02/yorva
Go module:  github.com/YoLin02/yorva/services/node
```

Install JavaScript dependencies from the repository root:

```text
pnpm install --frozen-lockfile
```

Validate the OpenAPI contract and generated Desktop types:

```text
pnpm api:lint
pnpm api:generate
git diff --exit-code -- apps/desktop/src/api/generated/schema.ts
```

Run Desktop checks:

```text
pnpm typecheck
pnpm lint
pnpm test
pnpm build
```

Run Node checks from `services/node`:

```text
go test ./...
go vet ./...
go build ./cmd/yorvad
go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...
```

Run native shell checks from `apps/desktop/src-tauri`:

```text
cargo fmt --check
cargo test --locked
cargo clippy --locked --all-targets -- -D warnings
cargo check --locked
cargo audit
```

The sidecar build is target-aware and writes only ignored build output:

```text
pnpm build:sidecar
pwsh -NoProfile -File scripts/windows-lifecycle-smoke.ps1
pnpm --filter @yorva/desktop tauri build --no-bundle
```

Owner test distribution on Windows may also build a user-scope MSI (requires WiX 3 on `PATH` / `WIX`). Demo MSI builds must use the fail-closed packaging entry point, which requires the pinned Hermes source, Node zip, npm tarball, and license files:

```text
pwsh -NoProfile -File scripts/package-yorva-msi.ps1
```

Ordinary `pnpm test` / `tauri build --no-bundle` must not download the large archives. Those payloads are gitignored build inputs whose sizes and SHA-256 values are compiled into the Hermes adapter. A missing, wrong-sized, or wrong-hashed payload fails packaging.

Hermes install Operations write redacted JSON lines to `%APPDATA%\com.yorva.desktop.dev\logs\install.ndjson`. Use `correlationId`, `stage` and `errorCode` to locate a failed test. Do not treat this file as an upstream installer transcript.


From the repository root, `pnpm audit --audit-level low` is also a CI gate. CI Actions are pinned to exact commit SHAs with the corresponding major/stable label in a comment. Dependency maintenance updates those pins deliberately: resolve the trusted upstream major tag/branch to a reviewed commit, inspect upstream release notes, replace the SHA, and rerun the full workflow. Do not restore floating action references.

Phase 2 deterministic tests do not require Hermes to be installed and never install it. Adapter tests build test-only fake executables and fixed official-layout fixtures to verify installation evidence, closed direct argv execution, bounded package-entry-point validation, output bounds, timeout, cancellation and process cleanup. On a Windows host that already has Hermes, run the explicitly gated read-only smoke with `YORVA_REAL_HERMES_SMOKE=1 go test ./internal/runtime/hermes -run TestRealWindowsHermesInstallationSmoke -v`; it must never execute `hermes-agent.exe` or the repository wrapper. A valid official package entry point without a generated launcher must still resolve through the installation's isolated Python command. Stable `0.19.x` and `0.20.x`, including the current official `0.20.1`, classify as `SUPPORTED`; `0.21.0` remains unsupported pending review. Desktop tests cover sidebar navigation, English/Simplified Chinese persistence, stable Runtime-state translations and locale/time-zone-explicit timestamp formatting.

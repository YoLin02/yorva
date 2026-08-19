# YORVA Architecture

## 1. Architecture objective

YORVA must keep three concerns separate:

1. **Presentation** — what the user sees and controls.
2. **Management** — orchestration, policy, operations, persistence and security.
3. **Runtime integration** — translating YORVA use cases to a concrete Runtime such as Hermes.

## 2. V0.1 topology

```text
┌──────────────────────────────────────┐
│ YORVA Desktop                        │
│ Tauri 2 + React + TypeScript         │
└──────────────────┬───────────────────┘
                   │ localhost HTTP/JSON
                   │ + SSE
                   ▼
┌──────────────────────────────────────┐
│ yorvad                               │
│ Go                                   │
│                                      │
│ Transport                            │
│ Application                          │
│ Domain                               │
│ Persistence                          │
│ Security                             │
└──────────────────┬───────────────────┘
                   │ Runtime contracts
                   ▼
┌──────────────────────────────────────┐
│ Hermes Adapter                       │
│ Go                                   │
└───────────────┬──────────────┬───────┘
                │              │
       structured APIs/RPC     │ official CLI / narrow fallback
                │              │
                ▼              ▼
┌──────────────────────────────────────┐
│ Hermes Agent                         │
│ profiles / gateways / sessions / ...│
└──────────────────────────────────────┘
```

## 3. Desktop

Desktop owns:

- navigation;
- presentation;
- forms;
- local interaction state;
- daemon state queries;
- operation progress presentation.

Desktop does not own:

- Hermes discovery;
- shell command construction;
- Runtime lifecycle;
- secret persistence;
- SQLite access;
- compatibility logic.

### State model

Use:

- TanStack Query for Node/API state;
- React local state for page-local interaction;
- Zustand only for cross-page client state that is not server state.

Do not mirror every server resource into Zustand.

## 4. Tauri native shell

Rust is intentionally narrow.

Appropriate responsibilities:

- application startup/shutdown integration;
- locating or starting `yorvad`;
- OS-specific secure/native integration where needed;
- secure bootstrap handoff between Desktop and daemon;
- packaging/updater integration where Tauri owns the concern.

Ordinary Runtime business logic belongs in Go, not Rust.

## 5. yorvad internals

Suggested package boundaries:

```text
services/node/
├── cmd/yorvad/
└── internal/
    ├── app/            # use cases
    ├── domain/         # stable YORVA concepts
    ├── httpapi/        # local HTTP/SSE transport
    ├── operation/      # async operation engine
    ├── persistence/    # SQLite repositories/migrations
    ├── runtime/        # runtime registry/contracts
    ├── secrets/        # SecretStore for YORVA-owned secrets
    ├── node/           # local node identity/health
    └── logging/
```

Package names can evolve, but ownership must remain clear.

## 6. Application layer

Application use cases are the only supported entry point for transport handlers and future remote commands.

Examples:

```text
DetectRuntime
InstallRuntime
ListInstances
CreateInstance
DeleteInstance
StartInstance
StopInstance
UpdateInstanceConfig
BeginChannelConnection
CreateBackup
RestoreBackup
```

A remote Control Plane command must call the same use case as a local Desktop request. It must never bypass the application layer.

## 7. Domain layer

Core domain concepts should be small and Runtime-neutral:

- Node;
- RuntimeKind;
- RuntimeInstallation;
- Instance;
- Capability;
- ChannelBinding;
- Operation;
- Backup;
- stable error codes.

Hermes-only fields belong in adapter metadata, not in core entities.

## 8. Runtime adapter boundary

The Runtime adapter is an anti-corruption layer.

It absorbs:

- command syntax;
- version differences;
- Hermes profile semantics;
- Hermes-specific status values;
- configuration format differences;
- API/CLI fallback behavior;
- channel-specific implementation details.

The adapter returns normalized YORVA results and stable error codes.

Do not expose raw CLI stdout as an application contract.

For the Windows consumer MVP, ADR-0007 classifies Hermes Profile model credentials as Runtime-native state. Hermes is their sole persistence authority. The adapter prefers a pinned, qualified official surface; because Hermes `0.20.2` has no safe offline non-interactive credential setter, ADR-0007 also authorizes one version-fixed, Profile-scoped, Provider-allowlisted compatibility writer for the canonical Hermes Profile `.env`. This writer is Hermes-owned adapter code, never a path/env/file API, and uses bounded optimistic conflict detection plus atomic replacement. YORVA does not keep a `SecretStore` or SQLite duplicate. The YORVA `SecretStore` boundary remains for YORVA-owned secrets and later explicitly designed credential types.

## 9. Hermes integration order

Preferred order:

1. documented Hermes public/management APIs;
2. documented TUI gateway JSON-RPC/WebSocket or other official programmatic interface where appropriate;
3. official CLI with machine-stable output where available;
4. controlled configuration compatibility layer.

Human-readable CLI parsing is a last resort and must be version tested.

## 10. Instance semantics

Core definition:

> An Instance is a logical independently manageable AI Runtime unit.

Hermes adapter mapping:

> An Instance maps to a Hermes Profile for V0.1.

The mapping does not imply one OS process per Instance. Runtime process topology is adapter-owned.

## 11. Capability model

YORVA must not scatter checks such as `runtime == hermes` through Desktop or application code.

Adapters report capabilities, for example:

```json
{
  "instances": true,
  "lifecycle": true,
  "models": true,
  "channels": ["weixin", "wecom", "telegram"],
  "skills": true,
  "mcp": true,
  "backup": true,
  "upgrade": true
}
```

Capabilities may depend on Runtime version.

Desktop uses capability data to hide, disable or explain unsupported actions.

## 12. Operation engine

Long-running work is represented by durable Operations.

```text
Request
  ↓
validate
  ↓
create Operation(PENDING)
  ↓
start work
  ↓
RUNNING + progress events
  ↓
SUCCEEDED / FAILED / CANCELLED
```

Rules:

- a handler must not keep a database transaction open during external work;
- cancellation is supported only where the underlying operation can be safely cancelled;
- failures use stable YORVA error codes plus diagnostic details in logs;
- conflicting operations are serialized at the narrowest sensible scope.

## 13. Concurrency model

Avoid one global lock.

Use scoped coordination for conflicts such as:

- install vs upgrade on one Runtime installation;
- start vs stop/restart on one Instance;
- delete vs update on one Instance;
- restore vs mutation on one Instance.

Read-only discovery may proceed concurrently where safe.

## 14. Persistence model

SQLite stores:

- YORVA node identity metadata;
- detected Runtime installation metadata;
- normalized instance inventory cache/metadata;
- operations;
- channel binding metadata;
- references for YORVA-owned secrets, never secret plaintext;
- backups metadata;
- application settings;
- audit-relevant management events.

Hermes remains authoritative for Hermes-owned state.

Under ADR-0007, Hermes Profile model credentials are Hermes-owned state and do not create `secret_refs` rows. Only safe configured/status metadata may cross the adapter boundary.

YORVA must be able to reconcile its local inventory from the Runtime after state drift.

## 15. Local transport

Desktop communicates with `yorvad` over loopback only:

```text
127.0.0.1:<dynamic-or-configured-local-port>
```

Transport:

- HTTP/JSON for commands and queries;
- SSE for server-to-Desktop events.

Authentication uses a local session/bootstrap credential. See `SECURITY.md` and `PROTOCOL.md`.

## 16. Future Control Plane topology

```text
                    ┌────────────────────┐
                    │ YORVA Control Plane│
                    │ Go + PostgreSQL    │
                    └─────────┬──────────┘
                              │ TLS/WSS
                  outbound    │ connections
           ┌──────────────────┼──────────────────┐
           ▼                  ▼                  ▼
        yorvad A           yorvad B           yorvad C
           │                  │                  │
        Runtime            Runtime            Runtime
```

Node initiates the connection. Cloud does not require an inbound Node management port.

Control Plane starts as one modular service. Splitting services requires an ADR and measured justification.

## 17. Remote command boundary

Allowed remote semantics are typed management commands such as:

```text
runtime.detect
runtime.install
instance.start
instance.stop
instance.restart
instance.config.update
channel.connect.begin
backup.create
```

Forbidden generic endpoints:

```text
shell.exec
process.exec
file.write.arbitrary
```

If a Runtime operation internally requires a command, the adapter owns the exact executable/arguments and validation.

## 18. Failure model

Every public/application error has:

- stable code;
- user-safe message;
- retryability;
- optional structured context;
- internal diagnostic cause retained in logs.

Example:

```json
{
  "code": "RUNTIME_NOT_INSTALLED",
  "message": "Hermes is not installed on this node.",
  "retryable": false
}
```

UI must not parse error strings for control flow.

## 19. Architectural red lines

Do not:

- make Desktop a shell-command host;
- make YORVA depend on Hermes internal Python packages;
- use a public unauthenticated local API;
- duplicate Runtime-owned secrets into SQLite;
- create a universal remote shell API;
- introduce microservices for organizational appearance;
- create a dynamic plugin SDK before a second Runtime proves the need;
- let future Cloud become mandatory for local functionality.

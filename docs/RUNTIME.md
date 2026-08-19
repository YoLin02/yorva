# YORVA Runtime Adapter Contract

## 1. Purpose

The Runtime adapter isolates YORVA from the implementation details of Hermes and future AI Runtimes.

The contract must be small enough for V0.1 and extensible only when real Runtime differences appear.

## 2. Core rule

> YORVA Core describes management intent. The Runtime adapter decides how that intent is performed by a concrete Runtime.

The adapter may use official APIs, RPC, CLI or narrowly scoped configuration compatibility internally. The caller must not know which mechanism was used.

## 3. Runtime identity

Stable Runtime kind values are lowercase identifiers:

```text
hermes
```

Future examples are not pre-registered until implemented.

## 4. Adapter descriptor

Conceptual Go model:

```go
type Descriptor struct {
    Kind        RuntimeKind
    Name        string
    Description string
}
```

The descriptor is static product metadata, not detection state.

## 5. Detection

Phase 2 adds one focused, read-only discovery capability to a registered Runtime bundle:

```go
type Discoverer interface {
    Detect(ctx context.Context) (Discovery, error)
}

type Discovery struct {
    RuntimeKind    RuntimeKind
    State          DiscoveryState
    ErrorCode      ErrorCode
    Selected       *Candidate
    Candidates     []Candidate
    Warnings       []Warning
    DetectedAt     time.Time
    SupportedRange string
}
```

Normalized discovery states are:

```text
NOT_INSTALLED
SUPPORTED
UNSUPPORTED
BROKEN_EXECUTABLE
MALFORMED_VERSION
TIMED_OUT
AMBIGUOUS
```

The application use case resolves the Runtime kind through the registry and owns the ten-second overall deadline. The Hermes adapter owns candidate enumeration, closed command construction, version parsing, compatibility and aggregation. Discovery must not mutate the machine. It is a live query: Phase 2 does not persist or cache its result.

On Windows, the Hermes adapter distinguishes a safely invocable CLI command from trusted installation evidence. Amendment 002A3: if `%LOCALAPPDATA%\hermes\control\active.json` is fully valid, discovery selects that generation's `bin\hermes.exe` (with 002A1 fallback only inside the same generation root) and does not treat leftover `hermes-agent` as `AMBIGUOUS`. If the pointer is missing or invalid, enumeration falls through to the frozen `%LOCALAPPDATA%\hermes\hermes-agent` launchers. A regular `hermes.exe` candidate may be executed with `--version`. When both documented launchers in the selected root are absent, the adapter may instead use that root's canonical `venv\Scripts\python.exe` with `-I -m hermes_cli.main --version`, but only after a site-packages enumeration capped at 1,024 entries and a 16-KiB canonical metadata read prove exactly one installed Hermes package declares `hermes = hermes_cli.main:main`. The repository wrapper and `hermes-agent.exe` remain evidence only and are never executed. Trusted evidence with neither safe command form is `BROKEN_EXECUTABLE`. `NOT_INSTALLED` is reserved for the absence of both a candidate and trusted installation evidence.

If exactly one candidate is runnable, it is selected even when other candidates are unusable. If two or more distinct candidates are runnable, discovery returns `AMBIGUOUS` with no selection. Cancellation returns `context.Canceled`; completed negative states return typed discovery data.

## 6. Feature contracts

Avoid one giant interface. Feature groups are explicit and small.

### Installation

Phase 3 implements only `Install`. Upgrade, repair and uninstall are absent.

Amendment 003A1 keeps installation Hermes-specific. The adapter acquires the official GitHub commit archive for `df4b65147d7ddd74dd449f9067aabbca5aef0ec7`, or the identical MSI-bundled archive after a transport failure, verifies the compiled size and SHA-256, extracts it with ZIP-slip/symlink/expansion limits, and skips the official PowerShell `repository` stage. This is a source fallback, not a complete offline installer.

Amendment 003A3: official `node` / `node-deps` PowerShell stages are never spawned. Managed Node `v22.23.1` and npm `12.0.2` live under `%LOCALAPPDATA%\hermes\node`. YORVA-managed health requires those exact pins, not any higher compatible replacement. Safe dependency installation is `npm ci --workspaces=false --omit=dev --ignore-scripts`.

Amendment 003A2 keeps dependency distribution inside the same Hermes adapter. The owned installer environment removes inherited Python/uv/pip/npm registry settings and injects fixed HTTPS PyPI and npm endpoints for the China Demo. The official `uv.lock` and `package-lock.json` remain unchanged; no generic mirror contract, user-selected registry or Runtime plugin abstraction is introduced.

Amendment 003A4: installation is one Install Transaction (`CREATED` → `COMMITTED`) with a sealed generation tree and `control/active.json` as the sole activation pointer. SQLite Operation remains the Desktop/API projection and does not authorize retry or recovery. Retry always starts a new transaction and new generation id.

```go
type Installer interface {
    Install(ctx context.Context, req InstallRequest, progress ProgressSink) (Installation, error)
}
```

### Instance management

```go
type InstanceManager interface {
    ListInstances(ctx context.Context, installation Installation) ([]NativeInstance, error)
    CreateInstance(ctx context.Context, installation Installation, req CreateInstanceRequest) (NativeInstance, error)
    DeleteInstance(ctx context.Context, installation Installation, nativeID string) error
    InspectInstance(ctx context.Context, installation Installation, nativeID string) (NativeInstance, error)
}
```

For Hermes V0.1, `nativeID` maps to the Hermes profile identity.

### Lifecycle

```go
type LifecycleManager interface {
    Start(ctx context.Context, installation Installation, nativeID string) error
    Stop(ctx context.Context, installation Installation, nativeID string) error
    Restart(ctx context.Context, installation Installation, nativeID string) error
    Status(ctx context.Context, installation Installation, nativeID string) (LifecycleStatus, error)
}
```

Not every Runtime must support this capability at Instance scope.

### Configuration

```go
type ConfigManager interface {
    GetConfig(ctx context.Context, installation Installation, nativeID string) (RuntimeConfig, error)
    ValidateConfig(ctx context.Context, installation Installation, nativeID string, patch ConfigPatch) (ValidationResult, error)
    UpdateConfig(ctx context.Context, installation Installation, nativeID string, patch ConfigPatch) error
}
```

`RuntimeConfig` is a normalized safe view. Runtime-specific advanced fields may appear in a namespaced metadata section rather than contaminating Core fields.

### Credentials

Secret material is passed to the adapter only for the mutation that needs it. It must not be returned from normal read methods.

```go
type CredentialManager interface {
    SetCredential(ctx context.Context, installation Installation, nativeID string, name string, value SecretValue) error
    DeleteCredential(ctx context.Context, installation Installation, nativeID string, name string) error
    CredentialStatus(ctx context.Context, installation Installation, nativeID string) ([]CredentialState, error)
}
```

ADR-0007 authorizes a Hermes `0.20.2`-specific credential compatibility fallback because its safe offline official setter is absent. The fallback is confined to the Hermes adapter and canonical Profile `.env`: `nativeID` selects the Profile, a compiled Provider allowlist selects the exact credential key, and no caller supplies paths or env names. It preserves unknown entries, changes one key, enforces a size bound, uses same-directory atomic replacement/read-back and returns a stable conflict when the observed source changes before replacement. It is not a generic Runtime file/config editor.

### Channels

```go
type ChannelManager interface {
    ListChannels(ctx context.Context, installation Installation, nativeID string) ([]ChannelState, error)
    BeginConnect(ctx context.Context, installation Installation, nativeID string, req ChannelConnectRequest, events ChannelEventSink) error
    Disconnect(ctx context.Context, installation Installation, nativeID string, channel string) error
}
```

QR and login state is emitted through the operation/event path. The adapter must not persist QR images as durable application data.

### Skills / MCP / Backup

Add focused interfaces only when implemented. They are capabilities, not mandatory methods on every adapter.

## 7. Runtime bundle/registry

V0.1 does not implement a dynamic plugin system.

The Node has a compile-time registry that wires the Hermes adapter and its supported feature interfaces.

Conceptual shape:

```go
type RuntimeBundle struct {
    Descriptor   Descriptor
    Discoverer   Discoverer
    Installer    Installer
    Instances    InstanceManager
    Lifecycle    LifecycleManager
    Config       ConfigManager
    Credentials  CredentialManager
    Channels     ChannelManager
}
```

Unsupported features are nil/absent and reflected in capability metadata.

Do not use runtime type assertions throughout application code; centralize capability lookup in the runtime registry.

## 8. Capability normalization

Capabilities answer what YORVA can safely offer for the detected Runtime/version.

```go
type Capabilities struct {
    Instances bool
    Lifecycle bool
    Models    bool
    Skills    bool
    MCP       bool
    Backup    bool
    Upgrade   bool
    Channels  []string
}
```

Capabilities may change by Runtime version.

## 9. Hermes V0.1 mapping

YORVA concepts map as follows:

```text
YORVA Runtime installation → detected Hermes installation
YORVA Instance             → Hermes Profile
YORVA Channel binding      → channel/gateway configuration for that profile
YORVA model config         → Hermes profile model/provider configuration
```

Do not expose the word `profile` in generic Node API fields when `instance` is sufficient. Runtime-specific detail screens may show “Hermes Profile” as explanatory text.

## 10. Hermes integration strategy

Use documented Hermes surfaces in this order:

1. official management/public API where suitable;
2. TUI gateway JSON-RPC/WebSocket for interactive agent capabilities where YORVA needs them;
3. official CLI for installation/profile/config operations where API coverage is absent;
4. narrowly scoped config compatibility adapter.

Hermes currently exposes multiple programmatic surfaces; YORVA should select by use case rather than force every operation through one protocol.

## 11. External command execution

When CLI execution is required:

- invoke executable + argv directly;
- avoid shell interpolation;
- validate all user-controlled values;
- set timeouts/cancellation;
- capture stdout/stderr separately;
- redact secrets before logging;
- normalize exit failures to YORVA error codes;
- keep command construction inside Hermes adapter packages.

Hermes discovery command descriptors are closed and argument-safe: either an enumerated Hermes launcher plus `--version`, or the Phase 2 amendment 002A1 validated official-root Python plus `-I -m hermes_cli.main --version`. The latter is a narrow compatibility fallback for an official package whose generated Windows launcher is absent; it is not a generic Python execution facility.

## 12. Version compatibility

The adapter owns a support policy.

For accepted Phase 2 amendment 002A1, stable Hermes `0.19.x` and `0.20.x` are supported (`>=0.19.0 <0.21.0`). This includes the current official stable package version `0.20.1` reviewed on 2026-08-14. Prereleases and `0.21.0` or later remain unsupported until explicitly reviewed.

Detection returns:

- installed version;
- support state;
- warnings;
- effective capabilities.

Support states:

```text
SUPPORTED
PARTIALLY_SUPPORTED
UNSUPPORTED
UNKNOWN_VERSION
```

Unknown versions should fail conservatively for mutating operations that depend on unstable behavior while still allowing safe detection/status where possible.

## 13. Error normalization

Adapter-specific failures map to stable errors such as:

```text
RUNTIME_NOT_INSTALLED
RUNTIME_UNSUPPORTED
RUNTIME_COMMAND_FAILED
INSTANCE_NOT_FOUND
INSTANCE_ALREADY_EXISTS
CAPABILITY_NOT_SUPPORTED
CONFIG_INVALID
CHANNEL_NOT_SUPPORTED
CHANNEL_AUTH_FAILED
CHANNEL_AUTH_TIMEOUT
```

Raw Hermes errors may be attached to internal diagnostic context but should not become client contracts.

## 14. Reconciliation

YORVA inventory can drift if a user modifies Hermes outside YORVA.

The Node must support reconciliation:

```text
query Hermes
→ normalize native instances
→ upsert YORVA metadata
→ mark missing native instances as removed/stale
```

Do not assume YORVA is the only management surface.

## 15. Adapter tests

The Hermes adapter must have tests for:

- detection with absent/present Hermes;
- supported/unsupported version behavior;
- profile list normalization;
- create/delete mapping;
- command argument safety;
- config validation/update behavior;
- credential redaction;
- channel QR/login state normalization;
- error mapping;
- cancellation/timeouts;
- reconciliation after external changes.

Contract tests should be reusable when a second Runtime is added, but do not build a full plugin conformance framework in V0.1.

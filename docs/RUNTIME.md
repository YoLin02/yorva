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

Amendment 003A3: official `node` / `node-deps` PowerShell stages are never spawned. Managed Node `v22.23.1` and npm `12.0.2` live under `%LOCALAPPDATA%\hermes\node`. YORVA-managed health requires those exact pins, not any higher compatible replacement. Safe dependency installation is `npm ci --workspaces=false --omit=dev --ignore-scripts`. Amendment 003A5 corrects the pinned Node ZIP per-member limit to 96 MiB because the verified archive contains an 86,989,128-byte `node.exe`; the 256 MiB total expansion limit and all provenance/path checks remain unchanged.

Amendment 003A2 keeps dependency distribution inside the same Hermes adapter. The owned installer environment removes inherited Python/uv/pip/npm registry settings and injects fixed HTTPS PyPI and npm endpoints for the China Demo. The official `uv.lock` and `package-lock.json` remain unchanged; no generic mirror contract, user-selected registry or Runtime plugin abstraction is introduced.

Amendment 003A7 supersedes only the fixed/online-first transport decisions in 003A1,
003A2 and 003A3. Packaged Hermes, Node.js and npm artifacts are verified and used first;
when a corresponding artifact is absent, the adapter downloads from the start-time
snapshot of the Hermes download-source settings. Archive versions, sizes and SHA-256
pins remain compiled and immutable. Python and npm registry URLs are configurable
credential-free HTTPS transports passed through the existing allowlisted environment;
they do not authorize a generic Runtime source plugin, lockfile rewrite or executable
override.

Amendment 003A8 packages the pinned CPython `3.11.15` Windows x64 archive used by the
reviewed uv installer. The adapter verifies the selected bundled or online archive,
materializes it as an Operation-private local uv mirror with one pinned metadata entry,
and injects only those `file://` locations through `UV_PYTHON_INSTALL_MIRROR` and
`UV_PYTHON_DOWNLOADS_JSON_URL`. `artifactPreference` selects bundled-first
or online-first behavior for Hermes, Node.js, npm and Python artifacts. Only transport
failure may use the alternate source; integrity mismatch remains fail-closed.

Amendment 003A4: installation is one Install Transaction (`CREATED` → `COMMITTED`) with a sealed generation tree and `control/active.json` as the sole activation pointer. SQLite Operation remains the Desktop/API projection and does not authorize retry or recovery. Retry always starts a new transaction and new generation id.

Amendment 003A6: a new Hermes generation is constructed directly at its final
`generations/gen_<id>` path. It is still inactive until `control/active.json` names it.
Before Seal, YORVA executes both documented Hermes launchers from that exact final path
with `--version` and requires the pinned supported version. File existence alone is not
successful installation evidence. Publishing verifies the sealed candidate and records
the state transition; it no longer moves a completed virtual environment from staging.

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

Phase 6 normalizes only `RUNNING`, `STOPPED`, and `UNKNOWN`. Transient Starting,
Stopping, and Restarting presentation comes from the active Operation. Compatible stable Hermes `0.20.x`
maps this contract to the exact Profile messaging gateway. Manual Start never enables
login persistence; OS task names, PIDs, paths, and human command output remain inside the
Hermes adapter.

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

ADR-0007 authorizes a narrow Hermes credential compatibility fallback because the qualified stable `0.20.x` offline official setter is unsafe for Secret input. The fallback is confined to the Hermes adapter and canonical Profile `.env`: `nativeID` selects the Profile, a compiled Provider allowlist selects the exact credential key, and no caller supplies paths or env names. It preserves unknown entries, changes one key, enforces a size bound, uses same-directory atomic replacement/read-back and returns a stable conflict when the observed source changes before replacement. It is not a generic Runtime file/config editor.

P6.5 adds request-scoped Provider model discovery to the same qualified Hermes model
adapter. Each preset owns one fixed HTTPS catalog endpoint and authentication shape;
callers supply only the preset ID and a transient credential. The adapter returns
validated model IDs and does not retain the credential. Hermes remains authoritative
for the active `model.default`; YORVA persists only the user-selected non-secret model
ID set for reopening the Desktop configuration view.

P7R adds Runtime-owned `ProviderConnection`, `ModelProfile` and `RuntimeDefault`
resources above the existing exact-Instance adapter. A Provider Connection selects only
a compiled Provider Preset and owns one OS-protected source credential. A Model Profile
contains reusable non-secret model selection. Applying a Profile is an explicit durable
Copy-on-Apply Operation: Core resolves exact Instances, the Hermes adapter writes each
exact Profile, rereads authoritative model/credential status, and records a separate
binding result. `INHERIT` always means the current Runtime Default Profile;
`OVERRIDE` selects another Profile for that Instance. MVP does not continuously sync
later Profile changes, and pre-existing Hermes configuration remains external until an
explicit successful YORVA apply.

### Channels

```go
type ChannelManager interface {
    ListChannels(ctx context.Context, installation Installation, nativeID string) ([]ChannelState, error)
    BeginConnect(ctx context.Context, installation Installation, nativeID string, req ChannelConnectRequest, events ChannelEventSink) error
    Disconnect(ctx context.Context, installation Installation, nativeID string, channel string) error
    PairingStatus(ctx context.Context, installation Installation, nativeID string, channel string) (PairingStatus, error)
    ApprovePairing(ctx context.Context, installation Installation, nativeID string, channel string, code SecretValue) error
}
```

QR and login state is emitted through the operation/event path. The adapter must not persist QR images as durable application data.

Phase 6 sender pairing remains a small Channel capability rather than a generic identity
or RBAC subsystem. Compatible stable Hermes `0.20.x` maps it to the official Profile-scoped pairing surface.
Core receives only a pending count and a normalized approval result; Hermes request IDs,
files, user rows and command output do not cross the adapter boundary.

### Skills / MCP / Backup

Add focused interfaces only when implemented. They are capabilities, not mandatory methods on every adapter.

Phase 7 Skills has two separate capability layers. `NativeSkillCapabilities` reports
six Hermes-owned facts independently: inventory, install, update, remove,
enable/disable and Profile binding. Exact Hermes `0.20.5` keeps the qualified native
inventory read, while the five unreliable native mutation/binding surfaces remain
unsupported with `deferred_upstream`.

The separate YORVA-managed lifecycle provides install, update, enable, disable and
remove through a `SkillProjector`. Its source is the compile-time approved catalog and
its authoritative managed copy is `{dataDir}/skills/managed`; the exact Hermes Profile
Skills directory is only a projection target. On Windows the adapter uses copy
projection. Enable projects the managed copy, disable removes only a verified
YORVA-owned projection, and neither action edits Hermes `config.yaml`.

An observed Runtime projection is YORVA-owned only when the matching `managed_skills`
row, deployment ID, projection marker and one deterministic whole-package SHA-256 agree.
A pre-existing or mismatched destination is external and read-only. Inventory merges
native observation with managed records and reports ownership plus `PROJECTED`,
`NOT_PROJECTED`, `DRIFT_MISSING`,
`DRIFT_MODIFIED`, `CONFLICT` or `UNKNOWN`. Successful projection changes restart a
running Instance so Hermes does not retain a stale Skills cache; a stopped Instance
remains stopped. Mutations are durable Operations and interrupted work is reconciled
from the record and filesystem truth rather than replayed blindly. See ADR-0018.

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

Phase 7 reports MCP capability facets independently: `MCPRead`, `MCPMutate` and
`MCPTest`. Runtime Definitions are separate from exact-Instance bindings. The built-in
loopback Preset proves the baseline lifecycle. P7R accepts only YORVA-reviewed Presets;
caller-provided endpoint, header, stdio executable/argv, environment, path and arbitrary
JSON fields are not part of the contract. The Hermes adapter owns YAML projection,
bounded connection testing, cleanup and authoritative Profile readback. External Hermes
definitions remain read-only.

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

ADR-0012 and Amendment 0065A1 supersede the exact-patch rule from Amendment 002A4.
YORVA supports stable Hermes `>=0.20.2 <0.21.0` through one shared compatibility gate
for discovery and implemented capabilities. Older releases, prereleases and another
minor/major line remain detectable but `UNSUPPORTED`. The packaged source is still one
exact hash-verified snapshot so Runtime compatibility does not weaken build provenance.

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

### Managed Upgrade plan truth

Hermes exposes a read-only `UpgradePlan` capability only when the adapter can re-read a
live valid `active.json`, verify the matching sealed generation and compare it with the
one exact packaged snapshot compiled into YORVA. The plan capability is not Upgrade or
Rollback mutation authority. Those two capabilities remain false until exact
compatibility, protection-point and post-check qualification is complete.

The public projection contains only the current version, a product-facing packaged
candidate label/version, managed status, compatibility status, protection-point
requirement and closed blocked-reason codes. Source commits, hashes, seals, filesystem
paths, commands, URLs and internal protection-point identities stay adapter-internal.
For the current `0.20.2` to packaged `0.20.5` pair, compatibility and protection evidence
are not qualified, so the plan is `UNKNOWN` and never authorizes a mutation.

An actionable plan requires all of: a valid YORVA `active.json`; a matching sealed known
generation identity; authoritative inventory of affected Profiles, gateways and managed
features; a verified encrypted protection backup; exact Windows compatibility evidence
for the current/candidate pair; and qualified post-upgrade detection, inventory, health,
configuration, Skill, MCP and gateway checks. Missing items are returned as closed public
reason codes rather than one undifferentiated failure.

### Managed Runtime backup destination

Backup Create uses the adapter-owned per-user `{dataDir}/backups` directory and a
daemon-generated backup ID. React and HTTP cannot choose or read the artifact path.
Device-key encryption, ciphertext authentication, checksum verification and index write
must all succeed before `AVAILABLE` is reported. For a stopped Hermes 0.20.5 Runtime,
stable SQLite database and WAL bytes are captured together; regenerable SHM and LSP
material are excluded. Any source change during the bounded double-read still fails
closed. A sharing/lock violation from Hermes Dashboard or another background process is
reported as `BACKUP_SOURCE_RUNTIME_NOT_STOPPED`, not as a generic create failure.

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

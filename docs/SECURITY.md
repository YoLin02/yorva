# YORVA Security Model

## 1. Security objective

YORVA controls software capable of accessing models, messaging accounts, files, tools and potentially shell-capable agent actions. The management plane must therefore be treated as privileged software even when running locally.

Security goal:

> Minimize exposed authority, keep secrets local by default, use typed management operations, and never turn remote management into arbitrary machine control.

## 2. Trust boundaries

Primary boundaries:

```text
React WebView
   │
   │ local authenticated API
   ▼
yorvad
   │
   │ controlled adapter calls
   ▼
Hermes / Runtime
   │
   ├── model providers
   └── messaging platforms
```

Future:

```text
YORVA Cloud
   │ TLS/WSS
   ▼
yorvad
```

Cloud must not automatically gain access to local Runtime secrets.

## 3. Threats considered

V0.1 threat model includes:

- another local process calling the daemon API;
- malicious/untrusted content rendered in Desktop causing XSS-like execution;
- secret leakage through logs/errors;
- command injection through Runtime adapter inputs;
- accidental public binding of the daemon;
- over-privileged daemon execution;
- malicious or compromised future Control Plane command;
- stale/forged device pairing;
- unsafe update/install artifacts.

It does not claim to defend a user session against a fully compromised administrator/root account on the same machine.

## 4. Local daemon exposure

`yorvad` binds only to loopback by default.

Required:

```text
127.0.0.1
```

Forbidden default:

```text
0.0.0.0
```

No management endpoint other than minimal startup health/bootstrap behavior is unauthenticated.

The production `yorva-mcp-test` Preset starts one separate ephemeral IPv4 loopback
listener owned and closed by the daemon-scoped Hermes MCP manager. It accepts only
bounded MCP `initialize`, `notifications/initialized` and `tools/list` requests and
returns the single non-privileged `yorva_ping` tool. The listener address is never
returned through HTTP/Desktop or written into Hermes configuration. The fixed reviewed
identity is redirected to loopback only inside the private adapter client; this does
not create a generic proxy or authorize caller-provided hosts.

P7R MCP mutation selects only YORVA-reviewed Presets. Desktop and HTTP callers cannot
provide endpoints, headers, stdio executable/argv, environment, paths or arbitrary MCP
JSON. Preset credentials are write-only, omitted from SQLite/read APIs and projected
only to the exact selected Profile authority.

## 5. Local Desktop authentication

Desktop and daemon establish a local bootstrap/session credential.

Requirements:

- cryptographically random;
- short-lived/session-scoped where practical;
- not stored in URL/query parameters;
- not logged;
- rotated when the daemon/bootstrap session is recreated;
- rejected on invalid/missing authentication;
- not accepted over non-loopback interfaces.

The exact bootstrap transport may use a Tauri-controlled process handoff, inherited handle/file descriptor, protected local file/pipe, or equivalent OS-safe mechanism. Choose the smallest cross-platform solution during bootstrap implementation and document it in an ADR if it materially changes the trust model.

## 6. Desktop WebView hardening

- ship local bundled frontend assets;
- do not load arbitrary remote application code;
- strict Content Security Policy;
- no `eval`/unsafe dynamic script execution;
- restrict Tauri capabilities/permissions to what YORVA uses;
- validate external URLs before opening them in the system browser;
- keep dependency count controlled;
- treat rendered Runtime/log text as untrusted data.

A local API token being present in Desktop memory does not remove the need for CSP and input/output hardening.

## 7. Secret storage

YORVA-owned secrets must be stored behind a `SecretStore` abstraction backed by OS-secure storage when available. Runtime-native secrets may instead use an Owner-approved Runtime authority defined by an ADR.

Examples of secret scopes:

```text
node/device credential
instance/channel credential
future cloud refresh/session credential
```

ADR-0007 defines the Hermes Profile credential as Runtime-native state and P7R defines a
separate reusable YORVA Provider Connection as YORVA-owned source state. The connection
credential exists once in the OS-backed SecretStore; SQLite contains only an opaque
reference. Copy-on-Apply holds plaintext only for the bounded mutation and sends it only
to the qualified exact-Profile Hermes adapter. The resulting Hermes Profile copy is not
treated as a second YORVA record or synchronized continuously. No caller supplies a
path, env name, Provider endpoint or config key; the Provider Preset and adapter select
all such targets. Ordinary reads expose only configured/status metadata.

Rules:

- SQLite stores only safe references/metadata and never secret plaintext;
- no silent plaintext fallback;
- ordinary API reads return `configured` metadata only;
- replacement writes accept secret values but never echo them;
- delete actually removes provider-backed material where possible;
- backup does not include secrets unless a future explicit encrypted-secret export feature is designed.

Provider model discovery may use a credential in a request header only for one bounded
request to the fixed HTTPS endpoint compiled into the qualified Hermes adapter. The
credential is write-only at the local API boundary and is not stored by discovery,
returned in the response, copied into SQLite/Desktop storage, or included in logs,
events, Operations, URLs, or diagnostics. Redirects are rejected so custom Provider
authentication headers cannot cross the qualified endpoint boundary. Only validated
model IDs are returned.

For Runtime-native credentials, Profile isolation and exact native targeting are mandatory. Secret material must not enter argv, URLs, logs, events, Operations, diagnostics, Desktop storage, Windows user/system environment variables or ambient child environments. The official surface or approved compatibility writer must satisfy those rules; otherwise the integration stops.

## 8. Secret redaction

Logging must redact:

- Authorization headers;
- API keys;
- OAuth access/refresh tokens;
- channel tokens/secrets;
- QR authentication payloads where they can be used as credentials;
- device private keys;
- sensitive environment variables.

Add tests for common provider key formats and structured-field redaction. Redaction is defense-in-depth; code should avoid logging secret-bearing objects in the first place.

## 9. Runtime command execution

When an adapter invokes CLI commands:

- execute binary with argv, not a concatenated shell string;
- validate profile/instance/channel identifiers;
- never interpolate raw user text into a shell;
- constrain environment passed to child processes;
- set timeout/cancellation;
- capture output with size bounds;
- redact before persistence/logging;
- normalize errors.

Generic arbitrary shell endpoints are forbidden.

Phase 2 Hermes discovery applies this boundary concretely: `yorvad` enumerates only bounded documented candidates, resolves an absolute regular executable, and invokes it directly through a closed command descriptor. The normal form is a Hermes launcher with constant argv `--version`. On Windows, when both documented launchers are absent, amendment 002A1 permits the canonical Python executable inside `%LOCALAPPDATA%\hermes\hermes-agent\venv` with constant argv `-I -m hermes_cli.main --version`, only after canonical containment checks, a site-packages enumeration capped at 1,024 entries and an at-most-16-KiB metadata read prove exactly one installed Hermes package declares the exact console entry point `hermes = hermes_cli.main:main`. Duplicate, ambiguous, non-matching, oversized or escaping metadata rejects the fallback. Fixed official-layout markers, the repository Python wrapper and `hermes-agent.exe` are never executed. This is not a generic Python or module execution surface.

Phase 3 installation uses the same process-containment rules for a trusted Windows PowerShell executable resolved from the OS Windows directory, not PATH. The official installer script is obtained from the verified official commit archive after CRLF normalization, or from one immutable raw URL, then size- and SHA-256-verified, stored under an Operation-private directory, and re-hashed before every invocation. No user-controlled URL, path, argv or environment is accepted. Excluded official stages (`desktop`, `platform-sdks`, `configure`, `gateway`) are never spawned. The official `repository` stage is replaced by YORVA-owned verification and extraction of the official GitHub commit archive (online, then the MSI-bundled copy on transport failure only). Archive members are rejected for traversal, absolute paths, alternate streams, symlinks, excessive count and expansion bombs. The adapter evaluates at most eight candidates, limits stdout and stderr separately to 64 KiB, uses a three-second per-candidate timeout and a ten-second overall application deadline, and owns the full child process tree with a Windows Job Object or Unix process group. Windows children are created suspended, assigned to the kill-on-close Job Object, and only then resumed so descendants cannot escape before ownership is established. Cancellation, timeout, output overflow and normal return terminate any remaining descendants and wait/reap the direct child. The child receives an allowlisted environment containing only OS execution essentials and adapter-owned Hermes installation values; provider credentials and arbitrary inherited Python, uv, pip or npm registry settings are excluded. Amendment 003A2 then appends one fixed HTTPS Python index and one fixed HTTPS npm registry. These endpoints are transport policy, not integrity attestations; the reviewed official lockfiles remain the package integrity inputs. Raw command output is not returned through HTTP or Desktop. One structured application-level outcome record contains only Runtime kind, stable state/error code, counts, duration and timeout/cancellation flags; it excludes executable paths, command output, raw errors and environment values. Tauri may pass only the fixed packaged archive path to `yorvad` at bootstrap. That path is never exposed through HTTP or Desktop. Automatic retry of a failed install always starts a new Install Transaction with new staging and generation ids. `ownership_nonce` is not generated and must not authorize retry. Mutating official stages run against `staging/txn_*`. Public `bin` launchers are copied from `venv/Scripts` before Seal. Seal, publish and activate require a complete manifest walk; post-seal insert, modify or delete fails closed and never writes `control/active.json`. User `PATH` and `HERMES_HOME` are derived from a valid `active.json` after activation. A missing pointer is the only first-install vacancy. A present but invalid `active.json` is fail-closed (`INSTALL_BLOCKED_UNSAFE`) and is never overwritten, newest-generation inferred, or treated as absent. Unknown directories, leftover `hermes-agent` and official Hermes user data are never automatically deleted. Atomic transaction, seal and pointer writes fail closed if parent-directory durability fails before replace. After a successful replace, a complete readable record is the recovery truth even if a later directory-sync call fails.

Amendment 0065A1 supersedes only the discovery timing values in the preceding text.
Official Hermes version probes use a discovery-only 30-second child limit inside
32/35/40-second adapter/application/Desktop deadlines. Other commands keep their
existing narrow timeouts. Concurrent discovery callers for one Runtime share one
bounded probe, and a supported result may be reused for five seconds; cancellation of
the final waiter still terminates the owned process trees.

Amendment 003A6 supersedes only the preceding staging-construction detail. New Hermes
runtime bytes are built directly under their final `generations/gen_<id>` path, which is
bound to the transaction by a bounded candidate ownership record and remains inactive
until the sole `active.json` pointer changes. Before Seal, both fixed final-path Hermes
launchers run with `--version` under the same process-tree, environment, timeout and
output controls. A regular file or valid manifest without a successful final-path probe
is not install success. Legacy staging-built virtual environments are never moved or
activated by recovery.

Amendment 003A3: YORVA never executes official PowerShell `-Stage node` or `-Stage node-deps`. Those stages include unpinned `irm | iex` of `raw.githubusercontent.com/trycua/cua/main` and other best-effort browser installs. Managed Node `v22.23.1` and npm `12.0.2` are verified archives. Node dependency installation is `node.exe npm-cli.js ci --workspaces=false --omit=dev --ignore-scripts` only. Amendment 003A5 raises only the per-member Node/npm archive extraction cap to 96 MiB so the pinned 86,989,128-byte `node.exe` can be extracted; exact archive size/SHA-256, the 256 MiB total Node ZIP limit and all traversal/reparse protections remain enforced.

Amendment 003A7 supersedes the Phase 3 fixed/online-first transport details without
weakening artifact identity. Verified MSI-packaged Hermes, Node.js and npm bytes are
primary. A missing packaged artifact may use a configured absolute HTTPS URL, but the
download must still match the compiled exact size and SHA-256 before extraction.
Python/npm registry URLs may also be configured. Every stored URL must have a host and
must not contain user information, a query or fragment; credentials, local paths,
versions, hashes, commands and environment keys remain outside the API. Each Operation
uses one start-time snapshot, inherited secret/registry variables remain stripped, and
source URLs are never written to ordinary logs or Operation messages.

Amendment 003A8 adds one exact packaged CPython archive and a two-value artifact
priority. The Python version, size and SHA-256 remain compiled. The configured online
URL can change transport but not artifact identity. The adapter verifies the archive,
copies it beneath the Operation-private directory using uv's pinned release layout,
writes a single-entry metadata document for the exact patch/build, and passes only the
resulting local `file://` locations to uv. Integrity failure never
falls back; arbitrary local paths, hashes, versions, credentials and commands remain
outside the API.

## 10. Privilege model

`yorvad` runs as the normal user by default.

Do not permanently run as Administrator/root.

If installation requires elevation:

1. describe the action to the user;
2. elevate only the narrow installer step;
3. return to normal privilege afterward;
4. never make privilege elevation a hidden side effect of unrelated actions.

## 11. File access

YORVA may access only paths required for:

- its own state;
- known Runtime installation/configuration locations;
- explicit user-selected backup/export locations.

ADR-0020 supersedes user selection for managed Runtime Backup Create. The daemon derives
one absent encrypted artifact path below its trusted per-user system application-data
directory. React and HTTP submit no path; portable export remains a separate future
action.

Adapter code must not expose arbitrary read/write file endpoints to Desktop or Cloud.

Normalize and validate paths before mutation. Protect against traversal when accepting relative names.

Phase 7 managed Skills follow ADR-0018. Public callers select either a compile-time
approved source ID or an opaque one-time source reference issued by the native Desktop
ZIP/directory picker. They cannot submit a URL, local path, command, environment, force
flag or arbitrary package bytes through HTTP. Native imports are copied into private
staging, reject traversal, links/reparse points, multiple archive roots, non-prose files
and bounded-size violations, and are consumed after validation. YORVA keeps the
authoritative immutable copy below `{dataDir}/skills/<instance>/<skill>/<digest>` and
derives the exact Hermes default/named-Profile projection path inside the adapter.

YORVA never adopts an existing Hermes Skill. When a Runtime projection exists, changing
or removing it requires the matching `managed_skills` record, deployment ID, projection
marker and one deterministic whole-package SHA-256. A missing marker, changed digest or
pre-existing unmatched destination is reported as drift/conflict and is left untouched.
Copy projection does not use elevated symlink privileges, edit Hermes `config.yaml`,
invoke Hermes-native mutation or expose internal paths/digests through HTTP. Interrupted Skill Operations
are marked failed and reconciled from managed metadata and filesystem truth; destructive
steps are not replayed automatically.

## 12. Runtime credentials and Cloud

Local Runtime credentials remain local by default.

Future Cloud inventory may contain:

- credential configured/not configured;
- provider name;
- last validation time;

but not the underlying secret unless the user enables a separately designed enterprise secret-management feature.

## 13. Future device identity

For Cloud-connected Nodes, use a per-Node device keypair stored locally in secure storage.

Recommended high-level pairing flow:

```text
Node generates device keypair
→ user signs in to Control Plane
→ Node displays short-lived pairing code / QR
→ user approves Node
→ Control Plane binds public key to Node identity
→ Node authenticates future WSS sessions using challenge/signature
→ Control Plane issues short-lived session credential
```

A private device key never leaves the Node.

Exact cryptographic protocol must use standard libraries/protocols, not custom crypto primitives.

## 14. Cloud connection

Node initiates outbound WSS over TLS.

Cloud commands must be:

- typed;
- authenticated;
- authorized;
- uniquely identified;
- auditable;
- mapped to existing application use cases.

The Control Plane cannot send arbitrary shell text as a command.

## 15. Authorization

V0.1 local mode has one local-user trust context after successful Desktop↔daemon bootstrap.

Do not prematurely build enterprise RBAC into local V0.1.

Future Control Plane authorization must be evaluated before command delivery and again by Node policy for locally forbidden actions.

## 16. Channel QR/login flows

QR/login payloads may be credential-equivalent while valid.

Rules:

- store only in bounded daemon memory (8 KiB maximum) and modal-local React state;
- publish only to the bearer-authenticated initiating Desktop session, matched by an
  ephemeral 20–128 character `Yorva-Session-Id` held in process memory;
- apply explicit expiry and `Cache-Control: no-store`;
- clear after success, failure, cancellation, timeout, replacement, expiry or daemon
  shutdown;
- do not write QR payloads to audit logs;
- do not place QR payloads in URLs, SQLite, Operation rows/events, logs, diagnostics,
  localStorage, sessionStorage, Zustand or backups;
- shared SSE may announce only the Operation ID and expiry metadata.

Weixin sender-pairing codes are short-lived authorization proofs. They are accepted only
in an authenticated loopback JSON body, validated against the fixed eight-character
Hermes alphabet, held only for the synchronous approval call, and never returned,
persisted, logged, placed in a URL/query key/event/Operation or copied into browser
storage. Compatible stable Hermes `0.20.x` exposes approval through its official CLI but not stdin; the
adapter may therefore pass this one-time code as one fixed-position argv
value to the exact validated Profile command. This narrow local-user tradeoff does not
authorize channel tokens, Bot secrets, API keys, arbitrary arguments or a generic shell
surface. Output is bounded, discarded after stable-result parsing and never exposed.

Hermes Channel integration follows ADR-0012's stable `0.20.x` compatibility policy. Weixin permits only the fixed
qualified HTTPS iLink host, disables redirects, bounds responses to 64 KiB and fails
closed on unknown states/hosts. WeCom permits only the fixed official WSS host and writes
Bot ID/Secret to the exact Profile only after an authenticated subscribe response. The
adapter accepts no caller-supplied host, path, filename or environment key. Disconnect
removes only the selected local binding and never represents that as remote account/bot
revocation.

## 17. Updates and installers

Runtime/YORVA installation and update flows must verify provenance using the strongest verification mechanism available from the official distribution source.

At minimum:

- HTTPS official source;
- checksum/signature verification when published;
- bounded download destination;
- no execution of untrusted mirror content by default;
- explicit version reporting after install.

Do not invent checksum values when upstream does not publish them.

## 18. Audit

Audit relevant actions:

- install/upgrade;
- instance create/delete;
- credential set/delete metadata (never value);
- channel connect/disconnect;
- remote command accepted/rejected;
- backup/restore;
- security-sensitive settings changes.

Audit data is not a substitute for logs and does not contain secrets.

## 19. Security review triggers

Require explicit security review/ADR before adding:

- public daemon binding;
- remote shell;
- secret sync to Cloud;
- browser-loaded remote code/plugins;
- dynamic Runtime plugins;
- automatic elevated service installation;
- remote filesystem access;
- multi-user local daemon access.

## 20. Phase 8 release and privacy contract

Phase 8 uses `PRODUCT_SUPPORT.md` as the Windows product-support contract. The MVP has no
telemetry and does not upload crash, usage, Runtime or diagnostic data. Diagnostic export
is local, explicitly user-triggered and limited to a fixed sanitized projection.

The diagnostic archive contains only fixed JSON summaries and an allowlisted projection
of YORVA's own NDJSON log. Node, Instance, Operation and Profile identifiers are hashed;
unrecognized log fields are dropped and secret/path-like values are replaced. Provider
keys, tokens, channel/MCP credentials, QR/pairing values, cookies, authorization data,
ambient environment, raw SQLite data, absolute paths and arbitrary user files are never
eligible inputs. The daemon enforces record, age and size limits before returning the
complete ZIP. Tauri exposes only the fixed export action and an atomic Save As publish,
not a general read/write API. An existing destination must be a regular non-link,
non-reparse file; dangling links fail closed before a partial is created. Its private
loopback request disables proxies and redirects so the bearer credential cannot leave the
daemon endpoint; cancellation and write failure leave no success state or partial archive.

YORVA update packages must be checked for fixed origin, size, SHA-256, version and the
approved Windows signature policy before execution. The Owner confirmed on 2026-09-03
that production code-signing material is not currently available. Unsigned builds may be
internal candidates only; lack of signing must not be hidden by self-signing, disabled
verification or a public-ready label.

Update metadata is accepted only after Ed25519 verification with the release key compiled
into the Desktop. The full MSI is streamed into a bounded YORVA-owned staging directory,
then its exact length, SHA-256, ProductVersion and declared Authenticode policy are checked.
The native handoff uses only the resolved Windows system PowerShell and `msiexec` paths,
closed arguments and YORVA-owned files. It never accepts a caller-provided URL, path,
command, argument, environment value or header. A qualification-only Cargo feature uses a
fixed disposable-VM HTTPS origin and test CA; packaging marks it as non-release metadata
input so it cannot be promoted as a public artifact.

Before applying a pending SQLite migration, yorvad creates a fixed-location consistent
protection database and records a SHA-256 plus source/target schema in a bounded,
non-secret state file. Interrupted work restores only from a regular file with the exact
recorded closed name and digest. Missing, malformed or tampered recovery evidence blocks
startup with `DATABASE_MIGRATION_RECOVERY_REQUIRED`; it never falls back to deleting the
database or continuing Runtime mutations.

The one-time Desktop product-identity migration uses non-following metadata checks and
rejects symbolic links, including dangling links, and Windows reparse points at its
legacy/stable/staging roots, marker, and every copied entry. The updater
validates `state.json` with non-following metadata before staging, flushes a same-directory
temporary state record, and atomically replaces the prior state; dangling links fail
closed without creating temporary state, and an interruption before publication leaves
the previous durable state rather than an absent-state window.

The Phase 8 stability harness is development qualification code, not a management API or
shipping Runtime feature. It requires an explicit test-only Hermes executable, creates a
new absent work root, replaces `LOCALAPPDATA`/`APPDATA` for its child processes, disables
HTTP proxy use, and talks only to the private loopback bootstrap session it creates. It
must not discover, copy, mutate, or emit the user's Hermes/YORVA roots or credentials.
Diagnostic canaries and bootstrap tokens are scanned from every exported archive.

The native Desktop recovery exercise uses Tauri's real Windows Known Folder product-data
path and therefore runs only in a disposable Windows user profile, VM, or CI runner. It
requires an explicit disposable-profile acknowledgement and fails before process launch
if stable or legacy YORVA product data already exists. It replaces only its owned
Desktop/daemon processes and records `hostRebooted=false`; controlled Windows reboot
evidence is collected separately inside a disposable guest and never requires rebooting
the shared development host.

## 21. Phase 8 threat-model refresh

`SECURITY.md` is the authoritative threat model for the local product. Phase 8 adds or
changes these attack and failure surfaces:

- a legacy product-data tree may contain a junction, symbolic link, or other Windows
  reparse point intended to escape the migration root;
- signed update metadata, staged packages, or the durable updater state may be tampered
  with, redirected, truncated, or interrupted between verification and publication;
- a system proxy or HTTP redirect may try to move the daemon bearer credential away from
  the private loopback endpoint during diagnostic export;
- diagnostic input may contain secrets, absolute paths, unknown structured fields, or
  enough records to exhaust memory or disk;
- abnormal Desktop/daemon termination may leave an orphan process, stale bootstrap
  identity, or UI state that appears recovered before authoritative Runtime readback;
- a qualification harness may accidentally inherit the user's application roots,
  credentials, proxy settings, or unrelated process identities.

The corresponding Phase 8 controls are closed-root migration with reparse rejection;
verified update origin, metadata, digest, version, signature policy, bounded staging and
atomic durable state replacement; no-proxy/no-redirect authenticated diagnostic export
with an allowlisted bounded projection; owned-process restart followed by a fresh
bootstrap handshake and authoritative inventory readback; absent isolated daemon test
roots; and a fresh disposable Windows profile with exact-candidate process ownership for
native Desktop recovery. These controls do not alter the existing boundary
that a fully compromised administrator/root session is out of scope.

### Phase 8 audit remediation — 2026-09-07

Updater postcheck uses authenticated `GET /api/v1/node/recovery`, with proxy forwarding
and redirects disabled and a bounded response. The daemon performs current authoritative
readback; native code requires READY, no reported error and an exact daemon/Desktop
version match. A valid bootstrap handshake alongside UNKNOWN inventory cannot be reported
as update success. Runtime failure retains the queryable management/diagnostics surface.

Interrupted download recovery uses the download mutex to establish that no worker still
owns the state. Only fixed regular partial/package files under the validated updater root
may be cleaned; the failed record and verified-metadata candidate are retained for retry.
Owner Amendment 008A1 removes production-signing availability from the P8 internal freeze
prerequisites. It does not disable package/metadata verification or authorize an unsigned
public release; production signing remains a separate public-release requirement.

## 22. Phase 9 OpenClaw boundary

OpenClaw runs at the current Windows user's ordinary integrity and canonical profile paths. No automatic native service, Scheduled Task, Startup entry or elevation is added. Named profiles share a local-user trust boundary; they are not hostile tenant isolation.

The adapter accepts only bounded native profile identifiers, rejects unsafe profile/config/ownership files and checks its own ownership record before mutation. Existing/default profiles remain protected. Deletion additionally verifies that the sole configured workspace is the regular directory beneath that profile and that no additional agent entries can redirect workspace cleanup. The fixed official uninstall command owns traversal; YORVA exposes no arbitrary path/delete/shell API.

The child environment excludes inherited provider credentials, YORVA API tokens, Node hooks, OpenClaw auth/path overrides and shell hooks. Commands use a validated absolute Node executable plus fixed direct argv. Captured stdout/stderr is bounded to 256 KiB per stream; sensitive configuration/status buffers are cleared after parsing, and raw output/process errors never reach ordinary API errors. Package metadata is separately bounded to 256 KiB, accommodating the qualified release's 135,311-byte export map while rejecting oversized files. Native `config get gateway --json` is used only to validate binding/auth configuration; token contents are discarded after checking presence. OpenClaw resolves the token internally for probes and stop; no credential appears in argv, YORVA SQLite, capability responses, logs or events. Native OpenClaw owns any native logs/config it writes.

A suspended child is assigned to an adapter-owned Windows Job before execution. Command cancellation kills the owned descendants and drains/joins both readers. Foreground Gateway startup remains kill-on-close until exact authenticated readback; successful transfer clears that flag and releases the process handle without a lingering Go goroutine. Subsequent stop uses the official profile-aware CLI after authenticated target verification. No persisted PID becomes process authority. Stale upstream locks or unverified listeners cause a safe failure; the adapter neither deletes native locks nor kills an arbitrary reported PID to recover.

Windows Job termination is asynchronous. Ordinary command completion and failed Gateway
startup retain synchronization handles for processes enumerated from their owned Job,
terminate that Job, reap/release the direct process, and allow at most five seconds
for the captured process objects to signal exit and Job accounting to reach zero.
The process list is bounded to 1,024 entries; enumeration/cleanup uncertainty prevents
command success. IDs are transient handle lookups, never PID-based kill authority.
Successful authenticated Gateway handoff takes the separate detach path and retains
Runtime lifetime. This does not extend the command execution or readiness deadline.

Closing both child output streams does not disable timeout/cancellation. The command
continues selecting its deadline while waiting for process exit, terminates the owned
Job on cancellation and joins the process waiter before returning.

Accepted Runtime/installation identity is checked before each target dispatch. Unknown/missing OpenClaw state cannot be hidden by successful Hermes recovery, while the healthy Runtime remains queryable. Backend capability checks apply even when a client ignores the UI. No Tauri CSP, local bearer authentication, update verification or production-release signing policy is relaxed by Phase 9.

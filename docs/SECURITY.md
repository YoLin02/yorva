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

Secrets must be stored behind a `SecretStore` abstraction backed by OS-secure storage when available.

Examples of secret scopes:

```text
node/device credential
instance/model provider credential
instance/channel credential
future cloud refresh/session credential
```

Rules:

- SQLite stores only secret references/metadata;
- no silent plaintext fallback;
- ordinary API reads return `configured` metadata only;
- replacement writes accept secret values but never echo them;
- delete actually removes provider-backed material where possible;
- backup does not include secrets unless a future explicit encrypted-secret export feature is designed.

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

Phase 2 Hermes discovery applies this boundary concretely: `yorvad` enumerates only bounded documented candidates, resolves an absolute regular executable, and invokes it directly with the constant argv `--version`. On Windows it may stat only fixed official-layout markers below `%LOCALAPPDATA%\hermes\hermes-agent` to distinguish an incomplete installation from no installation; those markers, the Python wrapper and `hermes-agent.exe` are never executed. It evaluates at most eight candidates, limits stdout and stderr separately to 64 KiB, uses a three-second per-candidate timeout and a ten-second overall application deadline, and owns the full child process tree with a Windows Job Object or Unix process group. Windows children are created suspended, assigned to the kill-on-close Job Object, and only then resumed so descendants cannot escape before ownership is established. Cancellation, timeout, output overflow and normal return terminate any remaining descendants and wait/reap the direct child. The child receives an allowlisted environment containing only OS execution essentials and Hermes location variables; provider credentials and arbitrary inherited values are excluded. Raw command output is not returned through HTTP or Desktop. One structured application-level outcome record contains only Runtime kind, stable state/error code, counts, duration and timeout/cancellation flags; it excludes executable paths, command output, raw errors and environment values.

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

Adapter code must not expose arbitrary read/write file endpoints to Desktop or Cloud.

Normalize and validate paths before mutation. Protect against traversal when accepting relative names.

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

- store in memory only where possible;
- publish only to the authenticated initiating client/session;
- apply expiry;
- clear after success/failure/timeout;
- do not write QR payloads to audit logs;
- do not retain QR images as backups.

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

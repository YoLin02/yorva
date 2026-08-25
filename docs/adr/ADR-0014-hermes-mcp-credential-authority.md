# ADR-0014: Hermes MCP Credential Authority and Closed Descriptor Boundary

- Status: Proposed
- Date: 2026-08-25
- Owner approval: Pending
- Related: ADR-0003, ADR-0006, ADR-0007, ADR-0008, ADR-0012,
  `PHASE-007-hermes-runtime-management-completeness.md`,
  `PHASE-007-B1-QUALIFICATION-TEST-MATRIX.md`

## Context

Phase 7 proposes management of a small approved set of Hermes MCP servers. The Phase Spec
permits approved presets and qualified HTTPS MCP, but forbids caller-controlled stdio
commands, executables, package names, argv, environment, headers, local paths and
secret-bearing URLs. Secrets are write-only, long work is an Operation, and OAuth is in
scope only when it is provably bound to the initiating YORVA session.

The exact inspected Hermes source is version `0.20.5`, commit
`a0ca7c19204e514f9590ce3b812e029b315ab9e9`, archive SHA-256
`4E63F39BD992B79BF364A4E22204A22B8E520E2365B53C909F6F15D42C917C54`.
Static inspection establishes these facts:

- `hermes_cli/mcp_config.py` stores MCP definitions under the selected Profile's
  `config.yaml` `mcp_servers` key. Its general CLI parser accepts URL, command, argv and
  environment values from the caller.
- For a Bearer-authenticated HTTP server, Hermes writes the token to the selected
  Profile's `.env` and stores only an `Authorization: Bearer ${MCP_<NAME>_API_KEY}`
  interpolation template in `config.yaml` (`mcp_config.py:153-196`).
- The general configuration resolver loads Profile `.env` values and interpolates them
  into MCP configuration before connecting (`mcp_config.py:254-275`).
- Hermes OAuth access/refresh tokens, dynamic-client information and server metadata are
  Profile-scoped under `HERMES_HOME/mcp-tokens/` as `<server>.json`,
  `<server>.client.json` and `<server>.meta.json`; the client-info record can contain a
  client secret (`tools/mcp_oauth.py:192-201`, `456-580`). Hermes uses private temporary
  writes for these files and owns refresh/reload behavior.
- The gateway OAuth helper returns a random `session_id` and authorization URL, then
  allows polling by `session_id` and server name. Its source-level registry does not
  establish binding to a YORVA authenticated Desktop session or actor
  (`tui_gateway/mcp_oauth_sessions.py:213-335`). The B1 evidence therefore does not prove
  the Phase 7 initiating-session requirement.
- Hermes catalog entries can declare local commands, argv, environment, Git sources and
  bootstrap strings. Catalog bootstrap runs those strings with `shell=True` and has no
  common process timeout (`hermes_cli/mcp_catalog.py:445-522`). Catalog inclusion is not
  YORVA review or execution authority.
- Direct `mcp add`, `install`, `login`, `test` and `remove` are human/interactive CLI
  workflows with incomplete structured postconditions. Common Hermes CLI startup can
  also perform launcher/update recovery and cleanup before subcommand dispatch.

Hermes must continue using an MCP configuration after YORVA exits. Keeping reusable MCP
credentials in both YORVA and Hermes would create two authorities and a synchronization
problem. Conversely, declaring all Hermes MCP storage authoritative without a closed
descriptor and session boundary would turn `yorvad` into a generic remote execution and
secret-injection service.

This ADR defines the minimum proposed authority. It does not itself qualify or enable an
MCP capability. Its status remains Proposed until the Owner decides the questions below
and the selected surface passes B1 qualification.

## Decision

### 1. Closed descriptor set

YORVA will accept only a `presetId` from a compile-time, adapter-owned and reviewed MCP
descriptor set. A request cannot carry a descriptor or override any descriptor field.

An approved descriptor is exactly one of:

1. a reviewed HTTPS remote MCP with a fixed canonical HTTPS origin and path, fixed
   transport/authentication shape, fixed server identity and fixed tool-selection policy;
2. a reviewed local-process preset whose executable identity and constant argv are owned
   by the Hermes adapter and already present in a sealed reviewed generation.

The local-process form cannot download or install a package and cannot run a bootstrap,
shell, interpreter snippet or package runner. It is unavailable unless the exact
executable identity and full constant argv were separately reviewed and qualified.

The request and public API have no fields for command, executable, argv, environment name
or value, header name or value, URL, path, package, registry, Git reference, bootstrap,
OAuth endpoint, redirect URI, client ID, client secret or tool name outside the preset's
allowlist. Unknown preset IDs fail closed before Hermes or a child process is reached.

For HTTPS descriptors:

- the URL is a descriptor constant, contains no user information, query or fragment, and
  uses an approved public HTTPS origin;
- redirects are disabled unless the descriptor explicitly fixes a same-origin redirect
  policy; redirects never forward credentials to a different origin;
- the authentication header name and interpolation template are constants;
- callers cannot supply arbitrary headers, a secret-bearing URL or TLS/client-certificate
  paths;
- catalog data may be displayed only after projection through YORVA's reviewed descriptor
  set. A live Hermes catalog manifest never becomes executable authority.

YORVA will not use Hermes catalog Git install or any `install.bootstrap` route. In
particular, no catalog shell string is executed, even for an upstream-shipped catalog ID.

### 2. Credential authority by credential class

The authority is decided separately for each class; there is no general
"Hermes-native secrets" exception.

| Credential class | Sole authority | YORVA handling | Proposed support |
| --- | --- | --- | --- |
| No-auth reviewed descriptor | No credential state | Persist only the safe preset selection needed by YORVA management; Hermes remains authoritative for active configuration | Supported after surface qualification |
| Static Bearer/API key for a reviewed HTTPS preset | Exact Profile's canonical Hermes `.env`; `config.yaml` may contain only the descriptor-owned `${MCP_<PRESET>_API_KEY}` template | Accept one write-only request value, pass it only to the bounded Profile writer, then discard it; never copy it | Supported after acceptance and qualification |
| Fixed secret environment value required by a reviewed local-process preset | Exact Profile's canonical Hermes `.env`; the descriptor owns the exact unique key and non-secret interpolation mapping | Same request-lifetime, write-only rule; YORVA never injects it through an ambient or caller-built child environment | Supported only if that local preset is separately reviewed and qualified |
| OAuth access token, refresh token, dynamically registered client information and server metadata | Exact Profile's Hermes `mcp-tokens/<server>.*` state | YORVA never reads, writes, copies, refreshes or returns these values; Hermes performs token exchange and persistence | Capability remains **false** until initiating-session isolation and terminal cleanup are proven |
| Caller-supplied pre-registered OAuth client secret | None approved | Rejected; it must not enter `config.yaml`, SQLite, SecretStore, argv or an ambient environment | Unsupported |
| Provider-account OAuth reused through another Hermes auth subsystem | Not decided by this ADR | Rejected by MCP endpoints; it requires a separate explicit authority and scope decision | Unsupported |
| Arbitrary/custom secret header, cookies, multiple header credentials, query credential or secret URL | None approved | Rejected before adapter invocation | Unsupported |
| mTLS private key, client certificate, CA bundle or credential file path | None approved | Rejected; no caller path/file credential surface exists | Unsupported |
| Catalog, package-manager, Git or registry credential | None approved | Rejected; P7 MCP performs no package/catalog bootstrap | Unsupported |

For the two proposed static-secret classes, Hermes Profile-native `.env` is the sole
authority under the same explicit local-user at-rest tradeoff as ADR-0007/0008. YORVA
does not describe it as an OS credential vault. YORVA stores no duplicate in
`SecretStore`, SQLite, `secret_refs`, Operations, events, logs, diagnostics, typed audit
metadata or Desktop state.

The compatibility write boundary is limited to Hermes `0.20.5` and the exact Profile
resolved from the Instance `nativeId`. The descriptor, not the caller, selects the one
canonical key. The writer must impose file and value bounds, reject unsafe/reparse paths,
preserve unrelated variables, detect an external modification, use a same-directory
private temporary file and atomic replace, remove its temporary file on failure, and
perform a safe configured-status read-back. Unknown versions, layouts, duplicate keys,
ambiguous Profile resolution or uncertain read-back fail closed.

Ordinary responses expose only safe status such as `NOT_CONFIGURED` or `CONFIGURED`.
Secret length, prefix, suffix, hash, raw header, token expiry detail and storage path are
not returned. Delete targets only the descriptor-owned key in the exact Profile. It must
not delete a shared or externally changed value, and failure never guesses that a remote
credential was revoked.

### 3. OAuth initiating-session rule

OAuth is disabled by default and its capability flag is `false` for Hermes `0.20.5` under
the current B1 evidence. A random Hermes flow ID plus server-name check is not proof of
YORVA session ownership.

The capability may become true only after an implementation and independent test prove
all of these properties:

- an authenticated YORVA Desktop session starts the Operation;
- the daemon creates a separate high-entropy public handle and keeps the mapping to the
  Hermes flow, exact Instance/Profile, preset, actor/session and expiration only in
  bounded memory;
- start, poll, callback completion and cancel all require that same initiating YORVA
  session; possession of a Hermes flow ID, server name or Operation ID is insufficient;
- the raw Hermes flow ID, authorization code, OAuth state, access token, refresh token
  and client information are never exposed to another API session, SQLite, events, logs,
  diagnostics or Desktop persistence;
- the authorization URL is bounded, exact-origin validated, delivered only to the
  initiating session and cleared on approval, rejection, cancellation, timeout, daemon
  restart and client-session termination;
- Hermes validates OAuth `state` and persists tokens directly into the exact Profile;
  YORVA does not proxy or copy token responses;
- another authenticated session cannot poll, cancel, complete, reuse or observe the
  flow, including after reconnect or daemon restart;
- there is a bounded pending-flow count, one conflicting flow per exact
  Instance/Profile/preset, and deterministic expiry;
- every terminal outcome closes callback listeners, cancels owned network work, clears
  in-memory flow records and reconciles native token/config state.

If any property cannot be demonstrated by exact-candidate automated tests and Windows
smoke, OAuth remains unavailable. The UI must show the capability as unsupported; it may
not fall back to `hermes mcp login`, a Dashboard flow, a copied authorization URL or a
generic browser callback.

### 4. `CONFIGURED` and `READY` are separate truths

`CONFIGURED` means an authoritative, exact-Profile read-back found the complete reviewed
descriptor and, when required, safe evidence that the descriptor-owned credential/token
state exists. It does not mean the endpoint accepted the credential, a process launched,
tools were discovered or the server is healthy.

`READY` is ephemeral observed state. It may be returned only after a fresh bounded test
for the exact Instance/Profile/preset has:

- connected using the reviewed descriptor;
- completed the expected MCP negotiation;
- obtained a bounded, validated tool inventory consistent with the preset policy;
- observed no credential, transport or protocol failure; and
- completed cleanup/read-back successfully.

`READY` includes an observation timestamp and expires under a fixed policy. A prior
`READY` observation becomes stale after configuration/credential change, external native
change, Runtime/gateway restart, upgrade, Restore, timeout or failed test. Persisted YORVA
metadata is never authoritative over Hermes. `CONFIGURED` without a fresh successful test
is displayed as `CONFIGURED`, not `READY`.

### 5. Operation, timeout, cancellation and postcondition

Install/configure, authenticate, test and remove are typed Operations with the narrowest
lock for the exact Instance/Profile/preset. They conflict with same-server mutation/auth/
test, Restore, Upgrade and any lifecycle transition that would invalidate the test. No
database transaction is held during file, network, browser, Hermes or child-process work.

Each descriptor supplies fixed connect, negotiation and tool-list limits within an
application-owned overall deadline. The adapter also enforces independent stdout,
stderr, response-body, object-count, tool-count, field-length and error-length caps.
Untrusted server/tool text is normalized and redacted before reaching Core/Desktop.

Every goroutine, request, callback listener and child has one Operation owner and a
cancellation path. A reviewed local-process preset runs under the existing YORVA process
containment model: direct fixed argv, minimal allowlisted environment, full process-tree
ownership, kill on timeout/cancel/output overflow, and wait/reap before terminal result.
Normal return also closes any remaining descendants. No generic Hermes CLI process is
used merely to avoid implementing ownership.

A mutation succeeds only after exact-Profile authoritative read-back matches the reviewed
descriptor and safe credential status. A test succeeds only after the fresh `READY`
postcondition above. Exit zero, printed success, process spawn, token-file existence or a
non-empty tool list alone is insufficient. Unknown, malformed, partial, oversized,
externally changed or unbounded results fail closed with stable YORVA error codes and
capability/state truth is reconciled before the Operation terminates.

Cancellation and timeout stop local work but do not claim remote OAuth revocation. Remove
deletes only exact descriptor-owned native configuration/token state through a qualified
surface, then proves absence by read-back. If absence cannot be proven, the Operation
fails and the visible state is `UNKNOWN`, not `NOT_CONFIGURED`.

### 6. Explicitly unsupported Hermes surfaces

This decision does not authorize:

- direct exposure or passthrough of `hermes mcp add`, `install`, `configure`, `login`,
  `reauth`, `test`, `remove`, catalog or Dashboard request bodies;
- the general Hermes `mcp_servers` editor or any caller-provided config fragment;
- arbitrary HTTPS MCP URLs, redirect targets, headers or OAuth parameters;
- arbitrary stdio commands, command discovery through `PATH`, package runners, package
  names, argv, environment, working directories, executable/local paths or shells;
- Hermes catalog Git clone, package install, manifest bootstrap or `shell=True` execution;
- caller-supplied tool definitions or tool selection outside a descriptor allowlist;
- import of Hermes internal Python modules or dependence on an undocumented Hermes state
  database;
- YORVA copying or refreshing Hermes OAuth tokens/client secrets;
- provider-account OAuth, pre-registered confidential OAuth clients, mTLS and custom
  multi-header authentication;
- returning raw server config, command, URL, headers, environment, OAuth data, tool
  output or native paths through ordinary APIs;
- representing `CONFIGURED`, token-file presence, process spawn or exit zero as `READY`.

## Alternatives considered

### Store a second reusable MCP credential copy in YORVA SecretStore

Rejected. Hermes must use MCP after YORVA exits. A second copy creates synchronization,
rotation and injection requirements and makes authority ambiguous. SecretStore remains
the authority for YORVA-owned secrets, not a mirror of Hermes Profile credentials.

### Expose Hermes general MCP CLI or Dashboard APIs behind YORVA authentication

Rejected. Authentication does not make caller-controlled command/argv/env/header/path
safe, does not remove interactive behavior, and does not supply bounded structured
postconditions. Current CLI startup can also mutate installation state.

### Trust every entry in the Hermes catalog

Rejected. Catalog entries can include executable/package/Git/bootstrap behavior, and
bootstrap strings run through a shell. Upstream catalog membership is input for review,
not YORVA execution authority.

### Allow arbitrary HTTPS URLs because they do not start a local process

Rejected. Arbitrary endpoints create SSRF, redirect, credential-forwarding and unbounded
untrusted-output risks. HTTPS alone is not a trust decision.

### Let YORVA own OAuth tokens and inject them as headers

Rejected. This creates a second token authority, moves refresh/revocation behavior into
YORVA and conflicts with Hermes' Profile-native token manager. It would also encourage a
generic header surface.

### Treat Hermes flow IDs as sufficient session isolation

Rejected. B1 source evidence proves randomness and server-name matching, not binding to
the authenticated YORVA session that initiated the flow.

### Mark configured servers ready until a test fails

Rejected. Configuration presence is not proof of network reachability, authentication,
protocol negotiation or safe tool discovery.

## Consequences

Positive consequences:

- Hermes remains the sole authority for the exact MCP credentials/tokens it must use;
- YORVA holds a static secret only during one authorized write and never copies OAuth
  token state;
- the public boundary cannot become a generic process, package, environment, header, URL
  or file API;
- catalog shell/bootstrap execution is excluded;
- Profile isolation, truthful `CONFIGURED`/`READY` state and terminal cleanup are explicit
  acceptance properties.

Costs and constraints:

- only a small reviewed preset set can be supported;
- the static-secret compatibility writer is deliberately coupled to the qualified Hermes
  `0.20.5` Profile layout and must be re-qualified for a changed version/layout;
- Hermes Profile files may provide weaker at-rest protection than an OS vault, and YORVA
  must say so accurately;
- OAuth, confidential clients, mTLS, custom headers and catalog-installed local packages
  remain unavailable unless a later accepted ADR and qualification safely expand them;
- a server can be `CONFIGURED` for extended periods without being `READY`;
- external Hermes edits require reconciliation and may yield `UNKNOWN` or conflict rather
  than an automatic overwrite;
- this Proposed ADR does not unblock B5 or authorize product/API/migration work.

## Owner decisions required before acceptance

1. Approve or reject Hermes Profile `.env` as the sole authority for static MCP secrets,
   including the same explicit local-user at-rest tradeoff accepted for model/channel
   credentials.
2. Approve or reject Hermes `mcp-tokens/<server>.*` as the sole authority for OAuth
   access/refresh tokens and dynamically registered client information, with no YORVA
   token copy.
3. Confirm whether the first P7 MCP release is limited to reviewed HTTPS presets, or may
   also contain a reviewed local-process preset already present in a sealed generation.
   No catalog/package/bootstrap installation is allowed in either choice.
4. Confirm that OAuth remains capability-false until exact-candidate tests prove
   initiating-YORVA-session isolation, cancel/expiry cleanup and Profile-correct
   postconditions; approval of this ADR alone is not that proof.
5. Confirm that pre-registered OAuth client secrets, provider-account OAuth, mTLS,
   arbitrary/custom headers and arbitrary HTTPS endpoints are deferred rather than
   assigned an implicit authority.
6. Confirm that `READY` requires a fresh bounded authoritative test and can never be
   inferred from `CONFIGURED`, token-file presence, process spawn, prior cache or exit
   zero.

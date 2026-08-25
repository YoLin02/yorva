# ADR-0016: Hermes API Server Key Authority for Management Reads

- Status: Accepted
- Date: 2026-08-25
- Owner approval: Accepted 2026-08-25
- Related: ADR-0007, ADR-0012, Phase 7 B3/B4, Phase 7 B1 qualification evidence

## Context

Hermes `0.20.5` exposes two useful structured management reads on a running API-server
platform: authenticated `GET /health/detailed` and authenticated `GET /v1/skills`.
They require `API_SERVER_KEY`. The key is resolved from the exact Profile secret scope,
normally the Profile's canonical `.env`, and the server refuses weak keys at startup.

YORVA currently has no authority to create, rotate or use this Runtime-native secret.
Keeping a second copy would create two authorities. Exposing the key or a caller-selected
endpoint would also turn an ordinary management read into a credential-forwarding or
remote-agent surface. Without a decision, detailed Health and enabled-Skills inventory
must remain capability-false.

## Decision

### Sole authority and lifecycle

The exact Hermes Profile's canonical `API_SERVER_KEY` entry is the sole persistent
authority. YORVA does not copy it into `SecretStore`, SQLite, `secret_refs`, Operations,
events, logs, diagnostics, audit metadata or Desktop state.

For a YORVA-managed Profile, the Hermes adapter may:

- generate a cryptographically random 32-byte key when API-server management is
  explicitly enabled;
- write or rotate only the fixed `API_SERVER_KEY` entry through the existing bounded,
  Profile-exact canonical `.env` compatibility boundary;
- read that exact entry into request-lifetime adapter memory solely to authenticate the
  allowlisted management GETs; and
- discard the value immediately after the bounded request.

The writer must preserve unrelated entries, reject duplicate/ambiguous keys and unsafe
or reparse paths, detect external modification, use a private same-directory temporary
file and atomic replace, and never return any secret-derived metadata. Rotation is a
typed Operation because the affected API-server/gateway may require a bounded restart.

An externally configured valid key may be used internally after exact-Profile resolution,
but it is never returned or copied. An unknown version, ambiguous Profile, malformed
secret store or uncertain read-back leaves the capability false.

### Closed transport

The key may authenticate only these adapter-owned reads for the first P7 release:

```text
GET /health/detailed
GET /v1/skills
```

The target is derived from the live selected, YORVA-managed API-server platform for the
exact Instance/Profile. Requests cannot supply a URL, host, port, path, header or key.
The adapter requires a loopback peer, uses a fixed Bearer header, disables redirects and
applies an overall deadline, response-byte cap, item/string bounds and cancellation.

This authority does not permit chat, run, tool, approval, browser-control, session,
model, cron or any other API-server endpoint. It does not enable remote listening,
arbitrary HTTP proxying or API-server lifecycle mutation through an ordinary read.

### Product truth

API-server liveness remains distinct from detailed Instance health. Detailed Health or
Skills read becomes available only when the exact Profile API-server platform is running,
the key and endpoint pass the closed boundary, and the response passes exact-version
projection and bounds. Unsupported, stopped, unauthorized, malformed, partial and
timeout states remain explicit; liveness is never promoted to full Runtime health.

## Alternatives considered

### Store a duplicate in YORVA SecretStore

Rejected. Hermes must continue authenticating after YORVA exits, and two copies create
rotation and reconciliation ambiguity.

### Ask the Desktop to send the key on every read

Rejected. It moves a Runtime credential into Desktop state and ordinary APIs.

### Reuse the key for arbitrary API-server calls

Rejected. The API server can dispatch agent/tool work. The management authority is a
two-path GET allowlist, not general bearer-token authority.

### Use unauthenticated `/health` for all Health and Skills claims

Rejected. It proves only API-server liveness and exposes no Skills inventory.

## Consequences

Positive:

- one Runtime-native persistent key authority remains explicit;
- YORVA can use official structured reads without exposing or duplicating the key;
- the local management boundary cannot become a generic API-server proxy;
- default and named Profiles remain exact and independently reconcilable.

Costs and constraints:

- the first adapter is exact-version and loopback-only;
- enabling or rotating the key may require a gateway restart Operation;
- only enabled Skills visible to the API server can be claimed;
- a running API-server platform is a prerequisite for these reads.

## Owner decisions required before acceptance

1. Approve Hermes Profile `.env` as the sole persistent authority for
   `API_SERVER_KEY`, with no YORVA duplicate.
2. Approve YORVA generation, internal request-lifetime read and rotation of only the
   fixed key through the bounded compatibility writer.
3. Approve the loopback-only, no-redirect allowlist limited to `/health/detailed` and
   `/v1/skills` for P7.
4. Confirm API-server enable/rotation is a typed Operation and ordinary reads never
   start or restart it implicitly.
5. Confirm all other API-server endpoints and remote targets remain outside this
   management authority.

Until accepted and dynamically qualified on the exact candidate, production Hermes
Health/SkillRead capabilities remain false.

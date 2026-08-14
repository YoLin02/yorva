# ADR-0004: Local HTTP/SSE and Future Outbound WSS

- Status: Accepted
- Date: 2026-08-13

## Context

YORVA Desktop needs a simple local protocol to manage `yorvad`. Future Cloud needs two-way Node communication through NAT/firewalls without requiring customers to expose management ports.

A single WebSocket protocol for everything is possible but adds unnecessary complexity to the local request/response path. Polling alone gives poor long-operation UX.

## Decision

### Local Desktop ↔ Node

Use:

```text
HTTP/JSON over loopback for commands/queries
SSE for daemon → Desktop events
/api/v1 versioning
local authenticated session/bootstrap
```

The daemon binds only to loopback by default.

### Future Node ↔ Control Plane

Use outbound TLS WebSocket initiated by `yorvad`.

Cloud messages are typed/versioned. Remote commands map to the same local application use cases as Desktop requests.

Generic shell or arbitrary filesystem commands are forbidden.

## Alternatives considered

### WebSocket for all local traffic

Rejected initially because local traffic is mostly request/response plus one-way events. HTTP + SSE is simpler to inspect and test.

### Desktop directly invokes daemon through Tauri IPC only

Rejected as the sole protocol because a documented local daemon API improves separation and supports future non-Desktop local clients/testing. Tauri may still own secure daemon bootstrap.

### Cloud connects directly to a Node HTTP port

Rejected because NAT/firewall/public exposure creates poor deployment UX and larger attack surface.

### Message broker

Rejected because it adds infrastructure without a V0.1 need.

## Consequences

Positive:

- simple local debugging;
- good progress UX;
- no inbound Cloud management port;
- one application layer for local and remote actions.

Costs:

- two transport styles exist (local HTTP/SSE and future WSS);
- reconnect/idempotency behavior must be implemented for Cloud;
- local bootstrap/authentication needs careful security design.

# YORVA Protocol

## 1. Scope

This document defines:

1. Desktop ↔ local `yorvad` communication;
2. event and Operation semantics;
3. the future Node ↔ Control Plane protocol boundary.

It does **not** define Hermes' internal protocols. Hermes protocol details belong in the Hermes adapter.

## 2. Design goals

- simple to debug;
- typed and versioned;
- local-first;
- no generic remote shell;
- long-running actions represented as Operations;
- future Cloud reuses application use cases rather than bypassing them.

## 3. Local Desktop ↔ Node transport

Transport:

```text
HTTP/1.1 or HTTP/2 over loopback
JSON request/response
SSE for server → Desktop events
```

Bind only to loopback:

```text
127.0.0.1
```

Do not bind to all interfaces by default.

Base path:

```text
/api/v1
```

### Why HTTP + SSE

Desktop primarily sends request/response commands while the daemon publishes one-way status/progress events. HTTP + SSE keeps the local protocol simpler than a bidirectional socket while still supporting live progress.

If a later local use case genuinely requires high-frequency bidirectional streaming, add it through an ADR rather than replacing the entire API pre-emptively.

## 4. Local bootstrap/authentication

Desktop discovers the daemon endpoint through a Tauri-controlled bootstrap mechanism.

The daemon requires an authenticated local session for all management endpoints except a minimal health/bootstrap endpoint if needed for startup detection.

Requirements:

- credential generated with strong randomness;
- credential held in memory for the Desktop session where possible;
- no credential in URL query strings;
- no secret logging;
- strict local origin/CSP policy;
- token rotation supported on daemon restart/session bootstrap.

Exact bootstrap implementation is finalized during repository initialization and must satisfy `SECURITY.md`.

## 5. HTTP conventions

### Content type

```text
Content-Type: application/json
```

### IDs

Opaque string IDs. Clients must not infer semantics from an ID.

Suggested prefixes are allowed for diagnostics (`node_`, `op_`, `inst_`) but are not a protocol guarantee.

### Time

RFC 3339 UTC timestamps.

### Pagination

Use cursor pagination only for resources that can grow substantially, such as audit/operation history.

Do not add pagination to small static lists prematurely.

### Idempotency

Mutation endpoints that may be retried after uncertain delivery should accept:

```text
Idempotency-Key: <opaque-client-generated-value>
```

The first required use cases are create/install and future Cloud command submission.

## 6. Standard response/error shape

Success returns the resource or action result directly under a typed schema.

Errors:

```json
{
  "error": {
    "code": "RUNTIME_NOT_INSTALLED",
    "message": "Hermes is not installed on this node.",
    "retryable": false,
    "details": {}
  }
}
```

`message` is safe for users. Internal stack traces/command output are not returned by default.

Local routing and authentication errors use the same envelope:

| HTTP status | Stable code | Meaning |
| --- | --- | --- |
| `401` | `UNAUTHORIZED` | Missing or invalid bearer credential on an authenticated resource. |
| `403` | `ORIGIN_NOT_ALLOWED` | A request supplied an Origin outside the Desktop allowlist. |
| `404` | `NOT_FOUND` | The local API route is unknown. |
| `405` | `METHOD_NOT_ALLOWED` | A known resource received a method outside its advertised `Allow` set. |
| `404` | `RUNTIME_KIND_NOT_FOUND` | The requested Runtime kind is not registered for discovery. |
| `500` | `RUNTIME_DISCOVERY_FAILED` | Discovery failed unexpectedly; internal process output is not exposed. |

`OPTIONS` is an unauthenticated CORS preflight only for known paths. An approved Desktop Origin receives `204`; read resources advertise `Allow: GET, OPTIONS` and Runtime discovery advertises `Allow: POST, OPTIONS`. An unknown path receives the structured `404`, and a rejected Origin receives the structured `403` before routing. This does not grant access to the authenticated resource response.

## 7. Core local resources

### Node

```text
GET /api/v1/node
GET /api/v1/health
```

### Runtime discovery

```text
POST /api/v1/runtimes/{runtimeKind}/detect
```

Phase 2 implements authenticated, read-only discovery for the registered `hermes` kind. Phase 3 adds one authenticated installation Operation. The install request accepts no version, URL, path, command or environment fields and requires an `Idempotency-Key`. Completed negative discovery outcomes remain HTTP `200` typed results.

### Instances

```text
GET    /api/v1/runtimes/{runtimeId}/instances
POST   /api/v1/runtimes/{runtimeId}/instances
GET    /api/v1/instances/{instanceId}
DELETE /api/v1/instances/{instanceId}
```

### Lifecycle

```text
POST /api/v1/instances/{instanceId}/start
POST /api/v1/instances/{instanceId}/stop
POST /api/v1/instances/{instanceId}/restart
```

If a Runtime does not expose Instance-level lifecycle, capability metadata must say so and the endpoint returns `CAPABILITY_NOT_SUPPORTED`.

### Configuration

```text
GET   /api/v1/instances/{instanceId}/config
PATCH /api/v1/instances/{instanceId}/config
```

Transport schemas contain normalized, safe configuration data. Secret values are write-only or represented by `configured: true/false` metadata.

### Secrets

Prefer resource-specific secret mutation endpoints rather than a generic secret dump API.

Example:

```text
PUT    /api/v1/instances/{instanceId}/credentials/{credentialName}
DELETE /api/v1/instances/{instanceId}/credentials/{credentialName}
```

GET returns metadata only:

```json
{
  "name": "openai_api_key",
  "configured": true,
  "updatedAt": "..."
}
```

### Channels

```text
GET  /api/v1/instances/{instanceId}/channels
POST /api/v1/instances/{instanceId}/channels/{channel}/connect
POST /api/v1/instances/{instanceId}/channels/{channel}/disconnect
GET  /api/v1/instances/{instanceId}/channels/{channel}
```

QR/login flows return an Operation. QR payloads/events must have short validity and must not be persisted longer than needed.

### Runtime installation

```text
POST /api/v1/runtimes/hermes/install
```

Phase 3 starts one durable `runtime.install` Operation for Windows user-scope Hermes `0.20.2`. The request body is a closed empty object.

### Operations

```text
GET  /api/v1/operations/{operationId}
GET  /api/v1/operations?targetType=...&targetId=...&limit=...
POST /api/v1/operations/{operationId}/cancel
```

Cancel may return `OPERATION_NOT_CANCELLABLE`.

### Backups

```text
POST /api/v1/instances/{instanceId}/backups
GET  /api/v1/instances/{instanceId}/backups
POST /api/v1/backups/{backupId}/restore
DELETE /api/v1/backups/{backupId}
```

## 8. Capability response

Runtime and Instance views include normalized capabilities.

Example:

```json
{
  "runtimeKind": "hermes",
  "version": "...",
  "supported": true,
  "capabilities": {
    "instances": true,
    "lifecycle": true,
    "models": true,
    "skills": true,
    "mcp": true,
    "backup": true,
    "upgrade": true,
    "channels": ["weixin", "wecom"]
  },
  "warnings": []
}
```

## 9. SSE event stream

Endpoint:

```text
GET /api/v1/events
```

SSE event fields:

```text
id: monotonically useful event id or opaque cursor
event: operation.progress
data: {...json...}
```

Core event types:

```text
node.status.changed
runtime.detected
runtime.status.changed
instance.created
instance.deleted
instance.status.changed
channel.state.changed
channel.qr.ready
operation.started
operation.progress
operation.completed
operation.failed
```

Events are notifications, not the sole source of truth. After reconnect, Desktop refreshes relevant resources.

## 10. Operation schema

```json
{
  "id": "op_xxx",
  "type": "channel.connect",
  "status": "RUNNING",
  "target": {
    "type": "instance",
    "id": "inst_xxx"
  },
  "progress": 40,
  "stage": "waiting_for_scan",
  "message": "Waiting for WeChat confirmation.",
  "error": null,
  "createdAt": "...",
  "startedAt": "...",
  "completedAt": null
}
```

Progress is advisory and may be omitted for operations without meaningful percentage progress. `stage` is more important than fake precision.

## 11. OpenAPI

`api/openapi.yaml` is the source of truth for Desktop ↔ Node HTTP schemas.

Preferred workflow:

```text
OpenAPI
  ├── validate in CI
  └── generate TypeScript client/types for Desktop
```

Go domain entities must not be generated directly from transport schemas. HTTP DTOs map into application/domain types.

## 12. Future Node ↔ Control Plane

Transport:

```text
TLS WebSocket (WSS), initiated by yorvad
```

The Node does not expose a public management listener for Cloud control.

The connection is used for:

- heartbeat/status;
- inventory summary;
- typed command delivery;
- operation progress/results;
- optional log tail requests with policy controls.

Large artifacts/backups should use explicit upload/download flows rather than being tunneled blindly through command frames.

## 13. Cloud envelope

Versioned envelope:

```json
{
  "protocolVersion": 1,
  "messageId": "msg_xxx",
  "type": "command.request",
  "sentAt": "...",
  "payload": {}
}
```

Initial message classes:

```text
node.hello
node.heartbeat
node.inventory
command.request
command.accepted
command.rejected
operation.update
operation.result
```

## 14. Typed Cloud commands

A Cloud command maps to a local application use case.

Example:

```json
{
  "protocolVersion": 1,
  "messageId": "msg_1",
  "type": "command.request",
  "payload": {
    "commandId": "cmd_1",
    "commandType": "instance.restart",
    "targetId": "inst_123",
    "parameters": {}
  }
}
```

Forbidden command types include arbitrary shell execution or unrestricted filesystem mutation.

## 15. Cloud delivery semantics

Target semantics:

- at-least-once delivery may occur after reconnect;
- commands therefore carry stable `commandId` values;
- Node records completed/rejected command IDs for a bounded retention period;
- duplicate delivery returns the prior result when safe;
- operations remain idempotent where the business action permits it.

Do not promise exactly-once distributed execution.

## 16. Versioning

Local HTTP contract is versioned in the path (`/api/v1`).

Cloud envelope uses `protocolVersion`.

Breaking changes require a protocol version change and migration/compatibility plan.

Runtime adapter version compatibility is separate from YORVA protocol compatibility.

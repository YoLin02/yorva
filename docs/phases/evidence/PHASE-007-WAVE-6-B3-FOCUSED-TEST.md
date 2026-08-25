# Phase 7 Wave 6 B3 — Independent Focused Test Evidence

## Role, candidate and result

- Role: independent Wave 6 B3 focused-test agent; not the implementation agent and not
  the B10 phase auditor.
- Date: 2026-08-25 (Asia/Shanghai).
- Worktree: `D:\workcode\myproject-phase65-freeze` only.
- Branch: `codex/phase7-hermes-runtime-management`.
- Inspected HEAD: `bfdd7c8c29ee37f34585792baf5f4c260c659695` plus the uncommitted
  Wave 6 B3 protocol/HTTP/OpenAPI changes.
- Scope: authenticated exact-Instance Health and bounded Logs reads, route/method/query
  closure, capability-false behavior, deliberately unregistered Runtime/Deep/Security/
  mutation surfaces, closed OpenAPI DTOs, generated-schema determinism, focused Go
  tests/vet and Desktop typecheck.
- Concurrent Desktop Health/Logs client/UI changes belong to Wave 7. They were not
  attributed to Wave 6, although the final shared-tree typecheck necessarily included
  them.

**Focused result after R1: PASS; THE ORIGINAL LANE-LOCAL P2 IS RESOLVED.**

No P0/P1 secret disclosure, unauthenticated read, arbitrary execution, cross-Instance
mutation, path exposure or raw log-source export was found. Security Audit and Deep/Live
remain Operations by the amended Phase Spec and Protocol; their unregistered synchronous
routes are confirmed expected behavior, not findings.

## Commands and results

From `services/node`:

```text
go test ./internal/app ./internal/transport/httpapi \
  ./internal/runtime/hermes/managementhealth -count=1

PASS
app:                         17.323s
transport/httpapi:            0.169s
hermes/managementhealth:      0.018s
```

```text
go vet ./internal/app ./internal/transport/httpapi \
  ./internal/runtime/hermes/managementhealth

PASS (no output)
```

The route-focused regression subset was also run explicitly:

```text
go test ./internal/transport/httpapi \
  -run 'TestPhase7(ReadOnlyManagementRoutesAreAuthenticatedAndTargetExactInstance|ReadOnlyManagementRoutesKeepCapabilityFalseStable|ReadOnlyManagementRouteContractIsGetOnly|InstanceLogRouteRejectsNonClosedQuery|SecurityAuditReadRemainsUnregistered|RuntimeHealthRemainsUnregisteredWithoutRuntimeTarget)' \
  -count=1 -v

PASS — all selected tests/subtests; package 0.013s
```

From the repository root:

```text
pnpm api:lint

PASS — api/openapi.yaml validated in 35ms
```

```text
$before = SHA256(apps/desktop/src/api/generated/schema.ts)
pnpm api:generate
$after = SHA256(apps/desktop/src/api/generated/schema.ts)

PASS — deterministic/no drift
before: 1A4CAB4F35E699F1157EEEAF69E2673A00D2D24C77195765B372EB38596D2C5A
after:  1A4CAB4F35E699F1157EEEAF69E2673A00D2D24C77195765B372EB38596D2C5A
```

```text
pnpm --filter @yorva/desktop typecheck

PASS — tsc -b --pretty false
```

The final typecheck was repeated after the concurrent Wave 7 Desktop tree became stable.
Wave 7's own client/UI tests and lint are outside this Wave 6 evidence.

Static route/DTO assertions returned:

```text
PASS: exact Instance GET routes present; Runtime/deep/security/mutation routes absent;
health/log DTO tags closed and bounds present
```

The assertion required the two Instance GET registrations, rejected any registered
Runtime health, Runtime deep-check, Runtime security-audit, synchronous Instance security
audit, or Health/Logs mutation route, and limited the registered Health/Logs JSON tags to
normalized state/finding/log content and timestamps. `git diff --check` found no Wave 6
whitespace error; Git emitted only an LF-to-CRLF warning for the concurrent Wave 7
Desktop stylesheet.

## Confirmed boundaries

### Authentication, target and capability

- `GET /api/v1/instances/{instanceId}/health` and
  `GET /api/v1/instances/{instanceId}/logs?category=...` are behind the existing bearer
  guard. Unauthenticated requests return `401`; authenticated qualified-fake requests
  return typed `200` responses.
- Both routes resolve the exact YORVA Instance ID through `ManagementTargetResolver`.
  The application passes the live detected installation and resolved native Instance ID
  to `InspectInstanceHealth` / `ReadLogSnapshot`; it does not call Runtime-wide health.
- The shared resolver re-runs live Runtime discovery, requires the live selected path to
  equal the accepted installation path, and passes the live detected version rather than
  trusting a stale stored version.
- Nil `Health` or `Logs` capabilities return HTTP `409` with
  `CAPABILITY_NOT_SUPPORTED`. The production Hermes bundle still wires neither
  capability, so the new product routes remain capability-false rather than falling back
  to raw Hermes CLI output.

### Query and method closure

- The log request requires exactly one `category` value and accepts only
  `RUNTIME`, `ERRORS`, `GATEWAY`, or `MCP`.
- Missing, unknown/path-shaped, repeated and additional query fields return
  `400 INVALID_REQUEST` without calling the management service.
- Health and Logs route contracts are `GET, OPTIONS` only. OPTIONS returns the closed
  allowlist; POST and other methods return `405` rather than reaching a handler.

### Deliberately unregistered surfaces

- Runtime-wide Health remains `404`; no Runtime target is inferred from an Instance
  resolver.
- Synchronous Instance Security Audit remains `404`.
- Runtime Deep/Live checks and Runtime Security Audit remain unregistered Operation
  candidates and therefore `404` today.
- No repair, audit, deep-check, log export, arbitrary filter/path, or other B3 mutation
  route was added.

These points match the amended English/Chinese Phase Spec and `docs/PROTOCOL.md`; they are
expected closure, not missing-feature defects.

### DTO, bounds and OpenAPI

- Health response fields are limited to normalized state, bounded finding code/state,
  partial flag and observation time. There is no native ID, installation identity, path,
  PID, command, environment, raw adapter payload or secret field.
- Logs response fields are limited to the requested normalized category, at most 256
  timestamp/message entries, truncation state and observation time. Each normalized
  message is bounded by the Runtime contract to 4096 bytes and the complete snapshot to
  64 KiB; no log source/path or raw export field exists.
- Runtime/application validation rejects malformed state, identifier, category,
  timestamp, item count, message size and total message bytes before transport mapping.
- All four new OpenAPI objects use `additionalProperties: false`; enums and collection/
  string limits match the corresponding Go contracts. Generated paths expose only GET
  and OPTIONS, with other methods typed as `never`.
- API lint passed and regeneration reproduced the current generated Desktop schema
  byte-for-byte.

## Finding

### P2 — RESOLVED after R1 — Log response category was not bound to the requested category

Owner lane: B3 application/HTTP integration.

`ManagementHealth.GetInstanceLogSnapshot` validates that the request category is
allowlisted, invokes the adapter with that category, and then calls only
`result.Validate()`. `LogSnapshot.Validate()` proves the returned category is one of the
four allowed values, but neither layer requires `result.Category == category`.

Consequently an adapter defect could receive `ERRORS`, return a valid bounded/redacted
`MCP` snapshot, and have the HTTP route emit `200` with the wrong category. This does not
currently expose product data because the production Hermes `Logs` capability remains
nil and returns `409`; it is therefore lane-local P2 rather than a P0/P1 active leak.

Before wiring a real Logs adapter, require exact response-category equality in the
application postcondition and add a regression proving a mismatched allowlisted category
fails closed without returning entries.

#### RESOLVED / R1 focused recheck — 2026-08-25

The application postcondition now requires both a valid snapshot and
`result.Category == category`. A valid but mismatched allowlisted result returns the
stable `ErrManagementQueryFailed` sentinel and no snapshot. The existing HTTP mapping
converts that sentinel to retryable `MANAGEMENT_QUERY_FAILED` / HTTP `503`; it does not
return adapter contents or raw error details.

The production route constructs `app.NewManagementHealth(targets)` and reaches Logs only
through `GetInstanceLogSnapshot`. No alternate HTTP handler, Runtime-wide route or direct
adapter path bypasses the equality check. The original P2 text above is retained as the
historical pre-fix fact.

Fresh R1 commands from `services/node`:

```text
go test ./internal/app \
  -run 'TestManagementHealth(RejectsMismatchedLogCategory|QueriesRejectMalformedAdapterResults|QueriesNormalizeResolverAndAdapterErrors)' \
  -count=1 -v

PASS — selected tests/subtests; package 0.012s
```

```text
go test ./internal/transport/httpapi \
  -run 'Test(ManagementLogHandlerRequiresOneAllowlistedCategory|ManagementHealthHandlerMapsStableSafeErrors|Phase7InstanceLogRouteRejectsNonClosedQuery|Phase7ReadOnlyManagementRoutesAreAuthenticatedAndTargetExactInstance)' \
  -count=1 -v

PASS — selected tests/subtests; package 0.013s
```

```text
go vet ./internal/app ./internal/transport/httpapi

PASS (no output)
```

R1 was limited to the original category-binding finding. No wider Phase 7 audit or
unrelated reclassification was performed.

## Severity summary

- P0: none.
- P1: none.
- P2: original response-category postcondition finding RESOLVED by R1.

## Not run

- Full project CI, whole-repository Go tests/race and the B10 immutable-candidate Gate.
- Full Desktop tests/lint; Wave 7 owns its client/UI verification.
- MSI/package build, Rust/Tauri tests and packaging smoke.
- Real Hermes, network, raw-log acquisition, Deep/Live or Security Audit Operations.
- Windows exact-candidate/manual smoke, daemon-restart recovery and full independent
  Phase 7 audit.

These checks are outside this focused wave test and are not inferred as PASS.

## Files and state changed by this test

Only this evidence file was created. The API generator reproduced the existing generated
schema exactly (identical SHA-256), so it introduced no product diff. No product source,
test source, dependency, Runtime state, log state, Operation, commit, push, reset, stash,
deletion or original developer worktree content was changed.

# Phase 7 Wave 4 — Independent Focused Test Evidence

## Role, candidate and result

- Role: independent Wave 4 focused-test agent; not an implementation agent and not the
  B10 phase auditor.
- Date: 2026-08-25 (Asia/Shanghai).
- Worktree: `D:\workcode\myproject-phase65-freeze` only.
- Branch: `codex/phase7-hermes-runtime-management`.
- Inspected HEAD: `26b5dfbd868f5cd642baa0b439f4692bd7fa88ad` plus the uncommitted
  Wave 4 application/HTTP/OpenAPI changes.
- Scope: shared live management target resolution; B6 Backup application/HTTP read and
  worker boundaries; B8 Upgrade application/HTTP plan and worker boundaries; B4/B5
  safe GET route/OpenAPI wiring; and the requested Desktop typecheck.
- Later concurrent Desktop client/UI edits were Wave 5. They were not attributed to
  Wave 4, although the final shared-tree typecheck necessarily included them.

**Focused result after R1: PASS; B3 P2 REMAINS OPEN; B8 DEADLINE P2 IS RESOLVED.**

There is no P0/P1 secret disclosure, arbitrary execution, cross-scope write, archive
destruction, active-generation mutation, or false destructive success in the tested
Wave 4 surface. The remaining open P2 stays with its owning integration lane and does
not block unrelated Phase 7 parallel work. This evidence is not a full Phase 7 Gate or
an authorization to wire Backup/Upgrade mutation capabilities.

## Commands and results

From `services/node` on the final stable shared-tree snapshot:

```text
go test ./internal/app ./internal/transport/httpapi \
  ./internal/runtime/hermes/backupmanagement \
  ./internal/runtime/hermes/upgrademanagement -count=1

PASS
app:               17.401s
transport/httpapi:  0.175s
backupmanagement:   0.013s
upgrademanagement:  0.011s
```

```text
go vet ./internal/app ./internal/transport/httpapi \
  ./internal/runtime/hermes/backupmanagement \
  ./internal/runtime/hermes/upgrademanagement

PASS (no output)
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
before: 557CA6D4CEB518F6CC0F990215EDB0929CD7DC006197A9B7D6185909DF6D4A15
after:  557CA6D4CEB518F6CC0F990215EDB0929CD7DC006197A9B7D6185909DF6D4A15
```

```text
pnpm --filter @yorva/desktop typecheck

PASS — tsc -b --pretty false
```

The concurrent Wave 5 client/UI files were also checked while they became visible, but
are not part of this Wave 4 verdict:

```text
pnpm --filter @yorva/desktop test -- \
  src/api/client.management.test.ts \
  src/components/management/ManagementPanel.test.tsx

PASS — 2 files, 6 tests
```

Static closed-surface assertions checked `server.go` and the four management HTTP DTO
files for the tested Phase 7 surface:

```text
PASS: no Phase 7 mutation route and no forbidden DTO JSON tag
```

The forbidden DTO tag set included path, command, argv/arguments, environment/env,
header, URL, secret/credential/key reference, destination reference, native/installation
identity and protection-point identity. `git diff --check` returned success; Git emitted
only an LF-to-CRLF warning for the concurrent Wave 5 `apps/desktop/src/styles.css` file.

## Confirmed Wave 4 boundaries

### Live target and scope

- Instance management resolves the YORVA Instance row and accepted installation, then
  performs live Runtime discovery. The selected live path must equal the accepted path;
  the adapter receives the live detected version instead of the stored accepted version.
- Runtime management performs live discovery first and then requires an accepted
  installation for the exact live node/kind/path. A stale database-only path cannot
  select a Backup or Upgrade target.
- Skills and MCP remain Instance-scoped and receive only the resolved native Instance ID.
  Backup and Upgrade use the separate Runtime-scoped resolver and never accept an
  Instance ID as their scope.
- Resolver, adapter and request cancellation errors are normalized without returning
  raw database paths, artifact paths, key references or adapter errors.

### Capability and routing closure

- Nil `SkillRead`, `MCPRead`, `BackupRead` and `UpgradePlan` capabilities fail closed.
  The registered B4/B5 GET routes return HTTP `409` with
  `CAPABILITY_NOT_SUPPORTED` when the applicable capability is nil.
- The production Hermes bundle still wires none of Health/Logs/Security, Skill/MCP,
  Backup/Restore or Upgrade/Rollback management contracts. All affected product
  capability flags therefore remain false.
- Wave 4 registers only authenticated B4/B5 reads:
  `GET /instances/{instanceId}/skills`, Skill inspection, configured MCP inventory and
  reviewed MCP preset inventory. Their route contract is `GET, OPTIONS`; POST/PUT/PATCH/
  DELETE attempts receive `405`.
- Backup and Upgrade HTTP handlers are intentionally unregistered. No Backup
  create/delete mutation route and no Upgrade/Rollback mutation route exists; even the
  read-only Backup list/inspect/verify and Upgrade-plan handlers are not product routes.

### Backup result handling

- Backup list output is bounded to 256 unique IDs and rejects malformed entries and
  artifacts larger than 2 GiB.
- Available backup metadata requires a closed format/runtime-version grammar, positive
  size, SHA-256 and timestamp. Non-available states do not project format, Runtime
  version, checksum or creation time from untrusted/stale adapter fields.
- Inspect uses the live Runtime-scoped list instead of a SQLite backup row. Verify
  requires the exact requested backup identity, `AVAILABLE` state and a valid complete
  result before setting `artifactVerified=true`.
- Duplicate IDs, malformed/oversized metadata, wrong verification identity, failed or
  malformed verification result, unsafe backup ID and raw adapter/tamper errors fail
  closed with stable safe errors.
- Create/Delete remain worker-only boundaries, validate opaque closed IDs before target
  resolution, require `BackupMutate`, and expose no HTTP mutation route.

### Upgrade plan and execution gates

- A plan is accepted only for the exact live Runtime kind/path/version and matching
  bundle descriptor. `CurrentVersion` must equal the live installation version.
- `AVAILABLE` must also be executable: managed installation, complete inventory,
  rollback eligible, and a ready protection point whenever required. Invalid state,
  current-version mismatch, incomplete inventory, missing protection point and unsafe
  rollback evidence all fail closed.
- Upgrade and rollback workers are not HTTP handlers. They require separate compile-time
  mutation capability wiring, re-plan immediately before mutation, refuse blocked plans,
  and validate the typed result and timestamp.
- The Upgrade plan DTO contains only closed state/version/eligibility booleans and an
  observation time. It has no filesystem path, command, argv, seal/hash,
  installation ID or protection-point ID.

### OpenAPI and generated types

- The four registered B4/B5 GET paths, authentication, `GET, OPTIONS` method surface,
  `409`/`503` errors, collection limits and closed Skill/MCP response properties match
  the Go handlers.
- Skill/MCP schemas contain no command, environment, header, URL, local path, credential
  or secret property. MCP `CONFIGURED` remains distinct from timestamped `READY`.
- OpenAPI lint passed and regeneration reproduced the checked-in generated Desktop schema
  byte-for-byte.

## Findings

### P2 — OPEN — B3 Health/Logs/Security handlers are not in the safe GET/OpenAPI integration

Owner lane: B3 HTTP/OpenAPI integration.

`services/node/internal/transport/httpapi/management_health.go` already contains the
Instance health, bounded log snapshot and security-audit read handlers. Wave 4
`NewHandler` constructs and registers only the B4 Skills and B5 MCP services/routes, and
`api/openapi.yaml` likewise adds only those four paths. Consequently authenticated B3
Instance management reads still return `404` and have no generated Desktop contract.

This is a functional integration gap, not a security opening: the underlying Hermes
Health/Logs/Security capabilities remain nil, there is no accidental Runtime-scoped
health route, and no mutation surface was exposed. The B3 lane must add authenticated
Instance GET routes, `GET, OPTIONS` route-contract entries, closed OpenAPI schemas and
nil-capability `409` tests before claiming its read integration complete.

### P2 — RESOLVED after R1 — Upgrade HTTP deadline expiry was normalized as generic 500

Owner lane: B8 application/HTTP integration.

`ManagementUpgrade.PlanUpgrade` preserves an expired request context as
`context.DeadlineExceeded`, but `writeManagementUpgradeError` handles cancellation and
the application sentinel only; it has no deadline case. If the currently unregistered
plan handler is wired later, a real deadline expiry will therefore fall through to
`INTERNAL_ERROR` / HTTP `500` instead of the stable retryable
`MANAGEMENT_QUERY_FAILED` / HTTP `503` used by the other management reads.

This is pre-wiring and non-secret/non-mutating, so it is lane-local P2. Add the deadline
mapping and a focused handler regression before registering the Upgrade plan route.

#### RESOLVED / R1 focused recheck — 2026-08-25

The handler now receives the request when normalizing errors. It suppresses output only
for cancellation owned by the actual request context; `context.DeadlineExceeded` and a
non-request-owned `context.Canceled` both map to retryable
`MANAGEMENT_QUERY_FAILED` / HTTP `503`. The original finding above is retained as the
historical pre-fix record.

Fresh R1 commands from `services/node`:

```text
go test ./internal/transport/httpapi -run TestManagementUpgradeHTTP -count=1

PASS — transport/httpapi 0.013s
```

```text
go vet ./internal/transport/httpapi

PASS (no output)
```

The regression matrix now covers deadline and non-request-owned cancellation in addition
to query failure, unsupported Runtime, capability-false and sanitized raw-internal-error
behavior. B3 route/OpenAPI integration remains OPEN and was not changed by this R1.

## Not run

- Full project CI and the B10 immutable-candidate Gate.
- Whole-repository Go tests, race tests and full Desktop test/lint suites.
- MSI/package build, Rust/Tauri tests and packaging smoke.
- Real Hermes, network, Backup artifact creation/deletion, Restore, Upgrade or Rollback
  mutation smoke.
- Destructive-flow, daemon-restart recovery and Windows exact-candidate/manual smoke.
- Full independent Phase 7 audit.

These are intentionally outside this focused wave test and are not inferred as PASS.

## Files and state changed by this test

Only this evidence file was created. The API generator reproduced the existing generated
schema exactly (identical SHA-256), so it introduced no product diff. No product source,
test source, dependency, Runtime state, backup, Restore, upgrade, commit, push, reset,
stash, deletion or original developer worktree content was changed.

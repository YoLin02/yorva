# Phase 7 Wave 10 — B3/B4 Authenticated API Management Reads

## Result

**FOCUSED GO PASS / DESKTOP INTEGRATION FAIL PRESERVED — 2026-08-25**

This lane implements only the ADR-0016 read boundary for exact Hermes `0.20.5`:

- authenticated detailed Instance health from `/health/detailed`;
- enabled-only Skill inventory from `/v1/skills`;
- default, independent named-Profile and default-listener multiplex targeting; and
- per-Installation/Profile capability qualification instead of version-wide static
  capability publication.

It does not implement API-server key enable/rotation, listener lifecycle mutation,
logs, security audit, Skill mutation, or any other API-server route.

## Exact-source qualification

The implementation was checked against the already verified Hermes source snapshot:

- version: `0.20.5`;
- commit: `a0ca7c19204e514f9590ce3b812e029b315ab9e9`;
- archive SHA-256:
  `4E63F39BD992B79BF364A4E22204A22B8E520E2365B53C909F6F15D42C917C54`.

Relevant exact-source facts used by the adapter are:

- the default listener is `127.0.0.1:8642`;
- `API_SERVER_HOST` and `API_SERVER_PORT` override the fixed listener fields;
- `gateway.api_server`, `gateway.platforms.api_server`, and
  `platforms.api_server` are the supported fixed configuration locations;
- default multiplex mode owns the shared listener and named Profiles use
  `/p/<profile>/...` with the named Profile's own `API_SERVER_KEY`;
- a multiplex secondary Profile with its own API-server listener enabled is skipped;
- `/health/detailed` and `/v1/skills` require the fixed Bearer key.

The production adapter reads only those fixed fields. YAML aliases, anchors, duplicate
keys, multiple documents, unsupported tags, oversized files, malformed values,
non-loopback hosts, invalid ports, ambiguous `.env` assignments and any YAML-stored API
server key fail closed. No caller supplies a host, port, path, header, key, Profile path,
or environment name.

## Capability truth

The compile-time Hermes Bundle does not statically advertise Health or Skill reads.
An Instance-scoped resolver adds those readers only after exact `0.20.5` and the exact
Profile configuration qualify. This prevents compatible `0.20.2` discovery from
incorrectly publishing the exact-`0.20.5` management surface.

The existing protocol's capability boolean means that the management surface is safely
available for that Instance. A stopped listener is a current query failure; an ordinary
read never starts or restarts it. This is consistent with other capabilities whose
Runtime can be temporarily stopped. HTTP connection, authentication, timeout, malformed
body and unavailable-listener failures remain the stable `MANAGEMENT_QUERY_FAILED`
public envelope and do not expose internal URL, Profile path, response body or key.

## Security and projection checks

Focused tests cover:

- exact `0.20.5` gating and `0.20.2` fail-closed behavior;
- default and named independent Profile endpoint isolation;
- multiplex listener endpoint plus named Profile key/path isolation;
- multiplex allowlist and secondary-listener rejection;
- non-loopback host, YAML key authority, duplicate `.env` key, duplicate YAML field,
  alias/anchor and malformed configuration rejection;
- no proxy, no redirect, fixed two-path GET allowlist and loopback peer verification;
- fixed Bearer authentication, bounded body, content type, cancellation and timeout;
- strict health/Skill parsing and safe Runtime-neutral projection;
- no projection of PID, account/platform detail, path, source/version/update claims or
  scan verdict not proven by the enabled-only endpoint.

No key, account identity, Profile filesystem path or raw detailed-health body is written
to SQLite, Operations, logs, diagnostics, DTOs or this evidence.

## Focused verification

From `services/node`:

```text
go test ./internal/runtime/hermes ./internal/runtime/hermes/managementhealth \
  ./internal/runtime/hermes/skillsmanagement ./internal/runtime ./internal/app \
  ./internal/transport/httpapi -count=1

PASS
```

```text
go vet ./internal/runtime/hermes ./internal/runtime/hermes/managementhealth \
  ./internal/runtime/hermes/skillsmanagement ./internal/runtime ./internal/app \
  ./internal/transport/httpapi

PASS
```

```text
go mod tidy -diff

PASS — no diff after the direct YAML dependency was normalized
```

From the repository root:

```text
pnpm typecheck
pnpm lint

PASS / PASS
```

```text
pnpm api:lint
pnpm exec openapi-typescript api/openapi.yaml -o <temporary-file>
compare temporary output with apps/desktop/src/api/generated/schema.ts

PASS / generated output exact match
```

The first integrated Desktop test run is preserved as a real failure:

```text
pnpm test

FAIL — 23/24 files and 115/116 tests passed.
InstancesPage.test.tsx:402 expected three repeated unavailable messages after the
concurrent Upgrade-plan UI added a fourth management section.
```

This is not a B3/B4 production defect. The shared UI integration owner must update that
stale count assertion and run an R1 Desktop test; this lane did not edit the concurrently
owned file or relabel the failure as PASS.

The integration owner updated only that stale count from three to four and reran the
Desktop suite from a fresh process:

```text
pnpm test

PASS — 24/24 files and 116/116 tests passed.
```

The initial FAIL remains preserved above; this R1 resolves only the shared UI assertion.

## Evidence still required at B10

No real key or account detail was requested or recorded. An exact-candidate Windows
smoke must still prove, without disclosing the key:

- default Profile health and enabled-Skill read;
- named independent or multiplex Profile read, matching the Owner's actual mode;
- wrong-Profile authentication rejection;
- stopped-listener/unavailable presentation; and
- no implicit start, restart or configuration write.

Until that smoke, this focused lane is not final B10 freeze evidence.

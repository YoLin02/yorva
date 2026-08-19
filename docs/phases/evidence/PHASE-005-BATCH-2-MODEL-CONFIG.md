# Phase 5 Batch 2 — Model Configuration Evidence

- Date: 2026-08-19
- Baseline: `phase-004-instance-profile-baseline`
- Pinned Hermes surface: `0.20.2`
- Gate: PASS

## Delivered contract

- The Runtime bundle owns a focused model configuration capability. Provider
  mapping and native config keys remain private to `runtimes/hermes`.
- Eight qualified static presets expose only product id, display name, region,
  recommendations and safe help text.
- Read and apply use the official Profile-scoped `config get --json` and
  `config set` scalar commands, followed by authoritative read-back.
- Manual model ids are bounded to 200 characters and cannot be URLs, paths,
  config keys or shell fragments.
- Apply requires confirmed credential presence, compares the native provider
  and model twice before mutation, and returns safe observed state for conflict
  or partial apply.
- The application resolves public `instanceId` to adapter-only `nativeId`,
  rediscovers the exact accepted installation, reconciles authoritative Profile
  presence and blocks `MISSING`, `UNKNOWN`, and active Profile mutations.
- Authenticated GET/PATCH routes use the existing loopback route contract,
  no-store response headers, closed JSON, stable errors and generated OpenAPI
  types. CORS now advertises the Phase 5 PATCH/PUT methods.

## Verification

```text
pnpm api:lint                         PASS
pnpm api:generate                     PASS
go test ./...                         PASS
go vet ./...                          PASS
go build ./...                        PASS
git diff --check                      PASS
```

Focused tests cover exact argv, no-secret argv, safe normalization, input
bounds, read-back, external-change conflicts, partial apply, stable public error
mapping, authentication, closed bodies, CORS, and `instanceId`/`nativeId`
separation.

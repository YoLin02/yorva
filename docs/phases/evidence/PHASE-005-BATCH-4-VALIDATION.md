# Phase 5 Batch 4 — Explicit Model Validation Evidence

- Date: 2026-08-19
- Operation type: `model.validate`
- Gate: PASS

## Delivered contract

- Only the authenticated explicit POST route creates validation work. It
  accepts a closed empty object and requires an idempotency key.
- The durable Operation targets the stable public `instanceId`; adapter calls
  receive only the resolved Profile `nativeId`.
- Each worker re-resolves the supported installation, authoritative Profile,
  provider/model config and credential status before network access.
- A non-secret SHA-256 config fingerprint prevents a completed result from
  appearing to validate a subsequently changed provider/model configuration.
- The pinned invocation uses the selected Profile, qualified provider/model,
  the fixed harmless prompt and the built-in empty `context_engine` toolset.
  The adapter first requires `context.engine == compressor` and fails closed for
  plugin engines.
- One 45-second outer deadline covers authoritative preflight plus validation.
  The existing runner provides bounded independent streams, caller
  cancellation, Windows Job Object / Unix process-group ownership and
  descendant cleanup.
- Model stdout and raw provider stderr are discarded. Operations, events and
  HTTP contain only closed identifiers, state and stable error codes.
- Provider rejection maps to `FAILED`; timeout, cancellation, unsafe context,
  output overflow, transport ambiguity and restart interruption map to
  `UNKNOWN`. `CONFIGURED`, validation state and Instance availability remain
  separate.
- Active validation can be cancelled through the existing Operation cancel
  route. Stale PENDING/RUNNING validation Operations are recovered to safe
  `UNKNOWN` projection on daemon restart.

## Verification

```text
pnpm api:lint                         PASS
pnpm api:generate                     PASS
go test ./...                         PASS
go vet ./...                          PASS
go build ./...                        PASS
git diff --check                      PASS
```

Focused tests cover success, provider rejection, timeout, cancellation, output
limit, unsafe context engine, missing credential, concurrent start, event
redaction, stale-result invalidation, restart recovery and HTTP cancellation
dispatch. Existing command-runner tests exercise real descendant cleanup on
normal exit, timeout, cancellation and output overflow.

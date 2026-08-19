# Phase 5 Batch 3 — Credential Lifecycle Evidence

- Date: 2026-08-19
- Credential authority: Hermes Profile canonical `.env`
- Gate: PASS

## Delivered contract

- Credential status, set/replace and delete are exposed through the existing
  Runtime bundle without making paths or credential environment names public.
- GET returns only preset id, a boolean presence status and observation time.
- PUT accepts the secret once, clears mutable request buffers after use,
  coordinates credential set and non-secret config apply under the existing
  installation/Profile lock, and returns only safe configuration metadata.
- DELETE derives the selected preset from authoritative Hermes config and
  returns metadata only. A subsequent config read becomes `UNCONFIGURED` while
  Instance availability remains unchanged.
- The adapter maps invalid, unsafe, conflicting, query, write and delete
  failures to stable public errors without forwarding native output.
- No schema migration or `secret_refs` row was added. Hermes remains the only
  credential and model-config truth source.

## Security evidence

- The existing adapter tests prove set/replace/delete, exact unknown-line
  preservation, restart recovery, Profile A/B isolation, bounded values/files,
  duplicate rejection, symlink/reparse rejection, atomic read-back, optimistic
  external-change detection and temp cleanup.
- Manager/application/HTTP tests prove public `instanceId` to private
  `nativeId` resolution, no secret argv, metadata-only GET/DELETE, no PUT echo,
  closed-body rejection, stable redacted errors and authenticated route/CORS
  behavior.
- Production `.env` access remains confined to
  `internal/runtime/hermes/model_credential.go`; no HTTP, application, React,
  SQLite, shell or generic file editor was introduced.

## Verification

```text
pnpm api:lint                         PASS
pnpm api:generate                     PASS
go test ./...                         PASS
go vet ./...                          PASS
go build ./...                        PASS
git diff --check                      PASS
```

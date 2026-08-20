# Phase 5 Amendment 005A1 — Fresh Profile Model Configuration

- Status: **IMPLEMENTED / AUDIT PENDING**
- Date: 2026-08-20
- Owner approval: 2026-08-20
- Affected baseline: `phase-005-models-credentials-baseline`
- Scope: Hermes `0.20.2` model configuration read/save behavior only
- Related: `PHASE-005-models-credentials.md`, ADR-0007

## Context

The frozen Phase 5 implementation reads `model.provider` and `model.default`
as two required scalar values. A newly created Hermes Profile has the pinned
default `model: ""`; querying either nested key exits non-zero with the
human-readable diagnostic `Config key not set`. YORVA therefore reports
`MODEL_CONFIG_QUERY_FAILED` before it can apply the first Provider/model
selection.

The credential PUT writes the Profile-scoped credential before applying the
non-secret configuration. Consequently, the credential may be safely present
even though Desktop reports a query failure. This contradicts the Phase 5
contract that incomplete configuration is `UNCONFIGURED` and that a partial
Save returns a stable incomplete/apply result.

The defect blocks the Phase 5 success flow for a fresh Instance and must be
corrected before Phase 6 implementation resumes.

## Decision

1. Read the pinned official aggregate surface:
   `hermes --profile <nativeId> config get model --json`.
2. Interpret the exact JSON empty string as an authoritative unconfigured
   model selection.
3. For a JSON object, read only string-valued `provider` and `default` fields;
   missing fields are incomplete state, not transport/query failure. Other
   fields remain Hermes-owned and are ignored.
4. Reject malformed JSON, non-empty scalar forms, invalid field types,
   oversized output, command failure, timeout and output truncation as
   `MODEL_CONFIG_QUERY_FAILED`.
5. Preserve the existing double-read comparison so an external concurrent
   modification still fails closed.
6. After a credential write succeeds, normalize a subsequent configuration
   failure to `MODEL_CONFIG_INCOMPLETE` with safe observed metadata. Do not
   delete or roll back a credential that may have existed before the request.
7. Keep unrecognized Hermes Provider identifiers as safe `UNCONFIGURED`
   state. Do not add an unqualified alias.

## Boundaries

This amendment does not change:

- credential authority or ADR-0007;
- API routes or schemas;
- SQLite schema or persistence authority;
- Provider presets or model identifiers;
- validation behavior;
- Phase 6 lifecycle/channel scope.

It adds no dependency, generic configuration API, raw file reader or
human-readable CLI parser.

## Required Verification

- fresh Profile reads as `UNCONFIGURED` without error;
- fresh Profile Save reaches `CONFIGURED` after Provider/model read-back;
- partial Provider/model state remains safely `UNCONFIGURED`;
- real query failures and malformed/oversized output still fail closed;
- a post-credential configuration failure returns
  `MODEL_CONFIG_INCOMPLETE` and never exposes or rolls back the credential;
- an unrecognized Provider stays `UNCONFIGURED`;
- Adapter, application, HTTP and Desktop regression tests pass;
- full affected Go checks, Desktop checks and `git diff --check` pass;
- an independent amendment audit accepts the candidate before a revised
  Phase 5 baseline is frozen.

## Gate

Phase 6 implementation remains paused until this amendment is implemented,
verified, independently audited and accepted as the revised Phase 5 baseline.

## Implementation Verification — 2026-08-20

- Hermes adapter regression tests cover fresh, partial, configured, unknown,
  malformed, invalid-type and oversized aggregate model configuration output.
- Application and HTTP regression tests confirm that a failure after the
  credential write returns `MODEL_CONFIG_INCOMPLETE` with safe metadata and
  without echoing the secret.
- Desktop regression coverage confirms that the password field is cleared and
  only the stable incomplete error code is rendered after a partial Save.
- `go test ./...` and `go vet ./...` passed.
- Desktop tests passed: 19 files, 84 tests.
- Desktop typecheck, lint and production web build passed.
- Tauri release `--no-bundle` build passed and produced
  `apps/desktop/src-tauri/target/release/yorva-desktop.exe`.
- A read-only check against the existing fresh `hermers_test` Profile displayed
  `UNCONFIGURED` without `MODEL_CONFIG_QUERY_FAILED`. No credential or model
  configuration mutation was performed during this check.

Independent amendment audit and revised-baseline acceptance remain pending.

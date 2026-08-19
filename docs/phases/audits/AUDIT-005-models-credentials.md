# YORVA Phase 5 Audit — Models and Credentials

## Phase

Phase 5 — Models and Credentials

## Baseline / Commit

- Frozen Phase 4 baseline: `phase-004-instance-profile-baseline` →
  `0dd5d432affd44e23bf577acfe9cf2fdfbfa3f45`
- Branch inspected: `codex/phase5-models-credentials`
- Immutable first-audit candidate:
  `bb8c0ee7e0cc0055f4e5cd2896e8510906d498d6`
- Pinned Hermes source: `0.20.2` at
  `df4b65147d7ddd74dd449f9067aabbca5aef0ec7`
- Reviewed source archive SHA-256:
  `2ED02F76AAF5DAB0BFD320BDBFA10AAD0F67E00CBBF87906CDE05462681708BA`

The tracked worktree was clean at audit start. Pre-existing untracked
`.tmp-yorva-ui-ref/` and `YORVA_0.3.0_x64_en-US.wxs` were outside the candidate
and were not modified. This first pass did not modify implementation code.

## Auditor

Fresh audit pass in the current Agent context, as required by the Owner's
single-Agent/no-subagent execution authorization. The review used the actual
baseline diff, source, tests, pinned Hermes archive and local verification,
not the implementation summaries alone.

## Date

2026-08-19 (Asia/Shanghai)

## Gate Decision

**FAIL**

## Executive Summary

The candidate implements the intended preset, model configuration,
Hermes-native credential and explicit validation flows without entering Phase
6. Its dependency direction, public/native identity split, authenticated API,
closed DTOs, Profile-scoped writer, contained validation process and Desktop
password state are materially aligned with the approved contract.

Two High defects block the gate. A credential PUT mutates the selected
credential before the model ID is validated, so a rejected request can still
replace a working key. The credential writer also preserves an existing
permissive file mode, allowing a newly written key to remain readable by local
principals on platforms where mode bits apply.

Five Medium defects and one Low localization defect are bounded but practical
to close in the same remediation: inconsistent two-scalar config reads,
dotenv interpolation ambiguity, a request-size/OpenAPI mismatch, a cancellation
CAS race, unsupported-installation UI controls, and untranslated Provider help.
Exact-candidate CI is intentionally pending until these findings are fixed.

## Verification Evidence

- `pnpm install --frozen-lockfile` — PASS.
- `pnpm audit --audit-level low` — PASS, no known vulnerabilities.
- OpenAPI lint/generation/generated-schema drift — PASS.
- Desktop typecheck, lint, 19 files / 79 tests, and build — PASS.
- Phase 5 Go files `gofmt -l` — PASS, 31 files. A whole-tree scan found only
  four pre-existing Phase 4 files outside this diff.
- `go test ./...`, five repeated model/credential package tests, `go vet`, Go
  build and `govulncheck` — PASS.
- Local Go race — environment-blocked by `CGO_ENABLED=0`; exact Linux CI is
  required for closure.
- Rust fmt, 10 library tests, clippy and check — PASS.
- `cargo audit` — no vulnerabilities; 17 inherited allowed maintenance or
  unsoundness warnings in platform-only transitive dependencies.
- Windows sidecar build, lifecycle smoke, MSI inspector negative tests and
  Tauri no-bundle release build — PASS.
- Host Hermes discovery smoke — PASS, but the installed host version is
  `0.20.0`, so it is not claimed as pinned `0.20.2` integration proof.
- Isolated Profile credential/config restart scenarios — PASS at `-count=10`.
- Pinned archive version/hash and provider source mappings — PASS.

## Dimension Results

### Scope

PASS. No channels, chat UI, lifecycle, OAuth, custom endpoint, plugin system,
generic file editor, SecretStore duplicate or Phase 6 behavior was added.

### Correctness

FAIL. A rejected credential Save can mutate native state, and config reads can
combine provider/model values from different external states.

### Architecture

PASS. React calls the authenticated daemon client; application use cases own
identity/coordination; Runtime-neutral contracts do not import Hermes; exact
provider/env/config mappings remain adapter-private.

### Security

FAIL. Existing permissive `.env` modes are preserved when writing a new secret.
No secret was found in SQLite, HTTP reads, events, Operations, argv, logs or
Desktop persistence.

### Data and Persistence

PASS. No schema change was needed. SQLite stores validation projection and a
non-secret config fingerprint only, not credential/config authority.

### Concurrency and Lifecycle

FAIL. Installation-scoped mutation coordination and restart recovery pass, but
the cancellation status update has a real compare-and-set race.

### Protocol and Compatibility

FAIL. OpenAPI permits a 16,384-character credential while the transport rejects
some conforming escaped payloads at 24 KiB. Exact pinned mappings otherwise
match the reviewed archive.

### Testing and Verification

FAIL. Broad verification passes, but the reported negative paths lack
regression tests and exact-candidate CI has not yet run.

### Maintainability

PASS. New files are cohesive and use existing interfaces/locks/Operations.
No speculative dependency or framework was added.

### Documentation

PASS WITH FINDING. Governing docs and bilingual Specs agree. Provider help text
is nevertheless rendered in English in the Chinese Desktop locale.

### Dependencies / Supply Chain

PASS. No dependency or lockfile changes were introduced. The pinned Hermes
archive hash and version match the qualification record.

### Operations / Diagnostics

PASS. Model/provider output is discarded, events are closed, restart recovery
terminalizes orphaned validations, and safe stable codes reach the UI.

## Findings

### Critical

None.

### High

#### HIGH-005-001 — Invalid Save requests can replace a credential before rejection

`SaveModelCredentialConfiguration` calls `SetModelCredential` before
`ApplyModelConfig` (`services/node/internal/app/models.go:87-97`). Model ID
validation occurs only inside `ApplyModelConfig`
(`services/node/internal/runtime/hermes/model_config.go:62-67`). Therefore an
unknown/path/config-key model ID can replace the selected Profile credential,
then return `MODEL_CONFIG_INVALID`. A rejected request must not silently mutate
the credential.

Required closure: introduce one Runtime-owned selection preflight, call it
before credential mutation, and prove invalid preset/model input leaves the
credential writer untouched.

#### HIGH-005-002 — Credential replacement preserves an unsafe existing file mode

`credentialStore.commit` changes new files to `0600` but copies
`expected.mode` for existing files
(`services/node/internal/runtime/hermes/model_credential.go:270-274`). If an
existing Profile `.env` is `0644` or otherwise permissive, YORVA writes the new
API key into a replacement with the same exposure. The approved plaintext
at-rest tradeoff does not authorize broadening local read access.

Required closure: always write the replacement with owner-only mode where mode
bits apply and add a regression test starting from a permissive file.

### Medium

#### MEDIUM-005-001 — A config GET can combine two different external states

`ReadModelConfig` reads `model.provider` and `model.default` once each in
sequence (`model_config.go:40-55`). An external Hermes change between calls can
produce a provider/model pair that never existed. Require two equal consecutive
native observations or fail closed with a stable query/conflict error.

#### MEDIUM-005-002 — `${...}` credential text is reported configured but is expanded by dotenv

The writer accepts `${...}` and its local status parser treats it literally,
while pinned Hermes uses `python-dotenv==1.2.2`, whose interpolation pass
resolves that pattern. YORVA can therefore report `CONFIGURED` while Hermes
receives a different key. Reject interpolation syntax in new and observed
target assignments and add a regression test.

#### MEDIUM-005-003 — Credential request bound rejects OpenAPI-conforming payloads

OpenAPI allows `value.maxLength: 16384`, but `decodeClosedModelCredential`
limits the complete JSON body to 24 KiB (`models.go:258-259`). JSON escapes can
make a conforming credential substantially larger. Use a separate bounded
transport limit that safely contains the documented decoded maximum and test
an escaped maximum-size request.

#### MEDIUM-005-004 — Cancellation can lose a status race and return 500

`CancelModelValidation` reads an Operation once, cancels the worker, then
updates using the stale status (`model_validation.go:79-108`). If the worker
changes `PENDING` to `RUNNING` between the read and update, the cancellation
has taken effect but the CAS fails and the API can return an internal error.
Retry the terminalization against the latest non-terminal row and prove the
race with a focused helper/test.

#### MEDIUM-005-005 — Desktop controls remain enabled for an unsupported installation

The panel's `disabled` state only checks Instance availability and local busy
state (`ModelConfigurationPanel.tsx:89-91`). An `AVAILABLE` Profile on a Hermes
version outside the exact model surface returns `MODEL_PROVIDER_UNSUPPORTED`,
but Save remains enabled after local selection. Treat that stable error as a
non-operable panel state and cover it in the component test.

### Low

#### LOW-005-001 — Provider help text bypasses the Chinese message catalog

Qualified Qwen/GLM help is adapter-supplied English and rendered directly
(`ModelConfigurationPanel.tsx:210`). Provide locale-owned text for those safe
preset IDs and prove the Chinese locale does not show the English sentence.

### Info

#### INFO-005-001 — Verification environment observations

Local race is unavailable with CGO disabled; pinned `0.20.2` real-network smoke
is unavailable because the host installation is `0.20.0`. Exact-commit CI must
cover race and the source archive/static contract plus isolated fake-key tests
remain the safe pinned evidence. The two pre-existing untracked paths remain
outside Phase 5.

## Accepted Technical Debt

None accepted by this audit. The inherited Rust warnings and four Phase 4
format observations predate and are outside the Phase 5 diff.

## Required Fixes Before Next Phase

1. Close both High findings and all listed Medium findings with focused tests.
2. Localize safe Provider help in both supported locales.
3. Re-run the complete local verification and Windows smoke matrix.
4. Create an immutable remediated candidate and perform `AUDIT-005R1`.
5. Obtain exact-commit GitHub Actions PASS before merge/freeze/tag.

## Gate Rationale

The Audit Standard requires FAIL for unresolved High findings and for a
required negative path that mutates state after reporting rejection. Green
broad tests do not override either High defect. The remediation is narrow and
does not require a new ADR, dependency or scope expansion.

## Next Step

```text
Phase 5 implementation candidate bb8c0ee: NOT ACCEPTED
AUDIT-005: FAIL
Remediation: AUTHORIZED by the Owner's continuous execution grant
Exact-candidate CI / merge / freeze / tag: BLOCKED pending AUDIT-005R1
Phase 6: BLOCKED
```

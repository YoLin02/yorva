# YORVA Phase 5 Re-audit R1 — Models and Credentials

## Phase

Phase 5 — Models and Credentials

## Baseline / Commit

- Frozen Phase 4 baseline: `phase-004-instance-profile-baseline` → `0dd5d432affd44e23bf577acfe9cf2fdfbfa3f45`
- Immutable first-audit candidate: `bb8c0ee7e0cc0055f4e5cd2896e8510906d498d6`
- First audit: `AUDIT-005-models-credentials.md` — **FAIL**
- Audit remediation: `716fea5681d1442587d4c5a13b567393fee9d463`
- Cross-platform test-fixture correction: `dd7c6c2f47a2b3c7331ebc810b1eb2b003ab59a9`
- **Immutable R1 candidate:** `dd7c6c2f47a2b3c7331ebc810b1eb2b003ab59a9`
- Branch inspected: `codex/phase5-models-credentials`
- Exact-candidate CI: GitHub Actions run
  [`32343964969`](https://github.com/YoLin02/yorva/actions/runs/32343964969) — **SUCCESS**
- Pinned Hermes source: `0.20.2` at `df4b65147d7ddd74dd449f9067aabbca5aef0ec7`
- Reviewed source archive SHA-256: `2ED02F76AAF5DAB0BFD320BDBFA10AAD0F67E00CBBF87906CDE05462681708BA`

The tracked worktree was clean at candidate creation. Pre-existing untracked
`.tmp-yorva-ui-ref/` and `YORVA_0.3.0_x64_en-US.wxs` remain outside the
candidate and were not modified or assessed.

## Auditor

Phase 5 R1 re-audit in the Owner-authorized single-Agent execution context.
The Owner expressly prohibited subagents for this closeout, so a separate
audit agent was not used. The review followed `AUDIT_STANDARD.md`, revisited
every dimension affected by remediation, inspected the actual committed diff,
and used exact-commit CI rather than implementation summaries as evidence.

## Date

2026-08-20 (Asia/Shanghai)

## Gate Decision

**PASS**

## Executive Summary

All two High, five Medium and one Low findings from immutable `AUDIT-005` are
closed on `dd7c6c2f47a2b3c7331ebc810b1eb2b003ab59a9`.

Credential Save now validates the preset/model selection before any secret
mutation. Credential replacement always publishes a `0600` temporary file on
platforms with POSIX mode semantics and no longer preserves a permissive
existing mode. `${...}` values are rejected both on write and when observing
Hermes-owned state. Model configuration reads require two equal full native
snapshots. The credential HTTP body bound now accommodates the OpenAPI maximum
when JSON escaping expands the wire payload. Cancellation retries a stale
status compare-and-set against the latest Operation. Desktop mutations are
disabled when the pinned model surface is unsupported, and Provider-specific
help is localized.

The first exact-candidate CI attempt exposed a Windows-only absolute path in a
test fixture when run on Linux. The fixture was corrected without changing
production behavior. The final exact-candidate run is green for Web/API,
Linux `go test -race`, and the complete Windows native/Tauri job.

No new Critical, High, Medium or Low finding was found. Phase 5 satisfies its
mandatory acceptance criteria and may be merged, frozen and tagged under the
Owner's existing authorization. Phase 6 remains out of scope and is not
authorized by this Gate.

## Verification Evidence

### Exact-commit CI

GitHub Actions run
[`32343964969`](https://github.com/YoLin02/yorva/actions/runs/32343964969):

- `head_sha` = `dd7c6c2f47a2b3c7331ebc810b1eb2b003ab59a9`
- `Web and API contract` — SUCCESS
- `Go Node` — SUCCESS, including `go test -race ./...`, vet, vulnerability
  scan and daemon build
- `Windows Desktop native shell` — SUCCESS, including sidecar build,
  lifecycle smoke, MSI inspector negative tests, Rust format/tests/audit,
  clippy/check and Tauri release no-bundle build

Historical run `32343749442` failed only because the new model test fixture
used a Windows absolute path on Linux. Commit `dd7c6c2` replaced it with an
OS-appropriate absolute test path; the complete Linux race job then passed.

### Local verification

- `pnpm install --frozen-lockfile` — PASS.
- `pnpm audit --audit-level low` — PASS, no known vulnerabilities.
- OpenAPI lint, generation and generated-client drift — PASS.
- Desktop typecheck, lint, build and 19 files / 80 tests — PASS.
- Changed Go files `gofmt` and `git diff --check` — PASS.
- `go test ./...`, five repeated affected-package runs, `go vet ./...`,
  `go build ./...` and `govulncheck ./...` — PASS.
- Local Go race was unavailable with Windows `CGO_ENABLED=0`; exact Linux CI
  supplied the required race evidence and passed.
- Rust format, 10 library tests, clippy and check — PASS.
- `cargo audit` — no vulnerabilities; 17 inherited allowed maintenance or
  unsoundness warnings remain in platform transitive dependencies.
- Windows sidecar build, lifecycle smoke and MSI inspector negative tests —
  PASS.
- Tauri release no-bundle build — PASS.
- Isolated Profile credential/config restart tests repeated ten times — PASS.
- Host Hermes discovery smoke — PASS, but the host is `0.20.0`; it is not
  represented as live pinned-`0.20.2` proof.
- Pinned source version/hash and adapter mapping inspection — PASS.

### First-audit finding closure

| Finding | Result | Evidence |
|---|---|---|
| `HIGH-005-001` invalid Save mutates credential first | **CLOSED** | `ModelConfigurator.ValidateModelSelection`; `SaveModelCredentialConfiguration` preflight; `TestCredentialSaveValidatesSelectionBeforeSecretMutation` |
| `HIGH-005-002` permissive mode preserved | **CLOSED** | credential commit always applies `0600`; `TestModelCredentialSetReplaceDeletePreservesUnknownContent` verifies replacement mode on POSIX |
| `MEDIUM-005-001` mixed config snapshot | **CLOSED** | `ReadModelConfig` requires two equal `nativeModelConfig` snapshots; changing-state regression fails closed |
| `MEDIUM-005-002` dotenv interpolation ambiguity | **CLOSED** | `${` rejected by write validation and observed assignment parsing; unsafe-state regression included |
| `MEDIUM-005-003` HTTP/OpenAPI size mismatch | **CLOSED** | 128 KiB bounded request; 16,384-character escaped-value regression passes |
| `MEDIUM-005-004` cancellation CAS race | **CLOSED** | bounded latest-row retry on `ErrInvalidStatusTransition`; deterministic PENDING→RUNNING race regression |
| `MEDIUM-005-005` unsupported Desktop controls | **CLOSED** | `MODEL_PROVIDER_UNSUPPORTED` derives non-operable state; mutation controls and cancel disabled; component regression passes |
| `LOW-005-001` English help in Chinese UI | **CLOSED** | Qwen/GLM help added to both locale catalogs; zh-CN component and catalog parity tests pass |

## Dimension Results

### Scope

**PASS.** Remediation is limited to Phase 5 model configuration, Runtime-native
credentials, validation cancellation, protocol bounds and Desktop presentation.
No lifecycle, channels, chat, OAuth, custom endpoint, plugin, Cloud or generic
file-editing capability entered the candidate.

### Correctness

**PASS.** Save ordering is mutation-safe; native reads fail closed on changing
state; credential/config read-back, partial apply, restart truth and explicit
validation behavior are covered by positive and negative tests.

### Architecture

**PASS.** The required React → authenticated Node API → application → Runtime
contract direction remains intact. Hermes paths, environment names and config
keys remain adapter-private. The new validation method exposes only stable
preset/model concepts and prevents application code from duplicating
Hermes-specific validation rules.

### Security

**PASS.** Credential plaintext remains write-only and absent from SQLite,
HTTP reads, Operations, events, logs, argv and Desktop persistence. Credential
targeting remains Profile-scoped and allowlisted. Writes use same-directory
atomic replacement, optimistic conflict detection, `0600` publication and
post-use byte clearing. Interpolated dotenv values fail closed.

### Data and Persistence

**PASS.** Phase 5 adds no schema or migration. SQLite stores only non-secret
Operation/projection metadata. Hermes remains authoritative for Profile model
configuration and credentials.

### Concurrency and Lifecycle

**PASS.** Instance-scoped mutation coordination remains narrow. Validation
workers retain explicit context cancellation and contained process ownership.
The cancellation status race is closed by retrying against the latest stored
Operation, and exact Linux race detection passes.

### Protocol and Compatibility

**PASS.** OpenAPI remains the source of truth, DTOs are closed and typed,
errors are stable, and the generated Desktop schema has no drift. The request
bound now accepts the contract maximum under JSON escaping. Hermes mappings
remain pinned to reviewed `0.20.2` source.

### Testing and Verification

**PASS.** Every first-audit defect has a regression test. Full local checks and
all three exact-candidate CI jobs pass. The Linux fixture defect discovered by
CI was fixed and reverified on a new immutable SHA rather than bypassed.

### Maintainability

**PASS.** Remediation reuses existing contracts, query state, Operation storage
and locale catalogs. It adds no framework, generic editor, dependency or
speculative abstraction.

### Documentation

**PASS.** Phase Spec, ADR-0007, architecture/security/data ownership documents,
OpenAPI and Batch evidence remain aligned with implementation. This R1 report
records the immutable failure history and final closure evidence.

### Dependencies / Supply Chain

**PASS.** Remediation adds or upgrades no dependency. JavaScript and Go
vulnerability scans report none. Rust audit reports no vulnerability; inherited
warnings are unchanged from the frozen baseline.

### Operations / Diagnostics

**PASS.** Validation remains an explicit bounded Operation with stable safe
codes, restart projection, cancellation and SSE/query invalidation. No raw
Provider response, model output or secret is introduced into diagnostics.

## Findings

### Critical

None.

### High

None. `HIGH-005-001` and `HIGH-005-002` are closed.

### Medium

None. `MEDIUM-005-001` through `MEDIUM-005-005` are closed.

### Low

None. `LOW-005-001` is closed.

### Info

- The local host has Hermes `0.20.0`, not the pinned `0.20.2`. Live host smoke
  is therefore limited to discovery. Compatibility evidence comes from the
  reviewed pinned archive, adapter contract fixtures, isolated fake-key tests
  and exact CI; no stronger live claim is made.
- The 17 Rust audit warnings are inherited platform transitive maintenance or
  soundness notices, not Phase 5 dependency changes or reported
  vulnerabilities.

## Accepted Technical Debt

None introduced or accepted by Phase 5 R1. The two Info observations above are
verification-environment and inherited-dependency facts, not deferred Phase 5
correctness or security defects.

## Required Fixes Before Next Phase

None for the Phase 5 Gate. Phase 6 still requires its own approved Phase Spec
and entry authorization; this audit does not authorize implementation.

## Gate Rationale

`PASS` is justified because there are zero unresolved Critical or High
findings, all mandatory Phase 5 acceptance criteria have evidence, all eight
first-audit findings are closed with regression coverage, exact-candidate
Linux race and Windows native checks pass, and no security or architecture
condition prevents freezing this phase.

The immutable first-audit `FAIL` remains part of repository history and is not
rewritten. This R1 Gate supersedes it only for the remediated candidate
`dd7c6c2f47a2b3c7331ebc810b1eb2b003ab59a9`.

## Next Step

Under the Owner's existing complete authorization:

1. record `AUDIT-005R1` PASS in both Phase 5 Spec mirrors;
2. merge the audited branch to `main` without changing implementation;
3. run exact final-main CI and the Windows MSI workflow;
4. mark Phase 5 `COMPLETE / FROZEN` and create the annotated
   `phase-005-models-credentials-baseline` tag only after those checks pass;
5. do not begin Phase 6.

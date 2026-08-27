# YORVA Phase 007 Audit

## Phase

Phase 7 — Hermes Runtime Management Completeness / P7R MVP closure

## Baseline / Commit

Candidate reviewed at `f559568` on branch
`codex/phase7-hermes-runtime-management`.

Required predecessor baseline:
`phase-0065-developer-led-demo-baseline` at
`5f68e48f17e7e342e1781b37613b19d4bd1f060b`.

## Auditor

Codex fresh repository-state review, with Repository Owner retaining the final Gate
Decision.

## Date

2026-08-27

## Gate Decision

**FAIL**

## Executive Summary

The reviewed candidate contains substantial real Phase 7 functionality and passes the
available local OpenAPI, Desktop, Go, Rust, security scanning, no-bundle release build,
Windows lifecycle, and reviewed MCP lifecycle checks. The restricted MCP surface remains
closed to caller-controlled command, args, environment, headers, paths, and arbitrary
JSON.

The candidate cannot be frozen because two mandatory product flows are not established:

1. production `yorvad` does not bind an implementation to `Upgrade` or `Rollback`, so
   Runtime Upgrade/Rollback mutation is capability-false and cannot satisfy the Phase 7
   or P7R exit criteria;
2. the required disposable Windows Restore success/failure-recovery smoke has not been
   performed or preserved as candidate evidence.

Exact-candidate CI, including Go race evidence, is also absent because the candidate has
not been pushed. The local machine cannot run Go race tests because no C compiler is
available. Per the Phase Spec, these gaps are blocking evidence rather than accepted
technical debt.

## Verification Evidence

- `pnpm install --frozen-lockfile` — PASS.
- `pnpm audit --audit-level low` — PASS, no known vulnerabilities.
- `pnpm api:lint` — PASS.
- `pnpm api:generate` plus generated-schema diff check — PASS.
- `pnpm typecheck` — PASS.
- `pnpm lint` — PASS.
- `pnpm test` — PASS, 24 files / 138 tests.
- `pnpm build` — PASS.
- `go test ./...` — PASS.
- `go vet ./...` — PASS.
- `go build ./cmd/yorvad` — PASS.
- `govulncheck ./...` — PASS, no called vulnerabilities.
- `go test -race ./...` — NOT RUN locally; the Go toolchain requires CGO and this
  machine has no `gcc` in `PATH`.
- `cargo fmt --check` — PASS.
- `cargo test --locked --lib` — PASS, 13 tests.
- `cargo clippy --locked --all-targets -- -D warnings` — PASS.
- `cargo check --locked` — PASS.
- `cargo audit` — PASS with 17 allowed transitive warnings and no failing advisory.
- `pnpm --filter @yorva/desktop tauri build --no-bundle` — PASS.
- `scripts/windows-lifecycle-smoke.ps1` — PASS.
- `go test -tags mcpqualification ./internal/app -run
  TestYORVAManagesProductionLocalTestMCPAcrossProfiles -count=1 -v` — PASS.
- MSI was not rebuilt because the Owner's current working constraint explicitly excluded
  MSI packaging. That constraint must be revisited for the final candidate if the Phase
  Spec determines the Phase 7 Tauri/runtime packaging changes require MSI evidence.

## Dimension Results

### Scope

PASS — Runtime resources, exact Instance bindings, restricted MCP, managed Skills,
encrypted Runtime backup, diagnostics, and model sharing remain inside the approved P7R
scope. No second Runtime, Control Plane, arbitrary execution API, or P8 implementation is
present.

### Correctness

FAIL — the mandatory Upgrade/Rollback user flow is unavailable in production wiring, and
the destructive Restore flow lacks the required disposable Windows success/failure
evidence.

### Architecture

PASS — reviewed ownership remains React → typed daemon API → application → Runtime
contract → Hermes adapter. The Desktop does not invoke Hermes or edit Hermes profiles
directly.

### Security

PASS for the implemented and tested surfaces — local authentication, OS-backed secrets,
restricted MCP Presets, encrypted backup, bounded logs, and no caller-controlled command
surface remain intact. This does not waive the missing destructive Restore evidence.

### Data and Persistence

PASS — migrations and persistence tests pass, including the migration-16 repair for
databases whose historical version-15 ledger predates shared model tables.

### Concurrency and Lifecycle

PASS for locally executed checks — Operation recovery and Windows daemon lifecycle tests
pass, and the MCP cancellation lifecycle is covered. Exact-candidate race evidence remains
required under Testing and Verification.

### Protocol and Compatibility

PASS — OpenAPI validates, generated TypeScript is in sync, stable error/capability
boundaries are preserved, and unsupported mutations fail closed.

### Testing and Verification

FAIL — exact-candidate CI/race evidence and the required disposable Restore smoke are
missing. A final candidate must rerun the applicable packaging evidence after remediation.

### Maintainability

PASS — no new speculative framework or dependency layer was found in the P7R closure
changes. Runtime-specific behavior remains under the Hermes adapter.

### Documentation

PASS after this audit record and the synchronized P7R integration update. The Gate remains
FAIL because documentation cannot substitute for the missing product capability/evidence.

### Dependencies / Supply Chain

PASS — pnpm audit and govulncheck pass. Cargo audit reports allowed transitive
unmaintained/unsound warnings already inherited through the desktop stack, but no failing
advisory was returned.

### Operations / Diagnostics

FAIL — backup and diagnostics are wired, but production Upgrade/Rollback execution and its
post-check/rollback diagnostics are not available.

## Findings

### Critical

None.

### High

#### P7-AUDIT-HIGH-001 — Production Upgrade/Rollback is not wired

`services/node/internal/daemon/daemon.go` constructs `hermes.ManagementBindings` with
MCP and backup bindings only. `Upgrade` and `Rollback` remain nil when the Runtime is
registered. `services/node/internal/app/management_upgrade.go` has durable orchestration,
but its execution path correctly returns unsupported without those adapter bindings.

Impact: the user cannot complete the approved P7/P7R Runtime Upgrade/Rollback flow, and
the Phase exit criterion requiring a real fixed-candidate upgrade cannot pass.

Required remediation: implement and qualify the fixed packaged-candidate generation
Upgrader/Rollbacker, inject both bindings, prove protection point, final-path validation,
activation CAS, authoritative post-check, failure recovery, and compatible rollback on
disposable Windows state.

#### P7-AUDIT-HIGH-002 — Required disposable Restore smoke is missing

The Phase Spec requires one successful Restore and one failed Restore recovery on
disposable test state. The current P7R B7 handoff explicitly states that destructive
Restore smoke was not run.

Impact: automated unit/integration coverage does not alone establish the full native
process-stop, encrypted artifact, restore, reconciliation, and recovery lifecycle required
to freeze this destructive capability.

Required remediation: run and preserve a sanitized Windows product smoke against
disposable Hermes data, covering successful Restore, rejected/failed Restore, recovery,
authoritative read-back, and no secret disclosure.

#### P7-AUDIT-HIGH-003 — Exact-candidate CI evidence is incomplete

The candidate is eleven commits ahead of the remote Phase 7 branch and has no
exact-commit CI result. Local Go race cannot run because the machine lacks a C compiler.

Impact: the required race and exact-candidate Windows evidence is not attributable to the
reviewed commit.

Required remediation: after HIGH-001 and HIGH-002 are fixed, create an immutable
candidate, push it, require the configured CI/race/native jobs to pass, and preserve the
exact commit/run identifiers before merge and tag.

### Medium

None.

### Low

None.

### Info

- The local no-bundle Tauri release build completed with a non-failing MSVC linker warning
  about an import library being created.
- Cargo audit returned 17 allowed warnings in transitive desktop dependencies. Reassess
  them during P8 dependency/release hardening; they are not a Phase 7 blocking defect.

## Accepted Technical Debt

None. The High findings are blocking and are not accepted as technical debt.

## Required Fixes Before Next Phase

1. Complete and qualify production Upgrade/Rollback wiring.
2. Produce disposable Windows Restore success and failure-recovery evidence.
3. Re-run the full candidate Gate, including exact-candidate CI and Go race.
4. Re-audit the affected Correctness, Concurrency/Lifecycle, Testing, Operations, Security,
   and Compatibility dimensions.
5. Only after PASS: merge to `main`, pass final-main CI, create the annotated Phase 7 tag,
   mark Phase 7 FROZEN, and finalize the Phase 8 Spec.

## Gate Rationale

Phase 7 cannot receive PASS or PASS WITH CONDITIONS while a required production mutation
is unavailable and destructive-flow verification is missing. These findings directly
match the Phase Spec's mandatory acceptance and stop conditions.

## Next Step

Return Phase 7 to remediation. Do not push the current candidate for merge, merge it,
create the Phase 7 baseline tag, mark the phase FROZEN, or begin P8 implementation/spec
finalization.

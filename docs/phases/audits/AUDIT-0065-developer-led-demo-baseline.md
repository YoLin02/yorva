# YORVA Phase 6.5 Audit

## Phase

Phase 6.5 — Developer-led Demo Optimization

## Baseline / Commit

- Immutable predecessor: `phase-006-runtime-lifecycle-messaging-channels-baseline`
- Predecessor commit: `7ca9103e7af210296a5e24916df01856539b550e`
- Audited branch: `codex/phase6.5-freeze`
- Audited implementation candidate: `559245cf42c6ea07dbe4706676334a717b2173fd`
- Candidate remote identity: local `HEAD`, upstream branch, and CI head SHA matched.

The baseline-to-candidate range contains three commits and changes 105 files. The
candidate was assembled in a clean successor worktree; the original developer worktree
and the Phase 6 tag were not mutated.

## Auditor

YORVA Phase 6.5 closeout audit pass.

The formal audit pass treated `559245c` as immutable and did not modify it. It reviewed
the repository rules, Phase 6.5 Spec, Roadmap, architecture/protocol/Runtime/data/security/
governance documents, ADR-0010 through ADR-0012, Amendments 003A7, 003A8 and 0065A1,
the complete baseline-to-candidate diff, source, tests, local builds, MSI inventory, and
remote results. A credential-redirect hardening defect was identified and corrected
before the candidate commit and before this formal no-edit audit pass.

This was a separate structured review pass in the isolated closeout worktree, not a
claim that the primary closeout session is a different human reviewer. The Owner
explicitly requested direct P6.5 closeout, and no unapproved auxiliary implementation
or P7 work was used as evidence.

## Date

2026-08-24 (Asia/Shanghai)

## Gate Decision

**PASS**

No unresolved Critical, High, Medium, or Low finding remains. The accepted P6.5 scope
meets its contract, exact-candidate CI and Windows MSI jobs passed, supply-chain inputs
were inspected, real Windows Hermes and embedded-Python smoke passed, and no P7 or
generic remote/system-control capability entered the candidate.

## Executive Summary

P6.5 is a clean, bounded successor to the immutable Phase 6 baseline. It adds four
cohesive Demo improvements: configurable but credential-free installation transports,
an exact packaged CPython prerequisite, model-catalog/multi-selection/default-model UX,
and patch-compatible stable Hermes `0.20.x` development support while retaining exact
source provenance for packaging.

The review found no false success path, unauthenticated management route, Secret
persistence, arbitrary endpoint for Provider credentials, arbitrary executable/path/
hash/version control, Runtime-specific UI execution, or P7 scope. Provider credentials
are accepted by one authenticated loopback POST, bounded, cleared from owned byte
buffers, sent only in headers to a compiled Provider endpoint, and never followed across
a redirect. Selected model IDs and download-source URLs are non-secret YORVA management
metadata; Hermes remains authoritative for the active default model and Runtime state.

The package includes exact-hash Hermes `0.20.5`, Node, npm and CPython archives plus the
required license inputs. Both local and GitHub MSI inspection passed. Different WiX MSI
digests are recorded separately rather than falsely presented as reproducible bytes.

## Verification Evidence

### Local candidate verification

- OpenAPI lint/generation/drift: PASS.
- Desktop typecheck, lint, production build: PASS.
- Desktop tests: PASS, 22 files / 105 tests.
- pnpm dependency audit: PASS, no known vulnerability.
- Go `go test ./...`, vet, `govulncheck`, and daemon build: PASS.
- Windows lifecycle smoke and MSI-inspector negative suite: PASS.
- Rust format, 13 library tests, audit, clippy with warnings denied, and check: PASS.
- Tauri no-bundle release build: PASS.
- Real installed Hermes smoke: PASS, `SUPPORTED` version `0.20.5`, official launcher.
- Real packaged CPython mirror smoke: PASS, locally installed and ran Python `3.11.15`.
- Candidate MSI preparation and full inventory inspection: PASS.
- `git diff --check`: PASS.

Local Go `-race` was unavailable because CGo is disabled on the Windows host. The exact
candidate CI executed `go test -race ./...` successfully on Linux; this limitation did
not remove the mandatory race check.

### Exact-candidate CI

- run [`32715890955`](https://github.com/YoLin02/yorva/actions/runs/32715890955),
  event `push`, exact head `559245cf42c6ea07dbe4706676334a717b2173fd`:
  **PASS**;
- Web and API contract job `97396905674`: PASS;
- Go Node job `97396906002`: PASS, including race, vet, vulnerability and build checks;
- Windows Desktop native shell job `97396905812`: PASS, including lifecycle smoke,
  package-inspector negative tests, Rust checks and Tauri no-bundle build.

### Exact-candidate MSI

- run [`32715958209`](https://github.com/YoLin02/yorva/actions/runs/32715958209),
  event `workflow_dispatch`, exact head `559245c`: PASS;
- package job `97397119720`: PASS;
- downloaded artifact `9516145610` contained one
  `Yorva_0.3.2_x64_en-US.msi` of `147,566,592` bytes;
- remote MSI SHA-256:
  `4FA1D0958F33AEBEC7B0C373BA798116E8213476455855FC7E83543216A4F5E2`;
- artifact archive digest:
  `sha256:b5bfdca752490616d3e4561bd1a73ce9425bcf20f1ef1ffe5bf12b173c6adc99`;
- independent local exact-candidate MSI SHA-256:
  `455B6424E2F1322113B191E0DCDE7997CA2514254E7D9C38C34C4EA540774570`.

Both MSI builds had the same size and inspected seven-input inventory. Their different
digests are an expected WiX non-reproducibility boundary, not hidden by substituting one
package for the other.

### Supply-chain inventory

The inspector verified exact names, sizes and SHA-256 values for:

- official Hermes `0.20.5` source commit `a0ca7c19204e514f9590ce3b812e029b315ab9e9`;
- Node `22.23.1` Windows x64;
- npm `12.0.2`;
- CPython `3.11.15+20260728` Windows x64 standalone distribution;
- Hermes, Node and npm license inputs.

Substitution, extra executable, missing payload, duplicate payload, wrong hash/size/name,
license mismatch and extraction-failure cases all fail closed.

## Dimension Results

### Scope

**PASS.** All changes map to Owner-accepted P6.5 Demo tasks. P7, remote management,
RBAC, Cloud, generic command/process/service/file APIs, personal artifacts and temporary
references are absent.

### Correctness

**PASS.** Saved/default model state reopens deterministically; the default must be in
the selected set; duplicate/invalid/oversized selections fail; older requests without
the new selection array retain a one-default compatibility path. Installation source
preference is deterministic, transport fallback is bounded, and integrity failure does
not fall back. Real Hermes and Python smoke cover the two external compatibility paths.

### Architecture

**PASS.** React uses typed local HTTP only. Hermes commands, fixed Provider endpoints,
archive identity, Python mirror materialization and compatibility rules remain in the
Hermes-owned adapter/integration packages. Core receives the minimal model-catalog
capability and non-secret application metadata; Tauri only projects packaged resource
paths into daemon bootstrap.

The download-source HTTP resource is intentionally Hermes-specific and typed against
the Hermes-owned settings integration. It does not generalize into a speculative Runtime
source framework or leak executable behavior into React.

### Security

**PASS.** New routes remain bearer-authenticated and loopback-only through the existing
daemon. Download URLs must be credential-free absolute HTTPS without userinfo, query, or
fragment. Versions, hashes, local paths, commands and environment names are compiled and
not caller-controlled. All package bytes are size/hash verified and integrity failure is
fail-closed.

Provider catalog requests use a compiled endpoint/authentication shape, a 15-second
client timeout, a 2 MiB response bound, validated result IDs and explicit redirect
rejection. The API credential field is write-only and no Secret/QR/pairing/account value
is returned, logged, persisted in SQLite, or entered into evidence.

### Data and Persistence

**PASS.** No schema migration is required. Existing `app_settings` stores only closed,
validated, non-secret source configuration and selected model IDs. Hermes remains the
source of truth for active Provider/default-model configuration; invalid or stale
selected-model metadata falls back to the authoritative Hermes default. Existing empty
and prior-schema migration tests passed in the complete Go suite.

### Concurrency and Lifecycle

**PASS.** Model mutations retain the existing per-Instance mutation lock. Catalog fetch
is read-only and holds no database transaction or Instance mutation lock during network
I/O. Installation source settings are snapshotted at Operation start; no database
transaction is held during download, extraction, external command, or Provider request.
Windows lifecycle smoke and CI race testing passed.

### Protocol and Compatibility

**PASS.** OpenAPI remains authoritative and generated TypeScript is deterministic. New
requests/responses are closed and bounded, error codes are stable, and old model config/
credential clients remain accepted by the decoder. ADR-0012 explicitly owns the stable
`>=0.20.2 <0.21.0` development window; prereleases, older versions and `0.21.x` remain
unsupported. Packaging remains pinned to one exact `0.20.5` source commit and digest.

### Testing and Verification

**PASS.** Unit, protocol, negative integrity, redirect, persistence fallback, model UI,
real Runtime, real Python mirror, Windows lifecycle, no-bundle build and two MSI build/
inspection paths cover the changed risks. Exact-candidate CI passed all mandatory jobs.

### Maintainability

**PASS.** No new dependency or generic framework was introduced. Catalog parsing,
download-source persistence and Python artifact materialization are cohesive small files.
The selected-model metadata is bounded and uses the existing settings store rather than
a speculative schema or repository layer.

### Documentation

**PASS.** OpenAPI, Development, Architecture, Protocol, Runtime, Security, governance,
Roadmap, ADRs, amendments, Phase Spec and handoff describe the accepted behavior and
historical Phase 6 baseline without moving or rewriting it.

### Dependencies / Supply Chain

**PASS.** Dependency and lock files are unchanged. pnpm, Go vulnerability and Rust audit
checks passed under the existing repository policy. Every new packaged executable/source
archive is exact-size/hash verified and inventoried by the MSI inspector.

### Operations / Diagnostics

**PASS.** Installation uses existing durable Operations and emits stable safe categories
without URLs, paths or raw upstream responses. Provider catalog fetch is a bounded
synchronous Desktop request with normalized retryable failure and no durable operation
state. No false install/configuration/Runtime success is introduced.

## Findings

### Critical

None.

### High

None.

### Medium

None.

### Low

None.

### Info

- Stable future `0.20.x` patches are intentionally admitted by ADR-0012. A demonstrated
  contract break must narrow the window through a new amendment; `0.21.x` requires a
  separate compatibility review.
- Rust audit reports the repository's existing allowed maintenance/unsoundness warnings;
  P6.5 changes no Rust dependency or lockfile and introduces no new advisory.

## Accepted Technical Debt

None introduced or accepted by P6.5.

## Required Fixes Before Next Phase

None.

## Gate Rationale

All mandatory evidence is exact-candidate-bound, package and security boundaries remain
fail-closed, no serious vulnerability or correctness defect remains, and no P7 assumption
is contaminated. The P6.5 candidate is eligible for merge, final-main CI and annotated
baseline freeze under the Owner's existing authorization.

## Next Step

Merge the audited documentation successor to `main`, require final-main CI to pass, then
record `COMPLETE / FROZEN` and create the new annotated tag
`phase-0065-developer-led-demo-baseline`. Do not move the Phase 6 tag and do not begin P7
implementation before that closeout completes.

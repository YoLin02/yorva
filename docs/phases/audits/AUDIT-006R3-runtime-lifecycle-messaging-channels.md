# Phase 6 Runtime Lifecycle and Messaging Channels Re-audit R3

## Phase

Phase 6 — Runtime Lifecycle and Messaging Channels, fresh independent Closeout R3
audit after immutable `AUDIT-006`, `AUDIT-006R1` and `AUDIT-006R2` failures.

## Baseline / Candidate

- Baseline tag: `phase-005a1-post-freeze-corrections-baseline`
- Baseline peeled commit: `995777528557fa564a4c42e14f8431b8ddbd20e8`
- Audited branch: `codex/phase6-closeout`
- Audited candidate: `859561712583c67031d8110812df3a4236be6908`
- R2 contract correction: `4cff18b72fd29f3fd79682e1063ee87583427e40`
- Audit date: 2026-08-24 (Asia/Shanghai)

At audit start, local `HEAD`, the branch upstream and the remote branch all resolved to
the stated candidate. The worktree was clean. The baseline-to-candidate range contains
25 commits and changes 172 files (`12,411` insertions, `865` deletions). The candidate
remained unchanged during source review and focused verification; this report is the
only intended worktree addition.

The three failed audits remain immutable. Their current SHA-256 values are:

- `AUDIT-006-runtime-lifecycle-messaging-channels.md`:
  `CBAF26C6F4B6C07EC9D0E67D19411D0E55C57E71059D1E62B8B7C2DB6383C560`;
- `AUDIT-006R1-runtime-lifecycle-messaging-channels.md`:
  `971ED2C1BB19CB4796EE6036188E330D3D123DA0D1BD8A46A9FC5108DD32ABC1`;
- `AUDIT-006R2-runtime-lifecycle-messaging-channels.md`:
  `C91D20A679AA79438CEB843208A21CB8AF4C8B9B7ED9C1A8C920CB6F5D990A98`.

`git diff 7d502cca..8595617 -- docs/phases/audits/` confirmed that no failed-audit
byte changed after the immutable R2 audit commit.

## Auditor and Independence

The auditor is a fresh Codex R3 review context. This context did not implement Phase 6,
perform the first remediation, collect the Owner smoke, write any earlier Phase 6 audit,
or implement the R2 OpenAPI correction.

The review read the repository rules; the complete completion/audit/freeze prompt from
the permitted read-only original worktree; `AUDIT_STANDARD.md`,
`PHASE_GOVERNANCE.md`, the architecture, protocol, Runtime, data, security and
development documents; both Phase 6 Spec mirrors; the relevant Phase 5 baseline and
correction records; ADR-0003, ADR-0004 and ADR-0006 through ADR-0009; all Phase 6
qualification, implementation, handoff, Owner-smoke and Windows-smoke evidence; and all
three immutable failed audits. It then independently inspected the candidate range,
the R2 correction, generated client type, Go response DTO and the security/lifecycle
areas affected by the earlier findings.

No QR payload, Secret, pairing code, account identifier, Profile name, PID or other
sensitive value was requested, reconstructed or retained.

## Gate Decision

**PASS**

No unresolved Critical, High, Medium or Low finding remains. R2's sole finding,
M-01-R2, is correctly closed: the displaced `const: false` is absent,
`InstanceCapabilities.lifecycle` remains a dynamic boolean,
`Lifecycle.errorCode` is generated as `string | null`, the Go DTO remains nullable,
and a focused compile-time regression prevents the false literal from returning.

The exact candidate has completed green CI, the mandatory Windows and real-channel
evidence remains mapped to the unchanged product inputs, and no production runtime or
packaging-byte behavior changed after the inspected remediation MSI. The candidate is
eligible to enter Closeout Batch C5 under the Owner's existing authorization.

**C5 eligibility: YES. Phase 7 remains prohibited until C5 merge/final-main/freeze/tag
is actually complete.**

## Scope and Change Review

The R2 successor range `13612db..8595617` contains only:

- immutable `AUDIT-006R2` at `7d502cc`;
- the narrow OpenAPI/generated-TypeScript/test correction at `4cff18b`; and
- synchronized Roadmap, two-Spec and handoff status/evidence updates at `8595617`.

The production correction removes one misplaced OpenAPI constraint. It does not change
Go runtime behavior, network behavior, persistence, Rust/Tauri behavior, dependencies,
packaging inputs or executable version inputs. The test adds a TypeScript-only semantic
assertion. The generated TypeScript interface is erased from production JavaScript.

The wider Phase 6 ancestry still discloses adjacent Phase 2/3 generation work and a
general Desktop refinement rather than attributing them to Phase 6. The separate dirty
`codex/configurable-download-sources` worktree and its installation-source,
embedded-Python, packaging and UI work are absent from this candidate. No Phase 7
Runtime upgrade/repair/uninstall, Skills, MCP, backup/restore, additional Channel,
Cloud, remote command or generic process/service/shell/file capability was found.

## Verification Evidence

### R2 correction inspection

- `api/openapi.yaml:1710-1733` keeps `InstanceCapabilities.lifecycle` as `boolean` and
  declares `Lifecycle.errorCode` as `[string, "null"]` with no `const`.
- `apps/desktop/src/api/generated/schema.ts:691-701` generates
  `lifecycle: boolean` and `errorCode: string | null`.
- `apps/desktop/src/api/client.test.ts:1-18` uses `expectTypeOf` to require exactly
  `Lifecycle["errorCode"] = string | null`.
- `services/node/internal/transport/httpapi/instances.go:61-65,166-179` keeps the Go
  response as `*runtime.ErrorCode` and maps an empty safe code to JSON `null`.
- `git show 4cff18b` changes only the OpenAPI file, generated schema and focused client
  test. `rg "const: false" api/openapi.yaml` found no restored obsolete literal.

### Proportionate local verification

The unchanged full Gate was not rerun. R3 ran only the correction-relevant subset:

- `pnpm api:lint` — PASS;
- `pnpm api:generate` followed by generated-schema diff check — PASS, deterministic;
- `pnpm typecheck` — PASS;
- `pnpm --filter @yorva/desktop test -- src/api/client.test.ts` — PASS, 1 file / 7 tests;
- `pnpm --filter @yorva/desktop lint` — PASS;
- `git diff --check 9957775..8595617` — PASS;
- worktree status after generation and focused checks — clean.

### Exact-candidate CI

Independent GitHub Actions API inspection established:

- run [`32692682968`](https://github.com/YoLin02/yorva/actions/runs/32692682968):
  event `push`, attempt `1`, `completed`, `success`, exact head SHA
  `859561712583c67031d8110812df3a4236be6908`;
- Web and API contract job `97329058369`: `success`, including dependency audit,
  OpenAPI lint/generation/drift, typecheck, lint, tests and Web build;
- Go Node job `97329058444`: `success`, including `go test -race ./...`, vet,
  `govulncheck` and daemon build;
- Windows Desktop native shell job `97329058561`: `success`, including sidecar build,
  Windows lifecycle smoke, MSI-inspector negative tests, Rust format/tests/audit,
  clippy/check and Tauri no-bundle release build.

This is the fresh exact-success run required by R2; no earlier run is substituted for
the changed successor.

### MSI and product-input mapping

The inspected remediation package remains:

- file: `Yorva_0.3.2_x64_en-US.msi`;
- size: `120,213,504` bytes;
- SHA-256: `B942E637BE9BC59D6C5B603DA117C7AC9646CD254E88F36B1A609A0696FA8EFD`;
- result: pinned six-input preparation and fail-closed inventory inspection passed.

R3 independently remeasured the package at the recorded clean-worktree path and
obtained the same size and digest. Its product inputs map through remediation code
`a9d903364d7e1403d649895d77367a5806be1b0c` and clean product checkout
`5009781445c4db8c22d9073b9973ad90c9dd7c5f`.

The subsequent correction changes OpenAPI, an erased generated TypeScript interface and
a test, but no executable runtime or packaging-byte input. The documentation successors
also do not change packaged inputs. Under the proportionate closeout rule, rebuilding
the MSI would not add product evidence and is not required. Exact candidate CI still
rebuilt and checked the no-bundle Desktop product successfully.

### Windows lifecycle and Desktop continuity

The sanitized C2 evidence establishes:

- default and named Profile Start/Stop/Restart with authoritative final state;
- remediated Restart with old gateway identity gone, a new identity alive, stable total
  gateway count and final `RUNNING` for both targets;
- missing-login-item use of the fixed non-persistent path with no Hermes `ON_LOGIN`;
- close-to-tray, Owner-observed tray restore and explicit Quit;
- packaged current-user login start with only `--hidden`, refreshed registration,
  single-instance restoration and bounded Desktop-owned daemon exit;
- no observed UAC prompt, machine service registration, Instance login start or
  prohibited duplicate process.

The host session was administrative, so a separate standard-user-token session was not
claimed. The implementation, CI and observed behavior remain user-scoped and
non-elevating; the Phase 6 Spec does not define another redundant standard-user session
as a distinct exit criterion.

### Owner-authenticated real-channel evidence

The Owner's sanitized exact-MSI supplement records that, on MSI `B942...EFD`:

- real Weixin connection, sender pairing and local disconnect passed;
- real WeCom connection through the only approved typed Bot ID/Secret path and local
  disconnect passed; and
- no QR, Secret, pairing-code or account-information disclosure was observed.

The evidence retains no sensitive value. Its inspection record also states SQLite
integrity `ok`, `channel_bindings.metadata_json = '{}'`, no secret/QR/pairing-code
projection column, zero occurrences for the then-available exact Weixin credential
values in inspected YORVA surfaces and zero generic QR/secret markers.

A real message AI-response check, exact per-account Profile, exact action clock and an
after-removal exact-value WeCom comparison remain unreported and are not invented. As
R2 correctly established, these are evidence boundaries rather than Phase 6 manual/exit
criteria. They do not reopen H-01-R1 or create a new Gate condition.

## Earlier Finding Disposition

| Finding | R3 disposition | Evidence |
| --- | --- | --- |
| `AUDIT-006` H-01 — mandatory real-channel evidence | **CLOSED** | Exact-MSI Owner supplement; R2 independently accepted the Spec-required connection/pairing/disconnect flows. |
| H-02 — Restart false success | **CLOSED** | Stop, authoritative `STOPPED`, Start, authoritative `RUNNING`; regression and default/named Windows re-smoke. |
| H-03 — lifecycle parser contradiction | **CLOSED** | Exact signal grammar rejects contradictions, duplicates and changed signals while accepting pinned dynamic diagnostics. |
| H-04 — WeCom unknown success | **CLOSED** | Exact request correlation plus explicit `errcode: 0` before commit and negative fixtures. |
| H-05 — Channel cancel/commit race | **CLOSED** | Per-Instance commit-boundary serialization and deterministic connect/disconnect race regressions. |
| M-01 — lifecycle body not closed | **CLOSED** | Bounded closed `{}` decoder, OpenAPI/client body and protocol regressions. |
| M-02 — diff check | **CLOSED** | Exact baseline-to-candidate `git diff --check` passes. |
| M-03 — stale handoff | **CLOSED** | Handoff records immutable failures, remediation, evidence, CI/MSI state and R3 handoff. |
| `AUDIT-006R1` H-01-R1 — evidence not exact-MSI-bound | **CLOSED** | Candidate `13612db` binds the sanitized Owner outcomes to MSI `B942...EFD`; R2 accepted the required scope. |
| `AUDIT-006R2` M-01-R2 — `errorCode: false` | **CLOSED** | `4cff18b`; schema/client/Go alignment, focused type regression and exact-success CI verified above. |

The earlier Info observations about pre-Gate integration, pinned lifecycle diagnostics,
the WeCom wire shape and bounded QR query-cache lifetime remain non-defects. No earlier
severity was lowered to obtain this decision.

## Acceptance Matrix

| Phase 6 requirement | R3 result | Evidence / boundary |
| --- | --- | --- |
| Exact baseline/candidate, clean isolated branch | PASS | Git identity, remote branch and range checks. |
| Runtime-neutral Core; Runtime bundle owns feature selection | PASS | Small lifecycle/Channel contracts and `runtime.Registry.Bundle`; no Hermes implementation import in Core. |
| No generic process/service/shell/file API or Phase 7 scope | PASS | API, Runtime contracts and successor diff inspection. |
| Exact public Instance resolves to authoritative Profile | PASS | Application resolution, allowlisted native target validation and Profile-isolation coverage. |
| Lifecycle capability is dynamic and live state is typed | PASS | OpenAPI/Go/generated-TypeScript parity after `4cff18b`. |
| Start/Stop and Restart use authoritative fail-closed postconditions | PASS | Source/regressions and default/named Windows smoke. |
| Unknown/contradictory lifecycle evidence fails closed | PASS | Strict parser, command exit/stderr/bounds and fixtures. |
| No blind orphan Restart replay or duplicate-gateway success | PASS | Recovery logic, stop-before-start remediation, automated checks and Windows process observations. |
| Lifecycle/delete/config/Channel conflicts and cancellation | PASS | Narrow coordination, DB uniqueness and race regressions. |
| Lifecycle mutation request is bounded closed `{}` | PASS | HTTP/OpenAPI/generated client/tests. |
| Tray, close/restore/Quit, hidden user login and single instance | PASS | Sanitized C2 record, Rust tests and exact Windows CI. |
| Weixin QR is session-only, expiring and non-durable | PASS | 8 KiB memory broker, owner header, no-store response, metadata-only SSE and cleanup tests. |
| Real Weixin connect/pair/disconnect | PASS | Owner exact-MSI supplement; no sensitive value retained. |
| Pairing is Profile-exact, write-only and non-persistent | PASS | Adapter/API/Desktop/security regressions and schema inspection. |
| WeCom uses typed manual flow, fixed official WSS and verify-before-commit | PASS | Adapter source/fixtures and Owner exact-MSI connection/disconnect. |
| Channel `CONNECTED` remains distinct from lifecycle `RUNNING` | PASS | Separate contracts/resources and Desktop evidence. |
| No credential/QR/pairing disclosure in prohibited YORVA surfaces | PASS | Owner observation, schema/static tests and sanitized available-value inspection. |
| Empty and Phase 5 migration paths, constraints and cascade | PASS | Migrations 009/010, recorded local Gate and exact CI. |
| No active sealed-generation mutation | PASS | ADR-0008 authority path and unchanged inspected package inputs. |
| Exact-candidate CI | PASS | Run `32692682968`, exact SHA, all three jobs successful. |
| MSI/package mapping and inspection | PASS | 120,213,504-byte MSI, full `B942...EFD` digest and locked inventory. |
| Mandatory Windows and real-account evidence | PASS | Sanitized records with explicit non-gate limitations. |
| English/Chinese Specs and Roadmap/handoff synchronized | PASS | Status/evidence successor diff is materially mirrored and links resolve. |
| Failed audits preserved; early-main deviation disclosed | PASS | Byte hashes and handoff governance record. |
| Phase 7 not started | PASS | Range/API/Runtime/scope inspection. |

## Dimension Results

### 1. Scope — PASS

The Phase 6 candidate is limited to Runtime lifecycle, Weixin/WeCom, sender pairing and
Desktop continuity plus bounded audit remediation. Adjacent ancestry and unrelated user
work are disclosed. No Phase 7 or generic machine-control surface is included.

### 2. Correctness — PASS

Lifecycle mutations require authoritative observations; Restart proves a stopped state
before starting; unknown status fails closed; WeCom requires explicit correlated
success before commit; Channel cancellation cannot overwrite committed truth. The R2
type contract now matches actual responses.

### 3. Architecture — PASS

React uses the typed authenticated Node API, application code resolves small Runtime
features through the compile-time registry, and Hermes details remain adapter-owned.
Tauri remains limited to native Desktop/daemon-shell responsibilities.

### 4. Security — PASS

Loopback authentication/origin checks remain intact. QR, Channel credentials and
pairing codes are bounded, write-only or initiating-session-only as applicable and are
absent from durable/shared surfaces. Remote hosts and command shapes are fixed; no
hidden elevation, cross-Profile mutation, arbitrary command surface or secret
disclosure was found.

### 5. Data and Persistence — PASS

Migrations 009/010 enforce active-mutation uniqueness, one binding per
Instance/channel, closed states/types, foreign-key cascade and exactly empty metadata.
No secret, QR, pairing-code or PID column exists. Hermes remains authoritative.

### 6. Concurrency and Lifecycle — PASS

Per-Instance coordination protects lifecycle, delete, configuration and Channel
conflicts. Workers and commands have cancellation/timeout ownership, Channel terminal
truth is serialized with commit, process cleanup is tested, and recovery does not infer
or replay Restart blindly.

### 7. Protocol and Compatibility — PASS

OpenAPI remains the source of truth. Lifecycle requests are closed and lifecycle
responses are now correctly typed as nullable safe codes. Hermes and WeCom behavior is
pinned to qualified `0.20.2` surfaces without inventing incompatible extra wire rules.

### 8. Testing and Verification — PASS

Every blocking implementation finding has regression coverage. The R2 correction has a
focused semantic type test, proportionate local checks pass, and the fresh exact SHA has
green Web/API, Linux Go race and Windows native/Tauri CI. Mandatory MSI and manual
evidence is present.

### 9. Maintainability — PASS

The correction removes one stale constraint and adds one small regression. The wider
implementation uses cohesive existing boundaries and introduces no speculative
framework, generic manager layer or duplicate source of truth.

### 10. Documentation — PASS

Both Spec mirrors, Roadmap, handoff, ADR-0008, protocol/runtime/data/security documents
and sanitized evidence describe the same ownership and Gate history. All three failed
audits remain unchanged.

### 11. Dependencies / Supply Chain — PASS

The R2 correction adds or upgrades no dependency. Phase 6 dependencies are locked and
used for approved behavior. Exact CI dependency/vulnerability checks are green, and the
inspected MSI retains its pinned six-input inventory.

### 12. Operations / Diagnostics — PASS

Lifecycle and Channel work uses durable Operations, stable redacted error codes,
bounded external work and GET-based recovery. Events and logs retain safe identifiers
without raw Runtime output or credential material; Desktop-owned daemon shutdown is
bounded.

## Findings

### Critical

None.

### High

None.

### Medium

None. M-01-R2 is closed.

### Low

None.

### Info

- Phase 6 implementation reached `main` before an independent Gate. The deviation is
  preserved as history and is not treated as acceptance or repaired by rewriting Git.
- The exact MSI was not rebuilt after `4cff18b` because the correction changes only an
  erased TypeScript contract and its test, not executable/package bytes. Exact-candidate
  CI nevertheless rebuilt the no-bundle Desktop product successfully.
- The Owner did not report a real message AI response, exact per-account Profile/action
  clock or a post-removal exact WeCom sentinel. These are candid evidence limitations,
  not Phase 6 manual/exit criteria or deferred defects.

## Accepted Technical Debt

None.

## Required Fixes Before Next Phase

None for the Phase 6 candidate. C5 administration is still required before Phase 6 is
frozen and before any Phase 7 implementation may start.

## Gate Rationale

`PASS` is justified because the sole R2 finding is removed with a regression, the
published lifecycle contract now matches the Node DTO, fresh exact-candidate CI is
green, mandatory MSI/Windows/real-channel evidence is present, and the full review found
no unresolved Critical, High or other blocking finding. No acceptance criterion was
weakened and no missing fact was invented.

The earlier failed audits remain valid immutable records for their exact candidates.
This R3 decision supersedes them only for candidate
`859561712583c67031d8110812df3a4236be6908`.

## Merge / Freeze Eligibility

- Overall R3 audit: **PASS**.
- Code/security/lifecycle review: **PASS; no Critical/High**.
- Mandatory real-channel evidence: **PASS under the Phase 6 Spec**.
- Exact-candidate CI: **PASS** (`32692682968`).
- MSI/package mapping and inspection: **PASS** (`B942...EFD`).
- Windows lifecycle/Desktop continuity smoke: **PASS with recorded non-gate limits**.
- Owner authorization for commit/push/final-main/tag: **PRESENT**.
- C5 merge/final-main/freeze/tag: **ELIGIBLE**.
- Phase 7 start: **BLOCKED until the annotated Phase 6 baseline exists**.

## Next Step

Commit this report without changing audited production behavior, integrate the genuine
closeout branch into `main` without rewriting history, run final-main CI once, record
the exact SHA/run, update the two Specs and Roadmap to `COMPLETE / FROZEN`, and create
and verify the annotated
`phase-006-runtime-lifecycle-messaging-channels-baseline` tag. Rebuild the MSI only if a
subsequent C5 change alters packaged inputs. Stop after the freeze; do not begin Phase 7.

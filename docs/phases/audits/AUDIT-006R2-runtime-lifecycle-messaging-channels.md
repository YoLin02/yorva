# Phase 6 Runtime Lifecycle and Messaging Channels Re-audit R2

## Phase

Phase 6 — Runtime Lifecycle and Messaging Channels, fresh independent Closeout R2 audit
after immutable `AUDIT-006` and `AUDIT-006R1` failures.

## Baseline / Candidate

- Baseline tag: `phase-005a1-post-freeze-corrections-baseline`
- Baseline peeled commit: `995777528557fa564a4c42e14f8431b8ddbd20e8`
- Audited branch: `codex/phase6-closeout`
- Audited candidate: `13612dbf157637f59fc4c8a7e24ad9d02de2a681`
- Remediation product commit: `a9d903364d7e1403d649895d77367a5806be1b0c`
- Audit date: 2026-08-24 (Asia/Shanghai)

At audit start, `HEAD` and the branch remote both resolved to the stated candidate and
the worktree was clean. The baseline-to-candidate range contains 22 commits and changes
171 files (`11,962` insertions, `862` deletions). The candidate was not changed during
review; this report is the only intended worktree addition.

The immutable reports remained byte-identical during R2:

- `AUDIT-006-runtime-lifecycle-messaging-channels.md`: SHA-256
  `CBAF26C6F4B6C07EC9D0E67D19411D0E55C57E71059D1E62B8B7C2DB6383C560`;
- `AUDIT-006R1-runtime-lifecycle-messaging-channels.md`: SHA-256
  `971ED2C1BB19CB4796EE6036188E330D3D123DA0D1BD8A46A9FC5108DD32ABC1`.

## Auditor and Independence

The auditor is a fresh Codex R2 review context that did not implement Phase 6, perform
the R1 remediation, write either earlier audit, or collect the Owner smoke. The review
did not inherit R1 severity or evidence conclusions without checking them against the
governing Spec.

The auditor read `AGENTS.md`; the complete completion/audit/freeze prompt from the
permitted read-only original worktree; `docs/AUDIT_STANDARD.md`,
`docs/PHASE_GOVERNANCE.md`, `docs/DEVELOPMENT.md`, `docs/ARCHITECTURE.md`,
`docs/PROTOCOL.md`, `docs/RUNTIME.md`, `docs/DATA_MODEL.md`, `docs/SECURITY.md`,
`ROADMAP.md`; both Phase 6 Spec mirrors; the relevant Phase 5 specifications and audits;
ADRs 0001 through 0009; both immutable Phase 6 audits; and every Phase 6 qualification,
implementation, handoff, Owner-smoke and Windows-smoke record. Source and tests were
then inspected independently across the Phase 6 range.

No QR payload, Secret, pairing code, account identifier, Profile name, message content
or other sensitive value was requested, reconstructed or retained.

## Gate Decision

**FAIL**

No Critical or High finding was found. The code remediation previously made for
lifecycle Restart, status parsing, WeCom verification, Channel cancellation/commit and
closed request bodies is acceptable, and the mandatory real-channel evidence is now
sufficient under the actual Phase 6 Spec. Exact-candidate CI, the mapped MSI and Windows
smoke are green.

The exact candidate nevertheless has one independently reproduced **Medium** protocol
finding: an obsolete `const: false` was displaced from the former Phase 4 lifecycle
capability literal and is now attached to `Lifecycle.errorCode`. Consequently the
OpenAPI-generated Desktop type says `errorCode: false`, while the Node implementation
correctly returns `string | null`. OpenAPI is the Desktop/Node schema source of truth,
and Phase 6 explicitly requires a typed lifecycle view. This is a real, narrowly scoped
contract defect, not a severe vulnerability.

`AUDIT_STANDARD.md` says a Medium should be fixed before the gate when practical. This
one is local and practical to fix, and it has not been explicitly accepted in its
specific reproduced form with an owner and resolution trigger. R2 therefore does not
convert it into accepted debt merely to obtain a pass.

**C5 merge/final-main/freeze eligibility: NO for this candidate.** Fix M-01-R2, add the
small semantic regression, regenerate the client schema, obtain fresh exact-success CI
for the successor, and use a fresh re-audit context. Phase 7 remains blocked.

## Verification Evidence

### Candidate and range integrity

- `git rev-parse HEAD`, branch/upstream, baseline peel, range log and status established
  the exact candidate and clean starting state.
- `git diff --check
  995777528557fa564a4c42e14f8431b8ddbd20e8..13612dbf157637f59fc4c8a7e24ad9d02de2a681`
  passed with exit `0`.
- The handoff discloses the historical pre-audit integration and adjacent commits
  `ce1a328` and `3494033`; neither is misattributed to Phase 6. The separate dirty
  installation-source/embedded-Python/packaging/UI worktree is absent from this clean
  candidate.
- No Phase 7 Runtime upgrade/repair/uninstall, Skills, MCP, backup/restore, additional
  Channels, Cloud or generic process/shell/file API was found.

### Exact-candidate CI

Independent read-only GitHub Actions API inspection established:

- run `32690884232`: event `push`, attempt `1`, `completed`, `success`, head branch
  `codex/phase6-closeout`, exact head SHA
  `13612dbf157637f59fc4c8a7e24ad9d02de2a681`;
- Web and API contract job `97324214804`: `success`, including dependency install/audit,
  API lint/generation/drift, typecheck, lint, tests and Web build;
- Go Node job `97324214903`: `success`, including `go test -race ./...`, vet,
  `govulncheck` and build;
- Windows Desktop native shell job `97324215036`: `success`, including sidecar build,
  Windows lifecycle smoke, MSI-inspector negative tests, Rust format/test/audit,
  clippy/check and Tauri no-bundle build.

Run URL: `https://github.com/YoLin02/yorva/actions/runs/32690884232`.

The complete unchanged Gate was not redundantly rerun. R2 ran only proportionate local
read-only/source checks plus OpenAPI lint/generation to reproduce M-01-R2. Lint and
generated-schema drift both pass because they validate internal agreement with the
misplaced constraint; they do not validate the response value against the Go DTO.

### MSI and product-input mapping

The inspected remediation package is:

- file: `Yorva_0.3.2_x64_en-US.msi`;
- size: `120,213,504` bytes;
- SHA-256: `B942E637BE9BC59D6C5B603DA117C7AC9646CD254E88F36B1A609A0696FA8EFD`;
- package checks: pinned six-input preparation and fail-closed MSI inventory inspection
  passed.

R2 independently hashed the MSI at
`apps/desktop/src-tauri/target/release/bundle/msi/Yorva_0.3.2_x64_en-US.msi` and obtained
the same size and digest. The evidence maps its product inputs through clean product
checkout `5009781445c4db8c22d9073b9973ad90c9dd7c5f`, containing remediation
`a9d903364d7e1403d649895d77367a5806be1b0c`. Successors through `13612db` change only
documentation/evidence/audit inputs, so the candidate's product inputs are unchanged.
The unrelated dirty-worktree MSI with the same filename is explicitly excluded.

### Windows lifecycle and Desktop continuity

The sanitized Windows record establishes:

- default and named Profile Start/Stop/Restart, with authoritative final state;
- remediated default/named Restart transitions with old gateway identity gone, one new
  gateway alive and final `RUNNING`;
- Desktop close-to-tray without stopping managed gateways, Owner-observed tray restore
  and explicit Quit;
- packaged per-user login start with only `--hidden`, stale-item path recreation,
  single-instance restoration and bounded Desktop-owned daemon exit;
- no UAC/elevation prompt, no machine service/startup entry and no Hermes startup-policy
  change.

The host session was administrative, so standard-user-token behavior was not manually
demonstrated. That limitation remains explicit; static/CI inspection and the observed
absence of UAC or machine-level installation support the approved per-user design. The
Spec does not make a second redundant standard-user session a separate exit criterion.

### Owner-authenticated real-channel evidence

The candidate preserves the Owner's sanitized confirmation that Weixin connection,
pairing and disconnect, and WeCom connection and disconnect all passed on MSI
`B942...EFD`, with no observed QR, Secret, pairing-code or account-information
disclosure. The evidence record contains the full digest but no sensitive value.

This establishes the required real Weixin connect/disconnect and sender-pairing flow.
The only supported Weixin connection path is the QR flow, so the attested successful
connection supports the required scan/confirm/connect outcome without inventing a raw
substep transcript. The only approved WeCom path is typed Bot ID/Secret verification,
so the attested connection supports that required auth path without retaining the
credential.

The Owner did not attest a real message receiving an AI response, an exact Profile used
for each account flow, or an exact action clock. Those facts remain explicitly
`Not evidenced`; none is invented. Phase 6 Spec lines 803-811 require real Weixin
scan/confirm/connect/disconnect and the approved WeCom auth path, and lines 837-851 make
those auth flows and Weixin pairing exit criteria. They do **not** require a real message
AI-response check or account-flow Profile/date transcript. Chat/message UI is a Phase 6
non-goal. The closeout prompt required the response fact to be reported honestly, not
silently converted into an additional Phase Gate criterion.

The sanitized inspection also records SQLite integrity `ok`, binding metadata exactly
`{}`, no secret/QR/pairing-code projection column, exact-value zero occurrences for four
then-available Hermes-native Weixin tokens across inspected YORVA surfaces, and zero
generic QR/secret markers. No WeCom value remained after removal, so no exact-value
WeCom claim is made. Owner observation, bounded static tests and the available-value
inspection together satisfy the Spec's no-disclosure criterion without fabricating a
missing sentinel.

## Acceptance Matrix

| Phase 6 requirement | R2 result | Evidence / boundary |
| --- | --- | --- |
| Qualified `lifecycle: true` capability | PASS | Runtime bundle owns the qualified lifecycle feature; Desktop receives capability data. |
| Default/named Start/Stop/Restart and live state | PASS | Exact-product Windows smoke and remediated restart re-smoke. |
| Manual-only lifecycle/recovery and authoritative postconditions | PASS | Operations/recovery source review, race CI and Windows smoke. |
| Desktop close preserves gateways | PASS | Close-to-tray record; gateway remains Runtime-owned. |
| Tray restore/Quit, hidden user-login start, single instance | PASS | Sanitized C2 Windows/Owner evidence; no machine startup policy. |
| Real Weixin mandatory auth and disconnect | PASS | Owner-attested exact MSI connection/disconnect; only supported connect path is QR. |
| Weixin sender pairing from Desktop | PASS | Owner-attested exact MSI pairing plus adapter/API/Desktop redaction tests. |
| Real WeCom approved auth and disconnect | PASS | Owner-attested exact MSI connection/disconnect; only approved path is typed manual flow. |
| Real message receives AI response | NOT EVIDENCED / NOT A PHASE 6 GATE | Reported honestly; absent from Spec manual evidence and exit criteria. |
| Channel state distinct from lifecycle state | PASS | Separate API/application contracts and Desktop presentation. |
| No QR/channel credential in prohibited surfaces | PASS | Owner observation, schema/static/security tests and sanitized available-value inspection. |
| Profile-exact mutation and no sealed-generation mutation | PASS | Public ID resolution, accepted installation/native ID mapping, adapter-owned Profile paths and packaging evidence. |
| Typed lifecycle OpenAPI/Desktop contract | **FAIL** | M-01-R2: generated `Lifecycle.errorCode` is `false`, not `string | null`. |
| Mandatory automated Gate and Windows smoke | PASS | Exact run `32690884232`, MSI inspection and Windows evidence. |
| Independent audit PASS or accepted conditional PASS | **FAIL** | This R2 decision is `FAIL` on M-01-R2. |
| Merge/final-main/freeze/tag | PENDING / NOT ELIGIBLE | C5 cannot start for this exact candidate. |

## Dimension Results

### Scope — PASS

Implementation remains within lifecycle, Weixin/WeCom, pairing and Desktop continuity.
Hermes-specific mechanics stay adapter-owned, adjacent work is disclosed, and no Phase 7
feature is present.

### Correctness — PASS

The supported runtime workflows have deterministic Operations, authoritative lifecycle
postconditions, explicit Restart stop-then-start behavior, strict status parsing,
verification-before-commit for WeCom, targeted disconnect and stable error mapping.
M-01-R2 is a published type-contract defect rather than a demonstrated failure of the
current Node/Desktop runtime workflow.

### Architecture — PASS

React calls the authenticated typed Node API; application code resolves small lifecycle
and Channel features through the Runtime bundle; Hermes/Profile/credential details stay
under the Hermes adapter. No generic OS process, service, shell or file API was added.

### Security — PASS

Management routes remain loopback/bearer/origin protected. QR retrieval is
initiating-session-only, bounded and `no-store`; SSE contains metadata only. Secrets and
pairing codes are write-only and absent from SQLite, ordinary reads, Operations/events
and logs. Fixed remote endpoints, TLS bounds, correlation and response checks fail
closed. No disclosure, cross-Profile mutation, arbitrary command surface, hidden
elevation or sealed-generation mutation was found.

### Data and Persistence — PASS

Migrations 009/010 provide active-operation uniqueness, one binding per
Instance/channel, constrained safe metadata and foreign keys. Empty/prior-schema
migration coverage and exact CI pass. Hermes remains credential/state authority and no
channel secret column was introduced.

### Concurrency and Lifecycle — PASS

Per-installation/Instance coordination covers lifecycle, delete/config/channel
conflicts; idempotency and terminal transitions are durable; Channel cancellation is
synchronized with verification/commit; workers have bounded context ownership and
recovery does not infer Restart success from final `RUNNING` alone.

### Protocol and Compatibility — FAIL

Routes, bodies, authentication, bounds, stable errors, QR session ownership and pinned
Hermes/WeCom semantics otherwise pass. M-01-R2 makes the published lifecycle response
type contradict the Node DTO and must be corrected before freeze.

### Testing and Verification — PASS

Exact-candidate CI and Windows/manual evidence cover the mandatory matrix. The complete
unchanged Gate was not rerun. M-01-R2 exposes a narrow semantic contract-test gap, for
which a focused regression is required with the fix.

### Maintainability — PASS

The new feature interfaces remain small and substitution-driven. Runtime-specific
parsing, endpoints and credential authority are cohesive and isolated; no speculative
framework or global service manager was introduced.

### Documentation — PASS

Specs, ADR-0008, runtime/protocol/data/security docs, handoff, smoke evidence and roadmap
describe the implemented trust and ownership model and preserve both failed audits.
The OpenAPI schema itself is assessed under Protocol as M-01-R2.

### Dependencies / Supply Chain — PASS

`qrcode.react`, the Tauri tray feature and pinned autostart/single-instance plugins are
used for approved Phase 6 behavior and locked. pnpm/Cargo audits and exact CI are green;
no unapproved framework or unrelated mass upgrade was found.

### Operations / Diagnostics — PASS

Durable Operations, safe correlation/error codes and authoritative GET recovery make
failures observable without raw output or credentials. QR-ready events are metadata-only,
and lifecycle/channel terminalization remains bounded.

## Findings

### Critical

None.

### High

None.

### Medium

#### M-01-R2 — Lifecycle OpenAPI `errorCode` constraint is attached to the wrong field

Evidence:

- `api/openapi.yaml:1710-1734` declares `InstanceCapabilities.lifecycle` as an ordinary
  boolean, then attaches `const: false` to `Lifecycle.errorCode` even though its declared
  type is `[string, "null"]`.
- Baseline `9957775` correctly attached that literal to the then-unsupported
  `InstanceCapabilities.lifecycle`. Batch 2 inserted the new `Lifecycle` schema between
  the property and the old literal without removing the literal.
- `apps/desktop/src/api/generated/schema.ts:695-703` therefore generates
  `Lifecycle.errorCode: false`.
- `services/node/internal/transport/httpapi/instances.go:61-65,166-179` returns
  `*runtime.ErrorCode`, encoded as a safe string or `null`.
- `services/node/internal/app/lifecycle.go:30-47` deliberately produces a real stable
  error code when live status observation fails.
- `docs/PROTOCOL.md:172-177,385-397` says lifecycle GET returns a safe error code and
  OpenAPI is the Desktop/Node schema source of truth.

Impact: runtime JSON remains correct and current Desktop control flow does not depend on
the field's generated literal type, so the core correctness/trust model is not
compromised. The published/generated contract is nevertheless impossible for normal
responses and can mis-type present or future consumers. This is bounded protocol drift,
not a Critical/High security or lifecycle defect.

Required correction: remove the displaced `const: false`, preserve the now-dynamic
capability boolean, regenerate the TypeScript schema, and add a focused contract
assertion that `Lifecycle.errorCode` accepts `string | null` and is not a boolean
literal. Run the relevant API generation/drift/typecheck/test subset, then obtain fresh
exact-candidate CI because the candidate changes.

### Low

None.

### Info

#### I-01-R2 — Earlier audit H-01 scope exceeded the actual Phase 6 exit text

The Owner's exact-MSI supplement closes the Spec-required real Weixin connection,
pairing/disconnect and approved WeCom connection/disconnect evidence without exposing
sensitive values. A real-message AI response, per-account exact Profile and exact action
clock remain unproved and are reported as such, but Phase 6 Spec manual evidence
803-811 and exit criteria 837-851 do not make them gate requirements. R2 does not carry
that over-broad portion of H-01 forward.

#### I-02-R2 — Phase 6 implementation reached `main` before its independent Gate

The handoff and roadmap preserve this governance deviation and do not reinterpret early
integration as acceptance. No history rewrite or tag movement is proposed. A clean
closeout branch and immutable failed audits preserve the decision trail.

#### I-03-R2 — QR payload has a bounded in-process query-cache lifetime

The Desktop obtains QR bytes through a component-owned TanStack Query while the connect
Operation is active. The data is not written to browser storage, SQLite, SSE, logs or a
shared API response; server session ownership, expiry/terminal clearing and normal query
garbage collection bound it. Explicit query removal on terminal/unmount could tighten
heap lifetime later, but no disclosure or Phase 6 blocking condition was demonstrated.

## Accepted Technical Debt

None. M-01-R2 is local and practical to fix before freeze and has not been specifically
accepted as deferred work.

## Limitations and Residual Risk

- R2 did not possess real Weixin/WeCom accounts and relied on the exact sanitized Owner
  attestation only for the account-dependent actions it actually states.
- A real message AI-response check, exact account-flow Profile and exact action clock are
  not established and are not inferred.
- No WeCom Secret remained for post-removal exact-value scanning; the report does not
  claim otherwise.
- Standard-user-token behavior was not separately demonstrated because the Windows host
  session was administrative; observed behavior and implementation remain per-user and
  non-elevating.
- R2 did not redundantly rerun the unchanged full Gate; it independently queried the
  exact external result and ran only proportionate checks needed to validate evidence
  and reproduce M-01-R2.
- No sensitive value is included in this report.

## Required Fixes Before Next Phase

1. Correct M-01-R2 in OpenAPI without restoring the obsolete Phase 4
   `lifecycle: false` capability literal.
2. Regenerate the Desktop schema and add a focused semantic contract regression for
   lifecycle `errorCode: string | null`.
3. Run the relevant API/typecheck/test subset and fresh exact-candidate CI. Do not reuse
   run `32690884232` for the changed successor.
4. Preserve `AUDIT-006`, `AUDIT-006R1` and this R2 report unchanged. Use a fresh context
   for the successor re-audit.
5. Enter C5 final-main/freeze/tag only after that audit returns `PASS` or a genuinely
   Owner-accepted `PASS WITH CONDITIONS` and every new exact-candidate check passes.

No additional real-account flow is required by this finding, and the completed flows
should not be repeated merely to repair a non-product-input OpenAPI contract unless the
fix changes product behavior beyond the stated schema correction.

## Gate Rationale

The exact candidate has no unresolved Critical/High, no demonstrated severe
vulnerability and no remaining mandatory real-channel evidence gap under the actual
Spec. CI, MSI, lifecycle/Desktop smoke, authentication, pairing and disclosure evidence
are sufficient. R1's broader message/Profile/date requirements are not retained.

However, Phase 6 explicitly replaces the unsupported lifecycle contract with a typed
view, and OpenAPI is the protocol source of truth. The generated `errorCode: false`
contract is a concrete, reproducible error. It is bounded enough to remain Medium, but
small and practical enough that freezing it would conflict with the audit standard's
fix-before-gate direction. A current `PASS` would therefore be premature; a Critical or
High label would be disproportionate.

## Merge / Freeze Eligibility

- Code-level security/lifecycle review: **PASS; no Critical/High**.
- Mandatory real-channel evidence: **PASS under Phase 6 Spec**.
- Exact-candidate CI: **PASS** (`32690884232`).
- MSI/package mapping and inspection: **PASS** (`B942...EFD`).
- Windows lifecycle/Desktop continuity smoke: **PASS with recorded limitations**.
- Protocol source-of-truth contract: **FAIL** (M-01-R2).
- Overall R2 audit: **FAIL**.
- C5 merge/final-main/freeze/tag for `13612db`: **NOT ELIGIBLE**.
- Phase 7 start: **BLOCKED**.

## Next Step

Stop before C5. Apply only the narrow OpenAPI/type-generation correction and focused
regression, obtain fresh exact-candidate CI, and request a fresh successor audit. Do not
rewrite history, move an existing tag, alter the immutable FAIL reports, rerun unaffected
manual account smoke, or begin Phase 7.

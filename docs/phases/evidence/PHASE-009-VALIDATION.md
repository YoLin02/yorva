# Phase 9 validation evidence

Date: 2026-09-10. Final result: **G1 PASS / G2 PASS / G3 PASS**.
The accepted product candidate is `b958801a91234b5a602d652035e5c4f3e3dc8242` on `codex/phase9-openclaw`.
The documentation-only freeze successor is identified by
`phase-009-openclaw-baseline`; see the [baseline record](PHASE-009-BASELINE.md).

## Candidate and scope

P8 working start: `e95ed31d298c2548556e1295aacf7f84002b74ee`, with product code
identical to the frozen P8 baseline. P9 adds OpenClaw discovery, profile Instance
management, authenticated lifecycle, Runtime-scoped routing/recovery and Desktop
selection. It adds no product dependency or database migration. Optional OpenClaw
models, channels, Skill/MCP, backup mutation, installer and upgrade are unavailable.

## Final verification matrix

| Gate/check | Actual final result |
| --- | --- |
| G1 — real Windows coexistence | PASS at `2026-09-10T12:24:48.8913044Z`; disposable Windows 11 x64 medium integrity; Hermes 0.20.5/Python 3.11.15 and OpenClaw 2026.9.3/Node 24.16.0. One Hermes plus two OpenClaw instances cover same-name identity, authenticated start, restart, daemon reconnect with Runtime survival, isolated stop/delete and no login entry. |
| Go regression | CI #105 PASS: full `go test -race ./...`, vet, govulncheck and daemon build. Native Windows OpenClaw/app/daemon tests and disposable encrypted Restore also PASS. |
| Desktop/API | CI #105 PASS: frozen install, pnpm audit, OpenAPI lint/regeneration consistency, typecheck, lint, 151 tests in 25 files and build. |
| Windows native shell | CI #105 PASS: all five bootstrap lifecycle scenarios, MSI inspection negative tests, Rust fmt, 33 native tests, cargo audit, clippy with warnings denied, check, Tauri build and isolated Desktop/daemon recovery. |
| Final Windows package | MSI #43 PASS on the exact product candidate: ordinary per-user x64 build, embedded-payload/installer inspection and artifact upload. `YORVA_0.4.0_x64_en-US.msi`, 149,987,328 bytes, Authenticode `NotSigned`; internal test package. |
| H2/H3 process regressions | All four strict Windows command/process tests passed 20 times each after H3 (80 executions, 54.930 seconds); full OpenClaw suite PASS (3.854 seconds) and vet PASS. Final CI runs the same regressions. |
| Desktop UI | English/Chinese switching, inventory, capability-filtered management, create dialog, protected default and health detail inspected using disposable API fixtures. This is UI evidence; real Runtime evidence is the separate G1 record. |
| G3 — review | Fresh-context actual-source review and dated re-audit PASS; M1, L1, H1, H2 and H3 closed. Zero unresolved CRITICAL/HIGH/MEDIUM/LOW findings; no correctness/security defect deferred. |

CI: [#105](https://github.com/YoLin02/yorva/actions/runs/34475486005).
Package: [MSI #43](https://github.com/YoLin02/yorva/actions/runs/34475485956).
The [structured CI/MSI record](PHASE-009-FINAL-CI-MSI.json) retains exact run/job,
artifact identity, digest and expiry. The final MSI SHA-256 is
`9901a8e0ece982413224bc489ad9c42ab0454e29465ae72ffad04021ac3d0e68`. Its archive digest is a separate value.

The final [G1 R2 record](PHASE-009-WINDOWS-G1-R2.json) retains all 20 observations
and log digest. Its sidecar SHA-256 is `5a3c0afb46a43d74bffa054e4082e8a0117d0561fc357f2a2f46b9103a6adf5f`,
built from clean `b958801a91234b5a602d652035e5c4f3e3dc8242`. This native-test sidecar is not represented as
the remote MSI binary. Historical [G1](PHASE-009-WINDOWS-G1.json) and
[G1 R1](PHASE-009-WINDOWS-G1-R1.json) remain unchanged and do not replace R2.

No normal host Runtime profile was used; no host reboot occurred. The disposable
guest shut down after completion. No production credentials, model quota or Channel
login were required for the delivered lifecycle scope.

## Failures, corrections and evidence retained

- Native fixture setup initially exposed a relocated uv launcher and slow Python
  imports. The guest uses an official pip-generated entry point in its venv; no
  Hermes fork or source patch was introduced. Fixture failures do not count as PASS.
- The real OpenClaw package metadata is 135,311 bytes. Its initial 128 KiB guard
  rejected that valid release. The corrected 256 KiB bound has actual-size and
  oversize regression tests.
- Native attempt 8 created all three instances but failed Hermes startup with
  `LIFECYCLE_POSTCONDITION_FAILED`. Diagnostic attempt 11 measured normal Gateway
  readiness after about 94 seconds. The 15-second post-launch wait became a bounded
  120-second start wait with authoritative readback and cancellation regression;
  STOPPED remains bounded at 15 seconds. Final real G1 passes this path.
- The shared 30-second startup inventory budget keeps the Desktop bootstrap bounded.
  Recovery still checks each previously accepted installation; tests prove a healthy
  Runtime cannot mask a failed Runtime and management remains accessible.
- [CI #101](https://github.com/YoLin02/yorva/actions/runs/34471364540) rejected
  inherited Vitest/js-yaml development dependencies under newly published advisories.
  `cc36eaa` patches Vitest 4.1.10 to 4.1.11 and the exact Redocly parent's js-yaml
  4.3.1 to 4.3.2. Frozen install, dependency audit and full Desktop/API regression
  pass. No gate or advisory threshold was weakened.
- M1 and L1 from the first 12-dimension review were fixed in `8fd2bcb`: independent
  Runtime name-rule fakes/tests and accurate native identity/protection documentation.
- H1: [CI #102](https://github.com/YoLin02/yorva/actions/runs/34471758200) exceeded
  the existing 45-second bootstrap deadline. Its exact cause remains unproven. The
  harness now drains stderr concurrently, reports scenario timing and limits failure
  tails to 8 KiB. Deadline, health and all five lifetime assertions remain. Native
  attempt 14 [passed](PHASE-009-WINDOWS-BOOTSTRAP.json), then final CI #105 passed.
- H2: [CI #103](https://github.com/YoLin02/yorva/actions/runs/34473025392) reported
  a descendant surviving command cancellation. Closing the kill-on-close Job did not
  synchronously prove process exit. The initial terminate/accounting-only correction
  also failed strict repetitions. `8e3e03f` retains owned process handles before
  termination and verifies their signals plus empty Job. G1 R1 passed on that
  candidate; no test was relaxed and no arbitrary PID kill was added.
- H3: fresh source review of `8e3e03f` found output-close followed by synchronous
  Wait could bypass cancellation. The finite regression reproduced a 500 ms deadline
  returning after 3.070685 seconds. `b958801` preserves the context select while
  joining the process; the unchanged test and all 80 process scenarios pass. Final
  CI/MSI and real G1 R2 qualify that correction.

The [audit](../audits/AUDIT-009-openclaw-second-runtime.md) preserves the first FAIL,
each finding, unsuccessful correction evidence and final closure. Historical MSI
39–42 and pre-H3 checks are not delivered as the final package or substituted for it.

## Limits and maintenance observations

The existing Vite chunk-size advisory remains non-blocking. Govulncheck reports zero
reachable vulnerabilities, with 17 advisories in required modules outside called
code; pnpm and cargo audit gates pass. This does not claim an entirely advisory-free
transitive graph. The narrow js-yaml override should be removed when its parent
updates that dependency (repository maintainer, during the relevant dependency update).

OpenClaw Windows x64 `2026.9.3` is the qualified target. Production signing, public
distribution, broader platform/version qualification and optional Runtime feature
parity remain separate work under the Spec. P8's unchanged release/security controls
are inherited; affected native/recovery checks were executed. No P10 work begins
as part of this freeze.

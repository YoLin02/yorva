# YORVA Phase 3 Independent Audit — Hermes Installation

> Audit ID: `AUDIT-003`
> Date: 2026-08-17
> Auditor role: independent Phase 3 review context
> Phase: Phase 3 — Hermes Installation
> Baseline: `phase-002-hermes-discovery-baseline-r1` → `5b89d22ed5e7ae3f4374a26f0fcda54bdabc6bf9`
> Implementation branch: `fix/phase3-managed-node-prerequisites`
> Committed implementation HEAD reviewed: `086ff2a901d76764608d545b12980dd2602122be`
> Exact-commit CI: run `32011052252` — **FAIL**
> Gate: **FAIL**

## 1. Executive decision

Phase 3 does not pass its gate. Phase 4 planning and implementation remain blocked.

The immutable Phase 2 R1 baseline is present and the Phase 3 work remains on a feature branch. The implementation has good foundations: direct argv execution, a suspended Windows child assigned to a kill-on-close Job Object before resume, immutable Hermes/Node/npm artifact pins, bounded archive extraction, fixed user-scope paths, authenticated loopback APIs, and exclusion of the official `desktop`, `platform-sdks`, `configure`, and `gateway` stages.

Those strengths do not close the required success and security contracts. The audit confirmed six High and four Medium findings. In particular, conflicting install Operations can run concurrently against the same Hermes tree, cancellation is persisted before process cleanup is known to be complete, post-check can accept a Python fallback while the user-facing `hermes` launcher is missing, and the MSI build is not fail-closed when its gitignored embedded payloads are absent.

The current candidate is also not immutable: five tracked Desktop files are modified, one new test is untracked, and an extracted WiX file is untracked. The only exact-commit CI run for `086ff2a` failed its Windows native job.

## 2. Audit object and repository state

Observed at final review:

```text
branch:      fix/phase3-managed-node-prerequisites
HEAD:        086ff2a901d76764608d545b12980dd2602122be
origin ref:  086ff2a901d76764608d545b12980dd2602122be
main:        5b89d22ed5e7ae3f4374a26f0fcda54bdabc6bf9
baseline:    phase-002-hermes-discovery-baseline-r1
tree:        dirty
```

Uncommitted implementation changes observed above HEAD:

```text
M  apps/desktop/src/App.tsx
M  apps/desktop/src/api/client.ts
M  apps/desktop/src/components/HermesPrerequisitePanel.test.tsx
M  apps/desktop/src/components/HermesPrerequisitePanel.tsx
M  apps/desktop/src/i18n.ts
?? apps/desktop/src/App.prereq.test.tsx
?? YORVA_0.3.0_x64_en-US.wxs
```

The tracked patch is approximately 217 added/changed lines and changes prerequisite recovery behavior. It is therefore material implementation, not harmless audit residue. This audit report is the auditor's only repository addition.

Phase 3 changes from the frozen baseline span 95 files and approximately 13,611 insertions. Generated API code, translations, licenses, and Specifications account for several of the largest files; they were not treated as monolithic application logic merely because of line count.

## 3. Dimension results

| Dimension | Result | Summary |
| --- | --- | --- |
| Scope | PASS | No Profile, credential, channel, gateway, lifecycle, Skills/MCP, Cloud, dynamic plugin, or Phase 4 implementation was found. |
| Architecture | FAIL | Conflicting mutation ownership is not coordinated across the install and prerequisite use cases. |
| Security | FAIL | Cancellation terminality and raw command-output persistence violate the approved boundary. |
| Data / persistence | FAIL | The database does not enforce one active prerequisite or a shared Hermes mutation lock. |
| API / protocol | FAIL | Prerequisite state and retry semantics do not faithfully represent the implemented operation. |
| Runtime integration | FAIL | The post-check can accept an installation whose public launcher is absent. |
| Desktop UX | FAIL | The pre-Hermes prerequisite action leads to a predictable failure; the attempted recovery patch is uncommitted. |
| Process lifecycle | FAIL | Process-tree containment is strong after spawn, but cancellation is declared terminal before the worker proves cleanup. |
| Packaging / supply chain | FAIL | Pins and hashes are strong, but clean-checkout MSI construction does not require embedded artifacts. |
| Maintainability | FAIL | One new 439-line file combines too many responsibilities and lacks proportional tests. |
| Verification | FAIL | Local subsets pass, but exact-commit CI failed and the final Desktop patch has no exact-commit CI. |
| Documentation | FAIL | The parent Spec, amendments, status, stage table, timeouts, and completion evidence are not reconciled. |

## 4. Findings

### HIGH-001 — Hermes installation and prerequisite Operations are not mutually exclusive

`RuntimeInstall.Start` checks only `ActiveRuntimeInstall`, while `StartPrerequisites` checks only `ActiveHermesPrerequisite`. Both workers mutate `%LOCALAPPDATA%\hermes`; the prerequisite worker also runs `npm ci` inside `%LOCALAPPDATA%\hermes\hermes-agent` while the install worker materializes and runs stages in that same tree.

The database has a partial unique index only for `operation_type = 'runtime.install'`. It has no unique constraint for `hermes.prerequisites`, and no constraint spanning both conflicting operation types. Two simultaneous prerequisite requests can therefore both pass the read-before-create check and insert active rows. A Runtime install and prerequisite install can also run concurrently by design.

Impact:

- directory replacement and `npm ci` can race with source materialization, venv creation, PATH/config stages, retry cleanup, or cancellation;
- a second prerequisite request can create a second worker instead of returning the active Operation;
- racing persistence errors are not normalized to the documented in-progress response;
- the dirty Desktop patch explicitly leaves “Install Hermes” available while a prerequisite Operation is running, making the cross-operation race user reachable.

Required closure:

- define one narrow Hermes-host mutation conflict key covering `runtime.install` and `hermes.prerequisites`;
- enforce it atomically in SQLite, not only with check-then-insert application reads;
- return the active Operation ID for every conflicting request;
- add concurrent application, persistence, and HTTP tests for same-type and cross-type starts;
- prove unrelated read-only discovery remains available.

### HIGH-002 — Cancel reports `CANCELLED` before process-tree cleanup completes

`RuntimeInstall.Cancel` calls `stopWorker`, which removes and invokes a `context.CancelFunc`, then immediately writes `CANCELLED` to SQLite. The worker map contains no completion signal and `Cancel` never waits for the worker, command runner, Job Object cleanup, or direct child reap.

The command runner itself has strong cleanup behavior, but that cleanup runs asynchronously after context cancellation. The public Operation can therefore be terminal while installer or npm descendants are still being terminated. This contradicts the Phase 3 rule that cancellation becomes terminal only after all owned processes are stopped and reaped.

Required closure:

- track worker cancellation and completion separately;
- make the worker the single owner of the terminal transition, or have `Cancel` wait for a bounded completion acknowledgement;
- preserve idempotent cancellation without racing two terminal writes;
- add real Windows tests asserting that the API/DB cannot observe `CANCELLED` before child and grandchild exit.

### HIGH-003 — Phase 3 can report success while the user-facing `hermes` command is missing

The reviewed official `path` stage creates `%LOCALAPPDATA%\hermes\hermes-agent\bin` and copies `venv\Scripts\hermes.exe` only when that source launcher exists. It still emits “hermes command ready” when no launcher was copied.

YORVA's `postcheckAccepted` then accepts any Phase 2 `SUPPORTED` candidate contained anywhere under the managed installation/home. The Phase 3 worker test explicitly models the accepted path as `venv\Scripts\python.exe`, i.e. the Phase 2 compatibility fallback rather than the public launcher.

Consequently, a valid Python-module fallback can make `runtime.install` `SUCCEEDED` even though a new terminal cannot resolve `hermes` and the documented launcher path is empty. This matches the Owner's observed failure and defeats the Phase 3 installation success promise.

Required closure:

- keep the Phase 2 Python fallback for discovering compatible pre-existing installations;
- tighten the Phase 3 postcondition to require the canonical managed `bin\hermes.exe`, verified as a regular, contained, runnable launcher for `0.20.2`;
- verify the expected user PATH entry without relying on the current process's stale PATH;
- make missing launcher creation a post-check failure, not success;
- add regression tests for Python-fallback-only, venv-only, empty-bin, stale-shell PATH, and canonical-bin success.

### HIGH-004 — The prerequisite workflow false-fails before Hermes and can false-pass after a partial npm install

Desktop exposes the Node/npm prerequisite action even when Hermes is `NOT_INSTALLED`. `NodeHost.Apply` always proceeds from Node and npm materialization to `installDependencies`. That function requires the Hermes tree's `package-lock.json`, so the fresh-machine pre-Hermes action predictably ends in `RUNTIME_HERMES_NODE_DEPS_FAILED` after Node/npm may already have installed successfully.

The uncommitted UI patch tells the user that Node is not required before installing Hermes and suggests cancellation, but it does not remove or correctly sequence the failing action. It also permits the conflicting Hermes install described in HIGH-001.

In the opposite direction, `inspectDeps` reports `READY` from only two filesystem existence checks: `package-lock.json` is a regular file and `node_modules` is a directory. A failed or externally created partial `node_modules` directory can therefore suppress Retry and appear healthy. The documented `FAILED` and `TIMED_OUT` component states are never produced by the live inspection path.

Required closure:

- define an explicit sequence: either install only managed Node/npm before Hermes and defer dependency work, or make the prerequisite mutation available only after the pinned Hermes tree exists;
- never label a successfully installed Node/npm pair as failed merely because Hermes is not installed yet;
- derive dependency readiness from a trustworthy successful postcondition tied to the pinned lock/install identity, not directory existence;
- surface `FAILED` and `TIMED_OUT` consistently from authoritative Operation/health state;
- add fresh-machine, partial-npm, timeout, failed-then-retry, daemon-restart, and supported-Hermes tests.

### HIGH-005 — A clean checkout can build an MSI without the required embedded payloads

The Hermes source ZIP, Node ZIP, and npm tarball are intentionally gitignored. Tauri uses resource globs and Rust resolves each resource as optional; missing files become `None`. The ordinary CI runs only `tauri build --no-bundle` and never prepares or inspects a release MSI.

Preparation scripts exist, but neither the package command nor CI/release workflow requires both scripts with `-RequirePresent`. The documented MSI command mentions only the Hermes source preparation and omits the Node/npm preparation. Therefore a clean checkout can produce or attempt a distributable build without the assets promised by Amendments 003A1 and 003A3, and the installed UI later fails with a missing-archive error.

The locally built `YORVA_0.3.0_x64_en-US.msi` was inspected and does contain the currently expected Hermes source, Node, npm, and three license files. That manually produced artifact is useful evidence but does not make the build reproducible or fail-closed, and it predates the final committed/dirty candidate.

Required closure:

- add one release packaging entry point that requires and verifies all three pinned payloads and licenses before invoking Tauri MSI bundling;
- fail the build when any required payload is absent, wrong-sized, or wrong-hashed;
- add a Windows release job that builds and inspects the MSI file table/content and publishes the exact artifact;
- record exact artifact SHA-256 and CI run against the final implementation commit;
- retain ordinary no-bundle CI without large downloads, but do not treat it as MSI acceptance evidence.

### HIGH-006 — There is no immutable, green Phase 3 audit candidate

The only exact-commit CI for `086ff2a901d76764608d545b12980dd2602122be` is run `32011052252`, event `push`, branch `fix/phase3-managed-node-prerequisites`, conclusion `failure`.

Job evidence:

- Web and API contract: PASS;
- Go Node, including `go test -race ./...`: PASS;
- Windows Desktop native shell: FAIL at `cargo test --locked`;
- cargo-audit, clippy, cargo check, and Tauri no-bundle were skipped after that failure.

The local Rust test now passes, but an uncommitted Desktop recovery patch and test sit above the failed commit. Local success cannot substitute for exact-commit CI, and neither committed HEAD nor the current worktree is a complete green audit target.

Required closure:

- commit only the reviewed remediation and remove generated inspection residue from the candidate;
- run the full local matrix;
- push the exact commit and require all jobs and required steps to pass;
- preserve this failed audit and request `AUDIT-003R1` from a fresh review context.

### MEDIUM-001 — Raw installer stdout/stderr is persisted and returned to Desktop

`HostInstaller.logCommand` writes bounded but otherwise raw stdout, stderr, and raw error strings into `install.ndjson`. `GET /api/v1/operations/{id}/log` reads matching JSON lines and returns them to Desktop, which renders the text.

This directly contradicts the parent Spec and `SECURITY.md`, which require structured redacted outcomes and prohibit raw command output, paths, raw errors, environment values, and installer reasons from persistence or HTTP/Desktop. Environment allowlisting reduces exposure but is not output redaction.

Required closure:

- remove raw stdout/stderr/raw error persistence;
- log only approved structured fields and stable codes/flags;
- retain bounded raw output only transiently for parsing/classification;
- add sentinel secret, path, upstream reason, and HTTP/DB/log exclusion tests.

### MEDIUM-002 — The 60-minute whole-Operation safety ceiling is not implemented

Each subprocess has a stage timeout, but the background worker context has no overall deadline. Sequential source, prerequisite, dependency, and finalization limits can exceed the approved 60-minute ceiling. The prerequisite worker likewise has no Operation-level deadline beyond the individual command.

Required closure:

- apply an adapter/application-owned whole-Operation deadline;
- normalize expiry to the approved timeout code;
- prove the same process-tree cleanup and terminal ordering as explicit cancellation;
- reconcile the parent Spec's now-stale per-stage timeout table with Amendments 003A1 and 003A3.

### MEDIUM-003 — Managed Node implementation is oversized for its responsibility and under-tested

`node_managed.go` is approximately 439 lines and combines:

- public prerequisite state models;
- live Node/npm/dependency health;
- loose version parsing;
- process probing;
- Node installation;
- npm replacement;
- npm dependency execution;
- digest verification;
- ZIP extraction;
- tar/gzip extraction.

This is the material single-file concentration found by the requested monolith review. By contrast, the large generated schema, translation table, license text, and archive-specific extractor are data/generated or cohesive for their purpose.

The Node file has only a small test set: basic version policy, one ZIP traversal case, one tar symlink case, wrong hash, missing Node, and a stage allowlist assertion. It lacks happy artifact materialization, exact Node/npm postconditions, environment/argv verification, archive count/size/root/cancellation cases, all component states, dependency success/failure/timeout, and Operation integration.

Required closure:

- split by concrete responsibility inside the existing Hermes package, without adding a generic package-manager framework or speculative interfaces;
- add proportional contract and integration tests for the managed prerequisite success and failure paths;
- keep `archive.go` and `host_installer.go` focused rather than moving everything into another large manager.

### MEDIUM-004 — Phase 3 governance documents contradict the implemented amendment set

Examples in the current parent Spec and roadmap:

- header says implementation `IN_PROGRESS`, while the approval paragraph says `NOT STARTED`;
- every Entry Criterion is checked, followed by “remaining unchecked criterion”;
- Amendment 003A2 is listed twice;
- the approved stage table still presents official `node` and `node-deps` execution, while 003A3 forbids both;
- timeout and exit text still says optional official Node stages run and “twelve approved stages execute”;
- completion evidence still says commit/CI/full verification pending and names an older branch;
- review record says implementation authorization is pending although it was granted;
- roadmap/status evidence does not fully record Amendments 003A2/003A3 or the current branch/candidate.

Required closure:

- reconcile the parent Spec with all accepted amendments without rewriting historical decisions;
- make executed, YORVA-owned replacement, verified-but-skipped, and excluded stages explicit;
- update status/evidence only after the remediation candidate and exact CI exist;
- do not weaken success, security, or testing criteria to obtain a PASS.

## 5. Large-file review

The largest current code/configuration files reviewed were:

| File | Approx. lines | Audit assessment |
| --- | ---: | --- |
| generated API schema | 847 | Generated; not a monolith finding. |
| OpenAPI contract | 728 | Large but authoritative transport schema. |
| Tauri `daemon.rs` | 603 | Pre-existing lifecycle file; Phase 3 adds only about 49 cohesive resource-handoff lines. Future extraction may help, but Phase 3 did not create a blocking monolith here. |
| `i18n.ts` | 501 | Mostly typed bilingual data; not business-logic concentration. |
| `node_managed.go` | 439 | Blocking maintainability/testability concern; see MEDIUM-003. |
| `archive.go` | 432 | Large, but cohesive around verified archive download/extraction. No size-only finding. |
| `host_installer.go` | 380 | Cohesive orchestration, but its raw logging is a separate security finding. |
| `App.tsx` | 302 | Growing coordinator; current prerequisite patch should be stabilized, but no standalone size finding beyond the workflow defect. |

Line count was used as a review trigger, not as an automatic defect.

## 6. Verification evidence

### Local PASS

```text
pnpm api:lint
pnpm typecheck
pnpm lint
pnpm test                    13 files / 48 tests
pnpm build
pnpm audit --audit-level low no known vulnerabilities

go test ./...
go test affected packages -count=20
go vet ./...
go build ./...

cargo fmt --all -- --check
cargo test --locked          10 tests
cargo clippy --locked --all-targets --all-features -- -D warnings
cargo check --locked
cargo audit                  0 vulnerabilities; 17 inherited allowed warnings

Windows lifecycle smoke      PASS
OpenAPI lint                 PASS
git diff --check             PASS
```

The 48 Desktop tests include the current untracked `App.prereq.test.tsx`; this is worktree evidence, not exact-commit evidence.

### Not locally PASS / blocked

```text
go test -race ./...          NOT RUN: local Go reports race requires CGO
govulncheck ./...            NOT RUN: tool is not installed locally
Tauri release --no-bundle   BLOCKED: PermissionDenied while the manually opened
                             target/release Desktop and yorvad processes hold the build output
```

The exact-commit CI Go job independently passed race and govulncheck. The active Desktop was not terminated during this audit because it belongs to the Owner's manual test session.

### Remote exact-commit evidence

Run: <https://github.com/YoLin02/yorva/actions/runs/32011052252>

```text
head:       086ff2a901d76764608d545b12980dd2602122be
event:      push
branch:     fix/phase3-managed-node-prerequisites
conclusion: failure
```

Local cargo success does not override a failed exact-commit gate.

### MSI inspection evidence

The existing local `YORVA_0.3.0_x64_en-US.msi` is 119,459,840 bytes and its file table contains:

```text
Hermes source ZIP  71,869,305 bytes
Node ZIP           35,682,836 bytes
npm tarball         3,045,132 bytes
Hermes LICENSE          1,070 bytes
NODE-LICENSE           148,217 bytes
NPM-LICENSE              9,742 bytes
```

The source files currently staged for packaging match the compiled Hermes/Node/npm SHA-256 values. This local artifact is not an exact-commit CI artifact and does not close HIGH-005 or HIGH-006.

## 7. Required remediation and re-audit order

1. Resolve the Operation conflict model and atomic persistence constraints.
2. Correct cancellation terminal ordering and prove Windows cleanup.
3. Tighten installation post-check to the canonical public launcher.
4. Correct prerequisite sequencing and authoritative dependency health.
5. Make MSI packaging fail-closed and reproducible from a clean checkout.
6. Remove raw output persistence and add the whole-Operation deadline.
7. Split/test the managed Node implementation proportionally.
8. Reconcile the parent Spec, amendments, roadmap, protocol/runtime/security documentation, and generated API contract.
9. Commit a clean candidate, run the full matrix, and obtain exact-commit CI PASS.
10. Preserve this report and create `AUDIT-003R1-hermes-installation.md` in a fresh independent review context.

## 8. Gate and next-phase decision

```text
Phase 3 Implementation: NOT ACCEPTED
AUDIT-003 Gate:          FAIL
Phase 3 freeze/tag:      NOT AUTHORIZED
Phase 4 planning:        BLOCKED
Phase 4 implementation: BLOCKED
```

No Phase 4 Specification was created because Phase 3 did not pass. This follows the explicit task condition (“when P3 passes, then formulate P4”) and the repository's phase governance.

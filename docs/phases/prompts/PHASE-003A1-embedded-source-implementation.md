# Phase 3 + Amendment 003A1 Implementation Agent Prompt

> Purpose: give this prompt to a fresh YORVA agent to add a verified Hermes source payload to the Phase 3 design and then implement Phase 3 end to end. The agent must pass the Amendment approval gate before writing implementation code.

## Owner Intent

YORVA must be demonstrable on Windows machines in China where GitHub clone/download may be slow or unavailable.

For the Demo MSI, package an immutable, reviewed offline copy of the official Hermes source at the already approved commit. If online source acquisition fails, YORVA verifies and materializes the bundled source, then continues with environment/dependency installation. This is a source fallback, not a complete offline dependency installer.

Do not maintain a modified Hermes fork. Do not add Gitee, `gitclone.com`, `kkgithub.com` or another proxy/mirror in this amendment. A later, separately reviewed amendment may add a signed mainland CDN/Gitee transport.

## Formal Repository State

State observed when this handoff prompt was written; verify it again because another implementation agent may still be changing the shared worktree:

```text
Phase 1: COMPLETE / FROZEN
Phase 2: ACCEPTED / COMPLETE / FROZEN
Phase 2 current baseline: phase-002-hermes-discovery-baseline-r1
Phase 2 current baseline commit: 5b89d22ed5e7ae3f4374a26f0fcda54bdabc6bf9
Phase 2 gate/amendment gate: PASS
Phase 3 Spec: READY — OWNER APPROVED
Phase 3 implementation: IN_PROGRESS
Phase 3 branch: phase/003-hermes-installation
Last committed Phase 3 work observed:
  5eb7afb feat: implement Hermes installation operations
  151774f fix: serialize install operation test store under race detector
Phase 3 audit: AUDIT-003 PENDING
Phase 4: NOT STARTED
```

Approved Hermes target:

```text
Upstream: https://github.com/NousResearch/hermes-agent
Release: v2026.8.16
Package version: 0.20.2
Commit: df4b65147d7ddd74dd449f9067aabbca5aef0ec7
Official install.ps1 size: 233712 bytes
Official install.ps1 YORVA-reviewed SHA-256:
2e1de1867299ce34d5fc73ce63022934acb8966f69f3f53306a37afc3dac29a3
Compatibility: >=0.19.0 <0.21.0
```

Current Tauri configuration uses an MSI bundle and already packages `yorvad` as an external binary. The embedded Hermes payload must be a Tauri/MSI resource; it must not move install orchestration into Rust.

The current working tree was dirty with substantial uncommitted Phase 3 application, Desktop, Tauri, OpenAPI and documentation changes. It also contained an uncommitted `ADR-0005-reviewed-git-mirrors.md` and an uncommitted `repository_fallback.go` that proposed `gitclone.com` and `kkgithub.com`. Those files are not accepted governance merely because an uncommitted ADR says `Accepted`. The Owner's current direction is the embedded official-source fallback in this prompt, not third-party proxy mirrors.

Inspect and preserve all existing work. Never reset, delete, overwrite or silently absorb unrelated user changes. If another agent is still running or file state continues to change, stop and request a clean handoff before editing. Do not run two implementation agents concurrently in the shared worktree.

## Mandatory Reading

Before any edit, read completely:

```text
AGENTS.md
docs/DEVELOPMENT.md
docs/ARCHITECTURE.md
docs/PROTOCOL.md
docs/RUNTIME.md
docs/DATA_MODEL.md
docs/SECURITY.md
docs/PHASE_GOVERNANCE.md
docs/AUDIT_STANDARD.md
docs/ROADMAP.md
docs/phases/PHASE-002-hermes-discovery.md
docs/phases/PHASE-003-hermes-installation.md
docs/phases/prompts/PHASE-003-hermes-installation-implementation.md
docs/phases/amendments/AMENDMENT-002A1-hermes-windows-command-resolution.md
docs/phases/audits/AUDIT-002-hermes-discovery.md
docs/phases/audits/AUDIT-002R1-hermes-discovery.md
docs/phases/audits/AUDIT-002A1-hermes-discovery.md
all relevant ADRs under docs/adr/
apps/desktop/src-tauri/tauri.conf.json
the existing daemon bootstrap/resource-path code
the official immutable Hermes install.ps1 and LICENSE at the approved commit
```

Verify rather than assume:

```text
branch and working tree/index
HEAD and origin/main
peeled Phase 2 R1 baseline tag
Phase 2 Gate
Phase 3 Spec status
current Tauri MSI/resource behavior
official commit, package version and script digest
official source archive/tree size and actual bytes
all commits and uncommitted Phase 3 changes already present
whether another agent/process is actively modifying the worktree
which existing batches are genuinely complete and tested
```

If the Phase 2 baseline/Gate or official pin differs, stop. Do not update Hermes to a newer release implicitly.

Do not restart Phase 3 from scratch. Review the committed and uncommitted implementation, retain correct work, and replace only behavior that conflicts with the accepted Amendment. In particular, third-party GitHub proxy remotes are not an approved substitute for the embedded-source design.

# Stage A — Draft and Approve Amendment 003A1

## A1. Create the Amendment

Create exactly one new governance document:

```text
docs/phases/amendments/AMENDMENT-003A1-embedded-hermes-source.md
```

Initial state:

```text
DRAFT — OWNER APPROVAL REQUIRED
Implementation: NOT STARTED
Audit: covered by AUDIT-003 unless governance requires a named amendment audit section
```

The Amendment must define all of the following before code:

1. objective and user-visible reason: reliable Demo installation when GitHub source acquisition fails;
2. exact relationship to the approved Phase 3 Spec;
3. source payload format selected for the Demo;
4. exact upstream repository, release, commit and license;
5. reproducible payload generation procedure;
6. exact payload byte size and SHA-256, obtained from real bytes rather than invented;
7. build-time source verification and fail-closed behavior;
8. MSI/Tauri resource ownership and path handoff to `yorvad`;
9. runtime size/hash verification before extraction;
10. extraction file-count, expanded-size, traversal, absolute-path, alternate-stream, symlink and reparse-point limits;
11. source acquisition order and bounded online timeout;
12. how bundled materialization replaces or alters the approved official `repository` stage;
13. how the remaining official stages continue without importing/patching Hermes;
14. exact UI wording: “bundled source prepared; dependencies may still require network”;
15. cancellation, cleanup, disk-space and corrupted-resource behavior;
16. MIT License/copyright inclusion and provenance presentation;
17. package-size impact and CI caching/build-time implications;
18. deterministic tests, MSI inspection and independent audit checks;
19. explicit non-goals: dependency wheelhouse, npm cache, Gitee/CDN, full offline install, provider login and Phase 4+;
20. rollback/removal rule: no unknown user data is deleted.

## A2. Required Design Direction

For this Demo, prefer one immutable official commit source archive over a Git history bundle unless repository inspection proves a Git bundle is clearly simpler and bounded. The selected payload must contain the complete official source needed by the approved Hermes installation stages, including `LICENSE`.

The approximate codeload ZIP for this commit was observed near 68.5 MiB. Do not use that estimate as the exact Amendment value. Fetch/generate the real payload, compute its actual size and SHA-256 twice, and record the exact result. If local GitHub connectivity prevents obtaining all bytes, use an auditable exact-commit CI job to produce and hash the payload; do not claim a partial download is complete.

Do not commit an approximately 70 MiB generated binary to normal Git history. Do not introduce Git LFS merely for convenience. Treat it as a verified build input/cache and MSI build artifact. If repository/CI constraints make that impossible, stop for an Owner decision instead of silently bloating the repository.

The expected payload digest must be anchored outside the replaceable payload itself:

- compile the expected digest/size/commit into the Hermes adapter or verify a signed manifest with a public key compiled into YORVA;
- package the original MIT `LICENSE`;
- make the release build fail if resource bytes do not match;
- verify again at runtime;
- rely on MSI code signing when available, but do not pretend SHA-256 alone authenticates an unsigned MSI against a fully capable local attacker.

Do not expose a generic resource path to HTTP/Desktop. Rust may resolve the signed application resource and pass one fixed internal path to the spawned `yorvad` process. Go owns verification, extraction, source selection and installation orchestration.

## A3. Source Selection Contract

The Amendment must choose one simple closed chain:

```text
exact official online source, bounded attempt
→ if transport/unavailability failure only
→ exact MSI-bundled source
→ verify identical approved commit/tree
→ materialize official source
```

Rules:

- no user-provided URL/path/source;
- no random GitHub proxy;
- no Gitee source in 003A1;
- integrity/protocol mismatch does not silently fall back as if it were a network failure; record the security-relevant failure and follow the explicit Amendment policy;
- online and embedded sources must represent the same approved commit and effective source tree;
- source fallback does not expand the compatibility range;
- source selection stays Hermes-specific; do not create a generic package-source/plugin framework.

Because the official script hardcodes GitHub repository transport, the embedded path must not accidentally re-enter its online `repository` stage. Define one of these narrowly and prove it:

1. YORVA materializes the exact source and deliberately replaces/skips only `repository`; or
2. another reviewed mechanism lets the official stage operate entirely against the local payload without leaving a broken remote.

Do not patch the upstream script merely to change URLs. Do not mark `repository` successful without actually materializing and validating the source.

## A4. Dependency Honesty

003A1 does not make Phase 3 fully offline. After source materialization, these may still need network access:

```text
uv / Python
PortableGit
Node.js
PyPI packages
npm packages
ripgrep / ffmpeg
```

The UX and audit report must say “embedded source fallback” or “source available offline,” never “offline installation.”

The Amendment may propose bundling one fixed, verified `uv.exe` only if research proves it is necessary for the Demo and records its official version, URL, size, digest, license, packaging and tests. Do not broaden 003A1 into a complete dependency bundle without a new Owner decision.

## A5. Amendment Review and Mandatory Pause

Self-review the Amendment against architecture, security, process ownership, source provenance, MSI packaging, licensing, testing and Phase 3 scope. Run `git diff --check` and confirm only requested governance/prompt changes exist.

Then stop implementation and request this exact approval from the Owner:

```text
Owner authorization: I approve docs/phases/amendments/AMENDMENT-003A1-embedded-hermes-source.md and authorize its implementation as part of Phase 3 in automatic batch-gate mode.
```

Do not write application/build implementation code before receiving that explicit approval. If the Owner changes the design, revise the Amendment and request approval again.

# Stage B — Implement Phase 3 With Accepted Amendment

Continue only after the Owner approval above is present in the task history. Update the Amendment to `ACCEPTED FOR IMPLEMENTATION` and record the approval date without claiming audit PASS.

Implement the full approved Phase 3 contract rather than leaving an isolated unused bundle subsystem. Follow `PHASE-003-hermes-installation-implementation.md`, with Amendment 003A1 taking precedence only for source acquisition/materialization and MSI packaging.

Continue on the verified `phase/003-hermes-installation` branch after the previous agent has cleanly handed it off. Do not create a second branch around a dirty shared worktree, and do not implement on `main`. If the branch or handoff state has changed, resolve that governance state without force, reset or history rewriting before code.

## Automatic Batch Gates

### Batch 1 — Phase 3 Contracts and Persistence

Implement the original Phase 3 Operation/install contracts, migrations, idempotency and concurrency exactly as approved.

Gate:

- migration, Operation transition and idempotency tests PASS;
- no generic installer/source framework;
- no external process yet.

### Batch 2 — Reproducible Embedded Resource Build

Implement the narrow build pipeline that obtains/generates the exact official source payload, verifies commit/tree/license, computes/validates fixed size and SHA-256, and places it in the Tauri MSI resource set without committing the generated payload to normal Git history.

Gate:

- wrong commit/tree/hash/size/license fails the build;
- repeated generation is deterministic or the Amendment explicitly defines and tests the stable verified input;
- ordinary web/unit workflows do not redownload 70 MiB unnecessarily;
- MSI resource inspection proves the payload and license are present once;
- no Hermes source is exposed as a frontend web asset.

### Batch 3 — Runtime Resource Verification and Safe Materialization

Implement Go-owned verification/extraction/materialization under an Operation-private directory and fixed target. Rust may only resolve/pass the fixed packaged-resource location during daemon bootstrap.

Gate:

- valid payload accepted;
- corrupt/truncated/substituted payload rejected before extraction;
- ZIP-slip, absolute path, drive/UNC path, ADS, symlink/reparse escape, excessive file count and expansion bomb fixtures rejected;
- insufficient disk, cancellation and cleanup tests PASS;
- no arbitrary filesystem API or user-controlled path.

### Batch 4 — Closed Source Fallback Orchestration

Implement bounded exact-online acquisition followed only by the accepted embedded fallback conditions. Integrate source materialization into the original Phase 3 stage driver without invoking a network repository stage after embedded fallback.

Gate:

- online exact source success path PASS;
- simulated DNS/connect/timeout/unavailable response falls back once to embedded source;
- online integrity mismatch follows the Amendment's fail-closed security policy;
- online and embedded paths yield the same approved source identity;
- official remaining stages receive the exact fixed install directory;
- no modified Hermes script, Gitee, proxy or arbitrary mirror.

### Batch 5 — Full Phase 3 Installer Lifecycle

Complete/reuse the original source/script verification, approved installer stages, Windows pre-resume Job Object ownership, durable cancellation/retry/interruption and Phase 2 post-install discovery.

Gate:

- all original Phase 3 focused gates PASS;
- embedded fallback process descendants are contained and reaped;
- fallback does not weaken source/script/environment/output controls;
- success still requires Phase 2 `SUPPORTED`, version `0.20.2`, managed root;
- no accepted installation row on failure/cancellation.

### Batch 6 — API, Desktop and Bilingual UX

Complete the original typed HTTP/OpenAPI/events/Desktop contract. Add safe source-status presentation only where useful:

```text
official source download unavailable
→ verified bundled source used
→ source prepared
→ downloading/installing dependencies
```

Gate:

- English and Simplified Chinese tests PASS;
- UI never claims complete offline install;
- no resource path, raw output or mirror controls leak through API;
- source fallback is a warning/provenance field, not a second installation product.

### Batch 7 — Documentation and Complete Verification

Synchronize the Phase 3 Spec's amendment reference, ROADMAP status/evidence, PROTOCOL, RUNTIME, DATA_MODEL, SECURITY, OpenAPI/generated types and build/release documentation. Do not alter historical Phase 1/2 audit conclusions or baseline tags.

Run the full original Phase 3 matrix plus amendment-specific checks:

```text
payload generation/verification tests
archive extraction adversarial tests
online-failure → embedded fallback integration
MSI resource inspection
MSI size report
Windows install source-materialization smoke
Windows cancellation/process-tree smoke
Tauri release MSI build, not only --no-bundle
git diff --check
```

Also run all repository-available equivalents of:

```text
pnpm api:lint
pnpm typecheck
pnpm lint
pnpm test
pnpm build
pnpm audit

go test ./...
affected Go packages -count=20
go test -race ./...
go vet ./...
go build ./...
govulncheck ./...

cargo fmt --all -- --check
cargo test
cargo clippy --all-targets --all-features -- -D warnings
cargo check
cargo audit

OpenAPI/generated-client verification
Windows daemon/Desktop lifecycle smoke
Tauri release build --no-bundle
Tauri MSI bundle build
```

Never report an unrun check as PASS. Record exact environmental blockers. A real install against the Owner's valued Hermes state is forbidden; use isolated directories, fake protocols and a disposable Windows VM/test account for destructive smoke.

## Mandatory Stop Conditions

Stop and report if:

1. the official pinned source cannot be obtained and verified completely;
2. exact payload bytes/size/hash cannot be made reproducible or auditable;
3. packaging requires committing the large binary to normal Git or introducing Git LFS without Owner approval;
4. a new generic framework, major dependency or architectural ADR is needed;
5. Tauri must own installation business logic;
6. safe archive extraction or process cleanup cannot be guaranteed;
7. embedded source requires patching/forking Hermes;
8. remaining stages cannot work without changing the approved install/runtime contract;
9. “full offline dependency installation,” Gitee/CDN, provider login or Phase 4 work becomes necessary;
10. tests require weakening source integrity, cancellation or post-check guarantees;
11. licensing/trademark/provenance cannot be represented truthfully;
12. an Owner product decision is required.

Otherwise continue automatically through all accepted batches after each focused gate passes.

## Commit, Push and CI

After complete verification:

1. inspect working tree, index, scope diff and `git diff --check`;
2. confirm generated approximately 70 MiB payload/cache is not accidentally staged;
3. commit only approved Phase 3 + 003A1 implementation/governance synchronization;
4. push the implementation branch without force;
5. verify exact-commit GitHub Actions event, branch, head SHA, jobs and relevant steps;
6. require a successful Tauri MSI artifact build and report its final size;
7. fix failures on the branch and verify the new exact commit.

Do not merge `main`, freeze Phase 3, create/move/delete a Phase 3 baseline tag, start Phase 4 or claim independent audit PASS.

## Required Final State

Stop at:

```text
Phase 3 + Amendment 003A1 Implementation = COMPLETE
Verification = PASS, with exact blockers separately stated
Exact-commit CI = PASS
AUDIT-003 = PENDING
Phase 4 = NOT STARTED
```

Final report must include:

1. baseline and Owner Amendment approval evidence;
2. Amendment path/status;
3. selected payload format, generation process, exact bytes, SHA-256 and embedded license;
4. online → embedded fallback contract;
5. Batch 1–7 results;
6. changed files grouped by owning module;
7. focused/full verification commands and results;
8. archive security and Windows process-cleanup evidence;
9. MSI artifact name and size increase;
10. implementation commit SHA;
11. exact-commit CI URL/head/jobs;
12. unrun checks/environmental blockers;
13. residual risks, especially dependencies still requiring network;
14. confirmation that no Gitee/CDN, full offline dependencies, provider login, merge, tag, freeze or Phase 4 work occurred.

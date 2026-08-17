# Phase 3 Implementation Agent Prompt

> Purpose: hand this prompt to a fresh implementation agent only after the Repository Owner explicitly approves `docs/phases/PHASE-003-hermes-installation.md`.

Before use, the Owner must include this authorization in the new task:

```text
Owner authorization: I approve docs/phases/PHASE-003-hermes-installation.md and authorize Phase 3 implementation in automatic batch-gate mode.
```

Without that explicit Owner authorization, stop after repository verification and do not write implementation code.

---

You are the YORVA Phase 3 Implementation Agent. Implement only Phase 3 — Hermes Installation.

## Formal Project State

```text
Phase 1: COMPLETE / FROZEN
Phase 1 Gate: PASS
Phase 2: ACCEPTED / COMPLETE / FROZEN
Phase 2 current baseline: phase-002-hermes-discovery-baseline-r1
Phase 2 current baseline commit: 5b89d22ed5e7ae3f4374a26f0fcda54bdabc6bf9
Phase 2 amendment gate: AUDIT-002A1 PASS
Phase 3: READY only if Owner approval is present in this task
Phase 3 implementation: NOT STARTED
Phase 4: NOT STARTED
```

The historical Phase 2 audit/freeze record must remain immutable. Do not move or delete either Phase 2 baseline tag.

## Mandatory Reading and Verification

Before editing, read completely:

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
docs/phases/amendments/AMENDMENT-002A1-hermes-windows-command-resolution.md
docs/phases/audits/AUDIT-002-hermes-discovery.md
docs/phases/audits/AUDIT-002R1-hermes-discovery.md
docs/phases/audits/AUDIT-002A1-hermes-discovery.md
all relevant ADRs under docs/adr/
```

Then verify rather than assume:

```text
current branch
working tree/index status
HEAD and origin/main
phase-002-hermes-discovery-baseline-r1 peeled SHA
Phase 2 current status and Gate
Owner approval text for the Phase 3 Spec
```

Required baseline:

```text
phase-002-hermes-discovery-baseline-r1
→ 5b89d22ed5e7ae3f4374a26f0fcda54bdabc6bf9
```

If the baseline, Gate or Owner authorization differs, stop and report. Preserve all user work in a dirty tree; do not reset, delete or overwrite it.

Create a dedicated implementation branch from the verified baseline using the repository's normal safe naming convention. Do not implement directly on `main`.

## Authoritative Phase 3 Contract

`docs/phases/PHASE-003-hermes-installation.md` is the implementation source of truth. `ROADMAP.md` alone does not authorize or redefine work.

The reviewed install target is fixed:

```text
Repository: NousResearch/hermes-agent
Release: v2026.8.16
Package: 0.20.2
Commit: df4b65147d7ddd74dd449f9067aabbca5aef0ec7
Script: scripts/install.ps1
Raw URL: https://raw.githubusercontent.com/NousResearch/hermes-agent/df4b65147d7ddd74dd449f9067aabbca5aef0ec7/scripts/install.ps1
Expected size: 233712 bytes
YORVA-reviewed SHA-256: 2e1de1867299ce34d5fc73ce63022934acb8966f69f3f53306a37afc3dac29a3
Installer protocol: 1
Compatibility: >=0.19.0 <0.21.0
```

Do not silently replace this with a newer release, `latest`, `main`, a moving tag URL or another digest. A source change requires a Phase Spec amendment and Owner approval.

## Scope

Implement:

- Windows user-scope Hermes installation;
- one durable `runtime.install` Operation;
- authenticated typed start/read/list/cancel HTTP contracts;
- idempotency and one active Hermes install per Node;
- exact source size/digest verification;
- official protocol/manifest verification;
- the exact approved stage allowlist;
- closed PowerShell executable-plus-argv execution;
- pre-resume Windows Job Object containment;
- timeout, output limit, cancellation and complete descendant cleanup;
- redacted structured logs/events/errors;
- fixed official paths and explicit bilingual confirmation;
- narrow retry for a proven YORVA-owned partial attempt;
- authoritative Phase 2 post-install discovery;
- deterministic migrations for Operations and accepted Runtime installation metadata;
- English and Simplified Chinese UX and local-time rendering;
- docs, OpenAPI, generated types and tests required by the Spec.

Approved stages, in exact order:

```text
uv
python
git
node
system-packages
repository
venv
dependencies
node-deps
path
config-templates
bootstrap-marker
```

Never execute:

```text
desktop
platform-sdks
configure
gateway
```

## Hard Prohibitions

Do not implement:

- macOS/Linux installation;
- arbitrary version/source/path/mirror/command/argv/environment input;
- upgrade, downgrade, repair, uninstall or destructive rollback;
- overwrite/adoption of an unknown installation;
- Hermes Profile or YORVA Instance management;
- models, API keys, persona editing, channels, Weixin/WeCom;
- Hermes agent/gateway/service lifecycle;
- YORVA Skills/MCP management;
- backup/restore, Cloud or remote install;
- Hermes Desktop build;
- automatic elevation or privileged service;
- generic shell/process/package-manager endpoints;
- generic Runtime plugin/installer frameworks;
- Phase 4 work.

Do not weaken the frozen Phase 2 discovery, Windows process-containment or compatibility contracts. Do not fork, patch or import Hermes internals.

## Automatic Batch-Gate Mode

Owner authorization permits continuous execution through all batches without waiting after each successful batch. Every batch must stay minimal and pass its focused gate before the next begins.

### Batch 1 — Contracts and Migrations

Implement the minimal Runtime-neutral Operation/install domain and application contracts, SQLite migrations/repositories, idempotency/concurrency constraints and migration tests.

Gate:

- empty and Phase 2 migration tests PASS;
- Operation transition/idempotency/concurrency tests PASS;
- no external install process exists yet;
- no unrelated schema or generic framework.

### Batch 2 — Verified Official Source

Implement the Hermes-owned immutable release manifest, exact HTTPS download, redirect/size/SHA-256 checks, Operation-private storage, re-hash and cleanup.

Gate:

- exact bytes accepted;
- changed/oversized/redirected/timed-out content rejected before execution;
- no mutable URL, shell download or user-controlled source;
- focused source/security tests PASS.

### Batch 3 — Protocol and Stage Driver

Implement trusted Windows PowerShell resolution, exact protocol/manifest validation, approved/excluded stage policy, bounded JSON result parsing, closed argv and environment construction.

Gate:

- exact reviewed manifest accepted;
- unknown/missing/duplicate/reordered changes fail closed;
- excluded stages are provably never spawned;
- no command string/interpolation or raw reason exposure;
- focused adapter contract tests PASS.

### Batch 4 — Windows Process Ownership

Implement/reuse the secure Windows runner: suspended creation, Job Object assignment before resume, bounded stdout/stderr, timeouts, cancellation, residual-descendant cleanup and handle/goroutine ownership.

Gate:

- immediate-spawn descendant cannot escape;
- timeout/cancel/output-limit/normal-parent-exit lifecycle tests PASS repeatedly;
- no Start-then-bind window;
- no process or goroutine leak.

### Batch 5 — Installation Orchestration

Wire preflight, fixed target validation, durable Operation worker, approved stages, stage transitions, narrow retry, daemon-interruption recovery and Phase 2 post-check/installation upsert.

Gate:

- all Phase 2 state-policy table tests PASS;
- only valid NOT_INSTALLED/retry state mutates;
- terminal state occurs only after cleanup;
- success requires supported 0.20.2 under managed root;
- failures never create accepted installation rows;
- focused application/integration tests PASS.

### Batch 6 — HTTP / OpenAPI / Events

Add authenticated start/get/list/cancel endpoints, idempotency validation, stable DTO/error mapping, bounded SSE operation notifications, OpenAPI and generated Desktop types.

Gate:

- API lint/validation PASS;
- protocol/auth/idempotency/cancel tests PASS;
- generation is clean/reproducible;
- raw installer output/environment/path secrets never cross transport.

### Batch 7 — Desktop UX and Documentation

Extend only the existing Runtime page with bilingual confirmation, host-change disclosure, stage progress, cancellation, retry eligibility, safe failure/correlation and success refresh. Synchronize governing docs and phase evidence without claiming audit PASS.

Gate:

- English/Simplified Chinese component tests PASS;
- local-time rendering PASS;
- explicit confirmation and double-click protection PASS;
- no fake percentage, raw output or duplicated discovery store;
- full scope-diff review PASS.

If a batch fails, diagnose and fix within the approved contract, rerun its focused gate, then continue. Do not weaken tests or acceptance criteria.

## Mandatory Stop Conditions

Stop and report to the Owner if any occurs:

1. the Spec conflicts materially with the frozen architecture or Phase 2 contract;
2. a frozen Phase 1/2 core invariant must change;
3. a new major dependency/framework, generic installer system or ADR is required;
4. automatic elevation or a new privilege boundary is required;
5. safe Windows process cleanup cannot be guaranteed;
6. the reviewed official script/source/stage protocol no longer matches the pin;
7. Phase 4/profile/model/channel/lifecycle functionality is required to continue;
8. tests cannot pass without changing the approved contract;
9. a security risk, secret leak or destructive recovery path is found;
10. an unresolved product decision requires Owner choice.

Otherwise, do not stop merely because one batch completed.

## Full Verification

After Batch 7, run all repository-available equivalents specified by the Phase 3 Spec, including:

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

OpenAPI validation and generated-client clean check
Windows daemon/Desktop lifecycle smoke
Windows installer process-tree smoke
Tauri release build --no-bundle
git diff --check
```

Record every exact command and result. If a required local check is blocked by environment, report it accurately and ensure exact-commit CI covers it; never label it local PASS.

Do not automatically run a real official install on the Owner's machine. A live install smoke requires a disposable Windows VM/test account and explicit authorization. Deterministic fake-protocol and real process-lifecycle tests remain mandatory.

## Commit, Push and CI

After focused and full verification pass:

1. inspect `git status`, staged/unstaged diff, `git diff --check` and changed-file scope;
2. commit only Phase 3 implementation/spec-synchronization changes with a clear message;
3. push the implementation branch without force;
4. obtain the GitHub Actions run for that exact commit;
5. verify event, branch, head SHA and every required job/step;
6. fix failures on the implementation branch, rerun locally, commit/push and verify the new exact commit.

Do not merge `main`, create/move/delete a baseline tag, freeze Phase 3 or begin Phase 4. Independent `AUDIT-003` must happen in a fresh review context after exact-commit CI PASS.

## Required Final State

Stop at:

```text
Phase 3 Implementation = COMPLETE
Verification = PASS, with any exact environmental blocker stated separately
Exact-commit CI = PASS
AUDIT-003 = PENDING
Phase 4 = NOT STARTED
```

Final report must contain:

1. verified baseline and Owner authorization;
2. Batch 1–7 outcomes;
3. key design/security decisions;
4. all changed files grouped by owner;
5. focused and full test commands/results;
6. Windows process-cleanup evidence;
7. implementation commit SHA;
8. GitHub Actions run URL, exact head SHA and job results;
9. any unrun check/environmental blocker;
10. residual risks and whether any Owner decision remains;
11. explicit confirmation that no merge/tag/freeze/Phase 4 work occurred.

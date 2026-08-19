# Agent Prompt — YORVA Phase 4 Instance / Profile Management

Use this file as the complete handoff prompt for a fresh implementation Agent.

## Role

You are the **YORVA Phase 4 Implementation Agent**. Implement only **Phase 4 — Instance / Profile Management** in small, sequential vertical batches. You are not the independent auditor and you do not have authority to merge, freeze, tag, delete the feature branch or begin Phase 5.

This prompt does **not** grant implementation authorization by itself.

## Mandatory Readiness Gate

Before editing production code:

1. verify the current branch, working tree, HEAD and Phase 3 baseline tag;
2. read every required document below in full;
3. verify both Phase 4 Specs are synchronized and marked `READY`;
4. verify D1–D4 in the Chinese Owner-review Spec are explicitly approved;
5. verify the current Owner message separately authorizes Phase 4 implementation.

If any readiness condition is missing, stop after reporting exact evidence. Do not infer approval from `ROADMAP.md`, this prompt, a previous phase Gate, or an unchecked Spec.

## Required Repository Documents

Read in this order:

1. `AGENTS.md`
2. `docs/DEVELOPMENT.md`
3. `docs/ARCHITECTURE.md`
4. `docs/PROTOCOL.md`
5. `docs/RUNTIME.md`
6. `docs/DATA_MODEL.md`
7. `docs/SECURITY.md`
8. `docs/PHASE_GOVERNANCE.md`
9. `docs/AUDIT_STANDARD.md`
10. `docs/ROADMAP.md`
11. `docs/adr/ADR-0001-local-first.md`
12. `docs/adr/ADR-0002-technology-stack.md`
13. `docs/adr/ADR-0003-runtime-adapter.md`
14. `docs/adr/ADR-0004-protocol-and-connectivity.md`
15. `docs/adr/ADR-0005-reviewed-git-mirrors.md`
16. `docs/adr/ADR-0006-generation-install-transaction.md`
17. `docs/phases/PHASE-003-hermes-installation.md`
18. `docs/phases/amendments/AMENDMENT-003A4-generation-install-transaction.md`
19. `docs/phases/audits/AUDIT-003R9-hermes-installation.md`
20. `docs/phases/PHASE-004-instance-profile.zh-CN.md` — Owner-review source
21. `docs/phases/PHASE-004-instance-profile.md` — English execution contract

The Chinese and English Phase 4 Specs define the same contract. If they differ materially, stop. Do not choose whichever version is easier to implement.

## Required Official Hermes Evidence

The reviewed official source is bundled read-only at:

```text
apps/desktop/src-tauri/resources/hermes/source/
hermes-agent-df4b65147d7ddd74dd449f9067aabbca5aef0ec7.zip
```

Identity:

```text
Hermes version: 0.20.2
Official commit: df4b65147d7ddd74dd449f9067aabbca5aef0ec7
```

Read the following official material inside the archive without modifying or casually extracting it into the repository:

```text
website/docs/reference/profile-commands.md
hermes_cli/profiles.py
tui_gateway/methods_profiles.py
hermes_cli/web_routers/profiles.py
```

Also read the directly relevant official tests for Profile list/create/delete and name validation. Use the official Markdown document as the behavior reference and the pinned source/tests to resolve ambiguity. Do not import Hermes Python internals into YORVA production code.

During Batch 1, confirm at minimum:

- exact `profile list/create/delete` syntax and exit behavior;
- `--no-alias` and `--no-skills` behavior;
- official name normalization, reserved names and length limits;
- whether an offline structured list output exists;
- REST startup/authentication requirements;
- TUI gateway Profile method coverage, including absence or presence of delete.

If the evidence differs from the approved Spec, stop and request an Amendment. Do not silently change the contract.

## Locked Product Semantics

- One YORVA Instance maps to one Hermes Profile.
- `instanceId` is an opaque, stable YORVA identity used by YORVA APIs, Operations and relations.
- `nativeId` is the Runtime-native Hermes Profile identity and is used only inside the Hermes adapter.
- The two identifiers are never interchangeable.
- `AVAILABLE` means only that the Profile exists in the latest successful authoritative query. It does not mean login, model, Agent, gateway or process readiness.
- `MISSING` rows are retained indefinitely in Phase 4. No TTL, startup-count or refresh-count cleanup exists.
- If the same `(runtimeInstallationId, nativeId)` reappears, preserve the existing `instanceId` and restore `AVAILABLE`.
- SQLite is a cache. Hermes remains authoritative.
- Deleting native Profile data marks the retained YORVA Instance `MISSING`; it does not delete the stable YORVA identity.
- The `default` Profile is visible and never deletable.
- Phase 4 lifecycle capability is false; do not implement Start/Stop/Restart.

## Hard Scope Boundary

Implement only Profile list/get, create, protected delete, reconciliation, Instance cache, local API/OpenAPI, Operations and bilingual Desktop UX.

Do not implement:

- Hermes install/repair/upgrade/uninstall;
- login, credentials, API keys, models or provider configuration;
- Profile clone/rename/import/export/selection;
- Skills, MCP, channels, Weixin/WeCom, memory, backup/restore or Cloud;
- Hermes process lifecycle or service startup;
- changes to Phase 3 generations, seals, transactions, `active.json`, `PATH` or `HERMES_HOME`;
- direct Profile-directory discovery or Hermes Python imports;
- arbitrary path, shell, environment, URL or file APIs;
- a dynamic Runtime plugin system.

Preserve the Phase 3 generation architecture and all historical audit reports.

## Git and Workspace Discipline

- Begin from the exact authorized baseline and create `codex/phase4-instance-profile` or another Owner-approved non-`main` branch.
- Do not modify, delete, stage or commit unrelated user work or untracked files.
- Inspect `git status`, staged diff and unstaged diff before every batch commit.
- Keep one reviewable commit per Batch, or a small isolated commit series clearly labeled with the Batch number.
- Never force-push or rewrite history.
- Push only the feature branch for exact-commit CI after authorization; never merge to `main`.

## Automatic Batch Gates

Execute `PHASE-004-instance-profile.md` Section 18 exactly in order.

```text
Batch 1  Official contract and fixtures
→ focused tests + diff review PASS
Batch 2  Read-only Instance inventory
→ focused tests + diff review PASS
Batch 3  Minimal Profile create
→ focused tests + diff review PASS
Batch 4  Protected Profile delete
→ focused tests + diff review PASS
Batch 5  Contract-bounded resilience
→ affected/full tests + diff review PASS
Batch 6  Full verification and audit handoff
```

A passing Gate automatically authorizes entry into the next Batch; no per-Batch Owner wait is required after implementation has been authorized. A failed Gate blocks the next Batch.

For each Batch:

1. state its exact intended files and behavior;
2. implement only that vertical capability;
3. add focused regression tests before broad verification;
4. run `git diff --check` and inspect the complete Batch diff;
5. confirm no later-Batch or Phase 5+ behavior entered the diff;
6. commit the isolated Batch only after its Gate passes;
7. record commands, results and anything not run.

## Stop Conditions

Stop and report to the Owner when:

1. the approved Spec conflicts materially with current architecture or pinned Hermes behavior;
2. the work requires changing frozen Phase 3 core architecture;
3. an official surface requires starting/managing a Hermes service;
4. safe deletion cannot be proven from a fresh authoritative identity query;
5. process-tree cleanup or secret redaction cannot be guaranteed;
6. a new major dependency/framework or public contract change is required;
7. Phase 5+ functionality is required to continue;
8. tests cannot pass without weakening the contract;
9. an Owner product decision is required.

Do not stop merely because a Batch completed successfully.

## Proportional Design Rule

Implement the smallest coherent design that satisfies the approved Spec. Reuse existing YORVA Operations, events, process containment, logging, persistence and generated API patterns.

Do not add:

- a generic workflow engine or second state machine;
- a plugin or dependency-injection framework;
- speculative interfaces for OpenClaw or hypothetical Runtimes;
- a new filesystem ownership/integrity architecture;
- ACL/sandbox infrastructure not required by the approved threat model;
- failpoints for every filesystem/system call;
- repositories/managers/helpers with no real ownership or substitution need.

Test concrete identity, Profile command, deletion, process, reconciliation and data boundaries. Record unrelated hardening as future work instead of expanding Phase 4.

## Required Implementation Boundaries

```text
React Desktop
→ generated typed client
→ authenticated local HTTP/OpenAPI
→ application Instance use cases and Operations
→ domain Instance model
→ Hermes adapter / SQLite adapter
```

Expected ownership:

```text
services/node/internal/domain/instance
services/node/internal/app
services/node/internal/runtime/hermes
services/node/internal/persistence/sqlite
services/node/internal/transport/httpapi
apps/desktop/src
api/openapi.yaml
```

Hermes-specific parsing remains under `internal/runtime/hermes`. Go is authoritative for Profile integration, identity, reconciliation and Operations. Rust/Tauri remains the native shell. React never invokes Hermes or reads Profile files.

## Verification

Use repository scripts and exact CI commands; do not invent green results.

Repository/Web/OpenAPI:

```text
pnpm install --frozen-lockfile
pnpm audit --audit-level low
pnpm api:lint
pnpm api:generate
git diff --exit-code -- apps/desktop/src/api/generated/schema.ts
pnpm typecheck
pnpm lint
pnpm test
pnpm build
```

Go, from `services/node`:

```text
gofmt check on changed Go files
go test ./...
go test <affected packages> -count=20
go test -race ./...
go vet ./...
govulncheck ./...
go build ./cmd/yorvad
```

Rust/Tauri, from `apps/desktop/src-tauri` unless stated otherwise:

```text
cargo fmt --check
cargo test --locked --offline --lib
cargo audit
cargo clippy --locked --all-targets -- -D warnings
cargo check --locked
pnpm --filter @yorva/desktop tauri build --no-bundle
```

Windows integration from repository root:

```text
pnpm build:sidecar
pwsh -NoProfile -File scripts/windows-lifecycle-smoke.ps1
pwsh -NoProfile -File scripts/inspect-yorva-msi.tests.ps1
```

Only run MSI packaging if Phase 4 changes packaging inputs or the approved verification contract explicitly requires it. Real Hermes delete smoke must use an isolated Windows account/VM with disposable Profile data; never delete an Owner Profile.

If local `go test -race` is environment-blocked, record the exact reason and require exact-commit CI race PASS. Do not report it as a local PASS.

## CI and Final Handoff

After Batch 6 local verification:

1. push the Phase 4 feature branch without force;
2. verify GitHub Actions `CI` is for the exact implementation commit and every required job passes;
3. do not merge `main` and do not create a Phase 4 baseline tag;
4. create no audit verdict yourself;
5. hand the immutable commit, branch, CI run and full matrix to a fresh independent audit Agent.

Stop at:

```text
Phase 4 Implementation = COMPLETE
Verification = PASS
AUDIT-004 = PENDING
```

## Required Final Report

Report once after all authorized batches complete:

1. readiness evidence and starting commit;
2. Batch 1–6 commits and Gate results;
3. official Hermes surface decision and evidence;
4. key design decisions;
5. changed files grouped by owning module;
6. focused and full verification results, including blocked/unrun checks;
7. final implementation commit SHA;
8. exact-commit GitHub Actions run URL and job results;
9. remaining non-blocking risks;
10. confirmation that `AUDIT-004` remains pending and no merge/tag/Phase 5 work occurred.

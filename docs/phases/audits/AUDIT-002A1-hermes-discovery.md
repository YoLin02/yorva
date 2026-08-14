# YORVA Phase 2 Amendment 002A1 Audit — Hermes Discovery Windows Command Resolution

## Phase

Phase 2 amendment 002A1 — Windows Hermes command resolution and 0.20 compatibility

## Baseline / Commit

- frozen historical baseline: `phase-002-hermes-discovery-baseline` → peeled `a67de04e900bc3ddce99cd76501eec13586082ed`;
- historical audits preserved and not edited: `AUDIT-002-hermes-discovery.md` (`FAIL`), `AUDIT-002R1-hermes-discovery.md` (`PASS`);
- amendment audit target: `dbcb54da4bc4bffcff51888426848246a1900ea6` on `fix/phase2-hermes-command-resolution`;
- working tree at audit start: `HEAD` = `dbcb54da4bc4bffcff51888426848246a1900ea6`, no local modification of implementation, tests, contracts, amendment or control documents;
- ancestry proof: `git merge-base dbcb54da4bc4bffcff51888426848246a1900ea6 phase-002-hermes-discovery-baseline` returned `a67de04e900bc3ddce99cd76501eec13586082ed`; `git merge-base --is-ancestor a67de04e900bc3ddce99cd76501eec13586082ed dbcb54da4bc4bffcff51888426848246a1900ea6` exited 0;
- reviewed diff: `git show --stat dbcb54d` and the full `git diff a67de04e900bc3ddce99cd76501eec13586082ed..dbcb54da4bc4bffcff51888426848246a1900ea6` (23 files, +798 / −77), plus the complete current source of the changed Go adapter files rather than hunks alone.

## Auditor

Independent Phase 2 amendment 002A1 auditor operating from a fresh review context. This agent is not the implementation agent and not the control agent. The only repository write is this report.

## Date

2026-08-15

## Gate Decision

`PASS`

## Executive Summary

Amendment 002A1 is a narrow, authorized post-freeze correction of a Phase 2 correctness defect at the Hermes command boundary. On the affected official Windows layout — documented root present, both `hermes.exe` launchers absent, package-declared `hermes = hermes_cli.main:main` present — the frozen detector reported `BROKEN_EXECUTABLE`. The audited commit constructs one fallback command only after canonical containment and unique entry-point validation, then executes without a shell:

```text
<official-root>\venv\Scripts\python.exe -I -m hermes_cli.main --version
```

Direct `hermes.exe --version` retains priority. `hermes-agent.exe` and the repository wrapper remain evidence only. Public HTTP/OpenAPI DTO shapes and stable error codes are unchanged. The compatibility window is the authorized `>=0.19.0 <0.21.0`: stable `0.20.0` and `0.20.1` are `SUPPORTED`; `0.21.0` and prereleases remain `UNSUPPORTED`. Windows Job Object create-suspended-then-assign behavior is unchanged.

Independent verification against `dbcb54da4bc4bffcff51888426848246a1900ea6` passed, including a gated real-host smoke that returned `SUPPORTED` / `0.20.0` / `venv\Scripts\python.exe` / one candidate. Exact-commit GitHub Actions run `31817886874` is independently confirmed `success` on that same head SHA, including `go test -race ./...`. No Critical or High finding was identified. Replacement baseline freeze is allowed only after this `PASS`, and only by the control agent, without moving the original historical tag.

## Verification Evidence

| Check | Result | Evidence |
|---|---|---|
| Exact audited commit | PASS | `git rev-parse HEAD` and `git rev-parse dbcb54da4bc4bffcff51888426848246a1900ea6` both returned `dbcb54da4bc4bffcff51888426848246a1900ea6` on `fix/phase2-hermes-command-resolution`. Commit subject: `fix: resolve official Windows Hermes discovery`. |
| Frozen baseline ancestry | PASS | Peeled `phase-002-hermes-discovery-baseline^{}` = `a67de04e900bc3ddce99cd76501eec13586082ed`. Merge-base with the audited commit is that SHA. The commit is a descendant of the frozen tag. Historical `AUDIT-002` / `AUDIT-002R1` paths are absent from the amendment diff. |
| Diff and source review | PASS | 23 files vs the frozen baseline. Production execution change is confined to `services/node/internal/runtime/hermes/{candidates,command,detector,version}.go`. OpenAPI, generated Desktop schema, `go.mod`/`go.sum`, migrations and persistence packages are unchanged. Desktop/HTTP edits are fixture `supportedRange` / 0.20.x classification updates only. |
| Forbidden execution search | PASS | Production Hermes spawn exists only at `command.go:49` as `exec.CommandContext(..., invocation.executable, invocation.args...)`. No `cmd /c`, PowerShell, shell string, `UseShellExecute`, PATH mutation, installer/download, or generic process HTTP API. `hermes-agent.exe` appears in production only as a regular-file evidence marker at `candidates.go:302`. |
| Fallback construction / containment | PASS | `officialPythonInvocation` (`candidates.go:138-181`) runs only for the Windows official root, returns false when either documented launcher exists, requires canonical regular `python.exe`, wrapper `hermes`, `hermes_cli\main.py`, contained site-packages, and unique `hermes_cli.main:main`. Argv is constant `-I -m hermes_cli.main --version`; `workingDir` is the canonical official root. |
| Entry-point uniqueness and bounds | PASS | `hasUniqueHermesConsoleEntryPoint` (`:183-211`) rejects more than 1,024 site-packages entries and requires exactly one `hermes_agent-*.dist-info`. `hasConsoleEntryPoint` (`:213-234`) requires exactly one `hermes` console definition equal to `hermes_cli.main:main`. Metadata is capped at 16 KiB (`:17`, `:236-247`). Paths are `EvalSymlinks` + `pathWithin` (`:271-292`). |
| Missing-launcher vs fallback vs `NOT_INSTALLED` | PASS | Evidence without a launcher or validated fallback remains `BROKEN_EXECUTABLE` / `HERMES_CLI_LAUNCHER_MISSING` and executes nothing (`detector.go:43-51`, `TestDetectorClassifiesMissingLauncherAsBrokenWithoutExecutingEvidence`). Validated fallback is executed and can return `SUPPORTED` (`TestDetectorUsesTrustedOfficialPythonEntrypointWhenLauncherIsMissing`). Absence of evidence and commands remains `NOT_INSTALLED`. |
| Compatibility window | PASS | `version.go:11` and `:89-90` implement `>=0.19.0 <0.21.0` and require empty prerelease. `TestParseVersionBanner` classifies `0.20.0` and `0.20.1` supported, `0.21.0` / `0.18.2` / `0.19.1-rc.1` / `0.20.2-rc.1` unsupported. |
| Windows Job Object ownership | PASS | `process_windows.go` is not in the amendment diff. `:16` still sets `CREATE_SUSPENDED`; `:47` assigns the Job; `:51` resumes only afterward. Independent `TestCommandRunnerOwnsImmediateDescendantBeforeExecutableRuns`, `TestCommandRunnerTimeoutReapsDescendantProcess` and `TestCommandRunnerCancellationReapsProcess` passed 5 consecutive times. |
| Affected Go packages | PASS | Independent `go test ./internal/runtime/hermes ./internal/app ./internal/transport/httpapi -count=1` passed. |
| Candidate-finder repeats | PASS | `go test ./internal/runtime/hermes -run TestCandidateFinder -count=20` passed. Coverage includes precedence, untrusted/duplicate/ambiguous/oversized metadata, site-packages bound, and launcher-over-fallback. |
| Full Go verification | PASS | From `services/node`: `go test ./... -count=1`, `go vet ./...`, and `go build -o %TEMP%\yorvad-audit-002a1.exe ./cmd/yorvad` all exited 0. |
| Local Go race | ENVIRONMENT BLOCKED | `CGO_ENABLED=1 go test -race ./...` failed before compilation: `cgo: C compiler "gcc" not found`. No local race PASS is claimed. Exact-commit CI `Go Node` step `go test -race ./...` succeeded on this SHA. |
| govulncheck | PASS | `govulncheck` was not on `PATH`. Independent `go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...` in `services/node` reported `No vulnerabilities found.` |
| Real Windows smoke | PASS | Host root `%LOCALAPPDATA%\hermes\hermes-agent` exists; `bin\hermes.exe` and `venv\Scripts\hermes.exe` are absent; `venv\Scripts\python.exe`, repository wrapper `hermes`, and `venv\Scripts\hermes-agent.exe` exist; one `hermes_agent-0.20.0.dist-info` declares exactly `hermes = hermes_cli.main:main`; site-packages has 267 entries. `$env:YORVA_REAL_HERMES_SMOKE='1'; go test ./internal/runtime/hermes -run TestRealWindowsHermesInstallationSmoke -v -count=1` logged `state=SUPPORTED version=0.20.0 command=C:\Users\Administrator\AppData\Local\hermes\hermes-agent\venv\Scripts\python.exe candidates=1` and would have failed if `hermes-agent.exe` were a candidate. |
| Exact-commit GitHub Actions | PASS | Independently fetched `https://api.github.com/repos/YoLin02/yorva/actions/runs/31817886874`: `event=push`, `head_branch=fix/phase2-hermes-command-resolution`, `head_sha=dbcb54da4bc4bffcff51888426848246a1900ea6`, `status=completed`, `conclusion=success`. Jobs: `Web and API contract` success (OpenAPI lint/generate/schema diff, typecheck, lint, test, build, pnpm audit); `Go Node` success (`go test -race ./...`, `go vet`, `govulncheck`, `go build ./cmd/yorvad`); `Windows Desktop native shell` success (sidecar, Windows lifecycle smoke, cargo fmt/test/clippy/check, cargo audit, Tauri `--no-bundle`). |
| Desktop / OpenAPI local rerun | RELIED ON EXACT-COMMIT CI | `api/openapi.yaml`, `apps/desktop/src/api/generated/schema.ts` and Desktop component logic are outside the amendment diff except test fixtures. DTO field set, closed enums and error codes are unchanged; only the documented `supportedRange` value became `>=0.19.0 <0.21.0`. Local Desktop/OpenAPI suites were not re-executed because run `31817886874` already passed those jobs on this SHA. |
| Data / dependencies | N/A with evidence | No migration, persistence, `go.mod`/`go.sum`, lockfile or new dependency change appears in `a67de04..dbcb54d`. |
| Diff hygiene | INFO | `git diff --check a67de04 dbcb54d` reports only trailing spaces on amendment/control markdown header lines. No production-code whitespace error. |

## Dimension Results

### Scope

`PASS` — the diff implements only the authorized fallback command and the Owner-approved 0.20.x window, plus matching tests and contract-doc synchronization. No installer, PATH mutation, Profile/Instance/lifecycle, persistence, generic Python API or Phase 3 work is present.

### Correctness

`PASS` — launcher precedence, fallback acceptance/rejection, missing-launcher `BROKEN_EXECUTABLE`, `NOT_INSTALLED`, 0.20.0/0.20.1 `SUPPORTED`, 0.21.0 and prerelease `UNSUPPORTED`, and the real affected-host smoke all match the amendment acceptance rule.

### Architecture

`PASS` — ownership remains React Desktop → authenticated HTTP → application use case → Runtime-neutral contract → Hermes adapter → direct argv. Domain/HTTP still do not construct Hermes commands. The new `commandInvocation` type is adapter-private.

### Security

`PASS` — trusted-installation recognition is not arbitrary Python execution. Fallback requires official-root containment, regular files, unique exact console entry point, 1,024-entry and 16 KiB bounds, isolated `-I`, constant argv and no shell. User-controlled argv is impossible on `POST /api/v1/runtimes/{runtimeKind}/detect`. `hermes-agent.exe` and the repository wrapper are never command hosts. Child environment remain allowlisted; Job Object ownership is unchanged.

### Data and Persistence

`N/A` — the amendment adds no schema, migration, repository or persisted discovery cache. Discovery remains a live read-only query.

### Concurrency and Lifecycle

`PASS` — per-candidate timeout, overall deadline, cancellation, output bounds and Windows create-suspended-then-assign Job Object behavior are unchanged and were re-tested. Local race remains environment-blocked; exact-commit CI race passed.

### Protocol and Compatibility

`PASS` — OpenAPI/DTO shapes and stable error codes are unchanged. The `supportedRange` string value change to `>=0.19.0 <0.21.0` is the amendment’s authorized compatibility policy, not a schema change. Desktop still renders the server-supplied range and does not parse Hermes output.

### Testing and Verification

`PASS` — required candidate, entry-point, argv/working-directory, compatibility, process-lifecycle and gated real-install cases exist and were independently re-run. Tests were tightened (smoke now requires a supported selected command) rather than weakened.

### Maintainability

`PASS` — fallback construction, metadata parsing and command descriptors are localized to the Hermes adapter. No speculative plugin/framework or extra service layer was introduced.

### Documentation

`PASS` — Phase Spec §6/§8/§14/§16, `RUNTIME.md`, `SECURITY.md` and `DEVELOPMENT.md` describe the closed fallback and 0.20.x window. The amendment and control records exist. Historical audits are untouched. Stale loop-status lines in the committed amendment/control snapshots are operational bookkeeping for the control agent, not implementation/contract drift.

### Dependencies / Supply Chain

`N/A` — no dependency or lockfile change. Independent `govulncheck` on the node module found no vulnerabilities. Exact-commit CI `pnpm audit` and `cargo audit` succeeded on this SHA.

### Operations / Diagnostics

`PASS` — missing-launcher warning `HERMES_CLI_LAUNCHER_MISSING` is retained when evidence exists without a safe command. Application structured discovery logs remain path-free. The smoke now records state, version, command host and candidate count.

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

- Local `go test -race ./...` is environment-blocked because `gcc` is absent. Exact-commit CI run `31817886874` job `Go Node` executed `go test -race ./...` successfully on `dbcb54da4bc4bffcff51888426848246a1900ea6`. That limitation is not a pass and is not accepted technical debt.
- Committed `AMENDMENT-002A1` / `CONTROL-002A1` status lines still describe the pre-commit / pre-CI loop snapshot. They are historical control records; the control agent owns updating loop state after this gate. This audit does not edit them.
- There is no dedicated fixture whose `entry_points.txt` is syntactically malformed (for example a `hermes` line without `=`). Adjacent tests already reject untrusted, duplicate, ambiguous and oversized metadata, and the parser accepts only exactly one exact `hermes = hermes_cli.main:main` definition.
- `git diff --check` reports trailing spaces on amendment/control markdown header lines only.
- This audit executed the official-root `python.exe` version command through the gated smoke. It did not execute `hermes-agent.exe` or the repository wrapper.

## Accepted Technical Debt

None.

## Required Fixes Before Next Phase

None in the audited commit. Phase 3 implementation remains prohibited until the control agent completes Owner merge and the replacement baseline freeze specified by `CONTROL-002A1-baseline-re-freeze.md`. The original tag `phase-002-hermes-discovery-baseline` must not be moved.

## Gate Rationale

`PASS` is required by `AUDIT_STANDARD.md` §§16–17: zero Critical findings, zero blocking High findings, every amendment acceptance criterion and required success flow is independently demonstrated, architecture and security boundaries hold, verification evidence is sufficient for this command-boundary change, and exact-commit CI on `dbcb54da4bc4bffcff51888426848246a1900ea6` is independently confirmed `success`. Correctness and security were treated as blocking dimensions and passed.

## Next Step

The control agent may, and only the control agent may, proceed with replacement baseline freeze after Owner merge:

1. keep the original annotated tag `phase-002-hermes-discovery-baseline` at `a67de04e900bc3ddce99cd76501eec13586082ed`;
2. do not rewrite historical `AUDIT-002` / `AUDIT-002R1` reports;
3. include this `AUDIT-002A1` report in the reviewed tree;
4. merge the exact audited implementation to `main` only after Owner acceptance;
5. create a replacement freeze commit and annotated tag `phase-002-hermes-discovery-baseline-r1` on `main`;
6. update Phase Spec / Roadmap / Development / Phase Governance acceptance records to name the replacement baseline.

Do not begin Phase 3 implementation before that replacement tag exists and is pushed.

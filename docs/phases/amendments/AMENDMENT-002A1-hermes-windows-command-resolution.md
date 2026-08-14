# YORVA Phase 2 Baseline Amendment 002A1 — Windows Hermes Command Resolution and 0.20 Compatibility

> Status: ACCEPTED / COMPLETE / FROZEN  
> Date: 2026-08-15  
> Owner: Repository owner  
> Historical frozen baseline: `phase-002-hermes-discovery-baseline` → `a67de04e900bc3ddce99cd76501eec13586082ed` (immutable)  
> Current baseline: `phase-002-hermes-discovery-baseline-r1`  
> Related historical audit: `AUDIT-002R1-hermes-discovery.md` — PASS (preserved)  
> Amendment gate: `AUDIT-002A1-hermes-discovery.md` — PASS  
> Implementation commit: `dbcb54da4bc4bffcff51888426848246a1900ea6`  
> Exact-commit CI: GitHub Actions run `31817886874` — PASS  
> Merge commit: `f786b9d73768de38875f0a57fa7d57c75168ed85`  
> Compatibility: `>=0.19.0 <0.21.0`  
> Control: `CONTROL-002A1-baseline-re-freeze.md`

## Defect classification

This is a post-freeze Phase 2 correctness defect with a security-sensitive Runtime-command boundary.

The official Windows installation can remain operational when the generated `hermes.exe` console launcher is absent. The frozen detector treated that layout as `BROKEN_EXECUTABLE` even when the installed package declared the official Hermes console entry point and the installation's own Python could execute the read-only version command. On the affected host, the safe command returns Hermes `0.20.0`; after the Owner's compatibility decision below, the correct Phase 2 result is `SUPPORTED`, not broken or unsupported.

The frozen tag and historical audit reports are not changed. This amendment must pass exact-commit CI and a fresh independent re-audit before it may be accepted into a replacement Phase 2 baseline.

## Evidence

The affected official root is `%LOCALAPPDATA%\hermes\hermes-agent`. It contains:

- `venv\Scripts\python.exe`;
- the official repository `hermes` marker and `hermes_cli\main.py`;
- one `hermes_agent-*.dist-info\entry_points.txt` declaring `hermes = hermes_cli.main:main`;
- no `bin\hermes.exe` or `venv\Scripts\hermes.exe`.

Direct, read-only execution of the package-declared console entry point through that installation's Python returns `Hermes Agent v0.20.0`. No installation, repair, PATH change, profile access or lifecycle operation is required.

The official GitHub release `v2026.8.13`, published 2026-08-13, declares package version `0.20.1` and is not a prerelease. On 2026-08-14 the Owner explicitly authorized compatibility with the official latest stable line.

## Narrow correction

Direct `hermes.exe --version` candidates retain priority and existing behavior. Only when both documented Windows launchers are absent may the Hermes adapter construct one fallback command descriptor from the fixed official root.

The fallback is accepted only when all of the following hold:

1. the root, Python executable, module marker, wrapper marker, site-packages directory and entry-point metadata resolve to canonical paths within the official root;
2. each required file is a regular file;
3. exactly one `hermes_agent-*.dist-info` entry-point file contains exactly one `hermes` console definition;
4. that definition is exactly `hermes_cli.main:main`;
5. site-packages enumeration is capped at 1,024 entries and metadata input is bounded to 16 KiB.

The adapter then directly executes, without a shell:

```text
<official-root>\venv\Scripts\python.exe -I -m hermes_cli.main --version
```

`-I` prevents inherited Python configuration and user-site state from altering module resolution. The working directory is the canonical official root. YORVA does not import Hermes Python modules into its process; it invokes the packaging-declared official CLI surface in the Hermes-owned environment.

The repository wrapper and `hermes-agent.exe` remain evidence only and are never executed. User-controlled argv, generic Python execution, package installation, launcher repair and PATH mutation remain prohibited. Existing environment filtering, 64 KiB per-stream output bounds, per-command timeout, overall deadline, cancellation and process-tree cleanup are unchanged.

## Contract and compatibility impact

- Public HTTP/OpenAPI DTOs and stable error codes do not change.
- The compatibility range becomes `>= 0.19.0 and < 0.21.0`.
- Stable `0.20.0` and current official `0.20.1` are `SUPPORTED`; `0.21.0` and prereleases remain `UNSUPPORTED` pending review.
- Evidence with absent launchers and an absent, ambiguous, oversized or non-matching package entry point remains `BROKEN_EXECUTABLE` and executes nothing.
- SQLite, persistence, Desktop behavior and dependencies do not change.

## Required verification

- candidate precedence, canonical containment, deduplication and bounds;
- exact entry-point parsing, including duplicate, ambiguous, malformed and oversized metadata rejection;
- exact fallback executable, argv, isolated Python mode and working directory;
- compatibility classification of stable `0.20.0` and `0.20.1` as `SUPPORTED`, and `0.21.0` as `UNSUPPORTED`;
- timeout, cancellation, output limits and Windows process-tree cleanup;
- real Windows installation smoke proving the affected host is detected without invoking `hermes-agent.exe`;
- full Go, protocol/OpenAPI, Desktop, Rust/Tauri and Windows lifecycle checks;
- exact-commit GitHub Actions, including Go race;
- independent re-audit of correctness, Runtime compatibility, security and lifecycle boundaries.

## Local verification result

Local verification on Windows passed on 2026-08-14:

- affected Hermes/application/HTTP packages, including 20-repeat regressions;
- candidate security cases repeated 20 times;
- gated real-installation smoke: `SUPPORTED`, version `0.20.0`, command host `venv\Scripts\python.exe`, one candidate;
- full Go test, vet, build and `govulncheck`;
- OpenAPI lint/generation consistency, TypeScript typecheck/lint, 34 Desktop tests and production build;
- pnpm audit with no known vulnerabilities;
- Rust fmt, 9 tests, clippy and check;
- cargo-audit 0.22.2 with no vulnerability failure and the unchanged 17 allowed warnings already documented by `AUDIT-002R1`;
- sidecar build, Windows lifecycle smoke and Tauri release no-bundle build.

Local `go test -race ./...` is environment-blocked because cgo cannot find `gcc`. The exact-commit CI Go race job remains mandatory evidence; this local limitation is not treated as a pass.

## Acceptance rule

Amendment 002A1 is accepted. The independent audit returned `PASS`, exact-commit CI run `31817886874` passed on `dbcb54da4bc4bffcff51888426848246a1900ea6`, and the amendment is merged to `main`. The historical tag `phase-002-hermes-discovery-baseline` remains at `a67de04e900bc3ddce99cd76501eec13586082ed`. The current Phase 2 baseline is `phase-002-hermes-discovery-baseline-r1`. Phase 3 implementation remains prohibited until that replacement tag exists and is pushed.

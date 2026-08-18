# YORVA Phase 3 Amendment 003A3 — Managed Node.js Prerequisites

> Status: ACCEPTED FOR IMPLEMENTATION
> Date: 2026-08-17
> Owner approval date: 2026-08-17
> Owner: Repository owner
> Phase: 3 — Hermes Installation
> Related: `AMENDMENT-003A1-embedded-hermes-source.md`, `AMENDMENT-003A2-china-dependency-distribution.md`
> Implementation: R5 REMEDIATION; AUDIT-003R6 PENDING
> Audit: historical `AUDIT-003` through `AUDIT-003R5` are FAIL; this status is not an audit PASS

## 1. Problem

Phase 3 still spawned official `node` and `node-deps`. Those stages are unsafe or unreliable for YORVA:

- official `Install-NodeDeps` downloads `raw.githubusercontent.com/trycua/cua/main/...` and pipes it to `Invoke-Expression`;
- it also best-effort installs browser-use, Playwright Chromium and other unpinned network tools;
- official Node selection uses `latest-v22.x`;
- YORVA treated `node` / `node-deps` as skippable, so failures vanished and Desktop hid repair UI after Hermes was `SUPPORTED`;
- the host Node 22.23.1 ships npm 10.9.8, which official `engine-strict` / npm-12 provisioning treats as insufficient.

## 2. Decision

YORVA never executes official PowerShell `-Stage node` or `-Stage node-deps`. Manifest verification still requires those stage names so protocol drift is detected.

YORVA owns a Hermes-specific Windows Node prerequisite:

```text
%LOCALAPPDATA%\hermes\node
```

Pinned artifacts (compiled; not `latest`):

| Artifact | Value |
| --- | --- |
| Node | official `v22.23.1` Windows x64 zip |
| Official Node URL | `https://nodejs.org/dist/v22.23.1/node-v22.23.1-win-x64.zip` |
| Official SHASUMS256 | `7df0bc9375723f4a86b3aa1b7cc73342423d9677a8df4538aca31a049e309c29` |
| npm | official `12.0.2` tarball |
| Official npm URL | `https://registry.npmjs.org/npm/-/npm-12.0.2.tgz` |
| Destination | `%LOCALAPPDATA%\hermes\node` |
| Licenses | Node.js / npm licenses packaged beside the archives |

Measured 2026-08-17 from complete official downloads, hashed twice:

| Artifact | Size | SHA-256 |
| --- | --- | --- |
| `node-v22.23.1-win-x64.zip` | `35682836` | `7df0bc9375723f4a86b3aa1b7cc73342423d9677a8df4538aca31a049e309c29` |
| `npm-12.0.2.tgz` | `3045132` | `5dbb86c71d07a1957f2e90734092dd6a58bdcd9ebc2d8d41ca1c6e6a21d364e1` |

The Node digest matches the official `SHASUMS256.txt` line for `node-v22.23.1-win-x64.zip`. A release build fails closed on mismatch.

Demo MSI embeds the verified Node zip and npm tarball as Tauri resources. Rust only resolves those paths and passes:

```text
hermesNodeArchivePath
hermesNpmArchivePath
```

on the existing bootstrap record. Go verifies, extracts, materializes and probes. HTTP/Desktop never receive resource paths.

Postcondition, direct argv only:

```text
<node>\node.exe --version     →  v22.23.1  (satisfies Hermes >=22.22.0)
<node>\npm.cmd --version      →  12.0.2    (satisfies official installer npm >=12)
```

Never `npm.ps1`, never a shell string, never winget/choco/scoop, never user URL/path/version/registry.

## 3. Safe Node dependencies

Do not call official `Install-NodeDeps`.

After managed Node/npm are ready, YORVA runs one command in the official Hermes tree:

```text
npm.cmd ci --workspaces=false --omit=dev --ignore-scripts --no-audit --no-fund --progress=false
```

Reasons from the pinned `package.json` / lockfile:

- the tree is an npm workspace including `apps/desktop`;
- a root `npm ci` would pull Electron and desktop native modules;
- official `node-deps` then runs Playwright, `uv tool install browser-use`, and CUA `irm | iex`;
- the pinned root has no production dependencies, so `--workspaces=false --omit=dev` is the lockfile-faithful CLI install that does not pull Desktop or browser runtimes.

Uses 003A2 `NPM_CONFIG_REGISTRY=https://registry.npmmirror.com`. Inherited `NPM_*` / `NODE_*` secrets are stripped except the compiled registry. Timeout is 15 minutes. Failure does not fall back to unlocked `npm install`.

## 4. Prerequisite Operation and API

Hermes `SUPPORTED` does not imply Node health.

```text
GET  /api/v1/runtimes/hermes/prerequisites
POST /api/v1/runtimes/hermes/prerequisites/install
```

`GET` is a live typed query. `POST` starts durable Operation type `hermes.prerequisites` with an Idempotency-Key. The start body is the same closed empty object as Hermes install (`requestBody` required). One active prerequisite Operation per Node. Desktop/SSE disconnect does not cancel.

Component states:

```text
node:             READY | MISSING | UNSUPPORTED | BROKEN
npm:              READY | MISSING | UNSUPPORTED | BROKEN
nodeDependencies: READY | NOT_INSTALLED | FAILED | TIMED_OUT
```

Stable error codes:

```text
RUNTIME_HERMES_NODE_MISSING
RUNTIME_HERMES_NODE_UNSUPPORTED
RUNTIME_HERMES_NPM_MISSING
RUNTIME_HERMES_NPM_UNSUPPORTED
RUNTIME_HERMES_NODE_ARCHIVE_INTEGRITY_FAILED
RUNTIME_HERMES_NPM_ARCHIVE_INTEGRITY_FAILED
RUNTIME_HERMES_NODE_DEPS_FAILED
RUNTIME_HERMES_NODE_DEPS_TIMEOUT
```

## 5. Desktop

The Runtime page shows Node/npm health whenever discovery has completed, including `SUPPORTED`. Starting a new Hermes install remains the `NOT_INSTALLED` confirmation flow; an already-running type/target-validated `runtime.install` stays visible under partial discovery and is never replaced by a prerequisite Operation. Prerequisite buttons: install/reinstall Node+npm; retry dependencies; cancel a running Operation. Identical launcher aliases must not present an ambiguity selector.

## 6. Non-goals

Generic package manager, user registry, Gitee/GitHub proxy, Playwright/CUA/browser-use, Hermes Desktop, Skills/MCP/Profile/Model/Channel, Phase 4, merging main, Phase 3 freeze/tag.

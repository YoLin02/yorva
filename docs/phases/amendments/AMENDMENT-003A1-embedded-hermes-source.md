# YORVA Phase 3 Amendment 003A1 — Embedded Official Hermes Source for Demo MSI

> Status: ACCEPTED FOR IMPLEMENTATION
> Date: 2026-08-17
> Owner approval date: 2026-08-17
> Owner: Repository owner
> Phase: 3 — Hermes Installation
> Phase Spec: `docs/phases/PHASE-003-hermes-installation.md`
> Implementation: R2 REMEDIATION; AUDIT-003R3 PENDING
> Audit: covered by `AUDIT-003` unless governance later requires a named amendment audit section
> Gate: this status is not an audit PASS
> Previous baseline: `phase-002-hermes-discovery-baseline-r1` → `5b89d22ed5e7ae3f4374a26f0fcda54bdabc6bf9`
> Phase 3 branch at drafting: `phase/003-hermes-installation`
> Phase 3 committed HEAD at drafting: `151774f3e58a5b68875a2c2dab937a15087dd639`
> Compatibility: unchanged `>=0.19.0 <0.21.0`
> Hermes pin: unchanged `v2026.8.16` / `0.20.2` / `df4b65147d7ddd74dd449f9067aabbca5aef0ec7`

The Owner approved this Amendment on 2026-08-17 and authorized implementation as part of Phase 3 in automatic batch-gate mode. That approval is not an audit PASS.

## 0. Classification and relationship to Phase 3

This is a Phase 3 scope amendment for **source acquisition and Demo MSI packaging only**.

It does not:

- change the approved Hermes release, commit, compatibility range or installer protocol;
- add upgrade, repair, uninstall, profile, credential, channel, gateway or Phase 4 behavior;
- move install orchestration into Rust/Tauri;
- create a generic package-source, plugin or multi-Runtime installer framework;
- authorize Gitee, `gitclone.com`, `kkgithub.com`, a university mirror or any other third-party Git proxy;
- claim or implement a complete offline installer.

The approved Phase 3 Spec remains the parent contract. Amendment 003A1 takes precedence only where the Spec's official `repository` transport and MSI resource rules would otherwise leave Demo installation blocked when GitHub source bytes cannot be fetched.

Uncommitted `docs/adr/ADR-0005-reviewed-git-mirrors.md` and `services/node/internal/runtime/hermes/repository_fallback.go` are **not** accepted governance. After this Amendment is approved, implementation must replace that third-party-remote direction rather than absorb it. No new architectural ADR is required: MSI resource packaging and a one-path bootstrap handoff stay inside the existing Tauri native-shell role defined by `ARCHITECTURE.md` and ADR-0002.

---

## 1. Objective and user-visible reason

YORVA must be demonstrable on Windows machines in China where GitHub clone or archive download is slow or unavailable.

The Demo MSI therefore packages one immutable, reviewed offline copy of the official Hermes source at the already approved commit. If the exact official online archive cannot be obtained for a **transport or unavailability** reason only, YORVA verifies the bundled archive, materializes that official source into the fixed user-scope install directory, then continues the remaining official environment and dependency stages.

This is an **embedded source fallback**. It is not a complete offline dependency installer. After source materialization, `uv` / Python, PortableGit, Node.js, PyPI, npm, ripgrep and ffmpeg may still require network access.

## 2. Exact relationship to the approved Phase 3 Spec

Parent contract: `docs/phases/PHASE-003-hermes-installation.md` (Owner-approved 2026-08-17).

Unchanged:

| Item | Value |
| --- | --- |
| Upstream | `https://github.com/NousResearch/hermes-agent` |
| Release | `v2026.8.16` |
| Package version | `0.20.2` |
| Commit | `df4b65147d7ddd74dd449f9067aabbca5aef0ec7` |
| Official `install.ps1` | 233712 bytes, SHA-256 `2e1de1867299ce34d5fc73ce63022934acb8966f69f3f53306a37afc3dac29a3` |
| Protocol | `1` |
| Compatibility | `>=0.19.0 <0.21.0` |
| Destination | `%LOCALAPPDATA%\hermes\hermes-agent` |
| Approved stage names | `uv`, `python`, `git`, `node`, `system-packages`, `repository`, `venv`, `dependencies`, `node-deps`, `path`, `config-templates`, `bootstrap-marker` |
| Excluded stages | `desktop`, `platform-sdks`, `configure`, `gateway` |
| Post-check | unchanged Phase 2 discovery: `SUPPORTED`, version `0.20.2`, managed root |

Changed by 003A1:

1. Source bytes for the Hermes tree are acquired as the official GitHub **commit archive**, not by spawning the official PowerShell `repository` stage.
2. A Demo MSI may include that exact archive as a Tauri resource.
3. The user-visible `install.repository` stage becomes YORVA-owned archive verification and materialization.
4. Confirmation UX must disclose bundled-source fallback and must never claim complete offline installation.
5. Third-party Git remotes are removed from the Phase 3 source contract.

## 3. Selected source payload format

Selected format for the Demo:

```text
Official GitHub commit archive ZIP for df4b65147d7ddd74dd449f9067aabbca5aef0ec7
```

Rejected for 003A1:

| Alternative | Reason |
| --- | --- |
| Git history bundle | Larger or equal, requires `git` to materialize, is not simpler, and is not what the official installer already treats as its own ZIP fallback. |
| Shallow clone snapshot | Not a single verifiable file; reintroduces Git transport. |
| PyPI `hermes-agent==0.20.2` sdist/wheel | Not the official checkout the remaining installer stages expect. |
| Third-party mirror ZIP | Unreviewed host; forbidden. |

Repository inspection of official `scripts/install.ps1` at the pin (`Install-Repository`) proves:

- the official stage hardcodes GitHub SSH then HTTPS clone;
- when clone fails it already downloads `https://github.com/NousResearch/hermes-agent/archive/$Commit.zip`;
- a directory that exists **without a valid git work tree** is moved aside as “broken” and then re-cloned from GitHub;
- `Install-Venv`, `Install-Dependencies` and `Write-BootstrapMarker` do not require `.git` when `-Commit` is supplied (YORVA already passes that flag).

Therefore a Git bundle is **not** clearly simpler. The official commit archive is the bounded payload. YORVA must **replace/skip** the official `repository` stage after materializing; it must not spawn that stage against a materialized ZIP tree.

The payload contains the complete official source needed by the approved remaining stages, including `LICENSE`, `pyproject.toml`, `uv.lock`, `hermes_cli` and the rest of the commit tree.

Honest limitation of GitHub's ZIP export: it is a working-tree archive of the commit, not a byte-for-byte `git cat-file` dump. In this payload, `scripts/install.ps1` is stored with CRLF (238490 bytes) while the git blob / `raw.githubusercontent.com` copy is LF (233712 bytes). The extra 4778 bytes are exactly one CR per line. YORVA does **not** execute the ZIP copy of `install.ps1` unless CRLF is normalized to LF and the result matches the already reviewed script digest. See §12.

## 4. Exact upstream repository, release, commit and license

```text
Repository:     NousResearch/hermes-agent
Clone URL:      https://github.com/NousResearch/hermes-agent.git
Release:        v2026.8.16
Package:        0.20.2  (confirmed in archive pyproject.toml)
Commit:         df4b65147d7ddd74dd449f9067aabbca5aef0ec7
License:        MIT
Copyright:      Copyright (c) 2025 Nous Research
LICENSE path:   LICENSE at repository root
```

Evidence:

- release: <https://github.com/NousResearch/hermes-agent/releases/tag/v2026.8.16>
- commit: <https://github.com/NousResearch/hermes-agent/commit/df4b65147d7ddd74dd449f9067aabbca5aef0ec7>
- license (immutable): <https://raw.githubusercontent.com/NousResearch/hermes-agent/df4b65147d7ddd74dd449f9067aabbca5aef0ec7/LICENSE>

YORVA is not a Hermes fork and does not modify these bytes.

## 5. Reproducible payload generation procedure

Canonical official URL (also used by official `install.ps1`):

```text
https://github.com/NousResearch/hermes-agent/archive/df4b65147d7ddd74dd449f9067aabbca5aef0ec7.zip
```

This URL redirects to `codeload.github.com` for the same commit ZIP. Implementation may follow that single official redirect host. No other host is approved.

Build/CI procedure (deterministic input, not committed):

1. Create a gitignored cache directory, for example `.cache/hermes-source/` (repository root) or `%LOCALAPPDATA%\yorva-build\hermes-source\`.
2. If a cached file already has size `71869305` and SHA-256 `2ed02f76aaf5dab0bfd320bdbfa10aad0f67e00cbbf87906cde05462681708ba`, reuse it.
3. Otherwise HTTPS GET the canonical URL with a size cap of `80 MiB`, write to a temporary name, hash, and promote into the cache only on exact match.
4. Fail the release/MSI build if the bytes are missing, truncated, redirected to an unapproved host, or do not match size and SHA-256.
5. Copy the verified file into the Tauri resource input path used by the MSI. Do not copy it into `apps/desktop/src/` or `frontendDist`.
6. Extract `LICENSE` from the verified archive (or fetch the immutable raw LICENSE) and confirm size `1070` and SHA-256 `821556e6336796450ab852d375117b48a4887e71d255794fd6318d99982a5ab6`. Package that LICENSE next to the archive resource and/or keep a reviewed text copy under `docs/` or `runtimes/hermes/` for provenance display. Do not invent license text.

Ordinary `pnpm test`, Go unit tests and `tauri --no-bundle` must not redownload 70 MiB. They use tiny fixtures. Only release MSI / dedicated payload-generation jobs fetch or consume the real archive.

The approximately 70 MiB archive **must not** be committed to normal Git history. Git LFS is not introduced. If a future CI constraint makes an ignored cache plus MSI artifact impossible, stop for an Owner decision.

## 6. Exact payload byte size and SHA-256

Measured on 2026-08-17 from a complete local file obtained as the official commit archive. SHA-256 was computed **twice** with `Get-FileHash -Algorithm SHA256`. Both runs matched.

| Field | Value |
| --- | --- |
| File size | `71869305` bytes |
| SHA-256 (hex, lowercase) | `2ed02f76aaf5dab0bfd320bdbfa10aad0f67e00cbbf87906cde05462681708ba` |
| ZIP entries | `10401` (`9411` non-directory, `990` directory, `39` empty files) |
| Uncompressed size sum | `149756982` bytes |
| Compressed size sum | `68830321` bytes |
| Single root prefix | `hermes-agent-df4b65147d7ddd74dd449f9067aabbca5aef0ec7/` |
| Largest file | `apps/desktop/public/ds-assets/filler-bg0.jpg` = `3871968` bytes |
| Traversal / absolute / ADS entries | none |
| UNIX symlink entries (mode `0120000`) | `0` |
| Embedded `LICENSE` | `1070` bytes, SHA-256 `821556e6336796450ab852d375117b48a4887e71d255794fd6318d99982a5ab6` (hashed twice) |
| Embedded `pyproject.toml` version | `0.20.2` |

An independent HEAD of both official archive URLs on 2026-08-17 returned HTTP 200 and `Content-Type: application/zip`. GitHub did not send `Content-Length` (chunked). Size and digest are therefore taken from the complete downloaded bytes, not from a header estimate. The earlier ~68.5 MiB planning figure is **not** the Amendment value.

These constants are compiled into the Hermes adapter. They are YORVA-reviewed content digests, not an upstream-published checksum or signature.

## 7. Build-time source verification and fail-closed behavior

Release / MSI build fails closed when any of the following is true:

- cache/resource bytes are absent;
- size is not exactly `71869305`;
- SHA-256 is not exactly `2ed02f76aaf5dab0bfd320bdbfa10aad0f67e00cbbf87906cde05462681708ba`;
- the ZIP cannot be opened or has more than one top-level directory prefix;
- the prefix is not `hermes-agent-df4b65147d7ddd74dd449f9067aabbca5aef0ec7/`;
- `LICENSE` is missing or its digest is not `821556e6336796450ab852d375117b48a4887e71d255794fd6318d99982a5ab6`;
- `pyproject.toml` is missing or does not declare `version = "0.20.2"`;
- `scripts/install.ps1` is missing.

The expected digest is **not** stored only inside the replaceable payload. It is compiled into `services/node/internal/runtime/hermes` next to the existing script pin. A signed YORVA public-key manifest is not required for 003A1.

MSI Authenticode signing, when available, is additional packaging integrity. SHA-256 of an unsigned MSI does **not** authenticate the payload against a fully capable local attacker.

Non-MSI developer builds may omit the resource. They then have online-archive-only source acquisition and must fail `RUNTIME_INSTALL_SOURCE_UNAVAILABLE` if the official archive cannot be fetched. They must not invent another fallback.

## 8. MSI / Tauri resource ownership and path handoff to `yorvad`

Current bootstrap (`apps/desktop/src-tauri/src/daemon.rs` → `services/node/internal/bootstrap`) passes only:

```text
protocolVersion, token, dataDir
```

`tauri.conf.json` already packages `yorvad` as `externalBin` and builds an MSI. It does not yet declare Hermes resources. Frontend assets remain the React bundle only.

003A1 ownership:

```text
Tauri / Rust
  resolve the signed application resource directory
  locate one fixed relative file:
    hermes/source/hermes-agent-df4b65147d7ddd74dd449f9067aabbca5aef0ec7.zip
  pass that absolute path to yorvad on bootstrap
        ↓
yorvad bootstrap (process-private JSON, not HTTP)
        ↓
Hermes adapter
  verify, extract, select source, orchestrate remaining official stages
```

Bootstrap grows by **one** optional Hermes-specific field, for example `hermesEmbeddedSourcePath`. Rules:

- Rust does not open, hash, extract or install Hermes;
- Rust does not accept a user-chosen path;
- the field is absent when the resource is not packaged;
- Go ignores a missing field for non-MSI runs;
- Go rejects a present path that is not a regular file, escapes the expected resource/containment rule, or is a reparse point;
- Desktop HTTP, OpenAPI and SSE never receive this path;
- no generic resource-map API is added.

Suggested MSI resource destination inside the application package:

```text
hermes/source/hermes-agent-df4b65147d7ddd74dd449f9067aabbca5aef0ec7.zip
hermes/source/LICENSE
```

These files are native application resources, not web assets.

## 9. Runtime size/hash verification before extraction

Before any extract, the Hermes adapter must:

1. resolve the candidate archive (downloaded temp file or bootstrap path);
2. require a regular file with no reparse-point escape;
3. require size exactly `71869305`;
4. hash the entire file and require SHA-256 `2ed02f76aaf5dab0bfd320bdbfa10aad0f67e00cbbf87906cde05462681708ba`;
5. open it as a ZIP and apply §10 limits **before** writing payload members.

Mismatch is `RUNTIME_INSTALL_INTEGRITY_FAILED`. The file is not extracted. The official `repository` stage is not spawned as a consolation path.

The separately pinned `install.ps1` continues to be re-hashed immediately before every PowerShell invocation, unchanged from Phase 3.

## 10. Extraction limits

All limits are adapter constants and must be tested with fixtures. Observed official-archive values are listed for justification, not as runtime-discovered parameters.

| Check | Limit | Observed |
| --- | --- | --- |
| ZIP entries | at most `20_000` | `10401` |
| Non-directory members | at most `16_000` | `9411` |
| Uncompressed sum | at most `256 MiB` | `149756982` |
| Single member | at most `16 MiB` | `3871968` |
| Expansion ratio (uncompressed / archive size) | at most `8` | `~2.08` |
| Top-level prefixes | exactly one, must equal `hermes-agent-df4b65147d7ddd74dd449f9067aabbca5aef0ec7` | one |
| `..` path segment | reject | none |
| Absolute / drive / UNC path | reject | none |
| Alternate data stream (`:` in a path component) | reject | none |
| Symlink / hardlink / UNIX mode `0120000` | reject | none |
| After extract: `FILE_ATTRIBUTE_REPARSE_POINT` | reject and abort | n/a |
| Required members after prefix strip | `LICENSE`, `pyproject.toml`, `scripts/install.ps1`, `hermes_cli/` or equivalent official package tree | present |

Extraction writes only into an Operation-private staging directory under the YORVA data dir, created without a reparse-point escape. After a complete successful extract and required-member check, the stripped tree is placed at the fixed official install directory according to Phase 3 target/retry rules.

Do not use `Expand-Archive` as the production extractor (host `AuthorizationManager` failures were already observed). Use a Go-owned ZIP reader that enforces the limits above.

## 11. Source acquisition order and bounded online timeout

Closed chain:

```text
exact official GitHub commit archive, bounded HTTPS attempt
  → if transport / unavailability failure only
  → exact MSI-bundled archive
  → verify identical size / SHA-256 / root prefix / LICENSE
  → materialize official source under the fixed install directory
  → do not spawn official repository
  → continue remaining official stages
```

Rules:

- no user-provided URL, path or source;
- no random GitHub proxy;
- no Gitee / CDN source in 003A1;
- online and bundled payloads must be the same archive identity (size + SHA-256);
- source fallback does not expand the compatibility range;
- selection stays inside the Hermes adapter.

Online attempt:

- URL: the canonical official commit-archive URL in §5;
- connect / TLS / headers: 30 seconds;
- overall online archive deadline: 180 seconds;
- body cap: `80 MiB`;
- approved redirect: only the official `github.com` → `codeload.github.com` archive path for this commit;
- cookies / provider credentials: never sent.

Transport / unavailability (fallback **allowed**, once):

- DNS failure;
- TCP / TLS connect failure;
- deadline exceeded;
- HTTP 408 / 429 / 5xx;
- HTTP 403 / 404 from the official archive host with no complete matching body;
- empty or reset stream before a complete file is written.

Integrity / protocol mismatch (fallback **forbidden**):

- HTTP 200 (or completed body) whose size or SHA-256 does not match;
- redirect to an unapproved host;
- ZIP that opens but fails §10;
- LICENSE / `pyproject.toml` / script-normalization mismatch.

Those cases are `RUNTIME_INSTALL_INTEGRITY_FAILED` or `RUNTIME_INSTALL_SOURCE_UNAVAILABLE` as specified in §15. They are recorded as security-relevant. They are not retried as if GitHub were merely slow.

If the online attempt succeeds and verifies, the bundled file is not used.

If the bundled path is required but the bootstrap field is absent (non-MSI), fail `RUNTIME_INSTALL_SOURCE_UNAVAILABLE`.

If the bundled path is present but fails §9, fail `RUNTIME_INSTALL_INTEGRITY_FAILED`.

## 12. How bundled materialization replaces the official `repository` stage

Chosen mechanism: **YORVA materializes the exact source and deliberately replaces/skips only `repository`.**

Proof that the official stage cannot be left in the loop:

1. `Stage-Repository` calls `Install-Repository`.
2. `Install-Repository` treats a non-git directory as broken, moves it to `hermes-agent.broken-<timestamp>`, then clones GitHub.
3. Even its own ZIP fallback then `git fetch origin <commit>` from GitHub.
4. Therefore spawning `repository` after a successful embed would undo the fallback and re-enter the unavailable network.

YORVA behavior at user-visible stage `install.repository`:

1. Acquire the archive per §11.
2. Verify per §9–§10.
3. Obtain the Phase 3 `install.ps1` as follows:
   - preferred when the archive is already verified: extract `scripts/install.ps1`, normalize CR+LF to LF, require size `233712` and SHA-256 `2e1de1867299ce34d5fc73ce63022934acb8966f69f3f53306a37afc3dac29a3`;
   - otherwise the existing official raw URL download remains valid;
   - never execute a CRLF ZIP copy or a normalized file that does not match the pin.
4. Materialize the stripped official tree into `%LOCALAPPDATA%\hermes\hermes-agent` (or the retry-eligible YORVA-owned target).
5. Record provenance internally: `official-archive` or `bundled-archive`.
6. **Do not** invoke PowerShell `-Stage repository`.
7. Mark `install.repository` successful only after materialization and required-member checks succeed.
8. Continue the remaining approved official stages in the existing order.

The official script is not patched. URLs are not rewritten. A success result is never forged without a validated tree on disk.

`.git` is not required for 003A1 remaining stages. YORVA does not run `git init` against the materialized tree merely to look like a clone.

## 13. How remaining official stages continue

After materialization, YORVA continues to invoke the verified official `install.ps1` with the existing closed argv, including `-Commit df4b65147d7ddd74dd449f9067aabbca5aef0ec7`, `-InstallDir` and `-HermesHome`.

Remaining official stages after materialization (superseded for Node by Amendment 003A3):

```text
uv
python
git
system-packages
venv
dependencies
path
config-templates
bootstrap-marker
```

Official `node` and `node-deps` stages are still verified in the official manifest, then never spawned. Amendment 003A3 replaces them with YORVA-managed Node `22.23.1` and npm `12.0.2` under `%LOCALAPPDATA%\hermes\node`.

Inspection of the pinned script shows `Install-Venv` and `Install-Dependencies` operate on `$InstallDir` files (`pyproject.toml`, `uv.lock`, `venv`) and do not require a git remote. `Write-BootstrapMarker` prefers the explicit `-Commit` flag YORVA already supplies.

YORVA still does not import Hermes Python modules, fork Hermes, or edit official files after extract except for the installer-owned mutations those stages already perform.

The `git` stage remains: PortableGit / Git may still be installed because later Hermes workflows and the official script expect it. 003A1 does not remove that stage.

## 14. Exact UI wording

Required English sentence, used whenever bundled materialization is the source that will be or has been used:

```text
Bundled source prepared; dependencies may still require network.
```

Required Simplified Chinese equivalent:

```text
已准备内置源码；依赖安装仍可能需要网络。
```

Allowed status sequence, localized from stable keys, never from raw installer text:

```text
official source download unavailable
→ verified bundled source used
→ source prepared
→ downloading/installing dependencies
```

Forbidden user-visible claims:

- “offline installation”
- “fully offline”
- “no network required”
- any mirror / Gitee / CDN control
- any resource filesystem path

Confirmation host-change copy must replace the uncommitted “reviewed Chinese Git mirror” wording. It may say that if the official GitHub source archive cannot be downloaded, YORVA uses the verified copy packaged in this YORVA installer, and that dependency downloads may still need the network.

Provenance is a warning / status field on the existing install Operation, not a second installation product.

## 15. Cancellation, cleanup, disk-space and corrupted-resource behavior

| Event | Behavior |
| --- | --- |
| User cancel during download | abort HTTP; delete incomplete temp file; no extract; Operation `CANCELLED` after process cleanup |
| User cancel during extract | stop writing; delete Operation-private staging; do not delete an unknown pre-existing install directory |
| User cancel during a later official stage | existing Phase 3 Job Object / process-tree rules |
| Corrupted / truncated / substituted archive | `RUNTIME_INSTALL_INTEGRITY_FAILED`; no extract |
| Unsafe ZIP member (§10) | `RUNTIME_INSTALL_INTEGRITY_FAILED`; delete staging |
| Insufficient free space | require free space ≥ `256 MiB` plus a `64 MiB` margin on the destination volume before extract; fail `RUNTIME_INSTALL_INSUFFICIENT_DISK` (`retryable=true`); write nothing into the official install directory |
| Missing bundled resource after online unavailability | `RUNTIME_INSTALL_SOURCE_UNAVAILABLE` |
| Staging leftover after terminal state | delete Operation-private archive/staging; cleanup failure is a redacted warning, not success |
| Unknown user directory | never deleted or moved by 003A1; Phase 3 `TARGET_OCCUPIED` still applies |

`RUNTIME_INSTALL_INSUFFICIENT_DISK` is the only new public error code this Amendment adds. All other outcomes reuse the Phase 3 table.

Cancellation still cannot roll back completed official stages (`uv`, `python`, …). 003A1 does not add uninstall.

## 16. MIT License, copyright and provenance presentation

The Demo MSI redistributes official Hermes source under the upstream MIT License.

Required:

- package the original `LICENSE` (`Copyright (c) 2025 Nous Research`);
- keep that copyright notice with the bundled source;
- Desktop confirmation may state that Hermes Agent is MIT-licensed software by Nous Research and that YORVA includes a reviewed copy of the official source for this pinned commit;
- do not display YORVA as the author of Hermes;
- do not claim an upstream signature or checksum that Nous Research did not publish;
- describe the ZIP digest as a YORVA-reviewed content digest of the official GitHub commit archive.

YORVA's own license, when declared, remains separate.

## 17. Package-size impact and CI / build-time implications

| Item | Impact |
| --- | --- |
| Archive | `71869305` bytes ≈ `68.54 MiB` added to the Demo MSI |
| LICENSE | `1070` bytes |
| Expected MSI growth | about `68.5–70 MiB` plus WiX/MSI overhead; the implementation batch must report the measured MSI size before and after |
| Git repository | no ~70 MiB blob in history |
| Unit / typecheck / lint | no download |
| Release MSI / payload job | fetch or cache-hit + verify |
| CI cache key | `hermes-source-2ed02f76aaf5dab0bfd320bdbfa10aad0f67e00cbbf87906cde05462681708ba` |

`.gitignore` must include the cache and Tauri resource input ZIP, for example:

```text
.cache/hermes-source/
apps/desktop/src-tauri/resources/hermes/source/*.zip
```

The small LICENSE text may be tracked.

## 18. Deterministic tests, MSI inspection and independent audit checks

Required tests (tiny fixtures, never the 70 MiB file in the default suite):

- compiled expected size/hash accept the golden fixture and reject flipped bytes, truncated files and wrong size;
- ZIP-slip (`../`), absolute path, drive/UNC, ADS and symlink fixtures are rejected before any write outside staging;
- expansion-bomb / excessive entry-count fixtures are rejected;
- insufficient disk (injected free-space function) does not extract;
- cancellation deletes staging and does not report success;
- online DNS/connect/timeout/5xx → exactly one bundled attempt → success when bundled verifies;
- online HTTP 200 with wrong digest → `RUNTIME_INSTALL_INTEGRITY_FAILED`, bundled not used;
- online success and bundled success produce the same required members and the same compiled archive identity;
- official PowerShell `-Stage repository` is never spawned after 003A1;
- excluded stages remain unspawned;
- UI English/Chinese tests include the exact §14 sentence and forbid “offline installation”;
- HTTP/OpenAPI tests prove the resource path is absent from responses.

MSI / audit checks (release artifact, not unit tests):

- the MSI contains the archive resource once;
- the MSI contains `LICENSE`;
- the archive resource hashes to `2ed02f76aaf5dab0bfd320bdbfa10aad0f67e00cbbf87906cde05462681708ba`;
- the frontend web asset tree does not contain Hermes source;
- measured MSI size is recorded in the implementation completion report.

Independent `AUDIT-003` must review this Amendment's implementation, including that ADR-0005 / `gitclone.com` / `kkgithub.com` did not land as the accepted source path.

## 19. Explicit non-goals

Hard boundaries, in addition to Phase 3 non-goals:

- dependency wheelhouse / vendored PyPI wheels;
- npm cache or Node installer embedding;
- bundling `uv.exe`, PortableGit, Python, Node.js, ripgrep or ffmpeg;
- Gitee, university mirrors, `gitclone.com`, `kkgithub.com` or any CDN transport;
- full offline installation;
- user-selected source path or URL;
- provider login, models, credentials;
- Hermes Desktop build;
- Phase 4 Instance/Profile work;
- a generic source-plugin framework;
- Git LFS;
- committing the ~70 MiB archive to Git;
- patching or forking Hermes;
- claiming the YORVA-reviewed ZIP digest is an upstream signature.

Research note on `uv.exe`: live Phase 3 attempts already showed the official `uv` stage can succeed from `astral.sh` on networks where GitHub git/archive fails. 003A1 therefore does **not** bundle `uv`. A later Owner decision would be required to pin a specific `uv.exe` version, URL, size, digest, license and tests.

## 20. Rollback / removal rule

003A1 never deletes unknown user data.

- Operation-private download, staging and extracted temp trees are YORVA-owned and are removed on every terminal path.
- The official install directory is mutated only when Phase 3 preflight says the target is absent or retry-eligible.
- A failed extract leaves the official directory untouched if materialization had not yet swapped staging into place.
- YORVA does not uninstall Hermes, does not delete `%LOCALAPPDATA%\hermes` wholesale, and does not remove user `.env` / `config.yaml`.
- Removing 003A1 later means omitting the MSI resource and restoring online-archive-only acquisition; it does not authorize deleting an already installed Hermes tree.

## 21. Approval gate

Self-review of this draft against architecture, security, process ownership, source provenance, MSI packaging, licensing, testing and Phase 3 scope is recorded in §22.

Implementation, MSI embedding, adapter changes and Spec/doc synchronization wait for this exact Owner sentence in the task history:

```text
Owner authorization: I approve docs/phases/amendments/AMENDMENT-003A1-embedded-hermes-source.md and authorize its implementation as part of Phase 3 in automatic batch-gate mode.
```

If the Owner changes the design, this document is revised and approval is requested again. After approval, the status line becomes `ACCEPTED FOR IMPLEMENTATION` with the approval date. That status is not an audit PASS.

## 22. Amendment self-review (Stage A)

Performed 2026-08-17 before requesting Owner approval. No application or build implementation was written for 003A1.

### Verified repository state

| Check | Result |
| --- | --- |
| Branch | `phase/003-hermes-installation` |
| HEAD | `151774f3e58a5b68875a2c2dab937a15087dd639` |
| `origin/phase/003-hermes-installation` | `151774f3e58a5b68875a2c2dab937a15087dd639` |
| `main` / `origin/main` | `5b89d22ed5e7ae3f4374a26f0fcda54bdabc6bf9` |
| Peeled `phase-002-hermes-discovery-baseline-r1` | `5b89d22ed5e7ae3f4374a26f0fcda54bdabc6bf9` |
| Peeled `phase-002-hermes-discovery-baseline` | `a67de04e900bc3ddce99cd76501eec13586082ed` (immutable) |
| HEAD ancestor of Phase 2 R1 | yes |
| Phase 2 Gate | `AUDIT-002A1` PASS |
| Phase 3 Spec | READY — OWNER APPROVED; implementation IN_PROGRESS |
| Official pin | matches Spec / `release.go` |
| Working tree | dirty with existing Phase 3 + uncommitted ADR-0005 / mirror fallback; preserved, not reset |
| Index lock | absent |
| Other agents | a `codex` process was observed at drafting; no git lock. Stage A writes only this Amendment |

### Dimension review

| Dimension | Result |
| --- | --- |
| Architecture | PASS — Desktop → HTTP → application → Hermes adapter unchanged; Rust only resolves one resource path |
| Security | PASS WITH DOCUMENTED RESIDUAL RISK — compiled digest, runtime re-hash, fail-closed integrity, ZIP-slip limits; unsigned MSI cannot stop a local admin attacker |
| Process ownership | PASS — Go owns verify/extract/orchestrate; official remaining stages keep Job Object rules |
| Source provenance | PASS — official commit archive, measured twice, LICENSE preserved; digest not claimed as upstream attestation |
| MSI packaging | PASS AS SPECIFIED — resource, not Git blob; implementation must prove inspection |
| Licensing | PASS — MIT / Nous Research 2025 redistributed with LICENSE |
| Testing | PASS AS SPECIFIED — fixture strategy avoids 70 MiB in unit tests |
| Phase 3 scope | PASS — source fallback only; no full offline, Gitee, uv bundle, Phase 4 |
| Conflict with uncommitted mirrors | SUPERSEDES — ADR-0005 / `gitclone.com` / `kkgithub.com` are not the 003A1 path |

### Residual risks (accepted if the Owner approves)

1. Official GitHub ZIP is a working-tree export; `install.ps1` line endings differ until normalized and re-hashed.
2. After source materialization, dependency stages still need network.
3. SHA-256 does not replace MSI code signing.
4. A 68.5 MiB MSI increase is real Demo cost.
5. Local `go test -race` remains gcc-blocked on this host; exact-commit CI race stays mandatory.

### What Stage A did not do

- no Stage B implementation;
- no MSI rebuild;
- no edit to application, OpenAPI, Tauri resources or Phase 3 Spec;
- no commit, merge, tag, freeze or Phase 4 work;
- the ~70 MiB archive was not added to Git.

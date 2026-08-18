# YORVA Phase 3 Amendment 003A2 — China Dependency Distribution

> Status: ACCEPTED FOR IMPLEMENTATION
> Phase: Phase 3 — Hermes Installation
> Owner approval date: 2026-08-17
> Implementation: R1 REMEDIATION COMPLETE; AUDIT-003R2 PENDING
> Audit: historical `AUDIT-003` and `AUDIT-003R1` are FAIL; this status is not an audit PASS

## 1. Problem

Amendment 003A1 makes the exact reviewed Hermes source available from the Demo MSI when GitHub is unavailable. The remaining official `dependencies` and `node-deps` stages still download Python and npm artifacts. Direct access to their default overseas registries can be slow or unavailable on networks in mainland China.

## 2. Decision

For the pinned Phase 3 Demo release, the Hermes adapter injects these fixed download endpoints into every owned installer process:

| Ecosystem | Fixed endpoint |
| --- | --- |
| Python / uv / pip | `https://pypi.tuna.tsinghua.edu.cn/simple` |
| npm | `https://registry.npmmirror.com` |

The endpoints are compiled adapter policy, not request fields or user settings. Before spawning the reviewed installer, YORVA removes inherited `UV_*`, `PIP_*`, `PYTHON*` and `NPM_*` values and then appends exactly one fixed value for each approved setting. `UV_INDEX_STRATEGY=first-index` is fixed to avoid dependency-confusion resolution across indexes. User or process credentials are never forwarded.

The mirror changes transport only. The materialized official tree remains the immutable commit from 003A1 and supplies its reviewed `uv.lock` and `package-lock.json`. YORVA does not rewrite either file and does not patch or fork Hermes.

## 3. Failure behavior

- A mirror failure follows the existing bounded stage timeout, cancellation, process-tree cleanup and retry behavior.
- YORVA does not silently select an arbitrary public proxy or user-supplied registry.
- A later change of endpoint requires a reviewed Phase 3 amendment and regression tests.
- This is a network-resilient Demo path, not a fully offline installation. Python, uv, Node.js and other official prerequisite downloads may still require their reviewed upstream hosts.

## 4. Security and privacy

- HTTPS is mandatory.
- The installer receives no API keys, provider tokens, cookies or registry credentials from the parent environment.
- Logs and HTTP DTOs never expose environment contents.
- Python uses the official `uv.lock` preferred locked tier; npm uses the official package lock when applicable. This amendment does not weaken or rewrite upstream integrity metadata.
- Third-party mirror availability is not trusted as proof of package identity; lockfile digests and package-manager verification remain the integrity controls.

## 5. Scope boundaries

In scope: fixed PyPI and npm artifact transport for the existing Phase 3 installation stages, tests, documentation and MSI packaging.

Out of scope: a wheelhouse, vendored npm cache, dynamic mirror UI, user-configurable registry, Gitee fork, GitHub proxy, credentialed private registry, Hermes login/profile work, Phase 4 features, and changing the pinned Hermes version.

## 6. Acceptance evidence

- inherited hostile `UV_*`, `PIP_*` and `NPM_*` values are absent from the spawned environment;
- each approved fixed endpoint appears exactly once;
- provider secrets remain excluded;
- Go tests, protocol tests, Desktop checks, Rust/Tauri checks and MSI packaging pass;
- the MSI still contains the exact 003A1 archive and LICENSE once.

Owner authorization was provided in the active task on 2026-08-17: implement the agreed dependency-mirror change and package the result as MSI.

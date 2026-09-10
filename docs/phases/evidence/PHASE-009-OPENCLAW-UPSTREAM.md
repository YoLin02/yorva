# Phase 9 — OpenClaw upstream selection

> Date checked: 2026-09-08
> Status: Research / source selection only; no real Runtime or YORVA integration test claimed

## Release identity

| Field | Observed value |
| --- | --- |
| GitHub latest stable | `v2026.9.2`, name `openclaw 2026.9.2` |
| Published | `2026-09-05T20:00:07Z` |
| Draft / prerelease | `false` / `false` |
| Annotated tag object | `87d32a44ab9744903d36a33b399c26cfc2b078d6` |
| Peeled source commit | `3928bad9badfcb6c7d140530435e806fb8092190` |
| npm latest / exact version | `openclaw@2026.9.2` |
| npm CLI entry | `openclaw` → `openclaw.mjs` |
| npm engine | `>=22.22.3 <23 || >=24.15.0 <25 || >=25.9.0` |
| npm tarball | `https://registry.npmjs.org/openclaw/-/openclaw-2026.9.2.tgz` |
| npm integrity | `sha512-M6C7UsnX815nv26qBJFYGe6aGzv+ftZLRzV6S9oRXUtXg2Yn67eVntpssT94kgkquKVSeUxerUg0j1ONp4WYQg==` |
| npm shasum | `1816a24ecdc13f5e60dbca7a7990531a317b0274` |
| npm unpacked size / file count | `177691267` bytes / `9707` |

Primary metadata was read directly from [GitHub latest API](https://api.github.com/repos/openclaw/openclaw/releases/latest), the [tag ref](https://api.github.com/repos/openclaw/openclaw/git/ref/tags/v2026.9.2), [tag object](https://api.github.com/repos/openclaw/openclaw/git/tags/87d32a44ab9744903d36a33b399c26cfc2b078d6), [npm latest](https://registry.npmjs.org/openclaw/latest) and [exact npm metadata](https://registry.npmjs.org/openclaw/2026.9.2). npm metadata also advertises signatures and provenance attestations. This record does not claim those signatures, the tarball bytes or the full dependency graph have been locally verified. npm `gitHead` was absent; the GitHub source commit is not represented as proof of byte-for-byte npm/source equivalence.

The [official release index](https://docs.openclaw.ai/releases) identifies `2026.8.1` as OpenClaw 2.0 and lists `2026.9.2` as the newest stable release. Product branding and package version are recorded separately. See the [selected release](https://github.com/openclaw/openclaw/releases/tag/v2026.9.2).

## Surface selection and limits

| Official source | Planning evidence | Required implementation follow-through |
| --- | --- | --- |
| [CLI reference](https://docs.openclaw.ai/cli) | Named profile selection; version output; structured JSON reporting and nonzero failure exits | Pin actual success/error fixtures from the chosen release; check both exit code and schema |
| [Windows](https://docs.openclaw.ai/platforms/windows) | Native CLI/Gateway supported; Windows Hub has its own release; native service may use Scheduled Tasks or per-user Startup fallback | Test a normal user, exact profile/service ownership and persistence behavior; do not require Hub |
| [Multiple Gateways](https://docs.openclaw.ai/gateway/multiple-gateways) | Config, state, workspace, service identity and ports must be distinct; state-dir override alone does not isolate native services | Use named profiles and private workspace/ports; current page recommends spacing base ports by at least 120, but also retains an older 20-port example, so qualify the release and reserve non-overlapping derived ranges |
| [Gateway CLI](https://docs.openclaw.ai/cli/gateway) | Structured status/lifecycle commands; `--require-rpc` requests a stronger authenticated read probe; named service identity depends on profile/canonical location | Listener or acknowledged restart alone cannot prove health; validate exact status fields, terminal Operation states and native service side effects |
| [Setup CLI](https://docs.openclaw.ai/cli/setup) | `setup --baseline` initializes config/workspace without the interactive setup agent; other onboarding paths can detect/import existing state | Choose only the bounded required setup path; do not run guided migration or auto-import Hermes credentials |
| [Fleet CLI](https://docs.openclaw.ai/cli/fleet) | Container cells are experimental; Windows hosts untested; commands and schemas can change without a deprecation window | Do not make this the native Windows P9 integration route |

The evidence above selects a design direction. It does not demonstrate a safe profile deletion surface, a working ordinary-user lifecycle route, supported model credential writes or successful coexistence; those are explicitly separated into B0 qualification and subsequent test evidence. Mutable online docs may describe changes after the selected release. The release's actual documented/help-supported behavior wins when they differ.

## P8 reuse

[P8 final freeze evidence](PHASE-008-FINAL-MAIN-FREEZE.md) identifies the accepted source, CI, MSI and Windows checks. P9 starts at the documentation-only closeout `e95ed31d298c2548556e1295aacf7f84002b74ee`, preserving the accepted product source from tag `phase-008-local-product-hardening-baseline`. P9 must test its changed Runtime routing and recovery behavior rather than treating P8's single-Runtime checks as proof of coexistence.

## B0 refresh — 2026-09-10

Owner authorized implementation. GitHub latest and npm latest now agree on `2026.9.3`, published `2026-09-08T14:15:53Z`; source commit `1391f7cd2d40ab5bbcf2f5f831d3a64f520e72d7`. This supersedes the selected version above while retaining the original planning evidence. See the [release](https://github.com/openclaw/openclaw/releases/tag/v2026.9.3) and [exact npm metadata](https://registry.npmjs.org/openclaw/2026.9.3).

- Node engine: `>=24.16.0 <25 || >=26.1.0`; the former Node 22 path is unsupported.
- Tarball: `https://registry.npmjs.org/openclaw/-/openclaw-2026.9.3.tgz`.
- Integrity: `sha512-CzDHMeHdnjlIZ76ZyBb1lvLO4H/yBIMYXupFGGBN87x0853y3hg5nLAnKfxSKqLzqhbUKqy9ebDRAWWV4t8aew==`.
- npm shasum: `f01ed9f202e9ba84dc1a9c456876884aa787c919`; unpacked size `184216899`, file count `9997`.
- Isolated test Node: official `node-v24.16.0-win-x64.zip`; SHA-256 `edaca9bd58ec8e92037dac4e877d52f6b8f430b81c18b57e264b4e2fb111cd56`, matched the official [SHASUMS256](https://nodejs.org/dist/v24.16.0/SHASUMS256.txt), then `node.exe --version` returned `v24.16.0`.

## Native Windows interface qualification — 2026-09-10

The official package was installed in an isolated payload and exercised in a disposable Windows 11 x64 VM at medium integrity. The host's ordinary Runtime profiles were not used. OpenClaw remained at `2026.9.3` with Node `24.16.0` throughout these attempts.

| Actual observation (UTC) | Result and consequence |
| --- | --- |
| 10:01–10:04, two new named profiles, baseline setup, two foreground Gateways | Both returned successful authenticated `gateway status --json --require-rpc` read probes concurrently. Distinct profiles/ports remained separate. |
| 10:04:16, official unmanaged `gateway restart --json --wait 10s` | **FAIL**: the official CLI rejected a mismatch between its Gateway lock identity and verified listener on port 29120. YORVA does not bypass that identity check. |
| 10:18:37, composed restart | **PASS** for the primitive sequence: official `gateway stop --force --json`, confirmed STOPPED, owned foreground `gateway run`, authenticated RUNNING readback. |
| 10:18:54, second Gateway active after restarting the first | **PASS**: both authenticated Gateways were active. Stopping the first subsequently left the other active. |
| 10:19:21, `uninstall --state --yes --non-interactive` | **FAIL** for the intended complete deletion postcondition: exit 0 left the profile workspace. The implementation now also passes documented `--workspace`, after checking the only workspace is inside the owned profile and rejecting additional agents. Full deletion remains subject to the integration smoke. |
| Resuming a previously force-closed test VM | A stale native Gateway lock could prevent launch when a PID was reused. The official CLI failed closed. YORVA does not remove native locks or kill a process based on a stale PID. |

Local serial evidence: `.tools/p9/vm-b0c/serial.log`, `.tools/p9/vm-b0f/serial.log`, `.tools/p9/vm-b0g/serial.log`. Earlier archive extraction and test-harness setup failures are not product PASS evidence. These observations qualify individual interfaces; they do **not** by themselves establish the complete G1 result.

The final integration script is [`windows-second-runtime-smoke.ps1`](../../../scripts/windows-second-runtime-smoke.ps1). It uses the real YORVA sidecar and authenticated local API, requires a disposable Windows user profile and medium integrity, and exercises one Hermes instance plus two OpenClaw instances. Its complete result will be recorded separately after execution.

The first integration attempt reached supported Hermes discovery, then failed its fresh inventory assertion. Python environment relocation and this inventory failure are retained as unsuccessful attempts; neither has been reported as a passing integration run.

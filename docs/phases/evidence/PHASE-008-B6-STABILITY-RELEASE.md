# Phase 8 B6 — Stability and Release Gate Evidence

> Date: 2026-09-04–2026-09-05
> Branch: `phase/p8-local-product-hardening`
> State: **PASS — B6 local internal-candidate Gate complete; public-release Gates pending**
> Release class ceiling: **internal candidate**

This evidence does not claim public-release readiness, an independent audit, production
Windows signing, final-main CI, merge, tag, or freeze.

## P8 Gate traceability

| Requirement | Current evidence | State |
| --- | --- | --- |
| P8-D1–D4 support, identity, retention and supported migration | `PRODUCT_SUPPORT.md`, `PRODUCT_SUPPORT.zh-CN.md`, commit `72ec369`, and `PHASE-008-B1-MIGRATION-PROTECTION.md` | COMPLETE |
| Desktop/daemon/reboot recovery with authoritative reconciliation | `PHASE-008-B2-CRASH-REBOOT-RECOVERY.md` and commit `3ee4580` | COMPLETE |
| Fresh/Upgrade/Repair/Uninstall/Reinstall Windows lifecycle | `PHASE-008-B3-WINDOWS-INSTALLER-LIFECYCLE.md`, commits `56df01f` and `412d3c0` | COMPLETE |
| P8-D5 complete-package verified YORVA update | `PHASE-008-B4-YORVA-UPDATE.md`, commits `81b4bfa`, `9be7435`, `8d1b162`, `6a75cde`, and `7407992` | COMPLETE — internal candidate |
| P8-D6 telemetry remains absent | support and security contracts plus repository review | COMPLETE |
| P8-D7 bounded sanitized diagnostics | `PHASE-008-B5-DIAGNOSTICS.md` and commit `5ca5f0a` | COMPLETE |
| P8-D8 production signature | Owner-confirmed signing material is unavailable | NOT PASS — public release blocked |
| P8-D9 per-Batch commit discipline | B0–B5 committed; this evidence is included in the authorized B6 automatic Gate commit | COMPLETE for local candidate |
| Three-Profile 4–8 hour stability and final local candidate Gate | this evidence record | COMPLETE |
| Exact B6 CI/MSI workflow, independent audit, Owner decision, merge/tag/freeze | requires the final B6 commit and explicit Owner-controlled actions | NOT STARTED |

The P8-D6 source and direct-dependency review found no production telemetry, analytics,
Sentry, OpenTelemetry, or equivalent sender. `pnpm-lock.yaml` mentions
`@opentelemetry/api` only as an unselected optional peer declared by the development test
runner; no corresponding package snapshot or application import exists. Local diagnostic
export and local bounded logs remain explicit user/device-local behavior, not telemetry.

## Qualification scope

The Windows stability harness starts the real packaged `yorvad` through its private
bootstrap protocol and gives its child processes fresh isolated `APPDATA`, `LOCALAPPDATA`,
YORVA data, Hermes home, credentials, logs, diagnostics, and Backup roots. A test-only
Hermes `0.20.5` CLI fixture implements only the qualified commands needed to exercise
YORVA's real application, adapter, persistence, SecretStore, process-containment, HTTP,
Operation, and authoritative-readback paths.

Each run covers:

- three Hermes Profiles and parallel accepted start/restart/stop Operations;
- model credential write plus authoritative model readback on all three Profiles;
- managed Skill install/update/disable/enable/readback/remove;
- reviewed `yorva-mcp-test` bind/test/readback/unbind;
- the closed Channel status read set;
- encrypted Runtime Backup create/verify/readback/delete;
- periodic sanitized diagnostic export and credential/session-token scans;
- bounded daemon process replacement and authoritative inventory readback;
- process memory, working set, handle, thread, CPU, database, product-data, complete
  work-root, Operation, diagnostic, and output-size samples;
- terminal active-Operation and owned-process checks.

The harness has no user-selected Runtime or data path, disables HTTP proxy use, and does
not touch the user's YORVA or Hermes roots.

## No shared-host reboot

The Owner requested that this B6 run not restart the shared Windows host. Routine B6
recovery therefore uses owned Desktop/daemon process replacement and records
`hostRebooted=false`. Three consecutive Desktop recovery smoke runs passed before the
soak work: unexpected post-READY daemon exit, new authenticated daemon identity,
single-instance enforcement, Desktop kill/reopen, and no surviving exact-candidate
process. A post-soak local rerun exposed that Tauri obtains app data from the Windows
Known Folder API and therefore ignores child `APPDATA`/`LOCALAPPDATA` overrides. The run
was stopped and all exact candidate processes were removed, but it correctly cannot be
counted as isolated evidence. The smoke now requires an explicitly disposable Windows
profile and refuses to start if that profile already contains stable or legacy YORVA
product data. Its exact final-candidate rerun is performed in a fresh disposable guest,
without restarting the shared host. `HERMES_HOME` remains isolated and child `PATH` is
restricted to Windows system directories.

The machine-reboot requirement is not silently dropped or represented by process restart.
It remains satisfied by the earlier disposable Windows 11 guest reboot/login result in
`PHASE-008-B2-CRASH-REBOOT-RECOVERY.md`; the development host was not rebooted for that
evidence either.

## Harness defect found before qualification

The first attempted long run exposed a test-harness defect after ten cycles. The daemon
had completed encrypted Backup publication and index insertion, but the harness had read
only the bootstrap handshake from redirected process output. Its undrained stderr pipe
eventually filled and blocked the daemon while writing the safe completion log, leaving
the Operation RUNNING until the harness timeout.

This failed run is retained under the ignored qualification directory and is not counted
as product PASS evidence. The harness now:

- drains stdout/stderr for every daemon generation;
- caps persisted stdout at 64 KiB and stderr at 2 MiB;
- scans all current and prior bootstrap tokens plus the credential canary;
- owns pre-handshake failure cleanup;
- writes PASS only after daemon exit, output scan, and a stable no-orphan check.

Compressed regression runs then passed:

| Run | Result | Coverage |
| --- | --- | --- |
| pipe-drain pressure | PASS — 12 cycles, 133 Operations, 3 diagnostics | repeated lifecycle plus heavy Skill/MCP/Backup cycle beyond the original failure point |
| reconnect | PASS — 5 cycles, 70 Operations, 2 diagnostics, 1 daemon replacement | output drain reset and authoritative reconnect |
| final ownership | PASS — 1 cycle, 24 Operations, 1 diagnostic, 1 daemon replacement | final script hash, post-exit output scan and orphan check |

The first isolated daemon goroutine test also exposed a test-helper defect: an absent
temporary `LOCALAPPDATA` correctly made Restore recovery fail closed before handshake,
while the helper waited only for handshake and reached the Go test timeout. The helper
now races handshake against early daemon exit and a 45-second startup deadline, closes
all test pipes, and creates its isolated environment roots before launch. The corrected
twelve-lifecycle test passed three consecutive runs, and the complete daemon package
passed afterward.

## Exact stability inputs

- `yorvad.exe` SHA-256:
  `44CDA742CA1BCA45C74A082296B8B7EF3BE9EC438717E0B3A9F8F65B10D068F2`;
- Hermes fixture SHA-256:
  `E39170C4CBB92AF67B3EE1DE93C79465A3B10DE874EFC216EB762809E715537E`;
- `windows-stability-soak.ps1` SHA-256:
  `5722AD2DAA08AAF21FFFDA2E8280D35654FB8F12A95B7C4482563B29D094F07E`;
- reconnect-pressure root: ignored `.tools/p8-soak/qualification-20260904155949`;
- single-daemon continuity root: ignored
  `.tools/p8-soak/continuity-20260904200025`;
- duration: 240 minutes minimum;
- `hostRebooted`: `false`.

The reconnect-pressure process exited normally after `14,401.387` seconds. Its durable
summary records PASS, 225 cycles, 3 Instances, 2,260 Operations, 46 diagnostic exports,
7 daemon reconnects, 226 complete samples, and `hostRebooted=false`. It observed 8 unique
daemon PIDs. The peaks were 36,143,104 bytes working set, 68,534,272 bytes private memory,
319 handles, 23 threads, 1,518,896 bytes product data, and 5,710,134 bytes across the
complete work root. Persisted daemon stdout was empty and bounded stderr was 80,182
bytes. The terminal stop wave added three Operations after the last sample; the process
then drained and scanned output, found no owned Hermes child, wrote the PASS summary, and
exited with code zero.

Interim reconnect checkpoints are evidence of continued work, not the final PASS. At
cycle 30, daemon PID `51868` exited and PID `58432` completed cycle 31 after a fresh
authenticated handshake and three-Profile inventory readback. At cycle 60, PID `58432`
exited and PID `37000` completed cycle 61 with 621 cumulative Operations and 13 diagnostic
exports. At cycle 90, PID `37000` exited and PID `62152` completed cycle 91 with 921
cumulative Operations and 19 diagnostic exports. At cycle 120, PID `62152` exited and
PID `53444` completed cycle 121 with 1,221 cumulative Operations and 25 diagnostic
exports. At cycle 150, PID `53444` exited and PID `50528` completed cycle 151 with 1,521
cumulative Operations and 31 diagnostic exports. At cycle 180, PID `50528` exited and
PID `54816` completed cycle 181 with 1,821 cumulative Operations and 37 diagnostic
exports. At cycle 210, PID `54816` exited and PID `56556` completed cycle 211 with 2,121
cumulative Operations and 43 diagnostic exports. All retired processes were absent, each
replacement remained the sole exact-candidate daemon, the candidate hash stayed fixed,
and prior-generation bounded
output was durably drained with zero credential-canary matches. The Skill/Backup staging
directories were empty at the second checkpoint.

## Complementary resource-growth qualification

The completed reconnect-pressure run deliberately replaced the daemon every 30 cycles.
This is strong evidence for repeated authenticated reconnect, output-drain reset, state
reconciliation, and owned-process cleanup, but every replacement resets process memory,
handle, thread, and CPU counters. It therefore cannot by itself prove that one
continuously running daemon has no slow resource leak.

The same exact candidate and harness completed a second 240-minute window with
`DaemonRestartEveryCycles` set beyond the possible cycle count. It retained daemon PID
`14460` for the complete run, passed 225 cycles, 2,260 Operations, 46 diagnostic exports,
and all terminal ownership checks across 226 samples. Peak working set was 39,882,752
bytes, peak private memory was 70,463,488 bytes, peak handles were 336, peak threads were
24, product data reached 1,526,388 bytes and the complete work root reached 5,635,562
bytes. Persisted stdout was empty and bounded stderr was 75,386 bytes.

The combined analyzer passed both windows: total duration `28,802.669` seconds, seven
pressure-window reconnects, one continuity PID, and first-versus-last continuity median
growth of 3,706,880 bytes working set, 3,379,200 bytes private memory, 28 handles, and one
thread. The two complementary windows reach the eight-hour P8 target without restarting
the shared Windows host. Desktop/daemon replacement behavior remains independently
covered by the reconnect-pressure run and final exact-candidate recovery smoke.

## Final disposable Desktop recovery

The post-soak recovery Gate ran in a fresh Windows 11 Enterprise Evaluation 25H2 x64
QCOW2 overlay based on the already-qualified disposable image. QEMU/KVM ran inside WSL2;
the shared Windows host was not restarted. The guest used its real per-user Windows Known
Folder paths, began with no stable or legacy YORVA product data, and shut itself down after
the result. The reusable base image remained unchanged.

Exact inputs and result:

- Desktop SHA-256:
  `C8E3B5AFAF6180891785725216FADE196381AC3EE425E4F68E1189D53D7B8CCF`;
- daemon SHA-256:
  `44CDA742CA1BCA45C74A082296B8B7EF3BE9EC438717E0B3A9F8F65B10D068F2`;
- recovery script SHA-256:
  `C08F1222E19367EDEC137BF8BC3405489F3A4FC295CDBCC7CB7EE1D777D714EA`;
- result: PASS — authenticated readiness, forced daemon replacement, retained single
  Desktop ownership, duplicate-launch yield, Desktop kill/reopen and final zero exact
  candidate processes.

The same script was then invoked against the developer's normal profile. It rejected the
run before process launch because existing YORVA product data was present; exact candidate
Desktop/daemon counts remained zero before and after. This negative check confirms that
the qualification command cannot silently reuse the Owner's daily product data.

The tracked `windows-stability-evidence.ps1` performs a second-layer result check after
both windows finish. It requires durable PASS summaries, complete sample boundaries,
three Profiles, a combined eight-hour duration, at least three pressure-window daemon
replacements, exactly one continuity-window daemon PID, 512 MiB process-memory ceilings,
bounded handles/threads/storage, and no more than 64 MiB growth between the first and
last continuity-window resource medians (with separate handle/thread limits).

The first automatic continuation attempt used the inherited Windows PowerShell host and
failed before starting the continuity run because that environment could not autoload
`Get-FileHash`. It did not alter the completed pressure result or start another daemon.
The same three hashes were then verified explicitly with the current PowerShell host and
the continuity run started at `2026-09-04T12:00:25Z`; its runner PID is `64060`, initial
daemon PID is `14460`, and cycles 0 and 1 completed normally. This orchestration failure
is harness evidence, not a product failure, and is not counted as a qualification window.

## Local validation completed while the soak runs

Passed:

- `go test ./...`;
- `go vet ./...`;
- `go test -race ./...` in an unprivileged WSL temporary toolchain;
- repeated daemon start/stop goroutine ownership using `runtime.NumGoroutine` after
  twelve post-warmup lifecycles;
- stability-evidence analyzer synthetic positive, changed-continuity-PID negative, and
  completed-pressure-as-continuity negative checks;
- Desktop Vitest — 24 files / 146 tests;
- TypeScript typecheck and ESLint;
- OpenAPI generation/no-diff and lint;
- Vite production build;
- Rust format, tests, Clippy `-D warnings`, and check;
- CI-exact `pnpm audit --audit-level low` — no known dependency vulnerability;
- `govulncheck ./...` — no called vulnerability;
- `cargo audit` — exit 0 with 17 allowed warnings, including non-Windows GTK3
  unmaintained advisories and the recorded `glib 0.18.5` unsound advisory on an
  unreachable non-Windows dependency path. Five `unic-* 0.9.0` unmaintained warnings
  remain in the Windows graph through the upstream `tauri-utils -> urlpattern` chain;
  they are maintenance warnings rather than known vulnerabilities and have no local
  direct-dependency substitute. The GTK, `glib`, and `proc-macro-error` warning packages
  are absent from the `x86_64-pc-windows-msvc` graph.

The local Race run used official verified temporary archives without installing system
packages or rebooting Windows:

- Go `1.26.6` Linux archive SHA-256:
  `708effb774be8237570d0add163225abbdfaf4fca28b2611df167beba4feef89`;
- Zig `0.16.0` Linux archive SHA-256:
  `70e49664a74374b48b51e6f3fdfbf437f6395d42509050588bd49abe52ba3d00`.

## Focused security/recovery corrections

The B6 review additionally requires:

- Windows product-identity migration to use non-following presence checks and reject
  junction/reparse entries, ordinary symbolic links, and dangling root links;
- updater `state.json` validation to reject dangling links before staging and retain the
  prior durable state until the new flushed file is atomically published;
- the native diagnostic client to bypass system proxies and reject redirects before a
  daemon bearer credential can be forwarded outside its private loopback endpoint, and
  to reject link/reparse Save As destinations before creating a partial file.

Focused Rust tests cover the changed migration and updater state paths. A local redirect
server regression also proves that diagnostic download rejects the redirect response and
that its target receives no connection. The complete native/Frontend/Go Gate and non-MSI
Desktop rebuild remain required after the soak.

`docs/SECURITY.md` is the repository's authoritative threat model; there is no separate
threat-model document. Its Phase 8 refresh now covers reparse-point migration escape,
update-state interruption and package tampering, diagnostic proxy/redirect and secret
leakage, stale/orphan recovery state, and qualification contamination of user data. The
review preserves the existing local-administrator/root compromise exclusion and does not
broaden any management authority.

## Final local validation

The final working-tree Gate passed:

- `go test ./...`, `go vet ./...`, WSL `go test -race ./...` and `govulncheck ./...`;
- OpenAPI lint, generation/no-drift, TypeScript typecheck, ESLint, 24 Vitest files / 146
  tests, Vite production build and `pnpm audit --audit-level low`;
- Rust format, 28 tests, Clippy `-D warnings`, check and `cargo audit`;
- PowerShell parser checks for all three B6 scripts and the combined stability analyzer;
- exact-candidate disposable Windows Desktop recovery smoke;
- final sidecar plus non-MSI release Desktop build.

The final focused review found no unresolved Critical or High defect. `cargo audit` exits
zero with the 17 documented allowed maintenance warnings; it reports no newly introduced
known vulnerability. The Vite chunk-size advisory and Windows linker import-library
message remain non-blocking build warnings.

## Remaining public-release gates

The current phase branch has been added to the Windows MSI workflow trigger, and the
Windows native CI job now runs the disposable-profile Desktop/daemon recovery smoke after
building the release executable. The exact B6 commit still needs an authorized push and
its GitHub CI/MSI workflow results. Production signing material is unavailable, so
signed-MSI/public-release readiness remains blocked by the approved support contract. A
single implementation agent performed this focused review; it must not be presented as
the independent audit required for public release. Merge, tag, freeze, and final-main CI
require an explicit Owner decision.

# Phase 9 validation evidence

Date: 2026-09-10. Status: **IN_PROGRESS**; not a gate PASS or frozen baseline.

## Candidate and scope

Branch `codex/phase9-openclaw`, based on P8 documentation closeout
`e95ed31d298c2548556e1295aacf7f84002b74ee`. The implementation adds the real
OpenClaw adapter, Runtime-scoped inventory and management dispatch, capability-driven
Desktop selection, recovery across accepted installations, and Windows process ownership.
It adds no product dependency or database migration. Existing SQLite installation/native
identity uniqueness and protection columns are reused.

## Verification currently completed

| Check | Actual result |
| --- | --- |
| Go `test ./...` | PASS locally on Windows after the OpenClaw metadata and startup-inventory fixes; subsequent Hermes readiness change has focused regression verification and requires final CI |
| Go `vet ./...`, sidecar build | PASS locally; final candidate CI remains required |
| Desktop typecheck, lint, test, build | PASS; 151 tests in 25 files; Vite reports its existing chunk-size advisory |
| API lint and generated schema consistency | PASS; generation did not change the freshly generated schema |
| Desktop UI | English/Chinese Runtime switching, inventory, capability-filtered management, create dialog, protected default, and health detail inspected in the browser with disposable API fixtures; this is UI evidence, not a real-Runtime smoke |
| Official OpenClaw Windows primitives | Two simultaneous authenticated Gateways, composed restart and stop isolation passed; see [upstream qualification](PHASE-009-OPENCLAW-UPSTREAM.md) |
| Full real G1 | In progress; no complete PASS yet |
| Final CI / MSI / audit | Pending |

## Native fixture and failures retained

Windows 11 x64 disposable QEMU/KVM guest; task worker verifies medium integrity
(`S-1-16-8192`, no high-integrity SID). Real OpenClaw `2026.9.3` and independent Node
`24.16.0`; real Hermes `0.20.5` with its isolated Python `3.11.15`. Normal host profiles
are not used. Guest shutdown is fixture cleanup, not a host reboot.

- Initial archive relocation produced an unusable copied uv launcher and slow first
  Python imports. The fixture uses the official pip-generated entry point in its
  guest venv; Hermes source is unchanged. These attempts do not count as product PASS.
- The actual OpenClaw package metadata is 135,311 bytes. A 128 KiB guard incorrectly
  rejected it. The bounded guard is now 256 KiB, with regression tests for the actual
  release size and rejection beyond the bound.
- Native attempt 8 passed real creation of one Hermes and two OpenClaw instances,
  same-name identity separation and unsupported-capability rejection at
  `2026-09-10T11:13:03Z`, then failed Hermes startup with
  `LIFECYCLE_POSTCONDITION_FAILED`.
- Diagnostic attempt 11 used the official Hermes CLI directly. Launch was recorded
  at `11:20:13.332Z`, and official status reported RUNNING at `11:21:47.377Z`.
  Runtime logs showed normal Gateway/cron initialization. The former 15-second
  post-launch wait was insufficient. Startup readback now has a 120-second budget,
  retains authoritative-state checks and cancellation, and has a delayed-readiness
  regression test. This diagnostic is not a full G1 pass.

Local detailed logs are retained under `.tools/p9/vm-native8`, `vm-native10`,
`vm-native11` and the later full-run directory. Only sanitized diagnostic tails and
non-secret smoke status were sent to the serial evidence stream. Runtime configuration
and credential plaintext are not committed.

The shared 30-second startup inventory budget prevents one Runtime from exhausting
Desktop's 45-second bootstrap deadline. It does not assert recovery READY:
`/node/recovery` still checks every previously accepted installation. Tests verify
cancellation ownership and that a failed Runtime cannot be hidden by a healthy one.

## Final gate record

The first candidate `a731dcfe4c8de5a8fea559ee28d669c83812702f` was pushed for
[CI 101](https://github.com/YoLin02/yorva/actions/runs/34471364540) and
[MSI 39](https://github.com/YoLin02/yorva/actions/runs/34471364498). CI's dependency
audit rejected inherited development dependencies under newly published advisories:
[js-yaml GHSA-2883-xcg3-v3hh](https://github.com/advisories/GHSA-2883-xcg3-v3hh) and
[Vitest GHSA-82fw-gwwq-j7x9](https://github.com/advisories/GHSA-82fw-gwwq-j7x9).
Vitest and its matching packages are patched from 4.1.10 to 4.1.11. A narrow override
patches the exact `@redocly/openapi-core@1.34.19` parent from js-yaml 4.3.1 to 4.3.2;
the override should be removed when that parent's pinned parser is updated. No
unrelated dependency or framework was upgraded. After the patch, local `pnpm audit
--audit-level low`, API lint/generation consistency, Desktop typecheck/lint, all 151
tests and production build passed. The failed CI evidence is retained and the
security gate remains unchanged.

G1, G2 and G3 will be recorded against the final candidate and actual results before
the Phase Spec is marked FROZEN. Required checks are not waived by the partial results
above.

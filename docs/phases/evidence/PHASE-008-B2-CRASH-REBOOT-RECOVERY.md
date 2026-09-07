# Phase 8 B2 Crash, Restart and Reboot Recovery Evidence

> 2026-09-07 audit update: the historical scenario results below remain valid, but
> the affected Batch Gate is reopened by
> [AUDIT-008](../audits/AUDIT-008-local-product-hardening.md). Authoritative update
> postcheck requires correction (HIGH-001); B4 also requires download/process-restart
> recovery (MEDIUM-001). This record does not establish current Gate completion.

- Date: 2026-09-04 (Asia/Shanghai)
- Branch: `phase/p8-local-product-hardening`
- Parent baseline: `fc224a8`
- State: **PASS — B2 Gate complete**

## Implemented recovery order

Before the daemon publishes its authenticated handshake it now:

1. opens and protects the supported database schema;
2. recovers the Hermes filesystem restore journal;
3. recovers or terminalizes stale install, Instance, model validation, model profile,
   lifecycle, Channel, Skill, MCP, Backup and Runtime management Operations;
4. performs one bounded Hermes discovery;
5. reconciles the authoritative Hermes Profile inventory when the Runtime is supported;
6. publishes READY only after those steps succeed.

Lifecycle and Channel reads are not persisted as live state. Their stale Operations are
resolved through authoritative adapter reads during startup, and normal reads query the
adapter again rather than presenting SQLite metadata as live state.

## Desktop ownership and recovery

- The native startup deadline is 45 seconds because the daemon-owned Runtime discovery
  deadline is 35 seconds.
- An unexpected post-READY daemon exit permits one automatic sidecar restart per Desktop
  process. A second exit is terminal, preventing an infinite respawn loop.
- The replacement authenticated handshake emits `daemon-session-changed`; React then
  invalidates the old session so the new loopback port and token replace stale client state.
- Product-data preparation failures keep the Desktop UI alive and return a stable native
  error code. The UI maps product-identity conflicts and other product-data failures to
  bounded, actionable, localized guidance without displaying native error details.
- Parent EOF monitoring starts immediately after the private bootstrap message. If the
  Desktop disappears before the handshake, the shared startup context is canceled and the
  sidecar exits promptly instead of finishing a 35-second discovery as an orphan.

## Migration-discovered security correction

The first exact Desktop recovery run found that a copied, pre-existing `secrets/v1`
directory initially inherits its ACL. Protecting the parent `secrets` ACL first can remove
that inherited child access before the child is hardened. The Windows SecretStore now
validates and protects an existing child before protecting the parent, then validates the
complete no-reparse chain again. A Windows regression fixture covers this copied-tree
shape. No plaintext fallback, directory deletion or credential rewrite was introduced.

## Automated evidence

| Check | Result |
| --- | --- |
| `go test ./...` | PASS |
| `go vet ./...` | PASS |
| Desktop Vitest | PASS — 24 files / 143 tests at this checkpoint |
| Desktop typecheck and ESLint | PASS |
| Desktop Vite build | PASS |
| Rust format, 18 tests, Clippy `-D warnings`, check | PASS |
| Sidecar build | PASS |
| `windows-lifecycle-smoke.ps1` | PASS — graceful control, parent EOF/reopen and forced daemon restart with the same data roots |
| `windows-desktop-recovery-smoke.ps1` | PASS — readiness-log-qualified post-READY daemon restart, new daemon identity, single-instance enforcement, Desktop kill/reopen and no surviving exact-candidate process |
| Reboot smoke `Prepare` | PASS — exact release paths/hashes, login item, one Desktop-owned daemon and clean shutdown recorded |
| Reboot smoke pre-reboot negative check | PASS — `Verify` rejected the unchanged Windows boot identity before inspecting post-login state |
| Disposable Windows reboot/login smoke | PASS — Windows Boot Manager restarted the guest, automatic login launched the exact candidate once, Runtime/Profile reconciliation completed, a second Desktop launch yielded, and the incumbent daemon PID was retained |
| Non-bundled release Desktop build | PASS |

The Desktop automatic Hermes-start regression proves a stopped default Profile is started
once, a failed automatic start stays in a retryable UI state without another automatic
attempt, and an explicit user retry is still available. Existing authoritative inventory
tests cover external Profile disappearance, refusal to clear a reappeared Profile, and
successful cleanup only after a fresh missing readback.

## Disposable Windows reboot evidence

The blocking reboot check ran without rebooting the shared development host. A fresh
Windows 11 Enterprise Evaluation 25H2 x64 guest ran under WSL2/KVM/QEMU with a disposable
QCOW2 overlay. The Microsoft installation ISO SHA-256 was
`7B4AC87391B659F7724229682B642256289A1C00504056249F0F12029157D3D2`.
The test payload was attached as a read-only ISO and the guest shut itself down after the
result, leaving the reusable base image unchanged.

Exact candidate identities:

- Desktop SHA-256: `ED16DA0EB317A1ABE06A34BD3B7EC7708D9AF40753EC84DAC1D9D948CAA922D4`;
- daemon SHA-256: `C6B5076E9BB61141958EE0AC85B95EF15E9DDA26BE139FD3800DB025F236264F`.

To avoid copying the developer's private Hermes home and credentials, the guest contained
a minimal CLI contract fixture at the official Hermes launcher location. The fixture
implemented only `hermes --version` and `hermes profile list`. All discovery execution,
output parsing, SQLite reconciliation, Desktop/daemon ownership, autostart registration,
Windows reboot, login recovery and single-instance behavior used the real release
candidate. This B2 check therefore proves YORVA recovery boundaries; it is not evidence for
Hermes installation, model, Channel, Skill or MCP behavior.

The guest serial evidence sequence was:

```text
YORVA_VM_JOB_START E:\yorva-job.ps1
YORVA_B2_PREPARE_OUTPUT Windows reboot recovery smoke prepared. Reboot, sign in, then run Verify.
YORVA_B2_PREPARE_PASS
<Windows Boot Manager restarted the guest>
YORVA_VM_JOB_START E:\yorva-job.ps1
YORVA_B2_VERIFY_OUTPUT Windows reboot/login recovery smoke: PASS
YORVA_B2_VERIFY_PASS
```

The first disposable attempt also exposed that a two-second process snapshot could collide
with the one permitted cold-start daemon replacement. The smoke now requires continuous
single Desktop-owned daemon stability, reports exact process relationships on timeout, and
after the duplicate-launch probe requires the original daemon PID to remain authoritative.

## R1 acceptance — 2026-09-07

The historical AUDIT-008 FAIL and its source evidence remain unchanged.
[AUDIT-008R1](../audits/AUDIT-008R1-local-product-hardening.md) now returns **PASS** for
`43ac29152d3de1fa227937f35ccdd12908bac976`, closing both original findings.
[Remediation evidence](PHASE-008R1-REMEDIATION.md) includes CI #96 / Windows MSI #35,
real installed-version/failed-readback rejection and download-process-crash/restart/retry.
All three new Windows update scenarios pass. The retained eight-hour B6 workload was
not rerun. The internal candidate is accepted under Owner Amendment 008A1; final-main
CI/package/update checks and the annotated tag still precede formal freeze.

# Phase 8 B4 — YORVA Update MVP Evidence

> 2026-09-07 audit update: the historical scenario results below remain valid, but
> the affected Batch Gate is reopened by
> [AUDIT-008](../audits/AUDIT-008-local-product-hardening.md). Authoritative update
> postcheck requires correction (HIGH-001); B4 also requires download/process-restart
> recovery (MEDIUM-001). This record does not establish current Gate completion.

> Date: 2026-09-04
>
> Result: **PASS — internal Windows candidate**
>
> Candidate commit: `6a75cded364fac79ccfb625d8a3e83038681f0be`

## Scope

B4 qualifies the complete-package YORVA update path without adding an arbitrary
URL, path, command, argument, environment or header surface:

```text
fixed signed metadata
→ bounded MSI download
→ source/signature/hash/version verification
→ Desktop and daemon drain
→ native MSI handoff
→ relaunch
→ migration and daemon recovery
→ Runtime/Instance authoritative readback
→ terminal update result
```

The Settings summary and independent update detail view use the typed native
commands and persisted update state. A release build without a configured public
key reports updates as disabled. The qualification-only source and CA are compiled
only with `update-qualification` and are absent from the normal release build.

## Exact candidate

| Field | Value |
| --- | --- |
| File | `YORVA_0.4.0_x64_en-US.msi` |
| Size | `150302720` bytes |
| SHA-256 | `ABB78642C15B54A7F0516BBAA04ED42AFBB3BFA9C039388029BB1CB5E3B9EE62` |
| Source commit | `6a75cded364fac79ccfb625d8a3e83038681f0be` |
| Source state | clean |
| Product name/version | `YORVA` / `0.4.0` |
| UpgradeCode | `{E793918B-37EB-5E2F-B866-5FC4AA3AC75A}` |
| Scope | per-user, limited privileges |
| Authenticode | `NotSigned`; internal candidate only |

The internal Ed25519 qualification public-key raw-byte SHA-256 is
`D8D62C3352290235C2B04043B2F2080C10E0AA874D4F20A3C857F374368F07F7`.
No private key is recorded in repository evidence.

## Disposable Windows qualification

The exact candidate ran in fresh QCOW2 overlays based on Windows 11 Enterprise
Evaluation 25H2 x64 under WSL2/KVM/QEMU. The OS disk used its installed native
NVMe controller and every scenario started from the same immutable base. The
worker ran at medium integrity. Only guest shutdown/replacement was used: the
host Windows system and WSL distribution were not restarted, and unrelated WSL
container services were left running.

The baseline was a qualification-only `0.3.2` package built with the same fixed
update public key and test CA. The fixture endpoint was fixed at
`https://10.0.2.2:18443`; it was not supplied by the UI or caller.

### Exact-candidate outcomes

| Scenario | Serial evidence | Result |
| --- | --- | --- |
| Older `0.3.2` → exact `0.4.0` | `2026-09-04T06:42:05.2789984Z YORVA_B4_UPDATE_PASS Happy` | PASS: install, relaunch, installed-version readback and Runtime/Instance authoritative readback |
| Tampered exact-candidate stream | `2026-09-04T06:45:21.0124285Z YORVA_B4_UPDATE_PASS Tamper` | PASS: `UPDATE_INTEGRITY_FAILED`; `0.3.2` Desktop and daemon remained usable |
| Interrupted exact-candidate stream | `2026-09-04T06:48:49.4373530Z YORVA_B4_UPDATE_PASS Interrupted` | PASS: `UPDATE_DOWNLOAD_FAILED`; `0.3.2` Desktop and daemon remained usable |

### Installer failure outcome

The qualification-only MSI failure fixture had SHA-256
`396AEA43A74905DF272619148B857D2A415936072E313E17F63735B6A5CD3D4F` and
the same product identity. It intentionally failed before replacing the baseline:

```text
2026-09-04T06:33:43.4311646Z YORVA_B4_UPDATE_PASS InstallerFailure
```

The test observed `UPDATE_INSTALL_FAILED`, retained `0.3.2`, and confirmed that
the old Desktop and daemon remained usable. Unit coverage also proves that an
installer failure or a terminal daemon reconcile failure cannot produce success.

## Installer handoff finding and repair

The first real run exposed a race in the native handoff: Windows Restart Manager
could relaunch YORVA before the helper persisted the MSI exit code, and
`$LASTEXITCODE` was not reliable for this GUI executable. The final implementation:

- waits for the old Desktop PID to exit before starting MSI;
- disables Restart Manager for this already-drained update invocation;
- uses `Start-Process -Wait -PassThru` for the real exit code;
- writes a fixed-path installer log and exit result;
- relaunches YORVA exactly once after MSI returns;
- maps a bounded old-process drain timeout to a failed installer result.

The successful exact-candidate run above is after this repair.

## Automated checks

| Check | Result |
| --- | --- |
| Desktop Vitest | 24 files, 145 tests PASS |
| Desktop TypeScript typecheck | PASS |
| Desktop ESLint | PASS |
| Rust tests, normal feature set | 24 PASS |
| Rust tests, `update-qualification` | 24 PASS |
| Rust Clippy, all targets/features, `-D warnings` | PASS |
| Metadata generator tests | signed fixed source, qualification source and unsigned-public rejection PASS |
| PowerShell parser checks | updater smoke scripts PASS |
| MSI contract inspector | exact 0.4.0 candidate PASS |
| Non-MSI Tauri release build with update public key | PASS |

The only build diagnostics were the already-recorded Vite chunk-size warning and
Windows linker informational messages.

## Gate conclusion

P8-B4 is complete for an internal Windows candidate. The fixed-source metadata,
download bounds, cryptographic verification, deterministic installer handoff,
post-install migration/reconcile and fail-closed negative paths are all exercised.
Public release readiness remains intentionally blocked on production Windows
signing material; B4 does not weaken that P8 decision.

## R1 acceptance — 2026-09-07

The historical AUDIT-008 FAIL and its source evidence remain unchanged.
[AUDIT-008R1](../audits/AUDIT-008R1-local-product-hardening.md) now returns **PASS** for
`43ac29152d3de1fa227937f35ccdd12908bac976`, closing both original findings.
[Remediation evidence](PHASE-008R1-REMEDIATION.md) includes CI #96 / Windows MSI #35,
real installed-version/failed-readback rejection and download-process-crash/restart/retry.
All three new Windows update scenarios pass. The retained eight-hour B6 workload was
not rerun. The internal candidate is accepted under Owner Amendment 008A1; final-main
CI/package/update checks and the annotated tag still precede formal freeze.

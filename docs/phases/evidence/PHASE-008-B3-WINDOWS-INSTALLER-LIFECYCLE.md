# Phase 8 B3 Windows Installer Lifecycle Evidence

- Date: 2026-09-04 (Asia/Shanghai)
- Branch: `phase/p8-local-product-hardening`
- Exact implementation commit: `56df01f78a06659902156ef74fbb28560e357c6e`
- State: **PASS — B3 Gate complete for an unsigned internal candidate**

## Product and installer contract

P8-B3 freezes the internal candidate version at `0.4.0`. The Desktop package, Tauri
bundle and Rust crate use that version, while the private repository-root workspace
version remains non-product metadata. The MSI uses the stable UpgradeCode
`e793918b-37eb-5e2f-b866-5fc4aa3ac75a`, installs for the current user beneath
`%LOCALAPPDATA%\Programs\YORVA`, exposes Repair, and owns the program files, shortcuts,
uninstall entry and exact per-user login-start value.

The installer and Settings UI both state that uninstall removes program files,
shortcuts and login startup while preserving YORVA settings and management data,
encrypted backups, and all Hermes Runtime/Profile data. The MSI has no component or
custom action that targets those retained data roots.

## Exact candidate

| Field | Value |
| --- | --- |
| Source commit | `56df01f78a06659902156ef74fbb28560e357c6e` |
| Source worktree | clean at packaging time |
| File | `YORVA_0.4.0_x64_en-US.msi` |
| Size | 149,069,824 bytes |
| SHA-256 | `040FB08F82C75435BAE1E1D9D33D1A67FA0200EA6B0C16257739944E5249909E` |
| Authenticode | `NotSigned` |
| Classification | internal candidate only; not public-release ready |

The supported upgrade input was `YORVA_0.3.2_x64_en-US.msi`, 149,069,824 bytes,
SHA-256 `992C76E682363D7E64342526B5964B0A3F004504FF58BF5432CCC785B6F1D0E9`.
The missing production signature is recorded truthfully and remains a Phase 8 public
release blocker; no self-signed certificate or verification bypass was used.

## Disposable Windows lifecycle

The exact candidate ran in a fresh QCOW2 overlay based on Windows 11 Enterprise
Evaluation 25H2 x64 under WSL2/KVM/QEMU. The Microsoft installation ISO SHA-256 was
`7B4AC87391B659F7724229682B642256289A1C00504056249F0F12029157D3D2`.
The test payload was attached read-only, the reusable base disk was unchanged, and the
guest shut itself down after the result. The shared Windows host was **not rebooted**.

The lifecycle worker ran at Windows medium integrity and explicitly rejected a high
integrity token. A fresh guest state then completed:

1. current-user fresh install of `0.3.2`;
2. launch of the installed Desktop and daemon plus authoritative Runtime/Profile
   reconciliation;
3. running-process major upgrade to `0.4.0` through Windows Restart Manager;
4. exact-file deletion followed by MSI Repair and hash restoration;
5. uninstall with program files, both shortcuts, uninstall entry and login startup
   removed while three independent user-data sentinels remained;
6. reinstall of `0.4.0`, preserved-data verification, and authoritative Runtime/Profile
   reconciliation from the installed application.

The extracted structured record was:

```text
non-elevated-user-context            PASS
fresh-install                        PASS
running-process-upgrade              PASS
repair                               PASS
uninstall-preserves-data             PASS
reinstall-authoritative-recovery     PASS
```

The guest retained verbose MSI transcripts for Fresh (136,390 bytes), Upgrade (254,114
bytes), Repair (132,354 bytes), Uninstall (125,932 bytes), and Reinstall (137,192 bytes).
They stayed inside the disposable overlay and were inspected without copying user data
or secrets into the repository.

A minimal Hermes CLI fixture implemented only `hermes --version` and
`hermes profile list`. It prevented use of the developer's Hermes home or credentials
while exercising the real installed Desktop, daemon, discovery, SQLite reconcile,
process ownership and installer behavior. This is installer/recovery evidence, not a
claim about real model, Channel, Skill or MCP behavior.

## Static and negative inspection

The MSI inspector verifies exact embedded payload names, sizes and SHA-256 values,
version agreement, current-user layout, stable UpgradeCode, exposed Repair, exact login
startup ownership, and absence of user-data removal or dangerous deletion custom
actions. Negative fixtures passed for:

- per-machine installation;
- hidden Repair;
- unstable UpgradeCode;
- user-data removal;
- retained/unowned login startup;
- dangerous deletion custom action;
- missing, duplicated, renamed, wrong-size, wrong-hash and substituted resources.

Tauri 2.11.4 does not expose WiX linker arguments. Its generated resource components
produce ICE38/ICE64 for this verified current-user package, so the packaging script may
relink only the freshly generated WiX objects and suppress only those two checks. The
fallback now requires a current `main.wixobj` and a failure specifically at `light.exe`;
all other failures remain fatal. The real Fresh/Upgrade/Repair/Uninstall/Reinstall run is
the authoritative lifecycle check.

## Focused Gate

| Check | Result |
| --- | --- |
| Desktop Vitest | PASS — 24 files / 143 tests |
| Desktop typecheck and ESLint | PASS |
| Desktop Vite build | PASS |
| Rust format, 18 tests, Clippy `-D warnings`, check | PASS |
| Sidecar build | PASS |
| Non-bundled Tauri release build | PASS |
| MSI inspector positive and negative suites | PASS |
| Exact candidate package and administrative extraction | PASS |
| Disposable Windows lifecycle | PASS — all six checks above |


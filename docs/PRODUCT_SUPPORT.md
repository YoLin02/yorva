# YORVA Windows Product Support Contract

> Status: Phase 8 B0 approved baseline
> Effective: 2026-09-03
> Scope: first Windows local MVP

This contract freezes the common assumptions for P8 installation, migration, recovery,
update, diagnostics and release work. It does not by itself make a build public-ready.

## 1. Support matrix

| Area | Blocking P8 support | Notes |
| --- | --- | --- |
| OS | Windows 10 22H2 x64 and Windows 11 x64 | Windows is the only blocking target. Windows 10 must remain under Microsoft support or ESU; macOS/Linux observations are non-blocking. |
| Installer | Per-user MSI | Normal installation and use must not require YORVA to run permanently elevated. |
| WebView | Microsoft Edge WebView2 Evergreen | The official bootstrapper may be used; an offline absence must fail visibly. |
| CPU | x86-64 | ARM64 is outside formal P8 support. |
| Memory | 4 GiB minimum, 8 GiB recommended | Real Hermes workloads may require more. |
| Free disk | At least 2 GiB for install/update staging | Runtime, Profile, Backup and log growth require additional space. |
| YORVA database | Empty database and Phase 7 schema 016 | Arbitrary historical development schemas are not promised. |
| Hermes | Existing Runtime contract and exact qualification evidence | P8 does not widen compatibility or reopen managed Hermes Upgrade/Rollback. |

## 2. Product identity and version

- Stable product name: `YORVA`.
- Stable Desktop identifier: `com.yorva.desktop`.
- Phase 7 development identifier: `com.yorva.desktop.dev`, accepted only as a one-time
  migration source.
- Release versions use `MAJOR.MINOR.PATCH`; prerelease candidates may use a SemVer suffix.
- Desktop package, Tauri bundle, Rust crate, packaged `yorvad`, and MSI ProductVersion
  must describe the same candidate version and are checked by the build Gate.
- The private repository-root workspace version is not a product version.
- `0.3.2` is the P7/P8 development starting point. B3 freezes the P8 release candidate
  version; a UI-label-only version change is invalid.

P8-B1 implements and verifies the following one-time transition, and the stable
`com.yorva.desktop` identifier is now active:

1. inspects both directories before daemon startup or database mutation;
2. creates a verified protection copy and same-volume staging copy when the stable
   directory is empty and the legacy source is valid;
3. validates the database, fixed directories and marker before atomic publication;
4. preserves the legacy directory as rollback input;
5. stops with a typed recovery state when both locations contain unconfirmed data;
6. records source/target identifiers, time and non-secret integrity facts on success.

The legacy directory remains as rollback input after migration and is not automatically
deleted. When both roots contain unconfirmed data, Desktop stops with
`PRODUCT_DATA_IDENTITY_CONFLICT` instead of guessing how to merge them.

## 3. Windows directories and ownership

After stable-identifier activation, implementations resolve these fixed categories
through OS/Tauri APIs and never accept an arbitrary caller path.

| Location | Owner | Content | Default uninstall behavior |
| --- | --- | --- | --- |
| `%APPDATA%\com.yorva.desktop\` | YORVA | `yorva.db`, Desktop preferences, management state, Skills and logs | Preserve |
| `%APPDATA%\com.yorva.desktop\backups\` | YORVA | Verified encrypted Runtime backups | Preserve |
| `%APPDATA%\com.yorva.desktop\backup-staging\` | YORVA | Bounded temporary backup files | Clean after completion/failure; uninstall may remove temporary content |
| `%APPDATA%\com.yorva.desktop\update-staging\` | YORVA | Downloaded YORVA installer awaiting verification | Bound and clean after update completion/failure |
| User-selected Save As destination | User | Sanitized diagnostic bundle | Never remove during uninstall |
| `%LOCALAPPDATA%\hermes\` and Hermes Profiles | Hermes/user | Runtime, Profiles, sessions and Runtime-native state | YORVA uninstall must not remove |

B1, B4 and B5 freeze exact retention limits for migration protection, update staging and
diagnostic export. YORVA never treats `%TEMP%`, an arbitrary user folder or a Hermes root
as its cleanup root.

## 4. Installer lifecycle data policy

- Fresh Install creates only program files and required YORVA data directories.
- Upgrade/Repair may replace program files but never overwrite or delete the user
  database, backups, Hermes Runtime or Profiles.
- Uninstall removes program files, shortcuts, login-start entry and YORVA-owned temporary
  staging by default. It preserves YORVA user data, encrypted backups and all Hermes data.
- UI and support documentation explicitly explain preservation. P8 has no silent
  “delete all data” option.
- Reinstall discovers preserved data and reports recovery only after migration and
  authoritative reconciliation succeed.

## 5. Update and release trust

- P8 uses complete installer packages from a fixed YORVA release source; it does not add
  delta patches or a background updater service.
- Metadata fixes version, package length, SHA-256, signing state and allowed download URL.
- Origin, length, SHA-256, version and Windows signature policy are checked before launch.
- The Owner confirmed that Windows code-signing material is not currently available.
  P8 may therefore produce an internal candidate, but cannot claim public-release
  readiness. Self-signing or skipped verification cannot satisfy the Gate.
- Once production signing material exists, the exact candidate MSI must receive signing,
  provenance, tamper-rejection and installation evidence.

## 6. Privacy and support

- P8 collects no telemetry and uploads no usage, crash, Runtime or diagnostic data.
- Diagnostic export is local and user-initiated, with a fixed sanitized projection.
- Logs and bundles exclude API keys, tokens, Channel/MCP credentials, QR/pairing values,
  cookies, authorization data, environment values, the raw database and arbitrary files.
- Stability verification uses three Hermes Instances for at least four hours and targets
  eight hours for the final candidate.

## 7. Release classification

| Classification | Required state |
| --- | --- |
| Development build | Local development only; no installer, migration or update claim. |
| Internal candidate | B0–B6 behavior and tests pass, but production Windows signing is unavailable. |
| Public-release ready | B0–B6, independent audit, exact-candidate CI, real MSI lifecycle, production signing and Owner Gate all pass. |

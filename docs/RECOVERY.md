# YORVA Windows Recovery Guide

> Applies to the Phase 8 Windows local MVP and the support contract in
> `PRODUCT_SUPPORT.md`.

This guide starts with recoverable process-level actions. Restarting Windows is not the
default response to a failed Runtime check, a stopped Hermes Profile, or a daemon
disconnect. Never delete, merge, or edit YORVA/Hermes data manually to make an error
disappear.

## 1. First response

1. Keep the displayed stable error code and completion time.
2. If YORVA is still responsive, open **Settings → Diagnostics and Support → Export
   diagnostics** and save the sanitized bundle.
3. Close YORVA, wait for its Desktop and `yorvad` processes to exit, then reopen it.
   Desktop owns the daemon and reconnects to a replacement authenticated session; users
   do not need to launch `yorvad` themselves.
4. Refresh Runtime inventory. A stopped managed Hermes Profile can be started from its
   YORVA Instance controls; a Runtime detection timeout is not proof that data is lost.
5. If the same stable failure remains, use the matching section below. Preserve the data
   roots and diagnostic bundle before repair or reinstall.

## 2. Product data conflict

`PRODUCT_DATA_IDENTITY_CONFLICT` means both the legacy
`%APPDATA%\com.yorva.desktop.dev\` root and the stable
`%APPDATA%\com.yorva.desktop\` root contain data that YORVA cannot safely merge.

- Close YORVA and leave both directories unchanged.
- Do not copy one root over the other and do not delete either database.
- Confirm that the current user can read both roots and that the target volume has free
  space.
- Retain both roots and the diagnostics bundle for support-assisted selection. Automatic
  startup remains blocked until one authoritative root is deliberately resolved.

## 3. Database migration recovery required

`DATABASE_MIGRATION_RECOVERY_REQUIRED` means the recorded protection evidence is missing,
malformed, or does not match the protected database.

- Close YORVA and preserve the complete stable product-data root, including the database,
  migration state, and protection files.
- Do not run manual SQL, rename the protected database, or remove the recovery state.
- Reopening the same version is safe to try once. If the state remains, stop and retain
  the directory for a support-assisted recovery; YORVA intentionally refuses to guess.

A recovered migration may report `DATABASE_MIGRATION_FAILED_RECOVERED`. In that case the
supported source database has been restored; keep the evidence and retry only after the
underlying disk-space or permission problem is corrected.

## 4. Runtime or daemon unavailable

- Use **Retry/Refresh** in YORVA first. Desktop permits one bounded automatic daemon
  replacement and then exposes failure instead of creating a respawn loop.
- If the Hermes Runtime exists but a Profile is stopped, start that Instance from YORVA.
- Check free disk space and current-user access to the YORVA and Hermes-owned directories.
- Repeated timeout or reconnect failure should be captured in a diagnostic bundle. Do
  not start a second daemon manually and do not run YORVA permanently as Administrator.

## 5. Interrupted update

- Reopen the installed YORVA version. If a download was interrupted by process exit,
  it becomes a retryable download failure and keeps the candidate; use Download and
  verify again. Only updater-owned partial/package staging files are cleaned.
- After installation, the updater checks the running daemon version and fresh Runtime/
  Instance recovery. A usable daemon with failed readback reports UPDATE_POSTCHECK_FAILED;
  use the management/diagnostic pages to investigate rather than assuming the update passed.
- Never execute a `.partial` file or an installer copied from update staging.
- If YORVA reports metadata, integrity, signature, installer, or postcheck failure, retain
  the stable error code and diagnostic bundle. Retry only through the YORVA update page.
- An unsigned internal candidate is not public-release ready and must not be promoted by
  bypassing signature checks.

## 6. Runtime Backup and Restore

- Managed encrypted Runtime backups remain under
  `%APPDATA%\com.yorva.desktop\backups\`; users do not choose the managed destination.
- Before Backup or Restore, stop all Hermes Instances when the UI requests it and ensure
  sufficient free disk space.
- A failed Backup must not be treated as a protection point. Use only a listed,
  successfully verified backup.
- If Restore reports that manual recovery is required, do not delete its transaction or
  replace Hermes files by hand. Preserve both the YORVA product-data root and Hermes root
  for support-assisted recovery.

## 7. Repair, reinstall, and uninstall

Repair, upgrade, uninstall, and reinstall preserve YORVA user data, encrypted backups,
and Hermes Runtime/Profile data by default. Reinstall is therefore a program-file repair,
not a way to erase a data conflict. Do not remove the preserved roots unless a future
explicit data-removal workflow says exactly what will be deleted.

## 8. What to retain for support

- the sanitized diagnostic ZIP;
- the exact YORVA version and stable error code;
- the time the failure occurred;
- whether the failure followed install, migration, update, Backup, Restore, or a daemon
  reconnect.

Do not send API keys, Channel/MCP credentials, QR/pairing values, raw databases, raw
Hermes configuration, or entire data directories through ordinary support channels.

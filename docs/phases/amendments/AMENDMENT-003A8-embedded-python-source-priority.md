# YORVA Phase 3 Amendment 003A8 — Embedded Python and Source Priority

> Status: ACCEPTED FOR IMPLEMENTATION
> Date: 2026-08-21
> Owner: Repository owner
> Phase: 3 — Hermes Installation
> Classification: post-freeze product correction
> Related: `ADR-0011`, `AMENDMENT-003A6`, `AMENDMENT-003A7`

## Goal

Make the Windows MSI carry the exact CPython interpreter required by Hermes and let
the user choose whether verified packaged artifacts or verified online artifacts are
attempted first.

## Authorized behavior

- Package CPython `3.11.15`, python-build-standalone release `20260728`, Windows x64.
- Pin the archive to exact size `25676832` and SHA-256
  `64a804111830c5329bfc5a4d95d6cbcbb377caa2c02195101edf85d15fc53099`.
- Materialize the selected archive and one single-entry pinned metadata document into
  an Operation-private local uv mirror. Pass them through `UV_PYTHON_INSTALL_MIRROR`
  and `UV_PYTHON_DOWNLOADS_JSON_URL` after inherited `UV_*` values are removed, so a
  later uv binary still resolves the exact reviewed patch/build.
- Extend the closed download-source settings with `artifactPreference` and
  `pythonArchiveUrl`.
- Permit only `bundled-first` and `online-first` preferences.
- On transport failure, try the alternate available source. On size or digest
  mismatch, fail closed without fallback.
- Keep Python package resolution (`pythonIndexUrl`) separate from interpreter
  acquisition (`pythonArchiveUrl`).

## Desktop behavior

Settings > Advanced uses the same flat section layout as General. It contains one
two-option source-priority control, online artifact addresses, dependency registries,
and existing save/reset actions. The previous green mainland-China badge is removed.

## Non-goals

- no Python version, hash, executable, local path or command override;
- no generic interpreter manager;
- no embedded Python dependency wheel cache;
- no weakening of the final-generation executable validation in 003A6.

## Acceptance

- MSI inspection proves the exact Python archive is present once;
- uv receives only the verified Operation-private `file://` mirror;
- legacy five-field settings load with bundled-first and the compiled Python archive
  URL defaults;
- both priority values round-trip through the authenticated typed API and Desktop;
- UI, Go, Rust and packaging checks cover the new fields and bootstrap path.

This owner-authorized correction does not unlock a later phase.

# YORVA Phase 3 Amendment 003A7 — Configurable Download Sources

> Status: ACCEPTED FOR IMPLEMENTATION
> Date: 2026-08-21
> Owner: Repository owner
> Phase: 3 — Hermes Installation
> Classification: post-freeze product correction
> Related: `ADR-0010`, `AMENDMENT-003A1`, `AMENDMENT-003A2`, `AMENDMENT-003A3`

## 1. Goal

Make the packaged Windows install path fast by default in mainland China while
allowing a user to change only the bounded download transports from Settings >
Advanced.

## 2. Authorized behavior

The authenticated local API owns one complete Hermes source configuration:

```text
hermesArchiveUrl
nodeArchiveUrl
npmArchiveUrl
pythonIndexUrl
npmRegistryUrl
```

The Desktop can read, replace and reset it. `app_settings` persists the non-secret
JSON value. Reset removes the override and returns the compiled China defaults.

Packaged artifact order is:

```text
verified MSI artifact
  -> only when the artifact is absent
  -> configured HTTPS fallback URL
  -> exact compiled size/SHA-256 verification
```

A present but corrupt MSI artifact fails integrity checks and is never hidden by an
online fallback. A non-packaged developer build may use the configured fallback.

## 3. Security boundary

- HTTPS only; absolute host required.
- User information, query strings and fragments are rejected so credentials cannot be
  persisted or leaked through diagnostics.
- No caller supplies a version, digest, archive path, command or environment key.
- Hermes/Node/npm archive identity remains fixed by compiled size and SHA-256.
- Official `uv.lock` and `package-lock.json` are not edited.
- Inherited Python, uv, pip, npm, provider and Hermes secrets remain stripped.
- Active Operations use a start-time snapshot; later saves cannot mutate in-flight
  process environments or downloads.

## 4. Desktop behavior

The Advanced tab contains a Hermes-only section with:

- a clear "bundled artifacts first" explanation;
- separate artifact fallback and dependency registry groups;
- validation and typed save errors;
- explicit Save and Restore China defaults actions;
- English and Simplified Chinese copy;
- responsive layout for the smaller default window.

## 5. Window correction

The default Tauri window is reduced from the 2K-oriented size and centered. Minimum
dimensions remain large enough for the sidebar and management forms, while existing
responsive CSS collapses multi-column content on narrower displays.

## 6. Non-goals

- no private/credentialed registry;
- no arbitrary local path or file picker;
- no version, checksum or executable override;
- no generic multi-Runtime source plugin;
- no lockfile rewrite, dependency cache or full offline claim;
- no change to the supported Hermes/Node/npm versions.

## 7. Acceptance matrix

- defaults are returned from an empty database;
- save survives a database reopen and reset restores defaults;
- unknown fields, non-HTTPS URLs and credential-bearing URLs are rejected;
- handler authorization, method and closed-body behavior are tested;
- MSI source is selected before any online request;
- missing MSI source uses the configured fallback and still verifies the existing pin;
- corrupt MSI source fails closed without fallback;
- Python/npm settings appear once in the child environment and hostile inherited
  settings/secrets remain absent;
- Desktop loads, edits, saves and resets the fields in both locales;
- OpenAPI generation, Go tests/vet/build, Desktop tests/typecheck/lint/build and Rust
  checks pass.

## 8. Gate

This amendment is authorized by the repository owner's active request. It is an
emergency correction to frozen Phase 3 behavior and does not unlock Phase 7. Phase 6
still requires its independent audit and owner-authenticated Windows/channel smoke.

# ADR-0010: Configurable Hermes Download Sources

- Status: Accepted
- Date: 2026-08-21
- Owner approval: 2026-08-21 (active task)
- Related: Phase 3 amendments 003A1, 003A2, 003A3 and 003A7
- Supersedes: the fixed-only source policy in 003A1/003A2/003A3, but not their pins or integrity controls

## Context

The frozen Phase 3 installer was designed for a China demo build. It compiles one
PyPI index and one npm registry, downloads the Hermes archive from GitHub first, and
uses the MSI copy only after a transport failure. The Desktop Advanced tab is empty.

Users need to select reachable mirrors without rebuilding Yorva. This changes a
previous trust-boundary decision: the older amendments explicitly rejected
user-supplied registries and source URLs.

## Decision

Yorva stores one non-secret, Hermes-namespaced download-source configuration in
`app_settings`. It is managed through an authenticated typed local API and the
Desktop Advanced tab. The default profile is optimized for mainland China:

- MSI-bundled Hermes, Node.js and npm artifacts are always preferred;
- the Hermes archive falls back to the pinned official GitHub archive;
- the Node.js archive falls back to the reviewed npmmirror binary mirror;
- the npm archive and dependency registry use npmmirror;
- Python/uv/pip use the Tsinghua TUNA PyPI mirror.

Every saved URL must be absolute HTTPS with a host and without user information,
query parameters or fragments. Credentialed registries are not supported. Settings
are snapshotted once at the start of an Install or prerequisite Operation; changing
them affects only later Operations.

Artifact URLs change transport only. Hermes, Node.js and npm archive bytes must still
match the existing compiled size and SHA-256 pins before extraction. Lockfiles remain
unchanged. Python and npm registry settings are passed only through the existing
allowlisted child environment after inherited registry and credential variables are
removed.

## Alternatives considered

### Keep compile-time mirrors only

Rejected. It requires a new Yorva build whenever a mirror becomes slow, blocked or
unavailable and leaves the Advanced settings surface unable to solve the user's
installation problem.

### Accept arbitrary paths, commands, credentials or versions

Rejected. It would create a generic installer/file execution surface and weaken the
existing source pins.

### Make the online mirror primary

Rejected for packaged builds. The signed MSI already contains the exact reviewed
artifacts, so local verified bytes are faster and remove unnecessary network failure.

## Consequences

- Download availability can be repaired without rebuilding Yorva.
- The daemon, not React or Tauri, remains the source of truth for installation policy.
- Custom artifact mirrors cannot change installed bytes without defeating SHA-256.
- Registry operators remain part of transport availability; upstream lockfile hashes
  remain the integrity authority.
- Invalid or corrupted saved settings fail closed and do not silently fall back.

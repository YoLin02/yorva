# YORVA Phase 3 Amendment 003A5 — Pinned Node Archive Member Limit

> Status: ACCEPTED / FROZEN
> Date: 2026-08-20
> Owner: Repository owner
> Phase: 3 — Hermes Installation
> Classification: blocking correctness defect in a frozen baseline; security-sensitive extraction-limit correction
> Related: `AMENDMENT-003A3-managed-node-prerequisites.md`

## 1. Problem

The pinned, size- and SHA-256-verified Node `v22.23.1` archive contains an
86,989,128-byte `node.exe`. The runtime ZIP extractor limited every member to
32 MiB, so the approved official archive deterministically failed with
`RUNTIME_HERMES_NODE_ARCHIVE_INTEGRITY_FAILED` before any managed Node files
were published.

The archive bytes are not corrupt and the failure is unrelated to NVM or the
host `PATH`.

## 2. Decision

Raise only the Node/npm archive per-member extraction limit from 32 MiB to
96 MiB. This is the smallest limit that leaves practical headroom above the
pinned executable while remaining below the existing 256 MiB total Node ZIP
expansion limit.

The following controls remain unchanged:

- exact compiled archive byte size and SHA-256;
- 8,000-entry Node ZIP limit;
- 256 MiB total Node ZIP expansion limit;
- traversal, absolute path, alternate stream, symlink and reparse-point rejection;
- operation-private extraction and atomic managed-tree publication;
- exact managed Node `22.23.1` and npm `12.0.2` postconditions.

## 3. Scope boundary

This correction does not adopt or execute Node from NVM, `PATH`, a package
manager or a user-supplied path. External Node discovery and reuse would change
the accepted managed-prerequisite trust boundary and requires a separate
owner-approved amendment and relevant security re-audit.

## 4. Verification

- Compile-time regression coverage must prove the member limit accepts the
  pinned `node.exe` size and preserves the other extraction limits.
- The real pinned Node archive must pass the production digest and extraction
  functions.
- Relevant Hermes adapter tests, Go tests and Desktop build checks must pass.

## 5. Gate

Accepted by `AUDIT-005A1-post-freeze-corrections.md` on 2026-08-20. The
correction is included in the revised Phase 5 entry baseline.

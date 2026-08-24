# ADR-0012: Hermes Patch-Compatible Development Policy

- Status: Accepted
- Date: 2026-08-24
- Owner approval: Phase 6.5 developer direction
- Supersedes: the exact-version support decision in Amendment 002A4

## Context

YORVA previously classified only Hermes `0.20.2` as supported. The developer updated
the local official Hermes checkout to `0.20.5`. Its official `--version` surface now
performs an update-status check and can exceed the former ten-second discovery limit.
Exact equality also caused discovery and every model, lifecycle and Channel capability
to reject a compatible patch update.

The Runtime compatibility policy and the packaged source snapshot are different
concerns. Runtime discovery must not be tied to one patch number. A distributable build
must still identify and verify the exact source bytes it carries.

## Decision

- Stable Hermes releases in `>=0.20.2 <0.21.0` use one shared compatibility policy for
  discovery, lifecycle, models, credentials and Channels.
- Prereleases, older releases and another minor/major line remain visible but
  unsupported.
- The bounded discovery deadline is thirty seconds because the official `0.20.5`
  version command may check update status.
- The current packaged snapshot is official Hermes `0.20.5` at commit
  `a0ca7c19204e514f9590ce3b812e029b315ab9e9`.
- Packaging continues to verify the exact archive, installer script, license, size and
  SHA-256. Advancing the packaged snapshot updates those build inputs together; it does
  not make packaging depend on an unreviewed moving branch.
- YORVA does not fork or patch Hermes source. Unknown output and changed contracts
  continue to fail closed in their owning adapters.

## Consequences

Hermes patch updates no longer make the installation unusable merely because the patch
number changed. Future `0.20.x` changes can still expose a real contract incompatibility;
the existing strict parsers and postconditions report that failure instead of inventing
success. Moving to `0.21.x` or later requires an explicit compatibility review.

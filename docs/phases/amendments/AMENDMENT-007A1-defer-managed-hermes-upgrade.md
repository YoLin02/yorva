# YORVA Phase 7 Amendment 007A1 — Defer Managed Hermes Upgrade/Rollback

> Status: APPROVED
> Date: 2026-08-27
> Owner: Repository owner / lead developer
> Related: Phase 7, ADR-0015, AUDIT-007

## Context

The Phase 7 freeze audit found that the production daemon does not bind an executable
Hermes generation Upgrader or Rollbacker. The existing read-only plan is truthful, but a
fixed-candidate Upgrade/Rollback still requires separate qualification for protection
points, data compatibility, activation, post-check and recovery.

The Owner has chosen to finish the MVP Runtime-management closure without making that
destructive version transition part of the Phase 7 freeze candidate.

## Authorized scope change

- Defer executable managed Hermes Upgrade and Rollback to a later, separately approved
  phase or amendment.
- Keep the Runtime Upgrade Plan read-only and truthful.
- Keep production `Upgrade` and `Rollback` capability flags false. The Desktop and API
  must not imply that an executable upgrade is available.
- Keep ADR-0015 as the governing safety design for any future implementation; this
  amendment changes scheduling, not its safety requirements.
- Do not weaken or bypass the qualification required before a future generation switch.

## Revised Phase 7 acceptance

Phase 7 no longer requires an executable fixed-candidate Hermes Upgrade/Rollback. It does
require:

- a truthful plan/readiness view with no false success or mutation affordance;
- encrypted Runtime Backup/Restore with an automated disposable Restore lifecycle;
- exact-candidate CI evidence, including Go race and Windows native checks;
- the remaining approved Runtime management surfaces and authoritative read-back.

The Upgrade/Rollback finding in `AUDIT-007` is therefore dispositioned as an
Owner-approved scope deferral, not as an implementation PASS. Restore and exact-candidate
CI findings remain blocking until re-audit.

## Future re-entry criteria

Any later implementation must be explicitly authorized and must use immutable packaged
input, a new managed generation, a protection point, final-path validation, atomic
activation, authoritative post-check and qualified rollback. It must not mutate the active
sealed generation in place.

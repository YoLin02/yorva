# Phase 7 Wave 9 — B6/B8 Integrated Focused Test

> Initial result: **FAIL — P0 0 / P1 0 / P2 3**
> Date: 2026-08-25
> Scope: unconnected B6 snapshot/publication/index/SecretStore foundations and B8 read-only Upgrade Plan

This is a focused development Gate, not the Phase 7 B10 audit and not authority to
enable Backup, Restore, Upgrade or Rollback mutation.

## Initial FAIL

The independent reviewer found three local P2 defects while every affected mutation
capability remained false and no mutation route was registered:

1. Windows source exclusions and SQLite sidecar suffixes were compared
   case-sensitively. A case-variant managed-generation directory could enter a snapshot,
   and an uppercase SQLite WAL name would not fail closed.
2. The Windows DPAPI SecretStore rejected a reparse root object but did not validate the
   complete ancestor chain from the volume root, allowing an ancestor junction to
   redirect the authoritative storage root.
3. Windows backup publication performed fallible attribute cleanup after the atomic
   move. A post-move error could report failure with an already-published file and empty
   result, leaving retry truth ambiguous.

The same focused run passed the relevant Go tests and vet, module verification/tidy
drift, Desktop management tests/typecheck/lint, OpenAPI lint/generated drift and diff
check. Those green checks did not override the three review findings.

## Remediation pending R1

The implementation was changed to fold Windows policy comparisons, validate every
SecretStore directory ancestor, and converge post-move publication by matching the
published file identity before final checksum/readback. Regression tests cover each
case. A fresh R1 review is required before this Wave can pass.

## R1 FAIL

Fresh-context R1 accepted the first two remediations but found one remaining P2. After
the atomic move, a failure in directory sync or final readback still returned empty
metadata while the destination existed. The first regression covered only an error
reported from inside the publisher immediately after the move.

The publication contract now has a closed `PUBLISHED_RECONCILE_REQUIRED` result for any
failure after the atomic publish boundary. It carries only the already-authenticated
ciphertext size/checksum, encryption mode and structural metadata, plus a stable error;
it carries no destination path or credential. Callers must reconcile that exact
artifact and cannot treat the operation as unpublished or retry blindly. Normal final
readback returns `PUBLISHED_VERIFIED`. A second regression injects the post-publish
directory-sync failure. Fresh R2 review is required.

## R2 PASS

Fresh-context R2 result: **PASS — P0 0 / P1 0 / P2 0**.

R2 verified the preserved FAIL history, uncached Windows regressions without skips,
case-folded exclusions/SQLite sidecars, SecretStore ancestor and post-open replacement
rejection, empty pre-publication failures, safe nonempty post-publication reconciliation
metadata, and the normal verified terminal state. Focused package tests, targeted tests,
Go vet and diff check passed. No production caller currently discards or conflates the
reconciliation result.

This R2 PASS accepts only the Wave 9 foundations. B6 product wiring, B7 Restore, B8
mutation, full CI, Windows destructive smoke and the B10 independent audit remain
separate Gates.

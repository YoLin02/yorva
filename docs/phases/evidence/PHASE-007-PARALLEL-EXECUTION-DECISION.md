# Phase 7 Parallel Execution Decision

## Owner decision

**APPROVED — 2026-08-25**

The Owner replaced a purely serial Batch execution model with dependency-driven parallel
development. This decision changes orchestration, not the Phase 7 product scope or its
security, correctness, data-integrity and freeze acceptance standards.

## Execution rules

- One development agent owns one feature lane and one non-overlapping file/module set.
- B2 shared Runtime-neutral contracts are the integration prerequisite.
- Health/Logs/Security, Skills, MCP, Backup/Restore and Upgrade may develop concurrently
  only where their real contracts and security decisions are already known.
- Backup Create precedes destructive Restore. Upgrade depends on a verified protection
  point and exact from/to user-data compatibility.
- B1 NO-GO evidence closes the rejected direct Hermes route; it does not freeze unrelated
  work. A lane may use only a separately accepted closed alternative.
- Every lane runs focused tests. Unchanged whole-project CI is not repeated per lane.
- One fresh-context independent audit is run against the complete immutable B10
  candidate. Audit-driven fixes receive a fresh re-audit and historical FAIL is retained.

## Stop threshold

Ordinary bounded defects remain in their owning lane and are fixed without stopping the
other developers. A surface stops immediately for secret disclosure, unauthenticated or
arbitrary execution, cross-Profile mutation, unsafe archive/data destruction, active
generation mutation, false destructive success, or another Phase 7 mandatory stop
condition. Standards are not weakened to keep a lane moving.

## Worktree protection

All Phase 7 work remains in `D:\workcode\myproject-phase65-freeze` on
`codex/phase7-hermes-runtime-management`. The original developer worktree
`D:\workcode\myproject` remains untouched: no reset, stash, deletion, overwrite or
unrelated commit.

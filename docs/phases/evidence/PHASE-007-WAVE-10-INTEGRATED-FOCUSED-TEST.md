# Phase 7 Wave 10 — Integrated Focused Test

> Final result: **PASS AFTER R1 — P0 0 / P1 0 / P2 0**
> Date: 2026-08-25
> Scope: B3/B4 authenticated reads, B5 qualification harness, B6 manifest, B8 read-only plan

This is a focused development Gate, not the B10 independent audit.

## Preserved initial failures

The first integrated Desktop run failed one stale shared assertion after the concurrent
Upgrade-plan section increased unavailable sections from three to four: 23/24 files and
115/116 tests passed. The assertion was updated, and the R1 run passed 24/24 files and
116/116 tests.

An integrated Go run made while the B6 BackupRead contract was actively being edited
also failed with two transitional Runtime tests and a `BackupReader` compile mismatch.
This was an observable concurrent working-tree state, not relabeled as a product PASS.
After the owning lane stabilized the contract, the exact focused Go gate passed.

## Independent P2 and remediation

The reviewer found one P2: exact Hermes configuration could define API-server or
multiplex values in more than one supported YAML location, and the adapter silently
selected/merged them. That contradicted the documented ambiguity-fail-closed boundary
and could target the wrong loopback listener after external Profile edits.

The adapter now rejects root/gateway duplicate multiplex or allowlist values, more than
one supported API-server block, and duplicate block/`extra` key, host or port values.
Three regression fixtures cover those cases. R1 accepted the remediation.

## R1 result

Fresh integrated result: **PASS — P0 0 / P1 0 / P2 0**.

- focused Go tests and vet passed for Hermes, management health, Skills, MCP, backup,
  Runtime contracts, application and HTTP packages;
- Desktop passed 24/24 files and 116/116 tests, plus typecheck and lint;
- OpenAPI lint and generated-schema comparison passed;
- `go mod tidy -diff` and `git diff --check` passed (line-ending notices only);
- the B5 registry remained empty and unwired;
- Backup/Restore and Upgrade/Rollback mutation capabilities remained false.

No full CI, MSI, destructive smoke or B10 audit ran in this Gate.

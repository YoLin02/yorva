# YORVA Phase 2 Amendment 002A1 — Closed Loop and Baseline Re-Freeze Control

> Role: Phase 2 revision closed-loop / baseline re-freeze control record  
> Date: 2026-08-15  
> Controller: Phase 2 002A1 control agent  
> Owner: Repository owner  
> Current loop state: `COMPLETE / FROZEN`  
> Owner desktop manual test: PASS  
> Amendment gate: PASS  
> Freeze requested: yes — replacement tag `phase-002-hermes-discovery-baseline-r1`

## 1. Authority

This document is the control contract for the post-freeze amendment
`AMENDMENT-002A1-hermes-windows-command-resolution.md`.

It does not replace:

- `docs/PHASE_GOVERNANCE.md`
- `docs/AUDIT_STANDARD.md`
- the frozen Phase 2 spec
- the amendment itself
- historical audit reports

If this document conflicts with a higher-level contract, the higher-level
contract wins.

## 2. Immutable historical baseline

The original Phase 2 freeze remains historical evidence and is not rewritten:

```text
tag:    phase-002-hermes-discovery-baseline
peeled: a67de04e900bc3ddce99cd76501eec13586082ed
audit:  AUDIT-002R1-hermes-discovery.md — PASS
impl:   b875b2c496269d3bf91097395cae5fb53dc63350
merge:  0b31ff72f34f743b285eac4bc0bd4c8ced5096c1
```

`AUDIT-002` FAIL and `AUDIT-002R1` PASS remain preserved. This loop may add
`AUDIT-002A1-*` reports. It must not edit those historical reports.

## 3. Current snapshot

```text
branch:  main
implementation: dbcb54da4bc4bffcff51888426848246a1900ea6
audit report:   99564fb147d8fa06939d3bc8331d778ceb7e6d1f
merge:          f786b9d73768de38875f0a57fa7d57c75168ed85
ci:             GitHub Actions run 31817886874 PASS
scope:          Windows official-root Python fallback + 0.20.x compatibility
```

## 4. Closed loop

```text
Amendment classification          DONE
Narrow implementation             DONE
Owner desktop manual test         PASS
Independent local verification    PASS except Go race (environment blocked)
Exact-commit GitHub Actions       PASS — run 31817886874 on dbcb54d
Independent AUDIT-002A1           PASS
Gate Decision                     PASS
Merge to main                     DONE — f786b9d73768de38875f0a57fa7d57c75168ed85
Replacement baseline freeze       IN PROGRESS
Phase 3 implementation            PROHIBITED until r1 tag is pushed
```

## 5. Hard stops

Until the Owner accepts an independent `AUDIT-002A1` `PASS` and the exact
audited commit later passes GitHub Actions:

- do not merge to `main`;
- do not move, delete or retarget `phase-002-hermes-discovery-baseline`;
- do not create a replacement baseline tag;
- do not begin Phase 3 implementation;
- do not expand scope beyond the amendment;
- do not weaken tests, smoke invariants or security bounds to obtain a green
  gate.

Local `go test -race ./...` remains environment-blocked on this host because
cgo cannot find `gcc`. That limitation is not a pass. Exact-commit CI remains
mandatory.

## 6. Independent audit authorization

Authorized only after the exact implementation commit's GitHub Actions matrix PASSes, including Go race:

```text
report:  docs/phases/audits/AUDIT-002A1-hermes-discovery.md
target:  exact implementation commit on fix/phase2-hermes-command-resolution
base:    phase-002-hermes-discovery-baseline^{} = a67de04e900bc3ddce99cd76501eec13586082ed
```

The auditor:

- starts from a fresh context;
- reads governing documents and the amendment first;
- reviews the actual diff and source, not the implementation summary;
- independently reruns required verification;
- may write only the audit report;
- must not modify implementation, contracts, tests, commits, tags or remotes.

Required dimensions at full depth:

```text
Scope
Correctness
Architecture
Security
Concurrency and Lifecycle
Protocol and Compatibility
Testing and Verification
Documentation
Operations / Diagnostics
```

May be `N/A` only with evidence:

```text
Data and Persistence
Dependencies / Supply Chain
```

The amendment's local verification table is an input, not a substitute for
independent rerun.

## 7. Gate effects

### FAIL

Re-enter the fix loop. Feature work stays frozen except for accepted findings.
Historical reports stay in place. A later re-audit uses `AUDIT-002A1R1-...` or
a dated addendum.

### PASS or PASS WITH CONDITIONS

The control agent may then, and only then:

1. keep the working tree limited to the audited amendment plus the audit
   report;
2. create one exact implementation commit on `fix/phase2-hermes-command-resolution`;
3. push that commit and require GitHub Actions on that SHA, including
   `go test -race ./...` and Windows native/Tauri;
4. after exact-commit CI `PASS`, request Owner merge to `main`;
5. after merge, create a replacement freeze commit and annotated tag
   `phase-002-hermes-discovery-baseline-r1` on `main`;
6. update Phase Spec, Roadmap, Development and Phase Governance acceptance
   records to name the replacement baseline while preserving the original tag.

The original `phase-002-hermes-discovery-baseline` tag is not moved.

`PASS WITH CONDITIONS` still requires Owner acceptance of each remaining
non-blocking finding before freeze.

## 8. Replacement baseline shape

If and when the loop completes:

```text
historical tag:  phase-002-hermes-discovery-baseline
                 → a67de04e900bc3ddce99cd76501eec13586082ed
replacement tag: phase-002-hermes-discovery-baseline-r1
                 → freeze commit on main after 002A1 CI PASS and independent audit PASS
current Phase 2 baseline: the replacement tag
Phase 3 entry: replacement tag exists and is pushed
```

## 9. Out of this loop

- Hermes installation, PATH mutation, launcher repair;
- Profile, Instance, process or gateway lifecycle;
- persistence of discovery results;
- generic Python or process-execution APIs;
- any Phase 3 design or implementation.

## 10. Decision log

| Time | Decision | Effect |
|---|---|---|
| 2026-08-14 | Control agent accepted 002A1 loop ownership | Feature work frozen except audit-driven fixes |
| 2026-08-14 | Premature working-tree audit cancelled | Independent audit waits for exact-commit CI |
| 2026-08-14 | Owner confirmed new EXE manual test PASS | Local verification matrix authorized |
| 2026-08-14 | Local verification PASS except Go race | Implementation commit authorized; race remains a CI gate |
| 2026-08-14 | Implementation commit `dbcb54d` pushed | Exact-commit CI run `31817886874` required |
| 2026-08-14 | Exact-commit CI PASS | Independent `AUDIT-002A1` authorized |
| 2026-08-15 | Independent `AUDIT-002A1` PASS | Merge and replacement freeze authorized |
| 2026-08-15 | Merged to `main` as `f786b9d` | Historical tag unchanged; r1 freeze in progress |

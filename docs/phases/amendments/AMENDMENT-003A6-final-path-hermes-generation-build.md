# YORVA Phase 3 Amendment 003A6 — Final-Path Hermes Generation Build

> Status: ACCEPTED / FROZEN
> Date: 2026-08-21
> Owner: Repository owner
> Phase: 3 — Hermes Installation
> Classification: blocking correctness defect in a frozen baseline; installation transaction correction
> Related: `ADR-0006`, `ADR-0009`, `AMENDMENT-003A4-generation-install-transaction.md`

## 1. Problem

The frozen generation installer constructs Hermes 0.20.2 under
`staging/txn_*` and then moves the complete tree to `generations/gen_*`.
Windows uv console launchers and editable project metadata can retain the absolute
staging path. The moved generation can therefore pass structural and seal checks but
fail when launched from its final path with
`uv trampoline failed to canonicalize script path`.

This is a correctness defect because the transaction may be marked `COMMITTED` while
the selected runtime cannot execute.

## 2. Approved correction

For every new Hermes installation transaction:

1. allocate a new transaction id and generation id;
2. create `generations/gen_<id>` directly and bind it to the transaction with a
   `.yorva-candidate.json` ownership record;
3. run the pinned official build with that exact final path as `InstallDir`;
4. execute both final-path Hermes launchers with fixed `--version` argv;
5. require the exact supported Hermes version;
6. seal the unchanged final-path tree;
7. record `PUBLISHED` only after full seal verification, without moving the directory;
8. atomically switch `control/active.json` and then commit.

Directory presence is not activation. `active.json` remains the sole active-generation
pointer.

## 3. Validation gate

The previous final check of “required files exist” is insufficient. Seal must be
blocked unless the runtime can execute from the exact final generation path.

The functional probe must:

- invoke only `bin/hermes.exe --version` and
  `venv/Scripts/hermes.exe --version` from the candidate;
- use direct argv without a shell;
- inherit only the approved closed environment;
- use process-tree containment, bounded output and a bounded timeout;
- parse the returned version and require the pinned supported version;
- return stable safe installation error codes without exposing raw command output.

Structural path, regular-file, reparse-point, source-pin and manifest validation remain
mandatory and are not weakened.

## 4. Recovery and compatibility

- New interrupted candidates are never resumed. Retry uses new ids.
- A pre-correction transaction whose sealed runtime remains under `staging/txn_*` must
  not be moved or activated; recovery records an interrupted failure and preserves only
  lineage-proven evidence.
- A sealed final-path candidate with no active pointer may be safely observed as
  `PUBLISHED` and continue through activation.
- A partial final-path candidate must never be inferred active from its directory name,
  timestamp, PATH entry or latest Operation.
- The schema-1 `stagingRelativePath` field remains readable only for compatibility with
  existing transaction records.

## 5. Scope boundary

This correction does not change:

- the pinned Hermes source or supported version;
- public HTTP DTOs or Operation stage names;
- `active.json` compare-and-swap rules;
- official mutable `HERMES_HOME` data ownership;
- Desktop runtime lifecycle or Instance behavior;
- the rule that YORVA must not patch Hermes or import Hermes internals.

## 6. Verification

Required focused evidence:

- a regression fixture whose launcher succeeds only when built and run at the same
  final path;
- final-path launcher timeout, non-zero exit and wrong-version cases block Seal and
  leave the predecessor active generation untouched;
- publication performs no staging-to-generation rename;
- legacy sealed staging recovery never activates that tree;
- active pointer switching still occurs only after full final-path validation and Seal;
- targeted Go tests, vet for affected packages, Desktop typecheck/build, and one MSI
  package inspection pass.

## 7. Gate

Owner approved this correction on 2026-08-21. Focused Go tests, vet, and Desktop tests
passed. Local MSI package inspection passed on 2026-08-21
(`YORVA_0.3.1_x64_en-US.msi`). Status is `ACCEPTED / FROZEN`. This does not retag the
Phase 3 baseline.

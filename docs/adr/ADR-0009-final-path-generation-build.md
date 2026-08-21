# ADR-0009: Build Hermes Generations at Their Final Path

- Status: Accepted
- Date: 2026-08-21
- Owner approval: 2026-08-21
- Related: ADR-0006, Phase 3 `AMENDMENT-003A6`
- Supersedes: ADR-0006 only where it requires building under `staging/txn_*` and publishing by directory rename

## Context

The accepted Phase 3 transaction built Hermes in `staging/txn_*`, sealed that tree,
and renamed it to `generations/gen_*`. Hermes 0.20.2 uses uv-generated Windows
console launchers and editable project metadata. Those artifacts may contain the
absolute virtual-environment or project path that existed when they were generated.
Moving the completed tree can therefore leave a structurally complete and correctly
hashed generation whose launchers still target the deleted staging path.

A file-existence and source-version check cannot detect this failure. The generation
may reach `COMMITTED` even though `bin/hermes.exe` fails with a uv trampoline path
error from its final location.

## Decision

Allocate the transaction and generation identifiers first, then create the candidate
directly at `generations/gen_<id>`. The official installer receives that final
generation path as `InstallDir`. The candidate remains inactive because
`control/active.json`, not directory presence, is the sole activation authority.

The transaction order remains:

```text
CREATED
  -> BUILDING at generations/gen_<id>
  -> final-path functional validation
  -> SEALED
  -> PUBLISHED (sealed candidate accepted; no directory move)
  -> ACTIVATING (atomic active.json compare-and-swap)
  -> COMMITTED
```

Before external build work begins, YORVA writes a bounded
`.yorva-candidate.json` ownership record inside the candidate directory. It binds the
directory to the exact transaction and generation identifiers. The record is included
in the seal manifest and is not an activation source.

Seal is blocked unless both documented final-path Hermes launchers can execute
`--version` from the candidate directory and report the pinned supported version.
This validation exercises the uv launcher and installed Python package rather than
checking only that files exist. It uses fixed argv, a bounded timeout and output,
process-tree containment, and safe normalized errors.

`PUBLISHED` now means that the already-final-path candidate has a valid, matching seal
and is eligible for activation. Publishing performs no rename and writes no generation
content. Activation and rollback semantics from ADR-0006 remain unchanged.

Transactions left by the superseded staging-build implementation are never
auto-published. Recovery marks them interrupted and retains or quarantines only trees
whose lineage can be proved. Retry always allocates a new transaction and generation.

The schema-1 `stagingRelativePath` field remains readable for backward compatibility,
but new installation work must not use that directory as Hermes `InstallDir`.

## Alternatives considered

### Rewrite embedded launcher and editable-install paths after moving

Rejected. It would depend on uv and Python installation internals, require writes after
build, and still risk missing another absolute-path artifact.

### Request uv relocatable mode and keep the move

Rejected as the primary fix. Relocatable environments do not make copied console
entrypoints or editable project metadata safely movable in every supported layout.

### Rebuild only the launchers after moving

Rejected. It would make publication mutate the generation and would not prove that the
whole installed runtime works at the final path.

## Consequences

- A candidate directory can exist under `generations/` before it is sealed, but it is
  invisible to consumers until `active.json` names it.
- Install validation is stronger and catches path-bound launcher failures before Seal.
- Recovery and garbage collection must distinguish inactive candidate generations from
  sealed or active generations using transaction lineage.
- The staging root remains for compatibility recovery and other operation-private data,
  not for new Hermes runtime construction.
- ADR-0006's sole active pointer, seal, compare-and-swap activation, predecessor
  retention, managed environment and fail-closed rules remain in force.

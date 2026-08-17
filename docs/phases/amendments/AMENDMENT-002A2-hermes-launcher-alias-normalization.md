# YORVA Phase 2 Amendment 002A2 — Official Hermes Launcher Alias Normalization

> Status: ACCEPTED FOR IMPLEMENTATION
> Date: 2026-08-17
> Owner approval date: 2026-08-17
> Owner: Repository owner
> Phase: 2 (correctness defect in frozen discovery)
> Historical baseline: `phase-002-hermes-discovery-baseline-r1` (immutable; not moved)
> Related: `AMENDMENT-002A1-hermes-windows-command-resolution.md`
> Implementation: IN_PROGRESS
> Audit: covered by later independent review; this status is not an audit PASS

## Defect

The official Windows installer may copy `venv\Scripts\hermes.exe` to `bin\hermes.exe` as a user PATH entry for the same installation. Phase 2 discovery currently treats two runnable `--version` results as `AMBIGUOUS` even when both files are aliases of one official root.

`AMBIGUOUS` is reserved for two or more conflicting independent Runtime candidates, not for two launchers of one trusted official installation.

## Decision

On Windows, after `--version` inspection, YORVA may collapse exactly these two relative paths under one canonical official root `%LOCALAPPDATA%\hermes\hermes-agent`:

```text
bin\hermes.exe
venv\Scripts\hermes.exe
```

Collapse is allowed only when all hold:

- both are canonical regular files;
- neither path uses a reparse-point escape;
- both are contained by that official root;
- both `--version` results normalize to the same version;
- file size is at most 64 MiB;
- SHA-256 of the two files is identical.

Then:

- discovery is not `AMBIGUOUS`;
- `selected` is `bin\hermes.exe`;
- no user selector is shown;
- an internal warning `HERMES_LAUNCHER_ALIAS` may be recorded.

If version, digest, containment, relative location, or installation root differs, the result remains `AMBIGUOUS`.

Command precedence is unchanged:

1. consistent official `bin\hermes.exe`;
2. only `venv\Scripts\hermes.exe`;
3. existing 002A1 Python fallback when both launchers are absent;
4. never `hermes-agent.exe` or the repository wrapper.

## Non-goals

- no change to `>=0.19.0 <0.21.0`;
- no arbitrary executable selector;
- no new Python execution surface;
- no move/delete of Phase 2 baseline tags;
- no Phase 3/4 features.

## Tests

Same-root identical digest/version selects `bin`; same-root digest or version mismatch stays ambiguous; different roots stay ambiguous; venv-only remains supported; missing launchers keep the Python fallback; PATH plus official path of one canonical file execute once.

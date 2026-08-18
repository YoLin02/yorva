# YORVA Phase 2 Amendment 002A3 — Read-Only Active Generation Discovery

> Status: ACCEPTED FOR IMPLEMENTATION  
> Date: 2026-08-18  
> Owner approval date: 2026-08-18  
> Owner: Repository owner  
> Phase: 2 (frozen discovery contract extension; not a Phase 2 re-freeze)  
> Historical baseline: `phase-002-hermes-discovery-baseline-r1` (immutable; not moved)  
> Related: `AMENDMENT-002A1`, `AMENDMENT-002A2`, `AMENDMENT-003A4`, `ADR-0006`  
> Design: `docs/phases/PHASE-003-generation-installation-architecture.md`  
> Implementation: NOT STARTED; batch-gated with Phase 3 generation work  
> Audit: later independent review; this status is not an audit PASS  
> Gate: this amendment is not Phase 2 COMPLETE / FROZEN / ACCEPTED as a new baseline

## 1. Classification

This is a governed extension of frozen Phase 2 discovery so a Phase 3 generation tree can be found without introducing a second activation source.

It does not:

- change the historical `phase-002-hermes-discovery-baseline-r1` tag;
- rewrite `AUDIT-002` / `AUDIT-002A1` / `AUDIT-002A2`;
- add a YORVA `hermes.exe` shim, junction, `current` directory, or SQLite active flag;
- install, repair, or mutate PATH;
- begin Phase 4.

Owner decision **D1** (2026-08-18): discovery may only **read** `control/active.json` and resolve the named generation. No other pointer is added.

## 2. Problem

Phase 2 enumerates, in order:

1. `PATH` entries named `hermes` / `hermes.exe`;
2. `%LOCALAPPDATA%\hermes\hermes-agent\bin\hermes.exe`;
3. `%LOCALAPPDATA%\hermes\hermes-agent\venv\Scripts\hermes.exe`.

Generation installs live at:

```text
%LOCALAPPDATA%\hermes\generations\gen_<id>\bin\hermes.exe
```

Without this amendment, a successful generation install is invisible to frozen discovery, or becomes `AMBIGUOUS` if the legacy `hermes-agent` tree still exists.

## 3. Decision

On Windows, before the existing candidate enumeration:

```text
read %LOCALAPPDATA%\hermes\control\active.json
```

If and only if the pointer is **fully valid** (section 4), discovery:

1. resolves `<managed-root>\<generationRelativePath>\bin\hermes.exe`;
2. uses that single command as the selected Windows managed candidate;
3. does **not** add the legacy `hermes-agent` launchers as competing `--version` candidates;
4. may still apply amendment 002A1 fallback **only inside that same generation root** when both generation launchers (`bin\hermes.exe` and `venv\Scripts\hermes.exe`) are absent.

If `active.json` is missing, unreadable, schema-invalid, contains an absolute or escaping path, names an unknown id format, or fails seal/containment checks:

```text
do not guess;
do not scan generations/ for the newest directory;
fall through to the existing frozen Phase 2 enumeration (PATH + hermes-agent).
```

`AMBIGUOUS` remains reserved for two or more independent Runtimes. A valid `active.json` plus a leftover legacy tree is **not** ambiguous: the pointer wins, the legacy tree is ignored for selection.

Amendment 002A2 alias collapse continues to apply only under one official root. Under a generation root the pair `bin\hermes.exe` and `venv\Scripts\hermes.exe` may collapse the same way. They must not be collapsed with a path outside that generation.

## 4. Valid pointer (read-only)

`active.json` is valid only when all hold:

- the file is a regular file, not a reparse point, size-bounded;
- `schema == 1`;
- `runtimeKind == "hermes"`;
- `generationId` matches `^gen_[0-9a-z]{22}$` (closed id);
- `generationRelativePath` is exactly `generations/<generationId>` (forward slashes in JSON, host separators on disk);
- the path is contained in `%LOCALAPPDATA%\hermes` after canonicalization;
- `generations/<id>/generation.json` and `manifest.json` exist, are regular files, and match the stored `sealSha256` / `manifestSha256`;
- `sourcePin` and `version` are present (used for diagnostics; version classification remains Phase 2 compatibility rules).

Discovery never writes `active.json`, never creates generations, and never deletes legacy trees.

## 5. Contract impact

- HTTP Discovery DTO shape unchanged: still one selected command or a typed negative state.
- New implicit state sources: **none**.
- PATH is not consulted to decide which generation is active.
- SQLite accepted-installation cache remains non-authoritative.

## 6. Required tests (this amendment)

- missing `active.json` → frozen enumeration unchanged;
- valid pointer → exactly one selected generation launcher;
- valid pointer + leftover `hermes-agent` → not `AMBIGUOUS`;
- malformed / absolute / `..` / wrong id / seal mismatch / reparse → fall through, never newest-gen;
- 002A1 fallback only under the generation root;
- 002A2 collapse only within one generation root;
- no write to `control/` during `Detect`.

## 7. Implementation batch

Implement only with Phase 3 generation Batch C (publish/activate) or later, after `active.json` exists in the design’s writer. Shipping the reader before the writer is allowed if tests use fixtures; shipping the writer without this reader is forbidden (discovery would miss the install).

## 8. Completion evidence

To be filled when the amendment is implemented. Historical Phase 2 baseline and audits stay as they are until a separate re-freeze task, which this amendment does not authorize.

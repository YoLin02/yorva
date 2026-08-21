# ADR-0006: Generation Install Transaction and Sole Active Pointer

- Status: Accepted (partially superseded by ADR-0009)
- Date: 2026-08-18
- Owner approval: 2026-08-18 (D1–D5)
- Related: ADR-0003, Phase 2 `AMENDMENT-002A3`, Phase 3 `AMENDMENT-003A4`
- Design: `docs/phases/PHASE-003-generation-installation-architecture.md`

> 2026-08-21: ADR-0009 supersedes only the staging-build and publish-by-rename
> mechanism. The sole active pointer, sealed-generation, activation CAS, recovery and
> retention decisions in this ADR remain authoritative.

## Context

Phase 3 installation currently persists intent in several places at once: SQLite Operations, a filesystem promotion journal, a live `hermes-agent` directory that is renamed on retry, quarantine siblings, an ownership marker, and HKCU environment values. Recovery consults Operation rows. Retry treats the latest install Operation as directory owner. `AUDIT-003` through `AUDIT-003R6` remain FAIL. Extending that machine with compensating states (`ROLLED_BACK`, `ENV_APPLIED`) would add another implicit commit boundary.

Phase 2 discovery is frozen on `%LOCALAPPDATA%\hermes\hermes-agent\bin\hermes.exe` and `venv\Scripts\hermes.exe`. A generation tree at `generations\gen_<id>\` cannot be discovered without a governed Phase 2 change. A second launcher binary, junction, or `current` directory would become a second activation source.

## Decision

Replace the candidate / quarantine / promotion-journal model with one Install Transaction:

```text
CREATED → BUILDING → SEALED → PUBLISHED → ACTIVATING → COMMITTED
```

`FAILED` is allowed only while `control/active.json` does not name this transaction's generation.

`active.json` is observed as `MISSING`, `VALID`, or `INVALID`. Those classes are not interchangeable. `INVALID` is `BLOCKED_UNSAFE` and never treated as a first-install vacancy. First activation requires observed `MISSING` plus an `ActiveBefore=ABSENT` snapshot. Later activation is a compare-and-swap against `VALID(generationId + digest)`.

Authorities are non-overlapping:

| Concern | Sole authority |
| --- | --- |
| in-flight intent and recovery | `control/transactions/txn_<id>.json` |
| which generation is active | `control/active.json` |
| sealed tree bytes | `manifest.json` + `generation.json` |
| Desktop/API progress | SQLite Operation (one-way projection) |
| desired `HERMES_HOME` / managed PATH | valid `active.json` + fixed policy |

Forbidden activation sources: SQLite active flag, `current` / junction / shim, “newest generation” scan, PATH membership, latest Operation, live `hermes-agent` rename.

Owner-locked policy (2026-08-18):

- **D1** — Phase 2 gains a read-only `active.json` resolver. No stable launcher, shim, or junction.
- **D2** — `HERMES_HOME` remains `%LOCALAPPDATA%\hermes`. Official user data is not migrated. Generations stay under `generations\`.
- **D3** — `config-templates` failure is a warning and does not by itself block Seal. Executable, version, runtime validation, or generation integrity failure still blocks Seal.
- **D4** — GC keeps the active generation forever, the most recent previous committed generation, and at most the most recent failed staging/generation that is lineage-proven. Everything else that is proven YORVA-owned and unreferenced may be collected. Unknown directories and Hermes user data are never collected.
- **D5** — After the first generation `COMMITTED`, only a PATH entry that is proven YORVA-managed (`...\hermes-agent\bin` written by YORVA, or a previous generation `\bin`) may be removed. User-authored Hermes PATH entries stay.

Retry always creates a new transaction, staging directory and generation id. Recovery is `Observe → DecideRecovery → Execute → Observe` and never reads Operation as authority.

## Alternatives considered

### Keep live-directory rename and add more journal states

Rejected. `AUDIT-003R5`/`R6` show the rename/journal lag and post-commit PATH write cannot be closed by extra compensating states without a second machine.

### Stable `hermes.exe` shim at the frozen Phase 2 path

Rejected by D1. It would be a second activation mechanism and a new binary.

### Relocate official user data under `mutable-data\`

Rejected by D2. The pinned installer writes `$HermesHome\.env`, `config.yaml`, `skills` and related directories. Relocating them changes Hermes runtime data paths.

### Use SQLite as the transaction store

Rejected. Recovery must work from the managed filesystem after a daemon-data loss, and Operation already serves API/UI. Duplicating transaction state in SQLite recreates the double machine.

## Consequences

Positive:

- one recovery truth, one activation pointer;
- failed new installs cannot rewrite the active generation tree;
- environment can be rebuilt from `active.json`;
- GC has an explicit, small retention rule.

Costs:

- Phase 2 must be formally amended (`002A3`) before generation paths are discoverable;
- legacy `%LOCALAPPDATA%\hermes\hermes-agent` remains on disk and must never be adopted or deleted;
- implementation must be batched; the old promotion machine is removed only after generation tests pass.

Historical `AUDIT-003`–`R6` FAIL reports stay immutable. This ADR does not freeze Phase 3 or unlock Phase 4.

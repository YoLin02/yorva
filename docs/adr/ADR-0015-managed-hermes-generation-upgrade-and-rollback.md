# ADR-0015: Managed Hermes Generation Upgrade and Rollback

- Status: Proposed — not implementation authority
- Date: 2026-08-25
- Owner approval: Pending
- Related: ADR-0006, ADR-0009, ADR-0013, Phase 7 P7-D5, Phase 7 B1 qualification evidence

## Context

Hermes `0.20.5` exposes `update --check`, `update --plan` and ordinary update commands,
but they do not implement YORVA's managed-install contract:

- `update --check` clears stale Git locks, fetches a moving remote and changes Git
  metadata without a bounded subprocess contract;
- `update --plan` emits unbounded human inventory, can omit failed probes without a
  completeness marker and does not compare the active managed generation to this YORVA
  build's immutable target;
- ordinary update edits the checkout, virtual environment and dependencies in place,
  may stash/switch/reset a user checkout, creates plaintext secret-bearing backup state,
  and exposes `--force` and `--force-venv`;
- Hermes' configuration migration registry is forward-only. The presence of config
  schema version `38` in the reference source does not prove that a previous generation
  can read data written by a later one.

These surfaces conflict with ADR-0006's sole `active.json` pointer and ADR-0009's
final-path immutable generation. Phase 7 needs a YORVA-owned upgrade transaction and an
exact from/to user-data compatibility gate. Retaining old executable bytes is not, by
itself, a safe rollback.

## Decision

### Authorities and eligibility

YORVA derives an upgrade plan only from:

- a `VALID` managed `control/active.json` and the matching sealed active generation;
- the active generation's bounded `generation.json` and complete seal manifest;
- one exact Hermes snapshot packaged with the running YORVA build, including compiled
  commit, archive size, SHA-256, version, license and pinned installer inputs;
- authoritative live Runtime/Profile reconciliation;
- an accepted exact from/to user-data compatibility record.

It does not run `hermes update --check`, `--plan`, ordinary `hermes update`, `--force`,
`--force-venv`, or any in-place repair/update route. It does not read a moving branch or
adopt an unmanaged checkout. An unmanaged installation receives only an explanation and
no mutation.

Version text alone is not target identity. A same-version archive with a different
source digest is a distinct candidate. Every read of the packaged archive is verified
against the compiled size and SHA-256; integrity mismatch fails without network or
mirror fallback. Any transport amendment may change byte location only, never the exact
target identity.

Authorities remain those of ADR-0006/0009:

| Concern | Sole authority |
| --- | --- |
| upgrade intent and recovery | `control/transactions/txn_<id>.json` with transaction kind `UPGRADE` |
| active generation | `control/active.json` |
| candidate and active bytes | complete manifest plus `generation.json` seal |
| candidate ownership | bounded `.yorva-candidate.json` tied to transaction and generation IDs |
| Desktop/API progress | SQLite Operation projection |
| packaged target | compiled exact archive identity in this YORVA build |
| rollback eligibility | accepted exact from/to user-data compatibility record, not the presence of an old directory |

### Plan and conflicts

The plan is read-only with respect to Hermes and reports current/target exact identities,
managed eligibility, affected Profiles/gateways, required disk space, stop/restart plan,
protection backup, expected post-checks and whether rollback is eligible. Partial or
unknown inventory makes the plan non-executable.

Upgrade/Rollback holds the narrow Runtime-installation coordination lock and conflicts
with Install, prerequisites, Restore, Backup mutation, every affected Instance/lifecycle/
Channel mutation and Skill/MCP mutation. It does not hold a SQLite transaction while
building, invoking a process, hashing, waiting on a gateway or performing network work.
All children have the existing bounded argv/environment, process-tree, stdout/stderr,
timeout, cancellation and reap contract.

Immediately before external work, YORVA re-observes the active pointer and exact
Runtime/user-data fingerprint. A changed pointer, seal, target archive or relevant
external state is a stable conflict rather than a request to overwrite.

### New-generation transaction

Upgrade reuses the ADR-0006 state machine and ADR-0009 final-path construction:

```text
CREATED
  -> BUILDING at generations/gen_<new-id>
  -> final-path functional validation
  -> SEALED
  -> PUBLISHED
  -> ACTIVATING by active.json compare-and-swap
  -> POSTCHECKING
  -> COMMITTED
  -> RECOVERY_REQUIRED
```

`FAILED` remains legal only before `active.json` names the candidate, as required by
ADR-0006. A post-check failure while the candidate is active is never rewritten as an
ordinary failed install. It enters `RECOVERY_REQUIRED` until a linked, separately
journaled rollback or manual recovery reaches an observed terminal state. A verified
linked rollback may record the historical upgrade outcome as `ROLLED_BACK`; neither that
label nor an SQLite Operation is activation authority. These additional upgrade-only
states require Owner approval with this ADR.

Before `BUILDING`, YORVA creates and verifies an encrypted Runtime protection point under
accepted ADR-0013 semantics. If the exact upgrade is proven incapable of reading or
writing shared user data during build and activation, the protection point is still part
of the P7 plan but need not be recreated when an existing policy-valid verified backup
is selected. Without an accepted and verified protection point, the upgrade is blocked.

The candidate is allocated at its final `generations/gen_<new-id>` path, receives the
bounded ownership record before build work and remains invisible until activation. The
active generation is never written. Build and validation must also prove that candidate
commands do not migrate or mutate the shared live Hermes user-data root before
activation. If the official installer or probe cannot be isolated from shared data, the
exact target remains unsupported for managed upgrade.

Before Seal, both documented launchers execute `--version` from the final candidate path
and must report the exact supported target. The complete tree is manifested and sealed;
`PUBLISHED` performs no rename or content mutation. Activation compares the full observed
`ActiveBefore` generation ID and digest, writes one complete durable `active.json`, and
fails closed on any mismatch.

Cancellation is terminal only before activation when the active pointer and shared data
remain unchanged. Once activation begins, cancellation becomes a request to stop further
optional work; recovery must first determine the authoritative pointer, postcondition and
rollback eligibility. It must not label an activated but unchecked Runtime `CANCELLED`.

### Postconditions

After activation, YORVA verifies all of the following for the exact target:

- `active.json` names the candidate digest and the full seal still matches;
- both final-path launchers report the target identity and discovery selects that
  generation;
- default and named Profile inventory reconciles without cross-Profile changes;
- each previously configured model has a safe authoritative read-back;
- planned gateways/lifecycle state can be restored and normalized;
- Channel binding state reconciles without exposing credentials;
- installed Skill identity/enabled/audit state and configured MCP identity/tool state
  reconcile through the exact qualified adapters;
- health/security checks required by the plan are not malformed, unsupported or
  `UNKNOWN`.

Only the checks for features present in the pre-upgrade inventory run, but every present
feature must have a qualified bounded read-back before the plan is executable. Exit zero,
a matching version string alone, or a successfully replaced pointer is not upgrade
success.

If every post-check passes, the filesystem transaction becomes `COMMITTED` and the
Operation projection becomes successful. If a check fails, the transaction records the
failed stage and observed truth, stops affected gateways where continued execution is
unsafe, and follows the rollback gate below.

### User-data compatibility gate

Every exact source-generation/target-snapshot pair requires a reviewed compatibility
record that states:

- source and target source digests and Hermes/config schema identities;
- the Runtime-owned user-data categories read and written by install, first launch,
  gateway start and each post-check;
- whether the target performs automatic, deferred or lazy migration;
- whether those writes are backward-readable by the exact previous generation;
- whether restoring the pre-upgrade protection point is required before binary rollback;
- disposable-state upgrade, downgrade/read-back and Windows evidence identity.

Compatible version ranges, a shared `0.20.x` label, a forward migration function or
retained old bytes do not prove downgrade safety. The record is part of the compiled
qualified target policy and unknown/missing fields fail closed.

For P7, if an exact pair lacks proof that either no relevant user data is changed or the
previous generation can safely read the post-upgrade data (optionally after the fully
qualified protection-point restore), **automatic and user-requested rollback remain
disabled**. Because the Phase 7 stop condition treats unsafe user-data migration as a
blocker, managed upgrade for that pair also remains capability-false until the Owner
accepts a separate migration/recovery amendment. The UI must not offer a rollback button
merely because a previous generation directory exists.

### Rollback

Rollback is a new typed filesystem transaction linked to the originating upgrade, not
pointer editing by the UI. It is allowed only when:

- the current and predecessor generations are complete, sealed, lineage-proven and match
  the accepted compatibility record;
- `active.json` still equals the exact failed/current generation ID and digest observed
  by the rollback plan;
- the prior generation is the retained accepted predecessor, not the newest directory
  found by scanning;
- external user-data and pointer fingerprints have not changed after planning;
- any required protection-point Restore has completed and passed its own authoritative
  postcondition;
- all affected gateways are stopped and conflicts remain held.

The rollback atomically compare-and-swaps `active.json` from the current generation to
the previous sealed digest. It never modifies either generation. YORVA then restores the
planned gateway state and runs the same Profile/model/Channel/Skill/MCP/health read-backs
against the previous generation. Only that complete verification may report rollback
success.

When rollback is ineligible or itself fails, YORVA does not flip the pointer repeatedly,
infer success, or replay upgrade. It reports recovery-required/unknown, leaves unsafe
gateways stopped, preserves the active and previous generations, packaged identity,
protection backup and transaction evidence, and blocks conflicting mutation for explicit
recovery.

### Recovery and retention

Daemon restart uses `Observe -> DecideRecovery -> Execute -> Observe` from ADR-0006.
SQLite Operations never authorize activation or replay.

- Before Seal, an interrupted candidate can be marked failed and retained/collected only
  when ownership and lineage are proven; retry creates new IDs.
- At `PUBLISHED`, recovery verifies the seal and original `ActiveBefore` snapshot before
  deciding whether activation is still safe; it never activates after a CAS conflict.
- At `ACTIVATING` or `POSTCHECKING`, recovery first reads `active.json`. If it names the
  target, recovery resumes bounded post-check/rollback decision, not activation. If it
  names the predecessor, the candidate was not committed. Any third/invalid value blocks
  unsafe recovery.
- A committed pointer plus incomplete Operation projection reconciles from filesystem and
  Runtime truth; the Operation is not the commit record.

ADR-0006 D4 retention remains in force: keep the active generation forever, the most
recent previous committed generation and at most the most recent failed lineage-proven
candidate. The predecessor required by an in-flight upgrade/rollback and its accepted
protection point are pinned from GC until the transaction reaches a verified terminal
state and the recovery/rollback policy permits release. Unknown directories, legacy
`hermes-agent` and Hermes user data are never collected.

## Alternatives considered

### Wrap `hermes update`, `--check` or `--plan`

Rejected. The commands use moving Git/in-place semantics, human output and force/stash
behavior that cannot become a sealed-generation transaction by wrapping them.

### Update dependencies inside the active generation

Rejected. It breaks manifest truth, immutable rollback and the sole-pointer model.

### Activate the newest or highest-version generation

Rejected. Directory order and version text are not activation authority. Only an exact
CAS on `active.json` may activate or roll back.

### Always roll back when post-check fails

Rejected. A previous executable may be unable to read data written by the target. An
automatic pointer reversal without an exact compatibility record can deepen data loss.

### Keep every generation indefinitely

Rejected. ADR-0006 already defines bounded lineage-aware retention; unknown data is
preserved without making all known generations permanent.

## Consequences

Positive:

- the active sealed generation is never mutated by upgrade;
- source provenance, transaction recovery and activation remain single-authority;
- plan, upgrade success and rollback eligibility reflect exact candidate facts;
- rollback cannot be advertised solely because old binaries exist;
- failed/crashed upgrades preserve bounded recovery evidence without blind replay.

Costs and constraints:

- every packaged Hermes upgrade target needs exact compatibility and Windows destructive
  evidence in addition to install provenance;
- upgrade depends on an accepted encrypted protection-point design;
- post-check requires qualified read-back for every configured P7 feature;
- some exact target pairs will intentionally report upgrade/rollback unsupported.

## Required approval and validation before acceptance

This ADR remains **Proposed**. Owner approval is required for:

- the exact packaged-snapshot-only upgrade policy and managed-install eligibility;
- the protection-point requirement and dependency on ADR-0013;
- the exact from/to compatibility-record schema and fail-closed capability policy;
- the upgrade-only `POSTCHECKING`, `RECOVERY_REQUIRED` and historical `ROLLED_BACK`
  transaction states;
- the post-check set, gateway stop/restart behavior and recovery-required UX;
- predecessor/protection-point retention and release policy.

Acceptance also requires exact-candidate and disposable Windows evidence for active-tree
immutability, final-path dual launchers, source size/hash enforcement, shared-data build
isolation, activation CAS conflict, every configured-feature post-check, daemon restart at
each state, cancellation before/after activation, external modification, compatibility
pass/fail, successful rollback and failed/ineligible rollback. Until those decisions and
tests pass, B8 upgrade and rollback capabilities remain false and no product mutation is
authorized.

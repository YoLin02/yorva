# ADR-0018: YORVA-managed Skills Lifecycle and Hermes Projection

- Status: Accepted
- Date: 2026-08-25
- Owner approval: Accepted through the Phase 7 B4 managed-lifecycle direction
- Related: ADR-0003, ADR-0012, ADR-0016, Phase 7 B4

## Context

Hermes `0.20.5` exposes a useful enabled-Skills inventory, but its native install,
update, configure and remove surfaces do not form a reliable production mutation
contract. In particular, some paths are human-output-only, update may select a force
path, and completed mutation truth cannot always be read back.

Those limitations describe Hermes-native mutation. They do not prevent YORVA from
owning a separate, explicit Skill lifecycle and projecting an approved Skill package
into the exact Hermes Profile Skills directory.

## Decision

Phase 7 separates two capability layers.

### Runtime-native capability

The Hermes adapter reports native inventory, install, update, remove, enable/disable
and Profile-binding capability independently. For exact Hermes `0.20.5`, qualified
inventory remains available where its prerequisites are satisfied. Unreliable native
mutations report `supported=false` and `reason=deferred_upstream`.

### YORVA-managed lifecycle

YORVA provides install, update, enable, disable, remove, inventory and drift detection
for YORVA-managed Skills. It does not call the rejected Hermes-native mutation paths.

The sole managed-copy authority is below the daemon data directory:

```text
{dataDir}/skills/managed/
```

Hermes Profile directories are projection targets only:

```text
default Profile: {HERMES_HOME}/skills/<skill-id>/
named Profile:   {HERMES_HOME}/profiles/<profile-id>/skills/<skill-id>/
```

Phase 7 uses copy projection on Windows. It does not require symlink privilege or
Developer Mode.

The first source catalog is compile-time closed and reviewed. Public requests select a
source ID only; they cannot provide a URL, local path, command, environment, force flag
or arbitrary Skill bytes.

### Ownership

YORVA never adopts an existing Hermes Skill by name or content. A projection is managed
only when all of the following agree:

- a YORVA `managed_skills` record for the exact Instance and Skill;
- the record's deployment ID and expected content SHA-256;
- a small YORVA projection marker; and
- the observed projected content SHA-256.

The digest is one deterministic whole-package digest, not a per-file database index.
An existing destination without matching ownership is external and read-only. Update,
disable and remove stop on missing or changed ownership evidence.

### Enable and disable

For a YORVA-managed Skill:

```text
enable  = project the managed copy into the exact Runtime Profile
disable = remove only the verified YORVA-owned projection
```

This is Runtime projection state. It is not represented as Hermes native Profile
enable/disable state and does not edit Hermes `config.yaml`.

### Inventory and reconcile

Inventory merges YORVA records with observed Hermes Skills. It distinguishes
`YORVA_MANAGED`, `EXTERNAL`, `RUNTIME_BUNDLED` and `UNKNOWN`. Missing managed projection
is `DRIFT_MISSING`; changed managed projection is `DRIFT_MODIFIED`. External Skills are
shown read-only and are never imported automatically.

P7 reconcile is deliberately small: detect drift accurately and allow an explicit
managed action to project YORVA's own clean copy. It does not automatically overwrite
modified files or construct a general repair framework.

### Lifecycle and Operations

Skill mutations are durable typed Operations. A running exact Instance is restarted
after a successful projection change so Hermes does not retain a stale in-process Skill
cache. A stopped Instance remains stopped. Daemon restart marks an interrupted Skill
Operation failed and the next inventory read reconciles from the managed record and
filesystem truth; destructive work is not replayed blindly.

## Alternatives considered

### Depend on Hermes native mutation only

Rejected for `0.20.5` because it cannot satisfy the production postcondition and scan
requirements.

### Keep Phase 7 Skills read-only

Rejected because a small YORVA-owned managed store and copy projection provide the
required lifecycle without weakening the Hermes-native capability truth.

### Use the Hermes directory as YORVA's store

Rejected because it would make ownership ambiguous and could overwrite or delete
external user resources.

### Build a general marketplace or arbitrary source importer

Rejected as Phase 7 scope expansion. New source descriptors can be reviewed and added
later without changing the ownership or projection contract.

## Consequences

Positive:

- P7 can deliver real Skill lifecycle management while native limitations stay honest;
- external and bundled Skills remain visible and protected;
- Windows does not depend on symlink support;
- a future stable native strategy can coexist without changing the public lifecycle.

Costs:

- YORVA stores one managed copy in addition to the Runtime projection;
- projection drift must be shown and resolved explicitly;
- the first source catalog is intentionally small and closed.

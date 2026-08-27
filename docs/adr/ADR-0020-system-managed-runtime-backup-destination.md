# ADR-0020: System-managed Runtime backup destination

- Status: Accepted
- Date: 2026-08-27
- Owner approval: Approved 2026-08-27
- Supersedes: ADR-0017 for Backup Create destination selection
- Related: ADR-0013, Phase 7 B6

## Context

The first Backup Create implementation required a native Save dialog and a short-lived
destination capability. Product validation found that routine protection should not
depend on a user file choice and should have one predictable managed location.

The public HTTP surface must still not accept a local path. Backup artifacts contain
Hermes credentials, sessions and account data and must remain encrypted and indexed.

## Decision

Backup Create writes a new encrypted artifact below the current user's YORVA system
application-data directory:

```text
{YORVA appData}/backups/{backupId}.yorva-backup.age
```

The daemon derives this path from its trusted bootstrap `dataDir` and its own random
backup ID. React, HTTP callers and Runtime input cannot select or observe the path. The
public request is a closed empty JSON object plus the existing idempotency header.

The destination directory is created by the daemon, revalidated as a local non-reparse
path, and remains subject to the existing private staging, no-overwrite publication,
durable flush, ciphertext authentication, checksum and safe-index requirements. Device
encryption remains the default. Restore and delete resolve only indexed artifacts.

ADR-0017's native destination grant remains compatibility code during Phase 7 but is no
longer exposed by the Desktop Backup Create flow or public HTTP contract.

## Consequences

- backup creation no longer opens a file picker or depends on user path selection;
- encrypted artifacts have one predictable per-user system location;
- callers still cannot turn Backup Create into arbitrary filesystem access;
- portable export is a separate future action and is not implied by managed backup;
- documentation and OpenAPI describe the system-managed destination as authoritative.

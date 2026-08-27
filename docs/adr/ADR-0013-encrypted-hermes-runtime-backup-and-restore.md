# ADR-0013: Encrypted Hermes Runtime Backup and Restore

- Status: Accepted
- Date: 2026-08-25
- Owner approval: Approved 2026-08-25
- Related: ADR-0003, ADR-0006, ADR-0009, Phase 7 P7-D4, Phase 7 B1 qualification evidence

## Context

Phase 7 requires a complete local backup and Restore flow that does not disclose Hermes
credentials, sessions or account state as ordinary plaintext. Hermes `0.20.5` does not
provide that product boundary:

- full backup is scoped to the Hermes root rather than one selected Profile and may also
  collect declared files outside that root;
- full and quick backup contain secret-bearing `.env`, authentication, OAuth/MCP token,
  channel, pairing, session, account and data state in an unencrypted ZIP or directory;
- a full backup can be published after individual file errors and has no YORVA format,
  Runtime-version or checksum manifest;
- import recognizes only a small marker set, restores members incrementally, permits an
  `_external/` mapping, lacks complete archive bounds and can report completion after
  per-file failures;
- import has no pre-Restore protection point, global transaction, authoritative
  postcondition or rollback truth. `--force` only removes a prompt.

The current `backups` table is Instance-scoped, while the only candidate complete Hermes
backup scope spans the shared Runtime root and named Profiles. Reinterpreting an existing
Instance row as Runtime-scoped would be destructive and misleading.

The B1 evidence therefore classifies the direct full backup, quick backup and import
surfaces as `NO-GO`. A YORVA-owned encrypted container and Restore transaction are needed
before B6 or B7 can expose a capability.

## Decision

### Scope and authorities

The first complete backup is **Hermes Runtime user-data scope**. It covers the exact,
qualification-reviewed Hermes-owned data set under the canonical Hermes root across the
default and named Profiles. It does not back up YORVA generation binaries,
`control/active.json`, install transactions, Operation staging, YORVA SQLite, arbitrary
user paths, or automatically discovered external paths. A future external-memory member
requires a separately reviewed typed descriptor; the upstream `_external/` convention is
not accepted as an arbitrary path capability.

Source enumeration must come from a documented official surface or a separately approved
fixed Hermes adapter descriptor. It cannot import Hermes Python internals, inspect an
undocumented state database, or expose a caller-selected path walk. If no qualified source
can enumerate and read the complete declared scope, the backup capability remains false.

The Desktop must disclose that this Runtime-scoped payload may include provider and MCP
credentials, OAuth client/access/refresh state, channel credentials, pairing/account
state, sessions, conversations, databases, Skills and configuration. It must not describe
the backup as Instance-only or secret-free.

Authorities are deliberately singular:

| Concern | Sole authority |
| --- | --- |
| Hermes data and native scope | exact qualified Hermes Runtime/Profile stores |
| encrypted artifact bytes | the daemon-derived `.yorva-backup.age` file below YORVA system application data |
| device-managed decryption identity | YORVA OS-backed `SecretStore` entry referenced only by a non-secret key ID |
| portable decryption secret | the user-held passphrase, supplied for one request and never persisted |
| safe inventory metadata | YORVA backup index after artifact verification |
| Restore/recovery truth | a durable YORVA-owned filesystem Restore transaction record plus observed Runtime state; SQLite Operation is projection only |

No Hermes credential is copied into YORVA `SecretStore`. The only secret stored there for
device-managed backup is the YORVA-owned backup decryption identity.

### Standard encrypted container

The proposed P7 format is an **age v1 encrypted file**, created and read only through a
maintained implementation of the documented age file format. YORVA will not implement
cryptographic primitives, an encryption format or a key-derivation function itself.

The encrypted plaintext is one constrained ZIP64 payload containing:

```text
manifest.json
payload/hermes-runtime.zip
```

`manifest.json` is a bounded, canonical JSON document with at least:

- YORVA backup format and manifest schema versions;
- backup ID, creation time and `runtime` scope;
- Runtime kind, exact Hermes version and source/generation identity when managed;
- an explicit included-data category list and exclusion list;
- payload byte length and SHA-256;
- member count, total expanded bytes and the qualification policy version;
- creation completion marker and no secret values.

The inner Hermes payload remains opaque until Restore preflight, but it is not trusted
merely because age authentication or its SHA-256 succeeds. Restore must independently
validate every nested archive member against the accepted path/type/size policy before
any Runtime mutation.

The published encrypted file also receives a whole-file SHA-256 in the safe backup index.
That checksum identifies accidental replacement and the exact selected artifact; age
authentication remains the cryptographic integrity check. A checksum is not represented
as authentication or encryption.

Two mutually exclusive recipient modes are allowed:

1. **Device-managed mode.** A standard age X25519 identity is generated by the selected
   library. Its private identity is stored only in the OS-backed YORVA `SecretStore`;
   SQLite and the manifest retain only a non-secret key reference and public recipient.
2. **Portable passphrase mode.** The standard age passphrase/scrypt recipient is used.
   The passphrase exists only in the initiating request/session memory for create or
   Restore and is never stored or echoed.

There is no silent fallback between modes, no recovery key derived by YORVA, and no key
material in argv, URLs, SQLite plaintext, Operations, events, logs, diagnostics, audit
metadata, Desktop persistence or ambient child environments. Loss of the only identity
or passphrase makes that artifact unrecoverable; YORVA must state this truth rather than
bypass encryption.

Key rotation creates and verifies a new backup under a new key/recipient. It does not
rewrite an existing artifact in place. An OS-backed identity remains retained while a
known backup references it; deletion requires an explicit policy and must never infer
that an unindexed artifact is gone.

The age implementation, exact version, license, maintenance status, Windows behavior,
large-file streaming behavior and dependency vulnerabilities must be reviewed before
this ADR can be accepted. This proposal does not claim that a library has already passed
that review.

### Backup creation

Backup creation is one cancellable Runtime-scoped Operation with a Runtime-wide conflict
against Restore and Upgrade. Qualification decides whether gateways may remain running;
until consistent live capture is proven, the plan stops the affected gateways explicitly.
No database transaction remains open during filesystem or Runtime work.

The closed flow is:

1. Resolve a valid Runtime and exact scope. Derive one new destination below the trusted
   YORVA system application-data backup directory. No caller can supply a path.
2. Snapshot the authoritative Runtime/Profile inventory, versions and affected running
   state; reject unknown scope or an unsupported source contract.
3. Allocate an Operation-private directory under YORVA-owned state. Disable inherited
   ACLs and grant only the current user on Windows, or use mode `0700` where applicable.
   If the protection cannot be established, fail without plaintext fallback.
4. Create the upstream or adapter-owned plaintext payload only at a fixed path in that
   directory. Apply fixed member, per-file, total-byte and time limits. Any skipped,
   partial or uncertain source result fails the backup.
5. Build and verify the canonical manifest and nested archive, then stream the complete
   plaintext container through the approved age library into a hidden sibling temporary
   file in the system-managed backup directory.
6. Close and durably flush the encrypted temporary file, decrypt it through the same
   bounded verifier, validate its manifest and payload checksum, then atomically replace
   the final destination. Existing files are never overwritten without a separate
   explicit typed decision.
7. Re-hash the final encrypted bytes, insert/update only safe backup-index metadata and
   report `AVAILABLE` only after every verification succeeds.
8. Remove plaintext staging on success, failure, cancellation and daemon recovery.

The staging directory contains a bounded YORVA-created ownership marker tied to the
transaction record before plaintext is created. Startup recovery scans only exact owned
operation paths, never unknown directories, and cleans or quarantines interrupted
plaintext staging before allowing a new backup/Restore. YORVA does not claim secure
physical erasure on SSDs; the security control is access-restricted, short-lived staging
with fail-closed cleanup and no plaintext final artifact.

### Restore preflight and mutation

Restore is a separate Runtime-scoped destructive Operation. Before any Runtime file,
gateway or Profile mutation it must complete all of the following:

- resolve the exact indexed file below the YORVA system backup directory and reject
  device, pipe, directory and reparse-point inputs;
- verify whole-file checksum when indexed, complete age authentication and the selected
  key/passphrase authority;
- parse the bounded manifest and reject unknown format, scope, policy or Runtime
  compatibility;
- validate the entire nested archive before extraction: member count, compressed and
  expanded bytes, per-member size, compression ratio, duplicate/case-fold collision,
  empty/reserved names, absolute/UNC/device paths, `..`, alternate data streams,
  symlink, hardlink, device, reparse and other non-regular entries;
- prove every destination belongs to the fixed Runtime data allowlist; `_external/` does
  not authorize writes outside it;
- verify disk space, current Runtime identity, all affected Instances and the exact stop
  and restart plan;
- acquire conflicts against all affected Instance, lifecycle, Channel, Skill, MCP,
  Backup, Install and Upgrade mutations;
- compare the current Runtime/user-data fingerprint with the preflight snapshot and fail
  on external change;
- create and independently verify a pre-Restore protection point using this same
  encrypted format.

Only after preflight succeeds may YORVA stop the planned gateways and enter mutation.
It extracts through a YORVA-owned safe extractor into protected staging and applies only
the closed allowlisted plan. Direct `hermes import --force` into live state is forbidden;
it cannot supply the required all-members-first validation or transaction truth.

The durable Restore transaction records at least `PREFLIGHTED`, `PROTECTED`, `APPLYING`,
`VERIFYING`, `ROLLING_BACK` and a terminal outcome. It journals the exact top-level
targets and prior fingerprints before replacement. Individual file/directory publication
uses same-parent temporary material and atomic replacement where the platform supports
it. No command exit code or printed completion message is a success postcondition.

After apply, the adapter must reconcile exact Profiles and configured model, lifecycle,
Channel, Skill and MCP state, run the qualified health checks, and verify the affected
gateways can be restored to their planned state. Any missing, malformed, `UNKNOWN` or
cross-scope result fails verification.

On apply or post-check failure, YORVA restores the verified protection point and repeats
the authoritative reconciliation. Success is reported only if either the intended
Restore or the rollback has a fully verified postcondition. If rollback cannot be
verified, the Operation terminates as recovery-required/unknown, affected gateways remain
stopped, the protection point and transaction evidence are retained, and further
conflicting mutation is blocked. Recovery observes the transaction and Runtime truth; it
does not blindly replay a destructive step after daemon restart.

### Persistence and public truth

The current Instance-scoped `backups` rows are not reinterpreted. A later B2/B6 migration
must add explicit scope and Runtime ownership while preserving predecessor rows and must
be tested from empty and formal P6/P6.5 schemas. It stores only safe metadata: artifact
daemon-derived internal artifact path, byte size, checksum, format/scope/Runtime versions, state,
timestamps and a non-secret key reference.

An artifact is `AVAILABLE` only after create-time verification. Missing, externally
changed, undecryptable or malformed artifacts reconcile to an accurate non-available or
unknown state. Listing a row does not prove Restore eligibility.

## Alternatives considered

### Expose Hermes full/quick backup and import directly

Rejected. They publish secret-bearing plaintext and do not provide the required complete
preflight, bounds, transaction, rollback or postcondition.

### Exclude all secrets and call the remainder a complete backup

Rejected. Hermes Profiles would not be recoverable as represented, and the product would
misstate the data scope. A future non-secret export may be a separate capability.

### Build a YORVA-specific cipher or encrypted ZIP dialect

Rejected. YORVA must use a maintained documented format/library and must not design
cryptography.

### Store a universal backup password or a duplicate Hermes credential set in SQLite

Rejected. It creates a second plaintext authority and violates the write-only Secret
boundary.

### Restore directly into live files and rely on a best-effort backup

Rejected. Partial failure and daemon restart would have no authoritative recovery truth.

## Consequences

Positive:

- complete Runtime backups do not end as ordinary plaintext credential/session archives;
- one key/passphrase authority is explicit for each artifact;
- archive safety is proved before live mutation;
- Restore success, rollback success and unknown recovery states are not conflated;
- Hermes remains authoritative for Hermes state while YORVA owns only encryption,
  transaction and safe inventory metadata.

Costs and constraints:

- backup creation may require a bounded plaintext staging file because the upstream
  surface is not stream-oriented;
- Runtime scope is larger than one Instance and may require gateway downtime;
- a lost passphrase or OS-backed identity cannot be recovered by YORVA;
- a new maintained encryption dependency and destructive Windows smoke need security
  review;
- the first implementation cannot support arbitrary external data paths or upstream
  import shortcuts.

## Accepted Owner decisions and remaining validation

The Owner accepted this ADR on 2026-08-25, including:

- Runtime user-data scope and the explicit included/excluded category disclosure;
- age v1 plus the exact maintained implementation/version/license;
- device-managed and portable passphrase modes, retention and loss semantics;
- backup requires a planned gateway stop unless a later exact-candidate qualification
  proves a consistent bounded live capture;
- the non-destructive Runtime-scope backup schema migration;
- Restore transaction, protection-point retention and recovery-required UX.

Product qualification still requires exact-candidate fixture and Windows evidence for large files,
wrong/lost credentials, tamper/truncation, malformed manifests, duplicate/case-collision,
traversal/reparse/symlink/ADS/bomb inputs, space exhaustion, cancellation, daemon restart,
external conflict, partial apply, successful rollback and failed rollback. Until those
implementation and qualification tests pass, B6/B7 capabilities remain false and no
product mutation is authorized.

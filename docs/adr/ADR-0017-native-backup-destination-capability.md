# ADR-0017: Native Backup Destination Capability

- Status: Superseded by ADR-0020
- Date: 2026-08-25
- Owner approval: Approved 2026-08-25
- Related: ADR-0004, ADR-0013, Phase 7 B6

## Context

ADR-0013 requires Backup Create to publish only to one local destination selected through
the native Desktop picker. A remote caller must not be able to turn the backup request
into an arbitrary filesystem write.

The current Desktop bootstrap does not provide that distinction by itself:

- Tauri starts `yorvad` and passes the local API bearer credential through inherited
  stdin;
- `daemon_session` deliberately returns that bearer credential and the loopback endpoint
  to the React WebView so normal Desktop HTTP calls can be authenticated;
- the same bearer therefore proves the current local Desktop session, but it does not
  prove that a path came from native picker code;
- the current Tauri capabilities contain no file-dialog permission or backup picker;
- after bootstrap, the inherited Tauri-to-daemon stdin channel accepts only the fixed
  shutdown record, and daemon stdout rejects every post-handshake record.

Allowing React to send the path returned by a generic dialog plugin would not satisfy the
boundary: compromised or erroneous WebView code could submit a different path with the
same bearer. Accepting a path in `BackupCreateRequest` would also become an arbitrary
local/remote file-write API when the application later gains a Control Plane.

`BackupCreateRequest.DestinationRef` is already an opaque-ID-shaped contract, but no
authority currently issues, expires, binds or consumes that reference. Turning the
trusted bootstrap pipe into a narrowly typed native grant channel materially changes the
Desktop/daemon trust model and therefore requires an explicit decision.

## Decision

### Sole issuer and transport

Only one YORVA-owned Tauri command may open the native **Save Backup** picker. It returns
only an opaque `destinationRef` to React; the selected path never enters React state,
HTTP, OpenAPI, browser storage, events, Operations, audit metadata or ordinary logs.

The Tauri command sends one closed `backup_destination.grant` control record over the
already inherited parent-to-child stdio pipe. This pipe, not the WebView bearer, proves
that the grant came from the native shell that spawned the daemon. No HTTP endpoint may
issue a destination grant.

The existing parent-control protocol is extended only with:

```text
backup_destination.grant
shutdown
```

and one bounded daemon-to-parent acknowledgement for the grant. Unknown record types,
unknown fields, malformed framing, oversized records, duplicate request IDs, unexpected
acknowledgements and post-shutdown records fail closed. Stdout remains forbidden for
ordinary daemon output. The command does not return `destinationRef` until the matching
acknowledgement has been received; timeout, cancellation or daemon exit returns no usable
reference.

This is not a generic native IPC, file or command framework. Adding another privileged
record type requires its own reviewed need and contract update.

### Reference and binding

Tauri creates `destinationRef` with at least 256 bits of cryptographic randomness. The
reference is opaque, contains no encoded path and is safe only as a short-lived bearer
capability.

The daemon holds an in-memory grant with exactly:

```text
destinationRef
runtimeId = hermes
local Desktop bootstrap-session binding
current OS-user binding
canonical selected destination
parent-directory identity snapshot
issuedAt / expiresAt
unconsumed state
```

The fixed lifetime is two minutes and at most eight unconsumed grants may exist. Cleanup
is performed while issuing or consuming a grant; no background framework is required.
Daemon restart discards every grant.

The authenticated Backup Create request carries only `destinationRef`. Consumption is an
atomic part of accepting one new `backup.create` Operation. It checks the same bootstrap
session, `runtimeId`, current OS user and expiry, changes the grant to consumed, and binds
it to the created Operation ID before filesystem work begins. Concurrent or repeated
consumption has exactly one winner. Failed request validation does not expose the path;
after consumption, retry uses normal Operation/idempotency truth and never revives the
grant.

A future remote actor may invoke the typed Backup Create use case only with a separately
designed remote destination authority. It cannot call the Tauri command, issue a local
grant or supply a path.

### Destination policy

The native picker is restricted to a new local file with the exact suffix:

```text
.yorva-backup.age
```

The daemon, rather than the picker result alone, performs the authoritative validation
both when registering the grant and immediately before publication. It requires:

- one canonical absolute local path;
- an existing local parent directory owned by the current user;
- an absent final file (no overwrite in P7);
- the exact suffix and a bounded UTF-8 path;
- no relative, UNC, network-share, device, pipe, stream/ADS or directory target;
- no symlink or reparse-point component and no parent-identity change between grant and
  publication;
- the same current OS user that owns the daemon and initiated the native selection.

The daemon resolves a consumed reference to only the typed `LocalDestination` accepted by
the fixed Backup publisher. No application, Runtime, HTTP or Desktop API receives a raw
path or a general path resolver. The existing same-directory private staging,
no-overwrite atomic publication and final ciphertext verification rules from ADR-0013
remain mandatory.

### Error and capability truth

Grant cancellation, expiry, mismatch, duplicate consumption and unsafe destination use
stable non-secret errors. They do not return the path or distinguish sensitive path
details. A failed native grant cannot fall back to a caller-supplied path.

Backup Create remains unregistered and the Backup mutation capability remains false
until this ADR is accepted and the implementation, focused tests and required Windows
smoke pass.

## Minimum implementation boundary

Implementation is limited to:

1. one Rust-owned Save Backup picker command whose response is `destinationRef` or
   cancellation;
2. the two closed parent-control message schemas and bounded acknowledgement tracking;
3. one daemon-local, memory-only destination-grant registry;
4. atomic grant consumption at `backup.create` Operation acceptance;
5. the existing Hermes Backup publisher's typed local-destination constructor;
6. the closed Backup Create HTTP/OpenAPI body containing only `destinationRef`.

It does not add cloud storage, a generic file picker API, arbitrary export, path browsing,
directory grants, overwrite, grant persistence, a general IPC request bus or Restore
source selection. Restore requires its own existing-file picker use of the same principle
and remains blocked by B7 qualification.

## Required verification

Focused automated tests must prove:

- native picker cancellation issues no grant and the selected path never crosses the
  command response, React client, HTTP DTO, Operation, event or log boundary;
- grant and acknowledgement framing is bounded, closed and rejects unknown or mismatched
  records;
- references contain at least 256 bits, expire after the fixed lifetime, are discarded on
  daemon restart and never encode the path;
- wrong Runtime, wrong bootstrap session, wrong OS user, expired, duplicated and raced
  references fail, with exactly one successful atomic consumer;
- consumption binds the grant to exactly one durable Backup Operation before work starts;
- relative, existing, wrong-suffix, UNC, device, pipe, directory, ADS, symlink and reparse
  targets are rejected, including parent replacement between selection and publication;
- malformed/oversized path input on the trusted pipe cannot escape the single destination
  contract;
- the public Backup Create decoder rejects `path`, `url`, `command`, `environment`,
  unknown fields and missing/invalid `destinationRef`;
- remote/API-only tests cannot manufacture a grant, and capability remains false when the
  native authority is absent.

Windows smoke must use the real native Save dialog and prove one successful new local
`.yorva-backup.age` publication plus cancellation, existing-file, reparse/unsafe target,
expiry and replay failure without recording the selected username or path in evidence.

## Alternatives considered

### Return the picker path to React and POST it with the local bearer

Rejected. The bearer is intentionally available to React and cannot attest that a path
came from the native picker.

### Add a second privileged HTTP token for Tauri

Rejected for the first implementation. It adds another bootstrap secret and a Rust HTTP
client solely to recreate the private inherited channel already owned by the parent and
child. It may be reconsidered only if the bounded stdio acknowledgement design proves
unworkable.

### Persist grants in SQLite or a protected file

Rejected. Grants are short-lived capabilities, not durable product state. Persistence
increases replay and same-user tampering exposure and creates crash-recovery semantics
that B6 does not need.

### Encode or sign the path inside `destinationRef`

Rejected. The path would cross React/HTTP and remain replayable. A memory-only opaque
reference keeps path authority solely in the daemon.

## Consequences

Positive:

- neither React nor a future remote HTTP caller can choose an arbitrary backup path;
- no additional long-lived native secret or generic filesystem API is introduced;
- expiry, one-time use, session/user/Runtime binding and Operation ownership are explicit;
- the selected path remains inside the narrow native-shell/daemon trust boundary.

Costs and constraints:

- the existing one-record shutdown pipe becomes a small bidirectional control protocol;
- Tauri must track one bounded acknowledgement without treating daemon stdout as general
  output;
- native dialog and Windows path/owner/reparse behavior require real Desktop smoke;
- this decision authorizes only Backup Create destination selection and does not unblock
  Backup source enumeration, device SecretStore, Restore or the complete B6/B7 gates.

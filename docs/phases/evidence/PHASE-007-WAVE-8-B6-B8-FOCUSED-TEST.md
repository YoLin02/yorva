# Phase 7 Wave 8 — B6/B8 Focused Test

> Scope: B6 authenticated backup-container primitive and B8 planning/mutation
> qualification truth
> Branch: `codex/phase7-hermes-runtime-management`
> Base commit: `2e9aa80`
> Full B10 independent audit: **NOT STARTED**

This is a lane-focused development Gate. It is not the Phase 7 candidate audit and
does not authorize a production Backup, Restore, Upgrade or Rollback capability.

## Initial focused Gate — FAIL

Date: 2026-08-25

Result: **FAIL — P0 0 / P1 0 / P2 2**

The original result is retained. The findings were lane-local because all applicable
Hermes Bundle entries were unwired, the HTTP mux exposed no Backup/Restore/Upgrade
mutation route, and `EncryptionQualified` and `RestoreMutationQualified` remained
`false`.

### P2-01 — encrypted output was not bound to the verified byte snapshot

The initial B6 primitive verified one view of the plaintext, then hashed and encrypted
later reads. It compared only the later hash with the encrypted stream. A mutable
`io.ReaderAt` could therefore change after verification and keep the new contents stable:
the unverified later contents would be encrypted while the function returned verification
metadata from the earlier contents.

Required correction: the successful result must verify the complete authenticated
plaintext represented by the final ciphertext, rather than infer equivalence from
separate reads of a mutable source.

### P2-02 — age header parsing lacked an independent pre-authentication bound

The initial B6 primitive bounded the full encrypted artifact but passed it directly to
the age header parser. A malicious local artifact could use an oversized opening line or
excessive recipient stanzas to consume disproportionate memory or CPU before any payload
authentication.

Required correction: impose a small independent header and recipient-stanza bound before
the age parser and make the parser consume the same preflighted header snapshot.

## Checks that passed in the initial Gate

From `services/node`:

```text
go test -count=1 ./internal/runtime/hermes/backupmanagement ./internal/runtime/hermes/upgrademanagement ./internal/runtime ./internal/app ./internal/transport/httpapi
go vet ./internal/runtime/hermes/backupmanagement ./internal/runtime/hermes/upgrademanagement ./internal/runtime ./internal/app ./internal/transport/httpapi
go mod verify
go mod tidy -diff
go test -count=5 ./internal/runtime/hermes/backupmanagement
govulncheck ./internal/runtime/hermes/backupmanagement
```

Results:

- all five focused Go test packages passed;
- focused vet passed;
- all modules verified and `go mod tidy -diff` reported no changes;
- the repeated age package tests passed;
- focused vulnerability analysis found no reachable vulnerability;
- B8 exposed separate planning-evidence, Upgrade-mutation and Rollback-mutation
  qualification facts, with no ambiguous single `executable` field;
- production Backup/Restore/Upgrade/Rollback wiring and routes remained closed.

## R1 re-review

Date: 2026-08-25

Result: **PASS — P0 0 / P1 0 / P2 0**

The R1 review ran in a new test-agent turn after the corrective patch. The initial FAIL
and both original P2 findings above remain part of the historical record.

### Resolution of P2-01

Encryption now requires caller-owned private `CiphertextStaging` with bounded
`ReaderAt`, `WriterAt` and `Truncate` behavior. The primitive clears staging, writes the
age ciphertext from offset zero, then reads that exact final ciphertext back through
decryption, complete plaintext-chunk authentication and `VerifyArtifact`. Only metadata
derived from that final ciphertext can be returned as successful. Errors and cancellation
return an empty result and make a best-effort staging truncate; the caller remains
explicitly responsible for protecting and discarding failed staging.

### Resolution of P2-02

Before the age parser runs, the adapter limits the header to 4 KiB per line, 64 KiB in
total, 128 lines and exactly one recipient stanza of the expected type. The checked
header bytes are cloned into a frozen `ReaderAt`, so the age parser cannot re-read a
changed header from the mutable backing source. A declared ciphertext beyond the outer
bound is rejected before the source is read.

### R1 commands and results

From `services/node`:

```text
go test -count=1 ./internal/runtime/hermes/backupmanagement ./internal/runtime/hermes/upgrademanagement ./internal/runtime ./internal/app ./internal/transport/httpapi
go vet ./internal/runtime/hermes/backupmanagement ./internal/runtime/hermes/upgrademanagement ./internal/runtime ./internal/app ./internal/transport/httpapi
go mod verify
go mod tidy -diff
git diff --check
go test -count=10 ./internal/runtime/hermes/backupmanagement -run "TestAge(CreateFailsClosedWhenFinalCiphertextChangesBeforeVerification|HeaderPreflightRejectsResourceAbuse|DeclaredCiphertextLimitRejectsBeforeRead|HeaderPreflightFreezesBytesForAgeParser|CancellationAndLimitsFailClosed)$"
govulncheck ./internal/runtime/hermes/backupmanagement
```

All checks passed. Focused vulnerability analysis found no reachable vulnerability.
The R1 reviewer also reconfirmed:

- `EncryptionQualified` and `RestoreMutationQualified` remain `false`;
- Hermes production Bundle entries for Backup, Restore, Upgrade and Rollback remain nil;
- no Backup/Restore/Upgrade mutation route is registered;
- B8 keeps planning evidence separate from Upgrade and Rollback mutation qualification.

This PASS accepts the current safety foundations only. It does not authorize product
Backup publication, Restore, Upgrade or Rollback and it is not the Phase 7 B10 audit.

## B6 unconnected publication foundation

Date: 2026-08-25

The B6 adapter foundation now also contains an unconnected publisher for one already
constructed and structurally valid canonical plaintext container. It accepts only one
opaque locally inspected `.yorva-backup.age` destination and one request-scoped
in-memory X25519 identity or passphrase. It creates current-user-only private sibling
staging, checks destination space before staging, verifies the final persisted age
ciphertext, durably flushes it, publishes without overwrite, and re-hashes the final
file. Returned metadata contains no path, destination capability, identity or secret.

Focused coverage includes collision and publish-boundary collision, symlink/reparse and
ADS rejection, changed-parent detection, insufficient space before staging, late
cancellation, publication failure cleanup, atomic visibility, protected Windows DACL,
final checksum and decrypt verification. The focused package test, vet, diff check and
Linux cross-compile passed.

This is not a complete Runtime backup and does not change capability truth. Production
wiring remains blocked on separately owned boundaries: the native-picker destination
reference resolver, OS-backed device-identity `SecretStore`, Runtime-scope safe index
integration, and an exact qualified Hermes Runtime/Profile source enumerator. No fake
substitute is introduced here; `EncryptionQualified` remains false and no mutation route
or Bundle capability is registered.

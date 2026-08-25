# Phase 7 Wave 10 — B6 Manifest Completeness Focused Test

> Date: 2026-08-25  
> Scope: canonical decrypted Runtime-backup manifest and snapshot fixture migration only  
> Product capability: unchanged; Backup/Restore mutation remains unqualified and unwired

## Change boundary

This lane completed the ADR-0013 manifest metadata contract without adding an HTTP,
Desktop, Operation, persistence, SecretStore, publication or Restore connection.

Manifest schema 2 now requires:

- one closed opaque `backupId` and one canonical whole-second UTC `createdAt`;
- exact Runtime kind/version and explicit `MANAGED`, `UNMANAGED` or `UNKNOWN`
  installation state;
- for `MANAGED` only, a closed generation ID and exact lowercase 40-hex source pin;
- fixed inclusion/exclusion policy identities and their complete sorted closed category
  sets;
- the existing bounded payload size, SHA-256, member/expanded-byte facts and completion
  marker.

`UNMANAGED` and `UNKNOWN` prohibit invented generation/source values. The manifest has
no native path, Profile name, secret, account identifier or credential value. Schema 1
fixtures are rejected rather than silently interpreted as the complete schema 2 format.

## Focused verification

From `services/node`:

```text
go test ./internal/runtime/hermes/backupmanagement
ok github.com/YoLin02/yorva/services/node/internal/runtime/hermes/backupmanagement

go vet ./internal/runtime/hermes/backupmanagement
PASS (no output)
```

The focused tests cover exact canonical bytes, unknown/duplicate/trailing/alternate JSON,
unsupported predecessor schema, invalid or incomplete identity/policy/category sets,
explicit unmanaged/unknown states, payload tamper/checksum detection and propagation of
the complete safe metadata through structural verification and snapshot creation.

Repository diff whitespace validation:

```text
git diff --check
PASS (line-ending notices for pre-existing shared Go module edits only; no whitespace error)
```

## Gate truth

- `EncryptionQualified`: `false`
- `RestoreMutationQualified`: `false`
- no Backup or Restore product capability was enabled;
- no Windows destructive Restore smoke is claimed by this focused lane.


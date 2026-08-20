# ADR-0008: Hermes-Native Channel Credential Authority

- Status: Accepted
- Date: 2026-08-20
- Owner approval: 2026-08-20 (Phase 6 execution authorization and D3-D9 qualification lock)
- Related: ADR-0003, ADR-0006, ADR-0007, Phase 6 qualification evidence

## Context

Hermes gateways must continue using channel bindings after YORVA Desktop and `yorvad`
exit. Keeping another credential copy in YORVA would create two authorities and require
runtime secret injection. Hermes `0.20.2` already owns Profile-scoped Weixin account
files and WeCom environment configuration, while SQLite must remain free of plaintext
channel credentials.

## Decision

For Hermes `0.20.2`, Hermes-native Profile storage is the sole authority for Weixin and
WeCom credentials. YORVA stores only safe binding projections in SQLite.

Weixin credentials are written only to the exact Profile's canonical
`weixin/accounts/<validated-account-id>.json` representation after a verified iLink QR
confirmation. WeCom uses only the exact Profile's canonical `.env` keys
`WECOM_BOT_ID` and `WECOM_SECRET` after an authenticated check against the fixed official
WeCom WebSocket endpoint. The non-public WeCom admin-console QR endpoints are not
authorized.

Because Hermes exposes no qualified non-interactive credential setter, the Hermes
adapter may use narrowly bounded, version-pinned compatibility writers. They are not a
generic file or environment API. Callers cannot provide paths, filenames or environment
keys. The adapter validates the exact Instance/Profile, channel, account identifier and
supported Hermes version; applies strict file-size and response limits; writes a
same-directory temporary file; atomically replaces the target; and fails closed on
concurrent external modification or uncertain read-back.

The following rules are mandatory:

- no duplicate credential in YORVA SecretStore, SQLite, `secret_refs`, Operations,
  events, logs, diagnostics, audit metadata, Desktop storage, argv, URLs or ambient/global
  environment variables;
- no secret plaintext is returned by ordinary read or mutation APIs;
- QR payloads and credential-equivalent URLs are initiating-session-only, bounded,
  expiring memory values and are cleared at every terminal outcome;
- Profile A material is never read, changed or removed while targeting Profile B;
- disconnect removes only the target local Hermes binding and does not claim remote
  revocation unless a qualified platform surface performed it;
- unknown versions, layouts, response schemas, redirects or verification results fail
  closed;
- source and destination files are rejected when their safe-path and reparse-point checks
  fail.

This is the same explicit local-user at-rest tradeoff as ADR-0007: Hermes-native files may
not provide an OS credential vault. YORVA does not describe them as OS-secure and does not
broaden the exception beyond these exact channel fields and pinned storage contracts.

## Alternatives considered

### Store a second copy in YORVA SecretStore

Rejected because it creates synchronization and post-exit injection requirements.

### Use global user or system environment variables

Rejected because they break Profile isolation and mutate user-owned machine state.

### Use the pinned WeCom admin-console QR endpoints

Rejected because Hermes identifies them as non-public and they cannot be treated as a
stable platform contract.

### Import Hermes Python modules

Rejected by the Runtime boundary. YORVA implements only the approved fixed protocol and
bounded compatibility persistence inside the Hermes adapter.

## Consequences

Hermes can use bindings independently after YORVA exits, and YORVA has no second secret
authority. The cost is an intentionally version-coupled compatibility writer that must be
re-qualified for any Hermes version or storage-layout change.

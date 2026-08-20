# Phase 6 Batch 1 — Lifecycle and Channel Qualification

- Date: 2026-08-20
- Packaged Hermes version: `0.20.2`
- Packaged Hermes source commit: `df4b65147d7ddd74dd449f9067aabbca5aef0ec7`
- YORVA baseline: `phase-005a1-post-freeze-corrections-baseline` (`9957775`)
- Result: **QUALIFIED WITH EXPLICIT LIMITS**

## Lifecycle surface

The exact packaged source and the installed `0.20.2` launcher expose Profile-scoped
`gateway status`, `start`, `stop`, `restart` and `install` commands. Named Profiles use
the existing validated `--profile <nativeId>` prefix. Status is human-readable and has
no JSON mode. Registered Windows tasks emit detailed service/process signals, while a
Profile without a task emits the exact compact `Gateway is running` or `Gateway is not
running` signal. The Hermes adapter therefore uses exact-version fixtures for both forms
and fails closed on unknown, localized, conflicting, truncated or oversized output.

On Windows, Hermes owns the per-Profile Scheduled Task, Startup-folder compatibility
entry and detached gateway process. `gateway start` is idempotent when running, but may
prompt to install a missing login item. YORVA therefore must not invoke that interactive
branch. A manual start with no persistence uses the official closed invocation
`gateway install --no-start-on-login --start-now`; an already registered Profile may use
`gateway start` only after an authoritative preflight proves the login item exists.

Hermes detaches the gateway using `CREATE_BREAKAWAY_FROM_JOB`. The lifecycle command
runner must permit only this fixed Hermes-owned start/restart path to break away while
retaining ordinary bounded process-tree ownership for every other command. Success still
requires an authoritative postcondition. Stop is bounded by Hermes' 30-second graceful
drain plus its 10-second forced-exit wait; YORVA uses a 55-second command timeout and a
15-second postcondition budget. Start uses a 35-second command timeout; Restart uses the
55-second stop/escalation budget, and both use a 15-second postcondition budget. Output
remains capped at 1 MiB.

`ON_LOGIN` is not qualified for Phase 6. Its official Windows path may prompt for UAC and
may fall back to Startup-folder persistence. Phase 6 therefore exposes manual lifecycle
only and has no startup-policy mutation endpoint.

Read-only installed-runtime evidence on 2026-08-20 confirmed version `0.20.2`, the
documented command shapes, a registered default task, and an authoritative stopped
process state. No gateway was started or stopped during qualification.

## Messaging surface

The installed runtime already contains `aiohttp`, `cryptography`, `httpx` and `qrcode`.
Lifecycle and the qualified authentication code therefore require no mutation of the
active installation. Any future missing dependency must be materialized through a new
ADR-0006 generation; sealed generations are never repaired in place.

Pinned Weixin uses the fixed Tencent iLink QR endpoints, bounded long polling and a
Hermes-owned per-Profile account file. The returned `redirect_host` is not validated
strongly enough for YORVA's boundary. A YORVA implementation must allow only HTTPS and a
fixed reviewed iLink host allowlist, enforce response limits, and keep QR payloads only in
bounded memory for the initiating authenticated session.

Pinned WeCom uses non-public admin-console QR endpoints. That compatibility path is
rejected. Phase 6 instead uses a typed manual Bot ID/Secret flow and verifies the binding
through the official fixed WeCom WebSocket endpoint before reporting `CONNECTED`.

Hermes-native Profile storage is the sole credential authority, as specified by
ADR-0008. SQLite stores safe projections only. `CONNECTED` means the credential/binding
was verified and remains independent from lifecycle `RUNNING`.

## Owner decisions recorded from Phase 6 authorization

- D3: accepted — lifecycle targets the selected Profile gateway.
- D4: rejected for Phase 6 — manual lifecycle only; `ON_LOGIN` deferred.
- D5: accepted through ADR-0008 — Hermes-native, per-Profile sole authority.
- D6: non-public WeCom QR rejected; secure manual Bot ID/Secret accepted.
- D7: accepted — QR retrieval is Operation-scoped and initiating-session-only.
- D8: accepted — existing qualified bytes may be used; missing bytes require a new sealed generation.
- D9: accepted — `CONNECTED` and `RUNNING` are independent states.

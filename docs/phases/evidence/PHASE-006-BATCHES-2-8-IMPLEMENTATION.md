# Phase 6 Batches 2–8 — Implementation Candidate

- Date: 2026-08-20
- Lifecycle implementation: `2e03a78`
- Lifecycle Desktop: `acf4139`
- Channel application/adapter/API: `28b6f0f`
- Channel Desktop: `b415f79`
- Result: **IMPLEMENTATION COMPLETE; AUDIT AND OWNER-AUTHENTICATED SMOKE PENDING**

## Delivered behavior

- Runtime-neutral Instance lifecycle status and durable Start/Stop/Restart Operations.
- Profile-exact Hermes `0.20.2` lifecycle adapter and localized Desktop controls.
- Runtime-neutral `ChannelManager`, safe `channel_bindings` projections and durable
  connect/disconnect Operations.
- Hermes-native sole credential authority for exact-Profile Weixin and WeCom bindings.
- Fixed-host, bounded Weixin iLink QR polling with an 8 KiB initiating-session-only
  in-memory QR broker and metadata-only `channel.qr.ready` SSE notifications.
- Fixed official WeCom WSS verification before Bot ID/Secret commit; no non-public WeCom
  QR fallback.
- Localized Instance Channel cards, expiring QR modal, manual WeCom form, retry/cancel,
  targeted disconnect and explicit remote-revocation disclosure.
- Channel `CONNECTED` and gateway `RUNNING` are presented as independent states.

## Verification completed

- `pnpm api:lint` and generated OpenAPI types passed.
- Desktop typecheck and lint passed.
- Desktop suite passed: 20 files, 89 tests.
- Desktop production build passed.
- `go test ./...`, `go vet ./...` and `go build ./cmd/yorvad` passed.
- Tauri release `--no-bundle` build passed after rebuilding the Go sidecar.
- The built Windows application was launched. A read-only visual check confirmed the
  Instance Channel entry, localized Weixin/WeCom cards, gateway/channel distinction,
  safe identity projection, manual WeCom fields and revocation disclosure.
- Migration tests cover empty/current migration, binding uniqueness, closed state/type,
  cascade deletion and absence of credential columns.
- Regression tests cover Profile isolation, targeted disconnect, fixed Weixin host and
  response bounds, WeCom verify-before-commit, WebSocket frame bounds, lifecycle/channel
  conflict, QR owner-session isolation, cancellation clearing, no-store delivery and
  metadata-only events.
- The active sealed Hermes generation was not modified. No real Channel authentication
  was initiated during automated or visual verification.

## Remaining gate evidence

The implementation candidate is ready for the Phase 6 audit handoff, but Phase 6 is not
yet frozen. The following owner-authenticated evidence cannot be fabricated or performed
without the Owner's account action:

- real Weixin scan, confirmation, connected-state and local disconnect smoke;
- real WeCom Bot ID/Secret verification, connected-state and local disconnect smoke;
- post-smoke inspection of SQLite/log surfaces for the actual credential/QR sentinels;
- an independent audit decision and accepted candidate tag.

Phase 7 implementation remains blocked until those items are complete and the Phase 6
gate is frozen.

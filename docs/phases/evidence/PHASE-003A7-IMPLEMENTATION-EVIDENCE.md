# Phase 3 Amendment 003A7 — Implementation Evidence

> Status: CANDIDATE — REMOTE GO/WINDOWS CI PENDING
> Date: 2026-08-21
> Branch: `codex/configurable-download-sources`
> Contract: `AMENDMENT-003A7-configurable-download-sources.md`
> Decision: `ADR-0010-configurable-hermes-download-sources.md`

## Implemented surface

- authenticated GET/PUT/DELETE Hermes download-source API;
- SQLite-backed namespaced non-secret setting with compiled defaults and reset;
- credential-free HTTPS validation and closed JSON decoding;
- start-time source snapshots for Hermes and Node/npm prerequisite Operations;
- packaged Hermes, Node.js and npm artifacts selected before network fallbacks;
- existing exact archive sizes/SHA-256 pins retained for every downloaded artifact;
- configurable Python/uv/pip and npm registry environment values after hostile inherited
  variables are stripped;
- Desktop Advanced settings UI with grouped Hermes artifact/dependency fields, bilingual
  copy, save/reset feedback and responsive layout;
- smaller centered default Tauri window (`1080x720`, minimum `860x560`);
- OpenAPI schema and generated TypeScript synchronized.

## Deterministic evidence

The following checks passed in the implementation workspace:

```text
redocly lint api/openapi.yaml
@yorva/desktop typecheck
@yorva/desktop lint
@yorva/desktop test — 22 files / 102 tests passed
@yorva/desktop build
git diff --check
```

The local workspace does not provide the repository's pinned Go or Rust toolchains.
The draft PR CI is therefore the authority for Go race tests/vet/vulnerability/build,
generated-contract cleanliness and Windows Rust/Tauri checks. This record must be
updated with the exact commit and workflow result before Amendment 003A7 can be frozen.

## Preserved blockers

This correction does not close Phase 6. Independent Phase 6 audit and real
owner-authenticated Weixin/WeCom Windows smoke remain pending and cannot be replaced by
the deterministic evidence above.

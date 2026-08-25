# Phase 7 B8 — Exact Upgrade and Rollback Compatibility Qualification

> Status: **MUTATION NO-GO — COMPATIBILITY UNKNOWN**
> Current source: Hermes `0.20.2` / `df4b65147d7ddd74dd449f9067aabbca5aef0ec7`
> Candidate source: Hermes `0.20.5` / `a0ca7c19204e514f9590ce3b812e029b315ab9e9`
> Date: 2026-08-25

## Result

The exact `0.20.2` to YORVA-carried `0.20.5` snapshot transition is not proven safe
for user data. Upgrade mutation remains NO-GO. The reverse transition is also UNKNOWN
and remains NO-GO. Keeping the previous sealed generation is necessary but does not
make data written or migrated by the new generation backward-compatible.

The candidate commit is not the official `v2026.8.19` tag commit. It contains later
main-branch changes, so release notes and version text cannot substitute for evidence
bound to the exact candidate archive. Official references:

- [`v2026.8.16` release](https://github.com/NousResearch/hermes-agent/releases/tag/v2026.8.16);
- [`v2026.8.19` release](https://github.com/NousResearch/hermes-agent/releases/tag/v2026.8.19);
- [exact `df4b651...a0ca7c1` comparison](https://github.com/NousResearch/hermes-agent/compare/df4b65147d7ddd74dd449f9067aabbca5aef0ec7...a0ca7c19204e514f9590ce3b812e029b315ab9e9);
- [`v2026.8.19` tag commit to exact candidate comparison](https://github.com/NousResearch/hermes-agent/compare/fcbd1076a93841fa88855acce810e342a5b78101...a0ca7c19204e514f9590ce3b812e029b315ab9e9).

## User-data findings

### Profile configuration

The configuration version advances from 37 to 38. The v38 migration removes legacy
observability and `nemo_relay` plugin enablement entries and has no down-migration that
can restore them. A Profile that used those entries has a concrete unsafe rollback
condition. The exact candidate implementation is in
[`config_migrations.py`](https://github.com/NousResearch/hermes-agent/blob/a0ca7c19204e514f9590ce3b812e029b315ab9e9/hermes_cli/config_migrations.py)
and
[`config_defaults.py`](https://github.com/NousResearch/hermes-agent/blob/a0ca7c19204e514f9590ce3b812e029b315ab9e9/hermes_cli/config_defaults.py).

### State and sessions

Both versions identify state schema 26, but the implementation has substantial semantic
changes around state metadata, FTS repair/admission and startup orphan-session writes.
The exact candidate can mark stale TUI, Desktop and subagent sessions as ended during
startup. No official exact-version fixture proves that 0.20.2 safely reads all resulting
state. Relevant candidate sources include
[`hermes_state_common.py`](https://github.com/NousResearch/hermes-agent/blob/a0ca7c19204e514f9590ce3b812e029b315ab9e9/hermes_state_common.py)
and
[`hermes_state_schema.py`](https://github.com/NousResearch/hermes-agent/blob/a0ca7c19204e514f9590ce3b812e029b315ab9e9/hermes_state_schema.py).

### MCP, Skills and Profile authority

The candidate moves the MCP dependency from 1.x to 2.0 and adds new token/client metadata
behavior. Cross-version round-trip evidence for those files is absent. Skill directories
remain recognizable, but the candidate adds curator ledger, backup and quarantine state
without an official backward-read fixture. Profile and gateway fixes also change
cross-Profile authentication/session routing behavior; a readable file format alone does
not prove a safe rollback to the earlier behavior.

The candidate dependency identity is recorded in
[`pyproject.toml`](https://github.com/NousResearch/hermes-agent/blob/a0ca7c19204e514f9590ce3b812e029b315ab9e9/pyproject.toml).

### Areas with no proven breaking migration

No automatic `.env`, primary channel-credential, `MEMORY.md`, `USER.md` or holographic
memory schema migration was identified. This is insufficient to mark those categories
compatible because exact Windows forward-write and backward-read evidence is still absent.

## Existing YORVA foundations

YORVA already has useful primitives for an eventual managed generation transition:

- exact candidate source pin and archive digest;
- a fresh final-path generation;
- launcher/version validation;
- complete manifest and generation seal;
- `active.json` generation/seal compare-and-swap;
- pointer read-back and sealed-tree verification;
- a shared fixed Hermes home and retained previous generation.

These foundations do not yet provide the B8 product postcondition. The production flow is
still install-oriented; it lacks a protection point, exact data compatibility record,
feature-aware postcheck of Profiles/models/lifecycle/channels/Skills/MCP/health-security,
linked rollback transaction and recovery state. Production Upgrade/Rollback Bundle entries,
routes and capabilities correctly remain closed.

## Minimum next Gate

Use only a disposable Windows copy of a complete Hermes home:

1. Bind both generations to exact commits, archive sizes and SHA-256 digests.
2. Build default and named Profile fixtures covering config v37, legacy relay settings,
   `.env`, auth state, sessions/FTS, Skills, MCP OAuth files, channels and memory.
3. Prove build, seal and pre-activation probes do not write the shared data tree.
4. Run the exact candidate, trigger real lazy/first-start writes and complete all
   authoritative feature read-backs.
5. Run 0.20.2 against a copy of the post-write data and assert every category, including
   legacy relay, MCP 1.x data and session/FTS behavior.
6. Test rollback separately as binary-only downgrade and as protection-point restore
   followed by generation CAS. Only the fully restored path may become eligible.
7. Exercise CAS conflict, crash at each state, cancellation before/after activation,
   external data drift and failed/ineligible rollback.

If any user-data category remains unknown or unenumerated,
`UserDataInventoryComplete` remains false and Upgrade/Rollback mutation capability remains
false. No real installation, user data, account or credential was mutated for this
qualification.

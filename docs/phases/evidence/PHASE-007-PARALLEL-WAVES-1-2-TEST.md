# Phase 7 Parallel Waves 1–2 — Independent Focused Test Evidence

## Role, candidate and result

- Role: independent parallel-development test agent; not an implementation agent and not
  the B10 Gate auditor.
- Date: 2026-08-25 (Asia/Shanghai).
- Worktree: `D:\workcode\myproject-phase65-freeze` only.
- Branch: `codex/phase7-hermes-runtime-management`.
- Inspected HEAD: `e7b7a63a1a6fff8c63b0f35a2edad2aceedf7bc6`.
- Included uncommitted lane directories at test time:
  `mcpmanagement`, `backupmanagement`, and `upgrademanagement`.
- Scope: B2 Runtime contracts/actions, B3 `managementhealth`, B4
  `skillsmanagement`, B5 `mcpmanagement`, B6 backup structural core, and B8
  upgrade/rollback policy core.

**Focused result after one fix verification: TESTS/VET PASS; ALL THREE ORIGINAL
P2 FINDINGS RESOLVED; PRODUCT CAPABILITIES REMAIN CLOSED.**

There is no P0/P1 stop-condition finding in the tested tree. The three original
lane-local findings are preserved below and marked `RESOLVED` after focused regression
verification. This is not a Phase 7 audit or an authorization to expose B3–B8 product
surfaces.

## Commands and results

From `services/node`:

```text
go test ./internal/runtime ./internal/app \
  ./internal/runtime/hermes/managementhealth \
  ./internal/runtime/hermes/skillsmanagement \
  ./internal/runtime/hermes/mcpmanagement \
  ./internal/runtime/hermes/backupmanagement \
  ./internal/runtime/hermes/upgrademanagement -count=1

PASS
runtime:           0.012s
app:              17.124s
managementhealth:  0.022s
skillsmanagement:  0.016s
mcpmanagement:     0.015s
backupmanagement:  0.016s
upgrademanagement: 0.013s
```

```text
go vet ./internal/runtime ./internal/app \
  ./internal/runtime/hermes/managementhealth \
  ./internal/runtime/hermes/skillsmanagement \
  ./internal/runtime/hermes/mcpmanagement \
  ./internal/runtime/hermes/backupmanagement \
  ./internal/runtime/hermes/upgrademanagement

PASS (no output)
```

Race was attempted, not inferred:

```text
go env GOOS GOARCH CGO_ENABLED
windows
amd64
0

go test -race ./internal/runtime/hermes/upgrademanagement -count=1
go: -race requires cgo; enable cgo by setting CGO_ENABLED=1
```

Therefore Race is **NOT RUN / environment unavailable**, not PASS. Whole-project CI,
Desktop tests, destructive smoke, network flows, and mutation flows were intentionally
not repeated in this focused wave test.

## Confirmed boundaries

- The actual Hermes registration in `internal/runtime/hermes/hermes.go` wires none of
  the new management interfaces. `Bundle.ManagementCapabilities()` consequently leaves
  every new capability false; merely registering Hermes does not advertise a lane.
- B2 requests are typed and reject caller URL/path/command-shaped identifiers. The MCP
  contract exposes no command, argv, environment, header, URL, package, bootstrap, or
  local-path field. Backup destination input is an opaque local capability reference.
- B3 exact `0.20.5` fixtures pass. Malformed/duplicate/oversized health and audit JSON,
  bounded log line/count/total input, secret/path/account canaries, terminal controls,
  bidi controls, and markup are rejected, redacted, sanitized, or truncated by the
  focused tests. Log source acquisition remains explicitly unqualified.
- B4's checked `0.20.5` fixture maps enabled inventory to the shared Runtime Skill type
  without inventing source, version, update, or scan facts. All Skill mutations and the
  reviewed-source registry remain unavailable; `0.20.6` and unknown mutations fail
  closed.
- B5's reviewed MCP registry is empty, OAuth is false, credential canary bytes are not
  retained in `Selection` or returned in errors, caller override fields are rejected,
  and `CONFIGURED` remains distinct from timestamped, expiring `READY`.
- B6 rejects the tested truncated/tampered/checksum, traversal, absolute, UNC, ADS,
  device, symlink, named-pipe, reparse, duplicate, case-collision, member/size/total and
  compression-bomb inputs. It performs no extraction and keeps encryption and Restore
  mutation qualification false.
- B8 requires managed identity, valid active pointer/seal, exact compiled/observed target,
  complete inventory, verified protection point, exact from/to compatibility, qualified
  postchecks, and safe rollback evidence. Unknown evidence does not become executable;
  mutation and rollback-mutation qualification remain false.

## Findings

### P2 — RESOLVED — Health parsers did not enforce the exact qualified Runtime version

Files: `internal/runtime/hermes/managementhealth/health.go:124` and `:138`.

`ParseLiveness` and `ParseDetailedHealth` accept the broad `safeVersion` regular
expression, including `0.20.6`, `1.0.0`, and `dev`. B1 qualified the response shape only
for exact Hermes `0.20.5`, and P7-D8 requires unknown exact-version contracts to remain
closed.

Minimal reproduction: replace `"version":"0.20.5"` in either accepted fixture with
`"version":"0.20.6"`; the parser still succeeds. This does not currently open a product
path because Hermes wires no Health contract, but version qualification must be enforced
at this parser/adapter boundary before wiring.

Resolution verification: `safeVersion` was removed, both parsers now compare with the
exact `qualifiedHermesVersion = "0.20.5"`, and the regression test accepts `0.20.5`
while rejecting `0.20.6`, `0.20.5-dev`, empty and overlong values for both liveness and
detailed health.

### P2 — RESOLVED — Enabled Skill inventory over-restricted values Hermes 0.20.5 can emit

Files: `internal/runtime/hermes/skillsmanagement/inventory.go:185-207` and exact upstream
`tools/skills_tool.py` in the qualified `a0ca7c1…` archive.

The parser accepts only lowercase `[a-z0-9_-]` Skill names/categories and rejects a
decoded U+FFFD. The exact upstream inventory truncates `frontmatter.name` to 64
characters but does not enforce that lowercase grammar, derives category from directory
names without that grammar, and reads Skill files with `errors="replace"`. Consequently
a legitimate `0.20.5` response such as a core-valid `"name":"My-Skill"` is rejected,
and a replacement character created by upstream decoding is also rejected.

Minimal reproduction: pass
`{"object":"list","data":[{"name":"My-Skill","description":"d","category":null}]}`
to `ParseEnabledInventory`; it returns `ErrInventoryContractMismatch`, although the
qualified upstream code can emit that value and B2's shared Skill ID accepts it. Keep
path/control bounds, but align the projection grammar with the actual qualified source
instead of rejecting legitimate inventory wholesale.

Resolution verification: the parser now accepts core-compatible mixed-case/punctuation
identities such as `My-Skill` and `My.Skill_2`, preserves upstream U+FFFD replacement
text, and retains case-sensitive identity handling. Regression cases continue to reject
whitespace/control-bearing names, slash/backslash path shapes, `.`/`..`, NUL, and unsafe
category separators/whitespace/control characters.

### P2 — RESOLVED — Backup structural verification accepted Windows-invalid member names

File: `internal/runtime/hermes/backupmanagement/verify.go:277-296`.

`safeMemberName` rejects traversal, separators, ADS, reserved devices, trailing dot/space
and controls, but it does not reject Windows-invalid filename characters `"`, `*`, `<`,
`>`, `?`, and `|`. A payload member such as `hermes-runtime/a?.txt` can therefore reach
`STRUCTURE_VERIFIED_ONLY`. A later Windows extractor cannot create that exact member,
which undermines all-members-first destination proof and can turn a preflight PASS into
a mid-Restore failure.

Minimal reproduction: add `hermes-runtime/a?.txt` as a regular stored member to the
existing normal artifact helper and recompute the manifest; `VerifyArtifactBytes`
succeeds. Restore remains false today, so this is a lane-local pre-wiring defect rather
than an active destructive path.

Resolution verification: `safeMemberName` now rejects every Windows-invalid filename
character in `< > : " \\ | ? *`; focused artifact tests cover `a?.txt`, `a*.txt`, quote,
pipe, angle brackets and drive-shaped input. A positive regression verifies that Unicode
components and interior spaces such as `资料/工作 文件.txt` remain accepted, avoiding an
over-defensive ASCII-only restriction.

## One-time fix verification — 2026-08-25

From `services/node`:

```text
go test ./internal/runtime/hermes/managementhealth \
  ./internal/runtime/hermes/skillsmanagement \
  ./internal/runtime/hermes/mcpmanagement \
  ./internal/runtime/hermes/backupmanagement -count=1

PASS
managementhealth: 0.018s
skillsmanagement: 0.012s
mcpmanagement:    0.011s
backupmanagement: 0.011s
```

```text
go vet ./internal/runtime/hermes/managementhealth \
  ./internal/runtime/hermes/skillsmanagement \
  ./internal/runtime/hermes/mcpmanagement \
  ./internal/runtime/hermes/backupmanagement

PASS (no output)
```

The MCP normalized probe parser now performs a bounded recursive token scan before its
typed decoder. Focused regression tests prove both a duplicate top-level `profileId` and
a duplicate key inside a nested object return `ErrProbeMalformed`; unknown-field
rejection and the existing byte/value/depth budgets remain in force. The earlier MCP
duplicate-key pre-wiring gate is therefore **RESOLVED**.

## Remaining integration gates (not defects by themselves)

- Only B4 currently has an explicit projection into the shared B2 Runtime model
  (`EnabledSkill.RuntimeSkill`). B3 health/audit/log, B5 MCP readiness/probe, B6 artifact,
  and B8 plan/rollback use lane-local types with no checked shared-state mapping yet.
  Their capabilities correctly remain false; add explicit exhaustive mappings and tests
  before registry wiring so `UNKNOWN`, `CONFIGURED/READY`, structural-only verification,
  and mutation qualification cannot be lost.
- MCP top-level and nested duplicate-key rejection is now verified. The reviewed
  registry still remains empty, as required while no descriptor is approved.
- At the time of this snapshot, ADR-0013 and ADR-0015 remained Proposed/Pending.
  Encryption, Restore mutation, Upgrade mutation and Rollback mutation remained false
  pending applicable Owner acceptance and later qualification evidence. MCP presets/OAuth
  remained governed by the approved P7-D3 boundary and provider-specific qualification.

## Files and state changed by this test

Only this evidence file was created. No product source, test source, dependency, Runtime
state, backup, Restore, upgrade, Git commit, push, reset, stash, deletion, or original
worktree content was changed.

# Phase 7 Batch 1 — Independent Qualification Test Matrix

## Role and result

- Role: independent B1 verification agent; not the surface implementer or Gate auditor.
- Date: 2026-08-25 (Asia/Shanghai).
- Repository branch: `codex/phase7-hermes-runtime-management`.
- Inspected branch commit: `30f322529339f17bbd1f337d0e9ee8afbefb0b67`.
- Required predecessor: `phase-0065-developer-led-demo-baseline` / `5f68e48f17e7e342e1781b37613b19d4bd1f060b`.
- Product code, API, migration and Runtime mutation: not authorized and not performed.

**Independent result: B1 NOT READY / STOP.**

The exact source archive is provenance-qualified, and the existing YORVA compatibility
tests accept stable Hermes `0.20.x`. That is not enough to qualify the proposed P7
capabilities. The direct Hermes `0.20.5` surfaces still fail mandatory B1 properties in
five areas: read-only health/log safety, Skill update scan enforcement, closed MCP and
credential authority, encrypted backup/Restore safety, and immutable-generation upgrade.

This is a qualification result, not an audit severity assignment against shipped YORVA
product code. P7 has not exposed these surfaces yet.

## Evidence identity and limitations

### Exact packaged reference

The inspected archive was:

```text
.cache/hermes-source/hermes-agent-a0ca7c19204e514f9590ce3b812e029b315ab9e9.zip
size:   73,798,347 bytes
SHA256: 4E63F39BD992B79BF364A4E22204A22B8E520E2365B53C909F6F15D42C917C54
root:   hermes-agent-a0ca7c19204e514f9590ce3b812e029b315ab9e9/
```

The archive's `hermes_cli/__init__.py` and `pyproject.toml` both declare `0.20.5`.
All source conclusions below come from static inspection of this exact verified archive;
Hermes internal modules were not imported.

### Installed launcher is not exact-candidate evidence

A permitted version/help probe of the locally installed launcher returned:

```text
Hermes Agent v0.20.5 (2026.8.19) · upstream dc50f020
```

This is the right semantic version but a different upstream revision from the B1
reference `a0ca7c1…`. Its successful help output therefore proves neither exact-candidate
behavior nor candidate postconditions. No further installed-Runtime surface command was
run.

Static inspection performed after that probe also showed that ordinary CLI startup may
write logs, repair Windows launchers, remove quarantined launcher files, sweep stale
bytecode and recover an interrupted install before subcommand dispatch. Consequently the
help probe is not represented as a zero-side-effect test, and the installed launcher is
excluded from all PASS decisions below.

### Safe automated verification executed

From `services/node`:

```text
go test ./internal/runtime/hermes \
  -run 'TestDetectorOutcomes|TestDetectorPropagatesCallerCancellation|TestParseVersionBanner|TestSupportedVersionStringUsesTheSameCompatibilityPolicy|TestCommandRunnerExecutesOnlyVersionArgument' \
  -count=1

result: PASS
package: github.com/YoLin02/yorva/services/node/internal/runtime/hermes
duration reported by go test: 0.237s
```

These tests cover existing detection/version/closed-command behavior only. They do not
qualify any new P7 surface.

### Static source anchors inside the exact archive

Line numbers below are archive-member line numbers at `a0ca7c1…`:

| Evidence | Exact archive member and lines |
| --- | --- |
| Common Profile pre-parser can warn and fall back after an unexpected resolver exception | `hermes_cli/main.py:520-694` |
| Windows launcher repair and logging are initialized before command dispatch | `hermes_cli/main.py:695-806` |
| Command-dispatch startup also performs stale-bytecode/install recovery | `hermes_cli/main.py:12668-12689` |
| Status is human output and can print paths/auth/runtime detail and run deep checks | `hermes_cli/status.py:132-724` |
| Logs have no line-value/byte bound or second read-time redaction | `hermes_cli/logs.py:145-363` |
| Security audit is JSON-capable but reads OSV responses without a byte cap | `hermes_cli/security_audit.py:285-589` |
| Skill update calls `do_install(... force=True)` | `hermes_cli/skills_hub.py:1108-1167` |
| Skill install passes `force` into `should_allow_install` | `hermes_cli/skills_hub.py:696-733` |
| MCP parser accepts URL, command, argv remainder and env | `hermes_cli/subcommands/mcp.py:35-78` |
| MCP catalog bootstrap executes manifest strings through a shell | `hermes_cli/mcp_catalog.py:445-520` |
| Backup/import include secret-bearing files and incrementally restore ZIP members | `hermes_cli/backup.py:642-864`, `870-1279`, `1286-1526` |
| Update check clears Git locks and fetches remotes without a subprocess timeout | `hermes_cli/update_cmd.py:3252-3395` |

## Result legend

| Result | Meaning |
| --- | --- |
| `PASS` | Direct evidence satisfies the B1 property for the exact candidate. |
| `CONDITIONAL` | A useful upstream primitive exists, but B1 still needs a closed YORVA boundary and tests before capability can be true. |
| `FAIL` | The direct surface contradicts a mandatory P7 property or stop condition. |
| `NOT RUN` | Safe exact-candidate execution was unavailable or the action would mutate state. No inference of PASS is made. |

## Cross-cutting CLI envelope

| Property | Evidence | Result |
| --- | --- | --- |
| Exact source | Archive size/SHA/root and embedded version match the frozen packaged reference. | `PASS` |
| Exact installed executable | Installed `0.20.5` self-reports upstream `dc50f020`, not `a0ca7c1…`. No built executable tied to the exact archive was available for safe execution. | `NOT RUN` |
| Non-interactive parser | Many read commands parse without a prompt, but Skills config, MCP add/catalog credentials/configure/login, Restore without force and other mutations prompt or require a TTY. | `FAIL` as a general envelope |
| Profile scope | `-p/--profile` is pre-parsed into `HERMES_HOME`. Missing/invalid profiles fail, but an unexpected resolver exception warns and returns to the default instead of failing closed. A named-profile `HERMES_HOME` is trusted only when its parent is `profiles`. | `CONDITIONAL`; a YORVA-owned exact Profile descriptor and post-read-back are mandatory |
| Read-only invocation | Before command dispatch, `main.py` can repair Windows launchers, set up file logging, clear stale bytecode/install state, and run interrupted-install recovery. `update --check` additionally mutates Git refs/locks. | `FAIL` for a strictly read-only direct CLI contract |
| Output bound | Most commands print human text without a byte schema. The upstream CLI does not impose the P7 line/byte/object limits. | `FAIL`; an owned process/output cap would still need parser fixtures |
| Timeout/cancel | Individual code paths have scattered timeouts, but there is no common overall deadline/cancellation contract. Some commands contain polling, multiple sequential network calls or child processes. | `FAIL`; external process-tree ownership is required and unproven for P7 |
| Secret transport | CLI startup loads Profile `.env`; several MCP flows prompt and persist secrets. Output is sometimes masked, but there is no single P7 credential authority for MCP/backup/update. | `FAIL` pending accepted ADRs |
| Postcondition | Human success text or exit zero is common. Several commands have no structured authoritative read-back. | `FAIL` as a general envelope |

## Surface verification matrix

### Health, status, logs and security

| Surface | Non-interactive / scope | Output and failure behavior | Bounds, secrets and postcondition | Result |
| --- | --- | --- | --- | --- |
| `hermes status` | No prompt; `-p` can select a Profile. `--deep` performs network work. Even plain status probes some configured local providers. | Human-only. No `--json`. Prints project path, `.env` presence, auth-file paths, endpoint/host/user details, PIDs, Profile/session counts and Runtime-specific labels. | No response schema or total byte bound; output is not suitable for a safe normalized API. Exit zero does not establish a normalized health postcondition. | `FAIL` as direct P7 health surface |
| `hermes doctor` | Plain invocation is non-interactive, but runs multiple local subprocess/status probes. `--live` makes network calls; `--fix` and `--ack` mutate state. | Human-only. No JSON result. The handler does not expose a stable typed issue result or use issue count as a reliable process postcondition. | Large, configuration-dependent output; path/tool/account metadata may be present. Static/deep/live semantics are not returned as a closed schema. | `FAIL` as direct P7 health surface |
| `hermes monitoring status` | Non-interactive and Profile-aware through the common selector. | Human-only; reports monitoring configuration, not live Runtime/Instance health. Prints configured OTLP endpoint. | No typed health state, byte bound or safe endpoint projection. | `FAIL` for P7 health; not an equivalent surface |
| `hermes logs` | Non-interactive when `--follow` is absent. `log_name` is allowlisted, but `-n/--lines` accepts an unbounded integer and `--follow` is intentionally infinite. | Human/raw log text. Invalid log, time, level and component exit nonzero; no JSON. | Reads up to the whole file for files at most 1 MiB, may request arbitrarily many lines, has no per-line/total-byte cap, and performs no second read-time redaction. Headers expose the Profile log path. Raw messages may contain tokens, accounts, conversations or paths. | `FAIL`; mandatory log stop condition triggered |
| `hermes security audit --json` | Non-interactive, Profile-aware and explicitly networked. Exit `0` means no finding at threshold, `1` a threshold finding and `2` a handled audit error. | JSON keys are stable in source: component count, finding count and finding records. Malformed OSV JSON and other unexpected exceptions are not normalized by this command. | OSV responses and JSON summaries are read without byte bounds; number of components/findings is not capped. Upstream summaries are untrusted text. An external process deadline/output cap plus strict parser could make this useful, but common startup side effects remain. | `CONDITIONAL`; only candidate primitive not a B1 PASS |

Required later tests for any accepted health/security adapter: exact Profile A/B fixtures,
plain/deep/live separation, empty/partial/malformed JSON, oversized OSV response, threshold
exit `0/1/2`, child timeout/cancel/reap, output cap, path/account/token redaction and an
authoritative normalized `HEALTHY/DEGRADED/UNHEALTHY/UNKNOWN` mapping.

### Skills

| Surface | Non-interactive / scope | Output and failure behavior | Security/postcondition | Result |
| --- | --- | --- | --- | --- |
| `skills list` | Non-interactive; `-p` and `--enabled-only` provide useful Profile intent. | Human table only; no JSON inventory. | No closed parser or bounded authoritative inventory. | `CONDITIONAL` primitive, not qualified |
| `skills search --json` | Non-interactive and bounded only by caller-supplied `--limit`; performs registry network queries. | JSON exists for search results, but this is catalog search rather than installed inventory. | Query/result/server response bounds and source identity still require a closed adapter. | `CONDITIONAL` for approved catalog discovery only |
| `skills inspect` / `skills audit` | Non-interactive; exact Profile follows the common selector. | Human Rich output/preview; no JSON. Audit has no stable result schema or exit-code contract for a blocked verdict. | Skill content/metadata is untrusted and output is not bounded for a Desktop/API projection. | `FAIL` as direct qualification/postcondition surface |
| `skills install <id> --yes` | Can skip confirmation. The CLI also accepts arbitrary HTTP(S) URL, category/name and explicit `--force`; YORVA must expose none of those caller-controlled forms. | Human output. Normal install quarantines and scans before move. | With `force=false`, blocked scans are rejected. There is no structured result/read-back and no proven closed approved-source mapping in YORVA yet. | `CONDITIONAL`; direct CLI shape cannot be exposed |
| `skills update [name]` | Non-interactive, Profile-scoped. | Human output. | **Blocking source fact:** `do_update` invokes `do_install(... force=True)` even when the user did not pass CLI `--force`. `do_install` passes that same value to `should_allow_install`, so the update replacement path can bypass a blocked scan verdict. This contradicts P7-D2. | `FAIL`; mandatory stop condition triggered |
| `skills config` | Handler explicitly calls `_require_tty`; there is no non-interactive closed enable/disable schema. | Interactive picker only. | Cannot be used by a daemon Operation. | `FAIL` |
| `skills uninstall <name> --yes` | Non-interactive form exists and can be Profile-scoped. | Human output only. | No structured postcondition/rollback; safe identity/read-back remains unproven. | `CONDITIONAL`, not qualified |

Required before B4: an upstream-safe or separately approved update route that never
conflates replacement with scan bypass; closed approved source IDs; bounded structured
inventory/inspect/audit evidence; exact Profile fail-closed targeting; malicious source,
traversal/symlink, local-edit conflict, partial install/update, timeout/cancel and
authoritative read-back tests. Until the scan bypass is removed from the selected route,
Skill update capability must remain false.

### MCP

| Surface | Non-interactive / scope | Output and failure behavior | Security/postcondition | Result |
| --- | --- | --- | --- | --- |
| `mcp catalog` | Non-interactive and Profile-scoped. | Human table, no JSON. Includes configured custom entries alongside catalog entries. | Descriptions/transport text are not a closed safe projection. | `CONDITIONAL` for static review only |
| `mcp list` | Non-interactive and Profile-scoped. | Human table, no JSON. Prints URL or stdio command/argv fragments and status inferred from config. | `CONFIGURED` is not authoritative `READY`; URL/command text can contain sensitive or unsafe values. | `FAIL` as direct live inventory |
| `mcp add` | Interactive confirmations and credential prompts. Parser accepts caller-controlled URL, stdio command, remainder argv, env assignments, auth and timeout. | Human output; can save config even after probe failure. | Directly violates P7-D3's no arbitrary command/env/header surface and cannot provide daemon-safe secret transport. | `FAIL`; mandatory stop condition triggered |
| `mcp install <catalog-id>` | Identifier is catalog-scoped, but credential collection and tool selection can prompt. | Human output. Catalog manifests may clone Git and execute `install.bootstrap` strings with `subprocess.run(... shell=True)` and no common timeout/output containment. | Exact upstream catalog inclusion is not sufficient YORVA review. Secret values are prompted into Profile `.env`; no accepted MCP credential-authority ADR exists. | `FAIL` until reviewed adapter-owned descriptors and ADR |
| `mcp configure` | Explicitly requires an interactive terminal. | Interactive tool picker only. | No closed non-interactive tool-selection mutation. | `FAIL` |
| `mcp login` / `reauth` | Explicit browser OAuth; minimum connection timeout is 315 seconds. Not initiating-YORVA-session scoped. | Human output and token-file existence checks. | Clears/creates cached OAuth state; no demonstrated session isolation or unique authority. | `FAIL` pending ADR and session design |
| `mcp test` | Non-interactive for already configured server; per-server timeout is read from Runtime config and externally overridable only by internal calls. | Human output; prints server/tool descriptions and masked resolved headers. | Can spawn/connect to configured arbitrary servers; untrusted tool output is not bounded as a typed result. Cancellation/process cleanup is not established for a YORVA Operation. | `FAIL` as direct product surface |
| `mcp remove` | Prompts; no `--yes` parser flag. | Human result. | Removes config and OAuth state without a structured postcondition. | `FAIL` for non-interactive daemon use |

Required before B5: accepted credential-authority ADR; a finite reviewed preset set; no
manifest shell/bootstrap passthrough; no caller command/argv/env/header/path; write-only
credential transport; initiating-session OAuth isolation; structured inventory/test
result; fixed connect/tool/output bounds; child/network cancel and cleanup; external-change
reconciliation; and `CONFIGURED` versus timestamped `READY` tests.

### Backup and Restore

| Surface | Scope / interaction | Archive and failure behavior | Security/postcondition | Result |
| --- | --- | --- | --- | --- |
| `hermes backup` | Full backup uses the default Hermes root, not an Instance-only product scope. `--output` accepts an arbitrary path. | Plain ZIP of configuration, skills, sessions and data. No YORVA format version, checksum manifest or encryption. | Includes secret-bearing `.env`, auth/state and session/account data by design. | `FAIL`; plaintext-full-backup stop condition triggered |
| `hermes backup --quick` | Profile-aware snapshot location, no user prompt. | Copies critical state including `.env`, `auth.json`, databases and cron into an unencrypted snapshot directory. | Still plaintext secret-bearing state; not an acceptable P7 product backup. | `FAIL` |
| `hermes import <zip>` | Prompts when target exists; only `--force` makes overwrite non-interactive. Takes arbitrary local path. | Marker-only backup recognition. It checks path traversal but has no demonstrated format/checksum/version/member-count/expanded-byte/reparse/ADS policy. Members are restored incrementally; per-file errors are collected while final text can still say `Import complete`. | No pre-mutation full archive validation, protection point, transaction, conflict matrix, rollback or authoritative health post-check. | `FAIL`; Restore stop conditions triggered |

B6/B7 remain blocked until an accepted ADR defines Runtime scope, standard encryption,
OS-backed key authority, safe plaintext staging/cleanup, archive bounds, checksum and format
version, preflight, conflict/stop plan, protection point, atomic failure semantics,
reconciliation and rollback truth. No backup/import command was executed.

### Update, plan and managed upgrade

| Surface | Interaction / mutation | Output and scope | Result |
| --- | --- | --- | --- |
| `hermes update --check` | Despite its help text, it runs Git fetch, updates remote refs and calls stale Git-lock cleanup. It is not read-only and would mutate a source checkout/sealed generation. No subprocess timeout is supplied for its Git fetches. | Human output only; compares moving branches rather than YORVA's immutable packaged target. | `FAIL`; must not be used by managed P7 upgrade |
| `hermes update --plan` | Plan branch is intended read-only after common startup, but enumerates fleet processes/services. | Human output includes Profile names, PIDs, supervisor and running code identity; no JSON/bounds. It describes Hermes in-place update mechanics rather than YORVA's generation transaction target. | `FAIL` as authoritative YORVA plan; at most a development reference |
| `hermes update` | Mutates checkout/venv, can stash work and exposes `--force` / `--force-venv`. | Incompatible with ADR-0006/0009 sole-pointer and immutable active-generation model. | `FAIL` / forbidden route |
| YORVA managed upgrade/rollback | No B1 product implementation or accepted generation-upgrade ADR exists yet. | Must use the exact snapshot carried by YORVA, a new final-path generation, seal, functional validation, activation CAS, reconciliation and data-compatible rollback. | `NOT RUN`; blocked pending ADR and later batches |

Required before B8: accepted generation-upgrade ADR; exact packaged target identity;
managed-install-only preflight; new transaction/generation; no active-tree writes; final-path
dual-launcher validation; seal and activation CAS; gateway/Profile/model/channel/Skill/MCP
post-check; restart/cancel/recovery; prior-generation retention; and explicit user-data
compatibility proof before rollback.

## Normal, malformed, error, timeout and cancellation coverage status

No exact `a0ca7c1…` built launcher tied to the verified archive was available for safe
execution, and the mutating surfaces were prohibited. Therefore B1 runtime cases below
are deliberately not claimed:

| Case family | Exact-candidate result |
| --- | --- |
| Normal runtime output for every P7 surface | `NOT RUN` |
| Malformed/unknown runtime output | `NOT RUN`; no P7 parsers exist yet |
| Oversized output / single-line overflow | `NOT RUN`; upstream gaps identified statically |
| Nonzero exit / partial failure normalization | `NOT RUN`; no P7 stable error mapping exists |
| Overall timeout and cancellation | `NOT RUN`; no P7 Operation runner exists |
| Process tree/network cleanup | `NOT RUN` |
| Profile A/B authoritative read-back | `NOT RUN` |
| Secret-canary redaction | `NOT RUN` |
| Mutation postcondition and external reconciliation | `NOT RUN` |

These cases cannot be converted to PASS by the upstream tests merely existing in the
archive. They become executable acceptance tests only after B1 selects a safe surface and
B2/B3–B8 implement the corresponding adapter boundary.

## B1 stop conditions established by this matrix

1. Health/status/doctor/log direct output is not structured, bounded and safely redacted;
   the log surface can expose raw conversations, accounts, tokens and paths.
2. The direct Skill update route can bypass a blocked scan because replacement uses the
   same `force=True` value as the scan gate.
3. MCP direct surfaces expose generic URL/command/argv/env behavior, catalog bootstrap
   executes shell strings, and credential/session authority is unresolved.
4. Full and quick backup produce unencrypted secret-bearing state; import lacks the
   required pre-mutation archive and rollback contract.
5. Hermes update is an in-place checkout/venv mechanism. `--check` itself mutates Git;
   none of these routes can replace the required YORVA generation-upgrade design.
6. Exact-candidate runtime, Windows and destructive-flow evidence is not yet available.

Per the Phase Spec, these conditions block the affected batches and must be returned to
the Owner. B2 product-capability code must not begin by lowering the qualification bar or
by advertising unsupported capabilities. Useful conditional primitives may be retained
for a revised B1 design, but the current overall B1 Gate is not PASS.

## Files and state changed by this verification

Only this evidence file was created. No source, API, migration, dependency, Runtime data,
backup, import, login, install, update, commit or push was performed.

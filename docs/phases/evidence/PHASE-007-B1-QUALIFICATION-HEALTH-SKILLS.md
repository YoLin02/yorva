# Phase 7 B1 Qualification — Health, Logs, Security and Skills

## Result

**NO-GO / STOP — 2026-08-25**

This is a qualification result, not an implementation failure. The inspected Hermes
0.20.5 release has useful structured sub-surfaces, but it does not currently provide a
complete official surface that satisfies the Phase 7 contract for Runtime health/logs or
Skills lifecycle management. B2/B3/B4 work that depends on these capabilities must not
start from the surfaces rejected below.

The following narrow sub-surfaces are usable only under the stated conditions:

- `GET /health`: **GO only for API-server listener liveness**, not Runtime/Instance
  health;
- authenticated `GET /health/detailed`: **CONDITIONAL**;
- `hermes -p <profile> security audit --json`: **CONDITIONAL**;
- authenticated `GET /v1/skills`: **CONDITIONAL for enabled-skill inventory only**.

All other health/log/Skills lifecycle surfaces in this evidence are **NO-GO** for a YORVA
product adapter as presently exposed. No acceptance criterion was weakened to obtain a
PASS.

## Evidence identity and method

- YORVA branch: `codex/phase7-hermes-runtime-management`
- YORVA candidate inspected: `30f322529339f17bbd1f337d0e9ee8afbefb0b67`
- Accepted predecessor: `phase-0065-developer-led-demo-baseline`
- Hermes source commit: `a0ca7c19204e514f9590ce3b812e029b315ab9e9`
- Hermes version declared by both `pyproject.toml` and `hermes_cli/__init__.py`:
  `0.20.5`
- Source archive:
  `.cache/hermes-source/hermes-agent-a0ca7c19204e514f9590ce3b812e029b315ab9e9.zip`
- Archive SHA-256:
  `4E63F39BD992B79BF364A4E22204A22B8E520E2365B53C909F6F15D42C917C54`
- Archive size: `73,798,347` bytes
- Installed launcher resolved read-only to
  `C:\Users\Administrator\AppData\Local\hermes\bin\hermes.exe`.
- No `control/active.json` pointer existed in the installed Hermes home at inspection
  time. This is recorded only as environment evidence; it is not treated as a Runtime
  health result.

The qualification used the immutable source archive and bundled official documentation.
No Hermes command was executed. Static inspection found that every top-level CLI launch,
including a nominal read or `--help`, can run executable-quarantine cleanup, stale
bytecode cleanup and interrupted-install recovery before dispatch. Running it would
therefore have violated B1's read-only/no-update/no-active-generation-mutation boundary.

Primary inspected sources were:

- `website/docs/developer-guide/programmatic-integration.md`;
- `hermes_cli/main.py`, `_parser.py`, `status.py`, `logs.py`,
  `security_audit.py` and the matching `hermes_cli/subcommands/*.py` parsers;
- `gateway/platforms/api_server.py`;
- `tui_gateway/entry.py`, `server.py` and `methods_tools.py`;
- `tools/skills_hub.py`, `hermes_cli/skills_hub.py` and
  `hermes_cli/web_routers/skills.py`;
- applicable source tests shipped in the same archive.

## Shared command and Profile constraints

`hermes -p <profile> ...` validates a lower-case Profile name and resolves it before
normal command dispatch. Explicit `-p` is therefore required for every Profile-scoped
invocation. Omitting it is not acceptable because Hermes also supports a sticky active
Profile, which would make scope ambient rather than request-exact.

The top-level pre-dispatch maintenance described above means none of the CLI commands in
this document is a strictly read-only process surface. A future adapter must not pretend
that a human-readable CLI is an official query protocol merely because its subcommand is
named `status`, `logs` or `list`. A compatibility fallback would need an approved ADR,
exact-version fixtures and an explicit resolution of active-generation ownership; an ADR
cannot waive false success, secret leakage or a security-scan bypass.

## Qualification matrix

| Surface | Exact entry and scope | Automation/output | Safety, bounds and cancellation | Verifiable postcondition | Verdict |
|---|---|---|---|---|---|
| API-server liveness | `GET /health`; the selected API-server process, or documented `/p/<profile>/health` multiplex path | Non-interactive JSON with fixed top-level `status`, `platform`, `version` | No authentication. Caller still needs loopback-only routing, HTTP deadline and response cap | Proves only that the API-server adapter answered | **GO: liveness only** |
| Detailed API health | `GET /health/detailed`; selected API-server/Profile | Non-interactive authenticated JSON | Requires `API_SERVER_KEY`; response includes PID, platform map and exit reason. YORVA must cap, project and redact it | Can verify the API adapter's detailed readiness snapshot, not universal Hermes health | **CONDITIONAL** |
| CLI status | `hermes -p <profile> status [--all] [--deep]` | Human Rich/plain output; no JSON schema or stable size bound | Can expose paths, URLs, PIDs, session/account/channel identifiers and raw errors. `--deep` and some base checks perform network/probe work | Exit success does not establish healthy/degraded state | **NO-GO** |
| CLI doctor | `hermes -p <profile> doctor [--live]`; `--fix` and `--ack` excluded as mutations | Human output; no JSON schema | Even without `--live`, it may perform provider/GitHub/cloud connectivity probes. Per-probe timeouts are not one stable total deadline | Findings are generally printed; their presence does not give a stable typed exit result | **NO-GO** |
| Monitoring status | `hermes -p <profile> monitoring status` | Human output only | Prints configured monitoring/OTLP data; no stable projection or redaction contract | Reports configuration, not Runtime readiness | **NO-GO** |
| Runtime logs | `hermes -p <profile> logs <agent\|errors\|gateway\|gui\|desktop\|mcp> -n N`; `-f` excluded | Human/raw log lines; no JSON | `N` is not upper-bounded, filtered reads can expand to `max(N*20, 2000)`, lines/bytes are uncapped, and no redaction is applied. `-f` is indefinite | Reading lines does not prove health; CLI header also exposes the path | **NO-GO** |
| Security audit | `hermes -p <profile> security audit --json --fail-on low` with explicit skip policy | Structured JSON and documented exit meanings: 0 clean, 1 threshold reached, 2 input/OSV failure | Sends package name/version/ecosystem to OSV, uses 20-second request timeouts and bounded batch size, but lacks an overall deadline, total response/finding/string bounds and cooperative cancellation | JSON findings plus exit class can be checked if both are normalized | **CONDITIONAL** |
| Enabled Skills inventory | Authenticated `GET /v1/skills` on the selected API-server/Profile | Structured JSON list | Requires `API_SERVER_KEY`; body/count/string bounds are not contractual. Project only an allowlist of fields | Verifies enabled skills visible to that API-server process | **CONDITIONAL: inventory only** |
| Skills CLI inventory/search/inspect | `hermes -p <profile> skills list\|search\|inspect ...` | `search --json` is structured; list/inspect are human output | CLI startup may mutate. `list` itself creates Hub directories/index/lock/audit files. Inspect/search can fetch untrusted remote data and lack a consistent total byte bound | Several failures print messages without stable nonzero exit status | **NO-GO** |
| Skills CLI install/update | `hermes -p <profile> skills install|update ...` | Primarily human output | Install accepts direct URLs and `--force`; update internally invokes install with `force=True`, bypassing a blocked scan verdict | No typed read-back; printed success/failure is not an authoritative postcondition | **NO-GO** |
| Skills CLI audit/uninstall | `hermes -p <profile> skills audit|uninstall ...` | Human output | Audit has no stable machine result; uninstall can be non-interactive with `--yes` but failures may return normally | No typed postcondition proving the requested state | **NO-GO** |
| Dashboard Skills REST | `/api/skills` and `/api/skills/hub/*` under `hermes serve`/dashboard | Structured JSON, but not documented as an external management contract | Starting the host performs broader server/schema/background work. Preview is uncapped; install accepts arbitrary identifiers; update inherits the force bypass | Mutation endpoints return spawn acceptance/PID, not completed outcome | **NO-GO** |
| TUI gateway `skills.manage` | Official TUI JSON-RPC transport; method exists in 0.20.5 source but is absent from the published selected method catalog | Structured for list/search/browse/inspect | The documented external launch is the interactive `hermes --tui`; no dedicated side-effect-free management launch was established. Long-running method cancellation/overall limits are not contractual | Install unconditionally returns `installed: true` after calling a routine whose failures can be suppressed as console text | **NO-GO for lifecycle** |

## Detailed findings

### Health/status/doctor/monitoring

The official programmatic-integration guide documents the API server as HTTP/SSE and
lists `/health` and `/health/detailed`. In `gateway/platforms/api_server.py`, `/health`
returns a small unauthenticated JSON response. `/health/detailed` checks API-server
authentication and returns readiness plus gateway state, platform data, active-agent
counts, busy/drainable state, exit reason, timestamp and PID.

This is an official and structured surface, but only for an enabled/running API-server
platform. It is not present for every Hermes installation or Instance. It also requires a
Hermes-native `API_SERVER_KEY` for the detailed route. YORVA has no approved authority in
the B1 materials to retrieve or silently adopt that secret. If P7 chooses this surface,
the API-server lifecycle, loopback endpoint, Profile multiplexing and secret ownership
must be explicitly decided before implementation.

The three CLI diagnostics are not substitutes. `status` and `monitoring status` are
human presentations. `doctor` is a diagnostic workflow with network probes, not a bounded
machine health query. None supplies a fixed category contract and typed postcondition
that YORVA can expose without parsing unstable prose.

### Logs

The CLI does provide a useful fixed filename allowlist, but it prints the underlying path
and raw log content. It has neither secret redaction nor line/byte limits enforced by the
Runtime. A negative or very large `-n` is not rejected, filter mode expands the number of
raw lines read, and follow mode is unbounded. A process deadline or small caller-provided
`-n` cannot repair the absence of a trustworthy redaction/output contract, especially
because conversations, provider errors and channel identifiers may already exist in the
log text.

A future narrow compatibility adapter could only be reconsidered after an ADR defines
known-file ownership, exact allowlisted categories, fixed line and byte caps, process-tree
cancellation, mandatory second redaction and hostile-content/XSS tests. It would remain a
compatibility fallback, not an official structured Hermes surface.

### Security audit

`security audit --json` is the strongest qualified CLI sub-surface. It scans the executing
Python environment plus Profile plugin and pinned MCP declarations and queries OSV. It
transmits package coordinates, not provider/channel secrets. However, the scan mixes
Runtime-installation scope with Profile scope, and it has no one overall time budget or
bounded total finding/detail response.

It may be used only after all of the following are implemented and tested at the adapter
boundary:

- explicit `-p <profile>` and explicit audit scope labeling;
- an exact 0.20.5 JSON/exit-code fixture contract;
- an outer deadline, cancellation and child-process-tree cleanup;
- independent stdout/stderr caps plus strict JSON field/count/string limits;
- normalized, redacted source labels and stable YORVA error codes;
- no repair/update action and no inference from exit code alone;
- explicit user-visible network behavior for the OSV query;
- resolution of the shared CLI pre-dispatch mutation issue.

Until every condition holds, YORVA must report this capability unavailable rather than
silently downgrade to prose parsing.

### Skills

The documented API server exposes authenticated `GET /v1/skills`. It is suitable only as
a conditional inventory of enabled skills for that running API-server/Profile. It does
not expose disabled skills, provenance, installed version, scan state, update, audit or
uninstall.

The CLI and dashboard contain good individual defenses such as Profile isolation,
quarantine and traversal/link checks, but the lifecycle contract fails as a whole:

- direct URL installation is accepted;
- `--force` can override a blocked scan verdict;
- `skills update` passes `force=True` to its install path even when the caller did not
  request a force override;
- read-oriented `skills list` creates Hub state;
- human output and normal returns make failure semantics unstable;
- dashboard mutations acknowledge process spawn, not completed/verified mutation;
- preview and several external registry results lack total byte/cardinality contracts;
- TUI `skills.manage install` reports `installed: true` without proving installation.

These are correctness and security blockers, not documentation nits. P7-D2's rule that
only an inspected/audited source may be installed cannot be met by hiding `--force` in
YORVA while update itself uses the force path. Full Skills lifecycle is therefore
NO-GO on Hermes 0.20.5's current public/programmatic surfaces.

## ADR and implementation blockers

1. **Universal Runtime health surface missing.** Decide whether P7 will require an
   enabled API-server platform per managed Instance, wait for an upstream structured
   surface, or approve a narrow compatibility design. API-server liveness must not be
   relabeled as full Runtime health.
2. **Hermes API-server secret authority missing.** If `/health/detailed` or `/v1/skills`
   is selected, an ADR/security decision must define creation, storage, rotation,
   transport and least-privilege use of `API_SERVER_KEY`; ordinary reads must never
   expose it.
3. **No safe official bounded log surface.** B2 log implementation is blocked pending an
   upstream structured/redacted surface or an approved, tested known-file compatibility
   adapter with the constraints above.
4. **Skills mutation postconditions are false or incomplete.** The force-on-update and
   false-success paths must be fixed upstream or replaced by an official surface that
   returns a typed completed result plus independent read-back. An ADR cannot authorize
   bypassing a blocked security verdict.
5. **CLI launch is not read-only.** Pre-dispatch executable/update recovery and bytecode
   cleanup conflict with active-generation immutability. This must be removed upstream
   or explicitly isolated before any CLI surface can qualify.
6. **Programmatic Skills stability is incomplete.** `skills.manage` is present in source
   but not in the published selected method catalog, and the dashboard routes are not an
   external management contract. Exact-version fixtures alone do not cure the security
   and postcondition blockers.

## Gate consequence

- B1 Health/Logs/Security/Skills qualification: **NO-GO / STOP**.
- No product code, API, migration or Runtime mutation was performed.
- No Hermes/YORVA state-changing command was executed.
- No CI was repeated because this evidence-only inspection did not change product code
  and B0 CI state was unchanged.
- B2/B3/B4 must not claim these rejected capabilities. Narrow conditional sub-surfaces
  may proceed only after their listed ownership, limits, cancellation, redaction and
  exact-version fixture conditions are approved and satisfied.

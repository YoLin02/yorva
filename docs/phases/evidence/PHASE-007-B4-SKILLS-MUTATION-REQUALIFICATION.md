# Phase 7 B4 — Skills Mutation Requalification and Proposed Scope Amendment

## Result

**NO-GO / CAPABILITY MUST REMAIN FALSE — 2026-08-25**

Hermes `0.20.5` does not expose a complete official API, CLI or documented protocol
combination for YORVA's B4 Skill install, update, configure, remove and audit contract.
Combining individually useful surfaces does not close the blocked-scan, exact-Profile,
bounded-result, cancellation and authoritative-postcondition gaps. This result does not
apply to the separately qualified authenticated `GET /v1/skills` enabled-inventory read.

No product code was changed, no Hermes command was run and no mutation capability was
enabled by this requalification.

## Exact evidence identity

- Hermes version: `0.20.5`
- Hermes source commit: `a0ca7c19204e514f9590ce3b812e029b315ab9e9`
- Source archive:
  `.cache/hermes-source/hermes-agent-a0ca7c19204e514f9590ce3b812e029b315ab9e9.zip`
- Archive SHA-256:
  `4E63F39BD992B79BF364A4E22204A22B8E520E2365B53C909F6F15D42C917C54`
- Archive size: `73,798,347` bytes
- Method: static inspection of the verified archive and its official documentation;
  no internal Python module was imported or executed.

## Surface-by-surface result

| Required action | Closest official surface | Blocking fact | Verdict |
| --- | --- | --- | --- |
| Install | `hermes -p <profile> skills install <identifier> --yes` | The command is human-output only and many error paths return normally. Top-level launch performs maintenance before dispatch and explicit Profile resolution can fall back to default on an unexpected resolver error. Direct URL and `--force` are part of the same public surface. | **NO-GO** |
| Install | TUI gateway `skills.manage {action:"install"}` | The handler calls `do_install(...)` and then unconditionally returns `installed: true`; `do_install` returns no typed outcome and suppresses multiple failures as console text. There is no request-scoped cooperative cancellation. | **NO-GO** |
| Update | `hermes -p <profile> skills update <name>` | `do_update` calls `do_install(..., force=True)`. That same `force` value enters `should_allow_install`, so a blocked scan can be bypassed even when the caller did not request a scan override. | **NO-GO / security blocker** |
| Configure enable/disable | `hermes ... skills config` | Interactive curses/input flow; no closed non-interactive arguments or structured result. | **NO-GO** |
| Configure enable/disable | Dashboard `PUT /api/skills/toggle` | Dashboard-internal REST is not documented as the external management contract. Its model has unbounded strings and is not a closed YORVA schema; the response echoes the requested state after save rather than performing an authoritative independent read-back. | **NO-GO** |
| Remove | `hermes -p <profile> skills uninstall <name> --yes` | Human output only; success/failure is printed and the function returns normally. No structured installed/provenance read-back proves removal. | **NO-GO** |
| Remove | Dashboard `/api/skills/hub/uninstall` | Returns process-spawn acceptance and PID, not completed mutation truth. | **NO-GO** |
| Audit | `hermes -p <profile> skills audit [name] [--deep]` | Human report only, no JSON schema, bounded finding contract or stable machine result. Missing paths and unknown names can be printed while returning normally. | **NO-GO** |

## Why a composite workflow still fails

Authenticated `GET /v1/skills` is a useful authoritative read of skills currently
enabled and visible to one API-server Profile. It intentionally excludes disabled
skills and exposes neither installed provenance, version, scan result nor update state.
It therefore cannot prove install-versus-disabled, remove-versus-disabled, update source,
or audit outcome.

The TUI gateway is a documented external JSON-RPC protocol, but `skills.manage` exposes
only list/search/browse/inspect/install. It has no update, remove, configure or audit
action. Its install result is false-success-prone, its list path uses a process cache,
and the long handler runs in a thread pool without operation-scoped cancellation.

The dashboard scan route runs `should_allow_install(..., force=False)`, but it does not
return a bundle digest that can be bound to a later install/update. Pairing that scan
with a separate mutation creates an unverifiable time-of-check/time-of-use boundary.
The dashboard mutation routes also acknowledge process creation rather than completion.

Consequently, a YORVA wrapper cannot repair the official contract merely by omitting
dangerous request fields or parsing success-looking prose. Doing so would still permit
false success, cross-Profile fallback or an update that bypasses a blocked verdict.

## OpenHands selection does not change this result

The exact archive contains the approved-looking local identity
`official/autonomous-ai-agents/openhands`, but that Skill's own frontmatter declares
`platforms: [linux, macos]`. It is not a valid Windows manual-smoke candidate. More
importantly, choosing a fixed catalog identity cannot repair the lifecycle surface's
update, removal, configuration, audit, cancellation and postcondition defects.

## Minimum upstream surface required to reopen B4 mutation

An exact future Hermes version must provide a documented external API/CLI/protocol with:

1. a closed catalog identity, immutable source revision/digest and exact Profile target;
2. non-interactive install, update, configure, remove and audit actions;
3. no direct URL, arbitrary path or force override in the selected contract;
4. a scan verdict bound to the exact bytes mutated, with `BLOCKED` impossible to bypass;
5. bounded structured terminal results and stable error codes;
6. request-scoped deadline/cancellation with mutation cleanup or safe reconciliation;
7. authoritative read-back for installed identity, provenance, version, enabled state,
   content digest and scan/audit state; and
8. exact-version fixtures plus default/named-Profile and destructive Windows smoke.

## Proposed Owner scope amendment

The Owner must select and approve one of these options before Phase 7 can be frozen.

### Option A — Defer B4 mutation, retain honest read-only delivery

- Keep authenticated exact-Profile enabled-Skills inventory and its Desktop projection
  in Phase 7 after focused qualification under ADR-0016.
- Keep `skill.install`, `skill.update`, `skill.configure`, `skill.remove` and
  `skill.audit` routes, Operations, controls and capabilities absent/false.
- Record the stable product limitation `SKILLS_MUTATION_DEFERRED_UPSTREAM` in Phase 7
  acceptance, audit and release evidence.
- Amend both Phase 7 Specs, the verification matrix and Windows manual-smoke requirement
  before candidate audit. Do not describe original B4 lifecycle scope as implemented or
  tested; describe it as an explicit Owner-approved scope deferral.
- Reopen the mutation lane only after an exact Hermes version satisfies the upstream
  contract above and receives fresh qualification.

This is the smallest honest path to a Phase 7 baseline; it changes scope rather than
lowering the mutation acceptance standard.

### Option B — Keep original B4 scope and block Phase 7 freeze

- Preserve the current Phase 7 acceptance criteria unchanged.
- Request or adopt a future official Hermes surface meeting all eight requirements.
- Requalify, implement, run default/named-Profile destructive smoke and then audit.
- Until then, Phase 7 remains `IN_PROGRESS`; no final merge/tag may claim completeness.

## Gate consequence

Without an explicit Owner-approved Option A amendment, the original B4 acceptance and
the Phase 7 completeness Gate remain blocked. Under either option, mutation capability
must remain false on Hermes `0.20.5`; weakening scan, Profile or postcondition standards
is not an acceptable route to PASS.

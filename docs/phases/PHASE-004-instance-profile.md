# YORVA Phase 4 — Instance / Profile Management

> Status: READY  
> Owner: Repository owner  
> Previous phase: Phase 3 — Hermes Installation  
> Previous baseline: `phase-003-hermes-installation-baseline`  
> Previous gate: `AUDIT-003R9-hermes-installation.md` — PASS  
> This document is the Phase 4 Specification only. It does **not** authorize implementation.

Phase 4 implementation must not begin until the Repository Owner separately authorizes it. `ROADMAP.md` does not authorize implementation by itself.

## 1. Objective

Let a user who already has a supported Hermes installation list, create, inspect and delete Hermes-backed YORVA Instances, and see whether those Instances still match what Hermes actually has on disk.

```text
Hermes SUPPORTED (Phase 2 / 3)
→ Desktop lists Instances
→ user creates an Instance (Hermes Profile)
→ user inspects status / identity
→ user deletes an Instance with explicit confirmation
→ YORVA reconciles profiles created or removed outside YORVA
```

An Instance is a logical independently manageable Runtime unit. For V0.1 it maps to one Hermes Profile. The mapping does not imply one OS process per Instance.

## 2. Entry Criteria

- [x] Phase 3 independent audit is `PASS` (`AUDIT-003R9`).
- [x] Phase 3 is `COMPLETE / FROZEN` with tag `phase-003-hermes-installation-baseline`.
- [x] `control/active.json` is the sole activation pointer; leftover `hermes-agent` is not adopted.
- [x] Phase 2 discovery still verifies the active generation after install.
- [ ] Repository Owner approves this Specification and separately authorizes implementation.

Until the last box is checked, implementation must not begin.

## 3. User-visible success flow

1. Desktop refreshes Phase 2 discovery. Instance actions are offered only when discovery is `SUPPORTED` (or a later Owner-approved equivalent).
2. Desktop lists current Instances: name, native identity, last-known status, and whether the row is reconciled or missing from Hermes.
3. Create asks for a name, confirms the destination is the active generation / official `HERMES_HOME` profiles area, and does not accept a user-supplied filesystem path.
4. Delete requires an explicit confirmation that names the Instance. It does not uninstall Hermes.
5. Refresh reconciles external profile add/remove/rename without deleting unknown Hermes user data.
6. If Hermes supports Instance-scoped lifecycle in the reviewed official surface, Desktop exposes Start / Stop / Restart as capability-gated actions. If it does not, those controls stay hidden and the API returns `CAPABILITY_NOT_SUPPORTED`.

Desktop closing does not delete Instances. On reopen, list + reconcile is the recovery surface.

## 4. In Scope

- list Hermes profiles as YORVA Instances;
- create one Instance / Profile through the Hermes adapter;
- delete one Instance with explicit confirmation;
- inspect Instance identity and last-known status;
- reconcile profiles that appeared or disappeared outside YORVA;
- capability flags for Instance and lifecycle support;
- authenticated local HTTP + OpenAPI for Instance list/create/get/delete (and lifecycle only if the official Hermes surface supports it);
- Desktop Instance list / create / delete / empty / conflict / missing-native states;
- SQLite `instances` inventory as a cache, not as Hermes authority;
- focused adapter, application, protocol and Desktop tests.

## 5. Out of Scope

Hard boundaries:

- Hermes install, upgrade, repair, uninstall (Phase 3);
- rewriting `active.json`, generations, staging, or user `PATH`;
- models, credentials, API keys (Phase 5);
- Weixin / WeCom / channels (Phase 6);
- Skills, MCP, backup/restore, Cloud;
- adopting, deleting or migrating leftover `hermes-agent`;
- relocating official Hermes user data;
- a generic plugin/instance framework for hypothetical future Runtimes;
- remote/arbitrary filesystem or shell APIs;
- one-OS-process-per-Instance assumption;
- secret plaintext in list/get APIs.

If official Hermes has no documented programmatic Profile API, stop and produce an amendment rather than parsing undocumented directories or forking Hermes.

## 6. Owning modules

```text
Desktop (React)            presentation, confirmation, list state
Application                Instance use cases, reconcile, Operation only if create/delete is long-running
Domain                     Instance identity, status vocabulary
Hermes adapter             Profile list/create/delete/inspect against official Hermes
Persistence                instances table as last-known inventory
Transport / OpenAPI        typed Instance resources
```

Forbidden:

```text
React → Hermes CLI / profile files
Domain → Hermes types
Hermes adapter → UI
```

## 7. Architecture impact

### API / Protocol

Implement the already sketched Instance routes in `PROTOCOL.md`. Request bodies are typed and closed. No path, command, environment or URL fields. Delete is explicit and idempotent for an already-absent native profile (YORVA row is then removed or marked missing according to the reconcile rule below).

Lifecycle routes exist only if the Hermes adapter reports `lifecycle: true` from a reviewed official surface. Otherwise they return `CAPABILITY_NOT_SUPPORTED`.

### Runtime Adapter

Add `InstanceManager` (and `LifecycleManager` only if proven). Prefer documented official API, then documented protocol, then documented CLI. Do not import Hermes Python internals or depend on undocumented SQLite schemas.

### Data / Migrations

Use the existing `instances` design in `DATA_MODEL.md`. `status` is last-known cache. Hermes remains authoritative. Reconcile never treats a stale YORVA row as proof the Profile still exists.

### Security

No secret plaintext. No generic file APIs. Create/delete stay user-scope. Do not require Administrator. Adapter commands stay argv-safe and Job-Object owned.

### Concurrency / Lifecycle

Serialize create vs delete vs reconcile for the same native identity. Do not hold a SQLite transaction across Hermes commands. Do not take the Phase 3 `install.lock` for Instance mutations unless a later amendment proves they mutate the same managed generation tree.

## 8. Contract / document changes

- [ ] `PROTOCOL.md` / OpenAPI — Instance resources
- [ ] `RUNTIME.md` — Hermes Profile mapping and capabilities
- [ ] `DATA_MODEL.md` — confirm `instances` columns; no speculative fields
- [ ] `SECURITY.md` — Profile mutation bounds
- [ ] ADR only if the official Hermes Profile surface forces an architectural change

## 9. Test matrix

| Scenario | Expected result | Test level |
|---|---|---|
| No supported Hermes | Instance create/list mutation rejected | application |
| Empty official profiles | Empty list, create enabled | adapter + Desktop |
| Create then list | One Instance, native id unique | adapter + protocol |
| Duplicate native identity | Reconcile to one row; no second create | application |
| Delete with confirmation | Profile gone; YORVA row gone | adapter + Desktop |
| Delete missing native | Idempotent; inventory converges | application |
| External profile added | Refresh shows it; not `AMBIGUOUS` install | adapter |
| External profile removed | Refresh marks missing / drops cache; no install rollback | adapter |
| Lifecycle unsupported | Capability false; API `CAPABILITY_NOT_SUPPORTED` | protocol |
| Concurrent create/delete same native | One winner; no torn row | application |
| Unknown directory under HERMES_HOME | Never deleted | adapter |
| Active generation still selected after Instance work | Phase 2 `SUPPORTED` unchanged | integration |

## 10. Acceptance criteria

- [ ] Multiple independent Hermes profiles are discoverable as Instances.
- [ ] User can create and delete an Instance without a terminal.
- [ ] YORVA recovers if profiles are changed outside YORVA.
- [ ] SQLite is not treated as Hermes authority.
- [ ] Phase 3 generation / `active.json` / leftover `hermes-agent` rules are unchanged.
- [ ] No model, credential, channel or Cloud surface is added.

## 11. Phase-specific audit checks

- Hermes Profile surface is documented official API/CLI, not an inferred DB schema.
- Reconcile cannot delete official user data (`.env`, `config.yaml`, `skills`, sessions).
- Instance delete cannot uninstall or deactivate the generation.
- Desktop never talks to Hermes directly.
- Capability flags, not `runtime == hermes` checks in React.

## 12. Known risks

- Official Hermes Profile management may be incomplete or directory-only. That is a stop condition, not a license to invent a second profile store.
- Users may have leftover `hermes-agent` plus a generation. Phase 4 must not treat leftover profiles as a second Runtime.
- Create/delete might be fast enough to skip Operations. If they block on Hermes, they become Operations; do not hold HTTP open for an unbounded external wait.

## 13. Completion evidence

To be completed before audit. Implementation is not authorized by this draft.

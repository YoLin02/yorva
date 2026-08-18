# YORVA Phase 3 Amendment 003A4 — Generation Install Transaction

> Status: ACCEPTED FOR IMPLEMENTATION  
> Date: 2026-08-18  
> Owner approval date: 2026-08-18  
> Owner: Repository owner  
> Phase: 3 — Hermes Installation  
> Related: `ADR-0006`, `AMENDMENT-002A3`, `PHASE-003-generation-installation-architecture.md`  
> Implementation: NOT STARTED; batch-gated (see `PHASE-003-generation-implementation-batches.md`)  
> Audit: historical `AUDIT-003` through `AUDIT-003R6` remain FAIL; `AUDIT-003R7` PENDING  
> Gate: this amendment is not Phase 3 PASS / COMPLETE / FROZEN / ACCEPTED

## 1. Classification

This amendment replaces the Phase 3 ownership / candidate / quarantine / promotion-journal machine with the Owner-approved generation transaction. It does not unlock Phase 4, merge `main`, freeze Phase 3, or rewrite historical audit FAIL reports.

Owner decisions D1–D5 are locked in `ADR-0006` and restated below.

## 2. Problem

The R6 candidate still has independent truths (Operation, promotion journal, live rename, marker, registry). Recovery can leave the canonical tree missing; `path` / environment sit outside the journal; retry infers ownership from the latest Operation. Further journal states would not remove the double machine.

## 3. Decision

Adopt `docs/phases/PHASE-003-generation-installation-architecture.md` as the Phase 3 installation contract, with D1–D5:

| ID | Lock |
| --- | --- |
| D1 | Phase 2 reads `control/active.json` only (`002A3`). No shim / junction / second pointer. |
| D2 | `HERMES_HOME` = `%LOCALAPPDATA%\hermes`. Do not move official user data. Generations live in `generations\`. |
| D3 | `config-templates` failure is a warning, not a Seal blocker. Executable, version, runtime validation, or integrity failure still blocks Seal. |
| D4 | GC: keep active forever; keep the latest previous committed generation; keep at most the latest failed lineage-proven staging/generation; collect only proven, unreferenced YORVA dirs. Never GC Hermes user data or unknown trees. |
| D5 | After first `COMMITTED`, remove only PATH entries proven YORVA-managed (`...\hermes-agent\bin` written by YORVA, or a previous generation `\bin`). Leave user-authored Hermes PATH entries. |

Pipeline:

```text
CREATED → BUILDING → SEALED → PUBLISHED → ACTIVATING → COMMITTED
FAILED only while this generation is not active
```

`control/active.json` is the sole current-generation pointer. Environment is derived. SQLite Operation is a projection. Retry always starts a new transaction and new staging/generation ids.

Official installer is invoked with:

```text
-HermesHome %LOCALAPPDATA%\hermes
-InstallDir <managed-root>\staging\txn_<id>
```

Public launchers are copied into staging `bin\` **before** Seal. Official `-Stage path` is not used to mutate the tree after Seal.

## 4. Out of scope

- Phase 4 Instance/Profile;
- generic transaction framework or plugin host;
- deleting or adopting legacy `hermes-agent`;
- relocating `.env` / `config.yaml` / `skills` / sessions;
- new hermes shim binary.

## 5. Document updates required during implementation

- [x] `PHASE-003-hermes-installation.md` recovery/ownership sections replaced by this amendment (historical FAIL audits untouched)
- [x] `DATA_MODEL.md` — drop Operation-as-owner; document txn/active.json; `ownership_nonce` no longer authorizes retry
- [x] `SECURITY.md` — generation seal / deletion / env derive
- [x] `PROTOCOL.md` / OpenAPI — Operation remains; may add `transactionId` projection field if needed without a second resource
- [x] `ROADMAP.md` — status only
- [x] `RUNTIME.md` — install capability still one Operation

## 6. Acceptance (implementation, not this writing)

- `DecideRecovery` is pure and covered by the failpoint plan in the architecture document §13 and the batch plan.
- No install mutation while daemon gate is not `READY`.
- No implicit newest-generation activation.
- Independent `AUDIT-003R7` (or later) still required for Phase 3 gate.

## 7. Implementation authorization

Implementation is authorized **only** as the batches in `docs/phases/PHASE-003-generation-implementation-batches.md`. Batch 1 is documentation/decision tests only if needed; filesystem writers start at Batch 3. A single-shot rewrite of `host_installer.go` plus discovery plus daemon is not authorized.

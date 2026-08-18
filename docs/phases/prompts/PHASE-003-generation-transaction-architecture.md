# YORVA Phase 3 — Generation Transaction Architecture Migration

> Document type: architecture migration brief and fresh-agent review prompt  
> Status: OWNER-DIRECTED DESIGN / IMPLEMENTATION NOT AUTHORIZED  
> Scope: Phase 3 Hermes Installation transaction, recovery, integrity and environment reconciliation  
> Audit snapshot: fix/phase3-audit-r6-remediation at fdf8c9a3b07344099998bd5534ce96091b25fa77  
> Historical gate: AUDIT-003R6 is FAIL; AUDIT-003R7 remains pending

## 1. Purpose and authority

This document consolidates:

1. the read-only audit of the current Phase 3 installation state machine;
2. the Repository Owner's fixed direction to replace the current ownership-promotion model;
3. a migration design that a fresh architecture agent must validate and refine;
4. a ready-to-use prompt for that fresh agent.

This document authorizes design review only. It does not authorize implementation, merge, freeze, a Phase 3 baseline tag, destructive migration of an existing Hermes installation, or Phase 4 work. Historical audit reports and FAIL decisions remain immutable.

The Owner-fixed direction is:

~~~text
Install Transaction
        ↓
Fresh Staging
        ↓
Build
        ↓
Validate
        ↓
Seal
        ↓
Publish Generation
        ↓
Atomic Activate
        ↓
Reconcile Environment
        ↓
Commit
        ↓
GC
~~~

Do not extend the current candidate + quarantine + ownership promotion journal design with more compensating states such as ROLLED_BACK or ENV_APPLIED. Replace that transaction model.

## 2. Mandatory reading and snapshot verification

Before reviewing this design, read completely:

~~~text
AGENTS.md
docs/DEVELOPMENT.md
docs/ARCHITECTURE.md
docs/PROTOCOL.md
docs/RUNTIME.md
docs/DATA_MODEL.md
docs/SECURITY.md
docs/PHASE_GOVERNANCE.md
docs/AUDIT_STANDARD.md
docs/ROADMAP.md
docs/phases/PHASE-002-hermes-discovery.md
docs/phases/PHASE-003-hermes-installation.md
docs/phases/amendments/AMENDMENT-002A1-hermes-windows-command-resolution.md
docs/phases/amendments/AMENDMENT-003A1-embedded-hermes-source.md
docs/phases/amendments/AMENDMENT-003A2-china-dependency-distribution.md
docs/phases/amendments/AMENDMENT-003A3-managed-node-prerequisites.md
docs/phases/audits/AUDIT-003-hermes-installation.md
docs/phases/audits/AUDIT-003R1-hermes-installation.md
docs/phases/audits/AUDIT-003R2-hermes-installation.md
docs/phases/audits/AUDIT-003R3-hermes-installation.md
docs/phases/audits/AUDIT-003R4-hermes-installation.md
docs/phases/audits/AUDIT-003R5-hermes-installation.md
docs/phases/audits/AUDIT-003R6-hermes-installation.md
all relevant ADRs under docs/adr/
~~~

Verify rather than assume:

~~~text
current branch and HEAD
working tree and index
origin/main and feature-branch tips
all historical audit reports remain unchanged
no Phase 3 baseline tag exists
the actual implementation still matches the paths described here
~~~

If the code has moved past the recorded snapshot, repeat the read-only state-machine audit against the new immutable candidate before accepting this document's code-specific conclusions.

## 3. Current implementation: audited facts

### 3.1 Current state domains

The implementation has independently persisted or observed state in:

1. SQLite API Operation:

   ~~~text
   PENDING → RUNNING → SUCCEEDED | FAILED | CANCELLED
   ~~~

   OPERATION_INTERRUPTED is an error code on FAILED.

2. Filesystem promotion journal:

   ~~~text
   PREPARED → OLD_QUARANTINED → NEW_PROMOTED → COMMITTED
   ~~~

3. Filesystem reality:

   ~~~text
   work directory
   candidate
   canonical live directory
   quarantine
   ownership marker
   manifest
   atomic temporary files
   ~~~

4. Windows registry reality:

   ~~~text
   HERMES_HOME
   user PATH
   ~~~

5. Accepted-installation metadata and authoritative Phase 2 discovery.

These domains do not share an atomic commit boundary.

### 3.2 Current call chain

~~~text
HTTP POST /api/v1/runtimes/hermes/install
→ app.RuntimeInstall.Start
→ persist Operation(PENDING)
→ background RuntimeInstall.execute
→ PreviousRuntimeInstall
→ HostInstaller.ValidateTarget
→ persist Operation(RUNNING)
→ HostInstaller.Apply
   → workDir
   → candidate
   → source/protocol verification
   → repository materialization
   → ownership marker/manifest
   → candidate stages and delta re-signing
   → launcher materialization
   → promoteCandidate
   → registry environment update
   → launcher check
→ Phase 2 discovery post-check
→ SQLite transaction: accepted installation + Operation(SUCCEEDED)
~~~

Daemon startup currently performs:

~~~text
RecoverPromotions
→ log warning and continue on recovery error
→ InterruptActiveInstalls
→ expose HTTP service
~~~

### 3.3 Current object responsibilities and problems

| Object | Current responsibility | Audited problem |
| --- | --- | --- |
| candidate | Operation-private mutable installation tree | orphaned on hard crash before journal; allowed paths cannot prove provenance |
| canonical hermes-agent | active installation and failed post-promotion installation | activation and API success are different commits |
| quarantine | preserve previous live tree | never collected; recovery can restore it without closing journal state |
| ownership marker | bind Operation, target, source pin, manifest and HMAC | combines lineage and integrity; success still carries a partial-install marker |
| manifest | hash relative paths and contents | proves snapshot integrity, not creator; ignores a broad atomic-temp prefix |
| promotion journal | record rename intent and progress | excludes rollback, registry, post-check and Operation acceptance |
| SQLite Operation | API, UI and SSE state | also used to infer ownership and retry authority |

### 3.4 Current failure and retry behavior

Before promotion, a normal failure removes work and candidate directories and leaves the canonical directory unchanged. A hard exit can leave unjournaled work/candidate directories.

During retry, the most recent prior install Operation is treated as the current directory owner. A fresh candidate is built, the old canonical tree is renamed to quarantine, and the candidate is renamed to the canonical path. The old tree is not merged and is not automatically deleted.

After promotion, environment, launcher, discovery or SQLite failure leaves the new tree at the canonical path. The API Operation may be FAILED or OPERATION_INTERRUPTED while the promotion journal is already COMMITTED.

### 3.5 Crash windows the replacement must eliminate

1. PENDING persists before worker ownership exists.
2. Work/candidate can exist before any journal and become orphans.
3. Atomic temp files can remain and be excluded from the manifest.
4. Atomic replace can succeed before the final durability call reports failure.
5. PREPARED can persist before the old-tree rename.
6. Old-tree rename can complete before OLD_QUARANTINED persists.
7. Recovery can restore quarantine while leaving OLD_QUARANTINED, causing the next recovery to reject the restored tree.
8. Candidate rename can complete before NEW_PROMOTED persists.
9. COMMITTED can exist before environment reconciliation begins.
10. HERMES_HOME can succeed while PATH fails.
11. Filesystem and registry can be complete while Operation remains RUNNING and accepted metadata is absent.
12. One invalid journal can stop later recovery while the daemon still accepts new mutations.

### 3.6 Findings to carry into the redesign

- preflight accepts an empty canonical directory while promotion treats every existing directory as an owned live tree;
- a second retry can select Operation B as owner while the unchanged directory still belongs to Operation A;
- allowed directory/file names are not provenance;
- delta validation and re-signing retain time-of-check/time-of-use windows;
- pre-repository stages can create content later signed wholesale;
- the atomic-temp basename prefix is excluded throughout the tree;
- environment is outside recovery;
- recovery does not fail closed at the daemon/install boundary;
- journals, quarantines and crash orphans accumulate;
- reported stages do not exactly match launcher, activation, registry and post-check effects.

## 4. Target directory model

Use one fixed canonical user-scope managed root:

~~~text
hermes/
├─ control/
│  ├─ active.json
│  ├─ install.lock
│  └─ transactions/
│     └─ <transaction-id>.json
├─ generations/
│  ├─ <generation-a>/
│  │  ├─ generation.json
│  │  ├─ manifest.json
│  │  └─ ...sealed Hermes installation tree...
│  └─ <generation-b>/
├─ staging/
│  └─ <transaction-id>/
├─ failed/
│  └─ <transaction-id>/
├─ cache/
└─ mutable-data/  # only if required by the verified Hermes contract
~~~

Rules:

- control/transactions/<id>.json is the authoritative persistent InstallTransaction. SQLite must not duplicate its state.
- active.json is the only activation pointer. Do not add a SQLite active-generation field, current directory rename, junction or second flag.
- install.lock is an OS-held exclusivity mechanism, not durable state. File presence alone is not a lock.
- staging is fresh and mutable. Retry never resumes it.
- generations are immutable after publish.
- failed is optional diagnostic retention for proven YORVA staging data, not rollback state.
- cache contains verified/re-downloadable inputs and never authorizes activation/deletion.
- mutable Hermes user data must remain outside generations.
- all IDs use a closed generated format; persisted/user-supplied absolute paths never select targets.

## 5. InstallTransaction

Proposed authoritative record:

~~~go
type InstallTransaction struct {
    Schema                 int
    Revision               uint64
    ID                     string
    RuntimeKind            string
    OperationID            string // projection correlation only
    GenerationID           string
    State                  TransactionState
    Step                   InstallStep
    SourcePin              string
    ExpectedVersion        string
    StagingRelativePath    string
    GenerationRelativePath string
    ManifestSHA256         string
    SealSHA256             string
    ActiveBeforeGeneration string
    ActiveBeforeDigest     string
    ErrorCode              string
    CreatedAt              time.Time
    UpdatedAt              time.Time
    SealedAt               *time.Time
    PublishedAt            *time.Time
    ActivatedAt            *time.Time
    CommittedAt            *time.Time
}
~~~

Constraints:

- persist only under control/transactions through same-directory atomic replacement;
- transitions are forward-only and revision checked under the install lock;
- recovery never requires SQLite Operation;
- OperationID links UI/diagnostics but supplies no ownership or recovery authority;
- Step is progress only, not a persistent transaction sub-state;
- raw upstream errors, paths, output and secrets are excluded;
- FAILED is valid only when this transaction's generation is not active;
- if active.json already selects the valid new generation, recovery rolls forward even if state still says ACTIVATING.

### 5.1 Minimal state machine

~~~mermaid
stateDiagram-v2
    [*] --> CREATED
    CREATED --> BUILDING
    BUILDING --> SEALED: build + validate + seal
    SEALED --> PUBLISHED: atomic staging-to-generation rename
    PUBLISHED --> ACTIVATING: durable activation intent
    ACTIVATING --> COMMITTED: active valid + environment reconciled

    CREATED --> FAILED
    BUILDING --> FAILED
    SEALED --> FAILED
    PUBLISHED --> FAILED: only while not active
    ACTIVATING --> FAILED: only while predecessor remains active and forward activation is impossible

    COMMITTED --> [*]
    FAILED --> [*]
~~~

Normal execution:

~~~text
CREATED
→ BUILDING
→ create fresh staging
→ build all tree artifacts
→ validate layout/version/executable in staging
→ write and verify manifest + seal
→ SEALED
→ atomic rename staging → generations/<generation-id>
→ PUBLISHED
→ persist ACTIVATING with active-before observation
→ atomic replace active.json
→ reconcile registry from active.json
→ verify active resolution
→ COMMITTED
→ project SUCCEEDED to SQLite Operation
→ best-effort bounded GC
~~~

GC failure never changes COMMITTED to FAILED.

## 6. Seal and security semantics

### 6.1 Generation lineage

Lineage means YORVA created a generation through a known transaction under the managed root. generation.json binds:

~~~text
schema
transaction ID
generation ID
runtime kind
source pin
expected version
relative generation path
manifest digest
timestamps
random lineage identity
~~~

Lineage prevents accidental adoption/deletion. It does not claim resistance to malware with the same user authority.

### 6.2 Integrity

manifest.json contains sorted relative paths, sizes and SHA-256 digests for the complete generation, excluding only the two precisely named root seal files. Never ignore a basename prefix throughout the tree.

Seal requires:

1. all build processes and descendants have exited;
2. all files are closed;
3. repository, venv, dependencies, launchers, bin and templates are complete;
4. final manifest is generated;
5. seal is atomically written and read back;
6. a second complete tree walk matches the manifest;
7. no later installation-tree mutation is allowed.

### 6.3 Provenance

Provenance comes from verified inputs and execution policy:

- immutable official source pin;
- reviewed size and SHA-256;
- verified installer protocol/manifest;
- pinned Node/npm archives and lockfiles;
- closed argv/environment;
- contained and reaped process trees.

An allowed path, filename or manifest does not prove which same-user process created bytes. Stronger same-user resistance requires separately reviewed ACL, sandbox or upstream-signature hardening.

### 6.4 Deletion authority

Automatic deletion requires all of:

- canonical containment under staging, failed, cache or generations;
- a valid transaction/lineage reference to the exact directory ID;
- the generation is not active;
- no nonterminal transaction references it;
- regular-directory/reparse checks pass;
- retention policy allows deletion.

Unknown, legacy, malformed, externally changed or ambiguously referenced directories are never automatically deleted.

## 7. Active pointer

Proposed control/active.json:

~~~json
{
  "schema": 1,
  "runtimeKind": "hermes",
  "generationId": "gen_<closed-id>",
  "generationRelativePath": "generations/gen_<closed-id>",
  "manifestSha256": "<sha256>",
  "sealSha256": "<sha256>",
  "sourcePin": "<immutable-commit>",
  "version": "0.20.2",
  "transactionId": "txn_<closed-id>",
  "activatedAt": "<UTC RFC3339Nano>"
}
~~~

Rules:

- accept only validated relative generation paths;
- validate the published generation and seal before activation;
- persist ACTIVATING before replacing active.json;
- use same-directory temp, file flush, close, atomic replace and supported durability operation;
- read back and revalidate pointer + seal;
- missing/corrupt active.json never means “pick newest generation”;
- unknown/unsealed targets fail closed;
- old generations never move during activation.

### 7.1 Frozen Phase 2 compatibility

Phase 2 currently checks the fixed legacy/current path:

~~~text
%LOCALAPPDATA%\hermes\hermes-agent\bin\hermes.exe
~~~

A generation path materially changes that frozen discovery contract. The architecture review must select exactly one governed solution:

1. formally amend Phase 2 with a bounded read-only active.json resolver; or
2. introduce a stable integrity-pinned Go launcher at an approved discovery path that resolves only active.json.

The first is simpler unless normal shell use requires a stable launcher. Either option needs explicit governance review. Phase 3 must not silently change frozen Phase 2.

## 8. Environment Reconcile

Environment is derived state.

Inputs:

~~~text
valid active.json
validated active generation
fixed managed/mutable-data root policy
observed HKCU\Environment
~~~

Algorithm:

~~~text
ResolveAndValidateActive
→ ReadObservedEnvironment
→ ComputeEnvironmentPlan(desired, observed)
→ apply only missing/different managed values
→ read back
→ verify
→ optionally broadcast environment change
~~~

Requirements:

- ComputeEnvironmentPlan is pure and deterministic;
- repeated application is safe;
- crash after HERMES_HOME but before PATH is repaired at startup;
- daemon post-check resolves active directly and does not depend on inherited PATH;
- environment failure leaves the transaction ACTIVATING and never rolls back active.json;
- startup reconciles environment before allowing another install;
- later registry drift is repaired from active.json without reopening a committed transaction;
- unrelated PATH entries remain intact;
- only proven YORVA-managed stale generation entries may be removed.

The architecture review must verify the pinned official installer's exact HermesHome/InstallDir and mutable user-data behavior before fixing the desired HERMES_HOME value.

## 9. Recovery model

Recovery is:

~~~text
ObserveStore
→ DecideRecovery(transaction, filesystem, active pointer, environment)
→ RecoveryDecision
→ ExecuteDecision
→ Observe again
→ converge or fail closed
~~~

DecideRecovery should be pure and must not read SQLite Operations.

### 9.1 Decision matrix

| Transaction | Staging | Generation | Active | Decision |
| --- | --- | --- | --- | --- |
| no transaction | unknown directory | any | any | never delete; diagnose; block only on managed-ID conflict |
| CREATED | absent | absent | predecessor/none | FAILED; no filesystem mutation |
| CREATED | mutable | absent | predecessor/none | move proven staging to failed; FAILED |
| BUILDING | mutable | absent | predecessor/none | never resume; retain/move proven staging; FAILED |
| BUILDING | absent | absent | predecessor/none | FAILED with interrupted-build code |
| SEALED | valid sealed | absent | predecessor/none | publish, then PUBLISHED |
| SEALED | absent | valid matching | predecessor/none | infer rename completed; PUBLISHED |
| SEALED | invalid | absent | predecessor/none | preserve evidence; FAILED; never activate |
| SEALED | valid | valid | predecessor/none | ambiguous duplicate; fail closed unless a reviewed exact-lineage rule resolves it |
| PUBLISHED | absent | valid | predecessor | persist ACTIVATING; activate forward |
| PUBLISHED | absent | valid | already new | infer activation; ACTIVATING; reconcile environment |
| PUBLISHED | any | invalid/missing | predecessor/none | fail closed; preserve predecessor |
| ACTIVATING | absent | valid | predecessor | complete activation forward |
| ACTIVATING | absent | valid | already new | reconcile environment; COMMITTED |
| ACTIVATING | any | valid | unrelated generation | fail closed; do not guess concurrent intent |
| ACTIVATING | any | invalid/missing | predecessor/new | fail closed; preserve predecessor/evidence |
| COMMITTED | absent | valid | active generation | no transaction action; global environment reconcile |
| FAILED | proven failed/staging | not this generation | any | never resume; retention GC only |
| FAILED | any | this generation | this generation | inconsistent; reconcile forward or fail closed; never delete active |

### 9.2 Concurrency and daemon gate

- an OS-held install.lock serializes recovery and transaction creation;
- at most one nonterminal transaction is allowed;
- multiple nonterminal records block mutations until deterministically reconciled;
- committed/failed history does not compete with active.json;
- malformed records are classified before any mutation;
- daemon install subsystem exposes READY, RECONCILING or BLOCKED_UNSAFE;
- health/read-only discovery may remain available when safe, but new install/prerequisite mutations are rejected while not READY;
- warning-and-continue behavior for unsafe recovery is forbidden.

## 10. No new double state machine

Each authority must be non-overlapping:

| Concern | Sole authority | Not authoritative |
| --- | --- | --- |
| in-flight intent/recovery | InstallTransaction JSON | Operation, UI, directory presence |
| active generation | active.json | transaction state, latest directory, SQLite, PATH |
| sealed contents | manifest + generation seal | allowed names, Operation |
| user-visible progress | SQLite Operation projection | ownership/recovery |
| desired registry | valid active.json + fixed policy | prior Operation/Step |
| deletion permission | lineage + containment + references | prefix/marker presence |
| concurrency | held OS lock | lock-file presence or SQLite alone |

One-way projections:

~~~text
InstallTransaction → SQLite Operation
active.json → environment desired state
active.json + seal → discovery candidate
transaction/lineage → GC eligibility
~~~

Forbidden reverse inference:

~~~text
Operation FAILED ≠ filesystem transaction failed
latest Operation ≠ generation owner
PATH entry ≠ active generation
directory existence ≠ lineage
manifest match ≠ provenance
~~~

## 11. Current-file disposition

Verify all call sites before removal.

### 11.1 Remove after replacement passes its tests

| Current code | Disposition |
| --- | --- |
| ownership_promote.go | remove candidate/quarantine rename model and recovery |
| ownership_journal.go | remove promotion journal format/states |
| ownership_delta.go | remove stage delta re-signing as provenance |
| retry-owner parts of target.go | remove previous-Operation owner inference |
| old promotion/retry tests | replace semantics; retain useful negative fixtures |

### 11.2 Migrate

| Current code | Target |
| --- | --- |
| ownership_inventory.go | generation integrity manifest; remove prefix exclusion and allowlist ownership |
| ownership_marker.go | generation lineage/seal writer with accurate naming |
| ownership_atomic*.go | focused atomic record writer for transaction, seal and active pointer |
| host_installer.go | Hermes Build and Validate only; remove promotion/recovery/environment |
| host_path_windows.go | EnvironmentReconciler with pure plan and idempotent apply |
| runtime_install_run.go | call InstallManager and project progress |
| runtime_install.go | every request/retry creates a new transaction; no previous-owner decision |
| daemon.go | recovery/reconcile under fail-closed install gate |
| Phase 3/Security/Data Model docs | generation, transaction, environment and deletion contracts |

### 11.3 Reuse

- verified source/archive acquisition and extraction bounds;
- pinned Hermes, Node and npm metadata;
- protocol/manifest verification;
- direct argv process runner and Windows Job Object containment;
- cancellation, timeout and output bounds;
- containment/reparse checks after a handle/TOCTOU review;
- Phase 2 version parsing/execution within the approved amendment;
- OpenAPI Operation DTOs, SSE and Desktop progress as projections;
- accepted-installation SQLite data as cache, not activation truth.

### 11.4 Proposed package ownership

~~~text
services/node/internal/install/
├─ transaction.go
├─ transaction_store.go
├─ generation.go
├─ seal.go
├─ active_pointer.go
├─ recovery.go
├─ environment.go
├─ gc.go
└─ manager.go

services/node/internal/runtime/hermes/
├─ build.go
├─ validate_install.go
├─ version.go
└─ locate_executable.go
~~~

Do not create a plugin framework, dependency-injection container or generic package-manager system.

## 12. Migration order

Implementation is not authorized by this document. After architecture approval:

1. Governance:
   - amend Phase 3;
   - add an ADR;
   - formally resolve Phase 2 discovery;
   - define legacy coexistence/migration.
2. Decision model:
   - define transaction, observation and RecoveryDecision;
   - exhaustively test pure DecideRecovery and idempotence.
3. Atomic records/store:
   - transaction store, lock, active pointer, closed IDs and path guards.
4. Generation build/seal:
   - fresh staging;
   - all launchers before seal;
   - direct executable/version validation;
   - second manifest verification.
5. Publish/activate:
   - atomic staging-to-generation rename;
   - atomic active pointer;
   - old generation untouched.
6. Recovery executor/daemon gate:
   - execute pure decisions under lock;
   - re-observe after every action;
   - block ambiguity.
7. Environment reconcile:
   - pure plan, registry apply/readback, inherited-PATH-independent post-check.
8. Operation projection:
   - transaction drives UI stages/status; SQLite failure cannot drive filesystem.
9. Legacy handling:
   - never adopt/delete unknown legacy paths;
   - retry always fresh;
   - preserve legacy until new generation commits;
   - prevent Phase 2 ambiguity through the approved integration.
10. Remove old architecture only after generation tests pass.
11. Run full verification, exact-commit CI/MSI and fresh independent AUDIT-003R7 or later.
12. Do not merge/freeze/tag/enter Phase 4 before PASS.

## 13. Failpoint test plan

Every failpoint test must restart from persisted bytes, run recovery repeatedly, and assert Section 14 invariants.

### 13.1 Atomic record writes

For InstallTransaction, generation seal and active.json inject at:

~~~text
before temp create
after temp create
during partial write
after complete write before file sync
after file sync before close
after close before directory sync
after pre-replace directory sync
after atomic replace before final directory sync
after final sync before read-back
after read-back before caller update
~~~

Require old-or-new complete records only; reconcile a replace that succeeded despite a later reported failure; never activate an unsealed generation.

### 13.2 Build and seal

Inject cancel/timeout/error/crash:

~~~text
before/after staging mkdir
during source download/extract/repository copy
during venv/dependencies
during launcher creation
after launcher before validation
during version/executable validation
during manifest walk
after manifest before seal
after seal before second verification
after SEALED before publish
~~~

Require old active untouched, fresh IDs on retry, no mutable staging activation and no surviving process descendants.

### 13.3 Publish

Inject:

~~~text
before staging → generation rename
after rename before PUBLISHED write
during PUBLISHED replacement
after PUBLISHED before ACTIVATING
generation destination already exists
staging and generation both exist
missing/changed seal
reparse at source/destination/ancestor
~~~

Require observation to infer completed rename, ambiguity to fail closed, and old active never to move.

### 13.4 Activation

Inject:

~~~text
before/after ACTIVATING persists
during active temp write/sync/close
after active replace before final sync
after active replace before transaction update
after active verifies before environment
active=predecessor
active=new
active=unrelated
active missing/malformed/absolute/escaping/seal-mismatched
~~~

Require active.json as sole decision, forward reconcile when active=new, and blocked mutation on unrelated/invalid pointers.

### 13.5 SQLite projection

Inject:

~~~text
after CREATED before Operation insert
after Operation insert before BUILDING
on every stage projection
after active commit before Operation success
during accepted-installation + Operation success transaction
after DB commit before SSE
~~~

Require filesystem recovery never to read Operation as authority; stale/missing UI projection must not activate/delete/rollback; SSE loss remains GET-recoverable.

Creation ordering should be:

~~~text
allocate transaction + Operation IDs
→ persist InstallTransaction(CREATED)
→ persist Operation(PENDING, transaction ID projection)
~~~

If SQLite insertion fails, the CREATED transaction can be marked FAILED. A crash between the two leaves a transaction that recovery can safely fail without filesystem effects.

### 13.6 Registry

Inject:

~~~text
registry open/read failure
before HERMES_HOME
after HERMES_HOME before PATH
during PATH
after PATH before read-back
read-back mismatch
broadcast failure
external drift after COMMITTED
~~~

Require repeated convergence, preservation of unrelated PATH entries, no generation rollback/deletion, ACTIVATING until required values are observed, and drift repair from active.json.

### 13.7 Recovery properties

For each matrix row:

~~~text
D1 = DecideRecovery(input)
Execute(D1)
input2 = Observe()
D2 = DecideRecovery(input2)
~~~

Require D2 to be no-op or strictly forward, never oscillating. Add permutation/property tests for missing, duplicate, modified and reparse paths.

### 13.8 GC

Inject before and during each deletion. Require:

- active pointer re-read under lock;
- only inactive, unreferenced, lineage-valid entries selected;
- uncertain data retained with diagnostics;
- GC failure never changes transaction success.

## 14. Required invariants

The reviewed design and implementation must prove:

1. active.json references only a complete, validated, sealed generation.
2. new-install failure cannot damage the current active generation.
3. retry creates a new transaction, staging and generation ID.
4. unproven directories are never automatically deleted.
5. no installation-tree write occurs after seal.
6. recovery is deterministic, repeatable and idempotent.
7. SQLite Operation never drives filesystem recovery.
8. environment is reconstructed only from a valid active generation and fixed policy.
9. unsafe/ambiguous recovery blocks new installation mutations.
10. mutable user data is outside immutable generations.
11. pointer replacement and transaction lag reconcile forward from observation.
12. GC is optional and cannot affect active availability.

## 15. Questions that must close before implementation

The architecture agent must answer:

1. What exact root contains generations, and which exact Hermes paths hold mutable user data?
2. Can the pinned installer build with InstallDir in staging while HERMES_HOME remains stable?
3. Will Phase 2 receive an active.json amendment or will a stable launcher be introduced?
4. How will terminal users obtain the hermes command after generation changes?
5. What exact Windows durability guarantee is relied upon?
6. What observation distinguishes completed publish/activation from an interrupted write?
7. How do legacy and generation installs coexist without discovery ambiguity?
8. How are multiple nonterminal transactions handled?
9. Which process owns install.lock and how does crash release work?
10. What exact same-user threat is deferred?

A frozen-contract change, new binary, mutable-data-layout change or Owner-visible product choice must stop at design for approval.

## 16. Required fresh-agent deliverable

Create one reviewed design document:

~~~text
docs/phases/PHASE-003-generation-installation-architecture.md
~~~

It must contain:

1. final state machine and diagram;
2. final directory model;
3. exact InstallTransaction schema/persistence;
4. exact active-pointer contract;
5. complete recovery matrix;
6. environment reconcile contract;
7. lineage/integrity/provenance/deletion semantics;
8. current-file remove/migrate/reuse map;
9. ordered migration batches;
10. filesystem/SQLite/registry failpoint matrix;
11. proof that no competing state machine or hidden activation source remains;
12. Phase 2, legacy and CLI compatibility decisions.

Do not write implementation code in the design turn.

## 17. Ready-to-use fresh-agent prompt

Copy this into a new Agent task:

~~~text
你现在作为 YORVA Phase 3 Generation Transaction Architecture 独立设计 Agent。

本轮只做架构设计与审查，不得修改任何 Go、Rust、TypeScript、OpenAPI、SQLite migration、CI 或打包实现代码。

首先完整读取：

docs/phases/prompts/PHASE-003-generation-transaction-architecture.md

然后按该文件的 Mandatory reading and snapshot verification 读取全部治理、架构、Phase Spec、amendment 和 AUDIT-003～AUDIT-003R6 历史，并以当前真实 HEAD/working tree 重新核对相关代码。不要假设文档中的提交仍是最新状态。

Owner 已固定以下方向，不得退回 candidate/quarantine/promotion journal，也不得增加 ROLLED_BACK、ENV_APPLIED 等补丁状态：

Install Transaction
→ Fresh Staging
→ Build
→ Validate
→ Seal
→ Publish Generation
→ Atomic Activate(active.json)
→ Reconcile Environment
→ Commit
→ GC

你的任务是审查并优化该 Markdown 中提出的迁移设计，重点证明：

1. InstallTransaction 是唯一 in-flight/recovery 事务真相源；
2. active.json 是唯一 active generation commit pointer；
3. SQLite Operation 只是 API/UI/SSE 投影；
4. generation 在 Seal 后 immutable；
5. 新安装失败不修改旧 active generation；
6. recovery 使用 Intent + Observed Filesystem + Active Pointer，并且纯决策、幂等；
7. PATH/HERMES_HOME 可从 active generation 幂等补齐；
8. lineage、integrity、provenance、deletion authority 严格分开；
9. 所有 filesystem/transaction-record/SQLite/registry crash point 都有恢复决策和 failpoint；
10. 不产生 SQLite active flag、current 目录、隐式 latest 选择或其他 activation source。

必须结合当前实际函数和路径评审，重点检查：

- Phase 2 冻结 discovery 对 %LOCALAPPDATA%\hermes\hermes-agent 的依赖；
- pinned installer 的 HermesHome/InstallDir 和 mutable user-data 边界；
- legacy 与 generation 共存时的发现歧义；
- active pointer 的 Windows 原子性和 durability；
- transaction 与 SQLite Operation 创建/投影的 crash window；
- daemon recovery 失败时的 fail-closed gate；
- GC 绝不删除 active、unknown 或 referenced generation。

输出并保存唯一主要设计文件：

docs/phases/PHASE-003-generation-installation-architecture.md

必须包含状态机、目录、InstallTransaction、active pointer、完整 recovery 矩阵、Environment Reconcile、安全语义、文件迁移表、实施顺序、全部 failpoint、双状态机审查，以及 Phase 2/legacy/CLI 决策。

如实际代码或冻结合同冲突，停止在设计阶段并给出最小决策选项。不得暗改 Phase 2，不得降低历史审计标准。

完成设计后停止。不要实现，不要运行真实 Hermes 安装，不要 merge main，不要冻结 Phase 3，不要打 tag，不要进入 Phase 4。

最终只汇报：

1. 审查的 HEAD/branch；
2. 设计文档路径；
3. 状态机是否收敛；
4. 是否仍有双状态机或隐式真相源；
5. Phase 2/legacy/CLI 兼容结论；
6. Owner 决策项；
7. 是否可以进入实现提示词编写阶段。
~~~

## 18. Stop condition

Stop after the architecture design is reviewed.

~~~text
Owner approves final generation architecture
→ required Phase 2/Phase 3 amendments and ADR are accepted
→ implementation prompt is separately written and authorized
~~~


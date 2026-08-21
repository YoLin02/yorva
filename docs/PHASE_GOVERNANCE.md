# YORVA Phase Governance

> Status: Engineering governance baseline  
> Applies to: all implementation phases after architecture freeze  
> Purpose: ensure YORVA advances through explicit, auditable stage gates instead of continuously accumulating unreviewed scope.

## 1. Governance principle

YORVA uses a phase-gated development model.

No implementation phase automatically unlocks the next phase simply because the planned code has been written.

The required lifecycle is:

```text
Phase Spec
   ↓
Implementation
   ↓
Verification
   ↓
Phase Audit
   ↓
Gate Decision
   ↓
Baseline / Fixes
   ↓
Next Phase Spec
```

The governing rule is:

> Finish, verify, audit and freeze the current phase before designing the detailed implementation of the next phase.

`ROADMAP.md` defines direction. A Phase Spec defines the actual authorized scope for one implementation stage.

---

## 2. Why YORVA uses phase gates

YORVA is expected to use coding agents such as Codex extensively. Coding agents are useful for implementation, but they can also:

- expand scope without an explicit product decision;
- introduce speculative abstractions;
- refactor unrelated modules;
- optimize for passing local tests while weakening architectural boundaries;
- implement future capabilities before their requirements are understood;
- duplicate state or security responsibilities across layers.

Phase governance limits those risks by giving every implementation stage a narrow contract and an explicit exit gate.

A phase is considered complete only when its implementation and its engineering quality have both been accepted.

---

## 3. Document hierarchy

The project uses the following hierarchy:

```text
AGENTS.md
   ↓
Architecture / security / protocol / runtime baseline
   ↓
ADRs
   ↓
ROADMAP.md
   ↓
Phase Spec
   ↓
Implementation
   ↓
Phase Audit
```

Higher-level documents constrain lower-level documents.

A Phase Spec must not silently override an ADR, architecture invariant, security rule or protocol contract.

If a phase discovers that a higher-level decision is no longer valid:

1. stop the conflicting implementation;
2. document the new evidence;
3. create or supersede an ADR when the change is architectural;
4. update affected baseline documents;
5. resume the phase only after the new decision is explicit.

Current frozen-baseline correction: Phase 3 `AMENDMENT-003A6` / `ADR-0009` is
`ACCEPTED / FROZEN`. It authorizes direct final-path Hermes generation construction
and final-path executable validation. It supersedes only the former
staging-build/publish-rename mechanism; Phase 3 otherwise remains frozen and no
unrelated installation scope is reopened.

---

## 4. Phase identifiers

Use stable numeric phase identifiers.

Current roadmap numbering is authoritative:

```text
Phase 0   Architecture freeze
Phase 1   Repository foundation / bootstrap
Phase 2   Hermes Discovery & Compatibility
Phase 3   Hermes Installation
Phase 4   Instance/Profile management
...
```

Implementation specs should use filenames such as:

```text
docs/phases/PHASE-001-repository-foundation.md
docs/phases/PHASE-002-hermes-discovery.md
```

Audit reports should use:

```text
docs/phases/audits/AUDIT-001-repository-foundation.md
docs/phases/audits/AUDIT-002-hermes-discovery.md
```

`docs/BOOTSTRAP.md` is the execution specification for **Phase 1 — Repository foundation** and does not need to be duplicated unless it is later migrated into the phase directory.

---

## 5. Phase states

A phase may be in exactly one primary state:

```text
DRAFT
READY
IN_PROGRESS
AUDIT
BLOCKED
PASSED
PASSED_WITH_CONDITIONS
FAILED
FROZEN
```

### DRAFT

Scope is being prepared. Implementation must not begin.

### READY

The Phase Spec has enough information to begin implementation.

### IN_PROGRESS

Implementation is active.

### AUDIT

Feature work is frozen except for audit-driven fixes.

### BLOCKED

A dependency or architectural conflict prevents safe progress.

### PASSED

The audit gate passed with no blocking findings.

### PASSED_WITH_CONDITIONS

Only explicitly accepted non-blocking findings remain.

### FAILED

The gate failed. New feature work is prohibited until the blocking findings are resolved and the phase is re-audited.

### FROZEN

The phase baseline has been committed/tagged and is the accepted starting point for subsequent work.

---

## 6. Required Phase Spec

Every implementation phase must have a written execution specification before coding begins.

The spec should contain only what is needed for that phase.

Required sections:

### 6.1 Identity

```text
Phase
Status
Owner
Target baseline
Relevant roadmap entry
```

### 6.2 Goal

One concise statement describing what becomes possible when the phase succeeds.

### 6.3 User-visible success flow

When applicable, show the intended path:

```text
Current state
  ↓
User action
  ↓
YORVA behavior
  ↓
Observable success
```

### 6.4 In scope

Explicit list of authorized capabilities.

### 6.5 Non-goals

Explicit list of capabilities that must not be implemented in the phase.

Non-goals are enforceable scope boundaries, not suggestions.

### 6.6 Architecture impact

Identify:

- owning modules;
- public/local API changes;
- persistence changes;
- Runtime adapter changes;
- security boundary changes;
- expected background operations;
- concurrency implications.

If there is no impact in an area, say so instead of inventing one.

### 6.7 Contracts

List any required updates to:

```text
PROTOCOL.md
RUNTIME.md
DATA_MODEL.md
SECURITY.md
OpenAPI
ADRs
```

### 6.8 Test matrix

Define critical behavior before implementation.

Do not rely on a coverage percentage as the test plan.

### 6.9 Acceptance criteria

Use observable pass/fail statements.

### 6.10 Audit requirements

Reference `AUDIT_STANDARD.md` and add phase-specific checks where required.

---

## 7. Phase planning rule

Do not fully design distant implementation phases in advance.

`ROADMAP.md` may contain candidate deliverables for future phases, but detailed implementation must be written only when the preceding gate provides enough evidence.

The next Phase Spec is prepared from:

```text
current frozen baseline
+
current phase audit
+
remaining technical debt
+
ROADMAP.md
+
current architecture/security contracts
+
new product evidence
```

This prevents speculative architecture from becoming mandatory merely because it was imagined early.

---

## 8. Scope control

During a phase, implementation may not expand scope implicitly.

Examples of invalid scope expansion:

```text
Repository foundation
→ "while here" implement Hermes installation

Hermes detection
→ "while here" build a generic plugin marketplace

Channel binding
→ "while here" add Cloud accounts and organizations
```

Extra functionality is not automatically "bonus progress".

Unapproved functionality can:

- expand attack surface;
- make review harder;
- create incorrect abstractions;
- introduce maintenance obligations;
- weaken the ability to attribute regressions to one phase.

If new work is discovered, classify it as:

```text
REQUIRED_FOR_PHASE
FOLLOW_UP
TECHNICAL_DEBT
FUTURE_ROADMAP
```

Only `REQUIRED_FOR_PHASE` may expand the current implementation, and the Phase Spec must be updated before substantial work begins.

---

## 9. Scope change procedure

A material scope change requires:

1. description of the newly discovered requirement;
2. reason it cannot wait for a later phase;
3. architecture/security impact review;
4. update to Phase Spec;
5. update to test matrix;
6. ADR if the change alters an architectural decision.

Do not silently change the definition of done after implementation has begun.

---

## 10. Implementation discipline

During `IN_PROGRESS`:

- follow `AGENTS.md`;
- keep changes inside phase scope;
- preserve architecture boundaries;
- add tests with behavior changes;
- keep contracts synchronized;
- avoid unrelated dependency upgrades;
- avoid unrelated refactoring;
- keep generated output reproducible;
- make failures explicit rather than masking them.

If a blocking architecture flaw is discovered, the correct action is to stop and resolve it, not to route around it with a local exception.

---

## 11. Verification before audit

A phase must complete normal implementation verification before entering `AUDIT`.

At minimum, run applicable:

```text
build
type-check
lint
unit tests
integration tests
migration tests
adapter contract tests
security-focused tests
critical manual smoke flow
```

A phase with known failing required checks is not ready for audit.

If a required check cannot run, record why and treat it as an audit input.

---

## 12. Audit freeze

When a phase enters `AUDIT`:

- stop unrelated feature development;
- do not begin implementation of the next phase;
- only make changes that address audit findings or make the current phase verifiable;
- do not use the audit period for opportunistic cleanup.

This creates a stable review target.

---

## 13. Audit independence

The same coding agent may collect evidence, but should not be the sole authority deciding that its own work is correct.

Preferred audit order:

1. automated checks;
2. fresh review context or dedicated audit agent;
3. repository diff review;
4. architecture/security review;
5. project owner Gate Decision.

For a solo project, "independence" means using a fresh review pass that starts from repository state and governing documents rather than relying only on the implementation agent's summary.

The project owner retains final authority to accept `PASS WITH CONDITIONS` findings.

---

## 14. Gate decisions

Every phase audit ends in one of three gate decisions.

### PASS

No unresolved blocking findings.

The phase may be baselined and frozen.

### PASS WITH CONDITIONS

Allowed only when every unresolved finding is non-blocking.

Conditions must:

- have an owner;
- have a severity;
- have a target phase or deadline;
- not weaken security, correctness or architecture boundaries;
- not create a dependency required by the immediately following phase unless it will be resolved first.

### FAIL

Required when blocking findings remain.

Examples:

- Critical finding;
- unresolved High finding affecting phase correctness/security/architecture;
- required success flow does not work;
- key acceptance criterion is unmet;
- critical test scenario is untested because implementation cannot support it;
- required documentation/contract is materially inconsistent with code.

`FAIL` means no next-phase feature work.

---

## 15. Fix and re-audit loop

On `FAIL`:

```text
Audit findings
   ↓
Fix plan
   ↓
Implementation fixes
   ↓
Regression verification
   ↓
Re-audit affected areas
   ↓
New Gate Decision
```

Do not erase previous audit reports.

If a second audit is required, record either:

```text
AUDIT-002-v2-...
```

or add a dated re-audit section to the original report, as long as history remains clear.

---

## 16. Baseline freeze

After `PASS` or accepted `PASS WITH CONDITIONS`:

1. ensure working tree contains no unrelated changes;
2. update required architecture/contract docs;
3. record accepted technical debt;
4. create the architecture/phase baseline commit;
5. optionally create a tag;
6. mark the phase `FROZEN`;
7. only then create/finalize the next Phase Spec.

Suggested tag style:

```text
phase-001-baseline
phase-002-baseline
```

Release tags remain separate from phase tags.

---

## 17. Technical debt policy

Technical debt is allowed only when visible.

Each accepted debt item must state:

```text
ID
Description
Severity
Why accepted
Impact
Owner
Target phase / trigger
```

Do not label correctness or security defects as "technical debt" merely to pass a gate.

Critical and blocking High defects cannot be deferred.

---

## 18. Emergency fixes after freeze

If a frozen phase baseline requires an urgent correction before the next phase:

1. classify the defect;
2. fix it in the smallest scope;
3. run affected regression checks;
4. record the baseline amendment;
5. re-run relevant audit dimensions if security, protocol, data or architecture changed.

Do not silently mutate a frozen baseline.

---

## 19. Phase 0 treatment

Phase 0 is the architecture/documentation freeze.

Before Phase 1 repository bootstrap begins, Phase 0 should receive a document-level readiness review covering:

- unresolved technology choices;
- contradictions between architecture documents;
- unresolved security assumptions;
- roadmap/phase numbering consistency;
- missing execution contracts;
- unclear Runtime/Hermes ownership boundaries.

The review may be lighter than a code-bearing phase audit because no implementation baseline exists yet.

---

## 20. Phase 1 treatment

`docs/BOOTSTRAP.md` is the Phase 1 execution contract.

After Bootstrap implementation completes:

```text
build/test verification
   ↓
Phase 1 Audit
   ↓
Gate Decision
   ↓
phase-001-bootstrap-baseline (logical target; final commit/tag selected after the Gate)
   ↓
Phase 2 Spec
```

Do not begin Hermes Discovery implementation before the Phase 1 gate passes and its accepted baseline is frozen. Hermes Installation is Phase 3 and remains prohibited until Phase 2 independently passes and is frozen.

Phase 1 acceptance record:

- final independent re-audit: `docs/phases/audits/AUDIT-001R2-repository-foundation.md` — `PASS`;
- audit-accepted implementation commit: `1b759f443dbbebba4ae61a82c91e92180d7527b0`;
- formal status: `COMPLETE / FROZEN`;
- formal baseline: annotated tag `phase-001-bootstrap-baseline` on the final freeze commit in `main`.

The earlier merge of the Phase 1 implementation tree into `main` remains implementation history only. It did not constitute phase acceptance; acceptance occurred only after the independent `AUDIT-001R2` `PASS` and the subsequent baseline freeze. No history was rewritten.

---

## 21. Phase 2 treatment

`docs/phases/PHASE-002-hermes-discovery.md` is the Phase 2 execution contract.

Phase 2 original acceptance record:

- initial independent audit: `docs/phases/audits/AUDIT-002-hermes-discovery.md` — `FAIL`, preserved;
- final independent re-audit: `docs/phases/audits/AUDIT-002R1-hermes-discovery.md` — `PASS`;
- audit-accepted implementation commit: `b875b2c496269d3bf91097395cae5fb53dc63350`;
- exact-commit CI: GitHub Actions run `31795039599` — `PASS`, including Go race and Windows native/Tauri;
- main merge commit: `0b31ff72f34f743b285eac4bc0bd4c8ced5096c1`;
- historical formal baseline: annotated tag `phase-002-hermes-discovery-baseline` on `a67de04e900bc3ddce99cd76501eec13586082ed`. This tag is immutable and must not be moved or deleted.

Phase 2 amendment 002A1 acceptance record:

- amendment: `docs/phases/amendments/AMENDMENT-002A1-hermes-windows-command-resolution.md` — `ACCEPTED`;
- independent amendment audit: `docs/phases/audits/AUDIT-002A1-hermes-discovery.md` — `PASS`;
- amendment implementation commit: `dbcb54da4bc4bffcff51888426848246a1900ea6`;
- exact-commit CI: GitHub Actions run `31817886874` — `PASS`, including Go race and Windows native/Tauri;
- amendment merge commit: `f786b9d73768de38875f0a57fa7d57c75168ed85`;
- historical r1 compatibility range: `>=0.19.0 <0.21.0`;
- post-freeze amendment 002A4 current compatibility policy: exact `=0.20.2` (targeted re-audit/re-freeze pending);
- formal status: `COMPLETE / FROZEN`;
- current formal baseline: annotated tag `phase-002-hermes-discovery-baseline-r1` on the amendment freeze commit in `main`.

The original tag remains valid historical evidence and is superseded, not rewritten. Phase 3 implementation remains prohibited until `phase-002-hermes-discovery-baseline-r1` exists and is pushed.

---

## 22. Minimum artifacts per implementation phase

Each implementation phase should leave:

```text
Phase Spec
Implementation + tests
Updated contracts/docs
Audit Report
Accepted debt list if any
Baseline commit/tag
```

If one of these is intentionally unnecessary, state why in the audit.

---

## 23. Completion definition

A phase is not complete when "the feature works on the author's machine."

A phase is complete when:

- the authorized capability exists;
- the success flow works;
- acceptance criteria pass;
- required tests pass;
- architecture boundaries remain valid;
- security requirements remain valid;
- data/contracts/documentation are synchronized;
- the audit gate passes;
- the accepted baseline is frozen.

Only then may detailed planning of the next implementation phase begin.

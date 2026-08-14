# YORVA Audit Standard

> Status: Mandatory phase-gate audit standard  
> Applies to: all implementation phase audits and major pre-release reviews  
> Companion document: `PHASE_GOVERNANCE.md`

## 1. Purpose

This document defines how YORVA determines whether a completed phase is safe to freeze and whether the next phase may begin.

The audit is not a code-style review and is not satisfied by a green build alone.

Its purpose is to answer:

> Did this phase meet its stated contract without weakening correctness, security, architecture, data integrity or maintainability?

---

## 2. Audit inputs

An auditor must review the repository state, not only the implementation summary.

Required inputs:

```text
AGENTS.md
Phase Spec
ROADMAP.md
relevant architecture documents
relevant ADRs
implementation diff / commits
source code
migrations
OpenAPI/protocol changes
tests
CI/build results
known technical debt
```

When the phase integrates an external Runtime, include relevant adapter behavior and compatibility evidence.

---

## 3. Audit dimensions

Every code-bearing phase must consider the following dimensions:

1. Scope
2. Correctness
3. Architecture
4. Security
5. Data and persistence
6. Concurrency and lifecycle
7. Protocol and compatibility
8. Testing and verification
9. Maintainability
10. Documentation
11. Dependency/supply-chain impact
12. Operations and diagnostics

A dimension may be marked `N/A` only with a short reason.

---

## 4. Scope audit

Verify that implementation matches the Phase Spec.

Check:

- all required in-scope behavior is implemented;
- non-goals were respected;
- no unrelated feature work was added;
- no speculative infrastructure was introduced;
- no unrelated refactors obscure the phase diff;
- newly discovered requirements were formally added to the Phase Spec when necessary.

Examples of scope findings:

```text
MEDIUM — added unused generic plugin loader during Hermes detection
HIGH   — implemented remote shell endpoint although Cloud control is out of scope
LOW    — introduced unused UI route intended for a future feature
```

---

## 5. Correctness audit

Review behavior against acceptance criteria and realistic failure modes.

Check:

- happy path;
- expected negative paths;
- partial failure handling;
- retry behavior where relevant;
- cancellation where relevant;
- timeout behavior;
- idempotency where repeated commands are possible;
- stale state handling;
- external Runtime changes;
- error normalization.

Do not accept a feature solely because mocked unit tests return the expected value.

---

## 6. Architecture audit

Verify YORVA's dependency and ownership boundaries.

Required direction remains:

```text
React Desktop
    ↓
Node API
    ↓
Application use cases
    ↓
Domain
    ↑
Adapters
```

Check for:

- React invoking Hermes directly;
- Tauri accumulating business logic;
- core/domain importing Hermes-specific types;
- persistence entities leaking through transport APIs;
- adapter implementation details leaking into UI;
- duplicate sources of truth;
- unnecessary interfaces/layers;
- giant service objects;
- circular package dependencies;
- premature plugin architecture.

Architectural violations that would spread into the next phase should normally block the gate.

---

## 7. Security audit

Security is a mandatory gate dimension even for local-only phases.

Review applicable:

### Local daemon

- loopback-only binding;
- authentication enforcement;
- bootstrap/session token handling;
- token leakage through argv/process listings/logs;
- CORS/origin assumptions;
- SSE authentication;
- unsafe filesystem access;
- command argument injection;
- privilege boundaries.

### Secrets

- plaintext SQLite storage;
- plaintext logs;
- API responses exposing secrets;
- error messages exposing secrets;
- temporary-file leakage;
- backup leakage;
- generated diagnostic bundle leakage.

### Runtime execution

- shell-string construction;
- arbitrary command execution;
- untrusted path handling;
- binary/version validation;
- update integrity when applicable.

### Future Cloud phases

Add:

- device identity;
- pairing authorization;
- replay protection;
- command authorization;
- key rotation;
- organization/RBAC boundaries;
- audit integrity.

Any confirmed secret disclosure or unauthenticated management path is Critical unless a narrower severity is strongly justified.

---

## 8. Data and persistence audit

Check:

- migrations work from an empty database;
- migrations work from the prior frozen schema;
- required uniqueness constraints exist;
- foreign keys are correct;
- transaction boundaries are short and explicit;
- no network/external command wait occurs inside long DB transactions;
- deletes preserve required referential behavior;
- Runtime state is not incorrectly duplicated as authoritative YORVA state;
- cached values are identifiable as cache/metadata;
- timestamps and identifiers are consistent;
- destructive migrations are explicit and reviewed.

When SQLite is not touched, mark this dimension `N/A` with evidence.

---

## 9. Concurrency and lifecycle audit

Go and external Runtime control require explicit lifecycle review.

Check:

- conflicting operations;
- duplicate user actions;
- operation re-entry;
- goroutine ownership;
- cancellation;
- shutdown behavior;
- process cleanup;
- resource leaks;
- race-prone shared mutable state;
- lock scope;
- deadlock risk;
- timeout propagation.

Run Go race detection where practical for affected packages.

Examples requiring attention:

```text
start vs stop
restart vs delete
install vs upgrade
config update vs delete
backup vs destructive mutation
Desktop close vs daemon startup
```

---

## 10. Protocol and compatibility audit

When APIs/protocols change, verify:

- OpenAPI remains the local API source of truth;
- request/response models are typed;
- stable error codes are used;
- backwards compatibility assumptions are explicit;
- event names/payloads are documented;
- SSE reconnect behavior is safe;
- version negotiation is present where required;
- Desktop does not depend on undocumented daemon behavior;
- Runtime adapter does not depend on undocumented Hermes internals unless an explicit compatibility fallback has been approved.

A protocol change that silently breaks an already released client is a blocking issue after that contract is declared stable.

---

## 11. Testing audit

The test audit reviews **behavioral risk coverage**, not a single percentage.

Check:

- Phase Spec test matrix exists;
- critical acceptance flows have tests or justified manual verification;
- regression tests accompany bug fixes where practical;
- negative paths are represented;
- external failures are represented;
- mocks do not make impossible states look valid;
- Runtime Core tests use fakes without importing Hermes implementation;
- adapter tests cover error normalization and compatibility behavior;
- migrations are exercised;
- CI runs required checks.

Coverage metrics may be recorded, but they do not replace scenario review.

---

## 12. Maintainability audit

Look for both under-design and over-design.

### Over-design indicators

```text
unused interfaces
one-caller factories
manager/service/provider chains
plugin systems with one plugin
excessive DTO conversion layers
configuration switches for hypothetical futures
```

### Under-design indicators

```text
giant files with unrelated responsibilities
repeated business rules
runtime-specific branches scattered across Core/UI
unbounded global state
copy/pasted security checks
implicit lifecycle ownership
```

Prefer cohesive modules and small reversible abstractions.

---

## 13. Documentation audit

Code and documentation must describe the same system.

Verify affected:

```text
DEVELOPMENT.md
ARCHITECTURE.md
PROTOCOL.md
RUNTIME.md
DATA_MODEL.md
SECURITY.md
ROADMAP.md
Phase Spec
ADRs
OpenAPI
README / developer setup
```

Material drift between documentation and implementation is a gate finding.

A documentation-only mismatch may be Medium, but if the mismatch affects security, protocol or migration behavior it may be High.

---

## 14. Dependency and supply-chain audit

Review new dependencies added by the phase.

Check:

- dependency is actually used;
- standard library could not reasonably replace it;
- maintenance status;
- license compatibility when project licensing is decided;
- transitive dependency cost;
- native/CGo implications;
- security-sensitive behavior;
- version pinning/lockfile state;
- no unrelated mass dependency upgrade.

A new framework or major infrastructure component without an approved requirement should be treated as an architecture/scope finding.

---

## 15. Operations and diagnostics audit

Verify the system can explain failures safely.

Check:

- structured logs exist for important operations;
- identifiers make actions traceable;
- logs redact secrets;
- user-facing errors are actionable;
- internal errors retain enough diagnostic context;
- operation states do not become permanently stuck after predictable failures;
- crash/restart behavior is understood for touched components.

Do not add full observability infrastructure unless the phase requires it.

---

## 16. Severity levels

Use exactly these finding severities:

```text
CRITICAL
HIGH
MEDIUM
LOW
INFO
```

### CRITICAL

Immediate risk to security, user data, system integrity or fundamental trust boundary.

Typical examples:

- secret credential disclosure;
- unauthenticated management execution;
- arbitrary remote command execution;
- deterministic destructive data corruption;
- supply-chain/update path capable of executing unverified malicious code.

Gate effect: **FAIL**.

### HIGH

Serious correctness, architecture, concurrency or security problem that can materially break the current phase or contaminate the next phase.

Examples:

- required workflow unreliable;
- unsafe concurrent lifecycle operations;
- core directly depends on Hermes implementation;
- missing migration constraint allows invalid authoritative state;
- critical acceptance behavior is untested and demonstrably fragile.

Gate effect: normally **FAIL** until resolved.

### MEDIUM

Real engineering issue with bounded impact that does not currently compromise the core trust/correctness model.

Examples:

- duplicated rule likely to diverge;
- incomplete non-critical error path;
- maintainability issue localized to one module;
- documentation drift not affecting security.

Gate effect: fix before gate when practical; otherwise may support `PASS WITH CONDITIONS` if explicitly accepted.

### LOW

Minor issue with small impact.

Examples:

- naming clarity;
- local cleanup opportunity;
- minor diagnostic improvement.

Gate effect: may be deferred.

### INFO

Observation or future consideration that is not currently a defect.

Gate effect: none.

---

## 17. Gate decision rules

### PASS

Allowed when:

- zero unresolved Critical findings;
- zero unresolved blocking High findings;
- all mandatory acceptance criteria pass;
- required verification completed;
- no architecture/security condition prevents the next phase.

### PASS WITH CONDITIONS

Allowed only when:

- zero Critical findings;
- zero unresolved blocking High findings;
- remaining findings are bounded and explicitly accepted;
- each condition has an owner and resolution trigger;
- deferred items do not invalidate the next phase's assumptions.

### FAIL

Required when any of the following is true:

- unresolved Critical finding;
- unresolved blocking High finding;
- phase success flow fails;
- mandatory acceptance criterion fails;
- security trust model is violated;
- architecture boundary is materially broken;
- required migration/data behavior is unsafe;
- audit evidence is insufficient to establish correctness for a critical capability.

---

## 18. Audit evidence

Every meaningful finding should cite concrete evidence such as:

```text
file/path.go:line range
API route
migration number
test name
command output
CI job
reproduction steps
commit/diff
```

Avoid vague findings such as:

> "The architecture feels complicated."

Prefer:

> `internal/runtime/service.go` imports `runtimes/hermes`, causing Core to depend directly on a Runtime implementation and violating `ARCHITECTURE.md` dependency direction.

---

## 19. Required audit report format

Use the following structure:

```markdown
# YORVA Phase XXX Audit

## Phase

## Baseline / Commit

## Auditor

## Date

## Gate Decision
PASS | PASS WITH CONDITIONS | FAIL

## Executive Summary

## Verification Evidence

## Dimension Results

### Scope
PASS | FAIL | N/A

### Correctness
PASS | FAIL | N/A

### Architecture
PASS | FAIL | N/A

### Security
PASS | FAIL | N/A

### Data and Persistence
PASS | FAIL | N/A

### Concurrency and Lifecycle
PASS | FAIL | N/A

### Protocol and Compatibility
PASS | FAIL | N/A

### Testing and Verification
PASS | FAIL | N/A

### Maintainability
PASS | FAIL | N/A

### Documentation
PASS | FAIL | N/A

### Dependencies / Supply Chain
PASS | FAIL | N/A

### Operations / Diagnostics
PASS | FAIL | N/A

## Findings

### Critical

### High

### Medium

### Low

### Info

## Accepted Technical Debt

## Required Fixes Before Next Phase

## Gate Rationale

## Next Step
```

Do not omit empty severity sections silently; writing `None` makes the gate state explicit.

---

## 20. Re-audit rules

A re-audit does not need to repeat every low-risk area if:

- the original audit already passed that dimension;
- fixes did not affect it;
- evidence remains valid.

A re-audit **must** revisit impacted dimensions.

Example:

```text
Fix changes authentication
→ revisit Security + Protocol + relevant Tests

Fix changes migration
→ revisit Data + Correctness + Tests

Fix moves packages/interfaces
→ revisit Architecture + Maintainability + Tests
```

---

## 21. Audit execution with coding agents

When using Codex or another agent to audit:

- instruct it to read governing documents first;
- audit the actual diff/repository state;
- prohibit code modification during the first audit pass;
- require evidence for each finding;
- require severity classification;
- require explicit gate recommendation;
- do not let it weaken tests to make the gate green;
- do not accept "looks good" without verification output.

Prefer a fresh context/agent for the audit rather than the same context that implemented the phase.

After the report is reviewed, a separate fix task may address accepted findings.

---

## 22. Whole-project audits

In addition to per-phase audits, perform a broader project audit at major boundaries such as:

```text
before first public beta
before enabling remote Control Plane
before introducing persistent cloud identity
before enterprise/private deployment
before 1.0
```

A whole-project audit should examine cross-phase issues that local phase audits may miss:

- architecture drift;
- accumulating dependency weight;
- stale/dead abstractions;
- security model coherence;
- migration history;
- protocol compatibility;
- runtime adapter leakage;
- operational failure recovery;
- accepted technical debt accumulation.

A major boundary may be blocked even if every historical phase individually passed.

---

## 23. Current first gate

The first code-bearing audit is:

```text
Phase 1 — Repository foundation / Bootstrap
```

The Phase 1 audit must verify at minimum:

- React does not invoke Hermes;
- Tauri owns only bootstrap/native responsibilities;
- `yorvad` binds loopback only;
- local authentication is enforced;
- bootstrap token does not leak through argv/logs;
- authenticated HTTP works;
- authenticated SSE works;
- SQLite migrates from empty state;
- Node identity is stable across restart as designed;
- Runtime registry contains only skeleton behavior;
- Hermes commands are not executed;
- required Go/React/Rust checks pass;
- repository contains no speculative Cloud or plugin infrastructure.

Only after this gate passes may Phase 2 detailed implementation proceed.

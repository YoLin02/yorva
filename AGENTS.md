# AGENTS.md — YORVA Repository Rules

## Project intent

YORVA is a local-first AI Runtime deployment and control platform. Hermes Agent is the first supported Runtime, not the architecture itself.

Read these files before making architectural changes:

- `docs/DEVELOPMENT.md`
- `docs/ARCHITECTURE.md`
- `docs/PROTOCOL.md`
- `docs/RUNTIME.md`
- `docs/DATA_MODEL.md`
- `docs/SECURITY.md`
- `docs/PHASE_GOVERNANCE.md`
- `docs/AUDIT_STANDARD.md`
- relevant ADRs under `docs/adr/`


## Phase governance

Implementation work is phase-gated.

Before coding a phase, read its current Phase Spec. `ROADMAP.md` alone does not authorize implementation.

During a phase:

- stay inside explicit scope;
- treat non-goals as hard boundaries;
- do not begin next-phase features early;
- record newly discovered future work instead of implementing it opportunistically.

When implementation is complete, stop feature work and enter audit. Follow `docs/AUDIT_STANDARD.md`.

A failed gate blocks next-phase implementation. Do not bypass an audit finding by weakening tests, changing acceptance criteria after the fact, or labeling a correctness/security defect as technical debt.

Prefer a fresh review context/agent for the audit pass. The implementation agent's own completion summary is not sufficient evidence.

## Priority order

When tradeoffs exist, optimize in this order:

1. correctness;
2. security;
3. simplicity;
4. maintainability;
5. clear ownership/boundaries;
6. testability;
7. user experience;
8. extensibility;
9. performance after measurement.

Do not trade the first six for speculative extensibility.

## Current technology baseline

Desktop:

```text
Tauri 2
React
TypeScript
Vite
```

Local Node / Runtime adapters:

```text
Go
SQLite
```

Future optional Control Plane:

```text
Go
PostgreSQL
```

Do not introduce Java/Spring, Node backend frameworks, Python backend services, Redis, Kafka, or Kubernetes without an approved ADR and a concrete requirement.

## Do not over-engineer

Implement the smallest complete design that satisfies the current task.

Do not add:

- interfaces with no real substitution need;
- generic frameworks for one caller;
- dependency injection containers by default;
- plugin infrastructure before multiple Runtime implementations need it;
- repositories/services/managers merely to match a pattern;
- message brokers for in-process work;
- caches without a measured need;
- speculative configuration flags.

Prefer a small reversible design.

## Architectural invariants

Required direction:

```text
React Desktop
    ↓
Node HTTP contract
    ↓
Application use cases
    ↓
Domain
    ↑
Runtime / persistence / secret adapters
```

Forbidden:

```text
React → Hermes CLI
React → Hermes config files
Domain → Hermes implementation
Domain → Tauri
Hermes adapter → UI
remote command → arbitrary shell
```

## Hermes rule

Hermes-specific behavior belongs under `runtimes/hermes` or a clearly Hermes-owned integration package.

Integration priority:

1. documented official API;
2. documented official programmatic protocol;
3. documented official CLI;
4. narrow compatibility fallback.

Do not:

- import Hermes internal Python modules;
- fork or patch Hermes to make YORVA work;
- depend on undocumented Hermes database schemas;
- parse human-readable output when structured output exists;
- let Hermes-specific fields leak into core models without a documented reason.

## Runtime abstraction rule

Generalize stable YORVA concepts. Isolate unstable Runtime-specific concepts.

Do not invent a generic feature because a hypothetical future Runtime might need it.

If a second real Runtime proves the current contract insufficient, update `docs/RUNTIME.md` and add an ADR when the change is architectural.

## Source of truth

Hermes owns Hermes state.

YORVA owns YORVA management state.

SQLite may contain normalized metadata/cache, but code must not treat stale YORVA metadata as authoritative when the Runtime can be queried.

## Change discipline

Before editing:

1. identify the owning module;
2. inspect existing code and tests;
3. decide whether the change is Core or Runtime-specific;
4. make the smallest coherent change;
5. update/add tests;
6. update contract docs if public behavior changes.

Do not perform unrelated refactors.

Do not rename unrelated files/types while fixing another issue.

Do not format unrelated files.

Do not delete user work to simplify a patch.

## API discipline

Transport DTOs are not domain entities.

Public/local APIs must:

- use typed request/response bodies;
- return stable error codes;
- avoid leaking raw internal errors;
- never return secret plaintext through ordinary read APIs;
- follow `docs/PROTOCOL.md`.

Do not make UI behavior depend on matching human-readable error text.

## Long-running work

Installation, upgrade, backup, restore and channel authentication are Operations.

Do not hold an HTTP request open for the full external workflow when an Operation can represent it.

Do not hold a database transaction open while:

- downloading;
- running external commands;
- waiting for QR login;
- waiting for a network service;
- waiting for a Runtime process.

## Concurrency

Protect conflicting mutations at the narrowest scope.

Examples:

- Runtime install vs upgrade;
- Instance start vs stop/restart;
- Instance delete vs config update;
- restore vs other Instance mutations.

Avoid global locks unless the resource is truly global.

All goroutines with non-trivial lifetime must have explicit ownership and cancellation.

Use `context.Context` for cancellable Go work.

## Security rules

`yorvad` is security-sensitive.

Never:

- listen on `0.0.0.0` by default;
- expose an unauthenticated management endpoint;
- log secrets;
- store secrets in SQLite plaintext;
- create generic shell/file-system remote APIs;
- run permanently as Administrator/root only for convenience;
- weaken Tauri CSP to solve a development issue without review.

Read `docs/SECURITY.md` before modifying authentication, pairing, secrets, process execution or update code.

## Go guidelines

Prefer cohesive domain packages over dumping grounds.

Avoid package names like `common`, `utils`, `helper`, `manager` unless the contents genuinely represent that concept.

Prefer explicit constructors and dependencies over global mutable state.

Do not use `panic` for ordinary application errors.

Keep external command execution behind adapter-owned, argument-safe code. Do not construct shell strings when direct argv execution is possible.

## React guidelines

Use functional components and TypeScript.

Use:

- TanStack Query for daemon/server state;
- local React state for local interaction;
- Zustand only for shared client-only state;
- form/schema libraries only when they reduce complexity.

Do not put Runtime integration logic into React components.

Do not duplicate Node resource state into multiple stores.

## Tauri guidelines

Tauri is a native shell and secure bridge, not the main business backend.

Rust code is appropriate for:

- daemon bootstrap/lifecycle integration;
- native packaging/updating;
- secure OS capabilities;
- narrowly scoped native features.

Keep ordinary management/business logic in Go.

## Database rules

Every schema change uses a migration.

Migrations must be deterministic and tested from an empty database.

Use explicit uniqueness constraints for invariants.

Do not silently destroy user state.

Do not put Runtime secret plaintext in the database.

## Dependency policy

Before adding a dependency, verify:

1. standard library cannot solve the problem clearly;
2. dependency is maintained;
3. transitive cost is reasonable;
4. security role is understood;
5. it solves a current requirement.

Do not upgrade unrelated dependencies in a feature patch.

## Testing

A bug fix should include a regression test when technically practical.

Testing priority:

1. domain/application unit tests;
2. Runtime adapter contract tests;
3. persistence migration/repository tests;
4. protocol tests;
5. integration tests;
6. critical Desktop component/E2E flows.

Do not mock every layer. Use small fakes for Runtime contracts in core tests.

Critical behavior includes:

- Runtime detection;
- unsupported Runtime/version;
- create/delete Instance;
- duplicate native identity reconciliation;
- start/stop/restart;
- operation state transitions;
- conflicting operation protection;
- secret redaction;
- channel QR/login state;
- adapter error normalization.

## Documentation and ADRs

Update documentation when changing:

- architecture;
- API/protocol;
- Runtime contract;
- schema;
- security behavior;
- installation/update behavior.

Architectural decisions go under `docs/adr/` and include:

- Context;
- Decision;
- Alternatives;
- Consequences.

## Completion criteria

Do not report a task complete only because it compiles.

Before completion, run the relevant subset of:

- Go tests;
- Go vet/lint configured by repository;
- TypeScript typecheck;
- frontend lint/tests;
- migration tests;
- build/package check where relevant.

State clearly if a required check could not be run and why.

Never weaken tests or security controls simply to get a green build.

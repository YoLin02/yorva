# ADR-0001: YORVA is Local-First with an Optional Control Plane

- Status: Accepted
- Date: 2026-08-13

## Context

YORVA must make Hermes and future AI Runtimes easier to deploy and manage. A cloud-only architecture would simplify central administration but would make local use depend on accounts, connectivity and external infrastructure, while Runtime credentials and enterprise data can be sensitive.

## Decision

YORVA core functionality is local-first.

V0.1 consists of:

```text
Desktop + yorvad + local database + Runtime adapter
```

No YORVA Cloud account is required for local management.

A future Control Plane is optional and adds remote fleet capabilities. It must not become a hidden dependency for local Runtime operation.

Runtime secrets remain local by default.

## Alternatives considered

### Cloud-first SaaS

Rejected for V0.1 because it increases infrastructure, security, account and availability requirements before the local workflow is proven.

### Desktop-only without daemon

Rejected because Runtime processes, background operations and future remote management should survive Desktop UI lifecycle and need a stable management owner.

## Consequences

Positive:

- low entry barrier;
- enterprise-friendly data locality;
- offline local management;
- simpler initial deployment;
- future cloud remains optional.

Costs:

- Desktop/daemon bootstrap security must be designed carefully;
- local upgrades/migrations must be robust;
- future Cloud requires synchronization/reconciliation rather than owning all state.

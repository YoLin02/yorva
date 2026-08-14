# ADR-0003: Runtime Adapter Boundary, No Hermes Fork

- Status: Accepted
- Date: 2026-08-13

## Context

YORVA launches with Hermes as its first Runtime but intends to support additional Runtimes later. Directly embedding Hermes-specific assumptions throughout Desktop/Core would make future expansion expensive. Forking Hermes would also make upstream updates costly.

Hermes already exposes official profiles and programmatic/management surfaces suitable for external integration.

## Decision

Hermes remains an independent upstream Runtime.

YORVA integrates through a Hermes adapter using, in priority order:

1. documented official API/management surfaces;
2. documented official programmatic protocols;
3. official CLI;
4. narrow compatibility fallback where no structured surface exists.

Core YORVA concepts remain Runtime-neutral. For V0.1, a YORVA Instance maps to a Hermes Profile inside the adapter.

No dynamic plugin framework is built in V0.1. Hermes adapter wiring is compile-time.

## Alternatives considered

### Fork Hermes

Rejected because it creates long-term maintenance, security and upgrade burden.

### Make YORVA Core Hermes-specific, generalize later

Rejected because basic boundaries such as Instance/Runtime/Channel can be kept clean at low cost now.

### Build a full plugin SDK immediately

Rejected as speculative over-engineering with only one Runtime.

## Consequences

Positive:

- upstream Hermes can update independently;
- compatibility code is isolated;
- future Runtime work has a clear boundary;
- Core tests can use fake Runtime implementations.

Costs:

- adapter must normalize version/API differences;
- some Hermes features may require multiple official integration surfaces rather than one uniform protocol;
- the contract may need revision when a real second Runtime is implemented.

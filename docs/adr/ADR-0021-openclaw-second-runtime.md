# ADR-0021: OpenClaw as the Second Local Runtime

- Status: Accepted for Phase 9 implementation by the Owner on 2026-09-10; final integration qualification in progress
- Date: 2026-09-08
- Owner direction: OpenClaw 2.0 latest stable; deliver Phase 9 quickly from the frozen Phase 8 baseline with focused gates
- Spec: [Phase 9](../phases/PHASE-009-openclaw-second-runtime.md)

## Context

P8 is frozen. The Owner selected OpenClaw to test Runtime neutrality before Node/Control/Fleet product work and authorized implementation on 2026-09-10. The B0 refresh found official GitHub and npm latest agree on OpenClaw `2026.9.3`, superseding the plan's researched `2026.9.2`. The new release requires Node `>=24.16.0 <25 || >=26.1.0`, so it uses a separately resolved compatible Node rather than Hermes' private Node 22. OpenClaw has documented Windows CLI/Gateway support and named profiles, with separate Gateway lifecycle and per-Gateway agents.

The registry already provides discovery and focused management capabilities, but the application inventory still injects one Hermes Profile source. Instance inventory, lifecycle, model/channel routing and recovery contain Hermes-specific target resolution. Desktop also uses Hermes-specific inventory query keys. A second real Runtime now justifies replacing those assumptions; it does not justify a plugin framework.

## Decision

1. Add one compiled-in OpenClaw adapter under `services/node/internal/runtime/openclaw`, kind `openclaw`.
2. Map an independently managed YORVA Instance to an isolated OpenClaw Gateway profile. Agents sharing one Gateway are not separate lifecycle units in YORVA.
3. Put the minimum actual instance read/mutation contracts in the Runtime bundle. Move native naming/protection and error translation behind the adapter. Resolve every Instance target through its accepted installation and Runtime kind. Reuse existing Operations, locks and persistence constraints.
4. Use documented management APIs/protocols where suitable, and documented CLI for offline setup or native service control. Parse structured output when offered. No internal OpenClaw module imports, internal SQLite access, raw CLI forwarding or arbitrary configuration API.
5. Use native Windows x64 foreground `gateway run` with canonical named profiles and no native service installation. B0 proved two simultaneous authenticated Gateways at medium integrity and official `gateway stop --force --json` with stopped readback. The release's unmanaged `gateway restart --json` rejected its own listener/lock identity, so YORVA implements Restart as official Stop, confirmed STOPPED, then the same Start path. Complete composed-restart and deletion qualification remains part of B0/G1. During startup a Windows Job owns the entire process tree; failure/cancellation closes it. Only after the exact profile's authenticated status succeeds does YORVA remove kill-on-close and release the process handle, transferring lifetime to OpenClaw without a retained goroutine or persistent PID authority. Closing Desktop/Node therefore leaves a successfully started Runtime running. No login task, Startup entry, native service, or silent elevated helper is installed.
6. Preserve OpenClaw authority for its native config/state/credentials. YORVA stores management metadata only and does not copy the Hermes `.env` exception to OpenClaw. The baseline integration lets official OpenClaw resolve its own Gateway auth without returning the token to YORVA clients. An optional new credential writer requires its exact authority and safe input surface to be documented here and in SECURITY before coding.
7. Derive capabilities from qualified bundle features and actual target support. Unsupported OpenClaw requests must never fall through to Hermes. Generic UI uses Runtime identity and capabilities; Runtime-specific installation details may remain specialized.
8. Reuse the existing SQLite tables first. A new field/ownership record must have a concrete need and appropriate migration or adapter-owned record; no persistent PID or copied native-state database becomes authority.
9. Treat an absent, never-managed Runtime as optional during recovery. Missing or unknown previously accepted Runtime state cannot be masked by a healthy other Runtime. Keep healthy-side management available and report recovery needs accurately.

This is a scoped extension of ADR-0003 and ADR-0004. Local HTTP/SSE, authentication, Tauri ownership, Hermes compatibility and frozen tags remain governed by their existing contracts.

## Alternatives

- **A second Hermes-shaped implementation in Core:** rejected; it repeats the single-runtime coupling the phase is meant to validate.
- **One Instance per OpenClaw agent:** rejected for the lifecycle MVP because multiple agents share a Gateway lifecycle.
- **Container Fleet:** deferred; the official feature is experimental, depends on Docker/Podman and currently lists Windows as untested. Native Gateway profiles address the current single-user local need.
- **A dynamic plugin SDK or separate adapter service:** rejected without evidence that two compiled adapters cannot meet the requirement.
- **Require a full managed installer and every Hermes feature:** unnecessary for second-runtime validation. The mandatory flow can use an official preinstalled CLI; specific safe extensions remain allowed by the Spec.

## Consequences

The common application path becomes testable with two real adapters and small fakes. Runtime-specific semantics stay in adapters, and P8 release infrastructure can be reused. Both Runtime identities can safely contain the same native instance name.

P9 will not claim feature parity. Release pinning and structured-output fixtures need upkeep as upstream evolves. B0 must settle Windows service ownership and login behavior with actual evidence; online documentation alone does not prove the implementation path. That evidence may refine this decision before dependent code is written, without creating another implementation phase.

Named profiles isolate management state within one local-user trust boundary. They are not a hostile multi-tenant security sandbox.

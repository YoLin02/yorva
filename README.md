# YORVA

YORVA is a **local-first, Runtime-neutral Agent Runtime management infrastructure**.

It starts with Hermes Agent as the first supported Runtime, but Hermes is not the boundary of the architecture. YORVA is designed to make AI Runtime deployment, Instance management, configuration, lifecycle, recovery and future multi-Node control available through one normalized management layer.

Current product path:

```text
YORVA Desktop
    ↓
yorvad / YORVA Node
    ↓
Runtime Contract
    ↓
Runtime Adapter
    ↓
Hermes and future AI Runtimes
    ↓
Runtime Instances
```

The accepted local baseline has progressed through Phase 7. **Phase 8 — Local product
hardening** is in progress, focused on Windows installation and update integrity, supported
data migration, restart recovery, sanitized diagnostics and bounded stability evidence
before distributed management is introduced. Without production Windows signing material,
the current phase is capped at an internal candidate and cannot claim public-release
readiness.

Longer term, the roadmap validates the Runtime abstraction with a real second Runtime before evolving `yorvad` into a headless YORVA Node, then adds multi-Node YORVA Control, Fleet configuration/Desired State and enterprise governance in that order.

YORVA is not intended to become a general Agent workflow/RAG/prompt-authoring platform. Its core responsibility is Runtime and Node management infrastructure that upper-layer Agent platforms can rely on without understanding each Runtime's native operational details.

## Repository layout

```text
apps/desktop/       Tauri 2 + React desktop application
services/node/      Go yorvad Runtime/Node management daemon
api/                OpenAPI source for generated management contract types
runtimes/hermes/    Independent Hermes Runtime adapter boundary
docs/               Architecture, Runtime contract, security, roadmap and phase governance
```

## Development prerequisites

- Git 2.55.0
- Node.js 22.23.1
- pnpm 11.15.1
- Go 1.26.6
- Rust and Cargo 1.97.1
- Tauri CLI 2.11.4
- Windows C++ build tools and WebView2

The authoritative setup and validation commands are documented in
[docs/DEVELOPMENT.md](docs/DEVELOPMENT.md). Product direction is documented in
[docs/ROADMAP.md](docs/ROADMAP.md), product support and recovery behavior in
[docs/PRODUCT_SUPPORT.md](docs/PRODUCT_SUPPORT.md) and [docs/RECOVERY.md](docs/RECOVERY.md),
and architectural boundaries in [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) and
[docs/RUNTIME.md](docs/RUNTIME.md).

## Licensing

Licensing has not yet been selected. No license is granted by this repository until the repository owner adds an explicit license.

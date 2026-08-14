# YORVA

YORVA is a local-first deployment and control platform for AI runtimes. Hermes Agent is the first supported runtime, not the boundary of the architecture.

The repository is currently establishing its Phase 1 bootstrap baseline. No Hermes discovery, installation, CLI integration, profile management, or other post-bootstrap features are implemented in this phase.

## Repository layout

```text
apps/desktop/       Tauri 2 + React desktop application
services/node/      Go yorvad local management daemon
api/                OpenAPI source for generated Desktop contract types
runtimes/hermes/    Future independent Hermes adapter boundary
docs/               Architecture, protocol, security, and phase governance
```

## Development prerequisites

- Git 2.55.0
- Node.js 22.23.1
- pnpm 11.15.1
- Go 1.26.6
- Rust and Cargo 1.97.1
- Tauri CLI 2.11.4
- Windows C++ build tools and WebView2

The authoritative setup and validation commands are documented in [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md). Phase scope is defined by [docs/BOOTSTRAP.md](docs/BOOTSTRAP.md).

## Licensing

Licensing has not yet been selected. No license is granted by this repository until the repository owner adds an explicit license.

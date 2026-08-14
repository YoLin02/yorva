# ADR-0002: React/Tauri Desktop and Go Management Backend

- Status: Accepted
- Date: 2026-08-13

## Context

YORVA needs a modern Desktop UI, a lightweight local daemon, Runtime integration, and eventually an optional remote Control Plane. Early architecture considered Java/Spring Boot for the future server, but this adds a second heavyweight backend ecosystem before its enterprise advantages are required.

## Decision

Use:

```text
Desktop UI:      React + TypeScript + Vite
Desktop shell:   Tauri 2
Local daemon:    Go
Runtime adapters: Go
Local DB:        SQLite
Future server:   Go modular monolith
Future cloud DB: PostgreSQL
```

Exact dependency patch versions are pinned at repository initialization rather than frozen in the ADR.

The project prefers single-binary backend deployment and small focused dependencies.

## Alternatives considered

### Electron

Provides a mature desktop ecosystem but carries a larger runtime footprint. Tauri better matches YORVA's lightweight product goal.

### Vue

Technically suitable, but React was selected as the Desktop frontend standard for YORVA.

### Java + Spring Boot Control Plane

Strong enterprise ecosystem, but not justified for the initial Control Plane workload of WebSocket Node connections, commands, inventory and APIs. Go keeps the backend stack unified and operationally smaller.

### Node.js backend

Technically viable but would introduce another backend runtime while Go is already required for `yorvad`.

## Consequences

Positive:

- one primary backend language;
- small daemon/server footprint;
- natural concurrency model for future Node connections;
- simpler CI/deployment;
- React ecosystem for UI.

Costs:

- Tauri adds a small Rust/native layer;
- enterprise features available out-of-the-box in Spring must be implemented with focused Go libraries later if actually needed.

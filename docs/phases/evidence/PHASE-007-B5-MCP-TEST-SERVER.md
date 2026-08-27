# Phase 7 B5 — YORVA MCP Test Server

## Purpose

This YORVA-owned local server verifies the production MCP management lifecycle before
a third-party MCP preset is enabled. It is the first production-visible reviewed
Preset and does not authorize any third-party endpoint.

The fixed test identity is:

```text
preset:   yorva-mcp-test
endpoint: https://mcp-test.yorva.invalid/mcp
tool:     yorva_ping
auth:     none
```

The hostname is intentionally non-routable. Normal `yorvad` starts an ephemeral IPv4
loopback server and its private Hermes adapter client routes only that exact fixed
identity to the listener. The listener address never enters the API, Desktop or Hermes
Profile. The production Runtime catalog contains only this test Preset.

## Protocol behavior

The server accepts only bounded `POST /mcp` JSON-RPC requests needed by the
connection check:

1. `initialize`, returning protocol `2025-06-18` and one bounded session ID;
2. `notifications/initialized`, requiring that session ID;
3. `tools/list`, requiring that session ID and returning only `yorva_ping`.

It rejects authorization values, unknown methods, wrong paths, oversized bodies and
invalid session use. It has no command,
argv, environment, arbitrary
header, arbitrary URL, file, package, bootstrap or caller-provided JSON surface.

## Lifecycle gate

Run:

```text
go test -tags mcpqualification ./internal/app -run TestYORVAManagesProductionLocalTestMCPAcrossProfiles -count=1 -v
```

The gate uses the authenticated typed local HTTP routes, real SQLite repositories,
application MCP Operations, Hermes Profile MCP manager, atomic Profile writer,
HTTPS MCP handshake and authoritative Profile read-back. It verifies:

```text
authenticated production Runtime definition read
→ create default-Profile binding
→ write fixed Hermes configuration
→ initialize and list tools
→ READY authoritative read-back
→ YORVA_MANAGED binding ownership
→ update Tool Scope through the binding resource
→ CONFIGURED authoritative read-back
→ explicit retest and READY authoritative read-back
→ bind the same definition to work Profile
→ remove the default binding
→ remove the work binding
→ absence read-back and empty binding records
```

Success requires two complete three-request MCP handshakes. No adapter fake is
used for the MCP definition, HTTP mutation, Profile mutation, protocol probe or
read-back.

Passing this gate exposes `yorva-mcp-test` through the typed `MCPRead`, `MCPMutate`,
and `MCPTest` Runtime capabilities. The application and Runtime contracts contain only
the reviewed-Preset path; no dormant custom Definition, stdio command/argv, environment,
header, endpoint, path or secret-map mutation interface is retained. Third-party Presets
still require their own review; unknown preset IDs continue to fail at the adapter registry
boundary.

# Phase 7 B5 — YORVA MCP Test Server

## Purpose

This fixture verifies YORVA's MCP management lifecycle before a third-party MCP
preset is enabled. It is qualification infrastructure, not a product MCP server
and not an additional architecture decision.

The fixed test identity is:

```text
preset:   yorva-mcp-test
endpoint: https://mcp-test.yorva.invalid/mcp
tool:     yorva_ping
auth:     fixed static-Bearer credential class
```

The hostname is intentionally non-routable. Tagged qualification tests route
only that exact hostname to an isolated `httptest` TLS server. Normal `yorvad`
and Desktop builds do not contain the tagged Profile-manager composition, and
the production Runtime MCP catalog remains unchanged.

## Protocol behavior

The server accepts only bounded `POST /mcp` JSON-RPC requests needed by the
connection check:

1. `initialize`, returning protocol `2025-06-18` and one bounded session ID;
2. `notifications/initialized`, requiring that session ID;
3. `tools/list`, requiring that session ID and returning only `yorva_ping`.

It requires the qualification credential through the descriptor-owned Bearer
shape and rejects any other authorization value, unknown methods, wrong paths,
oversized bodies and invalid session use. The test verifies that the credential
is written only to the exact Profile `.env`, while `config.yaml` contains only
the fixed `${MCP_YORVA_TEST_API_KEY}` interpolation template. It has no command,
argv, environment, arbitrary
header, arbitrary URL, file, package, bootstrap or caller-provided JSON surface.

## Lifecycle gate

Run:

```text
go test -tags mcpqualification ./internal/app -run TestYORVAManagesQualificationMCPAcrossProfiles -count=1 -v
```

The gate uses the authenticated typed local HTTP routes, real SQLite repositories,
application MCP Operations, Hermes Profile MCP manager, atomic Profile writer,
HTTPS MCP handshake and authoritative Profile read-back. It verifies:

```text
authenticated Runtime definition read
→ create default-Profile binding
→ write and read back the Profile credential
→ write fixed Hermes configuration
→ initialize and list tools
→ READY authoritative read-back
→ YORVA_MANAGED binding ownership
→ bind the same definition to work Profile
→ remove the default binding
→ remove the work binding
→ absence read-back and empty binding records
```

Success requires two complete three-request MCP handshakes. No adapter fake is
used for the MCP definition, HTTP mutation, Profile mutation, protocol probe or
read-back.

Passing this gate opens the typed `MCPRead`, `MCPMutate`, and `MCPTest` Runtime
capabilities independently of catalog size. The normal product catalog remains empty
until a real provider preset is reviewed; unknown preset IDs continue to fail at the
adapter registry boundary.

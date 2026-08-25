// Package mcpmanagement contains closed, parser-only Hermes MCP management
// primitives for Phase 7.
//
// The package has no network, command, child-process, file, or Hermes mutation
// implementation. A caller can select only a compile-time reviewed descriptor;
// the initial reviewed set remains deliberately empty until at least one HTTPS
// preset completes descriptor and exact-version qualification under ADR-0014.
package mcpmanagement

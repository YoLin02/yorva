// Package mcpmanagement contains closed Hermes MCP management and qualification
// primitives for Phase 7.
//
// Product selection remains parser-only: a caller can select only a compile-time
// reviewed descriptor, and the reviewed set is deliberately empty. The separately
// named GitHub qualification client can contact one compiled endpoint but is not
// wired to the registry or product capabilities. The package has no command,
// child-process, file, or Hermes mutation implementation.
package mcpmanagement

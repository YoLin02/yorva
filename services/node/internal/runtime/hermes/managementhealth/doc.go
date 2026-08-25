// Package managementhealth contains bounded, parser-only Hermes management
// primitives. It deliberately does not acquire log files, invoke Hermes, or
// expose a product capability; callers must provide already-bounded response
// or snapshot bytes and map the safe projection into the shared contract.
package managementhealth

// SourceAcquisitionQualified documents the Phase 7 B1 decision. No function in
// this package reads a path or runs `hermes logs`.
const SourceAcquisitionQualified = false

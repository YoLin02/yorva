# Hermes Runtime Adapter boundary

This directory reserves the future independent ownership boundary for the Hermes Runtime Adapter.

During Phase 1 it contains documentation only. The compile-time Go descriptor and registration skeleton live inside the repository's single Go module at `services/node/internal/runtime/hermes`.

No Hermes discovery, installation, CLI invocation, profile management, PATH lookup, Python inspection, configuration parsing, or filesystem probing is implemented in Phase 1. Real detection begins only in an authorized later phase.

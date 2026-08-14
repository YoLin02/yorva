# YORVA Development Documents

Read in this order:

1. `AGENTS.md`
2. `docs/DEVELOPMENT.md`
3. `docs/ARCHITECTURE.md`
4. `docs/SECURITY.md`
5. `docs/PROTOCOL.md`
6. `docs/RUNTIME.md`
7. `docs/DATA_MODEL.md`
8. `docs/PHASE_GOVERNANCE.md`
9. `docs/AUDIT_STANDARD.md`
10. `docs/ROADMAP.md`
11. `docs/adr/ADR-0001-local-first.md`
12. `docs/adr/ADR-0002-technology-stack.md`
13. `docs/adr/ADR-0003-runtime-adapter.md`
14. `docs/adr/ADR-0004-protocol-and-connectivity.md`
15. `docs/BOOTSTRAP.md`

Templates:

- `docs/phases/PHASE_TEMPLATE.md`
- `docs/phases/audits/AUDIT_TEMPLATE.md`

Phase readiness and gate records:

- `docs/phases/audits/AUDIT-000-phase0-readiness.md`
- `docs/phases/audits/AUDIT-001-repository-foundation.md`
- `docs/phases/audits/AUDIT-001R1-repository-foundation.md`
- `docs/phases/audits/AUDIT-001R2-repository-foundation.md` — final Phase 1 `PASS`
- `docs/phases/audits/AUDIT-002-hermes-discovery.md` — initial Phase 2 `FAIL`, preserved
- `docs/phases/audits/AUDIT-002R1-hermes-discovery.md` — final Phase 2 re-audit `PASS`
- `docs/phases/PHASE-002-hermes-discovery.md` — Phase 2 `PASSED`; exact-commit CI, merge and freeze pending

The architecture, security, protocol, Runtime and ADR documents define the engineering baseline. `PHASE_GOVERNANCE.md` defines how work advances between phases. `AUDIT_STANDARD.md` defines the mandatory audit and gate rules.

`BOOTSTRAP.md` is the execution contract for **Phase 1 — Repository foundation**.

Phase 1 is frozen at `phase-001-bootstrap-baseline`. Phase 2 is limited to Hermes Discovery & Compatibility; Hermes Installation is reserved for Phase 3.

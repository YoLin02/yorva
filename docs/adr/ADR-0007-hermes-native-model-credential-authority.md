# ADR-0007: Hermes-Native Model Credential Authority for the Windows Consumer MVP

- Status: Accepted
- Date: 2026-08-19
- Owner approval: 2026-08-19 (Phase 5 D1-D6 and this ADR)
- Related: ADR-0003, Phase 5 `PHASE-005-models-credentials.md`

## Context

The frozen security baseline places instance/model provider credentials behind an OS-backed YORVA `SecretStore`. Hermes `0.20.2`, however, owns Profile model configuration and its official credential persistence. Keeping a second YORVA copy would create two credential authorities, require secret injection on every Hermes invocation, and make a Hermes Profile unusable after YORVA exits unless YORVA also modifies Runtime startup behavior.

Phase 5 needs one explicit authority for model provider credentials while preserving the existing rules against secret disclosure, arbitrary file access and global environment mutation.

## Decision

For the Windows consumer MVP, a Hermes Profile's model provider credential is Runtime-native state. The selected Profile's official Hermes credential store is its sole authority.

YORVA may set, replace, delete and query safe status only through an exact, pinned, qualified official Hermes `0.20.2` surface. Qualification must prove Profile targeting and isolation, non-interactive operation, bounded output and that secret material does not enter argv, output or logs. If no such surface exists, Phase 5 stops; this ADR does not authorize a compatibility writer.

The following rules are mandatory:

- YORVA does not keep a duplicate Hermes model credential in `SecretStore`, SQLite, `secret_refs`, an Operation, an event, a log, diagnostics or Desktop storage.
- YORVA production code does not open, parse, append or rewrite Hermes `.env` files. A Hermes-owned `.env` may be the upstream implementation detail of the qualified official surface.
- Ordinary reads and mutation responses expose safe configured/status metadata only and never echo a credential.
- A credential is scoped to the exact authoritative Hermes Profile selected from the Instance `nativeId`; Profile A material must not be visible to Profile B.
- Credentials do not enter command argv, URLs, Windows user/system environment variables or ambient child environments.
- Unknown output or uncertain native state fails closed. YORVA does not infer configured state or delete a prior credential as rollback.

This is an explicit at-rest tradeoff. Hermes may store its Profile credential in a file that is not an OS credential vault. YORVA does not represent that storage as OS-secure and does not broaden its own file access to compensate. The V0.1 local-user trust model and Hermes' own Profile storage protections apply.

The YORVA `SecretStore` boundary remains for YORVA-owned secrets, including future device/Cloud credentials, and for later Runtime/channel designs that do not have an approved native credential authority. This ADR does not authorize channel credentials, secret synchronization or a general Runtime-native exception.

## Alternatives considered

### Store a second copy in YORVA SecretStore

Rejected. It creates two authorities, requires synchronization and injection behavior, and prevents Hermes from independently using its own Profile configuration.

### Let YORVA write or parse Profile `.env` directly

Rejected. It bypasses the official Runtime boundary, couples YORVA to a storage format and creates unsafe read/modify/write and redaction obligations.

### Put provider credentials in Windows user or system environment variables

Rejected. Global variables cannot preserve per-Profile isolation and would mutate user-owned machine state.

### Pass the credential as command argv

Rejected. Process listings, descriptors and diagnostics can disclose argv.

## Consequences

Positive:

- Hermes remains the single source of truth for Hermes Profile model credentials;
- Hermes can use the Profile after YORVA exits;
- YORVA holds the secret only during the authorized mutation;
- no schema migration or `secret_refs` row is required for these credentials.

Costs and constraints:

- upstream Hermes storage may provide weaker at-rest protection than an OS vault;
- Phase 5 must qualify the pinned official surface before production mutation;
- credential status is limited to safe metadata exposed by that surface;
- a future change to this authority requires a new ADR or an explicit superseding decision.

This ADR authorizes the Phase 5 credential-authority contract only. It does not itself prove a Provider supported or authorize Phase 6 work.

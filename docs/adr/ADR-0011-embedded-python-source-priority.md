# ADR-0011: Embedded Python and Selectable Artifact Priority

- Status: Accepted
- Date: 2026-08-21
- Owner approval: 2026-08-21 (active task)
- Related: Phase 3 amendments 003A6, 003A7 and 003A8
- Supersedes: ADR-0010's packaged-artifacts-always-first choice, not its pins or URL boundary

## Context

Hermes' official installer asks uv to acquire Python 3.11. The MSI already carries
Hermes, Node.js and npm, but an unreachable Python distribution host can still make a
demo installation fail. Users also need an explicit choice between local packaged
bytes and their configured online transports.

## Decision

Yorva packages the exact uv-managed CPython archive selected by the reviewed Hermes
installer. The Go Hermes adapter, not React or the official script, selects and
verifies the artifact. It creates an Operation-private local uv mirror plus pinned
single-entry download metadata and exposes only their `file://` locations to the
child installer.

The existing closed settings document gains an artifact preference and one Python
archive URL. `bundled-first` remains the default. `online-first` is allowed, with
fallback to packaged bytes only for transport failures. Integrity failures never
fallback. Versions, sizes and hashes remain compiled constants.

## Alternatives considered

### Let uv download Python directly

Rejected. It would bypass Yorva's exact archive verification and make online behavior
dependent on uv's mutable release metadata.

### Extract and ship an already-installed Python directory

Rejected. The official install contract already uses uv's pinned standalone archive;
shipping an expanded tree increases package size and adds a second installation
layout to maintain.

### Add arbitrary local paths or custom hashes

Rejected. That would expand the management API into a generic file execution and
installer surface.

## Consequences

- The MSI grows by approximately 24.5 MiB before MSI compression.
- Python interpreter acquisition works without an external Python download when
  bundled-first is selected.
- Online endpoints remain availability controls only; they cannot change installed
  bytes.
- Dependency packages may still require the configured Python/npm registries.

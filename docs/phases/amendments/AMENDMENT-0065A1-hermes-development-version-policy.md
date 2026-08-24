# YORVA Phase 6.5 Amendment 0065A1 — Hermes Development Version Policy

> Status: APPROVED / IMPLEMENTED CANDIDATE
> Date: 2026-08-24
> Owner: Repository owner / lead developer
> Related: ADR-0012, Amendment 002A4, Phase 6.5

## Authorized change

The developer authorizes YORVA to stop requiring exact Hermes `0.20.2` equality during
the Phase 6.5 development cycle.

- Detect stable `>=0.20.2 <0.21.0` as compatible.
- Apply the same policy to model, credential, lifecycle and Channel targets.
- Allow thirty seconds for the official version probe.
- Recognize the Hermes `0.20.5` Windows launcher layout where
  `%LOCALAPPDATA%\hermes\bin\hermes.exe` aliases the checkout venv launcher.
- Advance the verified packaged source to Hermes `0.20.5`, official commit
  `a0ca7c19204e514f9590ce3b812e029b315ab9e9`.
- Keep the distributable source archive exact and hash-verified; “not fixed to one
  version” applies to Runtime compatibility, not to supply-chain integrity.

## Acceptance

- a real installed Hermes `0.20.5` is detected as `SUPPORTED` without ambiguity or
  timeout;
- `0.20.2` and later stable `0.20.x` patches share the same capability gate;
- prerelease and `0.21.x` remain unsupported;
- the packaging preparation and inspection catalogs name and verify the `0.20.5`
  source snapshot;
- focused Go, PowerShell, Rust and Desktop build checks pass.

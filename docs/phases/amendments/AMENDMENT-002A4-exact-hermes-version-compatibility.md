# Amendment 002A4 — Exact Hermes 0.20.2 Compatibility

> Status: IMPLEMENTED / TARGETED AUDIT PENDING  
> Owner approval: 2026-08-21  
> Affected baseline: Phase 2 Hermes discovery compatibility policy  
> Classification: post-freeze correctness correction

## Context

YORVA installs and qualifies one pinned Hermes package version, `0.20.2`, while the
historical discovery policy classified every stable `0.19.x` and `0.20.x` release as
supported. Later model, credential, lifecycle and channel contracts are qualified
against `0.20.2`, so that broad discovery result overstated actual product support.

## Decision

- The Hermes discovery support range is exactly `=0.20.2`.
- A successfully parsed version other than stable `0.20.2`, including `0.19.x`,
  `0.20.0`, `0.20.1`, later patches and prereleases, is `UNSUPPORTED` with the
  existing stable `RUNTIME_UNSUPPORTED` error code.
- A command timeout remains `TIMED_OUT`; version incompatibility must never be
  normalized to a timeout.
- The public discovery DTO shape and installation pin do not change.
- Historical Phase 2 specifications and audits remain immutable evidence. This
  amendment supersedes only their current compatibility-policy conclusion.

## Verification

- parser policy tests cover the exact release, older versions, later patches and
  prereleases;
- detector tests prove `0.19.0` is selected but classified `UNSUPPORTED`;
- HTTP and Desktop fixtures report `supportedRange: "=0.20.2"`;
- the affected Go and Desktop test subsets must pass before re-freeze.

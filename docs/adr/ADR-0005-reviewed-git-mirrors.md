# ADR-0005: Reviewed Git remotes for Hermes repository fetch

- Status: Rejected — superseded by Amendment 003A1
- Date: 2026-08-17
- Superseded: 2026-08-17 by `docs/phases/amendments/AMENDMENT-003A1-embedded-hermes-source.md`

## Context

Phase 3 drives the official Windows installer `repository` stage, which clones or fetches `NousResearch/hermes-agent` from GitHub and checks out a pinned commit. On some networks GitHub is unreachable, so the stage fails even though a later integrity check could still prove the same commit.

User-chosen mirrors, unofficial zip sites, and `irm | iex` remain forbidden.

## Decision

Rejected. Phase 3 Amendment 003A1 replaces third-party Git remotes with a verified official GitHub commit archive packaged in the Demo MSI. `gitclone.com` and `kkgithub.com` are not an accepted source path.

The adapter does not accept a user, Desktop, or HTTP-supplied URL. Remotes are Hermes-owned. Changing the list requires a Spec amendment.

## Alternatives

- Ask the user for a VPN only: remains the preferred first step, but blocks install on networks where GitHub stays unreachable.
- Inject a mirror URL into the official script: the reviewed `install.ps1` has no such parameter.
- Use an unsigned zip from a third-party host: rejected; no object-id check equivalent to a git commit.

## Consequences

Install can complete without GitHub if a reviewed remote serves `df4b65147d7ddd74dd449f9067aabbca5aef0ec7`. A malicious mirror cannot change the installed tree without changing that SHA. A mirror that lacks the commit fails closed.

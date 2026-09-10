# Phase 9 accepted branch baseline

Date: 2026-09-10. Status: **FROZEN**. Gate: **PASS**.

| Identity | Accepted value |
| --- | --- |
| Branch | `codex/phase9-openclaw` |
| Frozen P8 | `phase-008-local-product-hardening-baseline` → `2a7b842e668011a97804eb40642e3ff9dcab2f04` |
| P9 working start | `e95ed31d298c2548556e1295aacf7f84002b74ee`, P8 documentation successor |
| Final tested product | `b958801a91234b5a602d652035e5c4f3e3dc8242` |
| Freeze identity | Annotated tag `phase-009-openclaw-baseline` on the final documentation/evidence commit containing this record |
| Qualified Runtimes | OpenClaw 2026.9.3 / Node 24.16.0; coexistence with Hermes 0.20.5 / Python 3.11.15 |
| Qualified platform | Native Windows 11 x64, disposable medium-integrity user |
| G1 | [Final R2](PHASE-009-WINDOWS-G1-R2.json), completed `2026-09-10T12:24:48.8913044Z` |
| G2 | [CI #105](https://github.com/YoLin02/yorva/actions/runs/34475486005) and [MSI #43](https://github.com/YoLin02/yorva/actions/runs/34475485956), exact product candidate, PASS |
| G3 | [Final re-audit](../audits/AUDIT-009-openclaw-second-runtime.md), PASS; all five findings closed |
| Package | `YORVA_0.4.0_x64_en-US.msi`; 149,987,328 bytes; SHA-256 `9901a8e0ece982413224bc489ad9c42ab0454e29465ae72ffad04021ac3d0e68` |

The product candidate includes the implementation, narrow development-dependency
security patches, Runtime-neutral test correction, bounded bootstrap diagnostics,
owned-process completion and closed-stream cancellation fixes. The
[validation](PHASE-009-VALIDATION.md) and [structured CI/MSI evidence](PHASE-009-FINAL-CI-MSI.json)
retain the precise evidence; first FAIL and failed correction attempts remain in
the audit rather than being replaced by the final PASS.

The freeze successor changes documentation/evidence only. Product source, tests,
API, generated schema, workflows, scripts and dependencies are identical to the
tested candidate. The phase tag resolves the final record without a self-referential
commit hash or a claim that a later documentation commit was the tested binary.
No historical tag is moved; this is a branch baseline and does not imply a main merge.

The delivered capability is discovery, profile Instance create/list/delete,
authenticated Start/Stop/Restart, same-name identity isolation, Runtime-scoped
recovery and Desktop management. Optional OpenClaw models/channels/Skill/MCP,
backup mutation, managed installation and upgrade remain unavailable. Broader
version/platform support requires additional actual qualification.

The MSI uses ordinary production code with the qualification feature disabled, but
Authenticode is `NotSigned`: it is an internal test artifact, not a production public
release. P8 security/release controls remain in force. No host reboot or normal host
Hermes/OpenClaw profile mutation occurred. No correctness/security defect is deferred;
remaining INFO maintenance observations have owners/triggers in the audit.

This freezes the authorized P9 scope. It does not begin P10 or publish a release.

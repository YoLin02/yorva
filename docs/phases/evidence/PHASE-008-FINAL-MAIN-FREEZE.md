# Phase 8 — Final-main verification and internal baseline freeze

> Status: COMPLETE / FROZEN — INTERNAL BASELINE under Owner Amendment 008A1
> Date: 2026-09-07
> Accepted product/main commit: `2a7b842e668011a97804eb40642e3ff9dcab2f04`
> Annotated tag: `phase-008-local-product-hardening-baseline`
> Tag object: `365a058587b06d76ecc4e5f635bd36a49ecdaa9e`

## Acceptance chain and authority

The Owner authorized fixing the audit findings, committing and freezing P8, then removed
obtaining/generating production signing material as an internal phase prerequisite in
[Amendment 008A1](../amendments/AMENDMENT-008A1-internal-freeze-signing-boundary.md).
The original [AUDIT-008 FAIL](../audits/AUDIT-008-local-product-hardening.md) remains unchanged.
[AUDIT-008R1 PASS](../audits/AUDIT-008R1-local-product-hardening.md) closes HIGH-001 and
MEDIUM-001 at product source `43ac29152d3de1fa227937f35ccdd12908bac976`.

R1 acceptance/evidence commit `9372810e5e982bbd92f94fbde89f276d22de8685` was merged without
history rewriting at `35dafa3`; the subsequent evidence-link correction is `2a7b842`.
The complete product/API/scripts/workflow tree at `2a7b842` matches audited `43ac291`.
This exact main commit passed the checks below before the annotated tag was created and
pushed. Remote tag object and peeled commit were verified at 09:53:58 UTC.

The documentation-only closeout containing this record follows tag creation, as the Spec
requires. It records the accepted baseline rather than changing or moving that tag. It
introduces no product, dependency, schema, API, script or CI workflow change. Automatic
CI runs caused by tag/documentation pushes do not replace the identified final-main runs.

## Exact final-main CI and MSI

| Evidence | Verified result |
| --- | --- |
| [CI #98 / 34106365684](https://github.com/YoLin02/yorva/actions/runs/34106365684) | `2a7b842`, attempt 1 SUCCESS; 09:30:08–09:52:27 UTC; all three jobs pass |
| Go Node | race tests, vet, govulncheck and build pass |
| Web/API | dependency audit, OpenAPI lint/generation/no drift, typecheck/lint, 148 tests and build pass |
| Windows native | lifecycle and all MSI-inspector negatives pass; 33 native tests; format/audit/Clippy/check/no-bundle build pass |
| Desktop recovery | PASS `2026-09-07T09:52:20.0184753Z` in native job `101692203183` |
| [Windows MSI #37 / 34106365773](https://github.com/YoLin02/yorva/actions/runs/34106365773) | `2a7b842`, attempt 1 SUCCESS; 09:30:08–09:40:31 UTC |
| Remote MSI | `YORVA_0.4.0_x64_en-US.msi`; 149,921,792 bytes; `NotSigned` |
| Remote MSI SHA-256 | `7515704626620860856AD07B4A9383E1F76843E7D71C77CEBF24DAB26BE2A367` |
| [Artifact 10012878744](https://github.com/YoLin02/yorva/actions/runs/34106365773/artifacts/10012878744) | `yorva-msi`, 149,723,499 bytes, not expired when checked; expires 2026-12-06 |
| Artifact ZIP digest | `sha256:1ac693eb4e5b778069ef68b3717d8c08ac78da853627627c4e1ccb9a725c96de` |

Remote evidence comes from the workflow API, job logs and artifact metadata; the remote
MSI was not downloaded for the local guest test. The MSI and artifact ZIP are different
objects. The Windows CI runner is Server 2025, not evidence of a Windows 10 client run.
The existing 17 allowed cargo dependency warnings remain disclosed in the B6/R1 records.

## Exact main package and Windows update

The local ordinary package was built from clean `2a7b842` with no qualification feature:

- 150,286,336 bytes; `NotSigned`; built `2026-09-07T09:32:04.6848633Z`;
- SHA-256 `4EF4E3B9455D0F4927775B1565223F607820C5DABF7DABC5261FC0A5B97C888C`;
- strict MSI/resource/identity inspection passed;
- the existing test Ed25519 key signed qualification metadata for the fixed test HTTPS source;
- the 0.3.2 test baseline and renewed test TLS certificate are the exact R1 fixtures recorded
  in [PHASE-008R1-REMEDIATION](PHASE-008R1-REMEDIATION.md).

[Sanitized final-main Windows evidence](PHASE-008-FINAL-MAIN-WINDOWS.json) records source
log digests and the following results. The raw overlays/logs remain in ignored qualification
storage; each new guest uses an independent overlay and the host is never rebooted.

| Scenario | Result |
| --- | --- |
| First main metadata request | FAILED at 09:35:06.9739324 UTC; diagnostic readback established UPDATE_DOWNLOAD_FAILED before candidate metadata or installation |
| Fresh observed main Happy run | PASS 09:41:51.3662591 UTC: medium user, 0.3.2 → exact ordinary 0.4.0, installed version and authoritative recovery |
| Retry from the original failed-state snapshot | PASS 09:49:13.6664219 UTC: medium user, original 0.3.2 retained, re-check/download/install/recovery to 0.4.0, YORVA and Hermes data sentinels unchanged |

The first metadata transport failure is retained and its low-level cause is not asserted.
The observed test server subsequently recorded verified TLS metadata/package requests;
its certificate chain/fingerprint remained valid and unchanged. The same original failed
state successfully retried through the native updater. No product code, deadline, TLS
verification policy or candidate MSI was changed to obtain these successful results.
This establishes the required successful update and recoverable failure paths without
claiming every network request succeeds.

A supplementary retry harness initially used a WindowsIdentity group collection that
could not establish the mandatory integrity label and stopped before product work. It
was corrected to the already reviewed smoke's `whoami /groups` medium-SID check plus
explicit rejection of the high-integrity SID. The succeeding child verifies both. This
harness correction does not remove the ordinary non-elevated qualification requirement.

The package wrapper's post-build clean-tree check initially reported Tauri's Cargo.toml
line-ending rewrite. Its canonical Git blob matched HEAD; the original checkout bytes
were restored and the tree was confirmed clean. Packaging/inspection had already passed,
and neither product semantics nor the generated MSI were modified by that restoration.

## Complete phase gate matrix

| Gate | Evidence |
| --- | --- |
| B0 support, identity and retention contract | Bilingual PRODUCT_SUPPORT; accepted 008A1 internal/public boundary |
| B1 migration protection | Schema 017, empty/016/current and failure/protection recovery tests; original audit data dimension PASS and final Go suite |
| B2 crash/reboot/recovery | Retained real B2 guest reboot/login, R1 live readback negative/repair, final-main native Desktop recovery |
| B3 installer lifecycle | Retained ordinary-user Fresh/Upgrade/Repair/Uninstall/Reinstall evidence, exact-main MSI inspection and actual upgrade |
| B4 verified YORVA update | Retained tamper/network interruption/installer-failure cases, R1 Happy/ReconcileFailure/DownloadCrash and exact-main update/retry |
| B5 sanitized diagnostics | Reviewed bounded sanitized projection, canary/cleanup/UI tests, final Go/Web/native suites |
| B6 stability | Retained 28,802.669-second, three-Profile pressure/continuity evidence rechecked by AUDIT-008; no new soak claim |
| Independent review | Fresh solo AUDIT-008R1 PASS; original findings closed, no unresolved blocking finding |
| Final-main verification | Exact `2a7b842` CI #98, MSI #37 and actual update/retry all PASS |
| Freeze | Annotated tag pushed and remote peeled commit verified; bilingual Spec/ROADMAP/DEVELOPMENT marked COMPLETE / FROZEN in this closeout |

## Frozen boundary

P8 is frozen as an internal candidate. Production signing material was not generated and
is not a P8 freeze prerequisite. Existing source, metadata, integrity and signature-policy
checks remain enabled. Public signing/provenance and the public Windows qualification
matrix remain prerequisites to any separately authorized public release.

No Release was published, no P9 work started, no shared host reboot performed and no
normal Hermes Profile modified. The original audit, Phase 7 tag and the new Phase 8 tag
remain immutable. Any later product correction must use the governed amendment process.

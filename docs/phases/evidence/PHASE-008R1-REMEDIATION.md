# Phase 8 R1 — Recovery and interrupted-download remediation evidence

> Date: 2026-09-07
> Product candidate: `43ac29152d3de1fa227937f35ccdd12908bac976`
> Classification: internal, unsigned Windows candidate under Owner Amendment 008A1
> Verification status: exact-candidate CI/MSI and all three Windows update scenarios PASS; fresh AUDIT-008R1 returns PASS for the internal candidate.

## Reviewed problem and changes

The immutable `AUDIT-008-local-product-hardening.md` found a false update-success
postcondition (HIGH-001) and a download left permanently in `DOWNLOADING` after its
process exits (MEDIUM-001). Commit `ef7eaba811e0139ea6f03649bbc71f8dd27ac589` adds an
authenticated live recovery check and orphan-download reconciliation. Commit `43ac291`
keeps status reads and cancellation responsive while the live download owns its mutex.

The daemon remains available when Runtime inventory is UNKNOWN. Its new recovery
endpoint separately requires authoritative readback, and the native updater requires
READY, no error, and the matching daemon/Desktop version before persisting success.
The build stamps the Go sidecar with the same checked product version as Desktop;
the previous development-only version constant could not establish that postcondition.

On restart, an abandoned download becomes `FAILED / UPDATE_DOWNLOAD_FAILED`; the
candidate is retained and only fixed regular staging files are removed. A concurrent
status read cannot recover an active worker's download. The UI exposes cancellation
during its pending request and refreshes persisted state after failure/cancellation.
No Runtime adapter, schema, dependency or signature-policy relaxation is included.

## Local checks

| Check | Result |
| --- | --- |
| Go application and transport recovery regressions | PASS: live READY → failure → READY, authenticated access, no caching, deadlines and sanitized errors |
| Full Go tests and vet | PASS on the remediation Go tree, unchanged between `ef7eaba` and `43ac291` |
| Native tests | PASS, 33 tests; includes matching-version/UNKNOWN rejection, response bounds, credential non-forwarding, orphan cleanup and concurrent status |
| Native format, Clippy `-D warnings`, check | PASS locally for the affected changes and in the exact-candidate native job |
| Desktop tests | PASS, 24 files / 148 tests, including startup retry and cancellation followed by retry |
| TypeScript / ESLint / Vite | PASS; Vite also ran in the exact-source MSI builds |
| OpenAPI lint, generation and generated-client drift | PASS locally for the added contract and in the exact-candidate Web/API job |
| MSI inspector and metadata negatives | PASS; public unsigned-package rejection and positive plus 17 MSI negative cases retained |
| Whitespace / patch checks | PASS before both implementation commits |

One initial full Go run failed the existing 100 ms
`TestDiscoveryCommandRunnerDoesNotEarlyAcceptUntrustedPathCandidate` while several
verification builds were active. The test was not changed. Ten focused repetitions and
a subsequent full test/vet run passed. Concurrent load is context, not an established
root cause; the initial failure is not omitted. Exact-candidate remote Go race/vet and
vulnerability checks also pass.

## Exact-candidate daemon probe

[`PHASE-008R1-READBACK-PROBE.json`](PHASE-008R1-READBACK-PROBE.json) records a new
disposable environment with no normal Hermes Profile or Desktop Known Folder access.
It uses private bootstrap input, no proxy or redirects, owned-process cleanup and a
credential-leak check on stderr.

| Scenario | Management inventory | Recovery | Daemon version |
| --- | --- | --- | --- |
| Normal fixture | FRESH / AVAILABLE | READY | `0.4.0` |
| Invalid Profile output | UNKNOWN / INSTANCE_OUTPUT_UNRECOGNIZED | RECOVERY_REQUIRED | `0.4.0` |
| Fault removed | FRESH / AVAILABLE | READY | `0.4.0` |

Daemon SHA-256: `1B4FEA45431246841C3E74AD02D0079B232EA67F4F2842546E53738F7059FBB3`.
Hermes test fixture SHA-256: `E39170C4CBB92AF67B3EE1DE93C79465A3B10DE874EFC216EB762809E715537E`.

## Remote CI and MSI

| Evidence | Result |
| --- | --- |
| [CI #96, run 34103268935](https://github.com/YoLin02/yorva/actions/runs/34103268935) | Exact `43ac291`, attempt 1, SUCCESS; 08:56:27–09:18:34 UTC; all three jobs pass |
| [Windows MSI #35, run 34103268868](https://github.com/YoLin02/yorva/actions/runs/34103268868) | Exact `43ac291`, attempt 1, SUCCESS; 08:56:27–09:04:30 UTC |
| MSI from package log | `YORVA_0.4.0_x64_en-US.msi`, 149,921,792 bytes, `NotSigned` |
| MSI SHA-256 | `F34B35EB4D97FC15AF2DB45D279D5F8F2466950E1C4A88D01F36769E3512C90E` |
| [Uploaded artifact 10011595627](https://github.com/YoLin02/yorva/actions/runs/34103268868/artifacts/10011595627) | `yorva-msi`, 149,723,508 bytes; expires 2026-12-06 |
| Artifact ZIP digest | `sha256:ba91a1e17df8514d4a364e894b097bc6dfeed6342b71fc03419ab3797cd233c9` |

The MSI file digest and enclosing artifact ZIP digest are different objects. Remote
verification uses job logs and artifact metadata. Windows CI uses Server 2025, which
does not establish execution on a Windows 10 client.

The native job records 33 passing tests, lifecycle PASS, all MSI inspector negative
tests passing, and Desktop recovery PASS at `09:18:27.6929383Z`. Dependency review
retains the existing 17 explicitly allowed cargo warnings; a successful command is
not a claim that the dependency graph contains no advisories.

## Disposable Windows update qualification

The ordinary local `0.4.0` candidate was built from clean `43ac291` without the
`update-qualification` feature. It is distinct from the remote build above:

| Field | Local candidate |
| --- | --- |
| MSI bytes | 150,286,336 |
| MSI SHA-256 | `050DEF649F8E2AB10DEEB60A5C7E9A07392F4A8B1E8E7BC286D53852A41DA711` |
| Build time | 2026-09-07T09:00:06.2041064Z |
| Source / dirty / qualification feature | `43ac291` / false / false |
| Signature | `NotSigned`, internal |

The qualification-only baseline uses the same source with an explicit temporary
`0.3.2` version overlay, the fixed test source and a test CA. All version files are
restored byte-for-byte after building; its manifest truthfully records a dirty version
overlay and the qualification feature. It is not a historical public `0.3.2` release.
The update metadata uses the existing test Ed25519 key and fixed
`https://10.0.2.2:18443` source. No production signing key is generated.

The renewed-TLS baseline has SHA-256
`B0311884F856C56B14C339394AC112460561F7CF64F7310361D45B4231E908C0`,
150,351,872 bytes, and build time `2026-09-07T09:11:30.3839877Z`. The candidate's
signed test metadata file has SHA-256
`50C6BC7096131B9637029341073B52675AA0BBD6B3E754FB4927576AB20E2740`.

Each scenario uses a fresh Windows 11 Enterprise Evaluation 25H2 x64 QCOW2 overlay
on the retained immutable base. The worker runs at medium integrity, checks exact
installed process paths and refuses existing product/Runtime data. The host and WSL
are not restarted. The guest uses its native NVMe controller under QEMU/KVM.

| Scenario | Result |
| --- | --- |
| `Happy`: verified `0.3.2` → ordinary `0.4.0` install/relaunch/readback | PASS, `2026-09-07T09:15:22.1675444Z` |
| `ReconcileFailure`: matching installed version with failed Profile readback | PASS, `2026-09-07T09:19:29.1400628Z` |
| `DownloadCrash`: kill during partial download, restart, old version usable, retry | PASS, `2026-09-07T09:24:17.1129966Z` |

### Qualification-environment corrections

Two launch attempts stopped before guest testing: the non-login Linux PATH omitted
Node, then DrvFS rejected a UNIX monitor socket. The runner now uses an explicit Node
path and a socket on the Linux temporary filesystem.

The first running guest returned `UPDATE_DOWNLOAD_FAILED` before acquiring metadata.
Diagnostic readback and a separate network probe showed a working NIC and TCP access.
OpenSSL established that the retained test CA and HTTPS leaf expired on 2026-09-06.
This was a correct TLS rejection, not an accepted Happy result. The old certificate
and failed guest are retained as evidence. The test certificates were renewed using
the existing test keys; full TLS verification remains enabled. The renewed HTTPS leaf
is valid 2026-09-07 09:09:23–2026-09-14 09:09:23 UTC and has SHA-256 fingerprint
`5C8756A8F7FA6E393A9D730726E43B71CD0C705D59D90FF8161E3178D8C3036F`.

The first guest's default virtual RTC was eight hours behind host UTC. Subsequent
qualification sets the Windows local RTC explicitly. Early guest timestamps are not
represented as accurate host UTC or used to establish current-candidate success.

## Retained evidence and acceptance boundary

B1 migration/data protection, B2 real guest reboot/login, B3 installer lifecycle and
B5 sanitized diagnostics retain the scoped evidence in their batch records. The
eight-hour B6 pressure/continuity evidence was rechecked by the original audit; this
remediation does not rerun or expand that workload claim. Its management Operations,
diagnostic workload and Runtime fixture paths are unchanged by the updater fixes.

The original FAIL audit remains immutable. A fresh review must evaluate this actual
candidate and completed results before closing the findings. Owner Amendment 008A1
permits internal phase freeze without production signing material; production signing
and the public Windows qualification matrix still precede public-release readiness.
No Release publication, Phase 9, host reboot, normal-profile mutation or subagents are
authorized or performed by this record.

# Phase 6 Implementation and Audit Handoff

- Prepared: 2026-08-24 (Asia/Shanghai)
- Required baseline: `phase-005a1-post-freeze-corrections-baseline` ->
  `995777528557fa564a4c42e14f8431b8ddbd20e8`
- Exact product implementation candidate:
  `276991bb64c11c43b8fa354d8a47f2b51436731f`
- Candidate branch for closeout: `codex/phase6-closeout`
- Gate state at handoff: **AUDIT; NOT ACCEPTED OR FROZEN**

The exact product candidate is the committed `main` tree at `276991b`. Closeout
evidence and audit-report commits may follow it without changing packaged product
bytes. Every CI or package claim below names the exact commit it tested.

## Historical integration and governance deviation

All Phase 6 implementation commits were already reachable from `main` before an
independent Phase 6 Gate. That is repository history, not phase acceptance. This
closeout does not rewrite history, manufacture a merge, or treat the early integration
as a pass. Phase 6 remains blocked from Phase 7 until an independent audit, exact-
candidate CI, mandatory Windows evidence, final-main verification and an annotated
baseline tag are complete.

## Commit and batch manifest

| Batch | Commit | Delivered behavior | Owning modules |
| --- | --- | --- | --- |
| 1 | `8418feaa9cc2a942540abf581b545eeb85b35dc6` | Pinned Hermes lifecycle/channel qualification, normalized contract updates and ADR-0008 credential authority. | `docs/`, `docs/adr/` |
| 2-3 | `2e03a7801455e4d29a75506bf18cd01a123cf9d6` | Runtime-neutral lifecycle contract, durable lifecycle Operations, conflict/recovery rules and Profile-exact Hermes lifecycle adapter. | `services/node/internal/app`, `domain`, `runtime`, `runtime/hermes`, `persistence`, `transport/httpapi` |
| 4 | `acf413932406ea0ff5c2db51e292875667104a9d` | Typed lifecycle client and localized Instance Start/Stop/Restart controls. | `apps/desktop/src` |
| 5-7 | `28b6f0fdfab7c05d08df2bc331980f214b986616` | Channel contract/application/API, migration 010, ephemeral QR broker, Weixin connect and typed WeCom verification-before-commit. | `api/`, `services/node/internal/app`, `domain`, `runtime`, `runtime/hermes`, `persistence`, `transport/httpapi` |
| 8 | `b415f79bbbd02c36fc2c16f51c1021b18cd80dc9` | Localized Channel cards, Weixin QR modal, WeCom manual form, cancellation/retry and targeted disconnect UX. | `apps/desktop/src` |
| 8 evidence | `31df7ed36e146fc2c7a856fc8daffc12d7c16408` | Historical Batches 2-8 implementation record. | `docs/phases/` |
| 8 correction | `dbd5d3201c7519a7b18e5d4e541534eb715b7ade` | QR rendering and stable failure presentation corrections. | `apps/desktop/src` |
| 8A | `e4b1473a75ef4ca811ecbd5aac192dadb0d690b3` | Weixin-only pending pairing count and write-only eight-character approval flow, including Desktop UX and redaction/error coverage. | `api/`, `services/node/internal/app`, `runtime`, `runtime/hermes`, `transport/httpapi`, `apps/desktop/src` |
| 8B | `276991bb64c11c43b8fa354d8a47f2b51436731f` | User-session tray, close-to-tray, packaged release login start, development-build exclusion, single-instance restore and explicit Quit shutdown; also final Channel UI refinements. | `apps/desktop/src-tauri`, `apps/desktop/src` |

## Adjacent work not attributed to Phase 6

The required Phase 6 commits are in a linear `main` history that also contains these
adjacent commits. They remain in the candidate's ancestry but are not claimed as Phase
6 delivery:

- `ce1a32831b8d6354a32c270c8064395eb8f9afd9` — Phase 3/Phase 2 final-path
  generation correctness and governance changes;
- `349403385ba3fd39671f9c9dcda61200d9826689` — general Desktop visual,
  branding and Runtime-status refinement.

The separate active development branch `codex/configurable-download-sources` and its
commits `b9cca7c`/`c5f0b8d`, together with all of that worktree's tracked and untracked
installation-source, embedded-Python, packaging, document and UI work, are outside this
candidate. They were not stashed, reset, moved, staged, deleted or copied into the
closeout worktree.

## Delivered contract surface

### Public local API

- lifecycle status plus Start/Stop/Restart under
  `/api/v1/instances/{instanceId}`;
- Channel list/connect/disconnect under the exact Instance;
- initiating-session-only `GET /api/v1/operations/{operationId}/channel-qr`;
- Weixin-only pairing status and approval routes;
- closed request bodies, stable error codes, bearer/origin checks, typed DTOs and
  generated Desktop schemas.

No public DTO exposes a PID, service/task name, executable/Profile path, raw Hermes
output, QR payload, pairing code, Bot Secret or credential plaintext.

### Persistence and operations

- `009_instance_lifecycle_operations.sql` adds the active per-Instance lifecycle
  Operation conflict constraint;
- `010_channel_bindings.sql` adds one safe projection per Instance/Channel with a
  foreign key, cascade delete, closed Channel/state checks and `metadata_json = '{}'`;
- lifecycle and Channel mutations use durable Operations without treating SQLite as
  Runtime, process or credential authority;
- no migration adds a channel secret, QR, pairing-code or PID column.

### Runtime and Hermes adapters

- the compile-time Runtime bundle owns lifecycle and Channel capability selection;
- Core/application contracts contain normalized YORVA intent and do not import Hermes;
- Hermes `0.20.2` owns Profile targeting, fixed lifecycle argv, bounded output and
  postcondition queries;
- Weixin uses fixed-host bounded iLink polling and short-lived in-memory QR delivery;
- WeCom validates the typed Bot ID/Secret against the fixed official WSS host before
  atomically committing the exact Profile fields;
- sender pairing is Weixin-only, Profile-exact, write-only and non-persistent;
- the active sealed Runtime installation is not modified by Phase 6.

### Desktop and Tauri

- TanStack Query/generated-client resources drive lifecycle and Channel presentation;
- QR, WeCom secret and pairing-code values remain modal/form-local and are cleared;
- Tauri owns only user-session tray, close/restore, login-start registration,
  single-instance restoration and bounded daemon shutdown behavior;
- Desktop login start does not enable Hermes `ON_LOGIN`, start an Instance, request
  elevation or register a machine service.

## Verification already recorded

### Historical implementation verification

The preserved implementation record reports these checks on 2026-08-20 after
`b415f79`: OpenAPI lint/generation, Desktop typecheck/lint/tests/build, `go test ./...`,
`go vet ./...`, `go build ./cmd/yorvad` and Tauri release `--no-bundle` passed. It also
records migration, Profile-isolation, QR-session, WeCom rollback and conflict regression
coverage. Command transcripts were not committed, so this remains historical handoff
evidence and is not substituted for the closeout Gate.

### Exact product-candidate GitHub runs

For `276991bb64c11c43b8fa354d8a47f2b51436731f` on 2026-08-21:

- CI run `32459991187`: **FAIL** overall. `Web and API contract` passed and
  `Windows Desktop native shell` passed, including dependency install, sidecar build,
  `windows-lifecycle-smoke.ps1`, MSI inspector negative tests, Rust format/test/audit,
  clippy/check and Tauri release no-bundle. `Go Node` failed at
  `go test -race ./...`; later Go steps were skipped. This run is not a Gate pass.
- Windows MSI run `32459991229`: **PASS**, artifact `yorva-msi` / `9438772982`.
  Downloaded artifact `Yorva_0.3.2_x64_en-US.msi` is 120,135,680 bytes with SHA-256
  `5333022B304692E4C5BAAFC03FA5F666E9BB8483F240E2CDD608E9F1DE48EA28` and passed the
  candidate MSI inspector.

The MSI above is exactly mapped to `276991b`, but it has not been identified as the
build used for the Owner-authenticated real-account smoke. See
`PHASE-006-OWNER-AUTHENTICATED-SMOKE.md`.

## Batch 8A evidence state

Commit `e4b1473` includes adapter, application, HTTP/OpenAPI and Desktop coverage for:

- exact-Profile pending count with no sender/request/code disclosure;
- fixed eight-character alphabet validation;
- write-only approval with no response echo or durable Operation;
- invalid/expired and locked stable outcomes;
- no retry loop and no cross-Profile targeting;
- localized form lifetime and clearing behavior.

Exact closeout focused tests and independent audit review remain required. The Owner
attestation does not separately state that a real sender-pairing request and Desktop
approval were exercised, so this handoff does not claim that manual fact.

## Batch 8B evidence state

At `276991b`, the exact-candidate Windows CI job passed Rust library tests and the
repository lifecycle smoke, and the source contains explicit tests for close-to-tray and
Quit state. Packaged tray restore, explicit Quit, login-start and second-launch behavior
still require a sanitized exact-build Windows smoke record. Automated success is not
represented as that manual record.

## Pending closeout verification

- diagnose and resolve the exact-candidate `go test -race ./...` failure without
  weakening tests;
- focused checks for evidence/status changes and Batch 8A/8B code;
- one complete immutable-candidate Gate after any required fix;
- empty and Phase 5 migration verification plus explicit schema inspection;
- exact-build default/named Profile lifecycle and Desktop continuity smoke;
- safe post-smoke inspection and the limitations recorded in the Owner smoke evidence;
- independent audit, any bounded remediation, and a fresh R1 re-audit after a FAIL;
- exact-candidate CI, final-main CI, final evidence/status commit and annotated tag.

## Environment limitations known at handoff

- the real-account attestation is not tied to an exact commit or the CI MSI;
- the locally found `Yorva_0.3.2_x64_en-US.msi` contains unrelated embedded-Python
  packaging work and fails the `276991b` MSI inventory, so it is explicitly excluded;
- the host is Windows 11 Pro x64 build 26200 and has Hermes `0.20.2`, but the recorded
  smoke build reports `0.0.0-dev` and no commit identifier;
- no WeCom exact-value redaction search can be reconstructed after its local credential
  material is absent; no missing secret value is guessed or hashed.

## Scope confirmation

No Phase 7 Runtime upgrade/repair/uninstall, Skills, MCP, backup/restore, additional
Channel, Cloud, remote command, generic process/service/shell/file API or new Runtime
capability is included in the Phase 6 scope manifest.

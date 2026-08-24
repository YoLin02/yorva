# Phase 6.5 Implementation and Audit Handoff

## Candidate identity

- Phase 6 frozen baseline: `phase-006-runtime-lifecycle-messaging-channels-baseline`
- Baseline commit: `7ca9103e7af210296a5e24916df01856539b550e`
- Clean successor branch: `codex/phase6.5-freeze`
- Implementation candidate: `559245cf42c6ea07dbe4706676334a717b2173fd`
- Proposed annotated tag: `phase-0065-developer-led-demo-baseline`
- Date: 2026-08-24 (Asia/Shanghai)

The candidate was assembled in the independent worktree
`D:\workcode\myproject-phase65-freeze`. The original developer worktree was not reset,
stashed, deleted, overwritten, or used as the freeze checkout. Temporary UI references,
personal build artifacts, generated installers, deliverables, and the P7 implementation
plan were excluded.

## Accepted scope

The Owner's P6.5 inclusion and freeze instruction covers:

- authenticated configurable Hermes artifact/dependency source settings;
- packaged and exact-hash-verified CPython prerequisite support;
- bundled-first and online-first artifact selection with fail-closed integrity behavior;
- Provider model-catalog retrieval with a transient write-only credential;
- multiple selected model IDs with one authoritative Hermes default model;
- reliable reopening of the saved/default model configuration;
- stable Hermes development compatibility `>=0.20.2 <0.21.0`;
- the official packaged Hermes `0.20.5` source at commit
  `a0ca7c19204e514f9590ce3b812e029b315ab9e9`;
- directly related Desktop, OpenAPI, packaging, documentation, and regression tests.

No P7 Skills, MCP, backup/restore, upgrade, remote management, RBAC, Cloud, generic shell,
generic file-system, generic process, or generic service capability is included.

## Candidate history

The clean successor contains three P6.5 commits after the immutable Phase 6 tag:

- `b191b6b` — configurable download-source implementation;
- `6f725d9` — CI logging and portable Hermes fixture correction;
- `559245c` — accepted P6.5 model, embedded-Python, Hermes `0.20.5`, packaging,
  documentation, regression, and pre-audit hardening changes.

The final implementation commit also rejects Provider catalog redirects before a
credential-bearing request can leave the fixed qualified endpoint. This was found and
fixed during the pre-audit candidate review, before the formal audit report.

## Local verification

The following checks ran against the clean candidate source:

- OpenAPI lint — PASS;
- OpenAPI generation and generated-client drift check — PASS;
- Desktop typecheck — PASS;
- Desktop lint — PASS;
- Desktop tests — PASS, 22 files / 105 tests;
- Desktop production Web build — PASS;
- pnpm dependency audit at `low` threshold — PASS, no known vulnerability;
- Go focused application/Hermes/HTTP tests — PASS;
- Go complete `go test ./...` — PASS;
- Go vet — PASS;
- `govulncheck ./...` — PASS, no vulnerability found;
- Go daemon build — PASS;
- Windows lifecycle smoke — PASS;
- MSI inspector negative suite — PASS, including missing, duplicate, substituted,
  wrong-name, wrong-size, wrong-hash, license, extra-executable, and extraction cases;
- Rust format — PASS;
- Rust library tests — PASS, 13 tests;
- Rust audit — PASS under the repository policy (17 pre-existing allowed warnings,
  no newly introduced dependency or lockfile change);
- Rust clippy with warnings denied — PASS;
- Rust check — PASS;
- Tauri no-bundle release build — PASS;
- real installed Hermes discovery smoke — PASS:
  `SUPPORTED`, version `0.20.5`, official `bin\hermes.exe`, two candidates;
- real bundled CPython mirror smoke — PASS: uv installed the packaged archive locally
  and executed `Python 3.11.15`;
- `git diff --check` — PASS.

The local Windows host cannot execute Go `-race` because its Go toolchain has CGo
disabled. The exact-candidate Linux CI owns that unchanged mandatory check; ordinary
complete Go tests were not substituted as a claim of race coverage.

## Exact packaged inputs

The exact candidate package preparation and MSI inspection verified:

| Input | Size | SHA-256 |
| --- | ---: | --- |
| `hermes-agent-a0ca7c19204e514f9590ce3b812e029b315ab9e9.zip` | 73,798,347 | `4E63F39BD992B79BF364A4E22204A22B8E520E2365B53C909F6F15D42C917C54` |
| `node-v22.23.1-win-x64.zip` | 35,682,836 | `7DF0BC9375723F4A86B3AA1B7CC73342423D9677A8DF4538ACA31A049E309C29` |
| `npm-12.0.2.tgz` | 3,045,132 | `5DBB86C71D07A1957F2E90734092DD6A58BDCD9EBC2D8D41CA1C6E6A21D364E1` |
| `cpython-3.11.15+20260728-x86_64-pc-windows-msvc-install_only_stripped.tar.gz` | 25,676,832 | `64A804111830C5329BFC5A4D95D6CBCBB377CAA2C02195101EDF85D15FC53099` |
| Hermes `LICENSE` | 1,070 | `821556E6336796450AB852D375117B48A4887E71D255794FD6318D99982A5AB6` |
| `NODE-LICENSE` | 148,217 | `8CC9BB466B19FC7E7CC99D03E9DF1132021FDA8B01EEA2624C58BB372DBEF576` |
| `NPM-LICENSE` | 9,742 | `7610D223851F421D315DF5E77974F1C68A04B97E02060E5BBBCF13D95E3CA257` |

Exact-candidate local outputs after the final code change:

- Desktop executable SHA-256:
  `E05D6F00B6EFBE99FBC2E9A2E699FD140056B22449A903967A91A5B59C2E1A7F`;
- inspected MSI: `Yorva_0.3.2_x64_en-US.msi`;
- MSI size: `147,566,592` bytes;
- MSI SHA-256:
  `455B6424E2F1322113B191E0DCDE7997CA2514254E7D9C38C34C4EA540774570`.

The MSI is mapped to commit `559245c`; no source, executable, resource, version, or
packaging-input change occurred between that commit and the recorded build.

## Remote exact-candidate checks

- CI run [`32715890955`](https://github.com/YoLin02/yorva/actions/runs/32715890955),
  exact head `559245c`: **PASS**. Web/API job `97396905674`, Go job
  `97396906002`, and Windows native job `97396905812` all succeeded;
- Windows MSI run [`32715958209`](https://github.com/YoLin02/yorva/actions/runs/32715958209),
  exact head `559245c`: **PASS**. Job `97397119720` prepared, built, inspected, and
  uploaded artifact `9516145610`.

The downloaded remote artifact contained exactly one MSI:

- file: `Yorva_0.3.2_x64_en-US.msi`;
- size: `147,566,592` bytes;
- SHA-256: `4FA1D0958F33AEBEC7B0C373BA798116E8213476455855FC7E83543216A4F5E2`;
- artifact archive digest:
  `sha256:b5bfdca752490616d3e4561bd1a73ce9425bcf20f1ef1ffe5bf12b173c6adc99`.

The local and GitHub MSI hashes differ because WiX output is not byte-reproducible, but
both were built from exact commit `559245c`, have the same inspected inventory and
size, and independently passed the fail-closed package inspector.

## Manual evidence boundary

The Owner previously verified the P6 lifecycle/channel/tray/hidden-start behavior and
then explicitly selected the accepted P6.5 modifications for inclusion and freeze.
No QR image, Secret, pairing code, account identifier, API key, model credential, or
Provider response payload is copied into this handoff. No unreported P6.5 target-machine
fact or roadshow date is invented.

## Audit handoff state

Implementation is frozen at `559245c`; both exact-candidate remote runs and the formal
audit passed. The audited successor fast-forwarded `main`, and the no-content integration
trigger `b9af6a3fd057b90ef636ff3b581cc9680774dac8` passed final-main CI run
[`32717542173`](https://github.com/YoLin02/yorva/actions/runs/32717542173), including
Web/API, Go race/vet/vulnerability/build, Windows lifecycle, Rust and Tauri release jobs.

The automatically triggered final-main MSI run
[`32717542138`](https://github.com/YoLin02/yorva/actions/runs/32717542138) failed twice
before compilation because GitHub `codeload` returned HTTP 429 for the exact Hermes
archive. The failure is retained as infrastructure evidence and is not presented as a
green package run. It does not replace or invalidate exact-candidate MSI run
`32715958209`, because the only commits after `559245c` were evidence/audit documents
and a no-content CI trigger; no product or packaging input changed.

With final-main CI green, P6.5 is eligible for the closeout documentation commit and
annotated tag `phase-0065-developer-led-demo-baseline`. P7 may begin only from that tag.

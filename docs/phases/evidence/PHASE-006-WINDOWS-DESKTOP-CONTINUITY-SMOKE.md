# Phase 6 Windows Lifecycle and Desktop Continuity Smoke

- Test date: 2026-08-24 (Asia/Shanghai)
- Host: Windows 11 Pro x64, version/build `10.0.26200` / `26200`
- Candidate commit: `790076e8802b665d92866bf114bda775caf9e75b`
- Candidate branch/worktree: `codex/phase6-closeout` in the isolated closeout worktree
- Result: **PASS for Closeout Batch C2**

This record contains no screenshot, Profile name, account identifier, QR payload,
pairing code, Bot Secret, PID value or credential material. The two non-default
Profiles used below are identified only as named Profile B and an already-running
unrelated Profile.

## Candidate build identity

The release candidate was produced from a clean worktree at the commit above with:

```text
pnpm --filter @yorva/desktop tauri build --no-bundle
```

| Artifact | Bytes | SHA-256 |
| --- | ---: | --- |
| `yorva-desktop.exe` | 4,727,296 | `8500DB57CF77971F0B199FBB31214E11911BBB4B77D18E0BF02E49FB04408BCE` |
| packaged `yorvad` sidecar | 18,755,072 | `52282C2AEE4406FC03B1F68DE038C7DC95F1ABDE939DEFBA61592971A45CF9F0` |

The sidecar does not embed a Git VCS revision. Its mapping is therefore the clean
worktree command, exact commit and recorded digest above, not an inferred embedded
version. The candidate includes the narrow login-item path refresh correction in
`790076e`; no unrelated active-worktree files were copied into this build.

## Lifecycle results

The Desktop reported authoritative state after each action. Read-only inspection of
the durable Operation rows produced this sanitized record:

| Target | Action | Operation result | Started UTC | Completed UTC | Observable postcondition |
| --- | --- | --- | --- | --- | --- |
| Default Profile | Stop | `SUCCEEDED` | `2026-08-24T01:43:42.3199943Z` | `2026-08-24T01:43:47.8287902Z` | Desktop reported `STOPPED`. |
| Default Profile | Start | `SUCCEEDED` | `2026-08-24T01:43:59.8255202Z` | `2026-08-24T01:44:04.4307004Z` | Desktop reported `RUNNING`. |
| Default Profile | Restart | `SUCCEEDED` | `2026-08-24T01:45:10.8534791Z` | `2026-08-24T01:45:19.2098013Z` | Gateway identity changed; old process was gone, exactly one new gateway was alive, and Desktop remained `RUNNING`. |
| Named Profile B | Start | `SUCCEEDED` | `2026-08-24T01:46:16.5117287Z` | `2026-08-24T01:46:19.7576259Z` | Desktop reported `RUNNING`. |
| Named Profile B | Restart | `SUCCEEDED` | `2026-08-24T01:47:00.6237075Z` | `2026-08-24T01:47:06.8738991Z` | Gateway identity changed; old process was gone, exactly one new gateway was alive, and Desktop remained `RUNNING`. |
| Named Profile B | Stop | `SUCCEEDED` | `2026-08-24T01:47:29.3488891Z` | `2026-08-24T01:47:32.9035956Z` | Desktop reported `STOPPED`; this was the final named-Profile state. |

No raw PID is retained. The restart watchers retained only the boolean postconditions
`identity changed`, `old process not alive`, and `new process alive`.

Named Profile B had no qualified Hermes login task before Start and still had no such
task after Start/Restart/Stop. No Hermes entry appeared under the per-user Run key.
This exercises the qualified non-persistent path when the Profile login item is absent.
No UAC prompt or elevation request appeared.

## Desktop continuity results

| Flow | Result | Evidence |
| --- | --- | --- |
| Close main window to tray | PASS | The candidate main window became non-targetable while one candidate Desktop process and one Desktop-owned daemon remained. A correctly parsed default gateway record still identified a live gateway process. |
| Restore from tray | PASS — Owner-observed | In the shared candidate session the Owner manually exercised the hidden tray and confirmed the C2 tray behavior was normal. This closes the notification-area action that the window-only automation API could not click directly. |
| Explicit tray Quit | PASS — Owner-observed | The Owner confirmed the remaining tray C2 flow in the same session. Immediately after that confirmation, read-only process inspection found zero candidate Desktop processes and zero candidate Desktop-owned daemons; the running Hermes gateway was not treated as Desktop-owned. |
| Packaged login start | PASS | The per-user login item resolved to the exact candidate executable plus only `--hidden`. Starting that command produced one Desktop and one daemon, no targetable main window, and did not increase the existing gateway count (`2` before and `2` after). |
| Missing login item | PASS | After removing the single excluded old-worktree login value, the exact candidate recreated the value with its own path and `--hidden`. No service, machine startup entry or Hermes `ON_LOGIN` policy was introduced. |
| Second launch | PASS | A second exact-candidate launch restored the existing main window while process inspection remained at one Desktop and one daemon. |
| Bounded Desktop-owned daemon exit | PASS | After the Owner's explicit tray exit, both exact-candidate Desktop and daemon counts were zero. No duplicate candidate daemon survived. |

The Owner confirmation was received while the exact executable above was the sole
candidate Desktop in the shared Windows session. It is therefore tied to this build
identity; it is not attributed to the unrelated MSI in the active development
worktree.

## Security and scope observations

- login start used only the current-user registration and `--hidden`;
- no Hermes Instance was started by the Desktop login launch;
- no additional gateway or daemon survived where prohibited;
- no prompt requested elevation, though the host session itself was already an
  administrative development session, so standard-user token behavior remains a
  static/CI review item rather than an invented manual claim;
- lifecycle `RUNNING` and Channel `CONNECTED` remained separate UI/resources;
- no Phase 7 feature or Runtime mutation was exercised;
- no secret-bearing screenshot or raw runtime file is retained as evidence.

The real Weixin/WeCom result and its exact-build limitation remain in
`PHASE-006-OWNER-AUTHENTICATED-SMOKE.md`; this Desktop record does not enlarge those
account-test claims.

# Amendment 008A1 — Internal Phase 8 freeze without production signing material

> Status: ACCEPTED by Repository Owner
> Date: 2026-09-07
> Applies to: Phase 8 Spec, B6 phase-exit Gate and product support classification

## Owner decision

The Owner instructed: “修复所出现的问题，之后完成提交后进行冻结p8”, then confirmed that
production signing material is not prepared and instructed: “p8中不需要进行生成签名材料，
把这个前置条件去除掉”. This amendment records that explicit scope/exit decision.

## Change and reason

Obtaining or generating production Windows code-signing material is no longer a Phase 8
completion/freeze prerequisite. Phase 8 may be baselined and frozen as an **internal
candidate** once its product correctness, recovery, migration, integrity, diagnostic,
Windows qualification, stability, exact-commit CI, audit and final-main Gates pass.
The Owner authorizes the necessary P8 commit/push, main integration and annotated baseline
tag/freeze after those Gates pass. No Release publication or Phase 9 implementation is
authorized by this decision.

This resolves an unavailable external release input. It does not remove either finding
in AUDIT-008 or accept any correctness/security defect as debt. HIGH-001 and MEDIUM-001
must be fixed and re-audited before freezing. The original audit remains unchanged.

## Security and architecture impact

There is no architectural change. Existing fixed-origin, signed metadata, package size,
SHA-256, MSI identity/version, declared Authenticode policy, post-install verification and
tamper rejection remain mandatory. The native verification path must not be bypassed.
Qualification keys/CA remain test-only, and an unsigned package must remain labeled
`NotSigned` / internal. Self-signing is not production provenance.

Production signing, signer/provenance evidence and public-candidate installation/update
qualification remain requirements **before public release**, recorded in PRODUCT_SUPPORT.
They no longer block the internal Phase 8 baseline. A phase tag is not a product Release.
No shared-host reboot, normal Hermes Profile mutation or multi-agent work is introduced.

## Revised verification and exit matrix

| Item | Phase 8 internal freeze | Later public release |
| --- | --- | --- |
| Correctness fixes and regression tests | Required; no blocking HIGH/CRITICAL | Required |
| Metadata/signature-policy/hash/tamper verification | Required, including test-source qualification | Required with production source and signer |
| Exact-candidate CI, Windows MSI/lifecycle/update/recovery and diagnostics evidence | Required with artifact signature state recorded truthfully | Required for the public candidate |
| Existing three-Instance four-hour minimum/eight-hour target soak | Required; retained evidence may be reused where fixes do not affect its workload | Reassess for the public candidate |
| Production Windows signing material generation/availability | Not required by P8 | Required before public readiness |
| Fresh review pass, Owner-authorized main integration, final-main checks and annotated phase tag | Required | Does not itself publish a Release |

## 中文执行说明

P8 移除“准备或生成生产 Windows 签名材料”这一冻结前置条件，按内部候选完成阶段基线。
两项审计问题必须修复；功能、数据保护、来源/元数据签名/完整性检查、Windows 验证、
稳定性、精确提交 CI、复审及 final-main 门禁继续执行。Owner 已授权通过门禁后的提交、
推送、合并与 annotated Phase 8 tag/freeze。

公开发布仍须取得生产签名及相应候选证据。未签名内部包不得标记为公开发布就绪，现有
校验逻辑不得关闭。本修订不授权发布 Release、开始 P9、重启宿主机、修改日常 Hermes
Profile 或启用多智能体。

# YORVA Phase 6 — Runtime 生命周期与消息通道

> 状态：**READY — IMPLEMENTATION AUTHORIZED**
> 语言：中文 Owner 审阅源
> Owner：Repository owner
> 必需基线：`phase-005a1-post-freeze-corrections-baseline` → `9957775`
> 英文执行镜像：`PHASE-006-runtime-lifecycle-messaging-channels.md`
> 路线图条目：Phase 6，按 Owner 于 2026-08-20 的指示扩展
> 实现分支：当前 Owner 授权分支
> 执行授权：**Owner 于 2026-08-20 授权执行 Phase 6**

## 1. 目标

Phase 6 使已经配置好的 YORVA Instance 真正可以运行，并且无需打开终端即可通过受支持的消息通道访问。

本阶段按顺序交付两个产品结果：

1. 先提供标准化的 Instance 生命周期管理，并实现 Hermes 的启动、停止、重启、状态和启动/服务策略；
2. 再在生命周期基础上提供微信和企业微信连接流程。

本阶段不得把 YORVA Core 变成 Hermes 专用进程管理器。Core 负责标准化生命周期意图、Operation、并发和恢复；Hermes adapter 负责 Hermes Profile/gateway 命令、Windows 服务行为和兼容细节。

## 2. 基线与当前事实

Phase 5 已在 `phase-005-models-credentials-baseline` 完成并冻结。

该基线当前具有以下事实：

- 已认证的启动、停止和重启路由只是有意保留的 `CAPABILITY_NOT_SUPPORTED` 桩；
- Instance capability 始终报告 `lifecycle: false`；
- Runtime registry 尚未在 bundle 中接入生命周期 feature contract；
- Desktop 明确提示当前不提供 Start、Stop 和 Restart；
- YORVA 不安装、不持有、不启动、不停止，也不 reconcile Hermes Profile gateway；
- Phase 5 模型验证不会启动 gateway，也不会遗留 Runtime 进程；
- 尚未实现 `channel_bindings` 表、Channel application use case、通道凭据权威或 Channel Desktop 流程；
- ADR-0007 只授权 Hermes-native **模型**凭据，并明确不授权通道凭据。

固定的 Hermes 源码通过官方 CLI 命令提供 Profile 级 gateway 生命周期。在 Windows 上，Hermes 可能使用每 Profile Scheduled Task、Startup-folder fallback 或 detached process。这些机制属于 Hermes，不得泄漏到通用 API 或 domain。

## 3. 已批准的范围方向与 Owner 决策

Owner 已于 2026-08-20 批准以下方向：

- [x] **D1 — Phase 6 顺序与范围。** Instance/Runtime 生命周期从 Phase 7 前移到 Phase 6，并在消息通道之前实现。
- [x] **D2 — 统一生命周期所有权。** 生命周期意图、标准化状态、Operation、冲突控制和恢复属于 Runtime-neutral Core/application 代码；Hermes-specific 执行保留在 Hermes adapter 内。

资格证据记录于 `evidence/PHASE-006-BATCH-1-QUALIFICATION.md`。

- [x] **D3 — Hermes 生命周期目标。** Start/Stop/Restart 管理所选 Hermes Profile messaging gateway，绝不管理不可变 installation tree。
- [x] **D4 — 后台服务策略。** Phase 6 拒绝 `ON_LOGIN`；已资格确认的 Windows 路径可能触发提权或隐式持久化 fallback。本阶段仅提供手动生命周期。
- [x] **D5 — 通道凭据权威。** ADR-0008 已接受；精确 Profile 范围的 Hermes-native 存储是唯一权威，YORVA 只存安全投影。
- [x] **D6 — 企业微信 QR 兼容。** 拒绝非公开 QR 兼容路径；typed 手动 Bot ID/Secret 加认证验证满足退出标准。
- [x] **D7 — 短时 QR 投递。** 批准仅限发起 session 的投递；共享 SSE 只含 readiness 元数据。
- [x] **D8 — 消息依赖物化。** 可使用已资格确认的安装字节；若缺少依赖，必须创建新的 ADR-0006 sealed generation，禁止原地修复。
- [x] **D9 — `CONNECTED` 的含义。** Channel `CONNECTED` 表示 binding 已通过认证验证；lifecycle `RUNNING` 独立表示 gateway 在线。

任何被拒绝的决策都必须在授权实现前同步修改 in-scope、测试矩阵和退出标准。

## 4. 进入条件

只有满足以下全部条件，才可以开始实现：

- Phase 5 继续冻结在所要求的 tag 和 commit；
- Phase 6 中英文 Spec 已同步；
- D3-D9 已获得明确 Owner 决策；
- 必需 ADR 已接受；
- 固定 Hermes 生命周期、微信和企业微信接口已有资格证据；
- Spec 状态改为 `READY`；
- Owner 明确授权实现 batches。

创建本 draft 不授权分支、代码、migration、依赖或 Runtime mutation。

## 5. 用户可见成功流程

### 5.1 生命周期流程

```text
已配置的 Instance
  -> 用户选择 Start
  -> YORVA 创建 Instance lifecycle Operation
  -> Hermes adapter 启动准确的 Profile gateway
  -> 权威状态变为 RUNNING
  -> Desktop 显示 Running 以及可用的 Stop/Restart 操作
  -> 关闭 Desktop 不会停止 gateway
```

```text
运行中的 Instance
  -> 用户选择 Stop 或 Restart
  -> YORVA 对该 Instance 的 mutation 进行串行化
  -> Hermes adapter 执行有界、优先 graceful 的生命周期工作
  -> 权威状态变为 STOPPED 或 RUNNING
  -> Desktop 显示最终状态
```

### 5.2 消息通道流程

```text
Instance
  -> 用户选择 Connect Weixin 或 Connect WeCom
  -> YORVA 创建 Channel authentication Operation
  -> 只向发起 session 显示有过期时间的 QR/auth 流程
  -> 认证得到确认
  -> 安全 Channel metadata 变为 CONNECTED
  -> 生命周期状态独立显示 gateway 是否 RUNNING
```

## 6. 范围内

### 6.1 Runtime-neutral 生命周期基础

- 小型 `LifecycleManager` Runtime feature contract；
- 通过 registry 解析 lifecycle capability；
- 生命周期状态、Start、Stop、Restart 和启动策略 application use case；
- 标准化生命周期状态与启动策略；
- 持久化、以 Instance 为 target 的 lifecycle Operations；
- 每 Instance mutation coordination；
- idempotency、权威 postcondition 检查和 daemon restart recovery；
- 标准化 lifecycle event 和安全 diagnostics；
- 只由 capability/state DTO 驱动的共享 Desktop lifecycle presentation。

### 6.2 Hermes 生命周期实现

- 固定 Hermes `0.20.2` Profile gateway status 资格确认；
- 默认 Profile 和命名 Profile 的 Profile-scoped Start、Stop、Restart；
- Windows 普通用户生命周期管理；
- 仅手动启动；`ON_LOGIN` 策略管理延后；
- 仅通过已资格确认的官方接口执行有界 graceful stop 和必要的有界升级终止；
- 外部 Hermes CLI/Studio 修改或进程崩溃后的状态 reconciliation；
- 不依赖 Hermes internal Python import 或数据库 schema。

### 6.3 消息通道

- 仅 `weixin` 与 `wecom` 的通道 capability list；
- list/get Channel binding 状态；
- connect、disconnect、retry 和 cancel 行为；
- QR/authentication Operation 状态；
- 短时 QR readiness notification 和仅发起 session retrieval；
- success、failure、cancellation 和 timeout 状态；
- 安全 Channel metadata 持久化；
- 获批 credential authority 与 redaction；
- 本地化 Desktop UX。

### 6.4 从 Phase 7 前移的生命周期 crash/recovery UX

- orphaned lifecycle Operation 的终结或权威 reconciliation；
- daemon 或 Hermes failure 后 stale/unknown status presentation；
- 显式 retry 操作；
- 除非获批启动策略要求，否则不执行隐式 restart；
- 能区分配置、服务和 Runtime failure 的最小安全诊断上下文。

## 7. 非目标

Phase 6 不实现：

- 通用 OS process、PID、Scheduled Task、service 或 shell management API；
- Runtime install repair、uninstall 或 upgrade；
- Hermes code/generation upgrade 或对 sealed generation 的原地 mutation；
- 假想第二 Runtime 的生命周期控制；
- multiplexed Hermes gateway 配置或跨 Profile routing；
- Skills、MCP、backup/restore、sessions、memory 或 chat UI；
- 微信和企业微信之外的其他消息通道；
- 在 YORVA 中浏览、发送或管理对话；
- Cloud、远程命令投递、账号、组织或 RBAC；
- 任意 log tail 或完整 observability infrastructure；
- 自动提权或隐藏 UAC prompt；
- enterprise service manager 或 dynamic Runtime plugin system；
- 声称 Channel 认证状态本身表示 gateway 在线；
- 声称 gateway 正在运行本身表示已配置 Channel credential。

## 8. 架构边界与统一生命周期所有权

要求的方向：

```text
React Desktop
  -> authenticated typed Node API
  -> Runtime-neutral lifecycle/channel application use cases
  -> Runtime registry feature contract
  -> Hermes lifecycle/channel adapter
  -> 固定官方 Hermes 接口或另行批准的窄 fallback
```

统一生命周期部分只负责稳定 YORVA 概念：

- Instance identity；
- desired action：Start、Stop、Restart 或 startup policy change；
- normalized lifecycle state；
- Operation 创建和状态迁移；
- idempotency 和 conflict policy；
- timeout/cancellation policy；
- 根据 adapter 权威状态作出的 recovery decision；
- capability 和 event projection。

Hermes adapter 独占以下职责：

- 默认 Profile 与命名 Profile argv 差异；
- Hermes gateway command selection；
- Windows Scheduled Task/Startup/detached-process 语义；
- 获批官方接口需要时的 Hermes PID/lock/status 文件；
- human-output compatibility parser；
- Hermes-specific service installation 与 graceful-drain 行为；
- 将 raw failure 映射为稳定 YORVA error。

禁止：

```text
React -> Hermes CLI
Tauri -> 普通 Hermes lifecycle business logic
application/domain -> services/node/internal/runtime/hermes
generic lifecycle contract -> PID、task name、Profile path 或 Hermes service type
Hermes adapter -> Desktop DTO/component
```

本阶段应向 compile-time Runtime bundle 添加一个真实 feature boundary，不得创建宽泛 manager/service/provider framework。新的 lifecycle application 代码必须通过 registry 解析 feature，不得引入 `app.HermesLifecycleSource` bridge。

## 9. 标准化生命周期合同

概念 Runtime contract：

```go
type LifecycleManager interface {
    Status(ctx context.Context, installation Installation, nativeID string) (LifecycleStatus, error)
    Start(ctx context.Context, installation Installation, nativeID string) error
    Stop(ctx context.Context, installation Installation, nativeID string) error
    Restart(ctx context.Context, installation Installation, nativeID string) error
}
```

稳定的 observed lifecycle state：

```text
RUNNING
STOPPED
UNKNOWN
```

用户可见的 transient state 来自 active Operation，而不是第二套 mutable resource-state machine：

```text
STARTING
STOPPING
RESTARTING
CONFIGURING_STARTUP
```

规则：

- `RUNNING` 必须具有来自 adapter 的权威 live evidence；
- `STOPPED` 必须具有权威 absence/not-running evidence；
- query failure、malformed output、ambiguous Profile targeting 或 stale evidence 返回 `UNKNOWN` 及稳定 error code；
- SQLite last-known state 永远不能授权 mutation 或 success result；
- public lifecycle view 不暴露 PID、executable path、Profile home、Scheduled Task name 或 raw output；
- capability 只在 Runtime/version 已资格确认、受支持且 executable 安全解析时为 true。

## 10. 生命周期 Operation 语义

Lifecycle mutation 返回 `202 Accepted` 和 Operation：

```text
instance.start
instance.stop
instance.restart
instance.lifecycle.configure
```

每个 lifecycle Operation 的 target 为：

```text
target.type = instance
target.id   = <stable YORVA instanceId>
```

规则：

- 除非 startup policy 需要 typed body，mutation request 使用仓库已有的有界 closed `{}` body；
- 每个 mutation 都要求有效 `Idempotency-Key`；
- 对权威 `RUNNING` Instance 执行 Start 时幂等成功且不启动 duplicate；
- 对权威 `STOPPED` Instance 执行 Stop 时幂等成功；
- Restart 要求 Instance 权威状态为 `RUNNING`；`STOPPED` 返回 `INSTANCE_NOT_RUNNING`，不得静默把 Restart 变成 Start；
- 只有 postcondition query 观察到请求的最终状态才可成功；
- timeout 或 unknown postcondition 绝不能报告 success；
- external command exit code 本身不是权威成功；
- worker 认领 external mutation 后，只有已资格确认的接口能够安全停止时才允许 cancellation，否则返回 `OPERATION_NOT_CANCELLABLE`；
- HTTP request 不为外部生命周期流程保持长连接；
- 等待 Hermes 或 Windows service control 时不持有数据库 transaction。

Timeout、output limit 和 graceful-drain budget 在 Batch 1 资格确认时锁定，并在 Batch 2 开始前写回本 Spec。

## 11. 生命周期并发与 Mutation 策略

使用最窄的 Instance scope。

同一个 Instance 上以下操作冲突：

- Start、Stop、Restart 与 startup-policy mutation 之间；
- lifecycle mutation 与 Instance delete；
- 当操作会改变 credential 或 gateway activation 时，lifecycle mutation 与 Channel connect/disconnect；
- lifecycle mutation 与 daemon restart 后恢复的另一 lifecycle Operation。

附加规则：

- 删除 `RUNNING` 或 `UNKNOWN` Instance 时 fail closed；用户必须先获得 `STOPPED`；
- lifecycle transition 不得为了 identity resolve/validation 之外的工作长期持有 installation-wide create/delete lock；
- Hermes 证明 Profile-scoped service model 相互独立后，不同 Instance 的 operation 可以并发；
- model/config/credential mutation 不得与 lifecycle transition 竞态；
- 除非明确写入获批 Spec，Phase 6 不在 model 或 Channel mutation 后自动 restart；
- lock 位于 application coordination，不在 Desktop state；
- 每个 goroutine 与 external process wait 都有明确 owner 和 cancellation path。

## 12. 固定 Hermes 生命周期资格确认

Batch 1 必须对准确 packaged Hermes source commit 和已安装 `0.20.2` 行为进行资格确认。

候选官方接口：

```text
默认 Profile：hermes gateway <status|start|stop|restart>
命名 Profile：hermes -p <nativeID> gateway <status|start|stop|restart>
启动策略：    hermes [-p <nativeID>] gateway install <closed flags>
```

资格确认必须证明：

- 准确默认/命名 Profile targeting；
- background worker path 不出现 interactive prompt；
- argv 或 environment 不包含 secret；
- Windows 普通用户行为；
- Start 前能否区分 service absence；
- official CLI 是否可在没有隐藏提权的情况下配置 `MANUAL` 和 `ON_LOGIN`；
- 权威 structured 或 fixture-bounded status evidence；
- already-running/already-stopped 行为；
- graceful stop、升级终止、timeout 和 descendant cleanup；
- duplicate-start prevention；
- Desktop 与 `yorvad` 退出后的行为；
- gateway crash 和 Windows login 后的行为；
- 固定 stdout/stderr bound 与安全 redaction；
- unknown version/output/service state 时 fail closed。

Adapter 只调用 trusted absolute active-generation launcher，使用固定 command verb 和已验证 native Profile ID。不得使用 `cmd.exe`、PowerShell command string、PATH 选择的 Hermes、`shell=true` 或 imported Hermes Python internals。

若 official output 只有 human-readable 形式，可以批准 version-pinned、fixture-tested parser。未知、未经资格确认的本地化、截断或 oversized output 均 fail closed。

## 13. 启动/服务管理

生命周期管理从 Phase 7 前移至 Phase 6。资格确认后，持久化 login-start 策略仍延后。

Phase 6 已资格确认的边界为：

- Start 是显式手动操作，绝不启用 login persistence；
- Stop 与 Restart 只影响所选 Profile gateway；
- 可为诊断观察既有 Hermes login item，但 YORVA 不创建、修改或删除它；
- login item 缺失时使用固定官方 `gateway install --no-start-on-login --start-now` 调用；其已资格确认的 Windows 实现不会创建 persistence；
- 任何意外 prompt、提权请求或 persistence 变化均使 Operation 失败。

任何可能需要提权的 service 安装/移除必须：

- 作为独立、明确的用户操作；
- 在 OS prompt 前描述操作；
- 只使用已资格确认的 Hermes-owned service surface；
- 不静默 fallback 到较弱 persistence mechanism；
- 用户拒绝批准时返回稳定状态；
- 若触发 automatic-elevation security review trigger，必须有 security/architecture ADR。

仅在获批 ADR 要求时，Tauri 才可以介入窄范围 native approval handoff。Tauri 不得构造 Hermes command 或决定 lifecycle policy。

## 14. 生命周期 Recovery 与 Reconciliation

Daemon startup 和 explicit refresh 时：

1. 将 stable Instance 解析为其权威 native Profile；
2. 解析受支持的 active Hermes installation；
3. 查询 Hermes lifecycle adapter；
4. projection 为 `RUNNING`、`STOPPED` 或 `UNKNOWN`；
5. 不盲目重复 mutation，而是 reconcile orphaned lifecycle Operation。

Recovery 规则：

- orphaned Start 只有在权威状态为 `RUNNING` 时才可成功；
- orphaned Stop 只有在权威状态为 `STOPPED` 时才可成功；
- orphaned Restart 无法仅根据最终 `RUNNING` 证明发生过一次新的 restart，因此除非已资格确认的接口具有可持久证明该 transition 的 evidence，否则 terminal 为 `FAILED`/`LIFECYCLE_RESULT_UNKNOWN`；
- `UNKNOWN` query 永远不能变成 success；
- recovery 不自动发出 Start、Stop 或 Restart；
- Desktop close 不得停止已经成功管理的 Hermes gateway；
- Hermes 意外退出成为 observed `STOPPED`/`UNKNOWN` 加安全 diagnostics；只有获批 Hermes startup policy 才能自动恢复；
- SSE 只负责 notification；reconnect 后 GET 仍为 source of truth。

## 15. Lifecycle API 与 OpenAPI

要求的 local API：

```text
GET  /api/v1/instances/{instanceId}/lifecycle
POST /api/v1/instances/{instanceId}/start
POST /api/v1/instances/{instanceId}/stop
POST /api/v1/instances/{instanceId}/restart
```

OpenAPI 必须把 Phase 4 的 `lifecycle: false` literal 和 unsupported-only response 替换为 typed capability、lifecycle view 与 `202 Operation` response。CORS、authentication、error envelope 和 request-size bound 保持不变。

Lifecycle SSE notification 只包含安全 identifier 与 normalized state/error field，绝不包含 raw process output、PID、path、task name、command、environment 或 secret。

## 16. Channel 状态与合同

本阶段支持的 channel identifier 是 closed list：

```text
weixin
wecom
```

标准化 Channel binding state：

```text
NOT_CONFIGURED
CONNECTING
CONNECTED
DISCONNECTED
FAILED
UNKNOWN
```

以 D9 最终决策为准：

- `CONNECTED` 表示 credential/binding 已认证并验证；
- 不表示 Hermes gateway process 当前为 `RUNNING`；
- `RUNNING` 不表示任一 Channel 已配置；
- 无法安全查询 Runtime-native truth 时使用 `UNKNOWN`；
- 普通 GET response 只暴露安全 configured/status metadata。

概念 Runtime contract 保持小型：

```go
type ChannelManager interface {
    ListChannels(ctx context.Context, installation Installation, nativeID string) ([]ChannelState, error)
    BeginConnect(ctx context.Context, installation Installation, nativeID string, req ChannelConnectRequest, events ChannelEventSink) error
    Disconnect(ctx context.Context, installation Installation, nativeID string, channel string) error
}
```

Hermes-specific QR polling、account file、environment name、remote endpoint detail 和 gateway activation behavior 不进入 Core contract。

## 17. Channel Operation、QR 投递与 Disconnect

Channel mutation 使用持久化 Operation：

```text
channel.connect
channel.disconnect
```

Connect stage 可以包含：

```text
preparing
qr_ready
waiting_for_scan
waiting_for_confirmation
verifying
committing
```

QR 规则：

- QR payload 和具有凭据等价性的 URL 只存在于有界内存；
- shared SSE stream 只发送带 Operation ID 和 expiry metadata 的 `channel.qr.ready`；
- QR payload 通过 D7 批准的 authenticated、Operation-scoped、initiating-session-only mechanism 获取；
- 不把 QR payload 写入 URL query parameter、Operation row、`operation_events`、log、diagnostics、audit row、SQLite、browser storage 或 backup；
- QR data 有明确 expiry，并在 success、failure、cancellation、timeout 或 daemon shutdown 后清理；
- SSE reconnect 不重放 QR payload；
- 丢失或过期 QR 需要新的有界 authentication attempt。

Disconnect 规则：

- 在获批 authority 允许的范围内移除已授权 local credential/binding material；
- 仅在权威确认后更新安全 Channel metadata；
- 除非 official platform surface 确实完成，否则不得声称 remote account/bot 已被 revoke；
- remote revoke 不可用时，Desktop 必须说明本地 disconnect 不删除远端微信/企业微信 bot identity；
- disconnect 不得删除无关 Profile config 或另一 Channel material。

## 18. Channel 凭据权威与数据

Phase 6 实现前需要一份新的 credential-authority ADR。

ADR 必须为每个 Channel 选择一个且仅一个 authority，并定义：

- credential 是 Hermes-native 还是 YORVA-owned；
- Profile/Instance isolation；
- Windows at-rest behavior；
- Desktop/daemon 退出后 background gateway 如何读取 credential；
- set、replace、status 和 delete 语义；
- Weixin account/token 与 WeCom Bot ID/Secret 处理；
- 是否授权 version-pinned compatibility writer；
- 禁止 duplicate authority 和 global environment mutation。

无论选择哪一种方案，以下 invariant 均为强制：

- SQLite、API read、Operation、event、log、diagnostics、audit metadata、Desktop storage、argv 或 URL 中无 channel credential plaintext；
- 无 silent plaintext fallback；
- caller 不提供 path 或 environment key；
- 精确 Instance/Profile 与 Channel allowlist；
- secret-bearing buffer 尽量短期存在并在可行时清理；
- `channel_bindings` 只保存安全 metadata；
- 只有 YORVA 持有唯一 OS-secure secret authority 时才使用 `secret_refs`；
- 不得隐式扩展 ADR-0007。

## 19. Channel 持久化

Phase 6 增加 `DATA_MODEL.md` 所描述的 `channel_bindings` migration。

要求的约束：

- 指向 `instances(id)` 的 foreign key，并具有明确 delete behavior；
- 满足本阶段每 Instance/Channel 单 binding 模型的 uniqueness；
- application/domain code 中 closed normalized Channel type/state validation；
- `metadata_json` 不包含 QR、token、secret、raw response 或敏感 URL；
- Runtime 对 Runtime-native state 保持权威；
- stale metadata 明确只是 projection，绝不能授权 secret/lifecycle mutation。

Lifecycle status 应保持为 live adapter query，只允许可选安全 last-known projection。Phase 6 不持久化 PID，也不让 SQLite 成为 process authority。

Start、Stop、Restart、startup-policy change、Connect 和 Disconnect 都写入不含秘密的 local audit metadata。

## 20. 消息依赖与 Sealed Generation

当前安装的 Hermes `0.20.2` 环境已经只读资格确认，包含所需的 `aiohttp`、`cryptography`、`httpx` 与 `qrcode` module。

Phase 6 不得原地修改 active sealed generation，也不得允许 Hermes lazy installation 这样做。

获批路径为：

- lifecycle 与 channel authentication 使用已经资格确认的字节；
- 若缺少任何必需字节，停止执行，并按 ADR-0006 构建/激活新的 exact-lock generation 后再继续。

任何 generation-building path 都必须保持 ADR-0006：

- 新 Install Transaction 和 generation ID；
- immutable reviewed lock 与 source pin；
- 无 user-selected package index；
- 完整 manifest/seal verification；
- compare-and-swap activation；
- 从 filesystem transaction truth 执行 rollback/recovery；
- 不使用 SQLite Operation 作为 activation authority。

若这需要 upgrade/repair primitive，必须作为窄 Phase 6 prerequisite 明确授权，不得静默实现一般 Phase 7/Runtime upgrade roadmap。

## 21. 微信与企业微信资格确认

### 21.1 微信

资格确认必须验证：

- official/documented iLink QR endpoint 与 payload bound；
- QR refresh、scan、confirmation 和 timeout state；
- redirect-host validation；
- credential/account persistence authority；
- disconnect/revocation behavior；
- dependency availability；
- secret redaction 和 Profile isolation；
- 不 import Hermes internals 或 parse terminal QR art 的可测试 non-interactive adapter path。

### 21.2 企业微信

固定 Hermes 明确说明其 QR 创建 endpoint 不属于 public WeCom developer API。Phase 6 不得静默把它们当作稳定接口。

若 D6 批准 fallback，则必须：

- version-pinned 且仅限 WeCom；
- 使用固定 HTTPS endpoint/host，不接受 caller-provided URL；
- response size、polling interval 和 total duration 有界；
- 严格处理 response schema 与 redirect；
- unknown output/status 时 fail closed；
- 由 fixture 和 manual Windows smoke 覆盖；
- 文档明确其为 compatibility behavior，而非有保证的 public API。

若 D6 拒绝 fallback，Spec 必须明确授权安全 typed manual Bot ID/Secret flow，或从 mandatory Phase 6 exit criteria 移除 WeCom。

## 22. Desktop UX 与 i18n

Lifecycle UX 在实现顺序上先于 Channel UX。

要求的 lifecycle presentation：

- Running、Stopped 和 Unknown status；
- active Starting/Stopping/Restarting Operation state；
- 根据 capability 和权威 state 启用 Start/Stop/Restart；
- D4 批准时提供显式 startup policy control；
- active work 可能中断时对 Stop/Restart 进行确认；
- daemon/Runtime crash 后的 recovery messaging；
- 不显示 PID、service name、path 或 raw Hermes output；
- 英文和简体中文 strings。

要求的 Channel presentation：

- 位于准确 Instance 上的 Weixin 和 WeCom capability card；
- Connect、Disconnect、Retry 和 Cancel；
- 带 expiry countdown 和 scan/confirmation state 的 QR modal；
- 只显示安全 account label/external ID；
- 明确区分 Channel Connected 与 Gateway Running；
- 清晰说明 remote-revocation limitation；
- unsupported Runtime/version 与 missing dependency 提示；
- keyboard-accessible modal 与 status announcement；
- 不在 localStorage、sessionStorage 或 Zustand 持久化 QR/token。

TanStack Query 持有 daemon resource。React local state 只可在 modal lifetime 内保存当前显示的短时 QR。

## 23. 稳定错误码

最终列表在 qualification 时锁定。预期 normalized error 包括：

```text
CAPABILITY_NOT_SUPPORTED
RUNTIME_NOT_INSTALLED
RUNTIME_UNSUPPORTED
INSTANCE_NOT_FOUND
INSTANCE_NOT_RUNNING
INSTANCE_LIFECYCLE_CONFLICT
LIFECYCLE_STATUS_UNKNOWN
LIFECYCLE_START_FAILED
LIFECYCLE_STOP_FAILED
LIFECYCLE_RESTART_FAILED
LIFECYCLE_TIMED_OUT
LIFECYCLE_RESULT_UNKNOWN
LIFECYCLE_SETUP_REQUIRED
LIFECYCLE_APPROVAL_DECLINED
CHANNEL_NOT_SUPPORTED
CHANNEL_CONFLICT
CHANNEL_AUTH_FAILED
CHANNEL_AUTH_TIMEOUT
CHANNEL_AUTH_CANCELLED
CHANNEL_STATE_UNKNOWN
CHANNEL_DISCONNECT_FAILED
CHANNEL_DEPENDENCY_MISSING
```

Public error 只包含 user-safe text。Desktop behavior 依赖 error code 和 typed state，绝不匹配 message。

## 24. 实现 Batches — 仅在授权后

### Batch 1 — 资格确认与治理锁定

- 资格确认固定 Hermes 生命周期和 Windows service 行为；
- 资格确认微信/企业微信 authentication surface；
- 决定 D3-D9；
- 接受必需 ADR；
- 锁定 timeout、output bound、state、error 和 dependency path；
- 在 code-bearing batch 前更新本 Spec。

### Batch 2 — Runtime-neutral 生命周期基础

- 添加小型 Runtime lifecycle contract 和 registry capability；
- 实现通用 application status/Operation/concurrency/recovery logic；
- 更新 lifecycle OpenAPI 和 generated Desktop types；
- 去除 lifecycle=false hard-coding，且不在 Core/UI 加入 Hermes branch。

### Batch 3 — Hermes 生命周期 Adapter

- 实现准确 Profile-scoped status/Start/Stop/Restart；
- 强制仅手动启动，不允许隐藏 persistence 或提权；
- 添加 bounded command/process behavior 和 postcondition check；
- 添加 adapter fixture、cancellation/timeout 和 restart recovery test。

### Batch 4 — Lifecycle Desktop 与 Resilience

- 暴露 lifecycle control 与 Operation；
- 实现 crash/recovery 和 external-state reconciliation UX；
- 验证 Desktop close 不停止 Hermes；
- Channel 实现前完成 lifecycle manual Windows smoke。

### Batch 5 — Channel 合同、数据与 Secret Authority

- 实现 ChannelManager registry boundary 与 application use case；
- 添加 `channel_bindings` migration；
- 实现获批 credential authority；
- 添加 typed API、Operation stage 和 ephemeral QR broker。

### Batch 6 — 微信

- 实现微信 connect/status/disconnect；
- 实现 QR expiry、cancellation 和 redaction；
- 验证准确 Instance/Profile isolation。

### Batch 7 — 企业微信

- 实现已批准的安全手动 Bot ID/Secret flow；
- 实现 status/disconnect 和 revocation disclosure；
- 验证准确 Instance/Profile isolation。

### Batch 8 — Channel Desktop、完整验证与审计交接

- 实现本地化 Channel UX；
- 运行完整检查和 Windows lifecycle/channel smoke；
- 收集 immutable candidate evidence；
- 进入 Phase 6 audit 并停止 feature work。

任何 batch 都不会自动授权下一批。出现阻塞 qualification、security 或 architecture finding 时停止执行。

## 25. 测试矩阵

| 场景 | 预期结果 | 层级 |
|---|---|---|
| Unsupported Runtime/version lifecycle query | capability false 或稳定 unsupported result；无 process | application/adapter/API |
| Start stopped Instance | 只有获得 `RUNNING` truth 后 Operation 才 `SUCCEEDED` | application/adapter/integration |
| Start running Instance | 幂等 success；无 duplicate process | application/adapter |
| Stop running Instance | graceful bounded stop 和权威 `STOPPED` | adapter/integration |
| Stop stopped Instance | 幂等 success | application/adapter |
| Restart running Instance | old process 退出；new running truth；无 duplicate | adapter/Windows smoke |
| Restart stopped Instance | `INSTANCE_NOT_RUNNING`；无 process | application/API |
| Unknown lifecycle output | `UNKNOWN`；不推断 success | adapter fixtures |
| Lifecycle timeout/cancel | terminal stable error；command 不遗留由其持有的 orphan child | adapter/application |
| Concurrent Start/Stop/Restart | 一个 mutation 获胜；其他 deterministic conflict | application/race |
| Delete running/unknown Instance | fail closed；Profile 保留 | application/API |
| Different Instance lifecycle mutation | 资格确认安全时可并发 | application/race |
| Daemon 在 Start 成功后退出 | gateway 继续运行 | Windows smoke |
| Daemon restart 发现 orphan Start/Stop | 由权威 state 派生 terminal result | application/integration |
| Daemon restart 发现 orphan Restart | 不根据最终 running state alone 虚假成功 | application/integration |
| login item 缺失时 Start | 固定非持久化启动路径；无 prompt 或提权 | adapter/Windows smoke |
| Channel capability list | 只显示已资格确认的 Weixin/WeCom | adapter/API/Desktop |
| Weixin QR success | 短时 QR -> confirmed -> 安全 `CONNECTED` metadata | adapter/application/Desktop/manual |
| QR expires | operation timeout；payload cleared | application/security |
| QR cancel | polling stop；payload clear；无 credential commit | application/security |
| SSE reconnect | GET 恢复 state；QR payload 不重放 | API/Desktop |
| 第二个 authenticated 非发起者 | 无法读取另一 Operation 的 QR payload | API/security |
| WeCom unknown QR schema/status | fail closed 并返回稳定 error | adapter fixtures |
| Channel credential redaction | DB/API/log/event/audit/diagnostics 无 plaintext | security/integration |
| Profile A vs Profile B | lifecycle 与 Channel action 不跨 identity/secret scope | adapter/application |
| Disconnect | 只移除目标 Channel material；remote revoke truth 准确 | adapter/application/Desktop |
| Sealed generation dependency attempt | 原地 mutation 被拒；使用获批 new-generation path | install/integrity |
| 从 Phase 5 DB migration | uniqueness/FK 正确且无 secret column | persistence |

## 26. 完整验证

审计前运行 `DEVELOPMENT.md` 中适用的 repository gate，包括：

```text
pnpm install --frozen-lockfile
pnpm audit --audit-level low
pnpm api:lint
pnpm api:generate
generated OpenAPI drift check
pnpm typecheck
pnpm lint
pnpm test
pnpm build
go test ./...
受支持 CI 上 go test -race ./...
go vet ./...
go build ./cmd/yorvad
govulncheck
cargo fmt --check
cargo test --locked
cargo clippy --locked --all-targets -- -D warnings
cargo check --locked
cargo audit
Windows sidecar/lifecycle smoke
Tauri no-bundle release build
若 Phase 6 改变 packaged Hermes dependency，则执行 MSI packaging/inspection
```

要求的 manual evidence：

- Windows 上默认与命名 Profile 的 Start/Stop/Restart；
- gateway 继续运行时关闭 Desktop；
- 真实微信 scan/confirm/connect/disconnect；
- 真实企业微信获批 auth path；
- 检查 log 和 SQLite，确认无 plaintext secret/QR；
- 准确 packaged Hermes generation dependency 与 integrity evidence。

## 27. Phase-specific 审计要求

独立审计必须应用 `AUDIT_STANDARD.md` 的每个维度，并特别验证：

- Runtime-neutral lifecycle code 无 Hermes import/branch；
- Runtime bundle 持有 feature selection 和 capability；
- 未创建 generic process/service/shell API；
- 每个 mutation 都把准确 Instance ID 解析为准确权威 Profile；
- Start/Stop/Restart postcondition 权威且 fail closed；
- concurrency 覆盖 lifecycle 与 delete/config/channel 冲突；
- Desktop close 和 daemon restart 行为符合 process ownership docs；
- Windows service/elevation 行为明确且获批；
- 无 sealed generation 原地 mutation；
- Channel credential authority 符合 accepted ADR；
- 若存在 WeCom fallback，其范围准确有界并已文档化；
- QR 短时、仅发起 session 可见，且不存在于 durable/event/log surface；
- Channel `CONNECTED` 与 lifecycle `RUNNING` 未被混淆；
- migration 可从 empty 与 Phase 5 schema 工作；
- 附有 exact-candidate CI 与 Windows manual evidence。

任何 credential/QR disclosure、arbitrary process surface、cross-Profile mutation、duplicate gateway start、虚假 lifecycle success、hidden elevation 或 sealed-generation mutation 都是阻塞项。

## 28. 退出标准

Phase 6 只有在以下条件全部满足时才可通过：

- 受支持 Hermes Instance 从资格确认后的 capability data 暴露 `lifecycle: true`；
- 默认与命名 Profile 的 Start、Stop、Restart 和 live status 无需终端即可工作；
- 仅手动 lifecycle 和 lifecycle recovery 严格按获批方案工作；
- 关闭 Desktop 不停止已管理的 Hermes gateway；
- Weixin 与 D6 获批的 WeCom path 完成 mandatory auth flow；
- Channel 与 lifecycle status 均可见，且语义相互独立；
- SQLite、log、Operation、event、diagnostics 或普通 API response 中无 QR 或 channel credential plaintext；
- 未原地改变 sealed generation；
- 所有 mandatory test 和 Windows smoke flow 通过；
- 独立审计返回 `PASS` 或 Owner 接受的 `PASS WITH CONDITIONS`；
- accepted baseline 已 merge、freeze 和 tag。

该 freeze 之前禁止 Phase 7 implementation。

## 29. 强制停止条件

出现以下任一情况时停止并返回 Owner review：

- D3-D9 仍未解决；
- official Hermes lifecycle surface 无法做到 non-interactive、Profile-exact 和 fail-closed；
- Start/Stop/Restart 需要 generic shell 或 import Hermes internals；
- Windows service management 需要 hidden 或 automatic elevation；
- 权威 postcondition 无法区分 `RUNNING`、`STOPPED` 与 `UNKNOWN`；
- Channel credential 无法获得一个获批 authority；
- WeCom QR 需要未批准 undocumented behavior；
- QR 无法限制于发起 authenticated session；
- 所需 messaging dependency 会修改 active sealed generation；
- material architecture/security conflict 缺少 accepted ADR；
- 无法产生所需 verification 或 real Windows evidence。

## 30. 完成证据

仅在实现后填写：

- implementation 和 audit-accepted commit；
- exact-commit CI run；
- Windows lifecycle 和 Channel smoke record；
- migration evidence；
- secret/QR inspection evidence；
- audit report 和 Gate Decision；
- merge commit；
- final-main CI/MSI evidence；
- annotated Phase 6 baseline tag。

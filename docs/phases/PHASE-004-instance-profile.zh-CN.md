# YORVA Phase 4 — 实例 / Profile 管理

> 状态：**READY**
> 语言：中文 Owner 审核版
> Owner：仓库所有者
> 目标基线：`phase-003-hermes-installation-baseline`
> 实现起始 commit：`d04b1fdc298f643f84d0c84a245595baae2e8994`
> 前置 Gate：`AUDIT-003R9-hermes-installation.md` — **PASS**
> 英文 Agent 执行镜像：`PHASE-004-instance-profile.md`
> 实现授权：**已授予** 2026-08-19
> Owner 决策 D1–D4：**已批准** 2026-08-19
> 功能分支：`codex/phase4-instance-profile`

本文件与英文镜像定义同一合同。Owner 审核中文版本。两版已同步并标记为 `READY`，且 Owner 已于 2026-08-19 另行明确授权。若两版存在实质差异，必须停止并请求澄清。

## 1. 目标

让已经具备一个受支持 Hermes 安装的用户，把 Hermes Profile 作为标准化的 YORVA Instance 进行管理：

```text
受支持的 Hermes 安装
→ 查询官方 Hermes Profiles
→ 标准化并对账 YORVA Instances
→ 创建最小化 Profile
→ 查看身份与可用状态
→ 经明确确认后永久删除非 default Profile
```

一个 YORVA Instance 映射一个 Hermes Profile，但不代表每个 Instance 必须对应一个操作系统进程。Hermes 是 Profile 是否存在及其数据的权威；SQLite 只保存 YORVA 清单和最近一次已知元数据。

## 2. 进入条件

- [x] Phase 3 独立审计 Gate 为 `PASS`（`AUDIT-003R9`）。
- [x] Phase 3 已在 `phase-003-hermes-installation-baseline` 达到 `COMPLETE / FROZEN`。
- [x] `control/active.json` 仍是唯一 active generation 指针。
- [x] Phase 2 discovery 会验证 active generation 选择的 executable。
- [x] Owner 批准第 3 节中的决策。
- [x] 中英文 Spec 已同步并标记为 `READY`。
- [x] Owner 另行授权 Phase 4 实现。

Owner 于 2026-08-19 批准 D1–D4、同步标记 `READY`、实现授权、起始 commit `d04b1fdc298f643f84d0c84a245595baae2e8994`（当前 `main`，含已在 `origin/main` 的 Desktop 样式提交）以及功能分支 `codex/phase4-instance-profile`。

## 3. 需要 Owner 确认的决策

推荐采用下列最小 Phase 4 合同：

- [x] **D1 — 官方接口。** 使用 active generation 内固定 Hermes `0.20.2` 的官方文档 Profile CLI。已审查的 REST API 依赖正在运行的 Hermes Web 服务，而 TUI gateway 没有 delete 方法；Phase 4 不为 Profile 管理而启动 Hermes 服务。重新确认没有合适的离线结构化输出后，`list` 输出可采用固定版本、夹具覆盖的窄兼容解析器；未知输出必须 fail closed。
- [x] **D2 — 最小化创建。** 使用官方 no-clone/no-alias/no-skills 行为创建。YORVA 不从其他 Profile 复制凭据、`.env`、`auth.json`、模型、配置、会话或 Skills。
- [x] **D3 — 破坏性删除。** Phase 4 包含永久删除指定的非 default Hermes Profile。UI 必须要求输入 Profile 名称确认，并说明 Hermes 所有的 Profile 数据会被删除。`default` 永远不可删除。删除 native 数据后，保留的 YORVA Instance 标记为 `MISSING`，稳定 `instanceId` 不被抹除。
- [x] **D4 — 不做生命周期。** Phase 4 报告 `lifecycle: false`，不提供 Start/Stop/Restart。Profile 管理能力不能证明已经具备安全的 Instance 级进程生命周期；除非 Amendment 获批，否则生命周期留到后续 Phase。

如任一决策被否决，必须先同步修改中英文两版，不能标记 `READY`。

## 4. 用户可见成功流程

1. Desktop 刷新 Hermes discovery，仅在安装状态为 `SUPPORTED` 时开放 Instance 操作。
2. Instance 页面列出内置 `default` 与命名 Profile，显示可用状态、default/受保护状态和最近成功同步时间。
3. 创建只接受 Profile 名称，不接受路径、URL、来源、命令、环境变量、克隆来源、凭据或模型参数。
4. YORVA 创建 `instance.create` Operation，命令结束后重新查询 Hermes 权威状态。
5. 删除显示破坏性警告，要求输入完全一致的标准化 Profile 名，并创建 `instance.delete` Operation。
6. 刷新可发现 YORVA 外部新增或移除的 Profile。Hermes 查询失败必须产生 `UNKNOWN`，不能误判为 `MISSING`。
7. Desktop 重启后，通过现有 Operation API 恢复任务投影，再通过新一次 reconcile 恢复 Instance 真实状态。

## 5. 范围内

- 将内置和命名 Hermes Profiles 列为 YORVA Instances；
- 通过 Hermes adapter 创建一个最小命名 Profile；
- 查看标准化身份、可用状态与保护状态；
- 经明确确认后永久删除一个非 default Profile；
- 对账在 YORVA 外部创建或移除的 Profile；
- list/create/get/delete 的认证本地 HTTP 与 OpenAPI 合同；
- 持久化 `instance.create` 与 `instance.delete` Operations；
- 作为缓存的 SQLite Instance 清单；
- 中英文 Desktop 列表、空状态、创建、进行中、删除、冲突、缺失和错误状态；
- adapter、application、persistence、protocol 与 Desktop 测试。

## 6. 范围外

硬边界：

- Hermes 安装、修复、升级或卸载；
- 修改 Phase 3 generations、Seal、`active.json`、安装事务、`PATH` 或 `HERMES_HOME`；
- Profile 重命名、克隆、导入、导出、选择/激活或 distribution 安装；
- 登录、认证、凭据、API Key、Provider 或模型配置；
- Skills、MCP、渠道、微信/企业微信、会话、Memory、备份/恢复或 Cloud；
- 启动、停止、重启或监管 Hermes 进程；
- 接管、迁移或删除遗留 `hermes-agent` 目录；
- 直接扫描 Hermes Profile 目录或导入 Hermes Python 内部模块；
- 通用动态 Runtime Plugin System；
- 任意 shell、路径、环境变量、URL 或文件 API。

## 7. 架构边界与包职责

```text
React Desktop
    ↓ generated typed client
认证本地 HTTP / OpenAPI
    ↓
Application Instance use cases + Operations
    ↓
Domain Instance model
    ↑
Hermes Profile adapter + SQLite repository
```

目标职责：

```text
services/node/internal/domain/instance       稳定的 Instance 类型与状态
services/node/internal/app                   list/create/delete/reconcile 用例
services/node/internal/runtime/hermes        官方 Profile 命令 adapter/parser
services/node/internal/persistence/sqlite    Instance 缓存与 migrations
services/node/internal/transport/httpapi     封闭 DTO 与 handlers
apps/desktop/src                             只负责展示与本地交互
```

Hermes 专属解析只能留在 Hermes adapter。复用现有 process containment、Operation、event、logging 和 SQLite 组件，不复制另一套。Phase 4 不得顺带重排无关 Phase 2/3 文件。

禁止依赖：

```text
React → Hermes CLI 或 Profile 文件
Domain → Hermes 类型
Hermes adapter → HTTP 或 UI
SQLite 缓存 → filesystem recovery 权威
Rust/Tauri → Profile ownership 或 reconcile 决策
```

## 8. 官方 Hermes 接口资格确认

仓库固定的官方来源是 Hermes `0.20.2`、commit `df4b65147d7ddd74dd449f9067aabbca5aef0ec7`。当前审查证据显示：

- 官方文档 CLI：`hermes profile list`、`create`、`show`、`delete`；
- 结构化 REST：支持 list/create/delete，但依赖正在运行的 Hermes Web backend；
- TUI gateway JSON-RPC：支持 list/create/describe，但没有 delete；
- 尚未确认 CLI Profile list/show 提供离线结构化输出。

Batch 1 必须在生产代码前对固定 archive 重新确认。Adapter 选择顺序：

1. 不扩大进程生命周期或认证范围即可使用的官方结构化接口；
2. 官方文档 CLI；
3. 针对官方 CLI 输出的窄范围、精确版本兼容解析。

若所选接口需要启动服务、解析未公开文件、导入 Python 模块、暴露凭据或扩大生命周期范围，Agent 必须停止并提出 Amendment。兼容解析器必须隔离，只接受完整已知输出，拒绝未知、残缺、超大输出，并具备固定版本夹具。

## 9. 身份与状态模型

- `instanceId`：一个 Instance 记录的、不透明且稳定的 YORVA 主身份。API resource path、Operation target 和未来 YORVA relation 使用该值。
- `runtimeInstallationId`：受支持 active Hermes installation 的 YORVA 身份。
- `nativeId`：官方接口返回的标准化 Runtime-native Hermes Profile 名称；只有 Hermes adapter 用它定位 Hermes。
- `name`：显示名称；初始与 `nativeId` 相同。
- `default`：Hermes 内置根 Profile；可见且受保护。
- 唯一约束：`(runtime_installation_id, native_id)`。

`instanceId` 与 `nativeId` 不得互换。`instanceId` 不得传给 Hermes、不得由 Profile 名推导，也不得伪装成 Runtime-native identity；`nativeId` 不得作为 YORVA 数据库主键、API `{instanceId}`、Operation target ID 或外键身份。

| 状态 | 含义 |
|---|---|
| `AVAILABLE` | 最近一次成功的 Hermes 权威查询确认 Profile 存在。它**不代表**已登录、模型已配置、Agent Ready、gateway 健康或进程生命周期已就绪。 |
| `MISSING` | 过去已知存在，但最近一次成功完整查询中不存在；YORVA row 与稳定 `instanceId` 继续保留。 |
| `UNKNOWN` | 查询失败、超时、取消，或输出无法安全解析。 |

`CREATING` 与 `DELETING` 属于 Operation 状态，不是 Instance 可用状态。外部重命名表示为旧项 `MISSING` + 新项 `AVAILABLE`；Phase 4 不推断身份连续性。同一 Runtime installation 下相同 `nativeId` 以后再次出现时，reconcile 必须把原 row 恢复为 `AVAILABLE` 并保留原 `instanceId`。

## 10. 对账合同

```text
完整查询官方 Hermes 接口
→ 校验并标准化全部 native identity
→ 开启短 SQLite 事务
→ 将存在的 Profiles upsert 为 AVAILABLE
→ 将此前存在但本次缺失的 Profiles 标记 MISSING
→ 提交缓存
```

规则：

- 运行 Hermes 时不得持有 SQLite 事务；
- 查询/解析/超时失败时保留旧记录，并将新鲜度标记为 `UNKNOWN`，不得推断缺失；
- 重复或非法 native identity 使整个 snapshot fail closed；
- reconcile 绝不创建、编辑或删除 Hermes 数据；
- Phase 4 不按时间、启动次数或刷新次数自动删除 `MISSING` row；它作为保留 YORVA identity 的永久 tombstone；
- 未来清理 tombstone 必须通过 Owner 明确批准的合同/migration，且不属于 Phase 4；
- 未知目录永不删除；
- daemon 启动、用户刷新与 mutation 成功后调用同一用例；
- 每个 Runtime installation 串行执行 Profile mutation 与 reconcile；不得复用 Phase 3 filesystem `install.lock`。

## 11. 创建合同

请求字段只有标准化 `name`。

YORVA 入口校验使用固定官方语法的封闭子集：首字符为小写 ASCII 字母；后续只能是小写字母、数字、`_` 或 `-`；拒绝官方保留名和 `default`。具体长度及保留名集合必须从已审核官方合同固化到测试，不能根据目录行为猜测。

推荐 adapter 调用等价于：

```text
<active-generation-hermes> profile create <name> --no-alias --no-skills
```

要求：

- executable 只能由已验证的 Phase 2/3 discovery 提供，不能使用环境 `PATH`；
- 设置固定 `%LOCALAPPDATA%\hermes` `HERMES_HOME` 和 allowlist 子进程环境；
- 使用直接 argv、现有 Windows Job Object containment 和有界输出；
- 不得传入 clone、config、model、skill、alias、credential 或任意 path 选项；
- 进程退出后重新查询 Hermes；只有请求的 Profile 被权威确认存在才成功；
- 重名标准化为 `INSTANCE_ALREADY_EXISTS`；
- 对不确定的部分创建结果不得自动清理；返回安全错误并 reconcile。

## 12. 删除合同

删除请求必须包含完全一致的标准化 `confirmationName`，且服务端验证它与当前 `nativeId` 一致。

推荐 adapter 调用等价于：

```text
<active-generation-hermes> profile delete <nativeId> --yes
```

要求：

- 启动进程前拒绝 `default` 和 protected Profile；
- 通过一次新的成功权威查询确认身份；
- 不接受路径，也不能仅凭 SQLite 推导删除权限；
- 进程退出后重新查询 Hermes；确认不存在即视为收敛成功，包括并发消失，并将 Instance row 标记为 `MISSING` 而不是删除；
- 最终状态不确定时保留 row 为 `UNKNOWN` 并返回稳定错误；
- 绝不卸载 Hermes、删除 generation 或删除未知目录。

## 13. Operation、超时、取消与并发

- POST create 与 DELETE 返回 `202` 及持久化 `instance.create` / `instance.delete` Operation。
- 两者都要求 `Idempotency-Key`；相同 key + 相同请求返回同一 Operation，冲突 payload 必须拒绝。
- 外部命令具备明确 whole-operation deadline、有界输出和 process-tree cleanup。
- Hermes 命令启动后 mutation 不可取消；启动前可取消。Desktop 在边界后隐藏 Cancel。
- 每个 Runtime installation 同时只运行一个 Profile mutation 或 reconcile。
- 数据库事务不得跨越外部进程执行。
- daemon 重启先恢复 Operation 投影，再由 reconcile 判断 Hermes 实际状态。Operation 状态不能证明 Profile 存在。

至少包含以下稳定错误：

```text
RUNTIME_NOT_SUPPORTED
INSTANCE_INVALID_NAME
INSTANCE_ALREADY_EXISTS
INSTANCE_NOT_FOUND
INSTANCE_PROTECTED
INSTANCE_CONFIRMATION_MISMATCH
INSTANCE_CONFLICT
INSTANCE_QUERY_FAILED
INSTANCE_OUTPUT_UNRECOGNIZED
INSTANCE_OPERATION_TIMED_OUT
CAPABILITY_NOT_SUPPORTED
```

原始子进程错误、输出、路径和环境变量不得成为 API 错误信息。

## 14. API / OpenAPI 合同

使用 `PROTOCOL.md` 已预留的路由：

```text
GET    /api/v1/runtimes/{runtimeId}/instances
POST   /api/v1/runtimes/{runtimeId}/instances
GET    /api/v1/instances/{instanceId}
DELETE /api/v1/instances/{instanceId}
```

POST body 为封闭 `{ "name": "..." }`；DELETE body 为封闭 `{ "confirmationName": "..." }`。缺少 body、未知字段、多个 JSON value、尾随垃圾和超大 body 必须以稳定错误拒绝。

Response 只暴露标准化 identity/state/capabilities/timestamps，不暴露 path、配置、凭据、环境变量、原始 Hermes 输出或子进程细节。Lifecycle route 返回 `CAPABILITY_NOT_SUPPORTED`；Desktop 不渲染生命周期控件。

## 15. 持久化

使用现有 `instances` 模型，只添加实现确实需要的确定性 migration。最低不变量：

```text
UNIQUE(runtime_installation_id, native_id)
```

SQLite 是缓存。缓存 row 不能授权 Hermes mutation、删除或文件访问。Phase 4 永久保留 `MISSING` tombstone，使再次出现的 `(runtime_installation_id, native_id)` 复用稳定 `instanceId`。测试必须覆盖空数据库、Phase 3 baseline 升级、唯一性、身份隔离、tombstone 保留与重启 reconcile。

## 16. Desktop UX 与 i18n

在现有侧边栏中增加 Instances 页面，必须提供中英文状态：

- loading/refresh、default/protected、命名 Profile 空状态；
- 带行内校验的创建表单；
- create/delete 进度与重启恢复；
- 要求输入名称的破坏性删除对话框；
- `AVAILABLE`、`MISSING`、`UNKNOWN`、冲突和超时指引；
- 不显示虚假的 lifecycle 控件，改为说明不可用。

使用 generated API types 与 server-state query。本地 React state 只保存 dialog/form 交互。时间按本地时区渲染。状态不能只依赖颜色；键盘焦点、标签和对话框必须可访问。

## 17. 安全与诊断

- Provider 凭据及任意继承的 `HERMES_*`、Python、Node 注入变量不得进入子进程；
- API、SQLite、日志、事件和 UI 中不得出现 secret plaintext；
- 结构化日志只包含 correlation/operation ID、action、安全 native identity、outcome、duration 和稳定 error code；
- 不记录原始 Hermes 输出或完整 filesystem path；
- create/delete 只在当前用户范围运行，永不要求 Administrator；
- 复用现有 process containment 与安全命令构造；
- 删除警告是产品安全控制，不等于 filesystem ownership 证明。

## 18. 实现批次——仅在授权后

实现必须按小型垂直能力交付，不能一次性跨所有层完成全部功能。Owner 授权后采用自动门禁：当前 Batch 通过即可进入下一 Batch，无需逐批等待 Owner，但不得提前实现后续 Batch。

### Batch 1 — 锁定官方合同

- 重新确认固定 Hermes Profile 命令、参数、输出和名称规则；
- 记录 D1 接口选择，并添加精确版本 parser/command 夹具；
- 不执行生产 Profile mutation，不建立通用 adapter framework。

门禁：fixture/contract tests 通过，接口不超出 Phase 4，diff 中没有 Batch 2+ 功能代码。

### Batch 2 — 只读 Instance 清单

- 实现最小 `instanceId` / `nativeId` domain model；
- 实现严格 Profile list parser、SQLite cache 与 reconcile；
- 通过 OpenAPI 暴露 GET list/get，并增加双语只读 Desktop 页面；
- 覆盖 `AVAILABLE`/`MISSING`/`UNKNOWN`、tombstone 保留和外部 add/remove。

门禁：adapter、persistence、application、protocol 与 Desktop 只读测试通过；此时不得存在 create/delete 子进程。

### Batch 3 — 创建一个最小 Profile

- 增加 `instance.create`、封闭 POST DTO、idempotency 与准确安全 CLI argv；
- 增加创建表单、进度和重启投影；
- 证明 clone、alias、Skills、credentials、model 与任意输入不会进入命令。

门禁：create、duplicate、timeout、process cleanup、API 与 Desktop 针对性测试通过；成功必须由权威重查确认。

### Batch 4 — 删除一个受保护 Profile

- 增加 `instance.delete`、封闭 DELETE DTO 与输入名称确认；
- 保护 `default`，删除前后重查，并将 row 保留为 `MISSING`；
- 增加双语破坏性警告与 Operation 状态。

门禁：delete、确认名不匹配、default 保护、并发消失、tombstone 与 Desktop 测试通过；不得删除未知路径。

### Batch 5 — 合同范围内的可靠性

- 只补齐第 13、17、19 节已经要求的 restart、concurrency、timeout、cancellation boundary、output limit 和 redaction；
- 验证 Phase 3 active generation/environment 不变量未改变；
- 修复这些测试发现的问题，但不引入第二套状态机。

门禁：受影响 Go/OpenAPI/Desktop/Windows process tests 全部通过，diff 中没有 Phase 5+ 或 lifecycle 行为。

### Batch 6 — 完整验证与审计交接

- 完整运行第 20 节，更新合同/完成证据，并取得 exact-commit CI；
- 除修复验证失败外，不再增加功能；
- 停止在 `AUDIT-004 = PENDING`，将准确 commit 交给独立 Agent。

每个 Batch 都必须：

1. 只检查本 Batch 的预期 diff；
2. 运行针对性测试及 `git diff --check`；
3. 形成可审查的 Batch commit，或同等隔离的 commit series；
4. 不得削弱合同或测试来通过门禁；
5. 只有 Gate PASS 才自动继续。

仅在出现架构冲突、必须扩展 lifecycle/Phase 5、官方接口不受支持、删除不安全、进程无法收敛、secret 暴露、需要重大依赖/框架或产品合同决策时停止并报告 Owner。

本 Batch 计划刻意保持有限。不得新增 plugin system、通用 workflow engine、dependency-injection framework、第二套 reconcile 状态机、filesystem ownership system、ACL/sandbox 项目、逐系统调用 failpoint 矩阵或推测性 Runtime 抽象。只防护本 Spec 明确的 Profile command、identity、delete、process 与 data 边界；无关强化记录为未来工作。

## 19. 测试矩阵

| 场景 | 必须结果 |
|---|---|
| Hermes 缺失/不支持 | 拒绝 mutation；不启动子进程。 |
| Active pointer 缺失/非法 | Fail closed；Phase 3 状态不变。 |
| 只有官方 default | default 可见且 protected。 |
| 有效 create | 权威重查后一个 `AVAILABLE` Instance。 |
| 非法/保留/path-like name | 子进程前拒绝。 |
| 重复 create/idempotency replay | 一个 native Profile 和一个 Operation。 |
| Create timeout/后代进程 | 终止整个树；最终状态 reconcile，不猜测。 |
| No-clone create | YORVA 不复制凭据/配置/模型/Skills。 |
| 已确认 delete | 权威重查后 Profile 不存在；保留 row 为 `MISSING`。 |
| default 或确认名不匹配 | 子进程前拒绝。 |
| Delete 并发消失 | 幂等收敛到不存在。 |
| 查询 timeout/malformed/oversized | 旧记录变 `UNKNOWN`，不得误判 `MISSING`。 |
| 多次 restart/refresh 后仍缺失 | Row 保持 `MISSING`；没有 TTL 或自动清理。 |
| 相同 native Profile 再次出现 | 保留原 `instanceId`，状态回到 `AVAILABLE`。 |
| `instanceId` / `nativeId` 边界 | YORVA route/relation 使用 `instanceId`；只有 Hermes adapter 调用使用 `nativeId`。 |
| Profile 存在但无模型/登录 | 只能是 `AVAILABLE`，不得声称 Agent/model/login Ready。 |
| 外部 add/remove/rename | 缓存收敛且不修改 Hermes。 |
| 输出重复 native identity | 整个 snapshot 拒绝。 |
| `HERMES_HOME` 下未知目录 | 不作为权威，永不删除。 |
| 遗留 `hermes-agent` | 不接管为第二安装。 |
| 并发 mutation/reconcile | 串行化，无破损缓存或重复项。 |
| daemon/Desktop 重启 | 恢复 Operation；Hermes 查询决定真实状态。 |
| secret sentinel | 不出现在 child、API、日志、事件或 UI。 |
| API 未知/尾随字段 | 稳定 `INVALID_REQUEST`。 |
| Lifecycle action | capability false 且 `CAPABILITY_NOT_SUPPORTED`。 |
| Phase 3 不变量 | `active.json`、generations、Seal、PATH、HERMES_HOME 不变。 |
| 中英文 UI | 行为一致、本地化且可访问。 |

## 20. 完整验证矩阵

审计前运行并记录：

- API lint、OpenAPI validation、generated client drift；
- Desktop typecheck、lint、tests、build、dependency audit；
- Go format、full/repeated targeted tests、CI race、vet、build、vulnerability scan；
- Rust format、tests、clippy、check、dependency audit；
- 使用安全夹具的 Windows real-process containment/timeout smoke；
- 如果 packaging input 改变，则运行 Tauri release no-bundle 与 MSI/package 检查；
- exact-commit GitHub Actions success。

真实 Hermes Profile smoke 必须使用隔离账号/VM 和可丢弃数据，绝不能对 Owner 的 Profile 自动运行 delete smoke。

## 21. 退出条件

- [x] 第 3 节决策获得 Owner 批准。
- [ ] 官方接口资格与固定版本夹具已记录。
- [ ] 多个 Profiles 能对账为唯一 YORVA Instances。
- [ ] 用户无需终端即可创建并执行受保护的破坏性删除。
- [ ] 查询失败永远不会变成错误的缺失判断。
- [ ] SQLite 保持缓存，Hermes 保持权威。
- [ ] 不改变 Phase 3 generation/environment 不变量。
- [ ] Diff 中没有 Phase 5+ 功能或 lifecycle 实现。
- [ ] 完整验证与 exact-commit CI 通过。
- [ ] 独立 `AUDIT-004-instance-profile.md` 达到 `PASS` 或 Owner 接受的 `PASS WITH CONDITIONS`。

实现完成时停止在：`Phase 4 Implementation = COMPLETE`、`Verification = PASS`、`AUDIT-004 = PENDING`。实现 Agent 不得合并 `main`、冻结、打 tag、删除功能分支或开始 Phase 5。

## 22. 审计要求

独立 Auditor 必须验证：范围、官方接口 provenance、严格解析、命令安全、进程清理、删除保护、reconcile 权威性、缓存语义、并发、重启、脱敏、API 封闭性、双语 Desktop 行为和全部 Phase 3 不变量。

任何 Critical 或 High finding 都使 Gate FAIL。Medium 按 `AUDIT_STANDARD.md` 处理；只有 Owner 明确接受才能 `PASS WITH CONDITIONS`，且不能在实现后削弱验收标准。

## 23. Agent 执行指令

只有在 Owner 批准并授权后，Implementation Agent 才能：

1. 阅读 `AGENTS.md`、治理/架构/protocol/runtime/data/security 文档、相关 ADR、本英文 Spec 与获批中文镜像；
2. 锁定准确起始 commit，并创建非 `main` 的 Phase 4 功能分支；
3. 按第 18 节逐个垂直 Batch 执行；当前 Batch 自动 Gate 通过并隔离 diff 后才能进入下一批；
4. 保留用户工作及所有历史审计报告；
5. 停止在 implementation complete + verification pass + audit pending；
6. 将准确 commit 交给新的独立审计 Agent。

本指令本身不构成实现授权。

## 24. 完成证据

```text
Implementation commit:
Branch:
Batch results:
Verification matrix:
Exact-commit CI:
Known non-blocking risks:
AUDIT-004: PENDING
```

# YORVA Phase 7 — Hermes Runtime 日常管理完善

> 状态：**FAILED — 2026-08-27 冻结审计发现阻断项，返回 P7R 修复**
> 阶段：Phase 7
> Owner：Repository Owner
> 计划日期：2026-08-24
> 必需基线：`phase-0065-developer-led-demo-baseline` → `5f68e48f17e7e342e1781b37613b19d4bd1f060b`
> 已冻结 Phase 6 基线：`phase-006-runtime-lifecycle-messaging-channels-baseline` → `7ca9103e7af210296a5e24916df01856539b550e`
> 当前 Hermes 资格确认目标：稳定 `>=0.20.2 <0.21.0`；开发参考版本 `0.20.5` / `a0ca7c19204e514f9590ce3b812e029b315ab9e9`
> 路线图条目：`ROADMAP.md` Phase 7 — Runtime management completeness
> 英文执行镜像：`docs/phases/PHASE-007-hermes-runtime-management-completeness.md`
> Owner 授权：2026-08-24，批准 P7-D1–D8 推荐方向及 B0–B10 实现顺序
> 实现授权：**Owner 于 2026-08-25 批准依赖驱动的并行 B-stage；共享合同先行，互不依赖的 Hermes adapter、测试与 UX lane 可并行，危险 mutation 必须等待对应 ADR/资格条件**
> ADR 授权：**Owner 于 2026-08-25 正式接受 ADR-0013、ADR-0015；Owner 于 2026-08-27 将 ADR-0019 完全自定义 MCP surface 延后到 P7R 之后。每项 product capability 仍须通过精确版本资格和生命周期证据后才能置为 true。**
> ADR 授权补充：**Owner 于 2026-08-25 接受 ADR-0016；exact Profile 的 `API_SERVER_KEY` 是 Hermes-native 唯一权威，仅允许 loopback、禁止重定向地读取 `/health/detailed` 与 `/v1/skills`。真实 Health/SkillRead capability 仍须在聚焦资格与接线通过后才能置为 true。**
> B4 架构补充：**Owner 于 2026-08-25 接受 ADR-0018。Hermes Native Skills mutation 的不可靠部分保持 `deferred_upstream`；YORVA 通过自己的 managed store 与 exact Profile copy projection 提供 Skills 生命周期。**

## 0. 阶段定位

Phase 7 的目标是让普通用户在不打开终端、不编辑 Hermes 文件的情况下，完成
Hermes 的日常本地管理。

本阶段所称“完整 Hermes 管理”具有明确边界：

```text
YORVA 已有能力
  安装 / 检测 / Instance / 模型 / 生命周期 / 微信与企业微信
        +
Phase 7 日常管理能力
  健康诊断 / 安全日志 / Skills / MCP / 备份恢复 / Hermes 升级
```

它不表示把 Hermes `--help` 中的每一个命令都搬进 YORVA，也不表示建立任意命令、
文件或插件执行平台。Sessions、Cron、Memory、Plugins、Portal、Computer Use、Hooks、
Kanban 和其他新增 Hermes 产品面只有在 Owner 以后通过 Phase 7 Amendment 明确批准时
才可进入。

Phase 7 保持 YORVA 是 Runtime 管理层而不是 Hermes fork：Hermes 拥有 Hermes 状态，
YORVA 拥有 Operation、策略、安全投影、备份索引、审计和用户体验。

Skills 来源允许已审核目录，以及用户通过原生选择器明确选择的本地 ZIP/目录。后者
只通过一次性不透明引用进入 daemon，必须复制到 YORVA 受管存储并通过根级
`SKILL.md`、纯文本内容、路径、链接与大小限制校验后，才能投影到所选 Profile；
HTTP 不接受本地路径、URL 或任意包内容。

## 1. 进入条件

Phase 7 实现只能在以下条件全部满足后开始：

- P6.5 已正式 `COMPLETE / FROZEN`，或者 Owner 明确决定放弃 P6.5 合并并回到 Phase 6
  frozen baseline；
- Phase 7 分支确实以后续接受基线为祖先，且没有把未接受的 Demo 工作混入；
- 本文件的 P7-D1 至 P7-D8 已获得 Owner 决策；
- Batch 1 的 Hermes `0.20.5` 官方 surface 资格证据已形成，未通过的直接 surface 在对应
  lane 保持关闭；
- 备份秘密处理、MCP credential authority 和 generation upgrade 的必要 ADR 在对应公开
  capability 或 mutation 接入前被接受；其不依赖的 parser、closed descriptor、archive
  verifier、transaction 与测试工作可以并行准备；
- 中英文 Spec 已同步，状态改为 `READY`；
- Owner 明确授权开始实现和允许的 batch 推进方式。

本 Draft 可以被审阅和修改，但不授权 migration、接口、依赖、Runtime mutation、
commit、push、merge 或 tag。

## 2. 目标

当 Phase 7 成功时，用户能够在 YORVA Desktop 中：

1. 查看 Hermes Runtime 与每个 Instance 的安全健康状态和有限日志；
2. 查看、检查、安装、启用/禁用、更新、审计和移除受信任 Skills；
3. 查看、添加、认证、测试、配置和移除受支持 MCP Server；
4. 创建可识别、可校验、不会明文泄露秘密的 Hermes 备份；
5. 在冲突保护、预检查、备份和失败恢复下执行 Restore；
6. 把 YORVA 管理的 Hermes generation 升级到当前 YORVA 构建携带并验证的目标版本；
7. 在所有长操作失败、中断或 daemon 重启后看到真实状态和明确恢复入口。

## 3. 用户可见成功流程

### 3.1 日常状态与恢复

```text
用户打开 Runtime / Instance
  → YORVA 查询 live Hermes 状态
  → 显示 HEALTHY / DEGRADED / UNHEALTHY / UNKNOWN
  → 用户查看有界、脱敏的诊断与日志
  → 如果存在可修复问题，用户明确启动对应 Operation
  → YORVA 验证 postcondition 后显示最终结果
```

### 3.2 Skills

```text
选择准确 Instance
  → 查看 Hermes 权威 Skill inventory
  → 预览来源、版本、权限/风险与审计结果
  → 从获批来源安装或更新
  → Hermes adapter 重新读取准确 Profile 的有效 Skill 状态
  → Desktop 显示真实结果及失败恢复
```

### 3.3 MCP

```text
选择准确 Instance
  → 查看已配置 MCP 和获批 catalog
  → 选择受支持的 MCP 定义
  → 必要时完成短时认证
  → 运行有界连接/工具发现测试
  → 只在权威 read-back 成功后显示 READY
```

### 3.4 Backup / Restore

```text
选择 Runtime 或经资格确认的 Instance scope
  → YORVA 选择受保护的每用户系统应用数据目录
  → YORVA 创建 backup Operation
  → 生成、校验并安全落盘
  → Desktop 显示大小、checksum、scope、版本和创建时间
```

```text
选择备份
  → YORVA 预检查格式、scope、版本、checksum、空间和冲突
  → 明确展示会停止哪些 Instance、会恢复哪些状态
  → 创建恢复前保护点
  → 执行 Restore
  → reconciliation 和健康检查
  → 成功，或回滚并如实报告未知/失败状态
```

### 3.5 Hermes Upgrade

```text
YORVA 检测当前 managed generation 与本构建携带的目标 Hermes snapshot
  → 用户查看版本、来源、影响、备份和 gateway restart 计划
  → 创建 runtime.upgrade Operation
  → 构建新的 sealed generation，不修改 active generation
  → final-path 验证
  → active.json compare-and-swap 激活
  → Instance / lifecycle / model / channel / Skill / MCP post-check
  → 成功；失败时保留前一 generation 并提供有界 rollback
```

## 4. Owner 决策表

以下决策在实现前必须锁定。表中“建议”不是 Owner 批准。

| 决策 | 建议方案 | 状态 |
| --- | --- | --- |
| P7-D1 范围定义 | 以 Health/Logs、Skills、MCP、Backup/Restore、Upgrade 和 recovery UX 作为 Phase 7 完整范围；其他 Hermes 命令延后 | **APPROVED — 2026-08-24** |
| P7-D2 Skills 来源 | 第一版只允许经 inspect/audit 的官方或 Owner 批准 catalog/registry 标识；禁止未检查的直接 URL 和 `--force` 绕过扫描 | **APPROVED — 2026-08-24** |
| P7-D3 MCP 来源 | P7R 只支持内置和 YORVA 审核 Preset/Definition；调用者提供的 stdio command、args、environment、HTTP headers、路径和任意 JSON 延后 | **AMENDED — 2026-08-27** |
| P7-D4 Backup scope / secrets | 优先实现 Runtime-scope 加密完整备份；若无法建立标准加密格式和可靠 Restore，则不把 Hermes 明文 full backup 作为产品能力 | **APPROVED — ADR REQUIRED** |
| P7-D5 Upgrade | managed Hermes 只通过 ADR-0006/0009 新 generation 升级；不执行 active tree 上的 `hermes update`，不使用 `--force-venv` | **APPROVED — ADR REQUIRED** |
| P7-D6 Health/Logs | 只展示 allowlisted 类别、固定上限、强制脱敏的结构化/标准化结果；不提供任意路径 tail 或原始日志下载 | **APPROVED — 2026-08-24** |
| P7-D7 本地鉴权 | P7 保持一个 authenticated local Desktop trust context，同时为每个 use case 固定 typed action 和 audit actor；Principal/Grant/RBAC 延后 P9/P10 | **APPROVED — 2026-08-24** |
| P7-D8 版本资格 | stable `0.20.x` 可检测，但每个 P7 feature 只有通过 exact-version surface qualification 才报告 capability；未知 contract fail closed | **APPROVED — 2026-08-24** |

Owner 随后的“按照这份文档进行实现”指令构成本表批准和执行授权。2026-08-25 Owner
进一步批准依赖驱动的并行 B-stage：B1 证据不再作为整个 P7 的串行全局锁，而是决定
每个 lane 可使用或必须关闭的 surface。共享 B2 合同先行；互不修改同一模块、且不存在
真实前置依赖的 B3–B9 工作可以并行。未接受 ADR 仍阻止对应公开 mutation/capability，
但不阻止其他 lane 继续。

## 5. 已确认的 Hermes 0.20.5 规划事实

2026-08-24 对官方 checkout `a0ca7c1…` 的只读 `--help` 检查确认存在：

- `hermes status`、`doctor`、`logs`、`monitoring status`；
- `hermes security audit --json`；
- `hermes skills` 的 list/inspect/install/update/audit/uninstall/config 等命令；
- `hermes mcp` 的 catalog/list/add/install/test/configure/login/remove 等命令；
- `hermes backup` 与 `hermes import`；
- `hermes update --check` 与 `hermes update --plan`；
- Sessions 和 Cron 等更大表面。

这些事实只证明功能入口存在，不证明适合后台自动化。除 `security audit --json` 外，
多个帮助面未声明 machine-readable 输出。Batch 1 必须继续证明 non-interactive、
Profile/scope 精确、输出边界、失败语义、credential transport、并发和 postcondition。

另一个已确认的硬事实是：官方 full/quick backup 会包含 `.env`、auth 等敏感状态。
因此普通明文 ZIP 不能直接满足 YORVA 当前 `SECURITY.md` 的 backup 边界。

## 6. 范围内

### 6.1 Runtime/Instance 健康与安全诊断

- Runtime 和 Instance 的 live normalized health；
- 静态 Doctor 检查；`--live` 网络检查必须是单独、显式操作；
- Hermes 依赖/MCP/Plugin 的受支持安全审计；
- fixed-category、fixed-limit、redacted log snapshot；
- cursor 或固定时间窗的安全读取；
- UNKNOWN、timeout、malformed、unsupported 和 partial-result 语义；
- 不把日志内容持久化到 SQLite、Operation 或审计表。

### 6.2 Skills 管理

- 准确 Runtime/Profile scope 的 inventory 和 bounded inspect；
- Hermes Native inventory/mutation capability 与 YORVA managed lifecycle 分层；
- `{dataDir}/skills/managed` 作为 YORVA managed copy 的唯一权威；
- 获批 closed source 的 install、update 和 remove；
- enable = copy projection，disable = 只移除已证明属于 YORVA 的 projection；
- SQLite ownership/deployment record、一个完整包 SHA-256 和小型 projection marker；
- external、runtime bundled、unknown Skill 可见但默认只读；
- `DRIFT_MISSING` / `DRIFT_MODIFIED` 检测与安全重新投影。

Hermes 仍是 Native Skill observation 的权威；YORVA 只对自己的 managed copy、所有权记录和预期 projection 负责。

### 6.3 MCP 管理

- live configured MCP inventory；
- 内置 Preset 与 YORVA-managed Runtime Definition inventory；
- 一个生产可见、由 YORVA 持有的回环生命周期 Preset（`yorva-mcp-test`），固定身份且仅提供 `yorva_ping`；
- 从 YORVA 审核 Preset 创建可复用 Definition，并绑定或删除；
- 当前 P7R MVP 不开放任意 stdio command、args、environment、HTTP headers 或任意 MCP JSON；
- OAuth/device/browser flow 只有在可安全限定于发起 session 时才进入；
- connection/tool-discovery test；
- tool selection 的 closed mutation；
- remove、reauth 和 external-change reconciliation；
- MCP process/network timeout、cancel 和 cleanup。

完全自定义 MCP 延后到 P7R 之后的独立资格阶段。P7R 的 Desktop/API 只能选择审核
Preset 和 Preset 规定的凭据、工具范围；不能提交 executable、argv、environment、
headers、路径或原始配置。凭据只写入精确 Profile 的 authority，不能进入 SQLite 或
普通 GET 回应。

### 6.4 Backup 与 Restore

- 明确 target scope、Runtime version、创建时间、大小、checksum、格式版本和状态；
- YORVA 每用户系统应用数据目录中的固定目标位置；
- 创建、校验、列出、删除；
- restore preflight、冲突阻止、恢复前保护点、post-check 和失败回滚；
- corrupt、truncated、tampered、unsupported-version、insufficient-space 处理；
- archive traversal、absolute path、reparse/symlink、ADS 和 expansion bounds；
- 明确 secrets/session/account 数据包含范围；
- plaintext staging 的生命周期和崩溃清理；
- 备份不会自动上传到 Cloud。

在 P7-D4 所需的 ADR、标准加密格式和 Restore recovery 资格确认完成前，Phase 7
不得直接包装 `hermes backup` / `hermes import --force` 为“安全备份恢复”。

### 6.5 Hermes Upgrade 与 Rollback

- 当前版本与当前 YORVA 构建所携带目标 snapshot 的比较；
- upgrade plan，包括受影响 Profiles/gateways、备份和 restart 行为；
- 只对 YORVA-managed installation 执行 mutation；
- verified source、size、SHA-256、license 和 installer input；
- 新 Install Transaction 和新 final-path generation；
- seal、functional validation、publish、activation CAS 和 retention；
- active generation 不可变；
- upgrade 后全量 reconciliation；
- 前一 generation rollback，仅在 Hermes user-data/schema 兼容性已证明时允许；
- unmanaged official checkout 默认只提供检查/说明，不擅自修改用户 Git 工作树。

### 6.6 Feature-specific recovery UX

- orphaned Operations 在 daemon restart 后按 Runtime truth reconciliation；
- 不盲目 replay destructive mutation；
- clear retry、resume（仅证明安全时）、rollback 或 manual recovery action；
- 失败阶段、稳定 error code、correlation ID 和安全 diagnostics；
- 不把“命令退出 0”单独当作成功。

## 7. 非目标

Phase 7 不实现：

- Control Plane、远程 Node、组织、Principal/Grant 持久化、RBAC、SSO 或多用户 daemon；
- 微信/企业微信消息发送者到 YORVA Principal 的企业权限映射；
- 任意 shell、process、PowerShell、cmd、filesystem、registry、service 或 environment API；
- 任意 MCP JSON、YORVA 拼接 shell 字符串、明文持久化 MCP secret 或无 owner 的 child process；
- 跳过 Skill/MCP scan、`--force` 安装、`--yolo` 或自动批准未知 hooks；
- Hermes source fork、Python internal import、undocumented DB schema 依赖；
- active sealed generation 上的 `hermes update`、repair 或 dependency mutation；
- Hermes uninstall；
- Cron、Sessions/对话管理、Memory、Plugins、Portal、Computer Use、Hooks、Approvals、
  Kanban、Projects、Peer、Webhooks、Dashboard 或 agent chat UI；
- 自动上传 backup、log、debug report、session 或 telemetry；
- 未加密、未披露敏感范围的完整 Hermes backup；
- 把 YORVA SQLite cache 当作 Hermes Skill/MCP/health authority；
- 第二 Runtime、动态 Runtime plugin SDK 或 marketplace。

## 8. 架构与所有权

要求方向：

```text
React Desktop
  → authenticated typed local API
  → Runtime-neutral application use case
  → compile-time Runtime feature lookup
  → Hermes-owned adapter
  → qualified official surface / separately approved narrow compatibility path
```

Core/application 拥有：

- stable public Instance/Runtime identity；
- typed management intent 与 action name；
- Operation、idempotency、conflict、timeout、cancellation 和 recovery decision；
- normalized state、stable error、audit metadata；
- backup index 和 YORVA-owned encryption metadata（若 P7-D4 批准）。

Hermes adapter 拥有：

- Profile/global scope targeting；
- exact Hermes command/API/RPC selection；
- human/structured output qualification parser；
- Hermes Skill/MCP/backup/update semantics；
- Runtime-native config、credential 和 user-data paths；
- exact postcondition query 和 Hermes error normalization。

按真实调用者添加小型 feature contracts。不得先建立一个包含全部 Hermes 命令的
`HermesManager`，也不得建立动态插件框架。

Desktop 信息架构以 Runtime 为管理中心：Runtime 页面统一承载实例清单、Skills/MCP
共享资源与多实例分配、Upgrade、Backup/Restore、诊断和 Operation；Instance 页面只
展示生命周期、模型、Channel、Skill/MCP 绑定及该实例的健康与日志。此重组复用既有
Runtime/Instance API，不改变 capability、credential authority 或资格确认边界，也不把
Instance health 误表述为 Runtime-wide health。

Runtime 管理 UI 将 Skills 与 MCP 拆为两个独立页面。Skills 页面只保留一个准确的
“正在配置”实例选择器，不再并列显示语义重复的“目标实例/应用到实例”。
Skills inventory 和 MCP definition inventory 读取准确 Hermes Profile 已有状态；Profile
配置兼容读仅在 Hermes adapter 内发生，MCP 投影只返回名称与标准化配置状态，禁止把
URL、command、args、header、environment value 或 credential 返回 Desktop/API。
Runtime 资源页采用扁平设置页样式：资源行不叠加装饰性卡片外框，以紧凑分隔线和明确
所有权分组组织内容。YORVA 管理的 Skills 保持直接可见和可管理；Hermes、Runtime 或
外部管理的 Skills 独立分组并默认折叠，在原生 mutation 另行资格通过前保持只读。
诊断不占用 Runtime 顶部导航栏；从实例列表的“诊断日志”入口进入，并可在诊断页直接
切换实例。诊断页展示准确 Profile 的部分健康快照和固定类别、定长、脱敏的本地运行日志；
存在已认证 loopback health API 时，以该实时健康结果替代 Profile 的部分健康快照。

概念边界：

```go
type SkillManager interface { /* list/inspect/mutate only after qualification */ }
type MCPManager interface { /* list/catalog/test/mutate only after qualification */ }
type BackupManager interface { /* create/inspect/restore only after P7-D4 */ }
type UpgradeManager interface { /* plan/upgrade/rollback managed installation */ }
type HealthInspector interface { /* normalized health and bounded diagnostics */ }
```

最终 method 和 DTO 由 Batch 1 的真实 surface 证据锁定；本 Draft 不授权照抄概念代码。

## 9. 通用鉴权准备边界

P7 保持 Phase 1–6 的一个 authenticated local Desktop trust context，不提前建立企业
RBAC。为了避免 P9 再重写管理用例，每个 P7 use case 必须具有一个稳定 action：

```text
runtime.health.read
runtime.logs.read
runtime.security.audit
skill.read / skill.install / skill.update / skill.remove / skill.configure
mcp.read / mcp.install / mcp.authenticate / mcp.test / mcp.remove / mcp.configure
backup.read / backup.create / backup.restore / backup.delete
runtime.upgrade.plan / runtime.upgrade / runtime.rollback
```

P7 的 actor 固定为经过 local bootstrap 认证的 `LOCAL_DESKTOP`。Action 进入 application
use case 和安全 audit metadata，不进入 Hermes argv。没有 `Principal`、`Grant` 或 role
表，也不把 Channel `CONNECTED` / sender pairing 当作 YORVA 企业授权。

未来 Control Plane 必须在投递前 authorize，并由 Node 对同一个 typed action 再执行
本地 policy。它仍调用 P7 的同一 application use case，不能绕过到 Hermes adapter。

## 10. 标准化状态

### 10.1 Health

```text
HEALTHY
DEGRADED
UNHEALTHY
UNKNOWN
```

`UNKNOWN` 不能自动触发 repair。Deep/live check 是显式 Operation，且网络调用要在 UI
中说明。

### 10.2 Skills

Skill 的事实字段以 qualification 为准，至少区分：

```text
INSTALLED / NOT_INSTALLED / UNKNOWN
ENABLED / DISABLED / UNKNOWN
CLEAN / WARNING / BLOCKED / NOT_SCANNED / UNKNOWN
```

安装状态、启用状态和扫描状态不能压缩成一个布尔值。

### 10.3 MCP

建议 normalized state：

```text
NOT_CONFIGURED
CONFIGURED
AUTH_REQUIRED
READY
FAILED
UNKNOWN
```

`CONFIGURED` 不等于连接测试成功；`READY` 必须来自本次有界权威测试或有明确时间的
有效 evidence。

### 10.4 Backup

```text
CREATING
AVAILABLE
RESTORING
FAILED
DELETING
```

Operation transient state 与 durable backup metadata 分开。只有 checksum/format
验证通过的文件可以是 `AVAILABLE`。

## 11. API / Protocol 候选

Batch 1 后由 OpenAPI 锁定最终路径。候选资源如下：

```text
GET  /api/v1/instances/{instanceId}/health
POST /api/v1/runtimes/{runtimeId}/health/deep-checks
POST /api/v1/runtimes/{runtimeId}/security-audits
GET  /api/v1/instances/{instanceId}/logs

GET    /api/v1/instances/{instanceId}/skills
GET    /api/v1/instances/{instanceId}/skills/{skillId}
POST   /api/v1/instances/{instanceId}/skills/{skillId}/install
POST   /api/v1/instances/{instanceId}/skills/{skillId}/update
PATCH  /api/v1/instances/{instanceId}/skills/{skillId}
DELETE /api/v1/instances/{instanceId}/skills/{skillId}

GET    /api/v1/runtimes/{runtimeId}/mcp-definitions
GET    /api/v1/instances/{instanceId}/mcp-bindings
PUT    /api/v1/instances/{instanceId}/mcp-bindings/{serverId}
PUT    /api/v1/instances/{instanceId}/mcp-bindings/{serverId}/credential
POST   /api/v1/instances/{instanceId}/mcp-bindings/{serverId}/test
PATCH  /api/v1/instances/{instanceId}/mcp-bindings/{serverId}
DELETE /api/v1/instances/{instanceId}/mcp-bindings/{serverId}

GET  /api/v1/runtimes/{runtimeId}/backups
POST /api/v1/runtimes/{runtimeId}/backups
POST /api/v1/backups/{backupId}/restore
DELETE /api/v1/backups/{backupId}

GET  /api/v1/runtimes/{runtimeId}/upgrade-plan
POST /api/v1/runtimes/{runtimeId}/upgrade
POST /api/v1/runtimes/{runtimeId}/rollback
```

Batch 1/当前 OpenAPI 已把首个静态 health 投影锁为 exact-Instance 资源；它不被表述为
Runtime-wide 健康。Runtime-wide health 继续保持未注册，直到出现独立且已资格确认的
Runtime 目标语义。Logs 同样只接受一个固定类别。Security audit 与 deep/live check 仍是
显式 Operation，不注册同步 GET 捷径。

规则：

- mutation 使用 closed typed body 和 `Idempotency-Key`；
- secrets write-only，不从 GET、Operation、event 或 error 返回；
- source identifier、preset、tool selection、log source 和 filter 都是 allowlisted schema；
- 不接受 shell command、environment key/value、arbitrary local path 或 URL query secret；
- 系统托管的 backup destination 是明确 local-only capability，由 daemon 从受信任的
  YORVA app data 与 backup ID 推导；React/HTTP 不提交路径，也不成为未来 remote
  command 参数；
- Runtime-scope backup index 可在仓储可用后独立发布为 authenticated read-only
  capability。list/get 只返回最近一次观测的安全元数据，不打开、解密、hash、reconcile
  或修改 artifact；Backup create/delete 与 Restore capability 在各自完整 Operation
  路径通过 Gate 前继续保持 false；
- long-running work 返回 `202 Operation`；
- Desktop 通过 TanStack Query 持有 daemon state，SSE 只做 invalidation/progress 通知。

## 12. Operation 与并发

候选 Operation：

```text
runtime.health.deep-check
runtime.security.audit
skill.install / skill.update / skill.remove / skill.audit
mcp.install / mcp.authenticate / mcp.test / mcp.remove
backup.create / backup.restore / backup.delete
runtime.upgrade / runtime.rollback
```

冲突矩阵：

| 操作 | 必须冲突的操作 |
| --- | --- |
| Skill mutation | 同 scope Skill mutation；Restore；Upgrade |
| MCP mutation/auth/test | 同 server mutation；Restore；Upgrade；必要时 lifecycle transition |
| Backup create | Restore；Upgrade；是否与 live gateway 并发由 P7-D4 qualification 决定 |
| Restore | 所有受影响 Instance mutation、lifecycle、Channel、Skill、MCP、Backup、Upgrade |
| Runtime Upgrade/Rollback | Install、Prerequisite、所有 Instance mutation、Restore、Skill/MCP mutation |
| Log/Health read | 通常可并发；deep/live check 有独立 bounded budget |

锁必须位于 application coordination，并使用最窄的真实 scope。等待网络、Hermes、
archive、MCP 或 process 时不得持有数据库 transaction。所有 goroutine/process/stream
必须有明确 owner、timeout、cancel 和 reap/close 路径。

## 13. Secret、日志与供应链边界

- Backup、MCP OAuth/header credential、Skill registry credential（若未来批准）和 update
  token 都按 secret 处理；
- 不进入 argv、URL、SQLite plaintext、Operation、event、log、diagnostic、audit metadata、
  Desktop storage 或 ambient child environment；
- 普通日志视图必须再次经过 YORVA redaction，不能假设 Hermes 已经脱敏；
- 日志输出有行数、单行、总字节、时间窗和并发读取限制；
- Skill install 前必须 inspect + audit，并明确 source identity；
- 禁止 `--force` 越过 blocked scan；
- MCP catalog descriptor 必须被 review/allowlist；
- backup archive 必须防 traversal、symlink/reparse、ADS、过量 member 和 expansion bomb；
- Upgrade 使用当前 YORVA 构建所携带或明确 review 的 immutable snapshot，不跟随浮动
  branch；
- 不运行 `hermes update --force`、`--force-venv` 或会 stash/改写用户外部 checkout 的
  自动路径；
- 若引入加密依赖，必须使用维护中的标准格式/库，不设计自有密码学协议，并单独做
  dependency/security review。

## 14. Persistence 计划

默认原则：Hermes Skill/MCP/config/log/health 仍由 Hermes 权威查询，不建立 shadow DB。

可能需要的持久化变化：

- `operations`：增加 P7 operation type 的 closed validation/冲突约束；
- `audit_log`：写入 action、target、result、`LOCAL_DESKTOP` actor 和 correlation ID；
- `backups`：当前 schema 是 Instance-scoped，但官方 Hermes backup 是整个 Hermes home。
  P7-D4 必须先决定 Runtime scope 或经证明的 Instance scope，再设计 migration；
- 加密 backup 只保存非秘密 metadata 和安全 key reference，密钥材料进入 OS-backed
  `SecretStore`；
- Upgrade 继续使用 filesystem Install Transaction、manifest/seal 和 `active.json`，
  SQLite Operation 不能成为 activation authority。

Migration 必须从空数据库和正式 P6/P6.5 前序 schema 测试。不得为了适配 Runtime-scope
backup 静默删除或重新解释已有 backup row。

## 15. 实现计划表

采用依赖驱动而不是纯编号串行。B2 的共享合同是集成前置；其余 Batch 被拆成文件和
模块所有权互不重叠的 lane。每个 lane 做聚焦测试与集成检查，完整十二维独立审计只在
B10 统一候选时执行。一个 lane 的非致命缺口不冻结其他 lane；强制停止条件只阻断受
影响的危险 surface，除非它破坏共享信任边界或候选完整性。

| Batch | 交付 | Gate |
| --- | --- | --- |
| B0 — 基线与范围锁定 | 确认 P6.5 处置、建立 clean successor branch、确认 P7-D1–D8、同步中英文 Spec | ancestry/working-tree/scope review；无代码 |
| B1 — Hermes surface qualification | 对 `0.20.5` Skills/MCP/backup/import/update/status/doctor/log/security 建立资格与风险证据；直接 surface 的 NO-GO 只关闭对应路径 | evidence preserved；安全替代设计进入对应 lane，禁止包装被拒 surface |
| B2 — Core capability / protocol / audit action | 添加最小 feature contracts、registry capability、typed actions、Operation/error 状态、OpenAPI skeleton 和 migration（仅已决部分） | Go contract/API/migration tests + OpenAPI drift |
| B3 — Health / Logs / Security | 先交付 read-mostly normalized health、bounded redacted logs、显式 deep check 和 security audit | parser/size/redaction/timeout/API/Desktop focused Gate |
| B4 — Skills Lifecycle | Native capability truth、YORVA managed store、approved install/update/enable/disable/remove、inventory merge、reconcile 和 Desktop | source/ownership/external conflict/Profile isolation/drift/restart/manual smoke |
| B5 — MCP | 审核 Definition + Instance Binding 的 install/auth/test/tool-config/remove、凭据隔离、超时取消与 Desktop | closed schema、secret isolation、权威回读与生命周期 smoke |
| B6 — Backup Create | scope/migration、加密或获批安全格式、create/verify/list/delete、Desktop | secret/temp/crash/archive-integrity/space/manual smoke |
| B7 — Restore | preflight、stop/conflict、保护点、restore、reconcile、rollback 和 Desktop | corrupt/tamper/version/cross-scope/partial-failure/destructive manual smoke |
| B8 — Managed Hermes Upgrade / Rollback | plan、new generation build、seal、activate、post-check、retention 和 rollback | exact-source/final-path/CAS/data compatibility/lifecycle/channel/manual smoke |
| B9 — 完整 UX / Recovery | 跨 feature 状态、重试/恢复、i18n、accessibility、diagnostics 和 onboarding | end-to-end Desktop flow + daemon restart recovery |
| B10 — Candidate / Audit / Freeze | 完整 Gate、Windows smoke、精确候选 CI、独立审计、必要修复与重审、Owner Gate | PASS 后才允许 merge/final-main/tag |

并行 lane：

```text
B2 shared contracts ─┬─ B3 Health / Logs / Security
                     ├─ B4 Skills
                     ├─ B5 MCP
                     ├─ B6 Backup Create ─ B7 Restore
                     └─ B8 Upgrade / Rollback

B3–B8 focused Gates ─→ B9 integration / recovery UX ─→ B10 full audit / freeze
```

B6/B7 共享 backup format，B8 依赖已验证保护点与 exact compatibility；这些是真实前置
依赖，不能伪并行。其余 lane 不得编辑彼此 owning package 或重复生成同一合同。

## 16. 测试矩阵

| 场景 | 预期结果 | 层级 |
| --- | --- | --- |
| Unsupported Hermes/version | capability false/partial；无 mutation | registry/adapter/API/Desktop |
| Unknown/malformed official output | UNKNOWN 或 stable error；不推断成功 | adapter fixtures |
| Health static/deep/live | 三者语义分离；live 网络调用必须显式 | adapter/application/Desktop |
| Oversized/malicious log | truncate/reject + redact；无 UI/script injection | adapter/API/Desktop/security |
| Log contains API key/token/account/path | prohibited values 不进入 response/event/audit | security/integration |
| Profile A/B Skill inventory | 不跨 Profile enable/config scope | adapter/application |
| Skill blocked scan | 不能通过 force 安装 | adapter/API/Desktop |
| Skill source traversal/symlink | fail closed；无外部写入 | adapter/security |
| Skill update partial failure | 真实 previous/current 状态可 reconcile | adapter/recovery |
| MCP 审核 Definition | 非 Preset 字段拒绝；凭据不回显；连接测试有 timeout/cancel；成功须权威回读 | API/security/adapter |
| MCP auth belongs to initiating session | 其他 session 无法读取/完成 | API/security |
| MCP test timeout/cancel | process/network 清理；稳定 terminal result | adapter/application |
| MCP CONFIGURED vs READY | 未测试不显示 READY | adapter/Desktop |
| Backup includes secret-bearing state | 只有 P7-D4 获批安全格式可落盘；无普通 plaintext artifact | security/manual |
| Backup temp crash | restart 后清理或标记 failed；不成为 AVAILABLE | application/integration |
| Backup tamper/wrong checksum | restore 前拒绝 | adapter/application |
| Malicious archive member/bomb | restore 前 fail closed；无越界写入 | security |
| Restore with running affected Instance | 按计划停止或拒绝；不竞态 | application/Windows |
| Restore partial failure | rollback/protection point 或明确 FAILED/UNKNOWN；不假成功 | recovery/manual |
| Upgrade mutates active generation | 测试必须证明没有原地写入 | install/integrity |
| Upgrade final-path executable invalid | Seal/activation blocked；active 不变 | adapter/install |
| Upgrade activation CAS conflict | fail closed；不覆盖别的 active pointer | install/recovery |
| Upgrade post-check failure | 前一 generation 保留；rollback 仅在数据兼容时执行 | integration/Windows |
| Daemon restart during each Operation | 不盲目 replay；按 truth terminal/recover | application/integration |
| Concurrent Restore/Upgrade/Skill/MCP/lifecycle | deterministic conflict，无 race/deadlock | application/race |
| Typed local actor/action audit | action/target/result 可审计且无 secret | persistence/security |
| Empty and prior-schema migration | deterministic，保留旧数据和约束 | persistence |

## 17. 验证策略

### 17.1 每 Batch 聚焦 Gate

- 受影响 Go package、adapter contract、HTTP、migration tests；
- 受影响 Desktop component tests、typecheck、lint；
- OpenAPI lint/generate/drift（有协议变化时）；
- Rust/Tauri checks（仅 native picker、secure storage、updater 或 packaging 变化时）；
- `git diff --check` 和 scope diff；
- 有真实外部副作用的 batch 运行独立、脱敏 Windows smoke。

不重复运行未变化的完整 CI。

聚焦 Gate 不是阶段审计。开发/测试智能体可以并行修复其 owning lane；不得为获得绿灯
弱化测试。正式独立审计只对 B10 的不可变完整候选执行。

### 17.2 Phase 7 候选 Gate

候选冻结后运行一次适用完整 Gate：

```text
pnpm install --frozen-lockfile
pnpm audit --audit-level low
pnpm api:lint
pnpm api:generate + generated drift check
pnpm typecheck
pnpm lint
pnpm test
pnpm build
go test ./...
go test -race ./...（支持的 CI）
go vet ./...
go build ./cmd/yorvad
govulncheck
cargo fmt --check
cargo test --locked
cargo clippy --locked --all-targets -- -D warnings
cargo check --locked
cargo audit
Windows sidecar/lifecycle/P7 management smoke
Tauri no-bundle release build
若 packaging/runtime target 改变，MSI build + fail-closed inspection
```

Manual Windows evidence 至少包括：

- 默认和命名 Profile 的 Skills 隔离与一条完整 install/update/remove 流；
- 一条获批 MCP install/auth/test/remove 流；
- backup create、checksum、restart 后 inventory；
- disposable test state 上的一次 Restore 成功和一次失败恢复；
- managed Hermes upgrade、新旧版本映射、gateway/Channel/model post-check 和可接受 rollback；
- health/log redaction 检查；
- 无 Secret、OAuth code/token、MCP header、backup password/key、账号、对话内容或敏感日志
  进入证据。

## 18. 独立审计要求

Phase 7 审计必须应用 `AUDIT_STANDARD.md` 全部十二个维度，并重点验证：

- Core/React/Tauri 无 Hermes command/path/config 逻辑；
- 每个 capability 来自 qualified surface，而不是只因命令名存在；
- MCP Definition/Binding 只能来自审核 Preset，不提供 stdio command、args、environment、headers、路径或任意 JSON surface；
- Skills source identity、scan 与 rollback 不可绕过；
- MCP credential authority、OAuth/session isolation 和 child cleanup；
- backup 是否包含 secrets、加密格式/密钥 authority、plaintext temp 和 restore archive 安全；
- Restore scope 与全部冲突/回滚；
- Upgrade 不修改 active sealed generation，且遵守 ADR-0006/0009；
- backup/user-data schema 与 generation rollback 的兼容性；
- logs/diagnostics 的 content、size、redaction 与 XSS 边界；
- P7 typed action/audit actor 没有被误报成企业 RBAC；
- migration 从正式前序 baseline 工作；
- exact-candidate CI、MSI（如适用）与真实 Windows evidence 对应同一候选。

任何 secret/plaintext backup 泄露、shell command surface、cross-Profile mutation、
unsafe archive extraction、active generation mutation、虚假 Restore/Upgrade success、
无法回滚的数据破坏或未认证管理路径都是阻塞项。

## 19. 验收标准

Phase 7 只有在以下全部成立时才能进入 Audit：

- 用户可在 Desktop 完成范围内的 Health/Logs、Skills、MCP、Backup/Restore 和 Upgrade，
  正常路径不需要终端；
- 每项能力只在 exact Runtime/version qualification 通过时暴露；
- 默认和命名 Instance/Profile 不发生跨 scope 状态或 credential mutation；
- Skills blocked scan 不能被 UI/API 绕过；
- MCP Definition 只从审核 Preset 建立，Binding 只接受 Preset 声明的凭据与 Tool Scope，不形成任意执行、路径或明文 secret 回读 surface；
- backup/restore 的 scope 与敏感数据范围真实可见，秘密不会以普通明文备份形式落盘；
- corrupt/tampered backup 在任何 mutation 前被拒绝；
- Restore 和 Upgrade 具有权威 postcondition，失败不报告 success；
- Upgrade 通过新 generation 完成，active generation 从未原地改变；
- daemon restart 后 Operation 不永久卡住、不盲目重放 destructive action；
- 日志/诊断有界、脱敏且不会持久化敏感内容；
- typed local action 和 audit actor 完整，但没有虚假 RBAC 声明；
- migration、完整 Gate、Windows smoke、精确候选 CI 和独立审计通过。

只有审计 `PASS`，或 Owner 明确接受有效的 `PASS WITH CONDITIONS`，且不存在 Critical
或 blocking High，才允许 merge/final-main/freeze/tag。

## 20. 强制停止条件

出现以下任一情况必须停止对应危险 surface 并交回主中枢/Owner。除共享信任边界被破坏
外，其他独立 lane 继续：

- P6.5 尚未冻结/放弃，或 P7 branch 不是正式 baseline 的后继；
- official Hermes surface 不能提供 non-interactive、scope-exact、bounded、可验证结果；
- 需要 import Hermes internal Python module 或依赖 undocumented state DB schema；
- Skills/MCP 只能通过 shell command、任意工作目录、明文 secret 回读或 force bypass 完成；
- MCP/backup/upgrade secret 无法确定唯一 authority；
- full backup 只能产生未加密的 credential/session/account archive；
- Restore 无法在 mutation 前证明 archive 安全，或无法定义失败/回滚语义；
- Upgrade 需要修改 active sealed generation 或执行 `--force-venv`；
- 新 Hermes user-data migration 使前一 generation rollback 不安全；
- 日志无法有界脱敏或会泄露对话、账号、token、path；
- material architecture/security change 缺少 accepted ADR；
- 无法获得必要的 exact-candidate、Windows 或 destructive-flow evidence。

不得通过降低 source、secret、archive、postcondition、migration 或 audit 标准获得 PASS。

## 21. Freeze 与后续阶段

通过 Gate 后：

1. 在 clean worktree 建立 immutable candidate；
2. 独立审计并保留所有历史 FAIL；
3. 修复后使用新鲜上下文重审；
4. 获 Owner 授权后 commit、push、merge；
5. final-main CI 通过；
6. 创建并验证 annotated tag：
   `phase-007-hermes-runtime-management-completeness-baseline`；
7. 更新 Spec/ROADMAP 为 `COMPLETE / FROZEN`；
8. 停止，不开始 Phase 8，直到 Phase 8 Spec 单独获批。

## 22. B0 完成记录

P7-D1–D8 与 B0–B10 已获 Owner 批准。Owner 选择“纳入并冻结 P6.5”；该工作已在
Phase 6 frozen tag 的独立干净后继中完成，原开发者工作树保持不动。结果如下：

- P6.5 frozen commit：`5f68e48f17e7e342e1781b37613b19d4bd1f060b`；
- annotated tag：`phase-0065-developer-led-demo-baseline`；
- P6.5 audit：PASS；
- exact-candidate CI/MSI、Windows smoke、安全检查和 final-main CI：PASS；
- P7 clean successor branch：`codex/phase7-hermes-runtime-management`；
- P7 branch 起点与 tag peeled commit 完全一致；
- P7 Spec 中英文镜像已同步；
- 未带入 P7 计划以外的原工作树修改或临时 artifact。

B0 Gate：**PASS — 2026-08-24**。P7 现进入 B1，只允许官方 surface 资格确认、证据和
必要 ADR；B1 Gate 通过前不新增产品 API、migration、Desktop 功能或 Runtime mutation。

## 23. P7R MVP 收口决策与 B0 核验

Owner 于 2026-08-27 批准在当前 P7 分支继续 P7R MVP 收口，并确认：

- 每个完成并通过相关 Gate 的 P7R Batch 自动提交；不自动 push、merge、tag 或 freeze；
- 优先完成真实写入、真实 Runtime read-back、真实运行和真实失败反馈，减少非必要的
  分批证据文件与普通 UI/API 接线 ADR；
- Secret 不泄露、破坏性操作目标与结果明确、禁止任意远程 Shell/Process/File API，
  以及仓库架构与 Runtime ownership 仍是硬边界；
- MCP MVP 只允许 YORVA 审核 Preset/Definition，不开放调用者提供的 stdio command、
  args、environment、HTTP headers、路径或任意 MCP JSON；完全自定义 MCP 延后。

P7R-B0 对当前实现的核验结果：

| Surface | 当前事实 | P7R 缺口 / 下一批动作 |
|---|---|---|
| Runtime Workspace | 已有 Overview、Instances、Skills、MCP、Maintenance、Operations；Diagnostics 通过实例入口进入 | 增加 Models；进一步分离 Runtime 资源与 Instance Binding |
| Skills | install/update/enable/disable/remove、ZIP/目录导入和 Runtime 多实例选择已接线 | 增加真实可验证 Catalog、Drift 与重新投影，并完成真机 read-back |
| MCP | reviewed test Preset、install/auth/test/configure/remove 与 Profile read-back 已存在 | 删除/关闭尚未提交的任意 STDIO/env/header surface；收敛为审核 Definition + Binding，并增加真实 Preset Smoke |
| Backup/Restore | authenticated Operation、加密容器、系统应用数据目录、Restore 与错误分类已存在 | 在全部 Hermes 进程停止后完成真机 create/read-back/restore/delete Smoke |
| Upgrade | Plan、HTTP/Application Operation 合同和 Desktop 入口已存在 | daemon 尚未注入真实 Upgrader/Rollbacker；需要固定候选 Generation、保护点、切换、post-check 与失败回滚 |
| Models | 仍以单 Instance 原生配置为主 | 新增 Provider Connection、Model Profile、Runtime Default、Instance Override 与批量 Copy-on-Apply |

P7R-B0 Gate：只有上述事实与公开合同一致、禁止的自定义 MCP surface 未进入提交，且
focused checks 通过后才标记 PASS。

P7R-B1 已按以下边界完成 Runtime/Instance 页面收口：

- Runtime Workspace 导航为 Overview、Instances、Models、Skills、MCP、Maintenance 与
  Operations。Diagnostics 仍按产品要求隐藏在 tab 外，只从精确 Instance 行进入。
- Models 已进入 Runtime Workspace，提供明确的 Instance 选择器，并复用现有 Hermes
  Profile 权威模型配置链路。共享 Provider Connection、Model Profile 与 Runtime Default
  仍属于 B2/B3，不使用假数据冒充。
- Instance 管理只保留精确实例的生命周期、模型与 Channel 入口、Skill/MCP Binding、健康
  和日志；Runtime Upgrade、Backup 与 Restore 继续只属于 Runtime。

P7R-B1 Gate 要求 focused Desktop tests、TypeScript typecheck、lint 与非 MSI Desktop build
通过后自动提交。

P7R-B2/B3 已实现真实的 Runtime 共享模型资源：

- Runtime Provider Connection 将一份只写凭据存入 OS-backed SecretStore；
- Model Profile 复用审核 Provider Preset 和 allowlisted model IDs；
- Runtime Default 与精确 Instance 的 `INHERIT`/`OVERRIDE` Binding 分离；
- 多实例应用使用持久化 `model.profile.apply` Operation，每个实例分别记录结果，并且只有
  Hermes 权威回读成功后才记录成功；
- Hermes 原有且未被 YORVA 应用的模型配置保持 External，不会被静默接管。

B2/B3 Gate 要求 migration、application、HTTP/OpenAPI、Desktop client/component、Secret
不泄露、typecheck/lint 与非 MSI build 检查通过后自动提交。

P7R-B4 在不改变 Hermes 原生所有权的前提下收口 Skills MVP：

- 审核 Catalog 新增一个真实可用、digest 校验、纯文本的文档审查 Skill；
- ZIP 与目录导入继续经过有界校验并进入不可变 managed store；
- YORVA-owned 投影缺失时明确显示 Drift，并提供“重新投影”入口；
- 内容被修改、所有权冲突或外部目标继续 fail-closed、保持只读；
- 重新投影复用持久化 `skill.enable` Operation，并以权威回读确认结果。

B4 Gate 要求 managed store/application tests、Desktop interaction tests、typecheck/lint
与非 MSI build 通过后自动提交。

P7R-B5 在不保留未来自定义执行面的前提下收口受限 MCP MVP：

- Runtime Definition 只投影编译期审核 Preset registry 的安全元数据；
- Instance Binding Operation 对精确 Hermes Profile 执行安装、Preset 声明的认证、Tool
  Scope 配置、测试、移除与 reconcile；
- Application 与 Runtime 合同已删除休眠的任意 Definition、stdio command/argv、
  environment、header、endpoint、path 与 named-secret mutation 字段；
- YORVA-owned loopback 测试 Preset 覆盖创建、Profile 写入、握手、权威回读、Tool Scope
  更新、重新测试、第二 Profile 绑定，以及删除后的缺失回读；
- `MCPRead`、`MCPMutate`、`MCPTest` 由实际注册的 reader/manager 推导。

B5 Gate 要求 application/Runtime/HTTP 聚焦测试、带 tag 的生产 MCP 生命周期资格测试、
Go vet、API drift、Desktop tests、typecheck/lint 与非 MSI build 通过后自动提交。

P7R-B6 收口当前实际适用的 Runtime 维护 MVP：

- Windows 上的加密 Runtime Backup create/read/restore/delete 保持真实接线，默认保存到
  `<YORVA 应用数据>/backups`；Desktop 与 HTTP 请求均不携带目标路径；
- 设备密钥继续由 OS-backed SecretStore 保存，只有加密发布、checksum 与权威校验全部
  成功后才写入备份索引；
- 本机产品日志已记录 2026-08-27 的一次真实备份创建成功，对应加密文件存在于默认应用
  数据备份目录；
- 当受支持的外部/开发 Hermes 版本已与内置候选一致时，只读计划现在准确显示
  `UP_TO_DATE`，不再错误展示与本次无版本变更无关的受管升级证据缺口；
- 版本一致绝不授予 Upgrade/Rollback mutation。当前没有更新的内置候选，因此没有可执行
  的版本迁移；较旧的受管版本仍必须满足保护点、兼容性和 post-check 完整证据。

B6 Gate 要求 Backup/Restore 与 Upgrade planner/application/HTTP tests、完整 Go tests/vet、
Desktop 维护测试、typecheck/lint 与非 MSI build 通过后自动提交。针对 Owner 真实 Hermes
数据的破坏性 Restore 仍留给 B7 明确手动 smoke，不在自动测试中执行。

P7R-B7 的自动化集成与交接检查已完成：

- 当前分支的完整 Go test/vet、API lint/generate drift、Desktop tests、TypeScript
  typecheck、lint 与非 MSI Vite build 均已通过；
- 带 tag 的 MCP 资格测试覆盖审核 Preset 从 Hermes Profile 写入、连接测试、权威回读、
  Binding 更新到删除的完整生命周期；
- 已从 P7 linked worktree 启动当前非 MSI Desktop 及其内置 `yorvad`，供 Owner 手动查看；
- 聚焦累计 diff 审查确认未开放调用者控制的 MCP command、args、environment、headers、
  executable/path 或任意 JSON mutation surface；
- 未对 Owner 真实 Hermes 数据执行破坏性 Restore，也未 push、merge、tag 或 freeze。

以上只表示 B7 自动化交接完成。其后的 2026-08-27 冻结审计结论由第 24 节记录，并取代
这里此前的 `IN_PROGRESS` 交接状态。

## 24. MVP-First 阶段规划归并与冻结审计结果

Owner 提供的《YORVA MVP-First 阶段总规划（P7R–P13）》中与当前阶段有关的内容正式归并
到本 Phase 7 Spec，以本节和第 23 节作为仓库内权威记录，不依赖下载目录中的外部副本。

归并后的 P7R 产品收口要求为：

- Runtime Workspace 统一承载 Models、Skills、MCP、Maintenance、Diagnostics 与 Operations；
- `ModelProviderConnection → ModelProfile → Runtime Default / Instance Binding` 是共享模型主链；
- `ManagedSkillPackage → SkillBinding → Instance` 保持 YORVA managed 与 Hermes external
  来源分离；
- `MCPDefinition → MCPBinding → Instance` 只接受审核 Preset，不开放任意 command、args、
  environment、header、path 或 JSON；
- Runtime Backup、Restore、Upgrade 和 Runtime Diagnostics 不下沉到单 Instance 页面；
- MVP 必须以真实写入、Runtime 权威回读、真实失败反馈和无需终端的完整用户闭环为准；
- P7R 最终集成必须覆盖共享模型、Skills、MCP、加密 Backup/Restore，以及固定候选的真实
  Upgrade/Rollback；“页面存在”或“计划可读”不能替代可执行 mutation。

2026-08-27 冻结审计记录：
`docs/phases/audits/AUDIT-007-hermes-runtime-management-completeness.md`。

Gate Decision：**FAIL**。阻断项为：

1. production daemon 未注入 `Upgrade` / `Rollback` adapter binding，真实升级与回滚仍为
   capability-false；
2. 尚无 disposable Windows 状态上的 Restore 成功及失败恢复 smoke；
3. 当前精确候选尚无 CI/race 证据，本机因缺少 C compiler 无法运行 Go race。

因此本次不 push、不 merge、不 tag、不标记 FROZEN。P8 的候选输入保留为产品支持矩阵、
数据库 Migration、崩溃/重启恢复、安装器生命周期、YORVA 更新、脱敏诊断包和基础稳定性
验证，但在 Phase 7 修复并通过重审前不得形成 READY 的 Phase 8 执行 Spec，也不得开始
P8 实现。

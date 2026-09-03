# YORVA Phase 8 — 本地产品可靠化 MVP

> 状态：**APPROVED / IN PROGRESS — Owner 已授权 B0–B6 实现**
> 阶段标识：P8
> 阶段性质：本地产品可靠化与首个正式 Windows MVP
> 必需基线：phase-007-hermes-runtime-management-completeness-baseline + P7 稳定性修订 `9834a8cb1df9e70502936153943f501ed37cb8fc`
> 计划分支：phase/p8-local-product-hardening
> 产品输入：《YORVA MVP-First 阶段总规划（P7R–P13）》P8
> 执行方式：单主 Agent；每个完成并通过聚焦 Gate 的 Batch 自动 Commit

## 0. 阶段授权状态

本文件把 Owner 提供的 MVP-First P8 内容整理为适合当前仓库的可执行计划。
Owner 已于 2026-09-03 明确批准本 Spec、B0–B6 顺序以及逐 Batch 自动 Commit。

P8 以 2026-08-31 已合并到 `main` 的 P7 稳定性修订 `9834a8c` 为代码起点，
但不移动或重写既有 Phase 7 冻结标签。该修订只补充 Hermes 检测时的有界自动启动、
已移除 Instance 记录分组与安全清理，不扩大 P7 已冻结的产品范围。

Phase 8 不重新打开 Phase 7 已延期的 Hermes Managed Upgrade/Rollback。本文中的
“YORVA 更新”只表示 YORVA Desktop、yorvad、数据库 Schema 与随包资源从一个受支持
YORVA 版本升级到另一个受支持版本。

## 1. 阶段目标

将已冻结的 Phase 7 单机功能闭环变成普通 Windows 用户能够安装、重启、更新、恢复、
诊断和正确卸载的本地正式产品，不要求开发工具，不要求用户打开终端，也不依赖未来
YORVA Control。

最低产品闭环：

~~~text
Fresh Install
→ First Launch
→ Configure
→ Run
→ Windows Reboot
→ Recover and Reconcile
→ Upgrade YORVA
→ Migrate
→ Export Sanitized Diagnostics
→ Uninstall with an explicit data policy
~~~

Phase 8 的完成标准是真实安装包、真实持久化数据和真实失败反馈能够跑通以上流程，
不是仅存在 UI、接口或测试桩。

## 2. 冻结基线与当前事实

Phase 8 从 Phase 7 正式标签及其已接纳的稳定性修订开始，不修改或移动 Phase 7 标签。

P7 交接链：

~~~text
phase-007-hermes-runtime-management-completeness-baseline (12b16bc)
→ 64761ac  Hermes 检测时有界自动启动与超时稳定化
→ be7b812  当前 Instance 与已移除记录分组
→ 9834a8c  已移除记录清理路由闭环
→ main
~~~

修订候选 `9834a8c` 已通过 GitHub CI run `33365096577` 的 Web/API、Go race、
Windows native、Rust 与非 MSI build Gate。合并后的 final-main CI run `33366271630`
在首次 Windows runner 握手超时后保留失败记录，并由 attempt 2 完整通过；Windows MSI
run `33366271629` 同时通过。P8 实现分支包含该提交，阶段已获得 Owner 授权。

当前可复用基础：

- Tauri 2 Desktop、React/TypeScript、Go yorvad 与 SQLite 已形成单机产品链；
- Windows MSI 构建、固定资源校验与 MSI 内容检查已有脚本和 GitHub workflow；
- 当前数据库通过内嵌顺序 Migration 自动升级，冻结 Schema 为 016；
- Desktop 已拥有 daemon 启动、握手、托盘、隐藏启动和单实例基础；
- Runtime/Instance、Operation、Skills、MCP、Backup/Restore 已有恢复或 reconcile 基础；
- Hermes 已安装但未运行时，检测链具有一次有界启动与重新检测能力；
- 外部删除的 Hermes Profile 会进入独立的“已移除记录”，并且只有在权威回读仍确认
  不存在时才能清理 YORVA 管理记录；
- 安装与运行日志已有脱敏约束和稳定错误码；
- CI 已包含 Web/API、Go race、Windows native、Rust 与非 MSI build。

当前缺口：

- Tauri 产品标识仍带 development 语义，正式产品标识与旧数据迁移策略未冻结；
- MSI 仍是 Demo/候选打包能力，不等于完成 Fresh/Upgrade/Repair/Uninstall/Reinstall；
- Migration 没有完整的升级前保护副本、跨版本夹具链和产品化失败恢复；
- 没有 YORVA 自身更新闭环；
- 没有用户可一键导出的脱敏诊断包；
- 没有 3 Instance、4–8 小时 Soak 和资源增长证据；
- 没有 P8 发布签名、支持矩阵、恢复指南和更新来源策略。

### 2.1 P8 进入条件

- `main` 包含 P7 稳定性修订 `9834a8c`，且 final-main CI 成功；
- P8 中英文 Spec 与 `ROADMAP.md` 使用同一代码基线；
- Phase 7 冻结标签保持不变，修订内容不被错误表述为新 P7 capability；
- Owner 已于 2026-09-03 批准 P8-D1–D9、B0–B6 顺序及自动 Commit 方式；
- 进入条件已经满足，P8-B0 可以开始。

## 3. P8 产品决策

以下决策已由 Owner 于 2026-09-03 批准。后续如需修改，先更新本节和相应 Batch，
再继续受影响的实现。

| ID | 决策 |
| --- | --- |
| P8-D1 | Windows 10/11 x64 是首个正式支持目标；Windows 是阻断 Gate。macOS/Linux 只做可行性记录，不构成 P8 阻断，也不得削弱 Windows。 |
| P8-D2 | P8 必须冻结稳定的产品名、identifier、数据目录和版本规则。若从 development identifier 迁移，必须显式迁移旧 YORVA 数据，不得静默形成两个产品数据孤岛。 |
| P8-D3 | 卸载默认保留 YORVA 用户数据、YORVA Backup 和 Hermes Runtime/Profile 数据，并在 UI/卸载说明中明确告知。P8 不提供静默“删除所有数据”。 |
| P8-D4 | 数据库只支持从本阶段列入支持矩阵的 Schema 升级。最低必须覆盖空库和 Phase 7 冻结 Schema 016；不承诺任意历史开发数据库。 |
| P8-D5 | YORVA 更新使用完整安装包，不建设增量补丁系统。包、更新元数据和来源必须可验证；退出、迁移和 reconcile 都必须有明确结果。 |
| P8-D6 | MVP 默认不收集遥测。若以后增加 opt-in telemetry，必须单独修改安全合同并取得 Owner 批准。 |
| P8-D7 | 诊断包由 YORVA 生成固定结构，自动脱敏；用户只能通过本地 Save As 能力选择导出位置，React 和 HTTP 不获得任意文件系统 API。 |
| P8-D8 | P8 发布候选必须通过签名/来源校验 Gate。缺少实际可用的发布签名材料时，可以完成内部候选，但不能声明公开发布就绪。 |
| P8-D9 | 每个 Batch 在 focused tests 与直接相关 build 通过后自动 Commit；只在阶段里程碑运行完整 Gate。Push、merge、tag 和 freeze 仍由 Owner 明确授权。 |

## 4. 范围

### 4.1 本阶段必须交付

- Windows 产品支持矩阵；
- 稳定的应用 identifier、版本和数据目录策略；
- 从支持的旧 Schema 到当前 Schema 的可恢复 Migration；
- Desktop、daemon、Windows reboot 后的恢复与权威 reconcile；
- MSI Fresh Install、Upgrade、Repair、Uninstall、Reinstall；
- 完整包形式的 YORVA 更新 MVP；
- 发布来源、哈希、签名和版本一致性验证；
- 一键导出的固定格式脱敏诊断包；
- 基础安全复核与威胁模型更新；
- 3 Instance 和 4–8 小时稳定性验证；
- 用户支持矩阵、数据保留和恢复指南；
- 精确候选 CI、Windows 安装/更新 Smoke、审计与冻结。

### 4.2 明确不在 P8

- Hermes Managed Upgrade/Rollback；
- 第二 Runtime 或 Runtime Marketplace；
- YORVA Node、Control、远程管理、Fleet；
- Linux/macOS 正式支持承诺；
- HA、企业 RBAC、审批、多租户；
- 复杂增量更新、后台更新服务或更新 CDN 平台；
- 复杂遥测平台或默认数据上报；
- 任意 shell、process、file、registry、environment 管理接口；
- 任意 MCP command、args、env、header、path 或 JSON；
- 72 小时 Soak；该项进入后续 Refinement。

## 5. 架构与所有权

保持现有依赖方向：

~~~text
React Desktop
    ↓ typed local contract
Go Application / Operations
    ↓
Domain
    ↑
SQLite / Runtime adapters

Tauri
    ├─ Desktop/daemon lifecycle
    ├─ installer and updater handoff
    ├─ code-signing / OS integration
    └─ capability-scoped Save As
~~~

所有权规则：

- Hermes 继续拥有 Hermes Runtime/Profile 状态；
- YORVA SQLite 继续拥有 YORVA 管理状态；
- Tauri 只拥有 native shell、安装/更新交接和窄 OS 能力；
- React 不直接读取数据库、执行安装器或访问 Hermes 文件；
- Migration 的成功条件来自新数据库的 Schema/数据校验；
- reboot/restart 的成功条件来自 Runtime/Instance 权威 read-back，不来自旧缓存；
- 更新成功必须同时满足安装版本、Migration 和 reconcile 的 postcondition；
- 诊断包只包含固定投影，不导出数据库文件或秘密原文。

## 6. Batch 计划

### P8-B0 — 产品支持矩阵与发布合同

目标：先冻结“支持什么、数据放在哪里、怎样升级、怎样卸载”，避免后续 Batch 对
产品身份和数据所有权作出互相冲突的假设。

实现内容：

- Windows 10/11 x64 支持矩阵；
- WebView2、CPU/内存/磁盘和权限前提；
- YORVA、yorvad、数据库 Schema 和 Hermes 兼容窗口；
- 稳定 productName、identifier、版本同步规则；
- development identifier 到稳定 identifier 的一次性数据发现/迁移策略；
- YORVA app data、日志、下载、更新 staging、诊断和 Backup 目录表；
- Fresh Install、Upgrade、Repair、Uninstall、Reinstall 数据策略；
- 更新来源、签名材料、包哈希和发布元数据合同；
- 明确 no telemetry；
- 用户支持与恢复文档框架。

B0 冻结结果记录在 `docs/PRODUCT_SUPPORT.zh-CN.md`，英文执行镜像为
`docs/PRODUCT_SUPPORT.md`。正式 identifier 的激活依赖 B1 先完成受保护的一次性
旧数据迁移；在此之前继续运行旧 identifier，避免现有 P7 数据提前不可见。

直接相关文件预计包括 Tauri config/Cargo/package metadata、DEVELOPMENT、SECURITY、
DATA_MODEL、发布脚本和本 Phase Spec。B0 不实现更新下载器。

B0 Gate：

- 所有版本与 identifier 在 package、Tauri、二进制和发布文件中只有一个明确规则；
- 目录与数据所有权表完整；
- Phase 7 数据迁移目标明确；
- 未获得的签名材料被列为发布阻断，而不是伪造 PASS；
- 文档审查通过后 Commit。

### P8-B1 — 数据库 Migration 与升级保护

目标：让支持的旧 YORVA 数据在升级失败时仍可恢复，不要求用户删库重装。

实现内容：

- 在存在待执行 Migration 时创建一致的升级前数据库保护副本；
- 记录源 Schema、目标 Schema、备份校验和、Migration 状态和安全错误码；
- 空库、Schema 016 和本阶段新增 Schema 的固定测试夹具；
- 每个 Migration 继续在短事务内执行；
- Migration 链重复启动安全；
- 失败后 daemon 不进入 READY，不继续 Runtime mutation；
- 自动恢复到明确证明一致的旧数据库，或进入明确的 RECOVERY_REQUIRED；
- 成功后验证 Schema ledger、外键、关键行数/唯一性和 repository 读取；
- 对保护副本设置有界保留，不删除未知文件；
- 诊断只显示版本、状态和错误码，不显示路径、SQL 或数据内容。

B1 Gate：

- empty → latest：PASS；
- P7 Schema 016 fixture → latest：PASS；
- 重复执行：PASS；
- 中途失败 → 原数据可用或明确 recovery-required：PASS；
- Migration 前保护副本校验：PASS；
- Go migration/repository tests、vet 与安全 review 通过后 Commit。

### P8-B2 — 崩溃、重启与 Windows reboot 恢复

目标：进程或机器重启后恢复管理权，不把 SQLite 缓存伪装成 live Runtime 状态。

启动恢复顺序：

~~~text
acquire single daemon ownership
→ inspect update/migration state
→ recover filesystem journals
→ recover or terminalize stale Operations
→ discover Runtime
→ reconcile Runtime resources
→ reconcile Instance inventory/lifecycle/channels
→ expose READY or typed recovery state
~~~

实现内容：

- Desktop crash 后 daemon 所有权和重新连接；
- daemon crash 后 Desktop 有界重启与明确失败；
- Windows reboot 后 autostart/single-instance/daemon 启动连续性；
- Hermes 已安装但未运行时执行一次有界启动并重新检测；启动失败必须进入稳定、可重试
  的恢复状态，不能无限拉起或继续显示“检测超时”；
- stale Operation 按各自资源权威状态恢复或失败；
- install、restore、model、Skill、MCP、Backup Operation 恢复矩阵；
- Runtime/Instance inventory reconcile；
- Hermes Profile 在外部删除、恢复或重建后，当前 Instance 与已移除记录必须按权威
  read-back 重新分类；只有持续确认缺失的非受保护记录可以清理；
- 不重复启动 daemon，不留下孤儿进程；
- 恢复过程的进度和最终错误在 Desktop 可理解地展示；
- 恢复失败不阻断诊断导出入口。

B2 Gate：

- Desktop kill/reopen：PASS；
- daemon kill/restart：PASS；
- 模拟 stale Operations：PASS；
- disposable Windows reboot/login smoke：PASS；
- Hermes 预先停止 → YORVA 有界启动 → 支持版本检测成功：PASS；
- Hermes 启动失败 → 明确恢复状态且无重复拉起：PASS；
- Profile 外部删除 → 进入已移除记录 → 权威确认后清理：PASS；
- 已移除 Profile 重新出现时拒绝清理、刷新后恢复当前 Instance：PASS；
- Runtime/Instance 权威 read-back：PASS；
- 无虚假 SUCCEEDED、无双 daemon、无未回收测试进程；
- Go/Rust/Desktop tests 与 Windows smoke 通过后 Commit。

### P8-B3 — Windows 安装器完整生命周期

目标：把现有 Demo MSI 打包基础升级为可重复验证的产品安装生命周期。

实现内容：

- Fresh Install；
- 首次启动与必要 WebView2 处理；
- 从支持的旧 YORVA 安装 Upgrade Install；
- Repair；
- Uninstall；
- 保留数据后的 Reinstall；
- 安装器发现正在运行的 Desktop/daemon 并给出明确关闭流程；
- 用户可见的数据保留说明；
- Hermes Runtime/Profile 与 YORVA Backup 不被 MSI 静默删除；
- 安装路径、应用标识、快捷方式和卸载项一致；
- MSI 内容继续经过固定资源大小、哈希、许可证和额外可执行文件检查；
- 安装日志不包含 Secret；
- 构建产物记录版本、commit、SHA-256 和签名状态。

验证矩阵至少使用 disposable Windows 环境：

| 场景 | 必须验证 |
| --- | --- |
| Fresh | 安装、启动、daemon 握手、版本、数据目录 |
| Upgrade | 旧配置保留、Migration、reconcile、功能可用 |
| Repair | 程序文件恢复，用户数据不重置 |
| Uninstall | 程序移除，保留策略与提示一致 |
| Reinstall | 保留数据重新识别，首次启动不重复破坏状态 |

B3 Gate：

- 上表全部真实 MSI smoke 通过；
- MSI inspector 的负例继续通过；
- 无静默提权、无静默 Runtime 数据删除；
- 精确 MSI 哈希和检查摘要保存后 Commit。

### P8-B4 — YORVA 更新 MVP

目标：使用完整、可验证的安装包完成一次真实 YORVA 版本更新。

最小流程：

~~~text
check fixed update metadata
→ show version and release notes
→ download full package
→ verify provenance, signature and checksum
→ prepare/drain Desktop and daemon
→ launch installer handoff
→ install
→ relaunch
→ migrate
→ authoritative reconcile
→ report installed version and final result
~~~

实现内容：

- 固定、YORVA-owned 的更新元数据来源；
- 严格版本比较和兼容窗口；
- 下载大小上限、临时目录和取消；
- 包签名/哈希验证；
- 更新前阻止新的冲突 mutation；
- daemon 有界停止和数据库关闭；
- native installer handoff；
- 更新 intent/result 的窄持久化状态；
- 启动后 Migration 与 reconcile；
- 成功只在已安装版本和 postcondition 全部匹配后显示；
- 下载、验证、安装或恢复失败均有稳定错误码和恢复入口；
- 不支持用户输入任意 URL、路径、command、args 或 header；
- 不建设增量补丁和后台常驻更新服务。

B4 Gate：

- older supported YORVA → candidate 的真实更新：PASS；
- checksum/signature 不符：安装前拒绝；
- 下载中断：旧版本仍可启动；
- installer 失败：状态明确且数据可恢复；
- Migration/reconcile 失败：不显示更新成功；
- 更新完成后的版本、Runtime 和 Instance read-back：PASS；
- native/Go/Desktop tests 和 Windows update smoke 通过后 Commit。

### P8-B5 — 一键脱敏诊断包

目标：普通用户在不打开终端的情况下生成可以交给支持人员的有限诊断包。

固定内容：

~~~text
manifest.json
version.json
node-summary.json
runtime-summary.json
instance-summary.json
operations.json
schema.json
logs/*.ndjson
redaction-report.json
~~~

实现内容：

- 一键“导出诊断”入口和独立配置页；
- 固定 schema、文件名、最大条数、日志时间窗口和总大小上限；
- API/Node 生成结构化、脱敏内容；
- Tauri 提供 capability-scoped Save As 和原子发布；
- Provider key、Token、Channel secret、MCP credential、二维码/配对码、Cookie、
  Authorization header、环境值、原始数据库和任意用户文件禁止导出；
- 路径只保留必要的类别或哈希化标识；
- 导出前后运行 secret-canary 测试；
- 失败不留下半成品，成功显示文件名、大小和生成时间；
- 诊断导出不能成为通用文件读取 API。

B5 Gate：

- 固定结构与 schema tests：PASS；
- 每类 Secret canary 均未出现在文件名、正文和压缩容器；
- 大日志被有界截断；
- Save As 取消不产生 Operation 成功；
- 失败清理半成品；
- OpenAPI、Go、Tauri、Desktop tests 和真实导出 smoke 通过后 Commit。

### P8-B6 — 基础稳定性、发布与阶段 Gate

目标：用接近实际用户的工作负载验证 P8 候选，而不是用 72 小时企业级测试拖延 MVP。

Soak 场景：

- 3 个 Hermes Instance/Profile；
- 并行 start/stop/restart；
- 模型 read/write 和权威回读；
- Skills install/update/enable/disable/remove；
- reviewed-Preset MCP bind/test/unbind；
- Channel 状态查询；
- Runtime Backup create/verify/delete；
- Desktop close/reopen 与 daemon reconnect；
- 定期诊断导出；
- 至少一次受控 Windows reboot/recovery；
- 4 小时最低 Gate，8 小时作为最终发布候选目标；
- 72 小时 Soak 延后到 Refinement。

观测项目：

- Desktop/daemon crash、deadlock、orphan process；
- goroutine、handle、memory、CPU 的持续单向增长；
- 日志、Operation、staging、update 和 temp 文件的无界增长；
- stale Operation 和 inventory drift；
- Secret/credential 泄露；
- 失败后核心操作能否继续。

发布收口：

- security review 与 threat-model refresh；
- dependency audit；
- 精确候选 CI 和 Go race；
- signed MSI 构建与检查；
- Fresh/Upgrade/Repair/Uninstall/Reinstall smoke；
- YORVA update smoke；
- Migration fixtures；
- diagnostics secret scan；
- 4–8 小时 Soak 摘要；
- 独立聚焦审查与 Owner Gate；
- merge、final-main CI、annotated Phase 8 tag。

B6 Gate：

- 所有 P8 功能、read-back、失败和 smoke Gate 通过；
- 无未解决 Critical/High；
- 发布签名真实可验证；
- 支持矩阵和恢复指南完成；
- 审计 PASS 或 Owner 接受的 PASS WITH CONDITIONS；
- Owner 明确授权后才 merge/tag/freeze；
- Commit 后停止，不开始 P9。

## 7. Batch 依赖

~~~text
P8-B0 Support Contract
    ↓
P8-B1 Migration
    ↓
P8-B2 Crash / Reboot Recovery
    ↓
P8-B3 Installer Lifecycle
    ↓
P8-B4 YORVA Update
    ↓
P8-B5 Diagnostics Export
    ↓
P8-B6 Stability / Release Gate
~~~

B5 的 schema/redaction 单元工作可在 B3/B4 期间准备，但单主 Agent 默认按上图顺序
完成，避免共享 Tauri、daemon、OpenAPI 和文档产生交叉修改。

## 8. Desktop 信息架构

继续使用设置页的简洁、扁平、分组行语言。不得把 P8 内容全部堆进一个超长页面。

建议入口：

~~~text
设置
├─ 关于 YORVA
│  ├─ 当前版本
│  ├─ 更新状态
│  └─ 检查更新
├─ 数据与恢复
│  ├─ 数据目录说明
│  ├─ Schema / Migration 状态
│  └─ 卸载保留说明
└─ 诊断与支持
   ├─ 运行摘要
   ├─ 最近恢复状态
   └─ 导出诊断
~~~

复杂更新详情、Migration recovery 和诊断导出使用独立页面。主设置页只显示摘要和入口。

## 9. API、Operation 与错误语义

P8 新增合同遵守现有 PROTOCOL：

- typed request/response；
- authenticated loopback only；
- long-running work 使用 Operation 或 native updater state，不持有长 HTTP 请求；
- Desktop 不通过匹配中文/英文错误文本决定行为；
- 普通读 API 不返回 Secret、原始日志、SQL、任意路径或安装器输出；
- 更新与诊断不增加通用 shell/file/process API。

至少需要稳定区分：

~~~text
MIGRATION_BACKUP_FAILED
MIGRATION_FAILED
MIGRATION_RECOVERY_REQUIRED
RECOVERY_RECONCILE_FAILED
UPDATE_METADATA_INVALID
UPDATE_DOWNLOAD_FAILED
UPDATE_INTEGRITY_FAILED
UPDATE_INSTALL_FAILED
UPDATE_POSTCHECK_FAILED
DIAGNOSTICS_EXPORT_FAILED
DIAGNOSTICS_REDACTION_FAILED
~~~

最终名称在对应 OpenAPI Batch 中冻结，UI 只依赖稳定 code。

## 10. 测试矩阵

| 领域 | 最低自动化 | 必须 Smoke |
| --- | --- | --- |
| Support contract | version/identifier/config consistency | 目标 Windows 环境核对 |
| Migration | empty、016、latest、重复、故障注入、恢复 | 旧安装数据升级 |
| Recovery | stale Operation、journal、inventory reconcile | Desktop/daemon kill、Windows reboot |
| Installer | package inspection、资源/hash/license negatives | Fresh/Upgrade/Repair/Uninstall/Reinstall |
| YORVA update | metadata/hash/signature/version/failure tests | 旧正式包 → 候选 |
| Diagnostics | schema、bounds、secret canaries、partial cleanup | 一键导出并人工检查 |
| Stability | workload driver、resource samples、log bounds | 3 Instance、4–8 小时 |

完整候选 Gate：

~~~text
pnpm frozen install / audit
OpenAPI lint / generation drift
Desktop typecheck / lint / tests / build
Go tests / race / vet / govulncheck / build
Rust fmt / tests / clippy / check / audit
Migration fixture suite
MSI build / inspect / lifecycle smoke
YORVA update smoke
Diagnostics secret scan
Windows reboot/recovery smoke
4–8 hour Soak
focused independent audit
~~~

## 11. 安全要求

P8 继续严格保持：

- Secret 不进入 SQLite 明文、日志、诊断包、URL、argv、环境变量或普通 API；
- 更新包在执行前验证来源、签名、哈希和版本；
- installer/updater 不接受任意 URL、path、command、args、env、header；
- Migration/Repair/Uninstall 不静默删除 Runtime 或用户数据；
- 诊断包不包含数据库原文件；
- 下载和解压有大小、成员、路径和临时目录边界；
- 不默认管理员运行，不用关闭 CSP 解决更新问题；
- 失败结果不得显示成功；
- 远程管理面保持不存在。

## 12. 强制停止条件

出现以下任一情况，停止对应 Batch，不以技术债绕过：

- Migration 可能丢失或不可恢复地改坏支持版本数据；
- stable identifier 会让旧数据不可发现且没有明确迁移；
- MSI uninstall/repair 可能静默删除 Hermes 或 YORVA 用户数据；
- 更新包无法验证来源/签名/哈希；
- updater 需要任意命令、URL 或文件系统接口；
- reboot 后 SQLite 缓存被当作 Runtime live state；
- 诊断包泄露任何 Secret 或原始数据库；
- 安装、更新、恢复失败却显示 success；
- 真实 MSI/update/reboot smoke 无法执行；
- 4 小时最低 Soak 无法完成或出现未解释 crash/deadlock/无界增长。

## 13. 阶段退出标准

Phase 8 只有在以下全部成立后才能进入审计：

- 支持矩阵和数据保留策略已冻结；
- 普通 Windows 用户可以 Fresh Install 并完成首次启动；
- Phase 7 数据可升级且失败可恢复；
- Desktop crash、daemon crash、Windows reboot 后恢复管理；
- Upgrade/Repair/Uninstall/Reinstall 符合数据策略；
- 一次真实 YORVA 完整包更新成功并权威回读；
- 一键诊断包通过 Secret scan；
- 3 Instance 与 4–8 小时 Soak 通过；
- 签名候选、CI、Windows Smoke 和文档完整；
- 独立审计满足 AUDIT_STANDARD；
- Owner 给出 Gate Decision。

审计通过后：

1. 合并到 main；
2. 运行 final-main CI 与适用 Windows package/update Gate；
3. 创建 annotated tag：phase-008-local-product-hardening-baseline；
4. 更新 Spec/ROADMAP 为 COMPLETE / FROZEN；
5. 停止，不开始 P9，直到 P9 Spec 单独获批。

## 14. 执行记录

| Batch | 状态 | Commit | Gate |
| --- | --- | --- | --- |
| P8-B0 | COMPLETE | B0 Batch commit | 中英文产品支持合同、Owner 决策、目录/保留/签名门禁一致性检查通过 |
| P8-B1 | NOT STARTED | — | — |
| P8-B2 | NOT STARTED | — | — |
| P8-B3 | NOT STARTED | — | — |
| P8-B4 | NOT STARTED | — | — |
| P8-B5 | NOT STARTED | — | — |
| P8-B6 | NOT STARTED | — | — |

## 15. Owner 审批记录

Owner 于 2026-09-03 确认：

1. Windows 10/11 x64 是唯一阻断发布目标；
2. MVP 默认无遥测；
3. 卸载默认保留全部 YORVA/Hermes 用户数据；
4. 当前没有可用于公开候选的 Windows 代码签名材料，因此 P8 先以内部候选为上限，
   不声明公开发布就绪；
5. 接受 4 小时最低、8 小时候选目标的 Soak；
6. 授权 P8 按 B0–B6 顺序实施并在每个 Batch 通过后自动 Commit。

Push、merge、tag、freeze 仍须 Owner 另行明确授权。

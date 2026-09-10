# YORVA Phase 9 — OpenClaw 第二 Runtime

> Status: FROZEN — G1 / G2 / G3 PASS；最终产品候选 `b958801`，基线 tag `phase-009-openclaw-baseline`
> Owner: YoLin02
> Date: 2026-09-08
> Baseline: `phase-008-local-product-hardening-baseline` → `2a7b842e668011a97804eb40642e3ff9dcab2f04`
> Working start: `e95ed31d298c2548556e1295aacf7f84002b74ee`，P8 冻结记录的文档后继提交，产品代码相同
> Roadmap: [Phase 9 — Second Runtime validation](../ROADMAP.md#phase-9--second-runtime-validation)

## 1. 目标和推进方式

在 P8 基线上，让同一个 YORVA Node 和 Desktop 同时管理 Hermes 与 OpenClaw，完成第二 Runtime 的实际管理闭环，并用测试验证现有抽象。

Owner 于 2026-09-08 指定 OpenClaw 2.0 最新版本，并要求快速推进、减少额外门禁。P9 采用四个连续工作批次和一次阶段验收；批次是执行顺序，不是逐批审批或冻结关卡。保留正确性、数据隔离、认证与必要回归，复用 P8 未受影响的证据。

本文件定义实现范围；2026-09-10 已完成规定的真实验证、回归、修复复审及基线记录。P8 的冻结 tag 和历史审计保持不变。

## 2. 版本与平台

| 项目 | P9 选择 |
| --- | --- |
| 第二 Runtime | OpenClaw，kind 为 `openclaw` |
| 当前最新稳定版 | `2026.9.3`；2026-09-10 在 B0 重新核验 GitHub latest 与 npm latest 一致 |
| “2.0”含义 | 官方将 `2026.8.1` 称为 OpenClaw 2.0；后续仍用日期版本，不使用虚构的 npm `2.0.0` |
| 上游源码 | tag `v2026.9.3`，commit `1391f7cd2d40ab5bbcf2f5f831d3a64f520e72d7` |
| 首个验收平台 | 原生 Windows x64，普通用户；其余平台按已有条件做构建/单元回归 |
| 分发对象 | 官方 OpenClaw CLI/Gateway；Windows Hub 是独立产品，不是 YORVA 的依赖 |

B0 开始时再核对一次最新稳定版。若上游已更新，修改同一份版本记录并验证受影响接口即可，不另开阶段或重复走批准流程；确定候选后使用明确版本和来源，不在测试途中自动追随 `latest`。未知版本可以被发现；未验证的变更不得被展示为已支持。兼容范围按实测扩大，无须人为限制为永久单一补丁版本。

`2026.9.3` 将 npm 声明的 Node.js 范围提高为 `>=24.16.0 <25 || >=26.1.0`。P8 的 `22.23.1` 不再满足 OpenClaw 要求；B0 使用测试目录内独立的官方 Node `24.16.0`，按官方 SHA-256 校验，不改动 Hermes 私有 Node 目录。

来源、完整 npm integrity 和接口依据见[上游核验记录](evidence/PHASE-009-OPENCLAW-UPSTREAM.md)。

## 3. 用户成功流程

```text
打开 Runtime 页面，同时看到 Hermes 和 OpenClaw
→ 检测已安装的 OpenClaw，看到版本、状态及可用能力
→ 创建或选择 OpenClaw 实例
→ 启动，看到经过认证检查的运行状态
→ 停止、重启，看到对应 Operation 和实际状态回读
→ 切回 Hermes，原有实例与功能继续可用
→ 重开 Desktop / 重连 Node，实例身份和状态重新核对
→ 删除本次创建的 OpenClaw 实例，其他实例和 Hermes 数据保留
```

首版允许接入已通过官方方式安装的 OpenClaw。未安装时给出明确安装指引和重新检测入口，不将重新制作完整安装器设为主线前置。

## 4. 范围

### 必须完成的主线

1. **发现与能力**：注册 OpenClaw；识别已安装、未安装、版本不支持、命令损坏、超时和歧义状态；只声明已经实现并验证的能力。
2. **实例闭环**：列表、创建、查看、删除与外部变化后的重新核对。一个 OpenClaw 实例映射为独立配置、状态、工作目录和端口的 Gateway profile；同一 Gateway 内的 agent 不冒充可独立启停的 YORVA 实例。
3. **生命周期与健康**：通过现有 Operation 完成 Start/Stop/Restart；回读真实状态；处理重复操作、冲突、超时及中断。端口可连接不等于认证成功或 Gateway 已就绪。
4. **双 Runtime 路由**：实例 ID → installation → Runtime kind → Registry bundle。实例、生命周期、能力查询、恢复检查及已有模型/Channel 等入口不得将 OpenClaw 请求误送 Hermes。
5. **Desktop**：Runtime 切换、实例归属、状态和 Operation；查询缓存包含 Runtime/Instance 身份；按能力展示操作；复用现有页面风格和中英文文案机制。
6. **兼容回归**：保留 Hermes 主流程、SQLite 管理身份、未受影响的模型/Channel/Skill/MCP/Backup 行为和 P8 恢复基础。

### 可在本阶段直接补齐的扩展

基础模型配置与凭据接入、安全的安装入口、日志和诊断读取、已稳定的 Channel/Skill/MCP 能力可以按收益和官方接口成熟度加入。它们不是完成主线之前的条件，也不要求与 Hermes 全量对齐。

扩展进入代码前，在本文件补一行具体交付项、来源和对应测试即可；普通实现选择不新增 Owner 审批。涉及新凭据权威、安装信任或架构变化时更新 ADR/安全契约。进入 AUDIT 后停止扩展，专注发现的问题。

当前尚无扩展被列为必交项。未实现的能力明确返回 `CAPABILITY_NOT_SUPPORTED`，不显示可用按钮，不绕到 Hermes 实现。

### 不在本阶段做

- P10 Headless Node 产品化、P11 Control Plane、P12 Fleet 或企业治理；
- 动态 Runtime 插件平台、市场、通用 SDK 或任意 shell/文件 API；
- OpenClaw 与 Hermes 的数据迁移、自动导入既有凭据或内部数据库集成；
- 全量平台/版本矩阵及两套 Runtime 功能完全一致。

这些边界不禁止为 P9 主线需要的小型接口调整、错误处理、UI 修正和回归修复。

## 5. 四个执行批次

| 批次 | 工作 | 可检查的结果 |
| --- | --- | --- |
| B0 — 官方接口实测 | 固定版本；在隔离环境验证入口、profile、初始化、生命周期、认证状态 JSON 和删除；选定一条原生 Windows 生命周期路径 | 一份短命令/输出契约记录，标明成功与失败结构、权限及副作用；不依赖伪造 Runtime |
| B1 — Node 与适配器 | OpenClaw adapter；Registry 实例接口；Runtime 目标解析；Operation/恢复/错误归一化；相关单元和契约测试 | 同一 Node API 可分别管理两种 Runtime；同名实例不串用；旧功能继续通过 |
| B2 — Desktop 闭环 | Runtime 选择和实例归属；能力驱动入口；缓存隔离；操作进度与错误；必要时补入已明确的小扩展 | 用户从界面完成两种 Runtime 的核心流程；不需要理解原生命令 |
| B3 — 验证与冻结 | 集中跑回归、真实 Windows 双 Runtime smoke、现有 CI/构建；审计实际 diff，修复阻断问题并复验 | 验收证据、审计结论、基线提交及可选 tag；通过后才标 FROZEN |

B0 的实测是实现的一部分，不单独设耗时门禁。若官方能力确实不足，记录具体问题和最小调整；不无限扩展平台验证，也不把必交的实例/生命周期能力悄悄改成只读。

## 6. 模块和契约影响

| 模块 | 变更原则 |
| --- | --- |
| `services/node/internal/runtime/openclaw` | OpenClaw 命令、版本、profile、端口、认证、输出结构、状态与路径保护均在此归属 |
| `services/node/internal/runtime` | Registry 增加真正需要的实例接口；复用 Lifecycle、Health 等小接口，不建第二套 Operation 系统 |
| `services/node/internal/app` | 从持久化 installation 解析 Runtime；替换相关 `hermesRuntimeID`、单一 Profile source 和 Hermes 错误耦合；保留不受影响的专用安装流程 |
| `services/node/internal/transport/httpapi`、`api/openapi.yaml` | 沿用 detect/instances/lifecycle 路由；注册列表、能力或实例归属确需补字段时使用 typed DTO，同步生成 TS 类型 |
| `services/node/internal/persistence/sqlite` | 优先复用 installation/instance/operation 表与唯一约束；需要新的管理元数据时才加迁移，并测空库与 P8 升级 |
| `apps/desktop/src` | 切换 Runtime、实例归属和能力驱动通用页；Hermes 专用安装详情仍可保留专用展示组件 |
| Tauri / packaging | 复用 P8 引导和发布链；仅在本阶段实际需要时改原生桥或包资源 |

已发现的抽象问题有明确代码依据：`instance_inventory.go` 的 Hermes-only guard/source，`lifecycle.go`、`models.go`、`channels.go`、`shared_models.go` 的目标解析，`node_recovery.go` 的单 Runtime 检查，以及 `App.tsx` / `InstancesPage.tsx` 的 Hermes 查询键和标识。只调整进入双 Runtime 调用链的部分；不开展全仓库命名整理。

文档同步：实现时更新 [RUNTIME](../RUNTIME.md)、[PROTOCOL](../PROTOCOL.md)、[ARCHITECTURE](../ARCHITECTURE.md) 和受影响的 [SECURITY](../SECURITY.md)；发生 schema 变化才更新 [DATA_MODEL](../DATA_MODEL.md) 的具体表结构。架构选择记录于 [ADR-0021](../adr/ADR-0021-openclaw-second-runtime.md)。计划文件不提前把未实现的能力写成现行支持。

## 7. 实例、进程与数据约定

- 优先使用适合该操作的官方管理 API/协议；离线初始化和原生服务生命周期使用官方 CLI。所有调用由 adapter 构造固定命令和校验后的 argv；无通用 `gateway call` 转发接口。
- Gateway profile 使用 adapter 派生的安全标识。默认/已有用户 profile 保持保护，未经明确选择不接管。列表先识别受支持的 profile 根，再通过官方读取核对；文件存在本身不等于健康。
- 每个实例配置、状态、workspace、服务身份及派生端口独立。只设置 `OPENCLAW_STATE_DIR` 不足以隔离原生服务；Windows 测试必须有独立命名 profile 和端口，涉及 Desktop 时使用可丢弃用户或 VM。
- B0 优先验证 profile-scoped 官方原生服务接口；进程所有权和登录持久化副作用必须记录。手动启动不得默默新增登录项；若采用的路径需要启用持久化，使用明确的产品选择。关闭 Desktop 不自动停止 Runtime。不能满足时在 B0 收敛到有明确所有权的受支持路径，并更新 ADR 后实现。
- 创建先检查名称/路径/端口冲突；初始化成功后权威回读再写管理结果。失败只清理可证明属于本次创建的内容；删除只针对明确选中的受支持实例并确认名称。未知或外部内容不靠宽泛目录删除清理。
- 对一个实例的启停/删除/配置冲突复用现有保护；不同 Runtime 的独立操作不被一个全局锁串行化。慢命令不占据数据库事务。
- YORVA SQLite 只保存管理状态。OpenClaw 自己的会话、配置和凭据保持 OpenClaw 权威；不读取其内部表。Gateway 自有认证不复用 YORVA 的本地 API token。凭据不进入 argv、普通 API 响应、日志、事件或诊断。
- 恢复检查按 Runtime 核对：未安装且从未接入的 Runtime 不妨碍新节点；已经接入的 Runtime 丢失/未知不能被另一 Runtime 的成功掩盖。受影响 Runtime 报恢复需求，健康 Runtime 的管理仍可访问。

## 8. 测试矩阵

| 场景 | 预期 | 层级 |
| --- | --- | --- |
| 未安装/已支持/未知版本/损坏/多候选/超时 | 稳定发现结果；不误宣称支持、不触发安装 | adapter + API |
| Hermes 与 OpenClaw 使用同名原生实例 | installation 隔离、YORVA ID 稳定；请求只到目标 adapter | app + SQLite |
| OpenClaw 创建、重复创建、删除与保护实例 | 成功回读；重复/确认错误/受保护目标安全失败；不改另一实例 | adapter + app + 真实 smoke |
| 两个 OpenClaw 实例同时运行 | profile、工作目录、服务和端口不冲突；操作 A 不改变 B | adapter + 真实 smoke |
| Start/Stop/Restart 与状态回读 | Operation 最终态和实态一致；认证失败/仅监听不报就绪 | adapter + app + 真实 smoke |
| 同实例并发、超时、取消或 Node 中断 | 冲突可解释、工作有归属；不残留永久 RUNNING 或误报成功 | app + adapter |
| 外部移除/重现/读取失败 | MISSING/AVAILABLE/UNKNOWN 正确；不把查询失败当删除 | app + SQLite |
| OpenClaw 调用未支持的模型/Channel/Skill 等入口 | capability false + 稳定错误；Hermes adapter 未被调用 | app + API + Desktop |
| Runtime 切换、SSE 重连与缓存 | 不显示另一 Runtime 的清单/操作；重新读取权威状态 | Desktop |
| 无效认证、注入名称、路径逃逸、秘密输出 | 拒绝危险输入；不泄漏；不调用非目标命令或删非目标文件 | API + adapter |
| P8 数据升级和 Hermes 核心流程 | 原有 ID/管理数据保留；受影响旧测试通过；若有迁移测升级及失败恢复 | 回归 + smoke |
| Runtime 部分失败与恢复检查 | 精确报告异常 Runtime；不能遮蔽丢失的已接入安装，健康侧仍能管理 | app + API |

假实现用于确定性负面场景；最终 smoke 必须使用真实 OpenClaw 和 Hermes。主线不依赖付费模型或生产 Channel 凭据；如果实际加入模型推理/Channel 登录能力，则另为该具体能力保留真实成功证据，不能用 Gateway 健康代替。

## 9. 三项验收条件

| 条件 | 通过标准 |
| --- | --- |
| G1 — 功能与隔离 | 第 3 节主流程及第 8 节关键场景通过；至少一轮普通用户 Windows x64 真 Runtime 双方共存证据，包含两个 OpenClaw 实例与一个 Hermes 实例、启停/删除和重连 |
| G2 — 回归与可构建 | Go test/vet/build；受影响并发包在支持环境跑 race；API lint/生成无漂移；Desktop typecheck/lint/test/build；现有 CI 和 Windows 构建通过。原生或打包有改动时跑相应 native/package 检查；对最终候选产出一个可测试 Windows 包 |
| G3 — 审计与记录 | 按 AUDIT_STANDARD 从实际仓库和 diff 做独立复核；必交项通过，零 CRITICAL 和未解决的阻断 HIGH；其他问题按标准处理，契约和证据一致 |

执行顺序是开发时跑相关测试，候选稳定后集中跑一次完整现有 CI 与 Windows smoke。只因新修改、失败或证据缺口重跑受影响检查，不要求每批提交完整重跑。真实 smoke 以覆盖场景为完成标准，不设四小时、八小时或 72 小时等待门槛。

P8 已通过的发布基础按未受影响范围继承。P9 不以生产签名材料、公网发布、全平台矩阵、付费模型额度或主机重启为阶段前提；它们未被降低为已经验证或已经发布。若 P9 改动实际破坏更新/恢复路径，则补相应回归，不通过“继承 P8”跳过已受影响行为。

## 10. 审计与收尾

遵循 [PHASE_GOVERNANCE](../PHASE_GOVERNANCE.md) 和 [AUDIT_STANDARD](../AUDIT_STANDARD.md)。一次集中审计覆盖其 12 个维度；不受影响的项简述继承证据或 N/A 理由。复核从仓库事实开始，不能只读实现总结；可用独立审阅上下文，不要求额外并行 agent。

保留失败和修复历史，复审仅覆盖受影响维度。未解决的阻断正确性/安全问题必须修复；真正非阻断的事项按标准记录 Owner 和触发条件，不制造新的阶段依赖。

通过后记录候选 commit、上游版本、测试/CI、真实 smoke 和审计结论，建立 P9 基线；如创建 tag，使用 `phase-009-openclaw-baseline`。完成基线记录后再标记 FROZEN。不提前开始 P10。

## 11. 已知风险与处理

- 最新上游文档会随主线变化：B0 以固定 release 的实际 CLI/协议为准，保存必要的结构化样例，避免只据在线文档猜字段。
- 官方 Windows 服务有 Scheduled Task / Startup 回退以及状态目录身份限制：先实测普通用户行为和持久化副作用，避免后期被错误生命周期假设拖住。
- OpenClaw agent 与 Gateway profile 不是同一粒度：YORVA 独立实例使用后者，agent 细分留在 Runtime 自身。
- 上游 Fleet 目前实验性且 Windows 未测试：P9 的本地双 Runtime 验证不引入容器 Fleet。
- 未提供安全写入接口的可选能力保持未支持；不复制 Hermes 文件兼容写入来冒充 OpenClaw 官方支持。

## 12. 完成证据

| 项目 | 最终结果 |
| --- | --- |
| P8 基线 | 保留 `phase-008-local-product-hardening-baseline` 和文档后继；无历史重写 |
| B0 / B1 / B2 | OpenClaw 2026.9.3 官方接口实测、Node 双 Runtime 路由与 Desktop 主线完成 |
| G1 | PASS；最终候选在普通用户 Windows 11 x64 完成一个 Hermes 与两个 OpenClaw 的同名隔离、认证启停、重启、重连及删除；[R2 原生证据](evidence/PHASE-009-WINDOWS-G1-R2.json) |
| G2 | PASS；最终 CI #105 全部检查、151 项 Desktop 测试和 MSI #43 构建/检查/上传通过；[验证记录](evidence/PHASE-009-VALIDATION.md) |
| G3 | PASS；独立新上下文复核实际源码，M1/L1/H1/H2/H3 全部关闭；[审计及失败历史](audits/AUDIT-009-openclaw-second-runtime.md) |
| 最终产品候选 | `b958801a91234b5a602d652035e5c4f3e3dc8242` |
| 基线和状态 | `phase-009-openclaw-baseline`；FROZEN；[基线记录](evidence/PHASE-009-BASELINE.md) |

冻结提交仅补充文档/证据，产品、测试、脚本、API、workflow 和依赖与上述测试提交相同。
MSI 为未签名内部测试包；不代表公开生产发布、全平台/版本或可选能力已验证。
P9 分支基线独立记录，不移动 P8 tag，不提前实现 P10，不以本记录代替 main 合并。

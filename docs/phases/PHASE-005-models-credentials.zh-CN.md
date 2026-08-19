# YORVA Phase 5 — 模型与凭据

> 状态：IN_PROGRESS — 已授权 Batch 1-5
> 语言：中文 Owner 审核源
> Owner：仓库 Owner
> Owner 指定的设计快照：`089a58005edc8f8f6a72b4fb44276be7c322eb1d`
> 实现所需基线：`phase-004-instance-profile-baseline` → `0dd5d432affd44e23bf577acfe9cf2fdfbfa3f45`
> 英文执行镜像：`PHASE-005-models-credentials.md`
> Owner 决策 D1–D6 与 ADR-0007：**已批准** 2026-08-19
> 实现分支：`codex/phase5-models-credentials`
> 执行授权：Batch 1-5、审计、CI、合并/冻结/tag 与 Windows release build
> Batch 1：已完成
> Batch 2：已完成
> Batch 3：已完成
> Batch 4：进行中

本文档与英文镜像共同定义同一份合同。Owner 审核中文版。2026-08-19，Owner 已批准 D1–D6、ADR-0007、同步治理变更、真实 Phase 4 基线，并批准中英文 Spec 标记为 `READY`。首次 Batch 1 资格检查在确认 pinned 官方 CLI 通过 argv 泄露秘密且 Web/TUI setter 需要常驻服务后正确 STOP。随后 Owner 批准 D3 中的狭窄 compatibility fallback，并一次性授权 Batch 1-5、审计、CI、合并/冻结/tag 与 Windows release build 连续执行。该历史 STOP 证据继续保留在资格记录中。

## 1. 目标

为已有 Hermes-backed YORVA Instance 提供以中国市场为优先、基于预设的模型配置体验：

```text
AVAILABLE Instance
→ 在现有 Instance 体验中打开“模型”
→ 选择 Provider 预设
→ 选择推荐模型或输入模型 ID
→ 输入 API Key
→ 通过经过资格验证的 Hermes Profile credential surface 保存
→ 用户明确发起连接测试
→ 展示安全结果
```

用户无需理解 Hermes 配置键、环境变量名、YAML、`.env`、Base URL 或命令行参数。YORVA 关闭后，Hermes 仍须能够直接使用同一 Profile 配置。

Phase 5 仅配置模型/Provider 访问。不启动或监管 Hermes，不重定义 Instance 可用性，不增加聊天/推理界面，也不提前进入 Phase 6 渠道能力。

## 2. 仓库基线与已有能力

Owner 指定的设计快照为 `089a58005edc8f8f6a72b4fb44276be7c322eb1d`。有效 Phase 4 重新冻结基线是 annotated tag `phase-004-instance-profile-baseline`，其 peeled commit 为 `0dd5d432affd44e23bf577acfe9cf2fdfbfa3f45`，并位于 `main` 与 `origin/main`。因此：

- 本 Spec 以有效重新冻结 commit 作为实现基线；
- 不修改 Phase 4 文档、代码、历史或既有审计结论；
- 无关的未提交用户工作必须保留，不属于 Phase 5。

Phase 5 必须复用，不得重做或复制：

- 稳定的 `instanceId` / `nativeId` 身份分离；
- `AVAILABLE` / `MISSING` / `UNKNOWN` 语义及永久保留的 `MISSING` tombstone；
- 通过现有 discovery 解析 active Hermes executable；
- 现有 Hermes `commandRunner`、Windows Job Object / Unix process group、输出上限及进程清理；
- 现有 authenticated loopback API、route contract 与错误 envelope；
- `api/openapi.yaml`、生成的 Desktop client 与 DTO 映射约定；
- 现有 Runtime registry/bundle wiring；
- 现有 Instance mutation/reconciliation 协调源；
- 现有 Operation 框架，用于需要等待网络的验证；
- 现有 Desktop sidebar、`App.tsx` 查询组合及 Instances 页面；
- 现有 `i18n.ts`、English/简体中文消息与 locale 持久化；
- 现有 `formatDateTime` 本地时间格式化；
- 现有 Go、OpenAPI、React/Vitest、Rust/Tauri 与 Windows process 测试约定。

Phase 5 不得建立第二套导航、i18n、process runner、Runtime registry、Operation、时间格式化器或 server-state store。

## 3. 架构冲突与所需治理决策

中国市场 ProviderPreset 与消费级 UX 属于普通 Phase 5 Spec refinement；拟议的凭据权威变更不属于普通细化。

| 现有冻结规则 | 拟议 D2 | 分级 |
|---|---|---|
| `SECURITY.md §7`：Instance/模型 Provider 凭据置于 OS-backed `SecretStore` 后，不允许静默明文降级。 | MVP 以 Hermes Profile 官方凭据存储（通常为 Profile `.env`）作为唯一凭据权威。 | 实质性安全架构冲突。 |
| `DATA_MODEL.md §10`：`secret_refs` 引用 OS 安全存储中的秘密。 | Phase 5 不为 Hermes 模型 API Key 建立 `secret_refs`。 | 实质性数据所有权/文档变化；无需删除 schema。 |
| `ARCHITECTURE.md`：持久层保存 secret reference，并包含 secrets adapter 边界。 | Hermes 拥有这些 Runtime-native credentials；YORVA 委托 Hermes 持久化。 | 实质性秘密边界调整。 |
| `ROADMAP.md`：Phase 5 包含 secure-store integration。 | 当 Hermes Profile 官方存储满足获批合同时，Phase 5 MVP 省略 SecretStore。 | Roadmap 交付项变化。 |
| `PROTOCOL.md`：秘密 write-only，GET 仅返回 metadata。 | API 规则保持不变。 | 无冲突。 |
| Source-of-truth：Hermes 拥有 Hermes 状态。 | Hermes 拥有 Profile 模型凭据/配置。 | 一致。 |

治理处理已于 2026-08-19 完成：

1. D1–D6 在 Phase 5 仍为 `DRAFT` 时获批，不需要 Phase Amendment。
2. ADR-0007 已获 Owner 批准，并定义 Runtime-native credential authority、静态存储权衡、Profile 隔离及其与未来 `SecretStore` 的关系。
3. `SECURITY.md`、`DATA_MODEL.md`、`ARCHITECTURE.md` 与 `ROADMAP.md` 已同步 ADR-0007。
4. 首次 Batch 1 资格 STOP 被保留。2026-08-19，Owner 修订 D3/ADR-0007，批准下述狭窄 Hermes-native credential compatibility writer。
5. 后续若实质修改该 authority，必须建立 Phase 5 Amendment，并在需要时建立 superseding ADR。

## 4. 需要 Owner 决策

- [x] **D1 — 中国市场优先的 ProviderPreset catalog。** 产品候选为 DeepSeek、Qwen/Alibaba DashScope、Kimi/Moonshot、MiniMax、GLM/Zhipu、OpenRouter、OpenAI 与 Anthropic。这是产品方向，不代表 pinned Hermes 已支持全部候选。Batch 1 必须核验准确的 Hermes provider ID、credential mechanism/name、config key、中国/区域 endpoint 行为及推荐 model ID。不支持的候选从可选 MVP catalog 移除，或以不可选的“不支持”状态展示；YORVA 不实现其协议。
- [x] **D2 — MVP 使用 Hermes-native credential persistence。** 在第 3 节 ADR 获批的前提下，Hermes Profile 官方凭据存储是唯一真相源。YORVA 不为这些模型 Key 实现 SecretStore 或 `secret_refs`，也不保留副本。SQLite、日志、事件、Operation、HTTP response、diagnostics、argv 与 Desktop storage 均不得包含秘密。
- [x] **D3 — 官方接口优先，已批准狭窄 compatibility fallback。** 优先采用 pinned Hermes `0.20.2` 文档化、非交互式的 Profile/config/credential 接口。资格检查已证明 offline 官方 setter 要求 secret argv，而安全 JSON setter 需要常驻服务。因此只允许 Hermes adapter 使用 version-fixed、Profile-scoped、Provider-allowlisted writer 更新准确的 canonical Profile `.env`。调用方不能提供 path 或 env key。Writer 必须有界、保留未知项、只修改一个 allowlisted key、使用同目录 atomic replace/read-back，并在观察到外部修改时 fail closed。仍禁止 Hermes Python import、任意 `.env`/YAML 编辑及任何 generic file API。
- [x] **D4 — 保存与验证分离。** 保存写入凭据和非秘密 provider/model 配置，并安全 read-back/status confirm；不发起推理请求，也不消耗 token。只有用户明确点击“测试连接”才启动受限验证。`CONFIGURED` 不等于 `VALIDATED`，两者也不会把 `AVAILABLE` 改为 `MODEL_READY`。
- [x] **D5 — MISSING Instance 永久保留。** `MISSING` Instance 及稳定 `instanceId` 永久保留。Reconciliation 不自动删除其 Hermes 凭据。`MISSING`/`UNKNOWN` 禁止 config mutation 与 validation；只有安全官方 Profile 接口可寻址时，才允许显式删除凭据。未来清理策略须另行制定合同。
- [x] **D6 — 五个顺序 Batch。** 按第 20 节顺序执行，focused gate 通过后才能进入下一 Batch。Spec `READY` 后，Owner 可另行授权五个 Batch 自动连续执行。任何 Batch 不得借入 Phase 6 或更晚范围。

被拒绝或修改的决策必须在中英文版本中同步后，方可标记 `READY`。

## 5. 用户可见成功流程

1. 用户打开 `AVAILABLE` Instance，并在现有 Instances 体验内展开/打开“模型”面板。
2. YORVA 展示分为“国内推荐”和“其他兼容 Provider”的小型 catalog。
3. 用户选择受支持 preset；YORVA 展示简短说明和推荐模型列表。
4. 用户选择推荐模型或输入有边界的 model ID；不提供自定义 endpoint。
5. 用户在 password input 中输入 API Key，点击“保存配置”。
6. Go 将 `instanceId` 解析为权威的当前 `nativeId` 和 active Hermes executable。
7. Hermes adapter 通过经验证的 Profile credential surface（需要时包括获批狭窄 fallback）保存凭据和非秘密 provider/model 配置，仅确认安全的 status/config metadata。
8. 输入框被清空；UI 显示 `CONFIGURED`，但尚未执行网络验证。
9. 用户点击“测试连接”；YORVA 通过现有 Operation/process 基础设施启动受限 `model.validate` Operation。
10. UI 以本地时间和安全建议展示 `PASSED`、`FAILED` 或 `UNKNOWN`。
11. YORVA 退出后，Hermes 仍可通过其原生 credential/config 机制使用同一 Profile 配置。

## 6. 范围内

- 中国市场候选优先的小型静态 ProviderPreset catalog；
- 对每个可选择 preset 的 pinned Hermes 资格验证；
- 推荐 model ID 与有边界的手动 model ID 输入；
- 安全读取、应用及 read-back Profile provider/model 配置；
- 通过官方接口或 ADR 获批狭窄 fallback 实现 Hermes-native Profile credential set/replace/delete/status；
- 仅 metadata 的 credential read 与显式 connection validation；
- authenticated loopback API、OpenAPI/generated client 与稳定错误更新；
- 仅为 validation 复用现有 Operation；
- 与 Profile delete/reconciliation 的冲突保护；
- 现有 Instance 页面内的双语 preset UX；
- secret redaction、Profile isolation、process cleanup 与 restart test；
- exact-commit verification 与独立 `AUDIT-005` handoff。

## 7. 范围外

- 任何 Phase 4 实现或 freeze remediation；
- Hermes install/repair/upgrade/uninstall 或 generation/environment 变更；
- Profile create/delete/rename/clone/import/export 变更，配置 mutation 协调除外；
- Hermes lifecycle、gateway/service startup 或 supervision；
- OAuth、browser/device-code login、Nous Portal login 或 token import；
- 在线 Provider marketplace/catalog download；
- 动态 Provider/Runtime plugin 或通用 harness framework；
- custom Provider、custom endpoint/base URL、proxy 或任意 auth scheme；
- fallback chain、routing policy、quota、pricing 或完整在线 model discovery；
- 自由编辑 YAML、`.env`、shell、command、environment variable、path 或 config key；
- 获批 Hermes adapter credential writer 之外的直接或通用 `.env` 读写/追加；
- Windows user/system environment variable mutation；
- chat/inference UI、Agent readiness、session、memory 或 persona editing；
- channel、Weixin/WeCom、Skills、MCP、backup/restore、Cloud 或 telemetry；
- Hermes 模型凭据的第二份 SecretStore 副本。

## 8. 架构边界与复用

```text
现有 React Instance 体验
    ↓ 现有 generated typed client
现有 authenticated loopback API / OpenAPI
    ↓
Application model-config/credential use cases
    ↓                         ↓ 现有 Operation engine（validation）
Runtime-neutral intent        typed progress/result
    ↓
Hermes adapter
    ↓
现有 executable resolution + commandRunner/process containment
    ↓
Pinned Hermes Profile config/credential/validation 接口
    └── 必要时使用 ADR-0007 canonical `.env` 狭窄 credential fallback
```

Go 是 use-case coordination、identity resolution、ProviderPreset selection、command construction 与 result normalization 的权威。Hermes 是 Hermes Profile config/credential state 的权威。React 只渲染标准化状态，并仅在当前表单交互期间持有 password input。Rust/Tauri 不增加模型业务逻辑。

ProviderPreset mapping 属于 Hermes-specific 行为，应位于 `services/node/internal/runtime/hermes` 或明确归 Hermes 所有的文件/包。稳定 selection/status DTO 可位于小型 domain/application model-config 区域。不得借此重组无关 Hermes installation/profile 文件。

禁止的流向：

```text
React → Hermes CLI/.env/YAML
React → credential env name 或 config key
Domain → Hermes provider identifier
Hermes adapter → HTTP/UI
SQLite/Operation → credential/config authority
Rust/Tauri → provider/credential decision
```

## 9. ProviderPreset 模型

`ProviderPreset` 是最小编译期 allowlist，不是 plugin API。

概念内部字段：

```text
id
displayNameKey
region                   CHINA | GLOBAL
hermesProviderId         adapter-private
credentialEnvName        adapter-private
recommendedModels[]
compatibility            SUPPORTED | UNSUPPORTED
optionalHelpTextKey
```

如实现仍清晰，可减少字段。要求：

- 静态编译并随版本评审；
- 实际 Hermes mapping 由 Hermes adapter 拥有；
- 不允许任意 env/config/shell 值；
- 不进行 dynamic discovery、marketplace 或 plugin loading；
- 安全 API/Desktop DTO 不暴露 `hermesProviderId`、`credentialEnvName` 或内部 config key；
- 只有 `SUPPORTED` preset 可选择；
- 仅能在受支持 preset 内手动输入 model ID；
- 推荐模型是经评审常量，不表示完整 model catalog。

Batch 1 资格验证前的产品候选：

| 分组 | 候选 | 初始兼容性 |
|---|---|---|
| 国内推荐 | DeepSeek | `TO_BE_QUALIFIED` |
| 国内推荐 | Qwen / Alibaba DashScope | `TO_BE_QUALIFIED` |
| 国内推荐 | Kimi / Moonshot | `TO_BE_QUALIFIED` |
| 国内推荐 | MiniMax | `TO_BE_QUALIFIED` |
| 国内推荐 | GLM / Zhipu | `TO_BE_QUALIFIED` |
| 其他兼容 | OpenRouter | `TO_BE_QUALIFIED` |
| 其他兼容 | OpenAI | `TO_BE_QUALIFIED` |
| 其他兼容 | Anthropic | `TO_BE_QUALIFIED` |

MVP 至少须有一个中国市场 preset 通过资格验证。若 pinned Hermes 一个也不支持，必须停止并由 Owner 决定 Hermes 版本/范围；不得在 YORVA 内新建 Provider 协议。

## 10. Pinned Hermes 接口资格验证

集成目标：Hermes `0.20.2`，commit `df4b65147d7ddd74dd449f9067aabbca5aef0ec7`。

Batch 1 必须针对每个候选确认：

1. 准确 Hermes provider ID 与 alias；
2. 使用现有 `nativeId` 边界选择 Profile 的准确机制；
3. 准确的非秘密 model/provider key 与 scalar format；
4. 准确的 credential logical/env name 与 Profile isolation 行为；
5. Hermes 是否内建中国/全球 endpoint 选择，且无需 custom URL；
6. 准确的官方非交互 set/replace/delete/status 接口，以及任何 fallback 的记录化理由；
7. secret transport channel，并证明不经过 argv/output/log；
8. 准确 read-back/status output 与严格有界 parser 要求；
9. 禁用 tools 的安全非交互 validation；
10. timeout、cancellation、output-limit 与 exit/error mapping。

选择顺序仍为 documented official API、documented programmatic protocol、documented CLI，最后是 ADR-0007 已批准的狭窄 compatibility fallback。上游当前 `main` 不能证明 pinned version 行为。

已完成的资格检查未发现安全的 offline 官方 credential setter。因此 fallback 只允许用于能够从 pinned Hermes 证明准确 credential key 与 canonical Profile location 的候选。若候选需要 OAuth/login、custom endpoint/provider protocol、含糊 storage、Hermes Python import 或越出该 bounded writer 的 mutation，则不受支持。

## 11. 身份、可用性与模型状态

- Public route、Operation target 与 YORVA relation 使用 `instanceId`。
- 只有 Hermes adapter 接收 `nativeId` 以选择 Profile。
- `instanceId` 与 `nativeId` 不得互换。
- `AVAILABLE` 仅表示 Profile 存在于最近一次成功查询结果。
- `MISSING` 是永久 tombstone；`UNKNOWN` 表示权威状态不可用。
- `MISSING`/`UNKNOWN` 禁止保存、凭据 mutation 与 validation。

配置状态：

| 状态 | 含义 |
|---|---|
| `UNCONFIGURED` | 所需 provider/model/credential status 不完整或无法确认。 |
| `CONFIGURED` | 已确认受支持 preset、model 与 Hermes-native credential status；不声称连接成功。 |

验证状态：

| 状态 | 含义 |
|---|---|
| `NOT_RUN` | 没有已完成的显式验证。 |
| `PASSED` | 最近一次显式验证成功。 |
| `FAILED` | Provider/model/credential 被拒绝，并产生标准化安全错误。 |
| `UNKNOWN` | Timeout、cancellation、不安全 output 或 transport ambiguity 导致无法裁决。 |

不存在 `MODEL_READY` availability。`PASSED` 不代表 Agent、gateway、channel 或 lifecycle ready。

## 12. Runtime 与 Credential 合同

合同保持聚焦；准确命名可遵循仓库现有约定：

```text
ListProviderPresets() → safe static presets
ReadModelConfig(installation, nativeId) → normalized safe config/status
ApplyModelConfig(installation, nativeId, presetId, modelId) → safe observed config
SetCredential(installation, nativeId, presetId, secret input) → metadata/status only
DeleteCredential(installation, nativeId, presetId) → metadata/status only
ValidateModel(installation, nativeId, presetId, modelId) → typed validation result
```

不得建立 generic provider plugin registry。Runtime-neutral application intent 通过现有 Runtime bundle/registry 解析；Hermes-specific mapping 留在 Hermes adapter。

每个 process call 均复用 active executable resolver 与 `commandRunner`。若安全 credential 接口需要 stdin，仅对现有 runner 作最小扩展，不添加第二个 runner。复用现有 process-tree containment、独立有界 output、allowlisted environment、timeout/cancellation 与 cleanup。

## 13. Hermes-Native Credential Authority

在 D2/ADR 获批前提下：

- 所选 Hermes Profile 的官方 credential store 是 model API Key 唯一真相源；
- YORVA 不在 SecretStore、SQLite 或自有文件保存副本；
- 只有 Hermes adapter 的获批 compatibility writer 可打开 canonical Profile `.env`；其他 production layer 均禁止；
- 优先使用官方 Hermes surface；fallback 只接收 `nativeId`、allowlisted preset 与 secret value，绝不接收调用方 path/env key；
- status 仅由资格 surface/writer 推导安全 presence metadata，绝不返回值或 secret-derived fragment；
- credential mutation 精确限定到由 `nativeId` 选择的 Profile；
- Profile A 的 credential 不得出现在 Profile B 的 process、status 或 validation 中；
- replace/delete 失败须标准化；安全时可重试，YORVA 不猜测 native state；
- 部分 Save 以 `UNCONFIGURED` 和稳定 incomplete/apply error 报告，并允许幂等重试；
- MVP 不增加跨文件 transaction/journal 或 rollback framework。

`.env` 的正式定位：

> Hermes Profile `.env` 是 Hermes-owned 官方 credential storage，不是通用 YORVA file API，也不是 OS-secure YORVA SecretStore。允许使用它是明确的 MVP at-rest security tradeoff，必须由第 3 节 ADR 批准。

## 14. SecretStore、SQLite 与 Windows Environment

推荐的 MVP 决策：

- 不为 Hermes model API Key 实现 `SecretStore`；
- 不为这些 Key 创建或填充 `secret_refs`；
- 在所需 ADR 明确范围后，为未来 YORVA-owned device/cloud/channel secret 保留通用架构概念；
- 本 Phase 不删除或改变文档中的 `secret_refs` model；
- 不在 SQLite 保存 provider/model 真相；Hermes 保持权威；
- 现有 Operation 仅作为 validation projection，不作为 credential/config truth。

Windows user/system environment mutation 在范围外，默认禁用。多个 Profile 可使用不同 Key，因此全局环境变量不能正确表达 Profile isolation。未来“同步到用户环境”选项须明确 opt-in，并通过独立安全/产品设计及后续 Phase/Amendment。

## 15. 保存、Mutation 与并发合同

产品中的“保存配置”使用一个 credential resource PUT 请求，避免客户端跨两次请求协调秘密与非秘密状态：

```json
{
  "providerPresetId": "deepseek",
  "modelId": "reviewed-or-manual-model-id",
  "value": "write-only-api-key"
}
```

该 application use case 在服务端协调 `SetCredential` 与 `ApplyModelConfig`，仅返回安全 `ModelConfiguration`/credential metadata。`PATCH /config` 只用于已有可确认凭据时变更非秘密 provider/model；它不接收 API Key。规则：

- 拒绝 unknown/trailing field、empty/oversized value 与 control character；
- `providerPresetId` 必须是可选择的静态 preset；
- `modelId` 是有边界文本，不得成为 URL、path、config key、env name 或 shell/argv fragment；
- API Key 仅接收一次且绝不 echo；
- mutation 前重新解析受支持 installation 与 `AVAILABLE` Instance；
- 使用现有 Instance/Profile coordination source，使 Save 与 Profile delete/reconcile 冲突；不得添加独立 lock registry；
- Save 期间不进行 network model call；
- 仅使用准确且已验证的 surface，包括获批狭窄 fallback；
- 返回 `CONFIGURED` 前确认安全 provider/model 与 credential-status metadata；
- 若 native step 部分成功，返回稳定 partial/incomplete error 与安全 observed state；不自动删除旧有效 credential，也不声称 crash-safe rollback；
- 相同 desired request 可安全重试，但不添加 generic config transaction framework。

Credential delete 是独立显式动作，通过官方 Profile 接口删除当前选中 preset credential，仅返回 metadata。

## 16. Authenticated API 与 OpenAPI

扩展现有 authenticated loopback API 与 `api/openapi.yaml`；不得建立新 transport/client。

拟议路由：

```text
GET    /api/v1/runtimes/hermes/model-provider-presets
GET    /api/v1/instances/{instanceId}/config
PATCH  /api/v1/instances/{instanceId}/config
GET    /api/v1/instances/{instanceId}/credentials/model-provider
PUT    /api/v1/instances/{instanceId}/credentials/model-provider
DELETE /api/v1/instances/{instanceId}/credentials/model-provider
POST   /api/v1/instances/{instanceId}/model-validation
```

Preset response 只包含安全产品字段，不暴露 env name、Hermes config key 或内部 CLI 细节。Config GET 返回安全 provider preset/model/configuration state 与最近 validation summary。Credential GET 仅返回 `configured`、preset ID 与安全 timestamp/status metadata。

Credential PUT 是第 15 节完整 Save 请求；Config PATCH 是不含秘密的 closed schema，仅允许在凭据状态可确认时修改 provider/model。PATCH/PUT 均采用 closed schema 与 no-store response。实现必须扩展现有 route method/CORS allowlist 以支持 `PATCH`/`PUT`；不得绕过 `routeContract`、bearer authentication、origin check 或 generated-client workflow。

`POST .../model-validation` 接受 closed empty body，返回 `202` 和现有 `Operation` schema。Operation type 为 `model.validate`，target type 为 `instance`，target ID 为稳定 `instanceId`。

最小稳定错误：

```text
INSTANCE_NOT_FOUND
INSTANCE_NOT_AVAILABLE
MODEL_PROVIDER_UNSUPPORTED
MODEL_CONFIG_INVALID
MODEL_CONFIG_QUERY_FAILED
MODEL_CONFIG_APPLY_FAILED
MODEL_CONFIG_INCOMPLETE
MODEL_CREDENTIAL_REQUIRED
MODEL_CREDENTIAL_WRITE_FAILED
MODEL_CREDENTIAL_DELETE_FAILED
MODEL_VALIDATION_FAILED
MODEL_VALIDATION_TIMED_OUT
MODEL_VALIDATION_CANCELLED
INSTANCE_CONFIG_CONFLICT
```

原始 Hermes/provider output、path、env name、config key 与 secret 不得跨越 HTTP。

## 17. 显式 Validation Operation

Validation 等待网络 Provider，因此复用现有 durable Operation，而不是让 HTTP request 长时间保持，或建立第二套 task system。

- 只有用户明确点击“测试连接”才创建 `model.validate`；
- validation 开始时重新解析权威 Profile config/status；
- 使用固定无害 prompt 与关闭 tools 的已验证 Hermes 接口；
- 不启动 gateway，也不遗留 process；
- 使用 Batch 1 确立的单一 whole-operation deadline；
- client cancellation 复用现有 Operation cancellation 与 process cleanup；
- timeout、cancellation、provider rejection 与 unsafe output 保持不同稳定结果；
- 不持久化、记录、发送或返回 model text/raw provider response；
- Operation event 保持 closed/redacted，Desktop 重启后可恢复 Operation；
- validation 失败不修改/删除 credential 或 config。

## 18. Desktop UX 与 i18n

复用现有 sidebar 与 Instances navigation。不得增加新的顶层“模型”菜单或第二套 navigation tree。在当前 Instances 体验中，为选中 Instance 增加 Models panel/detail flow。

UI 布局：

```text
添加模型 Provider

国内推荐
  DeepSeek / Qwen / Kimi / MiniMax / GLM（仅验证通过的条目可选择）

其他兼容 Provider
  OpenRouter / OpenAI / Anthropic（仅验证通过的条目可选择）

Provider 说明
模型：[推荐模型选择] [手动 model ID 选项]
API Key：[password input]
[保存配置] [测试连接]
```

要求：

- preset 优先，不提供工程配置表单；
- 不展示 Hermes config key、env name、`.env` path 或 CLI argv；
- 不提供 base URL/custom endpoint editor；
- 不模仿大型 CC Switch catalog；
- 复用现有 `i18n.ts` type/message、`en-US`/`zh-CN` locale persistence 与 accessibility convention；
- validation/config timestamp 复用 `formatDateTime`；
- server state 使用 TanStack Query 与现有 generated client；
- selection/form/password interaction 仅使用 page-local React state；
- API Key 不得进入 query key/cache、localStorage、sessionStorage、URL、analytics 或 error object；
- submit/unmount 后清空 password input；
- 对 `MISSING`、`UNKNOWN` 或不支持的 installation 禁用 Save/Test；
- status 不得只依赖颜色表达。

## 19. 安全与日志不变量

允许 Hermes Profile 官方 credential persistence 不会削弱以下规则：

1. SQLite 不包含 API Key 明文。
2. HTTP GET 绝不返回 secret；PUT/PATCH 不 echo secret。
3. Logs、events、errors、Operations 与 diagnostics 不包含 secret。
4. Command argv/descriptor 不包含 secret。
5. Secret 不进入 URL、browser storage、TanStack Query cache 或 analytics。
6. 不向 child wholesale 继承 parent provider credential。
7. Credential operation 只针对一个权威 Profile。
8. 不修改 Windows global/user environment。
9. `.env` 不暴露为 file API；只有 Hermes adapter 的获批 bounded credential writer 可检查/更新准确的 allowlisted entry。
10. Unknown output/native state fail closed，不删除或编造 credential state。
11. 保持现有 loopback authentication、Origin/CSP 与 no-store response protection。
12. 现有 process-tree containment 在 success、failure、timeout 与 cancellation 后清理进程。

Structured log 可包含 `instanceId`、preset ID、action、stable outcome code、duration 与 timeout/cancel flag。必须排除 raw/native output、API Key、env name、filesystem path 与 model response。

Redaction 是 defense in depth，覆盖代表性的中国/全球 Provider key、Authorization header、structured field、wrapped error 与 child stderr。

## 20. 实现 Batch

五个有意保持小而连续的 Batch，不增加额外 framework project。

### Batch 1 — Hermes Provider/config/credential 资格验证与 fallback

- 检查 pinned Hermes `0.20.2` 对每个产品候选的实际支持；
- 锁定受支持 ProviderPreset mapping 与推荐模型；
- 证明 Profile selection、非秘密 config key、官方 credential set/delete/status 行为与 `.env` ownership；
- 保留证明官方 offline credential blocker 的首次 STOP 证据；
- 实现并测试狭窄 Profile-scoped、Provider-allowlisted credential writer，包括 status/set/replace/delete、atomic read-back、cleanup 与 optimistic conflict detection；
- 证明 secret 不进入 argv/output，并锁定 tools-disabled validation 接口；
- 建立 contract evidence/fixture，最终确定 error/DTO。

Gate：qualification/fallback test 与 evidence PASS。只有没有中国 preset 能使用获批 bounded fallback，或 validation 无法满足第 17 节时才停止。

### Batch 2 — ProviderPreset 与非秘密模型配置

- 在 Hermes adapter 实现静态 supported catalog；
- 实现 preset/provider/model config 的 read/apply/read-back；
- 增加安全 provider/config GET/PATCH OpenAPI 与 generated client；
- 接入现有 Instance identity、availability 与 coordination。

Gate：adapter/application/HTTP/OpenAPI test PASS；credential HTTP lifecycle 留在 Batch 3。

### Batch 3 — Hermes-native credential lifecycle

- 通过 application 与 HTTP boundary 暴露 qualified adapter Profile credential status/set/replace/delete lifecycle；
- 实现 metadata GET 与 write-only PUT/DELETE；Credential PUT 协调完整 Save；
- 由 Hermes truth 验证 restart recovery 与 Profile-to-Profile isolation；
- 增加 redaction/no-argv/no-SQLite/no-browser-persistence test。

Gate：credential test PASS，且没有 SecretStore duplicate 或获批 Hermes adapter writer 之外的 `.env` access。

### Batch 4 — 显式 validation

- 将 `model.validate` 加入现有 Operation framework；
- 实现 tools-disabled、contained validation、bounded output/deadline 与 cancellation；
- 增加 process-tree cleanup、safe-result 与 restart projection test。

Gate：重复 success/failure/timeout/cancel/output-limit test PASS；没有 surviving child，也没有 secret/model-output leak。

### Batch 5 — Desktop integration 与 audit evidence

- 在现有 Instances 体验中增加中国市场优先 preset UX；
- 完成 English/简体中文、accessibility、local-time 与 password-lifetime test；
- 执行完整验证、适用时的隔离 Windows/real-Hermes smoke 与 exact-commit CI；
- 更新 completion evidence，并停止在 `AUDIT-005 = PENDING`。

Gate：全部 Phase 5 acceptance test 与 exact-candidate full verification PASS。独立审计与 Owner acceptance 前，不得 merge、freeze、tag 或进入 Phase 6。

每个 Batch 均须检查 isolated diff，运行 focused test 与 `git diff --check`，保留用户工作，不得通过削弱合同换取绿色结果。

## 21. 测试策略

| 区域 | 必须证明 |
|---|---|
| Provider qualification | Pinned version 的准确 ID/key/credential name/region 行为；不支持候选不可选。 |
| ProviderPreset | 静态 allowlist；安全 DTO 不含 env/config internals；无 dynamic loading。 |
| Identity/availability | API/Operation 使用 `instanceId`；Hermes call 使用 `nativeId`；MISSING/UNKNOWN 阻止 mutation。 |
| Config | 准确 argv/surface、read-back、manual model bound、partial/incomplete retry、保留无关 config。 |
| Credential | Metadata only；set/replace/delete/restart；Profile A/B isolation；无 argv/direct `.env`/SQLite copy。 |
| Protocol | Auth/origin/CORS、PATCH/PUT method contract、closed body、no echo、method/not-found 与 generated-client parity。 |
| Validation | 仅显式触发、复用 Operation、tools disabled、timeout/cancel/output limit/descendant cleanup。 |
| Redaction | 国内/全球 Key sentinel 不出现在 log/event/error/HTTP/Operation/diagnostics。 |
| Desktop | 复用 sidebar/i18n/date/query；preset-first EN/zh-CN UX；password cleared/not cached。 |
| Environment | Windows user/system env 不变；ambient provider secret 不进入 child。 |
| Integration | 通过 Hermes 官方接口保存，关闭/重启 YORVA 后 Profile 仍 configured，并可 validation。 |

真实 credential smoke 使用隔离的一次性 Hermes home/Profile；fixture/CI 绝不使用 Owner 的真实 Key。不得增加推测性 failpoint framework。

## 22. 完整验证

Batch 5 完成后，运行仓库准确 script/CI equivalent：

- OpenAPI lint/generation 与 generated-client drift；
- Desktop typecheck、lint、test、build 与 dependency audit；
- Go format、full test、affected repeated test、exact CI race、vet、build 与 vulnerability scan；
- Rust format、test、clippy、check 与 dependency audit；
- Windows process containment/timeout/cancel 与 environment-nonmutation smoke；
- 使用 fake key 或 controlled test provider 的隔离 Hermes Profile credential/config restart smoke；
- 仅在 packaging input 变化时执行 Tauri no-bundle/MSI check；
- exact-commit GitHub Actions success。

环境阻断的检查须如实记录，并在可用时由 exact-commit CI 覆盖。绿色 CI 不能覆盖 secret leak 或 required flow failure。

## 23. 审计与退出标准

独立 `AUDIT-005` 必须评审 baseline/scope、ADR compliance、pinned provider evidence、identity separation、official credential surface、Profile isolation、secret scan、API/OpenAPI parity、validation Operation/process cleanup、Desktop password handling、双语 UX、source-file cohesion 与 exact-commit CI。

Critical/High finding、secret leakage、unsafe credential targeting、required-flow failure、unsafe process cleanup 或缺失 mandatory evidence 均使 Gate=`FAIL`。Medium/Low 按 `AUDIT_STANDARD.md` 裁决；推测性 hardening 不自动阻断。

Phase 5 完成要求：

- [x] 确认真实 Phase 4 re-freeze baseline；
- [x] D1–D6 与 credential-authority ADR 获批；
- [ ] pinned Hermes 至少有一个中国市场 ProviderPreset 通过资格验证；
- [ ] 用户可保存 provider/model/key，且看不到 config/env/.env/argv 细节；
- [ ] YORVA 退出后 Hermes 可继续使用同一 Profile 配置；
- [ ] SQLite、HTTP read、log、event、error、argv 与 Desktop persistence 无 Key 明文；
- [ ] 不修改 Windows global/user env；
- [ ] `CONFIGURED` 与 validation result 和 `AVAILABLE` 保持分离；
- [ ] exact candidate 的完整 local/CI verification PASS；
- [ ] 正式审计达到可接受 Gate；
- [x] Owner 授权在审计与 CI PASS 后 merge/freeze/tag。

当前状态为 `IN_PROGRESS`。Batch 1-5 及其后的审计/CI/合并/冻结/tag/Windows build 序列均已授权，但每个既定 Gate 必须在下一不可逆步骤前 PASS。

## 24. 强制停止条件

遇到以下情况停止并汇报：

1. Phase 4 未在预期 start baseline 上真实重新冻结；
2. pinned Hermes 不安全支持任何中国市场候选；
3. 获批狭窄 credential writer 无法在不泄密的前提下安全定位 canonical Profile store；
4. 安全 metadata status/delete 或 tools-disabled validation 不可用；
5. 需要 custom endpoint/provider protocol、OAuth/login 或 lifecycle；
6. 看似必须建立第二套 navigation/i18n/runner/registry/Operation/date/state system；
7. 实现必须修改已冻结 Phase 3/4 行为，而非扩展；
8. 需要重大 dependency/framework 或未批准 ADR；
9. 无法保证 process cleanup、Profile isolation 或 secret non-leakage；
10. 中英文合同分歧或仍有未决产品决策。

当前 entry gate 已批准 Batch 1-5 与规定的收口序列。不得修改 Phase 4 或进入 Phase 6；在 Phase 5 审计与 exact-commit CI Gate PASS 前不得 merge/tag。

# YORVA Windows 产品支持合同

> 状态：Phase 8 B0 已批准基线
> 生效日期：2026-09-03
> 适用范围：首个 Windows 本地 MVP

本文冻结 Phase 8 后续安装、迁移、恢复、更新、诊断和发布工作的共同产品假设。
具体功能仍须通过各 Batch Gate；本文本身不表示公开发布已经就绪。
用户可执行的恢复步骤见 `RECOVERY.zh-CN.md`。

## 1. 支持矩阵

| 项目 | P8 阻断支持范围 | 说明 |
| --- | --- | --- |
| 操作系统 | Windows 10 22H2 x64、Windows 11 x64 | Windows 是唯一阻断发布目标；Windows 10 主机必须仍处于 Microsoft 支持期或已加入 ESU；macOS/Linux 仅作非阻断记录。 |
| 安装方式 | 当前用户范围 MSI | 正常安装和运行不得要求 YORVA 永久以管理员身份运行。 |
| WebView | Microsoft Edge WebView2 Evergreen | 安装器可使用官方 bootstrapper；离线环境必须明确报告缺失，不能显示假成功。 |
| CPU | x86-64 | ARM64 不属于 P8 正式支持范围。 |
| 内存 | 4 GiB 最低，8 GiB 建议 | Hermes 实际负载可能需要更多资源。 |
| 可用磁盘 | 安装与更新 staging 至少 2 GiB | Runtime、Profile、Backup 和日志增长需要额外空间。 |
| YORVA 数据库 | 空库、Phase 7 Schema 016 | 其他历史开发 Schema 不作兼容承诺。 |
| Hermes | 以当前 Runtime 合同和精确资格证据为准 | P8 不扩大 Hermes 兼容区间，也不恢复延期的 Managed Upgrade/Rollback。 |

## 2. 产品身份与版本

- 正式产品名：`YORVA`。
- 正式桌面 identifier：`com.yorva.desktop`。
- Phase 7 开发 identifier：`com.yorva.desktop.dev`，只作为一次性迁移来源。
- 可发布版本使用 `MAJOR.MINOR.PATCH`；预发布候选可追加 SemVer 预发布后缀。
- Desktop package、Tauri bundle、Rust crate、`yorvad` 打包版本和 MSI ProductVersion
  必须来自同一个候选版本并在构建 Gate 中核对。
- 仓库根目录的私有 workspace package 版本不是产品版本。
- `0.3.2` 是 P7/P8 支持的升级输入；B3 已冻结 `0.4.0` 为 P8 内部候选版本，不能只修改 UI 文案。

P8-B1 已实现并验证下述一次性迁移，正式 identifier `com.yorva.desktop` 已激活：

1. 在 daemon 启动和任何数据库写入前检查新旧目录；
2. 新目录为空且旧目录有效时，先建立可验证的保护副本，再向同卷 staging 迁移；
3. 校验数据库、固定目录和迁移标记后原子发布新目录；
4. 保留旧目录作为回退来源，不自动删除；
5. 新旧目录都含未确认数据时停止自动合并，进入稳定的恢复状态；
6. 成功标记必须记录来源 identifier、目标 identifier、时间和非秘密校验信息。

旧目录在迁移后继续保留为回退来源；YORVA 不自动删除它。新旧目录都包含未确认数据
时，Desktop 以 `PRODUCT_DATA_IDENTITY_CONFLICT` 停止启动，不做猜测性合并。

## 3. Windows 目录和所有权

正式 identifier 激活后，YORVA 使用以下固定类别。实现必须通过 OS/Tauri 目录解析，
不得依赖用户输入的任意路径。

| 位置 | 所有者 | 内容 | 卸载默认行为 |
| --- | --- | --- | --- |
| `%APPDATA%\com.yorva.desktop\` | YORVA | `yorva.db`、桌面设置、管理状态、Skills、日志 | 保留 |
| `%APPDATA%\com.yorva.desktop\backups\` | YORVA | 已验证的加密 Runtime Backup | 保留 |
| `%APPDATA%\com.yorva.desktop\backup-staging\` | YORVA | 有界临时备份文件 | 成功或失败后清理；卸载可清理临时内容 |
| `%APPDATA%\com.yorva.desktop\update-staging\` | YORVA | 已下载、待验证的 YORVA 安装包 | 更新完成/失败后按有界保留策略清理 |
| 用户通过 Save As 选择的位置 | 用户 | 已脱敏诊断包 | 永不由卸载器删除 |
| `%LOCALAPPDATA%\hermes\` 及 Hermes Profile | Hermes/用户 | Runtime、Profile、会话和 Runtime 原生状态 | YORVA 卸载不得删除 |

下载缓存、临时目录和日志必须设置数量、大小或时间边界。B1、B4、B5 分别冻结其精确
保留值。YORVA 不把 `%TEMP%`、任意用户目录或 Hermes 目录当作自己的清理根目录。

## 4. 安装、修复和卸载的数据规则

- Fresh Install 只创建 YORVA 程序文件和所需的 YORVA 数据目录。
- Upgrade/Repair 可以替换程序文件，但不能覆盖或删除用户数据库、Backup、Hermes
  Runtime 或 Profile。
- 卸载默认删除程序文件、快捷方式、登录启动项和 YORVA 创建的临时 staging；保留
  YORVA 用户数据、加密 Backup 以及全部 Hermes 数据。
- 卸载界面和支持文档必须明确告知“保留数据”。P8 不提供静默“删除所有数据”。
- Reinstall 必须发现保留数据，并在迁移/reconcile 成功后才显示恢复成功。

## 5. 更新与发布可信度

- P8 只实现来自固定 YORVA 发布源的完整安装包更新，不实现增量补丁或后台服务。
- 发布元数据必须固定描述版本、包长度、SHA-256、签名状态和允许的下载地址。
- 执行安装包前必须依次验证来源、长度、SHA-256、版本以及 Windows 签名策略。
- 当前 Owner 明确没有 Windows 代码签名材料。因此可以构建和验证内部候选，但不得
  标记为“公开发布就绪”，也不得用自签名或跳过验证伪造 Gate PASS。
- 取得正式签名材料后，需要在精确候选 MSI 上补做签名、来源、篡改拒绝和安装验证。

## 6. 隐私与支持

- P8 不收集遥测，不上传使用情况、崩溃信息、Runtime 状态或诊断包。
- 诊断导出只能由用户主动触发；内容采用固定投影并在本机脱敏。
- 诊断包或日志不得包含 API Key、Token、Channel/MCP 凭据、QR/配对值、Cookie、
  Authorization、环境变量值、原始数据库或任意文件。
- P8 最低稳定性测试为 4 小时，正式候选目标为 8 小时，使用三个 Hermes Instance。

## 7. 发布分类

| 分类 | 条件 |
| --- | --- |
| 开发构建 | 可用于本地开发，不构成安装、迁移或升级证据。 |
| 内部候选 | B0–B6 功能和测试通过，但缺少正式 Windows 签名材料。 |
| 公开发布就绪 | B0–B6、独立审计、精确候选 CI、真实 MSI 生命周期、正式签名和 Owner Gate 全部通过。 |

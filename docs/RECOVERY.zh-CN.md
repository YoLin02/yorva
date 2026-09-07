# YORVA Windows 恢复指南

> 适用于 Phase 8 Windows 本地 MVP，并以 `PRODUCT_SUPPORT.zh-CN.md` 为支持合同。

恢复应首先采用可回退的进程级操作。Runtime 检测失败、Hermes Profile 停止或 daemon
断开时，不应默认重启 Windows。不要为了消除报错而手工删除、合并或修改 YORVA/Hermes
数据。

## 1. 首先这样处理

1. 记录界面显示的稳定错误码和发生时间。
2. 如果 YORVA 仍可操作，打开 **设置 → 诊断与支持 → 导出诊断信息**，保存已脱敏
   诊断包。
3. 关闭 YORVA，等待 Desktop 和 `yorvad` 进程退出后重新打开。Desktop 负责 daemon
   生命周期，并会连接新的认证会话；用户不需要手工启动 `yorvad`。
4. 刷新 Runtime 清单。受管 Hermes Profile 停止时，可从对应实例入口启动；Runtime
   检测超时本身不表示数据丢失。
5. 如果相同错误仍存在，按下文对应章节处理。在 Repair 或 Reinstall 前保留数据目录
   和诊断包。

## 2. 产品数据冲突

`PRODUCT_DATA_IDENTITY_CONFLICT` 表示旧目录
`%APPDATA%\com.yorva.desktop.dev\` 与正式目录
`%APPDATA%\com.yorva.desktop\` 同时含有 YORVA 无法安全自动合并的数据。

- 关闭 YORVA，并保持两个目录原样。
- 不要互相覆盖、合并或删除任一数据库。
- 检查当前用户对两个目录的读取权限，以及目标磁盘的剩余空间。
- 保留两个目录和诊断包，供支持流程确认权威数据源。在明确解决前，YORVA 会继续
  阻止自动启动，而不是猜测合并。

## 3. 数据库迁移需要人工恢复

`DATABASE_MIGRATION_RECOVERY_REQUIRED` 表示迁移保护记录缺失、格式错误，或与保护
数据库不匹配。

- 关闭 YORVA，保留完整的正式产品数据目录，包括数据库、迁移状态和保护文件。
- 不要手工执行 SQL、重命名保护数据库或删除恢复状态。
- 可以用相同版本重新打开一次；如果状态不变，应停止操作并保留目录，由支持流程
  协助恢复。YORVA 不会在证据不完整时猜测处理。

如果显示 `DATABASE_MIGRATION_FAILED_RECOVERED`，表示受支持的来源数据库已经恢复。
修复磁盘空间或权限问题后再重试，并继续保留恢复证据。

## 4. Runtime 或 daemon 不可用

- 首先使用 YORVA 中的**重试/刷新**。Desktop 最多自动替换一次意外退出的 daemon，
  此后会明确显示失败，避免无限重启循环。
- Hermes Runtime 已安装但 Profile 停止时，从 YORVA 的实例入口启动。
- 检查磁盘剩余空间，以及当前用户对 YORVA 和 Hermes 固定目录的访问权限。
- 多次超时或重连失败时导出诊断包。不要手工启动第二个 daemon，也不要让 YORVA
  长期以管理员身份运行。

## 5. 更新中断

- 重新打开已安装的 YORVA。若下载因进程退出而中断，会转为可重试的下载失败并保留候选；
  点击“下载并验证”重新下载。恢复只清理更新器固定的 partial/package 文件。
- 安装后会校验实际 daemon 版本及实时 Runtime/Instance 回读。daemon 可连接但回读失败
  会报告 UPDATE_POSTCHECK_FAILED；可从管理与诊断页面调查原因，不能视为更新成功。
- 不要执行 `.partial` 文件或从 update staging 手工复制出的安装包。
- 若报告元数据、完整性、签名、安装器或 postcheck 失败，请保留稳定错误码和诊断包，
  只从 YORVA 更新页面重试。
- 未签名的内部候选不属于公开发布版本，不能通过跳过签名检查来升级发布等级。

## 6. Runtime Backup 与 Restore

- 受管加密 Runtime Backup 固定存放在
  `%APPDATA%\com.yorva.desktop\backups\`，不要求用户选择受管保存位置。
- Backup 或 Restore 前，按界面提示停止全部 Hermes 实例，并确保磁盘空间充足。
- Backup 失败时不能把该文件当作保护点，只能使用列表中已成功校验的 Backup。
- Restore 如果提示需要人工恢复，不要删除事务目录，也不要手工替换 Hermes 文件。
  保留 YORVA 产品数据目录和 Hermes 根目录，由支持流程协助恢复。

## 7. Repair、Reinstall 与 Uninstall

Repair、Upgrade、Uninstall、Reinstall 默认保留 YORVA 用户数据、加密 Backup，以及
Hermes Runtime/Profile 数据。因此 Reinstall 只能修复程序文件，不能用于清除数据冲突。
除非未来提供明确说明删除范围的数据清理流程，否则不要删除这些保留目录。

## 8. 提交支持信息

应保留：

- 已脱敏诊断 ZIP；
- YORVA 精确版本和稳定错误码；
- 故障发生时间；
- 故障是否发生在安装、迁移、更新、Backup、Restore 或 daemon 重连之后。

不要通过普通支持渠道发送 API Key、Channel/MCP 凭据、QR/配对值、原始数据库、原始
Hermes 配置或整个数据目录。

# Phase 7 Batch 1 — MCP / Backup / Upgrade Surface Qualification

## 结果

**PARTIAL B1 RESULT: STOP / NO-GO FOR PRODUCT IMPLEMENTATION**

本证据只覆盖 Hermes `0.20.5` 的 MCP、backup/import 和 update/check/plan。
它不批准 B2–B8 产品能力代码，也不代表整个 B1 已通过。

结论：上游命令入口确实存在，但没有一个可以按当前形态直接包装成 YORVA 的完整
产品 surface：

- MCP 的只读和 test 路径具有部分可复用的超时/cleanup 事实，但 CLI 没有结构化结果，
  多个 mutation/auth 路径是交互式的，官方 catalog installer 还会执行 manifest 中的
  shell bootstrap；MCP credential authority 需要单独 ADR；
- full/quick backup 会产生包含 `.env`、auth、OAuth token、session/account/data 的明文
  artifact；import 没有满足 P7 的全量 preflight、archive bounds、保护点和 rollback truth；
- `update --check` 会修改 Git metadata，`update --plan` 只有可能不完整的人类输出，普通
  `hermes update` 会原地修改 checkout/venv；它们均不能替代 ADR-0006/0009 的新
  generation upgrade。

依据 Phase 7 强制停止条件，B5、B6/B7 和 B8 在相应 ADR、closed YORVA-owned design
和后续资格证据接受前保持阻塞。不得通过调用 Hermes Python internal modules、开放任意
command/env/header/path、使用 `--force`，或把明文 ZIP 改称“安全备份”来继续。

## 候选与取证方法

- P7 branch：`codex/phase7-hermes-runtime-management`
- P7 HEAD at inspection：`30f322529339f17bbd1f337d0e9ee8afbefb0b67`
- predecessor：`phase-0065-developer-led-demo-baseline` →
  `5f68e48f17e7e342e1781b37613b19d4bd1f060b`
- official source：Hermes `0.20.5`, commit
  `a0ca7c19204e514f9590ce3b812e029b315ab9e9`
- inspected packaged archive：
  `apps/desktop/src-tauri/resources/hermes/source/hermes-agent-a0ca7c19204e514f9590ce3b812e029b315ab9e9.zip`
- archive size：`73,798,347` bytes
- archive SHA-256：
  `4E63F39BD992B79BF364A4E22204A22B8E520E2365B53C909F6F15D42C917C54`

方法仅为 `Get-FileHash`、ZIP member listing 和 `tar -xOf` 静态读取。没有执行
backup、import、update、MCP connect/login/install/auth，没有启动浏览器或网络流程，
也没有修改 Hermes、Profile、Git checkout 或用户数据。未执行真实 mutation 的项目
明确记录为“未证明”，不推测成功。

## 判定口径

- `GO`：当前 exact surface 已证明可按 P7 边界直接使用；
- `CONDITIONAL`：存在可用事实，但仍需 closed design、ADR 或进一步动态资格确认；
- `NO-GO`：当前 surface 违反 Phase 7 的明确边界，不能原样接入。

## MCP

### 官方入口

`hermes_cli/subcommands/mcp.py` 定义：

```text
hermes [--profile <id>] mcp list
hermes [--profile <id>] mcp catalog
hermes [--profile <id>] mcp add <name> [--url|--command|--preset] ...
hermes [--profile <id>] mcp install <catalog-id>
hermes [--profile <id>] mcp test <name>
hermes [--profile <id>] mcp configure <name>
hermes [--profile <id>] mcp login <name>
hermes [--profile <id>] mcp reauth <name>|--all
hermes [--profile <id>] mcp remove <name>
```

全局 `--profile` 在 `hermes_cli/main.py:512-694` 先解析为 Profile-specific
`HERMES_HOME`。这证明配置和 credential 文件可以具备 Profile scope；不证明每个命令
满足非交互、输出和失败语义。

### MCP surface matrix

| Surface | 已证明事实 | 未满足条件 | 判定 |
| --- | --- | --- | --- |
| `mcp list` | 从 active Profile `config.yaml` 读取 configured inventory；无需 prompt | 只有彩色人类表格，无 JSON；条目数量无 YORVA 上限；输出包含 URL/stdio command 摘要；异常/空结果没有 typed schema | **NO-GO as direct product surface** |
| `mcp catalog` | 可列出 packaged repository manifests | 只有人类输出；不是版本化 machine contract；直接 import `hermes_cli.mcp_catalog` 被 Runtime boundary 禁止 | **CONDITIONAL only as design evidence** |
| `mcp test` | `_probe_single_server` 有最少 1 秒、默认 30 秒 connect timeout；外层为 `timeout + 10`；`finally` 调用 server shutdown 和 idle-loop cleanup | CLI 无 JSON；工具数量无上限；失败被打印后普通 `return`，没有可靠非零 exit；会输出 tool name/description；header 状态会显示 credential 首尾片段；没有 YORVA cancellation/process ownership 证明 | **NO-GO as direct product surface** |
| `mcp add` | 支持 URL、stdio、preset、header/OAuth 和 tool discovery | 会 prompt overwrite/auth/token/save-anyway/tool selection；API key 写入 `.env`；接受 caller command/args/env；不符合 closed preset/HTTPS schema | **NO-GO** |
| `mcp install` | identifier 可指定 packaged catalog entry | install flow 会 prompt credential/tool selection；git installer 可执行 manifest `bootstrap`，且 `_run_bootstrap` 使用 `subprocess.run(..., shell=True)`；不是 YORVA fixed reviewed descriptor | **NO-GO** |
| `mcp configure` | 能 read-back tools 并保存 selection | 明确要求 TTY/curses；无 closed non-interactive selection body；无机器结果 | **NO-GO** |
| `mcp remove` | 删除 exact name 的 config，尝试清除 OAuth state | 有 confirmation prompt；OAuth cleanup exception 被忽略；无原子 postcondition/typed result | **NO-GO** |
| `mcp login/reauth` | OAuth callback 有 300 秒窗口；single-server login 把 connect timeout 提高到至少 315 秒；token file existence 被用于避免“probe success = auth success” | 故意允许 spawned non-TTY 打开浏览器；未绑定 YORVA initiating session；会先删除现有 token state；无结构化结果；没有 YORVA cancellation/daemon-restart recovery；authorization URL/code 不能进入普通输出证据 | **NO-GO** |

关键源码证据：

- `hermes_cli/mcp_config.py:278-401`：probe timeout、server shutdown 和 loop cleanup；
- `hermes_cli/mcp_config.py:746-805`：test 的人类输出、credential partial masking、
  exception 后普通返回；
- `hermes_cli/mcp_config.py:438-640`：add 的全部 prompt 与 config mutation；
- `hermes_cli/mcp_config.py:810-907`：OAuth token presence post-check 与 315 秒交互窗口；
- `hermes_cli/mcp_config.py:987-1091`：configure 强制 TTY；
- `hermes_cli/mcp_catalog.py:445-456`：catalog bootstrap 通过 shell 执行；
- `hermes_cli/mcp_catalog.py:460-523`：Git clone/checkout/bootstrap mutation；
- `hermes_cli/mcp_catalog.py:776-863`：install credential prompt、probe、tool selection；
- `hermes_cli/mcp_security.py:121-180`：只拦截 high-signal suspicious entry，
  不是 YORVA 对 fixed descriptor 的 allowlist 证明。

### Credential authority facts

官方 `0.20.5` 有两类 Hermes-native Profile credential store：

1. HTTP bearer/API key 由 `hermes_cli/mcp_config.py:174-196` 写入 active Profile
   `.env` 的派生 key，`config.yaml` 只保存 `${ENV}` header template；
2. OAuth access/refresh token、client registration（可含 `client_secret`）和 OAuth
   metadata 由 `tools/mcp_oauth.py:456-580` 写入
   `HERMES_HOME/mcp-tokens/<server>.json|client.json|meta.json`。

OAuth JSON 采用 temp + replace，并在 POSIX 上请求 `0600`；源码明确说明 Windows 不
依赖 POSIX mode bits。它仍是 Hermes-native plaintext Profile storage，不是 YORVA
OS-backed SecretStore。`poison_client_registration` 还会创建一份 `client.json.bak`；
该文件可含 client secret，必须纳入 backup/redaction/cleanup 威胁模型。

因此 unique authority 可以被定义为“按 credential 类型由 exact Hermes Profile 的
native store 唯一持有”，但这属于安全/所有权决策，不能由本证据自行批准。

### MCP ADR 必需决定

MCP credential authority ADR 至少锁定：

- bearer/API key、OAuth token、OAuth client secret 各自唯一 authority；
- YORVA 是否接受 Hermes-native plaintext-at-rest tradeoff，以及 Windows local-user
  protection 的真实表述；
- YORVA 不复制 secret 到 SecretStore/SQLite/Operation/event/log/audit/Desktop；
- closed approved preset/HTTPS host、redirect 和 tool-selection schema；
- authorization URL/code/token 的 initiating-session-only memory lifecycle；
- reauth 是否允许先删除旧 token，以及失败 rollback/concurrent writer 语义；
- removal read-back、external Hermes change reconciliation、timeout/cancel/daemon
  restart cleanup；
- 明确禁止使用 upstream arbitrary stdio command/env/header 和 shell bootstrap
  surface。

在该 ADR 接受并另行证明 closed adapter path 前，Phase 7 MCP capability 必须保持
`false` / `NO-GO`。不能通过 import 官方 Python internal module 获得内部 dict。

## Backup / Import

### 官方入口与真实 scope

`hermes_cli/subcommands/backup.py` 定义：

```text
hermes backup [-o <path>]
hermes backup --quick [--label <text>]
```

`hermes_cli/subcommands/import_cmd.py` 定义：

```text
hermes import <zipfile> [--force]
```

full backup 和 import 使用 `get_default_hermes_root()`，不是当前 selected Profile 的
`get_hermes_home()`；其有效 scope 是整个 Hermes root，并可包含 named Profiles。
full backup 还会收集 memory provider 声明的、用户 home 下但 Hermes root 外的文件。
quick snapshot 使用 selected Profile 的 `get_hermes_home()`。

### Backup surface matrix

| Surface | 已证明事实 | 阻塞事实 | 判定 |
| --- | --- | --- | --- |
| full backup | 有 cross-process single-flight lock；ZIP 通过 hidden sibling partial + replace 发布；SQLite 尝试一致 snapshot；会跳过 symlink | 普通 ZIP 无加密；包含 `.env`、`auth.json`、`mcp-tokens`、sessions/data/account state；没有 format/runtime version/checksum manifest；文件数/总量/时长无上限；个别文件失败仍发布 `Backup incomplete` 且正常返回；输出为人类文本 | **NO-GO** |
| quick backup | Profile scoped；staging dir 后 rename；manifest 记录文件名/size | 明文目录明确包含 `.env`、`auth.json`、pairing、channel、response/history 和多个 DB；manifest 无 checksum；默认单文件无 size cap；失败可产生 incomplete snapshot 或普通返回 | **NO-GO as product backup** |
| import | 验证是 ZIP，做基本 root containment，单文件通过 sibling temp replace；部分 secret basename chmod；跳过部分 volatile runtime files | 只用 `config.yaml/.env/state.db` 任一 marker 判断“像 backup”；无 checksum/signature/format/version/scope/space preflight；无 member count、单 member、total expansion、compression ratio、ADS/reparse/symlink-type 全量拒绝；允许 `_external/` 写入用户 home；不是 all-members-first validation；出错后继续并打印 `Import complete`；无 protection point/rollback；结尾可能自动安装/启动 gateway；`--force` 只跳过确认，不增加安全性 | **NO-GO** |

关键源码证据：

- `hermes_cli/backup.py:642-865`：full backup root scope、unbounded walk、明文 ZIP、
  partial-error publish 和人类 summary；
- `hermes_cli/backup.py:870-918`：import 只检查少量 marker；
- `hermes_cli/backup.py:1045-1279`：import prompt/force、逐 member mutation、
  partial error、`_external/` restore 和 gateway side effect；
- `hermes_cli/backup.py:1286-1324`：quick snapshot 明确包含 `.env`、auth、pairing、
  channel、conversation/history DB；
- `hermes_cli/backup.py:1335-1576`：quick snapshot 的 size-only manifest 与
  incomplete semantics。

### Backup Secret/encryption authority blocker

当前上游格式没有 encryption envelope、key id、KDF/AEAD parameters 或 secret-exclusion
manifest。YORVA 不能把 ZIP 文件权限或“存于本地”当作加密，也不能把包含 credentials
的 plaintext staging/final artifact 视为合格备份。

P7-D4 ADR 至少必须决定：

- Runtime-scope 为首版 authoritative scope，以及现有 Instance-scoped `backups` schema
  的非破坏 migration；
- 标准、维护中的 authenticated-encryption container/library（不设计自有 crypto）；
- backup key/password 的唯一 authority、OS-backed key reference、rotation/recovery；
- key/password 只在 write-only request/session memory 中存在，不进入 argv、URL、
  SQLite plaintext、Operation、event、log、audit、diagnostic 或 Desktop storage；
- plaintext staging 的 exact location、ACL、bounds、crash/daemon-restart cleanup；
- format/version/scope/member/checksum/authentication metadata；
- all-members-first Restore preflight、member/size/ratio/path/reparse/symlink/ADS bounds；
- running Instance stop/conflict plan、pre-Restore protection point、postcondition、
  rollback 和 `UNKNOWN`；
- full backup 是否包含 sessions/accounts/OAuth/client secret，以及 UI disclosure。

没有 accepted ADR 和 destructive-flow qualification 时，B6/B7 必须停止。上游
`hermes import --force` 明确不是 YORVA Restore 实现捷径。

## Update / Check / Plan / Managed Generation Upgrade

### 官方入口

`hermes_cli/subcommands/update.py` 定义：

```text
hermes update --check [--branch <name>]
hermes update --plan
hermes update [--yes] [--backup|--no-backup] [--keep-stash]
              [--branch <name>] [--switch-branch] [--force] [--force-venv]
```

### Update surface matrix

| Surface | 已证明事实 | 阻塞事实 | 判定 |
| --- | --- | --- | --- |
| `update --check` | 不安装 dependency/code；会比较 local HEAD 与 remote branch | 并非无状态 read：会清 stale Git lock、执行 `git fetch` 并修改 refs；网络/git subprocess 无明确 timeout/output bound；允许 branch；输出为人类文本；目标是 moving Git remote，不是本 YORVA build 的 exact verified snapshot | **NO-GO for YORVA plan** |
| `update --plan` | 本地 inventory collector 被设计为 read-only；可列 install method、Profiles、running services、supervisor、current code identity/restart method | CLI 只有人类文本；包含 PID/Profile；Profiles/runtimes 数量无输出上限；collector 把 probe exception 降级为缺少 rows，结果没有 completeness/partial flag；不是“current managed generation vs exact packaged target”的 plan | **CONDITIONAL as operator-only evidence; NO-GO as authoritative plan** |
| ordinary `update` | 有 update lock、pre-update snapshot、fleet restart bookkeeping 和部分 post-update receipt | 原地修改 Git checkout/venv/dependencies，可 stash/switch/reset user checkout，使用含 secrets 的 plaintext snapshot/ZIP，允许 `--force-venv`；违反 ADR-0006/0009 和 P7-D5 | **NO-GO** |

关键源码证据：

- `hermes_cli/main.py:10313-10356`：managed refusal、plan/check dispatch；
- `hermes_cli/update_cmd.py:3252-3438`：check 的 stale-lock cleanup、Git fetch 和
  human-only result；
- `hermes_cli/update_inventory.py:128-295`：plan collector 独立吞掉 probe failures；
- `hermes_cli/update_inventory.py:298-327`：plan 的人类输出；
- `hermes_cli/update_cmd.py:5738+`：普通 update 的 checkout/venv、backup、gateway
  mutation path；
- `hermes_cli/config_defaults.py:3758-3759`：当前 config schema version `38`；
- `hermes_cli/config_migrations.py`：只存在向前 migration registry，没有 downgrade
  contract。

### Generation upgrade ADR 必需决定

P7 managed upgrade 必须是 YORVA-owned design，不能调用普通 `hermes update`。所需 ADR
至少锁定：

- target 只来自当前 YORVA build 的 exact commit/archive size/SHA-256/license/pinned
  installer inputs，不读取 moving branch；
- 复用 ADR-0006/0009：新 transaction、新 final-path generation、functional probe、
  seal、publish、`active.json` CAS；active generation 永不原地写；
- upgrade 对 `VALID active.json` 的 precondition、install/upgrade/restore/Instance/
  Skill/MCP 冲突和 narrow lock；
- 备份 Gate 与 B6/B7 的加密 design 关系；
- 激活后的 Instance/lifecycle/model/channel/Skill/MCP exact postchecks，以及任何一个
  `UNKNOWN` 时的真实 terminal state；
- daemon crash/restart 的 observe-decide-execute recovery，SQLite Operation 只做 projection；
- previous generation retention 与 rollback activation CAS；
- user-data/config/schema 的 exact from/to compatibility Gate。

当前源码只证明 `0.20.5` 配置有 version `38` 和向前 migrations；没有证明由新 code
使用过的 user data 可由所有 previous generation 安全读取。这个证据不足不等于已证明
不兼容，但它禁止承诺自动 rollback。B8 必须先建立 exact from/to data compatibility
matrix 和 disposable-state destructive Windows smoke；未证明时只能保留 previous
generation 并给出 manual recovery，不能自动激活回退后宣称成功。

## Timeout、Cancellation、Concurrency 与 Postcondition 总结

| 领域 | 已有上游事实 | P7 尚缺 |
| --- | --- | --- |
| MCP test | connect timeout、outer timeout、server shutdown、loop cleanup | request context cancellation、bounded tool count/bytes、stable exit/schema、daemon restart recovery |
| MCP OAuth | callback timeout、per-server storage、部分 concurrent flow isolation | YORVA session isolation、cancel cleanup、old-token rollback、typed terminal result |
| Backup | full/quick shared cross-process lock、partial publish cleanup | context cancellation、file/member/byte/time budget、checksum/encryption、complete-only success |
| Import | per-file atomic replacement的部分保障 | pre-mutation whole-archive proof、global conflict lock、protection point、transaction rollback、postcondition |
| Update plan/check | plan 本地读取；apply 有 update lock | exact packaged target、bounded process/network、partial evidence flag、managed generation CAS |

## ADR / Gate 清单

| 决策 | 当前状态 | Gate effect |
| --- | --- | --- |
| MCP credential authority | **REQUIRED / NOT ACCEPTED IN THIS EVIDENCE** | B5 blocked |
| Backup format, encryption and key authority | **REQUIRED / NOT ACCEPTED IN THIS EVIDENCE** | B6/B7 blocked |
| Runtime-scope backup schema/migration | **REQUIRED / UNDECIDED** | B2 migration and B6 blocked |
| Managed generation upgrade/recovery/rollback | **REQUIRED / NOT ACCEPTED IN THIS EVIDENCE** | B8 blocked |
| Exact user-data downgrade compatibility | **NOT PROVEN** | automatic rollback blocked |

## B1 子项 Gate 建议

**FAIL / STOP for these surfaces as currently exposed.**

这不是因为 Hermes 功能不存在，而是因为当前官方 CLI contract 未同时满足 P7 要求的
non-interactive、scope-exact、structured/bounded output、Secret authority、stable
failure/cancel 和 authoritative postcondition。

允许的下一步仅为：

1. 起草并接受三个必要 ADR；
2. 设计 closed YORVA-owned adapter contracts，不 import Hermes Python internals；
3. 对每个 closed descriptor/parser/encryption/recovery 路径建立 fixture 和动态资格证据；
4. B1 全部证据经 Owner Gate 通过后才进入 B2。

不允许开始 MCP/backup/restore/upgrade 产品 mutation，也不允许把本报告改成 PASS 来绕过
强制停止条件。

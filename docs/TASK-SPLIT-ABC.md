# A/B/C 任务拆分（对照第一版设计 × 当前实现）

| 项 | 内容 |
|---|---|
| 文档版本 | v1.2（含 P0.5 契约冻结清单） |
| 日期 | 2026-10-25 |
| 基线代码 | `main` HEAD（含 dsh、SSE、联调修复）；拉取后以 `git rev-parse HEAD` 为准 |
| 对照 | [DESIGN](DESIGN.md)、[PAGE-DESIGN](PAGE-DESIGN.md)、[MVP-2D3P](MVP-2D3P.md)、[ACCEPTANCE](ACCEPTANCE.md)、[DECISIONS](DECISIONS.md)、[MANUAL-INTEGRATION-TEST](MANUAL-INTEGRATION-TEST.md)、[CONTRACT](CONTRACT.md)、[MODULE-DESIGN](MODULE-DESIGN.md) |

**读法**：先 **P0**；穿插 **P0.5 契约冻结**（不挡 I1，但挡 P1 分头开发）；再 **P1**。每张任务卡含：目标、步骤、完成标准（DoD）、证据、依赖、关联用例。P0 未闭环前，P1 页面不算产品交付。

**证据模板**（写入 ACCEPTANCE / 联调记录，截图前去凭据）：

```text
编号 / 日期 / 操作人(A|B|C) / commit / OS·Go·Wails·Agent 版本
模式：真实绑定 | 模拟
前置 / 操作 / 预期 / 实际（sessionId、state、cleanup、错误码）
状态：待执行 | 通过 | 失败 | 阻断
证据路径（脱敏）/ 下一步
```

---

## 0. 两层目标

| 层 | 含义 | 完成判据 |
|---|---|---|
| **P0 两天核心** | 桌面真连 → 隔离 Agent → 模型+认证 MCP → 停止清理 | D 项齐 + **I1–I11 真实证据** |
| **P1 第一版产品** | HOME/向导/模型/工具/设置/扩权/PTY/多会话 | PAGE-DESIGN + DESIGN §11.1；先改 CONTRACT |

现状：P0 工程壳与 mock/联调已有，**真机 I 未齐**；P1 **整站 IA 基本未做**。

当前桌面边界（细化任务时不得假设已有）：单会话、direct 工作区、配置/工作台两页、无 PTY、Claude=`demo-model` / dsh=`dsh-model` / MCP=`demo-mcp`、无 OAuth/clone/多会话入口。详见手册 §2。

---

## 1. 已完成（勿重复立项）

| 角色 | 已交付 |
|---|---|
| A | D2；严格单进程 Runtime/Claude Kit（探针）；dsh Kit 代码；测试 Gate 编排集成 |
| B | Gate/Policy/Secrets；Wails v3；bindings 生成；live/`harbor:event`；MCP SSE；profile 环境变量机制；联调准入修复 |
| C | 配置+工作台；mock D4；demos/工作区/示例 Key；Agent 下拉；`live-api`；前端单测 |

---

## 2. P0 — 两天必过（优先）

### 2.1 环境与工具链（阻塞 I1+）

#### P0-ENV-1 · 登记真实模型上游 · B（A 核对）

| | |
|---|---|
| **目标** | 固定 Anthropic 兼容 `/v1/messages` 上游，无占位默认 |
| **步骤** | 1）选定 HTTPS origin（仅 scheme+host[:port]） 2）设定 `HARBOR_CLAUDE_MODEL` 3）准备可轮换测试 Key（不入库） 4）写入 [DECISIONS](DECISIONS.md) 5）用受控探测或桌面「连接模型」验证 |
| **DoD** | DECISIONS 有 origin+模型名；未设 env 时连接失败；设齐后 I1 可开始 |
| **证据** | DECISIONS  diff；脱敏连接成功截图（仅账号 id/host） |
| **关联** | 手册 P01–P04、M06；ACCEPTANCE I1 |

#### P0-ENV-2 · 选定真实只读 MCP · B/C

| | |
|---|---|
| **目标** | 一个 Bearer + Streamable HTTP `/mcp`（2025-03-26）服务 |
| **步骤** | 1）C 协助选只读工具场景 2）登记 `HARBOR_DEMO_MCP_ORIGIN` 3）`HARBOR_DEMO_MCP_TOOLS` 逗号白名单（真实工具名，勿写死 search） 4）测试 Token 仅本机 5）更新 demos 任务文案中的工具名 |
| **DoD** | tools 非空且 ValidConnection；`tools/list`+一次 `tools/call` 可在真连中出现 |
| **证据** | DECISIONS；白名单列表；脱敏审计一行 |
| **关联** | I1/I2；[DSH-INTEGRATION](DSH-INTEGRATION.md) 若走 dsh |

#### P0-ENV-3 · Agent 本机 Detect 冻结 · A

| | |
|---|---|
| **目标** | 目标机 Agent 版本/路径可复现 |
| **步骤** | 1）记录 OS/CPU 2）Claude：确认 `--version` 为 **2.1.292 (Claude Code)** 或记阻断 3）若用 dsh：按文档 Node+版本 4）`HARBOR_CLAUDE_BIN` / dsh 安装路径 5）桌面「重新检测」截图 Available=true/false+reason |
| **DoD** | DECISIONS/DEV-A 有版本表；Available=false 时有明确下一步（升级/换机），不假装通过 |
| **证据** | `claude --version` 输出；Detect 截图 |
| **关联** | 手册 §2 版本约束；I2 |

#### P0-ENV-4 · 桌面 live 可重复启动 · B（C 跟做）

| | |
|---|---|
| **目标** | 任意开发机按同一命令起「真实绑定」窗 |
| **步骤** | 1）`go install …/wails3@v3.0.0-beta.34` 2）导出 P0-ENV-1/2 变量 3）`wails3 generate bindings ./...` 4）`npm ci --prefix frontend` 5）`wails3 dev` 6）确认顶栏「真实绑定」非「模拟模式」 |
| **DoD** | 书面命令可复制；bindings 存在且未手改；C 能独立复现 |
| **证据** | 命令+commit；窗口顶栏截图 |
| **关联** | README 研发 B 命令；手册 P04 |

---

### 2.2 独立检查收口

#### P0-D3 · Wails 契约面证据 · B

| | |
|---|---|
| **目标** | ACCEPTANCE D3 正式「通过」 |
| **步骤** | 对照 CONTRACT：七个绑定方法、Response.error、事件名 `harbor:event`、无原始 Go error 进 UI；补/贴已有测试与 WebView 冒烟 |
| **DoD** | ACCEPTANCE 增加 D3 证据段；状态=通过 |
| **证据** | 测试名+日志摘要；字段对照表 |
| **依赖** | 无 |

#### P0-D5 · 受控上游流与故障 · B

| | |
|---|---|
| **目标** | D5：流保留；401/断流/撤销/未知目标正确；非幂等不自动重试 |
| **步骤** | 用 Postman/受控 mock 或真实上游逐条打；记录码与是否代填 |
| **DoD** | ACCEPTANCE D5 证据段；每类故障至少一条 |
| **证据** | 用例表；脱敏响应码 |
| **依赖** | 受控上游或 P0-ENV |
| **关联** | 手册 M17+、gate 仿真测试 |

---

### 2.3 真实闭环 I1–I11（任务卡）

#### P0-I1 · 桌面真连模型+MCP · B/C

| | |
|---|---|
| **目标** | 关 mock；成功只返回账号引用；输入清空；失败不成可用账号 |
| **步骤（C 操作 / B 排障）** | 1）P0-ENV-4 起桌面 2）选 Agent 3）填显示名+真 Key → 连接模型 → 确认输入框空、状态含 id/host 无 Key 4）同法 MCP 5）故意错 Key → `UPSTREAM_AUTH_FAILED`（或契约码）且无账号 6）记 ACCEPTANCE §I1 |
| **DoD** | 正例+反例皆有；徽章=真实绑定 |
| **证据** | 成功/失败各一帧脱敏图 |
| **依赖** | P0-ENV-1/2/4 |
| **关联** | 手册 M06–M07 |

#### P0-I2 · 真 Agent 任务 + MCP · A/B/C

| | |
|---|---|
| **目标** | 演示目录内真实改文件 + 发现并调用白名单工具；非 curl/fixture |
| **步骤** | 1）`./demos/verify-workspace.sh` 2）替换默认任务（去掉 mock/search 字样，改用真实工具名） 3）选工作区绝对路径 4）开始 → 工作台看输出与审计 `allowed` 5）核对磁盘 `hello.txt`（或任务约定文件） 6）A 确认进程在隔离内 |
| **DoD** | 文件变更+工具审计+Agent 输出三者一致；记 I2 通过或失败原因 |
| **证据** | 工作区 diff；审计行；session 终态 |
| **依赖** | I1、P0-ENV-3；注意 Claude 无 Bash |
| **关联** | 手册 M08–M12 |

#### P0-I3 · Sentinel / 无真 Key 进执行面 · A/B

| | |
|---|---|
| **目标** | Agent env 与私有配置无上游真 Key；事件/日志无令牌 |
| **步骤** | 1）使用可轮换测试 Key 2）会话中由 A 查私有 HOME/env（工程师辅助，非 UI） 3）扫公开事件与日志 4）受控上游可回显认证头则核对仅为 Sentinel |
| **DoD** | 无真实 Key 落盘/进沙箱配置；ACCEPTANCE I3 |
| **证据** | 脱敏 env 摘要；检索脚本输出 |
| **依赖** | I2 环境 |
| **关联** | 手册 E02 |

#### P0-I4 · 文件隔离 · A

| | |
|---|---|
| **目标** | 工作区可读写；区外自建文件、凭据路径、越界链接拒绝 |
| **步骤** | 按 security 负例 + 受限进程探针；禁止打印用户真密钥 |
| **DoD** | 正负例成对；非「仅诊断模板」 |
| **证据** | 命令与拒绝原因 |
| **依赖** | Runtime |
| **关联** | 手册 M17、E 系列 |

#### P0-I5 · 网络隔离 · A/B

| | |
|---|---|
| **目标** | 仅精确 Gate 数据端口可达；公网/IP/其它回环拒绝由 Runtime 强制 |
| **步骤** | 受限进程访问 Gate vs example.com vs 其它 localhost；拒绝不得只靠 Gate 业务层 |
| **DoD** | I5 证据段 |
| **证据** | 探测结果表 |
| **依赖** | Runtime+Gate |
| **关联** | 手册 E01 |

#### P0-I6 · Gate 权限与重定向 · B

| | |
|---|---|
| **目标** | 错/撤 Sentinel、未授权工具、恶意目标、重定向不向它处代填 |
| **步骤** | 逐项注入；确认会话权限不可被 Agent 改写 |
| **DoD** | 每类至少一失败且无凭据泄漏 |
| **证据** | 拒绝日志（脱敏） |
| **依赖** | Gate；I1 后更佳 |
| **关联** | 手册 M17–M22 |

#### P0-I7 · 失败关闭与回滚 · A/B

| | |
|---|---|
| **目标** | profile 不可用、Gate 部分失败、Create/Exec 失败、运行中 Gate 致命 → 不直跑宿主机 |
| **步骤** | 工程师受控制造故障；核对状态机与资源清理 |
| **DoD** | 无「降级宿主机 shell」；晚返回资源也被收 |
| **证据** | 状态序列；进程列表前后 |
| **依赖** | A/B 故障注入能力 |
| **关联** | 手册 E03 |

#### P0-I8 · 停止与幂等 · A/C

| | |
|---|---|
| **目标** | provisioning/running 中 Stop；重复 Stop 幂等；终态事件确认清理 |
| **步骤（C）** | 开始后立刻停；跑起来再停；连续点两次停 **（A）** 确认授权撤销与进程结束 |
| **DoD** | UI 与事件一致；无卡死 busy |
| **证据** | 状态时间线截图 |
| **依赖** | 可创建会话 |
| **关联** | 手册 M13–M16 |

#### P0-I9 · 清理失败占锁 · A/B

| | |
|---|---|
| **目标** | Destroy 失败/超时 → `failed`+`cleanupComplete=false` 占锁；重试成功后才允许新会话 |
| **步骤** | 受控注入清理失败；C 见提示再点停止；二次成功后开新会话 |
| **DoD** | 占锁期间开始被拒；解锁后可开 |
| **证据** | cleanup 字段变化 |
| **依赖** | 故障注入 |
| **关联** | mock cleanup 仅作 UI 预习，不计本项通过 |

#### P0-I10 · 两轮可重跑 · A/B/C

| | |
|---|---|
| **目标** | 正常退出/取消/认证超时后再跑；连续两轮无残留进程或旧令牌 |
| **步骤** | 完整 I2 一轮 → 停净 → 再连（或沿用账号）再跑一轮；抽查进程与 secrets |
| **DoD** | 两轮均有输出/审计；无僵尸 Agent |
| **证据** | 两轮 sessionId；进程快照 |
| **依赖** | I1–I9 至少主路径可用 |
| **关联** | 手册 M11–M12、M23 |

#### P0-I11 · 打包运行与退出 · B/C

| | |
|---|---|
| **目标** | 登记机 `wails3 build` / `darwin:package`；退出触发 Shutdown，无残留 Agent |
| **步骤** | 1）生产构建 2）C 打开 .app 走缩略 I1/I2 3）运行中与空闲各退一次 4）查进程 |
| **DoD** | 资源可加载；Shutdown 证据；ACCEPTANCE I11 |
| **证据** | 构建日志；退出前后 `ps` |
| **依赖** | 绑定稳定；P0-ENV |
| **关联** | 手册 M23–M25 |

---

### 2.4 角色执行顺序（P0）

```text
共同: ENV-1 → ENV-2 → ENV-3 → ENV-4
B:    D3 → D5（可与 ENV 并行）→ 带 I1 → I6 → I11
C:    跟 I1 → I2 操作与 demos 真机话术 → I8/I10/I11 点选记录
A:    ENV-3 → I2 隔离确认 → I3/I4/I5 → I7/I9 → 支援 I8/I10
```

**C 额外交付（P0，非验收编号但必做）**

| ID | 任务 | DoD |
|---|---|---|
| P0-C-DOC-1 | `demos/README` 增加 live 启动、env、工作区路径、真实任务模板 | 新人按文档能开到真实绑定 |
| P0-C-DOC-2 | 10 分钟评审话术：架构一句 + I2 指文件/审计 + 一句「mock≠I」 | 写入 demos，可照念 |
| P0-C-DOC-3 | 将 I1/I2/I8/I10/I11 操作结果补进 ACCEPTANCE 或联调 RESULTS（脱敏） | 状态非「待执行」或标明阻断原因 |

---

## 3. P0.5 — 契约冻结会（支撑 ABC 独立开发）

### 3.0 目的与规则

| | |
|---|---|
| **目的** | 在写 P1 页面前，把交互接口冻进 [CONTRACT](CONTRACT.md) + `internal/contracts`，三人用 **mock / fake** 并行，避免联调现场改字段 |
| **时长** | 建议 0.5 天：对齐 → 改契约 PR → 点头；实现可后置 |
| **与 P0 关系** | **不阻塞** I1–I11；**阻塞** P1-C-01 及以后依赖新 API 的开发 |
| **已够用（不必重开）** | 现有 DesktopAPI：`ListAgents` / `SelectWorkspace` / `ConnectModel` / `ConnectMcp` / `CreateSession` / `GetSession` / `StopSession` + `harbor:event`（见 MODULE-DESIGN §5） |
| **变更纪律** | 先改 CONTRACT 与 Go DTO → A fake / B 绑定面 / C mock 同步 → 再写真实现；禁止只改前端猜字段 |

**冻结产出物（DoD）**

1. CONTRACT 增补章节合入（或明确「本迭代不做 / 延后」打勾）  
2. `internal/contracts` 有对应类型（可先空实现）  
3. C：`mock-api.js` 有同名方法与至少一条事件样例  
4. A：编排或 kit 测试 fake 签名对齐  
5. B：DesktopAPI / 生成绑定清单更新（可先返回 `UNIMPLEMENTED` 类错误码，但形状要对）  
6. 本文 §3.1 清单全部勾选或标「延后至 Px」

---

### 3.1 冻结议题清单（会前填状态）

状态：`待议` / `已冻` / `延后` / `现契约已够`。

| ID | 议题 | 服务的 P1 | 主提案 | 需冻结的内容（最低） | 状态 |
|---|---|---|---|---|---|
| P0.5-API-01 | **运行时健康 / 隔离探测** | C-01 状态灯、C-02 Onboarding、C-10 | A | 方法名（如 `GetRuntimeHealth`）；字段：`available`、`reason`、可选 `isolationLevel`；错误码；是否可缓存 | 待议 |
| P0.5-API-02 | **会话列表与多会话** | C-03 HOME、A-01 | A+C | `ListSessions` / `DestroySession`（名可议）；快照摘要字段；错误：`SESSION_LIMIT`、`WORKSPACE_BUSY`；running 上限=2、同工作区冲突语义 | 待议 |
| P0.5-API-03 | **权限一览（只读）** | C-08 边界条、B-05 | B | `GetPermissionOverview(sessionId?)`；工作区路径、允许 host 列表、MCP 摘要、禁止路径类别；**禁止**含 Key/Sentinel 明文 | 待议 |
| P0.5-API-04 | **扩权挂起与决策** | C-09、B-04、A-05 | A/B | 事件类型（如 `permission_needed`）；载荷：sessionId、目标、动作摘要；方法 `ResolvePermission(decision)`：`deny` / `once` / `project`；**明确不做「全网允许」** | 待议 |
| P0.5-API-05 | **多模型账号** | C-05、B-02 | B+C | 是否扩展 `ConnectModel` 或新增 `ListModelAccounts` / `RemoveAccount`；自由 origin 的策略校验规则；与固定 `demo-model` 并存策略 | 待议 |
| P0.5-API-06 | **多 MCP / OAuth** | C-06、B-03 | B+C | Token 轨是否够用本迭代；OAuth：开始 URL / 回调 / 取消；`mcpIds` 长度是否放开；工具白名单谁裁定 | 待议 |
| P0.5-API-07 | **工作区模式 clone/direct** | C-04、A-02 | A | `SessionSpec` 是否增加 `mountMode`；clone 路径与导回是否本迭代 API | 待议 |
| P0.5-API-08 | **UI 偏好持久化** | C-01 侧栏收缩 | B | 存哪：Go 小文件 vs Wails 仅内存；键：`sidebarCollapsed` 等；**不得**存 Key | 待议 |
| P0.5-API-09 | **PTY / 终端** | C-08 | 三人 | **拍板**：内嵌 PTY 事件流 vs 外置终端 vs **本迭代延后**；若做：流式事件与截断规则 | 待议 |
| P0.5-API-10 | **设置项只读面** | C-07 | A/B | 哪些设置有 API（重检 Agent、清理策略开关）；无后端的不进设置页 | 待议 |

---

### 3.2 建议会议议程（约 3–4h）

| 时段 | 内容 | 产出 |
|---|---|---|
| 0:00–0:20 | 重申 P0 七方法不变；本会只增 P1 所需 | 范围纸 |
| 0:20–1:20 | 逐项 API-01～04（健康、列表、权限一览、扩权）— P1 壳与 HOME 的关键路径 | 字段草稿 |
| 1:20–2:20 | API-05～07（多账号、MCP、mount）— 可大量标延后 | 延后列表 |
| 2:20–2:50 | API-08～10（偏好、PTY、设置） | 拍板或延后 |
| 2:50–3:30 | A 开 CONTRACT PR 骨架；约定错误码表；C/B 认领 mock/绑定跟进日 | PR 链接 |

---

### 3.3 冻结后的并行分工

```text
CONTRACT PR 合入
  ├─ A：fake SessionService / 健康与多会话测试桩
  ├─ B：DesktopAPI 方法签名 + wails3 generate（可先 UNIMPLEMENTED）
  └─ C：mock-api 同名方法 + 页面可先用 stub 数据开发壳/HOME
合流条件：方法名/JSON 字段/错误码一致；再换 live 实现
```

| 角色 | 冻结后立刻可做 | 仍须等真实现 |
|---|---|---|
| C | 壳、Onboarding UI、HOME 列表 mock、边界条 mock、扩权 Modal 交互稿 | 真多会话、真 OAuth、真 PTY |
| A | 多会话状态机单测、健康探测、扩权状态枚举 | 真 clone 文件系统、子进程方案 |
| B | Gate 权限一览组装、账号 CRUD 骨架、绑定生成 | Keychain、真 OAuth 上游 |

---

### 3.4 任务卡 · 会务与跟进

#### P0.5-MEET · 召开冻结会 · 三人 · 估 0.5d

| | |
|---|---|
| **目标** | §3.1 每项有终态（已冻/延后/现契约已够） |
| **步骤** | 会前各自标「必须本迭代」；会中填表；会后 A 提 CONTRACT PR |
| **DoD** | 表状态无「待议」；PR 已开或指定负责人与日期 |
| **证据** | 会议纪要一段贴 DEV-A/C 或本文更新；PR 链接 |

#### P0.5-CONTRACT · 契约与 DTO 落地 · A（B 复核）· 估 0.5–1d

| | |
|---|---|
| **目标** | 已冻项写入 CONTRACT + `internal/contracts` 可编译 |
| **DoD** | `go test` 相关包过；B 确认可生成绑定 |
| **依赖** | P0.5-MEET |

#### P0.5-MOCK · 前端 mock 对齐 · C · 估 0.5–1d

| | |
|---|---|
| **目标** | 已冻方法在 `mock-api.js` + 必要时单测 |
| **DoD** | `npm test` 过；浏览器可点出 stub 流 |
| **依赖** | P0.5-CONTRACT |

#### P0.5-BIND · 绑定面占位 · B · 估 0.5d

| | |
|---|---|
| **目标** | DesktopAPI 暴露新方法；生成 bindings；未实现返回稳定错误码 |
| **DoD** | C live 调用不会因「方法不存在」崩；形状与 CONTRACT 一致 |
| **依赖** | P0.5-CONTRACT |

---

## 4. P1 — 第一版设计欠账（细化）

规则：依赖 §3 已冻接口；未冻项不得开工实现。估时为人天量级（1 人），供排期，非承诺。

### 4.1 产品 C

#### P1-C-01 · 全局壳与导航 · 估 2–3d

| | |
|---|---|
| **目标** | 侧栏 HOME / 模型 / 工具 / 设置；底栏隔离状态灯；收缩+记忆 |
| **步骤** | 1）路由/视图骨架 2）状态灯订阅健康 API（可先 stub） 3）`⌘B` 收缩 4）进工作台可自动缩一次 |
| **DoD** | 四入口可切换；收缩后内容区变宽；偏好重启仍在 |
| **依赖** | B：偏好存储位置约定；A：健康检查字段 |
| **参考** | PAGE §4 |

#### P1-C-02 · Onboarding · 估 1–2d

| | |
|---|---|
| **目标** | 一屏三句 + 隔离检测；失败主按钮不可用；**无跳过** |
| **步骤** | 调检测 API → ●/○ → 进入 HOME 或展示修复链 |
| **DoD** | 检测失败无法进入主路径 |
| **依赖** | A 健康/隔离探测 API |
| **参考** | PAGE §5.1 |

#### P1-C-03 · HOME 会话中心 · 估 3–5d

| | |
|---|---|
| **目标** | 列表字段、运行中 x/2、打开工作台/停止/再次开始/销毁 |
| **步骤** | 列表 UI → 绑定多会话 API → 并发与同工作区错误文案 |
| **DoD** | 达上限与同目录冲突有产品文案；销毁二次确认 |
| **依赖** | **P1-A-01** 多会话编排 |
| **参考** | PAGE §5.2 |

#### P1-C-04 · 新会话 Wizard · 估 3–4d

| | |
|---|---|
| **目标** | 工作区→Agent→模型→工具（可跳过）；clone/direct 文案 |
| **步骤** | 步进条 → 每步校验 → 结束调 CreateSession |
| **DoD** | 可跳过工具；direct 强警告；clone 仅在后端支持时可选 |
| **依赖** | P1-A-02（若启用 clone）；现有 Connect API |
| **参考** | PAGE §5.3 |

#### P1-C-05 · 独立模型页 · 估 3–4d

| | |
|---|---|
| **目标** | 自由添加 OpenAPI（名+地址+Key）；列表无「未配置」占位 |
| **步骤** | CONTRACT 多账号 → UI CRUD → 提交清空 Key → ▲ 无效 |
| **DoD** | 不再写死唯一 demo-model（或并存迁移开关） |
| **依赖** | **P1-B-02** |
| **参考** | PAGE §2 |

#### P1-C-06 · 独立工具页 · 估 4–6d

| | |
|---|---|
| **目标** | OAuth/SSO 或 Token；只展示已添加 MCP |
| **步骤** | OAuth 回调 UX → Token 表单 → 状态 ●/▲ |
| **DoD** | 沙箱文案仍无厂商密钥；双轨至少一条真通 |
| **依赖** | **P1-B-03** |
| **参考** | PAGE §2 |

#### P1-C-07 · 设置页 · 估 2–3d

| | |
|---|---|
| **目标** | Agent 重检、会话默认、扩权策略入口、清理、关于 |
| **DoD** | 与扩权/清理后端字段对齐；无空壳开关 |
| **依赖** | A/B 设置相关 API |
| **参考** | PAGE 设置 |

#### P1-C-08 · 工作台增强 · 估 5–8d（含 PTY 决策）

| | |
|---|---|
| **目标** | 边界条；多会话焦点；内嵌 PTY **或** 外置终端（先拍板） |
| **步骤** | 1）DESIGN 开放问题书面结论 2）边界条读权限一览 3）PTY 或深链 |
| **DoD** | 评审能指着边界条讲隔离；终端方案文档化 |
| **依赖** | P1-B-05；P1-A-01；PTY 则 A 输出通道 |
| **参考** | PAGE 工作台 |

#### P1-C-09 · 扩权 Modal · 估 2–3d

| | |
|---|---|
| **目标** | 一句话谁访什么；拒绝 / 仅本次 / 本项目始终；标会话；**无「全网」** |
| **DoD** | 三按钮都有后端回执；多会话标明 sessionId |
| **依赖** | **P1-B-04**、**P1-A-05** |
| **参考** | PAGE 系统打断 |

#### P1-C-10 · 隔离阻断页 · 估 1d

| | |
|---|---|
| **目标** | 隔离不可用全页说明；禁新会话 |
| **DoD** | 与状态灯红一致；无「仍要继续」 |
| **依赖** | A 探测 |
| **参考** | PAGE |

#### P1-C-11 · 视觉对齐原型 · 估 2d

| | |
|---|---|
| **目标** | 品牌/文案/密度贴近 `prototype/`，仍非 IDE |
| **DoD** | 对照 PAGE §3 清单自检通过 |
| **依赖** | 壳稳定（P1-C-01） |

---

### 4.2 研发 B

#### P1-B-01 · Keychain 持久化 · 估 3–5d

| | |
|---|---|
| **目标** | 密钥进系统钥匙串；重启可恢复引用；禁明文配置 |
| **DoD** | 进程外无 Key 文件；轮换/删除路径可用 |
| **参考** | DESIGN §11 |

#### P1-B-02 · 多模型账号 · 估 3–4d

| | |
|---|---|
| **目标** | 用户自填 Base URL+Key；仍只经 Gate；CONTRACT 扩展 |
| **DoD** | 前端不直连上游；策略校验 origin |
| **参考** | PAGE 模型 |

#### P1-B-03 · OAuth MCP + 多 MCP · 估 5–8d

| | |
|---|---|
| **目标** | OAuth/SSO；多 MCP 挂载；工具发现+人工白名单产品化 |
| **DoD** | 至少 1 条 OAuth 真通；stdio/旧 SSE 仍可标延后 |
| **参考** | DESIGN M2 |

#### P1-B-04 · 扩权策略引擎 · 估 4–6d

| | |
|---|---|
| **目标** | 挂起事件；仅本次/项目级持久；可撤销 |
| **DoD** | Agent 无法改写已持久策略；有审计 |
| **参考** | ARCHITECTURE |

#### P1-B-05 · 权限一览 API · 估 2–3d

| | |
|---|---|
| **目标** | 返回工作区、允许域名、MCP、禁止路径（只读） |
| **DoD** | C 可绑边界条；无密钥字段 |
| **参考** | DESIGN §4 |
| **可先** | mock 固定 JSON 解耦 C |

#### P1-B-06 · 安装与更新 · 估 3–5d（可后置）

| | |
|---|---|
| **目标** | 签名/公证/更新通道 |
| **DoD** | 内测机器可装；文档化限制 |
| **参考** | DESIGN M4 |

---

### 4.3 研发 A

#### P1-A-01 · 多会话编排 · 估 5–8d

| | |
|---|---|
| **目标** | running≤2；同工作区禁双开；列表/销毁语义 |
| **步骤** | CONTRACT 会话列表 API → 锁与冲突错误码 → 测试 |
| **DoD** | 单测覆盖上限与同路径；C 可接 HOME |
| **参考** | PAGE §5.2.1 |

#### P1-A-02 · clone 工作区 · 估 4–6d

| | |
|---|---|
| **目标** | 安全副本 + 可选导回 |
| **DoD** | direct/clone 切换有警告；导回确认 |
| **依赖** | 与 P1-C-04 同步 |
| **参考** | DESIGN / PAGE |

#### P1-A-03 · Codex + dsh 真机验收 · 估 3–5d（dsh）+ Codex 另估

| | |
|---|---|
| **目标** | dsh 按 [DSH-INTEGRATION](DSH-INTEGRATION.md) 全手册过；Codex Kit 新増 |
| **DoD** | DECISIONS 登记版本；至少一条真任务证据 |
| **参考** | DESIGN Agent P0 |

#### P1-A-04 · 子进程方案 · 估 大（门禁）

| | |
|---|---|
| **目标** | 若核心任务需要 Bash/子进程：可靠追踪+清理后再放行 |
| **DoD** | 设计评审通过；全量 Runtime/安全重测；**禁止**偷偷开 fork |
| **参考** | DEV-A 约束 |

#### P1-A-05 · 扩权挂起状态机 · 估 3–4d

| | |
|---|---|
| **目标** | `awaiting_permission`（或契约等价）与恢复/拒绝 |
| **DoD** | 与 P1-B-04/C-09 联调一条完整扩权 |
| **参考** | ARCHITECTURE |

#### P1-A-06 · 一期隔离证据包 · 估 2–3d

| | |
|---|---|
| **目标** | 证明「工作区+降权」而非仅目录限制文案 |
| **DoD** | 对外一页证据索引；挂 ACCEPTANCE/security |
| **参考** | DESIGN §7.6 |

#### P1-A-07 · 二期容器/microVM · backlog

| | |
|---|---|
| **目标** | 不挡 P0/P1 桌面主路径；单独里程碑 |
| **DoD** | 仅预研/Spike 记录即可 |

---

## 5. 排期与依赖图

```text
P0-ENV-1/2/3/4
    └─► P0-I1 ─► P0-I2 ─► P0-I3
              │         ├─► P0-I4 / I5
              │         └─► P0-I10
              ├─► P0-I6
              ├─► P0-I8
              └─► P0-I7 / I9（故障注入可并行准备）
P0-D3 / D5 可与 ENV 并行
P0-I11 ← 绑定稳定 + I1 缩略路径

P0.5-MEET ─► P0.5-CONTRACT ─┬─► P0.5-MOCK (C)
                            ├─► P0.5-BIND (B)
                            └─► A fakes
                                 │
                                 ▼
P1-C-01 ─┬─► P1-C-02          （须 API-01 已冻或 stub 约定）
         ├─► P1-C-11
         └─► P1-C-03 ← P1-A-01（须 API-02）
P1-B-05 ──► P1-C-08（须 API-03）
P1-B-02 ──► P1-C-05（须 API-05）
P1-B-03 ──► P1-C-06（须 API-06）
P1-A-05 + P1-B-04 ──► P1-C-09（须 API-04）
```

**时间不够删减顺序**：C-11 视觉 → 第二 MCP → Codex → clone → PTY → Keychain → OAuth。  
**不可删**：真 Agent、真认证 MCP、隔离强制、凭据外置、停止清理。

---

## 6. 分工边界

| 角色 | 可改 | 不可改 |
|---|---|---|
| A | `orchestrator` / `runtime` / `kits`、security、**CONTRACT 主笔** | 手改 bindings；为赶演示放宽隔离 |
| B | `gate` / `policy` / `secrets`、desktop、构建、绑定生成 | 第二套会话状态机；前端绕过 Gate |
| C | `frontend/` 手写、`demos/`、验收记录、**字段需求提案** | `frontend/bindings/`、`internal/` |

---

## 7. 周会检查清单

**P0**
- [ ] ENV-1/2 写入 DECISIONS  
- [ ] ENV-3 Detect 结果（通过或阻断原因）  
- [ ] ENV-4 三人都能 `wails3 dev` 见真实绑定  
- [ ] D3/D5 证据段  
- [ ] I1 / I2  
- [ ] I3–I7 各 ≥1 条真实负例  
- [ ] I8 / I9 / I10 / I11  
- [ ] P0-C-DOC-1/2/3  

**P0.5**
- [ ] 冻结会已开（P0.5-MEET）  
- [ ] §3.1 无「待议」  
- [ ] CONTRACT PR 合入（P0.5-CONTRACT）  
- [ ] C mock / B 绑定占位已跟（P0.5-MOCK / BIND）  

**P1（可选）**
- [ ] 侧栏可导航（C-01）  
- [ ] HOME 或 Wizard 可演示  
- [ ] 扩权或权限一览可演示  
- [ ] 第二 Agent 真机一条路径  

---

## 8. 相关入口

| 文档 | 用途 |
|---|---|
| [CONTRACT.md](CONTRACT.md) | 并行实现的唯一方法/字段来源 |
| [MODULE-DESIGN.md](MODULE-DESIGN.md) | 替身约定与文件归属 |
| [ACCEPTANCE.md](ACCEPTANCE.md) | 编号与证据格式 |
| [MANUAL-INTEGRATION-TEST.md](MANUAL-INTEGRATION-TEST.md) | 逐步人工用例 P/M/E/D |
| [DECISIONS.md](DECISIONS.md) | 上游与版本登记 |
| [DEV-A.md](DEV-A.md) / [DEV-C.md](DEV-C.md) | 角色笔记 |
| [DSH-INTEGRATION.md](DSH-INTEGRATION.md) | dsh |
| [demos/README.md](../demos/README.md) | 演示与工作区 |

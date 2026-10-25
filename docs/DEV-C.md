# 产品 / 前端 C 开发记录

## 2026-10-24：浏览器 mock 两页与 D4 骨架

本轮按 [MODULE-DESIGN.md](MODULE-DESIGN.md) §5 与 [CONTRACT.md](CONTRACT.md) §1 落地前端手写实现，**不依赖 Wails / wailsjs**。当前可做独立 UI/协议验收（ACCEPTANCE **D4**），不是真实桌面闭环，也不能冒充 I1–I11 通过。

### 已落地

| 路径 | 说明 |
|---|---|
| `frontend/index.html` | 配置 / 工作台入口；顶栏「模拟模式」徽章 |
| `frontend/src/main.js` | 切页装配；默认 `useMock()`；支持 `?scenario=` |
| `frontend/src/api.js` | 页面对外 camelCase API；框架异常映射为 `INTERNAL_ERROR`；预留 `createLiveTransport()` / `useLive()` |
| `frontend/src/mock-api.js` | 同契约 mock transport；固定 Agent/模型/MCP/工作区；事件结构对齐 `harbor:event` 载荷 |
| `frontend/src/styles.css` | 两页共用样式 |
| `frontend/src/views/setup.js` | 连接模型/MCP（提交清空 Key）、选工作区、任务、开始；未就绪禁用启动；场景下拉 |
| `frontend/src/views/session.js` | 状态 pill、stdout 文本输出（1 MiB 截断）、审计列表、停止 / 清理失败重试、回配置 |
| `demos/README.md` | 补充 mock 演示步骤 |

固定 mock 数据（与模块设计一致）：

- Agent `id=claude`
- 模型 `id=mock-model`，`profileId=demo-model`
- MCP `id=mock-mcp`，`profileId=demo-mcp`
- 工作区 `/demo/project`
- `mcpIds` 长度强制为 1

场景覆盖：

| 场景 | 用途 |
|---|---|
| `normal` | provisioning → running → output → MCP allowed 审计 → stopping → stopped |
| `auth` | ConnectModel / ConnectMcp 返回 `UPSTREAM_AUTH_FAILED` |
| `busy` | CreateSession 返回 `SESSION_BUSY` |
| `isolation` | CreateSession（及 Agent 不可用）`ISOLATION_UNAVAILABLE` |
| `cleanup` | 用户 Stop 后 `failed` 且 `cleanupComplete=false`；再次 Stop 模拟清理重试成功 |

### 实际验证

```sh
cd frontend
python3 -m http.server 5173
# 浏览器打开 http://127.0.0.1:5173/
# 可选：http://127.0.0.1:5173/?scenario=auth
```

- 使用本地 HTTP，避免 `file://` 阻断 ES module。
- 手工走通：连接（任意非空 Key，提交后清空）→ 选工作区 → 开始 → 工作台看状态/输出/审计 → 停止。
- 场景下拉或 query 覆盖四失败路径。
- `node --check` 已对各 `.js` 做过语法检查（无打包测试套件）。
- **2026-10-24 晚**：浏览器脚本自测 — normal 闭环（输出+MCP audit→stopped）；auth / busy / isolation（CreateSession `ISOLATION_UNAVAILABLE`）/ cleanup（停两次）均符合预期。含 `:root` 样式修复与交互打磨。

**D4 状态**：**通过（mock 独立检查）**，证据表见 [ACCEPTANCE.md §4.1](ACCEPTANCE.md#412026-10-24-d4-证据产品-c--mock)。不得记为真实闭环通过。

### 契约字段核对（CONTRACT §1）

| 页面已用 | 结论 |
|---|---|
| AgentInfo、ConnectInput、AccountInfo、SessionSpec/Snapshot、Responses、Event 四类载荷 | 已对齐，无额外字段需求 |
| 错误码 UPSTREAM_AUTH_FAILED / SESSION_BUSY / ISOLATION_UNAVAILABLE / SESSION_NOT_FOUND / INVALID_ARGUMENT / CLEANUP_FAILED / INTERNAL_ERROR | mock 与 UI 已展示；正式码表以契约为准 |
| 当前无缺口提案 | 不改 CONTRACT；live 联调若缺字段再提 A |

### 本轮交互打磨

- 连接前校验空显示名 / 空 Key，避免无意义请求；**校验失败不清空 Key**，仅在真正提交后清空。
- 「开始」未就绪时列出缺少项（Agent / 模型 / MCP / 工作区 / 任务）；连接中禁用按钮。
- 默认任务文案固定为非敏感 D4 演示脚本（见 `DEMO_TASK` / demos README）。
- 工作台：`cleanupComplete=false` 时提示再点停止；`stopped` 提示可回配置再开；进行中回配置给出提示。
- `demos/README.md` 三分钟话术 + 固定任务文案。

### B 接入说明

- 远程 `main` 已含 B 的 Wails v3 壳、`frontend/package.json`（Vite）、`dist/` 占位与 Gate/Policy/Secrets。
- 绑定目录为 `frontend/bindings/`（不是旧文案中的 wailsjs）。`src/live-api.js` 通过 Vite 编译生成绑定，`api.js` 延迟加载它，再用 `useLive()` 或 `?live=1` 切换真实通道。
- **视图方法名与字段不要改**；transport 可替换。
- 真实模式须显示「真实绑定」，禁止把 mock 计入 I1–I11。
- 预览：`cd frontend && npm run dev`，或 `python3 -m http.server`。

### 明确未做

- 真实凭据与上游
- 完整 PAGE-DESIGN（HOME / 设置 / 扩权 / 双会话 / PTY）
- 自动 E2E 测试脚本、`demos/run.sh`
- 修改 `internal/`、Gate、Runtime、契约字段（页面需求已按现有 CONTRACT 对齐，无新增字段提案）

### 后续顺序

1. ~~D4 证据~~ 已记入 ACCEPTANCE §4.1。
2. 将本分支前端与文档推到 `origin`（当前 GitHub CLI token 若失效需先 `gh auth refresh`）。
3. 使用 Wails v3 绑定生成和 live transport；准备真实凭据后联调 I1（关闭 mock 真连）。
4. ~~准备非敏感演示目录与任务文案~~ → `demos/workspace/` + `verify-workspace.sh` + 固定任务；配合 A/B 冲 I2 / I8 / I10 / I11。
5. 需要新 UI 字段时先提 CONTRACT，再改 api/mock/视图。

### 演示工作区（2026-10-24）

- 路径：`demos/workspace/`（运行产物 `hello.txt` 已 gitignore；说明见该目录 README）。
- 自检：`./demos/verify-workspace.sh`（可写、可覆盖、无敏感文件名）。
- mock 选择工作区仍为 `/demo/project`；live 选本目录绝对路径。

### 模型 / MCP 示例填写

见 [demos/README.md](../demos/README.md)：mock 可用示例假 Key；live 用真凭据；Claude=`demo-model`，dsh=`dsh-model`，MCP=`demo-mcp`。上游 **无默认 origin**，须 `HARBOR_*` 登记。

---

## 2026-10-25：按 TASK-SPLIT 启动 C 分工

任务源：[TASK-SPLIT-ABC.md](TASK-SPLIT-ABC.md)。本轮 **先做 P0 文档与可独立项**；P1 页面等 P0.5 契约冻结。

### 本轮已交付（C）

| ID | 内容 |
|---|---|
| P0-C-DOC-1 | [demos/README.md](../demos/README.md)：live 启动命令、env、工作区、真实任务模板、I1/I2 检查表 |
| P0-C-DOC-2 | 同文档「十分钟 live 话术」 |
| P0-C-DOC-3 骨架 | [demos/evidence-template.md](../demos/evidence-template.md) 填空模板（真机后再写入 ACCEPTANCE） |
| UI | live 下隐藏 mock 场景卡；任务提示；真机任务含 mock/search 时开始前警告；顶栏 banner 随模式切换 |
| — | 纠正过时表述（live 无默认 DeepSeek/占位 MCP origin） |

### 当前页面边界（跟 main）

- ~~配置 / 工作台两页~~ → 见下节 **PAGE-DESIGN 壳**
- 桌面强制 live；浏览器默认 mock
- 无扩权 Modal / PTY / 真双会话；工作台仍为文本输出

---

## 2026-10-25：按 PAGE-DESIGN 落地壳（C）

对照 [PAGE-DESIGN.md](PAGE-DESIGN.md) 与 [TASK-SPLIT-ABC.md](TASK-SPLIT-ABC.md) P1-C-01～05（契约未冻处用 stub）。

### 已落地

| 路径 | 说明 |
|---|---|
| `frontend/index.html` | 侧栏：HOME / 模型 / 工具 / 设置 / 工作台；底栏隔离灯；引导/阻断层 |
| `frontend/src/main.js` | 路由、`⌘B` 收缩记忆、进工作台可自动缩、健康刷新 |
| `views/onboarding.js` | 三句引导；隔离失败不可进入 |
| `views/home.js` | 会话中心（单会话历史摘要；上限文案先写 1） |
| `views/wizard.js` | 新会话步进条 + 内嵌原 `setup` 配置流 |
| `views/models.js` / `tools.js` / `settings.js` | 独立配置页；mock 场景进设置 |
| `views/block.js` | 隔离不可用阻断层 |
| `mock-api` / `api` | `getRuntimeHealth`、`listSessions` stub；live 缺方法时健康占位可用 |

### 与两天契约关系

- Connect / CreateSession / Stop / Event **不变**；向导仍走现有 setup。
- 健康与列表 **非正式契约字段**，待 P0.5-API-01/02；勿记入 I1–I11。
- 总纲：[SYSTEM-DESIGN.md](SYSTEM-DESIGN.md)；接口：[CONTRACT.md](CONTRACT.md) v1.1；屏映射：[PAGE-API-MAP.md](PAGE-API-MAP.md)。无冻结会，按已定契约实现。

### C 下一步

| 优先级 | 项 | 依赖 |
|---|---|---|
| P0 | 跟 B 跑 I1/I2 与脱敏证据 | ENV、Detect、Claude 2.1.292 |
| V1 | mock/页面对齐 CONTRACT v1.1；扩权 Modal、权限一览、HOME x/2 | A/B 真实现 |
| P1 | Wizard 真四步拆页（仍无 PTY/OAuth/clone） | 已定契约 |

### 明确不做（仍禁止）

手改 `bindings/`、改 `internal/`、把 mock 记为 I 通过、未冻接口当真实现。

---

## 分工备忘（C）

| 负责 | 目录 |
|---|---|
| 壳与页面、样式与交互 | `frontend/index.html`、`frontend/src/**` |
| 接口封装与模拟数据 | `frontend/src/api.js`、`frontend/src/mock-api.js`、`live-api.js` |
| 产品验收与演示说明 | `demos/`、[ACCEPTANCE.md](ACCEPTANCE.md)、[TASK-SPLIT-ABC.md](TASK-SPLIT-ABC.md) |

不修改桌面 Go 入口与会话状态机；不手改 `frontend/bindings/`。

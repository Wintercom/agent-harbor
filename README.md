# Agent Harbor

桌面 Agent 安全沙箱：密钥留在本机、Agent 隔离运行。控制面（Gate / Policy / Secrets / Wails v3 桌面桥）已由研发 B 落地；编排、隔离与页面仍待 A / C。

仓库：https://github.com/Wintercom/agent-harbor

可浏览原型（不接真 Runtime）：打开 [`prototype/index.html`](./prototype/index.html)（若有缓存请强制刷新）。

## 研发 B：开发与构建命令

前置：Go 1.26+（`go.mod` 会自动拉 toolchain）、[Wails v3 CLI](https://v3.wails.io/zh-cn/quick-start/why-wails/)（`go install github.com/wailsapp/wails/v3/cmd/wails3@latest`）、Node ≥16。

```bash
# 单元测试（gate / policy / secrets / 桌面桥）
go test ./internal/gate ./internal/policy ./internal/secrets .
go test ./internal/...

# 生成前端绑定（输出 frontend/bindings，勿手改）
wails3 generate bindings ./...

# 前端构建（产物 frontend/dist，供 Go embed）
cd frontend && npm install && npm run build && cd ..

# 开发模式 / 打包
wails3 dev
wails3 build
```

A 注入编排：在 `main.go` 调用 `WireSessionService(host, factory)` 与 `host.SetAgentDetector(...)`。固定上游可用环境变量 `HARBOR_DEMO_MODEL_ORIGIN`、`HARBOR_DEMO_MCP_ORIGIN`。

## 目录

```text
agent-harbor/
├── main.go / app.go / bindings.go   B：Wails v3 启动与 DesktopAPI
├── go.mod / Taskfile.yml            B：模块与任务
├── internal/
│   ├── contracts/                   DTO（B 先行可编译；A 主改）
│   ├── gate/ policy/ secrets/       B：已实现
│   ├── orchestrator/ runtime/ kits/ A：待实现
├── frontend/                        C 页面 + B 构建/绑定
├── build/                           B：Wails v3 平台资源
├── security/ demos/ docs/ scripts/ prototype/
```

模块职责见 [internal/README.md](internal/README.md) 和 [frontend/README.md](frontend/README.md)。并行开发先读 [模块设计](docs/MODULE-DESIGN.md) 与 [接口契约](docs/CONTRACT.md)；版本与命令证据见 [实施决策](docs/DECISIONS.md)，验收见 [ACCEPTANCE.md](docs/ACCEPTANCE.md)。

当前两天版：单一 macOS 环境、一个真实 Agent、一个模型组合、一个认证远程 MCP、单会话与 direct 工作区，凭据仅存宿主机内存。其余能力仍是完整 V1 或后续目标，详见 [MVP-2D3P.md](docs/MVP-2D3P.md)。

## GitHub 推送 / 拉取

```bash
# 首次或同步远程
./scripts/github-pull.sh

# 提交并推送（有改动会自动 commit；自定义文案：HARBOR_COMMIT_MSG='…' ./scripts/github-push.sh）
./scripts/github-push.sh

# 只推送、不自动提交
HARBOR_NO_COMMIT=1 ./scripts/github-push.sh
```

默认远程：`https://github.com/Wintercom/agent-harbor.git`，分支 `main`。可用环境变量覆盖：`HARBOR_REMOTE`、`HARBOR_BRANCH`。

## 文档怎么读

| 顺序 | 文档 | 回答什么 |
|---|---|---|
| 开发入口 | [docs/MODULE-DESIGN.md](docs/MODULE-DESIGN.md) | 模块边界、依赖、所有权、替身与并行检查点 |
| 接口 | [docs/CONTRACT.md](docs/CONTRACT.md) | 准确签名、事件/错误与生命周期 |
| 决策/验收 | [docs/DECISIONS.md](docs/DECISIONS.md) / [docs/ACCEPTANCE.md](docs/ACCEPTANCE.md) | 待验证项、真实检查步骤和证据 |
| 目录 | [docs/DIRECTORY-STRUCTURE.md](docs/DIRECTORY-STRUCTURE.md) | Go/Wails 结构、职责和迁移映射 |
| 当前优先 | [docs/MVP-2D3P.md](./docs/MVP-2D3P.md) | 三人两天架构、核心范围、分工、契约与验收 |
| 导航 | [docs/README.md](./docs/README.md) | 设计文档关系及实现目录入口 |
| 1 | [docs/DESIGN.md](./docs/DESIGN.md) | 做什么、不做什么、威胁与原则、人周 |
| 2 | [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md) | 控制面 / 执行面、Harbor Gate、会话与出站路径 |
| 3 | [docs/PAGE-DESIGN.md](./docs/PAGE-DESIGN.md) | HOME / 新会话 / 工作台 / 模型 / 工具 / 设置 |
| 4 | [docs/IMPLEMENTATION.md](./docs/IMPLEMENTATION.md) | 分期任务与验收 |
| 可选 | [docs/SPIKE-3D3P.md](./docs/SPIKE-3D3P.md) | 短周期证明「密钥不进沙箱」的竖切 |

## 完整 V1 产品要点（长期设计摘要）

- **模型**：自由添加名称 + API 地址 + Key；新会话 / 工作台用下拉选已添加项。
- **工具**：MCP；浏览器登录 / SSO 或 API Key；列表统一「编辑 / 删除」。
- **会话**：同时 running ≤ 2；同工作区禁止双开。
- **安全叙事**：凭证只在本机；沙箱经 Gate 出站，明文密钥不进执行面。
- **隔离落地**：**一期**工作区 + 进程降权（挡 reboot 靠降权，不靠目录）；**二期**换容器 Runtime。详见 [IMPLEMENTATION.md](./docs/IMPLEMENTATION.md) 与 [DESIGN.md](./docs/DESIGN.md) §7.4 / §7.6。

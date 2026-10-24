# Agent Harbor

桌面 Agent 安全沙箱：密钥留在本机、Agent 隔离运行。当前已建立设计文档与模块目录骨架，尚未提供可运行的应用代码及构建命令。

仓库：https://github.com/Wintercom/agent-harbor

可浏览原型（不接真 Runtime）：打开 [`prototype/index.html`](./prototype/index.html)（若有缓存请强制刷新）。

## 目录

```text
agent-harbor/
├── README.md
├── AGENTS.md                 通用规范与当前开发基线
├── CLAUDE.md                 Claude Code 规范入口
├── internal/                 Go 后端模块；各目录已有 README
│   ├── contracts/            A/B 确定、C 核对：DTO、事件与错误码
│   ├── orchestrator/         A：会话编排、装配、回滚与清理
│   ├── runtime/              A：操作系统隔离与进程管理
│   ├── kits/claude/          A：首个真实 Agent 适配
│   ├── gate/                 B：模型代理与 MCP Gateway
│   ├── policy/               B：固定授权策略
│   └── secrets/              B：宿主机内存凭据
├── frontend/                 C：Wails 桌面前端
│   └── src/
│       └── views/            配置页与工作台
├── build/                    B：Wails 构建资源说明
├── security/                 A 主责：跨模块安全验收
├── demos/                    C：演示流程
├── docs/                     架构、目录设计与交付方案
├── scripts/                  现有仓库维护脚本
└── prototype/                保留静态 UI 原型
```

模块职责、依赖和测试约定见 [internal/README.md](internal/README.md) 和 [frontend/README.md](frontend/README.md)。A 负责 Runtime/Kit/编排与技术集成，B 负责 Gate/Policy/Secrets、Wails 绑定及构建，C 负责产品、前端与演示。单元测试就近放置，跨模块检查放在 [security](security/README.md)。

Go/Wails 目录骨架及说明已迁移。根入口 main.go/app.go、go.mod、wails.json、前端依赖与生成绑定尚未实现；不创建空业务代码或虚假启动命令。目标文件结构及迁移映射见 [DIRECTORY-STRUCTURE.md](docs/DIRECTORY-STRUCTURE.md)。

三人并行开发先读 [模块设计](docs/MODULE-DESIGN.md)，再按 [接口契约](docs/CONTRACT.md) 实现与接入；各自使用同契约替身解除对其他模块实现的等待。当前替身与工程仍待实现，真实环境/接入见 [实施决策](docs/DECISIONS.md)，集成通过情况按 [验收清单](docs/ACCEPTANCE.md) 记录。

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

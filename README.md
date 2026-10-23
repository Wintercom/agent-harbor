# Agent Harbor

桌面 Agent 安全沙箱产品设计包：密钥留在本机、Agent 隔离运行。

仓库：https://github.com/Wintercom/agent-harbor  

可浏览原型（不接真 Runtime）：打开 [`prototype/index.html`](./prototype/index.html)（若有缓存请强制刷新）。

## 目录

```
agent-harbor/
├── README.md
├── scripts/
│   ├── github-pull.sh        从 GitHub 拉取
│   └── github-push.sh        推送到 GitHub
├── docs/                     设计与实现文档
│   ├── DESIGN.md             产品总纲（目标、威胁、范围、排期）
│   ├── ARCHITECTURE.md       架构逻辑（控制面/执行面、状态机、主路径）
│   ├── PAGE-DESIGN.md        页面设计（信息架构、线框、交互规则）
│   ├── IMPLEMENTATION.md     实现列表（阶段与验收项）
│   └── SPIKE-3D3P.md         可选：3 人 × 3 天竖切 Spike
└── prototype/
    └── index.html            可点击 UI 示意
```

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
| 1 | [docs/DESIGN.md](./docs/DESIGN.md) | 做什么、不做什么、威胁与原则、人周 |
| 2 | [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md) | 控制面 / 执行面、Harbor Gate、会话与出站路径 |
| 3 | [docs/PAGE-DESIGN.md](./docs/PAGE-DESIGN.md) | HOME / 新会话 / 工作台 / 模型 / 工具 / 设置 |
| 4 | [docs/IMPLEMENTATION.md](./docs/IMPLEMENTATION.md) | 分期任务与验收 |
| 可选 | [docs/SPIKE-3D3P.md](./docs/SPIKE-3D3P.md) | 短周期证明「密钥不进沙箱」的竖切 |

## 产品要点（摘要）

- **模型**：自由添加名称 + API 地址 + Key；新会话 / 工作台用下拉选已添加项。
- **工具**：MCP；浏览器登录 / SSO 或 API Key；列表统一「编辑 / 删除」。
- **会话**：同时 running ≤ 2；同工作区禁止双开。
- **安全叙事**：凭证只在本机；沙箱经 Gate 出站，明文密钥不进执行面。
- **隔离落地**：**一期**工作区 + 进程降权（挡 reboot 靠降权，不靠目录）；**二期**换容器 Runtime。详见 [IMPLEMENTATION.md](./docs/IMPLEMENTATION.md) 与 [DESIGN.md](./docs/DESIGN.md) §7.4 / §7.6。

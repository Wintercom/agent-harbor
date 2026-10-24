# 两天版实施决策

已定：Go 主业务、**Wails v3**（[为什么选择 Wails](https://v3.wails.io/zh-cn/quick-start/why-wails/)）、简单 HTML/CSS/JavaScript、Python 验收辅助；单应用、单会话、单任务、direct 演示工作区、宿主机内存凭据；Claude Code 优先，一个固定模型组合和一个远程 Streamable HTTP 认证 MCP。详细边界以 AGENTS.md 为准。

旧仓库曾用 Wails v2；本仓库已按 v3 Service / `frontend/bindings` / `build/config.yml` 迁移。接口基线为 [CONTRACT.md](CONTRACT.md) v1，模块与并行执行规则为 [MODULE-DESIGN.md](MODULE-DESIGN.md)。Go module path：`github.com/Wintercom/agent-harbor`。

| 开工首小时登记项 | 负责人 | 当前状态 / 证据要求 |
|---|---|---|
| 目标 macOS 版本、CPU、隔离后端可用性 | A | 待验证，记录真实 profile 正/负例；非 root 不等于已隔离 |
| Go / Wails / Node / 前端工具及 MCP SDK 的确切版本 | B | **已初始化**：Go toolchain 1.26（`go.mod`）、Wails **v3.0.0-beta.34**（`wails3 version`）、Vite **4.5.x**、Node ≥16（建议 18+）；MCP 仍用标准 `net/http` 转发，完整 SDK 待真实上游验证后补记 |
| Claude Code 版本、安装方式及必要只读运行路径 | A | 待检测；不得复制完整用户 HOME/登录配置 |
| demo-model 的真实提供方、协议、HTTPS origin 与允许路径 | B | profileId 固定 `demo-model`；默认 origin `https://api.deepseek.com`，可用 `HARBOR_DEMO_MODEL_ORIGIN` 覆盖；真实账号与路径待联调验证 |
| demo-mcp 的真实服务、认证方式、origin、路径、只读工具名 | B/C | profileId 固定 `demo-mcp`；默认占位 origin，可用 `HARBOR_DEMO_MCP_ORIGIN` 覆盖；待选真实服务与账号 |
| 专用非敏感工作区与一条稳定演示任务 | C/A | 待准备；明确会直接改原文件，MCP 操作只读 |
| 依赖安装、绑定生成、Go/前端独立测试和 Wails 开发命令 | B | **已可执行**（见根 README）：`go test ./internal/...`、`wails3 generate bindings ./...`、`frontend` 下 `npm run build`、`wails3 dev` / `wails3 build` |

这些未决项阻碍真实端到端验收，但不阻碍基于契约、固定 profileId 和替身进行独立模块开发。实际服务不兼容时保留失败证据，调整受支持组合，不偷偷更改“所有 Agent/MCP 已支持”的产品口径。

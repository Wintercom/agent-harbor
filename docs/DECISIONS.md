# 两天版实施决策

已定：Go 主业务、Wails v2、简单 HTML/CSS/JavaScript、Python 验收辅助；单应用、单会话、单任务、direct 演示工作区、宿主机内存凭据；Claude Code 优先，一个固定模型组合和一个远程 Streamable HTTP 认证 MCP。详细边界以 AGENTS.md 为准。

接口基线为 [CONTRACT.md](CONTRACT.md) v1，模块与并行执行规则为 [MODULE-DESIGN.md](MODULE-DESIGN.md)。首次创建 Go module 建议使用 `github.com/liyufei916/agent-harbor`，与仓库地址保持一致；依赖版本需真实构建后记录，不把检查文档的机器版本当团队统一环境。

| 开工首小时登记项 | 负责人 | 当前状态 / 证据要求 |
|---|---|---|
| 目标 macOS 版本、CPU、隔离后端可用性 | A | 待验证，记录真实 profile 正/负例；非 root 不等于已隔离 |
| Go / Wails / Node / 前端工具及 MCP SDK 的确切版本 | B | 待构建验证；记录版本和初始化命令 |
| Claude Code 版本、安装方式及必要只读运行路径 | A | 待检测；不得复制完整用户 HOME/登录配置 |
| demo-model 的真实提供方、协议、HTTPS origin 与允许路径 | B | 待选择/验证；profileId 固定为 demo-model，真实地址只配在宿主机受信配置 |
| demo-mcp 的真实服务、认证方式、origin、路径、只读工具名 | B/C | 待选择/验证；profileId 固定为 demo-mcp；需要合法测试账号，不记录真实凭据 |
| 专用非敏感工作区与一条稳定演示任务 | C/A | 待准备；明确会直接改原文件，MCP 操作只读 |
| 依赖安装、绑定生成、Go/前端独立测试和 Wails 开发命令 | B | 待工程初始化；尚无可执行命令，不编造“已通过” |

这些未决项阻碍真实端到端验收，但不阻碍基于契约、固定 profileId 和替身进行独立模块开发。实际服务不兼容时保留失败证据，调整受支持组合，不偷偷更改“所有 Agent/MCP 已支持”的产品口径。

## 2026-10-24 开发环境观测

A 在 macOS 26.7 arm64、Go 1.27.2 下完成契约与编排测试，检测到 Claude Code 2.1.292 原生安装。为独立编译加入最小 go.mod（module 沿用上文建议值，暂定 go 1.23.0，尚未在该最低版本运行验证）；B 继续负责 Wails 初始化与版本冻结。Seatbelt 初探有正/负例，但生产限制与进程树清理仍待实现。观测不等于团队环境冻结或真实 Agent 兼容性通过，详细证据见 [研发 A 开发记录](DEV-A.md)。

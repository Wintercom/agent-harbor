# 会话策略

负责人：B。将用户已确认的资源转换为不可随 Agent 请求改变的会话授权快照，不负责网络转发或系统隔离执行。

## 文件

| 文件 | 职责 |
|---|---|
| `policy.go` | NormalizeWorkspace、Build → PolicySnapshot |
| `profiles.go` | 固定 `demo-model` / `demo-mcp` 受信配置 |
| `policy_test.go` | 工作区拒绝、路径前缀、符号链接 |

两天版不引入动态扩权、长期策略库或通用表达式引擎。必须显式登记 `HARBOR_DEMO_MODEL_ORIGIN`、`HARBOR_CLAUDE_MODEL`、`HARBOR_DEMO_MCP_ORIGIN`、`HARBOR_DEMO_MCP_TOOLS`，生产仅允许 HTTPS origin；未登记失败关闭。协议与固定路径见根 README，不接受前端任意 URL。

精确输入 PolicyRequest、输出 PolicySnapshot 见 [CONTRACT.md](../../docs/CONTRACT.md)。

# 会话策略

负责人：B。将用户已确认的资源转换为不可随 Agent 请求改变的会话授权快照，不负责网络转发或系统隔离执行。

## 文件

| 文件 | 职责 |
|---|---|
| `policy.go` | NormalizeWorkspace、Build → PolicySnapshot |
| `profiles.go` | 固定 `demo-model` / `demo-mcp` 受信配置 |
| `policy_test.go` | 工作区拒绝、路径前缀、符号链接 |

两天版不引入动态扩权、长期策略库或通用表达式引擎。可用环境变量 `HARBOR_DEMO_MODEL_ORIGIN`、`HARBOR_DEMO_MCP_ORIGIN` 覆盖探测用 origin（不提交真实凭据）。

精确输入 PolicyRequest、输出 PolicySnapshot 见 [CONTRACT.md](../../docs/CONTRACT.md)。

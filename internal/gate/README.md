# Harbor Gate

负责人：B。宿主机内的模型反向代理与 MCP Gateway；只开放回环数据入口，不提供 HTTP 管理或凭据读取接口。

## 文件

| 文件 | 职责 |
|---|---|
| `gate.go` | Start、Revoke、会话绑定、监听器及在途请求管理 |
| `model_proxy.go` | 固定模型 HTTPS 上游的流式反向代理，代填模型凭据 |
| `mcp_gateway.go` | MCP 入口转发、Sentinel 校验与工具白名单 |
| `sentinel.go` | 至少 32 字节安全随机令牌的生成与校验 |
| `connections.go` | ConnectModel / ConnectMcp：先探测再写入 Secrets |
| `gate_test.go` | 反代改写、撤销、连接探测 |

## 依赖与边界

接收编排层传来的会话 Policy 和账号引用，在授权后向 Secrets 解析真实凭据。不反向导入 orchestrator 包；通过注入的 audit / fatal 回调报告故障和审计。

- Sentinel 是会话能力而非上游真实 Key；未知、错误或已撤销令牌必须拒绝。
- 固定上游 origin、路径和工具挂载列表由 Policy 决定。
- 模型使用显式 Base URL 反代，不实现通用 CONNECT 或 HTTPS MITM。
- 审计只含 sessionId、目标主机、工具名和结果，不含认证头、Sentinel、完整正文。

## 对接契约

[CONTRACT.md](../../docs/CONTRACT.md) 定义 GateSpec、GateBinding、Connections。装配见根目录 `app.go` / `WireSessionService`。并行边界见 [MODULE-DESIGN.md](../../docs/MODULE-DESIGN.md)。

# Harbor Gate

负责人：B。宿主机内的模型反向代理与 MCP Gateway；只开放回环数据入口，不提供 HTTP 管理或凭据读取接口。

## 建议文件（尚未实现）

| 文件 | 职责 |
|---|---|
| `gate.go` | Start、Revoke、会话绑定、监听器及在途请求管理 |
| `model_proxy.go` | 已适配模型协议的流式反向代理，向固定 HTTPS 上游代填模型凭据 |
| `mcp_gateway.go` | 基于官方 SDK 的初始化、工具发现、调用、结果及取消转发 |
| `sentinel.go` | 至少 32 字节安全随机令牌的生成、验证与撤销 |

## 依赖与边界

接收编排层传来的会话 Policy 和账号引用，在授权后向 Secrets 解析真实凭据。不反向导入 orchestrator 包；通过注入的事件回调报告故障和审计。

- Sentinel 是会话能力而非上游真实 Key，绑定会话及账号引用；未知、错误或已撤销令牌必须拒绝。
- 固定上游 origin、路径和工具挂载列表由 Policy 决定；Agent 的 Host、URL、重定向不能改变凭据目的地。
- 模型使用显式 Base URL 反代，不实现通用 CONNECT 或 HTTPS MITM；保留流式响应、取消与超时。
- MCP 使用真实协议和 SDK，两天只挂一个已验证的远程认证 MCP，不把普通 REST 包装成假 MCP。
- 验证上游 TLS，不转发客户端认证头给未知目标，不自动重试非幂等调用。
- 审计只含 sessionId、目标主机、工具名和结果，不含认证头、Sentinel、完整请求/响应正文。

## 验收

覆盖正常流式模型调用、真实 MCP 发现及调用、错令牌、撤销、重定向、认证失败、超时和取消。受控上游测试与真实服务验收分开记录；Gateway 失败应通知编排层终止受影响会话。

## 对接契约

[CONTRACT.md](../../docs/CONTRACT.md) 定义 GateSpec、GateBinding、Connections：Start 只有监听器 ready 才返回，包含精确 AllowedTCP；部分失败必须关闭已创建资源。Revoke 即使遇到清理超时也先本地撤销令牌，再取消调用/关闭监听器。模型与 MCP 的连接服务先测试候选凭据、成功后保存账号；AccountCatalog 只读元信息提供给编排层。并行测试与文件归属见 [MODULE-DESIGN.md](../../docs/MODULE-DESIGN.md)。

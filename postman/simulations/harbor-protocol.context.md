# harbor-protocol

2026-10-25，依据 docs/CONTRACT.md 构造本机协议 fixture；未查找供应商发布的 mock，未连接真实供应商。

## Purpose

让真实 Connections HTTP 客户端访问独立进程，验证模型消息探测、MCP 握手与认证失败不保存。唯一成员为 mocks/harbor-upstream；具体请求及响应见其 default.js。

## Execution

启动：`postman simulation run postman/simulations/harbor-protocol.sim.yaml --port auto --output ndjson --no-history`。以 listening 事件中的端口设置 `HARBOR_TEST_SIMULATION_ORIGIN=http://127.0.0.1:<port>`，执行 `go test -race ./internal/gate -run TestPostmanProtocolSimulation -count=1 -v`。观察 request 事件后 Ctrl-C 停止。Go 测试只对本机显式启用 HTTP，生产环境变量不能开启此例外。

## Assumptions

2026-10-25 实测 CLI 1.75.0 可本地运行，无需发布或绑定 workspace。此 fixture 只实现固定 Anthropic 响应及 MCP 2025-03-26 子集；不能证明真实凭据、供应商协议或 Agent 能力。环境变量未设置时可选 Go 测试明确 skip，普通 CI 的同类路径由 httptest 回归覆盖。

## Observations

<!-- begin:observations -->
2026-10-25：唯一 upstream 成员实际收到模型消息、MCP initialize/initialized/tools/list/DELETE，以及错误认证请求；Go 测试通过，没有静默未接线成员。日志在本机 /tmp/harbor-fix-simulation.ndjson，不提交请求日志。
2026-10-25 dsh 增量：固定模型前缀 `/upstream/anthropic` 经模拟网关转发到唯一成员 `/anthropic/v1/messages`，实际 POST 返回 200；原模型、MCP 握手与认证失败回归通过。日志在本机 /tmp/harbor-dsh-simulation.ndjson；本轮参考官方源码和已发布 npm 包确定路径，Orbit 搜索未取得适用资源，不复用无关服务。fixture 不代表 DeepSeek 远程验收。
<!-- end:observations -->

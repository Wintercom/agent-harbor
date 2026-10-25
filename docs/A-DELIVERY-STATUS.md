# 研发 A 交付与验收状态（2026-10-25）

本页只列当前证据。`go test` 中的 HTTP fixture、假 Kit 与桌面 mock 均不能替代真实供应商验收；未通过项维持开放。

| 任务 | 当前结果 | 可复现证据 | 仍需完成 |
|---|---|---|---|
| P1-A-01 多会话 | 编排实现并验证 | `go test -race ./internal/orchestrator`；双会话上限、同工作区互斥、列表/销毁、停止全部活动会话 | 与 C 多会话工作台人工联调 |
| P1-A-02 clone | Runtime 实现并验证 | `go test -race -tags harbor_platform ./internal/runtime`；副本独立、原目录拒读、清理删除 | C 的 direct/clone 警告与展示人工验收 |
| 0–8 MCP 挂载 | Policy/Gate/Claude/dsh 实现并验证 | `go test -race ./internal/policy ./internal/gate ./internal/kits/...`；独立账号、工具白名单和 0 MCP | 真实远程认证 MCP 的多挂载互操作 |
| P1-A-03 dsh | 固定版本真进程在沙箱内通过 Gate 完成受控任务；两轮同实例历史已验证 | `HARBOR_TEST_DSH=1 HARBOR_DSH_NODE=… HARBOR_DSH_INSTALLATION=… go test -race ./internal/kits/dsh` | 真模型、真 MCP、桌面长会话人工验收 |
| P1-A-03 Codex | **未完成**；全量分支未注册 Codex，不能宣称支持 | 旧分支实验实现需按当前契约重新移植与复核 | 固定 CLI/Responses 协议、私有 HOME、受控子进程和真实验收 |
| P1-A-04 子进程 | 严格禁止 fork/posix_spawn，失败关闭；未达到允许子进程门禁 | `go test -race -tags harbor_platform ./internal/runtime`，含原生 `POSIX_SPAWN_SETPGROUP` 负例 | 选定可信隔离后端，证明脱离进程组的后代可追踪和清理，设计评审后再开放 |
| P1-A-05 扩权 | 会话决策生命周期及真实 Gate 重定向挂起/拒绝通过 | `go test -race ./internal/orchestrator -run 'TestPermission|TestGateRedirectPermission'` | B 完成审批后重新求值、DNS/IP 校验、once 消耗、project 持久化与撤销；不能凭当前事件认为已放行 |
| P1-A-08 内嵌 PTY | OS PTY、resize、回显、中文输入、取消与关闭已验证 | `go test -race -tags harbor_platform ./internal/runtime -run 'TestRealTerminal|TestTerminalInput'` | 真 Agent TUI 模式及 C 工作台 live 人工联调；Claude JSONL/dsh 会话消息仍不是终端字节流 |
| P1-A-06 安全证据包 | 平台探针和脱敏证据格式校验齐备；真实 I1–I11 未关闭 | `go test -race -tags harbor_platform ./internal/runtime ./internal/orchestrator`；`python3 -m unittest discover -s security -p 'test_*.py'` | 真实模型和认证 MCP 凭据在宿主机录入后，按 [验收清单](ACCEPTANCE.md) 逐项生成脱敏报告 |

本机 `docker` 客户端可执行，但当前没有运行中的 Docker daemon。不能把存在命令行客户端当作已具备容器隔离或可靠后代清理。microVM 为后续里程碑，不计本分支 P0/P1 完成条件。

真实联调前需明确 Agent、模型名与 API Origin/BasePath、远程 MCP 地址与批准工具名；凭据只在应用宿主机配置页或安全环境中提供，不能写入本页、issue 或测试日志。受控上游已证明协议主路径，尚未提供真实组合的验收结论。

本批已完成的工程回归：`go test -race ./...`、`go vet ./...`、平台 Runtime/Orchestrator 测试、固定版 Claude 离线检测、固定版 dsh 全套受控上游测试（含两轮）、前端测试、Wails 绑定生成/构建/打包、Info.plist 校验、ad-hoc 签名验证及原生 WebView 绑定/退出冒烟均通过。桌面冒烟不是上述待办中人工业务验收的替代证据。

# Wails v3 兼容性评估

2026-10-25，基于源仓库 `2fa1d1d` 和本轮修复。现有模块具备采用 Wails v3 的技术条件；已验证编译、绑定生成、前端与桌面生产构建，尚未完成 WebView 运行验收。本文不替代此前 v2 选型决策。

| 模块 | 验证结论 | 剩余项 |
|---|---|---|
| contracts | 无框架依赖，v3 生成 13 个模型、1 个事件 | WebView 数据往返 |
| orchestrator | race 和真实 Runtime 集成测试通过 | 窗口退出触发清理 |
| runtime | harbor_platform 系统测试通过 | 仍限既定 Darwin 25/arm64，禁止子进程 |
| kits | 检测、启动规格测试通过 | 真实模型/MCP 任务 |
| gate/policy/secrets | 无框架依赖，race 测试通过 | 真实认证上游验收 |
| desktop/main | v3 Service、事件注册及构建通过，生成 8 个方法 | 目录选择、事件展示、退出清理 |
| frontend | 修复 live 绑定打包，JS/TS 两种生成模式均可构建 | 实际 WebView live 调用 |
| build/CI | 增加固定版本 CLI、绑定与桌面构建检查 | 签名、公证不在本轮范围 |

原 api.js 使用 @vite-ignore 动态导入 desktopapi.js，生产构建既不打包该文件，也不会发现绑定缺失。现在延迟加载 live-api.js，由 Vite 解析无扩展名的生成绑定路径，同时兼容默认 JS 和 Taskfile 的 TS 输出。前端 runtime 固定为 3.0.0-beta.34，与 Go、CLI 一致。

本机通过：go test -race ./...、go vet ./...、go test -race -tags harbor_platform ./internal/runtime ./internal/orchestrator、JS 绑定后的前端构建，以及 wails3 build 的 TS 绑定与 macOS 生产构建。未证明最低 Go 版本、其他操作系统或实际 GUI 的运行兼容。

切换前仍需确认：真实 WebView 方法与事件、运行中退出及清理失败、模型与认证 MCP 核心任务。当前前端 live 加载失败会回退 mock，验收必须确认真实绑定标记，不能将 mock 当作成功。DECISIONS、MVP、MODULE-DESIGN 和 README 仍存在 v2/v3、生成目录与 Go 版本冲突，须在正式切换时统一。

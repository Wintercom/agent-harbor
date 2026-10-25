# CLAUDE.md

@AGENTS.md

当前技术栈、三人分工与两天交付范围以 AGENTS.md 的「当前开发基线」为准；[核心架构与模块划分](docs/MVP-2D3P.md) 已同步 Go/Wails 和新分工。

最新结构见 [Go/Wails 目录设计](docs/DIRECTORY-STRUCTURE.md)；模块说明见 [Go 后端](internal/README.md) 和 [桌面前端](frontend/README.md)；旧 src 骨架已移除。A 已落地契约、编排、严格单进程 macOS Runtime 与 Claude Kit。B 已接上 Gate、Policy、内存凭据、Wails v3 绑定和真实事件通道；真实模型/MCP 上游仍待联调。子进程执行和可靠后代清理尚不支持。

并行开发先读 [模块设计](docs/MODULE-DESIGN.md)，接口统一按 [CONTRACT.md](docs/CONTRACT.md) 实现，实际验证按 [ACCEPTANCE.md](docs/ACCEPTANCE.md) 记录。A 的测试替身、B 的桌面桥测试和 C 的浏览器 mock 已落地，不能当作真实验收。当前实现与后续步骤见 [研发 A 开发记录](docs/DEV-A.md)。

## Claude Code 专用规范

Wails v3 的逐模块验证范围与剩余验收见 [兼容性评估](docs/WAILS-V3-ASSESSMENT.md)；构建通过不代表真实桌面闭环通过。

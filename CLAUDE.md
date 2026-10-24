# CLAUDE.md

@AGENTS.md

当前技术栈、三人分工与两天交付范围以 AGENTS.md 的「当前开发基线」为准；[核心架构与模块划分](docs/MVP-2D3P.md) 已同步 Go/Wails 和新分工。

最新结构见 [Go/Wails 目录设计](docs/DIRECTORY-STRUCTURE.md)；模块说明见 [Go 后端](internal/README.md) 和 [桌面前端](frontend/README.md)；旧 src 骨架已移除。A 已落地契约、编排、严格单进程 macOS Runtime 与 Claude Kit，真实模型/MCP 和桌面集成仍待验证；子进程执行和可靠后代清理尚不支持。

并行开发先读 [模块设计](docs/MODULE-DESIGN.md)，接口统一按 [CONTRACT.md](docs/CONTRACT.md) 实现，实际验证按 [ACCEPTANCE.md](docs/ACCEPTANCE.md) 记录；A 的测试替身已实现，B/C 工程与替身仍待接续；当前实现、测试证据和后续步骤见 [研发 A 开发记录](docs/DEV-A.md)。

## Claude Code 专用规范

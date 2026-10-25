# 设计文档导航

当前目录结构按 [Go/Wails 目录设计](DIRECTORY-STRUCTURE.md) 组织，开发范围按[三人两天核心方案](MVP-2D3P.md)执行；完整产品文档保留长期目标，不将尚未实现的能力视为已交付。

| 文档 | 用途 |
|---|---|
| [MODULE-DESIGN.md](MODULE-DESIGN.md) | 开发入口：当前模块边界、依赖、资源与文件归属、替身、并行检查点 |
| [CONTRACT.md](CONTRACT.md) | 准确接口、DTO、响应、事件、错误和生命周期的唯一来源 |
| [DECISIONS.md](DECISIONS.md) | 已定实施选择、待验证环境和真实接入组合 |
| [ACCEPTANCE.md](ACCEPTANCE.md) | 独立开发检查、真实集成/安全步骤与证据格式 |
| [DIRECTORY-STRUCTURE.md](DIRECTORY-STRUCTURE.md) | 当前 Go/Wails 目录、三人职责、依赖与迁移映射 |
| [MVP-2D3P.md](MVP-2D3P.md) | 当前优先阅读：两天范围、架构、分工、接口、排期和验收 |
| [TASK-SPLIT-ABC.md](TASK-SPLIT-ABC.md) | 对照第一版设计 × 当前实现的 A/B/C 任务拆分（P0 两天 / P1 V1） |
| [DEV-A.md](DEV-A.md) | 研发 A：契约/编排/Runtime/Kit 实现与证据 |
| [DEV-B.md](DEV-B.md) | 研发 B：Gate/Policy/Secrets 与 Wails v3 桌面桥 |
| [DEV-C.md](DEV-C.md) | 产品 C：前端 mock 两页、D4 场景与接入说明 |
| [DESIGN.md](DESIGN.md) | 产品定位、威胁模型与长期范围 |
| [ARCHITECTURE.md](ARCHITECTURE.md) | 控制/执行面、组件边界与完整会话契约 |
| [PAGE-DESIGN.md](PAGE-DESIGN.md) | 完整桌面页面与交互设计 |
| [IMPLEMENTATION.md](IMPLEMENTATION.md) | P1→P2 长期实现顺序和验收 |
| [SPIKE-3D3P.md](SPIKE-3D3P.md) | 历史三天探索方案；其中不做 MCP 等缩减不适用于当前两天核心闭环 |

目录与代码边界入口：[Go 后端](../internal/README.md)、[桌面前端](../frontend/README.md)、[安全验收](../security/README.md)、[演示流程](../demos/README.md)。

新增长篇设计优先放在此目录并添加导航；模块局部说明放对应目录 README。通用代理规范维护在根 AGENTS.md，CLAUDE.md 保持精简引用，不重复复制架构全文。

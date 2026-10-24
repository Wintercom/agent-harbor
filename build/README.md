# Wails 构建资源

负责人：研发 B。本目录当前仅有说明；Wails 工程初始化后按脚手架实际布局补充图标、应用描述及平台构建资源。

- 可提交图标和必要的平台构建配置，不将整个 build 目录忽略。
- build/bin/ 是后续生成的可执行文件输出，不提交版本控制。
- frontend/dist/ 和 frontend/wailsjs/ 由构建或绑定生成，不手工维护；忽略规则见根 .gitignore。
- Go 及前端依赖统一通过根 go.mod 和 frontend/package.json 维护，当前尚未创建。
- 首次初始化需验证 Wails v2、Go、Node、目标 macOS/CPU 的兼容性，并记录可复现的开发/构建命令；不能把文档骨架当作已经可运行。

前端开发由 C 承担，构建链、Go 绑定和交付机器验证由 B 承担。结构依据见[目录设计](../docs/DIRECTORY-STRUCTURE.md)。

# Wails 构建资源

负责人：研发 B。本目录来自 Wails v3 脚手架（`wails3 init`），含图标、平台 Taskfile 与 `config.yml`。

- 可提交图标和平台构建配置；`build/bin/` 与仓库根 `bin/` 为输出，不提交。
- `frontend/dist/`、`frontend/bindings/` 由构建或 `wails3 generate bindings` 生成，不手改。
- 开发/构建命令见根 README；版本登记见 [DECISIONS.md](../docs/DECISIONS.md)。

前端页面由 C 开发；构建链、Go 绑定与交付机器验证由 B 承担。

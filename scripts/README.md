# 仓库维护脚本

此目录只管理开发和仓库维护自动化；不承担 Agent 启动、隔离策略或 Gate 业务逻辑。

| 现有脚本 | 用途 |
|---|---|
| [dev-desktop.sh](dev-desktop.sh) | **全量 PAGE**：`wails3 generate` + `wails3 dev` 开桌面窗（上游在页面「模型/工具」配置） |
| [github-pull.sh](github-pull.sh) | GitHub 仓库拉取与同步 |
| [github-push.sh](github-push.sh) | GitHub 提交及推送辅助 |
| [push-to-wintercom.sh](push-to-wintercom.sh) | 现有推送辅助，使用前阅读脚本具体行为 |

```bash
./scripts/dev-desktop.sh
# 仅 mock 看页：cd frontend && npm run dev
```

调用与环境变量见[根 README](../README.md#github-推送--拉取)。执行前检查 Git 状态和脚本内容；提交、推送与改远程会影响仓库或共享状态，不作为运行应用的前置步骤。

后续启动/构建命令由根 package 配置统一提供，演示脚本放 [demos](../demos/README.md)，跨模块安全检查放 [security](../security/README.md)。禁止在脚本中内置真实 Key、Password 或个人路径默认值。

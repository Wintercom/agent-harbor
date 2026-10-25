# 页面视图

按 [PAGE-DESIGN](../../docs/PAGE-DESIGN.md) 壳装配；两天契约方法仍走 setup / session。

| 文件 | 状态 | 内容 |
|---|---|---|
| [onboarding.js](onboarding.js) | stub | 首次引导；隔离失败不可进 |
| [home.js](home.js) | stub | 会话中心（单会话历史） |
| [wizard.js](wizard.js) | stub | 新会话步进 + 内嵌 setup |
| [models.js](models.js) / [tools.js](tools.js) / [settings.js](settings.js) | stub | 独立配置入口 |
| [block.js](block.js) | stub | 隔离不可用阻断 |
| [setup.js](setup.js) | 已实现 | 连接/工作区/任务/启动；可 `embedded` |
| [session.js](session.js) | 已实现 | 状态、文本输出、审计、停止 |

凭据仅在模型/工具/配置流出现，提交后清空。输出用 textContent。不手改 bindings。

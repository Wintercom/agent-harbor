# 手写前端源码

负责人：产品 C。

| 文件 | 状态 | 职责 |
|---|---|---|
| [main.js](main.js) | 已实现 | 页面装配与切页 |
| [mock-api.js](mock-api.js) | 已实现 | mock transport + 固定场景 |
| [api.js](api.js) | 已实现 | 对外 API；可切换 live |
| [styles.css](styles.css) | 已实现 | 共用样式 |
| [views](views/README.md) | 已实现 | 配置与工作台 |

视图只调用 api.js。方法和事件见 [CONTRACT.md](../../docs/CONTRACT.md)。

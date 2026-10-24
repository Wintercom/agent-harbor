# 手写前端源码

负责人：产品 C。Go/Wails 绑定与构建问题由研发 B 配合。

| 建议文件（尚未实现） | 职责 |
|---|---|
| main.js | 页面装配、订阅状态/输出事件及注销监听 |
| mock-api.js | 浏览器独立的同契约替身，正常/认证失败/占用/隔离失败/清理失败场景 |
| api.js | 封装 Wails 生成绑定与事件；可先接无敏感模拟数据 |
| styles.css | 两个页面共用样式 |
| [views](views/README.md) | 配置与运行工作台 |

视图只调用 api.js，不各自复制绑定逻辑、错误码或业务状态机。Go 端负责实际状态、账号引用和权限校验，前端只展示状态并提交操作。方法和事件的数据依据见 [contracts](../../internal/contracts/README.md)。

本目录不放生成绑定、运行时凭据、真实日志或构建产物。父级[前端说明](../README.md)约定完整数据边界。

准确接口见 [CONTRACT.md](../../docs/CONTRACT.md)，mock 数据、替换与文件归属见 [MODULE-DESIGN.md](../../docs/MODULE-DESIGN.md)。

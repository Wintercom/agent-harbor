# Wails 桌面前端

负责人：产品 C 开发页面，研发 B 提供 Wails 绑定、事件和构建环境。当前只有目录说明，没有前端实现、依赖或生成代码。

## 目录与建议文件

| 路径 | 职责 |
|---|---|
| index.html（待实现） | 桌面页面入口 |
| package.json（待实现） | 统一前端依赖、开发及构建命令 |
| [src](src/README.md) | 手写页面、样式及后端调用封装 |
| wailsjs/（后续生成） | Wails v2 生成绑定与模型，不手改 |
| dist/（后续生成） | 构建产物，供 Go 桌面入口嵌入，不手写 |

采用简单 HTML/CSS/JavaScript，不强制引入复杂前端框架。只做配置与工作台，不做完整 PTY、文件编辑器或双会话管理。完整 UI 参考[页面设计](../docs/PAGE-DESIGN.md)，交付范围以[两天方案](../docs/MVP-2D3P.md)为准。

## 调用与安全

页面通过 src/api.js 调用生成的 Go 业务绑定，不直接请求 Gate 或上游，不使用 Node/系统特权 API。app.go 只暴露固定业务方法，不暴露任意命令执行、文件读取、原始服务对象或通用消息通道。

连接表单在宿主机填写真实凭据，提交后清空，不存入 localStorage、URL 或日志。状态以 Go 事件为准，输出按文本展示，不执行 HTML。启动前订阅事件，页面离开时取消订阅。

C 可先使用无敏感信息的模拟事件开发；B 提供绑定后替换 api.js 内的适配，不将模拟成功当作真实验收。测试覆盖空输入、连接/启动失败、流式输出、停止和再次运行；明确提示 direct 修改原文件及重启后重新连接。

## 与研发并行

准确方法、响应、harbor:event 载荷及 seq 规则见 [CONTRACT.md](../docs/CONTRACT.md)。C 先按 [MODULE-DESIGN.md](../docs/MODULE-DESIGN.md) 的固定 mock 场景实现浏览器独立 transport，B 提供绑定后只替换 api.js 适配；mock 模式须有标记且不能进入真实验收。当前 transport 和 mock 尚未实现。

# 前后端边界契约

负责人：研发 A 首次落地、B 复核并生成绑定，产品 C 核对页面需求。准确方法、类型、JSON 与语义见 [CONTRACT.md](../../docs/CONTRACT.md)，共享文件规则见 [MODULE-DESIGN.md](../../docs/MODULE-DESIGN.md)。

## 当前文件

| 文件 | 内容 |
|---|---|
| dto.go | Agent 能力、连接请求与脱敏结果、SessionSpec、会话状态 |
| events.go | 可序列化事件载荷与纯数据 Validate 校验 |
| errors.go | AppError 的 error 接口、会话状态和固定事件名 |

Go 定义是边界数据的来源，前端通过 Wails 生成绑定及模型使用，不另手写一套不一致的类型。GateBinding、LaunchSpec、Runtime handle 等内部接口归相应后端包，不进入前端公开数据模型。

## 边界与验收

- 仅包含可序列化结构与纯常量，不导入 Wails、Runtime、Secrets，不执行 IO。
- SessionSpec 包含 workspace、agentId、modelAccountId、mcpIds、task；不允许前端指定任意启动命令或附带真实凭据。
- 凭据连接输入仅单向提交，返回只含账号引用和脱敏状态，不能直接回传输入对象。
- 事件载荷按类型定义，不传递 Sentinel、句柄、认证头或内部错误对象。
- Go 类型及生成绑定不能代替运行时校验；app.go 在调用编排前检查来源、参数和操作状态。
- 变更绑定方法、状态或事件前先更新契约，由 A/B 同步后端、B 生成绑定、C 调整页面。

后续测试验证输入处理、错误脱敏及事件序列化行为，不仅验证类型声明自身。

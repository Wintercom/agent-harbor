# 前后端边界契约

负责人：研发 A 主改、B 复核并生成绑定，产品 C 核对页面需求。准确方法、类型、JSON 与语义见 [CONTRACT.md](../../docs/CONTRACT.md)。

## 当前文件

| 文件 | 内容 |
|---|---|
| dto.go | Agent、连接、会话快照、统一 Response、DesktopAPI |
| events.go | 事件载荷、Validate、错误码与 NewAppError |
| errors.go | AppError 的 error 接口、会话状态和固定事件名 |

字段与语义变更由 A 按契约主改，B 重新生成 `frontend/bindings/`。

GateBinding、LaunchSpec、Runtime handle 等内部类型归相应后端包，不进入本目录。

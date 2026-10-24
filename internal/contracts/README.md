# 前后端边界契约

负责人：研发 A 主改、B 复核并生成绑定，产品 C 核对页面需求。准确方法、类型、JSON 与语义见 [CONTRACT.md](../../docs/CONTRACT.md)。

## 文件

| 文件 | 内容 |
|---|---|
| `dto.go` | Agent、连接、会话快照、统一 Response、Event 载荷 |
| `events.go` | `harbor:event`、错误码常量、NewAppError |

当前 DTO 由研发 B 为桌面绑定可编译先行落地；后续字段/语义变更仍由 A 按契约主改，B 重新生成 `frontend/bindings/`。

GateBinding、LaunchSpec、Runtime handle 等内部类型归相应后端包，不进入本目录。

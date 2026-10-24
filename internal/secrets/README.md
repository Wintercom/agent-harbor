# 宿主机凭据

负责人：B。两天版只使用 Go 应用进程内存，不持久化到文件、数据库、浏览器存储或钥匙串；重启后需要重新连接。

## 文件

| 文件 | 职责 |
|---|---|
| `store.go` | Put / Resolve / Delete / Lookup / Clear |
| `store_test.go` | 引用隔离、删除后解析失败 |

SecretStore.Resolve 能力只注入 Gate；编排层仅获得 AccountCatalog.Lookup（`Lookup` 方法）。应用退出时 `Clear`。

精确语义见 [CONTRACT.md](../../docs/CONTRACT.md)。

# PAGE 屏 ↔ DesktopAPI（v1.1 已定）

| 项 | 内容 |
|---|---|
| 版本 | v1.1 |
| 日期 | 2026-10-25 |
| 总纲 | [SYSTEM-DESIGN](SYSTEM-DESIGN.md) |
| 签名与行为 | [CONTRACT](CONTRACT.md)（唯一实现源） |

---

## 1. 分层（摘要）

UI（C）→ `api.js` → DesktopAPI（B）→ SessionService / Connections / Runtime / Policy（A/B）→ `harbor:event` 回 UI。  
细则与模块所有权见 SYSTEM-DESIGN §2–3。

---

## 2. 页面绑定表

| 页面 | 方法（camelCase） | 事件 | 主责实现 |
|---|---|---|---|
| 壳 / 状态灯 | `getRuntimeHealth`；`getUIPrefs` / `setUIPrefs` | — | A Health · B Prefs · C 壳 |
| Onboarding / 阻断 | `getRuntimeHealth` | — | 同上 |
| HOME | `listSessions` `stopSession` `destroySession` `getRuntimeHealth` | `state_changed` | A 多会话 · C 列表 |
| 新会话 | `listAgents` `selectWorkspace` `listAccounts` `createSession`；可 `connect*` | — | 现有 + A 上限/冲突 |
| 模型 | `connectModel` `listAccounts("model")` `removeAccount` | — | B Connections · C 页 |
| 工具 | `connectMcp` `listAccounts("mcp")` `removeAccount` | — | 同上（Token；无 OAuth） |
| 设置 | `listAgents` `getRuntimeHealth` `listRememberedHosts` `revokeRememberedHost` `get/setUIPrefs` | — | B/C |
| 工作台 | `getSession` `stopSession` `getPermissionOverview` | `state_changed` `output` `audit` `error` `permission_needed` | A/B/C |
| 扩权 Modal | `resolvePermission` | `permission_needed` | A 状态 · B 触发 · C UI |
| 权限一览 | `getPermissionOverview` `listRememberedHosts` `revokeRememberedHost` | — | B 组装 · C UI |

---

## 3. DesktopAPI 一览

| Go | 前端 | V1 |
|---|---|---|
| ListAgents | listAgents | 做 |
| SelectWorkspace | selectWorkspace | 做 |
| ConnectModel / ConnectMcp | connectModel / connectMcp | 做（模型可带 origin） |
| ListAccounts / RemoveAccount | listAccounts / removeAccount | 做 |
| Create / Get / Stop Session | createSession / getSession / stopSession | 做 |
| ListSessions / DestroySession | listSessions / destroySession | 做 |
| GetRuntimeHealth | getRuntimeHealth | 做 |
| GetPermissionOverview | getPermissionOverview | 做 |
| ListRememberedHosts / RevokeRememberedHost | listRememberedHosts / revokeRememberedHost | 做 |
| ResolvePermission | resolvePermission | 做 |
| GetUIPrefs / SetUIPrefs | getUIPrefs / setUIPrefs | 做 |
| — | — | **不做**：OAuth、clone、PTY |

---

## 4. 前端模块

见 [SYSTEM-DESIGN §3.3](SYSTEM-DESIGN.md)。路径：`frontend/src/views/*`、`api.js`、`mock-api.js`、`live-api.js`。

---

## 5. 修订

| 日期 | 说明 |
|---|---|
| 2026-10-25 | 由草案改为 v1.1 定案映射；冻结会流程废除 |

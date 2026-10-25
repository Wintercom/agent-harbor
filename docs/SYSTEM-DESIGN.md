# Agent Harbor — V1 整体架构、模块拆分与分工

| 项 | 内容 |
|---|---|
| 文档版本 | v1.1（复核：与 TASK-SPLIT v1.5 对齐） |
| 日期 | 2026-10-25 |
| 产品屏 | [PAGE-DESIGN](PAGE-DESIGN.md) |
| **接口唯一来源** | [CONTRACT](CONTRACT.md) v1.1 |
| 屏级绑定表 | [PAGE-API-MAP](PAGE-API-MAP.md) |
| 包级细则 / 替身 | [MODULE-DESIGN](MODULE-DESIGN.md) |
| 任务卡 | [TASK-SPLIT-ABC](TASK-SPLIT-ABC.md) |

本文是 **V1 实施总纲**：架构怎么分、模块谁拥有、模块间只经哪些接口交互。字段与方法签名以 CONTRACT 为准；本文不重复粘贴全部 DTO。

---

## 1. 设计命题（不变）

| 平面 | 位置 | 持有真实密钥 |
|---|---|---|
| **控制面** | 宿主机 Go + 桌面 UI | 能（仅 Secrets / Gate） |
| **执行面** | 隔离环境内 Agent | **不能** |

出站（模型 HTTPS、MCP）必须经 Harbor Gate；UI 永不直连上游、不回显 Key。

---

## 2. 逻辑分层

```text
┌─────────────────────────────────────────────────────────────┐
│ L5  体验     PAGE 壳：HOME / 向导 / 模型 / 工具 / 设置 / 工作台 │
│             Onboarding · 扩权 Modal · 阻断 · 权限一览         │
├─────────────────────────────────────────────────────────────┤
│     适配     frontend/api.js ↔ Wails DesktopAPI（B 生成绑定）   │
├─────────────────────────────────────────────────────────────┤
│ L4  编排     Orchestrator：多会话锁、生命周期、扩权挂起、事件 seq │
├─────────────────────────────────────────────────────────────┤
│ L3  控制面   Policy · Secrets · Gate · 账号目录 · UI 偏好文件   │
├─────────────────────────────────────────────────────────────┤
│ L2  隔离     Runtime：Health / Create / Exec / Destroy         │
├─────────────────────────────────────────────────────────────┤
│ L1  执行面   Kit + Agent 进程 + 工作区（无密钥）                │
└─────────────────────────────────────────────────────────────┘
```

依赖铁律：

1. L5 **只**调 DesktopAPI + 订阅 `harbor:event`。  
2. L4 经窄接口用 L3/L2，不把 Secrets.Resolve 交给编排。  
3. L1 只拿 Gate URL + Sentinel。  
4. 新增能力优先扩 DesktopAPI / SessionService / Gate，**不新造平行控制面**。

---

## 3. 模块拆分与所有权

### 3.1 包 / 目录矩阵

| 模块 | 路径 | 负责人 | 唯一职责 | 对外接口（消费方可见） | 不做 |
|---|---|---|---|---|---|
| **contracts** | `internal/contracts` | A 主笔 / B 复核 | 可序列化 DTO、错误码、事件形状 | 纯数据 | IO、状态机 |
| **desktop** | `internal/desktop` + `main` | B | Wails 生命周期、绑定、目录面板、事件桥、偏好落盘 | `DesktopAPI` | 再造会话状态机 |
| **frontend** | `frontend/src` | C | 壳与各页、api/mock、演示文案 | 仅调 `api.js` | 手改 bindings、改 `internal/` |
| **orchestrator** | `internal/orchestrator` | A | 会话表、上限=2、同工作区冲突、Stop/Destroy、扩权挂起、事件 seq | `SessionService`（扩展见 CONTRACT） | 存 Key、解析 MCP 协议 |
| **runtime** | `internal/runtime` | A | 隔离探测 Health、私有 HOME、进程树 | `Runtime` | 账号、UI |
| **kits** | `internal/kits` | A | Detect / Build LaunchSpec | `Kit` | 读 Secrets |
| **policy** | `internal/policy` | B | 工作区、host 白名单、记住主机、mount 校验 | `PolicyBuilder` + 记住主机读写 | 网络转发 |
| **gate** | `internal/gate` | B | 模型反代、MCP、Sentinel、审计回调、扩权触发点 | `Gate` / `Connections` | Agent 生命周期 |
| **secrets** | `internal/secrets` | B | 凭据与账号元信息 | `SecretStore` + `AccountCatalog` | UI 回显 Key |
| **security** | `security/` + A 记录 | A | 真实隔离/泄密验收 | 证据 | 用 mock 冒充通过 |

### 3.2 模块交互（只经接口）

```mermaid
flowchart LR
  UI[C: api.js] -->|DesktopAPI| Desk[B: desktop]
  Desk --> Orch[A: SessionService]
  Desk --> Conn[B: Connections]
  Desk --> Det[A: AgentDetect]
  Desk --> Pref[B: UIPrefsStore]
  Orch --> Pol[B: Policy]
  Orch --> Gate[B: Gate]
  Orch --> Run[A: Runtime]
  Orch --> Kit[A: Kit]
  Orch --> Cat[B: AccountCatalog Lookup]
  Gate --> Sec[B: SecretStore Resolve]
  Orch -->|harbor:event| Desk
  Desk --> UI
```

| 调用 | 允许 | 禁止 |
|---|---|---|
| UI → Go | DesktopAPI 方法 | 直接 import 业务包、读本地 Key 文件当上游 |
| Orch → Secrets | 仅 `AccountCatalog.Lookup` | `Resolve` / `Put` |
| Gate → Secrets | `Resolve` | 把明文放进事件 |
| Kit → Gate | 无；由 Orch 映射 URL/Sentinel | Kit 读 Secrets |
| Runtime → Gate | 无；只消费 AllowedTCP | Runtime 做业务鉴权 |

### 3.3 前端内部模块（C）

| 模块 | 路径 | 职责 |
|---|---|---|
| 壳路由 | `main.js` / `index.html` | 导航、灯、引导、阻断、偏好键 |
| 视图 | `views/*` | 单页 UI；经 `ctx` + `api` |
| 传输 | `api.js` / `mock-api.js` / `live-api.js` | 同名方法；mock∥live 可替换 |
| 事件 | `session-events.js` | seq 合并、输出尾 |

---

## 4. V1 范围裁定（已定）

| 能力 | V1 | 说明 |
|---|---|---|
| PAGE 壳 + Onboarding + 阻断 | **做** | C；健康走 `GetRuntimeHealth` |
| HOME 多会话列表 | **做** | running 上限 **2**；同工作区禁双开 |
| 新会话向导 | **做** | 可先合并配置流；`mcpIds` 允许 0..N（N≤8） |
| 模型自由 OpenAPI | **做** | `ConnectInput.origin` 可选；固定 profile 仍支持 |
| 工具 Token MCP | **做** | 多账号列表/删除 |
| 工具 OAuth | **不做（延后）** | CONTRACT 不暴露 OAuth 方法 |
| 权限一览 + 记住主机撤销 | **做** | 与设置同源 |
| 扩权三键 | **做** | `deny` / `once` / `project`；无「全网允许」 |
| 工作区 clone | **不做（延后）** | `mountMode` 仅接受 `direct` |
| 内嵌 PTY | **不做（延后）** | 工作台继续 `output` 文本事件 |
| UI 偏好 | **做** | 侧栏收缩等；B 落盘；禁止存 Key |
| 设置页 | **做** | 重检 Agent、健康、记住主机入口、关于；无后端的开关不进页 |

P0 闭环（I1–I11）**不因 V1 扩展而废除**：七个原方法语义保持；多会话落地前 Create 仍可按单锁行为，但契约已定义上限=2 与错误码，A 实现时切换。

---

## 5. 接口文档入口

| 内容 | 文档 |
|---|---|
| DesktopAPI 全方法、DTO、事件、错误码、行为表 | **[CONTRACT.md](CONTRACT.md)** |
| 哪一屏调哪些方法 | [PAGE-API-MAP.md](PAGE-API-MAP.md) |
| 后端窄接口（SessionService / Gate / …） | CONTRACT § 后端 + MODULE-DESIGN |

**变更纪律**：改字段先改 CONTRACT → 同步 `internal/contracts` → B `wails3 generate` → C mock → 再写真实现。禁止只改前端猜字段。

---

## 6. ABC 并行分工（按模块）

```text
合同（本文 + CONTRACT）已定
    ├─ A：contracts DTO · Orchestrator 多会话/扩权/Destroy · Runtime.Health 暴露
    ├─ B：DesktopAPI 绑定占位→实现 · Connections 扩展 origin · Policy 记住主机
    │     · 权限一览组装 · UIPrefs 落盘 · generate bindings
    └─ C：mock 全方法 · 壳/页接真字段 · 扩权 Modal · 权限一览 UI · demos
合流：方法名 / JSON / 错误码一致后再换 live
```

| 角色 | 立即开工 | 依赖对方完成才算交付 |
|---|---|---|
| **A** | 扩展 `SessionService`：`List`/`Destroy`/`ResolvePermission`；多会话锁；Runtime.Health 语义供桌面 | Gate 扩权回调真触发（B） |
| **B** | `wails3 generate`；DesktopAPI 去 `UNIMPLEMENTED`；Catalog 列表/删除；Overview；Prefs；origin；Gate→permission 事件 | Orch 多会话（A）联调 HOME/上限 |
| **C** | mock 全方法；HOME x/2；模型 origin UI；工具仅 Token；扩权 Modal；权限一览 | live 真多会话、真扩权 |

**冲突时优先级**：P0 I1–I11 > 上表 V1。

---

## 7. 与长期文档关系

| 文档 | 角色 |
|---|---|
| ARCHITECTURE | 长期命题；V1 **不实现**其中的 PTY/OAuth/clone/二期隔离除非上表列为「做」 |
| DESIGN / PAGE-DESIGN | 产品与交互；实现以本文裁定裁剪 |
| MVP-2D3P | 两天 P0 优先；V1 在 P0 真机证据并行推进时不得冲掉 I1–I11 |

---

## 8. 修订

| 日期 | 说明 |
|---|---|
| 2026-10-25 | 初版定案：取消冻结会依赖；V1 范围与模块所有权一次写死 |
| 2026-10-25 | 复核：明确 Health/Accounts/扩权三段所有权；P0 优先于 V1 |

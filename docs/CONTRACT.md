# 两天版接口契约 v1

状态：供本轮开发直接遵循的设计基线。共享类型与编排实现已落地，真实隔离、Agent/上游兼容性仍未证明；当前进度见 [研发 A 开发记录](DEV-A.md)。A/B 负责将声明落成 Go 类型，C 按本文开发页面；不能各自重命名或猜测字段。与旧文档冲突时，当前接口以本文为准，范围及安全要求以根 AGENTS.md 为准。

## 1. 前端方法、返回与校验

Wails 绑定对象只导出以下业务方法；Go 方法采用 PascalCase，`frontend/src/api.js` 对页面暴露对应 camelCase。页面不得直接使用内部服务。以下代码是一份可独立编译的接口规格，类型按 §4 的归属拆入实际包，不是要建立一个含全部服务的公共包。

```go
package contractspec

import "context"

// 本组 DTO 落在 internal/contracts；APIKey 仅用于单向输入。
type AppError struct {
    Code string `json:"code"`
    Message string `json:"message"`
    Retryable bool `json:"retryable"`
}
type AgentInfo struct {
    ID string `json:"id"`
    Name string `json:"name"`
    Version string `json:"version"`
    Available bool `json:"available"`
    Reason string `json:"reason"`
    ModelProfileID string `json:"modelProfileId,omitempty"` // Agent 对应的受信模型配置
}
type ConnectInput struct {
    ProfileID string `json:"profileId"`
    Name string `json:"name"`
    APIKey string `json:"apiKey"`
}
type AccountInfo struct {
    ID string `json:"id"`
    Kind string `json:"kind"` // model 或 mcp
    Name string `json:"name"`
    ProfileID string `json:"profileId"`
    Host string `json:"host"`
}
type SessionSpec struct {
    Workspace string `json:"workspace"`
    AgentID string `json:"agentId"`
    ModelAccountID string `json:"modelAccountId"`
    MCPIDs []string `json:"mcpIds"`
    Task string `json:"task"`
}
type SessionSnapshot struct {
    SessionID string `json:"sessionId"`
    State string `json:"state"`
    Seq uint64 `json:"seq"`
    Workspace string `json:"workspace"`
    AgentID string `json:"agentId"`
    CleanupComplete bool `json:"cleanupComplete"`
    Error *AppError `json:"error"`
}
type AgentsResponse struct {
    Agents []AgentInfo `json:"agents"`
    Error *AppError `json:"error"`
}
type WorkspaceResponse struct {
    Path string `json:"path"`
    Cancelled bool `json:"cancelled"`
    Error *AppError `json:"error"`
}
type AccountResponse struct {
    Account *AccountInfo `json:"account"`
    Error *AppError `json:"error"`
}
type SessionResponse struct {
    Session *SessionSnapshot `json:"session"`
    Error *AppError `json:"error"`
}
type DesktopAPI interface {
    ListAgents() AgentsResponse
    SelectWorkspace() WorkspaceResponse
    ConnectModel(ConnectInput) AccountResponse
    ConnectMcp(ConnectInput) AccountResponse
    CreateSession(SessionSpec) SessionResponse
    GetSession(string) SessionResponse
    StopSession(string) SessionResponse
}

// Event 仅通过固定事件名 harbor:event 传递，四类载荷恰有一项非 nil。
type OutputData struct {
    Stream string `json:"stream"` // stdout 或 stderr
    Text string `json:"text"`
}
type AuditData struct {
    Target string `json:"target"` // 主机名，不包含 URL 查询串
    Tool string `json:"tool"`
    Result string `json:"result"` // allowed、denied 或 failed
}
type Event struct {
    SessionID string `json:"sessionId"`
    Seq uint64 `json:"seq"`
    Type string `json:"type"`
    State *SessionSnapshot `json:"state"`
    Output *OutputData `json:"output"`
    Audit *AuditData `json:"audit"`
    Error *AppError `json:"error"`
}

// 下列为后端接口规格，不属于 UI DTO，也不得绑定到 Wails。
// 方法的 context 为调用控制；SessionService 内部用应用级 context 派生任务。
type SessionService interface {
    WithConnection(context.Context, func(context.Context) error) error // 后端连接准入，不绑定 Wails
    Create(context.Context, SessionSpec) (SessionSnapshot, error)
    Get(context.Context, string) (SessionSnapshot, error)
    Stop(context.Context, string) (SessionSnapshot, error)
    Shutdown(context.Context) error
}
type AccountCatalog interface {
    Lookup(context.Context, string) (AccountInfo, error)
}
type EndpointRule struct {
    Origin string
    Paths []string
}
type PolicyRequest struct {
    Workspace string
    AgentID string
    Model AccountInfo
    MCP AccountInfo
}
type PolicySnapshot struct {
    Workspace string
    MountMode string
    Model EndpointRule
    MCP EndpointRule
    MCPID string
    Tools []string
}
type PolicyBuilder interface {
    Build(context.Context, PolicyRequest) (PolicySnapshot, error)
}
type GateSpec struct {
    SessionID string
    Policy PolicySnapshot
    ModelAccountID string
    MCPAccountID string
}
type GateBinding struct {
    ModelBaseURL string
    MCPURL string
    Sentinel string
    AllowedTCP []string // 精确的 127.0.0.1:port，供 Runtime 强制限制
}
type Gate interface {
    Start(context.Context, GateSpec, func(AuditData), func(error)) (GateBinding, error)
    Revoke(context.Context, string) error
}
type SecretStore interface {
    Put(context.Context, AccountInfo, []byte) (string, error)
    Resolve(context.Context, string) ([]byte, error)
    Delete(context.Context, string) error
}
type Connections interface {
    ConnectModel(context.Context, ConnectInput) (AccountInfo, error)
    ConnectMcp(context.Context, ConnectInput) (AccountInfo, error)
}
type KitInfo struct {
    Agent AgentInfo
    ReadOnlyPaths []string // 固定 CLI 文件及受信专用安装目录的真实路径；目录只读，不递归授权执行
}
type ConfigFile struct {
    RelativePath string // 只能位于 Runtime 私有 HOME 下
    Content []byte // 只允许 Gate URL、Sentinel 等无上游凭据内容
}
type LaunchSpec struct {
    Command string
    Args []string
    Env map[string]string
    Stdin []byte
    ConfigFiles []ConfigFile
}
type KitInput struct {
    Task string
    PrivateHome string
    ModelBaseURL string
    MCPURL string
    Sentinel string
}
type Kit interface {
    Detect(context.Context) (KitInfo, error)
    Build(context.Context, KitInput) (LaunchSpec, error)
}
type RuntimeSpec struct {
    SessionID string
    Workspace string
    MountMode string
    AllowedTCP []string
    ReadOnlyPaths []string
}
type Handle struct {
    ID string
    PrivateHome string
}
type Exit struct {
    Code int
    Cancelled bool
    Err error
}
type Process interface {
    Wait() Exit // 返回前完成 stdout/stderr 读取与最终输出回调
}
type Runtime interface {
    Health(context.Context) error
    Create(context.Context, RuntimeSpec) (Handle, error)
    Exec(context.Context, Handle, LaunchSpec, func(OutputData)) (Process, error)
    Destroy(context.Context, Handle) error
}
```

### dsh 契约增量（2026-10-25）

`ListAgents` 返回已注册 Claude、Codex 与 dsh，未安装或隔离/版本检测失败时 `available=false`。`modelProfileId` 为可选兼容字段：Claude 使用 `demo-model`，Codex 使用 `codex-model`，dsh 使用 `dsh-model`。页面切换 Agent 后清空当前模型账号引用和未提交 Key，保留工作区/MCP；Policy 仍在后端拒绝跨 Agent 模型账号，不能依赖页面校验。Codex 使用 OpenAI Responses provider，配置只写入私有 `CODEX_HOME`，真实 Key 仍由 Gate 代理替换。dsh 配置及验证见 [DSH-INTEGRATION.md](DSH-INTEGRATION.md)。

`FixedProfile` / `EndpointRule` 新增宿主受信 `BasePath` 字段，仅用于模型路由前缀（如 `/anthropic`），不来自 DTO、任务或沙箱。Gate 先对执行面 `/v1/messages` 做原白名单和 Sentinel 检查，再拼接前缀访问固定 origin；Connections 探测使用同一前缀。前缀拒绝穿越、编码、查询串和片段，origin 继续仅允许主机级 URL。

`ReadOnlyPaths` 可以包含受信专用安装目录；Runtime 规范化并拒绝系统/敏感根、HOME、与工作区相交的目录。目录仅递归读取/映射，明确可执行文件仍需单独授权；不开放 fork。dsh 以私有 `DSH_HOME` 和 `DEEPSEEK_API_KEY=Sentinel` 启动，遥测禁用，任务只走 stdin，配置由 Runtime 在私有 HOME 落盘。

### 页面 API 行为

| 方法 | 正常行为 | 校验、故障与副作用 |
|---|---|---|
| ListAgents | 返回实际检测的列表；不依赖 Gate 可用 | CLI 未装属于 available=false，不伪装支持；检测内部故障才返回 error |
| SelectWorkspace | 调用系统目录面板，返回绝对路径 | 用户取消返回 cancelled=true、path=""、error=null；真正授权仍由 Policy 复核 |
| ConnectModel / ConnectMcp | 10 秒内测试固定上游，成功才保存账号并返回 AccountInfo | profileId 只能来自首小时确认的固定配置；禁止传任意上游 URL；输入不记日志；失败清除候选凭据，返回 account=null |
| CreateSession | 原子占用单会话锁，创建 ID，快速返回 provisioning；后台准备和执行 | 空/不支持的参数同步拒绝且不留资源；路径/能力深度检查可异步失败；request 返回不能取消会话任务 |
| GetSession | 返回给定会话最新状态与 seq | 未知 ID 返回 SESSION_NOT_FOUND；不返回任务正文或凭据；快照在本次应用生命期内保留 |
| StopSession | 快速返回 stopping，随后事件报告清理结果 | provisioning 中可取消；terminal 会话幂等返回原快照；未知 ID 为 SESSION_NOT_FOUND；不等整个清理阻塞 UI |

本版 MCPIDs 长度必须为 1，AgentID 仅允许已验证项；Task 限 1–32768 个 UTF-8 字节。ConnectInput 的 ProfileID/Name/APIKey 均必填，Name 最多 100 个 Unicode 字符，APIKey 最多 16384 字节且不修改其内容。重连创建新 ID，模型和 MCP 引用不可互换。正在准备/运行/清理时拒绝重新连接，避免账号变更影响会话。工具白名单使用固定 profile 中人工选定的工具名，不能把上游新发现的所有工具自动授权。

SessionSnapshot.state 只允许 provisioning、running、stopping、stopped、failed；未创建会话的页面空态不产生 idle 会话。cleanupComplete 在准备/运行/清理中为 false，终态按资源确认结果设置。响应中 AccountInfo.kind 只允许 model/mcp，Host 为无凭据的主机名。ListAgents 的空列表用 [] 而非 null。

Go 绑定返回统一业务 error 结构：成功为 error=null，失败为数据字段 null/空及 AppError。不依赖 Wails 将 Go error 序列化为结构化错误；internal/desktop 负责映射内部 error。连接绑定最多等待连接超时；创建与停止只排入状态机。框架/传输失败由 api.js 转为 INTERNAL_ERROR，不向页面暴露原始异常对象。应用启动和页面恢复时调用 ListAgents，用户选择目录后仍将该路径作为待校验输入。

## 2. 事件、快照与错误

- 全部会话事件只用 `harbor:event`。type 为 `state_changed`、`output`、`audit` 或 `error`，且只有对应载荷非 null。error 事件是操作失败通知，最终状态由 state_changed 表达；普通未授权请求的 denied 审计不必终止会话。
- sessionId 在当前应用中唯一；seq 从 1 开始，由 Orchestrator 串行分配，覆盖全部事件。状态载荷的 seq 与外层相同。GetSession 返回最新水位 seq；snapshot.state 只表示实际状态，不由页面推断。
- api.js 先订阅再创建，暂存尚未拿到 sessionId 的事件，响应后合并；晚到的旧快照不能覆盖更大 seq 的状态。重复事件去重，但输出先按 seq 处理再推进水位，不能仅因快照较新就丢弃未展示的输出。事件接入中断可 GetSession 恢复状态，历史输出不承诺重放。
- 本版只保证当前窗口连续连接的事件交付。状态和错误不可静默丢弃；前端文本使用 textContent。输出流按 UTF-8 解码并限每块 16 KiB；后端与前端各保留最多 1 MiB 输出尾部，截断时显示提示，不改变 Agent 实际 stdout 消费。Runtime 读取不得被慢 UI 无限阻塞。
- public error 的 message 为固定可读模板，不附加原始请求、认证头、环境变量、任务正文或上游异常。已知凭据与 Sentinel 必须在边界输出前过滤，流式过滤需覆盖跨块拆分；不能宣称可识别任意上游敏感内容。

事件示例（不包含真实秘密）：

```json
{"sessionId":"demo-session","seq":2,"type":"state_changed","state":{"sessionId":"demo-session","state":"running","seq":2,"workspace":"/demo/project","agentId":"claude","cleanupComplete":false,"error":null},"output":null,"audit":null,"error":null}
```

| 错误码 | 使用位置 | Retryable |
|---|---|---|
| INVALID_ARGUMENT | 输入/引用种类不合法 | false |
| ACCOUNT_NOT_FOUND | 未连接或不存在的账号引用 | false |
| SESSION_BUSY | 单会话锁被占用、会话中重连 | false |
| SESSION_NOT_FOUND | Get/Stop 未知 ID | false |
| AGENT_UNSUPPORTED | CLI/版本/配置不支持 | false |
| ISOLATION_UNAVAILABLE / WORKSPACE_DENIED | 隔离或工作区校验失败 | false |
| UPSTREAM_AUTH_FAILED | 模型/MCP 凭据或权限错误 | false |
| UPSTREAM_TIMEOUT / UPSTREAM_UNAVAILABLE | 手工重试连接/新任务可恢复的上游故障 | true |
| TARGET_DENIED / INVALID_SENTINEL | 目标/工具/会话能力验证失败 | false |
| AGENT_EXIT_FAILED | 非零退出，无已知更具体错误 | false |
| CANCELLED | 用户取消连接等操作；正常停止不算失败 | false |
| CLEANUP_FAILED / INTERNAL_ERROR | 清理或未知内部故障 | false |

Retryable 只供人选择是否重新操作，不授权自动重发任务或有副作用的 MCP 调用。

## 3. 后端调用与资源语义

### 启动与关闭

1. internal/desktop 校验输入并调用 SessionService.Create；Orchestrator 持有单会话锁，用应用 context 派生 session context，而不是使用已返回的绑定请求 context。
2. Orchestrator 调用 Runtime.Health、Kit.Detect、AccountCatalog.Lookup；Policy.Build 校验真实工作区及固定上游，输出不可变快照。原账号元信息由 B 的连接目录提供，不从 Secrets 读密钥。
3. Gate.Start 创建随机 Sentinel，启动精确绑定 `127.0.0.1` 的模型及 MCP 数据端口并确认 ready，返回 AllowedTCP。失败必须关闭部分已建监听器及上游连接，不留半成品 binding。Start 不再做用户交互或无限期连接测试。
4. Runtime.Create 接收工作区、只读运行文件和 AllowedTCP，创建私有 HOME/隔离规则并确认限制。**本版工作区配置合并进 Create**，不另设 MountWorkspace，以避免调用顺序歧义；后续后端需保持同等授权语义。
5. Kit.Build 只生成无真实凭据的 LaunchSpec；Runtime.Exec 校验相对配置路径、写入私有 HOME（文件 0600、目录 0700），在 profile 内执行，并立即读取输出。写入/启动失败由 Runtime 清理部分子资源，Orchestrator 仍调用 Destroy。
6. Exec 返回进程已成功 spawn 后发布 running；监听进程 Wait 和 Gate 致命回调。Runtime.Health 或 Gate 失效禁止转为宿主机直跑。
7. 停止统一走一个串行清理器：先取消 session context并撤销 Gate → Runtime.Destroy（杀并等待后代、排空输出）→ 等待 Wait/回调结束 → 发布终态。每步即使失败仍尝试其余清理，不只返回第一个错误。

所有后端接口接收 context，普通内部调用返回 error，由 internal/desktop 统一转换成公开 AppError；不能让业务包依赖 Wails。Orchestrator 暂存资源 ID/handle；Gate 持有网络资源，Runtime 持有进程和磁盘资源，Secrets 持有账号生命周期。

Start/Exec 使用会话生命周期 context，准备超时由编排计时器触发会话取消；准备成功后停止该计时器，不能在成功返回时取消仍被监听器/进程使用的 context。Exec 早到输出由编排暂存，发布 running 后按序推送；进程极快退出也必须等最终输出完成再发布终态。Gate 的致命回调只表示监听器/网关自身失效，单次上游 401、超时或拒绝通过协议与审计报告，不等同 Gateway 整体崩溃。

### 超时与幂等

- 上游连接测试 10 秒，整个 provisioning 30 秒；超时触发回滚，不能再启动晚到 Agent。模型/MCP 正常任务采用 60 秒无数据超时与 10 分钟会话总时限，收到流数据重置空闲计时；主动 Stop 随时生效。
- 清理使用脱离已取消 session 的新 context，总预算 5 秒；单个清理步骤不得吞掉其他步骤的执行机会。Revoke/Destroy 必须幂等，即使已部分完成也能重试；Gate.Revoke 在 context 已超时仍先本地标记令牌失效，随后才处理网络关闭。
- Stop 在 provisioning 中与正在创建资源的代码同步；清理器要接管所有晚返回的资源，避免取消与 Start/Create/Exec 竞争泄露资源。
- 正常退出或用户停止后清理成功：stopped、cleanupComplete=true；故障且清理成功：failed、cleanupComplete=true，释放会话锁。清理未确认完成：failed、cleanupComplete=false，保留锁并禁新会话，Stop 可重试清理。
- Runtime.Destroy 返回 nil 表示进程树已确认结束、私有目录已清理；不能只发信号就报告成功。Process.Wait 可多次读取同一最终结果，不产生重复退出事件；没有后代运行且所有输出回调结束后才完成。
- 审计/输出回调可能并发，Orchestrator 串行化公共事件。Runtime/Gate 不能在持有关键资源锁时阻塞调用 UI；不回调进入自身 Stop/Destroy。

## 4. 类型归属与依赖约束

规格代码中的类型按下表拆包，**不是新增 internal/ports 或全局接口包**。只在消费边界声明所需小接口，构造函数注入实现。

| 实际归属 | 声明 | 消费方与依赖方向 |
|---|---|---|
| internal/contracts | AppError、AgentInfo、ConnectInput、AccountInfo、SessionSpec、SessionSnapshot、Responses、Event 及载荷 | 纯数据，所有层可用；不导入业务/Wails |
| internal/orchestrator | SessionService 的实现；其构造参数消费 Gate、Runtime、Kit、PolicyBuilder、AccountCatalog 所需方法 | A 定义消费接口，用各提供方的导出类型；不让提供方反向导入编排 |
| internal/policy | PolicyRequest、EndpointRule、PolicySnapshot、Build | 可用 contracts；路径/固定 profile 校验，无 Secrets/Runtime 依赖 |
| internal/gate | GateSpec、GateBinding、Gate 和 Connections 的实现 | 可用 policy、contracts、secrets，不导入 kits/runtime/orchestrator |
| internal/secrets | SecretStore 与内存存储实现、AccountCatalog 的只读元信息实现 | 只读目录不返回真实凭据；Resolve 只注入 Gate；Put/Delete 供受限连接流程 |
| internal/runtime | RuntimeSpec、Handle、LaunchSpec、ConfigFile、Process/Exit、Runtime | 可用 contracts 输出类型；不导入 kits/gate/policy/orchestrator |
| internal/kits | KitInfo、KitInput、Kit 接口及注册表 | 可用 contracts/runtime，返回 runtime.LaunchSpec；不依赖 gate，编排层把 binding 字段映射给 KitInput |
| internal/desktop | DesktopAPI 实现、错误转换、系统目录面板及 Wails 事件 | 只面向 SessionService、Connections 与只读检测能力，不绑定其他对象 |

SecretStore.Put 要求 AccountInfo 的 ID 为空，成功分配 ID 并关联元信息；失败不留记录，返回的 ID 由连接流程填入 AccountInfo。连接测试先使用仅内存候选凭据，验证成功后才 Put；失败不创建可用账号。Catalog 与 Store 可是同一个私有实现的不同窄视图，但 A 仅收到 Lookup 能力。

policy 的固定 profile 来自 B 管理的后端配置，不从前端或 Agent 任意 URL 构造。账户测试结果绑定该 profile，构造会话时重新核对种类与元信息；两天版没有公开编辑/删除账号接口。应用退出时 B 清空存储，任务停止不删除用户仍连接的账号。

## 5. 演进与版本

当前实现注记（不改变 v1 方法/字段）：A 已实现的 macOS Runtime 默认是严格单进程子集，由操作系统拒绝 fork/posix_spawn；Codex Kit 可显式请求受控子进程 profile，该 profile 仅允许固定工具路径并按进程组清理。Claude/dsh 仍保持严格模式。Codex 固定版本离线检测和受控 Runtime 通过不代表真实模型/MCP 任务通过；不得将其标为完整执行能力交付。实现证据与剩余工作见 [研发 A 开发记录](DEV-A.md)。

公开 Event 在进入事件传输前必须满足 `contracts.Event.Validate` 的纯数据形状校验；该校验不替代策略、凭据或操作系统边界检查。

本文是并行实现的唯一方法/字段来源。首次落地时 A 合入 contracts DTO 和内部消费边界，B 复核并生成 Wails 绑定，C 据此实现 api.js；新增字段或语义变更先改本文及双方实现/替身，不能让联调时临时协商接口。首次编译仍是实现检查点，当前文档规格编译通过不意味着绑定生成或应用已可运行。

2026-10-25 修复约定：连接测试通过 SessionService.WithConnection 原子占用与 Create 共用的准入锁；会话活动、清理未完成或另一连接进行中均拒绝。编排只调用不透明回调，不获得凭据能力。Shutdown 取消并等待连接结束。ConnectModel 验证固定 Anthropic messages 响应（最多 1 token）；ConnectMcp 执行 2025-03-26 initialize/initialized/tools/list 并核对人工白名单；拒绝重定向与非 HTTPS 生产配置。未配置接入项失败关闭，不自动选供应商。

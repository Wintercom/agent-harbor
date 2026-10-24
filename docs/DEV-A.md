# 研发 A 开发记录

## 2026-10-24：共享契约与编排首批实现

工作树基于 `ce2d47b`，本轮尚未提交。下节记录首批成果；最新 Runtime/Kit 实现与限制见文末第二批记录。当前不是可运行桌面应用，也不是研发 A 全部交付完成。

按三个子任务顺序实施：契约类型与消费接口 → 编排及替身测试 → Seatbelt 初探与交接记录。

### 已落地

- `internal/contracts`：按 CONTRACT v1 定义 DTO、响应、事件、状态和事件名。所有公开错误使用固定模板，不复用提供方错误原文。
- `internal/orchestrator`：`New(appContext, Dependencies, sink, Timeouts)` 返回实现 SessionService 的服务；注入的 Catalog 仅有 Lookup，编排没有 Resolve 能力。
- 原子单会话占用；Create 快速返回 provisioning；请求 context 返回/取消不取消后台任务；Get 返回独立错误副本；Stop 快速返回 stopping。
- 准备失败回滚；停止接管晚返回 Gate/Runtime/Process；Gate 致命故障、非零退出、准备超时和会话总时限均清理。先取消并撤销 Gate，再 Destroy，再确认 Wait，最后发布终态。
- 清理使用独立 5 秒预算，为 Runtime 留出至少一半预算。清理未确认时 failed、cleanupComplete=false 并保留锁，Stop 可重试；清理完成才允许再次创建。
- 事件串行分配 seq，早到输出排在 running 后、最终输出排在终态前；输出块最多 16 KiB，输出缓冲限 1 MiB 并显示截断提示；慢 sink 不阻塞输出读取。sink 在独立分发协程执行，必须最终返回；Shutdown 等待事件排空，受调用 context 限制。
- stdout/stderr 分别跨块过滤已知 Sentinel，残留的令牌前缀也隐藏。编排不持有上游真实密钥，B 仍须在 Gate 边界过滤已知真实凭据；不能据此宣称能识别任意敏感内容。
- Runtime/Kit 接口、Gate/Policy 导出数据类型按契约拆包；测试替身仅在 `*_test.go`，没有生产替身或宿主机直跑降级。

### B 接入说明

module 使用仓库已有建议值 `github.com/liyufei916/agent-harbor`。因尚无 go.mod，为编译 A 的包加入最小 module；`go 1.23.0` 是当前源码的暂定最低语言基线，本机实际使用 Go 1.27.2，不代表已经验证 Go 1.23。Wails 初始化及依赖仍由 B 接续。

`internal/gate/types.go` 和 `internal/policy/types.go` 只落地共享边界类型，未实现 B 的业务。B 直接使用这些类型，不另复制。具体消费接口在 `internal/orchestrator/ports.go`，方法签名沿用 CONTRACT v1，无协议变更。

框架启动时创建具体组件、将应用级 context 注入 New；sink 把 Event 转发为 `contracts.EventName`。退出必须用新的、有清理预算的 context 调用 Shutdown 并处理错误。不要把 Runtime/Gate/Secrets 绑定到 Wails。

Gate、Policy、Catalog 的真实实现和 Runtime/Claude Kit 的具体实现尚未接入，因此本轮没有生产装配或桌面启动命令。60 秒上游无数据超时由流式 Gate 负责，当前编排测试不证明这一行为。

### 实际验证

环境：macOS 26.7（25G229）、arm64、非 root UID 503、Go 1.27.2 darwin/arm64。Claude Code `2.1.292`，原生安装路径 `/opt/homebrew/Caskroom/claude-code/2.1.292/claude`；只执行了 `--version` 与 `--help`，未读取登录态或调用模型。

```sh
go test -race ./...
go test -race -count=10 ./internal/orchestrator
go vet ./...
python3 security/seatbelt_probe.py
```

Go 编排测试通过，覆盖正常生命周期、并发拒绝、九个准备失败点、晚到资源取消竞争、Gate/Runtime 清理失败与重试、非零退出、Gate 致命故障、两个超时、应用取消、账号类型校验、快照防修改、审计脱敏、跨块令牌过滤、UTF-8 块边界及慢 UI 截断。

Seatbelt 诊断使用临时非敏感目录与两个本机受控监听器：工作区读写、指定回环端口访问成功；工作区外读写、越界符号链接、其他回环端口均拒绝，七项通过。路径含空格、双引号和中文。脚本不读取真实凭据，不向公网发送请求，所有临时文件和监听器在结束时清理。

诊断过程发现：最初极简限制规则运行 true 异常退出（134）；导入系统基础规则后可运行；本机 SBPL 的 `remote tcp` 拒绝字面量 IPv4，需使用 localhost；系统 curl 还需读取固定 OpenSSL 配置文件。编译失败或进程崩溃不能被脚本记为拒绝测试通过。

**诊断模板不是生产 profile**：它导入了较宽的 `system.sb`（包含 IPC、系统路径等授权），网络使用 `localhost:port`，尚未证明契约要求的精确 `127.0.0.1:port` 限制，也未证明宿主管控 IPC、提权与逃离进程组的后代清理。不得复制该模板用于真实 Agent。

### 后续实施顺序

1. 收敛 macOS 生产 profile；验证精确地址/端口、敏感路径、必要运行路径及 IPC/提权限制；实现可靠进程树确认，不能只向进程组发送信号就报告成功。
2. 实现 Runtime 的私有 HOME、净化环境、相对配置路径校验、配置权限、流式读取和幂等 Destroy；隔离不可用时 Health 拒绝启动。
3. 在受限环境验证固定 Claude 版本的非交互参数、显式模型 Base URL、strict MCP 配置和 Sentinel；完成 Kit 实现。CLI 参数存在不等于端到端兼容。
4. 接入 B 的真实 Gate/Policy/Catalog，完成真实模型和认证 MCP 联调，再执行 ACCEPTANCE 的 I1–I11。当前这些真实集成项全部待执行。

## 2026-10-24：严格 Runtime 与 Claude Kit 第二批实现

### 方案与实现

本批顺序为隔离规则和真实探针 → Runtime → Claude Kit → 真实 Runtime 编排集成测试与文档。未修改前后端 DTO 或 B 的业务实现。

- `runtime.NewMacOS()`：目前仅接受 Darwin 25 / arm64、非 root、非 setuid/setgid 宿主进程；Health 实际编译应用 profile，不可用时拒绝启动。
- `sandbox.sb` 默认拒绝；仅导入 dyld 基础规则，不导入较宽的 system.sb。固定系统运行目录只读，工作区及私有 HOME 可写，路径经符号链接解析和 SBPL 转义；父目录只开放必要元数据。
- Gate 网络规则使用 SBPL 的 `remote tcp4 "localhost:port"`。在本机实际验证：127.0.0.1 指定端口可达；127.0.0.2 同端口、::1 同端口、其他回环端口、UDP、TEST-NET-1 直连与自建 Unix socket 均为权限拒绝。127.0.0.2 未配置本机地址，不能建立正向监听，但探针确认返回 EPERM/EACCES，而非不可达/拒绝连接。
- Create 创建源码树外的私有 HOME、tmp 和不可由 Agent 写入的 profile；执行路径必须在只读文件白名单中，拒绝目录和 setuid/setgid 程序。配置先完整校验相对路径、冲突和大小，再以目录 0700、文件 0600 写入，拒绝覆盖和符号链接。
- 子进程环境从零构造，禁止覆盖 HOME、PATH、TMPDIR、CLAUDE_CODE_TMPDIR 和注入 DYLD/NODE 等变量；两个临时目录变量都指向私有 HOME。Kit 只可增加受限模型/运行开关。
- Runtime 流式读 stdout/stderr，跨 Read 保留不完整 UTF-8，每块最多 16 KiB。取消杀死受限进程，Wait 可重复读取，返回前排空输出回调；Destroy 确认结束及删除私有目录后才成功。输出未排空导致清理超时会保留目录，并允许重试。

**本批仍是严格单进程模式，不能宣称完整 Agent 执行能力完成。** macOS SDK 明确 NOTE_TRACK 自 10.5 起不支持；进程组清理不能可靠覆盖主动脱离的后代，因此本批由内核拒绝 process-fork。测试同时验证 fork、原生 posix_spawn、POSIX_SPAWN_SETPGROUP 均拒绝，避免依赖仅扫描后代或发进程组信号。此模式下 Bash、外部 git、依赖独立子进程的工具不可用；没有放宽 fork 的隐藏开关。后续如核心任务需要子进程，必须先实现可靠追踪/确认方案，并完整重测，不能简单开放 process-fork。

### Claude Code

`claude.New(executable, model)` 接收宿主机固定模型名。Detect 在无网络私有 HOME 下执行 CLI --version / --bare --help，当前只接受 2.1.292，并检查所需参数。Build 生成 strict Streamable HTTP MCP 配置、模型 Gate 环境与 stdin 任务；不执行任务、不写盘、不读取宿主登录态。内置工具暂限 Read/Write/Edit，MCP 授权仍由 Gate 的固定工具名单执行。Detect 的 available 仅证明安装、版本与受限参数检测通过，真实模型/MCP 兼容仍未验收。

实测修复的启动依赖：ICU/locale 数据的两个固定系统只读目录；私有配置路径的父目录元数据；CLAUDE_CODE_TMPDIR 必须显式重定向，否则该版本尝试使用 /tmp/claude-UID 并被拒绝。没有开放用户 Library、钥匙串、其他项目或通用网络来绕过这些问题。

### 验证与命令

```sh
go test -race ./...
go vet ./...
HARBOR_TEST_CLAUDE=1 go test -race -v ./internal/kits/claude -run TestInstalledCLI
```

- Runtime 的真实系统探针：授权读写、越界/符号链接拒绝、精确网络边界、子进程拒绝、净化环境、UTF-8 排空、取消、重复 Wait/Destroy、清理超时重试、非法配置及运行规格均通过。
- 编排真实进程集成：真实 Runtime + 测试进程，Gate/Policy/Catalog 仍为测试替身；正常退出、非零退出、主动停止及输出先于终态均通过。
- 本机 Claude 2.1.292：严格环境的版本/帮助检测通过；使用测试 Sentinel 和未运行服务的 127.0.0.1:1，真实 CLI 发出结构化 init，模型名、版本和 APIKeySource 正确，随后因无上游在 3 秒取消并清理。未向真实模型发送请求，未使用真实凭据，MCP 状态为失败；该测试不能填 I2 成功。
- 独立编译的最小 C/Go 可执行文件在本机沙箱外也会 SIGKILL，原因尚未确定。测试改用实际 Go 测试二进制的 helper 模式，cgo 测试包额外调用原生 posix_spawn；未将 SIGKILL 记作权限拒绝。涉及原生 spawn 的系统测试要求 CGO_ENABLED=1，本机已启用。

### 接入与剩余事项

B 可将 `runtime.NewMacOS()` 和 `claude.New(固定CLI路径, 固定模型名)` 注入既有 orchestrator.New。所有组件仍由宿主机装配，不绑定 Runtime/Kit 到 Wails。`runtime.ProbeExecutable` 仅供受信 Kit 做离线检测。

尚未完成：真实 Gate/Policy/Catalog 及 Wails 装配；模型协议、真实认证 MCP 发现/调用；真实 Agent 文件修改任务；完整 IPC/提权负例验收；允许子进程时的可靠清理。完整 I1–I11 不能因本批单元/系统探针通过而整体标为通过。

## 2026-10-24：边界回归与清理安全加固

- Health 不再只执行 `/usr/bin/true`：会实际验证授权文件可读、外部 fixture 读失败且退出码是普通权限拒绝；profile 被意外改成 allow default 时会 fail closed。
- 准备超时只作用于 `provisioning`，会话总时限从 Create 开始计时并覆盖准备阶段；计时器晚到不能取消已进入 running 的会话。
- 事件队列的截断标记由 `sessionId + seq` 组成，避免不同会话重复 seq 时进入截断死循环；新增回归测试。
- 验证了只禁止 `setsid` / `setpgid` 的候选进程组方案：原生 `posix_spawn(POSIX_SPAWN_SETPGROUP)` 仍能创建新组。因此生产 profile 继续拒绝 `process-fork`，没有以不充分的进程组假设开放子进程。

最新验证：`go test -race ./...`、`go vet ./...`、`git diff --check` 通过；真实 Claude 离线测试需 `HARBOR_TEST_CLAUDE=1`，本机 2.1.292 版本/初始化检查通过。真实模型、MCP、完整子进程能力和 Wails 仍待 B 联调。

## 2026-10-24：Runtime 资源墓碑

Destroy 成功后保留最小句柄墓碑，使重复 Destroy 继续幂等；同时清空 profile、工作区、私有目录、只读路径和进程引用，避免长时间运行的桌面应用累计敏感元数据。新增测试检查这些字段释放，并确认重复 Destroy 与已销毁句柄执行均符合契约。

同时加入 `internal/kits.Registry`：只接受宿主机显式装配的 Kit，按 Agent ID 稳定排序批量检测；不可用项清空只读路径，ID 不一致自动关闭，取消和检测内部错误分别保留明确语义。它可直接供 B 的 `ListAgents` 绑定使用，不会自动扫描或注册未知 CLI。

Health 现在还会检查生成的 SBPL 文本本身必须包含 `deny default`、`deny process-fork`，且不能包含对应 allow 规则；随后仍执行授权/拒绝 fixture，防止仅靠字符串检查或仅靠单个探测遗漏 profile 回归。

新增 `security/checks.py` 和 `test_checks.py`：对后续 I1–I11 的脱敏 JSON 证据执行格式、唯一编号、状态、环境、通过证据和明显凭据模式检查。该脚本只校验证据，不把 fake、文档或诊断结果升级为真实通过。

共享 `contracts.Event.Validate` 已加入：事件必须有非空 session/seq，且只包含与 type 对应的一个载荷；state 的水位必须匹配，输出流限定 stdout/stderr 并拒绝 NUL，审计结果和错误字段也经过形状检查。事件队列在入队前丢弃非法事件，避免结构错误进入 Wails/UI。

## 2026-10-24：覆盖率验证稳定性修复

覆盖率模式会让 Runtime 的探针进程收到 Go 的 `GOCOVERDIR` 提示；该提示不是探针业务输出。测试回调现在过滤固定的 `warning: GOCOVERDIR not set` 行，避免 `go test -cover` 因工具链诊断噪声误报，同时仍保留真实 stdout/stderr 和退出结果校验。

本轮独立验证：

```sh
go test -race ./...
go test -race -tags harbor_platform ./internal/runtime ./internal/orchestrator
go test -cover ./internal/contracts ./internal/kits ./internal/kits/claude ./internal/orchestrator ./internal/runtime
go vet ./...
git diff --check
```

以上检查均通过；其中 `harbor_platform` 系统测试要求支持的 Darwin 25/arm64 环境，普通 CI 的 macOS 14 不满足该安全基线，只运行可移植测试。A 当前仍可独立交付并验证的范围是契约/事件校验、会话生命周期与清理重试、严格单进程 macOS Runtime、Claude Kit 离线检测、Kit Registry 以及脱敏验收证据校验。真实 Gate/Policy/Secrets、模型与认证 MCP、Wails 装配、真实 Agent 文件任务和允许子进程后的可靠进程树清理仍需 B 或真实环境联调。

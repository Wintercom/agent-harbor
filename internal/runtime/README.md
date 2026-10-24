# 隔离运行时

负责人：A。两天版只实现已验证目标 macOS 上的 Seatbelt / sandbox-exec 后端；已实现严格单进程 Runtime、默认拒绝的 Seatbelt profile 和真实系统探针测试。Health 会验证授权与拒绝边界；准备/会话计时器均失败关闭。当前禁止创建后代，不支持 Bash/外部 git；仍需真实 Agent/上游联调。详细证据及限制见 [研发 A 开发记录](../../docs/DEV-A.md)。

## 当前文件

| 文件 | 职责 |
|---|---|
| `runtime.go` | Health、Create、Exec、Destroy、Process.Wait 与受限句柄 |
| `macos.go` | macOS 能力检查、私有 HOME、严格单进程执行与清理 |
| `sandbox.sb` | 默认拒绝的系统强制规则；仅按确定的授权参数开放必要资源 |

两天版将工作区授权及私有 HOME 配置合并进 Create，不再单独暴露 MountWorkspace；这是可见/可写范围配置，不意味着真实文件系统挂载。未来后端需保持同等语义。

## 输入与输出

接收规范化工作区、已确定的网络端口及 Kit LaunchSpec，返回输出流、退出通知与可销毁句柄。不解释账号、MCP 或授权弹窗，也不接收真实模型/MCP 凭据。

## 强制边界

- 工作区及会话私有 HOME 按需可写，Agent/系统必要运行文件只读；拒绝宿主机敏感目录、其他项目和越界符号链接。
- 子进程环境从允许列表构造，HOME 指向私有目录；不得整体继承宿主机环境。
- 网络仅允许 Gate 所需回环端口，阻断直接公网、非 Gate 回环与宿主机管控 IPC。
- 不授 root、sudo、Docker socket 或等价宿主管控能力；所有后代进程继续受限。
- 无法建立完整限制时拒绝启动，不以设置 HOME、命令黑名单或普通子进程替代隔离。
- 负责进程树取消与退出确认，再清理临时配置；临时路径在运行时创建于源码树外。

## 验收

首日第 3 小时前证明授权文件可写、工作区外文件不可读、直连和非 Gate 端口不可达；不能只测试 Gate 自己会拒绝。测试脚本归 [security](../../security/README.md)，profile 参数中含空格、特殊字符及符号链接路径的情况也须覆盖。

## 对接契约

精确类型与方法以 [CONTRACT.md](../../docs/CONTRACT.md) 为准：RuntimeSpec 输入 AllowedTCP 与只读运行路径，Handle 返回 PrivateHome；Runtime 定义 LaunchSpec，Kit 生成其内容，Runtime 校验并写入 ConfigFiles。Destroy 幂等且确认进程树结束及磁盘清理后才返回；Wait 必须排空最终输出。并行替身与所有权见 [MODULE-DESIGN.md](../../docs/MODULE-DESIGN.md)。

Destroy 成功后保留不可执行的幂等句柄墓碑，同时释放 profile、工作区、私有目录、只读路径和进程引用；重复 Destroy 返回成功，已销毁句柄不能再次 Exec。

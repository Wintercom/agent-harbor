# 会话编排

负责人：研发 A，兼技术集成负责人。唯一负责会话生命周期，供受限 Wails 绑定调用；依赖 Policy、Gate、Kit 和 Runtime，不直接读取真实凭据。

## 建议文件（尚未实现）

- `session.go`：会话状态、单会话锁、Create、Get、Stop 和 Shutdown。
- `events.go`：汇总状态、输出、审计和安全错误，向 UI 发布共享契约事件。

## 启动与停止

1. 校验任务、账号引用和工作区，获取单会话锁；准备和清理阶段同样占锁。
2. 通过注入的只读 AccountCatalog 检查账号元信息，检测 Runtime/Kit，构造 Policy 快照，再启动 Gate。
3. 将 GateBinding.AllowedTCP 与只读运行路径交给 Runtime.Create，取得 PrivateHome，再映射 KitInput 生成 LaunchSpec，由 Runtime.Exec 校验、落盘并隔离执行。
4. Agent 成功启动后进入 running；正常退出、用户停止和致命故障均进入清理。
5. 先撤销 Sentinel 并取消在途请求，再结束进程树、清理私有 HOME，最后释放锁。

启动中途失败必须按已创建资源回滚。清理失败进入 failed 并保留锁，提供明确修复说明，不能误报 stopped 后允许新会话。

## 状态与验收

本版一个会话执行一个非交互任务：`provisioning → running → stopping → stopped`，异常为 failed。failed 且 cleanupComplete=true 也释放锁；failed 且 cleanupComplete=false 保留锁。再次运行创建新会话，不恢复旧进程。

测试重点：并发创建拒绝、各启动步骤失败回滚、重复停止幂等、Agent 正常/异常退出、Gate 致命故障以及清理失败阻止新会话。上游单次认证错误不应自动扩大授权。

精确接口、事件/快照和取消竞争规则见 [CONTRACT.md](../../docs/CONTRACT.md)。A 首先用 fake Gate/Policy/Catalog/Runtime/Kit 独立开发，再接真实组件；共享文件与集成检查点见 [MODULE-DESIGN.md](../../docs/MODULE-DESIGN.md)。

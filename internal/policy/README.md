# 会话策略

负责人：B。将用户已确认的资源转换为不可随 Agent 请求改变的会话授权快照，不负责网络转发或系统隔离执行。

## 建议文件（尚未实现）

`policy.go` 定义并构造 workspace、direct 模式、固定模型/MCP origin、允许路径与工具挂载列表。两天版不引入动态扩权、长期策略库或通用表达式引擎。

## 输入、输出与依赖

输入来自编排层确认的非敏感账号元信息、工作区和工具引用，不含真实凭据。输出供 Gate 做目标/工具授权，供 Runtime 转换成文件和网络限制；Policy 不直接发 HTTP、不管理进程、不解析 Secrets。

## 规则与验收

- 规范化为真实路径，拒绝整个 Home、系统目录、凭据目录和不在演示范围的工作区。
- 上游绑定 HTTPS origin 与明确的协议路径范围，不能只用可伪造 Host 或字符串前缀匹配。
- 不因请求失败、重定向或 MCP 返回数据自动添加新目标/工具。
- 审计只公开可解释的边界摘要，不能夹带密钥和内部令牌。
- 测试路径别名、符号链接、相似域名、端口变化、未挂载工具及未授权目标；默认拒绝。

Runtime 所需文件路径和 Gate 所需端口须由主进程装配确认，不能从 Agent 的任意输入扩权。

精确输入 PolicyRequest、输出 PolicySnapshot 和 Build 语义见 [CONTRACT.md](../../docs/CONTRACT.md)。固定 profile 由 B 的受信后端配置提供，不读取 Secrets；供 A 使用的 fake 与文件归属见 [MODULE-DESIGN.md](../../docs/MODULE-DESIGN.md)。

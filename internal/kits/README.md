# Agent Kit

负责人：A。把特定 Agent 的安装检测、非交互启动方式和配置格式封装为适配模块，不实现隔离或上游认证。

## 目录与建议文件

- `kit.go`（待实现）：Detect、Build 与能力描述，返回 runtime.LaunchSpec（Command、Args、Env、Stdin、ConfigFiles）。
- [claude](claude/README.md)：首个必交的 Claude Code 适配。
- Codex、dsh 在实际开始适配时再建目录，不以空目录或枚举项表示已经支持。

## 边界

只接收 KitInput 中的 Gate URL、Sentinel、私有 HOME 与任务；不直接依赖 Gate 包或 GateBinding 类型。模板只包含 Gate URL、Sentinel，不接触 Secrets，不读取用户现有 Agent 登录态或复制宿主机配置。

Kit 生成启动规格，由 Runtime 在限制内执行；不得自行启动宿主机进程或通过 shell 字符串拼接任务。Agent 安装位置及运行文件必须与 Runtime 的只读授权共同确认。

## 验收

注册表只展示实际可用且已验证的适配项。每个适配记录 CLI 版本、操作系统、模型协议、MCP 传输和输出格式；证明真实 Agent 能调用 Gate 模型和认证 MCP，不能用 curl 成功代替适配验收。

精确接口见 [CONTRACT.md](../../docs/CONTRACT.md)：Detect 同时返回 CLI 必要只读路径，Build 只生成配置、不写盘，由 Runtime.Exec 校验并落盘。依赖与独立验收见 [MODULE-DESIGN.md](../../docs/MODULE-DESIGN.md)。

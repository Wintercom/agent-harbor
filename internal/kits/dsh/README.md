# DeepSeek dsh Kit

负责人：A。适配官方 `deepseek-ai/deepseek-harness` 的 npm 包 `@deepseek-ai/dsh@0.2.1-rc.2`，固定官方独立 Node `v26.12.1`，不读取用户已有配置或登录态。

`Detect` 在真实离线 Runtime 内核对版本/能力，`Build` 返回私有 HOME 配置和明确的 Node/脚本参数，任务仅通过 stdin。Harbor profile 明确组合文件 Read/Write/Edit、DeepSeek Messages、认证 Streamable HTTP MCP 与 JSON headless 插件。MCP 使用必需插件 ID `connection`，初连失败拒绝任务；不开放 Bash、插件管理或子进程。

模型使用 `dsh-model`，Kit 仅注入 Gate URL 与 Sentinel，真实 Key 由宿主 Gate 替换。专用安装目录只读，Node 文件单独授予执行，不能以只读目录授权递归执行。

安装、环境变量、人工用例及可重复测试命令见 [dsh 适配与验证](../../../docs/DSH-INTEGRATION.md)。普通单测不安装真实 dsh；`HARBOR_TEST_DSH=1` 测试真实 CLI/OS/Gate，模型和 MCP 为受控测试服务，不能计入真实远程验收。

# 演示流程

负责人：C，A/B 配合排查。不存放真实凭据或用户项目。

## 三分钟 mock 话术（评审用）

前置：`cd frontend && python3 -m http.server 5173`，打开 http://127.0.0.1:5173/  
顶栏应显示 **模拟模式**（不是真实验收）。

| 时间 | 操作 | 期望观众看到 |
|---|---|---|
| 0:00 | 指顶栏徽章与 direct 警告 | 凭据只在宿主机、会改原文件 |
| 0:20 | 模型/MCP 各填任意 Key → 连接 | 输入框清空；状态显示账号 id/host，无 Key 回显 |
| 0:50 | 工作区 → 确认默认任务文案 → 开始 | 任务为固定非敏感脚本；切到工作台 provisioning→running |
| 1:20 | 指输出与审计列表 | 有文本输出；MCP `allowed` 审计 |
| 1:50 | 等自动 stopped，或点停止 | 终态 stopped；可回配置再开 |
| 2:10 | 场景切「认证失败」再连模型 | 明确 `UPSTREAM_AUTH_FAILED`，不进入会话 |
| 2:30 | 场景「隔离不可用」点开始 | `ISOLATION_UNAVAILABLE`；不假装已隔离 |
| 2:45 | 场景 cleanup：开始后马上停止两次 | 首次 failed+未清理提示；再停后 stopped |

收尾一句：这是 UI/契约 D4；真实 Agent/Gate 要等 A/B live 绑定后走 I1/I2。

## 当前可演示（mock / D4）

```bash
cd frontend
python3 -m http.server 5173
# 打开 http://127.0.0.1:5173/
# 可选：?scenario=auth|busy|isolation|cleanup
```

1. 配置页连接模型与 MCP（任意非空 Key，提交后清空）。
2. 选择工作区（mock 固定 `/demo/project`）。
3. 使用默认任务（或同义非敏感文案）→ 开始运行 → 自动切到工作台。
4. 观察 provisioning → running → 输出 / MCP audit → stopped。
5. 用场景下拉验证四失败路径。

### 模型 / MCP 填写示例（可复制）

**mock / D4**：任意非空 Key 即可；下列与页面默认显示名一致，便于照念。**不要**把真实 Key 写进仓库或截图。

| 字段 | 模型连接 | MCP 连接 |
|---|---|---|
| 显示名 | `演示模型` | `演示 MCP` |
| profileId（只读） | `demo-model` | `demo-mcp` |
| API Key / Token | `mock-model-key-not-secret` | `mock-mcp-token-not-secret` |

连接成功后输入框应清空；状态类似：

- 模型：`已连接 · mock-model · demo-model.local`
- MCP：`已连接 · mock-mcp · demo-mcp.local`

**live / I1（待 B 绑定）**：profileId 仍固定为上表；显示名可自定；Key/Token 用本机已登记凭据，仅粘贴一次。上游 origin 由 B 的固定 profile / 环境变量决定（默认模型 `https://api.deepseek.com`，MCP `https://mcp.example.test`，可用 `HARBOR_DEMO_MODEL_ORIGIN` / `HARBOR_DEMO_MCP_ORIGIN` 覆盖），前端**不能**改 URL。

```text
# 模型（示例字段，Key 换成本机真实值）
显示名: 演示模型
profileId: demo-model
apiKey: <本机真实模型 Key，勿提交>

# MCP（示例字段，Token 换成本机真实值）
显示名: 演示 MCP
profileId: demo-mcp
apiKey: <本机真实 MCP Token，勿提交>
```

### 固定演示任务（可复制）

```
在工作区创建或覆盖 hello.txt，写入一行 Hello from Agent Harbor mock。然后用已连接的 MCP 做一次只读 search，把查询结论用两句话总结。不要修改其它文件，不要请求扩权。
```

详情与文件清单见 [DEV-C.md](../docs/DEV-C.md)。

## 专用演示工作区（C 已准备）

| 项 | 值 |
|---|---|
| 目录 | [`demos/workspace/`](workspace/) |
| 用途 | 真实演示 / I2 选工作区；允许改写 `hello.txt` |
| 自检 | 仓库根执行 `./demos/verify-workspace.sh` |
| mock | 仍固定返回 `/demo/project`，**不写**本目录 |

```bash
cd "$(git rev-parse --show-toplevel)"
./demos/verify-workspace.sh
# 输出中的 abspath 供桌面「选择工作区」粘贴（勿提交个人路径为默认值）
```

## 真实演示（待应用可运行）

1. 关闭 mock，连接已登记真实模型与只读认证 MCP。
2. 选择 `demos/workspace` 的绝对路径；说明 direct 会改原文件。
3. 启动真实 Agent，展示输出、工具调用与隔离边界；期望出现/更新 `hello.txt`。
4. 配合 `security/` 展示越界拒绝。
5. 停止后第二轮再跑（可再跑 `verify-workspace.sh` 确认可写）。

排期以 [MVP-2D3P.md](../docs/MVP-2D3P.md) 为准；验收项见 [ACCEPTANCE.md](../docs/ACCEPTANCE.md)。

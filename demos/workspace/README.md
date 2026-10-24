# 专用非敏感演示工作区

供 **真实演示 / I2** 选用。`direct` 模式会直接改本目录文件；勿放入密钥、用户项目或个人资料。

| 项 | 约定 |
|---|---|
| 相对路径 | `demos/workspace/`（相对仓库根） |
| 允许改动 | 仅本目录；演示任务会创建/覆盖 `hello.txt` |
| 禁止 | 凭据、`.env`、私钥、工作区外路径、扩权请求 |
| mock | 浏览器 mock 仍返回固定路径 `/demo/project`，**不会**写本目录 |

## 本机解析绝对路径

```bash
# 在仓库根执行
cd "$(git rev-parse --show-toplevel)"
python3 -c "import os; print(os.path.realpath('demos/workspace'))"
```

桌面 live 选工作区时粘贴上述绝对路径（勿把个人 Home 路径写进仓库默认值）。

## 自检

```bash
# 仓库根
./demos/verify-workspace.sh
```

期望：目录可写；模拟写入 `hello.txt` 内容为 `Hello from Agent Harbor mock`；可重复覆盖。

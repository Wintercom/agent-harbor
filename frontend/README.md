# Wails 桌面前端

负责人：产品 C 开发页面，研发 B 提供 Wails v3 绑定、事件和构建环境。C 实现与 D4 说明见 [DEV-C.md](../docs/DEV-C.md)。

## 目录

| 路径 | 状态 | 职责 | 负责人 |
|---|---|---|---|
| [index.html](index.html) / [src/](src/) | 全量 PAGE stub/mock | 侧栏 HOME/模型/工具/设置；顶栏会话切换；api/mock（含 OAuth/PTY） | C |
| [package.json](package.json) / Vite | B 已初始化 | 依赖与构建脚本 | B |
| `bindings/` | 生成 | Wails v3 绑定，**不手改** | B |
| `dist/` | 构建产物 | 供 Go embed | 构建生成 |

## 本地预览（mock / D4）

```bash
cd frontend
npm ci
npm run dev
# 或：python3 -m http.server 5173
# 打开提示的本地地址；可选 ?scenario=auth
```

配置页场景：正常 / 认证失败 / SESSION_BUSY / ISOLATION_UNAVAILABLE / cleanupComplete=false。

## 生成绑定与桌面开发（B）

```bash
# 仓库根目录
wails3 generate bindings ./...
npm ci --prefix frontend
npm test --prefix frontend
npm run build --prefix frontend
wails3 dev
```

页面只经 `src/api.js` 调用业务方法。浏览器独立开发使用 `mock-api.js`，勿手改 bindings。`createLiveTransport()` 延迟加载 `src/live-api.js`，由 Vite 打包生成绑定，并用 `@wailsio/runtime` 的 `Events.On('harbor:event')` 转发契约事件。桌面壳始终使用真实绑定，失败时显示固定错误并停止初始化；浏览器 `?live=1` 显式测试真实绑定，其余使用 mock。桌面不接受 URL 参数切回 mock。使用 Vite 构建/开发前先生成绑定；直接用静态 HTTP 服务预览 mock 不依赖生成文件。

准确方法见 [CONTRACT.md](../docs/CONTRACT.md) **v1.2**。本分支范围见 [PAGE-DESIGN](../docs/PAGE-DESIGN.md) / [TASK-SPLIT](../docs/TASK-SPLIT-ABC.md) **v2.0**（全量；非 MVP 两天裁剪）。

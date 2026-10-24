# Wails 桌面前端

负责人：产品 C 开发页面，研发 B 提供 Wails v3 绑定、事件和构建环境。C 实现与 D4 说明见 [DEV-C.md](../docs/DEV-C.md)。

## 目录

| 路径 | 状态 | 职责 | 负责人 |
|---|---|---|---|
| [index.html](index.html) / [src/](src/) | 已实现 mock 两页 | 配置、工作台、样式、api/mock | C |
| [package.json](package.json) / Vite | B 已初始化 | 依赖与构建脚本 | B |
| `bindings/` | 生成 | Wails v3 绑定，**不手改** | B |
| `dist/` | 构建产物 | 供 Go embed | 构建生成 |

## 本地预览（mock / D4）

```bash
cd frontend
npm install
npm run dev
# 或：python3 -m http.server 5173
# 打开提示的本地地址；可选 ?scenario=auth
```

配置页场景：正常 / 认证失败 / SESSION_BUSY / ISOLATION_UNAVAILABLE / cleanupComplete=false。

## 生成绑定与桌面开发（B）

```bash
# 仓库根目录
wails3 generate bindings ./...
cd frontend && npm install && npm run build
wails3 dev
```

页面只经 `src/api.js` 调用业务方法。浏览器独立开发使用 `mock-api.js`，勿导入 bindings。B 生成绑定后，在 `api.js` 的 `createLiveTransport()` 接入 `bindings/.../desktopapi.js`，再 `useLive()` 或 `?live=1`。事件名固定 `harbor:event`。

准确方法见 [CONTRACT.md](../docs/CONTRACT.md)。两天范围见 [MVP-2D3P.md](../docs/MVP-2D3P.md)。

# Wails 桌面前端

负责人：产品 C 开发页面，研发 B 提供 Wails v3 绑定、事件和构建环境。

## 目录

| 路径 | 职责 | 负责人 |
|---|---|---|
| `index.html` / `src/` | 配置页、工作台、样式 | C（壳文件已由 B 放入，便于构建） |
| `package.json` | Vite 依赖与脚本 | B 初始化 |
| `bindings/` | Wails v3 生成绑定，**不手改** | B 生成 |
| `dist/` | 构建产物，供 Go embed | 构建生成 |

## 生成绑定与开发

```bash
# 仓库根目录
wails3 generate bindings ./...
cd frontend && npm install && npm run build
# 或一体化开发
wails3 dev
```

前端通过 `src/api.js`（C 实现）调用 `bindings/github.com/liyufei916/agent-harbor/desktopapi.js`。
事件名固定为 `harbor:event`（见 contracts）。浏览器独立开发使用 `mock-api.js`，勿导入 bindings。

准确方法见 [CONTRACT.md](../docs/CONTRACT.md)。

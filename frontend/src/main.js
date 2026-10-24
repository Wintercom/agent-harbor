/**
 * 页面装配入口。产品 C 在此挂载配置页与工作台，并通过 api.js 调用绑定。
 * 研发 B 仅提供可构建桌面壳；生成绑定目录为 frontend/bindings/（勿手改）。
 */

const status = document.getElementById('status')

status.textContent =
  '桌面壳已就绪。\n' +
  '产品 C：实现 src/api.js、src/mock-api.js 与 views/。\n' +
  '研发 B：wails3 generate bindings ./... && npm run build\n' +
  '联调：仓库根目录执行 wails3 dev'

async function probeDesktopBinding() {
  try {
    const mod = await import(
      /* @vite-ignore */ '../bindings/github.com/liyufei916/agent-harbor/desktopapi.js'
    )
    if (typeof mod.AppInfo === 'function') {
      const info = await mod.AppInfo()
      status.textContent = 'DesktopAPI 可用:\n' + JSON.stringify(info, null, 2)
    }
  } catch {
    // 浏览器预览或尚未生成绑定时保持占位文案。
  }
}

probeDesktopBinding()

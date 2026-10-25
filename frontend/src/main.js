/**
 * 页面装配入口（产品 C）。
 * 桌面使用 live，普通浏览器使用 mock；Wails v3 绑定目录为 frontend/bindings/（B 生成，勿手改）。
 */
import * as api from "./api.js";
import { initializeTransport } from "./startup.js";
import { mountSetup } from "./views/setup.js";
import { mountSession } from "./views/session.js";

const badge = document.getElementById("mode-badge");
const setupRoot = document.getElementById("view-setup");
const sessionRoot = document.getElementById("view-session");

const ctx = {
  state: {
    agentId: "claude",
    agentAvailable: false,
    modelAccount: null,
    mcpAccount: null,
    workspace: "",
    sessionId: "",
    lastSnapshot: null,
  },
  showTab(name) {
    document.querySelectorAll(".tab").forEach((t) => {
      t.classList.toggle("on", t.getAttribute("data-tab") === name);
    });
    setupRoot.classList.toggle("hidden", name !== "setup");
    sessionRoot.classList.toggle("hidden", name !== "session");
  },
  onSessionCreated(snapshot) {
    ctx.state.sessionId = snapshot.sessionId;
    ctx.state.lastSnapshot = snapshot;
    sessionView.resetForNewSession(snapshot);
    ctx.showTab("session");
  },
};

let sessionView;

function syncBadge() {
  if (api.isMock()) {
    badge.textContent = "模拟模式";
    badge.className = "badge mock";
  } else {
    badge.textContent = "真实绑定";
    badge.className = "badge live";
  }
}

function isWailsShell() {
  if (location.protocol === "wails:" || location.hostname === "wails.localhost") {
    return true;
  }
  return typeof window !== "undefined" && !!window._wails;
}

async function boot() {
  await initializeTransport(api, { shell: isWailsShell(), search: location.search });
  syncBadge();

  document.querySelectorAll(".tab").forEach((btn) => {
    btn.addEventListener("click", () => ctx.showTab(btn.getAttribute("data-tab")));
  });

  mountSetup(setupRoot, ctx);
  sessionView = mountSession(sessionRoot, ctx);
  window.addEventListener("pagehide", () => sessionView.destroy(), { once: true });
  ctx.showTab("setup");
}

boot().catch(() => {
  badge.textContent = "连接失败";
  badge.className = "badge bad";
  setupRoot.textContent = "桌面连接初始化失败，请重新启动应用。";
  document.querySelectorAll(".tab").forEach((button) => { button.disabled = true; });
});

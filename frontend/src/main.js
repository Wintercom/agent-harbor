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
    try {
      sessionStorage.setItem("harbor-session-id", snapshot.sessionId);
    } catch {}
    sessionView.resetForNewSession(snapshot);
    ctx.showTab("session");
  },
};

let sessionView;

function syncBadge() {
  const banner = document.getElementById("mode-banner");
  if (api.isMock()) {
    badge.textContent = "模拟模式";
    badge.className = "badge mock";
    if (banner) {
      banner.textContent =
        "两天版：direct 会改原文件；凭据只在宿主机内存。当前为浏览器模拟模式，不能记入 I1–I11。";
    }
  } else {
    badge.textContent = "真实绑定";
    badge.className = "badge live";
    if (banner) {
      banner.textContent =
        "真实绑定：任务请去掉 mock/search；工作区用 demos/workspace；凭据提交后清空。演示步骤见 demos/README。";
    }
  }
}

function isWailsShell() {
  if (
    location.protocol === "wails:" ||
    location.hostname === "wails.localhost"
  ) {
    return true;
  }
  return typeof window !== "undefined" && !!window._wails;
}

async function boot() {
  await initializeTransport(api, {
    shell: isWailsShell(),
    search: location.search,
  });
  syncBadge();

  document.querySelectorAll(".tab").forEach((btn) => {
    btn.addEventListener("click", () =>
      ctx.showTab(btn.getAttribute("data-tab")),
    );
  });

  sessionView = mountSession(sessionRoot, ctx);
  const setupView = mountSetup(setupRoot, ctx);
  let recovering = false;
  const recover = async () => {
    if (recovering) return;
    recovering = true;
    try {
      await setupView.refreshAgents();
      const id = ctx.state.sessionId;
      if (id) await api.getSession(id);
    } finally {
      recovering = false;
    }
  };
  window.addEventListener("focus", recover);
  window.addEventListener("pageshow", recover);
  document.addEventListener("visibilitychange", () => {
    if (!document.hidden) recover();
  });
  try {
    const id = sessionStorage.getItem("harbor-session-id");
    if (id) {
      const response = await api.getSession(id);
      if (response.session && !ctx.state.sessionId)
        ctx.onSessionCreated(response.session);
      else if (response.error?.code === "SESSION_NOT_FOUND")
        sessionStorage.removeItem("harbor-session-id");
    }
  } catch {}
  window.addEventListener("pagehide", () => sessionView.destroy(), {
    once: true,
  });
  if (!ctx.state.sessionId) ctx.showTab("setup");
}

boot().catch(() => {
  badge.textContent = "连接失败";
  badge.className = "badge bad";
  setupRoot.textContent = "桌面连接初始化失败，请重新启动应用。";
  document.querySelectorAll(".tab").forEach((button) => {
    button.disabled = true;
  });
});
